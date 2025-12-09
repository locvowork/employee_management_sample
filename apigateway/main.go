package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/locvowork/employee_management_sample/apigateway/pkg/simpleexcel"
)

// func main() {
// 	ctx := context.Background()

// 	app := bootstrap.NewApp()
// 	if err := app.Initialize(ctx); err != nil {
// 		logger.ErrorLog(ctx, "Failed to initialize application: %v", err)
// 		panic(err)
// 	}

// 	if err := app.Run(); err != nil {
// 		logger.ErrorLog(ctx, "Application failed: %v", err)
// 		panic(err)
// 	}
// }

type Employee struct {
	ID         int
	Name       string
	Department string
	Salary     int
	UpdatedAt  string
}

func main() {
	// Example 1: Section with mixed column locking
	example1_MixedLocking()

	// Example 2: Title-only sections for headers
	example2_TitleOnlySections()

	// Example 3: Hidden sections for metadata
	example3_HiddenSections()

	// Example 4: Complex report with all features
	example4_ComplexReport()

	// Example 5: YAML configuration with column locking
	example5_YAMLConfig()
}

func example1_MixedLocking() {
	fmt.Println("Example 1: Section with mixed column locking")

	// Create boolean pointers for column locking
	unlocked := false

	employees := []Employee{
		{1, "John Doe", "Engineering", 75000, time.Now().Format("2006-01-02")},
		{2, "Jane Smith", "Marketing", 65000, time.Now().Format("2006-01-02")},
		{3, "Bob Johnson", "Sales", 70000, time.Now().Format("2006-01-02")},
	}

	exporter := simpleexcel.NewDataExporter().
		AddSheet("Employees").
		AddSection(&simpleexcel.SectionConfig{
			Title:      "Employee Directory",
			ShowHeader: true,
			Locked:     true, // Section is locked by default
			Data:       employees,
			TitleStyle: &simpleexcel.StyleTemplate{
				Font: &simpleexcel.FontTemplate{Bold: true, Color: "#FFFFFF"},
				Fill: &simpleexcel.FillTemplate{Color: "#1565C0"},
			},
			Columns: []simpleexcel.ColumnConfig{
				{FieldName: "ID", Header: "Employee ID", Width: 15},
				{FieldName: "Name", Header: "Full Name", Width: 25, Locked: &unlocked}, // Editable
				{FieldName: "Department", Header: "Department", Width: 20},
				{FieldName: "Salary", Header: "Salary", Width: 15, Locked: &unlocked}, // Editable
				{FieldName: "UpdatedAt", Header: "Last Updated", Width: 15},
			},
		}).
		Build()

	if err := exporter.ExportToExcel(context.Background(), "example1_mixed_locking.xlsx"); err != nil {
		log.Fatal(err)
	}

	fmt.Println("✓ Created example1_mixed_locking.xlsx")
	fmt.Println("  - ID, Department, and Last Updated columns are locked")
	fmt.Println("  - Name and Salary columns are editable\n")
}

func example2_TitleOnlySections() {
	fmt.Println("Example 2: Title-only sections for headers")

	employees := []Employee{
		{1, "John Doe", "Engineering", 75000, "2024-01-15"},
		{2, "Jane Smith", "Marketing", 65000, "2024-01-16"},
	}

	exporter := simpleexcel.NewDataExporter().
		AddSheet("Report").
		AddSection(&simpleexcel.SectionConfig{
			Title: "QUARTERLY EMPLOYEE REPORT - Q1 2024",
			Type:  simpleexcel.SectionTypeTitleOnly,
			TitleStyle: &simpleexcel.StyleTemplate{
				Font: &simpleexcel.FontTemplate{Bold: true, Color: "#000000"},
			},
		}).
		AddSection(&simpleexcel.SectionConfig{
			Title: "Generated on: " + time.Now().Format("2006-01-02 15:04:05"),
			Type:  simpleexcel.SectionTypeTitleOnly,
		}).
		AddSection(&simpleexcel.SectionConfig{
			Title:      "Employee Data",
			ShowHeader: true,
			Data:       employees,
			Columns: []simpleexcel.ColumnConfig{
				{FieldName: "ID", Header: "ID", Width: 10},
				{FieldName: "Name", Header: "Name", Width: 25},
				{FieldName: "Department", Header: "Department", Width: 20},
				{FieldName: "Salary", Header: "Salary", Width: 15},
			},
		}).
		Build()

	if err := exporter.ExportToExcel(context.Background(), "example2_title_sections.xlsx"); err != nil {
		log.Fatal(err)
	}

	fmt.Println("✓ Created example2_title_sections.xlsx")
	fmt.Println("  - Report header and timestamp as title-only sections")
	fmt.Println("  - Followed by employee data section\n")
}

func example3_HiddenSections() {
	fmt.Println("Example 3: Hidden sections for metadata")

	employees := []Employee{
		{1, "John Doe", "Engineering", 75000, "2024-01-15"},
		{2, "Jane Smith", "Marketing", 65000, "2024-01-16"},
	}

	// Metadata stored in hidden section
	metadata := []Employee{
		{0, "ReportVersion", "1.0", 0, "2024-01-20"},
		// {0, "ExportedBy", "admin", 0, time.Now().Format(time.RFC3339)},
		// {0, "RecordCount", fmt.Sprintf("%d", len(employees)), 0, ""},
	}

	exporter := simpleexcel.NewDataExporter().
		AddSheet("Data").
		AddSection(&simpleexcel.SectionConfig{
			// Title:      "Hidden Metadata",
			Type: simpleexcel.SectionTypeHidden,
			// ShowHeader: true,
			Data: metadata,
			Columns: []simpleexcel.ColumnConfig{
				{FieldName: "Name", Header: "Key", Width: 20},
				{FieldName: "Department", Header: "Value", Width: 30},
				{FieldName: "UpdatedAt", Header: "Timestamp", Width: 25},
			},
		}).
		AddSection(&simpleexcel.SectionConfig{
			Title:      "Employee Data",
			ShowHeader: true,
			Data:       employees,
			Columns: []simpleexcel.ColumnConfig{
				{FieldName: "ID", Header: "ID", Width: 10},
				{FieldName: "Name", Header: "Name", Width: 25},
				{FieldName: "Department", Header: "Department", Width: 20},
			},
		}).
		Build()

	if err := exporter.ExportToExcel(context.Background(), "example3_hidden_metadata.xlsx"); err != nil {
		log.Fatal(err)
	}

	fmt.Println("✓ Created example3_hidden_metadata.xlsx")
	fmt.Println("  - Employee data is visible")
	fmt.Println("  - Metadata section is hidden (can be unhidden by user)\n")
}

func example4_ComplexReport() {
	fmt.Println("Example 4: Complex report with all features")

	unlocked := false
	locked := true

	// Main data
	employees := []Employee{
		{1, "John Doe", "Engineering", 75000, "2024-01-15"},
		{2, "Jane Smith", "Marketing", 65000, "2024-01-16"},
		{3, "Bob Johnson", "Sales", 70000, "2024-01-17"},
	}

	// Metadata
	metadata := []Employee{
		{0, "Version", "2.0", 0, time.Now().Format(time.RFC3339)},
		{0, "Department", "HR", 0, ""},
		{0, "Confidential", "Yes", 0, ""},
	}

	exporter := simpleexcel.NewDataExporter().
		AddSheet("Employee Report").
		// Title section
		AddSection(&simpleexcel.SectionConfig{
			Title:   "EMPLOYEE COMPENSATION REPORT",
			Type:    simpleexcel.SectionTypeTitleOnly,
			ColSpan: 4,
			TitleStyle: &simpleexcel.StyleTemplate{
				Font: &simpleexcel.FontTemplate{Bold: true, Color: "#FFFFFF"},
				Fill: &simpleexcel.FillTemplate{Color: "#0D47A1"},
			},
		}).
		// Subtitle
		AddSection(&simpleexcel.SectionConfig{
			Title:   "For Internal Use Only",
			Type:    simpleexcel.SectionTypeTitleOnly,
			ColSpan: 4,
			TitleStyle: &simpleexcel.StyleTemplate{
				Font: &simpleexcel.FontTemplate{Bold: true, Color: "#D32F2F"},
				Fill: &simpleexcel.FillTemplate{Color: "#e2d6a8ff"},
			},
		}).
		// Main data with mixed locking
		AddSection(&simpleexcel.SectionConfig{
			Title:      "Employee Details",
			ShowHeader: true,
			Locked:     true,
			Data:       employees,
			TitleStyle: &simpleexcel.StyleTemplate{
				Font: &simpleexcel.FontTemplate{Bold: true, Color: "#FFFFFF"},
				Fill: &simpleexcel.FillTemplate{Color: "#1976D2"},
			},
			HeaderStyle: &simpleexcel.StyleTemplate{
				Font: &simpleexcel.FontTemplate{Bold: true},
				Fill: &simpleexcel.FillTemplate{Color: "#BBDEFB"},
			},
			Columns: []simpleexcel.ColumnConfig{
				{FieldName: "ID", Header: "ID", Width: 10, Locked: &locked},
				{FieldName: "Name", Header: "Employee Name", Width: 25, Locked: &unlocked},
				{FieldName: "Department", Header: "Department", Width: 20, Locked: &locked},
				{FieldName: "Salary", Header: "Annual Salary", Width: 15, Locked: &unlocked},
				{FieldName: "UpdatedAt", Header: "Last Modified", Width: 15, Locked: &locked},
			},
		}).
		// Hidden metadata section
		AddSection(&simpleexcel.SectionConfig{
			Title:      "Report Metadata",
			Type:       simpleexcel.SectionTypeHidden,
			ShowHeader: true,
			Data:       metadata,
			Columns: []simpleexcel.ColumnConfig{
				{FieldName: "Name", Header: "Property", Width: 20},
				{FieldName: "Department", Header: "Value", Width: 30},
				{FieldName: "UpdatedAt", Header: "Timestamp", Width: 25},
			},
		}).
		Build()

	if err := exporter.ExportToExcel(context.Background(), "example4_complex_report.xlsx"); err != nil {
		log.Fatal(err)
	}

	fmt.Println("✓ Created example4_complex_report.xlsx")
	fmt.Println("  - Title-only sections for report headers")
	fmt.Println("  - Mixed locking (Name and Salary editable)")
	fmt.Println("  - Hidden metadata section\n")
}

func example5_YAMLConfig() {
	fmt.Println("Example 5: YAML configuration with column locking")

	// Create YAML configuration
	yamlContent := `
sheets:
  - name: "Employee Data"
    sections:
      - id: "header"
        title: "COMPANY EMPLOYEE ROSTER"
        type: "title"
        title_style:
          font:
            bold: true
            color: "#FFFFFF"
          fill:
            color: "#1565C0"
      
      - id: "employees"
        title: "Active Employees"
        show_header: true
        locked: true
        title_style:
          font:
            bold: true
            color: "#000000"
          fill:
            color: "#E3F2FD"
        columns:
          - field_name: "ID"
            header: "Emp ID"
            width: 12
          - field_name: "Name"
            header: "Full Name"
            width: 25
            locked: false
          - field_name: "Department"
            header: "Department"
            width: 20
          - field_name: "Salary"
            header: "Salary"
            width: 15
            locked: false
      
      - id: "metadata"
        title: "Document Metadata"
        type: "hidden"
        show_header: true
        columns:
          - field_name: "Name"
            header: "Key"
            width: 20
          - field_name: "Department"
            header: "Value"
            width: 30
`

	// Write YAML to file
	if err := os.WriteFile("example5_config.yaml", []byte(yamlContent), 0644); err != nil {
		log.Fatal(err)
	}

	// Load from YAML
	exporter, err := simpleexcel.NewDataExporterFromYamlFile("example5_config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	// Bind data
	employees := []Employee{
		{1, "Alice Brown", "Engineering", 80000, "2024-01-15"},
		{2, "Charlie Davis", "Marketing", 70000, "2024-01-16"},
	}

	metadata := []Employee{
		{0, "CreatedDate", time.Now().Format("2006-01-02"), 0, ""},
		{0, "Version", "1.0", 0, ""},
	}

	exporter.BindSectionData("employees", employees)
	exporter.BindSectionData("metadata", metadata)

	if err := exporter.ExportToExcel(context.Background(), "example5_yaml_config.xlsx"); err != nil {
		log.Fatal(err)
	}

	fmt.Println("✓ Created example5_yaml_config.xlsx (from YAML)")
	fmt.Println("  - Configuration loaded from YAML file")
	fmt.Println("  - Name and Salary columns are editable")
	fmt.Println("  - Metadata section is hidden\n")
}
