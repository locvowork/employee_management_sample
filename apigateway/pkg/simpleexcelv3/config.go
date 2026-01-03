package simpleexcelv3

// ColumnConfig defines a column in a section.
// Kept similar to v2 for consistency.
type ColumnConfig struct {
	FieldName     string                        `json:"field_name"`
	Header        string                        `json:"header"`
	Width         float64                       `json:"width"`
	Formatter     func(interface{}) interface{} `json:"-"`
	FormatterName string                        `json:"formatter"`
}

// Style configuration (simplified for streaming)
type StyleConfig struct {
	Bold bool
}
