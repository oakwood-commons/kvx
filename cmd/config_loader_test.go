package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
