package simpleexcelv3

import (
	"fmt"
	"io"
	"reflect"

	"github.com/xuri/excelize/v2"
)

// StreamerV3 manages a streaming export session.
type StreamerV3 struct {
	exporter *ExcelDataExporterV3
	file     *excelize.File
	writer   io.Writer
	// streamWriters holds active stream writers for each sheet
	streamWriters map[string]*excelize.StreamWriter
	// currentSheetIndex tracks which sheet we are currently processing
	currentSheetIndex int
	// currentSectionIndex tracks which section we are in within the current sheet
	currentSectionIndex int
	// currentRow keeps track of the current row number for the current sheet
	currentRow int
	// sectionStarted indicates whether the current section's title/header has been written
	sectionStarted bool
}

// Write appends a batch of data to the specified section.
// The sectionID must match the ID of the current section or a future section.
// Strict ordering is enforced: you must write to sections in the order they are defined.
func (s *StreamerV3) Write(sectionID string, data interface{}) error {
	// 1. Validation
	if s.file == nil {
		return fmt.Errorf("stream is closed or not initialized")
	}

	sheet := s.getCurrentSheet()
	if sheet == nil {
		return fmt.Errorf("no active sheet to write to")
	}

	// Find the target section index
	targetIndex := -1
	for i := s.currentSectionIndex; i < len(sheet.sections); i++ {
		if sheet.sections[i].ID == sectionID {
			targetIndex = i
			break
		}
	}

	if targetIndex == -1 {
		return fmt.Errorf("section '%s' not found in remaining sections of sheet '%s' (already passed or does not exist)", sectionID, sheet.name)
	}

	sw := s.streamWriters[sheet.name]

	// 2. Advance if needed
	if targetIndex > s.currentSectionIndex {
		// We are moving to a new section.
		// Iterate through sections we are leaving/skipping.
		for i := s.currentSectionIndex; i < targetIndex; i++ {
			sec := sheet.sections[i]
			// If we are leaving the current section and we already started it (wrote Title/Header),
			// we don't need to do anything (data provided manually).
			// If we skipped it (sectionStarted == false), we render it as static (Title/Header only potentially).
			if i == s.currentSectionIndex && s.sectionStarted {
				// Just leaving.
			} else {
				// Skipping or Static section.
				if err := s.renderStaticSection(sw, sec); err != nil {
					return err
				}
			}
		}
		s.currentSectionIndex = targetIndex
		s.sectionStarted = false
	}

	// 3. Current Section
	sec := sheet.sections[s.currentSectionIndex]

	// 4. Resolve Columns (once if not done)
	initialWrite := false
	if len(sec.Columns) == 0 || (len(sec.Columns) > 0 && len(sec.Columns[0].FieldName) == 0) {
		// Dynamic discovery needed
		sec.Columns = mergeColumns(data, sec.Columns)
		initialWrite = true
	} else if !s.sectionStarted {
		// Columns exist but we haven't started this section (haven't written title/header)
		initialWrite = true
	}

	// 5. Render Title & Header (Lazy)
	if initialWrite {
		s.sectionStarted = true

		// Render Title
		if sec.Title != nil {
			cell, _ := excelize.CoordinatesToCellName(1, s.currentRow)
			defaultTitleOnly := &StyleTemplateV3{
				Font:      &FontTemplateV3{Bold: true},
				Alignment: &AlignmentTemplate{Horizontal: "center", Vertical: "top"},
			}
			styleTmpl := resolveStyle(sec.TitleStyle, defaultTitleOnly, sec.Locked)
			sid, err := s.exporter.createStyle(s.file, styleTmpl)
			if err != nil {
				return err
			}

			colSpan := sec.ColSpan
			if colSpan <= 0 {
				colSpan = len(sec.Columns)
			}
			if colSpan < 1 {
				colSpan = 1
			}

			if err := sw.SetRow(cell, []interface{}{
				excelize.Cell{Value: sec.Title, StyleID: sid},
			}); err != nil {
				return err
			}
			if colSpan > 1 {
				endCell, _ := excelize.CoordinatesToCellName(colSpan, s.currentRow)
				sw.MergeCell(cell, endCell)
			}
			s.currentRow++
		}

		// Render Header
		if sec.ShowHeader && len(sec.Columns) > 0 {
			cell, _ := excelize.CoordinatesToCellName(1, s.currentRow)
			headers := make([]interface{}, len(sec.Columns))
			for i, col := range sec.Columns {
				defaultHeader := &StyleTemplateV3{
					Font:      &FontTemplateV3{Bold: true},
					Alignment: &AlignmentTemplate{Horizontal: "center", Vertical: "top"},
				}
				styleTmpl := resolveStyle(sec.HeaderStyle, defaultHeader, col.IsLocked(sec.Locked))
				sid, err := s.exporter.createStyle(s.file, styleTmpl)
				if err != nil {
					return err
				}
				headers[i] = excelize.Cell{Value: col.Header, StyleID: sid}
				if col.Width > 0 {
					sw.SetColWidth(i+1, i+1, col.Width)
				}
			}
			if err := sw.SetRow(cell, headers); err != nil {
				return err
			}
			s.currentRow++
		}

		// REGISTER METADATA
		// Now s.currentRow is where data starts.
		fieldOffsets := make(map[string]int)
		for j, col := range sec.Columns {
			fieldOffsets[col.FieldName] = j
		}
		// Storing SectionPlacement for formula resolution
		s.exporter.sectionMetadata[sec.ID] = SectionPlacement{
			SectionID:    sec.ID,
			StartRow:     s.currentRow, // Current stream row is the data start row
			StartCol:     1,            // StreamerV3 always starts at col 1 for now
			FieldOffsets: fieldOffsets,
			DataLen:      0, // Unknown/Irrelevant for streaming lookup
		}
	}

	// 6. Write Data Rows
	return s.writeBatch(sw, sec, data)
}

// Close finishes the specified section (if any active) and moves to the next.
// ... (comments kept as is or removed for brevity) ...

// Close finishes the stream and writes the file to the output.
func (s *StreamerV3) Close() error {
	// Finish current sheet
	if err := s.finishCurrentSheet(); err != nil {
		return err
	}

	// Flush all stream writers
	for _, sw := range s.streamWriters {
		if err := sw.Flush(); err != nil {
			return err
		}
	}

	// Write entire file to output
	if _, err := s.file.WriteTo(s.writer); err != nil {
		return err
	}

	return nil
}

// finishCurrentSheet finishes processing the current sheet (render remaining static sections)
func (s *StreamerV3) finishCurrentSheet() error {
	// Process remaining sections in current sheet
	sheet := s.getCurrentSheet()
	if sheet == nil {
		return nil
	}

	for s.currentSectionIndex < len(sheet.sections) {
		idxStart := s.currentSectionIndex
		if err := s.advanceToNextStreamingSection(); err != nil {
			return err
		}
		if s.currentSectionIndex == idxStart {
			s.currentSectionIndex++
		}
	}
	return nil
}

func (s *StreamerV3) getCurrentSheet() *SheetBuilderV3 {
	if s.currentSheetIndex >= len(s.exporter.sheets) {
		return nil
	}
	return s.exporter.sheets[s.currentSheetIndex]
}

// advanceToNextStreamingSection renders all static sections until it hits a section
// that expects streaming data.
func (s *StreamerV3) advanceToNextStreamingSection() error {
	sheet := s.getCurrentSheet()
	if sheet == nil {
		return nil
	}

	sw := s.streamWriters[sheet.name]

	for s.currentSectionIndex < len(sheet.sections) {
		sec := sheet.sections[s.currentSectionIndex]

		isStatic := false
		if sec.Data != nil {
			isStatic = true
		} else if sec.ID != "" {
			if data, ok := s.exporter.data[sec.ID]; ok {
				sec.Data = data
				isStatic = true
			}
		} else {
			isStatic = true
		}

		if !isStatic {
			// Found a streaming section!
			s.sectionStarted = false
			return nil
		}

		// Render Static Section
		if err := s.renderStaticSection(sw, sec); err != nil {
			return err
		}

		s.currentSectionIndex++
	}

	if s.currentSectionIndex >= len(sheet.sections) {
		s.currentSheetIndex++
		s.currentSectionIndex = 0
		s.currentRow = 1
		return s.advanceToNextStreamingSection()
	}

	return nil
}

func (s *StreamerV3) renderStaticSection(sw *excelize.StreamWriter, sec *SectionConfigV3) error {
	// 1. Title
	if sec.Title != nil {
		cell, _ := excelize.CoordinatesToCellName(1, s.currentRow)
		defaultTitleOnly := &StyleTemplateV3{
			Font:      &FontTemplateV3{Bold: true},
			Alignment: &AlignmentTemplate{Horizontal: "center", Vertical: "top"},
		}
		styleTmpl := resolveStyle(sec.TitleStyle, defaultTitleOnly, sec.Locked)
		sid, err := s.exporter.createStyle(s.file, styleTmpl)
		if err != nil {
			return err
		}

		colSpan := sec.ColSpan
		if colSpan <= 0 {
			colSpan = len(sec.Columns)
		}
		if colSpan < 1 {
			colSpan = 1
		}

		if err := sw.SetRow(cell, []interface{}{
			excelize.Cell{Value: sec.Title, StyleID: sid},
		}); err != nil {
			return err
		}

		if colSpan > 1 {
			endCell, _ := excelize.CoordinatesToCellName(colSpan, s.currentRow)
			sw.MergeCell(cell, endCell)
		}
		s.currentRow++
	}

	// 2. Header
	if sec.ShowHeader && len(sec.Columns) > 0 {
		cell, _ := excelize.CoordinatesToCellName(1, s.currentRow)

		headers := make([]interface{}, len(sec.Columns))
		for i, col := range sec.Columns {
			defaultHeader := &StyleTemplateV3{
				Font:      &FontTemplateV3{Bold: true},
				Alignment: &AlignmentTemplate{Horizontal: "center", Vertical: "top"},
			}
			styleTmpl := resolveStyle(sec.HeaderStyle, defaultHeader, col.IsLocked(sec.Locked))
			sid, err := s.exporter.createStyle(s.file, styleTmpl)
			if err != nil {
				return err
			}
			headers[i] = excelize.Cell{Value: col.Header, StyleID: sid}
			if col.Width > 0 {
				sw.SetColWidth(i+1, i+1, col.Width)
			}
		}

		if err := sw.SetRow(cell, headers); err != nil {
			return err
		}
		s.currentRow++
	}

	// REGISTER METADATA for static sections too
	fieldOffsets := make(map[string]int)
	for j, col := range sec.Columns {
		fieldOffsets[col.FieldName] = j
	}
	s.exporter.sectionMetadata[sec.ID] = SectionPlacement{
		SectionID:    sec.ID,
		StartRow:     s.currentRow,
		StartCol:     1,
		FieldOffsets: fieldOffsets,
		DataLen:      0,
	}

	// 3. Data
	if sec.Data != nil {
		return s.writeBatch(sw, sec, sec.Data)
	}

	return nil
}

func (s *StreamerV3) writeBatch(sw *excelize.StreamWriter, sec *SectionConfigV3, data interface{}) error {
	// Resolve Columns
	if len(sec.Columns) == 0 {
		sec.Columns = mergeColumns(data, sec.Columns)
	}

	dataVal := reflect.ValueOf(data)
	if dataVal.Kind() == reflect.Ptr {
		dataVal = dataVal.Elem()
	}
	if dataVal.Kind() != reflect.Slice {
		return nil
	}

	// Prepare styles
	colStyles := make([]int, len(sec.Columns))
	for j, col := range sec.Columns {
		locked := col.IsLocked(sec.Locked)
		var defaultDataStyle *StyleTemplateV3
		if sec.Type == SectionTypeV3Hidden {
			defaultDataStyle = &StyleTemplateV3{Fill: &FillTemplate{Color: "FFFF00"}}
		}
		styleTmpl := resolveStyle(sec.DataStyle, defaultDataStyle, locked)
		sid, err := s.exporter.createStyle(s.file, styleTmpl)
		if err != nil {
			return err
		}
		colStyles[j] = sid
	}

	// Get metadata for formula resolution
	placement, hasMetadata := s.exporter.sectionMetadata[sec.ID]

	// Write rows
	for i := 0; i < dataVal.Len(); i++ {
		item := dataVal.Index(i)
		cell, _ := excelize.CoordinatesToCellName(1, s.currentRow)
		rowVals := make([]interface{}, len(sec.Columns))

		// Calculate rowOffset for formula.
		// If we have metadata, use (s.currentRow - placement.StartRow).
		rowOffset := 0
		if hasMetadata {
			rowOffset = s.currentRow - placement.StartRow
		}

		for j, col := range sec.Columns {
			if col.CompareWith != nil {
				// Generate Formula
				formula, err := s.exporter.generateDiffFormula(col, rowOffset)
				if err == nil {
					rowVals[j] = excelize.Cell{
						Formula: formula,
						StyleID: colStyles[j],
					}
				} else {
					rowVals[j] = excelize.Cell{
						Value:   fmt.Sprintf("Error: %v", err),
						StyleID: colStyles[j],
					}
				}
			} else {
				// Value Extraction
				val := s.exporter.extractValue(item, col.FieldName)
				if col.Formatter != nil {
					val = col.Formatter(val)
				} else if col.FormatterName != "" {
					if fmtFunc, ok := s.exporter.formatters[col.FormatterName]; ok {
						val = fmtFunc(val)
					}
				}
				rowVals[j] = excelize.Cell{
					Value:   val,
					StyleID: colStyles[j],
				}
			}
		}
		if err := sw.SetRow(cell, rowVals); err != nil {
			return err
		}
		s.currentRow++
	}
	return nil
}
