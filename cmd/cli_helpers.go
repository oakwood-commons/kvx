package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/oakwood-commons/kvx/internal/completion"
	"github.com/oakwood-commons/kvx/internal/navigator"
	ui "github.com/oakwood-commons/kvx/internal/ui"
)

type themeSelectionError struct {
	Selected     string
	Available    []string
	DefaultTheme string
}

func (e themeSelectionError) Error() string {
	return fmt.Sprintf("unknown theme %q\navailable themes: %v\ndefault theme: %s", e.Selected, e.Available, e.DefaultTheme)
}

func applyFunctionExamples(cfg ui.ThemeConfigFile, clearMissing bool) {
	examples := cfg.Help.CEL.FunctionExamples
	if len(examples) == 0 {
		examples = cfg.Help.FunctionExamples
	}
	if len(examples) == 0 {
		if clearMissing {
			completion.SetFunctionExamplesWithDescriptions(nil)
		}
		return
	}

	data := make(map[string]completion.FunctionExampleData, len(examples))
	for name, exValue := range examples {
		data[name] = completion.FunctionExampleData{
			Description: exValue.Description,
			Examples:    exValue.Examples,
		}
	}
	completion.SetFunctionExamplesWithDescriptions(data)
}

func defaultThemeName(cfg ui.ThemeConfigFile) string {
	if name := strings.TrimSpace(cfg.Theme.Default); name != "" {
		return name
	}
	if name := strings.TrimSpace(cfg.DefaultTheme); name != "" {
		return name
	}
	return "dark"
}

func applyThemeFromConfig(cfg ui.ThemeConfigFile, cliTheme string, themeFlagSet bool) error {
	selectedTheme := strings.TrimSpace(cliTheme)
	if !themeFlagSet {
		selectedTheme = ""
	}
	if selectedTheme == "" {
		selectedTheme = defaultThemeName(cfg)
	}

	applyTheme := func(name string) bool {
		if name == "" {
			return false
		}
		if th, ok := cfg.Themes[name]; ok {
			ui.SetTheme(ui.ThemeFromConfig(th))
			return true
		}
		if th, ok := ui.GetTheme(name); ok {
			ui.SetTheme(th)
			return true
		}
		return false
	}

	if applyTheme(selectedTheme) {
		return nil
	}

	if !themeFlagSet {
		fallback := defaultThemeName(cfg)
		if fallback != selectedTheme && applyTheme(fallback) {
			return nil
		}
	}

	available := getAllAvailableThemes(&cfg)
	def := defaultThemeName(cfg)
	return themeSelectionError{Selected: selectedTheme, Available: available, DefaultTheme: def}
}

func printThemeSelectionError(w io.Writer, err error) {
	var themeErr themeSelectionError
	if errors.As(err, &themeErr) {
		fmt.Fprintf(w, "unknown theme %q\n", themeErr.Selected)
		fmt.Fprintf(w, "available themes: %v\n", themeErr.Available)
		fmt.Fprintf(w, "default theme: %s\n", themeErr.DefaultTheme)
		return
	}
	fmt.Fprintln(w, err)
}

// loadConfigState centralizes config load + theme application for CLI flows.
// applyExamples clears/sets completion examples when requested.
// applyMenu applies menu config when data exists.
func loadConfigState(path string, cliTheme string, themeFlagSet bool, applyExamples bool, clearMissing bool, applyMenu bool) (ui.ThemeConfigFile, error) {
	cfg, err := loadMergedConfig(path)
	if err != nil {
		return cfg, err
	}

	if applyExamples {
		applyFunctionExamples(cfg, clearMissing)
	}

	if err := ui.InitializeThemes(&cfg); err != nil {
		// Log warning but continue - will use fallback dark theme
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize themes: %v\n", err)
	}

	if err := applyThemeFromConfig(cfg, cliTheme, themeFlagSet); err != nil {
		return cfg, err
	}

	order, err := resolveSortOrder(cfg)
	if err != nil {
		return cfg, err
	}
	navigator.SetSortOrder(order)

	if applyMenu && menuHasData(cfg.Menu) {
		ui.SetMenuConfig(ui.MenuFromConfig(cfg.Menu, cfg.Features.AllowEditInput))
	}

	return cfg, nil
}
