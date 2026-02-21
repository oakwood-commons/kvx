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

// ListViewModel holds state for the card-list rendering of an array of objects.
type ListViewModel struct {
	Items           []ListViewItem // Rendered items
	Selected        int            // Currently selected index
	ScrollTop       int            // First visible item index
	Filter          string         // Active real-time filter text (title+subtitle)
	SearchQuery     string         // Committed deep-search query (all fields)
	Width           int            // Available width
	Height          int            // Available height (content rows)
	CollectionTitle string         // Pre-computed border title (icon + collection name)
	Schema          *DisplaySchema // Back-reference for rendering
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

	lv := &ListViewModel{
		Items:  items,
		Width:  width,
		Height: height,
		Schema: schema,
	}

	// Pre-compute the border title from icon + collection title.
	var titleParts []string
	if schema.Icon != "" {
		titleParts = append(titleParts, schema.Icon)
	}
	if schema.CollectionTitle != "" {
		titleParts = append(titleParts, schema.CollectionTitle)
	}
	if len(titleParts) > 0 {
		lv.CollectionTitle = strings.Join(titleParts, " ")
	}

	return lv
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

// itemLineCount returns the number of content lines a single list item will render.
func itemLineCount(item ListViewItem, subtitleLines, maxSubWidth int, hasSecondary bool) int {
	count := 1 // title line
	if item.Subtitle != "" {
		wrapped := wrapAtWidth(item.Subtitle, maxSubWidth)
		subLines := strings.Split(wrapped, "\n")
		if len(subLines) > subtitleLines {
			subLines = subLines[:subtitleLines]
		}
		count += len(subLines)
	}
	if hasSecondary && len(item.Secondary) > 0 {
		count++
	}
	return count
}

// countVisibleItems returns how many items fit within availableHeight starting
// from startIdx, using actual per-item line counts rather than a fixed max.
func countVisibleItems(items []ListViewItem, startIdx, availableHeight, subtitleLines, maxSubWidth int, hasSecondary bool) int {
	usedLines := 0
	count := 0
	for i := startIdx; i < len(items); i++ {
		h := itemLineCount(items[i], subtitleLines, maxSubWidth, hasSecondary)
		if count > 0 {
			h++ // blank separator between items
		}
		if usedLines+h > availableHeight {
			break
		}
		usedLines += h
		count++
	}
	if count < 1 {
		count = 1
	}
	return count
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

	subtitleLines := 1
	if schema != nil && schema.List != nil && schema.List.SubtitleMaxLines > 0 {
		subtitleLines = schema.List.SubtitleMaxLines
	}
	hasSecondary := schema != nil && schema.List != nil && len(schema.List.SecondaryFields) > 0

	maxSubWidth := contentWidth - 2 // account for marker indent
	if maxSubWidth < 5 {
		maxSubWidth = 5
	}

	availableHeight := lv.Height
	if availableHeight < 3 {
		availableHeight = 3
	}

	// Adjust scrollTop to keep selected visible using actual item heights.
	if lv.Selected < lv.ScrollTop {
		lv.ScrollTop = lv.Selected
	}
	if lv.ScrollTop < 0 {
		lv.ScrollTop = 0
	}
	visibleCount := countVisibleItems(items, lv.ScrollTop, availableHeight, subtitleLines, maxSubWidth, hasSecondary)
	if lv.Selected >= lv.ScrollTop+visibleCount {
		// Scroll down until the selected item is visible.
		for lv.ScrollTop < lv.Selected {
			lv.ScrollTop++
			visibleCount = countVisibleItems(items, lv.ScrollTop, availableHeight, subtitleLines, maxSubWidth, hasSecondary)
			if lv.Selected < lv.ScrollTop+visibleCount {
				break
			}
		}
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

	// Collection title + icon are rendered in the panel border by
	// panelLayoutStateFromModel, so we skip them here.

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

// --- CustomView interface implementation ---

func (lv *ListViewModel) Title() string                { return lv.CollectionTitle }
func (lv *ListViewModel) FooterBar() string            { return "" }
func (lv *ListViewModel) HandlesSearch() bool          { return true }
func (lv *ListViewModel) Init() tea.Cmd                { return nil }
func (lv *ListViewModel) SearchTitle() string          { return "" } // caller sets based on ListPanelMode
func (lv *ListViewModel) FlashMessage() (string, bool) { return "", false }

func (lv *ListViewModel) Render(width, height int, noColor bool) string {
	lv.Width = width
	lv.Height = height
	return renderListView(lv, lv.Schema, noColor)
}

func (lv *ListViewModel) RowCount() (count int, selected int, label string) {
	total := listViewRowCount(lv)
	sel := lv.Selected + 1
	if sel > total {
		sel = total
	}
	filterSuffix := ""
	if lv.SearchQuery != "" {
		filterSuffix += " /" + lv.SearchQuery
	}
	if lv.Filter != "" {
		filterSuffix += " f:" + lv.Filter
	}
	return total, sel, fmt.Sprintf("list: %d/%d%s", sel, total, filterSuffix)
}

func (lv *ListViewModel) Update(_ tea.Msg) (CustomView, tea.Cmd) {
	return lv, nil // list view has no async messages
}
