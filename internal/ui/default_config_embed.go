package ui

import (
	_ "embed"
	"fmt"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed default_config.yaml
var embeddedDefaultConfig []byte

var (
	embeddedConfigOnce sync.Once
	embeddedConfig     ThemeConfigFile
	embeddedConfigErr  error
)

// DefaultConfigYAML returns a copy of the embedded default config YAML bytes.
func DefaultConfigYAML() []byte {
	return append([]byte(nil), embeddedDefaultConfig...)
}

// EmbeddedDefaultConfig parses and returns the embedded default configuration.
// This is used as the single source of truth for default settings and themes.
func EmbeddedDefaultConfig() (ThemeConfigFile, error) {
	embeddedConfigOnce.Do(func() {
		if len(embeddedDefaultConfig) == 0 {
			embeddedConfigErr = fmt.Errorf("embedded default config is empty")
			return
		}
		var raw struct {
			App AppConfig `yaml:"app"`
			UI  Config    `yaml:"ui"`
		}
		if err := yaml.Unmarshal(embeddedDefaultConfig, &raw); err != nil {
			embeddedConfigErr = fmt.Errorf("decode embedded default config: %w", err)
			return
		}
		embeddedConfig = ThemeConfigFile{
			About:        raw.App.About,
			CLI:          raw.App.CLI,
			Debug:        raw.App.Debug,
			HelpMenu:     raw.App.Help,
			Theme:        raw.UI.Theme,
			Features:     raw.UI.Features,
			Intellisense: raw.UI.Intellisense,
			Help:         raw.UI.Help,
			Display:      raw.UI.Display,
			Behavior:     raw.UI.Behavior,
			Performance:  raw.UI.Performance,
			Search:       raw.UI.Search,
			Formatting:   raw.UI.Formatting,
			Popup:        raw.UI.Popup,
			Themes:       raw.UI.Themes,
			Menu:         raw.UI.Menu,
		}
		if embeddedConfig.Themes == nil {
			embeddedConfig.Themes = map[string]ThemeConfig{}
		}
	})
	return embeddedConfig, embeddedConfigErr
}
