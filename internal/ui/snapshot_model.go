package ui

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/oakwood-commons/kvx/pkg/loader"
)

// snapshotSubmitTimeout bounds how long RenderModelSnapshot waits for a
// StartKeys-triggered OnStatusInput callback to return.
var snapshotSubmitTimeout = 5 * time.Second

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
	// Trigger custom view mode detection (list/detail) now that DisplaySchema may be set.
	m.updateViewMode(node)
	// Eager auto-decode: recursively decode all serialized scalars at load time
	if m.AllowDecode && m.AutoDecode == "eager" {
		m.Root = loader.RecursiveDecode(m.Root)
		m.Node = m.Root
		newModel := m.NavigateTo(m.Node, m.Path)
		m = *newModel
	}
	if len(cfg.StartKeys) > 0 {
		ApplyStartupKeys(&m, cfg.StartKeys)
		settleSnapshotStatusSubmission(&m)
		m.PendingStartupCmds = nil
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
		SnapshotHeader: m.ViewMode != "status",
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

// settleSnapshotStatusSubmission executes a StartKeys-triggered status input
// submission while concurrently servicing DoneChan so callbacks that write a
// StatusResult to an unbuffered Config.Done channel cannot deadlock.
//
// Termination policy:
//   - Consumes at most one StatusResult from DoneChan while waiting for the
//     submission command to return, then drains DoneChan non-blockingly once
//     after submission completes (covering buffered channels).
//   - Bounds total wait time by snapshotSubmitTimeout so a stalled callback
//     cannot block RenderModelSnapshot indefinitely. If timeout fires before
//     DoneChan delivers a result, a one-shot background select drains at most
//     one late DoneChan send until submitCmd returns so an unbuffered Done
//     channel cannot deadlock the timed-out callback goroutine.
func settleSnapshotStatusSubmission(m *Model) {
	if m == nil || m.ViewMode != "status" || m.StatusViewState == nil ||
		!m.StatusViewState.InputSubmitting || m.StatusViewState.lastSubmitCmd == nil {
		return
	}

	submitCmd := m.StatusViewState.lastSubmitCmd
	doneChan := m.StatusViewState.DoneChan

	submitResCh := make(chan tea.Msg, 1)
	go func() {
		submitResCh <- submitCmd()
	}()

	timer := time.NewTimer(snapshotSubmitTimeout)
	defer timer.Stop()

	var (
		submitMsg      tea.Msg
		gotSubmit      bool
		pendingDoneMsg statusDoneMsg
		gotDone        bool
	)

	for !gotSubmit {
		select {
		case submitMsg = <-submitResCh:
			gotSubmit = true
		case doneRes, ok := <-doneChan:
			if ok {
				pendingDoneMsg = statusDoneMsg(doneRes)
				gotDone = true
			}
			// Stop selecting on doneChan after receiving one completion or close.
			doneChan = nil
		case <-timer.C:
			m.StatusViewState.InputSubmitting = false
			m.StatusViewState.lastSubmitCmd = nil
			if !gotDone && doneChan != nil {
				select {
				case doneRes, ok := <-doneChan:
					if ok {
						pendingDoneMsg = statusDoneMsg(doneRes)
						gotDone = true
					}
				default:
					go func(ch <-chan StatusResult) {
						select {
						case <-submitResCh:
						case <-ch:
						}
					}(doneChan)
				}
			}
			if gotDone {
				if updated, _ := m.Update(pendingDoneMsg); updated != nil {
					if um, ok := updated.(*Model); ok {
						*m = *um
					}
				}
			}
			return
		}
	}

	if !gotDone && doneChan != nil {
		select {
		case doneRes, ok := <-doneChan:
			if ok {
				pendingDoneMsg = statusDoneMsg(doneRes)
				gotDone = true
			}
		default:
		}
	}

	if updated, _ := m.Update(submitMsg); updated != nil {
		if um, ok := updated.(*Model); ok {
			*m = *um
		}
	}
	if gotDone {
		if updated, _ := m.Update(pendingDoneMsg); updated != nil {
			if um, ok := updated.(*Model); ok {
				*m = *um
			}
		}
	}
}
