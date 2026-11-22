package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
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

const (
	defaultStatusInputPrompt    = "❯ "
	defaultStatusInputMaxLength = 8192
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

	// Inline text input (active while Phase == statusPhaseWaiting && !InputSubmitted)
	Input           textinput.Model
	InputSubmitted  bool
	InputSubmitting bool
	lastSubmitCmd   tea.Cmd
	OnStatusInput   StatusInputSubmitFunc
	OpenURLFn       func(url string) error

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

// statusInputResultMsg carries the outcome of an async OnStatusInput callback.
type statusInputResultMsg struct {
	Err error
}

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

	var ti textinput.Model
	if schema.Status.Input != nil {
		ic := schema.Status.Input
		ti = textinput.New()
		if ic.Prompt != "" {
			ti.Prompt = ic.Prompt
		} else {
			ti.Prompt = defaultStatusInputPrompt
		}
		ti.Placeholder = ic.Placeholder
		maxLen := ic.MaxLength
		if maxLen <= 0 {
			maxLen = defaultStatusInputMaxLength
		}
		ti.CharLimit = maxLen
		if ic.Masked {
			ti.EchoMode = textinput.EchoPassword
			ti.EchoCharacter = '•'
		}
		if width > 6 {
			ti.SetWidth(width - 6)
		} else {
			ti.SetWidth(40)
		}
		applyStatusInputStyles(&ti, noColor)
		_ = ti.Focus()
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
		Input:    ti,
		Width:    width,
		Height:   height,
	}
}

// applyStatusInputStyles configures styles on a status textinput according to NoColor and CurrentTheme.
func applyStatusInputStyles(ti *textinput.Model, noColor bool) {
	if noColor {
		ti.SetStyles(textinput.Styles{
			Cursor: textinput.CursorStyle{Blink: true},
		})
		return
	}
	th := CurrentTheme()
	styles := textinput.DefaultDarkStyles()
	if th.HeaderFG != nil {
		styles.Focused.Prompt = lipgloss.NewStyle().Foreground(th.HeaderFG).Bold(true)
		styles.Blurred.Prompt = lipgloss.NewStyle().Foreground(th.HeaderFG)
	}
	if th.InputFG != nil {
		styles.Focused.Text = lipgloss.NewStyle().Foreground(th.InputFG)
		styles.Blurred.Text = lipgloss.NewStyle().Foreground(th.InputFG)
	}
	if th.GhostFG != nil {
		styles.Focused.Placeholder = lipgloss.NewStyle().Foreground(th.GhostFG)
		styles.Blurred.Placeholder = lipgloss.NewStyle().Foreground(th.GhostFG)
	}
	ti.SetStyles(styles)
}

// HasActiveInput reports whether the status screen currently has an active
// inline text input accepting keystrokes and paste events.
func (sv *StatusViewModel) HasActiveInput() bool {
	return sv != nil && sv.Config != nil && sv.Config.Input != nil &&
		sv.Phase == statusPhaseWaiting && !sv.InputSubmitted
}

// IsMaskedInput reports whether an active inline input is configured as masked.
func (sv *StatusViewModel) IsMaskedInput() bool {
	return sv.HasActiveInput() && sv.Config.Input.Masked
}

// Init returns the initial commands for the status view (spinner tick + completion source + cursor blink).
func (sv *StatusViewModel) Init() tea.Cmd {
	cmds := []tea.Cmd{sv.Spinner.Tick}

	if sv.HasActiveInput() {
		cmds = append(cmds, textinput.Blink)
	}

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
		sv.FlashMsg = ""
		sv.FlashTimer++
		if sv.Config.Input != nil {
			sv.Input.SetValue("")
			sv.Input.SetCursor(0)
			sv.Input.Blur()
			sv.InputSubmitting = false
			sv.lastSubmitCmd = nil
		}
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
		sv.FlashMsg = ""
		sv.FlashTimer++
		if sv.Config.Input != nil {
			sv.Input.SetValue("")
			sv.Input.SetCursor(0)
			sv.Input.Blur()
			sv.InputSubmitting = false
			sv.lastSubmitCmd = nil
		}
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

	case statusInputResultMsg:
		sv.InputSubmitting = false
		sv.lastSubmitCmd = nil
		if sv.Phase != statusPhaseWaiting {
			return sv, nil
		}
		sv.FlashTimer++
		sv.Input.SetValue("")
		sv.Input.SetCursor(0)
		if msg.Err != nil {
			sv.FlashMsg = "⚠ " + msg.Err.Error()
			cmd := sv.Input.Focus()
			return sv, cmd
		}
		sv.Input.Blur()
		sv.InputSubmitted = true
		sv.FlashMsg = "✓ Submitted"
		return sv, nil

	case tea.PasteMsg:
		if sv.HasActiveInput() && !sv.InputSubmitting {
			clean := stripCRLF(msg.Content)
			if clean == "" {
				return sv, nil
			}
			var cmd tea.Cmd
			sv.Input, cmd = sv.Input.Update(tea.PasteMsg{Content: clean})
			return sv, cmd
		}
		return sv, nil

	case tea.KeyPressMsg:
		return sv.handleKey(msg)

	default:
		if sv.HasActiveInput() {
			var cmd tea.Cmd
			sv.Input, cmd = sv.Input.Update(msg)
			return sv, cmd
		}
	}

	return sv, nil
}

// handleKey processes key presses on the status screen.
func (sv *StatusViewModel) handleKey(msg tea.KeyPressMsg) (*StatusViewModel, tea.Cmd) {
	keyStr := normalizeStatusKey(msg)

	// Ctrl+C is always an immediate universal quit.
	if keyStr == "ctrl+c" {
		return sv, tea.Quit
	}

	// Active inline input owns printable keys, Esc clear-then-quit, Ctrl+U, and Enter.
	if sv.HasActiveInput() {
		switch keyStr {
		case "esc":
			if sv.Input.Value() != "" {
				sv.Input.SetValue("")
				sv.Input.SetCursor(0)
				return sv, nil
			}
			return sv, tea.Quit

		case "ctrl+u":
			sv.Input.SetValue("")
			sv.Input.SetCursor(0)
			return sv, nil

		case "enter":
			if sv.InputSubmitting {
				return sv, nil
			}
			val := strings.TrimSpace(stripCRLF(sv.Input.Value()))
			if val == "" || sv.OnStatusInput == nil {
				return sv, nil
			}
			sv.InputSubmitting = true
			submitFn := sv.OnStatusInput
			submitCmd := func() tea.Msg {
				return statusInputResultMsg{Err: submitFn(val)}
			}
			sv.lastSubmitCmd = submitCmd
			return sv, submitCmd
		}

		// Match only modifier/function action keys so bare single-character keys
		// (c, o, q, /, ?, f, e, v, etc.) flow directly into textinput during paste.
		for _, action := range sv.Config.Actions {
			if sv.matchesInputActionKey(keyStr, action.Keys) {
				return sv.executeAction(action)
			}
		}

		if sv.InputSubmitting {
			return sv, nil
		}

		var cmd tea.Cmd
		sv.Input, cmd = sv.Input.Update(msg)
		return sv, cmd
	}

	if keyStr == "esc" {
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

// normalizeStatusKey normalizes raw control key codes (e.g. from ApplyStartupKeys)
// to standard key strings.
func normalizeStatusKey(msg tea.KeyPressMsg) string {
	switch msg.Code {
	case 0x03:
		return "ctrl+c"
	case 0x15:
		return "ctrl+u"
	case tea.KeyEscape:
		if msg.Mod == 0 {
			return "esc"
		}
	case tea.KeyEnter:
		if msg.Mod == 0 {
			return "enter"
		}
	}
	return msg.String()
}

// stripCRLF removes carriage returns and newlines from pasted or submitted text.
func stripCRLF(s string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(s)
}

// isSafeInputActionKey reports whether key k uses a function key (f2..f9) or an
// alt+/ctrl+ modifier so it cannot collide with unmodified printable characters.
func isSafeInputActionKey(k string) bool {
	k = strings.ToLower(strings.TrimSpace(k))
	if k == "" {
		return false
	}
	if strings.HasPrefix(k, "alt+") && len(k) > len("alt+") {
		return true
	}
	if strings.HasPrefix(k, "ctrl+") && len(k) > len("ctrl+") {
		return k != "ctrl+c" && k != "ctrl+u" && k != "ctrl+m" && k != "ctrl+j"
	}
	switch k {
	case "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9":
		return true
	}
	return false
}

// activeInputActionKeys returns the safe modifier/function key bindings for an
// action while StatusDisplayConfig.Input is active.
func (sv *StatusViewModel) activeInputActionKeys(keys StatusKeyBindings) []string {
	var result []string
	addUnique := func(k string) {
		k = strings.ToLower(strings.TrimSpace(k))
		if !isSafeInputActionKey(k) {
			return
		}
		for _, existing := range result {
			if existing == k {
				return
			}
		}
		result = append(result, k)
	}

	trimmedVim := strings.TrimSpace(keys.Vim)
	trimmedEmacs := strings.TrimSpace(keys.Emacs)

	switch sv.KeyMode {
	case KeyModeFunction:
		addUnique(keys.Function)
	case KeyModeEmacs:
		if isSafeInputActionKey(trimmedEmacs) {
			addUnique(trimmedEmacs)
		} else {
			addUnique(keys.Function)
			if len([]rune(trimmedEmacs)) == 1 {
				addUnique("alt+" + trimmedEmacs)
			}
		}
		addUnique(keys.Function)
	case KeyModeVim:
		fallthrough
	default:
		if isSafeInputActionKey(trimmedVim) {
			addUnique(trimmedVim)
		} else {
			addUnique(keys.Function)
			if len([]rune(trimmedVim)) == 1 {
				addUnique("alt+" + trimmedVim)
			} else if isSafeInputActionKey(trimmedEmacs) {
				addUnique(trimmedEmacs)
			}
		}
	}

	return result
}

// matchesInputActionKey reports whether keyStr matches any active safe binding for keys.
func (sv *StatusViewModel) matchesInputActionKey(keyStr string, keys StatusKeyBindings) bool {
	lower := strings.ToLower(keyStr)
	for _, candidate := range sv.activeInputActionKeys(keys) {
		if lower == candidate {
			return true
		}
	}
	return false
}

// displayInputActionKey formats the active safe bindings for footer display while Input is active.
func (sv *StatusViewModel) displayInputActionKey(keys StatusKeyBindings) string {
	active := sv.activeInputActionKeys(keys)
	if len(active) == 0 {
		return ""
	}
	if sv.KeyMode == KeyModeEmacs {
		return formatEmacsKey(active[0])
	}
	if sv.KeyMode == KeyModeFunction {
		return formatInputKeyToken(active[0])
	}
	labels := make([]string, 0, len(active))
	for _, k := range active {
		labels = append(labels, formatInputKeyToken(k))
	}
	return strings.Join(labels, "/")
}

// formatInputKeyToken renders a normalized safe key token in human-readable form.
func formatInputKeyToken(k string) string {
	if strings.HasPrefix(k, "f") && len(k) >= 2 && k[1] >= '0' && k[1] <= '9' {
		return strings.ToUpper(k)
	}
	if strings.HasPrefix(k, "alt+") {
		return "Alt+" + strings.TrimPrefix(k, "alt+")
	}
	if strings.HasPrefix(k, "ctrl+") {
		return "Ctrl+" + strings.TrimPrefix(k, "ctrl+")
	}
	return k
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
		if sv.OpenURLFn != nil {
			err = sv.OpenURLFn(fieldValue)
		} else {
			err = OpenURL(fieldValue)
		}
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
		if sv.HasActiveInput() {
			sections = append(sections, sv.renderInputSection(th)...)
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

// renderInputSection renders the optional helper label and focused inline text input
// without mutating sv.Input.
func (sv *StatusViewModel) renderInputSection(th Theme) []string {
	ic := sv.Config.Input
	lines := make([]string, 0, 2)

	if ic.Label != "" {
		labelStyle := lipgloss.NewStyle()
		if !sv.NoColor && th.ValueColor != nil {
			labelStyle = labelStyle.Foreground(th.ValueColor)
		}
		lines = append(lines, "  "+labelStyle.Render(ic.Label))
	}

	val := sv.Input.Value()
	runeCount := len([]rune(val))
	counter := ""
	if ic.Masked && runeCount > 0 {
		counter = fmt.Sprintf(" (%d chars)", runeCount)
	}

	panelW := sv.Width
	if panelW <= 0 {
		panelW = 80
	}
	availW := panelW - 4
	if availW < 10 {
		availW = 40
	}

	ti := sv.Input
	applyStatusInputStyles(&ti, sv.NoColor)
	promptW := lipgloss.Width(ti.Prompt)
	bodyW := availW - promptW - len(counter) - 1
	if bodyW < 4 {
		bodyW = 4
	}
	ti.SetWidth(bodyW)
	ti.SetCursor(ti.Position())

	rawView := ti.View()
	if sv.NoColor {
		rawView = ansi.Strip(rawView)
		rawView = strings.TrimRight(rawView, " ")
		if len(rawView) < len(ti.Prompt) {
			rawView = ti.Prompt
		}
	} else {
		var desiredW int
		switch {
		case val != "":
			valW := lipgloss.Width(val)
			if ic.Masked {
				valW = runeCount
			}
			if valW > bodyW {
				valW = bodyW
			}
			desiredW = promptW + valW
			if ti.Focused() && ti.Position() >= runeCount {
				desiredW++
			}
		case ic.Placeholder != "":
			phW := lipgloss.Width(ic.Placeholder)
			if phW > bodyW {
				phW = bodyW
			}
			if phW < 1 {
				phW = 1
			}
			desiredW = promptW + phW
		default:
			desiredW = promptW
			if ti.Focused() {
				desiredW++
			}
		}
		rawView = ansi.Truncate(rawView, desiredW, "")
	}

	if counter != "" {
		counterStyle := lipgloss.NewStyle()
		if !sv.NoColor && th.HelpValue != nil {
			counterStyle = counterStyle.Foreground(th.HelpValue)
		}
		rawView += counterStyle.Render(counter)
	}

	lines = append(lines, "  "+rawView)
	return lines
}

// renderActionBar renders the footer-style action key labels.
func (sv *StatusViewModel) renderActionBar() string {
	th := CurrentTheme()
	fkeyStyle := lipgloss.NewStyle()
	if !sv.NoColor {
		fkeyStyle = fkeyStyle.Bold(true)
		if th.FooterFG != nil {
			fkeyStyle = fkeyStyle.Foreground(th.FooterFG)
		}
		if th.FooterBG != nil {
			fkeyStyle = fkeyStyle.Background(th.FooterBG)
		}
	}

	if sv.HasActiveInput() {
		entries := make([]string, 0, len(sv.Config.Actions)+2)
		entries = append(entries, fkeyStyle.Render("Enter")+" submit")
		for _, action := range sv.Config.Actions {
			key := sv.displayInputActionKey(action.Keys)
			if key != "" && action.Label != "" {
				entries = append(entries, fkeyStyle.Render(key)+" "+action.Label)
			}
		}
		escLabel := "quit"
		if sv.Input.Value() != "" {
			escLabel = "clear"
		}
		entries = append(entries, fkeyStyle.Render("Esc")+" "+escLabel)
		return fitFooterParts(entries, sv.Width)
	}

	entries := make([]string, 0, len(sv.Config.Actions)+1)

	// Configured actions
	for _, action := range sv.Config.Actions {
		key := sv.displayKeyForMode(action.Keys)
		if key != "" && action.Label != "" {
			entries = append(entries, fkeyStyle.Render(key)+" "+action.Label)
		}
	}

	// Quit action (always present)
	quitKey := sv.quitKeyDisplay()
	entries = append(entries, fkeyStyle.Render(quitKey)+" quit")

	return fitFooterParts(entries, sv.Width)
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
