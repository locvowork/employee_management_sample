package simpleexcelv3

import (
	"fmt"
	"io"
	"reflect"

	"github.com/xuri/excelize/v2"
)

// StreamExporter manages a streaming Excel export session.
type StreamExporter struct {
	file   *excelize.File
	writer io.Writer
	sheets map[string]*StreamSheet
}

// NewStreamExporter creates a new StreamExporter.
func NewStreamExporter(w io.Writer) *StreamExporter {
	return &StreamExporter{
		file:   excelize.NewFile(),
		writer: w,
		sheets: make(map[string]*StreamSheet),
	}
}

// StreamSheet represents a single sheet in a streaming export.
type StreamSheet struct {
	exporter    *StreamExporter
	stream      *excelize.StreamWriter
	name        string
	columns     []ColumnConfig
	currentRow  int
	headerShown bool
}

// AddSheet adds a new sheet and returns a StreamSheet builder.
func (e *StreamExporter) AddSheet(name string) (*StreamSheet, error) {
	if _, ok := e.sheets[name]; ok {
		return nil, fmt.Errorf("sheet %s already exists", name)
	}

	index, err := e.file.GetSheetIndex(name)
	if index == -1 {
		index, err = e.file.NewSheet(name)
		if err != nil {
			return nil, err
		}
	}

	sw, err := e.file.NewStreamWriter(name)
	if err != nil {
		return nil, err
	}

	sheet := &StreamSheet{
		exporter:   e,
		stream:     sw,
		name:       name,
		currentRow: 1,
	}
	e.sheets[name] = sheet
	return sheet, nil
}

// WriteHeader writes the header row for the sheet.
func (s *StreamSheet) WriteHeader(columns []ColumnConfig) error {
	s.columns = columns
	header := make([]interface{}, len(columns))
	for i, col := range columns {
		header[i] = col.Header

		// Set column width if specified
		if col.Width > 0 {
			_ = s.stream.SetColWidth(i+1, i+1, col.Width)
		}
	}

	cell, _ := excelize.CoordinatesToCellName(1, s.currentRow)
	if err := s.stream.SetRow(cell, header); err != nil {
		return err
	}
	s.currentRow++
	s.headerShown = true
	return nil
}

// WriteRow writes a single data row.
func (s *StreamSheet) WriteRow(item interface{}) error {
	if !s.headerShown {
		return fmt.Errorf("header must be written before data")
	}

	row := make([]interface{}, len(s.columns))
	v := reflect.ValueOf(item)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	for i, col := range s.columns {
		val := extractValue(v, col.FieldName)
		if col.Formatter != nil {
			val = col.Formatter(val)
		}
		row[i] = val
	}

	cell, _ := excelize.CoordinatesToCellName(1, s.currentRow)
	if err := s.stream.SetRow(cell, row); err != nil {
		return err
	}
	s.currentRow++
	return nil
}

// WriteBatch writes a slice of data as multiple rows.
func (s *StreamSheet) WriteBatch(slice interface{}) error {
	v := reflect.ValueOf(slice)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Slice {
		return fmt.Errorf("WriteBatch expects a slice, got %T", slice)
	}

	for i := 0; i < v.Len(); i++ {
		if err := s.WriteRow(v.Index(i).Interface()); err != nil {
			return err
		}
	}
	return nil
}

// Close finalizes all stream writers and writes the file to the output writer.
func (e *StreamExporter) Close() error {
	for _, sheet := range e.sheets {
		if err := sheet.stream.Flush(); err != nil {
			return err
		}
	}

	// Remove default Sheet1 if it wasn't used/renamed
	if _, ok := e.sheets["Sheet1"]; !ok {
		_ = e.file.DeleteSheet("Sheet1")
	}

	return e.file.Write(e.writer)
}

func extractValue(item reflect.Value, fieldName string) interface{} {
	if item.Kind() == reflect.Struct {
		f := item.FieldByName(fieldName)
		if f.IsValid() {
			return f.Interface()
		}
	} else if item.Kind() == reflect.Map {
		val := item.MapIndex(reflect.ValueOf(fieldName))
		if val.IsValid() {
			return val.Interface()
		}
	}
	return ""
}
