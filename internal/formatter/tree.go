package formatter

import (
	"fmt"
	"strings"

	"github.com/xlab/treeprint"
)

const (
	// defaultMaxArrayInline is the max number of array elements to show inline.
	defaultMaxArrayInline = 3
)

// TreeOptions controls tree output formatting.
type TreeOptions struct {
	// NoValues hides values at leaf nodes (structure only).
	NoValues bool
	// MaxDepth limits tree depth (0 = unlimited).
	MaxDepth int
	// ExpandArrays shows all array elements instead of "[N items]" summary.
	ExpandArrays bool
	// MaxArrayInline is max items to show inline for scalar arrays (default 3).
	MaxArrayInline int
	// MaxStringLen is max chars before truncating inline strings.
	// 0 or negative = no truncation (unlimited).
	MaxStringLen int
	// FieldHints provides per-field max lengths (e.g., from JSON Schema maxLength).
	// Keys are field names, values are max character lengths.
	FieldHints map[string]int
	// ArrayStyle controls how array indices are displayed:
	// "index" = [0], [1]; "numbered" = 1, 2; "bullet" = •; "none" = skip index.
	ArrayStyle string
}

// ValidArrayStyles contains all valid array style values.
var ValidArrayStyles = []string{"index", "numbered", "bullet", "none"}

// ValidateArrayStyle returns an error if the style is invalid.
func ValidateArrayStyle(style string) error {
	if style == "" {
		return nil // empty means use default
	}
	for _, valid := range ValidArrayStyles {
		if style == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid array-style %q: valid values are index, numbered, bullet, none", style)
}

// FormatArrayIndex formats an array index based on style.
// Exported for use by other formatters (mermaid, list).
func FormatArrayIndex(i int, style string) string {
	switch style {
	case "numbered":
		return fmt.Sprintf("%d", i+1)
	case "bullet":
		return "•"
	case "none":
		return ""
	default: // "index" or empty
		return fmt.Sprintf("[%d]", i)
	}
}

// formatKeyValue formats a key-value pair for display.
// If key is empty (e.g., from array-style none), returns just the value.
func formatKeyValue(key, value string) string {
	if key == "" {
		return value
	}
	return key + ": " + value
}

// formatKeyOnly returns the key or a placeholder if empty.
func formatKeyOnly(key string) string {
	if key == "" {
		return "(item)"
	}
	return key
}

// FormatAsTree renders data as an ASCII tree structure.
// Maps become branches with keys as labels, arrays show indexed children,
// and scalar values are displayed inline at leaves.
func FormatAsTree(node interface{}, opts TreeOptions) string {
	if opts.MaxArrayInline == 0 {
		opts.MaxArrayInline = defaultMaxArrayInline
	}
	// MaxStringLen <= 0 means no truncation (unlimited)

	tree := treeprint.New()
	buildTree(tree, node, opts, 0)
	return tree.String()
}

// buildTree recursively builds the tree structure.
func buildTree(branch treeprint.Tree, node interface{}, opts TreeOptions, depth int) {
	// Check depth limit
	if opts.MaxDepth > 0 && depth >= opts.MaxDepth {
		branch.AddNode("...")
		return
	}

	switch v := node.(type) {
	case map[string]interface{}:
		buildMapTree(branch, v, opts, depth)
	case []interface{}:
		buildArrayTree(branch, v, opts, depth)
	default:
		// Scalar at root level (unusual but handle it)
		branch.AddNode(formatScalarValue(v, opts))
	}
}

// buildMapTree adds map entries as tree branches/nodes.
func buildMapTree(branch treeprint.Tree, m map[string]interface{}, opts TreeOptions, depth int) {
	if len(m) == 0 {
		return
	}

	keys := getSortedKeys(m)

	for _, key := range keys {
		val := m[key]
		addNodeForValue(branch, key, val, opts, depth)
	}
}

// buildArrayTree adds array elements as indexed children.
func buildArrayTree(branch treeprint.Tree, arr []interface{}, opts TreeOptions, depth int) {
	if len(arr) == 0 {
		return
	}

	// Check if it's an array of scalars that can be shown inline
	if !opts.ExpandArrays && isScalarArray(arr) && len(arr) <= opts.MaxArrayInline {
		// Already handled inline by parent - this shouldn't be called for inline arrays
		return
	}

	for i, elem := range arr {
		indexKey := FormatArrayIndex(i, opts.ArrayStyle)
		addNodeForValue(branch, indexKey, elem, opts, depth)
	}
}

// addNodeForValue adds a node or branch for a key-value pair.
func addNodeForValue(branch treeprint.Tree, key string, val interface{}, opts TreeOptions, depth int) {
	// Check depth limit before recursing
	if opts.MaxDepth > 0 && depth >= opts.MaxDepth {
		branch.AddNode(formatKeyValue(key, "..."))
		return
	}

	switch v := val.(type) {
	case map[string]interface{}:
		if len(v) == 0 {
			if opts.NoValues {
				branch.AddNode(formatKeyOnly(key))
			} else {
				branch.AddNode(formatKeyValue(key, "{}"))
			}
		} else {
			child := branch.AddBranch(formatKeyOnly(key))
			buildMapTree(child, v, opts, depth+1)
		}

	case []interface{}:
		addArrayNode(branch, key, v, opts, depth)

	default:
		// Scalar value
		if opts.NoValues {
			branch.AddNode(formatKeyOnly(key))
		} else {
			branch.AddNode(formatKeyValue(key, formatScalarValueWithKey(key, v, opts)))
		}
	}
}

// addArrayNode handles array nodes with appropriate inline/summary/expand logic.
func addArrayNode(branch treeprint.Tree, key string, v []interface{}, opts TreeOptions, depth int) {
	switch {
	case len(v) == 0:
		if opts.NoValues {
			branch.AddNode(formatKeyOnly(key))
		} else {
			branch.AddNode(formatKeyValue(key, "[]"))
		}
	case !opts.ExpandArrays && isScalarArray(v) && len(v) <= opts.MaxArrayInline:
		// Inline short scalar arrays
		if opts.NoValues {
			branch.AddNode(formatKeyOnly(key))
		} else {
			branch.AddNode(formatKeyValue(key, formatInlineArray(v)))
		}
	case !opts.ExpandArrays && isScalarArray(v) && len(v) > opts.MaxArrayInline:
		// Summarize long scalar arrays
		if opts.NoValues {
			branch.AddNode(formatKeyOnly(key))
		} else {
			branch.AddNode(formatKeyValue(key, fmt.Sprintf("[%d items]", len(v))))
		}
	default:
		// Complex array or expand mode - create branch with indexed children
		child := branch.AddBranch(formatKeyOnly(key))
		buildArrayTree(child, v, opts, depth+1)
	}
}

// isScalarArray returns true if all elements are scalars (not maps or arrays).
func isScalarArray(arr []interface{}) bool {
	for _, elem := range arr {
		switch elem.(type) {
		case map[string]interface{}, []interface{}:
			return false
		}
	}
	return true
}

// formatInlineArray formats a scalar array as [a, b, c].
func formatInlineArray(arr []interface{}) string {
	parts := make([]string, len(arr))
	for i, elem := range arr {
		parts[i] = formatScalarSimple(elem)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// formatScalarValue formats a scalar with truncation for display.
func formatScalarValue(v interface{}, opts TreeOptions) string {
	return formatScalarValueWithKey("", v, opts)
}

// formatScalarValueWithKey formats a scalar with key-specific truncation.
// Checks FieldHints for per-field max lengths, then falls back to MaxStringLen.
func formatScalarValueWithKey(key string, v interface{}, opts TreeOptions) string {
	s := formatScalarSimple(v)

	// Determine max length: field hint > global setting
	maxLen := opts.MaxStringLen
	if key != "" && opts.FieldHints != nil {
		if hint, ok := opts.FieldHints[key]; ok && hint > 0 {
			maxLen = hint
		}
	}

	// 0 or negative = no truncation
	if maxLen <= 0 {
		return s
	}

	if len(s) > maxLen {
		if maxLen <= 3 {
			return "..."
		}
		return s[:maxLen-3] + "..."
	}
	return s
}

// formatScalarSimple converts a scalar to string without truncation.
func formatScalarSimple(v interface{}) string {
	if v == nil {
		return "null"
	}
	switch val := v.(type) {
	case bool:
		if val {
			return "true"
		}
		return "false"
	case string:
		return val
	case float64:
		// Format without trailing zeros for clean integers
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case int, int64, int32:
		return fmt.Sprintf("%d", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}
