# Implementation Guide: Horizontal Streaming Refactoring

## Overview

This guide provides step-by-step instructions for implementing the horizontal streaming capability in ExcelDataExporterV3. The implementation is designed to be incremental, maintaining backward compatibility while adding the new functionality.

## Implementation Phases

### Phase 1: Core Infrastructure (Week 1)

#### Step 1.1: Create DataProvider Interface and Implementations

**File: `pkg/simpleexcelv3/data_provider.go`**

```go
package simpleexcelv3

import (
    "context"
    "io"
    "reflect"
)

// DataProvider defines the contract for accessing data row-by-row
type DataProvider interface {
    // GetRow returns the data for a specific row index
    GetRow(rowIndex int) (interface{}, error)

    // GetRowCount returns the total number of rows and whether it's known
    GetRowCount() (int, bool)

    // HasMoreRows returns true if there are more rows available
    HasMoreRows() bool

    // Close releases any resources held by the provider
    Close() error
}

// SliceDataProvider implements DataProvider for in-memory slices
type SliceDataProvider struct {
    data     interface{}
    rowCount int
    valueType reflect.Type
    mu       sync.RWMutex
}

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
    p.mu.RLock()
    defer p.mu.RUnlock()

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
    p.mu.RLock()
    defer p.mu.RUnlock()
    return p.rowCount, true
}

func (p *SliceDataProvider) HasMoreRows() bool {
    p.mu.RLock()
    defer p.mu.RUnlock()
    return p.currentRow < p.rowCount
}

func (p *SliceDataProvider) Close() error {
    return nil
}

// ChannelDataProvider implements DataProvider for streaming data
type ChannelDataProvider struct {
    dataChan <-chan interface{}
    buffer   []interface{}
    closed   bool
    mu       sync.RWMutex
}

func NewChannelDataProvider(dataChan <-chan interface{}) *ChannelDataProvider {
    return &ChannelDataProvider{
        dataChan: dataChan,
        buffer:   make([]interface{}, 0),
    }
}

func (p *ChannelDataProvider) GetRow(rowIndex int) (interface{}, error) {
    p.mu.Lock()
    defer p.mu.Unlock()

    // Fill buffer if needed
    if rowIndex >= len(p.buffer) && !p.closed {
        p.fillBuffer(rowIndex + 1)
    }

    if rowIndex < len(p.buffer) {
        return p.buffer[rowIndex], nil
    }

    return nil, nil
}

func (p *ChannelDataProvider) fillBuffer(targetSize int) {
    for len(p.buffer) < targetSize && !p.closed {
        select {
        case item, ok := <-p.dataChan:
            if !ok {
                p.closed = true
                return
            }
            p.buffer = append(p.buffer, item)
        default:
            return
        }
    }
}

func (p *ChannelDataProvider) GetRowCount() (int, bool) {
    p.mu.RLock()
    defer p.mu.RUnlock()
    if p.closed {
        return len(p.buffer), true
    }
    return 0, false // Unknown until channel is closed
}

func (p *ChannelDataProvider) HasMoreRows() bool {
    p.mu.RLock()
    defer p.mu.RUnlock()
    return !p.closed || len(p.buffer) > 0
}

func (p *ChannelDataProvider) Close() error {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.closed = true
    return nil
}
```

#### Step 1.2: Create Horizontal Section Types

**File: `pkg/simpleexcelv3/horizontal_section.go`**

```go
package simpleexcelv3

import (
    "sync"
)

// FillStrategy defines how to handle sections with different row counts
type FillStrategy int

const (
    FillStrategyPad FillStrategy = iota  // Pad shorter sections with empty cells
    FillStrategyTruncate                 // Stop at shortest section
    FillStrategyError                    // Error if sections have different lengths
)

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

func (c *HorizontalSectionCoordinator) convertDataToCells(data interface{}, section *HorizontalSection, startCol int) ([]excelize.Cell, error) {
    if data == nil {
        // Create empty cells for this section
        cells := make([]excelize.Cell, len(section.Columns))
        for i := range cells {
            cells[i] = excelize.Cell{}
        }
        return cells, nil
    }

    cells := make([]excelize.Cell, len(section.Columns))
    for i, col := range section.Columns {
        val := extractValue(reflect.ValueOf(data), col.FieldName)

        // Apply formatter if any
        if col.Formatter != nil {
            val = col.Formatter(val)
        } else if col.FormatterName != "" && section.StyleCache != nil {
            if fn, ok := section.StyleCache[col.FormatterName]; ok {
                val = fn(val)
            }
        }

        cells[i] = excelize.Cell{
            Value: val,
            StyleID: c.getOrCreateCellStyle(section, col),
        }
    }

    return cells, nil
}

func (c *HorizontalSectionCoordinator) createPaddingCells(section *HorizontalSection, startCol int) []excelize.Cell {
    cells := make([]excelize.Cell, len(section.Columns))
    for i := range cells {
        cells[i] = excelize.Cell{}
    }
    return cells
}

func (c *HorizontalSectionCoordinator) getOrCreateCellStyle(section *HorizontalSection, col ColumnConfigV3) int {
    // Implementation for style caching
    // This would use the existing style creation logic
    return 0
}
```

### Phase 2: Interleaved Stream Writer (Week 2)

#### Step 2.1: Create InterleavedStreamWriter

**File: `pkg/simpleexcelv3/interleaved_writer.go`**

```go
package simpleexcelv3

import (
    "io"
    "sync"
)

// InterleavedStreamWriter handles row-by-row interleaved writing
type InterleavedStreamWriter struct {
    file         *excelize.File
    sheetName    string
    streamWriter *excelize.StreamWriter
    coordinator  *HorizontalSectionCoordinator
    currentRow   int
    styleCache   map[string]int
    colStyleCache map[int]int
    pool         *sync.Pool
}

// RowData represents a complete row with data from all sections
type RowData struct {
    Cells []excelize.Cell
    Row   int
}

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
        pool: &sync.Pool{
            New: func() interface{} {
                return &RowData{
                    Cells: make([]excelize.Cell, 0, 100),
                }
            },
        },
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

        // Return RowData to pool
        rowData.Cells = rowData.Cells[:0]
        rowData.Row = 0
        w.pool.Put(rowData)
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

// Style caching methods
func (w *InterleavedStreamWriter) getOrCreateTitleStyle(section *HorizontalSection) int {
    // Implementation using existing style creation logic
    return 0
}

func (w *InterleavedStreamWriter) getOrCreateHeaderStyle(col ColumnConfigV3, section *HorizontalSection) int {
    // Implementation using existing style creation logic
    return 0
}
```

### Phase 3: Enhanced API (Week 3)

#### Step 3.1: Extend ExcelDataExporterV3

**File: `pkg/simpleexcelv3/excel_data_exporter_horizontal.go`**

```go
package simpleexcelv3

import (
    "io"
)

// HorizontalSectionConfig configures a horizontal section
type HorizontalSectionConfig struct {
    ID           string
    Data         interface{} // Will be converted to DataProvider
    Columns      []ColumnConfigV3
    Title        interface{}
    ShowHeader   bool
}

// HorizontalStreamer manages horizontal streaming operations
type HorizontalStreamer struct {
    exporter        *ExcelDataExporterV3
    file            *excelize.File
    interleavedWriter *InterleavedStreamWriter
    writer          io.Writer
}

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

// createDataProvider creates appropriate DataProvider based on data type
func (e *ExcelDataExporterV3) createDataProvider(data interface{}) (DataProvider, error) {
    if data == nil {
        return nil, fmt.Errorf("data cannot be nil")
    }

    v := reflect.ValueOf(data)
    if v.Kind() == reflect.Ptr {
        v = v.Elem()
    }

    switch v.Kind() {
    case reflect.Slice:
        return NewSliceDataProvider(data)
    case reflect.Chan:
        // Convert channel to ChannelDataProvider
        // This would need to handle the channel type properly
        return nil, fmt.Errorf("channel data provider not yet implemented")
    default:
        return NewSliceDataProvider(data)
    }
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

### Phase 4: Backward Compatibility (Week 4)

#### Step 4.1: Maintain Existing API

**File: `pkg/simpleexcelv3/backward_compatibility.go`**

```go
package simpleexcelv3

import (
    "io"
)

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
    switch mode {
    case StreamModeVertical:
        // Use existing implementation
        return e.startStreamV3Vertical(w)
    case StreamModeHorizontal:
        return nil, fmt.Errorf("horizontal mode requires StartHorizontalStream")
    default:
        return e.startStreamV3Vertical(w)
    }
}

// startStreamV3Vertical contains the existing vertical streaming logic
func (e *ExcelDataExporterV3) startStreamV3Vertical(w io.Writer) (*StreamerV3, error) {
    // Copy existing StartStreamV3 implementation here
    // This ensures backward compatibility
    f := excelize.NewFile()
    streamer := &StreamerV3{
        exporter:      e,
        file:          f,
        writer:        w,
        streamWriters: make(map[string]*excelize.StreamWriter),
    }

    // Initialize sheets (existing logic)
    for i, sb := range e.sheets {
        sheetName := sb.name
        if i == 0 {
            f.SetSheetName("Sheet1", sheetName)
        } else {
            f.NewSheet(sheetName)
        }

        sw, err := f.NewStreamWriter(sheetName)
        if err != nil {
            return nil, fmt.Errorf("failed to create stream writer for sheet %s: %w", sheetName, err)
        }
        streamer.streamWriters[sheetName] = sw
    }

    streamer.currentSheetIndex = 0
    streamer.currentSectionIndex = 0
    streamer.currentRow = 1

    if err := streamer.advanceToNextStreamingSection(); err != nil {
        return nil, err
    }

    return streamer, nil
}
```

## Testing Strategy

### Unit Tests

1. **DataProvider Tests**: Test all DataProvider implementations
2. **Coordinator Tests**: Test section coordination logic
3. **Writer Tests**: Test interleaved writing functionality

### Integration Tests

1. **End-to-End Tests**: Test complete horizontal streaming workflow
2. **Compatibility Tests**: Ensure existing vertical streaming still works
3. **Performance Tests**: Validate memory usage and speed

### Manual Testing

1. **Excel File Validation**: Open generated files in Excel/LibreOffice
2. **Large Dataset Testing**: Test with realistic data sizes
3. **Error Scenario Testing**: Test error handling and recovery

## Deployment Strategy

### 1. Feature Flag Approach

```go
// Add feature flag for horizontal streaming
type FeatureFlags struct {
    EnableHorizontalStreaming bool
}

var GlobalFeatureFlags = &FeatureFlags{
    EnableHorizontalStreaming: true,
}
```

### 2. Gradual Rollout

1. **Internal Testing**: Test with internal applications first
2. **Beta Testing**: Enable for select users
3. **Full Rollout**: Enable for all users after validation

### 3. Monitoring

1. **Performance Metrics**: Monitor memory usage and processing time
2. **Error Rates**: Track error rates and types
3. **User Feedback**: Collect feedback on new functionality

## Migration Guide

### For Existing Users

1. **No Changes Required**: Existing code continues to work
2. **Optional Migration**: Users can migrate to horizontal streaming when needed
3. **Documentation**: Provide clear migration examples

### For New Users

1. **Default to Vertical**: Use vertical streaming as default
2. **Horizontal When Needed**: Use horizontal streaming for specific use cases
3. **Best Practices**: Document when to use each approach

This implementation guide provides a comprehensive roadmap for adding horizontal streaming capability to ExcelDataExporterV3 while maintaining full backward compatibility and ensuring robust, performant implementation.
