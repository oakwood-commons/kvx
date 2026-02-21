package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// statusPhase represents the lifecycle phase of the status screen.
type statusPhase int

const (
	statusPhaseWaiting statusPhase = iota
	statusPhaseSuccess
	statusPhaseError
)

// StatusViewModel holds state for the status screen view mode.
type StatusViewModel struct {
	Config  *StatusDisplayConfig
	Data    any // The loaded data node (map[string]any or similar)
	KeyMode KeyMode
	NoColor bool

	// Async completion
	DoneChan  <-chan StatusResult // From Config.Done (programmatic)
	HasDone   bool                // Whether a completion source is active
	Phase     statusPhase
	ResultMsg string // Message from StatusResult or timeout

	// Spinner
	Spinner spinner.Model

	// Action flash messages
	FlashMsg   string
	FlashTimer int // Correlates with flash clear messages

	// Dimensions
	Width  int
	Height int
}

// statusDoneMsg is sent when the async operation or timeout completes.
type statusDoneMsg struct {
	Err     error
	Message string
}

// statusFlashClearMsg clears an action flash message after a delay.
type statusFlashClearMsg struct {
	ID int
}

// statusTimeoutMsg is sent when the schema-defined timeout expires.
type statusTimeoutMsg struct{}

// statusDoneTimerMsg is sent after the done-delay to trigger exit.
type statusDoneTimerMsg struct{}

// buildStatusViewModel creates a StatusViewModel from the schema config and data.
func buildStatusViewModel(data any, schema *DisplaySchema, keyMode KeyMode, noColor bool, doneChan <-chan StatusResult, width, height int) *StatusViewModel {
	if schema == nil || schema.Status == nil {
		return nil
	}

	s := spinner.New()
	s.Spinner = spinner.Dot

	hasDone := doneChan != nil
	if !hasDone && schema.Status.Timeout != "" {
		if _, err := time.ParseDuration(schema.Status.Timeout); err == nil {
			hasDone = true
		}
	}

	return &StatusViewModel{
		Config:   schema.Status,
		Data:     data,
		KeyMode:  keyMode,
		NoColor:  noColor,
		DoneChan: doneChan,
		HasDone:  hasDone,
		Phase:    statusPhaseWaiting,
		Spinner:  s,
		Width:    width,
		Height:   height,
	}
}

// Init returns the initial commands for the status view (spinner tick + completion source).
func (sv *StatusViewModel) Init() tea.Cmd {
	cmds := []tea.Cmd{sv.Spinner.Tick}

	if sv.DoneChan != nil {
		// Listen on the programmatic Done channel
		cmds = append(cmds, waitForDone(sv.DoneChan))
	} else if sv.Config.Timeout != "" {
		// Start the schema-defined timeout
		if d, err := time.ParseDuration(sv.Config.Timeout); err == nil {
			cmds = append(cmds, startTimeout(d))
		}
	}

	return tea.Batch(cmds...)
}

// Update handles messages for the status view.
func (sv *StatusViewModel) Update(msg tea.Msg) (CustomView, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if sv.Phase == statusPhaseWaiting {
			var cmd tea.Cmd
			sv.Spinner, cmd = sv.Spinner.Update(msg)
			return sv, cmd
		}
		return sv, nil

	case statusDoneMsg:
		if msg.Err != nil {
			sv.Phase = statusPhaseError
			sv.ResultMsg = msg.Err.Error()
		} else {
			sv.Phase = statusPhaseSuccess
			switch {
			case msg.Message != "":
				sv.ResultMsg = msg.Message
			case sv.Config.SuccessMessage != "":
				sv.ResultMsg = sv.Config.SuccessMessage
			default:
				sv.ResultMsg = "Done"
			}
		}
		return sv, sv.doneDelayCmd()

	case statusTimeoutMsg:
		sv.Phase = statusPhaseSuccess
		if sv.Config.SuccessMessage != "" {
			sv.ResultMsg = sv.Config.SuccessMessage
		} else {
			sv.ResultMsg = "Done"
		}
		return sv, sv.doneDelayCmd()

	case statusDoneTimerMsg:
		return sv, tea.Quit

	case statusFlashClearMsg:
		if msg.ID == sv.FlashTimer {
			sv.FlashMsg = ""
		}
		return sv, nil

	case tea.KeyPressMsg:
		return sv.handleKey(msg)
	}

	return sv, nil
}

// handleKey processes key presses on the status screen.
func (sv *StatusViewModel) handleKey(msg tea.KeyPressMsg) (*StatusViewModel, tea.Cmd) {
	keyStr := msg.String()

	// Universal quit keys
	if keyStr == "ctrl+c" || keyStr == "esc" {
		return sv, tea.Quit
	}

	// If we're in a done phase and waiting for key, any key exits
	if sv.Phase != statusPhaseWaiting && sv.doneBehavior() == DoneBehaviorWaitForKey {
		return sv, tea.Quit
	}

	// Check mode-specific quit key
	if sv.isQuitKey(keyStr) {
		return sv, tea.Quit
	}

	// Check action keys
	for _, action := range sv.Config.Actions {
		actionKey := sv.keyForMode(action.Keys)
		if actionKey != "" && keyStr == actionKey {
			return sv.executeAction(action)
		}
	}

	return sv, nil
}

// executeAction runs a built-in action handler and shows a flash message.
func (sv *StatusViewModel) executeAction(action StatusActionConfig) (*StatusViewModel, tea.Cmd) {
	fieldValue := sv.getFieldValue(action.Field)
	if fieldValue == "" {
		sv.FlashMsg = fmt.Sprintf("⚠ %s: field %q not found", action.Label, action.Field)
		sv.FlashTimer++
		return sv, clearFlashAfter(sv.FlashTimer, 2*time.Second)
	}

	var err error
	switch action.Type {
	case "copy-value":
		err = CopyToClipboard(fieldValue)
	case "open-url":
		err = OpenURL(fieldValue)
	}

	if err != nil {
		sv.FlashMsg = fmt.Sprintf("⚠ %s: %v", action.Label, err)
	} else {
		switch action.Type {
		case "copy-value":
			sv.FlashMsg = "✓ Copied to clipboard"
		case "open-url":
			sv.FlashMsg = "✓ Opened in browser"
		default:
			sv.FlashMsg = fmt.Sprintf("✓ %s", action.Label)
		}
	}

	sv.FlashTimer++
	return sv, clearFlashAfter(sv.FlashTimer, 2*time.Second)
}

// View renders the status screen.
func (sv *StatusViewModel) View() string {
	th := CurrentTheme()

	var sections []string

	// Title is rendered in the panel border (set by panelLayoutStateFromModel),
	// so we skip it here to avoid duplication.

	// Messages
	messages := sv.getMessages()
	for _, msg := range messages {
		msgStyle := lipgloss.NewStyle()
		if !sv.NoColor && th.StatusColor != nil {
			msgStyle = msgStyle.Foreground(th.StatusColor)
		}
		sections = append(sections, "  "+msgStyle.Render(msg))
	}
	if len(messages) > 0 {
		sections = append(sections, "")
	}

	// Display fields (labeled key-value pairs)
	for _, df := range sv.Config.DisplayFields {
		val := sv.getFieldValue(df.Field)
		if val == "" {
			continue
		}
		labelStyle := lipgloss.NewStyle().Bold(true)
		if !sv.NoColor && th.HeaderFG != nil {
			labelStyle = labelStyle.Foreground(th.HeaderFG)
		}
		display := val
		if isURL(val) {
			display = ansi.SetHyperlink(val) + val + ansi.ResetHyperlink()
		}
		sections = append(sections, "  "+labelStyle.Render(df.Label+":")+" "+display)
	}
	if len(sv.Config.DisplayFields) > 0 {
		sections = append(sections, "")
	}

	// Phase-specific content
	switch sv.Phase {
	case statusPhaseWaiting:
		if sv.HasDone && sv.Config.WaitMessage != "" {
			spinnerView := sv.Spinner.View()
			waitStyle := lipgloss.NewStyle()
			if !sv.NoColor && th.StatusColor != nil {
				waitStyle = waitStyle.Foreground(th.StatusColor)
			}
			sections = append(sections, "  "+spinnerView+" "+waitStyle.Render(sv.Config.WaitMessage))
			sections = append(sections, "")
		}
	case statusPhaseSuccess:
		successStyle := lipgloss.NewStyle().Bold(true)
		if !sv.NoColor && th.StatusSuccess != nil {
			successStyle = successStyle.Foreground(th.StatusSuccess)
		}
		sections = append(sections, "  "+successStyle.Render("✓ "+sv.ResultMsg))
		sections = append(sections, "")
		if sv.doneBehavior() == DoneBehaviorWaitForKey {
			sections = append(sections, "  Press any key to exit")
			sections = append(sections, "")
		}
	case statusPhaseError:
		errorStyle := lipgloss.NewStyle().Bold(true)
		if !sv.NoColor && th.StatusError != nil {
			errorStyle = errorStyle.Foreground(th.StatusError)
		}
		sections = append(sections, "  "+errorStyle.Render("✗ "+sv.ResultMsg))
		sections = append(sections, "")
		if sv.doneBehavior() == DoneBehaviorWaitForKey {
			sections = append(sections, "  Press any key to exit")
			sections = append(sections, "")
		}
	}

	content := strings.Join(sections, "\n")

	return content
}

// renderActionBar renders the footer-style action key labels.
func (sv *StatusViewModel) renderActionBar() string {
	th := CurrentTheme()
	fkeyStyle := lipgloss.NewStyle().Bold(true)
	if !sv.NoColor {
		if th.FooterFG != nil {
			fkeyStyle = fkeyStyle.Foreground(th.FooterFG)
		}
		if th.FooterBG != nil {
			fkeyStyle = fkeyStyle.Background(th.FooterBG)
		}
	}

	parts := []string{}

	// Configured actions
	for _, action := range sv.Config.Actions {
		key := sv.displayKeyForMode(action.Keys)
		if key != "" && action.Label != "" {
			parts = append(parts, fkeyStyle.Render(key), action.Label)
		}
	}

	// Quit action (always present)
	quitKey := sv.quitKeyDisplay()
	parts = append(parts, fkeyStyle.Render(quitKey), "quit")

	return strings.Join(parts, " ")
}

// getTitleText extracts the title from the data using the schema's TitleField.
func (sv *StatusViewModel) getTitleText() string {
	if sv.Config.TitleField == "" {
		return ""
	}
	return sv.getFieldValue(sv.Config.TitleField)
}

// getMessages extracts messages from the data using the schema's MessageField.
func (sv *StatusViewModel) getMessages() []string {
	if sv.Config.MessageField == "" {
		return nil
	}
	data, ok := sv.Data.(map[string]any)
	if !ok {
		return nil
	}
	val, ok := data[sv.Config.MessageField]
	if !ok {
		return nil
	}
	// String value
	if s, ok := val.(string); ok {
		return []string{s}
	}
	// Array value
	if arr, ok := val.([]any); ok {
		msgs := make([]string, 0, len(arr))
		for _, v := range arr {
			if s, ok := v.(string); ok {
				msgs = append(msgs, s)
			} else {
				msgs = append(msgs, fmt.Sprintf("%v", v))
			}
		}
		return msgs
	}
	return []string{fmt.Sprintf("%v", val)}
}

// getFieldValue extracts a string value from the data by field name.
func (sv *StatusViewModel) getFieldValue(field string) string {
	data, ok := sv.Data.(map[string]any)
	if !ok {
		return ""
	}
	val, ok := data[field]
	if !ok {
		return ""
	}
	return fmt.Sprintf("%v", val)
}

// keyForMode returns the raw key binding for the active KeyMode.
func (sv *StatusViewModel) keyForMode(keys StatusKeyBindings) string {
	switch sv.KeyMode {
	case KeyModeEmacs:
		return keys.Emacs
	case KeyModeFunction:
		return keys.Function
	case KeyModeVim:
		return keys.Vim
	default:
		return keys.Vim
	}
}

// displayKeyForMode returns the formatted key label for the active KeyMode.
func (sv *StatusViewModel) displayKeyForMode(keys StatusKeyBindings) string {
	switch sv.KeyMode {
	case KeyModeEmacs:
		return formatEmacsKey(keys.Emacs)
	case KeyModeFunction:
		return strings.ToUpper(keys.Function)
	case KeyModeVim:
		return keys.Vim
	default:
		return keys.Vim
	}
}

// isQuitKey checks if the key matches the quit binding for the current mode.
func (sv *StatusViewModel) isQuitKey(keyStr string) bool {
	switch sv.KeyMode {
	case KeyModeEmacs:
		return keyStr == "ctrl+q"
	case KeyModeFunction:
		return keyStr == "f10"
	case KeyModeVim:
		return keyStr == "q"
	default:
		return keyStr == "q"
	}
}

// quitKeyDisplay returns the display label for the quit key in the current mode.
func (sv *StatusViewModel) quitKeyDisplay() string {
	switch sv.KeyMode {
	case KeyModeEmacs:
		return "C-q"
	case KeyModeFunction:
		return "F10"
	case KeyModeVim:
		return "q"
	default:
		return "q"
	}
}

// doneBehavior returns the effective done behavior from the config.
func (sv *StatusViewModel) doneBehavior() string {
	if sv.Config.DoneBehavior == DoneBehaviorWaitForKey {
		return DoneBehaviorWaitForKey
	}
	return DoneBehaviorExitAfterDelay
}

// doneDelayCmd returns a command that delays and then sends statusDoneTimerMsg.
func (sv *StatusViewModel) doneDelayCmd() tea.Cmd {
	if sv.doneBehavior() == DoneBehaviorWaitForKey {
		return nil
	}
	delay := 2 * time.Second
	if sv.Config.DoneDelay != "" {
		if d, err := time.ParseDuration(sv.Config.DoneDelay); err == nil {
			delay = d
		}
	}
	return func() tea.Msg {
		time.Sleep(delay)
		return statusDoneTimerMsg{}
	}
}

// waitForDone returns a tea.Cmd that blocks on the done channel and sends statusDoneMsg.
func waitForDone(ch <-chan StatusResult) tea.Cmd {
	return func() tea.Msg {
		result := <-ch
		return statusDoneMsg(result)
	}
}

// startTimeout returns a tea.Cmd that sleeps for the given duration then sends statusTimeoutMsg.
func startTimeout(d time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(d)
		return statusTimeoutMsg{}
	}
}

// clearFlashAfter returns a tea.Cmd that clears the flash message after a delay.
func clearFlashAfter(id int, delay time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(delay)
		return statusFlashClearMsg{ID: id}
	}
}

// isURL returns true if the value looks like an HTTP or HTTPS URL.
func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// --- CustomView interface implementation ---

// Title returns the panel border title extracted from the data.
func (sv *StatusViewModel) Title() string { return sv.getTitleText() }

// Render returns the status view content for the given dimensions.
func (sv *StatusViewModel) Render(width, height int, noColor bool) string {
	sv.Width = width
	sv.Height = height
	sv.NoColor = noColor
	return sv.View()
}

// RowCount returns a fixed 1/1 for status views.
func (sv *StatusViewModel) RowCount() (count int, selected int, label string) {
	return 1, 1, "status"
}

// FooterBar returns the action bar for the panel footer.
func (sv *StatusViewModel) FooterBar() string { return sv.renderActionBar() }

// FlashMessage returns the current flash message, if any.
func (sv *StatusViewModel) FlashMessage() (string, bool) {
	if sv.FlashMsg != "" {
		return sv.FlashMsg, strings.HasPrefix(sv.FlashMsg, "⚠")
	}
	return "", false
}

// SearchTitle returns empty — status views have no search.
func (sv *StatusViewModel) SearchTitle() string { return "" }

// HandlesSearch returns false — status views do not handle search.
func (sv *StatusViewModel) HandlesSearch() bool { return false }
