package simpleexcelv3

import (
	"bytes"
	"fmt"
	"testing"
)

// Test data structures
type TestData struct {
	Name  string
	Value int
}

// TestDataProvider tests the DataProvider interface implementations
func TestDataProvider(t *testing.T) {
	// Test data
	testData := []TestData{
		{Name: "Alice", Value: 100},
		{Name: "Bob", Value: 200},
		{Name: "Charlie", Value: 300},
	}

	// Test SliceDataProvider
	t.Run("SliceDataProvider", func(t *testing.T) {
		provider, err := NewSliceDataProvider(testData)
		if err != nil {
			t.Fatalf("Failed to create SliceDataProvider: %v", err)
		}

		// Test GetRow
		row0, err := provider.GetRow(0)
		if err != nil {
			t.Errorf("GetRow(0) failed: %v", err)
		}
		if row0 == nil {
			t.Error("GetRow(0) returned nil")
		}

		// Test GetRowCount
		count, known := provider.GetRowCount()
		if !known {
			t.Error("GetRowCount should be known for slice data")
		}
		if count != 3 {
			t.Errorf("Expected count 3, got %d", count)
		}

		// Test HasMoreRows
		if !provider.HasMoreRows() {
			t.Error("HasMoreRows should return true initially")
		}

		// Test Close
		err = provider.Close()
		if err != nil {
			t.Errorf("Close failed: %v", err)
		}
	})

	// Test ChannelDataProvider
	t.Run("ChannelDataProvider", func(t *testing.T) {
		dataChan := make(chan interface{}, 3)
		go func() {
			dataChan <- TestData{Name: "Stream1", Value: 1}
			dataChan <- TestData{Name: "Stream2", Value: 2}
			close(dataChan)
		}()

		provider := NewChannelDataProvider(dataChan)

		// Test GetRow - data should be available immediately since we're using buffered channel
		row0, err := provider.GetRow(0)
		if err != nil {
			t.Errorf("GetRow(0) failed: %v", err)
		}
		if row0 == nil {
			t.Error("GetRow(0) returned nil")
		}

		// Test GetRowCount (should be unknown until channel is closed)
		count, known := provider.GetRowCount()
		if known {
			t.Error("GetRowCount should be unknown for channel data initially")
		}
		if count != 0 {
			t.Errorf("Expected count 0, got %d", count)
		}

		// Test Close
		err = provider.Close()
		if err != nil {
			t.Errorf("Close failed: %v", err)
		}
	})
}

// TestHorizontalSectionCoordinator tests the section coordination logic
func TestHorizontalSectionCoordinator(t *testing.T) {
	// Test data
	sectionAData := []TestData{{Name: "A1", Value: 1}, {Name: "A2", Value: 2}}
	sectionBData := []TestData{{Name: "B1", Value: 10}, {Name: "B2", Value: 20}, {Name: "B3", Value: 30}}

	providerA, err := NewSliceDataProvider(sectionAData)
	if err != nil {
		t.Fatalf("Failed to create provider A: %v", err)
	}

	providerB, err := NewSliceDataProvider(sectionBData)
	if err != nil {
		t.Fatalf("Failed to create provider B: %v", err)
	}

	// Create sections
	sectionA := &HorizontalSection{
		ID: "section_a",
		DataProvider: providerA,
		Columns: []ColumnConfigV3{
			{FieldName: "Name", Header: "Name"},
			{FieldName: "Value", Header: "Value"},
		},
		RowCount: 2,
	}

	sectionB := &HorizontalSection{
		ID: "section_b",
		DataProvider: providerB,
		Columns: []ColumnConfigV3{
			{FieldName: "Name", Header: "Name"},
			{FieldName: "Value", Header: "Value"},
		},
		RowCount: 3,
	}

	// Create coordinator
	coordinator := NewHorizontalSectionCoordinator([]*HorizontalSection{sectionA, sectionB}, FillStrategyPad)

	// Test GetNextRowData
	t.Run("GetNextRowData", func(t *testing.T) {
		// Get first row
		rowData, err := coordinator.GetNextRowData()
		if err != nil {
			t.Errorf("GetNextRowData failed: %v", err)
		}
		if rowData == nil {
			t.Error("GetNextRowData returned nil")
		}
		if len(rowData.Cells) != 4 {
			t.Errorf("Expected 4 cells (2 from each section), got %d", len(rowData.Cells))
		}

		// Get second row
		rowData, err = coordinator.GetNextRowData()
		if err != nil {
			t.Errorf("GetNextRowData failed: %v", err)
		}
		if rowData == nil {
			t.Error("GetNextRowData returned nil")
		}

		// Get third row (A exhausted, B continues)
		rowData, err = coordinator.GetNextRowData()
		if err != nil {
			t.Errorf("GetNextRowData failed: %v", err)
		}
		if rowData == nil {
			t.Error("GetNextRowData returned nil")
		}

		// Get fourth row (should be EOF)
		rowData, err = coordinator.GetNextRowData()
		if err == nil {
			t.Error("Expected EOF error, but got nil")
		}
		if rowData != nil {
			t.Error("Expected nil rowData, but got data")
		}
	})
}

// TestHorizontalStreamingIntegration tests the complete horizontal streaming workflow
func TestHorizontalStreamingIntegration(t *testing.T) {
	// Test data - smaller dataset to avoid Excel row limits
	sectionAData := []TestData{{Name: "Alice", Value: 100}, {Name: "Bob", Value: 200}}
	sectionBData := []TestData{{Name: "Charlie", Value: 300}, {Name: "David", Value: 400}}

	// Create exporter
	exporter := NewExcelDataExporterV3V3()

	// Create horizontal sections
	configA := &HorizontalSectionConfig{
		ID: "section_a",
		Data: sectionAData,
		Columns: []ColumnConfigV3{
			{FieldName: "Name", Header: "Name"},
			{FieldName: "Value", Header: "Value"},
		},
		Title: "Section A",
		ShowHeader: true,
	}

	configB := &HorizontalSectionConfig{
		ID: "section_b",
		Data: sectionBData,
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
	if err != nil {
		t.Fatalf("StartHorizontalStream failed: %v", err)
	}
	defer streamer.Close()

	// Write all rows
	err = streamer.WriteAllRows()
	if err != nil {
		t.Errorf("WriteAllRows failed: %v", err)
	}

	// Verify output file
	if buf.Len() == 0 {
		t.Error("Output buffer is empty")
	}

	// Note: Full Excel file validation would require opening the file
	// For now, we just verify that we got some output
	t.Logf("Generated Excel file size: %d bytes", buf.Len())
}

// BenchmarkHorizontalStreaming compares horizontal vs vertical streaming performance
func BenchmarkHorizontalStreaming(b *testing.B) {
	// Create large test data
	largeData := make([]TestData, 10000)
	for i := range largeData {
		largeData[i] = TestData{
			Name:  fmt.Sprintf("Item%d", i),
			Value: i * 10,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		exporter := NewExcelDataExporterV3V3()

		config := &HorizontalSectionConfig{
			ID: "test_section",
			Data: largeData,
			Columns: []ColumnConfigV3{
				{FieldName: "Name", Header: "Name"},
				{FieldName: "Value", Header: "Value"},
			},
			Title: "Test Section",
			ShowHeader: true,
		}

		var buf bytes.Buffer
		streamer, err := exporter.StartHorizontalStream(&buf, config)
		if err != nil {
			b.Fatalf("StartHorizontalStream failed: %v", err)
		}

		err = streamer.WriteAllRows()
		if err != nil {
			b.Errorf("WriteAllRows failed: %v", err)
		}

		streamer.Close()
	}
}