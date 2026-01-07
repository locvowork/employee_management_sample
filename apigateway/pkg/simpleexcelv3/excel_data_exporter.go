package simpleexcelv3

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/xuri/excelize/v2"
	"gopkg.in/yaml.v2"
)

// =============================================================================
// Constants & Types
// =============================================================================

const (
	SectionDirectionV3Horizontal = "horizontal"
	SectionDirectionV3Vertical   = "vertical"
	SectionTypeV3Full            = "full"   // Normal section with title, header, and data
	SectionTypeV3TitleOnly       = "title"  // Only display title
	SectionTypeV3Hidden          = "hidden" // Hidden section (row will be hidden)
	DefaultLockedColorV3         = "E0E0E0" // Light Gray for locked cells
)

// ExcelDataExporterV3 is the main entry point for exporting data.
type ExcelDataExporterV3 struct {
	template *ReportTemplate
	// data holds data bound to specific section IDs (for YAML flow)
	data map[string]interface{}
	// sheets holds manually added sheets (for programmatic flow)
	sheets []*SheetBuilderV3
	// formatters holds registered formatter functions by name
	formatters map[string]func(interface{}) interface{}

	// Metadata for coordinate mapping
	sectionMetadata map[string]SectionPlacement

	// Performance Caches
	styleCache   map[string]int
	colNameCache map[int]string
	fieldCache   map[fieldCacheKey]int
}

// fieldCacheKey is a unique key for caching field indices.
type fieldCacheKey struct {
	Type      reflect.Type
	FieldName string
}

// SectionPlacement stores the starting coordinates and metadata of a rendered section.
type SectionPlacement struct {
	SectionID    string
	StartRow     int
	StartCol     int
	FieldOffsets map[string]int // Map of FieldName to ColumnOffset (relative to startCol)
	DataLen      int            // Number of data rows
}

// ReportTemplate represents the YAML structure.
type ReportTemplate struct {
	Sheets []SheetTemplate `yaml:"sheets"`
}

// SheetTemplate represents a sheet in the YAML.
type SheetTemplate struct {
	Name     string          `yaml:"name"`
	Sections []SectionConfigV3 `yaml:"sections"`
}

// SectionConfigV3 defines a section of data in a sheet.
type SectionConfigV3 struct {
	ID             string         `yaml:"id"`
	Title          interface{}    `yaml:"title"`
	ColSpan        int            `yaml:"col_span"`        // Number of columns to span for title-only sections
	Data           interface{}    `yaml:"-"`               // Data is bound at runtime
	SourceSections []string       `yaml:"source_sections"` // IDs of sections this depends on
	Type           string         `yaml:"type"`            // "full", "title", "hidden"
	Locked         bool           `yaml:"locked"`          // Section-level lock (default for all columns)
	ShowHeader     bool           `yaml:"show_header"`
	Direction      string         `yaml:"direction"` // "horizontal" or "vertical"
	Position       string         `yaml:"position"`  // e.g., "A1"
	TitleStyle     *StyleTemplateV3 `yaml:"title_style"`
	HeaderStyle    *StyleTemplateV3 `yaml:"header_style"`
	DataStyle      *StyleTemplateV3 `yaml:"data_style"`
	TitleHeight    float64        `yaml:"title_height"`
	HeaderHeight   float64        `yaml:"header_height"`
	DataHeight     float64        `yaml:"data_height"`
	HasFilter      bool           `yaml:"has_filter"`
	Columns        []ColumnConfigV3 `yaml:"columns"`
}

// CompareConfig defines how to compare a column with another section.
type CompareConfig struct {
	SectionID string `yaml:"section_id"`
	FieldName string `yaml:"field_name"`
}

// ColumnConfigV3 defines a column in a section.
type ColumnConfigV3 struct {
	FieldName       string                        `yaml:"field_name"` // Struct field name or map key
	Header          string                        `yaml:"header"`
	Width           float64                       `yaml:"width"`
	Height          float64                       `yaml:"height"`
	Locked          *bool                         `yaml:"locked"`            // Column-level lock override (overrides section Locked)
	Formatter       func(interface{}) interface{} `yaml:"-"`                 // Optional custom formatter function (Programmatic)
	FormatterName   string                        `yaml:"formatter"`         // Name of registered formatter (YAML)
	HiddenFieldName string                        `yaml:"hidden_field_name"` // Hidden field name for backend use
	CompareWith     *CompareConfig                `yaml:"compare_with"`      // For injecting comparison formulas
	CompareAgainst  *CompareConfig                `yaml:"compare_against"`   // For injecting comparison formulas
}

// IsLocked returns whether this column should be locked.
// If column has explicit Locked setting, use that; otherwise use section default.
func (c *ColumnConfigV3) IsLocked(sectionLocked bool) bool {
	if c.Locked != nil {
		return *c.Locked
	}
	return sectionLocked
}

// StyleTemplateV3 defines basic styling.
type StyleTemplateV3 struct {
	Font      *FontTemplateV3      `yaml:"font"`
	Fill      *FillTemplate      `yaml:"fill"`
	Alignment *AlignmentTemplate `yaml:"alignment"`
	Locked    *bool              `yaml:"locked"`
}

type AlignmentTemplate struct {
	Horizontal string `yaml:"horizontal"` // center, left, right
	Vertical   string `yaml:"vertical"`   // top, center, bottom
}

type FontTemplateV3 struct {
	Bold  bool   `yaml:"bold"`
	Color string `yaml:"color"` // Hex color
}

type FillTemplate struct {
	Color string `yaml:"color"` // Hex color
}

// =============================================================================
// Constructors
// =============================================================================

func NewExcelDataExporterV3V3() *ExcelDataExporterV3 {
	return &ExcelDataExporterV3{
		data:            make(map[string]interface{}),
		sheets:          []*SheetBuilderV3{},
		formatters:      make(map[string]func(interface{}) interface{}),
		sectionMetadata: make(map[string]SectionPlacement),
		styleCache:      make(map[string]int),
		colNameCache:    make(map[int]string),
		fieldCache:      make(map[fieldCacheKey]int),
	}
}

func NewExcelDataExporterV3V3FromYamlConfig(yamlConfig string) (*ExcelDataExporterV3, error) {
	var tmpl ReportTemplate
	if yamlConfig == "" {
		return nil, fmt.Errorf("yaml config is empty")
	}
	if err := yaml.Unmarshal([]byte(yamlConfig), &tmpl); err != nil {
		return nil, fmt.Errorf("decode yaml: %w", err)
	}

	exporter := &ExcelDataExporterV3{
		template:        &tmpl,
		data:            make(map[string]interface{}),
		formatters:      make(map[string]func(interface{}) interface{}),
		sheets:          make([]*SheetBuilderV3, 0),
		sectionMetadata: make(map[string]SectionPlacement),
		styleCache:      make(map[string]int),
		colNameCache:    make(map[int]string),
		fieldCache:      make(map[fieldCacheKey]int),
	}

	// Initialize sheets from template
	for i := range tmpl.Sheets {
		sheetTmpl := &tmpl.Sheets[i]
		sb := &SheetBuilderV3{
			exporter: exporter,
			name:     sheetTmpl.Name,
			sections: make([]*SectionConfigV3, len(sheetTmpl.Sections)),
		}
		for j := range sheetTmpl.Sections {
			sb.sections[j] = &sheetTmpl.Sections[j]
		}
		exporter.sheets = append(exporter.sheets, sb)
	}

	return exporter, nil
}

// =============================================================================
// Fluent API
// =============================================================================

// AddSheet starts a new sheet builder.
func (e *ExcelDataExporterV3) AddSheet(name string) *SheetBuilderV3 {
	sb := &SheetBuilderV3{
		exporter: e,
		name:     name,
		sections: []*SectionConfigV3{},
	}
	e.sheets = append(e.sheets, sb)
	return sb
}

// BindSectionData binds data to a section ID (for YAML-based export).
func (e *ExcelDataExporterV3) BindSectionData(id string, data interface{}) *ExcelDataExporterV3 {
	e.data[id] = data
	return e
}

// RegisterFormatter registers a formatter function with a name.
// This allows referencing formatters by name in YAML configurations.
func (e *ExcelDataExporterV3) RegisterFormatter(name string, f func(interface{}) interface{}) *ExcelDataExporterV3 {
	e.formatters[name] = f
	return e
}

// GetSheet returns a SheetBuilderV3 by name, or nil if not found.
func (e *ExcelDataExporterV3) GetSheet(name string) *SheetBuilderV3 {
	for _, sheet := range e.sheets {
		if sheet.name == name {
			return sheet
		}
	}
	return nil
}

// GetSheetByIndex returns a SheetBuilderV3 by index (0-based), or nil if out of bounds.
func (e *ExcelDataExporterV3) GetSheetByIndex(index int) *SheetBuilderV3 {
	if index < 0 || index >= len(e.sheets) {
		return nil
	}
	return e.sheets[index]
}

// BuildExcel constructs an Excel file (*excelize.File) based on the exporter's configuration and data.
// It processes both programmatically added sheets and sheets defined in a YAML template,
// returning the generated excelize.File instance or an error// BuildExcel generates the excel file
func (e *ExcelDataExporterV3) BuildExcel() (*excelize.File, error) {
	f := excelize.NewFile()

	// Process All Sheets (both fluent and YAML-initialized are now in e.sheets)
	for i, sb := range e.sheets {
		sheetName := sb.name
		if i == 0 {
			f.SetSheetName("Sheet1", sheetName)
		} else {
			// Check if sheet exists to avoid error if duplicates (though logic shouldn't produce duplicates easily)
			idx, _ := f.GetSheetIndex(sheetName)
			if idx == -1 {
				f.NewSheet(sheetName)
			}
		}

		// Perform Late Binding for any section that has an ID and matching data in e.data
		for _, sec := range sb.sections {
			if sec.ID != "" {
				if data, ok := e.data[sec.ID]; ok {
					sec.Data = data
				}
			}
		}

		if err := e.renderSections(f, sheetName, sb.sections); err != nil {
			return nil, err
		}
	}

	return f, nil
}

// ExportToExcel generates the Excel file on disk.
func (e *ExcelDataExporterV3) ExportToExcel(ctx context.Context, path string) error {
	f, err := e.BuildExcel()
	if err != nil {
		return err
	}
	defer f.Close()
	return f.SaveAs(path)
}

// ToBytes exports the Excel file to an in-memory byte slice.
func (e *ExcelDataExporterV3) ToBytes() ([]byte, error) {
	f, err := e.BuildExcel()
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Create a buffer and write the Excel file to it
	buf := new(bytes.Buffer)
	if _, err := f.WriteTo(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// StartStreamV3 initializes a streaming export session.
// It returns a StreamerV3 which can be used to write data incrementally.
func (e *ExcelDataExporterV3) StartStreamV3(w io.Writer) (*StreamerV3, error) {
	// 1. Initialize File
	f := excelize.NewFile()
	streamer := &StreamerV3{
		exporter:      e,
		file:          f,
		writer:        w,
		streamWriters: make(map[string]*excelize.StreamWriter),
	}

	// 2. Prepare Sheets
	for i, sb := range e.sheets {
		sheetName := sb.name
		if i == 0 {
			f.SetSheetName("Sheet1", sheetName)
		} else {
			f.NewSheet(sheetName)
		}

		// Initialize StreamWriter for this sheet
		sw, err := f.NewStreamWriter(sheetName)
		if err != nil {
			return nil, fmt.Errorf("failed to create stream writer for sheet %s: %w", sheetName, err)
		}
		streamer.streamWriters[sheetName] = sw
	}

	// Prepare state
	streamer.currentSheetIndex = 0
	streamer.currentSectionIndex = 0
	streamer.currentRow = 1

	// Initial processing (render static sections of first sheet)
	if err := streamer.advanceToNextStreamingSection(); err != nil {
		return nil, err
	}

	return streamer, nil
}

// ToWriter exports the Excel file directly to a writer.
func (e *ExcelDataExporterV3) ToWriter(w io.Writer) error {
	f, err := e.BuildExcel()
	if err != nil {
		return err
	}
	defer f.Close()

	return f.Write(w)
}

// ToCSV exports the first sheet of data to CSV format.
// This is significantly more memory-efficient for very large datasets as it avoids Excel overhead.
func (e *ExcelDataExporterV3) ToCSV(w io.Writer) error {
	if len(e.sheets) == 0 {
		return fmt.Errorf("no sheets to export")
	}

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	sheet := e.sheets[0]
	for _, sec := range sheet.sections {
		// Perform Late Binding if needed
		if sec.ID != "" && sec.Data == nil {
			if data, ok := e.data[sec.ID]; ok {
				sec.Data = data
			}
		}

		// Get data length
		dataLen := e.getDataLength(sec)
		if dataLen == 0 && !sec.ShowHeader {
			continue
		}

		// Resolve columns
		cols := mergeColumns(sec.Data, sec.Columns)

		// Title (if single title only)
		if sec.Title != nil {
			_ = csvWriter.Write([]string{fmt.Sprintf("%v", sec.Title)})
		}

		// Header
		if sec.ShowHeader && len(cols) > 0 {
			headerArr := make([]string, len(cols))
			for i, col := range cols {
				headerArr[i] = col.Header
			}
			if err := csvWriter.Write(headerArr); err != nil {
				return err
			}
		}

		// Data
		if dataLen > 0 {
			v := reflect.ValueOf(sec.Data)
			if v.Kind() == reflect.Ptr {
				v = v.Elem()
			}

			for i := 0; i < dataLen; i++ {
				item := v.Index(i)
				rowArr := make([]string, len(cols))
				for j, col := range cols {
					val := e.extractValue(item, col.FieldName)
					// Apply formatter if any
					if col.Formatter != nil {
						val = col.Formatter(val)
					} else if col.FormatterName != "" && e.formatters != nil {
						if fn, ok := e.formatters[col.FormatterName]; ok {
							val = fn(val)
						}
					}
					rowArr[j] = fmt.Sprintf("%v", val)
				}
				if err := csvWriter.Write(rowArr); err != nil {
					return err
				}
			}
		}

		// Empty line between sections
		_ = csvWriter.Write([]string{""})
	}

	return nil
}

// =============================================================================
// SheetBuilderV3
// =============================================================================

type SheetBuilderV3 struct {
	exporter *ExcelDataExporterV3
	name     string
	sections []*SectionConfigV3
}

func (sb *SheetBuilderV3) AddSection(config *SectionConfigV3) *SheetBuilderV3 {
	sb.sections = append(sb.sections, config)
	return sb
}

func (sb *SheetBuilderV3) Build() *ExcelDataExporterV3 {
	return sb.exporter
}

// =============================================================================
// Rendering Logic
// =============================================================================

// hasHiddenFields returns true if any column in the section has a HiddenFieldName.
func hasHiddenFields(sec *SectionConfigV3) bool {
	for _, col := range sec.Columns {
		if col.HiddenFieldName != "" {
			return true
		}
	}
	return false
}

// calculatePosition returns the start coordinates for a section.
func calculatePosition(sec *SectionConfigV3, nextColHorizontal, maxRow int) (int, int) {
	if sec.Position != "" {
		c, r, err := excelize.CellNameToCoordinates(sec.Position)
		if err == nil {
			return c, r
		}
	}

	isHorizontal := sec.Direction == SectionDirectionV3Horizontal
	if isHorizontal {
		return nextColHorizontal, 1
	}
	return 1, maxRow
}

// getDataLength returns the expected number of data rows for a section.
func (e *ExcelDataExporterV3) getDataLength(sec *SectionConfigV3) int {
	dataVal := reflect.ValueOf(sec.Data)
	if dataVal.Kind() == reflect.Slice {
		return dataVal.Len()
	}
	if len(sec.SourceSections) > 0 {
		if sourcePlacement, ok := e.sectionMetadata[sec.SourceSections[0]]; ok {
			return sourcePlacement.DataLen
		}
	}
	return 0
}

func (e *ExcelDataExporterV3) renderSections(f *excelize.File, sheet string, sections []*SectionConfigV3) error {
	// --- PASS 1: Layout Calculation ---
	tempRow, tempCol := 1, 1
	maxRowForPass1 := 1

	placements := make([]SectionPlacement, len(sections))

	for i, sec := range sections {
		// Determine section type
		sectionType := sec.Type
		if sectionType == "" {
			sectionType = SectionTypeV3Full
		}

		// Determine effective columns merging user config and data fields
		sec.Columns = mergeColumns(sec.Data, sec.Columns)

		// Determine start coordinates
		sCol, sRow := calculatePosition(sec, tempCol, tempRow)

		// Calculate data start row by skipping Title, Hidden Row, and Header
		dataStartRow := sRow
		if sectionType != SectionTypeV3TitleOnly {
			if sec.Title != nil {
				dataStartRow++
			}
			if hasHiddenFields(sec) {
				dataStartRow++
			}
			if sec.ShowHeader {
				dataStartRow++
			}
		} else {
			if sec.Title != nil {
				dataStartRow++
			}
		}

		fieldOffsets := make(map[string]int)
		for j, col := range sec.Columns {
			fieldOffsets[col.FieldName] = j
		}

		// We need to know DataLen for Pass 1 to update tempRow/tempCol trackers accurately
		dataLen := e.getDataLength(sec)

		placements[i] = SectionPlacement{
			SectionID:    sec.ID,
			StartRow:     dataStartRow,
			StartCol:     sCol,
			FieldOffsets: fieldOffsets,
			DataLen:      dataLen,
		}

		if sec.ID != "" {
			e.sectionMetadata[sec.ID] = placements[i]
		}

		// Update global trackers for Pass 1 layout
		finishRow := dataStartRow + dataLen
		if finishRow > maxRowForPass1 {
			maxRowForPass1 = finishRow
		}
		if finishRow > tempRow {
			tempRow = finishRow // This is for vertical stacking logic if we were purely vertical
		}

		// For horizontal tracking
		colSpan := len(sec.Columns)
		if sectionType == SectionTypeV3TitleOnly {
			colSpan = sec.ColSpan
			if colSpan <= 1 && len(sec.Columns) > 1 {
				colSpan = len(sec.Columns)
			}
		}
		tempCol = sCol + colSpan
	}

	// --- PASS 2: Actual Rendering ---
	maxRow := 1
	nextColHorizontal := 1
	hasLockedCells := false
	hiddenRows := []int{}

	// Check for locked cells first (to decide if we need to unlock sheet)
	for _, sec := range sections {
		if sec.Locked {
			hasLockedCells = true
		} else {
			for _, col := range sec.Columns {
				if col.Locked != nil && *col.Locked {
					hasLockedCells = true
					break
				}
			}
		}
		if hasLockedCells {
			break
		}
	}

	if hasLockedCells {
		unlocked := false
		defaultStyle := &StyleTemplateV3{Locked: &unlocked}
		styleID, _ := e.createStyle(f, defaultStyle)
		f.SetColStyle(sheet, "A:XFD", styleID)
	}

	for i, sec := range sections {
		placement := placements[i]

		// Re-calculate sCol, sRow for Pass 2 (should match Pass 1)
		sCol, sRow := calculatePosition(sec, nextColHorizontal, maxRow)
		currentRow := sRow

		sectionType := sec.Type
		if sectionType == "" {
			sectionType = SectionTypeV3Full
		}

		// Handle Title Only
		if sectionType == SectionTypeV3TitleOnly {
			if sec.Title != nil {
				cell := e.getCellAddress(sCol, currentRow)
				f.SetCellValue(sheet, cell, sec.Title)
				defaultTitleOnly := &StyleTemplateV3{
					Font:      &FontTemplateV3{Bold: true},
					Alignment: &AlignmentTemplate{Horizontal: "center", Vertical: "top"},
				}
				style := resolveStyle(sec.TitleStyle, defaultTitleOnly, sec.Locked)
				styleID, _ := e.createStyle(f, style)
				colSpan := sec.ColSpan
				if colSpan <= 1 && len(sec.Columns) > 1 {
					colSpan = len(sec.Columns)
				}
				if colSpan > 1 {
					endCell, _ := excelize.CoordinatesToCellName(sCol+colSpan-1, currentRow)
					f.MergeCell(sheet, cell, endCell)
					f.SetCellStyle(sheet, cell, endCell, styleID)
				} else {
					f.SetCellStyle(sheet, cell, cell, styleID)
				}
				if sec.TitleHeight > 0 {
					f.SetRowHeight(sheet, currentRow, sec.TitleHeight)
				}
				currentRow++
			}
			if currentRow > maxRow {
				maxRow = currentRow
			}
			colSpan := sec.ColSpan
			if colSpan <= 1 && len(sec.Columns) > 1 {
				colSpan = len(sec.Columns)
			}
			if colSpan <= 1 {
				colSpan = 1
			}
			nextColHorizontal = sCol + colSpan
			continue
		}

		// Render Title
		if sec.Title != nil {
			cell := e.getCellAddress(sCol, currentRow)
			f.SetCellValue(sheet, cell, sec.Title)
			defaultTitle := &StyleTemplateV3{
				Font:      &FontTemplateV3{Bold: true},
				Alignment: &AlignmentTemplate{Horizontal: "center", Vertical: "top"},
			}
			style := resolveStyle(sec.TitleStyle, defaultTitle, sec.Locked)
			styleID, _ := e.createStyle(f, style)
			if len(sec.Columns) > 1 {
				endCell := e.getCellAddress(sCol+len(sec.Columns)-1, currentRow)
				f.MergeCell(sheet, cell, endCell)
				f.SetCellStyle(sheet, cell, endCell, styleID)
			} else {
				f.SetCellStyle(sheet, cell, cell, styleID)
			}
			if sec.TitleHeight > 0 {
				f.SetRowHeight(sheet, currentRow, sec.TitleHeight)
			}
			currentRow++
		}

		// Render Hidden Field Name Row
		if hasHiddenFields(sec) {
			locked := true
			hiddenStyle := &StyleTemplateV3{Fill: &FillTemplate{Color: "FFFF00"}, Locked: &locked}
			styleID, _ := e.createStyle(f, hiddenStyle)
			for i, col := range sec.Columns {
				cell := e.getCellAddress(sCol+i, currentRow)
				f.SetCellValue(sheet, cell, col.HiddenFieldName)
				f.SetCellStyle(sheet, cell, cell, styleID)
			}
			hiddenRows = append(hiddenRows, currentRow)
			currentRow++
		}

		// Render Header
		if sec.ShowHeader {
			for i, col := range sec.Columns {
				cell := e.getCellAddress(sCol+i, currentRow)
				f.SetCellValue(sheet, cell, col.Header)
				locked := col.IsLocked(sec.Locked)
				defaultHeader := &StyleTemplateV3{
					Font:      &FontTemplateV3{Bold: true},
					Alignment: &AlignmentTemplate{Horizontal: "center", Vertical: "top"},
				}
				style := resolveStyle(sec.HeaderStyle, defaultHeader, locked)
				styleID, _ := e.createStyle(f, style)
				f.SetCellStyle(sheet, cell, cell, styleID)
				if col.Width > 0 {
					colName := e.getColName(sCol + i)
					f.SetColWidth(sheet, colName, colName, col.Width)
				}
			}
			if sec.HeaderHeight > 0 {
				f.SetRowHeight(sheet, currentRow, sec.HeaderHeight)
			}
			currentRow++
		}

		// Pre-calculate style IDs and common values for data rendering
		dataStyleIDs := make([]int, len(sec.Columns))
		maxColHeight := sec.DataHeight

		for j, col := range sec.Columns {
			locked := col.IsLocked(sec.Locked)
			var defaultDataStyle *StyleTemplateV3
			if sectionType == SectionTypeV3Hidden {
				defaultDataStyle = &StyleTemplateV3{Fill: &FillTemplate{Color: "FFFF00"}}
			}
			style := resolveStyle(sec.DataStyle, defaultDataStyle, locked)
			styleID, _ := e.createStyle(f, style)
			dataStyleIDs[j] = styleID

			if col.Height > maxColHeight {
				maxColHeight = col.Height
			}
		}

		// Render Data
		dataLen := placement.DataLen // Use pre-calculated length
		dataVal := reflect.ValueOf(sec.Data)
		for i := 0; i < dataLen; i++ {
			var item reflect.Value
			if dataVal.Kind() == reflect.Slice && i < dataVal.Len() {
				item = dataVal.Index(i)
			}
			for j, col := range sec.Columns {
				cell := e.getCellAddress(sCol+j, currentRow)
				if col.CompareWith != nil {
					formula, err := e.generateDiffFormula(col, i)
					if err == nil {
						f.SetCellFormula(sheet, cell, formula)
					} else {
						f.SetCellValue(sheet, cell, fmt.Sprintf("Error: %v", err))
					}
				} else if item.IsValid() {
					val := e.extractValue(item, col.FieldName)
					if col.Formatter != nil {
						val = col.Formatter(val)
					} else if col.FormatterName != "" {
						if fmtFunc, ok := e.formatters[col.FormatterName]; ok {
							val = fmtFunc(val)
						}
					}
					f.SetCellValue(sheet, cell, val)
				}
				f.SetCellStyle(sheet, cell, cell, dataStyleIDs[j])
			}
			if maxColHeight > 0 {
				f.SetRowHeight(sheet, currentRow, maxColHeight)
			}
			currentRow++
		}

		// Apply AutoFilter if requested
		if sec.HasFilter && sec.ShowHeader && len(sec.Columns) > 0 {
			headerRow := sRow
			if sec.Title != nil {
				headerRow++
			}
			if hasHiddenFields(sec) {
				headerRow++
			}
			// headerRow is now the row index of the header

			firstCell := e.getCellAddress(sCol, headerRow)
			lastCell := e.getCellAddress(sCol+len(sec.Columns)-1, currentRow-1)
			filterRange := fmt.Sprintf("%s:%s", firstCell, lastCell)
			f.AutoFilter(sheet, filterRange, nil)
		}

		if sectionType == SectionTypeV3Hidden {
			for r := sRow; r < currentRow; r++ {
				hiddenRows = append(hiddenRows, r)
			}
		}

		if currentRow > maxRow {
			maxRow = currentRow
		}
		nextColHorizontal = sCol + len(sec.Columns)
	}

	for _, r := range hiddenRows {
		f.SetRowVisible(sheet, r, false)
	}

	if hasLockedCells {
		f.ProtectSheet(sheet, &excelize.SheetProtectionOptions{
			Password:            "",
			FormatCells:         false,
			FormatColumns:       true,
			FormatRows:          true,
			InsertColumns:       false,
			InsertRows:          false,
			InsertHyperlinks:    false,
			DeleteColumns:       false,
			DeleteRows:          false,
			Sort:                false,
			AutoFilter:          true,
			PivotTables:         false,
			SelectLockedCells:   true,
			SelectUnlockedCells: true,
		})
	}
	return nil
}

func (e *ExcelDataExporterV3) resolveCellAddress(sectionID, fieldName string, rowOffset int) (string, error) {
	placement, ok := e.sectionMetadata[sectionID]
	if !ok {
		return "", fmt.Errorf("section %s not found", sectionID)
	}

	colOffset, ok := placement.FieldOffsets[fieldName]
	if !ok {
		return "", fmt.Errorf("field %s not found in %s", fieldName, sectionID)
	}

	// StartRow in metadata should point to the first row of DATA
	return excelize.CoordinatesToCellName(placement.StartCol+colOffset, placement.StartRow+rowOffset)
}

func (e *ExcelDataExporterV3) generateDiffFormula(col ColumnConfigV3, rowOffset int) (string, error) {
	if col.CompareWith == nil {
		return "", nil
	}

	cellA, err := e.resolveCellAddress(col.CompareWith.SectionID, col.CompareWith.FieldName, rowOffset)
	if err != nil {
		return "", err
	}

	if col.CompareAgainst != nil {
		cellB, err := e.resolveCellAddress(col.CompareAgainst.SectionID, col.CompareAgainst.FieldName, rowOffset)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf(`IF(%s<>%s, "Diff", "")`, cellA, cellB), nil
	}

	// Default comparison is not specified in the plan but let's assume it compares with something else if CompareAgainst is nil?
	// The plan says: =IF(Editable_Cell <> Original_Cell, "Diff", "")
	// If only CompareWith is provided, maybe it's compared against the current section's field?
	// Let's re-read the plan.
	// Plan says:
	// cellA, _ := e.resolveCellAddress(col.CompareWith.SectionID, col.CompareWith.FieldName, i)
	// cellB, _ := e.resolveCellAddress(col.CompareAgainst.SectionID, col.CompareAgainst.FieldName, i)
	// formula := fmt.Sprintf(`IF(%s<>%s, "Diff", "")`, cellA, cellB)

	// If CompareAgainst is nil, we should return an error or handle it.
	return "", fmt.Errorf("CompareAgainst is required for comparison column %s", col.FieldName)
}

// resolveStyle merges defined style with default style and applies conditional locked styling.
func resolveStyle(base *StyleTemplateV3, defaultStyle *StyleTemplateV3, locked bool) *StyleTemplateV3 {
	s := &StyleTemplateV3{}

	// Apply default if base is nil
	if base == nil {
		if defaultStyle != nil {
			*s = *defaultStyle
		}
	} else {
		*s = *base
		// If base has no font but default does, apply default font (rudimentary merge)
		if s.Font == nil && defaultStyle != nil && defaultStyle.Font != nil {
			s.Font = defaultStyle.Font
		}
		// If base has no fill but default does, apply default fill
		if s.Fill == nil && defaultStyle != nil && defaultStyle.Fill != nil {
			s.Fill = defaultStyle.Fill
		}
		// If base has no alignment but default does, apply default alignment
		if s.Alignment == nil && defaultStyle != nil && defaultStyle.Alignment != nil {
			s.Alignment = defaultStyle.Alignment
		}
	}

	// Apply explicit lock override
	s.Locked = &locked

	// Auto-gray locked cells if no fill is explicitly set
	if locked && s.Fill == nil {
		s.Fill = &FillTemplate{Color: DefaultLockedColorV3}
	}

	return s
}

// getColName returns the column name for a given column number, with caching.
func (e *ExcelDataExporterV3) getColName(col int) string {
	if name, ok := e.colNameCache[col]; ok {
		return name
	}
	name, _ := excelize.ColumnNumberToName(col)
	e.colNameCache[col] = name
	return name
}

// getCellAddress returns the cell address for given coordinates, with caching.
func (e *ExcelDataExporterV3) getCellAddress(col, row int) string {
	colName := e.getColName(col)
	return fmt.Sprintf("%s%d", colName, row)
}

func (e *ExcelDataExporterV3) createStyle(f *excelize.File, tmpl *StyleTemplateV3) (int, error) {
	if tmpl == nil {
		return 0, nil
	}

	// Generate a unique key for this style
	var sb strings.Builder
	if tmpl.Font != nil {
		fmt.Fprintf(&sb, "f:%v:%s|", tmpl.Font.Bold, tmpl.Font.Color)
	}
	if tmpl.Fill != nil {
		fmt.Fprintf(&sb, "i:%s|", tmpl.Fill.Color)
	}
	if tmpl.Alignment != nil {
		fmt.Fprintf(&sb, "a:%s:%s|", tmpl.Alignment.Horizontal, tmpl.Alignment.Vertical)
	}
	if tmpl.Locked != nil {
		fmt.Fprintf(&sb, "l:%v|", *tmpl.Locked)
	}
	key := sb.String()

	if id, ok := e.styleCache[key]; ok {
		return id, nil
	}

	style := &excelize.Style{}
	if tmpl.Font != nil {
		style.Font = &excelize.Font{
			Bold:  tmpl.Font.Bold,
			Color: strings.TrimPrefix(tmpl.Font.Color, "#"),
		}
	}
	if tmpl.Fill != nil {
		style.Fill = excelize.Fill{
			Type:    "pattern",
			Color:   []string{strings.TrimPrefix(tmpl.Fill.Color, "#")},
			Pattern: 1,
		}
	}
	if tmpl.Alignment != nil {
		style.Alignment = &excelize.Alignment{
			Horizontal: tmpl.Alignment.Horizontal,
			Vertical:   tmpl.Alignment.Vertical,
		}
	}
	if tmpl.Locked != nil {
		style.Protection = &excelize.Protection{
			Locked: *tmpl.Locked,
		}
	}
	id, err := f.NewStyle(style)
	if err == nil {
		e.styleCache[key] = id
	}
	return id, err
}

func (e *ExcelDataExporterV3) extractValue(item reflect.Value, fieldName string) interface{} {
	if item.Kind() == reflect.Struct {
		t := item.Type()
		key := fieldCacheKey{Type: t, FieldName: fieldName}
		index, ok := e.fieldCache[key]
		if !ok {
			f, found := t.FieldByName(fieldName)
			if found {
				index = f.Index[0]
				e.fieldCache[key] = index
			} else {
				e.fieldCache[key] = -1 // Not found
				return ""
			}
		}

		if index != -1 {
			return item.Field(index).Interface()
		}
	} else if item.Kind() == reflect.Map {
		val := item.MapIndex(reflect.ValueOf(fieldName))
		if val.IsValid() {
			return val.Interface()
		}
	}
	return ""
}

// mergeColumns merges user-defined columns with detected fields from data.
// It prioritizes user-defined columns, then appends remaining detected fields.
func mergeColumns(data interface{}, userConfigs []ColumnConfigV3) []ColumnConfigV3 {
	if data == nil {
		return userConfigs
	}

	// 1. Detect all fields from data
	detectedFields := getFields(data)

	// 2. Index user configs by FieldName for O(1) lookup
	userConfigMap := make(map[string]ColumnConfigV3)
	seen := make(map[string]bool)
	var finalCols []ColumnConfigV3

	for _, col := range userConfigs {
		userConfigMap[col.FieldName] = col
		seen[col.FieldName] = true
		finalCols = append(finalCols, col)
	}

	// 3. Append detected fields that are not in user config
	for _, field := range detectedFields {
		if !seen[field] {
			// Create default config
			col := ColumnConfigV3{
				FieldName: field,
				Header:    field, // Default header is field name
				Width:     20,    // Default width
			}
			finalCols = append(finalCols, col)
			seen[field] = true
		}
	}

	return finalCols
}

func getFields(data interface{}) []string {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// If not a slice, return empty (single item support could be added but usually export is slice)
	if v.Kind() != reflect.Slice {
		// Try to handle single struct if passed?
		// For now assume slice as per assumed usage, or standard usage.
		// If it's a single struct, we can treat it as one item.
		if v.Kind() == reflect.Struct {
			return getStructFields(v.Type())
		}
		return nil
	}

	if v.Len() == 0 {
		return nil
	}

	// Inspect first element
	elem := v.Index(0)
	if elem.Kind() == reflect.Ptr {
		elem = elem.Elem()
	}

	if elem.Kind() == reflect.Struct {
		return getStructFields(elem.Type())
	} else if elem.Kind() == reflect.Map {
		// Collect keys from all maps? Or just first?
		// Collecting from all is safer but slower.
		// For simplicity and performance, start with Union of first generic 10 rows?
		// Let's do union of all rows to be safe as maps can vary.
		// Limit to max 100 rows scan to prevent performance perf hit on large datasets?
		// Or just first row as convention?
		// "simpleexcel" implies simplicity. First row is standard convention for schema sniffing in basic libs.
		// BUT user said "no matter what... apply default".
		// To be robust, let's scan up to 10 rows.

		keysMap := make(map[string]bool)
		var keys []string

		limit := v.Len()
		if limit > 50 {
			limit = 50
		}

		for i := 0; i < limit; i++ {
			row := v.Index(i)
			if row.Kind() == reflect.Ptr {
				row = row.Elem()
			}
			if row.Kind() == reflect.Map {
				for _, key := range row.MapKeys() {
					k := key.String()
					if !keysMap[k] {
						keysMap[k] = true
						keys = append(keys, k)
					}
				}
			}
		}
		return keys
	}

	return nil
}

func getStructFields(t reflect.Type) []string {
	var fields []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// Skip unexported
		if field.PkgPath != "" {
			continue
		}
		fields = append(fields, field.Name)
	}
	return fields
}
