package ui

import (
	"fmt"
	"image/color"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"charm.land/lipgloss/v2"
	"gopkg.in/yaml.v3"

	"github.com/oakwood-commons/kvx/internal/formatter"
)

// Theme defines colors and styles used across the UI. Host apps can supply their own theme.
type Theme struct {
	KeyColor       color.Color // Color for keys in table (left column)
	ValueColor     color.Color // Color for values in table (right column)
	HeaderFG       color.Color // Border color text (top border)
	HeaderBG       color.Color // Border color background (top border)
	BorderStyle    string      // Border style (normal|rounded)
	SelectedFG     color.Color // Selected row foreground
	SelectedBG     color.Color // Selected row background
	SeparatorColor color.Color // Color for separator lines
	InputBG        color.Color // Path input background
	InputFG        color.Color // Path input text
	GhostFG        color.Color // Ghost/suggestion text in the input
	StatusColor    color.Color // Normal status bar text
	StatusError    color.Color // Error status bar text
	StatusSuccess  color.Color // Success status bar text
	DebugColor     color.Color // Debug bar text
	FooterFG       color.Color // Border color text (bottom border/footer)
	FooterBG       color.Color // Border color background (bottom border/footer)
	HelpKey        color.Color // Help key labels
	HelpValue      color.Color // Help value text
}

var (
	defaultThemeOnce sync.Once
	defaultTheme     Theme
	currentTheme     Theme
)

// DefaultTheme returns the palette defined in the embedded default configuration.
// Falls back to the legacy hard-coded palette only if the embedded config cannot be read.
func DefaultTheme() Theme {
	defaultThemeOnce.Do(func() {
		cfg, err := EmbeddedDefaultConfig()
		if err != nil {
			defaultTheme = fallbackDefaultTheme()
			return
		}

		// Populate ThemePresets from embedded config so downstream callers see consistent themes.
		// Only populate if presets are empty to avoid clobbering preloaded themes (e.g., tests calling InitializeThemes).
		// Use fallbackDefaultTheme as the base to avoid recursive DefaultTheme calls while building presets.
		if len(ThemePresets) == 0 {
			ThemePresets = make(map[string]Theme, len(cfg.Themes))
			base := fallbackDefaultTheme()
			for name, themeCfg := range cfg.Themes {
				ThemePresets[name] = themeFromConfigWithBase(themeCfg, base)
			}
		}

		selected := strings.TrimSpace(cfg.Theme.Default)
		if selected == "" {
			selected = "dark"
		}
		if th, ok := ThemePresets[selected]; ok {
			defaultTheme = th
			return
		}
		defaultTheme = fallbackDefaultTheme()
	})

	return defaultTheme
}

// fallbackDefaultTheme preserves the historical palette for safety.
func fallbackDefaultTheme() Theme {
	return Theme{
		KeyColor:       lipgloss.Color("81"),  // cyan keys for contrast
		ValueColor:     lipgloss.Color("246"), // muted gray values (avoid bright white)
		HeaderFG:       lipgloss.Color("81"),  // cyan title
		HeaderBG:       lipgloss.Color("236"), // charcoal header background
		BorderStyle:    "normal",              // default to square borders
		SelectedFG:     lipgloss.Color("250"), // muted light text on selection
		SelectedBG:     lipgloss.Color("24"),  // deep teal selection
		SeparatorColor: lipgloss.Color("238"), // subtle separators
		InputBG:        lipgloss.Color("236"), // match header/footer background
		InputFG:        lipgloss.Color("246"), // muted input text
		GhostFG:        lipgloss.Color("246"), // ghost text matches input by default
		StatusColor:    lipgloss.Color("81"),  // cyan status
		StatusError:    lipgloss.Color("203"), // softer red for errors
		StatusSuccess:  lipgloss.Color("114"), // mint success
		DebugColor:     lipgloss.Color("244"), // muted debug text
		FooterFG:       lipgloss.Color("244"), // muted footer text
		FooterBG:       lipgloss.Color("236"), // charcoal footer background
		HelpKey:        lipgloss.Color("81"),  // match accent
		HelpValue:      lipgloss.Color("245"), // muted gray help text
	}
}

// loadedThemes stores all available themes loaded from configuration.
// This is populated at startup by InitializeThemes() using default_config.yaml.
var loadedThemes = map[string]Theme{}

// Deprecated: ThemePresets exists only for backward compatibility with existing code.
// Use GetAvailableThemes() instead to get themes from loaded configuration.
// This will be removed in a future version.
var ThemePresets = map[string]Theme{}

// SetTheme overrides the global theme.
func SetTheme(t Theme) {
	t.BorderStyle = normalizeBorderStyle(t.BorderStyle)
	currentTheme = t
	// v2: colors are color.Color interface - formatter expects color.Color directly
	formatter.SetTableTheme(formatter.TableColors{
		HeaderFG:       t.HeaderFG, // Now color.Color instead of string
		HeaderBG:       t.HeaderBG, // Now color.Color instead of string
		KeyColor:       t.KeyColor,
		ValueColor:     t.ValueColor,
		SeparatorColor: t.SeparatorColor,
	})
}

// SetThemeByName sets the theme by name from loaded configuration.
// Returns an error if the theme name is not found in loaded themes.
// Themes must be initialized first via InitializeThemes() before this can be used.
func SetThemeByName(name string) error {
	if theme, ok := loadedThemes[name]; ok {
		SetTheme(theme)
		return nil
	}
	// If no themes loaded yet, return helpful error
	if len(loadedThemes) == 0 {
		return fmt.Errorf("no themes loaded; call InitializeThemes() before SetThemeByName()")
	}
	return fmt.Errorf("unknown theme %q (available: %s)", name, getAvailableThemeNames())
}

// getAvailableThemeNames returns a comma-separated list of available theme names.
func getAvailableThemeNames() string {
	if len(loadedThemes) == 0 {
		return "(none)"
	}
	names := make([]string, 0, len(loadedThemes))
	for name := range loadedThemes {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// GetTheme returns a theme by name from loaded configuration.
// Returns the theme and true if found, or a zero Theme and false if not found.
// Use this instead of accessing ThemePresets directly.
func GetTheme(name string) (Theme, bool) {
	if theme, ok := loadedThemes[name]; ok {
		return theme, true
	}
	// Fallback to ThemePresets for backward compatibility during transition
	if theme, ok := ThemePresets[name]; ok {
		return theme, true
	}
	return Theme{}, false
}

// GetAvailableThemes returns a map of all available theme names to their Theme values.
// This includes both themes from loaded configuration and built-in presets.
func GetAvailableThemes() map[string]Theme {
	result := make(map[string]Theme)
	// Add built-in presets first (lower priority)
	for name, theme := range ThemePresets {
		result[name] = theme
	}
	// Override with loaded themes (higher priority)
	for name, theme := range loadedThemes {
		result[name] = theme
	}
	return result
}

// Theme returns the currently configured theme.
func CurrentTheme() Theme {
	if currentTheme == (Theme{}) {
		currentTheme = DefaultTheme()
	}
	return currentTheme
}

// ColorValue stores a color token (number or name) and marshals numerics as YAML ints.
type ColorValue string

func (c ColorValue) MarshalYAML() (interface{}, error) {
	if c == "" {
		return "", nil
	}
	s := string(c)
	if _, err := strconv.Atoi(s); err == nil {
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!int",
			Value: s,
		}, nil
	}
	return s, nil
}

func (c *ColorValue) UnmarshalYAML(value *yaml.Node) error {
	if value == nil {
		*c = ""
		return nil
	}
	// Accept both ints and strings; store the literal value.
	*c = ColorValue(value.Value)
	return nil
}

// ThemeConfig is a YAML-friendly theme configuration (colors accept ints or strings).
type ThemeConfig struct {
	KeyColor       ColorValue `yaml:"key_color" yamlcomment:"Key column color"`
	ValueColor     ColorValue `yaml:"value_color" yamlcomment:"Value column color"`
	HeaderFG       ColorValue `yaml:"header_fg" yamlcomment:"Header foreground"`
	HeaderBG       ColorValue `yaml:"header_bg" yamlcomment:"Header background"`
	BorderStyle    string     `yaml:"border_style" yamlcomment:"Border style (normal|rounded)"`
	SelectedFG     ColorValue `yaml:"selected_fg" yamlcomment:"Selected row foreground"`
	SelectedBG     ColorValue `yaml:"selected_bg" yamlcomment:"Selected row background"`
	SeparatorColor ColorValue `yaml:"separator_color" yamlcomment:"Separator line color"`
	InputBG        ColorValue `yaml:"input_bg" yamlcomment:"Expression bar background"`
	InputFG        ColorValue `yaml:"input_fg" yamlcomment:"Expression bar foreground"`
	GhostFG        ColorValue `yaml:"ghost_fg" yamlcomment:"Ghost/suggestion text foreground"`
	StatusColor    ColorValue `yaml:"status_color" yamlcomment:"Status bar color"`
	StatusError    ColorValue `yaml:"status_error" yamlcomment:"Status error color"`
	StatusSuccess  ColorValue `yaml:"status_success" yamlcomment:"Status success color"`
	DebugColor     ColorValue `yaml:"debug_color" yamlcomment:"Debug text color"`
	FooterFG       ColorValue `yaml:"footer_fg" yamlcomment:"Footer foreground"`
	FooterBG       ColorValue `yaml:"footer_bg" yamlcomment:"Footer background"`
	HelpKey        ColorValue `yaml:"help_key" yamlcomment:"Help key color"`
	HelpValue      ColorValue `yaml:"help_value" yamlcomment:"Help value color"`
}

type InfoPopupConfig struct {
	Title        string `yaml:"title,omitempty" yamlcomment:"Optional popup title"`
	TitleJustify string `yaml:"title_justify,omitempty" yamlcomment:"Title alignment (left|center|right|middle)"`
	Text         string `yaml:"text,omitempty" yamlcomment:"Popup text content"`
	Anchor       string `yaml:"anchor,omitempty" yamlcomment:"Where to anchor the popup (inline|top)"`
	Justify      string `yaml:"justify,omitempty" yamlcomment:"Text alignment (left|center|right|middle)"`
	Modal        *bool  `yaml:"modal,omitempty" yamlcomment:"Whether the popup is modal (blocks interaction)"`
	Permanent    *bool  `yaml:"permanent,omitempty" yamlcomment:"Keep popup visible; ignore Esc"`
	Enabled      *bool  `yaml:"enabled,omitempty" yamlcomment:"Enable/disable this popup"`
}

// AboutConfig contains application metadata and information.
// Fields marked as dynamic are populated at runtime from build info.
type AboutConfig struct {
	Name        string `yaml:"name,omitempty" yamlcomment:"Application name"`
	Description string `yaml:"description,omitempty" yamlcomment:"Application description"`
	// Dynamic fields (populated at runtime, shown commented in default config)
	Version   string `yaml:"version,omitempty" yamlcomment:"Application version (dynamic: from build info)"`
	GoVersion string `yaml:"go_version,omitempty" yamlcomment:"Go version used to build (dynamic: from build info)"`
	BuildOS   string `yaml:"build_os,omitempty" yamlcomment:"Build operating system (dynamic: from build info)"`
	BuildArch string `yaml:"build_arch,omitempty" yamlcomment:"Build architecture (dynamic: from build info)"`
	GitCommit string `yaml:"git_commit,omitempty" yamlcomment:"Git commit hash (dynamic: from build info)"`
	// Static fields
	License          string   `yaml:"license,omitempty" yamlcomment:"License information"`
	RepositoryURL    string   `yaml:"repository_url,omitempty" yamlcomment:"Source code repository URL"`
	DocumentationURL string   `yaml:"documentation_url,omitempty" yamlcomment:"Documentation website URL"`
	Author           string   `yaml:"author,omitempty" yamlcomment:"Author or maintainer name"`
	Details          []string `yaml:"details,omitempty" yamlcomment:"Additional details (array of strings, supports templates)"`
}

// CLIConfig holds CLI-specific configuration.
type CLIConfig struct {
	HelpHeaderTemplate string `yaml:"help_header_template,omitempty" yamlcomment:"Template for CLI --help header (supports Go templates)"`
	HelpDescription    string `yaml:"help_description,omitempty" yamlcomment:"Description paragraph for CLI --help (supports Go templates)"`
	HelpUsage          string `yaml:"help_usage,omitempty" yamlcomment:"Usage instructions for CLI --help (supports Go templates)"`
}

// HelpMenuConfig holds the dynamically generated help menu text.
// This is populated from the menu config and navigation descriptions,
// allowing the formatted help text to be accessed via Go templating (e.g., {{.config.app.help.text}}).
type HelpMenuConfig struct {
	Text string `yaml:"text,omitempty" yamlcomment:"Formatted help menu text (dynamic: generated from menu config)"`
}

// DebugConfig holds debug and logging configuration.
type DebugConfig struct {
	MaxEvents *int `yaml:"max_events,omitempty" yamlcomment:"Maximum number of debug events to keep (default: 200)"`
}

// ThemeSelectionConfig holds theme selection configuration.
type ThemeSelectionConfig struct {
	Default string `yaml:"default,omitempty" yamlcomment:"Default theme name"`
}

// FeaturesConfig holds feature flags for UI features.
type FeaturesConfig struct {
	AllowEditInput    *bool   `yaml:"allow_edit_input,omitempty" yamlcomment:"Enable expression editing (expr toggle)"`
	AllowFilter       *bool   `yaml:"allow_filter,omitempty" yamlcomment:"Enable type-ahead filtering while navigating"`
	AllowSuggestions  *bool   `yaml:"allow_suggestions,omitempty" yamlcomment:"Show CEL/intellisense suggestions"`
	AllowIntellisense *bool   `yaml:"allow_intellisense,omitempty" yamlcomment:"Show CEL/intellisense dropdown hints"`
	KeyMode           *string `yaml:"key_mode,omitempty" yamlcomment:"Keybinding mode: vim (default), emacs, or function"`
}

// DisplayConfig holds display and layout settings.
type DisplayConfig struct {
	KeyColWidth *int    `yaml:"key_col_width,omitempty" yamlcomment:"Width of the KEY column (default: 30)"`
	Sort        *string `yaml:"sort,omitempty" yamlcomment:"Sort order for map keys: none|ascending|descending"`
}

// BehaviorConfig holds user behavior and interaction settings.
type BehaviorConfig struct {
	// Future fields will be added here
}

// PerformanceConfig holds performance and optimization settings.
type PerformanceConfig struct {
	// FilterDebounceMs is the debounce delay in milliseconds for real-time search filtering.
	// Higher values reduce lag when typing quickly in search mode.
	// Default: 150
	FilterDebounceMs *int `yaml:"filter_debounce_ms,omitempty" yamlcomment:"Debounce delay for search filtering (milliseconds)"`

	// SearchResultLimit is the maximum number of results returned by deep search.
	// When exceeded, a "showing first N of many results" indicator is shown.
	// Default: 500
	SearchResultLimit *int `yaml:"search_result_limit,omitempty" yamlcomment:"Maximum results for deep search"`

	// ScrollBufferRows is the number of rows to pre-render above/below the visible viewport
	// when virtual scrolling is enabled. Improves scroll smoothness.
	// Default: 5
	ScrollBufferRows *int `yaml:"scroll_buffer_rows,omitempty" yamlcomment:"Rows to buffer above/below viewport"`

	// VirtualScrolling enables rendering only visible rows instead of all rows.
	// Improves performance for large datasets.
	// Default: true
	VirtualScrolling *bool `yaml:"virtual_scrolling,omitempty" yamlcomment:"Enable virtual scrolling for large datasets"`
}

// SearchConfig holds search and filtering settings.
type SearchConfig struct {
	// Future fields will be added here
}

// FormattingConfig holds data formatting settings.
type FormattingConfig struct {
	YAML    YAMLFormattingConfig    `yaml:"yaml,omitempty" yamlcomment:"YAML output settings"`
	Table   TableFormattingConfig   `yaml:"table,omitempty" yamlcomment:"Table output settings"`
	Tree    TreeFormattingConfig    `yaml:"tree,omitempty" yamlcomment:"Tree output settings"`
	Mermaid MermaidFormattingConfig `yaml:"mermaid,omitempty" yamlcomment:"Mermaid diagram settings"`
}

// TableFormattingConfig controls table output formatting.
type TableFormattingConfig struct {
	// ArrayStyle controls how array indices are displayed:
	// "index" = [0], [1], [2]; "numbered" = 1, 2, 3; "bullet" = •; "none" = no index
	ArrayStyle *string `yaml:"array_style,omitempty" yamlcomment:"Array index style: index, numbered, bullet, none (default: numbered)"`

	// ColumnarMode controls when arrays render as multi-column tables:
	// "auto" = detect homogeneous arrays; "always" = force columnar; "never" = KEY/VALUE only
	ColumnarMode *string `yaml:"columnar_mode,omitempty" yamlcomment:"Columnar rendering: auto, always, never (default: auto)"`

	// ColumnOrder specifies preferred column order for columnar tables.
	ColumnOrder []string `yaml:"column_order,omitempty" yamlcomment:"Preferred column order for columnar display"`

	// HiddenColumns specifies columns to omit from columnar tables.
	HiddenColumns []string `yaml:"hidden_columns,omitempty" yamlcomment:"Columns to hide in columnar display"`

	// SchemaFile is a path to a JSON Schema file used to derive column display hints.
	SchemaFile *string `yaml:"schema_file,omitempty" yamlcomment:"JSON Schema file for column display hints"`

	// Schema is an inline JSON Schema object used to derive column display hints.
	// schema_file takes precedence over this if both are set.
	Schema map[string]any `yaml:"schema,omitempty" yamlcomment:"Inline JSON Schema for column display hints"`
}

// YAMLFormattingConfig controls YAML output formatting.
type YAMLFormattingConfig struct {
	Indent                *int  `yaml:"indent,omitempty" yamlcomment:"Indentation size in spaces (default: 2)"`
	LiteralBlockStrings   *bool `yaml:"literal_block_strings,omitempty" yamlcomment:"Render multiline strings with | literal blocks"`
	ExpandEscapedNewlines *bool `yaml:"expand_escaped_newlines,omitempty" yamlcomment:"Convert literal \n sequences to real newlines in YAML output"`
}

// TreeFormattingConfig controls tree output formatting.
type TreeFormattingConfig struct {
	// MaxDepth limits tree depth (0 = unlimited).
	MaxDepth *int `yaml:"max_depth,omitempty" yamlcomment:"Max tree depth (0 = unlimited)"`

	// MaxStringLength is the max chars for inline values before truncation.
	// 0 = auto (terminal-based when TTY, unlimited when piped).
	MaxStringLength *int `yaml:"max_string_length,omitempty" yamlcomment:"Max string length before truncation (0 = auto)"`

	// MaxArrayInline is max items to show inline for scalar arrays (default 3).
	MaxArrayInline *int `yaml:"max_array_inline,omitempty" yamlcomment:"Max array items to show inline (default: 3)"`

	// ExpandArrays shows all array elements instead of "[N items]" summary.
	ExpandArrays *bool `yaml:"expand_arrays,omitempty" yamlcomment:"Expand all array elements"`

	// NoValues hides values at leaf nodes (structure only).
	NoValues *bool `yaml:"no_values,omitempty" yamlcomment:"Show structure only (hide values)"`
}

// MermaidFormattingConfig controls Mermaid diagram output formatting.
type MermaidFormattingConfig struct {
	// Direction sets the diagram direction: TD (top-down), LR (left-right),
	// BT (bottom-top), RL (right-left). Default is TD.
	Direction *string `yaml:"direction,omitempty" yamlcomment:"Diagram direction: TD, LR, BT, RL (default: TD)"`

	// MaxDepth limits tree depth (0 = unlimited).
	MaxDepth *int `yaml:"max_depth,omitempty" yamlcomment:"Max diagram depth (0 = unlimited)"`

	// MaxStringLength is the max chars for inline values before truncation.
	MaxStringLength *int `yaml:"max_string_length,omitempty" yamlcomment:"Max string length before truncation (0 = auto)"`

	// MaxArrayInline is max items to show inline for scalar arrays (default 3).
	MaxArrayInline *int `yaml:"max_array_inline,omitempty" yamlcomment:"Max array items to show inline (default: 3)"`

	// ExpandArrays shows all array elements instead of "[N items]" summary.
	ExpandArrays *bool `yaml:"expand_arrays,omitempty" yamlcomment:"Expand all array elements"`

	// NoValues hides values at leaf nodes (structure only).
	NoValues *bool `yaml:"no_values,omitempty" yamlcomment:"Show structure only (hide values)"`
}

// IntellisenseConfig holds intellisense/completion configuration.
type IntellisenseConfig struct {
	MaxSuggestions *int `yaml:"max_suggestions,omitempty" yamlcomment:"Maximum number of suggestions to show"`
}

// FunctionExample holds a function's description and usage examples
type FunctionExample struct {
	Description string   `yaml:"description,omitempty"`
	Examples    []string `yaml:"examples,omitempty"`
}

// HelpConfig holds help text and function examples configuration.
type HelpConfig struct {
	ExprModeEntry    string                          `yaml:"expr_mode_entry,omitempty" yamlcomment:"Help text shown when entering expression mode"`
	FunctionHelp     map[string]string               `yaml:"function_help,omitempty" yamlcomment:"Custom help text overrides for specific functions"`
	FunctionExamples map[string]FunctionExampleValue `yaml:"function_examples,omitempty" yamlcomment:"[DEPRECATED] Use help.cel.function_examples"`
	CEL              CELHelpConfig                   `yaml:"cel,omitempty" yamlcomment:"Expression language help for CEL"`
}

// CELHelpConfig holds CEL-specific help configuration.
type CELHelpConfig struct {
	ExampleData      interface{}                     `yaml:"example_data,omitempty" yamlcomment:"Example data bound to '_' when validating CEL examples"`
	FunctionExamples map[string]FunctionExampleValue `yaml:"function_examples,omitempty" yamlcomment:"Usage examples for CEL functions"`
}

// FunctionExampleValue can be either an array of strings (legacy) or a FunctionExample struct
type FunctionExampleValue struct {
	Description string
	Examples    []string
}

// UnmarshalYAML implements custom unmarshaling to support both legacy and new formats
func (f *FunctionExampleValue) UnmarshalYAML(value *yaml.Node) error {
	// Try new format first (object with description and examples)
	var newFormat FunctionExample
	if err := value.Decode(&newFormat); err == nil && newFormat.Description != "" {
		f.Description = newFormat.Description
		f.Examples = newFormat.Examples
		return nil
	}

	// Fall back to legacy format (array of strings)
	var examples []string
	if err := value.Decode(&examples); err == nil {
		f.Examples = examples
		return nil
	}

	return fmt.Errorf("function_examples must be either an array of strings or an object with description and examples")
}

// PopupConfig holds popup and modal settings.
type PopupConfig struct {
	InfoPopup InfoPopupConfig `yaml:"info_popup,omitempty" yamlcomment:"Optional popup shown in a modal/inline"`
}

func (c *InfoPopupConfig) UnmarshalYAML(value *yaml.Node) error {
	if value == nil {
		return nil
	}
	switch value.Kind {
	case yaml.DocumentNode, yaml.SequenceNode, yaml.AliasNode:
		return fmt.Errorf("info_popup must be a string or map")
	case yaml.ScalarNode:
		c.Text = value.Value
		return nil
	case yaml.MappingNode:
		type alias InfoPopupConfig
		var tmp alias
		if err := value.Decode(&tmp); err != nil {
			return err
		}
		*c = InfoPopupConfig(tmp)
		return nil
	default:
		return fmt.Errorf("info_popup must be a string or map")
	}
}

// AppConfig holds application-level configuration (not UI-specific).
type AppConfig struct {
	About AboutConfig    `yaml:"about" yamlcomment:"Application metadata and information"`
	CLI   CLIConfig      `yaml:"cli" yamlcomment:"CLI-specific configuration"`
	Debug DebugConfig    `yaml:"debug" yamlcomment:"Debug and logging settings"`
	Help  HelpMenuConfig `yaml:"help,omitempty" yamlcomment:"Help menu information (populated dynamically from menu config)"`
}

// Config holds UI-specific configuration.
type Config struct {
	Theme        ThemeSelectionConfig   `yaml:"theme" yamlcomment:"Theme selection and configuration"`
	Features     FeaturesConfig         `yaml:"features" yamlcomment:"Feature flags - enable/disable UI features"`
	Intellisense IntellisenseConfig     `yaml:"intellisense,omitempty" yamlcomment:"Intellisense and completion settings"`
	Help         HelpConfig             `yaml:"help,omitempty" yamlcomment:"Help text and function examples"`
	Display      DisplayConfig          `yaml:"display" yamlcomment:"Display and layout settings"`
	Behavior     BehaviorConfig         `yaml:"behavior,omitempty" yamlcomment:"User behavior and interaction settings"`
	Performance  PerformanceConfig      `yaml:"performance,omitempty" yamlcomment:"Performance and optimization settings"`
	Search       SearchConfig           `yaml:"search,omitempty" yamlcomment:"Search and filtering settings"`
	Formatting   FormattingConfig       `yaml:"formatting,omitempty" yamlcomment:"Data formatting settings"`
	Popup        PopupConfig            `yaml:"popup" yamlcomment:"Popup and modal settings"`
	Themes       map[string]ThemeConfig `yaml:"themes" yamlcomment:"Theme definitions"`
	LegacyTheme  ThemeConfig            `yaml:",inline,omitempty"` // backward compatibility for single-theme files
	Menu         MenuConfigYAML         `yaml:"menu" yamlcomment:"Function key labels/actions"`
}

// ThemeConfigFile holds the complete configuration (app + ui).
// This is kept for backward compatibility and internal use.
type ThemeConfigFile struct {
	// Application-level settings
	About    AboutConfig    `yaml:"about" yamlcomment:"Application metadata and information"`
	CLI      CLIConfig      `yaml:"cli" yamlcomment:"CLI-specific configuration"`
	Debug    DebugConfig    `yaml:"debug" yamlcomment:"Debug and logging settings"`
	HelpMenu HelpMenuConfig `yaml:"help_menu,omitempty" yamlcomment:"Help menu information (populated dynamically from menu config)"`
	// UI-specific settings
	Theme        ThemeSelectionConfig   `yaml:"theme" yamlcomment:"Theme selection and configuration"`
	Features     FeaturesConfig         `yaml:"features" yamlcomment:"Feature flags - enable/disable UI features"`
	Intellisense IntellisenseConfig     `yaml:"intellisense,omitempty" yamlcomment:"Intellisense and completion settings"`
	Help         HelpConfig             `yaml:"help,omitempty" yamlcomment:"Help text and function examples"`
	Display      DisplayConfig          `yaml:"display" yamlcomment:"Display and layout settings"`
	Behavior     BehaviorConfig         `yaml:"behavior,omitempty" yamlcomment:"User behavior and interaction settings"`
	Performance  PerformanceConfig      `yaml:"performance,omitempty" yamlcomment:"Performance and optimization settings"`
	Search       SearchConfig           `yaml:"search,omitempty" yamlcomment:"Search and filtering settings"`
	Formatting   FormattingConfig       `yaml:"formatting,omitempty" yamlcomment:"Data formatting settings"`
	Popup        PopupConfig            `yaml:"popup" yamlcomment:"Popup and modal settings"`
	Themes       map[string]ThemeConfig `yaml:"themes" yamlcomment:"Theme definitions"`
	LegacyTheme  ThemeConfig            `yaml:",inline,omitempty"` // backward compatibility for single-theme files
	Menu         MenuConfigYAML         `yaml:"menu" yamlcomment:"Function key labels/actions"`
	// Legacy fields for backward compatibility (populated from new structure)
	DefaultTheme   string          `yaml:"default_theme,omitempty" yamlcomment:"[DEPRECATED] Use ui.theme.default instead"`
	AllowEditInput *bool           `yaml:"allow_edit_input,omitempty" yamlcomment:"[DEPRECATED] Use ui.features.allow_edit_input instead"`
	AllowFilter    *bool           `yaml:"allow_filter,omitempty" yamlcomment:"[DEPRECATED] Use ui.features.allow_filter instead"`
	AllowSuggest   *bool           `yaml:"allow_suggestions,omitempty" yamlcomment:"[DEPRECATED] Use ui.features.allow_suggestions instead"`
	AllowIntel     *bool           `yaml:"allow_intellisense,omitempty" yamlcomment:"[DEPRECATED] Use ui.features.allow_intellisense instead"`
	KeyColWidth    *int            `yaml:"key_col_width,omitempty" yamlcomment:"[DEPRECATED] Use ui.display.key_col_width instead"`
	ValueColWidth  *int            `yaml:"value_col_width,omitempty" yamlcomment:"[DEPRECATED] Use ui.display.value_col_width instead"`
	InfoPopup      InfoPopupConfig `yaml:"info_popup,omitempty" yamlcomment:"[DEPRECATED] Use ui.popup.info_popup instead"`
	DebugMaxEvents *int            `yaml:"debug_max_events,omitempty" yamlcomment:"[DEPRECATED] Use app.debug.max_events instead"`
}

// MenuConfigYAML represents menu configuration for YAML parsing.
// Uses action-based keys (help, search, filter, etc.) with mode-specific key bindings.
type MenuConfigYAML struct {
	// Action-based menu items
	Help   MenuItemConfig `yaml:"help,omitempty" yamlcomment:"Help action"`
	Search MenuItemConfig `yaml:"search,omitempty" yamlcomment:"Search action"`
	Filter MenuItemConfig `yaml:"filter,omitempty" yamlcomment:"Filter action"`
	Copy   MenuItemConfig `yaml:"copy,omitempty" yamlcomment:"Copy action"`
	Expr   MenuItemConfig `yaml:"expr,omitempty" yamlcomment:"Expression toggle action"`
	Quit   MenuItemConfig `yaml:"quit,omitempty" yamlcomment:"Quit action"`
	Custom MenuItemConfig `yaml:"custom,omitempty" yamlcomment:"Custom action"`

	// Legacy F-key based items (for backwards compatibility)
	F1  MenuItemConfig `yaml:"f1,omitempty" yamlcomment:"F1 menu item (legacy)"`
	F2  MenuItemConfig `yaml:"f2,omitempty" yamlcomment:"F2 menu item (legacy)"`
	F3  MenuItemConfig `yaml:"f3,omitempty" yamlcomment:"F3 menu item (legacy)"`
	F4  MenuItemConfig `yaml:"f4,omitempty" yamlcomment:"F4 menu item (legacy)"`
	F5  MenuItemConfig `yaml:"f5,omitempty" yamlcomment:"F5 menu item (legacy)"`
	F6  MenuItemConfig `yaml:"f6,omitempty" yamlcomment:"F6 menu item (legacy)"`
	F7  MenuItemConfig `yaml:"f7,omitempty" yamlcomment:"F7 menu item (legacy)"`
	F8  MenuItemConfig `yaml:"f8,omitempty" yamlcomment:"F8 menu item (legacy)"`
	F9  MenuItemConfig `yaml:"f9,omitempty" yamlcomment:"F9 menu item (legacy)"`
	F10 MenuItemConfig `yaml:"f10,omitempty" yamlcomment:"F10 menu item (legacy)"`
	F11 MenuItemConfig `yaml:"f11,omitempty" yamlcomment:"F11 menu item (legacy)"`
	F12 MenuItemConfig `yaml:"f12,omitempty" yamlcomment:"F12 menu item (legacy)"`
}

// MenuKeyBindings defines key bindings per mode for an action.
type MenuKeyBindings struct {
	Function string `yaml:"function,omitempty" yamlcomment:"Function key (e.g., f1, f3)"`
	Vim      string `yaml:"vim,omitempty" yamlcomment:"Vim mode key (e.g., ?, /, y)"`
	Emacs    string `yaml:"emacs,omitempty" yamlcomment:"Emacs mode key (e.g., ctrl+s, alt+w)"`
}

// MenuItemConfig describes a menu item in YAML.
type MenuItemConfig struct {
	Label     string          `yaml:"label" yamlcomment:"Display label"`
	Action    string          `yaml:"action,omitempty" yamlcomment:"Registered action name (defaults to key name)"`
	Enabled   *bool           `yaml:"enabled" yamlcomment:"Enable this keybinding"`
	Keys      MenuKeyBindings `yaml:"keys,omitempty" yamlcomment:"Mode-specific key bindings"`
	Popup     InfoPopupConfig `yaml:"popup,omitempty" yamlcomment:"Optional popup payload for this key"`
	PopupText string          `yaml:"popup_text,omitempty" yamlcomment:"(deprecated) Optional text payload for this key"`
	HelpText  string          `yaml:"help_text" yamlcomment:"Optional help overlay description for this key"`
}

// ThemeFromConfig builds a Theme from a ThemeConfig, falling back to defaults when fields are empty.
func ThemeFromConfig(cfg ThemeConfig) Theme {
	// Use fallbackDefaultTheme as the base to avoid recursive DefaultTheme() calls
	// when ThemeFromConfig is invoked during DefaultTheme initialization.
	return themeFromConfigWithBase(cfg, fallbackDefaultTheme())
}

// themeFromConfigWithBase builds a Theme from a ThemeConfig using the provided base theme.
// This avoids recursive DefaultTheme calls when DefaultTheme itself is constructing presets.
func themeFromConfigWithBase(cfg ThemeConfig, base Theme) Theme {
	th := base
	set := func(val ColorValue, dst *color.Color) {
		if val != "" {
			*dst = lipgloss.Color(string(val)) // lipgloss.Color() is now a function returning color.Color
		}
	}
	set(cfg.KeyColor, &th.KeyColor)
	set(cfg.ValueColor, &th.ValueColor)
	set(cfg.HeaderFG, &th.HeaderFG)
	set(cfg.HeaderBG, &th.HeaderBG)
	if cfg.BorderStyle != "" {
		th.BorderStyle = normalizeBorderStyle(cfg.BorderStyle)
	}
	set(cfg.SelectedFG, &th.SelectedFG)
	set(cfg.SelectedBG, &th.SelectedBG)
	set(cfg.SeparatorColor, &th.SeparatorColor)
	set(cfg.InputBG, &th.InputBG)
	set(cfg.InputFG, &th.InputFG)
	set(cfg.GhostFG, &th.GhostFG)
	set(cfg.StatusColor, &th.StatusColor)
	set(cfg.StatusError, &th.StatusError)
	set(cfg.StatusSuccess, &th.StatusSuccess)
	set(cfg.DebugColor, &th.DebugColor)
	set(cfg.FooterFG, &th.FooterFG)
	set(cfg.FooterBG, &th.FooterBG)
	set(cfg.HelpKey, &th.HelpKey)
	set(cfg.HelpValue, &th.HelpValue)
	th.BorderStyle = normalizeBorderStyle(th.BorderStyle)
	return th
}

// colorToString converts a color.Color interface to a ColorValue string.
// This is a best-effort conversion since color.Color is an interface without a String() method.
func colorToColorValue(c color.Color) ColorValue { //nolint:gosec // RGBA values are 16-bit; explicit scaling to 8-bit is safe
	if c == nil {
		return ""
	}
	r, g, b, a := c.RGBA()
	if a == 0 && r == 0 && g == 0 && b == 0 {
		return ""
	}
	// Normalize to 8-bit per channel hex string; RGBA returns 16-bit values so divide by 257 to scale safely.
	r8 := r / 257
	g8 := g / 257
	b8 := b / 257
	return ColorValue(fmt.Sprintf("#%02x%02x%02x", r8, g8, b8))
}

// ThemeConfigFromTheme converts a Theme into its YAML-friendly config form.
func ThemeConfigFromTheme(th Theme) ThemeConfig {
	return ThemeConfig{
		KeyColor:       colorToColorValue(th.KeyColor),
		ValueColor:     colorToColorValue(th.ValueColor),
		HeaderFG:       colorToColorValue(th.HeaderFG),
		HeaderBG:       colorToColorValue(th.HeaderBG),
		BorderStyle:    th.BorderStyle,
		SelectedFG:     colorToColorValue(th.SelectedFG),
		SelectedBG:     colorToColorValue(th.SelectedBG),
		SeparatorColor: colorToColorValue(th.SeparatorColor),
		InputBG:        colorToColorValue(th.InputBG),
		InputFG:        colorToColorValue(th.InputFG),
		GhostFG:        colorToColorValue(th.GhostFG),
		StatusColor:    colorToColorValue(th.StatusColor),
		StatusError:    colorToColorValue(th.StatusError),
		StatusSuccess:  colorToColorValue(th.StatusSuccess),
		DebugColor:     colorToColorValue(th.DebugColor),
		FooterFG:       colorToColorValue(th.FooterFG),
		FooterBG:       colorToColorValue(th.FooterBG),
		HelpKey:        colorToColorValue(th.HelpKey),
		HelpValue:      colorToColorValue(th.HelpValue),
	}
}

func normalizeBorderStyle(val string) string {
	v := strings.TrimSpace(strings.ToLower(val))
	switch v {
	case "", "normal", "square":
		return "normal"
	case "rounded", "round":
		return "rounded"
	default:
		return "normal"
	}
}

func borderForStyle(style string) lipgloss.Border {
	switch normalizeBorderStyle(style) {
	case "rounded":
		return lipgloss.RoundedBorder()
	default:
		return lipgloss.NormalBorder()
	}
}

func borderForTheme(th Theme) lipgloss.Border {
	return borderForStyle(th.BorderStyle)
}

func borderStyleForTheme(th Theme) lipgloss.Style { //nolint:unused // kept for potential theme-specific styling hooks
	return lipgloss.NewStyle().Border(borderForTheme(th)).BorderForeground(th.SeparatorColor)
}

// InfoPopupHasData reports whether any field is set on an InfoPopupConfig.
func InfoPopupHasData(cfg InfoPopupConfig) bool {
	return cfg.Text != "" || cfg.Title != "" || cfg.TitleJustify != "" || cfg.Anchor != "" || cfg.Justify != "" || cfg.Modal != nil || cfg.Permanent != nil || cfg.Enabled != nil
}

// LoadThemeFile reads a YAML theme file and returns a Theme.
func LoadThemeFile(path string) (Theme, error) {
	var cfg ThemeConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return Theme{}, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Theme{}, err
	}
	return ThemeFromConfig(cfg), nil
}

// LoadThemeConfig reads a config file that can contain multiple themes and settings.
// It supports the legacy single-theme format and the new themes map format.
func LoadThemeConfig(path string) (ThemeConfigFile, error) {
	var cfg ThemeConfigFile
	data, err := os.ReadFile(path)
	if err != nil {
		return ThemeConfigFile{}, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return ThemeConfigFile{}, err
	}
	// If Themes map is empty but inline LegacyTheme is set, build a map entry for backward compatibility.
	if len(cfg.Themes) == 0 && (cfg.LegacyTheme != ThemeConfig{}) {
		cfg.Themes = map[string]ThemeConfig{
			"default": cfg.LegacyTheme,
		}
		if cfg.Theme.Default == "" {
			cfg.Theme.Default = "dark"
		}
	}
	// Fallback default theme name
	if cfg.Theme.Default == "" {
		cfg.Theme.Default = "dark"
	}
	return cfg, nil
}

// InitializeThemes loads all themes from the provided configuration into loadedThemes.
// This is called at startup to populate themes from default_config.yaml and user config files.
// It should be called before any SetThemeByName() calls.
// If no themes are provided in the configuration, an error is returned.
func InitializeThemes(cfg *ThemeConfigFile) error {
	if cfg == nil {
		return fmt.Errorf("cannot initialize themes with nil configuration")
	}

	// Ensure loadedThemes is a fresh map
	loadedThemes = make(map[string]Theme)

	// Load all themes from the configuration
	if len(cfg.Themes) == 0 {
		return fmt.Errorf("no themes found in configuration")
	}

	// Convert ThemeConfig to Theme using ThemeFromConfig
	for name, themeCfg := range cfg.Themes {
		loadedThemes[name] = ThemeFromConfig(themeCfg)
	}

	// Ensure at least a "dark" theme exists as fallback
	if _, ok := loadedThemes["dark"]; !ok {
		loadedThemes["dark"] = DefaultTheme()
	}

	// Also sync ThemePresets for backward compatibility
	// This allows existing code that references ThemePresets to continue working
	for name, theme := range loadedThemes {
		ThemePresets[name] = theme
	}

	return nil
}
