package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	runewidth "github.com/mattn/go-runewidth"

	"github.com/oakwood-commons/kvx/internal/completion"
)

// categoryOrder defines the display order for function categories in the palette.
var categoryOrder = []string{
	"conversion",
	"string",
	"list",
	"map",
	"math",
	"encoding",
	"datetime",
	"regex",
	"general",
}

// FunctionPaletteModel represents the function palette overlay component.
// Triggered via Ctrl+Space in expression mode, it displays all available
// CEL functions grouped by category with search filtering.
type FunctionPaletteModel struct {
	Visible          bool
	AllFunctions     []completion.FunctionMetadata // All functions
	Categories       []string                      // Ordered categories with functions
	FuncsByCategory  map[string][]completion.FunctionMetadata
	SelectedCategory int    // Currently selected category tab
	SelectedIndex    int    // Selected function within category
	SearchQuery      string // Filter query typed while palette is open
	MethodsOnly      bool   // When true, show only methods (context-aware: expression ends with ".")
	Width            int
	Height           int
	NoColor          bool
}

// NewFunctionPaletteModel creates a new function palette from the completion engine.
func NewFunctionPaletteModel() FunctionPaletteModel {
	return FunctionPaletteModel{
		FuncsByCategory: make(map[string][]completion.FunctionMetadata),
		Width:           80,
		Height:          24,
	}
}

// LoadFunctions populates the palette with functions from the completion registry.
// The registry provides deduplicated, sorted functions grouped by category.
func (m *FunctionPaletteModel) LoadFunctions(registry *completion.FunctionRegistry) {
	if registry == nil {
		return
	}

	// Get all deduplicated functions from the registry
	m.AllFunctions = registry.GetAll()
	m.FuncsByCategory = make(map[string][]completion.FunctionMetadata)

	// Populate FuncsByCategory from registry
	for _, cat := range registry.GetCategories() {
		m.FuncsByCategory[cat] = registry.GetByCategory(cat)
	}

	// Build ordered category list using our display order preference,
	// only including categories that have functions.
	m.Categories = make([]string, 0, len(categoryOrder))
	for _, cat := range categoryOrder {
		if len(m.FuncsByCategory[cat]) > 0 {
			m.Categories = append(m.Categories, cat)
		}
	}
	// Add any discovered categories not in categoryOrder.
	seen := make(map[string]bool, len(categoryOrder))
	for _, c := range categoryOrder {
		seen[c] = true
	}
	for cat := range m.FuncsByCategory {
		if !seen[cat] {
			m.Categories = append(m.Categories, cat)
		}
	}
}

// Toggle toggles visibility of the palette.
func (m *FunctionPaletteModel) Toggle() {
	m.Visible = !m.Visible
	if m.Visible {
		m.SearchQuery = ""
		m.SelectedIndex = 0
		m.MethodsOnly = false
	}
}

// OpenMethodsOnly opens the palette in methods-only mode, useful when
// the expression ends with "." and the user wants to discover available methods.
func (m *FunctionPaletteModel) OpenMethodsOnly() {
	m.Visible = true
	m.SearchQuery = ""
	m.SelectedIndex = 0
	m.MethodsOnly = true
}

// Close hides the palette.
func (m *FunctionPaletteModel) Close() {
	m.Visible = false
	m.SearchQuery = ""
	m.SelectedIndex = 0
	m.MethodsOnly = false
}

// filteredFunctions returns functions for the current category filtered by search query.
// When MethodsOnly is true, returns only methods across all categories.
func (m *FunctionPaletteModel) filteredFunctions() []completion.FunctionMetadata {
	if m.MethodsOnly {
		return m.methodsFiltered()
	}

	if len(m.Categories) == 0 {
		return nil
	}

	catIdx := m.SelectedCategory
	if catIdx >= len(m.Categories) {
		catIdx = 0
	}
	cat := m.Categories[catIdx]
	funcs := m.FuncsByCategory[cat]

	if m.SearchQuery == "" {
		return funcs
	}

	query := strings.ToLower(m.SearchQuery)
	filtered := make([]completion.FunctionMetadata, 0)
	for _, fn := range funcs {
		if strings.Contains(strings.ToLower(fn.Name), query) ||
			strings.Contains(strings.ToLower(fn.Description), query) {
			filtered = append(filtered, fn)
		}
	}
	return filtered
}

// methodsFiltered returns all methods, optionally filtered by search query.
func (m *FunctionPaletteModel) methodsFiltered() []completion.FunctionMetadata {
	var result []completion.FunctionMetadata
	query := strings.ToLower(m.SearchQuery)
	for _, fn := range m.AllFunctions {
		if !fn.IsMethod {
			continue
		}
		if query == "" ||
			strings.Contains(strings.ToLower(fn.Name), query) ||
			strings.Contains(strings.ToLower(fn.Description), query) {
			result = append(result, fn)
		}
	}
	return result
}

// allFilteredFunctions returns functions across ALL categories matching the search query.
// When a search query is active, this shows results from every category.
func (m *FunctionPaletteModel) allFilteredFunctions() []completion.FunctionMetadata {
	if m.MethodsOnly {
		return m.methodsFiltered()
	}
	if m.SearchQuery == "" {
		return m.filteredFunctions()
	}
	query := strings.ToLower(m.SearchQuery)
	var result []completion.FunctionMetadata
	for _, cat := range m.Categories {
		for _, fn := range m.FuncsByCategory[cat] {
			if strings.Contains(strings.ToLower(fn.Name), query) ||
				strings.Contains(strings.ToLower(fn.Description), query) {
				result = append(result, fn)
			}
		}
	}
	return result
}

// SelectedFunction returns the currently selected function, or nil if none.
func (m *FunctionPaletteModel) SelectedFunction() *completion.FunctionMetadata {
	funcs := m.allFilteredFunctions()
	if len(funcs) == 0 {
		return nil
	}
	idx := m.SelectedIndex
	if idx >= len(funcs) {
		idx = len(funcs) - 1
	}
	if idx < 0 {
		idx = 0
	}
	return &funcs[idx]
}

// MoveUp moves the selection up.
func (m *FunctionPaletteModel) MoveUp() {
	funcs := m.allFilteredFunctions()
	if len(funcs) == 0 {
		return
	}
	m.SelectedIndex--
	if m.SelectedIndex < 0 {
		m.SelectedIndex = len(funcs) - 1
	}
}

// MoveDown moves the selection down.
func (m *FunctionPaletteModel) MoveDown() {
	funcs := m.allFilteredFunctions()
	if len(funcs) == 0 {
		return
	}
	m.SelectedIndex++
	if m.SelectedIndex >= len(funcs) {
		m.SelectedIndex = 0
	}
}

// NextCategory switches to the next category tab.
func (m *FunctionPaletteModel) NextCategory() {
	if len(m.Categories) == 0 {
		return
	}
	m.SelectedCategory = (m.SelectedCategory + 1) % len(m.Categories)
	m.SelectedIndex = 0
}

// PrevCategory switches to the previous category tab.
func (m *FunctionPaletteModel) PrevCategory() {
	if len(m.Categories) == 0 {
		return
	}
	m.SelectedCategory--
	if m.SelectedCategory < 0 {
		m.SelectedCategory = len(m.Categories) - 1
	}
	m.SelectedIndex = 0
}

// SelectFunction finds a function by name and selects it in the palette.
// It switches to the correct category and sets the selection index.
// Returns true if the function was found.
func (m *FunctionPaletteModel) SelectFunction(name string) bool {
	if name == "" {
		return false
	}
	lower := strings.ToLower(name)

	// In methods-only mode, search the flat methods list.
	if m.MethodsOnly {
		for i, fn := range m.methodsFiltered() {
			if strings.ToLower(fn.Name) == lower {
				m.SelectedIndex = i
				return true
			}
		}
		return false
	}

	// Search each category for the function.
	for catIdx, cat := range m.Categories {
		for fnIdx, fn := range m.FuncsByCategory[cat] {
			if strings.ToLower(fn.Name) == lower {
				m.SelectedCategory = catIdx
				m.SelectedIndex = fnIdx
				return true
			}
		}
	}
	return false
}

// HandleSearchKey processes a typed character for filtering.
func (m *FunctionPaletteModel) HandleSearchKey(key string) {
	if key == "backspace" {
		if len(m.SearchQuery) > 0 {
			m.SearchQuery = m.SearchQuery[:len(m.SearchQuery)-1]
			m.SelectedIndex = 0
		}
		return
	}
	// Only accept printable single characters.
	if len(key) == 1 && key[0] >= ' ' && key[0] <= '~' {
		m.SearchQuery += key
		m.SelectedIndex = 0
	}
}

// InsertText returns the text to insert when a function is selected.
// For global functions: "functionName("
// For methods: ".methodName("
func InsertText(fn *completion.FunctionMetadata) string {
	if fn == nil {
		return ""
	}
	name := fn.Name
	// Strip namespace prefix for display (e.g., "math.abs" stays "math.abs")
	if fn.IsMethod {
		return "." + name + "("
	}
	return name + "("
}

// View renders the palette overlay.
func (m *FunctionPaletteModel) View() string {
	if !m.Visible || len(m.Categories) == 0 {
		return ""
	}

	th := CurrentTheme()
	panelBorder := borderForTheme(th)

	// Determine available width for the palette (use most of the window).
	paletteWidth := m.Width
	if paletteWidth <= 0 {
		paletteWidth = 80
	}
	innerWidth := paletteWidth - 2 // borders only (panelWithTitle uses width-2)
	if innerWidth < 20 {
		innerWidth = 20
	}

	// Fixed palette height: ~40% of window, clamped to [10, 18].
	fixedInner := m.Height * 2 / 5
	if fixedInner < 10 {
		fixedInner = 10
	}
	if fixedInner > 18 {
		fixedInner = 18
	}

	// Category tabs (or "Methods" header in methods-only mode).
	tabLine := ""
	if m.MethodsOnly {
		label := "▸ Methods"
		if !m.NoColor {
			label = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render(label)
		}
		tabLine = label
	} else {
		tabLine = m.renderCategoryTabs(innerWidth)
	}

	// Search indicator.
	searchLine := ""
	if m.SearchQuery != "" {
		searchLine = fmt.Sprintf("🔍 %s", m.SearchQuery)
		if len(searchLine) > innerWidth {
			searchLine = searchLine[:innerWidth]
		}
	}

	// Fixed layout: tabs(1) + [search(1)] + list(up to 6) + blank + detail(3: sig+desc+examples) + blank + hint
	overhead := 6 // blank + sig + desc + examples + blank + hint
	if searchLine != "" {
		overhead++
	}
	maxVisible := fixedInner - 1 - overhead // 1 for tab line
	if maxVisible < 3 {
		maxVisible = 3
	}
	if maxVisible > 6 {
		maxVisible = 6
	}

	// Function list.
	funcs := m.allFilteredFunctions()

	// Window the function list around selected index.
	startIdx := 0
	if m.SelectedIndex >= maxVisible {
		startIdx = m.SelectedIndex - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(funcs) {
		endIdx = len(funcs)
		startIdx = endIdx - maxVisible
		if startIdx < 0 {
			startIdx = 0
		}
	}

	funcLines := make([]string, 0, maxVisible)
	for i := startIdx; i < endIdx; i++ {
		fn := funcs[i]
		line := m.renderFunctionLine(fn, i == m.SelectedIndex, innerWidth)
		funcLines = append(funcLines, line)
	}

	// Detail panel for selected function.
	detailLine := ""
	if sel := m.SelectedFunction(); sel != nil {
		detailLine = m.renderFunctionDetail(sel, innerWidth)
	}

	// Assemble content.
	var lines []string
	lines = append(lines, tabLine)
	if searchLine != "" {
		lines = append(lines, searchLine)
	}
	if len(funcLines) == 0 {
		noMatch := "  (no matching functions)"
		if !m.NoColor {
			noMatch = lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Render(noMatch)
		}
		lines = append(lines, noMatch)
	} else {
		lines = append(lines, funcLines...)
	}
	if detailLine != "" {
		lines = append(lines, "")
		// Split detail into separate lines so line count is accurate for padding.
		lines = append(lines, strings.Split(detailLine, "\n")...)
	}
	// Navigation hint (always pinned to the last line).
	hint := "↑↓ navigate  Tab category  Enter insert  Esc close"
	if m.SearchQuery == "" {
		hint = "↑↓ navigate  Tab category  Enter insert  type to filter  Esc close"
	}
	if !m.NoColor {
		hint = lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Render(hint)
	}

	// Pad content so the hint always lands on the last line of fixedInner.
	targetContentLines := fixedInner - 1 // reserve 1 line for the hint
	for len(lines) < targetContentLines {
		lines = append(lines, "")
	}
	if len(lines) > targetContentLines {
		lines = lines[:targetContentLines]
	}
	lines = append(lines, hint)

	content := strings.Join(lines, "\n")

	// Use panelWithTitle so the palette border matches the data and expression panels.
	title := "Functions (Ctrl+Space)"
	totalHeight := fixedInner + 2 // inner lines + top/bottom borders
	rendered := panelWithTitle(title, content, paletteWidth, totalHeight, panelBorder, m.NoColor)

	return strings.TrimRight(rendered, "\n") + "\n"
}

// renderCategoryTabs renders the category tab bar, truncated to fit within width.
func (m *FunctionPaletteModel) renderCategoryTabs(width int) string {
	if len(m.Categories) == 0 {
		return ""
	}
	var parts []string
	for i, cat := range m.Categories {
		label := cat
		count := len(m.FuncsByCategory[cat])
		tab := fmt.Sprintf(" %s(%d) ", label, count)
		if i == m.SelectedCategory && m.SearchQuery == "" {
			if !m.NoColor {
				th := CurrentTheme()
				tab = lipgloss.NewStyle().
					Foreground(th.HeaderFG).
					Bold(true).
					Underline(true).
					Render(tab)
			} else {
				tab = "[" + tab + "]"
			}
		} else {
			if !m.NoColor {
				tab = lipgloss.NewStyle().
					Foreground(lipgloss.Color("243")).
					Render(tab)
			}
		}
		parts = append(parts, tab)
	}

	line := strings.Join(parts, "")
	// If search is active, dim the tabs and highlight "all" instead.
	if m.SearchQuery != "" {
		allCount := len(m.allFilteredFunctions())
		suffix := fmt.Sprintf("  [all: %d matches]", allCount)
		if !m.NoColor {
			suffix = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(suffix)
		}
		line += suffix
	}
	// Truncate the tab line to fit in no-color mode.
	if m.NoColor && runewidth.StringWidth(line) > width {
		line = runewidth.Truncate(line, width, "…")
	}
	return line
}

// renderFunctionLine renders a single function entry in the list.
func (m *FunctionPaletteModel) renderFunctionLine(fn completion.FunctionMetadata, selected bool, width int) string {
	prefix := "  "
	if selected {
		prefix = "▸ "
	}

	tag := "global"
	if fn.IsMethod {
		tag = "method"
	}
	nameStr := fn.Name + "()"

	// Build: "▸ name()  [tag]  description…"
	fixedPart := fmt.Sprintf("%s%-16s [%s]", prefix, nameStr, tag)
	fixedWidth := runewidth.StringWidth(fixedPart)

	desc := fn.Description
	descSpace := width - fixedWidth - 2 // 2 for spacing before desc
	if descSpace > 3 && desc != "" {
		if len(desc) > descSpace {
			desc = desc[:descSpace-1] + "…"
		}
		fixedPart += "  " + desc
	}

	// Final truncation safety net.
	if runewidth.StringWidth(fixedPart) > width {
		fixedPart = runewidth.Truncate(fixedPart, width, "…")
	}

	if !m.NoColor && selected {
		th := CurrentTheme()
		fixedPart = lipgloss.NewStyle().
			Foreground(th.HelpKey).
			Bold(true).
			Render(fixedPart)
	}

	return fixedPart
}

// renderFunctionDetail renders the detail panel for the selected function.
func (m *FunctionPaletteModel) renderFunctionDetail(fn *completion.FunctionMetadata, width int) string {
	if fn == nil {
		return ""
	}

	var parts []string

	// Signature.
	sig := fn.Signature
	if sig == "" {
		sig = fn.Name + "()"
	}
	if !m.NoColor {
		sig = lipgloss.NewStyle().Bold(true).Render(sig)
	}
	parts = append(parts, sig)

	// Description.
	if fn.Description != "" {
		desc := fn.Description
		if len(desc) > width-2 {
			desc = desc[:width-3] + "…"
		}
		parts = append(parts, desc)
	}

	// Examples (joined on one line with | separator).
	if len(fn.Examples) > 0 {
		clean := make([]string, 0, len(fn.Examples))
		for _, ex := range fn.Examples {
			ex = strings.TrimSpace(ex)
			if ex != "" {
				clean = append(clean, ex)
			}
		}
		if len(clean) > 0 {
			exLine := strings.Join(clean, " | ")
			if len(exLine) > width-4 {
				exLine = exLine[:width-5] + "\u2026"
			}
			if !m.NoColor {
				exLine = lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true).Render("  " + exLine)
			} else {
				exLine = "  " + exLine
			}
			parts = append(parts, exLine)
		}
	}

	return strings.Join(parts, "\n")
}
