package simpleexcelv3

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/xuri/excelize/v2"
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
	titleRow := make([]interface{}, 0)
	
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
	headerRow := make([]interface{}, 0)
	
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
	// Check for Excel row limit (1,048,576)
	if w.currentRow > 1048575 {
		return fmt.Errorf("row number %d exceeds Excel maximum limit of 1048575", w.currentRow)
	}
	
	cell, _ := excelize.CoordinatesToCellName(1, w.currentRow)
	
	// Convert []excelize.Cell to []interface{}
	rowInterface := make([]interface{}, len(rowData.Cells))
	for i, cell := range rowData.Cells {
		rowInterface[i] = cell
	}
	
	if err := w.streamWriter.SetRow(cell, rowInterface); err != nil {
		return err
	}
	
	w.currentRow++
	return nil
}

// Style caching methods
func (w *InterleavedStreamWriter) getOrCreateTitleStyle(section *HorizontalSection) int {
	// Generate a unique key for this style
	var sb strings.Builder
	fmt.Fprintf(&sb, "title:%s", section.ID)
	key := sb.String()

	// Check if we already have this style cached
	if styleID, exists := w.styleCache[key]; exists {
		return styleID
	}

	// Create new style for title
	styleTmpl := &StyleTemplateV3{
		Font: &FontTemplateV3{
			Bold:  true,
			Color: "000000",
		},
		Alignment: &AlignmentTemplate{
			Horizontal: "center",
			Vertical:   "top",
		},
	}
	styleID, err := w.createStyle(styleTmpl)
	if err != nil {
		return 0
	}

	// Cache the style
	w.styleCache[key] = styleID
	return styleID
}

func (w *InterleavedStreamWriter) getOrCreateHeaderStyle(col ColumnConfigV3, section *HorizontalSection) int {
	// Generate a unique key for this style
	var sb strings.Builder
	fmt.Fprintf(&sb, "header:%s:%s", section.ID, col.FieldName)
	key := sb.String()

	// Check if we already have this style cached
	if styleID, exists := w.styleCache[key]; exists {
		return styleID
	}

	// Create new style for header
	styleTmpl := &StyleTemplateV3{
		Font: &FontTemplateV3{
			Bold:  true,
			Color: "000000",
		},
		Alignment: &AlignmentTemplate{
			Horizontal: "center",
			Vertical:   "top",
		},
		Locked: &[]bool{col.IsLocked(false)}[0],
	}
	styleID, err := w.createStyle(styleTmpl)
	if err != nil {
		return 0
	}

	// Cache the style
	w.styleCache[key] = styleID
	return styleID
}

// createStyle creates a new style in the Excel file
func (w *InterleavedStreamWriter) createStyle(tmpl *StyleTemplateV3) (int, error) {
	style := &excelize.Style{}
	if tmpl.Font != nil {
		style.Font = &excelize.Font{
			Bold:  tmpl.Font.Bold,
			Color: strings.TrimPrefix(tmpl.Font.Color, "#"),
		}
	}
	if tmpl.Fill != nil {
		style.Fill = excelize.Fill{
			Type:    "pattern",
			Color:   []string{strings.TrimPrefix(tmpl.Fill.Color, "#")},
			Pattern: 1,
		}
	}
	if tmpl.Alignment != nil {
		style.Alignment = &excelize.Alignment{
			Horizontal: tmpl.Alignment.Horizontal,
			Vertical:   tmpl.Alignment.Vertical,
		}
	}
	if tmpl.Locked != nil {
		style.Protection = &excelize.Protection{
			Locked: *tmpl.Locked,
		}
	}
	return w.file.NewStyle(style)
}