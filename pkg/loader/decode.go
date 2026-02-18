package loader

import (
	"fmt"
	"reflect"
)

// TryDecode attempts to parse a string value as structured data (JWT, JSON,
// YAML, TOML, NDJSON). It returns the decoded structure and true if the string
// contains serialized data that parses into a map or slice. Plain strings,
// numbers, and other scalars return (nil, false).
//
// This function lives in pkg/loader (not internal/ui) to avoid import cycles,
// since it depends on LoadRoot which has the full parsing pipeline.
func TryDecode(value string) (any, bool) {
	if value == "" {
		return nil, false
	}

	parsed, err := LoadRoot(value)
	if err != nil {
		return nil, false
	}

	// Only consider the decode successful if the result is structured data
	// (map or slice). If LoadRoot returns the same string back or a scalar,
	// the input wasn't really serialized data.
	if isStructured(parsed) {
		return parsed, true
	}

	return nil, false
}

// RecursiveDecode walks a data tree and replaces any string leaf that can
// be deserialized with its parsed structure. The replacement is applied
// recursively so that nested serialized strings are also expanded.
func RecursiveDecode(node any) any {
	return recursiveDecode(node, 0)
}

const maxDecodeDepth = 20

func recursiveDecode(node any, depth int) any {
	if depth > maxDecodeDepth {
		return node
	}

	switch v := node.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, val := range v {
			out[k] = recursiveDecode(val, depth+1)
		}
		return out

	case []any:
		out := make([]any, len(v))
		for i, val := range v {
			out[i] = recursiveDecode(val, depth+1)
		}
		return out

	case string:
		if decoded, ok := TryDecode(v); ok {
			// Recurse into the decoded structure in case it contains
			// further serialized strings.
			return recursiveDecode(decoded, depth+1)
		}
		return v

	default:
		// Handle typed maps and slices via reflection (e.g., map[string]string, []string)
		return recursiveDecodeReflect(node, depth)
	}
}

// recursiveDecodeReflect handles typed containers (map[K]V, []T) that don't match
// the common map[string]any / []any cases. It uses reflection to iterate and
// decode string values, converting to map[string]any / []any as needed.
func recursiveDecodeReflect(node any, depth int) any {
	if node == nil {
		return nil
	}

	rv := reflect.ValueOf(node)
	//exhaustive:ignore // We only care about containers (Map, Slice, Array, Ptr, Interface)
	switch rv.Kind() {
	case reflect.Map:
		out := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			k := iter.Key()
			// Convert key to string if possible
			var keyStr string
			if k.Kind() == reflect.String {
				keyStr = k.String()
			} else {
				// Non-string keys: use fmt representation
				keyStr = fmt.Sprintf("%v", k.Interface())
			}
			val := iter.Value().Interface()
			out[keyStr] = recursiveDecode(val, depth+1)
		}
		return out

	case reflect.Slice, reflect.Array:
		out := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out[i] = recursiveDecode(rv.Index(i).Interface(), depth+1)
		}
		return out

	case reflect.Ptr, reflect.Interface:
		if rv.IsNil() {
			return nil
		}
		return recursiveDecode(rv.Elem().Interface(), depth+1)

	default:
		return node
	}
}

// isStructured reports whether v is a map or slice (i.e. a navigable structure).
func isStructured(v any) bool {
	if v == nil {
		return false
	}
	switch v.(type) {
	case map[string]any, []any:
		return true
	}
	rv := reflect.ValueOf(v)
	kind := rv.Kind()
	return kind == reflect.Map || kind == reflect.Slice
}
