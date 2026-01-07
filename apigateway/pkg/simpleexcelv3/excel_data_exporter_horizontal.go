package simpleexcelv3

import (
	"fmt"
	"io"
	"reflect"

	"github.com/xuri/excelize/v2"
)

// HorizontalSectionConfig configures a horizontal section
type HorizontalSectionConfig struct {
	ID           string
	Data         interface{} // Will be converted to DataProvider
	Columns      []ColumnConfigV3
	Title        interface{}
	ShowHeader   bool
}

// HorizontalStreamer manages horizontal streaming operations
type HorizontalStreamer struct {
	exporter        *ExcelDataExporterV3
	file            *excelize.File
	interleavedWriter *InterleavedStreamWriter
	writer          io.Writer
}

// StartHorizontalStream initializes horizontal streaming with multiple sections
func (e *ExcelDataExporterV3) StartHorizontalStream(w io.Writer, sections ...*HorizontalSectionConfig) (*HorizontalStreamer, error) {
	if len(sections) == 0 {
		return nil, fmt.Errorf("at least one section is required")
	}
	
	// Create file and sheet
	f := excelize.NewFile()
	sheetName := "Sheet1"
	f.SetSheetName("Sheet1", sheetName)
	
	// Create horizontal sections
	horizontalSections := make([]*HorizontalSection, len(sections))
	for i, config := range sections {
		// Create DataProvider from config data
		provider, err := e.createDataProvider(config.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to create data provider for section %s: %w", config.ID, err)
		}
		
		horizontalSections[i] = &HorizontalSection{
			ID:           config.ID,
			DataProvider: provider,
			Columns:      config.Columns,
			Title:        config.Title,
			ShowHeader:   config.ShowHeader,
			RowCount:     0, // Will be determined by DataProvider
			HasMoreRows:  true,
			CurrentRow:   0,
			StyleCache:   make(map[string]int),
		}
	}
	
	// Create coordinator
	coordinator := NewHorizontalSectionCoordinator(horizontalSections, FillStrategyPad)
	
	// Create interleaved stream writer
	interleavedWriter, err := NewInterleavedStreamWriter(f, sheetName, coordinator)
	if err != nil {
		return nil, err
	}
	
	return &HorizontalStreamer{
		exporter:        e,
		file:            f,
		interleavedWriter: interleavedWriter,
		writer:          w,
	}, nil
}

// createDataProvider creates appropriate DataProvider based on data type
func (e *ExcelDataExporterV3) createDataProvider(data interface{}) (DataProvider, error) {
	if data == nil {
		return nil, fmt.Errorf("data cannot be nil")
	}
	
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	switch v.Kind() {
	case reflect.Slice:
		return NewSliceDataProvider(data)
	case reflect.Chan:
		// Convert channel to ChannelDataProvider
		// This would need to handle the channel type properly
		return nil, fmt.Errorf("channel data provider not yet implemented")
	default:
		return NewSliceDataProvider(data)
	}
}

// WriteAllRows writes all data rows
func (s *HorizontalStreamer) WriteAllRows() error {
	return s.interleavedWriter.WriteAllRows()
}

// Flush flushes the stream writer
func (s *HorizontalStreamer) Flush() error {
	return s.interleavedWriter.streamWriter.Flush()
}

// Close closes the streamer and writes the file
func (s *HorizontalStreamer) Close() error {
	if err := s.Flush(); err != nil {
		return err
	}
	
	if _, err := s.file.WriteTo(s.writer); err != nil {
		return err
	}
	
	return nil
}

// StreamMode maintains backward compatibility
type StreamMode int

const (
	StreamModeVertical StreamMode = iota
	StreamModeHorizontal
)

// StartStreamV3WithMode provides enhanced streaming with mode selection
func (e *ExcelDataExporterV3) StartStreamV3WithMode(w io.Writer, mode StreamMode) (*StreamerV3, error) {
	switch mode {
	case StreamModeVertical:
		// Use existing implementation
		return e.startStreamV3Vertical(w)
	case StreamModeHorizontal:
		return nil, fmt.Errorf("horizontal mode requires StartHorizontalStream")
	default:
		return e.startStreamV3Vertical(w)
	}
}

// startStreamV3Vertical contains the existing vertical streaming logic
func (e *ExcelDataExporterV3) startStreamV3Vertical(w io.Writer) (*StreamerV3, error) {
	// Copy existing StartStreamV3 implementation here
	// This ensures backward compatibility
	f := excelize.NewFile()
	streamer := &StreamerV3{
		exporter:      e,
		file:          f,
		writer:        w,
		streamWriters: make(map[string]*excelize.StreamWriter),
	}
	
	// Initialize sheets (existing logic)
	for i, sb := range e.sheets {
		sheetName := sb.name
		if i == 0 {
			f.SetSheetName("Sheet1", sheetName)
		} else {
			f.NewSheet(sheetName)
		}
		
		sw, err := f.NewStreamWriter(sheetName)
		if err != nil {
			return nil, fmt.Errorf("failed to create stream writer for sheet %s: %w", sheetName, err)
		}
		streamer.streamWriters[sheetName] = sw
	}
	
	streamer.currentSheetIndex = 0
	streamer.currentSectionIndex = 0
	streamer.currentRow = 1
	
	if err := streamer.advanceToNextStreamingSection(); err != nil {
		return nil, err
	}
	
	return streamer, nil
}