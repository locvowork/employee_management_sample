package simpleexcelv3

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestData struct {
	Name string
	Age  int
}

func TestStreamExporter(t *testing.T) {
	buf := new(bytes.Buffer)
	exporter := NewStreamExporter(buf)

	sheet, err := exporter.AddSheet("TestSheet")
	assert.NoError(t, err)

	cols := []ColumnConfig{
		{FieldName: "Name", Header: "Name", Width: 20},
		{FieldName: "Age", Header: "Age", Width: 10},
	}
	err = sheet.WriteHeader(cols)
	assert.NoError(t, err)

	// Individual row
	err = sheet.WriteRow(TestData{Name: "Alice", Age: 30})
	assert.NoError(t, err)

	// Batch write
	batch := []TestData{
		{Name: "Bob", Age: 25},
		{Name: "Charlie", Age: 35},
	}
	err = sheet.WriteBatch(batch)
	assert.NoError(t, err)

	// Map data
	mapBatch := []map[string]interface{}{
		{"Name": "David", "Age": 40},
		{"Name": "Eve", "Age": 45},
	}
	err = sheet.WriteBatch(mapBatch)
	assert.NoError(t, err)

	err = exporter.Close()
	assert.NoError(t, err)

	assert.True(t, buf.Len() > 0)
}
