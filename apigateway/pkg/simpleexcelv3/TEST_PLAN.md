# Unit Test Plan for simpleexcelv2

This document outlines the comprehensive unit testing strategy for the `simpleexcelv2` package to ensure all features are properly tested and documented.

## Current Test Coverage Analysis

### Existing Tests

- ✅ Basic alignment functionality
- ✅ Auto-filter functionality
- ✅ Column height configuration
- ✅ Comparison features
- ✅ Data conversion utilities
- ✅ Dynamic map export
- ✅ Formatter functionality
- ✅ Hidden field functionality
- ✅ Hidden row locking
- ✅ Hidden section styling
- ✅ Large export streaming
- ✅ Mixed configuration (YAML + programmatic)
- ✅ Named formatter functionality
- ✅ Partial configuration
- ✅ Sheet protection
- ✅ Style defaults
- ✅ YAML hidden field styling

### Missing Test Coverage

#### 1. Core API Methods

- [ ] `ToBytes()` method - Convert Excel to byte slice
- [ ] `GetSheetByIndex()` method - Retrieve sheet by index
- [ ] `BuildExcel()` method - Build Excel file in memory
- [ ] Error handling for invalid inputs

#### 2. Advanced Features

- [ ] Comparison formula generation with multiple sections
- [ ] Section metadata tracking and coordinate resolution
- [ ] Advanced protection with mixed locked/unlocked cells
- [ ] Title-only sections with colspan functionality
- [ ] Position-based section placement

#### 3. Edge Cases

- [ ] Empty data handling
- [ ] Nil data handling
- [ ] Invalid YAML configuration
- [ ] Circular section dependencies
- [ ] Very large datasets (>1000 rows)
- [ ] Memory usage under stress

#### 4. Integration Tests

- [ ] Full workflow: YAML config → data binding → export
- [ ] Mixed configuration workflow
- [ ] Streaming export with large datasets
- [ ] Error propagation through the entire pipeline

## Test Implementation Plan

### Phase 1: Core API Testing

#### TestToBytes.go

```go
package simpleexcelv2

import (
	"context"
	"testing"
)

func TestToBytes(t *testing.T) {
	// Test basic ToBytes functionality
	// Test with multiple sheets
	// Test with YAML configuration
	// Test with formatters
	// Test error handling
}

func TestToBytesWithMultipleSheets(t *testing.T) {
	// Test ToBytes with multiple sheets
}

func TestToBytesWithYAMLConfig(t *testing.T) {
	// Test ToBytes with YAML configuration
}

func TestToBytesWithFormatters(t *testing.T) {
	// Test ToBytes with custom formatters
}
```

#### TestGetSheetByIndex.go

```go
package simpleexcelv2

import (
	"testing"
)

func TestGetSheetByIndex(t *testing.T) {
	// Test valid index access
	// Test out of bounds index
	// Test with single sheet
	// Test with multiple sheets
}

func TestGetSheetByIndexWithYAML(t *testing.T) {
	// Test GetSheetByIndex with YAML configuration
}
```

#### TestBuildExcel.go

```go
package simpleexcelv2

import (
	"testing"
)

func TestBuildExcel(t *testing.T) {
	// Test basic BuildExcel functionality
	// Test with empty configuration
	// Test with invalid data
	// Test memory usage
}

func TestBuildExcelWithYAML(t *testing.T) {
	// Test BuildExcel with YAML configuration
}

func TestBuildExcelWithMixedConfig(t *testing.T) {
	// Test BuildExcel with mixed configuration
}
```

### Phase 2: Advanced Features Testing

#### TestComparisonAdvanced.go

```go
package simpleexcelv2

import (
	"testing"
)

func TestComparisonWithMultipleSections(t *testing.T) {
	// Test comparison formulas with multiple source sections
	// Test complex comparison scenarios
	// Test error handling for invalid comparisons
}

func TestSectionMetadataTracking(t *testing.T) {
	// Test section metadata tracking
	// Test coordinate resolution
	// Test field offset calculations
}
```

#### TestAdvancedProtection.go

```go
package simpleexcelv2

import (
	"testing"
)

func TestMixedLocking(t *testing.T) {
	// Test mixed locked and unlocked cells
	// Test protection with complex layouts
	// Test hidden row locking
}

func TestTitleOnlySections(t *testing.T) {
	// Test title-only sections
	// Test colspan functionality
	// Test positioning
}
```

### Phase 3: Edge Cases and Stress Testing

#### TestEdgeCases.go

```go
package simpleexcelv2

import (
	"testing"
)

func TestEmptyData(t *testing.T) {
	// Test with empty data slices
	// Test with nil data
	// Test with empty maps
}

func TestInvalidConfigurations(t *testing.T) {
	// Test invalid YAML
	// Test circular dependencies
	// Test invalid field names
}

func TestLargeDatasets(t *testing.T) {
	// Test with large datasets (>1000 rows)
	// Test memory usage
	// Test performance
}
```

#### TestStress.go

```go
package simpleexcelv2

import (
	"testing"
	"context"
	"time"
)

func TestMemoryUsage(t *testing.T) {
	// Test memory usage under stress
	// Test garbage collection
	// Test resource cleanup
}

func TestConcurrentAccess(t *testing.T) {
	// Test concurrent access to exporter
	// Test race conditions
	// Test thread safety
}
```

### Phase 4: Integration Tests

#### TestIntegration.go

```go
package simpleexcelv2

import (
	"testing"
	"context"
	"os"
)

func TestFullWorkflow(t *testing.T) {
	// Test complete workflow: config → data → export
	// Test error handling throughout pipeline
	// Test cleanup
}

func TestStreamingIntegration(t *testing.T) {
	// Test streaming with large datasets
	// Test ToWriter functionality
	// Test ToCSV functionality
}

func TestErrorPropagation(t *testing.T) {
	// Test error propagation
	// Test graceful degradation
	// Test error recovery
}
```

## Test Data and Fixtures

### Test Data Structures

```go
// Complex test data for comprehensive testing
type ComplexTestData struct {
	ID         int                    `json:"id"`
	Name       string                 `json:"name"`
	Metadata   map[string]interface{} `json:"metadata"`
	Nested     NestedData             `json:"nested"`
}

type NestedData struct {
	Field1 string `json:"field1"`
	Field2 int    `json:"field2"`
}

// Large dataset generator
func generateLargeDataset(size int) []ComplexTestData {
	data := make([]ComplexTestData, size)
	for i := 0; i < size; i++ {
		data[i] = ComplexTestData{
			ID:   i,
			Name: fmt.Sprintf("Item %d", i),
			Metadata: map[string]interface{}{
				"key1": fmt.Sprintf("value%d", i),
				"key2": i * 2,
			},
			Nested: NestedData{
				Field1: fmt.Sprintf("nested%d", i),
				Field2: i * 3,
			},
		}
	}
	return data
}
```

### YAML Test Fixtures

```yaml
# complex_test_config.yaml
sheets:
  - name: "Complex Test"
    sections:
      - id: "section_a"
        title: "Section A"
        show_header: true
        direction: "vertical"
        locked: true
        columns:
          - field_name: "ID"
            header: "ID"
            width: 10
            locked: true
          - field_name: "Name"
            header: "Name"
            width: 20
            hidden_field_name: "db_name"
      - id: "section_b"
        title: "Section B"
        show_header: true
        direction: "horizontal"
        columns:
          - field_name: "Metadata"
            header: "Metadata"
            width: 30
      - id: "comparison"
        title: "Comparison"
        show_header: true
        columns:
          - field_name: "Diff"
            header: "Diff"
            compare_with:
              section_id: "section_a"
              field_name: "ID"
            compare_against:
              section_id: "section_b"
              field_name: "ID"
```

## Performance Benchmarks

### Benchmark Tests

```go
package simpleexcelv2

import (
	"testing"
)

func BenchmarkToBytesSmall(b *testing.B) {
	// Benchmark ToBytes with small datasets
}

func BenchmarkToBytesLarge(b *testing.B) {
	// Benchmark ToBytes with large datasets
}

func BenchmarkBuildExcel(b *testing.B) {
	// Benchmark BuildExcel performance
}

func BenchmarkStreaming(b *testing.B) {
	// Benchmark streaming performance
}
```

## Test Execution Strategy

### 1. Unit Tests

- Run with `go test ./pkg/simpleexcelv2/...`
- Target 90%+ code coverage
- Focus on individual components

### 2. Integration Tests

- Run with `go test -tags=integration ./pkg/simpleexcelv2/...`
- Test complete workflows
- Use real file I/O

### 3. Performance Tests

- Run with `go test -bench=. ./pkg/simpleexcelv2/...`
- Monitor memory usage
- Track execution time

### 4. Stress Tests

- Run with large datasets
- Monitor resource usage
- Test error conditions

## Continuous Integration

### Test Matrix

- Go versions: 1.19, 1.20, 1.21
- Operating systems: Linux, macOS, Windows
- Architecture: amd64, arm64

### Quality Gates

- All tests must pass
- Code coverage > 90%
- No memory leaks
- Performance within acceptable bounds

## Documentation Updates

### Test Documentation

- Update README with test instructions
- Add CONTRIBUTING.md for test development
- Document test data structures
- Provide examples for common test scenarios

### Code Comments

- Add comprehensive comments to test files
- Document test assumptions
- Explain complex test logic
- Provide usage examples

This comprehensive test plan ensures that all features of the `simpleexcelv2` package are thoroughly tested, documented, and maintainable.
