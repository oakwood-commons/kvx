package tui

import (
	"strings"

	"github.com/oakwood-commons/kvx/internal/ui"
)

// Config holds host-provided settings for running the TUI.
type Config struct {
	AppName            string
	Width              int
	Height             int
	NoColor            bool
	HideFooter         bool // Hide the footer bar (for non-interactive display)
	DebugEnabled       bool
	DebugSink          func(string)
	Theme              ui.Theme
	ThemeName          string // Alternative to Theme: set a built-in theme by name (dark, warm, cool)
	ExpressionProvider ExpressionProvider
	Menu               *ui.MenuConfig
	InfoPopup          *ui.InfoPopupConfig
	AllowEditInput     *bool
	AllowFilter        *bool
	AllowSuggestions   *bool
	AllowIntellisense  *bool
	InitialExpr        string
	StartKeys          []string
	HelpAboutTitle     string
	HelpAboutLines     []string
	// UI text customization
	KeyHeader                  string              // Table header for key column (default: "KEY")
	ValueHeader                string              // Table header for value column (default: "VALUE")
	InputPromptUnfocused       string              // Input prompt when unfocused (default: "$ ")
	InputPromptFocused         string              // Input prompt when focused (default: "❯ ")
	InputPlaceholder           string              // Input placeholder text
	ExprModeEntryHelp          string              // Help text shown when entering expression mode (default: "Tab: complete | Type: show help | Enter: evaluate")
	FunctionHelpOverrides      map[string]string   // Custom help text for specific functions (e.g., {"filter": "Custom help for filter"})
	HelpNavigationDescriptions map[string]string   // Custom help navigation descriptions (keys: "navigate_up_down", "navigate_back_forward", "go_to_key", "cycle_suggestions", "keys_cel_functions", "array_indices", "quit")
	AllowDecode                *bool               // Whether Enter/Right can decode serialized scalars (default: true)
	AutoDecode                 string              // Auto-decode mode: "" (manual only), "lazy" (on navigate), "eager" (at load)
	DisplaySchema              *DisplaySchema      // Optional display schema for rich TUI rendering (list/detail/status views)
	KeyMode                    string              // Keybinding mode: "vim" (default), "emacs", or "function"
	Done                       <-chan StatusResult // Optional channel for async completion in status view mode
}

// DefaultConfig returns a baseline TUI config with the same defaults as the CLI.
func DefaultConfig() Config {
	embedded, err := ui.EmbeddedDefaultConfig()
	menu := ui.DefaultMenuConfig()
	allowEdit := true
	allowFilter := true
	allowSuggestions := true
	allowIntel := true
	allowDecode := true
	if err == nil {
		if embedded.Features.AllowEditInput != nil {
			allowEdit = *embedded.Features.AllowEditInput
		}
		if embedded.Features.AllowFilter != nil {
			allowFilter = *embedded.Features.AllowFilter
		}
		if embedded.Features.AllowSuggestions != nil {
			allowSuggestions = *embedded.Features.AllowSuggestions
		}
		if embedded.Features.AllowIntellisense != nil {
			allowIntel = *embedded.Features.AllowIntellisense
		}
	}

	appName := "kvx"
	if err == nil {
		if name := strings.TrimSpace(embedded.About.Name); name != "" {
			appName = name
		}
	}

	exprHelp := "Tab: complete | Type: show help | Enter: evaluate"
	if err == nil && strings.TrimSpace(embedded.Help.ExprModeEntry) != "" {
		exprHelp = strings.TrimSpace(embedded.Help.ExprModeEntry)
	}

	return Config{
		AppName:              appName,
		Theme:                ui.DefaultTheme(),
		Menu:                 &menu,
		AllowEditInput:       &allowEdit,
		AllowFilter:          &allowFilter,
		AllowDecode:          &allowDecode,
		AllowSuggestions:     &allowSuggestions,
		AllowIntellisense:    &allowIntel,
		KeyHeader:            "KEY",
		ValueHeader:          "VALUE",
		InputPromptUnfocused: "$ ",
		InputPromptFocused:   "❯ ",
		InputPlaceholder:     "Enter path (e.g. items[0] or items.filter(x, x.available))",
		ExprModeEntryHelp:    exprHelp,
	}
}

// Apply applies the config to the UI globals.
// (Model-scoped fields like AllowEditInput are applied in Run.)
func (c Config) Apply() {
	// ThemeName takes precedence over Theme if both are set
	if c.ThemeName != "" {
		if err := ui.SetThemeByName(c.ThemeName); err != nil {
			// If theme name is invalid, fall back to default theme
			// This allows the TUI to start even with an invalid theme name
			darkTheme, _ := ui.GetTheme("dark")
			ui.SetTheme(darkTheme)
		}
	} else if c.Theme != (ui.Theme{}) {
		ui.SetTheme(c.Theme)
	}
	if c.ExpressionProvider != nil {
		ui.SetExpressionProvider(c.ExpressionProvider)
	}
	if c.Menu != nil {
		ui.SetMenuConfig(*c.Menu)
	}
}
