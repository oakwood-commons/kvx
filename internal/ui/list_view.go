package ui

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	runewidth "github.com/mattn/go-runewidth"

	"github.com/oakwood-commons/kvx/internal/formatter"
)

// ListViewModel holds state for the card-list rendering of an array of objects.
type ListViewModel struct {
	Items       []ListViewItem // Rendered items
	Selected    int            // Currently selected index
	ScrollTop   int            // First visible item index
	Filter      string         // Active real-time filter text (title+subtitle)
	SearchQuery string         // Committed deep-search query (all fields)
	Width       int            // Available width
	Height      int            // Available height (content rows)
}

// ListViewItem is a pre-computed renderable card derived from an object.
type ListViewItem struct {
	Index      int      // Original array index
	Title      string   // Title text (from TitleField)
	Subtitle   string   // Subtitle text (from SubtitleField)
	Badges     []string // Badge labels (from BadgeFields)
	Secondary  []string // Secondary field values (from SecondaryFields)
	SearchText string   // Pre-computed concatenation of all field values for deep search
}

// buildListViewModel creates a ListViewModel from an array node using the display schema.
func buildListViewModel(node interface{}, schema *DisplaySchema, width, height int) *ListViewModel {
	arr, ok := node.([]interface{})
	if !ok || schema == nil || schema.List == nil || schema.List.TitleField == "" {
		return nil
	}

	items := make([]ListViewItem, 0, len(arr))
	for i, elem := range arr {
		obj, ok := elem.(map[string]interface{})
		if !ok {
			continue
		}
		item := ListViewItem{Index: i}
		item.Title = formatter.Stringify(obj[schema.List.TitleField])
		if schema.List.SubtitleField != "" {
			item.Subtitle = formatter.Stringify(obj[schema.List.SubtitleField])
		}
		for _, bf := range schema.List.BadgeFields {
			val := obj[bf]
			switch v := val.(type) {
			case []interface{}:
				for _, elem := range v {
					item.Badges = append(item.Badges, formatter.Stringify(elem))
				}
			case string:
				item.Badges = append(item.Badges, v)
			default:
				if val != nil {
					item.Badges = append(item.Badges, formatter.Stringify(val))
				}
			}
		}
		for _, sf := range schema.List.SecondaryFields {
			val := obj[sf]
			if val != nil {
				item.Secondary = append(item.Secondary, formatter.Stringify(val))
			}
		}

		// Build SearchText from all field values for deep search.
		var parts []string
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			parts = append(parts, formatter.Stringify(obj[k]))
		}
		item.SearchText = strings.Join(parts, " ")

		items = append(items, item)
	}

	return &ListViewModel{
		Items:  items,
		Width:  width,
		Height: height,
	}
}

// filterListItems returns items matching the active filter and/or committed search query.
// Filter matches title+subtitle (real-time). SearchQuery matches all fields (committed).
func filterListItems(lv *ListViewModel) []ListViewItem {
	if lv.Filter == "" && lv.SearchQuery == "" {
		return lv.Items
	}

	result := make([]ListViewItem, 0, len(lv.Items))
	for _, item := range lv.Items {
		// Deep search: match against all field values.
		if lv.SearchQuery != "" {
			if !strings.Contains(strings.ToLower(item.SearchText), strings.ToLower(lv.SearchQuery)) {
				continue
			}
		}
		// Filter: match against title + subtitle only.
		if lv.Filter != "" {
			lower := strings.ToLower(lv.Filter)
			if !strings.Contains(strings.ToLower(item.Title), lower) &&
				!strings.Contains(strings.ToLower(item.Subtitle), lower) {
				continue
			}
		}
		result = append(result, item)
	}
	return result
}

// renderListView renders the card-list as a string that fits into the data panel.
func renderListView(lv *ListViewModel, schema *DisplaySchema, noColor bool) string {
	if lv == nil || len(lv.Items) == 0 {
		return "  (empty)"
	}

	items := filterListItems(lv)
	if len(items) == 0 {
		return "  (no matches)"
	}

	th := CurrentTheme()
	contentWidth := lv.Width - 4 // borders + padding
	if contentWidth < 10 {
		contentWidth = 10
	}

	// Compute lines per item to know how many fit on screen
	subtitleLines := 1
	if schema != nil && schema.List != nil && schema.List.SubtitleMaxLines > 0 {
		subtitleLines = schema.List.SubtitleMaxLines
	}
	hasSecondary := schema != nil && schema.List != nil && len(schema.List.SecondaryFields) > 0
	linesPerItem := 1 + subtitleLines // title + subtitle
	if hasSecondary {
		linesPerItem++ // secondary metadata line
	}
	if linesPerItem < 2 {
		linesPerItem = 2
	}
	// Add blank line between items
	linesPerItemWithGap := linesPerItem + 1

	// Account for header lines (icon + collection title) when present
	headerLines := 0
	if schema != nil && (schema.Icon != "" || schema.CollectionTitle != "") {
		headerLines = 3 // header, item count, blank line
	}

	// Ensure scroll window
	availableHeight := lv.Height - headerLines
	if availableHeight < linesPerItemWithGap {
		availableHeight = linesPerItemWithGap
	}
	visibleCount := availableHeight / linesPerItemWithGap
	if visibleCount < 1 {
		visibleCount = 1
	}

	// Adjust scrollTop to keep selected visible
	if lv.Selected < lv.ScrollTop {
		lv.ScrollTop = lv.Selected
	}
	if lv.Selected >= lv.ScrollTop+visibleCount {
		lv.ScrollTop = lv.Selected - visibleCount + 1
	}
	if lv.ScrollTop < 0 {
		lv.ScrollTop = 0
	}

	// Styles
	titleStyle := lipgloss.NewStyle().Bold(true)
	subtitleStyle := lipgloss.NewStyle()
	selectedMarker := lipgloss.NewStyle()
	badgeStyle := lipgloss.NewStyle()
	if !noColor {
		titleStyle = titleStyle.Foreground(th.KeyColor)
		subtitleStyle = subtitleStyle.Foreground(th.ValueColor)
		selectedMarker = selectedMarker.Foreground(th.SelectedBG)
		badgeStyle = badgeStyle.Foreground(th.HeaderFG).Background(th.HeaderBG)
	}

	var lines []string

	// Header: icon + collection title
	if schema != nil && (schema.Icon != "" || schema.CollectionTitle != "") {
		header := ""
		if schema.Icon != "" {
			header = schema.Icon + " "
		}
		if schema.CollectionTitle != "" {
			header += schema.CollectionTitle
		}
		lines = append(lines, "  "+strings.TrimSpace(header))
		lines = append(lines, fmt.Sprintf("  %d items", len(items)))
		lines = append(lines, "")
	}

	endIdx := lv.ScrollTop + visibleCount
	if endIdx > len(items) {
		endIdx = len(items)
	}

	for i := lv.ScrollTop; i < endIdx; i++ {
		item := items[i]
		isSelected := i == lv.Selected

		// Selection indicator
		marker := "  "
		if isSelected {
			if noColor {
				marker = "│ "
			} else {
				marker = selectedMarker.Render("│") + " "
			}
		}

		// Title line
		title := item.Title
		if title == "" {
			title = fmt.Sprintf("[%d]", item.Index)
		}
		titleRendered := titleStyle.Render(title)

		// Badges inline after title
		badgeStr := ""
		if len(item.Badges) > 0 {
			badges := make([]string, 0, len(item.Badges))
			for _, b := range item.Badges {
				if noColor {
					badges = append(badges, " "+b+" ")
				} else {
					badges = append(badges, badgeStyle.Render(" "+b+" "))
				}
			}
			badgeStr = " " + strings.Join(badges, " ")
		}

		titleLine := marker + titleRendered + badgeStr
		// Clamp to width
		if runewidth.StringWidth(stripANSI(titleLine)) > contentWidth+2 {
			titleLine = clampANSITextWidth(titleLine, contentWidth+2)
		}
		lines = append(lines, titleLine)

		// Subtitle line(s)
		if item.Subtitle != "" {
			sub := item.Subtitle
			maxSubWidth := contentWidth - 2 // account for marker indent
			if maxSubWidth < 5 {
				maxSubWidth = 5
			}
			// Wrap subtitle to subtitleLines
			wrapped := wrapAtWidth(sub, maxSubWidth)
			subLines := strings.Split(wrapped, "\n")
			if len(subLines) > subtitleLines {
				subLines = subLines[:subtitleLines]
				// Add ellipsis to last line
				last := subLines[len(subLines)-1]
				if runewidth.StringWidth(last) > maxSubWidth-3 {
					last = runewidth.Truncate(last, maxSubWidth-3, "") + "..."
				} else {
					last += "..."
				}
				subLines[len(subLines)-1] = last
			}
			for _, sl := range subLines {
				rendered := subtitleStyle.Render(sl)
				lines = append(lines, "  "+rendered)
			}
		}

		// Secondary fields line
		if len(item.Secondary) > 0 {
			secondaryLine := "    " + subtitleStyle.Render(strings.Join(item.Secondary, " · "))
			if runewidth.StringWidth(stripANSI(secondaryLine)) > contentWidth+2 {
				secondaryLine = clampANSITextWidth(secondaryLine, contentWidth+2)
			}
			lines = append(lines, secondaryLine)
		}

		// Blank line between items
		if i < endIdx-1 {
			lines = append(lines, "")
		}
	}

	return strings.Join(lines, "\n")
}

// wrapAtWidth wraps text at the given width, breaking on word boundaries.
func wrapAtWidth(text string, width int) string {
	if width <= 0 || runewidth.StringWidth(text) <= width {
		return text
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	var lines []string
	current := words[0]
	for _, word := range words[1:] {
		test := current + " " + word
		if runewidth.StringWidth(test) > width {
			lines = append(lines, current)
			current = word
		} else {
			current = test
		}
	}
	lines = append(lines, current)
	return strings.Join(lines, "\n")
}

// listViewRowCount returns the number of displayable items (for footer label).
func listViewRowCount(lv *ListViewModel) int {
	if lv == nil {
		return 0
	}
	return len(filterListItems(lv))
}

// isHomogeneousObjectArray checks if a node is an array where all elements are maps.
func isHomogeneousObjectArray(node interface{}) bool {
	arr, ok := node.([]interface{})
	if !ok || len(arr) == 0 {
		return false
	}
	for _, elem := range arr {
		if _, ok := elem.(map[string]interface{}); !ok {
			return false
		}
	}
	return true
}

// collectObjectKeys returns sorted keys from a map, excluding hidden fields.
func collectObjectKeys(obj map[string]interface{}, hidden []string) []string {
	hiddenSet := make(map[string]bool, len(hidden))
	for _, h := range hidden {
		hiddenSet[h] = true
	}
	keys := make([]string, 0, len(obj))
	for k := range obj {
		if !hiddenSet[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}
