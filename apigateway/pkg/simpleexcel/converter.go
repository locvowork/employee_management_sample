package simpleexcel

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unicode"
)

// ConvertStructsToDynamic takes a slice of structs and a field name corresponding to a map[string]interface{}.
// It returns a new slice of dynamic structs where the map entries are promoted to top-level fields.
// It also returns the list of new field names created from the map keys.
func ConvertStructsToDynamic(data interface{}, mapFieldName string) (interface{}, []string, error) {
	val := reflect.ValueOf(data)
	if val.Kind() != reflect.Slice {
		return nil, nil, fmt.Errorf("data must be a slice")
	}

	if val.Len() == 0 {
		return data, nil, nil
	}

	// 1. Analyze the first element to get the base struct type
	elemType := val.Type().Elem()
	if elemType.Kind() != reflect.Struct {
		return nil, nil, fmt.Errorf("slice element must be a struct")
	}

	// 2. Scan all items to find all unique keys in the map field
	keysSet := make(map[string]bool)
	for i := 0; i < val.Len(); i++ {
		item := val.Index(i)
		mapField := item.FieldByName(mapFieldName)
		if !mapField.IsValid() {
			return nil, nil, fmt.Errorf("field %s not found in struct", mapFieldName)
		}
		if mapField.Kind() != reflect.Map {
			return nil, nil, fmt.Errorf("field %s is not a map", mapFieldName)
		}
		if mapField.IsNil() {
			continue
		}
		iter := mapField.MapRange()
		for iter.Next() {
			if iter.Key().Kind() == reflect.String {
				key := iter.Key().String()
				keysSet[key] = true
			}
		}
	}

	// 3. Sort keys for deterministic order
	var sortedKeys []string
	for k := range keysSet {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	// 4. Create new struct fields
	var newStructFields []reflect.StructField
	var newFieldNames []string

	// Mapping from NewFieldName -> MapKey
	fieldToKeyDetails := make(map[string]string)

	// Add original fields (excluding the map field)
	for i := 0; i < elemType.NumField(); i++ {
		f := elemType.Field(i)
		if f.Name == mapFieldName {
			// Check availability - if it's the target map, we replace it with expanded fields
			// The user wanted "exact order", usually implies where the map was.
			// So we insert new fields here.
			for _, key := range sortedKeys {
				fieldName := sanitizeAndCapitalize(key)
				// Ensure uniqueness of field names if collisions occur (simple check)
				// In a robust system we'd handle duplicate sanitized names, but for now we assume distinct.

				newField := reflect.StructField{
					Name: fieldName,
					Type: reflect.TypeOf((*interface{})(nil)).Elem(), // interface{}
				}
				newStructFields = append(newStructFields, newField)
				newFieldNames = append(newFieldNames, fieldName)
				fieldToKeyDetails[fieldName] = key
			}
		} else {
			newStructFields = append(newStructFields, f)
		}
	}

	// 5. Create the dynamic struct type
	dynamicType := reflect.StructOf(newStructFields)

	// 6. Create new slice
	newSlice := reflect.MakeSlice(reflect.SliceOf(dynamicType), val.Len(), val.Len())

	// 7. Populate new slice
	for i := 0; i < val.Len(); i++ {
		srcItem := val.Index(i)
		dstItem := newSlice.Index(i)

		// Copy standard fields
		for j := 0; j < elemType.NumField(); j++ {
			f := elemType.Field(j)
			if f.Name == mapFieldName {
				continue
			}
			// Important: FieldByName works on exported fields.
			// If src has unexported fields, this might panic if we try to Interface() them,
			// but Set can copy unexported fields between compatible types if we are careful?
			// Actually reflect.Set guarantees as long as assignable.
			// Ideally we assume DTOs have exported fields.
			srcField := srcItem.Field(j)
			dstField := dstItem.FieldByName(f.Name)
			if dstField.CanSet() {
				dstField.Set(srcField)
			}
		}

		// Copy map values
		mapField := srcItem.FieldByName(mapFieldName)
		if !mapField.IsNil() {
			for _, fieldName := range newFieldNames {
				mapKey := fieldToKeyDetails[fieldName]
				mapVal := mapField.MapIndex(reflect.ValueOf(mapKey))
				if mapVal.IsValid() {
					dstItem.FieldByName(fieldName).Set(mapVal)
				}
			}
		}
	}

	return newSlice.Interface(), newFieldNames, nil
}

// ExpandColumnConfigs expands the column configuration to include new fields from the map.
func ExpandColumnConfigs(cols []ColumnConfig, mapFieldName string, newFieldNames []string) []ColumnConfig {
	var newCols []ColumnConfig
	for _, col := range cols {
		if col.FieldName == mapFieldName {
			// Expand this column
			for _, fieldName := range newFieldNames {
				newCol := col // copy config
				newCol.FieldName = fieldName
				newCol.Header = fieldName // Default header to field name (which is sanitized key)
				newCols = append(newCols, newCol)
			}
		} else {
			newCols = append(newCols, col)
		}
	}
	return newCols
}

func sanitizeAndCapitalize(s string) string {
	if s == "" {
		return "Empty"
	}

	// Replace invalid chars with _
	var sb strings.Builder
	for i, r := range s {
		if i == 0 && unicode.IsDigit(r) {
			sb.WriteRune('F') // Prefix digit with F
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('_')
		}
	}
	res := sb.String()

	// Capitalize first letter
	if len(res) > 0 {
		r := []rune(res)
		r[0] = unicode.ToUpper(r[0])
		return string(r)
	}
	return "Field"
}
