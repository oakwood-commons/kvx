package navigator

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/oakwood-commons/kvx/internal/cel"
	"github.com/oakwood-commons/kvx/internal/formatter"
)

// SortOrder defines how map keys are ordered when rendered.
type SortOrder string

const (
	SortNone       SortOrder = "none"
	SortAscending  SortOrder = "ascending"
	SortDescending SortOrder = "descending"
)

var currentSortOrder = SortAscending

// SetSortOrder updates the global sort order for map key rendering and returns the previous value.
func SetSortOrder(order SortOrder) SortOrder {
	prev := currentSortOrder
	switch order {
	case SortAscending, SortDescending, SortNone:
		currentSortOrder = order
	default:
		currentSortOrder = SortNone
	}
	return prev
}

// Debug controls whether navigator prints troubleshooting logs.
// Set via CLI `--debug`.
var Debug bool

// DebugWriter is where navigator debug output is written. Defaults to discard.
var DebugWriter = io.Discard

// EvaluateFunc is a function type for evaluating CEL expressions.
// This allows injecting a custom evaluator (e.g., with custom functions).
type EvaluateFunc func(expr string, root interface{}) (interface{}, error)

// evaluator is the global CEL evaluator function.
// By default, it uses the standard CEL evaluator.
var evaluator EvaluateFunc

// SetEvaluator sets a custom CEL evaluator function for navigation.
// This allows library users to inject their custom CEL environment.
func SetEvaluator(fn EvaluateFunc) {
	evaluator = fn
}

// getEvaluator returns the current evaluator, falling back to the default if not set.
func getEvaluator() EvaluateFunc {
	if evaluator != nil {
		return evaluator
	}
	// Fall back to default CEL evaluator
	return func(expr string, root interface{}) (interface{}, error) {
		e, err := cel.NewEvaluator()
		if err != nil {
			return nil, err
		}
		return e.Evaluate(expr, root)
	}
}

// NodeAtPath navigates a dotted path or CEL expression into a parsed YAML structure.
// Keys are separated by '.'; numeric segments index arrays.
// Also supports full CEL syntax like "items[0].tags" or "data.items.filter(x, x.available)"
func NodeAtPath(root interface{}, path string) (interface{}, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return root, nil
	}
	if trimmed == "_" {
		return root, nil
	}

	// Try simple path navigation first (dotted paths and bracket notation)
	if !isComplexCEL(path) {
		if Debug {
			fmt.Fprintf(DebugWriter, "DBG(nav): simple path=%q\n", path)
		}
		return simpleNavigate(root, path)
	}

	// Fall back to full CEL evaluation for complex expressions
	// Use the configured evaluator (allows custom CEL environments)
	eval := getEvaluator()

	// Evaluate complex CEL expression exactly as typed; no auto "_." prefixing
	expr := path
	if Debug {
		fmt.Fprintf(DebugWriter, "DBG(nav): complex path=%q final_expr=%q\n", path, expr)
	}

	result, err := eval(expr, root)
	if err != nil {
		return nil, fmt.Errorf("CEL evaluation error: %w", err)
	}

	// Ensure we return proper Go types
	// If result is still a CEL type, try to extract the underlying value
	if slice, ok := result.([]interface{}); ok {
		return slice, nil
	}

	return result, nil
}

// isComplexCEL checks if a path requires full CEL evaluation (not just simple navigation)
func isComplexCEL(path string) bool {
	// Treat quoted string literals as CEL (e.g., "hi")
	trimmed := strings.TrimSpace(path)
	if strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"") && len(trimmed) >= 2 {
		return true
	}
	// Treat map literals as CEL (e.g., {"a":1})
	if strings.HasPrefix(trimmed, "{") {
		return true
	}
	// Check for array literals vs bracket navigation
	// [1], [0], ["key"] are navigation paths
	// [1, 2], [x.y], etc. are array literals
	if strings.HasPrefix(trimmed, "[") {
		// Find the closing bracket
		closeBracket := strings.Index(trimmed, "]")
		if closeBracket > 0 {
			inside := trimmed[1:closeBracket]
			// If it's just a number, it's navigation (index access)
			if _, err := strconv.Atoi(inside); err == nil {
				return false // simple index navigation
			}
			// If it's a quoted string, it's navigation (key access)
			if strings.HasPrefix(inside, "\"") && strings.HasSuffix(inside, "\"") {
				return false // quoted key navigation
			}
			// Otherwise it's likely an array literal
			return true
		}
	}
	// Check for function calls - any identifier followed by parentheses
	// This includes: filter(), map(), size(), has(), all(), exists(), etc.
	if strings.Contains(path, "(") && strings.Contains(path, ")") {
		return true
	}
	// Treat expressions starting with '_' as CEL
	if strings.HasPrefix(path, "_.") || strings.HasPrefix(path, "_[") {
		return true
	}
	// Check for comparisons outside of brackets
	comparisons := []string{"==", "!=", "<=", ">=", "<", ">", "&&", "||"}
	for _, op := range comparisons {
		if strings.Contains(path, op) {
			return true
		}
	}
	return false
}

// simpleNavigate handles dotted paths and bracket notation without full CEL
func simpleNavigate(root interface{}, path string) (interface{}, error) {
	cur := root
	// Parse the path handling both dots and brackets
	// Examples: "items.0" -> ["items", "0"]
	//           "items[0]" -> ["items", "0"]
	//           "items[0].tags" -> ["items", "0", "tags"]
	parts := parsePath(path)
	for _, p := range parts {
		cur = navigateStep(cur, p)
		if errResult, ok := cur.(error); ok {
			return nil, errResult
		}
	}
	return cur, nil
}

// parsePath splits a path into navigation steps, handling both dot and bracket notation
// Examples: "items.0" -> ["items", "0"]
//
//	"items[0]" -> ["items", "0"]
//	"items[0].tags" -> ["items", "0", "tags"]
//	"regions.asia.countries[1]" -> ["regions", "asia", "countries", "1"]
func parsePath(path string) []string {
	var parts []string
	var current strings.Builder

	for i := 0; i < len(path); i++ {
		ch := path[i]
		switch ch {
		case '.':
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		case '[':
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
			// Find the closing bracket
			j := i + 1
			for j < len(path) && path[j] != ']' {
				j++
			}
			if j < len(path) {
				parts = append(parts, path[i+1:j])
				i = j
			}
		default:
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

// navigateStep navigates a single step (key or index) in the data structure.
func navigateStep(cur interface{}, step string) interface{} {
	key := step
	if strings.HasPrefix(key, `"`) && strings.HasSuffix(key, `"`) && len(key) > 1 {
		key = key[1 : len(key)-1]
	}

	switch t := cur.(type) {
	case map[string]interface{}:
		v, ok := t[key]
		if !ok {
			return fmt.Errorf("key '%s' not found", key)
		}
		return v
	case []interface{}:
		// try parse step as integer index
		idx, err := strconv.Atoi(step)
		if err != nil {
			return fmt.Errorf("expected numeric index into array but got '%s'", step)
		}
		if idx < 0 || idx >= len(t) {
			return fmt.Errorf("index %d out of range", idx)
		}
		return t[idx]
	default:
		rv := reflect.ValueOf(cur)
		if !rv.IsValid() {
			return fmt.Errorf("cannot descend into %T at '%s'", cur, step)
		}

		for rv.Kind() == reflect.Ptr {
			if rv.IsNil() {
				return fmt.Errorf("cannot descend into %T at '%s'", cur, step)
			}
			rv = rv.Elem()
		}

		switch rv.Kind() { //nolint:exhaustive // only handle container kinds relevant to navigation
		case reflect.Map:
			if rv.Type().Key().Kind() != reflect.String {
				return fmt.Errorf("cannot descend into %T at '%s'", cur, step)
			}
			mapKey := reflect.ValueOf(key).Convert(rv.Type().Key())
			value := rv.MapIndex(mapKey)
			if !value.IsValid() {
				return fmt.Errorf("key '%s' not found", key)
			}
			return value.Interface()
		case reflect.Slice, reflect.Array:
			idx, err := strconv.Atoi(step)
			if err != nil {
				return fmt.Errorf("expected numeric index into array but got '%s'", step)
			}
			if idx < 0 || idx >= rv.Len() {
				return fmt.Errorf("index %d out of range", idx)
			}
			return rv.Index(idx).Interface()
		case reflect.Struct:
			if field, ok := structFieldValue(rv, key); ok {
				return field
			}
			return fmt.Errorf("key '%s' not found", key)
		default:
			return fmt.Errorf("cannot descend into %T at '%s'", cur, step)
		}
	}
}

func structFieldValue(rv reflect.Value, key string) (interface{}, bool) {
	typ := rv.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}
		tag := field.Tag.Get("json")
		tagName := strings.Split(tag, ",")[0]
		if tagName == "-" {
			continue
		}
		if tagName == key || field.Name == key {
			return rv.Field(i).Interface(), true
		}
	}
	return nil, false
}

// NodeToRows converts a node into rows of [key, value] pairs for table display
func NodeToRows(node interface{}) [][]string {
	var rows [][]string
	switch t := node.(type) {
	case map[string]interface{}:
		// Treat empty maps as scalar values
		if len(t) == 0 {
			rows = append(rows, []string{"(value)", formatter.Stringify(node)})
			return rows
		}
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		switch currentSortOrder {
		case SortAscending:
			sort.Strings(keys)
		case SortDescending:
			sort.Strings(keys)
			reverseStrings(keys)
		case SortNone:
			// Preserve natural/insertion order where possible (maps may be random)
		}
		for _, k := range keys {
			v := t[k]
			rows = append(rows, []string{k, formatter.Stringify(v)})
		}
	case []interface{}:
		// Treat empty arrays as scalar values
		if len(t) == 0 {
			rows = append(rows, []string{"(value)", formatter.Stringify(node)})
			return rows
		}
		for i, v := range t {
			rows = append(rows, []string{fmt.Sprintf("[%d]", i), formatter.Stringify(v)})
		}
	default:
		// Check if it's a map or slice type (could be []map, []string, typed maps, etc.)
		rv := reflect.ValueOf(node)
		if rv.Kind() == reflect.Map && rv.IsValid() && rv.Type().Key().Kind() == reflect.String {
			if rv.Len() == 0 {
				rows = append(rows, []string{"(value)", formatter.Stringify(node)})
				return rows
			}
			keys := make([]string, 0, rv.Len())
			for _, k := range rv.MapKeys() {
				keys = append(keys, k.String())
			}
			switch currentSortOrder {
			case SortAscending:
				sort.Strings(keys)
			case SortDescending:
				sort.Strings(keys)
				reverseStrings(keys)
			case SortNone:
			}
			for _, k := range keys {
				value := rv.MapIndex(reflect.ValueOf(k)).Interface()
				rows = append(rows, []string{k, formatter.Stringify(value)})
			}
			return rows
		}

		if rv.Kind() == reflect.Slice {
			for i := 0; i < rv.Len(); i++ {
				v := rv.Index(i).Interface()
				rows = append(rows, []string{fmt.Sprintf("[%d]", i), formatter.Stringify(v)})
			}
		} else {
			rows = append(rows, []string{"(value)", formatter.Stringify(node)})
		}
	}
	return rows
}

// ArrayStyle constants control how array indices are displayed.
const (
	ArrayStyleIndex    = "index"    // [0], [1], [2] (default legacy)
	ArrayStyleNumbered = "numbered" // 1, 2, 3 (1-based)
	ArrayStyleBullet   = "bullet"   // •
	ArrayStyleNone     = "none"     // no index column
)

// RowOptions configures how rows are generated from nodes.
type RowOptions struct {
	// ArrayStyle controls how array indices are displayed.
	// See ArrayStyle* constants. Default is "index" for backward compatibility.
	ArrayStyle string
}

// DefaultRowOptions returns the default options (legacy behavior).
func DefaultRowOptions() RowOptions {
	return RowOptions{
		ArrayStyle: ArrayStyleIndex,
	}
}

// NodeToRowsWithOptions converts a node into rows of [key, value] pairs for table display.
// Uses the provided options to customize output format.
func NodeToRowsWithOptions(node interface{}, opts RowOptions) [][]string {
	var rows [][]string
	switch t := node.(type) {
	case map[string]interface{}:
		// Treat empty maps as scalar values
		if len(t) == 0 {
			rows = append(rows, []string{"(value)", formatter.Stringify(node)})
			return rows
		}
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		switch currentSortOrder {
		case SortAscending:
			sort.Strings(keys)
		case SortDescending:
			sort.Strings(keys)
			reverseStrings(keys)
		case SortNone:
		}
		for _, k := range keys {
			v := t[k]
			rows = append(rows, []string{k, formatter.Stringify(v)})
		}
	case []interface{}:
		// Treat empty arrays as scalar values
		if len(t) == 0 {
			rows = append(rows, []string{"(value)", formatter.Stringify(node)})
			return rows
		}
		for i, v := range t {
			key := formatArrayIndex(i, opts.ArrayStyle)
			rows = append(rows, []string{key, formatter.Stringify(v)})
		}
	default:
		rv := reflect.ValueOf(node)
		if rv.Kind() == reflect.Map && rv.IsValid() && rv.Type().Key().Kind() == reflect.String {
			if rv.Len() == 0 {
				rows = append(rows, []string{"(value)", formatter.Stringify(node)})
				return rows
			}
			keys := make([]string, 0, rv.Len())
			for _, k := range rv.MapKeys() {
				keys = append(keys, k.String())
			}
			switch currentSortOrder {
			case SortAscending:
				sort.Strings(keys)
			case SortDescending:
				sort.Strings(keys)
				reverseStrings(keys)
			case SortNone:
			}
			for _, k := range keys {
				value := rv.MapIndex(reflect.ValueOf(k)).Interface()
				rows = append(rows, []string{k, formatter.Stringify(value)})
			}
			return rows
		}

		if rv.Kind() == reflect.Slice {
			for i := 0; i < rv.Len(); i++ {
				v := rv.Index(i).Interface()
				key := formatArrayIndex(i, opts.ArrayStyle)
				rows = append(rows, []string{key, formatter.Stringify(v)})
			}
		} else {
			rows = append(rows, []string{"(value)", formatter.Stringify(node)})
		}
	}
	return rows
}

// formatArrayIndex formats an array index according to the specified style.
func formatArrayIndex(index int, style string) string {
	switch style {
	case ArrayStyleNumbered:
		return fmt.Sprintf("%d", index+1)
	case ArrayStyleBullet:
		return "•"
	case ArrayStyleNone:
		return ""
	default: // ArrayStyleIndex
		return fmt.Sprintf("[%d]", index)
	}
}

func reverseStrings(values []string) {
	for i, j := 0, len(values)-1; i < j; i, j = i+1, j-1 {
		values[i], values[j] = values[j], values[i]
	}
}
