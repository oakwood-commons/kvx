package ui

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/term"

	"github.com/oakwood-commons/kvx/internal/formatter"
	"github.com/oakwood-commons/kvx/internal/navigator"
	"github.com/oakwood-commons/kvx/pkg/loader"
)

// RunModel starts the Bubble Tea TUI using the Model implementation.
// Width/height of 0 will auto-detect the terminal size (falling back to defaults).
// Extra ProgramOptions (e.g., custom IO) can be provided to mirror tea.NewProgram.
func RunModel(appName string, root interface{}, helpTitle, helpText string, debugEnabled bool, debugSink func(string), initialExpr string, width, height int, startKeys []string, noColor bool, exprModeEntryHelp string, functionHelpOverrides map[string]string, configure func(*Model), opts ...tea.ProgramOption) error {
	_ = appName
	_ = helpTitle
	_ = helpText
	_ = exprModeEntryHelp
	_ = functionHelpOverrides

	m := InitialModel(root)
	m.Root = root
	m.Node = root
	m.NoColor = noColor
	m.DebugMode = debugEnabled
	m.AppName = strings.TrimSpace(appName)
	m.HelpTitle = strings.TrimSpace(helpTitle)
	m.HelpText = strings.TrimRight(helpText, "\n")
	if len(functionHelpOverrides) > 0 {
		m.FunctionHelpOverrides = functionHelpOverrides
	}

	if configure != nil {
		configure(&m)
	}

	// Eager auto-decode: recursively decode all serialized scalars at load time
	if m.AllowDecode && m.AutoDecode == "eager" {
		m.Root = loader.RecursiveDecode(m.Root)
		m.Node = m.Root
		// Regenerate table rows from the decoded tree
		newModel := m.NavigateTo(m.Node, m.Path)
		m = *newModel
	}

	applyInitialExpr(&m, initialExpr)

	if width > 0 || height > 0 {
		runW := width
		runH := height
		if runW <= 0 || runH <= 0 {
			if w, h, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
				if runW <= 0 {
					runW = w
				}
				if runH <= 0 {
					runH = h
				}
			}
		}
		if runW <= 0 {
			runW = 80
		}
		if runH <= 0 {
			runH = 24
		}
		m.ForceWindowSize = true
		m.DesiredWinWidth = runW
		m.DesiredWinHeight = runH
		m.WinWidth = runW
		m.WinHeight = runH
		opts = append(opts, tea.WithWindowSize(runW, runH))
	}

	if len(startKeys) > 0 {
		ApplyStartupKeys(&m, startKeys)
	}
	if containsF1(startKeys) {
		m.HelpVisible = true
	}

	m.ApplyColorScheme()
	m.applyLayout(true)
	m.syncAllComponents()

	prog := tea.NewProgram(&m, opts...)
	finalModel, err := prog.Run()
	if finalModel != nil {
		if fm, ok := finalModel.(*Model); ok && fm != nil {
			flushDebugEvents(fm, debugSink)
			printPendingCLIExpr(fm)
		}
	}
	return err
}

func applyInitialExpr(m *Model, expr string) {
	if m == nil {
		return
	}
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" {
		return
	}
	node, err := navigator.Navigate(m.Root, trimmed)
	if err != nil {
		m.ErrMsg = fmt.Sprintf("explore expression error: %v", err)
		m.StatusType = "error"
		return
	}
	isExpr := (strings.Contains(trimmed, "(") && strings.Contains(trimmed, ")")) || IsExpression(trimmed)
	if isExpr {
		normalizedPath := normalizePathForModel(trimmed)
		if navigatedNode, err := navigator.Resolve(m.Root, normalizedPath); err == nil {
			// Path-like expression: keep path so left arrow can navigate back up.
			node = navigatedNode
			m.NavigateTo(node, normalizedPath)
		} else {
			// Non-path expression: keep behavior of landing at root path.
			m.NavigateTo(node, "")
		}
		m.setExprResult(trimmed, node)
		m.PathInput.SetValue(trimmed)
		m.PathInput.SetCursor(len(trimmed))
		return
	}
	m.NavigateTo(node, normalizePathForModel(trimmed))
}

func flushDebugEvents(m *Model, debugSink func(string)) {
	if m == nil || debugSink == nil {
		return
	}
	for _, ev := range m.DebugEvents {
		debugSink(ev.Message)
	}
}

func printPendingCLIExpr(m *Model) {
	if m == nil {
		return
	}
	expr := strings.TrimSpace(m.PendingCLIExpr)
	if expr == "" {
		return
	}
	node, err := EvaluateExpression(expr, m.Root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "explore expression error: %v\n", err)
		return
	}
	fmt.Fprint(os.Stdout, renderCLIOutput(node, m.NoColor))
}

// CLIOutputRenderer lets embedders override how F10/quit renders values to stdout.
// It should mirror the CLI output conventions; the default renderer preserves newlines
// for scalar strings so multiline values print cleanly.
type CLIOutputRenderer func(node interface{}, noColor bool) string

var cliOutputRenderer CLIOutputRenderer = defaultCLIOutputRenderer

// SetCLIOutputRenderer overrides the renderer used for printing CLI output on quit.
// Passing nil restores the default renderer.
func SetCLIOutputRenderer(fn CLIOutputRenderer) {
	if fn == nil {
		cliOutputRenderer = defaultCLIOutputRenderer
		return
	}
	cliOutputRenderer = fn
}

func renderCLIOutput(node interface{}, noColor bool) string {
	return cliOutputRenderer(node, noColor)
}

func defaultCLIOutputRenderer(node interface{}, noColor bool) string {
	if node == nil {
		return "\n"
	}
	if arr, ok := node.([]interface{}); ok {
		if isSimpleScalarArray(arr) {
			var b strings.Builder
			for i, v := range arr {
				if i > 0 {
					b.WriteString("\n")
				}
				b.WriteString(formatter.StringifyPreserveNewlines(v))
			}
			if !strings.HasSuffix(b.String(), "\n") {
				b.WriteString("\n")
			}
			return b.String()
		}
	}
	if isCompositeNode(node) {
		return RenderNodeTable(node, noColor, DefaultKeyColWidth, 0, 0)
	}
	return fmt.Sprintf("%s\n", formatter.StringifyPreserveNewlines(node))
}
