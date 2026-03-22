package ui

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestSetThemeByName(t *testing.T) {
	original := loadedThemes
	origCurrent := currentTheme
	t.Cleanup(func() {
		loadedThemes = original
		SetTheme(origCurrent)
	})

	// No themes loaded
	loadedThemes = map[string]Theme{}
	err := SetThemeByName("dark")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no themes loaded")

	// Load a theme
	loadedThemes = map[string]Theme{
		"custom": fallbackDefaultTheme(),
	}
	err = SetThemeByName("custom")
	assert.NoError(t, err)

	// Unknown theme
	err = SetThemeByName("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown theme")
}

func TestGetAvailableThemeNames(t *testing.T) {
	original := loadedThemes
	t.Cleanup(func() { loadedThemes = original })

	loadedThemes = map[string]Theme{}
	assert.Equal(t, "(none)", getAvailableThemeNames())

	loadedThemes = map[string]Theme{
		"dark":  fallbackDefaultTheme(),
		"light": fallbackDefaultTheme(),
	}
	names := getAvailableThemeNames()
	assert.Contains(t, names, "dark")
	assert.Contains(t, names, "light")
}

func TestGetTheme(t *testing.T) {
	original := loadedThemes
	origPresets := ThemePresets
	t.Cleanup(func() { loadedThemes = original; ThemePresets = origPresets })

	loadedThemes = map[string]Theme{
		"test": fallbackDefaultTheme(),
	}

	th, ok := GetTheme("test")
	assert.True(t, ok)
	assert.NotNil(t, th.KeyColor)

	_, ok = GetTheme("nonexistent")
	assert.False(t, ok)
}

func TestGetAvailableThemes(t *testing.T) {
	original := loadedThemes
	origPresets := ThemePresets
	t.Cleanup(func() { loadedThemes = original; ThemePresets = origPresets })

	loadedThemes = map[string]Theme{"dark": fallbackDefaultTheme()}
	ThemePresets = map[string]Theme{"preset1": fallbackDefaultTheme()}

	themes := GetAvailableThemes()
	assert.Contains(t, themes, "dark")
	assert.Contains(t, themes, "preset1")
}

func TestThemeFromConfig(t *testing.T) {
	cfg := ThemeConfig{
		KeyColor:   "81",
		ValueColor: "246",
	}
	theme := ThemeFromConfig(cfg)
	assert.NotNil(t, theme.KeyColor)
}

func TestColorToColorValue(t *testing.T) {
	assert.Equal(t, ColorValue(""), colorToColorValue(nil))
	c := lipgloss.Color("81")
	result := colorToColorValue(c)
	assert.NotEmpty(t, result)
}

func TestThemeConfigFromTheme(t *testing.T) {
	th := fallbackDefaultTheme()
	cfg := ThemeConfigFromTheme(th)
	assert.NotEmpty(t, string(cfg.KeyColor))
	assert.NotEmpty(t, string(cfg.ValueColor))
}

func TestColorValueMarshalYAML(t *testing.T) {
	// Empty
	v, err := ColorValue("").MarshalYAML()
	assert.NoError(t, err)
	assert.Equal(t, "", v)

	// Numeric - returns a *yaml.Node with int tag
	v, err = ColorValue("81").MarshalYAML()
	assert.NoError(t, err)
	node, ok := v.(*yaml.Node)
	assert.True(t, ok)
	assert.Equal(t, "81", node.Value)
	assert.Equal(t, "!!int", node.Tag)

	// String
	v, err = ColorValue("#ff0000").MarshalYAML()
	assert.NoError(t, err)
	assert.Equal(t, "#ff0000", v)
}

func TestInitializeThemes(t *testing.T) {
	original := loadedThemes
	origPresets := ThemePresets
	origCurrent := currentTheme
	t.Cleanup(func() {
		loadedThemes = original
		ThemePresets = origPresets
		SetTheme(origCurrent)
	})

	// Nil config
	err := InitializeThemes(nil)
	assert.Error(t, err)

	// Empty themes
	err = InitializeThemes(&ThemeConfigFile{})
	assert.Error(t, err)

	// Valid config
	cfg := &ThemeConfigFile{
		Themes: map[string]ThemeConfig{
			"custom": {KeyColor: "81"},
		},
	}
	err = InitializeThemes(cfg)
	assert.NoError(t, err)
	assert.Contains(t, loadedThemes, "custom")
	assert.Contains(t, loadedThemes, "dark") // auto-added fallback
}

func TestBorderStyleForTheme(t *testing.T) {
	th := fallbackDefaultTheme()
	style := borderStyleForTheme(th)
	// Just ensure it doesn't panic and returns a valid style
	_ = style.Render("test")
}
