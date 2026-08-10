package formatter

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"reflect"
	"strings"
	"sync"

	"charm.land/lipgloss/v2"
	"golang.org/x/term"
)

var (
	defaultHeaderFG   = lipgloss.Color("12")
	defaultHeaderBG   = lipgloss.Color("236")
	defaultKeyColor   = lipgloss.Color("14")
	defaultValueColor = lipgloss.Color("248")
	defaultSeparator  = lipgloss.Color("240")

	// styleMu guards headerStyle/keyStyle/valueStyle/separatorStyle/maxValueLines,
	// which are process-global defaults mutated at runtime via SetTableTheme and
	// SetMaxValueLines. Concurrent renders should prefer the *With functions
	// below, which take an explicit RenderStyles snapshot and touch no globals.
	// See issue #83.
	styleMu sync.RWMutex

	headerStyle    lipgloss.Style
	keyStyle       lipgloss.Style
	valueStyle     lipgloss.Style
	separatorStyle lipgloss.Style

	// maxValueLines caps how many lines a multi-line value renders in
	// the key-value table view. 0 disables multi-line (escapes newlines).
	// Negative means unlimited. Default: 10.
	maxValueLines = 10

	// defaultMaxValueLines is the initial value of maxValueLines, used to
	// reset the global when config omits the setting. Set once at package
	// init and never mutated afterward.
	defaultMaxValueLines = 10
)

// RenderStyles carries per-call render state (styles and the multi-line value
// cap) so concurrent renders never observe each other's settings. The zero
// value is not directly usable; construct one via CurrentRenderStyles() (a
// snapshot of the package defaults) or StylesFromColors, and override fields
// as needed (e.g. MaxValueLines) before passing to a *With render function.
type RenderStyles struct {
	Header, Key, Value, Separator lipgloss.Style
	MaxValueLines                 int
}

// CurrentRenderStyles returns a snapshot of the current package-level default
// render styles. Safe for concurrent use; the returned value is independent
// of subsequent SetTableTheme/SetMaxValueLines calls.
func CurrentRenderStyles() RenderStyles {
	styleMu.RLock()
	defer styleMu.RUnlock()
	return RenderStyles{
		Header:        headerStyle,
		Key:           keyStyle,
		Value:         valueStyle,
		Separator:     separatorStyle,
		MaxValueLines: maxValueLines,
	}
}

// StylesFromColors builds a RenderStyles from TableColors without mutating
// any package-level state. MaxValueLines is populated from the current
// package default; override it on the returned value if needed.
func StylesFromColors(tc TableColors) RenderStyles {
	s := stylesFromColors(tc)
	styleMu.RLock()
	s.MaxValueLines = maxValueLines
	styleMu.RUnlock()
	return s
}

func stylesFromColors(tc TableColors) RenderStyles {
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
	return RenderStyles{
		Header:    lipgloss.NewStyle().Bold(true).Foreground(hfg).Background(hbg),
		Key:       lipgloss.NewStyle().Foreground(kc),
		Value:     lipgloss.NewStyle().Foreground(vc),
		Separator: lipgloss.NewStyle().Foreground(sep),
	}
}

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
	s := stylesFromColors(tc)

	styleMu.Lock()
	headerStyle = s.Header
	keyStyle = s.Key
	valueStyle = s.Value
	separatorStyle = s.Separator
	styleMu.Unlock()
}

// SetTableTheme overrides the global table styles. Callers can pass zero-valued
// fields to fall back to formatter defaults.
//
// NOTE: this mutates process-global state and is unsafe for concurrent
// rendering (see issue #83). Prefer StylesFromColors plus a *With render
// function, which take styles explicitly and touch no globals.
func SetTableTheme(tc TableColors) {
	applyTableTheme(tc)
}

// SetMaxValueLines sets the maximum number of lines rendered for multi-line
// values in the key-value table view. 0 disables multi-line rendering
// (newlines are escaped). Negative values mean unlimited.
//
// NOTE: this mutates process-global state and is unsafe for concurrent
// rendering (see issue #83). Prefer setting RenderStyles.MaxValueLines and
// using a *With render function.
func SetMaxValueLines(n int) {
	styleMu.Lock()
	maxValueLines = n
	styleMu.Unlock()
}

// MaxValueLines returns the current multi-line value cap.
func MaxValueLines() int {
	styleMu.RLock()
	defer styleMu.RUnlock()
	return maxValueLines
}

// DefaultMaxValueLines returns the built-in default so callers can reset
// the global when their configuration omits an explicit value.
func DefaultMaxValueLines() int {
	return defaultMaxValueLines
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
		case reflect.Pointer:
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
//
// NOTE: this reads the process-global maxValueLines, which is unsafe for
// concurrent rendering (see issue #83). Prefer CalculateNaturalTableWidthWith.
func CalculateNaturalTableWidth(rows [][]string) int {
	return CalculateNaturalTableWidthWith(rows, CurrentRenderStyles())
}

// CalculateNaturalTableWidthWith is like CalculateNaturalTableWidth but takes
// styles explicitly and reads no global state.
func CalculateNaturalTableWidthWith(rows [][]string, styles RenderStyles) int {
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
			w := naturalValueWidthWith(row[1], styles)
			if w > maxValWidth {
				maxValWidth = w
			}
		}
	}

	return maxKeyWidth + sepWidth + maxValWidth
}

// reorderRows reorders key-value row pairs according to columnOrder.
// Rows whose key (first element) appears in columnOrder come first in that order;
// remaining rows keep their original relative order.
func reorderRows(rows [][]string, columnOrder []string) [][]string {
	if len(columnOrder) == 0 || len(rows) == 0 {
		return rows
	}

	ordered := make([][]string, 0, len(rows))
	used := make(map[int]bool, len(columnOrder))

	for _, col := range columnOrder {
		for i, row := range rows {
			if len(row) > 0 && row[0] == col && !used[i] {
				ordered = append(ordered, row)
				used[i] = true
				break
			}
		}
	}

	for i, row := range rows {
		if !used[i] {
			ordered = append(ordered, row)
		}
	}

	return ordered
}

// RenderTableFitContent renders a KEY/VALUE table sized to fit its content.
// maxWidth limits the table width (truncation occurs if content exceeds it).
// If maxWidth is 0, no truncation is applied.
// columnOrder specifies the preferred key ordering for rows; nil means no reordering.
//
// NOTE: this reads process-global styles/maxValueLines, which is unsafe
// for concurrent rendering (see issue #83). Prefer RenderTableFitContentWith.
func RenderTableFitContent(rows [][]string, noColor bool, maxWidth int, columnOrder []string) string {
	return RenderTableFitContentWith(rows, noColor, maxWidth, columnOrder, CurrentRenderStyles())
}

// RenderTableFitContentWith is like RenderTableFitContent but takes styles
// explicitly and reads no global state.
func RenderTableFitContentWith(rows [][]string, noColor bool, maxWidth int, columnOrder []string, styles RenderStyles) string {
	rows = reorderRows(rows, columnOrder)
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
			w := naturalValueWidthWith(row[1], styles)
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
		headerKey = styles.Header.Render(headerKey)
		headerValue = styles.Header.Render(headerValue)
	}
	b.WriteString(headerKey + sep + headerValue + "\n")

	separator := strings.Repeat("─", headerWidth)
	if !noColor {
		separator = styles.Separator.Render(separator)
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
		renderMultilineRowWith(&b, keyStr, val, keyWidth, valueWidth, noColor, sep, styles)
	}

	return b.String()
}

// RenderTable prints a two-column table (key, value) for the given node
// with terminal width awareness, value truncation, and color styling
// keyColWidth: width for KEY column (0 = use default 30)
// valueColWidth: width for VALUE column (0 = auto-calculate from remaining space)
//
// NOTE: this reads process-global styles/maxValueLines, which is unsafe
// for concurrent rendering (see issue #83). Prefer RenderTableWith.
func RenderTable(node any, noColor bool, keyColWidth, valueColWidth int, columnOrder []string) string {
	return RenderTableWith(node, noColor, keyColWidth, valueColWidth, columnOrder, CurrentRenderStyles())
}

// RenderTableWith is like RenderTable but takes styles explicitly and reads
// no global state, making it safe to call concurrently with different styles.
func RenderTableWith(node any, noColor bool, keyColWidth, valueColWidth int, columnOrder []string, styles RenderStyles) string {
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
		headerKey = styles.Header.Render(headerKey)
		headerValue = styles.Header.Render(headerValue)
	}
	b.WriteString(headerKey + sep + headerValue + "\n")

	// Separator line
	separator := strings.Repeat("─", headerWidth)
	if !noColor {
		separator = styles.Separator.Render(separator)
	}
	b.WriteString(separator + "\n")

	switch t := node.(type) {
	case map[string]any:
		keys := orderedMapKeys(t, columnOrder)
		for _, k := range keys {
			v := t[k]
			keyStr := padRight(truncate(k, keyWidth), keyWidth)
			valRaw := StringifyPreserveNewlines(v)
			renderMultilineRowWith(&b, keyStr, valRaw, keyWidth, valueWidth, noColor, sep, styles)
		}
	case []any:
		for i, v := range t {
			keyStr := padRight(fmt.Sprintf("[%d]", i), keyWidth)
			valRaw := StringifyPreserveNewlines(v)
			renderMultilineRowWith(&b, keyStr, valRaw, keyWidth, valueWidth, noColor, sep, styles)
		}
	default:
		// Check if it's a slice type (could be []map, []string, etc.)
		sliceVal := reflect.ValueOf(node)
		if sliceVal.Kind() == reflect.Slice {
			for i := 0; i < sliceVal.Len(); i++ {
				v := sliceVal.Index(i).Interface()
				keyStr := padRight(fmt.Sprintf("[%d]", i), keyWidth)
				valRaw := StringifyPreserveNewlines(v)
				renderMultilineRowWith(&b, keyStr, valRaw, keyWidth, valueWidth, noColor, sep, styles)
			}
		} else {
			// scalar value - must match navigator.ScalarValueKey (can't import due to cycle)
			keyStr := padRight("(value)", keyWidth)
			valRaw := StringifyPreserveNewlines(node)
			renderMultilineRowWith(&b, keyStr, valRaw, keyWidth, valueWidth, noColor, sep, styles)
		}
	}

	return b.String()
}

// RenderRows prints a two-column table (key, value) for precomputed rows.
// rows should contain [key, value] pairs in display order.
//
// NOTE: this reads process-global styles/maxValueLines, which is unsafe
// for concurrent rendering (see issue #83). Prefer RenderRowsWith.
func RenderRows(rows [][]string, noColor bool, keyColWidth, valueColWidth int) string {
	return RenderRowsWith(rows, noColor, keyColWidth, valueColWidth, CurrentRenderStyles())
}

// RenderRowsWith is like RenderRows but takes styles explicitly and reads no
// global state.
func RenderRowsWith(rows [][]string, noColor bool, keyColWidth, valueColWidth int, styles RenderStyles) string {
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
		headerKey = styles.Header.Render(headerKey)
		headerValue = styles.Header.Render(headerValue)
	}
	b.WriteString(headerKey + sep + headerValue + "\n")

	separator := strings.Repeat("─", headerWidth)
	if !noColor {
		separator = styles.Separator.Render(separator)
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
		renderMultilineRowWith(&b, keyStr, val, keyWidth, valueWidth, noColor, sep, styles)
	}

	return b.String()
}

// naturalValueWidth returns the display width of a value as it will actually be
// rendered. When multiline rendering is disabled (maxValueLines == 0) the value
// is flattened with escaped newlines, so the width is measured from the flattened
// form. Otherwise it returns the width of the widest individual line.
//
// NOTE: reads the process-global maxValueLines. Prefer naturalValueWidthWith.
func naturalValueWidth(val string) int {
	return naturalValueWidthWith(val, CurrentRenderStyles())
}

// naturalValueWidthWith is like naturalValueWidth but takes styles explicitly
// and reads no global state.
func naturalValueWidthWith(val string, styles RenderStyles) int {
	if styles.MaxValueLines == 0 {
		return lipgloss.Width(escapeScalarString(val))
	}
	best := 0
	for _, line := range strings.Split(val, "\n") {
		if w := lipgloss.Width(line); w > best {
			best = w
		}
	}
	return best
}

// padRight pads a string to the specified width, right-aligned
func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

// renderMultilineRowWith writes a key-value row to b, splitting multi-line
// values across multiple display rows. The first line appears next to the
// key; continuation lines are indented to align under the value column with
// an empty key column. Takes styles explicitly and reads no global state.
func renderMultilineRowWith(b *strings.Builder, keyStr, valRaw string, keyWidth, valueWidth int, noColor bool, sep string, styles RenderStyles) {
	// When multi-line rendering is disabled (maxValueLines == 0), flatten to single line.
	if styles.MaxValueLines == 0 {
		valFlat := padRight(truncate(escapeScalarString(valRaw), valueWidth), valueWidth)
		k := keyStr
		if !noColor {
			k = styles.Key.Render(k)
			valFlat = styles.Value.Render(valFlat)
		}
		b.WriteString(k + sep + valFlat + "\n")
		return
	}

	lines := strings.Split(valRaw, "\n")
	// Trim trailing empty line that YAML block scalars often leave
	if len(lines) > 1 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	// Cap visible lines when a positive limit is set.
	truncated := false
	if styles.MaxValueLines > 0 && len(lines) > styles.MaxValueLines {
		lines = lines[:styles.MaxValueLines]
		truncated = true
	}

	for i, line := range lines {
		var k string
		if i == 0 {
			k = keyStr
		} else {
			k = padRight("", keyWidth)
		}
		v := padRight(truncate(line, valueWidth), valueWidth)
		if !noColor {
			k = styles.Key.Render(k)
			v = styles.Value.Render(v)
		}
		b.WriteString(k + sep + v + "\n")
	}

	// Show truncation indicator
	if truncated {
		k := padRight("", keyWidth)
		v := padRight(truncate("...", valueWidth), valueWidth)
		if !noColor {
			k = styles.Key.Render(k)
			v = styles.Value.Render(v)
		}
		b.WriteString(k + sep + v + "\n")
	}
}
