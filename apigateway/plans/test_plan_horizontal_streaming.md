# Test Plan: Horizontal Streaming Implementation

## Overview

This test plan validates the refactored ExcelDataExporterV3 horizontal streaming capability, ensuring it works correctly while maintaining backward compatibility with existing vertical streaming.

## Test Categories

### 1. Unit Tests

#### 1.1 DataProvider Tests

**Test: SliceDataProvider Basic Functionality**

```go
func TestSliceDataProvider_GetRow(t *testing.T) {
    data := []TestData{
        {Name: "Alice", Value: 100},
        {Name: "Bob", Value: 200},
    }

    provider, err := NewSliceDataProvider(data)
    assert.NoError(t, err)

    // Test valid rows
    row0, err := provider.GetRow(0)
    assert.NoError(t, err)
    assert.Equal(t, "Alice", row0.(TestData).Name)

    row1, err := provider.GetRow(1)
    assert.NoError(t, err)
    assert.Equal(t, "Bob", row1.(TestData).Name)

    // Test out of bounds
    row2, err := provider.GetRow(2)
    assert.NoError(t, err)
    assert.Nil(t, row2)
}
```

**Test: ChannelDataProvider Streaming**

```go
func TestChannelDataProvider_Streaming(t *testing.T) {
    dataChan := make(chan interface{}, 3)
    go func() {
        dataChan <- TestData{Name: "Stream1", Value: 1}
        dataChan <- TestData{Name: "Stream2", Value: 2}
        close(dataChan)
    }()

    provider := NewChannelDataProvider(dataChan)

    row0, err := provider.GetRow(0)
    assert.NoError(t, err)
    assert.Equal(t, "Stream1", row0.(TestData).Name)

    row1, err := provider.GetRow(1)
    assert.NoError(t, err)
    assert.Equal(t, "Stream2", row1.(TestData).Name)
}
```

#### 1.2 HorizontalSectionCoordinator Tests

**Test: Basic Row Coordination**

```go
func TestHorizontalSectionCoordinator_GetNextRowData(t *testing.T) {
    // Create two sections with different data
    sectionAData := []TestData{{Name: "A1"}, {Name: "A2"}}
    sectionBData := []TestData{{Name: "B1"}, {Name: "B2"}, {Name: "B3"}}

    providerA, _ := NewSliceDataProvider(sectionAData)
    providerB, _ := NewSliceDataProvider(sectionBData)

    sectionA := &HorizontalSection{
        ID: "section_a",
        DataProvider: providerA,
        Columns: []ColumnConfigV3{{FieldName: "Name", Header: "Name"}},
        RowCount: 2,
    }

    sectionB := &HorizontalSection{
        ID: "section_b",
        DataProvider: providerB,
        Columns: []ColumnConfigV3{{FieldName: "Name", Header: "Name"}},
        RowCount: 3,
    }

    coordinator := NewHorizontalSectionCoordinator([]*HorizontalSection{sectionA, sectionB}, FillStrategyPad)

    // Get first row
    rowData, err := coordinator.GetNextRowData()
    assert.NoError(t, err)
    assert.Equal(t, 2, len(rowData.Cells)) // A1, B1

    // Get second row
    rowData, err = coordinator.GetNextRowData()
    assert.NoError(t, err)
    assert.Equal(t, 2, len(rowData.Cells)) // A2, B2

    // Get third row (A exhausted, B continues)
    rowData, err = coordinator.GetNextRowData()
    assert.NoError(t, err)
    assert.Equal(t, 2, len(rowData.Cells)) // empty, B3
}
```

**Test: Fill Strategy Behavior**

```go
func TestHorizontalSectionCoordinator_FillStrategies(t *testing.T) {
    // Test FillStrategyTruncate
    coordinator := NewHorizontalSectionCoordinator(sections, FillStrategyTruncate)

    // Should stop when shortest section is exhausted
    for i := 0; i < expectedRowCount; i++ {
        _, err := coordinator.GetNextRowData()
        assert.NoError(t, err)
    }

    // Next call should return EOF
    _, err := coordinator.GetNextRowData()
    assert.Equal(t, io.EOF, err)
}
```

#### 1.3 InterleavedStreamWriter Tests

**Test: Header Writing**

```go
func TestInterleavedStreamWriter_WriteHeaders(t *testing.T) {
    // Create mock file and stream writer
    file := excelize.NewFile()
    sw, _ := file.NewStreamWriter("Sheet1")

    // Create coordinator with sections having titles and headers
    coordinator := createTestCoordinator()

    writer := &InterleavedStreamWriter{
        file: file,
        sheetName: "Sheet1",
        streamWriter: sw,
        coordinator: coordinator,
    }

    err := writer.writeHeaders()
    assert.NoError(t, err)

    // Verify headers were written correctly
    // This would require reading back from the file or mocking
}
```

**Test: Row Writing**

```go
func TestInterleavedStreamWriter_WriteRow(t *testing.T) {
    file := excelize.NewFile()
    sw, _ := file.NewStreamWriter("Sheet1")

    writer := &InterleavedStreamWriter{
        file: file,
        sheetName: "Sheet1",
        streamWriter: sw,
        currentRow: 1,
    }

    rowData := &RowData{
        Row: 1,
        Cells: []excelize.Cell{
            {Value: "A1"},
            {Value: "B1"},
        },
    }

    err := writer.writeRow(rowData)
    assert.NoError(t, err)
    assert.Equal(t, 2, writer.currentRow)
}
```

### 2. Integration Tests

#### 2.1 Complete Horizontal Streaming Workflow

**Test: Basic Horizontal Streaming**

```go
func TestExcelDataExporterV3_HorizontalStreaming(t *testing.T) {
    // Create test data
    sectionAData := []TestData{{Name: "Alice", Value: 100}, {Name: "Bob", Value: 200}}
    sectionBData := []TestData{{Name: "Charlie", Value: 300}, {Name: "David", Value: 400}}

    // Create exporter
    exporter := NewExcelDataExporterV3V3()

    // Create horizontal sections
    configA := &HorizontalSectionConfig{
        ID: "section_a",
        DataProvider: NewSliceDataProvider(sectionAData),
        Columns: []ColumnConfigV3{
            {FieldName: "Name", Header: "Name"},
            {FieldName: "Value", Header: "Value"},
        },
        Title: "Section A",
        ShowHeader: true,
    }

    configB := &HorizontalSectionConfig{
        ID: "section_b",
        DataProvider: NewSliceDataProvider(sectionBData),
        Columns: []ColumnConfigV3{
            {FieldName: "Name", Header: "Name"},
            {FieldName: "Value", Header: "Value"},
        },
        Title: "Section B",
        ShowHeader: true,
    }

    // Start horizontal stream
    var buf bytes.Buffer
    streamer, err := exporter.StartHorizontalStream(&buf, configA, configB)
    assert.NoError(t, err)
    defer streamer.Close()

    // Write all rows
    err = streamer.WriteAllRows()
    assert.NoError(t, err)

    // Verify output file
    assert.NotEmpty(t, buf.Bytes())

    // Open and verify Excel file structure
    file, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
    assert.NoError(t, err)
    defer file.Close()

    // Verify sheet exists
    sheets := file.GetSheetList()
    assert.Contains(t, sheets, "Sheet1")

    // Verify data layout (A1, A2 in columns A-B, B1, B2 in columns C-D)
    // This would require reading specific cells and verifying content
}
```

**Test: Mixed Section Types**

```go
func TestExcelDataExporterV3_MixedHorizontalSections(t *testing.T) {
    // Test with sections having different properties:
    // - Some with titles, some without
    // - Some with headers, some without
    // - Different column counts
    // - Different row counts

    // Verify the layout is correct and no conflicts occur
}
```

#### 2.2 Error Handling Tests

**Test: DataProvider Errors**

```go
func TestHorizontalSectionCoordinator_DataProviderErrors(t *testing.T) {
    // Create a DataProvider that returns errors
    errorProvider := &ErrorDataProvider{}

    section := &HorizontalSection{
        ID: "error_section",
        DataProvider: errorProvider,
        Columns: []ColumnConfigV3{{FieldName: "Name", Header: "Name"}},
    }

    coordinator := NewHorizontalSectionCoordinator([]*HorizontalSection{section}, FillStrategyContinue)

    // Should handle errors gracefully based on error handling strategy
    rowData, err := coordinator.GetNextRowData()
    // Verify error handling behavior
}
```

**Test: Stream Writer Errors**

```go
func TestInterleavedStreamWriter_StreamErrors(t *testing.T) {
    // Test behavior when StreamWriter returns errors
    // Should propagate errors appropriately
}
```

### 3. Performance Tests

#### 3.1 Memory Usage Tests

**Test: Large Dataset Memory Usage**

```go
func TestHorizontalStreaming_MemoryEfficiency(t *testing.T) {
    // Create large datasets (100k+ rows)
    // Monitor memory usage during streaming
    // Verify memory usage remains constant (not growing with dataset size)

    var m1, m2 runtime.MemStats
    runtime.GC()
    runtime.ReadMemStats(&m1)

    // Stream large dataset
    streamLargeDataset()

    runtime.ReadMemStats(&m2)

    // Memory growth should be minimal (only for buffering, not full dataset)
    memoryGrowth := m2.Alloc - m1.Alloc
    assert.Less(t, memoryGrowth, uint64(10*1024*1024)) // Less than 10MB growth
}
```

**Test: Concurrent Section Processing**

```go
func TestHorizontalStreaming_ConcurrentProcessing(t *testing.T) {
    // Test with multiple sections processing concurrently
    // Verify no race conditions or data corruption
    // Use -race flag during testing
}
```

#### 3.2 Speed Tests

**Test: Horizontal vs Vertical Performance**

```go
func BenchmarkHorizontalStreaming(b *testing.B) {
    // Benchmark horizontal streaming performance
    // Compare with equivalent vertical streaming
    // Should be comparable for same total data size
}

func BenchmarkVerticalStreaming(b *testing.B) {
    // Benchmark existing vertical streaming for comparison
}
```

### 4. Compatibility Tests

#### 4.1 Backward Compatibility

**Test: Existing Vertical Streaming Still Works**

```go
func TestExcelDataExporterV3_BackwardCompatibility(t *testing.T) {
    // Test that existing StartStreamV3() still works
    // Test that existing Write() calls work
    // Test that existing Close() works

    exporter := NewExcelDataExporterV3V3()
    var buf bytes.Buffer

    streamer, err := exporter.StartStreamV3(&buf)
    assert.NoError(t, err)

    // Use existing API
    streamer.Write("section1", testData)
    err = streamer.Close()
    assert.NoError(t, err)

    // Verify output is valid Excel file
    file, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
    assert.NoError(t, err)
    file.Close()
}
```

**Test: Migration Path**

```go
func TestExcelDataExporterV3_MigrationPath(t *testing.T) {
    // Test that code can be migrated from vertical to horizontal
    // Test helper functions for converting data to DataProvider
    // Test deprecation warnings
}
```

#### 4.2 Cross-Version Compatibility

**Test: Different Excel Versions**

```go
func TestExcelDataExporterV3_ExcelCompatibility(t *testing.T) {
    // Test generated files work in different Excel versions
    // Test in Excel Online, Excel Desktop, LibreOffice
    // Verify no compatibility issues
}
```

### 5. Edge Case Tests

#### 5.1 Empty Data Handling

```go
func TestHorizontalStreaming_EmptySections(t *testing.T) {
    // Test with empty sections
    // Test with nil data providers
    // Test with sections that have no rows
}
```

#### 5.2 Single Section Horizontal

```go
func TestHorizontalStreaming_SingleSection(t *testing.T) {
    // Test horizontal streaming with only one section
    // Should behave like vertical streaming
}
```

#### 5.3 Very Large Sections

```go
func TestHorizontalStreaming_VeryLargeSections(t *testing.T) {
    // Test with sections having millions of rows
    // Test memory and performance characteristics
}
```

## Test Data Structures

```go
// TestData for testing
type TestData struct {
    Name  string `json:"name" yaml:"name"`
    Value int    `json:"value" yaml:"value"`
    Date  time.Time `json:"date" yaml:"date"`
}

// ErrorDataProvider for testing error scenarios
type ErrorDataProvider struct {
    returnError bool
}

func (p *ErrorDataProvider) GetRow(rowIndex int) (interface{}, error) {
    if p.returnError {
        return nil, fmt.Errorf("simulated error")
    }
    return TestData{Name: fmt.Sprintf("ErrorRow%d", rowIndex)}, nil
}

func (p *ErrorDataProvider) GetRowCount() (int, bool) {
    return 10, true
}

func (p *ErrorDataProvider) HasMoreRows() bool {
    return true
}

func (p *ErrorDataProvider) Close() error {
    return nil
}
```

## Test Execution Strategy

### 1. Test Organization

- Unit tests in `*_test.go` files alongside implementation
- Integration tests in separate `integration_test.go` files
- Performance tests using Go's built-in benchmarking

### 2. Test Environment

- Use temporary files for Excel output testing
- Use in-memory buffers where possible
- Clean up resources after each test

### 3. Continuous Integration

- Run unit tests on every commit
- Run integration tests on pull requests
- Run performance tests nightly or on major changes

### 4. Test Coverage Goals

- 90%+ code coverage for new horizontal streaming code
- 100% coverage for error handling paths
- Full coverage of public API methods

This comprehensive test plan ensures the horizontal streaming implementation is robust, performant, and maintains backward compatibility while providing the new functionality.
