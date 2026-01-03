# simpleexcelv3: Batch Streaming Excel Exporter

`simpleexcelv3` is a memory-efficient Go library for exporting large datasets to Excel using a streaming approach. It is built on top of `github.com/xuri/excelize/v2` and specifically leverages the `StreamWriter` interface to ensure that data is written directly to the output stream in batches, rather than being held in memory.

## Why v3?

While `simpleexcelv2` provides a high-level, feature-rich API for complex report generation, it often requires the entire dataset to be loaded into memory before processing. `v3` addresses use cases where:
- Datasets are too large for memory (millions of rows).
- Data is processed in long-running pipelines (e.g., `pkg/dataflow`) and should be written as soon as it's available.
- Minimal latency is required for the first bytes of the Excel file.

## Architecture

The package is designed around two core concepts:

1.  **`StreamExporter`**: Manages the lifecycle of the Excel file and the underlying `io.Writer`. It coordinates multiple sheets.
2.  **`StreamSheet`**: Wraps the `excelize.StreamWriter`. It maintains the current row state and handles the translation from Go objects/maps to Excel rows.

### Internal Logic Flow
- **Initialization**: `NewStreamExporter` initializes an `excelize.File`.
- **Sheet Creation**: `AddSheet` creates a new `StreamWriter` for a specific sheet.
- **Header**: `WriteHeader` sets column widths and writes the frozen top row.
- **Data Streaming**: `WriteRow` and `WriteBatch` use reflection to extract values and write them immediately to the `StreamWriter`.
- **Finalization**: `Close` flushes all internal buffers and writes the finalized ZIP structure to the provided `io.Writer`.

## Usage Example

```go
exporter := simpleexcelv3.NewStreamExporter(writer)
sheet, _ := exporter.AddSheet("Main Report")

cols := []simpleexcelv3.ColumnConfig{
    {FieldName: "Name", Header: "Full Name", Width: 30},
    {FieldName: "Score", Header: "Exam Score", Width: 15},
}
sheet.WriteHeader(cols)

// Streaming individual records
sheet.WriteRow(Student{Name: "Alice", Score: 95})

// Streaming batches from a channel or dataflow
sheet.WriteBatch(batchOfStudents)

exporter.Close()
```

## AI Refactoring Guidelines

To ensure this package remains stable when refactored by other models:

1.  **State Management**: `StreamSheet` relies on `currentRow` for coordinate calculation. Never skip or decrement this counter during a stream.
2.  **Reflection**: The `extractValue` function handles both `struct` fields and `map` keys. If extending type support, ensure `reflect.Value` safety checks (e.g., `IsValid()`) are maintained.
3.  **Flush Requirement**: Always ensure `Flush()` is called on every `StreamWriter` within the `Exporter.Close()` method. Failure to do so will result in truncated files.
4.  **No Random Access**: By definition, `StreamWriter` does not support modifying cells once written. Do not attempt to use `SetCellValue` on older rows.
5.  **Coordinate Mapping**: Use `excelize.CoordinatesToCellName` for all cell addressing to avoid hardcoding "A1" style logic.

## Comparison with v2

| Feature | simpleexcelv2 | simpleexcelv3 |
| :--- | :--- | :--- |
| **Memory Usage** | High (Buffering) | Low (Streaming) |
| **Complex Layouts** | Supported (Vertical/Horizontal) | Limited (Sequential) |
| **Formula Support** | Advanced | Row-relative only |
| **Best For** | Rich reports & UI-driven exports | Big Data & Pipeline integration |
