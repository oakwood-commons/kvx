package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	ui "github.com/oakwood-commons/kvx/internal/ui"
)

func TestThemeSelectionError_Error(t *testing.T) {
	err := themeSelectionError{
		Selected:     "nope",
		Available:    []string{"dark", "light"},
		DefaultTheme: "dark",
	}
	msg := err.Error()
	assert.Contains(t, msg, "nope")
	assert.Contains(t, msg, "dark")
	assert.Contains(t, msg, "light")
}

func TestPrintThemeSelectionError_ThemeError(t *testing.T) {
	var buf bytes.Buffer
	err := themeSelectionError{
		Selected:     "bad",
		Available:    []string{"dark"},
		DefaultTheme: "dark",
	}
	printThemeSelectionError(&buf, err)
	assert.Contains(t, buf.String(), "bad")
	assert.Contains(t, buf.String(), "dark")
}

func TestPrintThemeSelectionError_GenericError(t *testing.T) {
	var buf bytes.Buffer
	printThemeSelectionError(&buf, assert.AnError)
	assert.NotEmpty(t, buf.String())
}

func TestApplyFunctionExamples_Empty(t *testing.T) {
	cfg := ui.ThemeConfigFile{}
	// Should not panic with empty config
	applyFunctionExamples(cfg, true)
	applyFunctionExamples(cfg, false)
}

func TestApplyFunctionExamples_WithData(t *testing.T) {
	cfg := ui.ThemeConfigFile{
		Help: ui.HelpConfig{
			CEL: ui.CELHelpConfig{
				FunctionExamples: map[string]ui.FunctionExampleValue{
					"filter": {
						Description: "Filter elements",
						Examples:    []string{"_.items.filter(x, x > 0)"},
					},
				},
			},
		},
	}
	applyFunctionExamples(cfg, false)
}

func TestApplyThemeFromConfig(t *testing.T) {
	cfg := ui.ThemeConfigFile{
		Themes: map[string]ui.ThemeConfig{
			"dark": {},
		},
		Theme: ui.ThemeSelectionConfig{Default: "dark"},
	}

	// Initialize themes first
	_ = ui.InitializeThemes(&cfg)

	err := applyThemeFromConfig(cfg, "", false)
	assert.NoError(t, err)
}

func TestApplyThemeFromConfig_UnknownTheme(t *testing.T) {
	cfg := ui.ThemeConfigFile{
		Themes: map[string]ui.ThemeConfig{
			"dark": {},
		},
		Theme: ui.ThemeSelectionConfig{Default: "dark"},
	}
	_ = ui.InitializeThemes(&cfg)

	err := applyThemeFromConfig(cfg, "nonexistent", true)
	assert.Error(t, err)
}
