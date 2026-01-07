package simpleexcelv3

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestDataExporterWithFormatter(t *testing.T) {
	type Product struct {
		Name     string
		Price    float64
		Category string
	}

	data := []Product{
		{"Laptop", 1200.50, "electronics"},
		{"Mouse", 25.00, "ELECTRONICS"},
	}

	exporter := NewExcelDataExporterV3V3()
	exporter.AddSheet("Formatter Test").
		AddSection(&SectionConfigV3{
			Data:       data,
			ShowHeader: true,
			Columns: []ColumnConfigV3{
				{FieldName: "Name", Header: "Product"},
				{
					FieldName: "Price",
					Header:    "Price (Formatted)",
					Formatter: func(v interface{}) interface{} {
						if price, ok := v.(float64); ok {
							return fmt.Sprintf("$%.2f", price)
						}
						return v
					},
				},
				{
					FieldName: "Category",
					Header:    "Category (Upper)",
					Formatter: func(v interface{}) interface{} {
						if cat, ok := v.(string); ok {
							return strings.ToUpper(cat)
						}
						return v
					},
				},
			},
		})

	outputFile := "formatter_test.xlsx"
	defer os.Remove(outputFile)

	err := exporter.ExportToExcel(context.Background(), outputFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	// Verification relies on no error for now
}
