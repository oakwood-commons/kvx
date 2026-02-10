//go:build ignore
// +build ignore

package expression

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"

	"github.com/oakwood-commons/kvx/internal/completion"
	"github.com/oakwood-commons/kvx/internal/ui"
	"github.com/oakwood-commons/kvx/internal/ui/table"
)

// Suggestion represents a CEL function or key suggestion
type Suggestion struct {
	Name        string // Function/key name
	Description string // Usage hint or description
	IsFunction  bool   // Whether this is a CEL function vs a key
}

// Model represents the expression input child model for CEL expression editing
type Model struct {
	input              textinput.Model
	root               interface{}                  // Root data for context
	currentNode        interface{}                  // Current node for context-aware suggestions
	path               string                       // Current path context
	completionEngine   *completion.CompletionEngine // Completion engine for suggestions
	completions        []completion.Completion      // Current filtered completions
	selectedCompletion int                          // Currently selected completion index
	showCompletions    bool                         // Whether to show completions
	suggestionTable    *table.Model[Suggestion]     // Table for displaying suggestions (legacy)
	width              int
	height             int
	focused            bool
	noColor            bool
	allowIntellisense  bool                  // Whether to show CEL function suggestions
	enableGhostText    bool                  // Whether to show ghost text completions
	submitted          bool                  // Whether expression was submitted (Enter pressed)
	cancelled          bool                  // Whether input was cancelled (Esc pressed)
	result             string                // Final expression result (after submission)
	resultType         string                // Type of the last expression result
	parentTableUpdater func(tea.Msg) tea.Cmd // Callback to update parent table (for Up/Down navigation)
}

// NewModel creates a new expression input model
func NewModel(root interface{}, path string, currentNode interface{}) *Model {
	ti := textinput.New()
	ti.Placeholder = "Enter CEL expression (e.g., _.items.filter(x, x.available))"
	ti.CharLimit = 500
	ti.SetWidth(80)
	ti.Prompt = "❯ "

	// Initialize with current path (or "_" for root)
	initialValue := path
	if initialValue == "" {
		initialValue = "_"
	}
	ti.SetValue(initialValue)
	ti.Focus()

	// Create suggestion table with toRow and keyFunc (legacy, for compatibility)
	toRow := func(s Suggestion) table.Row {
		if s.IsFunction {
			return table.Row{s.Name + "()", s.Description}
		}
		return table.Row{s.Name, s.Description}
	}
	keyFunc := func(s Suggestion) string {
		return s.Name
	}

	columns := []table.Column{
		{Title: "NAME", Width: 30},
		{Title: "DESCRIPTION", Width: 50},
	}
	suggestionTable := table.NewModel(columns, toRow, keyFunc)
	suggestionTable.SetHeight(10)

	// Create CEL completion engine
	celProvider, err := completion.NewCELProvider()
	var engine *completion.CompletionEngine
	if err == nil {
		engine = completion.NewEngine(celProvider)
	}

	return &Model{
		input:              ti,
		root:               root,
		currentNode:        currentNode,
		path:               path,
		completionEngine:   engine,
		completions:        []completion.Completion{},
		selectedCompletion: 0,
		showCompletions:    false,
		suggestionTable:    suggestionTable,
		width:              80,
		height:             24,
		focused:            true,
		noColor:            false,
		allowIntellisense:  true,
		enableGhostText:    false, // Disabled by default for Bubble Tea 2.0 compatibility
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) (ui.ChildModel, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			// Submit the expression
			m.submitted = true
			m.result = strings.TrimSpace(m.input.Value())
			// Infer result type if completion engine is available
			if m.completionEngine != nil {
				m.resultType = m.completionEngine.InferType(m.result, m.buildCompletionContext())
			}
			return m, nil

		case "esc":
			// Cancel input
			m.cancelled = true
			return m, nil

		case "tab":
			// Accept currently selected completion from status panel
			if m.showCompletions && len(m.completions) > 0 {
				selected := m.completions[m.selectedCompletion]
				m.insertCompletion(selected)
				m.filterCompletions() // Refresh completions after insertion
			}
			return m, nil

		case "up", "down":
			// Forward Up/Down to parent table for data navigation
			// Do NOT move cursor in textinput
			if m.parentTableUpdater != nil {
				return m, m.parentTableUpdater(msg)
			}
			return m, nil

		case "ctrl+u":
			// Clear input
			m.input.SetValue("")
			m.input.SetCursor(0)
			m.showCompletions = false
			return m, nil
		}
	default:
		return m, nil
	}

	// Update input and filter completions
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.filterCompletions()

	return m, cmd
}

// View renders the expression input view
func (m *Model) View() string {
	var b strings.Builder

	// Input prompt and field
	b.WriteString(m.input.View())
	b.WriteString("\n")

	// Suggestion dropdown (if visible)
	if m.showSuggestions && len(m.filteredSuggestions) > 0 {
		// Show suggestions as a simple list or table
		if m.noColor {
			b.WriteString("\nSuggestions:\n")
		} else {
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
			b.WriteString("\n" + style.Render("Suggestions:") + "\n")
		}

		// Show up to 10 suggestions
		maxShow := 10
		if len(m.filteredSuggestions) < maxShow {
			maxShow = len(m.filteredSuggestions)
		}

		for i := 0; i < maxShow; i++ {
			s := m.filteredSuggestions[i]
			prefix := "  "
			if i == m.selectedSuggestion {
				prefix = "❯ "
			}

			line := fmt.Sprintf("%s%s", prefix, s.Name)
			if s.IsFunction {
				line += "()"
			}
			if s.Description != "" {
				line += " - " + s.Description
			}

			if !m.noColor && i == m.selectedSuggestion {
				style := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
				line = style.Render(line)
			}

			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	return b.String()
}

// SetSize sets the model dimensions
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.input.SetWidth(width - 4) // Account for prompt and padding
}

// Focus sets the focus state
func (m *Model) Focus() tea.Cmd {
	m.focused = true
	return m.input.Focus()
}

// Blur removes focus
func (m *Model) Blur() {
	m.focused = false
	m.input.Blur()
}

// Focused returns whether the model is focused
func (m *Model) Focused() bool {
	return m.focused
}

// Title returns the model title
func (m *Model) Title() string {
	return "Expression"
}

// ID returns a unique identifier for this model instance
func (m *Model) ID() string {
	return fmt.Sprintf("expression:%s", m.path)
}

// SetNoColor sets the color mode
func (m *Model) SetNoColor(noColor bool) {
	m.noColor = noColor
	m.suggestionTable.SetNoColor(noColor)
}

// SetAllowIntellisense sets whether to show CEL function suggestions
func (m *Model) SetAllowIntellisense(allow bool) {
	m.allowIntellisense = allow
}

// SetEnableGhostText sets whether to show ghost text completions
func (m *Model) SetEnableGhostText(enable bool) {
	m.enableGhostText = enable
}

// SetParentTableUpdater sets the callback for updating parent table (for Up/Down navigation)
func (m *Model) SetParentTableUpdater(updater func(tea.Msg) tea.Cmd) {
	m.parentTableUpdater = updater
}

// SetSuggestions sets the available suggestions (CEL functions and context keys)
func (m *Model) SetSuggestions(suggestions []Suggestion) {
	m.suggestions = suggestions
	m.filterSuggestions()
}

// Submitted returns whether the expression was submitted
func (m *Model) Submitted() bool {
	return m.submitted
}

// ResultType returns the type of the last expression result
func (m *Model) ResultType() string {
	return m.resultType
}

// Completions returns the current filtered completions
func (m *Model) Completions() []completion.Completion {
	return m.completions
}

// SelectedCompletion returns the index of the currently selected completion
func (m *Model) SelectedCompletion() int {
	return m.selectedCompletion
}

// Cancelled returns whether the input was cancelled
func (m *Model) Cancelled() bool {
	return m.cancelled
}

// Result returns the final expression result
func (m *Model) Result() string {
	return m.result
}

// Reset resets the submission state
func (m *Model) Reset() {
	m.submitted = false
	m.cancelled = false
	m.result = ""
}

// filterCompletions filters completions using the completion engine
func (m *Model) filterCompletions() {
	if m.completionEngine == nil || !m.allowIntellisense {
		m.completions = []completion.Completion{}
		m.showCompletions = false
		return
	}

	input := m.input.Value()
	if input == "" {
		input = "_"
	}

	// Build completion context
	context := m.buildCompletionContext()

	// Get filtered completions from engine
	m.completions = m.completionEngine.GetCompletions(input, context)
	m.showCompletions = len(m.completions) > 0
	m.selectedCompletion = 0
}

// buildCompletionContext creates a completion context from current state
func (m *Model) buildCompletionContext() completion.CompletionContext {
	input := m.input.Value()

	// Determine partial token after last dot
	partialToken := ""
	isAfterDot := strings.HasSuffix(input, ".")
	if !isAfterDot && len(input) > 0 {
		if idx := strings.LastIndex(input, "."); idx >= 0 {
			partialToken = input[idx+1:]
		}
	}

	return completion.CompletionContext{
		CurrentNode:          m.currentNode,
		CurrentType:          inferNodeType(m.currentNode),
		CursorPosition:       m.input.Position(),
		ExpressionResult:     nil, // Could be populated from last evaluation
		ExpressionResultType: m.resultType,
		PartialToken:         partialToken,
		IsAfterDot:           isAfterDot,
	}
}

// insertCompletion inserts a completion into the input
func (m *Model) insertCompletion(c completion.Completion) {
	currentValue := m.input.Value()
	hadTrailingDot := strings.HasSuffix(currentValue, ".")

	// Get the text to insert
	text := c.Text

	// For simpler insertion, just set the full completion text
	m.input.SetValue(text)
	m.input.SetCursor(len(text))
}

// insertSuggestion inserts the selected suggestion into the input (legacy - for compatibility)
func (m *Model) insertSuggestion(s Suggestion) {
	// Convert legacy Suggestion to Completion and use insertCompletion
	c := completion.Completion{
		Text:    s.Name,
		Display: s.Name,
	}
	m.insertCompletion(c)
}

// getKeysFromNode extracts available keys from a node
func getKeysFromNode(node interface{}) []string {
	switch t := node.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(t))
		for k := range t {
			// Check if key needs bracket notation
			if needsBracketNotation(k) {
				keys = append(keys, fmt.Sprintf(`["%s"]`, k))
			} else {
				keys = append(keys, k)
			}
		}
		return keys
	case []interface{}:
		indices := make([]string, len(t))
		for i := range t {
			indices[i] = fmt.Sprintf("[%d]", i)
		}
		return indices
	default:
		return []string{}
	}
}

// needsBracketNotation checks if a key needs bracket notation
func needsBracketNotation(key string) bool {
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

func inferNodeType(node interface{}) string {
	if node == nil {
		return "null"
	}
	switch node.(type) {
	case map[string]interface{}:
		return "map"
	case []interface{}:
		return "list"
	case string:
		return "string"
	case bool:
		return "bool"
	case float64:
		return "double"
	case int, int64:
		return "int"
	case uint, uint64:
		return "uint"
	default:
		return "unknown"
	}
}

// isSuggestionCompatibleWithNode checks if a CEL function is compatible with the node type (legacy)
func isSuggestionCompatibleWithNode(s Suggestion, node interface{}) bool {
	if !s.IsFunction {
		return true
	}

	if node == nil {
		return true
	}

	nodeType := "any"
	switch node.(type) {
	case map[string]interface{}:
		nodeType = "map"
	case []interface{}:
		nodeType = "list"
	case string:
		nodeType = "string"
	}

	desc := strings.ToLower(s.Description)
	switch nodeType {
	case "map", "list":
		return true
	case "string":
		return strings.Contains(desc, "string")
	default:
		return !strings.Contains(desc, "string")
	}
}
