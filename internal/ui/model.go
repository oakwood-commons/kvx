package ui

import (
	"context"
	"fmt"
	"os/exec"
	"reflect"
	"runtime"
	rdebug "runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	runewidth "github.com/mattn/go-runewidth"

	celhelper "github.com/oakwood-commons/kvx/internal/cel"
	"github.com/oakwood-commons/kvx/internal/completion"
	"github.com/oakwood-commons/kvx/internal/formatter"
	"github.com/oakwood-commons/kvx/internal/navigator"
	"github.com/oakwood-commons/kvx/pkg/intellisense"
)

type tabCandidate struct {
	name   string
	idx    int
	isFunc bool
}

//nolint:gocritic // nested loops and branches are acceptable for suggestion building
func (m *Model) buildTabCandidates(token string) []tabCandidate {
	candidates := make([]tabCandidate, 0, len(m.FilteredSuggestions))
	seenFuncs := make(map[string]bool)
	keys := make([]tabCandidate, 0, len(m.FilteredSuggestions))
	funcs := make([]tabCandidate, 0, len(m.FilteredSuggestions))

	for i, s := range m.FilteredSuggestions {
		isFunction := strings.Contains(s, "(") || strings.Contains(s, " - ")
		if token != "" {
			if isFunction {
				name := extractFunctionName(s)
				if !strings.HasPrefix(strings.ToLower(name), token) {
					continue
				}
			} else {
				cand := matchKeyToken(s)
				if !strings.HasPrefix(strings.ToLower(cand), token) {
					continue
				}
			}
		}
		if isFunction {
			norm := normalizeFunctionName(extractFunctionName(s))
			if norm != "" && seenFuncs[norm] {
				continue
			}
			if norm != "" {
				seenFuncs[norm] = true
			}
			funcs = append(funcs, tabCandidate{name: s, idx: i, isFunc: true})
		} else {
			keys = append(keys, tabCandidate{name: s, idx: i, isFunc: false})
		}
	}
	candidates = append(candidates, keys...)
	candidates = append(candidates, funcs...)
	return candidates
}

// Model represents the minimal root Bubble Tea UI model for kvx.
// It delegates to RootModel for core navigation/expression/filtering logic.
// Search (Task 7) and Help (Task 8) logic are deferred and kept here temporarily.
type Model struct {
	Tbl                        table.Model
	AllRows                    []table.Row
	AllRowKeys                 []string // Untruncated keys aligned with AllRows
	PathInput                  textinput.Model
	SearchInput                textinput.Model
	Status                     StatusModel      // Status bar component
	Footer                     FooterModel      // Footer component with key bindings
	Help                       HelpModel        // Help overlay component
	Debug                      DebugModel       // Debug bar component
	SuggestionsComponent       SuggestionsModel // Suggestions dropdown component
	Layout                     *LayoutManager   // Layout manager for consistent spacing
	Node                       interface{}
	Root                       interface{}
	Path                       string
	PathKeys                   []string
	CursorByPath               map[string]int // Remember cursor positions per path for back navigation
	DebugVersion               string
	ErrMsg                     string
	ErrSticky                  bool
	ErrStickyInput             string
	LastKey                    string
	InputFocused               bool
	ExprDisplay                string                          // Last committed expression/path for data panel label in expr mode
	ExprType                   string                          // Type of last evaluated expression result
	AllowEditInput             bool                            // Whether path input can be focused/edited
	InfoPopup                  string                          // Optional info popup text
	ShowInfoPopup              bool                            // Whether to show the info popup
	InfoPopupAnchor            string                          // Where to anchor the info popup (inline/top)
	InfoPopupJustify           string                          // Alignment for popup text
	InfoPopupModal             bool                            // Whether popup is modal (layout-reserved)
	InfoPopupPermanent         bool                            // Whether popup ignores hide requests (e.g., Esc)
	InfoPopupEnabled           bool                            // Whether popup is enabled
	HelpPopupText              string                          // Optional popup shown above help overlay
	HelpPopupJustify           string                          // Alignment for help popup text
	HelpPopupAnchor            string                          // Anchor for help popup (inline|top)
	AllowFilter                bool                            // Whether type-ahead filter is enabled
	AllowSuggestions           bool                            // Whether suggestions/intellisense are shown
	AllowIntellisense          bool                            // Whether to show CEL/intellisense dropdown hints
	HelpAboutTitle             string                          // About section title
	HelpAboutLines             []string                        // About section lines
	HelpAboutAlign             string                          // About section alignment
	KeyHeader                  string                          // Table header for key column
	ValueHeader                string                          // Table header for value column
	InputPromptUnfocused       string                          // Input prompt when unfocused
	InputPromptFocused         string                          // Input prompt when focused
	InputPlaceholder           string                          // Input placeholder text
	HelpNavigationDescriptions map[string]string               // Custom help navigation descriptions
	StatusType                 string                          // "error", "success", or ""
	PrintedInTUI               bool                            // Whether CLI output was already printed inside TUI
	Suggestions                []string                        // Available CEL suggestions
	FilteredSuggestions        []string                        // Filtered suggestions based on input
	SelectedSuggestion         int                             // Currently selected suggestion index
	ShowSuggestions            bool                            // Whether to show suggestion dropdown
	CompletionEngine           *completion.CompletionEngine    // Completion engine for expression mode
	SuggestionSummary          string                          // One-shot summary of CEL functions after typing a trailing dot
	ShowSuggestionSummary      bool                            // Whether to render the trailing-dot summary in the status bar
	FunctionHelpOverrides      map[string]string               // Optional function help overrides (normalized function names)
	FunctionExamples           map[string]FunctionExampleValue // Optional function examples (normalized function names)
	ShowPanelTitle             bool                            // Whether to render panel title/header
	HelpVisible                bool                            // Whether inline help is shown (F1)
	LastTabPosition            int                             // Track cursor position on last tab to detect cycling
	LastTabToken               string                          // Track initial token for tab cycling
	LastTabTokenIsFunc         bool                            // Track whether tab cycling is for functions
	DebugMode                  bool                            // Show debug info in status bar
	DebugTransient             string                          // Extra transient debug info (e.g., navigate path)
	LastDebugOutput            string                          // Cached debug output to prevent flicker
	LastDebugValues            string                          // Hash of debug values to detect changes
	WinWidth                   int                             // Window width for dynamic layout
	WinHeight                  int                             // Window height for dynamic layout
	DesiredWinWidth            int                             // Forced width when provided via CLI flags
	DesiredWinHeight           int                             // Forced height when provided via CLI flags
	ForceWindowSize            bool                            // Whether to ignore terminal resize events and stick to desired size
	KeyColWidth                int                             // Current width for the KEY column
	ConfiguredKeyColWidth      int                             // Desired/capped key column width from config (<=0 means default)
	ValueColWidth              int                             // Computed width for the VALUE column
	ConfiguredValueColWidth    int                             // Configured value column width (0 = auto-calculate)
	TableHeight                int                             // Computed height for the table rows area
	NoColor                    bool                            // Disable color output
	FilterBuffer               string                          // Type-ahead filter buffer
	FilterActive               bool                            // Whether type-ahead filter is active
	FilteredSuggestionRows     []table.Row                     // Filtered rows for path input suggestions (temporary)
	SuggestionFilterActive     bool                            // Whether suggestion filtering is active
	PendingCLIExpr             string                          // Expression to print after quitting TUI (for real terminal output)
	AdvancedSearchActive       bool                            // Whether F3 advanced search is active
	AdvancedSearchQuery        string                          // Current advanced search query
	AdvancedSearchResults      []SearchResult                  // Search results with full paths
	AdvancedSearchBasePath     string                          // Base path where search was initiated (for combining with result paths)
	AdvancedSearchCommitted    bool                            // Whether Enter was pressed to commit deep search (vs real-time filter)
	PreviousNode               interface{}                     // Previous node before search (for restoring on Esc)
	PreviousPath               string                          // Previous path before search (for restoring on Esc)
	PreviousAllRows            []table.Row                     // Previous rows before search (for restoring on Esc)
	SearchContextActive        bool                            // Whether we navigated from search (right arrow maintains context)
	SearchContextResults       []SearchResult                  // Search results to restore when navigating back
	SearchContextQuery         string                          // Search query to restore when navigating back
	SearchContextBasePath      string                          // Base path from search context (for combining with result paths)
	DebugEvents                []DebugEvent                    // Collected debug events for post-exit logging
	TruncateTableCells         bool                            // Whether to pre-truncate cell content to column widths
	AppName                    string                          // App title for panel layout rendering
	HelpTitle                  string                          // Help title for panel layout rendering
	HelpText                   string                          // Help text for panel layout rendering
	KeyMode                    KeyMode                         // Keybinding mode: vim, emacs, or function
	PendingVimKey              string                          // Pending key for multi-key sequences (e.g., "g" for gg)

	// Map filter mode ('f' key) - real-time filter of current map's keys only
	MapFilterActive bool            // Whether map filter mode is active
	MapFilterQuery  string          // Current map filter query
	MapFilterInput  textinput.Model // Text input for map filter mode

	// Performance settings
	SearchDebounceID     int    // Counter for debounce message correlation
	SearchDebounceMs     int    // Debounce delay in milliseconds (from PerformanceConfig)
	SearchResultLimit    int    // Max results for deep search (from PerformanceConfig)
	SearchResultsLimited bool   // Whether search results were truncated due to limit
	SearchPendingQuery   string // Query pending debounce timer
	VirtualScrolling     bool   // Whether virtual scrolling is enabled
	ScrollBufferRows     int    // Extra rows to render above/below viewport
}

// DebugEvent captures a debug message with a timestamp for post-exit logging.
type DebugEvent struct {
	Time    time.Time
	Message string
}

// SearchDebounceMsg is sent after a debounce delay to trigger search execution.
// The ID is compared against SearchDebounceID to ensure only the latest query is executed.
// Currently unused as top-level filtering is fast enough to run synchronously.
type SearchDebounceMsg struct {
	ID    int    // Correlates with Model.SearchDebounceID
	Query string // The query to search for
}

// debouncedSearch returns a tea.Cmd that waits for the debounce delay then sends SearchDebounceMsg.
// Currently unused as top-level filtering is fast enough to run synchronously.
// Deep search only runs on Enter, so no debounce is needed.
var _ = debouncedSearch // silence unused lint

func debouncedSearch(id int, query string, delayMs int) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
		return SearchDebounceMsg{ID: id, Query: query}
	}
}

// SearchResult represents a search result with full path information
type SearchResult struct {
	FullPath string      // Full path to this key-value pair (e.g., "user.name" or "items[0].tags")
	Key      string      // Local key name (e.g., "name" or "[0]")
	Value    string      // Stringified value for display
	Node     interface{} // The actual node value (for navigation)
}

func (m *Model) clearSearchState() {
	m.AdvancedSearchActive = false
	m.AdvancedSearchQuery = ""
	m.AdvancedSearchResults = []SearchResult{}
	m.AdvancedSearchBasePath = ""
	m.AdvancedSearchCommitted = false
	m.SearchInput.SetValue("")
	m.SearchInput.SetCursor(0)
	m.SearchInput.Blur()
	m.clearSearchContext()
}

func (m *Model) clearSearchContext() {
	m.SearchContextActive = false
	m.SearchContextResults = nil
	m.SearchContextQuery = ""
	m.SearchContextBasePath = ""
}

func (m *Model) setStickyError(msg string) {
	m.ErrMsg = msg
	m.StatusType = "error"
	m.ErrSticky = true
	m.ErrStickyInput = m.PathInput.Value()
}

func (m *Model) clearError() {
	m.ErrMsg = ""
	m.StatusType = ""
	m.ErrSticky = false
	m.ErrStickyInput = ""
}

func (m *Model) clearErrorUnlessSticky() {
	if m.ErrSticky {
		return
	}
	m.clearError()
}

func (m *Model) clearStickyErrorIfInputChanged() {
	if !m.ErrSticky {
		return
	}
	if m.PathInput.Value() == m.ErrStickyInput {
		return
	}
	m.clearError()
}

// navigateBack handles back navigation (going to parent node).
// This is used by vim 'h' key and left arrow.
func (m *Model) navigateBack() (tea.Model, tea.Cmd) {
	// For CEL expressions (containing filter, map, etc.), going back returns to root
	if strings.Contains(m.Path, "filter(") || strings.Contains(m.Path, "map(") {
		newModel := m.NavigateTo(m.Root, "")
		newModel.applyLayout(true)
		return newModel, nil
	}
	// For regular paths, remove the last segment from the path string
	if m.Path != "" {
		newPath := removeLastSegment(m.Path)
		var newNode interface{}
		var err error
		if newPath == "" {
			newNode = m.Root
		} else {
			newNode, err = navigator.Resolve(m.Root, newPath)
			if err != nil {
				m.ErrMsg = fmt.Sprintf("Error: %v", err)
				return m, nil
			}
		}
		newModel := m.NavigateTo(newNode, newPath)
		newModel.applyLayout(true)
		newModel.restoreCursorForPath(newModel.Path)
		return newModel, nil
	}
	return m, nil
}

// navigateForward handles forward navigation (drilling into selected item).
// This is used by vim 'l' key and right arrow.
func (m *Model) navigateForward() (tea.Model, tea.Cmd) {
	selectedKey, ok := m.selectedRowKey()
	if !ok {
		return m, nil
	}

	// If user drills into a scalar value, do nothing
	if selectedKey == "(value)" {
		return m, nil
	}

	// Navigate to the child node
	newPath := buildPathWithKey(m.Path, selectedKey)
	newNode, err := navigator.Resolve(m.Root, newPath)
	if err != nil {
		m.ErrMsg = fmt.Sprintf("Error: %v", err)
		m.StatusType = "error"
		return m, nil
	}

	// Save cursor position for this path before navigating
	m.storeCursorForPath(m.Path)

	newModel := m.NavigateTo(newNode, normalizePathForModel(newPath))
	newModel.PathKeys = parsePathKeys(newModel.Path)
	newModel.applyLayout(true)
	return newModel, nil
}

// styleRows applies consistent styling to table rows: cyan keys, gray values
// It also truncates content to fit within the expected column widths
func styleRows(stringRows [][]string) []table.Row {
	return styleRowsWithWidths(stringRows, 30, 60)
}

// styleRowsWithWidths truncates and styles rows to match column widths
func styleRowsWithWidths(stringRows [][]string, keyWidth, valueWidth int) []table.Row {
	rows := make([]table.Row, len(stringRows))

	// Truncate to the exact column width - the table will handle its own padding.
	// Ensure that the widths are at least 1.
	keyContentWidth := keyWidth
	valueContentWidth := valueWidth
	if keyContentWidth < 1 {
		keyContentWidth = 1
	}
	if valueContentWidth < 1 {
		valueContentWidth = 1
	}

	for i, sr := range stringRows {
		row := make([]string, len(sr))
		// Truncate and copy key column
		if len(sr) > 0 {
			keyCell := truncateString(sr[0], keyContentWidth)
			row[0] = padToWidth(keyCell, keyContentWidth)
		}
		// Truncate value column (no styling - table cell style handles it)
		if len(sr) > 1 {
			valCell := truncateString(sr[1], valueContentWidth)
			row[1] = padToWidth(valCell, valueContentWidth)
		}
		rows[i] = table.Row(row)
	}
	return rows
}

func extractRowKeys(rows [][]string) []string {
	keys := make([]string, len(rows))
	for i, r := range rows {
		if len(r) > 0 {
			keys[i] = r[0]
		}
	}
	return keys
}

// styleRowsNoTruncation copies rows without truncating; the table component will clip to column width.
func styleRowsNoTruncation(stringRows [][]string, keyWidth, valueWidth int) []table.Row {
	rows := make([]table.Row, len(stringRows))
	for i, sr := range stringRows {
		row := make([]string, len(sr))
		if len(sr) > 0 {
			row[0] = padToWidth(truncateNoEllipsis(sr[0], keyWidth), keyWidth)
		}
		if len(sr) > 1 {
			row[1] = padToWidth(truncateNoEllipsis(sr[1], valueWidth), valueWidth)
		}
		rows[i] = table.Row(row)
	}
	return rows
}

// autoKeyColumnWidth shrinks the key column to the widest visible key, capped by the configured preset.
func (m *Model) autoKeyColumnWidth(maxPreset int) int {
	if maxPreset <= 0 {
		maxPreset = DefaultKeyColWidth
	}
	maxKey := 0
	if m.AdvancedSearchActive && len(m.AdvancedSearchResults) > 0 {
		for _, res := range m.AdvancedSearchResults {
			w := lipgloss.Width(strings.TrimSpace(res.Key))
			if w > maxKey {
				maxKey = w
			}
		}
	} else {
		rows := navigator.NodeToRows(m.Node)
		for _, row := range rows {
			if len(row) == 0 {
				continue
			}
			w := lipgloss.Width(strings.TrimSpace(row[0]))
			if w > maxKey {
				maxKey = w
			}
		}
	}
	if maxKey <= 0 {
		return maxPreset
	}
	if maxKey > maxPreset {
		maxKey = maxPreset
	}
	if maxKey < 8 {
		maxKey = 8
	}
	return maxKey
}

// AutoKeyColumnWidth is an exported wrapper so external callers (e.g., CLI rendering) can reuse the
// same auto key width logic while honoring the configured maximum.
func (m *Model) AutoKeyColumnWidth(maxPreset int) int {
	return m.autoKeyColumnWidth(maxPreset)
}

// truncateString truncates a string to maxLen, adding ellipsis if needed
func truncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return s
	}
	// Measure display width (handles wide chars like CJK)
	w := runewidth.StringWidth(s)
	if w <= maxLen {
		return s
	}
	if maxLen < 3 {
		// Too short for ellipsis, just truncate
		return runewidth.Truncate(s, maxLen, "")
	}
	// Truncate to fit width, leaving room for ellipsis
	return runewidth.Truncate(s, maxLen-3, "") + "..."
}

// padToWidth right-pads the string to the given display width using spaces.
func padToWidth(s string, width int) string {
	if width <= 0 {
		return s
	}
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func truncateNoEllipsis(s string, maxLen int) string {
	if maxLen <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}

// adjustSuggestionNamespace avoids duplicating the namespace when inserting a suggestion.
// If the current token before the insertion dot matches the suggestion's namespace prefix,
// e.g., base="base64" and name="base64.decode()", it returns just "decode()".
// Otherwise, returns name unchanged.
func adjustSuggestionNamespace(base, name string) string {
	if base == "" {
		return name
	}
	// Only consider dot-qualified suggestion names
	if idx := strings.Index(name, "."); idx >= 0 {
		ns := name[:idx]
		if ns == base {
			return name[idx+1:]
		}
	}
	return name
}

func matchKeyToken(candidate string) string {
	c := strings.TrimSpace(candidate)
	if strings.HasPrefix(c, "_.") {
		c = strings.TrimPrefix(c, "_.")
	} else if strings.HasPrefix(c, "_") {
		c = strings.TrimPrefix(c, "_")
	}
	if idx := strings.LastIndex(c, "."); idx >= 0 {
		c = c[idx+1:]
	}
	if idx := strings.Index(c, "["); idx >= 0 {
		// Prefer the bracketed segment (e.g., "[0]" -> "0")
		inner := strings.Trim(c[idx:], `[]"`)
		if inner != "" {
			c = inner
		} else {
			c = c[:idx]
		}
	}
	c = strings.Trim(c, `[]"`)
	return strings.TrimSpace(c)
}

// normalizeExprBase ensures a base path is CEL-safe with root prefix.
func normalizeExprBase(base string) string {
	b := strings.TrimSpace(base)
	if b == "" {
		return "_"
	}
	// Keep CEL literals intact: quoted strings, arrays, maps
	if strings.HasPrefix(b, "_") || strings.HasPrefix(b, "\"") || strings.HasPrefix(b, "[") || strings.HasPrefix(b, "{") {
		return b
	}
	return "_." + b
}

// InitialModel creates a new UI model
func InitialModel(node interface{}) Model {
	stringRows := navigator.NodeToRows(node)
	rows := styleRows(stringRows)
	rowKeys := extractRowKeys(stringRows)
	// Use default headers; can be overridden via config
	keyHeader := "KEY"
	valueHeader := "VALUE"
	columns := []table.Column{
		{Title: keyHeader, Width: 30},
		{Title: valueHeader, Width: 60},
	}
	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithRows(rows),
		// Use a small initial height; we'll resize after WindowSizeMsg
		table.WithHeight(5),
	)
	// In v2, WithRows might not work, so set rows explicitly too
	t.SetRows(rows)
	// Make sure height is set after rows
	t.SetHeight(5)
	th := CurrentTheme()
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(borderForTheme(th)).
		BorderBottom(true).
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false).
		Bold(true).
		Align(lipgloss.Left).
		PaddingLeft(0).
		PaddingRight(0)
	s.Selected = s.Selected.
		PaddingLeft(0).
		PaddingRight(0)
	s.Cell = lipgloss.NewStyle().
		Align(lipgloss.Left).
		PaddingLeft(0).
		PaddingRight(0)
	t.SetStyles(s)

	ti := textinput.New()
	// Use default placeholder; can be overridden via config
	ti.Placeholder = "Enter path (e.g. items[0] or items.filter(x, x.available))"
	ti.CharLimit = 500
	ti.SetWidth(80) // Initial width, will be adjusted in applyLayout
	ti.Prompt = ""
	// At root, show '_' by default in the expr section
	ti.SetValue("_")

	si := textinput.New()
	si.Placeholder = ""
	si.CharLimit = 500
	si.SetWidth(80) // Initial width, will be adjusted in applyLayout
	si.Prompt = ""
	si.SetValue("")

	// Map filter input (for 'f' key filter mode)
	fi := textinput.New()
	fi.Placeholder = ""
	fi.CharLimit = 500
	fi.SetWidth(80)
	fi.Prompt = ""
	fi.SetValue("")

	status := NewStatusModel()
	footer := NewFooterModel()
	help := NewHelpModel()
	debug := NewDebugModel()
	suggestionsComponent := NewSuggestionsModel()
	layout := NewLayoutManager(0, 0) // Will be set on first WindowSizeMsg

	// Load default FunctionExamples from embedded config
	var functionExamples map[string]FunctionExampleValue
	if cfg, err := EmbeddedDefaultConfig(); err == nil {
		functionExamples = cfg.Help.CEL.FunctionExamples
		// Derive example hints for CEL function discovery only if not already
		// set by the CLI config loader (which merges user overrides on top).
		if celhelper.GetExampleHints() == nil && len(functionExamples) > 0 {
			hints := make(map[string]string, len(functionExamples))
			for name, v := range functionExamples {
				if len(v.Examples) > 0 {
					hints[name] = "e.g. " + v.Examples[0]
				}
			}
			celhelper.SetExampleHints(hints)
		}
	}

	// Initialize CEL function suggestions with usage hints; fall back to static list
	suggestionsList := DiscoverExpressions()

	var completionEngine *completion.CompletionEngine
	if provider, err := intellisense.NewProvider(); err == nil {
		completionEngine = completion.NewEngine(provider)
	}

	return Model{
		Tbl:                        t,
		AllRows:                    rows,
		AllRowKeys:                 rowKeys,
		PathInput:                  ti,
		SearchInput:                si,
		MapFilterInput:             fi,
		Status:                     status,
		Footer:                     footer,
		Help:                       help,
		Debug:                      debug,
		SuggestionsComponent:       suggestionsComponent,
		Layout:                     layout,
		Node:                       node,
		Root:                       node,
		Path:                       "",
		PathKeys:                   []string{},
		CursorByPath:               map[string]int{},
		DebugVersion:               modelVersionString(),
		InputFocused:               false,
		ExprDisplay:                "",
		ExprType:                   "",
		Suggestions:                suggestionsList,
		FilteredSuggestions:        []string{},
		SelectedSuggestion:         0,
		ShowSuggestions:            false,
		CompletionEngine:           completionEngine,
		KeyColWidth:                30,
		ConfiguredKeyColWidth:      30,
		AllowEditInput:             true,
		AllowFilter:                true,
		AllowSuggestions:           true,
		AllowIntellisense:          true,
		HelpPopupJustify:           "center",
		HelpPopupAnchor:            "top",
		ShowInfoPopup:              false,
		InfoPopupAnchor:            "inline",
		InfoPopupJustify:           "left",
		InfoPopupModal:             true,
		InfoPopupPermanent:         false,
		InfoPopupEnabled:           true,
		HelpAboutTitle:             "",
		HelpAboutLines:             nil,
		HelpAboutAlign:             "right",
		KeyHeader:                  "KEY",
		ValueHeader:                "VALUE",
		InputPromptUnfocused:       "$ ",
		InputPromptFocused:         "❯ ",
		InputPlaceholder:           "Enter path (e.g. items[0] or items.filter(x, x.available))",
		HelpNavigationDescriptions: nil,
		TruncateTableCells:         true,
		FunctionExamples:           functionExamples,
		KeyMode:                    KeyModeVim, // Default to vim-style keybindings
		// Performance defaults
		SearchDebounceMs:  150,  // 150ms debounce for search input
		SearchResultLimit: 500,  // Limit deep search to 500 results
		ScrollBufferRows:  5,    // Pre-render 5 rows above/below viewport
		VirtualScrolling:  true, // Enable virtual scrolling by default
	}
}

// ApplyColorScheme applies or removes color styling based on NoColor flag
func (m *Model) ApplyColorScheme() {
	s := table.DefaultStyles()
	th := CurrentTheme()
	s.Header = s.Header.
		BorderStyle(lipgloss.HiddenBorder()).
		BorderBottom(false).
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false).
		Bold(true).
		Align(lipgloss.Left).
		PaddingLeft(0).
		PaddingRight(0)

	baseCell := lipgloss.NewStyle().
		Align(lipgloss.Left).
		PaddingLeft(0).
		PaddingRight(0)
	s.Cell = baseCell
	selected := baseCell

	if m.NoColor {
		// Clear all colors and use inverse for selection
		s.Header = s.Header.
			UnsetForeground().
			UnsetBackground()
		selected = selected.
			UnsetForeground().
			UnsetBackground().
			Reverse(true)
		s.Cell = s.Cell.
			UnsetForeground().
			UnsetBackground()
	} else {
		// Apply colors
		greyBG := lipgloss.Color("240")
		s.Header = s.Header.
			Foreground(lipgloss.Color("15")).
			Background(greyBG)
		s.Cell = s.Cell.
			Background(greyBG)
		selected = selected.
			Foreground(th.SelectedFG).
			Background(th.SelectedBG)
	}
	s.Selected = selected
	m.Tbl.SetStyles(s)
}

// normalizePathForModel normalizes numeric segments and ensures path starts with _.
// Empty paths represent root and are returned as empty string.
func normalizePathForModel(path string) string {
	cleaned := strings.TrimSpace(path)
	if cleaned == "" || cleaned == "_" {
		return ""
	}
	// Normalize segments and quote invalid identifiers (e.g., tasks.build-windows -> tasks["build-windows"]).
	prefixed := cleaned
	if !strings.HasPrefix(prefixed, "_") {
		// If path starts with bracket notation (array index), don't add a dot
		if strings.HasPrefix(cleaned, "[") {
			prefixed = "_" + prefixed
		} else {
			prefixed = "_." + prefixed
		}
	}
	segs := splitPathSegments(prefixed)
	if len(segs) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("_")
	for _, seg := range segs {
		segOut := renderSegment(seg)
		if strings.HasPrefix(segOut, "[") {
			b.WriteString(segOut)
			continue
		}
		b.WriteString(".")
		b.WriteString(segOut)
	}

	return b.String()
}

// logEvent captures a debug event snapshot for later printing on exit.
func (m *Model) logEvent(label string) {
	shown := len(m.Tbl.Rows())
	all := len(m.AllRows)
	cur := m.Tbl.Cursor()
	prev := len(m.PreviousAllRows)
	first := ""
	if shown > 0 && len(m.Tbl.Rows()[0]) > 0 {
		first = m.Tbl.Rows()[0][0]
	}

	// Calculate layout information for debugging
	helpInline := m.HelpVisible && strings.ToLower(m.HelpPopupAnchor) != "top"
	// Don't reserve space for suggestions - they're shown in status bar
	heights := m.Layout.CalculateHeights(helpInline, m.DebugMode, false, m.ShowPanelTitle)
	bottomBlockHeight := heights.PathInputHeight +
		heights.SuggestionHeight +
		heights.StatusHeight +
		heights.DebugHeight +
		heights.FooterHeight +
		1 // Separator line

	// Calculate table pad
	tablePad := 0
	if m.TableHeight > 0 {
		actualTableLines := countLines(m.Tbl.View())
		desiredTableLines := m.TableHeight + TableHeaderLines
		if desiredTableLines > actualTableLines {
			tablePad = desiredTableLines - actualTableLines
		}
	}
	if m.HelpVisible && strings.ToLower(m.HelpPopupAnchor) == "top" {
		tablePad = 0
	}

	// Suggestions are now shown in status bar, not as dropdown
	entry := fmt.Sprintf("%s | path=%q shown=%d all=%d cursor=%d adv=%v ctx=%v filter=%v/%q prevRows=%d first=%q | layout: th=%d sh=%d ph=%d st=%d dh=%d fh=%d bb=%d pad=%d reserve=false inputFocused=%v filtered=%d",
		label, m.Path, shown, all, cur, m.AdvancedSearchActive, m.SearchContextActive, m.FilterActive, m.FilterBuffer, prev, first,
		heights.TableHeight, heights.SuggestionHeight, heights.PathInputHeight,
		heights.StatusHeight, heights.DebugHeight, heights.FooterHeight,
		bottomBlockHeight, tablePad, m.InputFocused,
		len(m.FilteredSuggestions))
	m.DebugEvents = append(m.DebugEvents, DebugEvent{Time: time.Now(), Message: entry})
	// Keep a reasonable cap to avoid unbounded growth
	if len(m.DebugEvents) > 200 {
		m.DebugEvents = m.DebugEvents[len(m.DebugEvents)-200:]
	}
}

// logKeyEvent records specific key events we care about for debugging.
func (m *Model) logKeyEvent(keyStr string) {
	switch keyStr {
	case "enter", "esc", "f1", "f2", "f3", "f4", "f5", "f6", "f10":
		m.logEvent("key:" + keyStr)
	}
}

// formatPathForDisplay formats a path for display in the expression bar.
// It adds the "_." prefix if needed, or returns "_" for empty paths.
// This centralizes the repeated path formatting logic throughout the codebase.
func formatPathForDisplay(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "_"
	}
	// Leave literals/expressions untouched.
	if strings.HasPrefix(trimmed, "\"") || strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "{") {
		return trimmed
	}
	if (strings.Contains(trimmed, "(") && strings.Contains(trimmed, ")")) || IsExpression(trimmed) {
		return trimmed
	}

	// Ensure underscore prefix and split into segments (dot and bracket aware)
	prefixed := trimmed
	if prefixed == "_" {
		return "_"
	}
	if !strings.HasPrefix(prefixed, "_") {
		prefixed = "_." + prefixed
	}

	segments := splitPathSegments(prefixed)

	var b strings.Builder
	b.WriteString("_")
	for _, seg := range segments {
		segOut := renderSegment(seg)
		if strings.HasPrefix(segOut, "[") {
			b.WriteString(segOut)
			continue
		}
		b.WriteString(".")
		b.WriteString(segOut)
	}

	return b.String()
}

// splitPathSegments splits a CEL path into segments, removing leading underscores and dots,
// separating bracketed indices/literals as their own segments, and dropping the leading root marker.
func splitPathSegments(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	for strings.HasPrefix(s, "_.") || strings.HasPrefix(s, "_") || strings.HasPrefix(s, ".") {
		s = strings.TrimPrefix(s, "_.")
		s = strings.TrimPrefix(s, "_")
		s = strings.TrimPrefix(s, ".")
	}

	segments := []string{}
	var cur strings.Builder

	flushCur := func() {
		if cur.Len() == 0 {
			return
		}
		segments = append(segments, cur.String())
		cur.Reset()
	}

	i := 0
	for i < len(s) {
		ch := s[i]
		switch ch {
		case '.':
			flushCur()
			i++
			continue
		case '[':
			flushCur()
			// capture until closing bracket
			j := i + 1
			for j < len(s) && s[j] != ']' {
				j++
			}
			if j < len(s) {
				segments = append(segments, s[i+1:j])
				i = j + 1
				continue
			}
		}
		cur.WriteByte(ch)
		i++
	}
	flushCur()
	return segments
}

// renderSegment converts a raw segment into display-safe CEL syntax.
func renderSegment(seg string) string {
	if seg == "" {
		return seg
	}
	if _, err := strconv.Atoi(seg); err == nil {
		return "[" + seg + "]"
	}
	if strings.HasPrefix(seg, "\"") && strings.HasSuffix(seg, "\"") {
		return "[" + seg + "]"
	}
	if isValidCELIdentifier(seg) {
		return seg
	}
	return "[\"" + seg + "\"]"
}

func isValidCELIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !unicode.IsLetter(r) && r != '_' {
				return false
			}
			continue
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '.' {
			return false
		}
	}
	return true
}

func formatExprDisplay(expr string) string {
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" {
		return ""
	}
	if (strings.Contains(trimmed, "(") && strings.Contains(trimmed, ")")) || IsExpression(trimmed) {
		return trimmed
	}
	return formatPathForDisplay(trimmed)
}

func completionRootForInput(root, current interface{}, input string) interface{} {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" || strings.HasPrefix(trimmed, "_") {
		return root
	}
	return current
}

func inferExprType(node interface{}) string {
	if node == nil {
		return ""
	}
	return nodeTypeLabel(node)
}

func (m *Model) setExprResult(expr string, node interface{}) {
	if m == nil {
		return
	}
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" {
		trimmed = "_"
	}
	m.ExprDisplay = trimmed
	m.ExprType = inferExprType(node)
}

func cursorPathKey(path string) string {
	normalized := normalizePathForModel(path)
	if normalized == "" {
		return "_"
	}
	return normalized
}

func (m *Model) storeCursorForPath(path string) {
	if m == nil {
		return
	}
	if m.CursorByPath == nil {
		m.CursorByPath = map[string]int{}
	}
	m.CursorByPath[cursorPathKey(path)] = m.Tbl.Cursor()
}

// storeCursorForPathByKey stores the cursor position for a path based on where a specific key
// appears in the full (unfiltered) node. This is needed when navigating from a filtered view
// so that going back restores the cursor to the correct position in the full list.
func (m *Model) storeCursorForPathByKey(path string, selectedKey string) {
	if m == nil || m.Node == nil {
		return
	}
	if m.CursorByPath == nil {
		m.CursorByPath = map[string]int{}
	}

	// Find the index of the selected key in the full node
	idx := 0
	switch node := m.Node.(type) {
	case map[string]interface{}:
		// Get sorted keys (same order as NodeToRows)
		keys := make([]string, 0, len(node))
		for k := range node {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, k := range keys {
			if k == selectedKey {
				idx = i
				break
			}
		}
	case []interface{}:
		// For arrays, try to parse the index from the key (e.g., "[0]" -> 0)
		if strings.HasPrefix(selectedKey, "[") && strings.HasSuffix(selectedKey, "]") {
			if parsed, err := strconv.Atoi(selectedKey[1 : len(selectedKey)-1]); err == nil {
				idx = parsed
			}
		}
	}

	m.CursorByPath[cursorPathKey(path)] = idx
}

func (m *Model) restoreCursorForPath(path string) {
	if m == nil || m.CursorByPath == nil {
		return
	}
	idx, ok := m.CursorByPath[cursorPathKey(path)]
	if !ok {
		return
	}
	rows := m.Tbl.Rows()
	if len(rows) == 0 {
		return
	}
	if idx < 0 {
		idx = 0
	}
	if idx >= len(rows) {
		idx = len(rows) - 1
	}
	m.Tbl.SetCursor(idx)
	m.SyncTableState()
	m.syncPathInputWithCursor()
}

func (m *Model) selectedRowKey() (string, bool) {
	if m == nil {
		return "", false
	}
	rows := m.Tbl.Rows()
	if len(rows) == 0 {
		return "", false
	}
	cur := m.Tbl.Cursor()
	if cur < 0 {
		cur = 0
	}
	if cur >= len(rows) {
		cur = len(rows) - 1
	}
	originalKeys := m.AllRowKeys
	if len(originalKeys) == 0 && m.Node != nil {
		originalKeys = extractRowKeys(navigator.NodeToRows(m.Node))
	}
	if len(originalKeys) == 0 {
		return "", false
	}
	if m.FilterActive && m.FilterBuffer != "" {
		filterLower := strings.ToLower(m.FilterBuffer)
		matches := []string{}
		for _, key := range originalKeys {
			origKey := key
			cand := key
			if strings.HasPrefix(cand, "[") && strings.HasSuffix(cand, "]") {
				cand = cand[1 : len(cand)-1]
			}
			if strings.HasPrefix(strings.ToLower(cand), filterLower) {
				matches = append(matches, origKey)
			}
		}
		if cur < len(matches) {
			return matches[cur], true
		}
		if len(matches) > 0 {
			return matches[0], true
		}
		return "", false
	}
	if cur >= len(originalKeys) {
		return "", false
	}
	return originalKeys[cur], true
}

func buildPathWithKey(basePath, selectedKey string) string {
	pathKey := selectedKey
	useBracketNotation := false
	isNumericIndex := false

	if strings.HasPrefix(selectedKey, "[") && strings.HasSuffix(selectedKey, "]") {
		inner := selectedKey[1 : len(selectedKey)-1]
		if strings.HasPrefix(inner, "\"") && strings.HasSuffix(inner, "\"") && len(inner) > 1 {
			pathKey = inner[1 : len(inner)-1]
			useBracketNotation = true
		} else {
			pathKey = inner
			if _, err := strconv.Atoi(pathKey); err == nil {
				isNumericIndex = true
				useBracketNotation = true
			}
		}
	}

	// Quote keys that are not valid CEL identifiers.
	if !useBracketNotation && !isValidCELIdentifier(pathKey) {
		useBracketNotation = true
	}

	if basePath != "" {
		if useBracketNotation {
			if isNumericIndex {
				return basePath + "[" + pathKey + "]"
			}
			return basePath + `["` + pathKey + `"]`
		}
		return basePath + "." + pathKey
	}

	if useBracketNotation {
		if isNumericIndex {
			return "_[" + pathKey + "]"
		}
		return `_["` + pathKey + `"]`
	}
	return "_." + pathKey
}

func (m *Model) selectedRowPath() string {
	if m == nil {
		return ""
	}
	if m.AdvancedSearchActive && len(m.AdvancedSearchResults) > 0 {
		cur := m.Tbl.Cursor()
		if cur < 0 || cur >= len(m.AdvancedSearchResults) {
			cur = 0
		}
		if cur >= 0 && cur < len(m.AdvancedSearchResults) {
			result := m.AdvancedSearchResults[cur]
			fullPath := result.FullPath
			if m.AdvancedSearchBasePath != "" {
				if strings.HasPrefix(result.FullPath, "[") {
					fullPath = m.AdvancedSearchBasePath + result.FullPath
				} else {
					fullPath = m.AdvancedSearchBasePath + "." + result.FullPath
				}
			}
			return normalizePathForModel(fullPath)
		}
		return ""
	}
	selectedKey, ok := m.selectedRowKey()
	if !ok || strings.TrimSpace(selectedKey) == "" || selectedKey == "(value)" {
		return m.Path
	}
	return buildPathWithKey(m.Path, selectedKey)
}

// NavigateTo updates the existing model to navigate to the given node and path.
// This updates the model in-place to avoid recreating components and prevent flicker.
func (m *Model) NavigateTo(node interface{}, path string) *Model {
	normalizedPath := normalizePathForModel(path)

	// Preserve search context before updating (needed for left arrow navigation back to search)
	searchContextActive := m.SearchContextActive
	searchContextResults := make([]SearchResult, len(m.SearchContextResults))
	copy(searchContextResults, m.SearchContextResults)
	searchContextQuery := m.SearchContextQuery
	searchContextBasePath := m.SearchContextBasePath

	// Generate new rows for the new node using existing column widths
	keyW := m.KeyColWidth
	if keyW <= 0 {
		keyW = 30
	}
	valueW := m.ValueColWidth
	if valueW <= 0 {
		valueW = 60
	}
	stringRows := navigator.NodeToRows(node)
	newRows := styleRowsWithWidths(stringRows, keyW, valueW)
	newRowKeys := extractRowKeys(stringRows)

	// Update model state in-place (don't recreate components - this prevents flicker)
	m.Node = node
	m.Path = normalizedPath
	m.PathKeys = parsePathKeys(normalizedPath)
	m.AllRows = newRows
	m.AllRowKeys = newRowKeys

	// Temporarily clear AdvancedSearchActive so SyncTableState() uses the new node's rows
	// (not search results). We're navigating to a new node, so we want to show its rows.
	wasInSearch := m.AdvancedSearchActive
	m.AdvancedSearchActive = false

	// Clear type-ahead filter state before syncing the new context.
	// If we leave the previous filter active, new rows (e.g., array indices "[0]") can be filtered out,
	// leading to empty tables after navigation.
	m.FilterActive = false
	m.FilterBuffer = ""
	// Clear suggestion filter state
	m.SuggestionFilterActive = false
	m.FilteredSuggestionRows = nil

	// Use SyncTableState() to update table rows and cursor consistently
	m.SyncTableState(true)

	// Restore AdvancedSearchActive if it was set (it will be cleared by the caller if needed)
	m.AdvancedSearchActive = wasInSearch

	// Update path input: in table mode show `_` root, in expr mode show literal (original path)
	if !m.InputFocused {
		m.PathInput.SetValue(formatPathForDisplay(normalizedPath))
	} else {
		// In expr mode, preserve the original path format (user may have typed it without _ prefix)
		m.PathInput.SetValue(path)
	}

	// Clear error state
	m.clearError()

	// Restore search context (preserved above)
	m.SearchContextActive = searchContextActive
	m.SearchContextResults = searchContextResults
	m.SearchContextQuery = searchContextQuery
	m.SearchContextBasePath = searchContextBasePath

	// Columns and focus are already set by SyncTableState() above
	// Only need to focus path input if in input mode
	if m.InputFocused {
		m.PathInput.Focus()
	}

	return m
}

// nodeTypeLabel maps a Go node to a simple CEL-like type label used in suggestion usage strings.
func nodeTypeLabel(node interface{}) string {
	switch node.(type) {
	case []interface{}:
		return "list"
	case map[string]interface{}:
		return "map"
	case string:
		return "string"
	case bool:
		return "bool"
	case int, int64:
		return "int"
	case uint, uint64:
		return "uint"
	case float32, float64:
		return "double"
	default:
		return "any"
	}
}

// isSuggestionCompatibleWithNode uses the usage hint in the suggestion string to filter by type.
// Suggestion format example: "lowerAscii() - string.lowerAscii() -> string" or "flatten() - flatten(list) -> list".
func isSuggestionCompatibleWithNode(suggestion string, node interface{}) bool {
	// Non-function suggestions (keys) are always compatible
	if !strings.Contains(suggestion, "(") && !strings.Contains(suggestion, " - ") {
		return true
	}

	// Extract function name for special handling
	funcName := suggestion
	if idx := strings.Index(suggestion, "("); idx >= 0 {
		funcName = suggestion[:idx]
	}

	// Common transformation functions that should be available for all collection types (map/list)
	// These are useful even if type conversion is needed
	universalFunctions := map[string]bool{
		"map":    true,
		"filter": true,
		"all":    true,
		"exists": true,
		"size":   true,
		"has":    true,
	}
	if universalFunctions[funcName] {
		// For maps and lists, show these functions as they're commonly used
		nodeType := nodeTypeLabel(node)
		if nodeType == "map" || nodeType == "list" || nodeType == "any" {
			return true
		}
	}

	nodeType := nodeTypeLabel(node)
	// Extract usage after " - " if present
	usage := suggestion
	if idx := strings.Index(suggestion, " - "); idx >= 0 {
		usage = suggestion[idx+3:]
	}
	usage = strings.TrimSpace(usage)
	// If usage starts with a receiver (e.g., "string.func(args)")
	dotIdx := strings.Index(usage, ".")
	parenIdx := strings.Index(usage, "(")
	if dotIdx >= 0 && (parenIdx == -1 || dotIdx < parenIdx) {
		recv := usage[:dotIdx]
		recv = strings.TrimSpace(recv)
		// Treat "any"/"dyn" as universal
		if recv == "any" || recv == "dyn" {
			return true
		}
		// For list methods, also allow them for maps (user might convert map to list)
		if recv == "list" && nodeType == "map" && universalFunctions[funcName] {
			return true
		}
		return recv == nodeType
	}
	// Otherwise, check first arg type in global-style usage: name(type, ...)
	if parenIdx >= 0 {
		end := strings.Index(usage[parenIdx+1:], ")")
		if end >= 0 {
			params := strings.TrimSpace(usage[parenIdx+1 : parenIdx+1+end])
			// Take first parameter type token
			if params != "" {
				first := params
				if comma := strings.Index(params, ","); comma >= 0 {
					first = params[:comma]
				}
				first = strings.TrimSpace(first)
				if first == "any" || first == "dyn" {
					return true
				}
				// For list-parameter functions, also allow them for maps
				if first == "list" && nodeType == "map" && universalFunctions[funcName] {
					return true
				}
				return first == nodeType
			}
		}
	}
	// If we can't determine, allow by default
	return true
}

// usageStyleLabel returns a compact tag indicating suggestion style: [method] or [global].
// It inspects the usage hint part of the suggestion string, and falls back to the
// name prefix (before " - ") when no usage hint is present (e.g., macros like has()).
func usageStyleLabel(suggestion string) string {
	classify := func(part string) string {
		part = strings.TrimSpace(part)
		dotIdx := strings.Index(part, ".")
		parenIdx := strings.Index(part, "(")
		if dotIdx >= 0 && (parenIdx == -1 || dotIdx < parenIdx) {
			return "[method]"
		}
		if strings.Contains(part, "(") {
			return "[global]"
		}
		return ""
	}

	usage := suggestion
	if idx := strings.Index(suggestion, " - "); idx >= 0 {
		usage = suggestion[idx+3:]
	}

	if style := classify(usage); style != "" {
		return style
	}

	// Fallback to the name prefix (e.g., "has()") when usage is missing or generic.
	name := suggestion
	if idx := strings.Index(suggestion, " - "); idx >= 0 {
		name = suggestion[:idx]
	}
	return classify(name)
}

// wrapGlobal trims trailing parentheses from a function name and wraps the base expression.
func wrapGlobal(name, baseExpr string) string {
	n := strings.TrimSpace(name)
	n = strings.TrimSuffix(n, "()")
	return n + "(" + baseExpr + ")"
}

// baseForGlobal extracts the parent expression for global function wrapping.
// Drops trailing dot (if present) OR drops incomplete trailing token (after last dot).
// Examples:
//
//	"_.items." (trailing dot) → "_.items"
//	"_.items.f" (partial token) → "_.items"
//	"items" → "_.items"
func baseForGlobal(currentValue string) string {
	v := strings.TrimSpace(currentValue)
	// Drop incomplete trailing bracket opener
	v = strings.TrimSuffix(v, "[")

	// If trailing dot, remove it - that's the full base
	if strings.HasSuffix(v, ".") {
		v = strings.TrimSuffix(v, ".")
		if v == "" {
			return "_"
		}
		return normalizeExprBase(v)
	}

	// No trailing dot: drop the last token (everything after last dot)
	if idx := strings.LastIndex(v, "."); idx >= 0 {
		v = v[:idx]
	}
	if v == "" {
		return "_"
	}
	return normalizeExprBase(v)
}

// clearSuggestionSummary resets the one-shot trailing-dot summary shown in the status bar.
func (m *Model) clearSuggestionSummary() {
	m.SuggestionSummary = ""
	m.ShowSuggestionSummary = false
}

// normalizeFunctionName strips adornments (parentheses, receiver, case) to dedupe function suggestions.
func normalizeFunctionName(name string) string {
	n := strings.TrimSpace(name)
	if idx := strings.LastIndex(n, "."); idx >= 0 {
		n = n[idx+1:]
	}
	n = strings.TrimSuffix(strings.TrimSuffix(n, "()"), "(")
	n = strings.ToLower(strings.TrimSpace(n))
	return n
}

func extractFunctionName(candidate string) string {
	name := candidate
	if idx := strings.Index(name, " - "); idx >= 0 {
		name = name[:idx]
	}
	if idx := strings.Index(name, "("); idx >= 0 {
		name = name[:idx]
	}
	if dotIdx := strings.LastIndex(name, "."); dotIdx >= 0 {
		name = name[dotIdx+1:]
	}
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(strings.TrimSuffix(name, "()"), "(")
	return name
}

func isFunctionSuggestion(candidate string) bool {
	return strings.Contains(candidate, "(") || strings.Contains(candidate, " - ")
}

func findFunctionSuggestion(funcName string, suggestions []string) string {
	norm := normalizeFunctionName(funcName)
	if norm == "" {
		return ""
	}
	for _, suggestion := range suggestions {
		if !isFunctionSuggestion(suggestion) {
			continue
		}
		name := extractFunctionName(suggestion)
		if normalizeFunctionName(name) == norm {
			return suggestion
		}
	}
	return ""
}

func (m *Model) lookupFunctionOverride(funcName string) string {
	if m == nil || len(m.FunctionHelpOverrides) == 0 {
		return ""
	}
	norm := normalizeFunctionName(funcName)
	if norm != "" {
		if help, ok := m.FunctionHelpOverrides[norm]; ok && strings.TrimSpace(help) != "" {
			return strings.TrimSpace(help)
		}
	}
	if help, ok := m.FunctionHelpOverrides[funcName]; ok && strings.TrimSpace(help) != "" {
		return strings.TrimSpace(help)
	}
	lower := strings.ToLower(strings.TrimSpace(funcName))
	if help, ok := m.FunctionHelpOverrides[lower]; ok && strings.TrimSpace(help) != "" {
		return strings.TrimSpace(help)
	}
	return ""
}

func (m *Model) lookupFunctionExample(funcName string) (FunctionExampleValue, bool) {
	var empty FunctionExampleValue
	if m == nil || funcName == "" {
		return empty, false
	}
	norm := normalizeFunctionName(funcName)
	if len(m.FunctionExamples) > 0 {
		if norm != "" {
			if ex, ok := m.FunctionExamples[norm]; ok {
				return ex, true
			}
		}
		if ex, ok := m.FunctionExamples[funcName]; ok {
			return ex, true
		}
		lower := strings.ToLower(strings.TrimSpace(funcName))
		if ex, ok := m.FunctionExamples[lower]; ok {
			return ex, true
		}
	}
	examplesData := completion.GetFunctionExamplesData()
	if len(examplesData) == 0 {
		return empty, false
	}
	if norm != "" {
		if ex, ok := examplesData[norm]; ok {
			return FunctionExampleValue{Description: ex.Description, Examples: ex.Examples}, true
		}
	}
	if ex, ok := examplesData[funcName]; ok {
		return FunctionExampleValue{Description: ex.Description, Examples: ex.Examples}, true
	}
	lower := strings.ToLower(strings.TrimSpace(funcName))
	if ex, ok := examplesData[lower]; ok {
		return FunctionExampleValue{Description: ex.Description, Examples: ex.Examples}, true
	}
	return empty, false
}

func formatFunctionExample(description string, examples []string) string {
	desc := strings.TrimSpace(description)
	clean := make([]string, 0, len(examples))
	for _, ex := range examples {
		ex = strings.TrimSpace(ex)
		if ex != "" {
			clean = append(clean, ex)
		}
	}
	parts := []string{}
	if desc != "" {
		parts = append(parts, desc)
	}
	if len(clean) > 0 {
		parts = append(parts, "e.g. "+strings.Join(clean, " | "))
	}
	return strings.Join(parts, "\n")
}

func formatSuggestionHelp(suggestion, funcName, baseExpr string) string {
	idx := strings.Index(suggestion, " - ")
	if idx < 0 {
		return ""
	}
	helpText := strings.TrimSpace(suggestion[idx+3:])
	if helpText == "" {
		return ""
	}
	if pipeIdx := strings.Index(helpText, " | "); pipeIdx >= 0 {
		usageHint := strings.TrimSpace(helpText[:pipeIdx])
		example := strings.TrimSpace(helpText[pipeIdx+3:])
		if strings.EqualFold(usageHint, "CEL function") {
			if example != "" {
				return example
			}
		} else if usageHint != "" {
			return usageHint
		}
	}
	if strings.EqualFold(helpText, "CEL function") {
		base := strings.TrimSpace(baseExpr)
		if base == "" {
			base = "_"
		}
		name := extractFunctionName(funcName)
		if name == "" {
			name = funcName
		}
		return fmt.Sprintf("%s(%s)", name, base)
	}
	return helpText
}

func usageHintFromSuggestion(suggestion, funcName, baseExpr string) string {
	idx := strings.Index(suggestion, " - ")
	if idx < 0 {
		return ""
	}
	helpText := strings.TrimSpace(suggestion[idx+3:])
	if helpText == "" {
		return ""
	}
	if pipeIdx := strings.Index(helpText, " | "); pipeIdx >= 0 {
		helpText = strings.TrimSpace(helpText[:pipeIdx])
	}
	if strings.EqualFold(helpText, "CEL function") {
		base := strings.TrimSpace(baseExpr)
		if base == "" {
			base = "_"
		}
		name := extractFunctionName(funcName)
		if name == "" {
			name = funcName
		}
		return fmt.Sprintf("%s(%s)", name, base)
	}
	return helpText
}

func (m *Model) lookupFunctionHelp(funcName string, suggestions []string, baseExpr string) string {
	name := extractFunctionName(funcName)
	if name == "" {
		name = strings.TrimSpace(funcName)
	}
	if name == "" {
		return ""
	}
	if help := m.lookupFunctionOverride(name); help != "" {
		return help
	}

	// Priority: Check FunctionExamples first (most detailed help)
	if ex, ok := m.lookupFunctionExample(name); ok {
		if formatted := formatFunctionExample(ex.Description, ex.Examples); formatted != "" {
			return formatted
		}
	}

	// Fall back to suggestion-based help
	suggestion := findFunctionSuggestion(name, suggestions)
	usage := ""
	if suggestion != "" {
		usage = usageHintFromSuggestion(suggestion, name, baseExpr)
		if idx := strings.Index(suggestion, " - "); idx >= 0 {
			detail := strings.TrimSpace(suggestion[idx+3:])
			if detail != "" && !strings.Contains(strings.ToLower(detail), "cel function") {
				if formatted := formatSuggestionHelp(suggestion, name, baseExpr); formatted != "" {
					return formatted
				}
			}
		}
	}
	if usage != "" {
		return usage
	}
	if suggestion != "" {
		if formatted := formatSuggestionHelp(suggestion, name, baseExpr); formatted != "" {
			return formatted
		}
	}
	return ""
}

// buildFunctionSummary returns a comma-separated list of function names (with parentheses)
// extracted from the provided suggestion candidates. It de-dupes by function name.
func buildFunctionSummary(candidates []string, maxCount int) string {
	if len(candidates) == 0 {
		return ""
	}

	names := make([]string, 0, len(candidates))
	seen := map[string]bool{}

	for _, cand := range candidates {
		if !strings.Contains(cand, "(") && !strings.Contains(cand, " - ") {
			continue
		}

		name := extractFunctionName(cand)
		norm := normalizeFunctionName(name)
		if norm == "" || seen[norm] {
			continue
		}
		seen[norm] = true
		names = append(names, name+"()")
	}

	if len(names) == 0 {
		return ""
	}
	if maxCount > 0 && len(names) > maxCount {
		names = names[:maxCount]
	}
	return strings.Join(names, ", ")
}

// buildFunctionSummaryFromCompletions builds a summary from completion results (functions only).
func buildFunctionSummaryFromCompletions(completions []completion.Completion, maxCount int) string {
	if len(completions) == 0 {
		return ""
	}
	candidates := make([]string, 0, len(completions))
	for _, c := range completions {
		if c.Kind != completion.CompletionFunction {
			continue
		}
		display := c.Display
		if display == "" {
			display = c.Text
		}
		if display == "" && c.Function != nil {
			display = c.Function.Name
		}
		if display == "" {
			continue
		}
		if !strings.Contains(display, "(") {
			display += "()"
		}
		candidates = append(candidates, display)
	}
	return buildFunctionSummary(candidates, maxCount)
}

// filterWithCompletionEngine uses the completion engine for expression mode intellisense
func (m *Model) filterWithCompletionEngine(allowDropdown bool) {
	if !m.AllowSuggestions {
		m.FilteredSuggestions = []string{}
		m.ShowSuggestions = false
		return
	}

	input := m.PathInput.Value()
	if input == "" {
		m.FilteredSuggestions = []string{}
		m.ShowSuggestions = false
		return
	}

	trimmedInput := strings.TrimSpace(input)
	trailingDot := strings.HasSuffix(trimmedInput, ".")

	// Prefer the evaluated result (data panel) type; only fall back to inference if empty.
	resultType := strings.TrimSpace(m.ExprType)
	if resultType == "" {
		resultType = inferExprType(m.Node)
	}
	currentType := resultType

	if m.CompletionEngine != nil && currentType == "" {
		exprForType := strings.TrimSpace(input)
		exprForType = strings.TrimSuffix(exprForType, ".")
		exprForType = strings.TrimSuffix(exprForType, "[")
		if exprForType != "" {
			typeCtx := completion.CompletionContext{
				CurrentNode:    completionRootForInput(m.Root, m.Node, exprForType),
				CursorPosition: len(exprForType),
			}
			if inferred := m.CompletionEngine.InferType(exprForType, typeCtx); inferred != "" {
				currentType = inferred
				if resultType == "" {
					resultType = inferred
				}
			}
		}
	}

	// Get completions from the engine
	partialTok := ""
	if !strings.HasSuffix(input, ".") && !strings.HasSuffix(input, "[") {
		if idx := strings.LastIndex(input, "."); idx >= 0 && idx+1 < len(input) {
			afterDot := input[idx+1:]
			if !strings.ContainsAny(afterDot, "[") {
				partialTok = strings.TrimSpace(afterDot)
			}
		}
	}
	currentNodeForCtx := m.Node
	if currentNodeForCtx == nil {
		currentNodeForCtx = completionRootForInput(m.Root, m.Node, input)
	}

	ctx := completion.CompletionContext{
		CurrentNode:          currentNodeForCtx,
		CurrentType:          currentType,
		CursorPosition:       len(input),
		PartialToken:         partialTok,
		IsAfterDot:           strings.HasSuffix(input, "."),
		ExpressionResultType: resultType,
	}

	completions := m.CompletionEngine.GetCompletions(input, ctx)

	// Convert completions to string format for display
	suggestions := make([]string, 0, len(completions))
	for _, c := range completions {
		entry := c.Text
		if c.Kind == completion.CompletionFunction {
			if c.Detail != "" {
				entry = c.Text + " - " + c.Detail
			}
		}
		suggestions = append(suggestions, entry)
	}

	m.FilteredSuggestions = suggestions
	m.Status.Completions = completions // Update status bar with completions
	m.Status.ShowCompletions = len(completions) > 0
	m.Status.SelectedCompletion = 0
	m.ShowSuggestions = len(suggestions) > 0
	m.SelectedSuggestion = 0

	if trailingDot {
		if summary := buildFunctionSummaryFromCompletions(completions, 0); summary != "" {
			parts := strings.Split(summary, ",")
			for i, p := range parts {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				if !strings.HasPrefix(p, ".") {
					p = "." + p
				}
				parts[i] = p
			}
			m.SuggestionSummary = strings.Join(parts, " ")
			m.ShowSuggestionSummary = true
		}
	}

	if !allowDropdown && !m.AllowIntellisense {
		m.FilteredSuggestions = []string{}
		m.ShowSuggestions = false
		return
	}
}

// filterSuggestions filters suggestions based on the current input text.
// If forceDropdown is true, the dropdown can be shown even when intellisense is disabled.
func (m *Model) filterSuggestions(forceDropdown ...bool) {
	allowDropdown := len(forceDropdown) > 0 && forceDropdown[0]
	m.clearSuggestionSummary()

	// If in expression mode and completion engine is available, use it
	if m.InputFocused && m.CompletionEngine != nil {
		m.filterWithCompletionEngine(allowDropdown)
		return
	}

	// Otherwise, use legacy filtering logic for navigation mode
	trailDot := strings.HasSuffix(m.PathInput.Value(), ".")
	setDotSummary := func(candidates []string) {
		if !trailDot {
			return
		}
		if summary := buildFunctionSummary(candidates, 0); summary != "" {
			parts := strings.Split(summary, ",")
			for i, p := range parts {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				if !strings.HasPrefix(p, ".") {
					p = "." + p
				}
				parts[i] = p
			}
			m.SuggestionSummary = strings.Join(parts, " ")
			m.ShowSuggestionSummary = true
		}
	}
	if !m.AllowSuggestions {
		m.FilteredSuggestions = []string{}
		m.ShowSuggestions = false
		return
	}
	if !allowDropdown && !m.AllowIntellisense {
		// Even when intellisense dropdowns are disabled, still surface function summary after a trailing dot.
		setDotSummary(m.Suggestions)
		m.FilteredSuggestions = []string{}
		m.ShowSuggestions = false
		return
	}
	input := m.PathInput.Value()
	if input == "" {
		m.FilteredSuggestions = []string{}
		m.ShowSuggestions = false
		// Restore full table when input is empty - use SyncTableState() for consistency
		// SyncTableState() will restore AllRows if they exist
		m.SyncTableState()
		return
	}

	// Look for the last word/token after dot or start
	lastToken := input
	pathBeforeToken := ""
	if idx := strings.LastIndex(input, "."); idx >= 0 {
		lastToken = input[idx+1:]
		pathBeforeToken = input[:idx]
	}

	defer func() {
		if !m.AllowSuggestions {
			m.ShowSuggestions = false
			return
		}
		if !allowDropdown && !m.AllowIntellisense {
			m.ShowSuggestions = false
		}
	}()

	// Gather context keys (current menu keys) for suggestions; used in both table and expr modes
	var contextKeys []string
	if pathBeforeToken != "" {
		// Handle special case: "_" means root
		if pathBeforeToken == "_" {
			contextKeys = getKeysFromNode(m.Root)
		} else {
			if node, err := navigator.Navigate(m.Root, pathBeforeToken); err == nil {
				contextKeys = getKeysFromNode(node)
			}
		}
	} else {
		contextKeys = getKeysFromNode(m.Node)
	}

	// Check if the current input is a complete path to an array or object
	// If so, show its children without filtering
	if !m.InputFocused {
		if node, err := navigator.Navigate(m.Root, input); err == nil {
			childKeys := getKeysFromNode(node)
			if len(childKeys) > 0 {
				allIndices := true
				for _, k := range childKeys {
					if !strings.HasPrefix(k, "[") {
						allIndices = false
						break
					}
				}
				if allIndices {
					m.FilteredSuggestions = childKeys
					m.ShowSuggestions = true
					m.SelectedSuggestion = 0
					return
				}
			}
		}
	}

	// If user opened a bracket after a path (e.g., "items[" or CEL result like "regions.map(x,x)["),
	// and the base resolves to an array, show index suggestions immediately.
	if !m.InputFocused && strings.HasSuffix(input, "[") {
		base := strings.TrimSuffix(input, "[")
		if node, err := navigator.Navigate(m.Root, base); err == nil {
			if arr, ok := node.([]interface{}); ok {
				indices := make([]string, len(arr))
				for i := range arr {
					indices[i] = fmt.Sprintf("[%d]", i)
				}
				m.FilteredSuggestions = indices
				m.ShowSuggestions = true
				m.SelectedSuggestion = 0
				return
			}
		}
	}

	// If input ends with a closed index (e.g., "items[3]"), offer sibling indices from the parent array
	// so Tab can cycle through array elements even after selecting one.
	if !m.InputFocused && strings.HasSuffix(input, "]") {
		lastOpen := strings.LastIndex(input, "[")
		lastClose := strings.LastIndex(input, "]")
		if lastOpen >= 0 && lastClose > lastOpen {
			parent := input[:lastOpen]
			if node, err := navigator.Navigate(m.Root, parent); err == nil {
				if arr, ok := node.([]interface{}); ok {
					indices := make([]string, len(arr))
					for i := range arr {
						indices[i] = fmt.Sprintf("[%d]", i)
					}
					m.FilteredSuggestions = indices
					m.ShowSuggestions = true
					m.SelectedSuggestion = 0
					return
				}
			}
		}
	}

	// If user just typed a dot and nothing after, show context keys + CEL functions
	// Build context node for type-aware filtering
	var contextNode interface{}
	if pathBeforeToken != "" {
		if pathBeforeToken == "_" {
			contextNode = m.Root
		} else {
			if node, err := navigator.Navigate(m.Root, pathBeforeToken); err == nil {
				contextNode = node
			}
		}
	} else {
		contextNode = m.Node
	}
	if strings.HasSuffix(input, ".") && lastToken == "" {
		// Place type-compatible CEL functions first when trailing dot
		funcs := []string{}
		for _, suggestion := range m.Suggestions {
			if strings.Contains(suggestion, "(") || strings.Contains(suggestion, " - ") {
				if isSuggestionCompatibleWithNode(suggestion, contextNode) {
					funcs = append(funcs, suggestion)
				}
			}
		}
		ordered := make([]string, 0, len(contextKeys)+len(funcs))
		ordered = append(ordered, funcs...)
		ordered = append(ordered, contextKeys...)
		m.FilteredSuggestions = ordered
		// Only show suggestions if there are actually suggestions to show
		m.ShowSuggestions = len(ordered) > 0
		m.SelectedSuggestion = 0
		// select first CEL function if available, else first item
		for i, cand := range ordered {
			if strings.Contains(cand, "(") || strings.Contains(cand, " - ") {
				m.SelectedSuggestion = i
				break
			}
		}
		setDotSummary(funcs)
		return
	}

	// Filter both context keys and CEL functions that start with the last token
	filtered := []string{}
	lowerToken := strings.ToLower(lastToken)
	// Special case: if input is exactly "_" or starts with "_" (root context), show all compatible functions
	isRootContext := (input == "_" || strings.HasPrefix(input, "_.") || strings.HasPrefix(input, "_[")) && m.InputFocused
	if isRootContext && contextNode == nil {
		// Use root node for type checking when at root
		contextNode = m.Root
	}

	// Keys first (current menu context). Match either raw or bracket-stripped (for array indices like [0]).
	for _, key := range contextKeys {
		kLower := strings.ToLower(key)
		stripped := kLower
		if strings.HasPrefix(key, "[") && strings.HasSuffix(key, "]") {
			stripped = strings.Trim(key, "[]\"")
			stripped = strings.ToLower(stripped)
		}
		if strings.HasPrefix(kLower, lowerToken) || strings.HasPrefix(stripped, lowerToken) {
			filtered = append(filtered, key)
		}
	}
	// Then CEL functions, filtered by type validity
	for _, suggestion := range m.Suggestions {
		funcName := suggestion
		if idx := strings.Index(suggestion, "("); idx >= 0 {
			funcName = suggestion[:idx]
		}
		// Special handling for root context: if input is "_", show all compatible functions
		// Otherwise, check if function name matches the token
		shouldInclude := false
		if isRootContext && (input == "_" || lowerToken == "_") {
			// At root context, show all compatible functions regardless of name match
			shouldInclude = contextNode == nil || isSuggestionCompatibleWithNode(suggestion, contextNode)
		} else {
			// Normal filtering: function name must match the token
			shouldInclude = strings.HasPrefix(strings.ToLower(funcName), lowerToken)
			if shouldInclude {
				// Include function only if compatible with current type (using usage hints)
				shouldInclude = contextNode == nil || isSuggestionCompatibleWithNode(suggestion, contextNode)
			}
		}
		if shouldInclude {
			filtered = append(filtered, suggestion)
		}
	}

	m.FilteredSuggestions = filtered
	// Only show suggestions if there are actually suggestions to show
	// The len(lastToken) > 0 check was preventing showing suggestions when input is "_",
	// but we should show suggestions if there are any, regardless of token length
	m.ShowSuggestions = len(filtered) > 0
	// If trailing dot, prioritize CEL functions before keys to favor function insertion.
	if strings.HasSuffix(input, ".") && len(filtered) > 0 {
		ordered := make([]string, 0, len(filtered))
		funcs := []string{}
		keys := []string{}
		for _, s := range filtered {
			if strings.Contains(s, "(") || strings.Contains(s, " - ") {
				funcs = append(funcs, s)
			} else {
				keys = append(keys, s)
			}
		}
		ordered = append(ordered, funcs...)
		ordered = append(ordered, keys...)
		filtered = ordered
		m.FilteredSuggestions = filtered
	}

	// Auto-select best match; when trailing dot with functions present, pick first function.
	m.SelectedSuggestion = 0
	lowerToken = strings.ToLower(lastToken)
	exactIdx, prefixIdx, containsIdx := -1, -1, -1
	exactFuncIdx, prefixFuncIdx, containsFuncIdx := -1, -1, -1
	for i, s := range filtered {
		isFunction := strings.Contains(s, "(") || strings.Contains(s, " - ")
		if strings.HasSuffix(input, ".") && isFunction {
			// first function wins under trailing dot
			exactIdx = i
			break
		}
		// Extract the name to match against
		cand := s
		if isFunction {
			// For functions, extract the function name (before " - " or first "(")
			funcName := s
			if idx := strings.Index(s, " - "); idx >= 0 {
				funcName = s[:idx]
			} else if idx := strings.Index(s, "("); idx >= 0 {
				funcName = s[:idx]
			}
			// Remove namespace prefix if present (e.g., "string.matches" -> "matches")
			if dotIdx := strings.LastIndex(funcName, "."); dotIdx >= 0 {
				funcName = funcName[dotIdx+1:]
			}
			// Remove parentheses if present
			funcName = strings.TrimSuffix(strings.TrimSuffix(funcName, "()"), "(")
			cand = funcName
		} else if strings.HasPrefix(cand, "[") && strings.HasSuffix(cand, "]") {
			// normalize bracketed keys for matching
			cand = cand[1 : len(cand)-1]
		}
		lc := strings.ToLower(cand)
		if isFunction {
			// Track function matches separately
			if lc == lowerToken && exactFuncIdx == -1 {
				exactFuncIdx = i
			}
			if strings.HasPrefix(lc, lowerToken) && prefixFuncIdx == -1 {
				prefixFuncIdx = i
			}
			if strings.Contains(lc, lowerToken) && containsFuncIdx == -1 {
				containsFuncIdx = i
			}
		} else {
			// Track key matches
			if lc == lowerToken && exactIdx == -1 {
				exactIdx = i
			}
			if strings.HasPrefix(lc, lowerToken) && prefixIdx == -1 {
				prefixIdx = i
			}
			if strings.Contains(lc, lowerToken) && containsIdx == -1 {
				containsIdx = i
			}
		}
	}
	// Prefer function matches over key matches when there's a partial token in expr mode
	if m.InputFocused && lastToken != "" {
		switch {
		case exactFuncIdx >= 0:
			m.SelectedSuggestion = exactFuncIdx
		case prefixFuncIdx >= 0:
			m.SelectedSuggestion = prefixFuncIdx
		case containsFuncIdx >= 0:
			m.SelectedSuggestion = containsFuncIdx
		case exactIdx >= 0:
			m.SelectedSuggestion = exactIdx
		case prefixIdx >= 0:
			m.SelectedSuggestion = prefixIdx
		case containsIdx >= 0:
			m.SelectedSuggestion = containsIdx
		default:
			m.SelectedSuggestion = 0
		}
		return
	}

	// Original logic: prefer keys over functions when not in focused token mode
	switch {
	case exactIdx >= 0:
		m.SelectedSuggestion = exactIdx
	case prefixIdx >= 0:
		m.SelectedSuggestion = prefixIdx
	case containsIdx >= 0:
		m.SelectedSuggestion = containsIdx
	case exactFuncIdx >= 0:
		m.SelectedSuggestion = exactFuncIdx
	case prefixFuncIdx >= 0:
		m.SelectedSuggestion = prefixFuncIdx
	case containsFuncIdx >= 0:
		m.SelectedSuggestion = containsFuncIdx
	default:
		m.SelectedSuggestion = 0
	}

	setDotSummary(filtered)

	// Live table filtering: only in table mode (not expr mode)
	if lastToken != "" && !m.InputFocused {
		// Determine source rows
		sourceRows := m.AllRows
		// Get current column widths for proper truncation
		keyW := m.KeyColWidth
		if keyW <= 0 {
			keyW = 30
		}
		valueW := m.ValueColWidth
		if valueW <= 0 {
			valueW = 60
		}
		if pathBeforeToken != "" {
			if ctxNode, err := navigator.Navigate(m.Root, pathBeforeToken); err == nil {
				ctxStringRows := navigator.NodeToRows(ctxNode)
				sourceRows = styleRowsWithWidths(ctxStringRows, keyW, valueW)
			}
		}
		// Build a filtered set of rows from sourceRows based on token
		tokenLower := strings.ToLower(lastToken)
		filteredRows := []table.Row{}
		for _, r := range sourceRows {
			key := r[0]
			// Keys may be bracketed indices; normalize for matching
			cand := key
			if strings.HasPrefix(cand, "[") && strings.HasSuffix(cand, "]") {
				cand = cand[1 : len(cand)-1]
			}
			if strings.HasPrefix(strings.ToLower(cand), tokenLower) {
				filteredRows = append(filteredRows, r)
			}
		}
		// Store filtered rows and activate suggestion filtering
		if len(filteredRows) > 0 {
			m.FilteredSuggestionRows = filteredRows
			m.SuggestionFilterActive = true
		} else {
			// If no row matches, keep original rows to avoid empty table confusion
			m.FilteredSuggestionRows = m.AllRows
			m.SuggestionFilterActive = true
		}
		// Use SyncTableState() to apply the filtered rows
		m.SyncTableState()
	}
}

// getKeysFromNode extracts available keys from a node
func getKeysFromNode(node interface{}) []string {
	switch t := node.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(t))
		for k := range t {
			// Check if key needs bracket notation (contains special chars)
			if needsBracketNotation(k) {
				keys = append(keys, fmt.Sprintf(`["%s"]`, k))
			} else {
				keys = append(keys, k)
			}
		}
		// Ensure stable ordering for suggestions/navigation
		sort.Strings(keys)
		return keys
	case []interface{}:
		// For arrays, return numeric indices in bracket notation
		indices := make([]string, len(t))
		for i := range t {
			indices[i] = fmt.Sprintf("[%d]", i)
		}
		return indices
	default:
		return []string{}
	}
}

// needsBracketNotation checks if a key contains characters that require bracket notation
func needsBracketNotation(key string) bool {
	// Keys with hyphens, spaces, or other special chars need bracket notation
	for _, ch := range key {
		if !isAlphaNumOrUnderscore(ch) {
			return true
		}
	}
	return false
}

func isAlphaNumOrUnderscore(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_'
}

// performAdvancedSearch recursively searches a node (and its children) for key-value pairs matching the query.
// Returns results with relative paths (relative to the search root). Searches both keys and values (case-insensitive substring match).
// If limit > 0, stops after collecting that many results and returns limited=true.
func performAdvancedSearch(node interface{}, query string, limit int) (results []SearchResult, limited bool) {
	if query == "" {
		return []SearchResult{}, false
	}

	queryLower := strings.ToLower(query)
	results = []SearchResult{}

	// Recursive function to search a node at a given path
	var searchRecursive func(node interface{}, currentPath string) bool
	searchRecursive = func(node interface{}, currentPath string) bool {
		// Check limit
		if limit > 0 && len(results) >= limit {
			return true // stop searching
		}

		switch t := node.(type) {
		case map[string]interface{}:
			// Sort keys for deterministic results
			keys := make([]string, 0, len(t))
			for k := range t {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, k := range keys {
				// Check limit before processing each key
				if limit > 0 && len(results) >= limit {
					return true
				}

				v := t[k]
				valueStr := formatter.Stringify(v)

				// Check if key or value matches (case-insensitive substring)
				keyMatches := strings.Contains(strings.ToLower(k), queryLower)
				valueMatches := strings.Contains(strings.ToLower(valueStr), queryLower)

				if keyMatches || valueMatches {
					// Build full path
					var fullPath string
					if currentPath == "" {
						fullPath = k
					} else {
						// Check if key needs bracket notation
						if needsBracketNotation(k) {
							fullPath = currentPath + `["` + k + `"]`
						} else {
							fullPath = currentPath + "." + k
						}
					}

					// Display key with bracket notation if needed
					displayKey := k
					if needsBracketNotation(k) {
						displayKey = `["` + k + `"]`
					}

					results = append(results, SearchResult{
						FullPath: fullPath,
						Key:      displayKey,
						Value:    valueStr,
						Node:     v,
					})

					// Check if we hit the limit after adding
					if limit > 0 && len(results) >= limit {
						return true
					}
				}

				// Recursively search nested structures
				if vMap, ok := v.(map[string]interface{}); ok {
					var nextPath string
					if currentPath == "" {
						if needsBracketNotation(k) {
							nextPath = `["` + k + `"]`
						} else {
							nextPath = k
						}
					} else {
						if needsBracketNotation(k) {
							nextPath = currentPath + `["` + k + `"]`
						} else {
							nextPath = currentPath + "." + k
						}
					}
					if searchRecursive(vMap, nextPath) {
						return true
					}
				} else if vArr, ok := v.([]interface{}); ok {
					var basePath string
					if currentPath == "" {
						if needsBracketNotation(k) {
							basePath = `["` + k + `"]`
						} else {
							basePath = k
						}
					} else {
						if needsBracketNotation(k) {
							basePath = currentPath + `["` + k + `"]`
						} else {
							basePath = currentPath + "." + k
						}
					}
					if searchRecursive(vArr, basePath) {
						return true
					}
				}
			}

		case []interface{}:
			for i, v := range t {
				// Check limit before processing each element
				if limit > 0 && len(results) >= limit {
					return true
				}

				valueStr := formatter.Stringify(v)
				// Check if value matches (arrays don't have keys to match)
				valueMatches := strings.Contains(strings.ToLower(valueStr), queryLower)

				if valueMatches {
					// Build full path with array index
					fullPath := fmt.Sprintf("%s[%d]", currentPath, i)
					displayKey := fmt.Sprintf("[%d]", i)

					results = append(results, SearchResult{
						FullPath: fullPath,
						Key:      displayKey,
						Value:    valueStr,
						Node:     v,
					})

					// Check if we hit the limit after adding
					if limit > 0 && len(results) >= limit {
						return true
					}
				}

				// Recursively search nested structures
				if vMap, ok := v.(map[string]interface{}); ok {
					nextPath := fmt.Sprintf("%s[%d]", currentPath, i)
					if searchRecursive(vMap, nextPath) {
						return true
					}
				} else if vArr, ok := v.([]interface{}); ok {
					nextPath := fmt.Sprintf("%s[%d]", currentPath, i)
					if searchRecursive(vArr, nextPath) {
						return true
					}
				}
			}
		}
		return false
	}

	// Start search from the provided node with empty path (paths will be relative to this node)
	limited = searchRecursive(node, "")
	return results, limited
}

// SearchRows returns key/value rows that match the query, using the same search
// logic and display rules as the advanced search view.
func SearchRows(node interface{}, query string) [][]string {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil
	}
	results, _ := performAdvancedSearch(node, q, 0) // 0 = no limit
	rows := make([][]string, 0, len(results))
	for _, res := range results {
		key := res.Key
		if !isCompositeNode(res.Node) {
			key = "(value)"
		}
		if strings.TrimSpace(key) == "" {
			key = res.FullPath
		}
		rows = append(rows, []string{key, res.Value})
	}
	return rows
}

func (m *Model) Init() tea.Cmd {
	return textinput.Blink
}

// syncPathInputWithCursor updates the expr input to reflect the currently highlighted row in table mode.
// It does not change the active path (navigation occurs on Enter); it only mirrors the selection.
func (m *Model) syncPathInputWithCursor() {
	if m.InputFocused {
		return
	}
	path := strings.TrimSpace(m.selectedRowPath())
	if path == "" {
		m.PathInput.SetValue("_")
		return
	}
	// Update the path input to reflect the selected row
	// In table mode, reflect the `_` root context in the synced input
	m.PathInput.SetValue(formatPathForDisplay(path))
}

// navigateExprForward drills into the currently selected table row while staying in expr mode.
// It mirrors the non-input navigation logic but preserves input focus/state.
//
//nolint:unparam,unused // command return currently unused
func (m *Model) navigateExprForward() (tea.Model, tea.Cmd) {
	selectedKey, ok := m.selectedRowKey()
	if !ok {
		return m, nil
	}

	if selectedKey == "(value)" {
		// Scalar: stay put but keep input focused
		formattedPath := formatPathForDisplay(m.Path)
		m.PathInput.SetValue(formattedPath)
		m.PathInput.SetCursor(len(formattedPath))
		return m, nil
	}

	navigatePath := buildPathWithKey(m.Path, selectedKey)

	newNode, err := navigator.Resolve(m.Root, navigatePath)
	if err != nil {
		m.ErrMsg = fmt.Sprintf("Error: %v", err)
		m.StatusType = "error"
		return m, nil
	}

	newModel := m.NavigateTo(newNode, normalizePathForModel(navigatePath))
	newModel.PathKeys = parsePathKeys(newModel.Path)
	newModel.LastKey = m.LastKey
	// Stay in expr mode/input-focused
	newModel.InputFocused = true
	formatted := formatPathForDisplay(newModel.Path)
	newModel.setExprResult(formatted, newModel.Node)
	newModel.PathInput.SetValue(formatted)
	newModel.PathInput.SetCursor(len(formatted))
	newModel.PathInput.Focus()
	return newModel, nil
}

// navigateExprBackward ascends one level while staying in expr mode.
//
//nolint:unparam,unused // command return currently unused
func (m *Model) navigateExprBackward() (tea.Model, tea.Cmd) {
	if m.Path == "" {
		return m, nil
	}
	// For CEL expressions that include map/filter, bail out (cannot ascend cleanly)
	if strings.Contains(m.Path, "filter(") || strings.Contains(m.Path, "map(") {
		newModel := m.NavigateTo(m.Root, "")
		newModel.InputFocused = true
		newModel.PathInput.SetValue("_")
		newModel.PathInput.SetCursor(len(newModel.PathInput.Value()))
		newModel.PathInput.Focus()
		return newModel, nil
	}

	newPath := removeLastSegment(m.Path)
	var newNode interface{}
	var err error
	if newPath == "" {
		newNode = m.Root
	} else {
		newNode, err = navigator.Resolve(m.Root, newPath)
		if err != nil {
			m.ErrMsg = fmt.Sprintf("Error: %v", err)
			m.StatusType = "error"
			return m, nil
		}
	}

	newModel := m.NavigateTo(newNode, normalizePathForModel(newPath))
	newModel.InputFocused = true
	formatted := formatPathForDisplay(newModel.Path)
	newModel.setExprResult(formatted, newModel.Node)
	newModel.PathInput.SetValue(formatted)
	newModel.PathInput.SetCursor(len(formatted))
	newModel.PathInput.Focus()
	return newModel, nil
}

// applyLayout adjusts table height and column widths based on window size
// forceRegenerate forces row regeneration even if dimensions haven't changed
func (m *Model) applyLayout(forceRegenerate ...bool) {
	// Fallback dimensions before the first WindowSizeMsg arrives to avoid 0-width popups
	// that would wrap text vertically (e.g., a modal rendered with width=1).
	if m.WinWidth <= 0 {
		m.WinWidth = 80
	}
	if m.WinHeight <= 0 {
		m.WinHeight = 24
	}
	// Update layout manager with current window dimensions
	m.Layout.SetDimensions(m.WinWidth, m.WinHeight)

	// Calculate component heights using layout manager (reserve debug line only when debug is on)
	helpInline := m.HelpVisible && strings.ToLower(m.HelpPopupAnchor) != "top"
	// Suggestions are now shown in the status bar, not as a dropdown - no space reservation needed
	heights := m.Layout.CalculateHeights(helpInline, m.DebugMode, false, m.ShowPanelTitle)
	popupTopLines := m.infoPopupTopHeight() + m.helpAnchoredTopHeight()
	if popupTopLines > 0 && heights.TableHeight > MinTableHeight {
		heights.TableHeight -= popupTopLines
		if heights.TableHeight < MinTableHeight {
			heights.TableHeight = MinTableHeight
		}
	}

	// Set table height
	m.Tbl.SetHeight(heights.TableHeight + TableHeaderLines)
	m.TableHeight = heights.TableHeight

	// Calculate column widths using layout manager; shrink key column to visible max but cap at configured/default.
	keyW, valueW := m.Layout.CalculateColumnWidths(m.ConfiguredKeyColWidth, m.ConfiguredValueColWidth, m.autoKeyColumnWidth)

	// Check if column widths actually changed
	widthsChanged := m.KeyColWidth != keyW || m.ValueColWidth != valueW
	shouldRegenerate := len(forceRegenerate) > 0 && forceRegenerate[0]

	m.KeyColWidth = keyW
	m.ValueColWidth = valueW

	// Set PathInput width using layout manager
	if m.WinWidth > 0 {
		m.PathInput.SetWidth(m.Layout.CalculateInputWidth())

		// Only regenerate rows if widths changed or forced, to prevent flicker
		if widthsChanged || shouldRegenerate {
			// If in search mode, regenerate from search results, not from node
			switch {
			case m.AdvancedSearchActive && len(m.AdvancedSearchResults) > 0:
				stringRows := make([][]string, len(m.AdvancedSearchResults))
				for i, result := range m.AdvancedSearchResults {
					stringRows[i] = []string{result.Key, result.Value}
				}
				m.AllRows = styleRowsWithWidths(stringRows, keyW, valueW)
				m.AllRowKeys = extractRowKeys(stringRows)
			case m.MapFilterActive:
				// Re-apply map filter with new column widths
				m.applyMapFilter()
			case m.Node != nil:
				stringRows := navigator.NodeToRows(m.Node)
				m.AllRows = styleRowsWithWidths(stringRows, keyW, valueW)
				m.AllRowKeys = extractRowKeys(stringRows)
				// Preserve active type-ahead filter across resizes
				if m.FilterActive && m.FilterBuffer != "" {
					m.applyTypeAheadFilter()
				}
			}
			// Note: AllRows is preserved if widths didn't change
		}
	}

	// Sync table state (sets columns, rows, cursor, focus) - centralized state management
	m.SyncTableState()
}

// SyncTableState ensures the table state (columns, rows, cursor, focus) matches the model state.
// This should be called from Update() after any state change, NOT from View().
// View() should be pure and only read state.
// resetCursorToZero: if true, always reset cursor to 0 (e.g., when search results first appear)
// This is exported so it can be used by CLI rendering code.
func (m *Model) SyncTableState(resetCursorToZero ...bool) {
	// Calculate column widths
	keyW := m.KeyColWidth
	if keyW <= 0 {
		keyW = 30
	}
	valueW := m.ValueColWidth
	if valueW <= 0 {
		valueW = 60
	}

	// Set columns with configurable headers
	keyHeader := m.KeyHeader
	if keyHeader == "" {
		keyHeader = "KEY"
	}
	valueHeader := m.ValueHeader
	if valueHeader == "" {
		valueHeader = "VALUE"
	}
	cols := []table.Column{
		{Title: keyHeader, Width: keyW},
		{Title: valueHeader, Width: valueW},
	}
	m.Tbl.SetColumns(cols)
	// In v2, we need to explicitly set the table width (include column separator spacing).
	tableWidth := keyW + valueW + ColumnSeparatorWidth
	m.Tbl.SetWidth(tableWidth)

	// Determine which rows to use based on current state
	var rows []table.Row
	switch {
	case m.AdvancedSearchActive && len(m.AdvancedSearchResults) > 0:
		// In search mode with results: use search results
		stringRows := make([][]string, len(m.AdvancedSearchResults))
		for i, result := range m.AdvancedSearchResults {
			key := result.Key
			value := result.Value
			if key == "" && result.FullPath != "" {
				key = result.FullPath
			}
			if value == "" && result.Node != nil {
				value = formatter.Stringify(result.Node)
			}
			stringRows[i] = []string{key, value}
		}
		if m.TruncateTableCells {
			rows = styleRowsWithWidths(stringRows, keyW, valueW)
		} else {
			rows = styleRowsNoTruncation(stringRows, keyW, valueW)
		}
		m.AllRows = rows
		m.AllRowKeys = extractRowKeys(stringRows)
	case m.AdvancedSearchActive:
		// In search mode but no results yet (e.g., F3 pressed, query empty) - show empty table
		rows = []table.Row{}
		m.AllRows = rows
		m.AllRowKeys = nil
	case m.SuggestionFilterActive && len(m.FilteredSuggestionRows) > 0:
		// Use filtered suggestion rows (from path input filtering)
		rows = m.FilteredSuggestionRows
	case len(m.AllRows) > 0:
		// Use existing rows (preserves filtered state, etc.)
		rows = m.AllRows
	case m.Node != nil:
		// Generate rows from current node
		stringRows := navigator.NodeToRows(m.Node)
		if m.TruncateTableCells {
			rows = styleRowsWithWidths(stringRows, keyW, valueW)
		} else {
			rows = styleRowsNoTruncation(stringRows, keyW, valueW)
		}
		m.AllRows = rows
		m.AllRowKeys = extractRowKeys(stringRows)
	default:
		// No data - empty table
		rows = []table.Row{}
		m.AllRows = rows
		m.AllRowKeys = nil
	}

	// Apply type-ahead filter if active
	if m.FilterActive && m.FilterBuffer != "" && !m.AdvancedSearchActive && !m.SuggestionFilterActive {
		// Filter rows based on FilterBuffer
		filterLower := strings.ToLower(m.FilterBuffer)
		filteredRows := []table.Row{}
		for _, r := range rows {
			key := r[0]
			// Keys may be bracketed indices; normalize for matching
			cand := key
			if strings.HasPrefix(cand, "[") && strings.HasSuffix(cand, "]") {
				cand = cand[1 : len(cand)-1]
			}
			if strings.HasPrefix(strings.ToLower(cand), filterLower) {
				filteredRows = append(filteredRows, r)
			}
		}
		if len(filteredRows) > 0 {
			rows = filteredRows
		} else {
			// If no row matches, show empty table to indicate no matches
			rows = []table.Row{}
		}
	}

	// Set rows
	m.Tbl.SetRows(rows)
	// Use the layout-reserved table height for overall layout, but render only the rows we have.
	// This prevents padding rows inside the table while still keeping the footer anchored via outer padding.
	reservedHeight := m.TableHeight
	if reservedHeight <= 0 {
		reservedHeight = len(rows)
	}
	if reservedHeight < MinTableHeight {
		reservedHeight = MinTableHeight
	}

	// The table component's SetHeight sets the body height (rows), not including the header
	// The header (2 lines: header + border) is always shown separately
	// We need to ensure the table has enough height to show the header properly
	actualHeight := reservedHeight
	if len(rows) < actualHeight {
		// If we have fewer rows than reserved, use the number of rows
		// But ensure we always have at least 1 row to show header + 1 row
		actualHeight = len(rows)
		if actualHeight < 1 {
			actualHeight = 1
		}
	}
	// Always ensure minimum height to show header properly
	// MinTableHeight (2) ensures header + 1 row, but since SetHeight is body-only, we need at least 1
	if actualHeight < 1 {
		actualHeight = 1
	}
	// However, if we have enough rows, use the reserved height to maintain layout spacing
	// This prevents the table from shrinking when suggestions appear/disappear
	if len(rows) >= reservedHeight {
		actualHeight = reservedHeight
	}
	m.Tbl.SetHeight(actualHeight + TableHeaderLines)

	// Handle cursor positioning
	shouldResetToZero := len(resetCursorToZero) > 0 && resetCursorToZero[0]
	currentCursor := m.Tbl.Cursor()
	switch {
	case shouldResetToZero:
		// Explicitly reset to 0 (e.g., when search results first appear)
		m.Tbl.SetCursor(0)
	case len(rows) > 0:
		// Only reset cursor if it's out of bounds (preserve valid positions)
		if currentCursor >= len(rows) {
			m.Tbl.SetCursor(0)
		}
		// Note: We preserve valid cursor positions even when search is active
		// This allows navigation (up/down) to work after restoring search context
	default:
		m.Tbl.SetCursor(0)
	}

	// Ensure focus is correct (centralized focus management)
	if !m.InputFocused {
		m.Tbl.Focus()
		m.PathInput.Blur() // Hide cursor in path input when not in input mode
	} else {
		m.Tbl.Blur()
		// PathInput.Focus() is called explicitly when entering input mode (expr toggle handler)
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	defer m.clearStickyErrorIfInputChanged()

	// Ensure all components stay synchronized with state after handling any message.
	// This keeps View() free of mutations.
	defer m.syncAllComponents()

	switch msg := msg.(type) {
	case SearchDebounceMsg:
		// Only execute search if this is the latest debounce request
		if msg.ID == m.SearchDebounceID && msg.Query == m.SearchPendingQuery {
			m.AdvancedSearchQuery = msg.Query
			m.applyAdvancedSearch()
		}
		return m, nil

	case tea.WindowSizeMsg:
		targetW := msg.Width
		targetH := msg.Height
		if m.ForceWindowSize {
			if m.DesiredWinWidth > 0 {
				targetW = m.DesiredWinWidth
			}
			if m.DesiredWinHeight > 0 {
				targetH = m.DesiredWinHeight
			}
		}

		// Only update if dimensions actually changed to avoid unnecessary resets
		if m.WinWidth == targetW && m.WinHeight == targetH {
			return m, nil
		}

		// Capture terminal size and adjust table height to keep header visible
		m.WinWidth = targetW
		m.WinHeight = targetH
		// Apply layout WITHOUT forcing row regeneration - let it decide based on width changes
		m.applyLayout(false) // Don't force regenerate on every resize
		// Focus is handled by SyncTableState() below
		return m, nil

	case tea.KeyMsg:
		m.LastKey = msg.String()
		keyStr := msg.String()
		m.logKeyEvent(keyStr)

		if m.ShowInfoPopup && m.InfoPopupModal {
			switch keyStr {
			case "esc":
				m.setShowInfoPopup(false)
			case "ctrl+c":
				return m, tea.Quit
			}
			return m, nil
		}

		// When help is visible, treat it as modal: allow closing with F1/esc or quitting with ctrl+c.
		if m.HelpVisible {
			switch keyStr {
			case "f1", "esc":
				m.HelpVisible = false
				m.applyLayout(true) // Recalculate layout when help closes
				return m, nil
			case "ctrl+c":
				return m, tea.Quit
			default:
				return m, nil
			}
		}

		if handled, cmd := m.handleMenuKey(keyStr); handled {
			return m, cmd
		}
		if handled, cmd := m.handleSearchInput(msg, keyStr); handled {
			return m, cmd
		}

		// Handle left arrow for back navigation only when not in expr (input) mode
		if !m.InputFocused && (keyStr == "left" || keyStr == "alt+left") {
			// If we're in active search mode (F3 pressed, viewing search results), check if we're at the base path
			// If so, prevent navigation further left - the base path is the limit
			if m.AdvancedSearchActive && m.AdvancedSearchBasePath != "" {
				normalizedCurrent := navigator.NormalizePath(m.Path)
				normalizedBase := navigator.NormalizePath(m.AdvancedSearchBasePath)
				if normalizedCurrent == normalizedBase {
					// We're already at the base path - don't allow navigation further left
					return m, nil
				}
			}

			// If we're in search context (navigated from search with right arrow), check if we should restore search
			if m.SearchContextActive {
				basePath := m.SearchContextBasePath
				currentPath := m.Path

				// Normalize paths for comparison
				normalizedBase := navigator.NormalizePath(basePath)
				normalizedCurrent := navigator.NormalizePath(currentPath)

				// Calculate what the parent path would be after going back one level
				parentPath := removeLastSegment(currentPath)
				normalizedParent := normalizePathForModel(parentPath)

				// Build a set of all search result absolute paths for smarter checking
				// This allows us to check if we're navigating within any search result's subtree
				searchResultPaths := make(map[string]bool)
				for _, result := range m.SearchContextResults {
					// Combine base path with result path (same logic as when navigating into result)
					var absPath string
					if basePath == "" {
						absPath = result.FullPath
					} else {
						if strings.HasPrefix(result.FullPath, "[") {
							absPath = basePath + result.FullPath
						} else {
							absPath = basePath + "." + result.FullPath
						}
					}
					normalizedAbsPath := normalizePathForModel(absPath)
					searchResultPaths[normalizedAbsPath] = true
				}

				// Helper function to check if a path is within the search space
				isWithinSearchSpace := func(path string) bool {
					normalizedPath := normalizePathForModel(path)

					// Check if path equals base path
					if normalizedPath == normalizedBase {
						return true
					}

					// Check if path is a descendant of base path
					if normalizedBase != "" {
						prefix := normalizedBase + "."
						if strings.HasPrefix(normalizedPath, prefix) {
							return true
						}
					}

					// Check if path equals any search result path
					if searchResultPaths[normalizedPath] {
						return true
					}

					// Check if path is a descendant of any search result path
					for resultPath := range searchResultPaths {
						if resultPath != "" {
							prefix := resultPath + "."
							if strings.HasPrefix(normalizedPath, prefix) {
								return true
							}
						}
					}

					return false
				}

				// Check if current path is within search space
				currentInSearchSpace := isWithinSearchSpace(currentPath)

				// Check if parent path would be at or past the base path
				// (i.e., parent is base, or parent is not within search space anymore)
				parentInSearchSpace := isWithinSearchSpace(parentPath)
				parentIsAtOrPastBase := normalizedParent == normalizedBase || !parentInSearchSpace

				// Check if current path is exactly a search result path (top-level result)
				// This is important: if we're at a top-level result and go left, we should restore search
				normalizedCurrentPath := normalizePathForModel(currentPath)
				currentIsSearchResult := searchResultPaths[normalizedCurrentPath]

				// Restore search if:
				// 1. We're already at the base path, OR
				// 2. We're at a search result path and going back would take us to or past the base path, OR
				// 3. We're within search space and going back would take us to or past the base path
				// This ensures the base path is the "farthest left" we can go while preserving search context
				shouldRestoreSearch := normalizedCurrent == normalizedBase ||
					(currentIsSearchResult && parentIsAtOrPastBase) ||
					(currentInSearchSpace && parentIsAtOrPastBase)

				if shouldRestoreSearch {
					// We're at or would go past the base path - restore search view
					var searchNode interface{}
					var err error
					if basePath == "" {
						searchNode = m.Root
					} else {
						searchNode, err = navigator.Resolve(m.Root, basePath)
						if err != nil {
							// If we can't navigate to base path, just clear search context
							m.SearchContextActive = false
							return m, nil
						}
					}

					// Restore search state - ensure we have valid search context data
					// Validate that search context has valid data (not just empty structs)
					hasValidResults := len(m.SearchContextResults) > 0 && m.SearchContextQuery != ""
					if hasValidResults {
						// Check if at least one result has non-empty Key or Value
						hasValidData := false
						for _, result := range m.SearchContextResults {
							if result.Key != "" || result.Value != "" {
								hasValidData = true
								break
							}
						}
						hasValidResults = hasValidData
					}

					if !hasValidResults {
						// Search context is invalid - clear it and do normal navigation
						m.SearchContextActive = false
						// Fall through to normal back navigation
					} else {
						// We have valid search context - restore it
						m.AdvancedSearchActive = true
						m.AdvancedSearchCommitted = true // Context came from a committed deep search
						m.AdvancedSearchQuery = m.SearchContextQuery
						m.AdvancedSearchResults = make([]SearchResult, len(m.SearchContextResults))
						copy(m.AdvancedSearchResults, m.SearchContextResults)
						m.AdvancedSearchBasePath = m.SearchContextBasePath
						// Set Node to searchNode for path tracking, but rows will come from AdvancedSearchResults
						// NOT from m.Node - this is critical to prevent showing wrong data
						m.Node = searchNode
						// Normalize basePath to ensure it has _ prefix for consistency
						m.Path = normalizePathForModel(basePath)
						// CRITICAL: Keep SearchContextActive = true so that right arrow can navigate and preserve context
						// Only Enter should clear SearchContextActive
						// m.SearchContextActive = false  // REMOVED - keep it true so right arrow works
						// CRITICAL: Ensure AdvancedSearchActive is set BEFORE building rows
						// This ensures applyLayout and other functions use search results, not m.Node

						// Restore search results in table (same logic as applyAdvancedSearch)
						keyW := m.KeyColWidth
						if keyW <= 0 {
							keyW = 30
						}
						valueW := m.ValueColWidth
						if valueW <= 0 {
							valueW = 60
						}

						// Build table rows from search results
						// Verify results have data before using them
						stringRows := make([][]string, 0, len(m.AdvancedSearchResults))
						for i, result := range m.AdvancedSearchResults {
							// Ensure Key and Value are not empty (defensive check)
							key := result.Key
							value := result.Value
							if key == "" && result.FullPath != "" {
								// Fallback: use FullPath if Key is empty
								key = result.FullPath
							}
							if value == "" {
								// Fallback: stringify the Node if Value is empty
								if result.Node != nil {
									value = formatter.Stringify(result.Node)
								}
							}
							stringRows = append(stringRows, []string{key, value})

							// Debug: log first few results to see what we have
							if i < 3 {
								m.DebugTransient = fmt.Sprintf("Restore[%d]: Key=%q Value=%q FullPath=%q", i, key, value, result.FullPath)
							}
						}

						rows := styleRowsWithWidths(stringRows, keyW, valueW)
						// Set AllRows so SyncTableState() can use it
						m.AllRows = rows
						m.AllRowKeys = extractRowKeys(stringRows)

						// Ensure table height is set (critical for rendering)
						// This needs to be done before SyncTableState() since SyncTableState() doesn't set height
						if m.Layout != nil {
							m.Layout.SetDimensions(m.WinWidth, m.WinHeight)
							heights := m.Layout.CalculateHeights(m.HelpVisible, m.DebugMode, m.ShowSuggestions, m.ShowPanelTitle)
							if heights.TableHeight > 0 {
								m.Tbl.SetHeight(heights.TableHeight + TableHeaderLines)
								m.TableHeight = heights.TableHeight
							} else {
								// Fallback: ensure table has at least some height
								m.Tbl.SetHeight(5 + TableHeaderLines)
								m.TableHeight = 5
							}
						}

						// Use SyncTableState() to set columns, rows, cursor, and focus consistently
						// Since AdvancedSearchActive is true and AdvancedSearchResults is set,
						// SyncTableState() will use the search results (which we just set in AllRows)
						m.SyncTableState()

						// Update expr window with first result's path
						if len(rows) > 0 {
							m.syncPathInputWithCursor()
						} else {
							m.PathInput.SetValue("_")
						}

						m.clearErrorUnlessSticky()

						// IMPORTANT: Do NOT call applyLayout here - it might regenerate rows from m.Node
						// Instead, we've already set the rows from search results above
						// Sync all components to update UI state
						m.syncAllComponents()

						// Final verification: ensure table has rows after restore
						// Check immediately after setting to catch any issues
						actualRows := m.Tbl.Rows()
						if len(actualRows) == 0 && len(m.AdvancedSearchResults) > 0 {
							// Force re-set if table is still empty - something cleared it
							// Use SyncTableState() to restore table state consistently
							m.SyncTableState()
						}

						// Final check: verify table is properly configured and has rows
						// This is a defensive check to catch any issues after all setup
						if m.AdvancedSearchActive && len(m.AdvancedSearchResults) > 0 {
							// Ensure layout is applied
							if m.Layout != nil {
								m.Layout.SetDimensions(m.WinWidth, m.WinHeight)
								heights := m.Layout.CalculateHeights(m.HelpVisible, m.DebugMode, m.ShowSuggestions, m.ShowPanelTitle)
								if heights.TableHeight > 0 {
									m.Tbl.SetHeight(heights.TableHeight + TableHeaderLines)
									m.TableHeight = heights.TableHeight
								}
							}

							// Final verification: table must have rows
							if len(m.Tbl.Rows()) == 0 {
								// Table is empty - force restore using SyncTableState()
								// Ensure height is set first
								if m.TableHeight <= 0 {
									if m.Layout != nil {
										heights := m.Layout.CalculateHeights(m.HelpVisible, m.DebugMode, m.ShowSuggestions, m.ShowPanelTitle)
										if heights.TableHeight > 0 {
											m.Tbl.SetHeight(heights.TableHeight + TableHeaderLines)
											m.TableHeight = heights.TableHeight
										} else {
											m.Tbl.SetHeight(5 + TableHeaderLines)
											m.TableHeight = 5
										}
									}
								}
								// Use SyncTableState() to restore table state consistently (including focus)
								m.SyncTableState()
							}
						}

						// Sync all components to ensure UI is updated
						m.syncAllComponents()

						return m, nil
					}
				}
				// If we're deeper than base path, fall through to normal back navigation
				// (which will preserve SearchContextActive for the next left arrow)
			}
			// For CEL expressions (containing filter, map, etc.), going back returns to root
			if strings.Contains(m.Path, "filter(") || strings.Contains(m.Path, "map(") {
				newModel := m.NavigateTo(m.Root, "")
				newModel.applyLayout(true)
				return newModel, nil
			} // For regular paths, remove the last segment from the path string
			if m.Path != "" {
				newPath := removeLastSegment(m.Path)
				var newNode interface{}
				var err error
				if newPath == "" {
					newNode = m.Root
				} else {
					newNode, err = navigator.Resolve(m.Root, newPath)
					if err != nil {
						m.ErrMsg = fmt.Sprintf("Error: %v", err)
						return m, nil
					}
				}
				newModel := m.NavigateTo(newNode, newPath)
				newModel.applyLayout(true)
				newModel.restoreCursorForPath(newModel.Path)
				return newModel, nil
			}
			return m, nil
		} // Handle input mode
		if m.InputFocused {
			// Cache current input state for reuse in multiple key handlers
			currentValue := m.PathInput.Value()
			cursorPos := m.PathInput.Position()

			// Normalize special keys from Type to string for tests that construct KeyMsg directly
			switch msg.Key().Code { //nolint:exhaustive
			case tea.KeyTab:
				// Check Mod field directly for shift detection (more reliable than String())
				if msg.Key().Mod&tea.ModShift != 0 {
					keyStr = "shift+tab"
				} else {
					keyStr = "tab"
				}
			case tea.KeyEnter:
				keyStr = "enter"
			case tea.KeyRight:
				keyStr = "right"
			case tea.KeyLeft:
				keyStr = "left"
			case tea.KeyUp:
				keyStr = "up"
			case tea.KeyDown:
				keyStr = "down"
			case tea.KeyEscape:
				keyStr = "esc"
			case 0x03: // Ctrl+C
				keyStr = "ctrl+c"
			case 0x15: // Ctrl+U
				keyStr = "ctrl+u"
			default:
				// Other key types are handled by default keyStr value
			}

			switch keyStr {
			case "ctrl+u":
				// Clear the input line and suggestions to recover from stuck states
				m.PathInput.SetValue("")
				m.PathInput.SetCursor(0)
				m.ShowSuggestions = false
				m.FilteredSuggestions = nil
				m.clearError()
				return m, nil
			case "up", "down":
				// Up/Down arrows: cycle only through CEL functions (not keys)
				// Check if suggestions are visible (not just when input ends with '.')
				if m.ShowSuggestions && len(m.FilteredSuggestions) > 0 {
					// Filter to only CEL functions
					celFuncs := []string{}
					celIndices := []int{} // Track original indices
					for i, s := range m.FilteredSuggestions {
						if strings.Contains(s, "(") || strings.Contains(s, " - ") {
							celFuncs = append(celFuncs, s)
							celIndices = append(celIndices, i)
						}
					}

					// If there are CEL functions, cycle through them
					if len(celFuncs) > 0 {
						// Find current selection in CEL function list
						currentFuncIdx := -1
						for i, origIdx := range celIndices {
							if origIdx == m.SelectedSuggestion {
								currentFuncIdx = i
								break
							}
						}

						increment := 1
						if keyStr == "up" {
							increment = -1
						}

						if currentFuncIdx >= 0 {
							// Cycle to next/previous CEL function
							currentFuncIdx = (currentFuncIdx + increment + len(celFuncs)) % len(celFuncs)
						} else {
							// Not currently on a function, start at first (or last for up)
							if keyStr == "up" {
								currentFuncIdx = len(celFuncs) - 1
							} else {
								currentFuncIdx = 0
							}
						}

						// Update SelectedSuggestion to point to the selected CEL function
						m.SelectedSuggestion = celIndices[currentFuncIdx]
						return m, nil
					}
				}
				// No dropdown: arrows operate on the table above
				m.Tbl, cmd = m.Tbl.Update(msg)
				// Clear any error/success messages when navigating (so status bar shows index)
				m.clearErrorUnlessSticky()
				// Use SyncTableState() to ensure columns and cursor are set correctly after table update
				m.SyncTableState()
				return m, cmd
			case "tab":
				// Tab: cycle suggestions; if a key is selected, complete it into input
				// If input ends with '[', immediately insert the first index ([0]) unconditionally
				if strings.HasSuffix(m.PathInput.Value(), "[") {
					base := strings.TrimSuffix(m.PathInput.Value(), "[")
					newValue := base + "[0]"
					m.PathInput.SetValue(newValue)
					m.PathInput.SetCursor(len(newValue))
					m.LastTabPosition = len(newValue)
					m.ShowSuggestions = false
					return m, nil
				}

				// If input ends with a closed index like '...][n]', cycle to next sibling index in expr mode too
				if strings.HasSuffix(m.PathInput.Value(), "]") {
					curVal := m.PathInput.Value()
					lastOpen := strings.LastIndex(curVal, "[")
					lastClose := strings.LastIndex(curVal, "]")
					if lastOpen >= 0 && lastClose > lastOpen {
						parent := curVal[:lastOpen]
						idxStr := curVal[lastOpen+1 : lastClose]
						if i, err := strconv.Atoi(idxStr); err == nil {
							// Resolve parent; supports both dotted paths and CEL like '_.items'
							if node, err := navigator.Navigate(m.Root, parent); err == nil {
								if arr, ok := node.([]interface{}); ok && len(arr) > 0 {
									next := (i + 1) % len(arr)
									newValue := parent + "[" + strconv.Itoa(next) + "]"
									m.PathInput.SetValue(newValue)
									m.PathInput.SetCursor(len(newValue))
									m.LastTabPosition = len(newValue)
									m.ShowSuggestions = false
									return m, nil
								}
							}
						}
					}
				}

				// If suggestions aren't visible (e.g., after selecting an array index), refresh them now.
				// This enables Tab cycling across sibling indices when input ends with "]".
				if !m.ShowSuggestions {
					m.filterSuggestions(true)
					m.syncSuggestions() // Sync suggestions component after filtering
				}

				if m.ShowSuggestions && len(m.FilteredSuggestions) > 0 {
					currentValue := m.PathInput.Value()
					lastDotForMatch := strings.LastIndex(currentValue, ".")
					tokenForMatch := ""
					tokenIsFunction := false
					if lastDotForMatch >= 0 && !strings.HasSuffix(currentValue, ".") {
						rawToken := strings.TrimSpace(currentValue[lastDotForMatch+1:])
						if rawToken != "" && !strings.ContainsAny(rawToken, "[") {
							if strings.Contains(rawToken, "(") || strings.HasSuffix(rawToken, ")") {
								tokenIsFunction = true
							}
							if idx := strings.Index(rawToken, "("); idx >= 0 {
								rawToken = rawToken[:idx]
							}
							rawToken = strings.TrimSuffix(rawToken, ")")
							rawToken = strings.TrimSpace(rawToken)
							if rawToken != "" {
								tokenForMatch = strings.ToLower(rawToken)
							}
						}
					}
					cycling := m.LastTabPosition == len(currentValue)
					if cycling {
						// When cycling, preserve the original token (even if empty for trailing dot case)
						tokenForMatch = m.LastTabToken
					} else {
						m.LastTabToken = tokenForMatch
						m.LastTabTokenIsFunc = tokenIsFunction
					}
					// Special case: when typing an array index (input ends with '['),
					// always select the first index and insert it immediately.
					if strings.HasSuffix(m.PathInput.Value(), "[") {
						m.SelectedSuggestion = 0
						selected := m.FilteredSuggestions[m.SelectedSuggestion]
						// Ensure it's an index suggestion
						if strings.HasPrefix(selected, "[") {
							base := strings.TrimSuffix(m.PathInput.Value(), "[")
							name := strings.Trim(selected, `[]"`)
							newValue := base + "[" + name + "]"
							m.PathInput.SetValue(newValue)
							m.PathInput.SetCursor(len(newValue))
							m.LastTabPosition = len(newValue)
							return m, nil
						}
					}

					currentValue = m.PathInput.Value()
					currentCursor := len(currentValue)

					candidates := m.buildTabCandidates(tokenForMatch)
					if len(candidates) == 0 {
						return m, nil
					}

					// Find current selection among candidates
					currentIdx := -1
					for i, c := range candidates {
						if c.idx == m.SelectedSuggestion {
							currentIdx = i
							break
						}
					}

					if m.LastTabPosition == currentCursor && currentIdx >= 0 {
						currentIdx = (currentIdx + 1) % len(candidates)
					} else {
						currentIdx = 0
					}
					m.LastTabPosition = currentCursor
					selected := candidates[currentIdx]
					m.SelectedSuggestion = selected.idx

					// Track whether the user had a trailing dot to avoid dropping the parent segment
					hadTrailingDot := strings.HasSuffix(currentValue, ".")

					// Prepare name and detect types
					name := selected.name
					if idx := strings.Index(name, " - "); idx >= 0 {
						name = name[:idx]
					}

					// Function completion: append base function name with ()
					if selected.isFunc {
						baseFuncName := extractFunctionName(name)
						if baseFuncName == "" {
							return m, nil
						}
						lastDot := strings.LastIndex(currentValue, ".")
						if lastDot < 0 {
							return m, nil
						}
						newValue := currentValue[:lastDot+1] + baseFuncName + "()"
						m.PathInput.SetValue(newValue)
						m.PathInput.SetCursor(len(newValue))
						m.LastTabPosition = len(newValue)
						m.ShowSuggestions = true
						return m, nil
					}

					// Key/index completion follows prior behavior
					baseValue := strings.TrimSuffix(currentValue, ".")
					baseValue = strings.TrimSuffix(baseValue, "[")
					if strings.HasPrefix(name, "_") ||
						(baseValue != "" && (name == baseValue ||
							strings.HasPrefix(name, baseValue+".") ||
							strings.HasPrefix(name, baseValue+"["))) {
						m.PathInput.SetValue(name)
						m.PathInput.SetCursor(len(name))
						m.LastTabPosition = len(name)
						m.ShowSuggestions = true
						return m, nil
					}

					isFullPath := !strings.HasPrefix(name, "[") && strings.Contains(name, "[")
					if isFullPath {
						m.PathInput.SetValue(name)
						m.PathInput.SetCursor(len(name))
						m.LastTabPosition = len(name)
						m.ShowSuggestions = true
						return m, nil
					}

					isArrayIndex := strings.HasPrefix(name, "[")
					if isArrayIndex {
						name = strings.Trim(name, `[]"`)
						name = "[" + name + "]"
					}

					pathWithoutDot := strings.TrimSuffix(currentValue, ".")
					pathWithoutBracket := strings.TrimSuffix(pathWithoutDot, "[")

					if isArrayIndex && strings.Contains(pathWithoutBracket, "[") {
						lastBracketIdx := strings.LastIndex(pathWithoutBracket, "[")
						pathWithoutBracket = pathWithoutBracket[:lastBracketIdx]
					}

					var newValue string
					if isArrayIndex {
						base := pathWithoutBracket
						if idx := strings.LastIndex(base, "."); idx >= 0 {
							base = base[:idx]
						}
						newValue = base + name
					} else {
						lastDot := strings.LastIndex(pathWithoutBracket, ".")
						insertName := name
						if lastDot >= 0 { //nolint:gocritic
							lastToken := pathWithoutBracket[lastDot+1:]
							insertName = adjustSuggestionNamespace(lastToken, name)
							if hadTrailingDot {
								newValue = pathWithoutBracket + "." + insertName
							} else {
								newValue = pathWithoutBracket[:lastDot+1] + insertName
							}
						} else if hadTrailingDot {
							newValue = pathWithoutBracket + "." + insertName
						} else {
							newValue = insertName
						}
					}

					m.PathInput.SetValue(newValue)
					m.PathInput.SetCursor(len(newValue))
					m.LastTabPosition = len(newValue)
					return m, nil
				}
				return m, nil
			case "right":
				// If a global-style function is selected, do not auto-complete; just move cursor.
				if m.InputFocused && m.ShowSuggestions && len(m.FilteredSuggestions) > 0 && m.SelectedSuggestion < len(m.FilteredSuggestions) {
					if usageStyleLabel(m.FilteredSuggestions[m.SelectedSuggestion]) == "[global]" {
						m.PathInput.CursorEnd()
						return m, nil
					}
				}
				// Right arrow: only operate on the input when in expr mode (no navigation).

				// If cursor is not at the end, user is navigating - just move cursor
				if cursorPos < len(currentValue) {
					m.PathInput, cmd = m.PathInput.Update(msg)
					return m, cmd
				}
				// Cursor already at end; consume the key so we don't fall through to table navigation.
				m.PathInput.CursorEnd()
				return m, nil
			case "left":
				// In expr mode, when cursor is at the start, allow navigating to parent; otherwise move cursor.
				if m.InputFocused {
					if m.PathInput.Position() <= 0 {
						return m.navigateExprBackward()
					}
					m.PathInput, cmd = m.PathInput.Update(msg)
					return m, cmd
				}

				// Cursor is at the end - proceed with completion
				// Always ensure suggestions are filtered and up-to-date before completion
				// Use true to force filtering regardless of AllowIntellisense setting
				m.filterSuggestions(true)
				m.syncSuggestions() // Sync suggestions component after filtering

				// Check if we have suggestions and a selected function
				// Allow completion even if ShowSuggestions is false, as long as we have filtered suggestions
				// Ensure SelectedSuggestion is within bounds
				if m.SelectedSuggestion >= len(m.FilteredSuggestions) {
					m.SelectedSuggestion = 0
				}
				if len(m.FilteredSuggestions) > 0 && m.SelectedSuggestion < len(m.FilteredSuggestions) {
					selected := m.FilteredSuggestions[m.SelectedSuggestion]

					// Check if selected is a CEL function
					isFunction := strings.Contains(selected, "(") || strings.Contains(selected, " - ")
					if !isFunction {
						// Not a function, don't complete - just move cursor
						m.PathInput.CursorEnd()
						return m, nil
					}

					// Extract just the function name (before description)
					funcName := selected
					if idx := strings.Index(selected, " - "); idx >= 0 {
						funcName = selected[:idx]
					}

					// Case 1: Input ends with "." - complete as method-style function (e.g., "_.all()")
					// But only if the function is actually method-style, not global-style
					if strings.HasSuffix(currentValue, ".") {
						// Check if this is a method-style function (can be used as _.function())
						// Global-style functions (like greatest()) are shown in suggestions but NOT auto-completed
						// They must be used as greatest(_), not _.greatest(), so we skip completion
						if usageStyleLabel(selected) == "[global]" {
							// Don't auto-complete global-style functions as method-style - it would be incorrect
							// Global-style functions are still shown in suggestions, just not auto-completed
							// Help text will still be shown via detectFunctionHelp()
							m.PathInput.CursorEnd()
							return m, nil
						}
						// Extract base function name (remove namespace and parentheses if present)
						baseFuncName := funcName
						if dotIdx := strings.LastIndex(baseFuncName, "."); dotIdx >= 0 {
							baseFuncName = baseFuncName[dotIdx+1:]
						}
						// Remove parentheses if present
						if strings.HasSuffix(baseFuncName, "()") {
							baseFuncName = baseFuncName[:len(baseFuncName)-2]
						} else if strings.HasSuffix(baseFuncName, "(") {
							baseFuncName = baseFuncName[:len(baseFuncName)-1]
						}
						// Complete as method-style: "_.all()"
						newValue := currentValue + baseFuncName + "()"
						m.PathInput.SetValue(newValue)
						m.PathInput.SetCursor(len(newValue))
						m.ShowSuggestions = true
						return m, nil
					}

					// Case 2: Skip auto-completion for global-style functions when NOT at trailing dot
					// EXCEPTION: String methods (matches, contains, startsWith, endsWith) can be used
					// as methods even if they also have global-style overloads
					style := usageStyleLabel(selected)
					if style == "[global]" {
						// Check if this is a known string method that can be used as a method
						baseFuncName := funcName
						if dotIdx := strings.LastIndex(baseFuncName, "."); dotIdx >= 0 {
							baseFuncName = baseFuncName[dotIdx+1:]
						}
						baseFuncName = strings.TrimSuffix(strings.TrimSuffix(baseFuncName, "()"), "(")
						stringMethods := map[string]bool{
							"matches":    true,
							"contains":   true,
							"startsWith": true,
							"endsWith":   true,
						}
						if !stringMethods[strings.ToLower(baseFuncName)] {
							// Not a string method, don't auto-complete - just move cursor
							m.PathInput.CursorEnd()
							return m, nil
						}
						// Allow string methods to be completed as methods even if shown as global-style
					}

					// Case 2: Input has a partial function name (e.g., "_.map") - complete it
					// Find the last token after the last dot
					lastDotForPartial := strings.LastIndex(currentValue, ".")
					if lastDotForPartial >= 0 {
						lastToken := currentValue[lastDotForPartial+1:]
						// Don't attempt completion if cursor is before the last dot
						// Check if input ends with the token after the last dot
						if !strings.HasSuffix(currentValue, lastToken) {
							// Cursor appears to be before the last dot - user is editing, don't complete
							m.PathInput, cmd = m.PathInput.Update(msg)
							return m, cmd
						}

						// Extract base function name from suggestion (remove namespace and parentheses if present)
						suggFuncName := funcName
						if dotIdx := strings.LastIndex(suggFuncName, "."); dotIdx >= 0 {
							suggFuncName = suggFuncName[dotIdx+1:]
						}
						// Remove parentheses if present to get base name
						baseFuncName := suggFuncName
						if strings.HasSuffix(baseFuncName, "()") {
							baseFuncName = baseFuncName[:len(baseFuncName)-2]
						} else if strings.HasSuffix(baseFuncName, "(") {
							baseFuncName = baseFuncName[:len(baseFuncName)-1]
						}
						// Check if lastToken is a prefix of the base function name
						if strings.HasPrefix(strings.ToLower(baseFuncName), strings.ToLower(lastToken)) {
							// Complete the function name with parentheses
							newValue := currentValue[:lastDotForPartial+1] + baseFuncName + "()"
							m.PathInput.SetValue(newValue)
							m.PathInput.SetCursor(len(newValue))
							m.ShowSuggestions = true
							return m, nil
						}
					}
					// If the selected suggestion doesn't match, try to find a matching function
					// This handles cases where the user types a partial function name but it's not selected
					lastDotForSearch := strings.LastIndex(currentValue, ".")
					if lastDotForSearch >= 0 {
						lastToken := currentValue[lastDotForSearch+1:]
						// Don't attempt completion if cursor is before the last dot
						// Check if input ends with the token after the last dot
						if !strings.HasSuffix(currentValue, lastToken) {
							// Cursor appears to be before the last dot - user is editing, don't complete
							m.PathInput, cmd = m.PathInput.Update(msg)
							return m, cmd
						}

						lowerToken := strings.ToLower(lastToken)
						// Search through all filtered suggestions for a matching function
						// Prioritize exact prefix matches and prefer shorter function names (more specific)
						bestMatch := -1
						bestMatchName := ""
						for i, sugg := range m.FilteredSuggestions {
							if !strings.Contains(sugg, "(") && !strings.Contains(sugg, " - ") {
								continue // Skip non-functions
							}
							// Skip global-style functions
							if usageStyleLabel(sugg) == "[global]" {
								continue
							}
							// Extract function name from suggestion
							suggFuncName := sugg
							if idx := strings.Index(sugg, " - "); idx >= 0 {
								suggFuncName = sugg[:idx]
							}
							if dotIdx := strings.LastIndex(suggFuncName, "."); dotIdx >= 0 {
								suggFuncName = suggFuncName[dotIdx+1:]
							}
							// Remove parentheses
							if strings.HasSuffix(suggFuncName, "()") {
								suggFuncName = suggFuncName[:len(suggFuncName)-2]
							} else if strings.HasSuffix(suggFuncName, "(") {
								suggFuncName = suggFuncName[:len(suggFuncName)-1]
							}
							lowerSuggName := strings.ToLower(suggFuncName)
							// Check if this function matches the partial token
							if strings.HasPrefix(lowerSuggName, lowerToken) {
								// Prefer exact matches or shorter matches (more specific)
								if bestMatch == -1 || len(suggFuncName) < len(bestMatchName) {
									bestMatch = i
									bestMatchName = suggFuncName
								}
							}
						}
						if bestMatch >= 0 {
							// Found a matching function - complete it
							newValue := currentValue[:lastDotForSearch+1] + bestMatchName + "()"
							m.PathInput.SetValue(newValue)
							m.PathInput.SetCursor(len(newValue))
							m.SelectedSuggestion = bestMatch
							m.ShowSuggestions = true
							return m, nil
						}
					}
				}

				// If we reach here, no completion was possible - just move cursor normally
				m.PathInput, cmd = m.PathInput.Update(msg)
				return m, cmd
			case "shift+tab":
				// Shift+Tab: cycle backwards through keys only (not CEL functions)
				if !m.ShowSuggestions {
					m.filterSuggestions(true)
					m.syncSuggestions() // Sync suggestions component after filtering
				}
				// Reuse candidate builder from Tab (keys first, dedup functions)

				if len(m.FilteredSuggestions) == 0 {
					return m, nil
				}

				currentValue := m.PathInput.Value()
				lastDotForMatch := strings.LastIndex(currentValue, ".")
				tokenForMatch := ""
				if lastDotForMatch >= 0 && !strings.HasSuffix(currentValue, ".") {
					rawToken := strings.TrimSpace(currentValue[lastDotForMatch+1:])
					if rawToken != "" && !strings.ContainsAny(rawToken, "[") {
						if idx := strings.Index(rawToken, "("); idx >= 0 {
							rawToken = rawToken[:idx]
						}
						rawToken = strings.TrimSuffix(rawToken, ")")
						rawToken = strings.TrimSpace(rawToken)
						if rawToken != "" {
							tokenForMatch = strings.ToLower(rawToken)
						}
					}
				}
				cycling := m.LastTabPosition == len(currentValue)
				if cycling {
					// When cycling, preserve the original token (even if empty for trailing dot case)
					tokenForMatch = m.LastTabToken
				} else {
					m.LastTabToken = tokenForMatch
					m.LastTabTokenIsFunc = false
				}

				candidates := m.buildTabCandidates(tokenForMatch)
				if len(candidates) == 0 {
					return m, nil
				}

				currentCursor := len(currentValue)
				currentIdx := -1
				for i, c := range candidates {
					if c.idx == m.SelectedSuggestion {
						currentIdx = i
						break
					}
				}
				if m.LastTabPosition == currentCursor && currentIdx >= 0 {
					currentIdx = (currentIdx - 1 + len(candidates)) % len(candidates)
				} else {
					currentIdx = len(candidates) - 1
				}
				m.LastTabPosition = currentCursor

				selected := candidates[currentIdx]
				m.SelectedSuggestion = selected.idx

				hadTrailingDot := strings.HasSuffix(currentValue, ".")
				name := selected.name
				if idx := strings.Index(name, " - "); idx >= 0 {
					name = name[:idx]
				}

				if selected.isFunc {
					baseFuncName := extractFunctionName(name)
					if baseFuncName == "" {
						return m, nil
					}
					lastDot := strings.LastIndex(currentValue, ".")
					if lastDot < 0 {
						return m, nil
					}
					newValue := currentValue[:lastDot+1] + baseFuncName + "()"
					m.PathInput.SetValue(newValue)
					m.PathInput.SetCursor(len(newValue))
					m.LastTabPosition = len(newValue)
					m.ShowSuggestions = true
					return m, nil
				}

				baseValue := strings.TrimSuffix(currentValue, ".")
				baseValue = strings.TrimSuffix(baseValue, "[")
				if strings.HasPrefix(name, "_") ||
					(baseValue != "" && (name == baseValue ||
						strings.HasPrefix(name, baseValue+".") ||
						strings.HasPrefix(name, baseValue+"["))) {
					m.PathInput.SetValue(name)
					m.PathInput.SetCursor(len(name))
					m.LastTabPosition = len(name)
					m.ShowSuggestions = true
					return m, nil
				}

				isArrayIndex := strings.HasPrefix(name, "[")
				if isArrayIndex {
					name = strings.Trim(name, `[]"`)
					name = "[" + name + "]"
				}

				pathWithoutDot := strings.TrimSuffix(currentValue, ".")
				pathWithoutBracket := strings.TrimSuffix(pathWithoutDot, "[")
				if isArrayIndex && strings.Contains(pathWithoutBracket, "[") {
					lastBracketIdx := strings.LastIndex(pathWithoutBracket, "[")
					pathWithoutBracket = pathWithoutBracket[:lastBracketIdx]
				}

				var newValue string
				if isArrayIndex {
					base := pathWithoutBracket
					if idx := strings.LastIndex(base, "."); idx >= 0 {
						base = base[:idx]
					}
					newValue = base + name
				} else {
					lastDot := strings.LastIndex(pathWithoutBracket, ".")
					switch {
					case lastDot >= 0:
						newValue = pathWithoutBracket[:lastDot+1] + name
					case hadTrailingDot:
						newValue = pathWithoutBracket + "." + name
					default:
						newValue = name
					}
				}

				m.PathInput.SetValue(newValue)
				m.PathInput.SetCursor(len(newValue))
				m.LastTabPosition = len(newValue)
				m.ShowSuggestions = true
				return m, nil
			case "ctrl+k":
				// Ctrl+K: force show all suggestions
				if len(m.Suggestions) > 0 {
					m.FilteredSuggestions = m.Suggestions
					m.ShowSuggestions = true
					m.SelectedSuggestion = 0
				}
				return m, nil
			case "f5":
				// Copy current expression with quoting
				p := strings.TrimSpace(m.PathInput.Value())
				if p == "" {
					p = "_"
				}
				cliSafe := makePathCLISafe(p)
				if err := copyToClipboard(cliSafe); err != nil {
					m.ErrMsg = fmt.Sprintf("Clipboard unavailable: %v", err)
				} else {
					m.ErrMsg = fmt.Sprintf("Copied: %s", cliSafe)
				}
				return m, nil
			case "f10":
				// Validate eval then print and quit
				evalExpr := strings.TrimSpace(m.PathInput.Value())
				if evalExpr == "" {
					evalExpr = "_"
				}
				// Use expression exactly as typed - no modifications
				// Use the configured expression provider to respect custom CEL environments
				if _, err := EvaluateExpression(evalExpr, m.Root); err != nil {
					m.ErrMsg = fmt.Sprintf("explore expression error: %v", err)
					return m, nil
				}
				return m, m.printCLIOutput(evalExpr)
			case "ctrl+c":
				return m, tea.Quit
			case "enter":
				// Enter in input mode: always run the current expression as-is (no auto-completion)
				currentValue := m.PathInput.Value()

				// If the typed input resolves to a node, navigate to it directly
				pathValue := strings.TrimSpace(currentValue)
				if pathValue == "_" {
					// Root navigation: keep expr text as-is but reset path state
					newModel := InitialModel(m.Root)
					newModel.Root = m.Root
					newModel.DebugMode = m.DebugMode
					newModel.NoColor = m.NoColor
					newModel.WinWidth = m.WinWidth
					newModel.WinHeight = m.WinHeight
					newModel.Path = ""
					newModel.PathKeys = []string{}
					newModel.ApplyColorScheme()
					newModel.applyLayout(true)
					newModel.InputFocused = true
					newModel.setExprResult(pathValue, newModel.Node)
					newModel.PathInput.SetValue(pathValue)
					newModel.PathInput.SetCursor(len(pathValue))
					newModel.PathInput.Focus()
					newModel.Tbl.Blur()
					return &newModel, nil
				}
				// Try to evaluate the expression as-is, even if it ends with "." or "["
				// This allows showing errors for invalid expressions like "_.", "_.items[", etc.
				if pathValue != "" {
					// First, try to navigate/evaluate the expression
					node, err := navigator.Navigate(m.Root, pathValue)
					if err == nil {
						// For free-form CEL, avoid NavigateTo to preserve input exactly
						if (strings.Contains(pathValue, "(") && strings.Contains(pathValue, ")")) || IsExpression(pathValue) {
							newModel := InitialModel(node)
							newModel.Root = m.Root
							newModel.DebugMode = m.DebugMode
							newModel.NoColor = m.NoColor
							newModel.WinWidth = m.WinWidth
							newModel.WinHeight = m.WinHeight
							newModel.Path = pathValue
							newModel.PathKeys = parsePathKeys(pathValue)
							newModel.ApplyColorScheme()
							newModel.applyLayout(true)
							newModel.InputFocused = true
							newModel.setExprResult(pathValue, newModel.Node)
							newModel.PathInput.SetValue(pathValue)
							newModel.PathInput.SetCursor(len(pathValue))
							newModel.PathInput.Focus()
							newModel.Tbl.Blur()
							// Do not auto-copy on scalar results; user can press F5 to copy
							return &newModel, nil
						}
						// Create model without altering typed input
						newModel := InitialModel(node)
						newModel.Root = m.Root
						newModel.DebugMode = m.DebugMode
						newModel.NoColor = m.NoColor
						newModel.WinWidth = m.WinWidth
						newModel.WinHeight = m.WinHeight
						newModel.Path = normalizePathForModel(pathValue)
						newModel.PathKeys = parsePathKeys(newModel.Path)
						newModel.ApplyColorScheme()
						newModel.applyLayout(true)
						newModel.InputFocused = true
						newModel.setExprResult(pathValue, newModel.Node)
						newModel.PathInput.SetValue(pathValue)
						newModel.PathInput.SetCursor(len(pathValue))
						newModel.PathInput.Focus()
						newModel.Tbl.Blur()
						return &newModel, nil
					}
					// Navigation failed - show error message
					// This handles cases like "_.", "_.items[", etc. that are invalid
					m.setStickyError(fmt.Sprintf("Path error: %v", err))
					// Keep the input value as-is so user can see what they typed
					m.PathInput.SetValue(pathValue)
					return m, nil
				}
				// Check if the current value exactly matches a root key - if so, navigate directly
				if !strings.Contains(currentValue, ".") && !strings.Contains(currentValue, "[") && currentValue != "" {
					for _, k := range getKeysFromNode(m.Root) {
						if k == currentValue {
							newNode, err := navigator.Navigate(m.Root, currentValue)
							if err == nil {
								// Build model and keep input exactly as typed
								nm := InitialModel(newNode)
								nm.Root = m.Root
								nm.DebugMode = m.DebugMode
								nm.NoColor = m.NoColor
								nm.WinWidth = m.WinWidth
								nm.WinHeight = m.WinHeight
								nm.Path = normalizePathForModel(currentValue)
								nm.PathKeys = parsePathKeys(nm.Path)
								nm.ApplyColorScheme()
								nm.applyLayout(true)
								nm.InputFocused = true
								nm.setExprResult(currentValue, nm.Node)
								nm.PathInput.SetValue(currentValue)
								nm.PathInput.SetCursor(len(currentValue))
								nm.InputFocused = true
								// Focus will be handled by SyncTableState() when InputFocused is true
								nm.SyncTableState()
								return &nm, nil
							}
							break
						}
					}
				}
				// Bare token fallback: navigate to first root key matching prefix
				if !strings.Contains(currentValue, ".") && !strings.Contains(currentValue, "[") && currentValue != "" {
					token := strings.ToLower(currentValue)
					for _, k := range getKeysFromNode(m.Root) {
						cand := k
						if strings.HasPrefix(cand, "[") && strings.HasSuffix(cand, "]") {
							cand = cand[1 : len(cand)-1]
						}
						if strings.HasPrefix(strings.ToLower(cand), token) {
							navigatePath := cand
							newNode, err := navigator.Navigate(m.Root, navigatePath)
							if err == nil {
								// Build model and keep input exactly as typed
								nm := InitialModel(newNode)
								nm.Root = m.Root
								nm.DebugMode = m.DebugMode
								nm.NoColor = m.NoColor
								nm.WinWidth = m.WinWidth
								nm.WinHeight = m.WinHeight
								nm.Path = normalizePathForModel(navigatePath)
								nm.PathKeys = parsePathKeys(nm.Path)
								nm.ApplyColorScheme()
								nm.applyLayout(true)
								nm.InputFocused = true
								nm.setExprResult(currentValue, nm.Node)
								nm.PathInput.SetValue(currentValue)
								nm.PathInput.SetCursor(len(currentValue))
								nm.InputFocused = true
								// Focus will be handled by SyncTableState() when InputFocused is true
								nm.SyncTableState()
								return &nm, nil
							}
							break
						}
					}
				}
				// Otherwise, navigate to the typed path (no auto-completion)
				m.ShowSuggestions = false
				pathValue = strings.TrimSpace(currentValue)

				// Handle special case: "_" or "_." means navigate to root
				if pathValue == "_" || pathValue == "_." || pathValue == "" {
					// Go to root but preserve literal expr input
					newModel := InitialModel(m.Root)
					newModel.Root = m.Root
					newModel.DebugMode = m.DebugMode
					newModel.NoColor = m.NoColor
					newModel.WinWidth = m.WinWidth
					newModel.WinHeight = m.WinHeight
					newModel.Path = ""
					newModel.PathKeys = []string{}
					newModel.ApplyColorScheme()
					newModel.applyLayout(true)
					newModel.InputFocused = true
					newModel.setExprResult(pathValue, newModel.Node)
					newModel.PathInput.SetValue(pathValue)
					newModel.PathInput.Focus()
					newModel.Tbl.Blur()
					return &newModel, nil
				}

				// Use unified Navigate interface that handles both dotted paths and CEL
				newNode, err := navigator.Navigate(m.Root, pathValue)
				if err != nil {
					m.setStickyError(fmt.Sprintf("Path error: %v", err))
					m.PathInput.SetValue(pathValue)
					return m, nil
				}

				newModel := InitialModel(newNode)
				newModel.Root = m.Root
				newModel.DebugMode = m.DebugMode
				newModel.NoColor = m.NoColor
				newModel.WinWidth = m.WinWidth
				newModel.WinHeight = m.WinHeight
				// Store path with _ prefix preserved (normalizePathForModel will ensure it)
				newModel.Path = normalizePathForModel(pathValue)
				if pathValue != "" {
					newModel.PathKeys = parsePathKeys(pathValue)
				}
				newModel.ApplyColorScheme()
				newModel.applyLayout(true)
				// Keep input focused so user can continue typing/exploring
				newModel.InputFocused = true
				newModel.setExprResult(pathValue, newModel.Node)
				newModel.PathInput.SetValue(pathValue)
				newModel.PathInput.Focus()
				newModel.Tbl.Blur()
				return &newModel, nil
			case "esc":
				if m.HelpVisible {
					m.HelpVisible = false
					m.applyLayout(true) // Recalculate layout when help closes
				}
				m.ShowSuggestions = false
				m.setShowInfoPopup(false)
				m.InputFocused = false
				// Focus will be handled by SyncTableState() when InputFocused changes
				m.SyncTableState()
				return m, nil
			default:
				m.PathInput, cmd = m.PathInput.Update(msg)
				// Let user type freely without forcing prefix
				// The navigator will handle wrapping in "_." for simple paths
				m.filterSuggestions(m.AllowIntellisense)
				m.syncSuggestions() // Sync suggestions component after filtering
				return m, cmd
			}
		}

		switch keyStr {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.HelpVisible {
				m.HelpVisible = false
				m.applyLayout(true) // Recalculate layout when help closes
				return m, nil
			}
			if m.ShowInfoPopup && !m.InfoPopupPermanent {
				m.setShowInfoPopup(false)
				return m, nil
			}
			// Exit advanced search mode and restore previous state
			if m.AdvancedSearchActive {
				m.clearSearchState()

				// Restore node/path from saved state; resolve from root to ensure a full set of rows.
				restorePath := m.PreviousPath
				restoreNode := m.Root
				if restorePath != "" {
					if n, err := navigator.Resolve(m.Root, restorePath); err == nil {
						restoreNode = n
					}
				}

				m.Node = restoreNode
				m.Path = restorePath
				m.PathInput.SetValue(formatPathForDisplay(restorePath))

				// Regenerate rows from the restored node to avoid stale/partial caches.
				keyW := m.KeyColWidth
				if keyW <= 0 {
					keyW = 30
				}
				valueW := m.ValueColWidth
				if valueW <= 0 {
					valueW = 60
				}
				stringRows := navigator.NodeToRows(m.Node)
				m.AllRows = styleRowsWithWidths(stringRows, keyW, valueW)
				m.FilterActive = false
				m.FilterBuffer = ""
				m.applyLayout(true)
				m.SyncTableState(true)
				m.logEvent("cancel-search:esc")
				m.clearErrorUnlessSticky()
				return m, nil
			}
			// Clear type-ahead filter if active
			if m.FilterActive {
				m.FilterActive = false
				m.FilterBuffer = ""
				// Use SyncTableState() to restore table state consistently
				m.SyncTableState()
				return m, nil
			}
			// Exit map filter mode if active
			if m.MapFilterActive {
				m.MapFilterActive = false
				m.MapFilterQuery = ""
				m.MapFilterInput.SetValue("")
				// Regenerate rows from current node
				keyW := m.KeyColWidth
				if keyW <= 0 {
					keyW = 30
				}
				valueW := m.ValueColWidth
				if valueW <= 0 {
					valueW = 60
				}
				stringRows := navigator.NodeToRows(m.Node)
				m.AllRows = styleRowsWithWidths(stringRows, keyW, valueW)
				m.AllRowKeys = extractRowKeys(stringRows)
				m.SyncTableState(true)
				m.logEvent("cancel-map-filter:esc")
				return m, nil
			}
			return m, nil
		case "backspace", "ctrl+h":
			// Handle backspace in map filter mode
			if m.MapFilterActive && len(m.MapFilterQuery) > 0 {
				runes := []rune(m.MapFilterQuery)
				m.MapFilterQuery = string(runes[:len(runes)-1])
				m.MapFilterInput.SetValue(m.MapFilterQuery)
				m.MapFilterInput.SetCursor(len(m.MapFilterQuery))
				m.applyMapFilter()
				return m, nil
			}
			// Handle backspace in advanced search
			if m.AdvancedSearchActive && len(m.AdvancedSearchQuery) > 0 {
				// Remove last character
				runes := []rune(m.AdvancedSearchQuery)
				m.AdvancedSearchQuery = string(runes[:len(runes)-1])
				// Reset committed state since query changed
				m.AdvancedSearchCommitted = false
				// Update input display and apply real-time top-level filter
				m.SearchInput.SetValue(m.AdvancedSearchQuery)
				m.SearchInput.SetCursor(len(m.AdvancedSearchQuery))
				m.applyAdvancedSearch()
				return m, nil
			}
			// Handle backspace in type-ahead filter
			if m.FilterActive && len(m.FilterBuffer) > 0 {
				// Remove last character
				runes := []rune(m.FilterBuffer)
				m.FilterBuffer = string(runes[:len(runes)-1])

				// If buffer is now empty, deactivate search and restore all rows
				if m.FilterBuffer == "" {
					m.FilterActive = false
					// Use SyncTableState() to restore table state consistently
					m.SyncTableState()
				} else {
					// Re-filter with updated buffer
					m.applyTypeAheadFilter()
				}
				return m, nil
			}
			return m, nil
		case "enter", "right":
			// Handle Enter/Right in advanced search mode
			if m.AdvancedSearchActive {
				// Enter commits the search (triggers deep recursive search)
				// Right arrow navigates to selected result
				if keyStr == "enter" && !m.AdvancedSearchCommitted {
					// First Enter: Commit the search and perform deep search
					m.AdvancedSearchCommitted = true
					m.applyAdvancedSearch()
					return m, nil
				}

				// Navigate to result (Enter after committed, or Right arrow anytime)
				if len(m.AdvancedSearchResults) > 0 {
					// Ensure table rows match search results (in case they got out of sync)
					if len(m.Tbl.Rows()) != len(m.AdvancedSearchResults) {
						// Re-sync table rows with search results using SyncTableState()
						// SyncTableState() will use AdvancedSearchResults since AdvancedSearchActive is true
						m.SyncTableState()
					}

					// Ensure cursor is within bounds using SyncTableState()
					// This will fix cursor if it's out of bounds
					m.SyncTableState()
					cur := m.Tbl.Cursor()
					// Double-check cursor is valid (defensive)
					if cur >= len(m.AdvancedSearchResults) {
						cur = 0
					}
					if cur >= 0 && cur < len(m.AdvancedSearchResults) {
						result := m.AdvancedSearchResults[cur]
						// Combine base path with result path (result path is relative to search base)
						var fullPath string
						if m.AdvancedSearchBasePath == "" {
							fullPath = result.FullPath
						} else {
							// Check if result path starts with bracket notation (array index)
							if strings.HasPrefix(result.FullPath, "[") {
								fullPath = m.AdvancedSearchBasePath + result.FullPath
							} else {
								fullPath = m.AdvancedSearchBasePath + "." + result.FullPath
							}
						}
						// Navigate to the combined path
						newNode, err := navigator.Resolve(m.Root, fullPath)
						if err != nil {
							m.ErrMsg = fmt.Sprintf("Error: %v", err)
							m.StatusType = "error"
							return m, nil
						}

						// Use NavigateTo which updates existing model to avoid flicker
						newModel := m.NavigateTo(newNode, navigator.NormalizePath(fullPath))

						// Enter: Exit search context completely
						// Right arrow: Maintain search context (for left arrow to restore)
						if keyStr == "enter" {
							newModel.clearSearchState()
						} else {
							// Right arrow: Save search context for navigation back
							newModel.AdvancedSearchActive = false
							newModel.SearchContextActive = true
							newModel.SearchContextResults = make([]SearchResult, len(m.AdvancedSearchResults))
							copy(newModel.SearchContextResults, m.AdvancedSearchResults)
							newModel.SearchContextQuery = m.AdvancedSearchQuery
							newModel.SearchContextBasePath = m.AdvancedSearchBasePath
							// Keep search query/results available for restoring search view
							newModel.AdvancedSearchQuery = m.AdvancedSearchQuery
							newModel.AdvancedSearchResults = make([]SearchResult, len(m.AdvancedSearchResults))
							copy(newModel.AdvancedSearchResults, m.AdvancedSearchResults)
							newModel.AdvancedSearchBasePath = m.AdvancedSearchBasePath
						}

						newModel.SearchInput.Blur()
						newModel.applyLayout(true)
						return newModel, nil
					}
				}
				// Cursor out of bounds - use SyncTableState() to fix cursor and don't navigate
				if len(m.AdvancedSearchResults) > 0 {
					m.SyncTableState()
				}
				return m, nil
			}
			// Not in active search: Enter should clear any lingering search context
			if keyStr == "enter" && m.SearchContextActive {
				m.SearchContextActive = false
				m.SearchContextResults = nil
				m.SearchContextQuery = ""
				m.SearchContextBasePath = ""
			}

			// Clear map filter mode when navigating into a child
			if m.MapFilterActive {
				m.MapFilterActive = false
				m.MapFilterQuery = ""
				m.MapFilterInput.SetValue("")
				// Don't regenerate rows here - we're about to navigate to a new node
			}

			// Regular navigation (not in active search mode)
			// Only proceed if we're definitely not in active search mode
			if m.AdvancedSearchActive {
				// We're in search mode but somehow fell through - this shouldn't happen
				// But if it does, don't navigate from regular table rows
				return m, nil
			}

			// If we're at a scalar value and filter is active, clear filter on Enter even if table is empty
			if m.FilterActive {
				originalKeys := m.AllRowKeys
				if len(originalKeys) == 1 && originalKeys[0] == "(value)" {
					m.FilterActive = false
					m.FilterBuffer = ""
					// Use SyncTableState() to restore table state consistently
					m.SyncTableState()
					formattedPath := formatPathForDisplay(m.Path)
					m.PathInput.SetValue(formattedPath)
					m.PathInput.SetCursor(len(formattedPath))
					return m, nil
				}
			}

			selectedKey, ok := m.selectedRowKey()
			if !ok {
				return m, nil
			}

			// If user drills into a scalar value, do nothing; use F5/F10 actions instead
			if selectedKey == "(value)" {
				// Clear search state and keep expression stable (avoid appending dots)
				if m.FilterActive {
					m.FilterActive = false
					m.FilterBuffer = ""
					// Use SyncTableState() to restore table state consistently
					m.SyncTableState()
				}
				formattedPath := formatPathForDisplay(m.Path)
				m.PathInput.SetValue(formattedPath)
				m.PathInput.SetCursor(len(formattedPath))
				return m, nil
			}

			navigatePath := buildPathWithKey(m.Path, selectedKey)

			// Store cursor position based on the selected key's position in the full node
			// This ensures correct cursor restoration when navigating back from a filtered view
			m.storeCursorForPathByKey(m.Path, selectedKey)

			newNode, err := navigator.Resolve(m.Root, navigatePath)
			if err != nil {
				m.ErrMsg = fmt.Sprintf("Error: %v", err)
				m.StatusType = "error"
				return m, nil
			}

			// Preserve search context BEFORE calling NavigateTo (since NavigateTo modifies in-place)
			// This ensures the context is preserved through multiple levels of navigation
			searchContextActive := m.SearchContextActive
			searchContextResults := make([]SearchResult, len(m.SearchContextResults))
			copy(searchContextResults, m.SearchContextResults)
			searchContextQuery := m.SearchContextQuery
			searchContextBasePath := m.SearchContextBasePath

			// Use NavigateTo which updates existing model to avoid flicker
			newModel := m.NavigateTo(newNode, normalizePathForModel(navigatePath))
			newModel.PathKeys = parsePathKeys(newModel.Path)
			newModel.LastKey = m.LastKey
			// Ensure search context is preserved (NavigateTo should preserve it, but be explicit)
			if searchContextActive {
				newModel.SearchContextActive = true
				newModel.SearchContextResults = make([]SearchResult, len(searchContextResults))
				copy(newModel.SearchContextResults, searchContextResults)
				newModel.SearchContextQuery = searchContextQuery
				newModel.SearchContextBasePath = searchContextBasePath
			}
			newModel.applyLayout(true)
			return newModel, nil
		default:
			// Handle vim-style keybindings when vim mode is enabled
			if m.KeyMode == KeyModeVim && !m.InputFocused && !m.AdvancedSearchActive && !m.MapFilterActive {
				action := m.handleVimKey(keyStr)
				switch action {
				case VimActionNone, VimActionPendingG:
					// No action or pending key sequence, fall through to normal handling
				case VimActionBack:
					// Simulate left arrow by falling through to normal left handling
					return m.handleVimBackNavigation()
				case VimActionForward, VimActionEnter:
					// Simulate right arrow/enter - handled by existing code path
					return m.handleVimForwardNavigation()
				case VimActionDown, VimActionUp, VimActionSearch, VimActionFilter, VimActionNextMatch, VimActionPrevMatch,
					VimActionTop, VimActionBottom, VimActionHelp, VimActionCopy, VimActionExpr,
					VimActionQuit, VimActionClearSearch:
					return m.executeVimAction(action)
				}
			}

			// Handle emacs-style keybindings when emacs mode is enabled
			if m.KeyMode == KeyModeEmacs && !m.InputFocused && !m.AdvancedSearchActive && !m.MapFilterActive {
				action := m.handleEmacsKey(keyStr)
				switch action {
				case VimActionNone, VimActionPendingG:
					// No action (emacs doesn't use pending keys), fall through to normal handling
				case VimActionBack:
					return m.handleVimBackNavigation()
				case VimActionForward, VimActionEnter:
					return m.handleVimForwardNavigation()
				case VimActionDown, VimActionUp, VimActionSearch, VimActionNextMatch, VimActionPrevMatch,
					VimActionTop, VimActionBottom, VimActionHelp, VimActionCopy, VimActionExpr,
					VimActionQuit, VimActionClearSearch, VimActionFilter:
					return m.executeVimAction(action)
				}
			}

			if keyStr == "up" || keyStr == "down" {
				m.Tbl, cmd = m.Tbl.Update(msg)
				// Clear any error/success messages when navigating (so status bar shows index)
				m.clearErrorUnlessSticky()
				// Use SyncTableState() to ensure columns and cursor are set correctly after table update
				// SyncTableState() will preserve the cursor position if it's valid
				m.SyncTableState()
				// Sync expr window with selected result (works for both normal and advanced search)
				m.syncPathInputWithCursor()
				return m, cmd
			}
			// Handle input in map filter mode
			if m.MapFilterActive {
				// Capture alphanumeric input and some special characters
				if len(keyStr) == 1 && ((keyStr[0] >= 'a' && keyStr[0] <= 'z') || (keyStr[0] >= 'A' && keyStr[0] <= 'Z') || (keyStr[0] >= '0' && keyStr[0] <= '9') || keyStr[0] == '_' || keyStr[0] == '-' || keyStr[0] == ' ' || keyStr[0] == '.') {
					m.MapFilterQuery += keyStr
					m.MapFilterInput.SetValue(m.MapFilterQuery)
					m.MapFilterInput.SetCursor(len(m.MapFilterQuery))
					m.applyMapFilter()
					return m, nil
				}
				return m, nil
			}
			// Handle input in advanced search mode
			if m.AdvancedSearchActive {
				// Capture alphanumeric input and some special characters
				if len(keyStr) == 1 && ((keyStr[0] >= 'a' && keyStr[0] <= 'z') || (keyStr[0] >= 'A' && keyStr[0] <= 'Z') || (keyStr[0] >= '0' && keyStr[0] <= '9') || keyStr[0] == '_' || keyStr[0] == '-' || keyStr[0] == ' ' || keyStr[0] == '.' || keyStr[0] == '/') {
					m.AdvancedSearchQuery += keyStr
					// Reset committed state since query changed
					m.AdvancedSearchCommitted = false
					// Update input display and apply real-time top-level filter
					m.SearchInput.SetValue(m.AdvancedSearchQuery)
					m.SearchInput.SetCursor(len(m.AdvancedSearchQuery))
					m.applyAdvancedSearch()
					return m, nil
				}
				return m, nil
			}
			// Type-ahead filter: capture alphanumeric input (only when function mode is enabled)
			if m.KeyMode == KeyModeFunction && m.AllowFilter && len(keyStr) == 1 && ((keyStr[0] >= 'a' && keyStr[0] <= 'z') || (keyStr[0] >= 'A' && keyStr[0] <= 'Z') || (keyStr[0] >= '0' && keyStr[0] <= '9') || keyStr[0] == '_' || keyStr[0] == '-') {
				m.FilterActive = true
				m.FilterBuffer += keyStr
				m.applyTypeAheadFilter()
				return m, nil
			}
			return m, nil
		}
	}
	var cmds []tea.Cmd
	if m.InputFocused {
		var inputCmd tea.Cmd
		m.PathInput, inputCmd = m.PathInput.Update(msg)
		cmds = append(cmds, inputCmd)
	}
	if m.AdvancedSearchActive {
		var searchCmd tea.Cmd
		m.SearchInput, searchCmd = m.SearchInput.Update(msg)
		cmds = append(cmds, searchCmd)
	}
	if m.MapFilterActive {
		var filterCmd tea.Cmd
		m.MapFilterInput, filterCmd = m.MapFilterInput.Update(msg)
		cmds = append(cmds, filterCmd)
	}
	m.Tbl, cmd = m.Tbl.Update(msg)
	cmds = append(cmds, cmd)
	// Clear any error/success messages when navigating (so status bar shows index)
	m.clearErrorUnlessSticky()
	// Use SyncTableState() to ensure columns and cursor are set correctly after table update
	m.SyncTableState()
	return m, tea.Batch(cmds...)
}

// applyTypeAheadFilter filters table rows based on the filter buffer
func (m *Model) applyTypeAheadFilter() {
	// Update filter state - SyncTableState() will apply the filter
	if m.FilterBuffer == "" {
		m.FilterActive = false
	} else {
		m.FilterActive = true
	}
	// Sync table state (will apply filter if active)
	m.SyncTableState()
}

func (m *Model) handleSearchInput(msg tea.KeyMsg, keyStr string) (bool, tea.Cmd) {
	if !m.AdvancedSearchActive || m.InputFocused {
		return false, nil
	}
	switch keyStr {
	case "up", "down", "left", "right", "enter", "esc", "ctrl+c":
		return false, nil
	case "ctrl+u":
		m.SearchInput.SetValue("")
		m.SearchInput.SetCursor(0)
		m.AdvancedSearchQuery = ""
		m.AdvancedSearchCommitted = false
		m.AdvancedSearchResults = []SearchResult{}
		m.applyAdvancedSearch()
		return true, nil
	}
	prev := m.SearchInput.Value()
	var cmd tea.Cmd
	m.SearchInput, cmd = m.SearchInput.Update(msg)
	if m.SearchInput.Value() != prev {
		m.AdvancedSearchQuery = m.SearchInput.Value()
		// Reset committed state since query changed
		m.AdvancedSearchCommitted = false
		// Apply real-time top-level filter (fast, children of current node only)
		m.applyAdvancedSearch()
	}
	return true, cmd
}

// applyAdvancedSearch performs the advanced search and updates the table with results.
// Search only executes when committed (Enter pressed).
func (m *Model) applyAdvancedSearch() {
	if m.AdvancedSearchActive {
		if m.SearchInput.Value() != m.AdvancedSearchQuery {
			m.SearchInput.SetValue(m.AdvancedSearchQuery)
			m.SearchInput.SetCursor(len(m.AdvancedSearchQuery))
		}
	}
	if m.AdvancedSearchQuery == "" {
		m.AdvancedSearchResults = []SearchResult{}
		m.AllRows = []table.Row{}
		m.PathInput.SetValue("_")
		// Sync table state (will set empty rows and cursor)
		m.SyncTableState()
		return
	}

	// Deep search mode (/ key): only execute search when committed (Enter pressed)
	// No real-time filtering while typing - just show the query
	if !m.AdvancedSearchCommitted {
		// User is still typing - don't search yet, just update UI to show query
		// Keep previous results visible (or empty if none)
		m.SyncTableState()
		return
	}

	// Determine the node to search from based on AdvancedSearchBasePath
	// This ensures we search from the correct subkey, not always from root
	var searchNode interface{}
	if m.AdvancedSearchBasePath == "" || m.AdvancedSearchBasePath == "_" {
		// Search from root
		searchNode = m.Root
	} else {
		// Search from the node at AdvancedSearchBasePath
		// This is the path where F3 was pressed, which should be the current subkey
		var err error
		searchNode, err = navigator.Resolve(m.Root, m.AdvancedSearchBasePath)
		if err != nil {
			// Fallback to m.Node if path lookup fails
			searchNode = m.Node
		}
	}

	// Full recursive deep search (user pressed Enter)
	// Apply search result limit to prevent excessive memory/time usage
	limit := m.SearchResultLimit
	if limit <= 0 {
		limit = 500 // default
	}
	results, limited := performAdvancedSearch(searchNode, m.AdvancedSearchQuery, limit)
	m.SearchResultsLimited = limited
	m.AdvancedSearchResults = results

	// Note: AllRows will be set by SyncTableState() based on AdvancedSearchResults
	// Sync table state (will generate rows from search results)
	// Reset cursor to 0 when search results first appear
	if len(m.AdvancedSearchResults) > 0 {
		// Recompute layout so key/value widths fit the search results immediately.
		m.applyLayout(true)
	}
	m.SyncTableState(true)

	// Update path input with first result's path
	if len(m.AdvancedSearchResults) > 0 {
		m.syncPathInputWithCursor()
	} else {
		m.PathInput.SetValue("_")
	}
}

// applyMapFilter filters the current map's keys based on MapFilterQuery.
// This is real-time filtering - updates as user types.
// Only works on maps - does nothing for arrays.
func (m *Model) applyMapFilter() {
	// Only works on maps
	mapNode, isMap := m.Node.(map[string]interface{})
	if !isMap {
		return
	}

	keyW := m.KeyColWidth
	if keyW <= 0 {
		keyW = 30
	}
	valueW := m.ValueColWidth
	if valueW <= 0 {
		valueW = 60
	}

	// If query is empty, show all keys
	if m.MapFilterQuery == "" {
		// Regenerate rows from the current node
		stringRows := navigator.NodeToRows(mapNode)
		m.AllRows = styleRowsWithWidths(stringRows, keyW, valueW)
		m.AllRowKeys = extractRowKeys(stringRows)
		m.SyncTableState(true)
		m.syncPathInputWithCursor()
		return
	}

	// Filter map keys (case-insensitive substring match on keys and values)
	queryLower := strings.ToLower(m.MapFilterQuery)
	var filteredRows []table.Row
	var filteredKeys []string

	// Get sorted keys for deterministic order
	keys := make([]string, 0, len(mapNode))
	for k := range mapNode {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := mapNode[k]
		valueStr := formatter.Stringify(v)

		// Check if key starts with the query (prefix match)
		keyMatches := strings.HasPrefix(strings.ToLower(k), queryLower)

		if keyMatches {
			// Display key with bracket notation if needed
			displayKey := k
			if needsBracketNotation(k) {
				displayKey = `["` + k + `"]`
			}
			// Truncate key and value to fit column widths
			if len(displayKey) > keyW {
				displayKey = displayKey[:keyW-3] + "..."
			}
			if len(valueStr) > valueW {
				valueStr = valueStr[:valueW-3] + "..."
			}
			filteredRows = append(filteredRows, table.Row{displayKey, valueStr})
			filteredKeys = append(filteredKeys, k)
		}
	}

	m.AllRows = filteredRows
	m.AllRowKeys = filteredKeys
	m.SyncTableState(true)
	m.syncPathInputWithCursor()
}

// viewSnapshot holds the pre-rendered pieces of the TUI for pure rendering.
type viewSnapshot struct {
	Table        string
	TablePad     int
	Separator    string
	HelpPopup    string
	Help         string
	InfoPopupTop string
	HelpTop      string
	InfoPopup    string
	Input        string
	Suggestions  string
	Status       string
	Debug        string
	Footer       string
	TotalHeight  int
}

// renderSnapshot renders a snapshot into a string without mutating model state.
func renderSnapshot(snap viewSnapshot) string { //nolint:unused // retained for snapshot test helpers
	var b strings.Builder

	// Optional top-anchored popups (info + help) render before the table
	if snap.InfoPopupTop != "" {
		b.WriteString(snap.InfoPopupTop)
	}
	if snap.HelpTop != "" {
		b.WriteString(snap.HelpTop)
	}

	// Table with header (always visible at top)
	b.WriteString(snap.Table)

	// Add spacer lines to keep the table/header pinned to the top while reserving the
	// layout height needed to keep the footer at the bottom.
	tablePad := snap.TablePad
	if tablePad > 0 {
		b.WriteString(strings.Repeat("\n", tablePad))
	}

	// Add separator line between table and expression bar without creating an extra blank line
	if !strings.HasSuffix(snap.Table, "\n") {
		b.WriteString("\n")
	}
	b.WriteString(snap.Separator + "\n")

	// Render help above expr window if visible
	b.WriteString(snap.Help)
	// Render info popup if provided
	if snap.InfoPopup != "" {
		b.WriteString(snap.InfoPopup)
	}

	// Path input line with conditional grey background (only when not in expr mode)
	b.WriteString(snap.Input + "\n")

	// Suggestions area: render dropdown if visible
	if snap.Suggestions != "" {
		b.WriteString(snap.Suggestions)
	}

	// Write status bar directly below expr section (no gap)
	b.WriteString(snap.Status)

	// Debug bar - use cached output, only regenerate on actual state changes
	b.WriteString(snap.Debug)

	// Footer immediately after status/debug bar
	// No padding - the table height should be adjusted in applyLayout() to ensure footer is at the bottom
	b.WriteString(snap.Footer)

	return b.String()
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	// strings.Count counts separators, so add 1 for the last line when non-empty
	lines := strings.Count(s, "\n") + 1
	// If the string ends with a newline, avoid over-counting
	if strings.HasSuffix(s, "\n") {
		lines--
	}
	return lines
}

// isScalarNode reports whether the current node is a scalar (non-map/slice) and returns a simple type label.
func isScalarNode(node interface{}) (bool, string) {
	if node == nil {
		return true, "null"
	}
	switch node.(type) {
	case map[string]interface{}, []interface{}:
		return false, ""
	}
	v := reflect.ValueOf(node)
	if v.IsValid() {
		if v.Kind() == reflect.Map || v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
			return false, ""
		}
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				return true, "null"
			}
			return isScalarNode(v.Elem().Interface())
		}
		if v.Kind() == reflect.Interface {
			return isScalarNode(v.Elem().Interface())
		}
	}
	typeLabel := "value"
	switch node.(type) {
	case string:
		typeLabel = "string"
	case bool:
		typeLabel = "bool"
	case int, int8, int16, int32, int64:
		typeLabel = "int"
	case uint, uint8, uint16, uint32, uint64:
		typeLabel = "uint"
	case float32, float64:
		typeLabel = "float"
	case time.Time:
		typeLabel = "time"
	}
	return true, typeLabel
}

func (m Model) renderScalarView() string {
	width := m.KeyColWidth + ColumnSeparatorWidth + m.ValueColWidth
	if width <= 0 {
		width = m.WinWidth
	}
	if width <= 0 {
		width = 80
	}
	isScalar, _ := isScalarNode(m.Node)
	if !isScalar {
		return ""
	}
	th := CurrentTheme()
	valueStyle := lipgloss.NewStyle().Width(width)
	if !m.NoColor {
		valueStyle = valueStyle.Foreground(th.ValueColor)
	}
	// Preserve real line breaks in scalar view so users can read multiline content.
	val := formatter.StringifyPreserveNewlines(m.Node)
	valueLine := valueStyle.MaxWidth(width).Render(val)
	return valueLine
}

func (m Model) renderInfoPopupBox(content, justify string) string {
	th := CurrentTheme()
	boxStyle := lipgloss.NewStyle().Border(borderForTheme(th)).PaddingLeft(1).PaddingRight(1)
	if !m.NoColor {
		boxStyle = boxStyle.BorderForeground(th.SeparatorColor)
	}
	width := m.WinWidth - 2
	if width < 1 {
		width = 1
	}
	boxStyle = boxStyle.Width(width)
	switch strings.ToLower(justify) {
	case "center", "middle":
		boxStyle = boxStyle.Align(lipgloss.Center)
	case "right":
		boxStyle = boxStyle.Align(lipgloss.Right)
	default:
		boxStyle = boxStyle.Align(lipgloss.Left)
	}
	return boxStyle.Render(content)
}

// infoPopupTopHeight returns the rendered line count of the top-anchored popup (including spacing).
func (m Model) infoPopupTopHeight() int {
	if !m.ShowInfoPopup || !m.InfoPopupEnabled || m.InfoPopup == "" || m.InfoPopupAnchor != "top" {
		return 0
	}
	rendered := m.renderInfoPopupBox(strings.TrimRight(m.InfoPopup, "\n"), m.InfoPopupJustify)
	return countLines(rendered)
}

// helpAnchoredTopHeight returns rendered line count of help popup+help content when anchored top.
func (m Model) helpAnchoredTopHeight() int {
	if !m.HelpVisible || strings.ToLower(m.HelpPopupAnchor) != "top" {
		return 0
	}
	height := 0
	if m.HelpPopupText != "" {
		rendered := m.renderInfoPopupBox(strings.TrimRight(m.HelpPopupText, "\n"), m.HelpPopupJustify)
		height += countLines(rendered)
	}
	// Render help content with forced visibility so height is accurate even before syncHelp().
	helpModel := m.Help
	helpModel.Visible = true
	helpModel.NoColor = m.NoColor
	helpModel.AllowEditInput = m.AllowEditInput
	helpModel.KeyMode = m.KeyMode
	helpModel.AboutTitle = m.HelpAboutTitle
	helpModel.AboutLines = m.HelpAboutLines
	helpModel.AboutAlign = m.HelpAboutAlign
	helpModel.HelpNavigationDescriptions = m.HelpNavigationDescriptions
	helpModel.SetWidth(m.WinWidth)
	helpContent := helpModel.View()
	if helpContent != "" {
		height += countLines(helpContent)
	}
	return height
}

// buildViewSnapshot assembles the view parts without mutating state.
// Note: syncSuggestions() should be called before this function to ensure
// the suggestions component's InputFocused state is up-to-date.
func (m Model) buildViewSnapshot() viewSnapshot {
	snap := viewSnapshot{}

	// Prefer a scalar view (type bar + value) when the current node is a scalar.
	if isScalar, _ := isScalarNode(m.Node); isScalar {
		snap.Table = m.renderScalarView()
	} else {
		// Table with header (always visible at top)
		snap.Table = m.Tbl.View()
	}
	// Pad table lines to the full panel width so the status/footer align.
	tableWidth := m.KeyColWidth + ColumnSeparatorWidth + m.ValueColWidth
	if tableWidth <= 0 {
		tableWidth = m.WinWidth
	}
	if tableWidth > 0 && snap.Table != "" {
		lines := strings.Split(strings.TrimRight(snap.Table, "\n"), "\n")
		for i, line := range lines {
			lines[i] = padANSIToWidth(line, tableWidth)
		}
		snap.Table = strings.Join(lines, "\n")
	}
	// Pad the table block so the separator stays aligned at the expected height even
	// when the table renders fewer body rows than the reserved height.
	if m.TableHeight > 0 {
		actualTableLines := countLines(snap.Table)
		desiredTableLines := m.TableHeight + TableHeaderLines
		if desiredTableLines > actualTableLines {
			snap.TablePad = desiredTableLines - actualTableLines
		}
	}
	// When in expression mode, we've already reduced table height to reserve space for suggestions.
	// The TablePad should not add extra padding that would push content below the footer.
	// The suggestion space is handled separately in the rendering order.
	// When help/about is anchored to the top, avoid adding extra padding that can
	// push the top popup off-screen in smaller terminals.
	if snap.HelpTop != "" {
		snap.TablePad = 0
	}

	// Separator between table and expression bar
	if !m.NoColor {
		greyBG := lipgloss.Color("240")
		separatorStyle := lipgloss.NewStyle().
			Background(greyBG).
			Width(m.WinWidth)
		snap.Separator = separatorStyle.Render(strings.Repeat(" ", m.WinWidth))
	} else {
		snap.Separator = strings.Repeat("-", m.WinWidth)
	}

	// Path input section - use configurable prompts
	inputLabel := m.InputPromptUnfocused
	if inputLabel == "" {
		inputLabel = "$ "
	}
	if m.InputFocused {
		focusedPrompt := m.InputPromptFocused
		if focusedPrompt == "" {
			focusedPrompt = "❯ "
		}
		if m.NoColor {
			inputLabel = focusedPrompt
		} else {
			inputLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render(focusedPrompt)
		}
	}
	inputContent := inputLabel + m.PathInput.View()
	if !m.NoColor {
		if fg := CurrentTheme().InputFG; fg != nil {
			inputContent = lipgloss.NewStyle().Foreground(fg).Render(inputContent)
		}
	}
	if !m.NoColor && !m.InputFocused {
		th := CurrentTheme()
		// Apply grey background to make expression bar stand out (only in table mode, not expr mode)
		// Set width to window width to ensure full-width background
		// The width includes the padding, so we set it to WinWidth
		inputContainerStyle := lipgloss.NewStyle().
			Background(th.InputBG). // Dark grey background
			Padding(0, 1).          // Small horizontal padding
			Width(m.WinWidth).      // Full width to match window (includes padding)
			Align(lipgloss.Left)    // Left-align content within full-width background
		inputContent = inputContainerStyle.Render(inputContent)
	}
	snap.Input = inputContent

	// Info popup
	if m.ShowInfoPopup && m.InfoPopup != "" && m.InfoPopupEnabled {
		anchor := m.InfoPopupAnchor
		if anchor == "" {
			anchor = "inline"
		}
		justify := m.InfoPopupJustify
		content := strings.TrimRight(m.InfoPopup, "\n")
		rendered := m.renderInfoPopupBox(content, justify)
		switch anchor {
		case "top":
			// Render before help
			snap.InfoPopupTop = rendered + "\n"
		default:
			snap.InfoPopup = rendered + "\n"
		}
	}

	// Help popup (special help-bound popup rendered above help)
	if m.HelpVisible && m.HelpPopupText != "" {
		justify := m.HelpPopupJustify
		if justify == "" {
			justify = "right"
		}
		content := strings.TrimRight(m.HelpPopupText, "\n")
		rendered := m.renderInfoPopupBox(content, justify)
		if strings.ToLower(m.HelpPopupAnchor) == "top" {
			// Will be rendered in HelpTop alongside help content
			snap.HelpTop = rendered + "\n"
		} else {
			snap.HelpPopup = rendered + "\n"
		}
	}

	// Help overlay and popup
	if m.HelpVisible {
		// Render overlay content
		helpModel := m.Help
		helpModel.Visible = true
		helpModel.NoColor = m.NoColor
		helpModel.AllowEditInput = m.AllowEditInput
		helpModel.KeyMode = m.KeyMode
		helpModel.AboutTitle = m.HelpAboutTitle
		helpModel.AboutLines = m.HelpAboutLines
		helpModel.AboutAlign = m.HelpAboutAlign
		helpModel.HelpNavigationDescriptions = m.HelpNavigationDescriptions
		helpModel.SetWidth(m.WinWidth)
		helpContent := helpModel.View()
		if helpContent != "" {
			snap.Help = helpContent
		}

		// Place popup in configured anchor position
		if snap.HelpPopup != "" {
			if strings.ToLower(m.HelpPopupAnchor) == "top" {
				snap.HelpTop = snap.HelpPopup
			} else {
				snap.Help = snap.HelpPopup + snap.Help
			}
			snap.HelpPopup = ""
		}
	}

	// Render suggestions component (dropdown for CEL functions when typing ".")
	snap.Suggestions = m.SuggestionsComponent.View("")

	// Status bar
	if m.Status.TotalRows == 0 && len(m.AllRows) > 0 {
		m.Status.TotalRows = len(m.AllRows)
	}
	if m.Status.CursorIndex == 0 {
		m.Status.CursorIndex = m.Tbl.Cursor() + 1
		if m.Status.CursorIndex < 1 && len(m.AllRows) > 0 {
			m.Status.CursorIndex = 1
		}
	}
	snap.Status = m.Status.View()
	if strings.TrimSpace(stripANSI(snap.Status)) == "" {
		page := ""
		if len(m.AllRows) > 0 {
			page = fmt.Sprintf("%d/%d", m.Status.CursorIndex, len(m.AllRows))
		}
		padding := m.WinWidth
		if padding <= 0 {
			padding = 92
		}
		if len(page) < padding {
			page = strings.Repeat(" ", padding-len(page)) + page
		}
		snap.Status = page + "\n"
	}

	// Debug bar
	snap.Debug = m.Debug.View()

	// Footer
	snap.Footer = renderFooter(m.NoColor, m.AllowEditInput, m.WinWidth, m.KeyMode)

	return snap
}

func (m *Model) View() tea.View {
	// Sync all components before building snapshot to ensure suggestions are up-to-date
	m.syncAllComponents()
	return m.panelLayoutView()
}

func (m *Model) panelLayoutView() tea.View {
	inputVisible := m.InputFocused || m.AdvancedSearchActive || m.MapFilterActive
	helpTitle := strings.TrimSpace(m.HelpTitle)
	if helpTitle == "" {
		helpTitle = "Help"
	}
	// Always regenerate help text to reflect the current KeyMode
	menu := CurrentMenuConfig()
	helpText := strings.TrimSpace(GenerateHelpText(menu, m.AllowEditInput, m.HelpNavigationDescriptions, m.KeyMode))
	if m.NoColor {
		helpText = stripANSI(helpText)
	}
	state := panelLayoutStateFromModel(m, PanelLayoutModelOptions{
		AppName:        m.AppName,
		HelpTitle:      helpTitle,
		HelpText:       helpText,
		SnapshotHeader: false,
		InputVisible:   inputVisible,
	})
	view := RenderPanelLayout(state)
	if m.NoColor {
		view = stripANSIExceptInverse(view)
	}
	v := tea.NewView(view)
	v.AltScreen = true
	// Enable keyboard enhancements for proper modifier key detection (e.g., Shift+Tab)
	v.KeyboardEnhancements.ReportEventTypes = true
	return v
}

func stripANSI(s string) string {
	return ansiRegexp.ReplaceAllString(s, "")
}

// replaceHelpTextForKeyMode replaces the navigation help section in popup text
// with the correct version for the given KeyMode. The popup text may have been
// pre-rendered with DefaultKeyMode (vim), so we need to swap it out.
func replaceHelpTextForKeyMode(popupText string, keyMode KeyMode, allowEditInput bool) string {
	if keyMode == DefaultKeyMode {
		return popupText // Already correct
	}
	// Generate help text for both default and current mode
	menu := CurrentMenuConfig()
	defaultHelp := GenerateHelpText(menu, allowEditInput, nil, DefaultKeyMode)
	currentHelp := GenerateHelpText(menu, allowEditInput, nil, keyMode)
	if defaultHelp == "" || currentHelp == "" {
		return popupText
	}
	// Strip ANSI codes for plain text comparison
	defaultHelpPlain := stripANSI(defaultHelp)
	popupTextPlain := stripANSI(popupText)
	if !strings.Contains(popupTextPlain, defaultHelpPlain) {
		// The popup text doesn't contain the default help - return as-is
		return popupText
	}
	// Do the replacement on plain text, then we need to return plain text
	// since the styled version won't match exactly due to ANSI codes
	currentHelpPlain := stripANSI(currentHelp)
	result := strings.Replace(popupTextPlain, defaultHelpPlain, currentHelpPlain, 1)
	return result
}

// stripANSIExceptInverse removes color/formatting codes but preserves inverse
// video sequences so selection highlighting stays visible in no-color mode.
func stripANSIExceptInverse(s string) string {
	return ansiRegexp.ReplaceAllStringFunc(s, func(seq string) string {
		switch seq {
		case "\x1b[7m", "\x1b[27m", "\x1b[0m", "\x1b[m":
			return seq
		default:
			return ""
		}
	})
}

func renderFooter(noColor, allowEditInput bool, maxWidth int, keyMode KeyMode) string {
	fkeyStyle := lipgloss.NewStyle()
	if !noColor {
		th := CurrentTheme()
		if th.FooterFG != nil {
			fkeyStyle = fkeyStyle.Foreground(th.FooterFG)
		}
		if th.FooterBG != nil {
			fkeyStyle = fkeyStyle.Background(th.FooterBG)
		}
		fkeyStyle = fkeyStyle.Bold(true)
	}

	var parts []string
	actionOrder := []string{"help", "search", "filter", "copy", "expr", "quit"}
	menu := CurrentMenuConfig()

	renderKey := func(key string) string {
		if !noColor {
			return fkeyStyle.Render(key)
		}
		return key
	}

	// Build parts from menu config for all key modes
	for _, actionName := range actionOrder {
		item, ok := menu.Items[actionName]
		if !ok || !item.Enabled || item.Label == "" {
			continue
		}
		if item.Action == "expr_toggle" && !allowEditInput {
			continue
		}

		var key string
		switch keyMode {
		case KeyModeVim:
			key = item.Keys.Vim
		case KeyModeEmacs:
			key = formatEmacsKey(item.Keys.Emacs)
		case KeyModeFunction:
			key = strings.ToUpper(item.Keys.Function)
		}

		if key != "" {
			parts = append(parts, renderKey(key)+" "+item.Label)
		}
	}

	// Fallback: if no parts from config, try DefaultMenuConfig
	if len(parts) == 0 {
		defaultMenu := DefaultMenuConfig()
		for _, actionName := range actionOrder {
			item, ok := defaultMenu.Items[actionName]
			if !ok || !item.Enabled || item.Label == "" {
				continue
			}
			if item.Action == "expr_toggle" && !allowEditInput {
				continue
			}
			var key string
			switch keyMode {
			case KeyModeVim:
				key = item.Keys.Vim
			case KeyModeEmacs:
				key = formatEmacsKey(item.Keys.Emacs)
			case KeyModeFunction:
				key = strings.ToUpper(item.Keys.Function)
			}
			if key != "" {
				parts = append(parts, renderKey(key)+" "+item.Label)
			}
		}
	}

	// Last resort: hardcoded defaults
	if len(parts) == 0 {
		switch keyMode {
		case KeyModeVim:
			parts = []string{
				renderKey("?") + " help",
				renderKey("/") + " search",
				renderKey("f") + " filter",
				renderKey("y") + " copy",
				renderKey(":") + " expr",
				renderKey("q") + " quit",
			}
		case KeyModeEmacs:
			parts = []string{
				renderKey("F1") + " help",
				renderKey("C-s") + " search",
				renderKey("C-l") + " filter",
				renderKey("M-w") + " copy",
				renderKey("M-x") + " expr",
				renderKey("C-q") + " quit",
			}
		case KeyModeFunction:
			parts = []string{
				renderKey("F1") + " help",
				renderKey("F3") + " search",
				renderKey("F4") + " filter",
				renderKey("F5") + " copy",
				renderKey("F6") + " expr",
				renderKey("F10") + " quit",
			}
		}
	}

	fitParts := func(entries []string, width int) string {
		if width <= 0 {
			return strings.Join(entries, " ")
		}
		if len(entries) == 0 {
			return ""
		}
		var out []string
		curWidth := 0
		for _, entry := range entries {
			entryWidth := ansiVisibleWidth(entry)
			sep := 0
			if len(out) > 0 {
				sep = 1
			}
			if curWidth+sep+entryWidth > width {
				if len(out) == 0 {
					out = append(out, clampANSITextWidth(entry, width))
				}
				break
			}
			if sep == 1 {
				out = append(out, " ")
				curWidth++
			}
			out = append(out, entry)
			curWidth += entryWidth
		}
		return strings.Join(out, "")
	}

	helpLine := fitParts(parts, maxWidth)
	return helpLine
}

func (m *Model) setShowInfoPopup(visible bool) {
	if !m.InfoPopupEnabled {
		return
	}
	if !visible && m.InfoPopupPermanent {
		return
	}
	if m.ShowInfoPopup == visible {
		return
	}
	m.ShowInfoPopup = visible
	m.applyLayout(true)
}

func modelVersionString() string {
	info, ok := rdebug.ReadBuildInfo()
	if !ok {
		return "kvx dev"
	}
	version := info.Main.Version
	if version == "" || version == "(devel)" {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" && len(s.Value) >= 7 {
				version = s.Value[:7]
				break
			}
		}
	}
	if version == "" || version == "(devel)" {
		version = "dev"
	}
	return fmt.Sprintf("kvx %s", version)
}

// SnapshotConfig configures snapshot construction without running a Bubble Tea program.
type SnapshotConfig struct {
	Width              int
	Height             int
	NoColor            bool
	HelpVisible        bool
	ShowSuggestions    bool
	InputValue         string
	InputFocused       bool
	Path               string
	FilteredSuggs      []string
	SelectedSuggestion int
	Configure          func(*Model)
}

// buildSnapshotFromNode builds a model snapshot and rendered string for the given node using config.
// This is used by tests to assert view output without spinning up a Bubble Tea program.
func buildSnapshotFromNode(node interface{}, cfg SnapshotConfig) string { //nolint:unused // snapshot helper preserved for future tests
	m := InitialModel(node)
	m.Root = node
	m.NoColor = cfg.NoColor
	m.HelpVisible = cfg.HelpVisible
	m.ShowSuggestions = cfg.ShowSuggestions
	m.InputFocused = cfg.InputFocused
	if cfg.Path != "" {
		m.Path = cfg.Path
	}
	if cfg.InputValue != "" {
		m.PathInput.SetValue(cfg.InputValue)
	}
	if cfg.Width > 0 {
		m.WinWidth = cfg.Width
	} else {
		m.WinWidth = 80
	}
	if cfg.Height > 0 {
		m.WinHeight = cfg.Height
	} else {
		m.WinHeight = 24
	}
	if len(cfg.FilteredSuggs) > 0 {
		m.FilteredSuggestions = cfg.FilteredSuggs
		m.SelectedSuggestion = cfg.SelectedSuggestion
	}
	if cfg.Configure != nil {
		cfg.Configure(&m)
	}
	m.ApplyColorScheme()
	m.applyLayout(true)
	m.syncAllComponents()
	snap := m.buildViewSnapshot()
	return renderSnapshot(snap)
}

// syncStatus updates the status component with current model state
func (m *Model) syncStatus() {
	m.Status.ErrMsg = m.ErrMsg
	m.Status.StatusType = m.StatusType
	m.Status.AdvancedSearchActive = m.AdvancedSearchActive
	m.Status.AdvancedSearchQuery = m.AdvancedSearchQuery
	m.Status.AdvancedSearchResults = m.AdvancedSearchResults
	m.Status.FilterActive = m.FilterActive
	m.Status.FilterBuffer = m.FilterBuffer
	m.Status.InputFocused = m.InputFocused
	m.Status.FilteredSuggestions = m.FilteredSuggestions
	m.Status.SelectedSuggestion = m.SelectedSuggestion
	m.Status.ShowSuggestions = m.ShowSuggestions
	m.Status.SuggestionSummary = m.SuggestionSummary
	m.Status.ShowSuggestionSummary = m.ShowSuggestionSummary
	m.Status.HelpVisible = m.HelpVisible
	m.Status.FunctionHelpText = m.detectFunctionHelp()
	m.Status.InputValue = m.PathInput.Value() // Pass input value to status bar
	m.Status.NoColor = m.NoColor
	m.Status.SetWidth(m.WinWidth)

	// If user just typed a trailing dot, prefer a summary built from current completions.
	if m.InputFocused && strings.HasSuffix(strings.TrimSpace(m.PathInput.Value()), ".") {
		if summary := buildFunctionSummaryFromCompletions(m.Status.Completions, 0); summary != "" {
			m.Status.SuggestionSummary = summary
			m.Status.ShowSuggestionSummary = true
		}
	}

	// Calculate current cursor index (1-based) and total rows
	cursor := m.Tbl.Cursor()
	var totalRows int

	if m.AdvancedSearchActive && len(m.AdvancedSearchResults) > 0 {
		// In search mode, use search results count
		totalRows = len(m.AdvancedSearchResults)
	} else {
		// In normal mode, use table rows (which may be filtered)
		tableRows := m.Tbl.Rows()
		if len(tableRows) > 0 {
			totalRows = len(tableRows)
		} else if len(m.AllRows) > 0 {
			totalRows = len(m.AllRows)
		}
	}

	// Set cursor index (1-based, or 0 if no rows)
	if totalRows > 0 && cursor >= 0 && cursor < totalRows {
		m.Status.CursorIndex = cursor + 1
	} else {
		m.Status.CursorIndex = 0
	}
	m.Status.TotalRows = totalRows
}

// detectFunctionHelp detects the function being typed or at cursor position and returns its help text
func (m *Model) detectFunctionHelp() string {
	if !m.InputFocused {
		return ""
	}

	input := m.PathInput.Value()
	trimmed := strings.TrimSpace(input)
	// If the user just typed a trailing dot, prefer suggestion summary instead of stale function help.
	if strings.HasSuffix(trimmed, ".") {
		return ""
	}
	// Only show CEL help after a dot has been typed
	if !strings.Contains(input, ".") {
		return ""
	}

	// PRIORITY: Case 1 - Detect function call (e.g., "_.map()" or "_.map(x,")
	// This handles when user is editing inside a function call - help should persist
	// Look for the pattern: .functionName( ... with cursor potentially inside
	parenIdx := strings.LastIndex(input, "(")
	if parenIdx >= 0 {
		// We have a function call - extract the function name
		funcPart := input[:parenIdx]
		lastDot := strings.LastIndex(funcPart, ".")
		if lastDot >= 0 {
			funcName := strings.TrimSpace(funcPart[lastDot+1:])
			if funcName != "" {
				baseExpr := "_"
				if lastDot > 0 {
					baseExpr = strings.TrimSpace(funcPart[:lastDot])
					if baseExpr == "" {
						baseExpr = "_"
					}
				}
				suggestionsToSearch := m.FilteredSuggestions
				if len(suggestionsToSearch) == 0 {
					suggestionsToSearch = m.Suggestions
				}
				if help := m.lookupFunctionHelp(funcName, suggestionsToSearch, baseExpr); help != "" {
					return help
				}
			}
		}
	}

	// Case 2: Detect function being typed (e.g., "_.m", "_.ma" -> "map")
	// Look for the last token that might be a partial function name
	// Use full input since we can't easily get exact cursor position
	textUpToCursor := input
	if len(textUpToCursor) > 0 {
		// Find the last token (after last dot)
		lastDot := strings.LastIndex(textUpToCursor, ".")
		if lastDot < 0 {
			return ""
		}
		lastToken := textUpToCursor[lastDot+1:]

		// Remove any trailing "(" or incomplete brackets
		lastToken = strings.TrimSuffix(lastToken, "(")
		lastToken = strings.TrimSuffix(lastToken, "[")
		lastToken = strings.TrimSpace(lastToken)

		if lastToken != "" {
			lowerToken := strings.ToLower(lastToken)
			// Search through all filtered suggestions first, then fall back to all suggestions
			suggestionsToSearch := m.FilteredSuggestions
			if len(suggestionsToSearch) == 0 {
				// If no filtered suggestions, search all suggestions
				suggestionsToSearch = m.Suggestions
			}
			baseExpr := "_"
			if lastDot > 0 {
				baseExpr = strings.TrimSpace(textUpToCursor[:lastDot])
				if baseExpr == "" {
					baseExpr = "_"
				}
			}
			for _, suggestion := range suggestionsToSearch {
				if !isFunctionSuggestion(suggestion) {
					continue
				}
				funcName := extractFunctionName(suggestion)
				if strings.HasPrefix(strings.ToLower(funcName), lowerToken) {
					if help := m.lookupFunctionHelp(funcName, suggestionsToSearch, baseExpr); help != "" {
						return help
					}
				}
			}
		}
	}

	return ""
}

// syncFooter updates the footer component with current model state
func (m *Model) syncFooter() {
	m.Footer.NoColor = m.NoColor
	m.Footer.KeyMode = m.KeyMode
	m.Footer.SetWidth(m.WinWidth)
}

// syncHelp updates the help component with current model state
func (m *Model) syncHelp() {
	m.Help.Visible = m.HelpVisible
	m.Help.NoColor = m.NoColor
	m.Help.KeyMode = m.KeyMode
	m.Help.SetWidth(m.WinWidth)
}

// syncSuggestions updates the suggestions component with current model state
func (m *Model) syncSuggestions() {
	// Always sync InputFocused so suggestions component can render empty lines when in expression mode
	// even if AllowSuggestions is false
	m.SuggestionsComponent.InputFocused = m.InputFocused
	m.SuggestionsComponent.NoColor = m.NoColor
	m.SuggestionsComponent.WinHeight = m.WinHeight
	m.SuggestionsComponent.TableHeight = m.TableHeight

	if !m.AllowSuggestions {
		// Still sync basic state even if suggestions are disabled
		m.SuggestionsComponent.ShowSuggestions = false
		m.SuggestionsComponent.FilteredSuggestions = []string{}
		return
	}
	m.SuggestionsComponent.FilteredSuggestions = m.FilteredSuggestions
	m.SuggestionsComponent.SelectedSuggestion = m.SelectedSuggestion
	m.SuggestionsComponent.ShowSuggestions = m.ShowSuggestions
	m.SuggestionsComponent.InputValue = m.PathInput.Value()
	m.SuggestionsComponent.InputValue = m.PathInput.Value()
}

func (m *Model) handleMenuKey(keyStr string) (bool, tea.Cmd) {
	menu := CurrentMenuConfig()
	getItem := func(k string) *MenuItem {
		switch k {
		case "f1":
			return &menu.F1
		case "f2":
			return &menu.F2
		case "f3":
			return &menu.F3
		case "f4":
			return &menu.F4
		case "f5":
			return &menu.F5
		case "f6":
			return &menu.F6
		case "f7":
			return &menu.F7
		case "f8":
			return &menu.F8
		case "f9":
			return &menu.F9
		case "f10":
			return &menu.F10
		case "f11":
			return &menu.F11
		case "f12":
			return &menu.F12
		default:
			return nil
		}
	}

	item := getItem(keyStr)
	if item == nil || !item.Enabled {
		return false, nil
	}
	if item.Action == "expr_toggle" && !m.AllowEditInput {
		return true, nil
	}

	if item.Action == "help" {
		// Help uses a dedicated popup rendered above the help overlay (not the global info popup).
		m.HelpPopupText = ""
		m.HelpPopupJustify = "center"
		m.HelpPopupAnchor = "top"

		popupEnabled := true
		if item.Popup.Enabled != nil {
			popupEnabled = *item.Popup.Enabled
		}
		// Set anchor and justify from config if present, regardless of whether text is set
		if item.Popup.Anchor != "" {
			m.HelpPopupAnchor = item.Popup.Anchor
		}
		if item.Popup.Justify != "" {
			m.HelpPopupJustify = item.Popup.Justify
		}
		// Set popup text only if enabled and text is provided
		if popupEnabled && item.Popup.Text != "" {
			m.HelpPopupText = item.Popup.Text
			// Replace the default-KeyMode help text with the correct one for current KeyMode
			m.HelpPopupText = replaceHelpTextForKeyMode(m.HelpPopupText, m.KeyMode, m.AllowEditInput)
		}
	} else if InfoPopupHasData(item.Popup) {
		if item.Popup.Enabled != nil && !*item.Popup.Enabled {
			return true, nil
		}
		if item.Popup.Text != "" {
			m.InfoPopup = item.Popup.Text
		}
		if item.Popup.Anchor != "" {
			m.InfoPopupAnchor = item.Popup.Anchor
		}
		if item.Popup.Justify != "" {
			m.InfoPopupJustify = item.Popup.Justify
		}
		if item.Popup.Modal != nil {
			m.InfoPopupModal = *item.Popup.Modal
		}
		if item.Popup.Permanent != nil {
			m.InfoPopupPermanent = *item.Popup.Permanent
		} else {
			m.InfoPopupPermanent = false
		}
		if item.Popup.Enabled != nil {
			m.InfoPopupEnabled = *item.Popup.Enabled
		} else {
			m.InfoPopupEnabled = true
		}
		if m.InfoPopupPermanent {
			m.setShowInfoPopup(true)
		} else {
			m.setShowInfoPopup(!m.ShowInfoPopup)
		}
	}

	actions := CurrentMenuActions()
	actionFn, ok := actions[item.Action]
	if !ok || actionFn == nil {
		return true, nil
	}
	return true, actionFn(m)
}

// Built-in menu actions
func menuActionHelp(m *Model) tea.Cmd {
	// Toggle help popup visibility (not the overlay)
	m.HelpVisible = !m.HelpVisible
	if m.HelpVisible {
		m.ShowSuggestions = false
		// If no help popup text is set (e.g., help bound to another key), seed from the help menu item.
		if m.HelpPopupText == "" {
			menu := CurrentMenuConfig()
			items := []MenuItem{menu.F1, menu.F2, menu.F3, menu.F4, menu.F5, menu.F6, menu.F7, menu.F8, menu.F9, menu.F10, menu.F11, menu.F12}
			for _, it := range items {
				if it.Action != "help" || !it.Enabled || !InfoPopupHasData(it.Popup) {
					continue
				}
				popupEnabled := true
				if it.Popup.Enabled != nil {
					popupEnabled = *it.Popup.Enabled
				}
				if !popupEnabled || it.Popup.Text == "" {
					continue
				}
				m.HelpPopupText = it.Popup.Text
				// Replace the default-KeyMode help text with the correct one for current KeyMode
				m.HelpPopupText = replaceHelpTextForKeyMode(m.HelpPopupText, m.KeyMode, m.AllowEditInput)
				if it.Popup.Justify != "" {
					m.HelpPopupJustify = it.Popup.Justify
				}
				if it.Popup.Anchor != "" {
					m.HelpPopupAnchor = it.Popup.Anchor
				}
				break
			}
		} else {
			// Even if text was already set, honor anchor/justify overrides from config.
			menu := CurrentMenuConfig()
			if InfoPopupHasData(menu.F1.Popup) {
				if menu.F1.Popup.Anchor != "" {
					m.HelpPopupAnchor = menu.F1.Popup.Anchor
				}
				if menu.F1.Popup.Justify != "" {
					m.HelpPopupJustify = menu.F1.Popup.Justify
				}
			}
		}
		// Help popup is rendered separately; do not toggle the shared info popup.
	} else {
		// Hide any lingering info popup when closing help unless permanent.
		if !m.InfoPopupPermanent {
			m.setShowInfoPopup(false)
		}
		m.HelpPopupText = strings.TrimRight(m.HelpPopupText, "\n")
	}
	// Recompute layout when help visibility changes (affects top popups/table height).
	m.applyLayout(true)
	return nil
}

func menuActionExprToggle(m *Model) tea.Cmd {
	if !m.AllowEditInput {
		return nil
	}
	syncToCursor := !m.InputFocused && (m.AdvancedSearchActive || m.SearchContextActive)
	if syncToCursor {
		// Sync expr bar to current selection before switching modes when search context is active
		m.syncPathInputWithCursor()
	}
	m.InputFocused = !m.InputFocused
	if m.InputFocused {
		// Entering expr mode cancels search/filter context
		m.clearSearchState()
		// Use the current PathInput value directly - it already has the user's expression
		currentValue := strings.TrimSpace(m.PathInput.Value())
		if currentValue == "" {
			currentValue = "_"
		}
		m.ExprDisplay = currentValue
		m.PathInput.SetValue(currentValue)
		m.PathInput.SetCursor(len(currentValue))
		// Focus will be handled by SyncTableState() when InputFocused changes
		// Clear filter state after transferring to path input
		m.FilterActive = false
		m.FilterBuffer = ""
		m.filterSuggestions(m.AllowIntellisense)
		// When entering expression mode, if there are no filtered suggestions, ensure ShowSuggestions is false
		// so the suggestions component can render empty lines to reserve space
		// Also, if AllowIntellisense is false, don't show suggestions even if there are filtered ones
		if len(m.FilteredSuggestions) == 0 || !m.AllowIntellisense {
			m.ShowSuggestions = false
		}
		// Sync suggestions component before recalculating layout
		m.syncSuggestions()
		// Recalculate layout to account for suggestions taking up space
		m.applyLayout(true)
		return m.PathInput.Focus()
	}
	// Leaving expr mode: clear suggestions and show current path; focus table.
	m.ShowSuggestions = false
	m.SuggestionFilterActive = false
	m.FilteredSuggestionRows = nil
	m.PathInput.SetCursor(len(m.PathInput.Value()))
	m.PathInput.Blur()
	m.Tbl.Focus()
	// Recalculate layout when leaving expr mode (suggestions are now hidden)
	m.applyLayout(true)
	return nil
}

func menuActionSearch(m *Model) tea.Cmd {
	// Don't activate if already in input mode
	if m.InputFocused {
		return nil
	}
	// Save current state for restoration on Esc
	m.PreviousNode = m.Node
	m.PreviousPath = m.Path
	currentRows := m.Tbl.Rows()
	m.PreviousAllRows = append([]table.Row(nil), currentRows...)
	m.logEvent("enter-search:f3")

	// Activate search mode
	m.clearSearchState()
	m.AdvancedSearchActive = true
	m.AdvancedSearchCommitted = false // Start with top-level filter, Enter commits to deep search
	m.AdvancedSearchBasePath = m.Path // Store current path as base for search results
	m.FilterActive = false            // Clear type-ahead filter
	m.FilterBuffer = ""

	// Clear table initially (empty query shows no results)
	m.AllRows = []table.Row{}
	m.AllRowKeys = nil
	m.AllRowKeys = nil
	m.SyncTableState()
	m.clearErrorUnlessSticky()
	m.logEvent("search-cleared")
	m.SearchInput.SetValue("")
	m.SearchInput.SetCursor(0)
	return m.SearchInput.Focus()
}

func menuActionFilter(m *Model) tea.Cmd {
	// Don't activate if already in input mode or another filter/search
	if m.InputFocused || m.AdvancedSearchActive || m.MapFilterActive {
		return nil
	}
	// Filter only works on maps
	if _, isMap := m.Node.(map[string]interface{}); !isMap {
		return nil
	}
	m.logEvent("enter-filter:f4")

	// Save current state for restoration on Esc
	m.PreviousNode = m.Node
	m.PreviousPath = m.Path
	currentRows := m.Tbl.Rows()
	m.PreviousAllRows = append([]table.Row(nil), currentRows...)

	// Activate map filter mode
	m.MapFilterActive = true
	m.MapFilterQuery = ""
	m.MapFilterInput.SetValue("")
	m.MapFilterInput.SetCursor(0)
	m.FilterActive = false // Clear type-ahead filter
	m.FilterBuffer = ""

	m.clearErrorUnlessSticky()
	return m.MapFilterInput.Focus()
}

func menuActionCopy(m *Model) tea.Cmd {
	// Always use PathInput value (works for both input mode and search mode)
	expr := strings.TrimSpace(m.PathInput.Value())
	if expr == "" {
		expr = "_"
	}
	cliSafe := makePathCLISafe(expr)
	if err := copyToClipboard(cliSafe); err != nil {
		m.ErrMsg = fmt.Sprintf("Clipboard unavailable: %v", err)
		m.StatusType = "error"
	} else {
		m.ErrMsg = fmt.Sprintf("Copied: %s", cliSafe)
		m.StatusType = "success"
	}
	return nil
}

func menuActionQuit(m *Model) tea.Cmd {
	// Both expr and table modes use the PathInput value
	evalExpr := strings.TrimSpace(m.PathInput.Value())
	if evalExpr == "" {
		evalExpr = "_"
	}
	// Expressions are already properly formatted - don't modify them
	if _, err := EvaluateExpression(evalExpr, m.Root); err != nil {
		m.ErrMsg = fmt.Sprintf("explore expression error: %v", err)
		m.StatusType = "error"
		return nil
	}
	return m.printCLIOutput(evalExpr)
}

// Custom action hook (no-op by default, can be overridden via RegisterMenuAction).
func menuActionCustom(_ *Model) tea.Cmd { return nil }

// syncAllComponents syncs all UI components with the current model state.
// This centralizes component syncing to ensure consistency and reduce duplication.
func (m *Model) syncAllComponents() {
	m.syncStatus()
	m.syncFooter()
	m.syncHelp()
	m.syncDebug()
	m.syncSuggestions()
}

// syncDebug updates the debug component with current model state
func (m *Model) syncDebug() {
	m.Debug.NoColor = m.NoColor
	m.Debug.SetWidth(m.WinWidth)

	if !m.DebugMode {
		m.Debug.Visible = false
		m.Debug.LastDebugOutput = ""
		m.Debug.LastDebugValues = ""
		m.LastDebugOutput = ""
		m.LastDebugValues = ""
		return
	}

	m.Debug.Visible = true

	// Build state signature to detect changes
	stateKey := fmt.Sprintf("%d:%d:%d:%d:%v:%d:%v",
		len(m.Tbl.Rows()), m.Tbl.Cursor(), m.TableHeight,
		len(m.FilteredSuggestions), m.InputFocused,
		m.SelectedSuggestion, m.ShowSuggestions)

	// Get first row preview for debugging
	firstRowPreview := ""
	switch {
	case len(m.Tbl.Rows()) > 0:
		firstRow := m.Tbl.Rows()[0]
		if len(firstRow) > 0 {
			// Get first cell content, truncate to 20 chars
			cellContent := firstRow[0]
			if len(cellContent) > 20 {
				firstRowPreview = cellContent[:20] + "..."
			} else {
				firstRowPreview = cellContent
			}
		} else {
			firstRowPreview = "(empty row)"
		}
	case len(m.AllRows) > 0:
		firstRow := m.AllRows[0]
		if len(firstRow) > 0 {
			cellContent := firstRow[0]
			if len(cellContent) > 20 {
				firstRowPreview = cellContent[:20] + "..."
			} else {
				firstRowPreview = cellContent
			}
		} else {
			firstRowPreview = "(empty AllRows)"
		}
	default:
		firstRowPreview = "(no rows)"
	}

	// Get column count
	columnCount := 0
	if len(m.Tbl.Rows()) > 0 {
		columnCount = len(m.Tbl.Rows()[0])
	} else if len(m.AllRows) > 0 {
		columnCount = len(m.AllRows[0])
	}

	// Check if table is focused (table component has Focus() method)
	tableFocused := false
	// We can't directly check if table is focused, but we can infer from InputFocused
	// If InputFocused is false, table should be focused
	tableFocused = !m.InputFocused

	// Calculate layout information for debugging
	helpInline := m.HelpVisible && strings.ToLower(m.HelpPopupAnchor) != "top"
	// Don't reserve space for suggestions - they're shown in status bar
	heights := m.Layout.CalculateHeights(helpInline, m.DebugMode, false, m.ShowPanelTitle)
	bottomBlockHeight := heights.PathInputHeight +
		heights.SuggestionHeight +
		heights.StatusHeight +
		heights.DebugHeight +
		heights.FooterHeight +
		1 // Separator line

	// Calculate table pad (same logic as buildViewSnapshot)
	tablePad := 0
	if m.TableHeight > 0 {
		actualTableLines := countLines(m.Tbl.View())
		desiredTableLines := m.TableHeight + TableHeaderLines
		if desiredTableLines > actualTableLines {
			tablePad = desiredTableLines - actualTableLines
		}
	}
	if m.HelpVisible && strings.ToLower(m.HelpPopupAnchor) == "top" {
		tablePad = 0
	}

	// Update debug info with current state
	debugInfo := DebugInfo{
		WinWidth:             m.WinWidth,
		WinHeight:            m.WinHeight,
		TableHeight:          m.TableHeight,
		KeyColWidth:          m.KeyColWidth,
		ValueColWidth:        m.ValueColWidth,
		ShownRows:            len(m.Tbl.Rows()),
		TotalRows:            len(m.AllRows),
		AllRowsCount:         len(m.AllRows),
		Cursor:               m.Tbl.Cursor(),
		InputFocused:         m.InputFocused,
		TableFocused:         tableFocused,
		ColumnCount:          columnCount,
		FilteredSuggestions:  len(m.FilteredSuggestions),
		TotalSuggestions:     len(m.Suggestions),
		SelectedSuggestion:   m.SelectedSuggestion,
		ShowSuggestions:      m.ShowSuggestions,
		InputValue:           m.PathInput.Value(),
		Path:                 m.Path,
		AdvancedSearchActive: m.AdvancedSearchActive,
		SearchContextActive:  m.SearchContextActive,
		FirstRowPreview:      firstRowPreview,
		// Layout information
		LayoutTableHeight:            heights.TableHeight,
		LayoutSuggestionHeight:       heights.SuggestionHeight,
		LayoutPathInputHeight:        heights.PathInputHeight,
		LayoutStatusHeight:           heights.StatusHeight,
		LayoutDebugHeight:            heights.DebugHeight,
		LayoutFooterHeight:           heights.FooterHeight,
		LayoutBottomBlockHeight:      bottomBlockHeight,
		LayoutTablePad:               tablePad,
		LayoutReserveSuggestionSpace: false, // Always false now - suggestions in status bar
	}

	m.Debug.UpdateDebugInfo(stateKey, debugInfo)

	// Sync cached values back to model for compatibility
	m.LastDebugOutput = m.Debug.LastDebugOutput
	m.LastDebugValues = m.Debug.LastDebugValues
}

// removeLastSegment removes the last path segment from a path string.
// Handles both dot notation and bracket notation.
// Examples:
//
//	"items.0.tags" -> "items.0"
//	"items[0].tags" -> "items[0]"
//	"items[0].tags[1]" -> "items[0].tags"
func removeLastSegment(path string) string {
	if path == "" {
		return ""
	}

	// Find the last dot or bracket
	lastDotIdx := strings.LastIndex(path, ".")
	lastBracketIdx := strings.LastIndex(path, "[")

	// If there's a bracket after the last dot, we need to handle it carefully
	if lastBracketIdx > lastDotIdx {
		// Remove the bracket segment: "items[0].tags[1]" -> "items[0].tags"
		return path[:lastBracketIdx]
	}

	// Otherwise just remove after the last dot: "items.0.tags" -> "items.0"
	if lastDotIdx >= 0 {
		return path[:lastDotIdx]
	}

	// No dots or brackets, return empty
	return ""
}

// parsePathKeys extracts path segments from a path string.
// Handles both dot notation and bracket notation.
// Examples:
//
//	"items.0.tags" -> ["items", "0", "tags"]
//	"items[0].tags" -> ["items", "0", "tags"]
//	"items[0].tags[1]" -> ["items", "0", "tags", "1"]
func parsePathKeys(path string) []string {
	if path == "" {
		return []string{}
	}

	var keys []string
	var current strings.Builder

	for i := 0; i < len(path); i++ {
		ch := path[i]
		switch ch {
		case '.':
			if current.Len() > 0 {
				keys = append(keys, current.String())
				current.Reset()
			}
		case '[':
			if current.Len() > 0 {
				keys = append(keys, current.String())
				current.Reset()
			}
			// Find the closing bracket
			j := i + 1
			for j < len(path) && path[j] != ']' {
				j++
			}
			if j < len(path) {
				keys = append(keys, path[i+1:j])
				i = j
			}
		default:
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		keys = append(keys, current.String())
	}
	return keys
}

// makePathCLISafe wraps a CEL path expression in quotes if needed for CLI safety.
// Simple paths like "_.foo.bar" are returned as-is.
// Paths with special shell characters are quoted appropriately:
// - Single quotes in path → wrap in double quotes
// - Double quotes in path → wrap in single quotes
// - Brackets, spaces, or other special chars → wrap in single quotes (default)
func makePathCLISafe(path string) string {
	if path == "" {
		return path
	}

	// Check if path needs quoting (has special shell characters)
	needsQuoting := false
	hasSingleQuote := false
	hasDoubleQuote := false

	specialChars := []rune{'[', ']', '(', ')', ' ', '\t', '&', '|', ';', '<', '>', '$', '`', '\\', '!', '*', '?', '{', '}', '#', '~'}

	for _, ch := range path {
		switch ch {
		case '\'':
			hasSingleQuote = true
			needsQuoting = true
		case '"':
			hasDoubleQuote = true
			needsQuoting = true
		default:
			for _, special := range specialChars {
				if ch == special {
					needsQuoting = true
					break
				}
			}
		}
	}

	// Simple path like "_.foo.bar" - no quoting needed
	if !needsQuoting {
		return path
	}

	// Path has both single and double quotes - escape double quotes and wrap in double quotes
	if hasSingleQuote && hasDoubleQuote {
		escaped := strings.ReplaceAll(path, `"`, `\"`)
		return `"` + escaped + `"`
	}

	// Path has single quotes - wrap in double quotes
	if hasSingleQuote {
		return `"` + path + `"`
	}

	// Path has double quotes - wrap in single quotes
	if hasDoubleQuote {
		return `'` + path + `'`
	}

	// Default: wrap in single quotes (safer for most shells)
	return `'` + path + `'`
}

// copyToClipboard attempts to copy text to the system clipboard using platform-specific commands.

// Returns an error if the clipboard command is not available or fails.
func copyToClipboard(text string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "pbcopy")
	case "linux":
		// Try xclip first, then xsel, then wl-copy (Wayland)
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.CommandContext(ctx, "xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.CommandContext(ctx, "xsel", "--clipboard", "--input")
		} else if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.CommandContext(ctx, "wl-copy")
		} else {
			return fmt.Errorf("no clipboard command found (install xclip, xsel, or wl-clipboard)")
		}
	case "windows":
		cmd = exec.CommandContext(ctx, "clip")
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	_, _ = stdin.Write([]byte(text))
	_ = stdin.Close()

	return cmd.Wait()
}

// printCLIOutput evaluates the given expression against the root and prints
// results using the same rules as the CLI default output, then quits.
func (m *Model) printCLIOutput(expr string) tea.Cmd {
	// Use expression exactly as provided - no normalization
	evalExpr := strings.TrimSpace(expr)
	if evalExpr == "" {
		return tea.Quit
	}
	m.PendingCLIExpr = evalExpr
	m.PrintedInTUI = false
	return tea.Quit
}
