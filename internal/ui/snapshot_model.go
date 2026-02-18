package ui

import (
	"strings"

	"github.com/oakwood-commons/kvx/pkg/loader"
)

// ModelSnapshotConfig configures snapshot rendering using the Model implementation.
type ModelSnapshotConfig struct {
	Width       int
	Height      int
	NoColor     bool
	HelpVisible bool
	HideFooter  bool // Hide the footer bar (for non-interactive display)
	StartKeys   []string
	InitialExpr string
	Configure   func(*Model)
	Root        interface{}
	AppName     string
	HelpTitle   string
	HelpText    string
}

// RenderModelSnapshot renders a snapshot using the Model code path.
func RenderModelSnapshot(node interface{}, cfg ModelSnapshotConfig) string {
	return renderModelLayoutSnapshot(node, cfg)
}

func renderModelLayoutSnapshot(node interface{}, cfg ModelSnapshotConfig) string {
	m := InitialModel(node)
	m.Root = node
	m.NoColor = cfg.NoColor
	m.HelpVisible = cfg.HelpVisible
	// Disable debouncing in snapshot mode so search results appear immediately
	m.SearchDebounceMs = 0
	if cfg.Width > 0 {
		m.WinWidth = cfg.Width
	} else {
		m.WinWidth = 80
	}
	if cfg.Height > 0 {
		m.WinHeight = cfg.Height
	} else {
		m.WinHeight = 24
	}
	if cfg.InitialExpr != "" {
		m.PathInput.SetValue(cfg.InitialExpr)
	}
	if cfg.Configure != nil {
		cfg.Configure(&m)
	}
	// Eager auto-decode: recursively decode all serialized scalars at load time
	if m.AllowDecode && m.AutoDecode == "eager" {
		m.Root = loader.RecursiveDecode(m.Root)
		m.Node = m.Root
		newModel := m.NavigateTo(m.Node, m.Path)
		m = *newModel
	}
	if len(cfg.StartKeys) > 0 {
		ApplyStartupKeys(&m, cfg.StartKeys)
	}
	if containsF1(cfg.StartKeys) {
		m.HelpVisible = true
	}
	if strings.TrimSpace(cfg.HelpText) != "" {
		// When an explicit help text is provided (e.g., CLI popup text), avoid rendering an additional popup.
		m.HelpPopupText = ""
	}
	m.ApplyColorScheme()
	m.applyLayout(true)
	m.syncAllComponents()

	inputVisible := m.InputFocused || m.AdvancedSearchActive || m.MapFilterActive

	// Use explicit help text if provided, otherwise regenerate based on KeyMode
	helpText := strings.TrimSpace(cfg.HelpText)
	if helpText == "" && m.KeyMode != "" {
		menu := CurrentMenuConfig()
		helpText = strings.TrimSpace(GenerateHelpText(menu, m.AllowEditInput, nil, m.KeyMode))
	}

	state := panelLayoutStateFromModel(&m, PanelLayoutModelOptions{
		AppName:        cfg.AppName,
		HelpTitle:      cfg.HelpTitle,
		HelpText:       helpText,
		SnapshotHeader: true,
		InputVisible:   inputVisible,
		HideFooter:     cfg.HideFooter,
	})
	view := RenderPanelLayout(state)
	if cfg.NoColor {
		view = stripANSIExceptInverse(view)
	}
	if cfg.Height > 0 {
		view = padSnapshotHeight(view, cfg.Height, cfg.Width)
	}
	return view
}

func padSnapshotHeight(view string, height, width int) string {
	if height <= 0 {
		return view
	}
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	if len(lines) >= height {
		return strings.Join(lines, "\n")
	}
	padLine := " "
	if width > 1 {
		padLine = strings.Repeat(" ", width)
	}
	for len(lines) < height {
		lines = append(lines, padLine)
	}
	return strings.Join(lines, "\n")
}
