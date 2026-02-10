package navigator

import (
	"reflect"
	"sort"

	"github.com/oakwood-commons/kvx/internal/formatter"
)

// ShapeKind describes the general structure of data.
type ShapeKind string

const (
	ShapeScalar           ShapeKind = "scalar"
	ShapeMap              ShapeKind = "map"
	ShapeArray            ShapeKind = "array"
	ShapeHomogeneousArray ShapeKind = "homogeneous_array" // Array of objects with consistent keys
)

// ShapeInfo describes the structure of data for rendering decisions.
type ShapeInfo struct {
	Kind   ShapeKind
	Fields []string // For homogeneous arrays: the common field names (sorted)
	Length int      // For arrays: number of elements
}

// DetectShape analyzes data and returns its structural characteristics.
func DetectShape(data any) ShapeInfo {
	if data == nil {
		return ShapeInfo{Kind: ShapeScalar}
	}

	rv := reflect.ValueOf(data)
	switch rv.Kind() { //nolint:exhaustive // only map and slice are structurally relevant
	case reflect.Map:
		return ShapeInfo{Kind: ShapeMap, Length: rv.Len()}

	case reflect.Slice, reflect.Array:
		length := rv.Len()
		if length == 0 {
			return ShapeInfo{Kind: ShapeArray, Length: 0}
		}

		// Check if it's a homogeneous array of objects
		isHomogeneous, fields := IsHomogeneousArray(data)
		if isHomogeneous {
			return ShapeInfo{
				Kind:   ShapeHomogeneousArray,
				Fields: fields,
				Length: length,
			}
		}
		return ShapeInfo{Kind: ShapeArray, Length: length}

	default:
		return ShapeInfo{Kind: ShapeScalar}
	}
}

// IsHomogeneousArray checks if data is an array where all elements are objects
// (maps) with the same set of keys. Returns true and the sorted list of
// common field names if so.
func IsHomogeneousArray(data any) (bool, []string) {
	rv := reflect.ValueOf(data)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return false, nil
	}

	length := rv.Len()
	if length == 0 {
		return false, nil
	}

	// Try type assertion first for common case
	if arr, ok := data.([]any); ok {
		return isHomogeneousInterfaceArray(arr)
	}
	if arr, ok := data.([]interface{}); ok {
		return isHomogeneousInterfaceArray(arr)
	}

	// Use reflection for other slice types
	return isHomogeneousReflectArray(rv)
}

func isHomogeneousInterfaceArray(arr []any) (bool, []string) {
	if len(arr) == 0 {
		return false, nil
	}

	// Check first element is a map
	firstMap, ok := toStringKeyMap(arr[0])
	if !ok {
		return false, nil
	}

	// Get keys from first element
	baseKeys := mapKeys(firstMap)
	if len(baseKeys) == 0 {
		return false, nil
	}

	// Check all subsequent elements have the same keys
	for i := 1; i < len(arr); i++ {
		elemMap, ok := toStringKeyMap(arr[i])
		if !ok {
			return false, nil
		}

		// Check key sets match
		if !sameKeySet(baseKeys, elemMap) {
			return false, nil
		}
	}

	sort.Strings(baseKeys)
	return true, baseKeys
}

func isHomogeneousReflectArray(rv reflect.Value) (bool, []string) {
	length := rv.Len()
	if length == 0 {
		return false, nil
	}

	// Check first element
	first := rv.Index(0)
	if !first.IsValid() {
		return false, nil
	}

	// Get interface value and check if it's a map
	firstMap, ok := toStringKeyMap(first.Interface())
	if !ok {
		return false, nil
	}

	baseKeys := mapKeys(firstMap)
	if len(baseKeys) == 0 {
		return false, nil
	}

	// Check remaining elements
	for i := 1; i < length; i++ {
		elem := rv.Index(i)
		if !elem.IsValid() {
			return false, nil
		}

		elemMap, ok := toStringKeyMap(elem.Interface())
		if !ok {
			return false, nil
		}

		if !sameKeySet(baseKeys, elemMap) {
			return false, nil
		}
	}

	sort.Strings(baseKeys)
	return true, baseKeys
}

// toStringKeyMap attempts to convert a value to a map with string keys.
func toStringKeyMap(v any) (map[string]any, bool) {
	if v == nil {
		return nil, false
	}

	// Direct type assertion
	if m, ok := v.(map[string]any); ok {
		return m, true
	}
	if m, ok := v.(map[string]interface{}); ok {
		return m, true
	}

	// Use reflection for other map types with string keys
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Map {
		return nil, false
	}
	if rv.Type().Key().Kind() != reflect.String {
		return nil, false
	}

	result := make(map[string]any, rv.Len())
	for _, key := range rv.MapKeys() {
		result[key.String()] = rv.MapIndex(key).Interface()
	}
	return result, true
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func sameKeySet(keys []string, m map[string]any) bool {
	if len(keys) != len(m) {
		return false
	}
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			return false
		}
	}
	return true
}

// ExtractColumnarData extracts data from a homogeneous array into columnar format.
// Returns the field names and rows of stringified values.
// If the array is not homogeneous, returns nil, nil.
func ExtractColumnarData(data any, fieldOrder []string) ([]string, [][]string) {
	isHomogeneous, fields := IsHomogeneousArray(data)
	if !isHomogeneous {
		return nil, nil
	}

	// Use provided field order, or fall back to detected fields
	columns := fields
	if len(fieldOrder) > 0 {
		// Filter to only include fields that exist
		columns = make([]string, 0, len(fieldOrder))
		fieldSet := make(map[string]bool, len(fields))
		for _, f := range fields {
			fieldSet[f] = true
		}
		for _, f := range fieldOrder {
			if fieldSet[f] {
				columns = append(columns, f)
			}
		}
		// Add any remaining fields not in fieldOrder
		for _, f := range fields {
			found := false
			for _, fo := range fieldOrder {
				if fo == f {
					found = true
					break
				}
			}
			if !found {
				columns = append(columns, f)
			}
		}
	}

	// Extract rows
	rv := reflect.ValueOf(data)
	rows := make([][]string, 0, rv.Len())

	for i := 0; i < rv.Len(); i++ {
		elem := rv.Index(i).Interface()
		m, _ := toStringKeyMap(elem)

		row := make([]string, len(columns))
		for j, col := range columns {
			if val, ok := m[col]; ok {
				row[j] = stringify(val)
			}
		}
		rows = append(rows, row)
	}

	return columns, rows
}

// stringify converts a value to a display string using the formatter package.
func stringify(v any) string {
	return formatter.Stringify(v)
}
