package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oakwood-commons/kvx/internal/ui"
)

func TestConfigLoaderLoadMergedConfigDefaults(t *testing.T) {
	cfg, err := loadMergedConfig("")
	require.NoError(t, err)
	require.NotEmpty(t, cfg.Themes)
	require.NotEmpty(t, strings.TrimSpace(cfg.About.Name))
	require.NotEmpty(t, strings.TrimSpace(cfg.About.Version))
	require.NotEmpty(t, strings.TrimSpace(cfg.HelpMenu.Text))
	require.NotEmpty(t, strings.TrimSpace(cfg.Theme.Default))
}

func TestConfigLoaderLoadMergedConfigNestedOverrides(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	configYAML := `app:
  about:
    name: custom-kvx
  debug:
    max_events: 321
ui:
  theme:
    default: midnight
  features:
    allow_edit_input: false
  themes:
    midnight:
      key_color: "#00ff00"
  menu:
    f1:
      label: Support
      popup:
        text: Custom help text
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(configYAML), 0o600))

	cfg, err := loadMergedConfig(cfgPath)
	require.NoError(t, err)
	require.Equal(t, "custom-kvx", cfg.About.Name)
	require.Equal(t, "midnight", cfg.Theme.Default)
	require.NotNil(t, cfg.Features.AllowEditInput)
	require.False(t, *cfg.Features.AllowEditInput)
	require.Contains(t, cfg.Themes, "midnight")
	require.Equal(t, ui.ColorValue("#00ff00"), cfg.Themes["midnight"].KeyColor)
	require.NotEmpty(t, strings.TrimSpace(cfg.HelpMenu.Text))
	// Help popup text should pass through templating without being cleared.
	require.Equal(t, "Custom help text", strings.TrimSpace(cfg.Menu.F1.Popup.Text))
}

func TestSanitizeConfigClearsDynamicFields(t *testing.T) {
	cfg, err := loadMergedConfig("")
	require.NoError(t, err)

	sanitized := sanitizeConfig(cfg)

	require.Empty(t, sanitized.App.About.Version)
	require.Empty(t, sanitized.App.About.GoVersion)
	require.Empty(t, sanitized.App.About.BuildOS)
	require.Empty(t, sanitized.App.About.BuildArch)
	require.Empty(t, sanitized.App.About.GitCommit)
	require.Empty(t, sanitized.App.Help.Text)
}

func TestAddConfigCommentsAddsPopupHints(t *testing.T) {
	raw := "ui:\n  popup:\n    info_popup:\n      enabled: true\n"
	annotated := addConfigComments(raw)

	require.Contains(t, annotated, "Popup options: enabled (bool)")
	require.Contains(t, annotated, "# show on startup")
}

func TestConfigLoaderUserOverrideMergesWithDefaults(t *testing.T) {
	// Load defaults to know what we start with
	defaults, err := loadMergedConfig("")
	require.NoError(t, err)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	// User config: override one function_example, add a custom one, change a theme color
	configYAML := `ui:
  help:
    cel:
      function_examples:
        filter:
          description: "Custom filter description"
          examples:
            - "_.items.filter(x, x.custom)"
        myCustomFunc:
          description: "A user-defined function"
          examples:
            - "myCustomFunc('test')"
  themes:
    midnight:
      key_color: "#abcdef"
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(configYAML), 0o600))

	cfg, err := loadMergedConfig(cfgPath)
	require.NoError(t, err)

	// User override replaces the default filter entry
	require.Contains(t, cfg.Help.CEL.FunctionExamples, "filter")
	require.Equal(t, "Custom filter description", cfg.Help.CEL.FunctionExamples["filter"].Description)
	require.Equal(t, []string{"_.items.filter(x, x.custom)"}, cfg.Help.CEL.FunctionExamples["filter"].Examples)

	// User addition is present
	require.Contains(t, cfg.Help.CEL.FunctionExamples, "myCustomFunc")

	// Default entries that the user did NOT override are still present
	require.Contains(t, cfg.Help.CEL.FunctionExamples, "map", "default 'map' should survive user override of 'filter'")
	require.Contains(t, cfg.Help.CEL.FunctionExamples, "size", "default 'size' should survive user override")

	// Theme override applied
	require.Equal(t, ui.ColorValue("#abcdef"), cfg.Themes["midnight"].KeyColor)

	// Other built-in themes still present from defaults
	require.True(t, len(cfg.Themes) >= len(defaults.Themes), "user override should not remove other themes")

	// Default theme name still set (not cleared by partial override)
	require.NotEmpty(t, cfg.Theme.Default)
}

func TestProcessTemplateString_NoTemplate(t *testing.T) {
	result := processTemplateString("plain text", nil)
	assert.Equal(t, "plain text", result)
}

func TestProcessTemplateString_WithTemplate(t *testing.T) {
	data := map[string]interface{}{
		"version": "1.0.0",
		"name":    "kvx",
	}
	result := processTemplateString("{{.name}} v{{.version}}", data)
	assert.Equal(t, "kvx v1.0.0", result)
}

func TestProcessTemplateString_InvalidTemplate(t *testing.T) {
	result := processTemplateString("{{invalid template", nil)
	assert.Equal(t, "{{invalid template", result)
}

func TestProcessTemplateString_MissingKey(t *testing.T) {
	data := map[string]interface{}{"name": "kvx"}
	result := processTemplateString("{{.name}} v{{.missing}}", data)
	// Go templates produce empty string for missing keys by default
	assert.Contains(t, result, "kvx")
}

func TestDefaultThemeName_Default(t *testing.T) {
	cfg := ui.ThemeConfigFile{}
	assert.Equal(t, "dark", defaultThemeName(cfg))
}

func TestDefaultThemeName_ThemeDefault(t *testing.T) {
	cfg := ui.ThemeConfigFile{}
	cfg.Theme.Default = "midnight"
	assert.Equal(t, "midnight", defaultThemeName(cfg))
}

func TestDefaultThemeName_Legacy(t *testing.T) {
	cfg := ui.ThemeConfigFile{DefaultTheme: "light"}
	assert.Equal(t, "light", defaultThemeName(cfg))
}

func TestDefaultThemeName_ThemeDefaultOverridesLegacy(t *testing.T) {
	cfg := ui.ThemeConfigFile{DefaultTheme: "light"}
	cfg.Theme.Default = "midnight"
	assert.Equal(t, "midnight", defaultThemeName(cfg))
}

func TestDefaultThemeName_WhitespaceOnly(t *testing.T) {
	cfg := ui.ThemeConfigFile{}
	cfg.Theme.Default = "   "
	cfg.DefaultTheme = "  "
	assert.Equal(t, "dark", defaultThemeName(cfg))
}

func TestGetAllAvailableThemes_BuiltIn(t *testing.T) {
	themes := getAllAvailableThemes(nil)
	assert.NotEmpty(t, themes)
	// Built-in themes should include "dark"
	assert.Contains(t, themes, "dark")
}

func TestGetAllAvailableThemes_WithCustom(t *testing.T) {
	cfg := &ui.ThemeConfigFile{
		Themes: map[string]ui.ThemeConfig{
			"custom-theme": {KeyColor: "#ff0000"},
		},
	}
	themes := getAllAvailableThemes(cfg)
	assert.Contains(t, themes, "custom-theme")
	assert.Contains(t, themes, "dark")
}

func TestGetAllAvailableThemes_Sorted(t *testing.T) {
	themes := getAllAvailableThemes(nil)
	for i := 1; i < len(themes); i++ {
		assert.LessOrEqual(t, themes[i-1], themes[i], "themes should be sorted")
	}
}

func TestMergeThemeConfig_OverrideColors(t *testing.T) {
	base := ui.ThemeConfig{KeyColor: "14", ValueColor: "248"}
	override := ui.ThemeConfig{KeyColor: "#ff0000"}
	result := mergeThemeConfig(base, override)
	assert.Equal(t, ui.ColorValue("#ff0000"), result.KeyColor)
	assert.Equal(t, ui.ColorValue("248"), result.ValueColor)
}

func TestMergeThemeConfig_OverrideBorderStyle(t *testing.T) {
	base := ui.ThemeConfig{BorderStyle: "rounded"}
	override := ui.ThemeConfig{BorderStyle: "double"}
	result := mergeThemeConfig(base, override)
	assert.Equal(t, "double", result.BorderStyle)
}

func TestMergeThemeConfig_EmptyOverride(t *testing.T) {
	base := ui.ThemeConfig{KeyColor: "14", ValueColor: "248", BorderStyle: "rounded"}
	override := ui.ThemeConfig{}
	result := mergeThemeConfig(base, override)
	assert.Equal(t, base, result)
}

func TestMergeMenuConfig_OverrideLabel(t *testing.T) {
	base := ui.MenuConfigYAML{
		Help:   ui.MenuItemConfig{Label: "Help"},
		Search: ui.MenuItemConfig{Label: "Search"},
	}
	override := ui.MenuConfigYAML{
		Help: ui.MenuItemConfig{Label: "Info"},
	}
	result := mergeMenuConfig(base, override)
	assert.Equal(t, "Info", result.Help.Label)
	assert.Equal(t, "Search", result.Search.Label)
}

func TestMergeMenuConfig_EmptyOverride(t *testing.T) {
	base := ui.MenuConfigYAML{
		Help: ui.MenuItemConfig{Label: "Help", HelpText: "Show help"},
	}
	override := ui.MenuConfigYAML{}
	result := mergeMenuConfig(base, override)
	assert.Equal(t, "Help", result.Help.Label)
	assert.Equal(t, "Show help", result.Help.HelpText)
}
