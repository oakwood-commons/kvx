package loader

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-logr/logr"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// formatName is used for logging parse attempts.
type formatName string

const (
	fmtMultiDocYAML formatName = "multi-doc YAML"
	fmtNDJSON       formatName = "NDJSON"
	fmtTOML         formatName = "TOML"
	fmtJSON         formatName = "JSON"
	fmtYAML         formatName = "YAML"
)

var (
	// tomlSectionPattern matches TOML section headers: [section] or [[array]]
	// Must start at column 0 (no leading whitespace) — TOML section headers are
	// always top-level. This prevents matching indented YAML values like
	// ["legacy"] that appear inside multi-line scalars.
	// Supports bare keys, quoted keys, and dotted keys:
	//   [server], [[items]], ["table name"], [database.credentials], [server."host.name"]
	tomlSectionPattern = regexp.MustCompile(`^\[{1,2}(?:[a-zA-Z_][a-zA-Z0-9_-]*|"[^"]+"|'[^']+')+(?:\.(?:[a-zA-Z_][a-zA-Z0-9_-]*|"[^"]+"|'[^']+'))*\]{1,2}\s*$`)

	// tomlKeyValuePattern matches TOML key = value (not key: value which is YAML)
	// Must start at column 0 (no leading whitespace). Also requires the line
	// to start with a bare key or quoted key — not a YAML indicator like '-'.
	// Supports bare keys, quoted keys, and dotted keys:
	//   name = "value", "table name" = "value", database.host = "localhost"
	tomlKeyValuePattern = regexp.MustCompile(`^(?:[a-zA-Z_][a-zA-Z0-9_-]*|"[^"]+"|'[^']+')+(?:\.(?:[a-zA-Z_][a-zA-Z0-9_-]*|"[^"]+"|'[^']+'))*\s*=\s*.+$`)
)

// parseFunc is a parser that returns parsed data or an error.
type parseFunc func(string) ([]interface{}, error)

// candidate pairs a format name with a lazy parser. The parser is only
// invoked when the candidate is actually attempted.
type candidate struct {
	name  formatName
	parse parseFunc
}

// tryParsers attempts each candidate in order. On the first success it
// returns the result. On failure it logs the error at V(1) and continues.
// If all candidates fail, it returns the collected error messages.
func tryParsers(input string, candidates []candidate, lgr logr.Logger) ([]interface{}, error) {
	var errs []string
	for _, c := range candidates {
		result, err := c.parse(input)
		if err == nil {
			return result, nil
		}
		lgr.V(1).Info("parse attempt failed, trying next format",
			"format", string(c.name), "error", err.Error())
		errs = append(errs, fmt.Sprintf("%s: %s", c.name, err.Error()))
	}
	return nil, fmt.Errorf("all parsers failed:\n  %s", strings.Join(errs, "\n  "))
}

// LoadData loads structured data from a string, auto-detecting format.
// Supports:
// - JWT tokens (3-part base64url-encoded tokens)
// - Single JSON object/array
// - Newline-delimited JSON (NDJSON): one JSON object per line
// - YAML: single document or multi-document (separated by ---)
// - TOML
//
// All formats return an []interface{} where each element is a parsed document/object.
// For single-document inputs, the array contains one element.
//
// If the preferred format fails, remaining formats are attempted and
// failures are logged at verbosity level 1.
func LoadData(input string) ([]interface{}, error) {
	return LoadDataWithLogger(input, logr.Discard())
}

// LoadDataWithLogger is like LoadData but accepts a logger for
// recording fallback parse attempts.
func LoadDataWithLogger(input string, lgr logr.Logger) ([]interface{}, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty input")
	}

	// Check for JWT first (single-line, dot-separated base64url)
	if IsJWT(input) {
		return loadJWT(input)
	}

	// Build an ordered list of candidates based on heuristics.
	// The preferred (heuristic-detected) format goes first, followed
	// by all remaining formats as fallbacks.
	candidates := buildCandidates(input)

	return tryParsers(input, candidates, lgr)
}

// extToFormat maps common file extensions to format names.
func extToFormat(ext string) (formatName, bool) {
	switch strings.ToLower(ext) {
	case ".yaml", ".yml":
		return fmtYAML, true
	case ".json":
		return fmtJSON, true
	case ".toml":
		return fmtTOML, true
	case ".ndjson", ".jsonl":
		return fmtNDJSON, true
	default:
		return "", false
	}
}

// parsersForFormat returns the primary parser for a given format name,
// followed by all other parsers as fallbacks (in heuristic order).
func parsersForFormat(name formatName, input string) []candidate {
	// For YAML, decide between single-doc and multi-doc based on content.
	if name == fmtYAML {
		if strings.Contains(input, "\n---") || strings.HasPrefix(input, "---") {
			name = fmtMultiDocYAML
		}
	}

	primary := candidate{name: name, parse: parserFor(name)}

	// Build remaining candidates from heuristic order, excluding the primary.
	heuristic := buildCandidates(input)
	others := make([]candidate, 0, len(heuristic))
	for _, c := range heuristic {
		if c.name != name {
			others = append(others, c)
		}
	}

	return append([]candidate{primary}, others...)
}

// parserFor returns the parse function for a given format name.
func parserFor(name formatName) parseFunc {
	switch name {
	case fmtMultiDocYAML:
		return loadMultiDocYAML
	case fmtNDJSON:
		return loadNDJSON
	case fmtTOML:
		return loadTOML
	case fmtJSON:
		return loadJSON
	case fmtYAML:
		return loadYAML
	default:
		return loadYAML
	}
}

// buildCandidates returns an ordered list of format candidates based
// on content heuristics. The most-likely format appears first.
func buildCandidates(input string) []candidate {
	var primary []candidate
	used := map[formatName]bool{}

	// Multi-document YAML (most restrictive signal)
	if strings.Contains(input, "\n---") || strings.HasPrefix(input, "---") {
		primary = append(primary, candidate{fmtMultiDocYAML, loadMultiDocYAML})
		used[fmtMultiDocYAML] = true
	}

	// Newline-delimited JSON
	lines := strings.Split(input, "\n")
	if len(lines) > 1 && isLikelyNDJSON(lines) {
		primary = append(primary, candidate{fmtNDJSON, loadNDJSON})
		used[fmtNDJSON] = true
	}

	// TOML
	if isLikelyTOML(input) {
		primary = append(primary, candidate{fmtTOML, loadTOML})
		used[fmtTOML] = true
	}

	// Single JSON
	if strings.HasPrefix(input, "{") || strings.HasPrefix(input, "[") {
		primary = append(primary, candidate{fmtJSON, loadJSON})
		used[fmtJSON] = true
	}

	// Remaining formats as fallbacks in a stable order.
	// Note: NDJSON is excluded from unconditional fallbacks because loadNDJSON
	// never errors (it treats invalid JSON lines as plain strings), which would
	// silently accept malformed inputs and prevent stricter parsers from running.
	allFormats := []candidate{
		{fmtMultiDocYAML, loadMultiDocYAML},
		{fmtTOML, loadTOML},
		{fmtJSON, loadJSON},
		{fmtYAML, loadYAML},
	}
	for _, c := range allFormats {
		if !used[c.name] {
			primary = append(primary, c)
		}
	}
	return primary
}

// LoadRoot parses input into a single root node. Multi-document inputs are returned as a slice.
func LoadRoot(input string) (interface{}, error) {
	return LoadRootWithLogger(input, logr.Discard())
}

// LoadRootWithLogger is like LoadRoot but accepts a logger.
func LoadRootWithLogger(input string, lgr logr.Logger) (interface{}, error) {
	results, err := LoadDataWithLogger(input, lgr)
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

// LoadRootBytesWithLogger is like LoadRootBytes but accepts a logger.
func LoadRootBytesWithLogger(data []byte, lgr logr.Logger) (interface{}, error) {
	return LoadRootWithLogger(string(data), lgr)
}

// LoadFile reads a file and parses it into a single root node.
// When the file extension is recognized (.yaml, .yml, .json, .toml, .ndjson,
// .jsonl) the corresponding parser is tried first. If it fails, the
// remaining parsers are attempted as fallbacks and failures are logged.
func LoadFile(path string) (interface{}, error) {
	return LoadFileWithLogger(path, logr.Discard())
}

// LoadFileWithLogger is like LoadFile but accepts a logger.
func LoadFileWithLogger(path string, lgr logr.Logger) (interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	input := string(data)

	// Honor the file extension first.
	ext := filepath.Ext(path)
	if fmtName, ok := extToFormat(ext); ok {
		lgr.V(1).Info("file extension detected, trying preferred format",
			"ext", ext, "format", string(fmtName))
		candidates := parsersForFormat(fmtName, input)
		results, parseErr := tryParsers(input, candidates, lgr)
		if parseErr != nil {
			return nil, parseErr
		}
		if len(results) == 1 {
			return results[0], nil
		}
		return results, nil
	}

	// No recognized extension — fall back to content heuristics.
	return LoadRootWithLogger(input, lgr)
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

	sectionCount := 0
	keyValueCount := 0
	nonEmptyCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		nonEmptyCount++

		if tomlSectionPattern.MatchString(line) {
			sectionCount++
		}
		if tomlKeyValuePattern.MatchString(line) {
			keyValueCount++
		}
	}

	// Section headers at column 0 are a strong TOML signal (the pattern
	// rejects indented YAML values like ["legacy"] that appear in multi-line scalars).
	if sectionCount > 0 {
		return true
	}
	// Key-value-only TOML (no section headers): require a clear majority.
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
