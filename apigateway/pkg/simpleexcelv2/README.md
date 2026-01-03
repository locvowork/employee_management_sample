# simpleexcelv2 - Simple Data Exporter

A lightweight Go library for exporting data to Excel files with advanced styling, layout support, and mixed configuration capabilities.

## Features

- **Simple API**: Easy-to-use fluent interface for building Excel exports
- **YAML Support**: Define templates with YAML for consistent report generation
- **Mixed Configuration**: Combine YAML templates with programmatic dynamic updates
- **Hidden Data**: Support for hidden columns (metadata) and hidden sections with distinct styling
- **Advanced Protection**: Smart cell locking (unused cells unlocked) and formatting permissions
- **Formatters**: Custom data formatting (e.g., currency, dates) via function registration
- **Flexible Layouts**: Position sections vertically or horizontally
- **Runtime Data Binding**: Bind data to templates at runtime
- **Comparison Features**: Generate comparison formulas between sections
- **Streaming Support**: Efficient memory usage for large exports with `ToWriter()` and `ToCSV()`
- **AutoFilter**: Built-in Excel auto-filter support

## Installation

```bash
go get github.com/your-org/your-repo/apigateway/pkg/simpleexcelv2
```

## Quick Start

### Programmatic Usage

```go
package main

import (
	"context"
	"github.com/your-org/your-repo/apigateway/pkg/simpleexcelv2"
)

type Employee struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

func main() {
	// Sample data
	employees := []Employee{
		{1, "John Doe", "Developer"},
		{2, "Jane Smith", "Designer"},
	}

	// Create and configure exporter
	err := simpleexcelv2.NewExcelDataExporter().
		AddSheet("Employees").
		AddSection(&simpleexcelv2.SectionConfig{
			Title:      "Team Members",
			Data:       employees,
			ShowHeader: true,
			Columns: []simpleexcelv2.ColumnConfig{
				{FieldName: "ID", Header: "Employee ID", Width: 15},
				{FieldName: "Name", Header: "Full Name", Width: 25},
				{FieldName: "Role", Header: "Position", Width: 20},
			},
		}).
		Build().
		ExportToExcel(context.Background(), "employees.xlsx")

	if err != nil {
		panic(err)
	}
}
```

### YAML Template Example

```yaml
# report.yaml
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
        columns:
          - field_name: "ID"
            header: "Employee ID"
            width: 15
          - field_name: "Name"
            header: "Full Name"
            width: 25
          - field_name: "Role"
            header: "Position"
            width: 20
```

### Using YAML Template

```go
import (
    "os"
    "github.com/your-org/your-repo/apigateway/pkg/simpleexcelv2"
)

// Read YAML file
data, err := os.ReadFile("report.yaml")
if err != nil {
    log.Fatal(err)
}

// Initialize exporter
exporter, err := simpleexcelv2.NewExcelDataExporterFromYamlConfig(string(data))
if err != nil {
    log.Fatal(err)
}

// Bind data to the section defined in YAML
exporter.BindSectionData("employees", employees)

// Export to file
err = exporter.ExportToExcel(context.Background(), "employee_report.xlsx")
if err != nil {
    log.Fatal(err)
}
```

## Advanced Features

### Hidden Data & Metadata

You can include data in your Excel report that is hidden from the user by default but can be unhidden for inspection or processing.

**Hidden Fields (Columns)**
Add a `HiddenFieldName` to any column configuration. This will generate a hidden row immediately below the section title containing these field names. This is useful for mapping Excel columns back to database fields.

**Hidden Sections**
Set a section's type to `hidden` (or `SectionTypeHidden` in Go).

- **Behavior**: Data rows in this section are automatically hidden.
- **Styling**: Hidden rows have a **yellow background** (`#FFFF00`) by default to distinguish them as metadata when unhidden.

### Sheet Protection

When `Locked: true` is set on any section or column:

1.  **Unused Cells Unlocked**: All cells outside the specific report sections are automatically **unlocked**. Users can freely add data to the rest of the sheet.
2.  **Report Integrity**: Cells within `Locked` sections are read-only.
3.  **Hidden Row Locking**: Hidden metadata rows are explicitly locked to prevent tampering, even if unhidden.
4.  **Formatting Allowed**: Row and Column formatting is enabled in protected sheets, allowing users to **hide/unhide** rows to view metadata.

### Mixed Configuration (YAML + Fluent)

You can load a base template from YAML and then extend it programmatically.

```go
// 1. Load from YAML
exporter, _ := simpleexcelv2.NewExcelDataExporterFromYamlConfig(yamlConfig)

// 2. Bind Data to YAML sections
exporter.BindSectionData("employees", employees)

// 3. Extend programmatically
if sheet := exporter.GetSheet("Employee Report"); sheet != nil {
    sheet.AddSection(&simpleexcelv2.SectionConfig{
        Title: "Debug Info",
        Type:  simpleexcelv2.SectionTypeHidden,
        Data:  debugData,
        // ...
    })
}
```

### Comparison Features

Generate comparison formulas between sections automatically:

```yaml
sheets:
  - name: "Comparison Report"
    sections:
      - id: "section_a"
        title: "Original Data"
        columns:
          - field_name: "Value"
            header: "Value"
      - id: "section_b"
        title: "Modified Data"
        columns:
          - field_name: "Value"
            header: "Value"
      - id: "comparison"
        title: "Differences"
        columns:
          - field_name: "Diff"
            header: "Diff Status"
            compare_with:
              section_id: "section_a"
              field_name: "Value"
            compare_against:
              section_id: "section_b"
              field_name: "Value"
```

### Custom Formatters

Register custom formatters for data transformation:

```go
exporter := simpleexcelv2.NewExcelDataExporter()

// Register formatter by name
exporter.RegisterFormatter("currency", func(v interface{}) interface{} {
    if price, ok := v.(float64); ok {
        return fmt.Sprintf("$%.2f", price)
    }
    return v
})

// Use in configuration
exporter.AddSheet("Products").
    AddSection(&simpleexcelv2.SectionConfig{
        Data: products,
        Columns: []simpleexcelv2.ColumnConfig{
            {
                FieldName:     "Price",
                Header:        "Price",
                FormatterName: "currency", // References registered formatter
            },
        },
    })
```

## API Reference

### ExcelDataExporter

#### Constructors

- `NewExcelDataExporter()` - Creates a new ExcelDataExporter instance
- `NewExcelDataExporterFromYamlConfig(config string)` - Creates an ExcelDataExporter from a YAML string

#### Methods

- `AddSheet(name string) *SheetBuilder` - Start building a new sheet
- `GetSheet(name string) *SheetBuilder` - Retrieve an existing sheet by name
- `GetSheetByIndex(index int) *SheetBuilder` - Retrieve an existing sheet by index
- `RegisterFormatter(name string, fn func(interface{}) interface{})` - Register a value formatter
- `BindSectionData(id string, data interface{}) *ExcelDataExporter` - Bind data to a YAML section
- `ExportToExcel(ctx context.Context, path string) error` - Export to Excel file
- `ToBytes() ([]byte, error)` - Export to in-memory byte slice
- `ToWriter(w io.Writer) error` - Stream export to writer (memory efficient)
- `ToCSV(w io.Writer) error` - Export to CSV format (memory efficient for large datasets)
- `BuildExcel() (*excelize.File, error)` - Build Excel file in memory

### SheetBuilder

#### Methods

- `AddSection(config *SectionConfig) *SheetBuilder` - Add a section to the sheet
- `Build() *ExcelDataExporter` - Complete sheet building and return to exporter

### SectionConfig

```go
type SectionConfig struct {
    ID             string         `yaml:"id"`
    Title          string         `yaml:"title"`
    ColSpan        int            `yaml:"col_span"`        // Number of columns to span for title-only sections
    Data           interface{}    `yaml:"-"`               // Data is bound at runtime
    SourceSections []string       `yaml:"source_sections"` // IDs of sections this depends on
    Type           string         `yaml:"type"`            // "full", "title", "hidden"
    Locked         bool           `yaml:"locked"`          // Section-level lock (default for all columns)
    ShowHeader     bool           `yaml:"show_header"`
    Direction      string         `yaml:"direction"`       // "horizontal" or "vertical"
    Position       string         `yaml:"position"`        // e.g., "A1"
    TitleStyle     *StyleTemplate `yaml:"title_style"`
    HeaderStyle    *StyleTemplate `yaml:"header_style"`
    DataStyle      *StyleTemplate `yaml:"data_style"`
    TitleHeight    float64        `yaml:"title_height"`
    HeaderHeight   float64        `yaml:"header_height"`
    DataHeight     float64        `yaml:"data_height"`
    HasFilter      bool           `yaml:"has_filter"`
    Columns        []ColumnConfig `yaml:"columns"`
}
```

### ColumnConfig

```go
type ColumnConfig struct {
    FieldName       string                        `yaml:"field_name"` // Struct field name or map key
    Header          string                        `yaml:"header"`
    Width           float64                       `yaml:"width"`
    Height          float64                       `yaml:"height"`
    Locked          *bool                         `yaml:"locked"`            // Column-level lock override (overrides section Locked)
    Formatter       func(interface{}) interface{} `yaml:"-"`                 // Optional custom formatter function (Programmatic)
    FormatterName   string                        `yaml:"formatter"`         // Name of registered formatter (YAML)
    HiddenFieldName string                        `yaml:"hidden_field_name"` // Hidden field name for backend use
    CompareWith     *CompareConfig                `yaml:"compare_with"`      // For injecting comparison formulas
    CompareAgainst  *CompareConfig                `yaml:"compare_against"`   // For injecting comparison formulas
}
```

### StyleTemplate

```go
type StyleTemplate struct {
    Font      *FontTemplate      `yaml:"font"`
    Fill      *FillTemplate      `yaml:"fill"`
    Alignment *AlignmentTemplate `yaml:"alignment"`
    Locked    *bool              `yaml:"locked"`
}

type AlignmentTemplate struct {
    Horizontal string `yaml:"horizontal"` // center, left, right
    Vertical   string `yaml:"vertical"`   // top, center, bottom
}

type FontTemplate struct {
    Bold  bool   `yaml:"bold"`
    Color string `yaml:"color"` // Hex color
}

type FillTemplate struct {
    Color string `yaml:"color"` // Hex color
}
```

## Performance Considerations

### Memory Management

1. **Use Streaming for Large Exports**: Always use `ToWriter()` or `ToCSV()` for large datasets
2. **CSV for Very Large Datasets**: For extremely large datasets, consider using CSV format
3. **Batch Processing**: For large datasets, process in batches
4. **Background Jobs**: For very large exports, consider using a job queue

### Web Server Configuration

1. **Timeouts**: Set appropriate timeouts for long-running exports

   ```go
   // In your route handler
   ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Minute)
   defer cancel()

   // Pass this context to your data fetching logic
   data, err := fetchLargeDataset(ctx)
   ```

2. **Response Compression**: Enable gzip compression for smaller network transfer
3. **Background Processing**: For very large exports, consider using a background job system

### Streaming Examples

#### Basic Streaming

```go
func exportHandler(w http.ResponseWriter, r *http.Request) {
    exporter := simpleexcelv2.NewExcelDataExporter()

    // Configure your exporter
    exporter.AddSheet("Large Data").
        AddSection(&simpleexcelv2.SectionConfig{
            Title:      "Large Dataset",
            Data:       fetchLargeData(),
            ShowHeader: true,
            // ... other config
        })

    // Set headers for file download
    w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    w.Header().Set("Content-Disposition", `attachment; filename="large_export.xlsx"`)

    // Stream directly to response
    if err := exporter.ToWriter(w); err != nil {
        log.Printf("Export failed: %v", err)
        http.Error(w, "Export failed", http.StatusInternalServerError)
    }
}
```

#### CSV Streaming for Very Large Datasets

```go
func streamLargeCSV(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/csv")
    w.Header().Set("Content-Disposition", `attachment; filename="large_export.csv"`)

    exporter := simpleexcelv2.NewExcelDataExporter()
    exporter.AddSheet("Report").
        AddSection(&simpleexcelv2.SectionConfig{
            Data: fetchMillionsOfRows(),
        })

    if err := exporter.ToCSV(w); err != nil {
        log.Printf("CSV export failed: %v", err)
        http.Error(w, "Export failed", http.StatusInternalServerError)
    }
}
```

## Best Practices

1. **Error Handling**: Always handle errors from exporter methods
2. **Memory Management**: For large exports, consider streaming to disk first
3. **Content Type Headers**: Always set appropriate content type headers
4. **File Names**: Use meaningful filenames with proper extensions
5. **Timeouts**: Consider adding timeouts for large exports
6. **Caching**: Cache generated reports when possible
7. **Validation**: Validate data before passing to exporter
8. **Resource Cleanup**: Always close files and clean up resources

## Error Handling

### Common Errors

1. **Timeout Errors**: Handle context timeouts gracefully
2. **Memory Issues**: Monitor memory usage and implement circuit breakers
3. **File System Errors**: Check disk space and handle permission issues
4. **Data Validation**: Validate input data structure and types

### Example Error Handler

```go
func handleExportError(err error, c echo.Context) error {
    if errors.Is(err, context.DeadlineExceeded) {
        return c.JSON(http.StatusRequestTimeout, map[string]string{
            "error": "Export took too long, please try with a smaller dataset",
        })
    }

    log.Printf("Export error: %v", err)
    return c.JSON(http.StatusInternalServerError, map[string]string{
        "error": "Failed to generate export",
    })
}
```

## License

[MIT](LICENSE)
