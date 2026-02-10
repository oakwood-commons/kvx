package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// DebugModel represents the debug bar component
type DebugModel struct {
	Visible         bool
	NoColor         bool
	Width           int
	LastDebugOutput string // Cached output to prevent flicker
	LastDebugValues string // Hash of debug values to detect changes
}

// NewDebugModel creates a new debug model
func NewDebugModel() DebugModel {
	return DebugModel{
		Width: 92, // Default width
	}
}

// Update handles messages for the debug component
func (m DebugModel) Update(_ tea.Msg) (DebugModel, tea.Cmd) {
	// Debug bar is passive - it just displays when visible
	// Updates are handled by parent model via syncDebug
	return m, nil
}

// View renders the debug bar if visible
func (m DebugModel) View() string {
	if !m.Visible {
		return ""
	}

	// Return cached output if available
	if m.LastDebugOutput != "" {
		return m.LastDebugOutput
	}

	return ""
}

// UpdateDebugInfo updates the debug information and regenerates output if state changed
func (m *DebugModel) UpdateDebugInfo(stateKey string, debugInfo DebugInfo) {
	// Only regenerate if state changed
	if m.LastDebugValues != stateKey {
		debugStyle := lipgloss.NewStyle()
		if !m.NoColor {
			debugStyle = debugStyle.Foreground(CurrentTheme().DebugColor)
		}

		message := fmt.Sprintf("DBG: win=%dx%d th=%d keyW=%d valW=%d | table=%d/%d(all=%d) cursor=%d focus=%v tblFocus=%v cols=%d | search=%v ctx=%v | firstRow=%q | sugg=%d/%d sel=%d show=%v | input=%q path=%q | layout: th=%d sh=%d ph=%d st=%d dh=%d fh=%d bb=%d pad=%d reserve=%v",
			debugInfo.WinWidth, debugInfo.WinHeight, debugInfo.TableHeight,
			debugInfo.KeyColWidth, debugInfo.ValueColWidth,
			debugInfo.ShownRows, debugInfo.TotalRows, debugInfo.AllRowsCount, debugInfo.Cursor,
			debugInfo.InputFocused, debugInfo.TableFocused, debugInfo.ColumnCount,
			debugInfo.AdvancedSearchActive, debugInfo.SearchContextActive,
			truncateDebug(debugInfo.FirstRowPreview),
			debugInfo.FilteredSuggestions, debugInfo.TotalSuggestions,
			debugInfo.SelectedSuggestion, debugInfo.ShowSuggestions,
			truncateDebug(debugInfo.InputValue), truncateDebug(debugInfo.Path),
			debugInfo.LayoutTableHeight, debugInfo.LayoutSuggestionHeight, debugInfo.LayoutPathInputHeight,
			debugInfo.LayoutStatusHeight, debugInfo.LayoutDebugHeight, debugInfo.LayoutFooterHeight,
			debugInfo.LayoutBottomBlockHeight, debugInfo.LayoutTablePad, debugInfo.LayoutReserveSuggestionSpace)

		// Pad to configured width (defaults to 92) to align with other bars
		target := 92
		if m.Width > 0 {
			target = m.Width
		}
		padded := message
		if len(padded) > target {
			padded = padded[:target-3] + "..."
		}
		if len(padded) < target {
			padded += strings.Repeat(" ", target-len(padded))
		}

		m.LastDebugOutput = debugStyle.Render(padded) + "\n"
		m.LastDebugValues = stateKey
	}
}

// DebugInfo contains all the debug information to display
type DebugInfo struct {
	WinWidth             int
	WinHeight            int
	TableHeight          int
	KeyColWidth          int
	ValueColWidth        int
	ShownRows            int
	TotalRows            int
	Cursor               int
	InputFocused         bool
	FilteredSuggestions  int
	TotalSuggestions     int
	SelectedSuggestion   int
	ShowSuggestions      bool
	InputValue           string
	Path                 string
	AdvancedSearchActive bool
	SearchContextActive  bool
	TableFocused         bool
	FirstRowPreview      string // First few chars of first row for debugging
	ColumnCount          int
	AllRowsCount         int
	// Layout information
	LayoutTableHeight            int
	LayoutSuggestionHeight       int
	LayoutPathInputHeight        int
	LayoutStatusHeight           int
	LayoutDebugHeight            int
	LayoutFooterHeight           int
	LayoutBottomBlockHeight      int
	LayoutTablePad               int
	LayoutReserveSuggestionSpace bool
}

// SetWidth sets the width of the debug bar
func (m *DebugModel) SetWidth(width int) {
	m.Width = width
}

// SetVisible sets the visibility of the debug bar
func (m *DebugModel) SetVisible(visible bool) {
	m.Visible = visible
	if !visible {
		// Clear cache when hidden
		m.LastDebugOutput = ""
		m.LastDebugValues = ""
	}
}

func truncateDebug(s string) string {
	return s
}
