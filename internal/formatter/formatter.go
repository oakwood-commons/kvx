package formatter

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"reflect"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"golang.org/x/term"
)

var (
	defaultHeaderFG   = lipgloss.Color("12")
	defaultHeaderBG   = lipgloss.Color("236")
	defaultKeyColor   = lipgloss.Color("14")
	defaultValueColor = lipgloss.Color("248")
	defaultSeparator  = lipgloss.Color("240")

	headerStyle    lipgloss.Style
	keyStyle       lipgloss.Style
	valueStyle     lipgloss.Style
	separatorStyle lipgloss.Style
)

// TableColors controls the rendered colors for the formatter table.
// Empty fields fall back to legacy defaults (ANSI 256 codes).
type TableColors struct {
	HeaderFG       color.Color
	HeaderBG       color.Color
	KeyColor       color.Color
	ValueColor     color.Color
	SeparatorColor color.Color
}

func applyTableTheme(tc TableColors) {
	hfg := tc.HeaderFG
	hbg := tc.HeaderBG
	kc := tc.KeyColor
	vc := tc.ValueColor
	sep := tc.SeparatorColor
	if hfg == nil {
		hfg = defaultHeaderFG
	}
	if hbg == nil {
		hbg = defaultHeaderBG
	}
	if kc == nil {
		kc = defaultKeyColor
	}
	if vc == nil {
		vc = defaultValueColor
	}
	if sep == nil {
		sep = defaultSeparator
	}

	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(hfg).Background(hbg)
	keyStyle = lipgloss.NewStyle().Foreground(kc)
	valueStyle = lipgloss.NewStyle().Foreground(vc)
	separatorStyle = lipgloss.NewStyle().Foreground(sep)
}

// SetTableTheme overrides the global table styles. Callers can pass zero-valued
// fields to fall back to formatter defaults.
func SetTableTheme(tc TableColors) {
	applyTableTheme(tc)
}

//nolint:gochecknoinits // initialize default table theme for package consumers
func init() {
	applyTableTheme(TableColors{})
}

// Stringify returns a compact string representation for an arbitrary YAML node
func Stringify(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return escapeScalarString(t)
	case bool, int, int64, float64:
		return fmt.Sprint(t)
	case map[string]any, []any:
		// marshal to compact JSON for readability in single column
		if b, err := json.Marshal(t); err == nil {
			return string(b)
		}
		return fmt.Sprintf("%v", t)
	default:
		// Use reflection to handle arbitrary maps, slices, and structs.
		// This ensures embedded users passing native Go types (e.g. structs,
		// []interface{}, map[string]interface{}) get proper JSON output instead
		// of Go's default fmt representation like "map[key:value]".
		rv := reflect.ValueOf(v)
		switch rv.Kind() { //nolint:exhaustive // only complex types need JSON marshaling
		case reflect.Map, reflect.Slice, reflect.Array, reflect.Struct:
			if b, err := json.Marshal(v); err == nil {
				return string(b)
			}
		case reflect.Ptr:
			if !rv.IsNil() {
				elem := rv.Elem()
				if elem.Kind() == reflect.Struct || elem.Kind() == reflect.Map || elem.Kind() == reflect.Slice {
					if b, err := json.Marshal(v); err == nil {
						return string(b)
					}
				}
			}
		}
		return fmt.Sprintf("%v", v)
	}
}

// StringifyPreserveNewlines returns a string representation while keeping real line breaks
// for scalar strings (used in scalar view so users can read multiline values).
// Non-string types fall back to Stringify for consistency.
func StringifyPreserveNewlines(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return normalizeScalarString(t, false, true)
	default:
		return Stringify(v)
	}
}

// escapeScalarString flattens control characters in scalar strings so table rows stay single-line.
func escapeScalarString(s string) string {
	return normalizeScalarString(s, true, false)
}

// normalizeScalarString prepares scalar strings for display. When escapeNewlines is true, newline
// characters are rendered as literal "\\n" so table rows stay single-line. When false, real
// line breaks are preserved (used in scalar view) while carriage returns are normalized away.
// When expandEscapedNewlines is true, literal "\\n" sequences are converted to real newlines so
// scalar string values that contain escaped newlines display as multi-line content.
func normalizeScalarString(s string, escapeNewlines bool, expandEscapedNewlines bool) string {
	if s == "" {
		return s
	}
	// Normalize Windows newlines first, then escape remaining control chars.
	s = strings.ReplaceAll(s, "\r\n", "\n")
	// Convert bare carriage returns to newlines to avoid control characters in output
	s = strings.ReplaceAll(s, "\r", "\n")
	if expandEscapedNewlines {
		if strings.Contains(s, "\\r\\n") {
			s = strings.ReplaceAll(s, "\\r\\n", "\n")
		}
		if strings.Contains(s, "\\r") {
			s = strings.ReplaceAll(s, "\\r", "\n")
		}
		if strings.Contains(s, "\\n") {
			s = strings.ReplaceAll(s, "\\n", "\n")
		}
	}
	if escapeNewlines {
		if strings.Contains(s, "\n") {
			s = strings.ReplaceAll(s, "\n", "\\n")
		}
	}
	return s
}

// truncate truncates a string to maxLen and adds ellipsis if needed
func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return s
	}
	// Measure display width using lipgloss (handles wide chars and ANSI codes)
	w := lipgloss.Width(s)
	if w <= maxLen {
		return s
	}
	if maxLen < 3 {
		// Too short for ellipsis - truncate to exact width
		// Strip ANSI and truncate by visual width
		result := ""
		width := 0
		for _, r := range s {
			rw := lipgloss.Width(string(r))
			if width+rw > maxLen {
				break
			}
			result += string(r)
			width += rw
		}
		return result
	}
	// Truncate to fit width, leaving room for ellipsis
	target := maxLen - 3
	result := ""
	width := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if width+rw > target {
			break
		}
		result += string(r)
		width += rw
	}
	return result + "..."
}

// getTerminalWidth returns the terminal width, or a default if detection fails
func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return 120 // sensible default
	}
	return width
}

// CalculateNaturalTableWidth calculates the natural width needed for a KEY/VALUE table
// without truncation. Returns the width needed for the content (key + sep + value).
func CalculateNaturalTableWidth(rows [][]string) int {
	sepWidth := 2
	maxKeyWidth := 3 // "KEY" header
	maxValWidth := 5 // "VALUE" header

	for _, row := range rows {
		if len(row) > 0 {
			w := lipgloss.Width(row[0])
			if w > maxKeyWidth {
				maxKeyWidth = w
			}
		}
		if len(row) > 1 {
			w := lipgloss.Width(row[1])
			if w > maxValWidth {
				maxValWidth = w
			}
		}
	}

	return maxKeyWidth + sepWidth + maxValWidth
}

// RenderTableFitContent renders a KEY/VALUE table sized to fit its content.
// maxWidth limits the table width (truncation occurs if content exceeds it).
// If maxWidth is 0, no truncation is applied.
func RenderTableFitContent(rows [][]string, noColor bool, maxWidth int) string {
	sepWidth := 2
	sep := strings.Repeat(" ", sepWidth)

	// Calculate natural widths
	maxKeyWidth := 3 // "KEY" header
	maxValWidth := 5 // "VALUE" header

	for _, row := range rows {
		if len(row) > 0 {
			w := lipgloss.Width(row[0])
			if w > maxKeyWidth {
				maxKeyWidth = w
			}
		}
		if len(row) > 1 {
			w := lipgloss.Width(row[1])
			if w > maxValWidth {
				maxValWidth = w
			}
		}
	}

	keyWidth := maxKeyWidth
	valueWidth := maxValWidth

	// Apply max width constraint if needed
	if maxWidth > 0 {
		totalNeeded := keyWidth + sepWidth + valueWidth
		if totalNeeded > maxWidth {
			// Need to truncate - distribute space proportionally
			available := maxWidth - sepWidth
			if available < 10 {
				available = 10
			}
			// Give key column 30% max, rest to value
			maxKeyAlloc := available * 30 / 100
			if maxKeyAlloc < 5 {
				maxKeyAlloc = 5
			}
			if keyWidth > maxKeyAlloc {
				keyWidth = maxKeyAlloc
			}
			valueWidth = available - keyWidth
			if valueWidth < 5 {
				valueWidth = 5
			}
		}
	}

	headerWidth := keyWidth + sepWidth + valueWidth

	var b strings.Builder

	headerKey := padRight("KEY", keyWidth)
	headerValue := padRight("VALUE", valueWidth)
	if !noColor {
		headerKey = headerStyle.Render(headerKey)
		headerValue = headerStyle.Render(headerValue)
	}
	b.WriteString(headerKey + sep + headerValue + "\n")

	separator := strings.Repeat("─", headerWidth)
	if !noColor {
		separator = separatorStyle.Render(separator)
	}
	b.WriteString(separator + "\n")

	for _, row := range rows {
		key := ""
		val := ""
		if len(row) > 0 {
			key = row[0]
		}
		if len(row) > 1 {
			val = row[1]
		}
		keyStr := padRight(truncate(key, keyWidth), keyWidth)
		valStr := padRight(truncate(val, valueWidth), valueWidth)
		if !noColor {
			keyStr = keyStyle.Render(keyStr)
			valStr = valueStyle.Render(valStr)
		}
		b.WriteString(keyStr + sep + valStr + "\n")
	}

	return b.String()
}

// RenderTable prints a two-column table (key, value) for the given node
// with terminal width awareness, value truncation, and color styling
// keyColWidth: width for KEY column (0 = use default 30)
// valueColWidth: width for VALUE column (0 = auto-calculate from remaining space)
func RenderTable(node any, noColor bool, keyColWidth, valueColWidth int) string {
	// Caller supplies column widths based on their layout (panel width). Do not
	// recompute from terminal width here or the rendered rows will overflow the
	// caller's panel (causing wrapping in interactive mode).
	sepWidth := 2
	minValueWidth := 20
	sep := strings.Repeat(" ", sepWidth)

	keyWidth := keyColWidth
	if keyWidth <= 0 {
		keyWidth = 30
	}

	valueWidth := valueColWidth
	if valueWidth <= 0 {
		valueWidth = minValueWidth
	}
	if valueWidth < minValueWidth {
		valueWidth = minValueWidth
	}

	headerWidth := keyWidth + sepWidth + valueWidth

	var b strings.Builder

	// Render header with optional styling
	headerKey := padRight("KEY", keyWidth)
	headerValue := padRight("VALUE", valueWidth)
	if !noColor {
		headerKey = headerStyle.Render(headerKey)
		headerValue = headerStyle.Render(headerValue)
	}
	b.WriteString(headerKey + sep + headerValue + "\n")

	// Separator line
	separator := strings.Repeat("─", headerWidth)
	if !noColor {
		separator = separatorStyle.Render(separator)
	}
	b.WriteString(separator + "\n")

	switch t := node.(type) {
	case map[string]any:
		// sort keys for deterministic output
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := t[k]
			keyStr := padRight(truncate(k, keyWidth), keyWidth)
			valStr := padRight(truncate(Stringify(v), valueWidth), valueWidth)
			if !noColor {
				keyStr = keyStyle.Render(keyStr)
				valStr = valueStyle.Render(valStr)
			}
			b.WriteString(keyStr + sep + valStr + "\n")
		}
	case []any:
		for i, v := range t {
			keyStr := padRight(fmt.Sprintf("[%d]", i), keyWidth)
			valStr := padRight(truncate(Stringify(v), valueWidth), valueWidth)
			if !noColor {
				keyStr = keyStyle.Render(keyStr)
				valStr = valueStyle.Render(valStr)
			}
			b.WriteString(keyStr + sep + valStr + "\n")
		}
	default:
		// Check if it's a slice type (could be []map, []string, etc.)
		sliceVal := reflect.ValueOf(node)
		if sliceVal.Kind() == reflect.Slice {
			for i := 0; i < sliceVal.Len(); i++ {
				v := sliceVal.Index(i).Interface()
				keyStr := padRight(fmt.Sprintf("[%d]", i), keyWidth)
				valStr := padRight(truncate(Stringify(v), valueWidth), valueWidth)
				if !noColor {
					keyStr = keyStyle.Render(keyStr)
					valStr = valueStyle.Render(valStr)
				}
				b.WriteString(keyStr + sep + valStr + "\n")
			}
		} else {
			// scalar value - must match navigator.ScalarValueKey (can't import due to cycle)
			keyStr := padRight("(value)", keyWidth)
			valStr := padRight(truncate(Stringify(node), valueWidth), valueWidth)
			if !noColor {
				keyStr = keyStyle.Render(keyStr)
				valStr = valueStyle.Render(valStr)
			}
			b.WriteString(keyStr + sep + valStr + "\n")
		}
	}

	return b.String()
}

// RenderRows prints a two-column table (key, value) for precomputed rows.
// rows should contain [key, value] pairs in display order.
func RenderRows(rows [][]string, noColor bool, keyColWidth, valueColWidth int) string {
	sepWidth := 2
	minValueWidth := 20
	sep := strings.Repeat(" ", sepWidth)

	keyWidth := keyColWidth
	if keyWidth <= 0 {
		keyWidth = 30
	}

	valueWidth := valueColWidth
	if valueWidth <= 0 {
		valueWidth = minValueWidth
	}
	if valueWidth < minValueWidth {
		valueWidth = minValueWidth
	}

	headerWidth := keyWidth + sepWidth + valueWidth

	var b strings.Builder

	headerKey := padRight("KEY", keyWidth)
	headerValue := padRight("VALUE", valueWidth)
	if !noColor {
		headerKey = headerStyle.Render(headerKey)
		headerValue = headerStyle.Render(headerValue)
	}
	b.WriteString(headerKey + sep + headerValue + "\n")

	separator := strings.Repeat("─", headerWidth)
	if !noColor {
		separator = separatorStyle.Render(separator)
	}
	b.WriteString(separator + "\n")

	for _, row := range rows {
		key := ""
		val := ""
		if len(row) > 0 {
			key = row[0]
		}
		if len(row) > 1 {
			val = row[1]
		}
		keyStr := padRight(truncate(key, keyWidth), keyWidth)
		valStr := padRight(truncate(val, valueWidth), valueWidth)
		if !noColor {
			keyStr = keyStyle.Render(keyStr)
			valStr = valueStyle.Render(valStr)
		}
		b.WriteString(keyStr + sep + valStr + "\n")
	}

	return b.String()
}

// padRight pads a string to the specified width, right-aligned
func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}
