package loader

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// LoadData loads structured data from a string, auto-detecting format.
// Supports:
// - JWT tokens (3-part base64url-encoded tokens)
// - Single JSON object/array
// - Newline-delimited JSON (NDJSON): one JSON object per line
// - YAML: single document or multi-document (separated by ---)
//
// All formats return an []interface{} where each element is a parsed document/object.
// For single-document inputs, the array contains one element.
func LoadData(input string) ([]interface{}, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty input")
	}

	// Check for JWT first (single-line, dot-separated base64url)
	if IsJWT(input) {
		return loadJWT(input)
	}

	// Try multi-document YAML first (most restrictive)
	if strings.Contains(input, "\n---") || strings.HasPrefix(input, "---") {
		return loadMultiDocYAML(input)
	}

	// Try newline-delimited JSON (check for multiple lines starting with '{' or '[')
	lines := strings.Split(input, "\n")
	if len(lines) > 1 {
		// Heuristic: if multiple lines and each looks like JSON, treat as NDJSON
		if isLikelyNDJSON(lines) {
			return loadNDJSON(input)
		}
	}

	// Check for TOML before JSON - TOML [section] headers look like JSON arrays
	// but are distinct (e.g., "[server]" vs "[1, 2, 3]")
	if isLikelyTOML(input) {
		return loadTOML(input)
	}

	// Fall back to single JSON object/array
	if strings.HasPrefix(input, "{") || strings.HasPrefix(input, "[") {
		return loadJSON(input)
	}

	// Fall back to single YAML document
	return loadYAML(input)
}

// LoadRoot parses input into a single root node. Multi-document inputs are returned as a slice.
func LoadRoot(input string) (interface{}, error) {
	results, err := LoadData(input)
	if err != nil {
		return nil, err
	}
	if len(results) == 1 {
		return results[0], nil
	}
	return results, nil
}

// LoadRootBytes parses input bytes into a single root node.
func LoadRootBytes(data []byte) (interface{}, error) {
	return LoadRoot(string(data))
}

// LoadFile reads a file and parses it into a single root node.
func LoadFile(path string) (interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadRootBytes(data)
}

// LoadObject accepts an already parsed object (maps, slices, structs, etc.).
// Strings and byte slices are parsed using the existing loaders for format detection.
// Custom structs are converted to maps via JSON marshaling to ensure CEL compatibility.
func LoadObject(value any) (interface{}, error) {
	if value == nil {
		return nil, fmt.Errorf("object input is nil")
	}

	rv := reflect.ValueOf(value)
	if (rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Slice || rv.Kind() == reflect.Map || rv.Kind() == reflect.Interface || rv.Kind() == reflect.Func || rv.Kind() == reflect.Chan) && rv.IsNil() {
		return nil, fmt.Errorf("object input is nil")
	}

	switch v := value.(type) {
	case string:
		return LoadRoot(v)
	case []byte:
		return LoadRootBytes(v)
	default:
		// Convert to JSON-compatible type via marshal/unmarshal for CEL compatibility.
		// This handles custom structs by converting them to maps, allowing CEL to
		// evaluate expressions without "unsupported conversion to ref.Val" errors.
		return normalizeCELType(value)
	}
}

// normalizeCELType converts arbitrary Go types to JSON-compatible types that CEL can handle.
// Custom structs are converted to maps via JSON marshaling. Standard types (maps, slices, primitives)
// are returned as-is for efficiency. Slices are recursively normalized.
func normalizeCELType(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	rv := reflect.ValueOf(value)
	kind := rv.Kind()

	// Handle pointer types - dereference once
	if kind == reflect.Ptr {
		if rv.IsNil() {
			return nil, nil
		}
		rv = rv.Elem()
		kind = rv.Kind()
	}

	// Standard JSON-compatible types that CEL can already handle
	switch kind {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.String:
		return value, nil
	case reflect.Slice, reflect.Array:
		// Recursively normalize slice elements to handle custom structs within slices
		length := rv.Len()
		normalized := make([]interface{}, length)
		for i := 0; i < length; i++ {
			elemVal := rv.Index(i).Interface()
			val, err := normalizeCELType(elemVal)
			if err != nil {
				return nil, fmt.Errorf("element [%d]: %w", i, err)
			}
			normalized[i] = val
		}
		return normalized, nil
	case reflect.Map:
		// Maps are handled by CEL
		return value, nil
	case reflect.Interface:
		// Recursively normalize interface values
		return normalizeCELType(rv.Interface())
	case reflect.Struct:
		// Custom structs need to be converted via JSON to be CEL-compatible
		// This respects struct tags (json, yaml, etc.)
		data, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("cannot marshal custom type to JSON: %w", err)
		}
		var result interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("cannot unmarshal to standard type: %w", err)
		}
		return result, nil
	case reflect.Chan, reflect.Func, reflect.Invalid, reflect.Complex64, reflect.Complex128, reflect.UnsafePointer, reflect.Pointer:
		// These types cannot be reliably converted, attempt JSON conversion as last resort
		data, err := json.Marshal(value)
		if err != nil {
			// If JSON marshaling fails, return the value as-is and let CEL attempt to handle it
			return value, err
		}
		var result interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			return value, err
		}
		return result, nil
	default:
		// This should not be reached if exhaustive checker is working correctly
		return nil, fmt.Errorf("unsupported type: %v", kind)
	}
}

// loadJSON parses a single JSON object or array and wraps it in []interface{}
func loadJSON(input string) ([]interface{}, error) {
	var data interface{}
	if err := json.Unmarshal([]byte(input), &data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return []interface{}{data}, nil
}

// loadYAML parses a single YAML document and wraps it in []interface{}
func loadYAML(input string) ([]interface{}, error) {
	var data interface{}
	if err := yaml.Unmarshal([]byte(input), &data); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}
	return []interface{}{data}, nil
}

// loadMultiDocYAML parses YAML with multiple documents (separated by ---) and returns []interface{}
func loadMultiDocYAML(input string) ([]interface{}, error) {
	var results []interface{}
	decoder := yaml.NewDecoder(strings.NewReader(input))

	for {
		var doc interface{}
		if err := decoder.Decode(&doc); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("invalid multi-document YAML: %w", err)
		}
		if doc != nil {
			results = append(results, doc)
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no documents found in multi-document YAML")
	}
	return results, nil
}

// loadNDJSON parses newline-delimited JSON and returns []interface{}
// Lines that are valid JSON objects become map/array elements.
// Lines that are not valid JSON are treated as plain strings.
func loadNDJSON(input string) ([]interface{}, error) {
	lines := strings.Split(input, "\n")
	results := make([]interface{}, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var obj interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			// If JSON parsing fails, treat the line as a plain string
			results = append(results, line)
			continue
		}
		results = append(results, obj)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no data found in input")
	}
	return results, nil
}

// isLikelyNDJSON heuristic: returns true if the input looks like newline-delimited JSON.
// Uses positive JSON matching: a majority of non-empty lines must start with '{' or '['
// to be classified as NDJSON. This prevents YAML files (which may have many bare list
// items like "- name" that lack colons) from being misclassified as NDJSON.
func isLikelyNDJSON(lines []string) bool {
	jsonCount := 0
	nonEmptyCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		nonEmptyCount++

		// Positive match: line looks like a JSON object or array
		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
			jsonCount++
		}
	}

	// Require multiple lines and a majority must look like JSON
	return nonEmptyCount > 1 && jsonCount > nonEmptyCount/2
}

// isLikelyTOML heuristic: returns true if the input looks like TOML.
// Detects TOML by looking for section headers [name] or key = value patterns
// that are distinct from YAML syntax.
func isLikelyTOML(input string) bool {
	lines := strings.Split(input, "\n")

	// Pattern for TOML section headers: [section] or [[array]]
	// Supports bare keys, quoted keys, and dotted keys:
	//   [server], [[items]], ["table name"], [database.credentials], [server."host.name"]
	// Excludes JSON arrays like [1, 2, 3] which have spaces/commas without quotes
	sectionPattern := regexp.MustCompile(`^\s*\[{1,2}(?:[a-zA-Z_][a-zA-Z0-9_-]*|"[^"]+"|'[^']+')+(?:\.(?:[a-zA-Z_][a-zA-Z0-9_-]*|"[^"]+"|'[^']+'))*\]{1,2}\s*$`)

	// Pattern for TOML key = value (not key: value which is YAML)
	// Supports bare keys, quoted keys, and dotted keys:
	//   name = "value", "table name" = "value", database.host = "localhost"
	keyValuePattern := regexp.MustCompile(`^\s*(?:[a-zA-Z_][a-zA-Z0-9_-]*|"[^"]+"|'[^']+')+(?:\.(?:[a-zA-Z_][a-zA-Z0-9_-]*|"[^"]+"|'[^']+'))*\s*=\s*.+$`)

	sectionCount := 0
	keyValueCount := 0
	nonEmptyCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		nonEmptyCount++

		if sectionPattern.MatchString(line) {
			sectionCount++
		}
		if keyValuePattern.MatchString(line) {
			keyValueCount++
		}
	}

	// Consider it TOML if we have sections, or if majority of lines are key=value
	if sectionCount > 0 {
		return true
	}
	if nonEmptyCount > 0 && keyValueCount > nonEmptyCount/2 {
		return true
	}
	return false
}

// loadTOML parses TOML content and wraps it in []interface{}
func loadTOML(input string) ([]interface{}, error) {
	var data interface{}
	if err := toml.Unmarshal([]byte(input), &data); err != nil {
		return nil, fmt.Errorf("invalid TOML: %w", err)
	}
	return []interface{}{data}, nil
}
