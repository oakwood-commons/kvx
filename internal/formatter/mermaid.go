package formatter

import (
	"fmt"
	"regexp"
	"strings"
)

// MermaidOptions controls Mermaid diagram output formatting.
type MermaidOptions struct {
	// Direction sets the diagram direction: TD (top-down), LR (left-right),
	// BT (bottom-top), RL (right-left). Default is TD.
	Direction string
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
	FieldHints map[string]int
	// ArrayStyle controls how array indices are displayed:
	// "index" = [0], [1]; "numbered" = 1, 2; "bullet" = •; "none" = skip index.
	ArrayStyle string
}

// mermaidBuilder tracks state during diagram generation.
type mermaidBuilder struct {
	lines  []string
	nodeID int
	opts   MermaidOptions
}

// FormatAsMermaid renders data as a Mermaid flowchart diagram.
// Maps become nodes with edges to child nodes, arrays show indexed children,
// and scalar values are displayed as node labels.
func FormatAsMermaid(node interface{}, opts MermaidOptions) string {
	if opts.Direction == "" {
		opts.Direction = "TD"
	}
	if opts.MaxArrayInline == 0 {
		opts.MaxArrayInline = defaultMaxArrayInline
	}

	b := &mermaidBuilder{
		lines: []string{fmt.Sprintf("graph %s", opts.Direction)},
		opts:  opts,
	}

	rootID := b.nextID()
	b.addNode(rootID, "root", "")
	b.buildMermaid(rootID, node, 0)

	return strings.Join(b.lines, "\n") + "\n"
}

// nextID generates a unique node identifier.
func (b *mermaidBuilder) nextID() string {
	id := fmt.Sprintf("n%d", b.nodeID)
	b.nodeID++
	return id
}

// addNode adds a node definition to the diagram.
func (b *mermaidBuilder) addNode(id, label, value string) {
	escaped := b.escapeLabel(label, value)
	b.lines = append(b.lines, fmt.Sprintf("    %s[%q]", id, escaped))
}

// addEdge adds an edge between two nodes.
func (b *mermaidBuilder) addEdge(fromID, toID string) {
	b.lines = append(b.lines, fmt.Sprintf("    %s --> %s", fromID, toID))
}

// escapeLabel creates a safe label for Mermaid nodes.
func (b *mermaidBuilder) escapeLabel(key, value string) string {
	var label string
	switch {
	case b.opts.NoValues || value == "":
		// Structure-only or no value to show
		if key == "" {
			label = "(item)"
		} else {
			label = key
		}
	case key == "":
		// Value-only (e.g., array-style none)
		label = value
	default:
		label = fmt.Sprintf("%s: %s", key, value)
	}
	// Mermaid uses quotes, so escape internal quotes
	label = strings.ReplaceAll(label, `"`, `'`)
	// Remove or escape other problematic characters
	label = strings.ReplaceAll(label, "\n", " ")
	label = strings.ReplaceAll(label, "\r", "")
	return label
}

// buildMermaid recursively builds the diagram structure.
func (b *mermaidBuilder) buildMermaid(parentID string, node interface{}, depth int) {
	// Check depth limit
	if b.opts.MaxDepth > 0 && depth >= b.opts.MaxDepth {
		ellipsisID := b.nextID()
		b.addNode(ellipsisID, "...", "")
		b.addEdge(parentID, ellipsisID)
		return
	}

	switch v := node.(type) {
	case map[string]interface{}:
		b.buildMapMermaid(parentID, v, depth)
	case []interface{}:
		b.buildArrayMermaid(parentID, v, depth)
	default:
		// Scalar value - add as child node with the value
		// Always show scalar roots even with NoValues since there's nothing else to display
		if node != nil {
			childID := b.nextID()
			valStr := formatScalarSimple(node)
			b.addNode(childID, valStr, "")
			b.addEdge(parentID, childID)
		}
	}
}

// buildMapMermaid adds map entries as child nodes.
func (b *mermaidBuilder) buildMapMermaid(parentID string, m map[string]interface{}, depth int) {
	if len(m) == 0 {
		return
	}

	keys := getSortedKeys(m)

	for _, key := range keys {
		val := m[key]
		b.addNodeForValue(parentID, key, val, depth)
	}
}

// buildArrayMermaid adds array elements as child nodes.
func (b *mermaidBuilder) buildArrayMermaid(parentID string, arr []interface{}, depth int) {
	if len(arr) == 0 {
		return
	}

	// Handle scalar arrays
	if isScalarArray(arr) && !b.opts.ExpandArrays {
		if len(arr) <= b.opts.MaxArrayInline {
			// Show inline
			childID := b.nextID()
			inlineVal := formatInlineArray(arr)
			b.addNode(childID, inlineVal, "")
			b.addEdge(parentID, childID)
		} else {
			// Show summary
			childID := b.nextID()
			b.addNode(childID, fmt.Sprintf("[%d items]", len(arr)), "")
			b.addEdge(parentID, childID)
		}
		return
	}

	// Expand array elements
	for i, elem := range arr {
		indexLabel := FormatArrayIndex(i, b.opts.ArrayStyle)
		b.addNodeForValue(parentID, indexLabel, elem, depth)
	}
}

// addNodeForValue adds a node for a key-value pair.
func (b *mermaidBuilder) addNodeForValue(parentID, key string, val interface{}, depth int) {
	// Check depth limit before adding complex children
	if b.opts.MaxDepth > 0 && depth >= b.opts.MaxDepth {
		ellipsisID := b.nextID()
		b.addNode(ellipsisID, "...", "")
		b.addEdge(parentID, ellipsisID)
		return
	}

	switch v := val.(type) {
	case map[string]interface{}:
		childID := b.nextID()
		b.addNode(childID, key, "")
		b.addEdge(parentID, childID)
		b.buildMapMermaid(childID, v, depth+1)

	case []interface{}:
		childID := b.nextID()
		b.addNode(childID, key, "")
		b.addEdge(parentID, childID)
		b.buildArrayMermaid(childID, v, depth+1)

	default:
		// Scalar value
		childID := b.nextID()
		scalarVal := b.formatScalar(key, v)
		b.addNode(childID, key, scalarVal)
		b.addEdge(parentID, childID)
	}
}

// formatScalar formats a scalar value with optional truncation.
func (b *mermaidBuilder) formatScalar(key string, v interface{}) string {
	if b.opts.NoValues {
		return ""
	}

	s := formatScalarSimple(v)

	// Determine max length: field hint > global setting
	maxLen := b.opts.MaxStringLen
	if key != "" && b.opts.FieldHints != nil {
		if hint, ok := b.opts.FieldHints[key]; ok && hint > 0 {
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

// SanitizeMermaidID creates a valid Mermaid node ID from a string.
// Mermaid IDs should be alphanumeric with underscores.
var nonAlphanumeric = regexp.MustCompile(`[^a-zA-Z0-9_]`)

func SanitizeMermaidID(s string) string {
	return nonAlphanumeric.ReplaceAllString(s, "_")
}
