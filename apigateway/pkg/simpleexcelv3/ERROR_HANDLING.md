# Error Handling Guide for simpleexcelv2

This document provides comprehensive guidance on error handling when using the `simpleexcelv2` package.

## Table of Contents

- [Common Error Types](#common-error-types)
- [Error Handling Patterns](#error-handling-patterns)
- [Validation and Input Errors](#validation-and-input-errors)
- [YAML Configuration Errors](#yaml-configuration-errors)
- [Data Processing Errors](#data-processing-errors)
- [Export and I/O Errors](#export-and-io-errors)
- [Memory and Performance Errors](#memory-and-performance-errors)
- [Best Practices](#best-practices)
- [Error Recovery Strategies](#error-recovery-strategies)

## Common Error Types

### 1. Configuration Errors

#### Invalid YAML Configuration

```go
// Example of invalid YAML
invalidYAML := `
sheets:
  - name: "Test Sheet"
    sections:
      - id: "test"
        # Missing required fields
`

exporter, err := simpleexcelv2.NewExcelDataExporterFromYamlConfig(invalidYAML)
if err != nil {
    // Handle YAML parsing error
    log.Printf("YAML configuration error: %v", err)
    return err
}
```

#### Missing Required Fields

```go
// Missing required section data
exporter := simpleexcelv2.NewExcelDataExporter()
exporter.AddSheet("Test").
    AddSection(&simpleexcelv2.SectionConfig{
        // Missing Title or Data
        Columns: []simpleexcelv2.ColumnConfig{
            {FieldName: "Name", Header: "Name"},
        },
    })

// This will likely cause issues during export
err := exporter.Build().ExportToExcel(ctx, "test.xlsx")
if err != nil {
    log.Printf("Export error: %v", err)
}
```

### 2. Data Processing Errors

#### Invalid Data Types

```go
// Data with incompatible types
data := []map[string]interface{}{
    {"Name": "Product A", "Price": "invalid_price"}, // String instead of number
    {"Name": 123, "Price": 25.00},                   // Number instead of string
}

exporter := simpleexcelv2.NewExcelDataExporter()
exporter.AddSheet("Products").
    AddSection(&simpleexcelv2.SectionConfig{
        Data: data,
        Columns: []simpleexcelv2.ColumnConfig{
            {FieldName: "Name", Header: "Product Name"},
            {FieldName: "Price", Header: "Price"},
        },
    })

// This may cause formatting issues
err := exporter.Build().ExportToExcel(ctx, "products.xlsx")
```

#### Nil or Empty Data

```go
// Nil data
var data []map[string]interface{} // nil slice

exporter := simpleexcelv2.NewExcelDataExporter()
exporter.AddSheet("Empty").
    AddSection(&simpleexcelv2.SectionConfig{
        Data: data, // nil data
        Columns: []simpleexcelv2.ColumnConfig{
            {FieldName: "Name", Header: "Name"},
        },
    })

// This will create an empty section but may not be what you want
err := exporter.Build().ExportToExcel(ctx, "empty.xlsx")
```

### 3. Export and I/O Errors

#### File System Errors

```go
// Permission denied
err := exporter.ExportToExcel(ctx, "/root/protected/file.xlsx")
if err != nil {
    if os.IsPermission(err) {
        log.Printf("Permission denied: %v", err)
        return fmt.Errorf("cannot write to file: %w", err)
    }
    log.Printf("File system error: %v", err)
}

// Disk full
err = exporter.ExportToExcel(ctx, "/full/disk/file.xlsx")
if err != nil {
    if strings.Contains(err.Error(), "no space left") {
        log.Printf("Disk full error: %v", err)
        return fmt.Errorf("disk full: %w", err)
    }
}
```

#### Network/Streaming Errors

```go
// HTTP response streaming error
func exportToHTTP(w http.ResponseWriter, r *http.Request) error {
    exporter := simpleexcelv2.NewExcelDataExporter()
    // ... configure exporter ...

    // Set headers
    w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    w.Header().Set("Content-Disposition", `attachment; filename="export.xlsx"`)

    // Stream to response
    err := exporter.ToWriter(w)
    if err != nil {
        // Check if client disconnected
        if err == http.ErrHandlerTimeout || strings.Contains(err.Error(), "client disconnected") {
            log.Printf("Client disconnected during export: %v", err)
            return nil // Not really an error we can fix
        }
        return fmt.Errorf("streaming error: %w", err)
    }
    return nil
}
```

### 4. Memory and Performance Errors

#### Out of Memory

```go
// Large dataset causing memory issues
largeData := generateLargeDataset(1000000) // 1M records

exporter := simpleexcelv2.NewExcelDataExporter()
exporter.AddSheet("Large").
    AddSection(&simpleexcelv2.SectionConfig{
        Data: largeData,
        Columns: []simpleexcelv2.ColumnConfig{
            {FieldName: "ID", Header: "ID"},
            {FieldName: "Name", Header: "Name"},
        },
    })

// This may cause out of memory error
err := exporter.Build().ExportToExcel(ctx, "large.xlsx")
if err != nil {
    if strings.Contains(err.Error(), "out of memory") {
        log.Printf("Memory error: dataset too large")
        return fmt.Errorf("dataset too large for memory: %w", err)
    }
}
```

#### Timeout Errors

```go
// Context timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Long-running export
err := exporter.ExportToExcel(ctx, "large_export.xlsx")
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        log.Printf("Export timeout after 30 seconds")
        return fmt.Errorf("export timed out: %w", err)
    }
    return fmt.Errorf("export failed: %w", err)
}
```

## Error Handling Patterns

### 1. Defensive Programming

```go
func safeExport(data interface{}, filename string) error {
    // Validate input
    if data == nil {
        return fmt.Errorf("data cannot be nil")
    }

    // Check data type
    v := reflect.ValueOf(data)
    if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
        return fmt.Errorf("data must be a slice or array, got %s", v.Kind())
    }

    // Check if data is empty
    if v.Len() == 0 {
        return fmt.Errorf("data cannot be empty")
    }

    // Create exporter with error handling
    exporter := simpleexcelv2.NewExcelDataExporter()

    // Add error handling for each step
    sheet := exporter.AddSheet("Data")
    if sheet == nil {
        return fmt.Errorf("failed to create sheet")
    }

    section := sheet.AddSection(&simpleexcelv2.SectionConfig{
        Title:      "Export Data",
        Data:       data,
        ShowHeader: true,
        Columns: []simpleexcelv2.ColumnConfig{
            {FieldName: "Name", Header: "Name"},
            {FieldName: "Value", Header: "Value"},
        },
    })
    if section == nil {
        return fmt.Errorf("failed to create section")
    }

    // Build and export with error handling
    builtExporter := exporter.Build()
    if builtExporter == nil {
        return fmt.Errorf("failed to build exporter")
    }

    err := builtExporter.ExportToExcel(context.Background(), filename)
    if err != nil {
        return fmt.Errorf("export failed: %w", err)
    }

    return nil
}
```

### 2. Graceful Degradation

```go
func exportWithFallback(data interface{}, filename string) error {
    // Try primary export method
    err := tryPrimaryExport(data, filename)
    if err == nil {
        return nil
    }

    log.Printf("Primary export failed: %v", err)

    // Try fallback method
    err = tryFallbackExport(data, filename)
    if err == nil {
        log.Printf("Fallback export succeeded")
        return nil
    }

    log.Printf("Fallback export also failed: %v", err)
    return fmt.Errorf("all export methods failed: primary=%v, fallback=%v",
        tryPrimaryExport(data, filename),
        tryFallbackExport(data, filename))
}

func tryPrimaryExport(data interface{}, filename string) error {
    exporter := simpleexcelv2.NewExcelDataExporter()
    return exporter.AddSheet("Primary").
        AddSection(&simpleexcelv2.SectionConfig{
            Data: data,
            Columns: []simpleexcelv2.ColumnConfig{
                {FieldName: "Name", Header: "Name"},
            },
        }).
        Build().
        ExportToExcel(context.Background(), filename)
}

func tryFallbackExport(data interface{}, filename string) error {
    // Use CSV format as fallback
    exporter := simpleexcelv2.NewExcelDataExporter()
    return exporter.AddSheet("Fallback").
        AddSection(&simpleexcelv2.SectionConfig{
            Data: data,
            Columns: []simpleexcelv2.ColumnConfig{
                {FieldName: "Name", Header: "Name"},
            },
        }).
        Build().
        ToCSV(os.Stdout) // Stream to stdout instead
}
```

### 3. Error Recovery

```go
func exportWithRecovery(data interface{}, filename string) error {
    maxRetries := 3
    var lastErr error

    for attempt := 1; attempt <= maxRetries; attempt++ {
        log.Printf("Export attempt %d", attempt)

        err := performExport(data, filename)
        if err == nil {
            log.Printf("Export successful on attempt %d", attempt)
            return nil
        }

        lastErr = err
        log.Printf("Export attempt %d failed: %v", attempt, err)

        // Check if retry is worth it
        if isRetryableError(err) {
            // Wait before retry
            time.Sleep(time.Duration(attempt) * time.Second)
            continue
        }

        // Non-retryable error, give up
        log.Printf("Non-retryable error, giving up")
        break
    }

    return fmt.Errorf("export failed after %d attempts: %w", maxRetries, lastErr)
}

func isRetryableError(err error) bool {
    errMsg := err.Error()

    // Retry on network timeouts
    if strings.Contains(errMsg, "timeout") {
        return true
    }

    // Retry on temporary file system errors
    if strings.Contains(errMsg, "temporary") {
        return true
    }

    // Don't retry on permanent errors
    if strings.Contains(errMsg, "permission denied") ||
       strings.Contains(errMsg, "invalid data") {
        return false
    }

    // Default to retryable
    return true
}
```

## Validation and Input Errors

### 1. Data Validation

```go
func validateExportData(data interface{}) error {
    if data == nil {
        return fmt.Errorf("data cannot be nil")
    }

    v := reflect.ValueOf(data)

    // Check if it's a slice or array
    if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
        return fmt.Errorf("data must be slice or array, got %s", v.Kind())
    }

    // Check if empty
    if v.Len() == 0 {
        return fmt.Errorf("data cannot be empty")
    }

    // Validate each item
    for i := 0; i < v.Len(); i++ {
        item := v.Index(i)
        if err := validateDataItem(item); err != nil {
            return fmt.Errorf("invalid data at index %d: %w", i, err)
        }
    }

    return nil
}

func validateDataItem(item reflect.Value) error {
    // Handle different data types
    switch item.Kind() {
    case reflect.Struct:
        return validateStructItem(item)
    case reflect.Map:
        return validateMapItem(item)
    case reflect.Ptr:
        if item.IsNil() {
            return fmt.Errorf("item cannot be nil pointer")
        }
        return validateDataItem(item.Elem())
    default:
        return fmt.Errorf("unsupported data type: %s", item.Kind())
    }
}

func validateStructItem(item reflect.Value) error {
    // Check for required fields (example)
    nameField := item.FieldByName("Name")
    if !nameField.IsValid() {
        return fmt.Errorf("missing required field 'Name'")
    }

    if nameField.Kind() != reflect.String {
        return fmt.Errorf("field 'Name' must be string, got %s", nameField.Kind())
    }

    return nil
}

func validateMapItem(item reflect.Value) error {
    // Check for required keys (example)
    nameKey := reflect.ValueOf("Name")
    if !item.MapIndex(nameKey).IsValid() {
        return fmt.Errorf("missing required key 'Name'")
    }

    return nil
}
```

### 2. Configuration Validation

```go
func validateSectionConfig(config *simpleexcelv2.SectionConfig) error {
    if config == nil {
        return fmt.Errorf("section config cannot be nil")
    }

    if config.Title == "" {
        return fmt.Errorf("section title cannot be empty")
    }

    if config.Data == nil {
        return fmt.Errorf("section data cannot be nil")
    }

    if len(config.Columns) == 0 {
        return fmt.Errorf("section must have at least one column")
    }

    // Validate columns
    for i, col := range config.Columns {
        if err := validateColumnConfig(col); err != nil {
            return fmt.Errorf("invalid column at index %d: %w", i, err)
        }
    }

    return nil
}

func validateColumnConfig(col simpleexcelv2.ColumnConfig) error {
    if col.FieldName == "" {
        return fmt.Errorf("column field_name cannot be empty")
    }

    if col.Header == "" {
        return fmt.Errorf("column header cannot be empty")
    }

    if col.Width < 0 {
        return fmt.Errorf("column width cannot be negative")
    }

    return nil
}
```

## YAML Configuration Errors

### 1. YAML Parsing Errors

```go
func handleYAMLErrors(yamlContent string) error {
    exporter, err := simpleexcelv2.NewExcelDataExporterFromYamlConfig(yamlContent)
    if err != nil {
        // Check for specific YAML errors
        if strings.Contains(err.Error(), "yaml: line") {
            return fmt.Errorf("YAML syntax error: %w", err)
        }

        if strings.Contains(err.Error(), "unmarshal") {
            return fmt.Errorf("YAML unmarshaling error: %w", err)
        }

        return fmt.Errorf("YAML configuration error: %w", err)
    }

    return nil
}
```

### 2. YAML Validation

```go
func validateYAMLConfig(yamlContent string) error {
    // Parse YAML to validate structure
    var config map[string]interface{}
    if err := yaml.Unmarshal([]byte(yamlContent), &config); err != nil {
        return fmt.Errorf("invalid YAML structure: %w", err)
    }

    // Check required fields
    sheets, ok := config["sheets"].([]interface{})
    if !ok {
        return fmt.Errorf("missing required 'sheets' field")
    }

    if len(sheets) == 0 {
        return fmt.Errorf("sheets cannot be empty")
    }

    // Validate each sheet
    for i, sheet := range sheets {
        sheetMap, ok := sheet.(map[interface{}]interface{})
        if !ok {
            return fmt.Errorf("invalid sheet structure at index %d", i)
        }

        if _, ok := sheetMap["name"]; !ok {
            return fmt.Errorf("sheet at index %d missing required 'name' field", i)
        }

        if _, ok := sheetMap["sections"]; !ok {
            return fmt.Errorf("sheet at index %d missing required 'sections' field", i)
        }
    }

    return nil
}
```

## Data Processing Errors

### 1. Type Conversion Errors

```go
func safeDataConversion(data interface{}) (interface{}, error) {
    v := reflect.ValueOf(data)

    switch v.Kind() {
    case reflect.String:
        return data, nil
    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
        return fmt.Sprintf("%d", data), nil
    case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
        return fmt.Sprintf("%d", data), nil
    case reflect.Float32, reflect.Float64:
        return fmt.Sprintf("%.2f", data), nil
    case reflect.Bool:
        return fmt.Sprintf("%t", data), nil
    case reflect.Struct:
        return convertStructToString(v), nil
    case reflect.Map:
        return convertMapToString(v), nil
    default:
        return nil, fmt.Errorf("unsupported data type for conversion: %s", v.Kind())
    }
}

func convertStructToString(v reflect.Value) string {
    // Convert struct to string representation
    fields := make([]string, 0, v.NumField())
    for i := 0; i < v.NumField(); i++ {
        field := v.Field(i)
        fieldName := v.Type().Field(i).Name
        fieldValue, _ := safeDataConversion(field.Interface())
        fields = append(fields, fmt.Sprintf("%s=%v", fieldName, fieldValue))
    }
    return fmt.Sprintf("{%s}", strings.Join(fields, ", "))
}

func convertMapToString(v reflect.Value) string {
    // Convert map to string representation
    entries := make([]string, 0, v.Len())
    for _, key := range v.MapKeys() {
        keyStr, _ := safeDataConversion(key.Interface())
        valueStr, _ := safeDataConversion(v.MapIndex(key).Interface())
        entries = append(entries, fmt.Sprintf("%v=%v", keyStr, valueStr))
    }
    return fmt.Sprintf("{%s}", strings.Join(entries, ", "))
}
```

### 2. Formatter Errors

```go
func safeFormatter(formatter func(interface{}) interface{}, value interface{}) interface{} {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Formatter panic: %v", r)
        }
    }()

    return formatter(value)
}

func createSafeFormatter(original func(interface{}) interface{}) func(interface{}) interface{} {
    return func(value interface{}) interface{} {
        defer func() {
            if r := recover(); r != nil {
                log.Printf("Formatter error: %v", r)
            }
        }()

        return original(value)
    }
}
```

## Export and I/O Errors

### 1. File System Error Handling

```go
func exportToFileWithRetry(exporter *simpleexcelv2.ExcelDataExporter, filename string, maxRetries int) error {
    var lastErr error

    for attempt := 1; attempt <= maxRetries; attempt++ {
        err := exporter.ExportToExcel(context.Background(), filename)
        if err == nil {
            return nil
        }

        lastErr = err

        // Check error type and decide whether to retry
        if isFileSystemError(err) {
            log.Printf("File system error on attempt %d: %v", attempt, err)
            if attempt < maxRetries {
                time.Sleep(time.Duration(attempt) * time.Second)
                continue
            }
        } else {
            // Non-file system error, don't retry
            break
        }
    }

    return fmt.Errorf("export failed after %d attempts: %w", maxRetries, lastErr)
}

func isFileSystemError(err error) bool {
    errMsg := err.Error()

    // Common file system error indicators
    fileSystemErrors := []string{
        "permission denied",
        "no such file or directory",
        "disk full",
        "quota exceeded",
        "too many open files",
    }

    for _, fsErr := range fileSystemErrors {
        if strings.Contains(errMsg, fsErr) {
            return true
        }
    }

    return false
}
```

### 2. Streaming Error Handling

```go
func streamToWriterWithRecovery(exporter *simpleexcelv2.ExcelDataExporter, w io.Writer) error {
    // Create a buffered writer for better error handling
    bw := bufio.NewWriter(w)
    defer bw.Flush()

    // Use a channel to handle errors from goroutine
    errChan := make(chan error, 1)

    go func() {
        defer func() {
            if r := recover(); r != nil {
                errChan <- fmt.Errorf("streaming panic: %v", r)
            }
        }()

        err := exporter.ToWriter(bw)
        errChan <- err
    }()

    // Wait for completion or context cancellation
    select {
    case err := <-errChan:
        if err != nil {
            return fmt.Errorf("streaming error: %w", err)
        }
        return nil
    case <-context.Background().Done():
        return fmt.Errorf("streaming cancelled: %w", context.Background().Err())
    }
}
```

## Memory and Performance Errors

### 1. Memory Usage Monitoring

```go
func exportWithMemoryMonitoring(exporter *simpleexcelv2.ExcelDataExporter, filename string) error {
    // Get initial memory usage
    var m1, m2 runtime.MemStats
    runtime.GC()
    runtime.ReadMemStats(&m1)

    // Perform export
    err := exporter.ExportToExcel(context.Background(), filename)

    // Get final memory usage
    runtime.GC()
    runtime.ReadMemStats(&m2)

    // Log memory usage
    memoryUsed := m2.Alloc - m1.Alloc
    log.Printf("Memory used for export: %d bytes (%.2f MB)", memoryUsed, float64(memoryUsed)/1024/1024)

    // Check for excessive memory usage
    if memoryUsed > 100*1024*1024 { // 100MB
        log.Printf("Warning: High memory usage detected")
    }

    return err
}
```

### 2. Large Dataset Handling

```go
func exportLargeDatasetWithChunking(data []interface{}, filename string) error {
    const chunkSize = 1000

    exporter := simpleexcelv2.NewExcelDataExporter()
    sheet := exporter.AddSheet("Large Dataset")

    // Process data in chunks
    for i := 0; i < len(data); i += chunkSize {
        end := i + chunkSize
        if end > len(data) {
            end = len(data)
        }

        chunk := data[i:end]

        // Add section for each chunk
        sheet.AddSection(&simpleexcelv2.SectionConfig{
            Title:      fmt.Sprintf("Chunk %d-%d", i+1, end),
            Data:       chunk,
            ShowHeader: i == 0, // Only show header for first chunk
            Columns: []simpleexcelv2.ColumnConfig{
                {FieldName: "ID", Header: "ID", Width: 10},
                {FieldName: "Name", Header: "Name", Width: 30},
            },
        })

        // Force garbage collection after each chunk
        runtime.GC()
    }

    return exporter.Build().ExportToExcel(context.Background(), filename)
}
```

## Best Practices

### 1. Always Validate Input

```go
func validateExportInput(data interface{}, filename string) error {
    // Check data
    if data == nil {
        return fmt.Errorf("data cannot be nil")
    }

    // Check filename
    if filename == "" {
        return fmt.Errorf("filename cannot be empty")
    }

    // Check filename extension
    if !strings.HasSuffix(filename, ".xlsx") && !strings.HasSuffix(filename, ".csv") {
        return fmt.Errorf("filename must have .xlsx or .csv extension")
    }

    // Check if directory exists
    dir := filepath.Dir(filename)
    if _, err := os.Stat(dir); os.IsNotExist(err) {
        return fmt.Errorf("directory does not exist: %s", dir)
    }

    return nil
}
```

### 2. Use Context for Cancellation

```go
func exportWithContext(ctx context.Context, exporter *simpleexcelv2.ExcelDataExporter, filename string) error {
    // Create a channel to receive the export result
    resultChan := make(chan error, 1)

    // Start export in goroutine
    go func() {
        resultChan <- exporter.ExportToExcel(ctx, filename)
    }()

    // Wait for completion or cancellation
    select {
    case err := <-resultChan:
        return err
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

### 3. Implement Proper Cleanup

```go
func exportWithCleanup(exporter *simpleexcelv2.ExcelDataExporter, filename string) error {
    // Create temporary file
    tmpFile, err := os.CreateTemp("", "export_*.xlsx")
    if err != nil {
        return fmt.Errorf("failed to create temp file: %w", err)
    }
    defer func() {
        tmpFile.Close()
        os.Remove(tmpFile.Name())
    }()

    // Export to temp file
    err = exporter.ExportToExcel(context.Background(), tmpFile.Name())
    if err != nil {
        return fmt.Errorf("export to temp file failed: %w", err)
    }

    // Copy to final location
    err = copyFile(tmpFile.Name(), filename)
    if err != nil {
        return fmt.Errorf("failed to copy to final location: %w", err)
    }

    return nil
}

func copyFile(src, dst string) error {
    source, err := os.Open(src)
    if err != nil {
        return err
    }
    defer source.Close()

    destination, err := os.Create(dst)
    if err != nil {
        return err
    }
    defer destination.Close()

    _, err = io.Copy(destination, source)
    return err
}
```

### 4. Log Errors Appropriately

```go
func exportWithLogging(exporter *simpleexcelv2.ExcelDataExporter, filename string) error {
    log.Printf("Starting export to %s", filename)

    start := time.Now()
    err := exporter.ExportToExcel(context.Background(), filename)
    duration := time.Since(start)

    if err != nil {
        log.Printf("Export failed after %v: %v", duration, err)
        return err
    }

    log.Printf("Export completed successfully in %v", duration)
    return nil
}
```

## Error Recovery Strategies

### 1. Circuit Breaker Pattern

```go
type ExportCircuitBreaker struct {
    failureCount int
    lastFailure  time.Time
    state        string // "closed", "open", "half-open"
    threshold    int
    timeout      time.Duration
}

func (cb *ExportCircuitBreaker) Execute(exportFunc func() error) error {
    if cb.state == "open" {
        if time.Since(cb.lastFailure) > cb.timeout {
            cb.state = "half-open"
        } else {
            return fmt.Errorf("circuit breaker is open")
        }
    }

    err := exportFunc()
    if err != nil {
        cb.failureCount++
        cb.lastFailure = time.Now()

        if cb.failureCount >= cb.threshold {
            cb.state = "open"
        }

        return err
    }

    // Success, reset circuit breaker
    cb.failureCount = 0
    cb.state = "closed"
    return nil
}
```

### 2. Retry with Exponential Backoff

```go
func exportWithExponentialBackoff(exporter *simpleexcelv2.ExcelDataExporter, filename string) error {
    maxRetries := 5
    baseDelay := time.Second

    for attempt := 1; attempt <= maxRetries; attempt++ {
        err := exporter.ExportToExcel(context.Background(), filename)
        if err == nil {
            return nil
        }

        if !isRetryableError(err) {
            return err
        }

        // Calculate delay with exponential backoff
        delay := time.Duration(math.Pow(2, float64(attempt))) * baseDelay
        if delay > 30*time.Second {
            delay = 30 * time.Second
        }

        log.Printf("Export attempt %d failed, retrying in %v: %v", attempt, delay, err)
        time.Sleep(delay)
    }

    return fmt.Errorf("export failed after %d attempts", maxRetries)
}
```

This comprehensive error handling guide provides patterns and strategies for handling various types of errors that can occur when using the `simpleexcelv2` package, ensuring robust and reliable export functionality.
