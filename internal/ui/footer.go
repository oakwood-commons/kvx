package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// FooterModel represents the footer component with function key bindings
type FooterModel struct {
	NoColor          bool
	Width            int
	AllowEditInput   bool
	ExpressionMode   bool    // Whether in expression mode
	ExpressionType   string  // Type of current expression result
	ShowExprTypefoot bool    // Show expression type in footer
	KeyMode          KeyMode // Current keybinding mode
}

// NewFooterModel creates a new footer model
func NewFooterModel() FooterModel {
	return FooterModel{
		Width:          92, // Default width
		AllowEditInput: true,
		KeyMode:        DefaultKeyMode,
	}
}

// Update handles messages for the footer component
func (m FooterModel) Update(_ tea.Msg) (FooterModel, tea.Cmd) {
	// Footer is passive - it just displays key bindings
	// No interactive updates needed
	return m, nil
}

// View renders the footer with function key bindings
func (m FooterModel) View() string {
	// Debug guard: if menu actions all disabled/empty, return empty string
	fkeyStyle := lipgloss.NewStyle()
	if !m.NoColor {
		// Grey background with white text across the whole footer
		fkeyStyle = fkeyStyle.Foreground(lipgloss.Color("15")).Background(lipgloss.Color("240")).Bold(true)
	} else {
		// In no-color mode still highlight keys with true black on white
		fkeyStyle = fkeyStyle.Foreground(lipgloss.Color("#000000")).Background(lipgloss.Color("#ffffff")).Bold(true)
	}

	parts := []string{}

	// Render keybindings dynamically from menu config based on mode
	menu := CurrentMenuConfig()
	actionOrder := []string{"help", "search", "filter", "copy", "expr", "quit"}

	for _, actionName := range actionOrder {
		item, ok := menu.Items[actionName]
		if !ok || !item.Enabled || item.Label == "" {
			continue
		}
		if item.Action == "expr_toggle" && !m.AllowEditInput {
			continue
		}

		// Get the key for the current mode
		var key string
		switch m.KeyMode {
		case KeyModeVim:
			key = item.Keys.Vim
		case KeyModeEmacs:
			key = formatEmacsKey(item.Keys.Emacs)
		case KeyModeFunction:
			key = strings.ToUpper(item.Keys.Function)
		}

		if key != "" {
			parts = append(parts, fkeyStyle.Render(key), item.Label)
		}
	}

	// Fallback to defaults if everything was filtered/disabled
	if len(parts) == 0 {
		defaultMenu := DefaultMenuConfig()
		for _, actionName := range actionOrder {
			item, ok := defaultMenu.Items[actionName]
			if !ok || !item.Enabled || item.Label == "" {
				continue
			}
			if item.Action == "expr_toggle" && !m.AllowEditInput {
				continue
			}
			var key string
			switch m.KeyMode {
			case KeyModeVim:
				key = item.Keys.Vim
			case KeyModeEmacs:
				key = formatEmacsKey(item.Keys.Emacs)
			case KeyModeFunction:
				key = strings.ToUpper(item.Keys.Function)
			}
			if key != "" {
				parts = append(parts, fkeyStyle.Render(key), item.Label)
			}
		}
	}

	// Last resort: hardcoded defaults to avoid an empty footer
	if len(parts) == 0 {
		switch m.KeyMode {
		case KeyModeVim:
			parts = []string{
				fkeyStyle.Render("?"), "help",
				fkeyStyle.Render("/"), "search",
				fkeyStyle.Render("f"), "filter",
				fkeyStyle.Render("y"), "copy",
				fkeyStyle.Render(":"), "expr",
				fkeyStyle.Render("q"), "quit",
			}
		case KeyModeEmacs:
			parts = []string{
				fkeyStyle.Render("F1"), "help",
				fkeyStyle.Render("C-s"), "search",
				fkeyStyle.Render("C-l"), "filter",
				fkeyStyle.Render("M-w"), "copy",
				fkeyStyle.Render("M-x"), "expr",
				fkeyStyle.Render("C-q"), "quit",
			}
		case KeyModeFunction:
			parts = []string{
				fkeyStyle.Render("F1"), "help",
				fkeyStyle.Render("F3"), "search",
				fkeyStyle.Render("F4"), "filter",
				fkeyStyle.Render("F5"), "copy",
				fkeyStyle.Render("F6"), "expr",
				fkeyStyle.Render("F10"), "quit",
			}
		}
	}

	helpLine := strings.Join(parts, " ")

	// Add expression type indicator on the right if in expression mode
	if m.ExpressionMode && m.ShowExprTypefoot && m.ExpressionType != "" {
		typeIndicator := "[type: " + m.ExpressionType + "]"
		typeStyle := lipgloss.NewStyle()
		if !m.NoColor {
			th := CurrentTheme()
			if th.StatusColor != nil {
				typeStyle = typeStyle.Foreground(th.StatusColor)
			}
		}

		// Calculate spacing to right-align the type
		helpLineLen := len(helpLine)
		typeLen := len(typeIndicator)
		if m.Width > helpLineLen+typeLen+2 {
			spacing := m.Width - helpLineLen - typeLen - 2
			helpLine = helpLine + strings.Repeat(" ", spacing) + typeStyle.Render(typeIndicator)
		}
	}

	return helpLine
}

// formatEmacsKey converts internal key format to display format (ctrl+s -> C-s)
func formatEmacsKey(key string) string {
	if key == "" {
		return ""
	}
	// Uppercase F-keys (f1 -> F1)
	if len(key) >= 2 && (key[0] == 'f' || key[0] == 'F') && key[1] >= '0' && key[1] <= '9' {
		return strings.ToUpper(key)
	}
	key = strings.ReplaceAll(key, "ctrl+", "C-")
	key = strings.ReplaceAll(key, "alt+", "M-")
	return key
}

// SetWidth sets the width of the footer (not used for rendering, but kept for consistency)
func (m *FooterModel) SetWidth(width int) {
	m.Width = width
}
