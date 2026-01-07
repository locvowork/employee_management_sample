# Comparison Handler Fixes Plan

## Overview

Fix `internal/handler/comparison_handler.go` to properly use `pkg/simpleexcelv2/excel_data_exporter.go` and `pkg/simpleexcelv2/streamer.go` components.

## Issues Identified

### 1. ExportWikiStreaming Function (Lines 180-241)

**Problems:**

- Line 194: `simpleexcelv2.NewStreamsststresExporter` - Typo in function name
- Line 195: `exporter.AddSheet("Wikipedia People")` - Should use `AddSheet` method
- Line 200: `simpleexcelv3.ColumnConfig` - Wrong package reference
- Line 205: `sheet.WriteBatch(people)` - Method doesn't exist on SheetBuilder
- Line 225: `sheet.WriteBatch(people)` - Same issue

**Fixes Needed:**

```go
// Replace lines 194-206 with:
exporter := simpleexcelv2.NewExcelDataExporter()
sheet := exporter.AddSheet("Wikipedia People")

// Create a streamer for writing data
streamer, err := exporter.StartStream(c.Response().Writer)
if err != nil {
    return err
}
defer streamer.Close()

// Configure the section for streaming
sheet.AddSection(&simpleexcelv2.SectionConfig{
    ID:         "wiki-data",
    Title:      "Wikipedia People Export (Streaming)",
    ShowHeader: true,
    Columns: []simpleexcelv2.ColumnConfig{
        {FieldName: "Name", Header: "Person Name", Width: 40},
        {FieldName: "URL", Header: "Wiki URL", Width: 60},
    },
})
```

### 2. ExportWikiStreamingV2 Function (Lines 265-328)

**Problems:**

- Line 294: `exporter.StartStream(c.Response().Writer)` - Should use `StartStream` method
- Line 318: `streamer.Write("wiki-data", people)` - Should use correct method

**Fixes Needed:**

```go
// Line 294 is already correct
// Line 318: streamer.Write("wiki-data", people) - This is correct
```

### 3. ExportMultiSectionStreamYAML Function (Lines 330-437)

**Problems:**

- Line 394: `exporter.StartStream(c.Response())` - Should use `StartStream` method

**Fixes Needed:**

```go
// Replace line 394:
streamer, err := exporter.StartStream(c.Response())
```

### 4. Import Statement Issues

**Problems:**

- Missing proper imports for simpleexcelv2 components

**Fixes Needed:**

```go
// Ensure these imports are present:
"github.com/locvowork/employee_management_sample/apigateway/pkg/simpleexcelv2"
```

## Implementation Steps

### Step 1: Fix ExportWikiStreaming Function

1. Replace the incorrect exporter creation
2. Use proper SheetBuilder methods
3. Implement streaming with Streamer interface
4. Fix column configuration references

### Step 2: Fix ExportWikiStreamingV2 Function

1. Verify StartStream method usage
2. Ensure Write method is used correctly
3. Check section ID references

### Step 3: Fix ExportMultiSectionStreamYAML Function

1. Fix StartStream method call
2. Ensure proper response writer usage

### Step 4: Update Import Statements

1. Verify all necessary imports are present
2. Remove any unused imports

### Step 5: Testing

1. Verify the code compiles
2. Test streaming functionality
3. Test YAML configuration loading

## Code Changes Summary

### ExportWikiStreaming Function Changes:

- Replace `NewStreamsststresExporter` with `NewExcelDataExporter`
- Use `AddSheet` method instead of direct sheet creation
- Implement proper streaming with `StartStream` and `Streamer`
- Fix column configuration to use `simpleexcelv2.ColumnConfig`

### ExportWikiStreamingV2 Function Changes:

- Verify `StartStream` method usage is correct
- Ensure `Write` method is used properly

### ExportMultiSectionStreamYAML Function Changes:

- Fix `StartStream` method call to use proper response writer

## Expected Outcome

After these fixes:

1. All functions will properly use the simpleexcelv2 package components
2. Streaming functionality will work correctly
3. YAML configuration loading will function properly
4. The code will compile without errors
5. Excel export functionality will be restored

## Testing Strategy

1. **Compilation Test**: Ensure the code compiles without errors
2. **Unit Test**: Test each function individually
3. **Integration Test**: Test the complete export flow
4. **Performance Test**: Verify streaming works for large datasets
