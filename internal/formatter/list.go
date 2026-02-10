package formatter

import (
	"strings"
)

// ListOptions controls list output formatting.
type ListOptions struct {
	NoColor    bool   // disable color output
	ArrayStyle string // array index style: index, numbered, bullet, none
}

// FormatAsList renders data in a vertical list format.
// Arrays of objects display each element with an index header and indented properties.
// Maps display as key/value pairs with indentation.
// Scalar values display as "value: <scalar>".
func FormatAsList(node interface{}, opts ListOptions) string {
	var b strings.Builder

	switch v := node.(type) {
	case []interface{}:
		b.WriteString(formatArrayAsList(v, opts))
	case map[string]interface{}:
		b.WriteString(formatMapAsList(v, "", opts.NoColor))
	default:
		// Scalar values: display with "value:" label
		labelStr := "value"
		if !opts.NoColor {
			labelStr = keyStyle.Render(labelStr)
		}
		valueStr := StringifyPreserveNewlines(v)
		if !opts.NoColor {
			valueStr = valueStyle.Render(valueStr)
		}
		b.WriteString(labelStr)
		b.WriteString(": ")
		b.WriteString(valueStr)
		b.WriteString("\n")
	}

	return b.String()
}

func formatArrayAsList(arr []interface{}, opts ListOptions) string {
	if len(arr) == 0 {
		return ""
	}

	var b strings.Builder

	// Check if this is a homogeneous array of maps (objects)
	isHomogeneousObjects := len(arr) > 0
	firstIsMap := isObjectType(arr[0])

	if firstIsMap {
		for _, item := range arr {
			if !isObjectType(item) {
				isHomogeneousObjects = false
				break
			}
		}
	} else {
		isHomogeneousObjects = false
	}

	if isHomogeneousObjects {
		// Array of objects: show each with index header and indented properties
		for i, elem := range arr {
			if i > 0 {
				b.WriteString("\n")
			}

			// Header for this element (skip if style is "none")
			headerStr := FormatArrayIndex(i, opts.ArrayStyle)
			if headerStr != "" {
				if !opts.NoColor {
					headerStr = headerStyle.Render(headerStr)
				}
				b.WriteString(headerStr)
				b.WriteString("\n")
			}

			// Indented properties
			if m, ok := elem.(map[string]interface{}); ok {
				b.WriteString(formatMapAsList(m, "  ", opts.NoColor))
			}
		}
	} else {
		// Array of scalars or mixed types: print each on its own line (same as table)
		for _, elem := range arr {
			b.WriteString(StringifyPreserveNewlines(elem))
			b.WriteString("\n")
		}
	}

	return b.String()
}

func formatMapAsList(m map[string]interface{}, indent string, noColor bool) string {
	if len(m) == 0 {
		return ""
	}

	var b strings.Builder

	// Get sorted keys for consistent output
	keys := getSortedKeys(m)

	for i, key := range keys {
		if i > 0 {
			b.WriteString("\n")
		}

		keyStr := indent + key
		if !noColor {
			keyStr = keyStyle.Render(keyStr)
		}

		val := m[key]
		valStr := StringifyPreserveNewlines(val)
		if !noColor {
			valStr = valueStyle.Render(valStr)
		}

		b.WriteString(keyStr)
		b.WriteString(": ")
		b.WriteString(valStr)
	}

	b.WriteString("\n")
	return b.String()
}

// isObjectType returns true if v is a map[string]interface{} (representing JSON object)
func isObjectType(v interface{}) bool {
	_, ok := v.(map[string]interface{})
	return ok
}

// getSortedKeys returns sorted keys from a map
func getSortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	// Sort for consistent output
	sortStringSlice(keys)
	return keys
}

// sortStringSlice sorts a slice of strings in place
func sortStringSlice(s []string) {
	for i := 0; i < len(s); i++ {
		for j := i + 1; j < len(s); j++ {
			if s[j] < s[i] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}
