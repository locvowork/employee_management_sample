package simpleexcel

import (
	"reflect"
	"testing"
)

type TestSource struct {
	Name  string
	Value int
	Meta  map[string]interface{}
}

func TestConvertStructsToDynamic(t *testing.T) {
	data := []TestSource{
		{"A", 1, map[string]interface{}{"foo": "bar", "baz": 123}},
		{"B", 2, map[string]interface{}{"foo": "qux", "extra": true}},
		{"C", 3, nil},
	}

	result, newFields, err := ConvertStructsToDynamic(data, "Meta")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify newFields
	expectedFields := []string{"Baz", "Extra", "Foo"} // keys sorted: baz, extra, foo -> Capitalized
	if !reflect.DeepEqual(newFields, expectedFields) {
		t.Errorf("expected fields %v, got %v", expectedFields, newFields)
	}

	// Verify result is a slice
	resVal := reflect.ValueOf(result)
	if resVal.Kind() != reflect.Slice {
		t.Fatalf("expected slice, got %v", resVal.Kind())
	}
	if resVal.Len() != 3 {
		t.Fatalf("expected 3 items, got %d", resVal.Len())
	}

	// Verify Item 0
	item0 := resVal.Index(0)
	// Check standard fields
	if item0.FieldByName("Name").String() != "A" {
		t.Errorf("Item 0 Name mismatch")
	}
	if item0.FieldByName("Value").Int() != 1 {
		t.Errorf("Item 0 Value mismatch")
	}
	// Check dynamic fields
	if item0.FieldByName("Foo").Interface() != "bar" {
		t.Errorf("Item 0 Foo mismatch")
	}
	if item0.FieldByName("Baz").Interface() != 123 {
		t.Errorf("Item 0 Baz mismatch")
	}
	if item0.FieldByName("Extra").Interface() != nil {
		t.Errorf("Item 0 Extra mismatch, expected nil")
	}

	// Verify Item 1
	item1 := resVal.Index(1)
	if item1.FieldByName("Name").String() != "B" {
		t.Errorf("Item 1 Name mismatch")
	}
	if item1.FieldByName("Foo").Interface() != "qux" {
		t.Errorf("Item 1 Foo mismatch")
	}
	if item1.FieldByName("Extra").Interface() != true {
		t.Errorf("Item 1 Extra mismatch")
	}

	// Verify Item 2 (Nil map)
	item2 := resVal.Index(2)
	if item2.FieldByName("Name").String() != "C" {
		t.Errorf("Item 2 Name mismatch")
	}
	if item2.FieldByName("Foo").Interface() != nil {
		t.Errorf("Item 2 Foo mismatch, expected nil")
	}
}

func TestConvertStructsToDynamic_Sanitization(t *testing.T) {
	data := []TestSource{
		{"A", 1, map[string]interface{}{"123key": "val", "bad space": "val"}},
	}

	_, newFields, err := ConvertStructsToDynamic(data, "Meta")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 123key -> F123key
	// bad space -> Bad_space
	// Sorted keys: "123key" (starts with 1), "bad space" (starts with b). 1 < b.
	// So F123key comes first.
	expected := []string{"F123key", "Bad_space"}
	if !reflect.DeepEqual(newFields, expected) {
		t.Errorf("expected fields %v, got %v", expected, newFields)
	}
}

func TestExpandColumnConfigs(t *testing.T) {
	locked := true
	cols := []ColumnConfig{
		{FieldName: "Name", Header: "Name"},
		{FieldName: "Meta", Header: "Meta Data", Width: 20, Locked: &locked},
		{FieldName: "Value", Header: "Value"},
	}

	newFields := []string{"Foo", "Bar"}
	expanded := ExpandColumnConfigs(cols, "Meta", newFields)

	if len(expanded) != 4 {
		t.Errorf("expected 4 columns, got %d", len(expanded))
	}

	// Check order
	if expanded[0].FieldName != "Name" {
		t.Errorf("col 0 mismatch")
	}

	// Expanded columns
	// Foo
	if expanded[1].FieldName != "Foo" {
		t.Errorf("col 1 FieldName mismatch")
	}
	if expanded[1].Header != "Foo" {
		t.Errorf("col 1 Header mismatch")
	}
	if expanded[1].Width != 20 {
		t.Errorf("col 1 Width mismatch")
	}
	if expanded[1].Locked == nil || *expanded[1].Locked != true {
		t.Errorf("col 1 Locked mismatch")
	}

	// Bar
	if expanded[2].FieldName != "Bar" {
		t.Errorf("col 2 FieldName mismatch")
	}
	if expanded[2].Width != 20 {
		t.Errorf("col 2 Width mismatch")
	}

	// Value
	if expanded[3].FieldName != "Value" {
		t.Errorf("col 3 mismatch")
	}
}
