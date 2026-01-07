package simpleexcelv3

import (
	"testing"
)

func TestDataExporter_HiddenSectionStyle(t *testing.T) {
	exporter := NewExcelDataExporterV3V3()

	type Product struct {
		Name  string
		Price float64
	}

	data := []Product{
		{"Hidden Item", 10.0},
	}

	exporter.AddSheet("HiddenSectionTest").
		AddSection(&SectionConfigV3{
			Title:      "Hidden Section",
			Type:       SectionTypeV3Hidden, // This should trigger the default style
			ShowHeader: true,
			Data:       data,
			Columns: []ColumnConfigV3{
				{FieldName: "Name", Header: "Name"},
				{FieldName: "Price", Header: "Price"},
			},
		})

	excelFile, err := exporter.BuildExcel()
	if err != nil {
		t.Fatalf("Failed to build excel: %v", err)
	}

	sheetName := "HiddenSectionTest"

	// Logic:
	// Row 1: Title
	// Row 2: Header
	// Row 3: Data (Should be hidden and styled)

	// Check Data Row (Row 3)
	val, _ := excelFile.GetCellValue(sheetName, "A3")
	if val != "Hidden Item" {
		t.Errorf("Expected data value 'Hidden Item', got '%s'", val)
	}

	// Verify Style (Non-zero ID implies style applied)
	styleID, err := excelFile.GetCellStyle(sheetName, "A3")
	if err != nil {
		t.Errorf("Failed to get cell style: %v", err)
	}
	if styleID == 0 {
		t.Errorf("Expected style ID > 0 for hidden data row, got 0")
	}

	// Verify Visibility (Should be hidden because of SectionTypeV3Hidden)
	visible, _ := excelFile.GetRowVisible(sheetName, 3)
	if visible {
		t.Errorf("Row 3 should be hidden")
	}
}
