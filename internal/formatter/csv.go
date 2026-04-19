package formatter

import (
	"sort"
	"strings"
)

// escapeCSVField escapes a CSV field according to RFC 4180.
// Fields are quoted if they contain:
//   - Commas (required by RFC 4180)
//   - Double quotes (required by RFC 4180)
//   - Line breaks (newlines/carriage returns, required by RFC 4180)
//   - Spaces (common practice for readability, not required by RFC 4180)
//
// When a field is quoted, any double quotes inside are escaped by doubling them.
func escapeCSVField(field string) string {
	needsQuoting := strings.Contains(field, ",") ||
		strings.Contains(field, "\"") ||
		strings.Contains(field, "\n") ||
		strings.Contains(field, "\r") ||
		strings.Contains(field, " ")

	if needsQuoting {
		escaped := strings.ReplaceAll(field, `"`, `""`)
		return `"` + escaped + `"`
	}
	return field
}

// FormatAsCSV converts data to CSV format.
// Arrays of objects use object keys as column headers.
// Simple arrays produce a single "value" column.
// Maps produce "key,value" rows.
// Scalars produce a single "value" row.
func FormatAsCSV(node any) string {
	var b strings.Builder

	writeCSVRow := func(fields []string) {
		for i, field := range fields {
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString(escapeCSVField(field))
		}
		b.WriteString("\n")
	}

	switch v := node.(type) {
	case []any:
		if len(v) == 0 {
			return ""
		}
		if _, ok := v[0].(map[string]any); ok {
			keySet := make(map[string]bool)
			for _, elem := range v {
				if obj, ok := elem.(map[string]any); ok {
					for k := range obj {
						keySet[k] = true
					}
				}
			}
			keys := make([]string, 0, len(keySet))
			for k := range keySet {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			writeCSVRow(keys)

			for _, elem := range v {
				if obj, ok := elem.(map[string]any); ok {
					row := make([]string, len(keys))
					for i, key := range keys {
						if val, ok := obj[key]; ok {
							row[i] = Stringify(val)
						}
					}
					writeCSVRow(row)
				}
			}
		} else {
			writeCSVRow([]string{"value"})
			for _, elem := range v {
				writeCSVRow([]string{Stringify(elem)})
			}
		}
	case map[string]any:
		writeCSVRow([]string{"key", "value"})
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			writeCSVRow([]string{k, Stringify(v[k])})
		}
	default:
		writeCSVRow([]string{"value"})
		writeCSVRow([]string{Stringify(node)})
	}

	return b.String()
}
