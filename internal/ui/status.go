package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/oakwood-commons/kvx/internal/completion"
)

// StatusModel represents the status bar component
type StatusModel struct {
	ErrMsg                string
	StatusType            string // "error", "success", or ""
	AdvancedSearchActive  bool
	AdvancedSearchQuery   string
	AdvancedSearchResults []SearchResult
	FilterActive          bool
	FilterBuffer          string
	CursorIndex           int                     // Current cursor position (1-based)
	TotalRows             int                     // Total number of rows
	InputFocused          bool                    // Whether in expression mode
	FilteredSuggestions   []string                // Available suggestions (legacy)
	SelectedSuggestion    int                     // Currently selected suggestion index (legacy)
	ShowSuggestions       bool                    // Whether suggestions are available (legacy)
	Completions           []completion.Completion // New completion engine results
	SelectedCompletion    int                     // Currently selected completion index
	ShowCompletions       bool                    // Whether to show completions
	HelpVisible           bool                    // Whether help overlay is visible
	FunctionHelpText      string                  // Help text for the current function being typed or at cursor
	SuggestionSummary     string                  // One-shot summary of functions after typing a trailing dot
	ShowSuggestionSummary bool                    // Whether to render the trailing-dot summary in the status bar
	InputValue            string                  // Current input value to check if it ends with "."
	DecodeHint            string                  // Contextual hint shown when the selected value is decodable
	NoColor               bool
	Width                 int
}

// NewStatusModel creates a new status model
func NewStatusModel() StatusModel {
	return StatusModel{
		Width: 92, // Default width
	}
}

// Update handles messages for the status component
func (m StatusModel) Update(_ tea.Msg) (StatusModel, tea.Cmd) {
	// Status bar is mostly passive - it just displays state
	// No interactive updates needed
	return m, nil
}

// View renders the status bar
func (m StatusModel) View() string {
	// Base styling for the status panel; derive from theme and avoid ANSI when no-color.
	baseStyle := lipgloss.NewStyle()
	if !m.NoColor {
		th := CurrentTheme()
		// v2: FooterBG/FooterFG are now color.Color interface
		if th.FooterBG != nil {
			baseStyle = baseStyle.Background(th.FooterBG)
		}
		if th.FooterFG != nil {
			baseStyle = baseStyle.Foreground(th.FooterFG)
		}
	}
	statusStyle := baseStyle
	message := ""

	switch {
	case m.ErrMsg != "" && m.StatusType == "success":
		statusStyle = statusStyle.Foreground(CurrentTheme().StatusSuccess)
		message = m.ErrMsg
	case m.ErrMsg != "":
		statusStyle = statusStyle.Foreground(CurrentTheme().StatusError)
		message = m.ErrMsg
	case m.HelpVisible:
		// Show help text left-justified, no counter
		statusStyle = statusStyle.Foreground(CurrentTheme().StatusColor)
		message = "Help (F1): Press F1 or Esc to close"
	case m.AdvancedSearchActive:
		statusStyle = statusStyle.Foreground(CurrentTheme().StatusColor)
		switch {
		case m.AdvancedSearchQuery == "":
			message = ""
		case m.TotalRows > 0 && m.CursorIndex > 0:
			message = fmt.Sprintf("%d/%d", m.CursorIndex, m.TotalRows)
		default:
			message = ""
		}
	case m.FilterActive && m.FilterBuffer != "":
		statusStyle = statusStyle.Foreground(CurrentTheme().StatusColor)
		if m.TotalRows > 0 && m.CursorIndex > 0 {
			message = fmt.Sprintf("Filter: '%s' - %d/%d", m.FilterBuffer, m.CursorIndex, m.TotalRows)
		} else {
			message = fmt.Sprintf("Filter: '%s'", m.FilterBuffer)
		}
	case m.InputFocused:
		// In expression mode - prioritize function help text, then suggestions, then default message
		statusStyle = statusStyle.Foreground(CurrentTheme().StatusColor)
		switch {
		case m.FunctionHelpText != "":
			// Show function help text (left-justified)
			message = m.FunctionHelpText
		case m.ShowSuggestionSummary && m.SuggestionSummary != "":
			// Show a short summary of available functions after typing a trailing dot
			message = m.SuggestionSummary
		case m.ShowSuggestions && len(m.FilteredSuggestions) > 0 && m.SelectedSuggestion >= 0 && m.SelectedSuggestion < len(m.FilteredSuggestions) && strings.HasSuffix(m.InputValue, "."):
			// Show the currently selected suggestion (prioritize CEL functions)
			selected := m.FilteredSuggestions[m.SelectedSuggestion]
			// If it's a CEL function, show it; otherwise show the key
			isCELFunctions := strings.Contains(selected, "(") || strings.Contains(selected, " - ")
			switch {
			case isCELFunctions:
				// It's a CEL function - show it
				message = selected
			case m.InputFocused:
				// It's a key, but we're in expression mode - show it if no CEL functions available
				// Check if there are any CEL functions in the list
				hasCELFunctions := false
				for _, s := range m.FilteredSuggestions {
					if strings.Contains(s, "(") || strings.Contains(s, " - ") {
						hasCELFunctions = true
						break
					}
				}
				if !hasCELFunctions {
					message = selected
				} else {
					// Find the first CEL function to show
					for _, s := range m.FilteredSuggestions {
						if strings.Contains(s, "(") || strings.Contains(s, " - ") {
							message = s
							break
						}
					}
				}
			default:
				message = selected
			}
		default:
			message = "Type . to see CEL suggestions"
		}
	default:
		statusStyle = statusStyle.Foreground(CurrentTheme().StatusColor)
		switch {
		case m.DecodeHint != "" && m.TotalRows > 0 && m.CursorIndex > 0:
			message = fmt.Sprintf("%d/%d  %s", m.CursorIndex, m.TotalRows, m.DecodeHint)
		case m.TotalRows > 0 && m.CursorIndex > 0:
			message = fmt.Sprintf("%d/%d", m.CursorIndex, m.TotalRows)
		case m.DecodeHint != "":
			message = m.DecodeHint
		}
	}

	// Pad the status bar to the window width (fallback to 92 if unknown)
	target := 92
	if m.Width > 0 {
		target = m.Width
	}

	// Only truncate non-function-help messages to preserve full examples in function help
	if m.FunctionHelpText == "" && len(message) > target {
		message = message[:target-3] + "..."
	}

	// Left-justify help text, function help, and expression mode suggestions; right-justify everything else
	msgLen := len(message)
	var padded string
	if m.HelpVisible || m.FunctionHelpText != "" || m.InputFocused {
		// Left-justify: pad on the right
		if msgLen < target {
			padded = message + strings.Repeat(" ", target-msgLen)
		} else {
			padded = message
		}
	} else {
		// Right-justify: pad on the left
		if msgLen < target {
			padding := strings.Repeat(" ", target-msgLen)
			padded = padding + message
		} else {
			padded = message
		}
	}

	return statusStyle.Width(target).Render(padded) + "\n"
}

// SetWidth sets the width of the status bar
func (m *StatusModel) SetWidth(width int) {
	m.Width = width
}

// RenderCompletions renders completion results in the status panel.
// This shows filtered suggestions with the selected item highlighted.
func (m StatusModel) RenderCompletions(maxLines int) string {
	if !m.ShowCompletions || len(m.Completions) == 0 {
		return ""
	}

	var lines []string
	maxShow := len(m.Completions)
	if maxShow > maxLines {
		maxShow = maxLines
	}

	th := CurrentTheme()
	for i := 0; i < maxShow; i++ {
		c := m.Completions[i]
		prefix := "  "
		if i == m.SelectedCompletion {
			prefix = "❯ "
		}

		// Format: name - detail
		line := fmt.Sprintf("%s%s", prefix, c.Display)
		if c.Detail != "" {
			line += " - " + c.Detail
		}

		// Highlight selected item
		if !m.NoColor && i == m.SelectedCompletion {
			style := lipgloss.NewStyle().Foreground(th.StatusSuccess).Bold(true)
			line = style.Render(line)
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// RenderCompletionHelp returns the description for the currently selected completion.
// This is shown separately from the completion list to provide detailed help.
func (m StatusModel) RenderCompletionHelp() string {
	if !m.ShowCompletions || len(m.Completions) == 0 {
		return ""
	}

	if m.SelectedCompletion < 0 || m.SelectedCompletion >= len(m.Completions) {
		return ""
	}

	selected := m.Completions[m.SelectedCompletion]
	if selected.Description != "" {
		return selected.Description
	}

	return ""
}
