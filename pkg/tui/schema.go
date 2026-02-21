package tui

import (
	"encoding/json"
	"fmt"
	"sort"
)

// ColumnHint provides display hints for a specific column in columnar table rendering.
// Hints can be constructed directly in Go code or parsed from a JSON Schema
// using [ParseSchema].
type ColumnHint struct {
	// MaxWidth caps the column width (in characters).
	// Derived from JSON Schema maxLength, format, or enum values.
	// 0 means no cap (use natural width).
	MaxWidth int

	// Priority controls column importance during width allocation.
	// Higher values = more important; columns with higher priority resist
	// shrinking when the table exceeds available width.
	// Derived from JSON Schema required membership and property declaration order.
	// 0 is the default; negative values are valid (lower priority).
	Priority int

	// DisplayName overrides the column header text.
	// Derived from JSON Schema title.
	// Empty string means use the original field name.
	DisplayName string

	// Align controls text alignment within the column.
	// "right" for numeric columns, "left" (default) for everything else.
	// Derived from JSON Schema type (integer/number → right).
	Align string

	// Hidden omits this column from output entirely.
	// Derived from JSON Schema deprecated: true.
	// Merges with the existing HiddenColumns mechanism.
	Hidden bool
}

// ParseSchema extracts [ColumnHint] values from a standard JSON Schema document.
// It reads the schema's properties (for objects) or items.properties (for arrays
// of objects) and derives display hints from standard JSON Schema fields:
//
//   - title → DisplayName
//   - maxLength → MaxWidth
//   - enum → MaxWidth (longest enum value length)
//   - format (date, date-time, uuid, uri, email, ipv4, ipv6) → MaxWidth
//   - type (integer, number) → Align "right"
//   - deprecated: true → Hidden
//   - required array → Priority boost (+10 for required properties)
//   - Property declaration order → Priority tiebreaker (first declared = highest)
//
// The schemaJSON must be valid JSON. Returns a map keyed by property name.
func ParseSchema(schemaJSON []byte) (map[string]ColumnHint, error) {
	hints, _, err := ParseSchemaWithDisplay(schemaJSON)
	return hints, err
}

// ParseSchemaWithDisplay extracts both [ColumnHint] values and an optional
// [DisplaySchema] from a JSON Schema document. The display schema is derived
// from x-kvx-* vendor extension keys (e.g., x-kvx-list, x-kvx-detail,
// x-kvx-icon, x-kvx-collectionTitle).
//
// When no x-kvx-* extensions are present, the returned *DisplaySchema is nil.
// This function is a superset of [ParseSchema].
func ParseSchemaWithDisplay(schemaJSON []byte) (map[string]ColumnHint, *DisplaySchema, error) {
	var raw map[string]any
	if err := json.Unmarshal(schemaJSON, &raw); err != nil {
		return nil, nil, fmt.Errorf("invalid JSON schema: %w", err)
	}
	hints, err := parseSchemaObject(raw)
	if err != nil {
		return nil, nil, err
	}
	ds := extractDisplaySchemaFromJSONSchema(raw)
	if ds != nil {
		if err := validateDisplaySchema(ds); err != nil {
			return hints, nil, err
		}
	}
	return hints, ds, nil
}

func parseSchemaObject(raw map[string]any) (map[string]ColumnHint, error) { //nolint:unparam
	// Determine where properties live:
	// 1. If type=array with items.properties → use items.properties
	// 2. If type=object with properties → use properties directly
	// 3. If properties exists at top level → use it
	properties, required := findProperties(raw)
	if properties == nil {
		return nil, nil
	}

	requiredSet := make(map[string]bool, len(required))
	for _, r := range required {
		requiredSet[r] = true
	}

	// Parse ordered keys from properties using json.Decoder to preserve order.
	// Since we already unmarshalled, we use the map but also try to extract order.
	// Standard encoding/json doesn't preserve order in map[string]any, so we
	// derive order from the sorted keys as a fallback. If the caller wants
	// explicit order, they can set ColumnOrder on TableOptions.
	propKeys := sortedKeys(properties)

	hints := make(map[string]ColumnHint, len(propKeys))
	numProps := len(propKeys)

	for i, key := range propKeys {
		propRaw, ok := properties[key]
		if !ok {
			continue
		}
		propMap, ok := propRaw.(map[string]any)
		if !ok {
			continue
		}

		hint := ColumnHint{}

		// title → DisplayName
		if title, ok := propMap["title"].(string); ok && title != "" {
			hint.DisplayName = title
		}

		// maxLength → MaxWidth
		if ml, ok := toInt(propMap["maxLength"]); ok && ml > 0 {
			hint.MaxWidth = ml
		}

		// enum → MaxWidth (longest value)
		if enumVals, ok := propMap["enum"].([]any); ok && len(enumVals) > 0 {
			maxEnum := 0
			for _, v := range enumVals {
				s := fmt.Sprintf("%v", v)
				if len(s) > maxEnum {
					maxEnum = len(s)
				}
			}
			if maxEnum > 0 && (hint.MaxWidth == 0 || maxEnum < hint.MaxWidth) {
				hint.MaxWidth = maxEnum
			}
		}

		// format → MaxWidth for known formats
		if format, ok := propMap["format"].(string); ok {
			if fw := formatWidth(format); fw > 0 && (hint.MaxWidth == 0 || fw < hint.MaxWidth) {
				hint.MaxWidth = fw
			}
		}

		// type → Align
		propType := jsonSchemaType(propMap)
		if propType == "integer" || propType == "number" {
			hint.Align = "right"
		}

		// deprecated → Hidden
		if dep, ok := propMap["deprecated"].(bool); ok && dep {
			hint.Hidden = true
		}

		// Priority: required fields get a boost of +10,
		// then ordered by declaration (first property = highest base priority).
		basePriority := numProps - i // first property gets highest base
		if requiredSet[key] {
			basePriority += 10
		}
		hint.Priority = basePriority

		hints[key] = hint
	}

	return hints, nil
}

// findProperties locates the properties map and required list from a schema.
// Handles both object schemas and array-of-objects schemas.
func findProperties(schema map[string]any) (map[string]any, []string) {
	schemaType, _ := schema["type"].(string)

	// Array with items
	if schemaType == "array" {
		if items, ok := schema["items"].(map[string]any); ok {
			return findProperties(items)
		}
		return nil, nil
	}

	// Object or top-level properties
	if props, ok := schema["properties"].(map[string]any); ok {
		required := extractStringArray(schema["required"])
		return props, required
	}

	return nil, nil
}

// extractStringArray converts an []any to []string.
func extractStringArray(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// jsonSchemaType extracts the type string from a property schema.
// Handles both "type": "string" and "type": ["string", "null"].
func jsonSchemaType(prop map[string]any) string {
	switch t := prop["type"].(type) {
	case string:
		return t
	case []any:
		for _, v := range t {
			if s, ok := v.(string); ok && s != "null" {
				return s
			}
		}
	}
	return ""
}

// formatWidth returns a suggested max width for known JSON Schema format values.
func formatWidth(format string) int {
	switch format {
	case "date":
		return 10 // 2024-01-15
	case "date-time":
		return 26 // 2024-01-15T14:30:00+00:00
	case "time":
		return 15 // 14:30:00+00:00
	case "uuid":
		return 36 // f81d4fae-7dec-11d0-a765-00a0c91e6bf6
	case "ipv4":
		return 15 // 255.255.255.255
	case "ipv6":
		return 39 // full IPv6
	case "email":
		return 40 // reasonable email cap
	case "uri", "iri":
		return 60 // URIs can be long, cap conservatively
	default:
		return 0
	}
}

// toInt converts a JSON number (float64) to int.
func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return int(i), true
		}
	}
	return 0, false
}

// sortedKeys returns the keys of a map in sorted order.
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
