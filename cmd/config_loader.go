package cmd

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"

	"github.com/oakwood-commons/kvx/internal/ui"
)

// configLoader centralizes config/theme loading so callers avoid duplicating merge logic.
type configLoader struct {
	defaultConfig func() ([]byte, error)
}

var cfgLoader = configLoader{defaultConfig: loadDefaultConfigYAML}

func loadMergedConfig(cfgPath string) (ui.ThemeConfigFile, error) {
	return cfgLoader.loadMergedConfig(cfgPath)
}

func loadDefaultConfigRaw() ([]byte, error) {
	return cfgLoader.loadDefaultConfigRaw()
}

func sanitizeConfig(cfg ui.ThemeConfigFile) outputConfig {
	return cfgLoader.sanitizeConfig(cfg)
}

func addConfigComments(yml string) string {
	return cfgLoader.addConfigComments(yml)
}

func loadDefaultConfigYAML() ([]byte, error) {
	data := ui.DefaultConfigYAML()
	if len(data) == 0 {
		return nil, fmt.Errorf("embedded default config is empty")
	}
	return data, nil
}

func (l configLoader) loadMergedConfig(cfgPath string) (ui.ThemeConfigFile, error) {
	var cfg ui.ThemeConfigFile

	defaultData, err := l.loadDefaultConfigRaw()
	if err != nil {
		return cfg, fmt.Errorf("load default config: %w", err)
	}

	// Attempt nested v2 schema first (supports both app: and ui: sections)
	var nested struct {
		App ui.AppConfig `yaml:"app"`
		UI  uiBlock      `yaml:"ui"`
	}
	if err := yaml.Unmarshal(defaultData, &nested); err != nil {
		return cfg, fmt.Errorf("decode default config: %w", err)
	}

	// Merge default config (authoritative source of defaults)
	cfg = mergeConfigFromNested(nested, ui.ThemeConfigFile{})
	if cfg.Theme.Default == "" || len(cfg.Themes) == 0 {
		return cfg, fmt.Errorf("default config is missing required theme defaults")
	}

	// Merge from provided config file if present
	if cfgPath != "" {
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			return cfg, err
		}

		// Attempt nested v2 schema first (supports both app: and ui: sections)
		var nested struct {
			App ui.AppConfig `yaml:"app"`
			UI  uiBlock      `yaml:"ui"`
		}
		if err := yaml.Unmarshal(data, &nested); err == nil && (nested.UI.Theme.Default != "" || nested.UI.Defaults != (uiDefaults{}) || len(nested.UI.Themes) > 0 || menuHasData(nested.UI.Menu) || nested.App.Debug.MaxEvents != nil || nested.App.About.Name != "") {
			// Merge user config on top of defaults
			cfg = mergeConfigFromNested(nested, cfg)
			// Continue to populate themes if needed
			if len(cfg.Themes) == 0 {
				cfg.Themes = make(map[string]ui.ThemeConfig)
				for name, theme := range ui.GetAvailableThemes() {
					cfg.Themes[name] = ui.ThemeConfigFromTheme(theme)
				}
			}
			// Populate dynamic fields and process templates
			buildData := buildVersionData(&cfg)
			applyBuildData(&cfg, buildData)
			// Name is always set from default config, no need to fallback
			// Populate dynamic help menu text from menu config
			allowEditInput := true
			if cfg.Features.AllowEditInput != nil {
				allowEditInput = *cfg.Features.AllowEditInput
			}
			menuConfig := ui.MenuFromConfig(cfg.Menu, &allowEditInput)
			cfg.HelpMenu.Text = ui.GenerateHelpText(menuConfig, allowEditInput, nil, ui.DefaultKeyMode)
			cfg = processConfigTemplates(cfg)
			return cfg, nil
		}

		// Fallback to legacy flat schema
		fileCfg, err := ui.LoadThemeConfig(cfgPath)
		if err != nil {
			return cfg, err
		}
		// Map legacy fields to new structure
		if fileCfg.DefaultTheme != "" {
			cfg.Theme.Default = fileCfg.DefaultTheme
		}
		if fileCfg.AllowEditInput != nil {
			cfg.Features.AllowEditInput = fileCfg.AllowEditInput
		}
		if fileCfg.AllowFilter != nil {
			cfg.Features.AllowFilter = fileCfg.AllowFilter
		}
		if fileCfg.AllowSuggest != nil {
			cfg.Features.AllowSuggestions = fileCfg.AllowSuggest
		}
		if fileCfg.AllowIntel != nil {
			cfg.Features.AllowIntellisense = fileCfg.AllowIntel
		}
		if fileCfg.KeyColWidth != nil {
			cfg.Display.KeyColWidth = fileCfg.KeyColWidth
		}
		if ui.InfoPopupHasData(fileCfg.InfoPopup) {
			cfg.Popup.InfoPopup = mergeInfoPopup(cfg.Popup.InfoPopup, fileCfg.InfoPopup)
		}
		if fileCfg.DebugMaxEvents != nil {
			cfg.Debug.MaxEvents = fileCfg.DebugMaxEvents
		}
		// If any menu item is set, replace menu with fileCfg menu
		if menuHasData(fileCfg.Menu) {
			cfg.Menu = mergeMenuConfig(cfg.Menu, fileCfg.Menu)
		}
		if len(fileCfg.Themes) > 0 {
			mergedThemes := make(map[string]ui.ThemeConfig)
			for name, themeCfg := range fileCfg.Themes {
				base, ok := cfg.Themes[name]
				if !ok {
					darkTheme, _ := ui.GetTheme("dark")
					base = ui.ThemeConfigFromTheme(darkTheme)
				}
				mergedThemes[name] = mergeThemeConfig(base, themeCfg)
			}
			cfg.Themes = mergedThemes
		}
	}

	// Clear inline legacy theme to avoid emitting empty placeholder fields
	cfg.LegacyTheme = ui.ThemeConfig{}

	// Populate dynamic about fields from build info
	buildData := buildVersionData(&cfg)
	applyBuildData(&cfg, buildData)
	// Name is always set from default config, no need to fallback

	// Populate dynamic help menu text from menu config
	allowEditInput := true
	if cfg.Features.AllowEditInput != nil {
		allowEditInput = *cfg.Features.AllowEditInput
	}
	menuConfig := ui.MenuFromConfig(cfg.Menu, &allowEditInput)
	// Get navigation descriptions from config if available (future: could be in config)
	cfg.HelpMenu.Text = ui.GenerateHelpText(menuConfig, allowEditInput, nil, ui.DefaultKeyMode)

	// Process templates in config values (menu popups, info popups)
	cfg = processConfigTemplates(cfg)

	return cfg, nil
}

func (l configLoader) loadDefaultConfigRaw() ([]byte, error) {
	if l.defaultConfig != nil {
		return l.defaultConfig()
	}
	return loadDefaultConfigYAML()
}

func (l configLoader) sanitizeConfig(cfg ui.ThemeConfigFile) outputConfig {
	// Create a sanitized AboutConfig that excludes dynamic fields (version, go_version, build_os, build_arch, git_commit)
	// These are populated at runtime from build info and should not appear in --config output
	sanitizedAbout := ui.AboutConfig{
		Name:             cfg.About.Name,
		Description:      cfg.About.Description,
		License:          cfg.About.License,
		RepositoryURL:    cfg.About.RepositoryURL,
		DocumentationURL: cfg.About.DocumentationURL,
		Author:           cfg.About.Author,
		Details:          cfg.About.Details,
		// Exclude dynamic fields: Version, GoVersion, BuildOS, BuildArch, GitCommit
	}
	// Create a sanitized HelpMenuConfig that excludes dynamic text field
	// Help text is populated at runtime and should not appear in --config output
	sanitizedHelp := ui.HelpMenuConfig{
		// Exclude Text field (dynamic: generated from menu config)
	}
	return outputConfig{
		App: ui.AppConfig{
			About: sanitizedAbout,
			CLI:   cfg.CLI,
			Debug: cfg.Debug,
			Help:  sanitizedHelp,
		},
		UI: uiBlock{
			Theme:        cfg.Theme,
			Features:     cfg.Features,
			Intellisense: cfg.Intellisense,
			Help:         cfg.Help,
			Display:      cfg.Display,
			Behavior:     cfg.Behavior,
			Performance:  cfg.Performance,
			Search:       cfg.Search,
			Formatting:   cfg.Formatting,
			Popup:        cfg.Popup,
			Themes:       cfg.Themes,
			Menu:         cfg.Menu,
		},
	}
}

func (l configLoader) addConfigComments(yml string) string {
	return addConfigCommentsInternal(yml)
}

// mergeConfigFromNested merges a nested config structure into the base config
func mergeConfigFromNested(nested struct {
	App ui.AppConfig `yaml:"app"`
	UI  uiBlock      `yaml:"ui"`
}, base ui.ThemeConfigFile) ui.ThemeConfigFile {
	cfg := base
	// Merge app-level settings
	if nested.App.Debug.MaxEvents != nil {
		cfg.Debug.MaxEvents = nested.App.Debug.MaxEvents
	}
	if nested.App.About.Name != "" {
		cfg.About.Name = nested.App.About.Name
	}
	if nested.App.About.Description != "" {
		cfg.About.Description = nested.App.About.Description
	}
	// Note: Dynamic fields (version, go_version, build_os, build_arch, git_commit)
	// are populated from build info, not from config file
	if nested.App.About.License != "" {
		cfg.About.License = nested.App.About.License
	}
	if nested.App.About.RepositoryURL != "" {
		cfg.About.RepositoryURL = nested.App.About.RepositoryURL
	}
	if nested.App.About.DocumentationURL != "" {
		cfg.About.DocumentationURL = nested.App.About.DocumentationURL
	}
	if nested.App.About.Author != "" {
		cfg.About.Author = nested.App.About.Author
	}
	if len(nested.App.About.Details) > 0 {
		cfg.About.Details = nested.App.About.Details
	}
	// Merge CLI settings
	if nested.App.CLI.HelpHeaderTemplate != "" {
		cfg.CLI.HelpHeaderTemplate = nested.App.CLI.HelpHeaderTemplate
	}
	if nested.App.CLI.HelpDescription != "" {
		cfg.CLI.HelpDescription = nested.App.CLI.HelpDescription
	}
	if nested.App.CLI.HelpUsage != "" {
		cfg.CLI.HelpUsage = nested.App.CLI.HelpUsage
	}
	// Merge UI-level settings - new structure
	if len(nested.UI.Help.CEL.FunctionExamples) > 0 {
		if cfg.Help.CEL.FunctionExamples == nil {
			cfg.Help.CEL.FunctionExamples = make(map[string]ui.FunctionExampleValue)
		}
		for k, v := range nested.UI.Help.CEL.FunctionExamples {
			cfg.Help.CEL.FunctionExamples[k] = v
		}
	}
	if nested.UI.Help.CEL.ExampleData != nil {
		cfg.Help.CEL.ExampleData = nested.UI.Help.CEL.ExampleData
	}
	if nested.UI.Theme.Default != "" {
		cfg.Theme.Default = nested.UI.Theme.Default
	}
	if nested.UI.Features.AllowEditInput != nil {
		cfg.Features.AllowEditInput = nested.UI.Features.AllowEditInput
	}
	if nested.UI.Features.AllowFilter != nil {
		cfg.Features.AllowFilter = nested.UI.Features.AllowFilter
	}
	if nested.UI.Features.AllowSuggestions != nil {
		cfg.Features.AllowSuggestions = nested.UI.Features.AllowSuggestions
	}
	if nested.UI.Features.AllowIntellisense != nil {
		cfg.Features.AllowIntellisense = nested.UI.Features.AllowIntellisense
	}
	if nested.UI.Display.KeyColWidth != nil {
		cfg.Display.KeyColWidth = nested.UI.Display.KeyColWidth
	}
	if nested.UI.Display.Sort != nil {
		cfg.Display.Sort = nested.UI.Display.Sort
	}
	if ui.InfoPopupHasData(nested.UI.Popup.InfoPopup) {
		cfg.Popup.InfoPopup = mergeInfoPopup(cfg.Popup.InfoPopup, nested.UI.Popup.InfoPopup)
	}
	// Legacy defaults support (backward compatibility)
	if nested.UI.Defaults.DefaultTheme != "" {
		cfg.Theme.Default = nested.UI.Defaults.DefaultTheme
	}
	if nested.UI.Defaults.AllowEditInput != nil {
		cfg.Features.AllowEditInput = nested.UI.Defaults.AllowEditInput
	}
	if nested.UI.Defaults.AllowFilter != nil {
		cfg.Features.AllowFilter = nested.UI.Defaults.AllowFilter
	}
	if nested.UI.Defaults.AllowSuggest != nil {
		cfg.Features.AllowSuggestions = nested.UI.Defaults.AllowSuggest
	}
	if nested.UI.Defaults.AllowIntel != nil {
		cfg.Features.AllowIntellisense = nested.UI.Defaults.AllowIntel
	}
	if nested.UI.Defaults.KeyColWidth != nil {
		cfg.Display.KeyColWidth = nested.UI.Defaults.KeyColWidth
	}
	if ui.InfoPopupHasData(nested.UI.Defaults.InfoPopup) {
		cfg.Popup.InfoPopup = mergeInfoPopup(cfg.Popup.InfoPopup, nested.UI.Defaults.InfoPopup)
	}
	if len(nested.UI.Themes) > 0 {
		mergedThemes := make(map[string]ui.ThemeConfig)
		for name, themeCfg := range nested.UI.Themes {
			baseTheme, ok := cfg.Themes[name]
			if !ok {
				baseTheme = ui.ThemeConfig{}
			}
			mergedThemes[name] = mergeThemeConfig(baseTheme, themeCfg)
		}
		cfg.Themes = mergedThemes
	}
	// Always merge menu config if it exists in the nested config (default config always has menu)
	if menuHasData(nested.UI.Menu) {
		cfg.Menu = mergeMenuConfig(cfg.Menu, nested.UI.Menu)
	}
	return cfg
}

// processConfigTemplates processes Go templates in config string values.
// Templates can access:
//   - .config - the entire config structure
//   - .build - build information (version, go_version, build_os, build_arch, git_commit)
func processConfigTemplates(cfg ui.ThemeConfigFile) ui.ThemeConfigFile {
	// Build template data
	templateData := map[string]interface{}{
		"config": map[string]interface{}{
			"app": map[string]interface{}{
				"about": map[string]interface{}{
					"name":              cfg.About.Name,
					"description":       cfg.About.Description,
					"version":           cfg.About.Version,
					"go_version":        cfg.About.GoVersion,
					"build_os":          cfg.About.BuildOS,
					"build_arch":        cfg.About.BuildArch,
					"git_commit":        cfg.About.GitCommit,
					"license":           cfg.About.License,
					"repository_url":    cfg.About.RepositoryURL,
					"documentation_url": cfg.About.DocumentationURL,
					"author":            cfg.About.Author,
					"details":           cfg.About.Details,
				},
				"cli": map[string]interface{}{
					"help_header_template": cfg.CLI.HelpHeaderTemplate,
					"help_description":     cfg.CLI.HelpDescription,
					"help_usage":           cfg.CLI.HelpUsage,
				},
				"debug": map[string]interface{}{
					"max_events": cfg.Debug.MaxEvents,
				},
				"help": map[string]interface{}{
					"text": cfg.HelpMenu.Text,
				},
			},
			"ui": map[string]interface{}{
				"theme": map[string]interface{}{
					"default": cfg.Theme.Default,
				},
				"features": map[string]interface{}{
					"allow_edit_input":   cfg.Features.AllowEditInput,
					"allow_filter":       cfg.Features.AllowFilter,
					"allow_suggestions":  cfg.Features.AllowSuggestions,
					"allow_intellisense": cfg.Features.AllowIntellisense,
				},
				"display": map[string]interface{}{
					"key_col_width": cfg.Display.KeyColWidth,
				},
				"popup": map[string]interface{}{
					"info_popup": cfg.Popup.InfoPopup,
				},
			},
		},
		"build": map[string]interface{}{
			"version":    cfg.About.Version,
			"go_version": cfg.About.GoVersion,
			"build_os":   cfg.About.BuildOS,
			"build_arch": cfg.About.BuildArch,
			"git_commit": cfg.About.GitCommit,
		},
	}

	// Process info popup text
	if cfg.Popup.InfoPopup.Text != "" {
		cfg.Popup.InfoPopup.Text = processTemplateString(cfg.Popup.InfoPopup.Text, templateData)
	}

	// Process about details (each line can contain templates)
	if len(cfg.About.Details) > 0 {
		processedDetails := make([]string, 0, len(cfg.About.Details))
		for _, detail := range cfg.About.Details {
			processedDetails = append(processedDetails, processTemplateString(detail, templateData))
		}
		cfg.About.Details = processedDetails
	}

	// Process menu popup texts
	processMenuItem := func(item *ui.MenuItemConfig) {
		if item.Popup.Text != "" {
			item.Popup.Text = processTemplateString(item.Popup.Text, templateData)
		}
		if item.Popup.Title != "" {
			item.Popup.Title = processTemplateString(item.Popup.Title, templateData)
		}
	}
	processMenuItem(&cfg.Menu.F1)
	processMenuItem(&cfg.Menu.F2)
	processMenuItem(&cfg.Menu.F3)
	processMenuItem(&cfg.Menu.F4)
	processMenuItem(&cfg.Menu.F5)
	processMenuItem(&cfg.Menu.F6)
	processMenuItem(&cfg.Menu.F7)
	processMenuItem(&cfg.Menu.F8)
	processMenuItem(&cfg.Menu.F9)
	processMenuItem(&cfg.Menu.F10)
	processMenuItem(&cfg.Menu.F11)
	processMenuItem(&cfg.Menu.F12)

	return cfg
}

// processTemplateString processes a template string, returning the original string if templating fails.
func processTemplateString(text string, data map[string]interface{}) string {
	// Check if text contains template syntax
	if !strings.Contains(text, "{{") {
		return text
	}

	tmpl, err := template.New("config").Parse(text)
	if err != nil {
		// If template parsing fails, return original text
		return text
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		// If template execution fails, return original text
		return text
	}

	return buf.String()
}

func menuHasData(menu ui.MenuConfigYAML) bool {
	// Check action-based menu items (new format)
	actionItems := []ui.MenuItemConfig{
		menu.Help, menu.Search, menu.Filter, menu.Copy, menu.Expr, menu.Quit, menu.Custom,
	}
	for _, it := range actionItems {
		if it.Label != "" || it.Action != "" || it.Enabled != nil || it.PopupText != "" || ui.InfoPopupHasData(it.Popup) || it.Keys.Function != "" || it.Keys.Vim != "" || it.Keys.Emacs != "" {
			return true
		}
	}
	// Check legacy F-key based menu items
	legacyItems := []ui.MenuItemConfig{
		menu.F1, menu.F2, menu.F3, menu.F4, menu.F5, menu.F6,
		menu.F7, menu.F8, menu.F9, menu.F10, menu.F11, menu.F12,
	}
	for _, it := range legacyItems {
		if it.Label != "" || it.Action != "" || it.Enabled != nil || it.PopupText != "" || ui.InfoPopupHasData(it.Popup) {
			return true
		}
	}
	return false
}

func mergeInfoPopup(base, override ui.InfoPopupConfig) ui.InfoPopupConfig {
	out := base
	if override.Text != "" {
		out.Text = override.Text
	}
	if override.Title != "" {
		out.Title = override.Title
	}
	if override.TitleJustify != "" {
		out.TitleJustify = override.TitleJustify
	}
	if override.Anchor != "" {
		out.Anchor = override.Anchor
	}
	if override.Justify != "" {
		out.Justify = override.Justify
	}
	if override.Modal != nil {
		out.Modal = override.Modal
	}
	if override.Permanent != nil {
		out.Permanent = override.Permanent
	}
	if override.Enabled != nil {
		out.Enabled = override.Enabled
	}
	return out
}

func mergeMenuConfig(base, override ui.MenuConfigYAML) ui.MenuConfigYAML {
	out := base
	apply := func(src ui.MenuItemConfig, dst *ui.MenuItemConfig) {
		if src.Label != "" {
			dst.Label = src.Label
		}
		if src.HelpText != "" {
			dst.HelpText = src.HelpText
		}
		if src.Action != "" {
			dst.Action = src.Action
		}
		if src.Enabled != nil {
			dst.Enabled = src.Enabled
		}
		// Merge Keys
		if src.Keys.Function != "" {
			dst.Keys.Function = src.Keys.Function
		}
		if src.Keys.Vim != "" {
			dst.Keys.Vim = src.Keys.Vim
		}
		if src.Keys.Emacs != "" {
			dst.Keys.Emacs = src.Keys.Emacs
		}
		if ui.InfoPopupHasData(src.Popup) {
			dst.Popup = mergeInfoPopup(dst.Popup, src.Popup)
		} else if src.PopupText != "" {
			dst.Popup.Text = src.PopupText
		}
	}
	// New action-based menu items
	apply(override.Help, &out.Help)
	apply(override.Search, &out.Search)
	apply(override.Filter, &out.Filter)
	apply(override.Copy, &out.Copy)
	apply(override.Expr, &out.Expr)
	apply(override.Quit, &out.Quit)
	apply(override.Custom, &out.Custom)
	// Legacy F-key based items
	apply(override.F1, &out.F1)
	apply(override.F2, &out.F2)
	apply(override.F3, &out.F3)
	apply(override.F4, &out.F4)
	apply(override.F5, &out.F5)
	apply(override.F6, &out.F6)
	apply(override.F7, &out.F7)
	apply(override.F8, &out.F8)
	apply(override.F9, &out.F9)
	apply(override.F10, &out.F10)
	apply(override.F11, &out.F11)
	apply(override.F12, &out.F12)
	return out
}

func mergeThemeConfig(base, override ui.ThemeConfig) ui.ThemeConfig {
	out := base
	apply := func(src ui.ColorValue, dst *ui.ColorValue) {
		if src != "" {
			*dst = src
		}
	}
	if strings.TrimSpace(override.BorderStyle) != "" {
		out.BorderStyle = override.BorderStyle
	}
	apply(override.KeyColor, &out.KeyColor)
	apply(override.ValueColor, &out.ValueColor)
	apply(override.HeaderFG, &out.HeaderFG)
	apply(override.HeaderBG, &out.HeaderBG)
	apply(override.SelectedFG, &out.SelectedFG)
	apply(override.SelectedBG, &out.SelectedBG)
	apply(override.SeparatorColor, &out.SeparatorColor)
	apply(override.InputBG, &out.InputBG)
	apply(override.InputFG, &out.InputFG)
	apply(override.GhostFG, &out.GhostFG)
	apply(override.StatusColor, &out.StatusColor)
	apply(override.StatusError, &out.StatusError)
	apply(override.StatusSuccess, &out.StatusSuccess)
	apply(override.DebugColor, &out.DebugColor)
	apply(override.FooterFG, &out.FooterFG)
	apply(override.FooterBG, &out.FooterBG)
	apply(override.HelpKey, &out.HelpKey)
	apply(override.HelpValue, &out.HelpValue)
	return out
}

func addConfigCommentsInternal(yml string) string {
	// Describe the popup options once above the first info_popup.
	if strings.Contains(yml, "\n    info_popup:\n") {
		comment := "    # Popup options: enabled (bool), text, anchor (inline|top), justify (left|center|right|middle), modal (true|false), permanent (ignore Esc)\n"
		yml = strings.Replace(yml, "\n    info_popup:\n", "\n"+comment+"    info_popup:\n", 1)
	}
	// Add a brief comment before the first menu popup block.
	if strings.Contains(yml, "\n      popup:\n") {
		comment := "      # Per-key popup payload: enabled/text/anchor/justify/modal/permanent\n"
		yml = strings.Replace(yml, "\n      popup:\n", "\n"+comment+"      popup:\n", 1)
	}
	// Add inline hints for the first popup block (both defaults and menu).
	inline := func(find, repl string) {
		yml = strings.Replace(yml, find, repl, 1)
	}
	inline("    info_popup:\n      enabled:", "    info_popup:\n      enabled:") // keep alignment
	inline("      enabled: true\n", "      enabled: true # show on startup\n")
	inline("      enabled: false\n", "      enabled: false # show on startup\n")
	inline("      anchor: top\n", "      anchor: top # inline|top\n")
	inline("      anchor: inline\n", "      anchor: inline # inline|top\n")
	inline("      justify: center\n", "      justify: center # left|center|right|middle\n")
	inline("      justify: left\n", "      justify: left # left|center|right|middle\n")
	inline("      justify: right\n", "      justify: right # left|center|right|middle\n")
	inline("      modal: true\n", "      modal: true # block input\n")
	inline("      modal: false\n", "      modal: false # allow input\n")
	inline("      permanent: true\n", "      permanent: true # ignore Esc\n")
	inline("      permanent: false\n", "      permanent: false # allow Esc to close\n")
	return yml
}

func applyBuildData(cfg *ui.ThemeConfigFile, buildData map[string]interface{}) {
	if v, ok := buildData["Version"].(string); ok {
		cfg.About.Version = v
	}
	if v, ok := buildData["GoVersion"].(string); ok {
		cfg.About.GoVersion = v
	}
	if v, ok := buildData["BuildOS"].(string); ok {
		cfg.About.BuildOS = v
	}
	if v, ok := buildData["BuildArch"].(string); ok {
		cfg.About.BuildArch = v
	}
	if v, ok := buildData["GitCommit"].(string); ok {
		cfg.About.GitCommit = v
	}
}

// outputConfig mirrors ThemeConfigFile without the inline legacy Theme field.
type outputConfig struct {
	App ui.AppConfig `yaml:"app,omitempty" json:"app,omitempty"`
	UI  uiBlock      `yaml:"ui,omitempty" json:"ui,omitempty"`
}

// uiBlock groups UI config for nested output/inputs.
type uiBlock struct {
	Theme        ui.ThemeSelectionConfig   `yaml:"theme,omitempty" json:"theme,omitempty"`
	Features     ui.FeaturesConfig         `yaml:"features,omitempty" json:"features,omitempty"`
	Intellisense ui.IntellisenseConfig     `yaml:"intellisense,omitempty" json:"intellisense,omitempty"`
	Help         ui.HelpConfig             `yaml:"help,omitempty" json:"help,omitempty"`
	Display      ui.DisplayConfig          `yaml:"display,omitempty" json:"display,omitempty"`
	Behavior     ui.BehaviorConfig         `yaml:"behavior,omitempty" json:"behavior,omitempty"`
	Performance  ui.PerformanceConfig      `yaml:"performance,omitempty" json:"performance,omitempty"`
	Search       ui.SearchConfig           `yaml:"search,omitempty" json:"search,omitempty"`
	Formatting   ui.FormattingConfig       `yaml:"formatting,omitempty" json:"formatting,omitempty"`
	Popup        ui.PopupConfig            `yaml:"popup,omitempty" json:"popup,omitempty"`
	Themes       map[string]ui.ThemeConfig `yaml:"themes,omitempty" json:"themes,omitempty"`
	Menu         ui.MenuConfigYAML         `yaml:"menu,omitempty" json:"menu,omitempty"`
	// Legacy fields for backward compatibility
	Defaults uiDefaults `yaml:"defaults,omitempty" json:"defaults,omitempty"`
}

type uiDefaults struct {
	DefaultTheme   string             `yaml:"default_theme,omitempty" json:"default_theme,omitempty" yamlcomment:"[DEPRECATED] Use ui.theme.default instead"`
	AllowEditInput *bool              `yaml:"allow_edit_input,omitempty" json:"allow_edit_input,omitempty" yamlcomment:"[DEPRECATED] Use ui.features.allow_edit_input instead"`
	AllowFilter    *bool              `yaml:"allow_filter,omitempty" json:"allow_filter,omitempty" yamlcomment:"[DEPRECATED] Use ui.features.allow_filter instead"`
	AllowSuggest   *bool              `yaml:"allow_suggestions,omitempty" json:"allow_suggestions,omitempty" yamlcomment:"[DEPRECATED] Use ui.features.allow_suggestions instead"`
	AllowIntel     *bool              `yaml:"allow_intellisense,omitempty" json:"allow_intellisense,omitempty" yamlcomment:"[DEPRECATED] Use ui.features.allow_intellisense instead"`
	KeyColWidth    *int               `yaml:"key_col_width,omitempty" json:"key_col_width,omitempty" yamlcomment:"[DEPRECATED] Use ui.display.key_col_width instead"`
	ValueColWidth  *int               `yaml:"value_col_width,omitempty" json:"value_col_width,omitempty" yamlcomment:"[DEPRECATED] Use ui.display.value_col_width instead"`
	ShowPanelTitle *bool              `yaml:"show_panel_title,omitempty" json:"show_panel_title,omitempty" yamlcomment:"[DEPRECATED] Use ui.display.show_panel_title instead"`
	InfoPopup      ui.InfoPopupConfig `yaml:"info_popup,omitempty" json:"info_popup,omitempty" yamlcomment:"[DEPRECATED] Use ui.popup.info_popup instead"`
}

// getAllAvailableThemes collects all available theme names from built-in presets and config file.
func getAllAvailableThemes(cfgFile *ui.ThemeConfigFile) []string {
	themeMap := make(map[string]bool)

	// Add built-in presets
	for name := range ui.GetAvailableThemes() {
		themeMap[name] = true
	}

	// Add themes from config file if provided
	if cfgFile != nil {
		for name := range cfgFile.Themes {
			themeMap[name] = true
		}
	}

	// Convert to sorted slice
	themes := make([]string, 0, len(themeMap))
	for name := range themeMap {
		themes = append(themes, name)
	}
	sort.Strings(themes)
	return themes
}
