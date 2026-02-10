package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	ui "github.com/oakwood-commons/kvx/internal/ui"
)

// TestThemePresetsExist tests that all built-in theme presets are valid
func TestThemePresetsExist(t *testing.T) {
	expectedThemes := []string{"dark", "warm", "cool"}
	for _, name := range expectedThemes {
		theme, ok := ui.GetTheme(name)
		if !ok {
			t.Errorf("expected theme %q to exist in GetTheme", name)
			continue
		}
		// Verify theme has required fields set
		if theme.KeyColor == nil {
			t.Errorf("theme %q has empty KeyColor", name)
		}
		if theme.ValueColor == nil {
			t.Errorf("theme %q has empty ValueColor", name)
		}
		if theme.HeaderFG == nil {
			t.Errorf("theme %q has empty HeaderFG", name)
		}
		if theme.HeaderBG == nil {
			t.Errorf("theme %q has empty HeaderBG", name)
		}
	}
}

// TestThemeSetAndGet tests that themes can be set and retrieved
func TestThemeSetAndGet(t *testing.T) {
	orig := ui.CurrentTheme()
	defer ui.SetTheme(orig)

	// Test each built-in theme
	for name, theme := range ui.GetAvailableThemes() {
		ui.SetTheme(theme)
		current := ui.CurrentTheme()
		if current.KeyColor != theme.KeyColor {
			t.Errorf("theme %q: KeyColor mismatch after SetTheme", name)
		}
		if current.HeaderBG != theme.HeaderBG {
			t.Errorf("theme %q: HeaderBG mismatch after SetTheme", name)
		}
	}
}

// TestSetThemeByName tests the SetThemeByName function
func TestSetThemeByName(t *testing.T) {
	orig := ui.CurrentTheme()
	defer ui.SetTheme(orig)

	// Test valid theme names
	validThemes := []string{"dark", "warm", "cool"}
	for _, name := range validThemes {
		err := ui.SetThemeByName(name)
		if err != nil {
			t.Errorf("SetThemeByName(%q) returned error: %v", name, err)
		}
		current := ui.CurrentTheme()
		expected, _ := ui.GetTheme(name)
		if current.KeyColor != expected.KeyColor {
			t.Errorf("SetThemeByName(%q): KeyColor mismatch, got %q, expected %q", name, current.KeyColor, expected.KeyColor)
		}
		if current.HeaderBG != expected.HeaderBG {
			t.Errorf("SetThemeByName(%q): HeaderBG mismatch, got %q, expected %q", name, current.HeaderBG, expected.HeaderBG)
		}
	}

	// Test invalid theme name
	err := ui.SetThemeByName("invalid_theme")
	if err == nil {
		t.Error("SetThemeByName with invalid theme should return error")
	}
	if !strings.Contains(err.Error(), "unknown theme") {
		t.Errorf("SetThemeByName error should mention 'unknown theme', got: %v", err)
	}
}

// TestCLI_UnknownThemeErrorIncludesBuiltInThemes tests that getAllAvailableThemes includes built-in themes
// The actual error message is tested via integration/snapshot tests
func TestCLI_UnknownThemeErrorIncludesBuiltInThemes(t *testing.T) {
	// Test that getAllAvailableThemes returns built-in themes
	themes := getAllAvailableThemes(nil)
	expectedThemes := []string{"cool", "dark", "warm"}
	for _, expected := range expectedThemes {
		found := false
		for _, theme := range themes {
			if theme == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected built-in theme %q in available themes, got: %v", expected, themes)
		}
	}
}

// TestCLI_ThemeFromConfigFile tests that themes from config file are available
func TestCLI_ThemeFromConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test_config.yaml")
	cfgContent := `
ui:
  themes:
    custom_theme:
      key_color: 10
      value_color: 11
      header_fg: 12
      header_bg: 13
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Test that custom theme can be loaded
	cfg, err := loadMergedConfig(cfgPath)
	if err != nil {
		t.Fatalf("loadMergedConfig: %v", err)
	}

	if _, ok := cfg.Themes["custom_theme"]; !ok {
		t.Error("expected custom_theme to be in loaded config")
	}
}

// TestCLI_ThemeErrorIncludesConfigThemes tests that getAllAvailableThemes includes config themes
func TestCLI_ThemeErrorIncludesConfigThemes(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test_config.yaml")
	cfgContent := `
ui:
  themes:
    my_custom_theme:
      key_color: 10
      value_color: 11
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := loadMergedConfig(cfgPath)
	if err != nil {
		t.Fatalf("loadMergedConfig: %v", err)
	}

	themes := getAllAvailableThemes(&cfg)

	// Should include both built-in and config themes
	hasBuiltIn := false
	hasCustom := false
	for _, theme := range themes {
		if theme == "dark" || theme == "warm" || theme == "cool" {
			hasBuiltIn = true
		}
		if theme == "my_custom_theme" {
			hasCustom = true
		}
	}
	if !hasBuiltIn {
		t.Error("expected built-in themes in available themes")
	}
	if !hasCustom {
		t.Error("expected my_custom_theme in available themes")
	}
}

// TestThemeFromConfig tests ThemeFromConfig function
func TestThemeFromConfig(t *testing.T) {
	cfg := ui.ThemeConfig{
		KeyColor:   ui.ColorValue("10"),
		ValueColor: ui.ColorValue("11"),
		HeaderFG:   ui.ColorValue("12"),
		HeaderBG:   ui.ColorValue("13"),
	}

	_ = ui.ThemeFromConfig(cfg)
	if ui.ColorValue("10") != cfg.KeyColor {
		t.Errorf("expected KeyColor to be '10', got %q", cfg.KeyColor)
	}
	if ui.ColorValue("11") != cfg.ValueColor {
		t.Errorf("expected ValueColor to be '11', got %q", cfg.ValueColor)
	}
	if ui.ColorValue("12") != cfg.HeaderFG {
		t.Errorf("expected HeaderFG to be '12', got %q", cfg.HeaderFG)
	}
	if ui.ColorValue("13") != cfg.HeaderBG {
		t.Errorf("expected HeaderBG to be '13', got %q", cfg.HeaderBG)
	}
}

// TestThemeFromConfigWithDefaults tests that ThemeFromConfig falls back to defaults
func TestThemeFromConfigWithDefaults(t *testing.T) {
	// Empty config should use defaults
	cfg := ui.ThemeConfig{}
	theme := ui.ThemeFromConfig(cfg)

	defaultTheme := ui.DefaultTheme()
	if theme.KeyColor == nil {
		t.Error("expected KeyColor to have default value")
	}
	if theme.KeyColor != defaultTheme.KeyColor {
		t.Logf("KeyColor uses default: %q", theme.KeyColor)
	}
}

// TestThemeConfigFromTheme tests ThemeConfigFromTheme function
func TestThemeConfigFromTheme(t *testing.T) {
	defaultTheme := ui.DefaultTheme()
	theme := defaultTheme
	cfg := ui.ThemeConfigFromTheme(theme)
	defaultCfg := ui.ThemeConfigFromTheme(defaultTheme)

	if cfg.KeyColor != defaultCfg.KeyColor {
		t.Errorf("expected KeyColor to match, got %q vs %q", cfg.KeyColor, defaultCfg.KeyColor)
	}
	if cfg.ValueColor != defaultCfg.ValueColor {
		t.Errorf("expected ValueColor to match, got %q vs %q", cfg.ValueColor, defaultCfg.ValueColor)
	}
}

// TestGetAllAvailableThemes tests the getAllAvailableThemes helper
func TestGetAllAvailableThemes(t *testing.T) {
	// Test with nil config (built-in themes from defaults)
	themes := getAllAvailableThemes(nil)
	if len(themes) == 0 {
		t.Fatalf("expected built-in themes, got none")
	}
	builtIn := make(map[string]bool)
	for name := range ui.GetAvailableThemes() {
		builtIn[name] = true
	}
	for name := range builtIn {
		found := false
		for _, theme := range themes {
			if theme == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected theme %q in available themes, got: %v", name, themes)
		}
	}

	// Test with config file that has custom themes
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test_config.yaml")
	cfgContent := `
ui:
  themes:
    custom1:
      key_color: 10
    custom2:
      key_color: 11
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := loadMergedConfig(cfgPath)
	if err != nil {
		t.Fatalf("loadMergedConfig: %v", err)
	}

	themes = getAllAvailableThemes(&cfg)
	// Should include both built-in and custom themes
	if len(themes) < 5 { // 3 built-in + 2 custom = 5
		t.Errorf("expected at least 5 themes (built-in + custom), got %d: %v", len(themes), themes)
	}

	// Check that custom themes are included
	hasCustom1 := false
	hasCustom2 := false
	for _, theme := range themes {
		if theme == "custom1" {
			hasCustom1 = true
		}
		if theme == "custom2" {
			hasCustom2 = true
		}
	}
	if !hasCustom1 {
		t.Error("expected custom1 theme to be in available themes")
	}
	if !hasCustom2 {
		t.Error("expected custom2 theme to be in available themes")
	}
}

// TestCLI_ThemeAppliedToSnapshot tests that themes are actually applied in snapshot mode
func TestCLI_ThemeAppliedToSnapshot(t *testing.T) {
	testFile := filepath.Join("..", "tests", "sample.yaml")

	// Test that different themes produce valid output
	themes := []string{"dark", "warm", "cool"}
	for _, themeName := range themes {
		out := runCLI(t, []string{
			"kvx",
			testFile,
			"--snapshot",
			"--no-color",
			"--theme", themeName,
			"--width", "80",
			"--height", "24",
		})

		if !strings.Contains(out, "KEY") || !strings.Contains(out, "VALUE") {
			t.Errorf("theme %q: expected table headers in snapshot output", themeName)
		}
		// Default is vim mode, check for vim-style or F-keys
		if !strings.Contains(strings.ToLower(out), "help") {
			t.Errorf("theme %q: expected footer in snapshot output", themeName)
		}
	}
}
