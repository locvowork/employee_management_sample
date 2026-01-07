# simpleexcelv2 - Comprehensive Examples

This document provides comprehensive examples demonstrating all major features of the `simpleexcelv2` package.

## Table of Contents

- [Basic Usage](#basic-usage)
- [YAML Configuration](#yaml-configuration)
- [Mixed Configuration](#mixed-configuration)
- [Advanced Styling](#advanced-styling)
- [Hidden Data and Metadata](#hidden-data-and-metadata)
- [Sheet Protection](#sheet-protection)
- [Comparison Features](#comparison-features)
- [Custom Formatters](#custom-formatters)
- [Large Dataset Handling](#large-dataset-handling)
- [Error Handling](#error-handling)
- [Performance Optimization](#performance-optimization)

## Basic Usage

### Simple Programmatic Export

```go
package main

import (
	"context"
	"fmt"
	"github.com/your-org/your-repo/apigateway/pkg/simpleexcelv2"
)

type Employee struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Position string `json:"position"`
}

func main() {
	// Sample data
	employees := []Employee{
		{1, "John Doe", "john@example.com", "Developer"},
		{2, "Jane Smith", "jane@example.com", "Designer"},
		{3, "Bob Johnson", "bob@example.com", "Manager"},
	}

	// Create and configure exporter
	exporter := simpleexcelv2.NewExcelDataExporter()

	err := exporter.AddSheet("Employees").
		AddSection(&simpleexcelv2.SectionConfig{
			Title:      "Employee Directory",
			Data:       employees,
			ShowHeader: true,
			Columns: []simpleexcelv2.ColumnConfig{
				{FieldName: "ID", Header: "Employee ID", Width: 15},
				{FieldName: "Name", Header: "Full Name", Width: 25},
				{FieldName: "Email", Header: "Email Address", Width: 30},
				{FieldName: "Position", Header: "Job Title", Width: 20},
			},
		}).
		Build().
		ExportToExcel(context.Background(), "employees.xlsx")

	if err != nil {
		panic(fmt.Sprintf("Export failed: %v", err))
	}

	fmt.Println("Export completed successfully!")
}
```

### Dynamic Data Export

```go
// Exporting dynamic data (maps)
func exportDynamicData() error {
	data := []map[string]interface{}{
		{"Product": "Laptop", "Price": 1200.50, "Category": "Electronics"},
		{"Product": "Mouse", "Price": 25.00, "Category": "Accessories"},
		{"Product": "Keyboard", "Price": 89.99, "Category": "Accessories"},
	}

	exporter := simpleexcelv2.NewExcelDataExporter()

	return exporter.AddSheet("Products").
		AddSection(&simpleexcelv2.SectionConfig{
			Title:      "Product Catalog",
			Data:       data,
			ShowHeader: true,
			Columns: []simpleexcelv2.ColumnConfig{
				{FieldName: "Product", Header: "Product Name", Width: 20},
				{FieldName: "Price", Header: "Price", Width: 15},
				{FieldName: "Category", Header: "Category", Width: 20},
			},
		}).
		Build().
		ExportToExcel(context.Background(), "products.xlsx")
}
```

## YAML Configuration

### Basic YAML Template

```yaml
# basic_report.yaml
sheets:
  - name: "Employee Report"
    sections:
      - id: "employees"
        title: "Team Members"
        show_header: true
        direction: "vertical"
        title_style:
          font:
            bold: true
            color: "#FFFFFF"
          fill:
            color: "#1565C0"
        header_style:
          font:
            bold: true
          alignment:
            horizontal: "center"
        columns:
          - field_name: "ID"
            header: "Employee ID"
            width: 15
          - field_name: "Name"
            header: "Full Name"
            width: 25
          - field_name: "Email"
            header: "Email Address"
            width: 30
          - field_name: "Position"
            header: "Job Title"
            width: 20
```

### Using YAML Template

```go
func exportWithYAML() error {
	// Read YAML configuration
	data, err := os.ReadFile("basic_report.yaml")
	if err != nil {
		return fmt.Errorf("failed to read YAML: %w", err)
	}

	// Initialize exporter from YAML
	exporter, err := simpleexcelv2.NewExcelDataExporterFromYamlConfig(string(data))
	if err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Sample data
	employees := []Employee{
		{1, "John Doe", "john@example.com", "Developer"},
		{2, "Jane Smith", "jane@example.com", "Designer"},
	}

	// Bind data to YAML section
	exporter.BindSectionData("employees", employees)

	// Export to file
	return exporter.ExportToExcel(context.Background(), "employee_report.xlsx")
}
```

## Mixed Configuration

### YAML + Programmatic Extension

```yaml
# mixed_config.yaml
sheets:
  - name: "Sales Report"
    sections:
      - id: "sales_data"
        title: "Sales Data"
        show_header: true
        columns:
          - field_name: "Product"
            header: "Product"
          - field_name: "Quantity"
            header: "Quantity"
          - field_name: "Price"
            header: "Unit Price"
```

```go
func exportMixedConfig() error {
	// Load YAML configuration
	yamlData, err := os.ReadFile("mixed_config.yaml")
	if err != nil {
		return err
	}

	exporter, err := simpleexcelv2.NewExcelDataExporterFromYamlConfig(string(yamlData))
	if err != nil {
		return err
	}

	// Sample data
	salesData := []map[string]interface{}{
		{"Product": "Widget A", "Quantity": 100, "Price": 10.50},
		{"Product": "Widget B", "Quantity": 75, "Price": 15.75},
	}

	// Bind data to YAML section
	exporter.BindSectionData("sales_data", salesData)

	// Get sheet and add programmatic section
	if sheet := exporter.GetSheet("Sales Report"); sheet != nil {
		// Add summary section
		sheet.AddSection(&simpleexcelv2.SectionConfig{
			Title:      "Summary",
			Type:       simpleexcelv2.SectionTypeTitleOnly,
			ColSpan:    3,
			ShowHeader: false,
			Columns: []simpleexcelv2.ColumnConfig{
				{FieldName: "Summary", Header: "Summary Information"},
			},
		})

		// Add calculations section
		sheet.AddSection(&simpleexcelv2.SectionConfig{
			Title:      "Calculations",
			ShowHeader: true,
			Columns: []simpleexcelv2.ColumnConfig{
				{FieldName: "Total", Header: "Total Sales", Width: 20},
				{FieldName: "Average", Header: "Average Price", Width: 20},
			},
		})
	}

	// Bind data to programmatic sections
	exporter.BindSectionData("summary", []map[string]interface{}{
		{"Summary": "Q1 Sales Report - Generated on " + time.Now().Format("2006-01-02")},
	})

	exporter.BindSectionData("calculations", []map[string]interface{}{
		{"Total": "175", "Average": "13.13"},
	})

	return exporter.ExportToExcel(context.Background(), "mixed_report.xlsx")
}
```

## Advanced Styling

### Custom Styles

```go
func exportWithAdvancedStyling() error {
	// Define custom styles
	titleStyle := &simpleexcelv2.StyleTemplate{
		Font: &simpleexcelv2.FontTemplate{
			Bold:  true,
			Color: "#FFFFFF",
		},
		Fill: &simpleexcelv2.FillTemplate{
			Color: "#2E7D32",
		},
		Alignment: &simpleexcelv2.AlignmentTemplate{
			Horizontal: "center",
			Vertical:   "center",
		},
	}

	headerStyle := &simpleexcelv2.StyleTemplate{
		Font: &simpleexcelv2.FontTemplate{
			Bold: true,
		},
		Fill: &simpleexcelv2.FillTemplate{
			Color: "#E8F5E8",
		},
		Alignment: &simpleexcelv2.AlignmentTemplate{
			Horizontal: "center",
		},
	}

	dataStyle := &simpleexcelv2.StyleTemplate{
		Alignment: &simpleexcelv2.AlignmentTemplate{
			Horizontal: "left",
		},
	}

	// Sample data with different types
	products := []map[string]interface{}{
		{"ID": 1, "Name": "Premium Widget", "Price": 99.99, "Stock": 50, "Status": "Active"},
		{"ID": 2, "Name": "Basic Widget", "Price": 29.99, "Stock": 200, "Status": "Active"},
		{"ID": 3, "Name": "Deluxe Widget", "Price": 199.99, "Stock": 10, "Status": "Discontinued"},
	}

	exporter := simpleexcelv2.NewExcelDataExporter()

	return exporter.AddSheet("Product Catalog").
		AddSection(&simpleexcelv2.SectionConfig{
			Title:        "Product Inventory",
			TitleHeight:  30,
			HeaderHeight: 25,
			DataHeight:   20,
			Data:         products,
			ShowHeader:   true,
			TitleStyle:   titleStyle,
			HeaderStyle:  headerStyle,
			DataStyle:    dataStyle,
			Columns: []simpleexcelv2.ColumnConfig{
				{FieldName: "ID", Header: "Product ID", Width: 15},
				{FieldName: "Name", Header: "Product Name", Width: 30},
				{FieldName: "Price", Header: "Unit Price", Width: 15},
				{FieldName: "Stock", Header: "Stock Level", Width: 15},
				{FieldName: "Status", Header: "Status", Width: 20},
			},
		}).
		Build().
		ExportToExcel(context.Background(), "styled_catalog.xlsx")
}
```

## Hidden Data and Metadata

### Hidden Fields

```go
func exportWithHiddenFields() error {
	// Sample data
	employees := []Employee{
		{1, "John Doe", "john@example.com", "Developer"},
		{2, "Jane Smith", "jane@example.com", "Designer"},
	}

	exporter := simpleexcelv2.NewExcelDataExporter()

	return exporter.AddSheet("Employee Data").
		AddSection(&simpleexcelv2.SectionConfig{
			Title:      "Employee Directory",
			Data:       employees,
			ShowHeader: true,
			Columns: []simpleexcelv2.ColumnConfig{
				{FieldName: "ID", Header: "Employee ID", Width: 15, HiddenFieldName: "db_employee_id"},
				{FieldName: "Name", Header: "Full Name", Width: 25, HiddenFieldName: "db_full_name"},
				{FieldName: "Email", Header: "Email Address", Width: 30, HiddenFieldName: "db_email_address"},
				{FieldName: "Position", Header: "Job Title", Width: 20, HiddenFieldName: "db_job_title"},
			},
		}).
		Build().
		ExportToExcel(context.Background(), "hidden_fields.xlsx")
}
```

### Hidden Sections

```go
func exportWithHiddenSections() error {
	// Main data
	products := []map[string]interface{}{
		{"Name": "Widget A", "Price": 10.50, "Stock": 100},
		{"Name": "Widget B", "Price": 15.75, "Stock": 75},
	}

	// Hidden metadata
	metadata := []map[string]interface{}{
		{"Field": "Name", "Type": "string", "Source": "product_catalog"},
		{"Field": "Price", "Type": "float", "Source": "pricing_system"},
		{"Field": "Stock", "Type": "int", "Source": "inventory_system"},
	}

	exporter := simpleexcelv2.NewExcelDataExporter()

	return exporter.AddSheet("Product Report").
		AddSection(&simpleexcelv2.SectionConfig{
			Title:      "Product Information",
			Data:       products,
			ShowHeader: true,
			Columns: []simpleexcelv2.ColumnConfig{
				{FieldName: "Name", Header: "Product Name", Width: 25},
				{FieldName: "Price", Header: "Unit Price", Width: 15},
				{FieldName: "Stock", Header: "Stock Level", Width: 15},
			},
		}).
		AddSection(&simpleexcelv2.SectionConfig{
			Title:      "Data Dictionary",
			Type:       simpleexcelv2.SectionTypeHidden,
			Data:       metadata,
			ShowHeader: true,
			Columns: []simpleexcelv2.ColumnConfig{
				{FieldName: "Field", Header: "Field Name", Width: 20},
				{FieldName: "Type", Header: "Data Type", Width: 15},
				{FieldName: "Source", Header: "Data Source", Width: 25},
			},
		}).
		Build().
		ExportToExcel(context.Background(), "hidden_sections.xlsx")
}
```

## Sheet Protection

### Basic Protection

```go
func exportWithProtection() error {
	employees := []Employee{
		{1, "John Doe", "john@example.com", "Developer"},
		{2, "Jane Smith", "jane@example.com", "Designer"},
	}

	exporter := simpleexcelv2.NewExcelDataExporter()

	return exporter.AddSheet("Protected Report").
		AddSection(&simpleexcelv2.SectionConfig{
			Title:      "Employee Data",
			Locked:     true, // This locks the entire section
			Data:       employees,
			ShowHeader: true,
			Columns: []simpleexcelv2.ColumnConfig{
				{FieldName: "ID", Header: "Employee ID", Width: 15},
				{FieldName: "Name", Header: "Full Name", Width: 25},
				{FieldName: "Email", Header: "Email Address", Width: 30},
				{FieldName: "Position", Header: "Job Title", Width: 20},
			},
		}).
		Build().
		ExportToExcel(context.Background(), "protected_report.xlsx")
}
```

### Mixed Protection

```go
func exportWithMixedProtection() error {
	products := []map[string]interface{}{
		{"Name": "Widget A", "Price": 10.50, "Stock": 100},
		{"Name": "Widget B", "Price": 15.75, "Stock": 75},
	}

	exporter := simpleexcelv2.NewExcelDataExporter()

	return exporter.AddSheet("Mixed Protection").
		AddSection(&simpleexcelv2.SectionConfig{
			Title:      "Product Data",
			Locked:     true, // Lock entire section by default
			Data:       products,
			ShowHeader: true,
			Columns: []simpleexcelv2.ColumnConfig{
				{FieldName: "Name", Header: "Product Name", Width: 25},
				{FieldName: "Price", Header: "Unit Price", Width: 15, Locked: simpleexcelv2.BoolPtr(false)}, // Override: unlock this column
				{FieldName: "Stock", Header: "Stock Level", Width: 15},
			},
		}).
		Build().
		ExportToExcel(context.Background(), "mixed_protection.xlsx")
}

// Helper function to create pointer to bool
func BoolPtr(b bool) *bool {
	return &b
}
```

## Comparison Features

### Basic Comparison

```yaml
# comparison_config.yaml
sheets:
  - name: "Comparison Report"
    sections:
      - id: "original_data"
        title: "Original Data"
        show_header: true
        columns:
          - field_name: "Product"
            header: "Product"
          - field_name: "Price"
            header: "Original Price"
      - id: "modified_data"
        title: "Modified Data"
        show_header: true
        columns:
          - field_name: "Product"
            header: "Product"
          - field_name: "Price"
            header: "Modified Price"
      - id: "comparison"
        title: "Price Comparison"
        show_header: true
        columns:
          - field_name: "Product"
            header: "Product"
          - field_name: "Diff"
            header: "Price Difference"
            compare_with:
              section_id: "original_data"
              field_name: "Price"
            compare_against:
              section_id: "modified_data"
              field_name: "Price"
```

```go
func exportWithComparison() error {
	// Original data
	originalData := []map[string]interface{}{
		{"Product": "Widget A", "Price": 10.50},
		{"Product": "Widget B", "Price": 15.75},
	}

	// Modified data
	modifiedData := []map[string]interface{}{
		{"Product": "Widget A", "Price": 11.00},
		{"Product": "Widget B", "Price": 15.75},
	}

	// Load YAML configuration
	yamlData, err := os.ReadFile("comparison_config.yaml")
	if err != nil {
		return err
	}

	exporter, err := simpleexcelv2.NewExcelDataExporterFromYamlConfig(string(yamlData))
	if err != nil {
		return err
	}

	// Bind data
	exporter.BindSectionData("original_data", originalData)
	exporter.BindSectionData("modified_data", modifiedData)

	// The comparison section will automatically generate formulas
	// Cell E4 will contain: =IF(B4<>D4, "Diff", "")
	// Cell E5 will contain: =IF(B5<>D5, "Diff", "")

	return exporter.ExportToExcel(context.Background(), "comparison_report.xlsx")
}
```

## Custom Formatters

### Programmatic Formatters

```go
func exportWithCustomFormatters() error {
	type Product struct {
		Name      string
		Price     float64
		Stock     int
		CreatedAt time.Time
	}

	products := []Product{
		{"Widget A", 10.50, 100, time.Now().Add(-24 * time.Hour)},
		{"Widget B", 15.75, 75, time.Now().Add(-48 * time.Hour)},
		{"Widget C", 25.00, 200, time.Now().Add(-72 * time.Hour)},
	}

	exporter := simpleexcelv2.NewExcelDataExporter()

	// Register custom formatters
	exporter.RegisterFormatter("currency", func(v interface{}) interface{} {
		if price, ok := v.(float64); ok {
			return fmt.Sprintf("$%.2f", price)
		}
		return v
	})

	exporter.RegisterFormatter("date", func(v interface{}) interface{} {
		if t, ok := v.(time.Time); ok {
			return t.Format("2006-01-02")
		}
		return v
	})

	exporter.RegisterFormatter("stock_status", func(v interface{}) interface{} {
		if stock, ok := v.(int); ok {
			if stock > 100 {
				return "High"
			} else if stock > 50 {
				return "Medium"
			} else {
				return "Low"
			}
		}
		return v
	})

	return exporter.AddSheet("Formatted Products").
		AddSection(&simpleexcelv2.SectionConfig{
			Title:      "Product Catalog with Formatting",
			Data:       products,
			ShowHeader: true,
			Columns: []simpleexcelv2.ColumnConfig{
				{FieldName: "Name", Header: "Product Name", Width: 25},
				{
					FieldName:     "Price",
					Header:        "Unit Price",
					Width:         15,
					FormatterName: "currency",
				},
				{
					FieldName:     "Stock",
					Header:        "Stock Level",
					Width:         15,
					FormatterName: "stock_status",
				},
				{
					FieldName:     "CreatedAt",
					Header:        "Created Date",
					Width:         20,
					FormatterName: "date",
				},
			},
		}).
		Build().
		ExportToExcel(context.Background(), "formatted_products.xlsx")
}
```

### YAML with Formatters

```yaml
# formatted_config.yaml
sheets:
  - name: "Formatted Report"
    sections:
      - id: "products"
        title: "Product Information"
        show_header: true
        columns:
          - field_name: "Name"
            header: "Product Name"
            width: 25
          - field_name: "Price"
            header: "Unit Price"
            width: 15
            formatter: "currency"
          - field_name: "Stock"
            header: "Stock Level"
            width: 15
            formatter: "stock_status"
          - field_name: "CreatedAt"
            header: "Created Date"
            width: 20
            formatter: "date"
```

```go
func exportYAMLWithFormatters() error {
	// Sample data
	products := []map[string]interface{}{
		{
			"Name":      "Widget A",
			"Price":     10.50,
			"Stock":     100,
			"CreatedAt": time.Now().Add(-24 * time.Hour),
		},
		{
			"Name":      "Widget B",
			"Price":     15.75,
			"Stock":     75,
			"CreatedAt": time.Now().Add(-48 * time.Hour),
		},
	}

	// Load YAML configuration
	yamlData, err := os.ReadFile("formatted_config.yaml")
	if err != nil {
		return err
	}

	exporter, err := simpleexcelv2.NewExcelDataExporterFromYamlConfig(string(yamlData))
	if err != nil {
		return err
	}

	// Register formatters
	exporter.RegisterFormatter("currency", func(v interface{}) interface{} {
		if price, ok := v.(float64); ok {
			return fmt.Sprintf("$%.2f", price)
		}
		return v
	})

	exporter.RegisterFormatter("stock_status", func(v interface{}) interface{} {
		if stock, ok := v.(int); ok {
			if stock > 100 {
				return "High"
			} else if stock > 50 {
				return "Medium"
			} else {
				return "Low"
			}
		}
		return v
	})

	exporter.RegisterFormatter("date", func(v interface{}) interface{} {
		if t, ok := v.(time.Time); ok {
			return t.Format("2006-01-02")
		}
		return v
	})

	// Bind data
	exporter.BindSectionData("products", products)

	return exporter.ExportToExcel(context.Background(), "yaml_formatted.xlsx")
}
```

## Large Dataset Handling

### Streaming Export

```go
func exportLargeDatasetStreaming() error {
	// Simulate large dataset
	generateData := func() []map[string]interface{} {
		data := make([]map[string]interface{}, 10000)
		for i := 0; i < 10000; i++ {
			data[i] = map[string]interface{}{
				"ID":       i,
				"Name":     fmt.Sprintf("Item %d", i),
				"Category": fmt.Sprintf("Category %d", i%10),
				"Price":    float64(i*10 + 50),
				"Stock":    i % 1000,
			}
		}
		return data
	}

	data := generateData()

	exporter := simpleexcelv2.NewExcelDataExporter()

	return exporter.AddSheet("Large Dataset").
		AddSection(&simpleexcelv2.SectionConfig{
			Title:      "Large Dataset Export",
			Data:       data,
			ShowHeader: true,
			Columns: []simpleexcelv2.ColumnConfig{
				{FieldName: "ID", Header: "ID", Width: 10},
				{FieldName: "Name", Header: "Name", Width: 30},
				{FieldName: "Category", Header: "Category", Width: 20},
				{FieldName: "Price", Header: "Price", Width: 15},
				{FieldName: "Stock", Header: "Stock", Width: 15},
			},
		}).
		Build().
		ExportToExcel(context.Background(), "large_dataset.xlsx")
}
```

### CSV Export for Very Large Datasets

```go
func exportVeryLargeDatasetCSV() error {
	// For extremely large datasets, use CSV format
	generateLargeData := func() []map[string]interface{} {
		data := make([]map[string]interface{}, 100000)
		for i := 0; i < 100000; i++ {
			data[i] = map[string]interface{}{
				"ID":       i,
				"Name":     fmt.Sprintf("Item %d", i),
				"Category": fmt.Sprintf("Category %d", i%10),
				"Price":    float64(i*10 + 50),
				"Stock":    i % 1000,
			}
		}
		return data
	}

	data := generateLargeData()

	exporter := simpleexcelv2.NewExcelDataExporter()

	// Use ToCSV for memory efficiency
	return exporter.AddSheet("Very Large Dataset").
		AddSection(&simpleexcelv2.SectionConfig{
			Title:      "Very Large Dataset Export",
			Data:       data,
			ShowHeader: true,
			Columns: []simpleexcelv2.ColumnConfig{
				{FieldName: "ID", Header: "ID"},
				{FieldName: "Name", Header: "Name"},
				{FieldName: "Category", Header: "Category"},
				{FieldName: "Price", Header: "Price"},
				{FieldName: "Stock", Header: "Stock"},
			},
		}).
		Build().
		ToCSV(os.Stdout) // Stream to stdout or any io.Writer
}
```

### Streaming to HTTP Response

```go
func exportToHTTPResponse(w http.ResponseWriter, r *http.Request) error {
	// Generate or fetch data
	data := generateLargeData()

	exporter := simpleexcelv2.NewExcelDataExporter()

	// Configure exporter
	exporter.AddSheet("HTTP Export").
		AddSection(&simpleexcelv2.SectionConfig{
			Title:      "HTTP Response Export",
			Data:       data,
			ShowHeader: true,
			Columns: []simpleexcelv2.ColumnConfig{
				{FieldName: "ID", Header: "ID", Width: 10},
				{FieldName: "Name", Header: "Name", Width: 30},
				{FieldName: "Category", Header: "Category", Width: 20},
				{FieldName: "Price", Header: "Price", Width: 15},
				{FieldName: "Stock", Header: "Stock", Width: 15},
			},
		})

	// Set response headers
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="http_export.xlsx"`)

	// Stream directly to response
	return exporter.ToWriter(w)
}
```

## Error Handling

### Comprehensive Error Handling

```go
func exportWithErrorHandling() error {
	// Sample data that might cause issues
	products := []map[string]interface{}{
		{"Name": "Widget A", "Price": 10.50, "Stock": 100},
		{"Name": "Widget B", "Price": "invalid", "Stock": 75}, // Invalid price
		{"Name": nil, "Price": 25.00, "Stock": 200},           // Invalid name
	}

	exporter := simpleexcelv2.NewExcelDataExporter()

	// Add custom formatter with error handling
	exporter.RegisterFormatter("safe_currency", func(v interface{}) interface{} {
		switch val := v.(type) {
		case float64:
			return fmt.Sprintf("$%.2f", val)
		case int:
			return fmt.Sprintf("$%.2f", float64(val))
		case string:
			// Try to parse string as float
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				return fmt.Sprintf("$%.2f", f)
			}
			return "Invalid Price"
		default:
			return "Unknown Price"
		}
	})

	err := exporter.AddSheet("Error Handling").
		AddSection(&simpleexcelv2.SectionConfig{
			Title:      "Products with Error Handling",
			Data:       products,
			ShowHeader: true,
			Columns: []simpleexcelv2.ColumnConfig{
				{FieldName: "Name", Header: "Product Name", Width: 25},
				{
					FieldName:     "Price",
					Header:        "Unit Price",
					Width:         15,
					FormatterName: "safe_currency",
				},
				{FieldName: "Stock", Header: "Stock Level", Width: 15},
			},
		}).
		Build().
		ExportToExcel(context.Background(), "error_handling.xlsx")

	if err != nil {
		// Handle specific error types
		if strings.Contains(err.Error(), "invalid") {
			return fmt.Errorf("data validation error: %w", err)
		}
		return fmt.Errorf("export failed: %w", err)
	}

	return nil
}
```

### YAML Configuration Error Handling

```go
func exportYAMLErrorHandling() error {
	// Invalid YAML configuration
	invalidYAML := `
sheets:
  - name: "Invalid Config"
    sections:
      - id: "test"
        title: "Test Section"
        # Missing required fields or invalid structure
`

	// Try to create exporter from invalid YAML
	exporter, err := simpleexcelv2.NewExcelDataExporterFromYamlConfig(invalidYAML)
	if err != nil {
		return fmt.Errorf("YAML parsing failed: %w", err)
	}

	// Try to bind data to non-existent section
	exporter.BindSectionData("non_existent", []map[string]interface{}{{"test": "data"}})

	// This should handle the error gracefully
	return exporter.ExportToExcel(context.Background(), "yaml_error.xlsx")
}
```

## Performance Optimization

### Memory-Efficient Large Exports

```go
func exportMemoryOptimized() error {
	// Use streaming for large datasets
	exporter := simpleexcelv2.NewExcelDataExporter()

	// Configure with minimal memory footprint
	exporter.AddSheet("Memory Optimized").
		AddSection(&simpleexcelv2.SectionConfig{
			Title:      "Large Dataset",
			Data:       generateLargeData(),
			ShowHeader: true,
			// Use minimal styling to reduce memory usage
			TitleStyle:  nil,
			HeaderStyle: nil,
			DataStyle:   nil,
			Columns: []simpleexcelv2.ColumnConfig{
				{FieldName: "ID", Header: "ID", Width: 10},
				{FieldName: "Name", Header: "Name", Width: 20},
				{FieldName: "Category", Header: "Category", Width: 15},
			},
		})

	// Use ToWriter for streaming
	return exporter.ToWriter(os.Stdout)
}
```

### Batch Processing

```go
func exportBatchProcessing() error {
	// Process data in batches for very large datasets
	batchSize := 1000
	totalRecords := 50000

	exporter := simpleexcelv2.NewExcelDataExporter()
	sheet := exporter.AddSheet("Batch Processing")

	for i := 0; i < totalRecords; i += batchSize {
		end := i + batchSize
		if end > totalRecords {
			end = totalRecords
		}

		// Generate batch data
		batchData := generateBatchData(i, end)

		// Add section for each batch
		sheet.AddSection(&simpleexcelv2.SectionConfig{
			Title:      fmt.Sprintf("Batch %d-%d", i+1, end),
			Data:       batchData,
			ShowHeader: i == 0, // Only show header for first batch
			Columns: []simpleexcelv2.ColumnConfig{
				{FieldName: "ID", Header: "ID", Width: 10},
				{FieldName: "Name", Header: "Name", Width: 30},
				{FieldName: "Category", Header: "Category", Width: 20},
			},
		})
	}

	return exporter.Build().ExportToExcel(context.Background(), "batch_processing.xlsx")
}

func generateBatchData(start, end int) []map[string]interface{} {
	data := make([]map[string]interface{}, end-start)
	for i := start; i < end; i++ {
		data[i-start] = map[string]interface{}{
			"ID":       i,
			"Name":     fmt.Sprintf("Item %d", i),
			"Category": fmt.Sprintf("Category %d", i%10),
		}
	}
	return data
}
```

### Concurrent Export

```go
func exportConcurrent() error {
	// Export multiple sheets concurrently
	sheets := []struct {
		name   string
		data   []map[string]interface{}
		config *simpleexcelv2.SectionConfig
	}{
		{
			name: "Products",
			data: generateProductData(),
			config: &simpleexcelv2.SectionConfig{
				Title:      "Product Catalog",
				Data:       nil, // Will be set below
				ShowHeader: true,
				Columns: []simpleexcelv2.ColumnConfig{
					{FieldName: "ID", Header: "Product ID", Width: 15},
					{FieldName: "Name", Header: "Product Name", Width: 30},
					{FieldName: "Price", Header: "Price", Width: 15},
				},
			},
		},
		{
			name: "Categories",
			data: generateCategoryData(),
			config: &simpleexcelv2.SectionConfig{
				Title:      "Category List",
				Data:       nil, // Will be set below
				ShowHeader: true,
				Columns: []simpleexcelv2.ColumnConfig{
					{FieldName: "ID", Header: "Category ID", Width: 15},
					{FieldName: "Name", Header: "Category Name", Width: 30},
				},
			},
		},
	}

	// Set data for each config
	for i := range sheets {
		sheets[i].config.Data = sheets[i].data
	}

	// Create exporter and add all sheets
	exporter := simpleexcelv2.NewExcelDataExporter()
	for _, sheet := range sheets {
		exporter.AddSheet(sheet.name).AddSection(sheet.config)
	}

	return exporter.Build().ExportToExcel(context.Background(), "concurrent_export.xlsx")
}
```

This comprehensive examples document demonstrates all major features of the `simpleexcelv2` package, from basic usage to advanced scenarios including error handling, performance optimization, and large dataset processing.
