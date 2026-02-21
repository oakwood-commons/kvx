package tui

import (
	"io"
	"os"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/google/cel-go/cel"
	"golang.org/x/term"

	"github.com/oakwood-commons/kvx/internal/ui"
)

// defaultFallbackTermWidth is used when terminal size cannot be detected.
const defaultFallbackTermWidth = 120

// DetectTerminalSize returns the best-effort terminal width and height by probing
// stdout, stderr, and stdin, then falling back to the COLUMNS environment variable.
// If detection fails completely, returns generous defaults (120, 24) to avoid
// overly narrow output in CI or non-TTY environments.
//
// This is useful for library consumers who want auto-sizing behavior:
//
//	width, _ := tui.DetectTerminalSize()
//	fmt.Print(tui.RenderBorderedTable(data, tui.BorderedTableOptions{Width: width}))
//
// Alternatively, pass Width: 0 to RenderBorderedTable for automatic detection.
func DetectTerminalSize() (width int, height int) {
	fds := []uintptr{os.Stdout.Fd(), os.Stderr.Fd(), os.Stdin.Fd()}
	for _, fd := range fds {
		if w, h, err := term.GetSize(int(fd)); err == nil && (w > 0 || h > 0) {
			return w, h
		}
	}
	if col := os.Getenv("COLUMNS"); col != "" {
		if w, err := strconv.Atoi(col); err == nil && w > 0 {
			return w, 0
		}
	}
	return defaultFallbackTermWidth, 24
}

// ExpressionProvider matches the UI expression provider interface.
type ExpressionProvider interface {
	Evaluate(expr string, root interface{}) (interface{}, error)
	DiscoverSuggestions() []string
	IsExpression(expr string) bool
}

// SetExpressionProvider sets a global expression provider for the TUI.
// Host applications can call this before launching the UI to plug in a custom evaluator.
func SetExpressionProvider(p ExpressionProvider) {
	ui.SetExpressionProvider(p)
}

// ResetExpressionProvider restores the default CEL expression provider.
// Useful for testing or resetting state.
func ResetExpressionProvider() {
	ui.ResetExpressionProvider()
}

// DefaultExpressionProvider returns the built-in CEL-based provider.
func DefaultExpressionProvider() ExpressionProvider {
	return ui.DefaultExpressionProvider()
}

// NewCELExpressionProvider creates an ExpressionProvider from a CEL environment.
// This helper makes it easy to use custom CEL environments (with extended functions)
// with the TUI. The environment should include the "_" variable for root data access.
//
// exampleHints is optional and can provide helpful usage examples for functions.
// For example: map[string]string{"myFunc": "e.g. myFunc(arg)"}
//
// Example:
//
//	env, _ := cel.NewEnv(
//	    cel.Variable("_", cel.DynType),
//	    cel.Function("myFunc", ...),
//	)
//	provider := tui.NewCELExpressionProvider(env, nil)
//	tui.SetExpressionProvider(provider)
func NewCELExpressionProvider(env *cel.Env, exampleHints map[string]string) ExpressionProvider {
	return ui.NewCELExpressionProvider(env, exampleHints)
}

// Run starts the shell-backed Bubble Tea TUI with the provided root data and config.
// Host applications can pass optional tea.ProgramOption values to control IO.
func Run(root interface{}, cfg Config, opts ...tea.ProgramOption) error {
	cfg.Apply()

	appName := strings.TrimSpace(cfg.AppName)
	if appName == "" {
		appName = "kvx"
	}

	menu := ui.DefaultMenuConfig()
	if cfg.Menu != nil {
		menu = *cfg.Menu
	}
	allowEdit := true
	if cfg.AllowEditInput != nil {
		allowEdit = *cfg.AllowEditInput
	}

	helpTitle := "Help"
	if strings.TrimSpace(cfg.HelpAboutTitle) != "" {
		helpTitle = strings.TrimSpace(cfg.HelpAboutTitle)
	}
	helpText := strings.TrimSpace(strings.Join(cfg.HelpAboutLines, "\n"))
	if helpText == "" {
		helpText = strings.TrimSpace(ui.GenerateHelpText(menu, allowEdit, cfg.HelpNavigationDescriptions, ui.DefaultKeyMode))
	}

	configure := func(m *ui.Model) {
		if cfg.AllowEditInput != nil {
			m.AllowEditInput = *cfg.AllowEditInput
		}
		if cfg.AllowFilter != nil {
			m.AllowFilter = *cfg.AllowFilter
		}
		if cfg.AllowSuggestions != nil {
			m.AllowSuggestions = *cfg.AllowSuggestions
		}
		if cfg.AllowIntellisense != nil {
			m.AllowIntellisense = *cfg.AllowIntellisense
		}
		if strings.TrimSpace(cfg.KeyHeader) != "" {
			m.KeyHeader = cfg.KeyHeader
		}
		if strings.TrimSpace(cfg.ValueHeader) != "" {
			m.ValueHeader = cfg.ValueHeader
		}
		if strings.TrimSpace(cfg.InputPromptUnfocused) != "" {
			m.InputPromptUnfocused = cfg.InputPromptUnfocused
		}
		if strings.TrimSpace(cfg.InputPromptFocused) != "" {
			m.InputPromptFocused = cfg.InputPromptFocused
		}
		if strings.TrimSpace(cfg.InputPlaceholder) != "" {
			m.InputPlaceholder = cfg.InputPlaceholder
			m.PathInput.Placeholder = cfg.InputPlaceholder
		}
		if strings.TrimSpace(cfg.HelpAboutTitle) != "" {
			m.HelpAboutTitle = cfg.HelpAboutTitle
		}
		if len(cfg.HelpAboutLines) > 0 {
			m.HelpAboutLines = cfg.HelpAboutLines
		}
		if cfg.HelpNavigationDescriptions != nil {
			m.HelpNavigationDescriptions = cfg.HelpNavigationDescriptions
		}
		if cfg.AllowDecode != nil {
			m.AllowDecode = *cfg.AllowDecode
		}
		if cfg.AutoDecode != "" {
			m.AutoDecode = cfg.AutoDecode
		}
		if cfg.DisplaySchema != nil {
			m.DisplaySchema = cfg.DisplaySchema
		}
		if cfg.Done != nil {
			m.DoneChan = cfg.Done
		}
		if cfg.KeyMode != "" && ui.IsValidKeyMode(cfg.KeyMode) {
			m.KeyMode = ui.KeyMode(cfg.KeyMode)
		}
	}

	return ui.RunModel(appName, root, helpTitle, helpText, cfg.DebugEnabled, cfg.DebugSink, cfg.InitialExpr, cfg.Width, cfg.Height, cfg.StartKeys, cfg.NoColor, cfg.ExprModeEntryHelp, cfg.FunctionHelpOverrides, configure, opts...)
}

// RenderSnapshot renders a fullscreen snapshot of the TUI and returns it as a string.
// This is useful for non-interactive display in the alt-screen buffer.
// The caller is responsible for printing to stdout and handling alt-screen transitions.
func RenderSnapshot(root interface{}, cfg Config) string {
	cfg.Apply()

	appName := strings.TrimSpace(cfg.AppName)
	if appName == "" {
		appName = "kvx"
	}

	menu := ui.DefaultMenuConfig()
	if cfg.Menu != nil {
		menu = *cfg.Menu
	}
	allowEdit := true
	if cfg.AllowEditInput != nil {
		allowEdit = *cfg.AllowEditInput
	}

	helpTitle := "Help"
	if strings.TrimSpace(cfg.HelpAboutTitle) != "" {
		helpTitle = strings.TrimSpace(cfg.HelpAboutTitle)
	}
	helpText := strings.TrimSpace(strings.Join(cfg.HelpAboutLines, "\n"))
	if helpText == "" {
		helpText = strings.TrimSpace(ui.GenerateHelpText(menu, allowEdit, cfg.HelpNavigationDescriptions, ui.DefaultKeyMode))
	}

	snapCfg := ui.ModelSnapshotConfig{
		Root:       root,
		Width:      cfg.Width,
		Height:     cfg.Height,
		NoColor:    cfg.NoColor,
		HideFooter: cfg.HideFooter,
		StartKeys:  cfg.StartKeys,
		AppName:    appName,
		HelpTitle:  helpTitle,
		HelpText:   helpText,
	}
	if cfg.InitialExpr != "" {
		snapCfg.InitialExpr = cfg.InitialExpr
	}
	snapCfg.Configure = func(m *ui.Model) {
		if cfg.AllowEditInput != nil {
			m.AllowEditInput = *cfg.AllowEditInput
		}
		if cfg.AllowFilter != nil {
			m.AllowFilter = *cfg.AllowFilter
		}
		if cfg.AllowSuggestions != nil {
			m.AllowSuggestions = *cfg.AllowSuggestions
		}
		if cfg.AllowIntellisense != nil {
			m.AllowIntellisense = *cfg.AllowIntellisense
		}
		if cfg.AllowDecode != nil {
			m.AllowDecode = *cfg.AllowDecode
		}
		if cfg.AutoDecode != "" {
			m.AutoDecode = cfg.AutoDecode
		}
		if cfg.DisplaySchema != nil {
			m.DisplaySchema = cfg.DisplaySchema
		}
		if cfg.KeyMode != "" && ui.IsValidKeyMode(cfg.KeyMode) {
			m.KeyMode = ui.KeyMode(cfg.KeyMode)
		}
	}

	return ui.RenderModelSnapshot(root, snapCfg)
}

// WithIO returns tea.ProgramOptions to set custom input/output.
func WithIO(in io.Reader, out io.Writer) []tea.ProgramOption {
	opts := []tea.ProgramOption{}
	if in != nil {
		opts = append(opts, tea.WithInput(in))
	}
	if out != nil {
		opts = append(opts, tea.WithOutput(out))
	}
	return opts
}
