# Web Framework Integration Guide

This guide shows how to use the `simpleexcelv2` package with popular Go web frameworks, with a focus on efficiently handling large exports using streaming.

## Table of Contents

- [Streaming Large Exports](#streaming-large-exports)
- [Echo Framework](#echo-framework)
- [Gin Framework](#gin-framework)
- [Performance Best Practices](#performance-best-practices)
- [Error Handling](#error-handling)

## Streaming Large Exports

The `simpleexcelv2` package provides efficient streaming capabilities for handling large exports with minimal memory usage. The key methods are:

- `ToWriter(w io.Writer) error` - Streams Excel data directly to any writer
- `ToCSV(w io.Writer) error` - Efficiently exports to CSV format
- `ToBytes() ([]byte, error)` - Exports to in-memory byte slice
- `BuildExcel() (*excelize.File, error)` - Builds Excel file in memory

### Basic Streaming Example

```go
// Basic HTTP handler with streaming
func exportHandler(w http.ResponseWriter, r *http.Request) {
    exporter := simpleexcelv2.NewExcelDataExporter()

    // Configure your exporter
    exporter.AddSheet("Large Data").
        AddSection(&simpleexcelv2.SectionConfig{
            Title:      "Large Dataset",
            Data:       fetchLargeData(), // Your data fetching function
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

## Echo Framework

### Excel Export

```go
// In your handler function
func exportEmployees(c echo.Context) error {
    // Your data
    data := []struct {
        ID   int    `json:"id"`
        Name string `json:"name"`
        Role string `json:"role"`
    }{{
        ID: 1, Name: "John Doe", Role: "Developer",
    }}

    // Create and configure exporter
    exporter := simpleexcelv2.NewExcelDataExporter().
        AddSheet("Employees").
        AddSection(&simpleexcelv2.SectionConfig{
            Title:      "Team Members",
            Data:       data,
            ShowHeader: true,
            Columns: []simpleexcelv2.ColumnConfig{
                {FieldName: "ID", Header: "Employee ID", Width: 15},
                {FieldName: "Name", Header: "Full Name", Width: 25},
                {FieldName: "Role", Header: "Position", Width: 20},
            },
        }).
        Build()

    // Set headers for file download
    c.Response().Header().Set(echo.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="employees.xlsx"`)

    // Stream directly to response
    return exporter.ToWriter(c.Response().Writer)
}
```

### CSV Export

```go
func exportCSV(c echo.Context) error {
    // ... configure exporter ...

    c.Response().Header().Set(echo.HeaderContentType, "text/csv")
    c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="report.csv"`)

    return exporter.ToCSV(c.Response().Writer)
}
```

### Large Dataset Streaming with Echo

```go
func exportLargeData(c echo.Context) error {
    // For very large datasets, consider using a background job or streaming
    data := fetchLargeDataset() // This should be paginated or streamed

    exporter := simpleexcelv2.NewExcelDataExporter().
        AddSheet("Large Dataset").
        AddSection(&simpleexcelv2.SectionConfig{
            Title:      "Large Dataset",
            Data:       data,
            ShowHeader: true,
            Columns: []simpleexcelv2.ColumnConfig{
                {FieldName: "ID", Header: "ID", Width: 10},
                {FieldName: "Name", Header: "Name", Width: 20},
                {FieldName: "Description", Header: "Description", Width: 50},
            },
        }).
        Build()

    c.Response().Header().Set(echo.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="large_dataset.xlsx"`)

    // Use ToWriter for memory efficiency
    return exporter.ToWriter(c.Response().Writer)
}
```

## Gin Framework

### Excel Export

```go
func exportExcel(c *gin.Context) {
    // ... configure exporter ...

    c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    c.Header("Content-Disposition", `attachment; filename="employees.xlsx"`)

    exporter.ToWriter(c.Writer)
}
```

### CSV Export

```go
func exportCSV(c *gin.Context) {
    // ... configure exporter ...

    c.Header("Content-Type", "text/csv")
    c.Header("Content-Disposition", `attachment; filename="report.csv"`)

    exporter.ToCSV(c.Writer)
}
```

### Streaming with Progress Tracking

```go
func exportWithProgress(c *gin.Context) {
    // For very large exports, you might want to track progress
    // This example shows how to use context for cancellation

    ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
    defer cancel()

    // Your data processing logic here
    data := fetchLargeData(ctx)

    exporter := simpleexcelv2.NewExcelDataExporter().
        AddSheet("Progress Report").
        AddSection(&simpleexcelv2.SectionConfig{
            Title:      "Large Dataset with Progress",
            Data:       data,
            ShowHeader: true,
            Columns: []simpleexcelv2.ColumnConfig{
                {FieldName: "ID", Header: "ID"},
                {FieldName: "Name", Header: "Name"},
            },
        }).
        Build()

    c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    c.Header("Content-Disposition", `attachment; filename="progress_report.xlsx"`)

    // Check for cancellation during streaming
    if err := exporter.ToWriter(c.Writer); err != nil {
        if errors.Is(err, context.Canceled) {
            c.JSON(http.StatusRequestTimeout, gin.H{"error": "Export was cancelled"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Export failed"})
        return
    }
}
```

## Best Practices

### 1. Error Handling

Always handle errors from exporter methods:

```go
func exportWithErrorHandling(c echo.Context) error {
    exporter := simpleexcelv2.NewExcelDataExporter()
    // ... configure ...

    if err := exporter.ToWriter(c.Response().Writer); err != nil {
        log.Printf("Export error: %v", err)
        return echo.NewHTTPError(http.StatusInternalServerError, "Export failed")
    }
    return nil
}
```

### 2. Memory Management

For large exports, consider streaming to disk first:

```go
func exportToDisk(c echo.Context) error {
    // Create temporary file
    tmpFile, err := os.CreateTemp("", "export_*.xlsx")
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create temp file")
    }
    defer os.Remove(tmpFile.Name())
    defer tmpFile.Close()

    exporter := simpleexcelv2.NewExcelDataExporter()
    // ... configure ...

    if err := exporter.ExportToExcel(c.Request().Context(), tmpFile.Name()); err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "Export failed")
    }

    // Serve the file
    return c.Attachment(tmpFile.Name(), "export.xlsx")
}
```

### 3. Content Type Headers

Always set appropriate content type headers:

```go
// Excel files
c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

// CSV files
c.Response().Header().Set("Content-Type", "text/csv")
```

### 4. File Names

Use meaningful filenames with proper extensions:

```go
filename := fmt.Sprintf("report_%s.xlsx", time.Now().Format("2006-01-02"))
c.Response().Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
```

### 5. Timeouts

Consider adding timeouts for large exports:

```go
func exportWithTimeout(c echo.Context) error {
    ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Minute)
    defer cancel()

    exporter := simpleexcelv2.NewExcelDataExporter()
    // ... configure ...

    // Pass context to data fetching if possible
    data, err := fetchDataWithContext(ctx)
    if err != nil {
        return echo.NewHTTPError(http.StatusRequestTimeout, "Data fetch timeout")
    }

    exporter.AddSheet("Timeout Test").AddSection(&simpleexcelv2.SectionConfig{
        Data: data,
        // ...
    })

    return exporter.ToWriter(c.Response().Writer)
}
```

### Bulk Data Export with `ToWriter`

For many scenarios, `ToWriter` combined with a pre-fetched dataset is sufficient and easy to implement.

#### Echo Framework

```go
func exportLargeData(c echo.Context) error {
    data := fetchEmployeesFromDB() // Returns []Employee

    exporter := simpleexcelv2.NewExcelDataExporter().
        AddSheet("Employees").
        AddSection(&simpleexcelv2.SectionConfig{
            Title: "All Employees",
            Data:  data,
            ShowHeader: true,
        }).Build()

    c.Response().Header().Set(echo.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="employees.xlsx"`)

    return exporter.ToWriter(c.Response().Writer)
}
```

### Extreme Scale: CSV Export

When dealing with millions of rows where Excel's 1M row limit or memory overhead is an issue, use `ToCSV`.

```go
func streamLargeCSV(c echo.Context) error {
    data := fetchMillionsOfRows()

    exporter := simpleexcelv2.NewExcelDataExporter().
        AddSheet("Report").
        AddSection(&simpleexcelv2.SectionConfig{
            Data: data,
        }).Build()

    c.Response().Header().Set(echo.HeaderContentType, "text/csv")
    c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="large_report.csv"`)

    return exporter.ToCSV(c.Response().Writer)
}
```

### CSV Streaming for Very Large Datasets

For extremely large datasets, consider using CSV format which is more memory-efficient:

```go
func streamLargeCSV(c echo.Context) error {
    c.Response().Header().Set(echo.HeaderContentType, "text/csv")
    c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="large_export.csv"`)

    // Create a writer that streams directly to the response
    w := csv.NewWriter(c.Response())

    // Write headers
    if err := w.Write([]string{"ID", "Name", "Email", "Department", "Salary"}); err != nil {
        return err
    }

    // Stream data in chunks
    for i := 1; i <= 1000000; i++ {
        // Get data in chunks (e.g., from database)
        row := []string{
            strconv.Itoa(i),
            fmt.Sprintf("Employee %d", i),
            fmt.Sprintf("employee%d@example.com", i),
            "Engineering",
            strconv.Itoa(rand.Intn(100000) + 50000),
        }

        if err := w.Write(row); err != nil {
            return err
        }

        // Flush periodically
        if i%1000 == 0 {
            w.Flush()
            if err := w.Error(); err != nil {
                return err
            }
        }
    }

    // Flush any remaining data
    w.Flush()
    return w.Error()
}
```

## Performance Best Practices

### Memory Management

1. **Use Streaming for Large Exports**

   - Always use `ToWriter` or `ToCSV` for large datasets
   - These methods process data in chunks (1000 rows at a time)

2. **CSV for Very Large Datasets**
   - For extremely large datasets, consider using CSV format
   - The `ToCSV` method is optimized for memory efficiency

### Web Server Configuration

1. **Timeouts**

   - Set appropriate timeouts for long-running exports
   - Example for Echo:

   ```go
   e.Server.WriteTimeout = 30 * time.Minute
   e.Server.ReadTimeout = 30 * time.Minute
   ```

2. **Response Compression**

   - Enable gzip compression for smaller network transfer
   - Example for Gin:

   ```go
   router.Use(gzip.Gzip(gzip.DefaultCompression))
   ```

3. **Context Management**
   - Use context for cancellation and timeouts
   - Pass context through your data fetching pipeline

### Background Processing

For very large exports, consider using a background job system:

```go
// Example using a simple background worker
func exportInBackground(userID string, data interface{}) {
    go func() {
        // Process and save to storage
        tmpFile, err := os.CreateTemp("", "export_*.xlsx")
        if err != nil {
            log.Printf("Failed to create temp file: %v", err)
            return
        }
        defer os.Remove(tmpFile.Name())
        defer tmpFile.Close()

        exporter := simpleexcelv2.NewExcelDataExporter()
        // ... configure ...

        if err := exporter.ExportToExcel(context.Background(), tmpFile.Name()); err != nil {
            log.Printf("Export failed: %v", err)
            return
        }

        // Notify user or update job status
        notifyUser(userID, "/downloads/export.xlsx")
    }()
}
```

## Error Handling

### Common Errors

1. **Timeout Errors**

   - Handle context timeouts gracefully
   - Provide meaningful error messages to users

2. **Memory Issues**

   - Monitor memory usage
   - Implement circuit breakers for very large exports

3. **File System Errors**
   - Check disk space before starting large exports
   - Handle permission issues

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

## Monitoring and Logging

1. **Log Export Events**

   - Log start/end of exports
   - Track export sizes and durations

2. **Metrics**
   - Track number of exports
   - Monitor memory usage
   - Track export durations

Example with Prometheus:

```go
var (
    exportDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
        Name:    "export_duration_seconds",
        Help:    "Time taken to generate exports",
        Buckets: []float64{0.1, 0.5, 1, 5, 10, 30, 60},
    })
)

// In your handler
func exportHandler(c echo.Context) error {
    start := time.Now()
    defer func() {
        exportDuration.Observe(time.Since(start).Seconds())
    }()

    // ... export logic ...
}
```

## Integration with Other Systems

### Database Integration

```go
func exportFromDatabase(c echo.Context) error {
    // Use database cursor for large datasets
    rows, err := db.Query("SELECT id, name, email FROM users")
    if err != nil {
        return err
    }
    defer rows.Close()

    var users []User
    for rows.Next() {
        var u User
        if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
            return err
        }
        users = append(users, u)
    }

    exporter := simpleexcelv2.NewExcelDataExporter().
        AddSheet("Users").
        AddSection(&simpleexcelv2.SectionConfig{
            Data: users,
            Columns: []simpleexcelv2.ColumnConfig{
                {FieldName: "ID", Header: "User ID"},
                {FieldName: "Name", Header: "Name"},
                {FieldName: "Email", Header: "Email"},
            },
        }).
        Build()

    c.Response().Header().Set(echo.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="users.xlsx"`)

    return exporter.ToWriter(c.Response().Writer)
}
```

### API Integration

```go
func exportFromAPI(c echo.Context) error {
    // Fetch data from external API
    resp, err := http.Get("https://api.example.com/data")
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    var data []Item
    if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
        return err
    }

    exporter := simpleexcelv2.NewExcelDataExporter().
        AddSheet("API Data").
        AddSection(&simpleexcelv2.SectionConfig{
            Data: data,
            Columns: []simpleexcelv2.ColumnConfig{
                {FieldName: "ID", Header: "ID"},
                {FieldName: "Name", Header: "Name"},
            },
        }).
        Build()

    c.Response().Header().Set(echo.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="api_data.xlsx"`)

    return exporter.ToWriter(c.Response().Writer)
}
```
