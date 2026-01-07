# Technical Specification: Horizontal Streaming Implementation

## Core Data Structures

### 1. DataProvider Interface

```go
// DataProvider defines the contract for accessing data row-by-row
type DataProvider interface {
    // GetRow returns the data for a specific row index
    // Returns nil if row doesn't exist
    GetRow(rowIndex int) (interface{}, error)

    // GetRowCount returns the total number of rows and whether it's known
    // Returns (0, false) if unknown (streaming data)
    GetRowCount() (int, bool)

    // HasMoreRows returns true if there are more rows available
    HasMoreRows() bool

    // Close releases any resources held by the provider
    Close() error
}

// SliceDataProvider implements DataProvider for in-memory slices
type SliceDataProvider struct {
    data     interface{} // slice of structs or maps
    rowCount int
    valueType reflect.Type
}

// ChannelDataProvider implements DataProvider for streaming data
type ChannelDataProvider struct {
    dataChan <-chan interface{}
    buffer   []interface{}
    closed   bool
    mu       sync.RWMutex
}
```

### 2. Horizontal Section Management

```go
// HorizontalSection represents a section in horizontal layout
type HorizontalSection struct {
    ID           string
    DataProvider DataProvider
    Columns      []ColumnConfigV3
    Title        interface{}
    ShowHeader   bool
    RowCount     int
    HasMoreRows  bool
    CurrentRow   int
    StyleCache   map[string]int
}

// HorizontalSectionCoordinator manages multiple horizontal sections
type HorizontalSectionCoordinator struct {
    sections     []*HorizontalSection
    currentRow   int
    maxRowCount  int
    fillStrategy FillStrategy
    mu           sync.RWMutex
}

type FillStrategy int

const (
    FillStrategyPad FillStrategy = iota  // Pad shorter sections with empty cells
    FillStrategyTruncate                 // Stop at shortest section
    FillStrategyError                    // Error if sections have different lengths
)
```

### 3. Interleaved Stream Writer

```go
// InterleavedStreamWriter handles row-by-row interleaved writing
type InterleavedStreamWriter struct {
    file         *excelize.File
    sheetName    string
    streamWriter *excelize.StreamWriter
    coordinator  *HorizontalSectionCoordinator
    currentRow   int
    styleCache   map[string]int
    colStyleCache map[int]int
}

// RowData represents a complete row with data from all sections
type RowData struct {
    Cells []excelize.Cell
    Row   int
}
```

## Implementation Details

### 1. DataProvider Implementations

```go
// NewSliceDataProvider creates a DataProvider for slice data
func NewSliceDataProvider(data interface{}) (*SliceDataProvider, error) {
    v := reflect.ValueOf(data)
    if v.Kind() == reflect.Ptr {
        v = v.Elem()
    }

    if v.Kind() != reflect.Slice {
        return nil, fmt.Errorf("data must be a slice, got %s", v.Kind())
    }

    return &SliceDataProvider{
        data:     data,
        rowCount: v.Len(),
        valueType: v.Type().Elem(),
    }, nil
}

func (p *SliceDataProvider) GetRow(rowIndex int) (interface{}, error) {
    v := reflect.ValueOf(p.data)
    if v.Kind() == reflect.Ptr {
        v = v.Elem()
    }

    if rowIndex < 0 || rowIndex >= v.Len() {
        return nil, nil
    }

    return v.Index(rowIndex).Interface(), nil
}

func (p *SliceDataProvider) GetRowCount() (int, bool) {
    return p.rowCount, true
}

func (p *SliceDataProvider) HasMoreRows() bool {
    return p.currentRow < p.rowCount
}

func (p *SliceDataProvider) Close() error {
    return nil
}
```

### 2. Horizontal Section Coordinator

```go
// NewHorizontalSectionCoordinator creates a coordinator for horizontal sections
func NewHorizontalSectionCoordinator(sections []*HorizontalSection, strategy FillStrategy) *HorizontalSectionCoordinator {
    return &HorizontalSectionCoordinator{
        sections:     sections,
        fillStrategy: strategy,
        maxRowCount:  0,
    }
}

// GetNextRowData combines data from all sections for the next row
func (c *HorizontalSectionCoordinator) GetNextRowData() (*RowData, error) {
    c.mu.Lock()
    defer c.mu.Unlock()

    if !c.hasMoreRows() {
        return nil, io.EOF
    }

    rowData := &RowData{
        Cells: make([]excelize.Cell, 0),
        Row:   c.currentRow + 1,
    }

    colIndex := 1
    allSectionsExhausted := true

    for _, section := range c.sections {
        if section.CurrentRow < section.RowCount || section.HasMoreRows {
            allSectionsExhausted = false

            // Get data for this section's row
            data, err := section.DataProvider.GetRow(section.CurrentRow)
            if err != nil {
                return nil, fmt.Errorf("error getting row %d from section %s: %w",
                    section.CurrentRow, section.ID, err)
            }

            // Convert data to cells for this section
            sectionCells, err := c.convertDataToCells(data, section, colIndex)
            if err != nil {
                return nil, fmt.Errorf("error converting data for section %s: %w",
                    section.ID, err)
            }

            rowData.Cells = append(rowData.Cells, sectionCells...)
            colIndex += len(section.Columns)

            section.CurrentRow++
        } else {
            // Section is exhausted, add padding if needed
            if c.fillStrategy == FillStrategyPad {
                paddingCells := c.createPaddingCells(section, colIndex)
                rowData.Cells = append(rowData.Cells, paddingCells...)
                colIndex += len(section.Columns)
            }
        }
    }

    if allSectionsExhausted {
        return nil, io.EOF
    }

    c.currentRow++
    return rowData, nil
}

func (c *HorizontalSectionCoordinator) hasMoreRows() bool {
    for _, section := range c.sections {
        if section.CurrentRow < section.RowCount || section.HasMoreRows {
            return true
        }
    }
    return false
}
```

### 3. Interleaved Stream Writer Implementation

```go
// NewInterleavedStreamWriter creates a new interleaved stream writer
func NewInterleavedStreamWriter(file *excelize.File, sheetName string, coordinator *HorizontalSectionCoordinator) (*InterleavedStreamWriter, error) {
    sw, err := file.NewStreamWriter(sheetName)
    if err != nil {
        return nil, fmt.Errorf("failed to create stream writer: %w", err)
    }

    return &InterleavedStreamWriter{
        file:         file,
        sheetName:    sheetName,
        streamWriter: sw,
        coordinator:  coordinator,
        currentRow:   1,
        styleCache:   make(map[string]int),
        colStyleCache: make(map[int]int),
    }, nil
}

// WriteAllRows writes all rows from the coordinator
func (w *InterleavedStreamWriter) WriteAllRows() error {
    // Write headers first
    if err := w.writeHeaders(); err != nil {
        return fmt.Errorf("failed to write headers: %w", err)
    }

    // Write data rows
    for {
        rowData, err := w.coordinator.GetNextRowData()
        if err == io.EOF {
            break
        }
        if err != nil {
            return fmt.Errorf("failed to get next row: %w", err)
        }

        if err := w.writeRow(rowData); err != nil {
            return fmt.Errorf("failed to write row %d: %w", rowData.Row, err)
        }
    }

    return nil
}

// writeHeaders writes the title and header rows for all sections
func (w *InterleavedStreamWriter) writeHeaders() error {
    // Write titles
    titleRow := make([]excelize.Cell, 0)
    colIndex := 1

    for _, section := range w.coordinator.sections {
        if section.Title != nil {
            // Create title cell spanning all columns for this section
            titleCell := excelize.Cell{
                Value: section.Title,
                StyleID: w.getOrCreateTitleStyle(section),
            }

            // Add title cell at the start of this section's columns
            for i := 0; i < len(section.Columns); i++ {
                if i == 0 {
                    titleRow = append(titleRow, titleCell)
                } else {
                    titleRow = append(titleRow, excelize.Cell{})
                }
            }
        } else {
            // Add empty cells for sections without titles
            for range section.Columns {
                titleRow = append(titleRow, excelize.Cell{})
            }
        }
    }

    if len(titleRow) > 0 {
        cell, _ := excelize.CoordinatesToCellName(1, w.currentRow)
        if err := w.streamWriter.SetRow(cell, titleRow); err != nil {
            return err
        }
        w.currentRow++
    }

    // Write headers
    headerRow := make([]excelize.Cell, 0)

    for _, section := range w.coordinator.sections {
        if section.ShowHeader {
            for _, col := range section.Columns {
                headerCell := excelize.Cell{
                    Value:   col.Header,
                    StyleID: w.getOrCreateHeaderStyle(col, section),
                }
                headerRow = append(headerRow, headerCell)
            }
        }
    }

    if len(headerRow) > 0 {
        cell, _ := excelize.CoordinatesToCellName(1, w.currentRow)
        if err := w.streamWriter.SetRow(cell, headerRow); err != nil {
            return err
        }
        w.currentRow++
    }

    return nil
}

// writeRow writes a single row of data
func (w *InterleavedStreamWriter) writeRow(rowData *RowData) error {
    cell, _ := excelize.CoordinatesToCellName(1, w.currentRow)

    if err := w.streamWriter.SetRow(cell, rowData.Cells); err != nil {
        return err
    }

    w.currentRow++
    return nil
}
```

## New API Methods

### 1. Enhanced ExcelDataExporterV3

```go
// StartHorizontalStream initializes horizontal streaming with multiple sections
func (e *ExcelDataExporterV3) StartHorizontalStream(w io.Writer, sections ...*HorizontalSectionConfig) (*HorizontalStreamer, error) {
    if len(sections) == 0 {
        return nil, fmt.Errorf("at least one section is required")
    }

    // Create file and sheet
    f := excelize.NewFile()
    sheetName := "Sheet1"
    f.SetSheetName("Sheet1", sheetName)

    // Create horizontal sections
    horizontalSections := make([]*HorizontalSection, len(sections))
    for i, config := range sections {
        // Create DataProvider from config data
        provider, err := e.createDataProvider(config.Data)
        if err != nil {
            return nil, fmt.Errorf("failed to create data provider for section %s: %w", config.ID, err)
        }

        horizontalSections[i] = &HorizontalSection{
            ID:           config.ID,
            DataProvider: provider,
            Columns:      config.Columns,
            Title:        config.Title,
            ShowHeader:   config.ShowHeader,
            RowCount:     0, // Will be determined by DataProvider
            HasMoreRows:  true,
            CurrentRow:   0,
            StyleCache:   make(map[string]int),
        }
    }

    // Create coordinator
    coordinator := NewHorizontalSectionCoordinator(horizontalSections, FillStrategyPad)

    // Create interleaved stream writer
    interleavedWriter, err := NewInterleavedStreamWriter(f, sheetName, coordinator)
    if err != nil {
        return nil, err
    }

    return &HorizontalStreamer{
        exporter:        e,
        file:            f,
        interleavedWriter: interleavedWriter,
        writer:          w,
    }, nil
}

// StartStreamV3Enhanced provides enhanced streaming with mode selection
func (e *ExcelDataExporterV3) StartStreamV3Enhanced(w io.Writer, mode StreamMode) (*StreamerV3, error) {
    switch mode {
    case StreamModeVertical:
        return e.StartStreamV3(w)
    case StreamModeHorizontal:
        return nil, fmt.Errorf("horizontal mode requires StartHorizontalStream")
    default:
        return e.StartStreamV3(w)
    }
}
```

### 2. Horizontal Streamer

```go
// HorizontalStreamer manages horizontal streaming operations
type HorizontalStreamer struct {
    exporter        *ExcelDataExporterV3
    file            *excelize.File
    interleavedWriter *InterleavedStreamWriter
    writer          io.Writer
}

// WriteAllRows writes all data rows
func (s *HorizontalStreamer) WriteAllRows() error {
    return s.interleavedWriter.WriteAllRows()
}

// Flush flushes the stream writer
func (s *HorizontalStreamer) Flush() error {
    return s.interleavedWriter.streamWriter.Flush()
}

// Close closes the streamer and writes the file
func (s *HorizontalStreamer) Close() error {
    if err := s.Flush(); err != nil {
        return err
    }

    if _, err := s.file.WriteTo(s.writer); err != nil {
        return err
    }

    return nil
}
```

## Configuration and Options

### 1. Stream Configuration

```go
// HorizontalStreamOptions configures horizontal streaming behavior
type HorizontalStreamOptions struct {
    BufferSize     int             // Buffer size for data providers
    FillStrategy   FillStrategy    // How to handle sections with different row counts
    ErrorHandling  ErrorHandling   // How to handle errors
    StyleCaching   bool            // Enable style caching for performance
    ColumnWidths   map[string]float64 // Custom column widths by field name
}

type ErrorHandling int

const (
    ErrorHandlingStop ErrorHandling = iota  // Stop on first error
    ErrorHandlingContinue                   // Continue with error markers
    ErrorHandlingSkip                       // Skip problematic rows
)
```

### 2. Backward Compatibility

```go
// StreamMode maintains backward compatibility
type StreamMode int

const (
    StreamModeVertical StreamMode = iota
    StreamModeHorizontal
)

// Deprecated: Use StartHorizontalStream instead
func (e *ExcelDataExporterV3) StartStreamV3(w io.Writer) (*StreamerV3, error) {
    return e.StartStreamV3WithMode(w, StreamModeVertical)
}

func (e *ExcelDataExporterV3) StartStreamV3WithMode(w io.Writer, mode StreamMode) (*StreamerV3, error) {
    // Existing implementation for vertical streaming
    // This maintains full backward compatibility
}
```

## Performance Optimizations

### 1. Style Caching

```go
// getOrCreateStyle efficiently caches styles to avoid recreation
func (w *InterleavedStreamWriter) getOrCreateStyle(styleTemplate *StyleTemplateV3) (int, error) {
    key := w.generateStyleKey(styleTemplate)

    if styleID, exists := w.styleCache[key]; exists {
        return styleID, nil
    }

    styleID, err := w.exporter.createStyle(w.file, styleTemplate)
    if err != nil {
        return 0, err
    }

    w.styleCache[key] = styleID
    return styleID, nil
}

func (w *InterleavedStreamWriter) generateStyleKey(styleTemplate *StyleTemplateV3) string {
    // Generate unique key based on style properties
    var key strings.Builder
    if styleTemplate.Font != nil {
        key.WriteString(fmt.Sprintf("f:%v:%s|", styleTemplate.Font.Bold, styleTemplate.Font.Color))
    }
    if styleTemplate.Fill != nil {
        key.WriteString(fmt.Sprintf("i:%s|", styleTemplate.Fill.Color))
    }
    if styleTemplate.Alignment != nil {
        key.WriteString(fmt.Sprintf("a:%s:%s|", styleTemplate.Alignment.Horizontal, styleTemplate.Alignment.Vertical))
    }
    return key.String()
}
```

### 2. Memory Management

```go
// RowDataPool manages row data objects to reduce allocations
type RowDataPool struct {
    pool sync.Pool
}

func NewRowDataPool() *RowDataPool {
    return &RowDataPool{
        pool: sync.Pool{
            New: func() interface{} {
                return &RowData{
                    Cells: make([]excelize.Cell, 0, 100), // Pre-allocate capacity
                }
            },
        },
    }
}

func (p *RowDataPool) Get() *RowData {
    return p.pool.Get().(*RowData)
}

func (p *RowDataPool) Put(rowData *RowData) {
    rowData.Cells = rowData.Cells[:0] // Reset slice
    rowData.Row = 0
    p.pool.Put(rowData)
}
```

This technical specification provides the concrete implementation details needed to refactor the ExcelDataExporterV3 to support horizontal streaming while maintaining backward compatibility and performance.
