package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// HelpModel represents the help overlay component
type HelpModel struct {
	Visible                    bool
	NoColor                    bool
	Width                      int
	AllowEditInput             bool
	AboutTitle                 string
	AboutLines                 []string
	AboutAlign                 string
	HelpNavigationDescriptions map[string]string // Custom navigation descriptions
	KeyMode                    KeyMode           // Current keybinding mode
}

// NewHelpModel creates a new help model
func NewHelpModel() HelpModel {
	return HelpModel{
		Width:          92, // Default width
		AllowEditInput: true,
		KeyMode:        DefaultKeyMode,
		AboutTitle:     "",
		AboutLines:     nil,
		AboutAlign:     "right",
	}
}

// Update handles messages for the help component
func (m HelpModel) Update(_ tea.Msg) (HelpModel, tea.Cmd) {
	// Help overlay is passive - it just displays when visible
	// Toggle is handled by parent model
	return m, nil
}

// navigationHelpRows returns the navigation keybinding rows based on the key mode.
func navigationHelpRows(keyMode KeyMode, descs map[string]string) [][]string {
	var rows [][]string
	switch keyMode {
	case KeyModeVim:
		// Show vim-style keys only (no function key references)
		rows = [][]string{
			{"j/k", descs["navigate_up_down"]},
			{"h/l", descs["navigate_back_forward"]},
			{"/", "search/filter"},
			{"n/N", "next/prev match"},
			{"gg/G", "go to top/bottom"},
			{":", "expression mode"},
			{"y", "copy path"},
			{"?", "toggle help"},
			{"q", descs["quit"]},
		}
	case KeyModeEmacs:
		// Show emacs-style keys only (no function key references)
		rows = [][]string{
			{"C-n/C-p", descs["navigate_up_down"]},
			{"C-b/C-f", descs["navigate_back_forward"]},
			{"C-s", "search/filter"},
			{"C-r", "prev match"},
			{"M-</M->", "go to top/bottom"},
			{"M-x", "expression mode"},
			{"M-w", "copy path"},
			{"F1", "toggle help"},
			{"C-g", "cancel/clear"},
			{"C-q", descs["quit"]},
		}
	case KeyModeFunction:
		// Show arrow keys only - function keys are in the Keys section
		rows = [][]string{
			{"↑/↓", descs["navigate_up_down"]},
			{"←/→", descs["navigate_back_forward"]},
			{"Home/End", "go to top/bottom"},
		}
	}
	return rows
}

// View renders the help overlay if visible
func (m HelpModel) View() string {
	if !m.Visible {
		return ""
	}

	menu := CurrentMenuConfig()
	helpRows := [][]string{}
	// Only show function key menu items in function mode
	// In vim/emacs modes, these are covered in the navigation section
	if m.KeyMode == KeyModeFunction {
		for _, kv := range MenuItems(menu) {
			if kv.Item.Action == "expr_toggle" && !m.AllowEditInput {
				continue
			}
			if !kv.Item.Enabled || kv.Item.Label == "" {
				continue
			}
			desc := kv.Item.HelpText
			if desc == "" {
				desc = kv.Item.Label
			}
			helpRows = append(helpRows, []string{kv.Key, desc})
		}
	}
	// Use configurable navigation descriptions with defaults
	defaultDescs := map[string]string{
		"navigate_up_down":      "navigate up/down",
		"navigate_back_forward": "navigate back/forward",
		"go_to_key":             "go to key (resets context)",
		"cycle_suggestions":     "cycle suggestions",
		"keys_cel_functions":    "keys + CEL functions",
		"array_indices":         "array indices",
		"quit":                  "quit",
	}
	// Merge with custom descriptions if provided
	descs := defaultDescs
	if m.HelpNavigationDescriptions != nil {
		descs = make(map[string]string)
		for k, v := range defaultDescs {
			descs[k] = v
		}
		for k, v := range m.HelpNavigationDescriptions {
			descs[k] = v
		}
	}
	// Show keybindings based on current key mode
	navRows := navigationHelpRows(m.KeyMode, descs)
	helpRows = append(helpRows, navRows...)

	// Expression mode keybindings (separate section)
	exprRows := [][]string{
		{"Enter", descs["go_to_key"]},
		{"Tab / Shift+Tab", descs["cycle_suggestions"]},
		{"Ctrl+C", descs["quit"]},
	}

	// Mode switch hint
	modeHint := keyModeSwitchHint(m.KeyMode)

	leftStyle := lipgloss.NewStyle().PaddingLeft(1)
	rightStyle := lipgloss.NewStyle()
	boxStyle := lipgloss.NewStyle()
	aboutStyle := rightStyle
	if !m.NoColor {
		th := CurrentTheme()
		leftStyle = leftStyle.Foreground(th.HelpKey).Bold(true)
		rightStyle = rightStyle.Foreground(th.HelpValue)
		aboutStyle = aboutStyle.Foreground(th.HelpValue)
		boxStyle = boxStyle.Border(borderForTheme(th)).PaddingLeft(1).PaddingRight(1).AlignVertical(lipgloss.Top)
	} else {
		// In no-color mode still highlight key labels with true black on white
		leftStyle = leftStyle.Foreground(lipgloss.Color("#000000")).Background(lipgloss.Color("#ffffff")).Bold(true)
		boxStyle = boxStyle.Border(borderForTheme(CurrentTheme())).PaddingLeft(1).PaddingRight(1).AlignVertical(lipgloss.Top)
	}

	lines := []string{}

	// Optional About section at the top
	if len(m.AboutLines) > 0 {
		alignment := strings.ToLower(m.AboutAlign)
		switch alignment {
		case "center", "middle":
			aboutStyle = aboutStyle.Align(lipgloss.Center)
		case "left":
			aboutStyle = aboutStyle.Align(lipgloss.Left)
		default:
			aboutStyle = aboutStyle.Align(lipgloss.Right)
		}
		if m.Width > 4 {
			aboutStyle = aboutStyle.Width(m.Width - 4)
		}
		for _, l := range m.AboutLines {
			lines = append(lines, aboutStyle.Render(l))
		}
		lines = append(lines, "")
	}

	// Render menu items (Keys section) - only in function mode
	menuItemCount := len(helpRows) - len(navRows)
	if menuItemCount > 0 {
		for i := 0; i < menuItemCount; i++ {
			row := helpRows[i]
			key := leftStyle.Render(fmt.Sprintf("%-16s", row[0]))
			val := rightStyle.Render(row[1])
			lines = append(lines, key+" "+val)
		}
		lines = append(lines, "")
	}

	// Navigation section
	for i := menuItemCount; i < len(helpRows); i++ {
		row := helpRows[i]
		key := leftStyle.Render(fmt.Sprintf("%-16s", row[0]))
		val := rightStyle.Render(row[1])
		lines = append(lines, key+" "+val)
	}

	// Expressions section
	lines = append(lines, "")
	for _, row := range exprRows {
		key := leftStyle.Render(fmt.Sprintf("%-16s", row[0]))
		val := rightStyle.Render(row[1])
		lines = append(lines, key+" "+val)
	}

	// Mode switch hint at the bottom
	lines = append(lines, "")
	lines = append(lines, rightStyle.Render(modeHint))

	content := strings.Join(lines, "\n")
	box := boxStyle.Render(content)
	// Constrain width a bit so we do not overflow narrow terminals
	if m.Width > 0 && len(box) > m.Width {
		box = boxStyle.Width(m.Width - 2).Render(content)
	}

	return box + "\n"
}

// GenerateHelpText generates the formatted help menu text as a plain string.
// This is used to populate the config struct for Go templating.
func GenerateHelpText(menu MenuConfig, allowEditInput bool, navDescs map[string]string, keyMode KeyMode) string {
	th := CurrentTheme()
	keyColor := th.HelpKey
	valColor := th.HelpValue
	headingColor := th.HeaderFG
	if keyColor == nil {
		keyColor = lipgloss.Color("12")
	}
	if valColor == nil {
		valColor = lipgloss.Color("250")
	}
	if headingColor == nil {
		headingColor = lipgloss.Color("14")
	}
	keyStyle := lipgloss.NewStyle().Foreground(keyColor)
	valStyle := lipgloss.NewStyle().Foreground(valColor)
	headingStyle := lipgloss.NewStyle().Foreground(headingColor).Bold(true)

	helpRows := [][]string{}
	// Only show function key menu items in function mode
	// In vim/emacs modes, these are covered in the navigation section
	if keyMode == KeyModeFunction {
		for _, kv := range MenuItems(menu) {
			if kv.Item.Action == "expr_toggle" && !allowEditInput {
				continue
			}
			if !kv.Item.Enabled || kv.Item.Label == "" {
				continue
			}
			desc := kv.Item.HelpText
			if desc == "" {
				desc = kv.Item.Label
			}
			helpRows = append(helpRows, []string{kv.Key, desc})
		}
	}
	// Use configurable navigation descriptions with defaults
	defaultDescs := map[string]string{
		"navigate_up_down":      "navigate up/down",
		"navigate_back_forward": "navigate back/forward",
		"go_to_key":             "go to key (resets context)",
		"cycle_suggestions":     "cycle suggestions",
		"keys_cel_functions":    "keys + CEL functions",
		"array_indices":         "array indices",
		"quit":                  "quit",
	}
	// Merge with custom descriptions if provided
	descs := defaultDescs
	if navDescs != nil {
		descs = make(map[string]string)
		for k, v := range defaultDescs {
			descs[k] = v
		}
		for k, v := range navDescs {
			descs[k] = v
		}
	}

	// Add navigation keybindings based on key mode
	navRows := navigationHelpRows(keyMode, descs)
	helpRows = append(helpRows, navRows...)

	// Expression mode keybindings (separate section)
	exprRows := [][]string{
		{"Enter", descs["go_to_key"]},
		{"Tab / Shift+Tab", descs["cycle_suggestions"]},
		{".", descs["keys_cel_functions"]},
		{"[", descs["array_indices"]},
	}

	lines := []string{}
	menuItemCount := len(helpRows) - len(navRows)

	// Only show "Keys" section if we have menu items (function mode)
	if menuItemCount > 0 {
		lines = append(lines, headingStyle.Render("Keys"))
		for i := 0; i < menuItemCount; i++ {
			row := helpRows[i]
			key := fmt.Sprintf("%-16s", row[0])
			lines = append(lines, fmt.Sprintf("  %s %s", keyStyle.Render(key), valStyle.Render(row[1])))
		}
		lines = append(lines, "")
	}

	lines = append(lines, headingStyle.Render("Navigation"))
	for i := menuItemCount; i < len(helpRows); i++ {
		row := helpRows[i]
		key := fmt.Sprintf("%-16s", row[0])
		lines = append(lines, fmt.Sprintf("  %s %s", keyStyle.Render(key), valStyle.Render(row[1])))
	}

	lines = append(lines, "")
	lines = append(lines, headingStyle.Render("Expressions"))
	for _, row := range exprRows {
		key := fmt.Sprintf("%-16s", row[0])
		lines = append(lines, fmt.Sprintf("  %s %s", keyStyle.Render(key), valStyle.Render(row[1])))
	}
	lines = append(lines, fmt.Sprintf("  %s %s", keyStyle.Render(fmt.Sprintf("%-16s", "Ctrl+C")), valStyle.Render(descs["quit"])))

	// Mode switch hint at the bottom
	lines = append(lines, "")
	hint := keyModeSwitchHint(keyMode)
	lines = append(lines, valStyle.Render(hint))

	return strings.Join(lines, "\n")
}

// keyModeSwitchHint returns a footer hint showing the current mode and how to switch.
func keyModeSwitchHint(mode KeyMode) string {
	var current string
	switch mode {
	case KeyModeVim:
		current = "vim"
	case KeyModeEmacs:
		current = "emacs"
	case KeyModeFunction:
		current = "function"
	default:
		current = string(mode)
	}
	return fmt.Sprintf("Mode: %s  (switch with --keymap vim|emacs|function)", current)
}

// SetWidth sets the width of the help overlay
func (m *HelpModel) SetWidth(width int) {
	m.Width = width
}

// SetVisible sets the visibility of the help overlay
func (m *HelpModel) SetVisible(visible bool) {
	m.Visible = visible
}
