package simpleexcelv3

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestBatchWriting(t *testing.T) {
	// 1. Setup Exporter
	exporter := NewExcelDataExporterV3V3()
	sheet := exporter.AddSheet("StreamOps")

	// Title Section (Static)
	sheet.AddSection(&SectionConfigV3{
		Type:  SectionTypeV3TitleOnly,
		Title: "Batch Export Test",
	})

	// Data Section (Streaming)
	// We don't provide Data here, we will stream it.
	// ID is required to target the section during streaming.
	sheet.AddSection(&SectionConfigV3{
		ID:         "stream-data",
		ShowHeader: true,
		Columns: []ColumnConfigV3{
			{FieldName: "ID", Header: "ID", Width: 10},
			{FieldName: "Value", Header: "Value", Width: 30},
		},
	})

	// 2. Start Stream
	buf := new(bytes.Buffer)
	streamer, err := exporter.StartStreamV3(buf)
	if err != nil {
		t.Fatalf("StartStreamV3 failed: %v", err)
	}

	// 3. Write Batch 1
	type DataItem struct {
		ID    int
		Value string
	}
	batch1 := []DataItem{
		{1, "A"},
		{2, "B"},
	}
	if err := streamer.Write("stream-data", batch1); err != nil {
		t.Fatalf("Write batch 1 failed: %v", err)
	}

	// 4. Write Batch 2
	batch2 := []DataItem{
		{3, "C"},
		{4, "D"},
	}
	if err := streamer.Write("stream-data", batch2); err != nil {
		t.Fatalf("Write batch 2 failed: %v", err)
	}

	// 5. Close
	if err := streamer.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// 6. Verify Content
	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Failed to open generated excel: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows("StreamOps")
	if err != nil {
		t.Fatalf("GetRows failed: %v", err)
	}

	// Expected Rows:
	// Row 1: Batch Export Test (Title)
	// Row 2: ID, Value (Header)
	// Row 3: 1, A
	// Row 4: 2, B
	// Row 5: 3, C
	// Row 6: 4, D

	if len(rows) != 6 {
		t.Errorf("Expected 6 rows, got %d", len(rows))
	}

	if rows[0][0] != "Batch Export Test" {
		t.Errorf("Title incorrect, got %s", rows[0][0])
	}
	if rows[2][0] != "1" || rows[5][1] != "D" {
		t.Errorf("Data seemingly incorrect: %v", rows)
	}
}

func TestMultiSectionStreamYAML(t *testing.T) {
	// Replicating user's report config structure simplified
	yamlConfig := `
sheets:
  - name: "MultiStream"
    sections:
      - id: "editable"
        type: "full"
        title: "Editable Data"
        show_header: true
        direction: "vertical"
        columns:
          - header: "ID"
            field_name: "ID"
      
      - id: "original"
        type: "full"
        title: "Original Data"
        show_header: true
        direction: "vertical"
        columns:
          - header: "ID"
            field_name: "ID"

      - id: "comparison"
        type: "full"
        title: "Comparison"
        show_header: true
        direction: "vertical"
        source_sections: ["editable"]
        columns:
          - header: "Status"
            field_name: "Status" 
`
	// Note: using simple field_name for comparison to test rendering, not formulas here yet.

	exporter, err := NewExcelDataExporterV3V3FromYamlConfig(yamlConfig)
	if err != nil {
		t.Fatalf("NewExcelDataExporterV3V3FromYamlConfig failed: %v", err)
	}

	var buf bytes.Buffer
	streamer, err := exporter.StartStreamV3(&buf)
	if err != nil {
		t.Fatalf("StartStreamV3 failed: %v", err)
	}

	// Batch 1 -> Editable
	data1 := []struct {
		ID int
	}{{ID: 101}, {ID: 102}}
	if err := streamer.Write("editable", data1); err != nil {
		t.Fatalf("Write to editable batch 1 failed: %v", err)
	}

	// Batch 2 -> Editable (Append)
	data1b := []struct {
		ID int
	}{{ID: 103}}
	if err := streamer.Write("editable", data1b); err != nil {
		t.Fatalf("Write to editable batch 2 failed: %v", err)
	}

	// Batch 3 -> Original (New Section)
	data2 := []struct {
		ID int
	}{{ID: 201}, {ID: 202}}
	if err := streamer.Write("original", data2); err != nil {
		t.Fatalf("Write to original batch 1 failed: %v", err)
	}

	// Batch 4 -> Comparison (New Section)
	// Passing same data just to trigger rows
	data3 := []struct {
		ID     int
		Status string
	}{{ID: 101, Status: "Diff"}, {ID: 102, Status: "Same"}}
	if err := streamer.Write("comparison", data3); err != nil {
		t.Fatalf("Write to comparison batch 1 failed: %v", err)
	}

	if err := streamer.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify
	f, err := excelize.OpenReader(&buf)
	if err != nil {
		t.Fatalf("Failed to open generated excel: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows("MultiStream")
	if err != nil {
		t.Fatalf("GetRows failed: %v", err)
	}

	// Expected Layout:
	// 1. Title: Editable Data
	// 2. Header: ID
	// 3. 101
	// 4. 102
	// 5. 103 (Appended)
	// 6. Title: Original Data
	// 7. Header: ID
	// 8. 201
	// 9. 202
	// 10. Title: Comparison
	// 11. Header: Status
	// 12. Diff
	// 13. Same

	expectedRows := 13
	if len(rows) != expectedRows {
		// Print rows for debugging
		for i, r := range rows {
			t.Logf("Row %d: %v", i+1, r)
		}
		t.Fatalf("Expected %d rows, got %d", expectedRows, len(rows))
	}

	if rows[0][0] != "Editable Data" {
		t.Errorf("Row 1 expected 'Editable Data', got '%s'", rows[0][0])
	}
	if rows[5][0] != "Original Data" {
		t.Errorf("Row 6 expected 'Original Data', got '%s'", rows[5][0])
	}
	if rows[9][0] != "Comparison" {
		t.Errorf("Row 10 expected 'Comparison', got '%s'", rows[9][0])
	}
}
