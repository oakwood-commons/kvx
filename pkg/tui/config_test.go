package tui

import (
	"testing"

	ui "github.com/oakwood-commons/kvx/internal/ui"
)

//nolint:gochecknoinits // test helper to seed theme presets once
func init() {
	// Initialize themes with basic configuration for testing
	// This populates ui.ThemePresets for tests that reference it
	cfg := &ui.ThemeConfigFile{
		Themes: map[string]ui.ThemeConfig{
			"dark": {
				KeyColor:       "14",
				ValueColor:     "248",
				HeaderFG:       "15",
				HeaderBG:       "236",
				SelectedFG:     "16",
				SelectedBG:     "33",
				SeparatorColor: "240",
				InputBG:        "235",
				InputFG:        "255",
				StatusColor:    "248",
				StatusError:    "9",
				StatusSuccess:  "10",
				DebugColor:     "8",
				FooterFG:       "15",
				FooterBG:       "236",
				HelpKey:        "11",
				HelpValue:      "248",
			},
			"warm": {
				KeyColor:       "215",
				ValueColor:     "248",
				HeaderFG:       "15",
				HeaderBG:       "94",
				SelectedFG:     "16",
				SelectedBG:     "130",
				SeparatorColor: "94",
				InputBG:        "235",
				InputFG:        "255",
				StatusColor:    "248",
				StatusError:    "9",
				StatusSuccess:  "10",
				DebugColor:     "8",
				FooterFG:       "15",
				FooterBG:       "94",
				HelpKey:        "215",
				HelpValue:      "248",
			},
			"cool": {
				KeyColor:       "39",
				ValueColor:     "248",
				HeaderFG:       "15",
				HeaderBG:       "24",
				SelectedFG:     "16",
				SelectedBG:     "32",
				SeparatorColor: "24",
				InputBG:        "235",
				InputFG:        "255",
				StatusColor:    "248",
				StatusError:    "9",
				StatusSuccess:  "10",
				DebugColor:     "8",
				FooterFG:       "15",
				FooterBG:       "24",
				HelpKey:        "39",
				HelpValue:      "248",
			},
		},
	}
	_ = ui.InitializeThemes(cfg)
}

// TestConfig_Apply_ThemeName tests that ThemeName works correctly
func TestConfig_Apply_ThemeName(t *testing.T) {
	orig := ui.CurrentTheme()
	defer ui.SetTheme(orig)

	// Test valid theme names
	validThemes := []string{"dark", "warm", "cool"}
	for _, name := range validThemes {
		cfg := Config{
			ThemeName: name,
		}
		cfg.Apply()

		current := ui.CurrentTheme()
		expected, _ := ui.GetTheme(name)
		if current.KeyColor != expected.KeyColor {
			t.Errorf("ThemeName %q: KeyColor mismatch, got %q, expected %q", name, current.KeyColor, expected.KeyColor)
		}
		if current.HeaderBG != expected.HeaderBG {
			t.Errorf("ThemeName %q: HeaderBG mismatch, got %q, expected %q", name, current.HeaderBG, expected.HeaderBG)
		}
	}

	// Test invalid theme name (should fall back to dark)
	cfg := Config{
		ThemeName: "invalid_theme",
	}
	cfg.Apply()

	current := ui.CurrentTheme()
	expected, _ := ui.GetTheme("dark")
	if current.KeyColor != expected.KeyColor {
		t.Errorf("Invalid ThemeName should fall back to dark, got KeyColor %q, expected %q", current.KeyColor, expected.KeyColor)
	}
}

// TestConfig_Apply_Theme tests that Theme field works correctly
func TestConfig_Apply_Theme(t *testing.T) {
	orig := ui.CurrentTheme()
	defer ui.SetTheme(orig)

	warmTheme, _ := ui.GetTheme("warm")
	customTheme := ui.Theme{
		KeyColor:   warmTheme.KeyColor,
		ValueColor: warmTheme.ValueColor,
		HeaderFG:   warmTheme.HeaderFG,
		HeaderBG:   warmTheme.HeaderBG,
		SelectedFG: warmTheme.SelectedFG,
		SelectedBG: warmTheme.SelectedBG,
	}

	cfg := Config{
		Theme: customTheme,
	}
	cfg.Apply()

	current := ui.CurrentTheme()
	if current.KeyColor != customTheme.KeyColor {
		t.Errorf("Theme: KeyColor mismatch, got %q, expected %q", current.KeyColor, customTheme.KeyColor)
	}
	if current.HeaderBG != customTheme.HeaderBG {
		t.Errorf("Theme: HeaderBG mismatch, got %q, expected %q", current.HeaderBG, customTheme.HeaderBG)
	}
}

// TestConfig_Apply_ThemeNamePrecedence tests that ThemeName takes precedence over Theme
func TestConfig_Apply_ThemeNamePrecedence(t *testing.T) {
	orig := ui.CurrentTheme()
	defer ui.SetTheme(orig)

	warmTheme, _ := ui.GetTheme("warm")
	// Set both ThemeName and Theme - ThemeName should win
	cfg := Config{
		ThemeName: "cool",
		Theme:     warmTheme, // This should be ignored
	}
	cfg.Apply()

	current := ui.CurrentTheme()
	expected, _ := ui.GetTheme("cool")
	if current.KeyColor != expected.KeyColor {
		t.Errorf("ThemeName should take precedence: got KeyColor %q (from cool), expected %q", current.KeyColor, expected.KeyColor)
	}
	if current.HeaderBG != expected.HeaderBG {
		t.Errorf("ThemeName should take precedence: got HeaderBG %q (from cool), expected %q", current.HeaderBG, expected.HeaderBG)
	}
}

// TestConfig_Apply_NoTheme tests that default theme is used when no theme is set
func TestConfig_Apply_NoTheme(t *testing.T) {
	orig := ui.CurrentTheme()
	defer ui.SetTheme(orig)

	warmTheme, _ := ui.GetTheme("warm")
	// Set a non-default theme first
	ui.SetTheme(warmTheme)

	// Apply empty config - should keep current theme (not reset to default)
	cfg := Config{}
	cfg.Apply()

	current := ui.CurrentTheme()
	expected := warmTheme
	if current.KeyColor != expected.KeyColor {
		t.Errorf("Empty config should not change theme: got KeyColor %q, expected %q", current.KeyColor, expected.KeyColor)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.AppName != "kvx" {
		t.Errorf("DefaultConfig AppName = %q, want %q", cfg.AppName, "kvx")
	}
	if cfg.Menu == nil {
		t.Fatal("DefaultConfig Menu is nil")
	}
	if cfg.AllowEditInput == nil || !*cfg.AllowEditInput {
		t.Fatal("DefaultConfig AllowEditInput should be true")
	}
	if cfg.AllowFilter == nil || !*cfg.AllowFilter {
		t.Fatal("DefaultConfig AllowFilter should be true")
	}
	if cfg.AllowSuggestions == nil || !*cfg.AllowSuggestions {
		t.Fatal("DefaultConfig AllowSuggestions should be true")
	}
	if cfg.AllowIntellisense == nil || !*cfg.AllowIntellisense {
		t.Fatal("DefaultConfig AllowIntellisense should be true")
	}
	if cfg.KeyHeader != "KEY" {
		t.Errorf("DefaultConfig KeyHeader = %q, want %q", cfg.KeyHeader, "KEY")
	}
	if cfg.ValueHeader != "VALUE" {
		t.Errorf("DefaultConfig ValueHeader = %q, want %q", cfg.ValueHeader, "VALUE")
	}
	if cfg.InputPromptUnfocused != "$ " {
		t.Errorf("DefaultConfig InputPromptUnfocused = %q, want %q", cfg.InputPromptUnfocused, "$ ")
	}
	if cfg.InputPromptFocused == "" {
		t.Error("DefaultConfig InputPromptFocused should be set")
	}
	if cfg.InputPlaceholder == "" {
		t.Error("DefaultConfig InputPlaceholder should be set")
	}
	if cfg.ExprModeEntryHelp == "" {
		t.Error("DefaultConfig ExprModeEntryHelp should be set")
	}
	if cfg.Theme == (ui.Theme{}) {
		t.Error("DefaultConfig Theme should be non-zero")
	}
}
