package simpleexcelv3

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"

	"github.com/xuri/excelize/v2"
)

// RowData represents a complete row with data from all sections
type RowData struct {
	Cells []excelize.Cell
	Row   int
}

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
		} else if col.FormatterName != "" {
			// FormatterName is for style caching, not value formatting
			// We'll handle this in style creation
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
	// Generate a unique key for this style
	var sb strings.Builder
	// We don't have direct access to style templates in ColumnConfigV3
	// For now, we'll use a simple key based on the column properties
	fmt.Fprintf(&sb, "col:%s|locked:%v", col.FieldName, col.IsLocked(false))
	key := sb.String()

	// Check if we already have this style cached
	if styleID, exists := section.StyleCache[key]; exists {
		return styleID
	}

	// For now, we'll return 0 (no style) since we don't have access to the file
	// This will be fixed when we integrate with the InterleavedStreamWriter
	return 0
}

// Helper function to extract value from reflect.Value
func extractValue(item reflect.Value, fieldName string) interface{} {
	if item.Kind() == reflect.Struct {
		t := item.Type()
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.Name == fieldName {
				return item.Field(i).Interface()
			}
		}
	} else if item.Kind() == reflect.Map {
		val := item.MapIndex(reflect.ValueOf(fieldName))
		if val.IsValid() {
			return val.Interface()
		}
	}
	return ""
}