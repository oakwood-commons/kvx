package ui

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	runewidth "github.com/mattn/go-runewidth"

	"github.com/oakwood-commons/kvx/internal/formatter"
)

// DetailViewModel holds state for the sectioned detail rendering of a single object.
type DetailViewModel struct {
	Object    map[string]interface{} // The object being displayed
	Sections  []renderedSection      // Pre-computed rendered sections
	TitleText string                 // Header title (from TitleField)
	ScrollTop int                    // First visible line
	Width     int                    // Available width
	Height    int                    // Available height
}

// renderedSection is a pre-computed section of the detail view.
type renderedSection struct {
	Title  string   // Section heading (may be empty)
	Lines  []string // Rendered lines
	Layout string   // Layout type for reference
}

// BuildDetailView creates a DetailViewModel from an object using the display schema.
// This is the public entry point for library consumers (pkg/tui).
func BuildDetailView(node interface{}, schema *DisplaySchema, width, height int) *DetailViewModel {
	return buildDetailViewModel(node, schema, width, height)
}

// buildDetailViewModel creates a DetailViewModel from an object using the display schema.
func buildDetailViewModel(node interface{}, schema *DisplaySchema, width, height int) *DetailViewModel {
	obj, ok := node.(map[string]interface{})
	if !ok || schema == nil || schema.Detail == nil {
		return nil
	}

	dv := &DetailViewModel{
		Object: obj,
		Width:  width,
		Height: height,
	}

	// Resolve title
	if schema.Detail.TitleField != "" {
		dv.TitleText = formatter.Stringify(obj[schema.Detail.TitleField])
	}

	// Build hidden set
	hiddenSet := make(map[string]bool, len(schema.Detail.HiddenFields))
	for _, h := range schema.Detail.HiddenFields {
		hiddenSet[h] = true
	}
	// Title field is also hidden from sections (it's in the header)
	if schema.Detail.TitleField != "" {
		hiddenSet[schema.Detail.TitleField] = true
	}

	// Track which fields are covered by explicit sections
	covered := make(map[string]bool)
	for _, s := range schema.Detail.Sections {
		for _, f := range s.Fields {
			covered[f] = true
		}
	}

	contentWidth := width - 4
	if contentWidth < 10 {
		contentWidth = 10
	}

	// Render explicit sections
	for _, s := range schema.Detail.Sections {
		rs := renderDetailSection(obj, s, contentWidth, hiddenSet)
		if len(rs.Lines) > 0 {
			dv.Sections = append(dv.Sections, rs)
		}
	}

	// Collect uncovered fields into an "Other" section
	otherKeys := collectObjectKeys(obj, nil)
	var otherFields []string
	for _, k := range otherKeys {
		if !covered[k] && !hiddenSet[k] {
			otherFields = append(otherFields, k)
		}
	}
	if len(otherFields) > 0 {
		other := DetailSection{
			Fields: otherFields,
			Layout: DisplayLayoutTable,
		}
		rs := renderDetailSection(obj, other, contentWidth, hiddenSet)
		if len(rs.Lines) > 0 {
			dv.Sections = append(dv.Sections, rs)
		}
	}

	return dv
}

// renderDetailSection renders a single section of the detail view.
func renderDetailSection(obj map[string]interface{}, section DetailSection, width int, hidden map[string]bool) renderedSection {
	rs := renderedSection{
		Title:  section.Title,
		Layout: section.Layout,
	}

	layout := section.Layout
	if layout == "" {
		layout = DisplayLayoutTable
	}

	switch layout {
	case DisplayLayoutInline:
		rs.Lines = renderInlineSection(obj, section.Fields, width, hidden)
	case DisplayLayoutParagraph:
		rs.Lines = renderParagraphSection(obj, section.Fields, width, hidden)
	case DisplayLayoutTags:
		rs.Lines = renderTagsSection(obj, section.Fields, width, hidden)
	default: // table
		rs.Lines = renderTableSection(obj, section.Fields, width, hidden, section.ColumnOrder)
	}

	return rs
}

// renderInlineSection renders fields as "value1 · value2 · value3".
func renderInlineSection(obj map[string]interface{}, fields []string, width int, hidden map[string]bool) []string {
	var parts []string
	for _, f := range fields {
		if hidden[f] {
			continue
		}
		v := obj[f]
		if v == nil {
			continue
		}
		s := formatter.Stringify(v)
		if s == "" {
			continue
		}
		parts = append(parts, s)
	}
	if len(parts) == 0 {
		return nil
	}
	line := strings.Join(parts, " · ")
	if runewidth.StringWidth(line) > width {
		line = runewidth.Truncate(line, width-3, "...")
	}
	return []string{line}
}

// renderParagraphSection renders fields as wrapped text paragraphs.
func renderParagraphSection(obj map[string]interface{}, fields []string, width int, hidden map[string]bool) []string {
	var lines []string
	for _, f := range fields {
		if hidden[f] {
			continue
		}
		v := obj[f]
		if v == nil {
			continue
		}
		text := formatter.StringifyPreserveNewlines(v)
		if text == "" {
			continue
		}
		// Split by newlines first to preserve paragraph structure, then
		// wrap each line individually so wrapAtWidth does not collapse them.
		for _, paragraph := range strings.Split(text, "\n") {
			wrapped := wrapAtWidth(paragraph, width)
			lines = append(lines, strings.Split(wrapped, "\n")...)
		}
	}
	return lines
}

// renderTagsSection renders array fields as colored pill badges.
func renderTagsSection(obj map[string]interface{}, fields []string, width int, hidden map[string]bool) []string {
	th := CurrentTheme()
	badgeStyle := lipgloss.NewStyle().
		Foreground(th.HeaderFG).
		Background(th.HeaderBG).
		PaddingLeft(1).
		PaddingRight(1)

	var badges []string
	for _, f := range fields {
		if hidden[f] {
			continue
		}
		val := obj[f]
		switch v := val.(type) {
		case []interface{}:
			for _, elem := range v {
				badges = append(badges, badgeStyle.Render(formatter.Stringify(elem)))
			}
		case string:
			badges = append(badges, badgeStyle.Render(v))
		default:
			if val != nil {
				badges = append(badges, badgeStyle.Render(formatter.Stringify(val)))
			}
		}
	}
	if len(badges) == 0 {
		return nil
	}

	// Wrap badges into lines that fit within width
	var lines []string
	currentLine := ""
	currentWidth := 0
	for _, badge := range badges {
		bw := runewidth.StringWidth(stripANSI(badge))
		spaceNeeded := bw
		if currentWidth > 0 {
			spaceNeeded++ // space separator
		}
		if currentWidth+spaceNeeded > width && currentWidth > 0 {
			lines = append(lines, currentLine)
			currentLine = badge
			currentWidth = bw
		} else {
			if currentWidth > 0 {
				currentLine += " "
				currentWidth++
			}
			currentLine += badge
			currentWidth += bw
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}
	return lines
}

// renderTableSection renders fields as KEY/VALUE rows.
// When a field contains []map[string]any, it renders an inline columnar table
// instead of the placeholder "[N items]" text.
func renderTableSection(obj map[string]interface{}, fields []string, width int, hidden map[string]bool, columnOrder []string) []string {
	th := CurrentTheme()
	keyStyle := lipgloss.NewStyle().Foreground(th.KeyColor)
	valStyle := lipgloss.NewStyle().Foreground(th.ValueColor)

	// Find longest key for alignment
	maxKeyLen := 0
	for _, f := range fields {
		if hidden[f] {
			continue
		}
		if len(f) > maxKeyLen {
			maxKeyLen = len(f)
		}
	}
	if maxKeyLen > width/3 {
		maxKeyLen = width / 3
	}

	var lines []string
	for _, f := range fields {
		if hidden[f] {
			continue
		}
		v := obj[f]
		if v == nil {
			continue
		}

		// Check if this field is a homogeneous array of maps -> render as inline table.
		if rows, cols := extractNestedTable(v, columnOrder); rows != nil {
			tbl := renderInlineColumnarTable(cols, rows, width, &th)
			lines = append(lines, tbl...)
			continue
		}

		key := f
		if len(key) > maxKeyLen {
			key = runewidth.Truncate(key, maxKeyLen, "...")
		}
		// Pad key to alignment width
		key += strings.Repeat(" ", maxKeyLen-runewidth.StringWidth(key))

		val := stringifyValue(v, width-maxKeyLen-3)
		line := keyStyle.Render(key) + "  " + valStyle.Render(val)
		lines = append(lines, line)
	}
	return lines
}

// stringifyValue converts a value to a display string with width limiting.
func stringifyValue(v interface{}, maxWidth int) string {
	if maxWidth < 3 {
		maxWidth = 3
	}
	switch val := v.(type) {
	case []interface{}:
		parts := make([]string, 0, len(val))
		for _, elem := range val {
			parts = append(parts, formatter.Stringify(elem))
		}
		s := "[" + strings.Join(parts, ", ") + "]"
		if runewidth.StringWidth(s) > maxWidth {
			s = runewidth.Truncate(s, maxWidth-3, "") + "..."
		}
		return s
	case map[string]interface{}:
		s := fmt.Sprintf("{%d keys}", len(val))
		return s
	default:
		s := formatter.Stringify(v)
		if runewidth.StringWidth(s) > maxWidth {
			s = runewidth.Truncate(s, maxWidth-3, "") + "..."
		}
		return s
	}
}

// renderDetailView renders the full detail view for the panel layout.
func renderDetailView(dv *DetailViewModel, _ *DisplaySchema, noColor bool) string {
	if dv == nil {
		return "  (no data)"
	}

	th := CurrentTheme()
	contentWidth := dv.Width - 4
	_ = contentWidth // used by sections indirectly via dv.Width

	var allLines []string

	// Title is rendered in the panel border by panelLayoutStateFromModel,
	// so we skip it here to avoid duplication.

	// Render sections
	for _, sec := range dv.Sections {
		// Blank line before section
		allLines = append(allLines, "")

		// Section title
		if sec.Title != "" {
			sectionStyle := lipgloss.NewStyle().Bold(true)
			if !noColor {
				sectionStyle = sectionStyle.Foreground(th.StatusColor)
			}
			allLines = append(allLines, sectionStyle.Render(sec.Title))
		}

		// Section content
		allLines = append(allLines, sec.Lines...)
	}

	// Scrolling
	totalLines := len(allLines)
	visibleHeight := dv.Height - 2 // borders
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	if dv.ScrollTop > totalLines-visibleHeight {
		dv.ScrollTop = totalLines - visibleHeight
	}
	if dv.ScrollTop < 0 {
		dv.ScrollTop = 0
	}

	endLine := dv.ScrollTop + visibleHeight
	if endLine > totalLines {
		endLine = totalLines
	}

	visible := allLines[dv.ScrollTop:endLine]

	return strings.Join(visible, "\n")
}

// --- CustomView interface implementation ---

func (dv *DetailViewModel) Title() string {
	if dv == nil {
		return ""
	}
	return dv.TitleText
}
func (dv *DetailViewModel) FooterBar() string            { return "" }
func (dv *DetailViewModel) HandlesSearch() bool          { return false }
func (dv *DetailViewModel) Init() tea.Cmd                { return nil }
func (dv *DetailViewModel) SearchTitle() string          { return "" }
func (dv *DetailViewModel) FlashMessage() (string, bool) { return "", false }

func (dv *DetailViewModel) Render(width, height int, noColor bool) string {
	dv.Width = width
	dv.Height = height
	return renderDetailView(dv, nil, noColor) // schema not needed; sections pre-computed
}

func (dv *DetailViewModel) RowCount() (count int, selected int, label string) {
	return 1, 1, "detail"
}

func (dv *DetailViewModel) Update(_ tea.Msg) (CustomView, tea.Cmd) {
	return dv, nil // detail view has no async messages
}

// ---------------------------------------------------------------------------
// Nested table helpers (#62)
// ---------------------------------------------------------------------------

// extractNestedTable checks whether v is a []map[string]any (homogeneous).
// If so it returns the rows and an ordered list of column names, applying
// columnOrder to put preferred columns first.  Returns (nil, nil) otherwise.
func extractNestedTable(v interface{}, columnOrder []string) ([]map[string]interface{}, []string) {
	// Accept both []interface{} and []map[string]any (typed slices from Go callers).
	var arr []interface{}
	switch typed := v.(type) {
	case []interface{}:
		arr = typed
	case []map[string]any:
		arr = make([]interface{}, len(typed))
		for i, m := range typed {
			arr[i] = m
		}
	default:
		return nil, nil
	}
	if len(arr) == 0 {
		return nil, nil
	}

	rows := make([]map[string]interface{}, 0, len(arr))
	colSet := map[string]bool{}

	for _, elem := range arr {
		m, mOK := elem.(map[string]interface{})
		if !mOK {
			return nil, nil // not homogeneous maps
		}
		rows = append(rows, m)
		for k := range m {
			colSet[k] = true
		}
	}

	// Build ordered column list: preferred first, then remaining sorted.
	var cols []string
	used := map[string]bool{}
	for _, c := range columnOrder {
		if colSet[c] {
			cols = append(cols, c)
			used[c] = true
		}
	}
	remaining := make([]string, 0, len(colSet))
	for c := range colSet {
		if !used[c] {
			remaining = append(remaining, c)
		}
	}
	sort.Strings(remaining)
	cols = append(cols, remaining...)
	return rows, cols
}

// renderInlineColumnarTable renders a list of maps as a compact columnar table.
func renderInlineColumnarTable(cols []string, rows []map[string]interface{}, width int, th *Theme) []string {
	if len(cols) == 0 || len(rows) == 0 {
		return nil
	}
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(th.HeaderFG)
	cellStyle := lipgloss.NewStyle().Foreground(th.ValueColor)

	// Compute column widths from header + data.
	colWidths := make([]int, len(cols))
	for i, c := range cols {
		colWidths[i] = runewidth.StringWidth(c)
	}
	cellValues := make([][]string, len(rows))
	for r, row := range rows {
		cellValues[r] = make([]string, len(cols))
		for c, col := range cols {
			var s string
			if row[col] != nil {
				s = fmt.Sprintf("%v", row[col])
			}
			cellValues[r][c] = s
			if w := runewidth.StringWidth(s); w > colWidths[c] {
				colWidths[c] = w
			}
		}
	}

	// Clamp total width.
	gap := 2
	totalWidth := 0
	for _, w := range colWidths {
		totalWidth += w
	}
	totalWidth += gap * (len(colWidths) - 1) // gaps between columns only
	if totalWidth > width {
		// Shrink widest columns proportionally.
		excess := totalWidth - width
		for excess > 0 {
			maxIdx, maxW := 0, 0
			for i, w := range colWidths {
				if w > maxW {
					maxIdx, maxW = i, w
				}
			}
			if maxW <= 3 {
				break
			}
			colWidths[maxIdx]--
			excess--
		}
	}

	// Render header.
	var headerParts []string
	for i, col := range cols {
		cell := runewidth.Truncate(col, colWidths[i], "...")
		cell += strings.Repeat(" ", colWidths[i]-runewidth.StringWidth(cell))
		headerParts = append(headerParts, headerStyle.Render(cell))
	}
	sep := strings.Repeat(" ", gap)
	lines := []string{strings.Join(headerParts, sep)}

	// Render rows.
	for _, cells := range cellValues {
		var parts []string
		for i, val := range cells {
			cell := runewidth.Truncate(val, colWidths[i], "...")
			cell += strings.Repeat(" ", colWidths[i]-runewidth.StringWidth(cell))
			parts = append(parts, cellStyle.Render(cell))
		}
		lines = append(lines, strings.Join(parts, sep))
	}
	return lines
}
