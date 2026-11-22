package ui

import (
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testStatusData() map[string]any {
	return map[string]any{
		"title": "Sign in to Entra",
		"url":   "https://microsoft.com/devicelogin",
		"code":  "EH5HFPGJJ",
		"user":  "user@example.com",
		"messages": []any{
			"Already authenticated as user@example.com",
			"Use 'myapp auth logout entra' to sign out first",
		},
	}
}

func testStatusSchema() *DisplaySchema {
	return &DisplaySchema{
		Version: "v1",
		Status: &StatusDisplayConfig{
			TitleField:     "title",
			MessageField:   "messages",
			WaitMessage:    "Waiting for authentication...",
			SuccessMessage: "Authenticated successfully!",
			DoneBehavior:   DoneBehaviorExitAfterDelay,
			DoneDelay:      "1s",
			DisplayFields: []StatusFieldDisplay{
				{Label: "URL", Field: "url"},
				{Label: "Code", Field: "code"},
			},
			Actions: []StatusActionConfig{
				{
					Label: "Copy code",
					Type:  "copy-value",
					Field: "code",
					Keys:  StatusKeyBindings{Vim: "c", Emacs: "alt+c", Function: "f2"},
				},
				{
					Label: "Open URL",
					Type:  "open-url",
					Field: "url",
					Keys:  StatusKeyBindings{Vim: "o", Emacs: "alt+o", Function: "f3"},
				},
			},
		},
	}
}

func TestBuildStatusViewModel(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()

	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)
	require.NotNil(t, sv)
	assert.Equal(t, statusPhaseWaiting, sv.Phase)
	assert.Equal(t, KeyModeVim, sv.KeyMode)
	assert.True(t, sv.NoColor)
	assert.Equal(t, 80, sv.Width)
	assert.Equal(t, 24, sv.Height)
}

func TestBuildStatusViewModel_NilSchema(t *testing.T) {
	sv := buildStatusViewModel(nil, nil, KeyModeVim, true, nil, 80, 24)
	assert.Nil(t, sv)
}

func TestBuildStatusViewModel_NilStatus(t *testing.T) {
	schema := &DisplaySchema{Version: "v1"}
	sv := buildStatusViewModel(nil, schema, KeyModeVim, true, nil, 80, 24)
	assert.Nil(t, sv)
}

func TestBuildStatusViewModel_WithDoneChannel(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	done := make(chan StatusResult, 1)

	sv := buildStatusViewModel(data, schema, KeyModeVim, true, done, 80, 24)
	require.NotNil(t, sv)
	assert.True(t, sv.HasDone)
	assert.NotNil(t, sv.DoneChan)
}

func TestBuildStatusViewModel_WithTimeout(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	schema.Status.Timeout = "5s"

	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)
	require.NotNil(t, sv)
	assert.True(t, sv.HasDone)
	assert.Nil(t, sv.DoneChan)
}

func TestStatusViewModel_GetTitleText(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	assert.Equal(t, "Sign in to Entra", sv.getTitleText())
}

func TestStatusViewModel_GetMessages(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	msgs := sv.getMessages()
	require.Len(t, msgs, 2)
	assert.Equal(t, "Already authenticated as user@example.com", msgs[0])
	assert.Equal(t, "Use 'myapp auth logout entra' to sign out first", msgs[1])
}

func TestStatusViewModel_GetMessages_StringValue(t *testing.T) {
	data := map[string]any{
		"title":   "Test",
		"message": "single message",
	}
	schema := &DisplaySchema{
		Version: "v1",
		Status: &StatusDisplayConfig{
			TitleField:   "title",
			MessageField: "message",
		},
	}
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)
	msgs := sv.getMessages()
	require.Len(t, msgs, 1)
	assert.Equal(t, "single message", msgs[0])
}

func TestStatusViewModel_GetFieldValue(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	assert.Equal(t, "EH5HFPGJJ", sv.getFieldValue("code"))
	assert.Equal(t, "https://microsoft.com/devicelogin", sv.getFieldValue("url"))
	assert.Equal(t, "", sv.getFieldValue("nonexistent"))
}

func TestStatusViewModel_KeyForMode_Vim(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	keys := StatusKeyBindings{Vim: "c", Emacs: "alt+c", Function: "f2"}
	assert.Equal(t, "c", sv.keyForMode(keys))
}

func TestStatusViewModel_KeyForMode_Emacs(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeEmacs, true, nil, 80, 24)

	keys := StatusKeyBindings{Vim: "c", Emacs: "alt+c", Function: "f2"}
	assert.Equal(t, "alt+c", sv.keyForMode(keys))
}

func TestStatusViewModel_KeyForMode_Function(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeFunction, true, nil, 80, 24)

	keys := StatusKeyBindings{Vim: "c", Emacs: "alt+c", Function: "f2"}
	assert.Equal(t, "f2", sv.keyForMode(keys))
}

func TestStatusViewModel_DisplayKeyForMode(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()

	tests := []struct {
		mode     KeyMode
		expected string
	}{
		{KeyModeVim, "c"},
		{KeyModeEmacs, "M-c"},
		{KeyModeFunction, "F2"},
	}

	keys := StatusKeyBindings{Vim: "c", Emacs: "alt+c", Function: "f2"}
	for _, tt := range tests {
		sv := buildStatusViewModel(data, schema, tt.mode, true, nil, 80, 24)
		assert.Equal(t, tt.expected, sv.displayKeyForMode(keys), "mode: %s", tt.mode)
	}
}

func TestStatusViewModel_IsQuitKey(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()

	tests := []struct {
		mode     KeyMode
		key      string
		expected bool
	}{
		{KeyModeVim, "q", true},
		{KeyModeVim, "ctrl+c", false}, // ctrl+c is universal, not mode-specific
		{KeyModeEmacs, "ctrl+q", true},
		{KeyModeEmacs, "q", false},
		{KeyModeFunction, "f10", true},
		{KeyModeFunction, "q", false},
	}

	for _, tt := range tests {
		sv := buildStatusViewModel(data, schema, tt.mode, true, nil, 80, 24)
		assert.Equal(t, tt.expected, sv.isQuitKey(tt.key), "mode=%s key=%s", tt.mode, tt.key)
	}
}

func TestStatusViewModel_QuitKeyDisplay(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()

	tests := []struct {
		mode     KeyMode
		expected string
	}{
		{KeyModeVim, "q"},
		{KeyModeEmacs, "C-q"},
		{KeyModeFunction, "F10"},
	}

	for _, tt := range tests {
		sv := buildStatusViewModel(data, schema, tt.mode, true, nil, 80, 24)
		assert.Equal(t, tt.expected, sv.quitKeyDisplay(), "mode: %s", tt.mode)
	}
}

func TestStatusViewModel_DoneBehavior(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()

	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)
	assert.Equal(t, DoneBehaviorExitAfterDelay, sv.doneBehavior())

	schema.Status.DoneBehavior = DoneBehaviorWaitForKey
	sv = buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)
	assert.Equal(t, DoneBehaviorWaitForKey, sv.doneBehavior())
}

func TestStatusViewModel_View_TitleNotInContent(t *testing.T) {
	// Title is rendered in the panel border by panelLayoutStateFromModel,
	// so View() should NOT contain it (avoids duplication).
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	view := sv.View()
	assert.NotContains(t, view, "Sign in to Entra")
}

func TestStatusViewModel_View_ContainsMessages(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	view := sv.View()
	assert.Contains(t, view, "Already authenticated as user@example.com")
	assert.Contains(t, view, "Use 'myapp auth logout entra' to sign out first")
}

func TestStatusViewModel_View_ActionLabelsNotInContent_Vim(t *testing.T) {
	// Action bar is rendered in the footer (below border), not in View() content.
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	view := sv.View()
	assert.NotContains(t, view, "Copy code")
	assert.NotContains(t, view, "Open URL")

	// But renderActionBar should contain them.
	bar := sv.renderActionBar()
	assert.Contains(t, bar, "Copy code")
	assert.Contains(t, bar, "Open URL")
	assert.Contains(t, bar, "quit")
}

func TestStatusViewModel_View_ContainsDisplayFields(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	view := sv.View()
	assert.Contains(t, view, "URL:")
	assert.Contains(t, view, "https://microsoft.com/devicelogin")
	assert.Contains(t, view, "Code:")
	assert.Contains(t, view, "EH5HFPGJJ")
}

func TestStatusViewModel_View_ActionLabelsNotInContent_Emacs(t *testing.T) {
	// Action bar is rendered in the footer (below border), not in View() content.
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeEmacs, true, nil, 80, 24)

	view := sv.View()
	assert.NotContains(t, view, "Copy code")
	assert.NotContains(t, view, "Open URL")

	// But renderActionBar should contain them.
	bar := sv.renderActionBar()
	assert.Contains(t, bar, "Copy code")
	assert.Contains(t, bar, "Open URL")
	assert.Contains(t, bar, "quit")
}

func TestStatusViewModel_Update_DoneMsg_Success(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	updated, _ := sv.Update(statusDoneMsg{Message: "Auth complete"})
	sv = updated.(*StatusViewModel)
	assert.Equal(t, statusPhaseSuccess, sv.Phase)
	assert.Equal(t, "Auth complete", sv.ResultMsg)
}

func TestStatusViewModel_Update_DoneMsg_Error(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	updated, _ := sv.Update(statusDoneMsg{Err: assert.AnError})
	sv = updated.(*StatusViewModel)
	assert.Equal(t, statusPhaseError, sv.Phase)
	assert.Contains(t, sv.ResultMsg, "assert.AnError")
}

func TestStatusViewModel_Update_DoneMsg_DefaultSuccess(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	schema.Status.SuccessMessage = ""
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	updated, _ := sv.Update(statusDoneMsg{})
	sv = updated.(*StatusViewModel)
	assert.Equal(t, statusPhaseSuccess, sv.Phase)
	assert.Equal(t, "Done", sv.ResultMsg)
}

func TestStatusViewModel_Update_TimeoutMsg(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	updated, _ := sv.Update(statusTimeoutMsg{})
	sv = updated.(*StatusViewModel)
	assert.Equal(t, statusPhaseSuccess, sv.Phase)
	assert.Equal(t, "Authenticated successfully!", sv.ResultMsg)
}

func TestStatusViewModel_Update_SpinnerTick_OnlyInWaiting(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	// Spinner tick should work in waiting phase
	tickMsg := spinner.TickMsg{ID: sv.Spinner.ID(), Time: time.Now()}
	updated, cmd := sv.Update(tickMsg)
	sv = updated.(*StatusViewModel)
	assert.NotNil(t, cmd, "spinner should produce tick cmd in waiting phase")

	// Spinner tick should be a no-op in success phase
	sv.Phase = statusPhaseSuccess
	_, cmd = sv.Update(tickMsg)
	assert.Nil(t, cmd, "spinner should not produce cmd in success phase")
}

func TestStatusViewModel_Update_FlashClear(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	sv.FlashMsg = "test flash"
	sv.FlashTimer = 5

	// Wrong ID should not clear
	updated, _ := sv.Update(statusFlashClearMsg{ID: 4})
	sv = updated.(*StatusViewModel)
	assert.Equal(t, "test flash", sv.FlashMsg)

	// Correct ID should clear
	updated, _ = sv.Update(statusFlashClearMsg{ID: 5})
	sv = updated.(*StatusViewModel)
	assert.Equal(t, "", sv.FlashMsg)
}

func TestStatusViewModel_View_SuccessPhase(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	sv.Phase = statusPhaseSuccess
	sv.ResultMsg = "All done"

	view := sv.View()
	assert.Contains(t, view, "All done")
}

func TestStatusViewModel_View_ErrorPhase(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	sv.Phase = statusPhaseError
	sv.ResultMsg = "auth failed"

	view := sv.View()
	assert.Contains(t, view, "auth failed")
}

func TestStatusViewModel_View_WaitForKeyPhase(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	schema.Status.DoneBehavior = DoneBehaviorWaitForKey
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	sv.Phase = statusPhaseSuccess
	sv.ResultMsg = "done"

	view := sv.View()
	assert.Contains(t, view, "Press any key to exit")
}

func TestStatusViewModel_HandleKey_Quit_Vim(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	_, cmd := sv.handleKey(tea.KeyPressMsg{Code: 'q', Text: "q"})
	assert.NotNil(t, cmd, "q should quit in vim mode")
}

func TestStatusViewModel_HandleKey_CtrlC(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	// ctrl+c is universal quit
	msg := tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	_, cmd := sv.handleKey(msg)
	assert.NotNil(t, cmd, "ctrl+c should quit in all modes")
}

func TestStatusViewModel_HandleKey_Esc(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	msg := tea.KeyPressMsg{Code: tea.KeyEscape}
	_, cmd := sv.handleKey(msg)
	assert.NotNil(t, cmd, "esc should quit")
}

func TestStatusViewModel_HandleKey_WaitForKey_AnyKeyExits(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	schema.Status.DoneBehavior = DoneBehaviorWaitForKey
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)
	sv.Phase = statusPhaseSuccess

	// Any key should exit after done with wait-for-key behavior
	msg := tea.KeyPressMsg{Code: 'x', Text: "x"}
	_, cmd := sv.handleKey(msg)
	assert.NotNil(t, cmd, "any key should exit in wait-for-key mode after done")
}

func TestStatusViewModel_FlashMessage_StoredForStatusPanel(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	sv.FlashMsg = "✓ Copied to clipboard"
	// Flash is no longer rendered inline in View(); it is surfaced via the
	// panel layout's InfoMessage (right-aligned status panel below the border).
	view := sv.View()
	assert.NotContains(t, view, "Copied to clipboard", "flash should not appear in View() output")
	assert.Equal(t, "✓ Copied to clipboard", sv.FlashMsg, "flash message should be stored on the model")
}

func TestStatusViewModel_View_WaitMessage_WithDone(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	done := make(chan StatusResult, 1)
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, done, 80, 24)

	view := sv.View()
	assert.Contains(t, view, "Waiting for authentication...")
}

func TestStatusViewModel_View_NoWaitMessage_WithoutDone(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	view := sv.View()
	assert.NotContains(t, view, "Waiting for authentication...")
}

// Test updateViewMode sets status view correctly
func TestUpdateViewMode_Status(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()

	m := InitialModel(data)
	m.DisplaySchema = schema
	m.WinWidth = 80
	m.WinHeight = 24
	m.updateViewMode(data)

	assert.Equal(t, "status", m.ViewMode)
	assert.NotNil(t, m.StatusViewState)
	assert.Nil(t, m.ListViewState)
	assert.Nil(t, m.DetailViewState)
}

// Test updateViewMode prefers status over list/detail
func TestUpdateViewMode_StatusTakesPriority(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	// Also set list config — status should still take priority
	schema.List = &ListDisplayConfig{TitleField: "title"}

	m := InitialModel(data)
	m.DisplaySchema = schema
	m.WinWidth = 80
	m.WinHeight = 24
	m.updateViewMode(data)

	assert.Equal(t, "status", m.ViewMode)
}

func TestStatusViewModel_RenderCustomViewContent(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()

	m := InitialModel(data)
	m.DisplaySchema = schema
	m.WinWidth = 80
	m.WinHeight = 24
	m.NoColor = true
	m.updateViewMode(data)

	content, ok := m.renderCustomViewContent()
	assert.True(t, ok)
	// Title is rendered in the panel border, not in the content area.
	assert.NotContains(t, content, "Sign in to Entra")
	// But messages should still be present.
	assert.Contains(t, content, "Already authenticated as user@example.com")
}

func TestStatusViewModel_CustomViewRowCount(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()

	m := InitialModel(data)
	m.DisplaySchema = schema
	m.WinWidth = 80
	m.WinHeight = 24
	m.updateViewMode(data)

	count, sel, label := m.customViewRowCount()
	assert.Equal(t, 1, count)
	assert.Equal(t, 1, sel)
	assert.Equal(t, "status", label)
}

// Test: menu copy action is swallowed in status view mode so "Copied: _"
// doesn't leak into the status panel after the flash clears.
func TestHandleMenuKey_CopySwallowedInStatusView(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()

	m := InitialModel(data)
	m.DisplaySchema = schema
	m.WinWidth = 80
	m.WinHeight = 24
	m.NoColor = true
	m.updateViewMode(data)
	require.Equal(t, "status", m.ViewMode)

	// Simulate pressing F5 (copy action in function mode)
	handled, cmd := m.handleMenuKey("f5")
	assert.True(t, handled, "F5 copy should be intercepted")
	assert.Nil(t, cmd, "no command should be returned")
	assert.Empty(t, m.ErrMsg, "ErrMsg should NOT be set — menu copy is swallowed")
}

// Test: m.ErrMsg doesn't leak into info panel when a custom view is active.
func TestInfoMessageFallthrough_SuppressedInCustomView(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()

	m := InitialModel(data)
	m.DisplaySchema = schema
	m.WinWidth = 80
	m.WinHeight = 24
	m.NoColor = true
	m.updateViewMode(data)

	// Simulate a stale ErrMsg from a prior copy action.
	m.ErrMsg = "Copied: _"
	m.StatusType = "success"

	state := panelLayoutStateFromModel(&m, PanelLayoutModelOptions{
		AppName:      "kvx",
		InputVisible: false,
	})

	// The info message should be empty — ErrMsg must not leak into custom views.
	assert.Empty(t, state.InfoMessage, "ErrMsg should not appear in status panel for custom views")
}

// Test: m.ErrMsg still appears when no custom view is active (table mode).
func TestInfoMessageFallthrough_ShownInTableMode(t *testing.T) {
	m := InitialModel(map[string]any{"key": "value"})
	m.WinWidth = 80
	m.WinHeight = 24
	m.NoColor = true
	m.ErrMsg = "Copied: _.key"
	m.StatusType = "success"

	state := panelLayoutStateFromModel(&m, PanelLayoutModelOptions{
		AppName:      "kvx",
		InputVisible: false,
	})

	assert.Equal(t, "Copied: _.key", state.InfoMessage, "ErrMsg should appear normally in table mode")
	assert.False(t, state.InfoError)
}

// TestStatusViewKey_RoutedThroughModelUpdate verifies that a key press reaches
// handleStatusViewKey when ViewMode is "status", exercising the simplified
// routing path in Model.Update that no longer needs a type assertion.
func TestStatusViewKey_RoutedThroughModelUpdate(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	m := InitialModel(data)
	m.Root = data
	m.ViewMode = "status"
	m.StatusViewState = sv
	m.InputFocused = false
	m.KeyMode = KeyModeVim
	m.WinWidth = 80
	m.WinHeight = 24
	m.applyLayout(true)
	m.Tbl.Focus()

	// Send a key that the status view handles (e.g. any unrecognised key
	// is accepted without error by StatusViewModel.handleKey).
	result, _ := m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})

	m2, ok := result.(*Model)
	require.True(t, ok)
	assert.Equal(t, "status", m2.ViewMode)
	assert.NotNil(t, m2.StatusViewState)
}

func testStatusInputSchema(masked bool) *DisplaySchema {
	schema := testStatusSchema()
	schema.Status.Input = &StatusInputConfig{
		Label:       "Paste callback URL:",
		Placeholder: "http://localhost:45189/?token=...",
		Masked:      masked,
	}
	return schema
}

func TestStatusViewModel_Input_Frame1AutoFocusAndDefaults(t *testing.T) {
	data := testStatusData()
	schema := testStatusInputSchema(true)
	done := make(chan StatusResult, 1)

	sv := buildStatusViewModel(data, schema, KeyModeVim, true, done, 100, 24)
	require.NotNil(t, sv)
	assert.True(t, sv.HasActiveInput())
	assert.True(t, sv.IsMaskedInput())
	assert.True(t, sv.Input.Focused(), "input must be focused on frame 1")
	assert.Equal(t, defaultStatusInputMaxLength, sv.Input.CharLimit)
	assert.Equal(t, defaultStatusInputPrompt, sv.Input.Prompt)

	view := sv.View()
	assert.Contains(t, view, "Paste callback URL:")
	assert.Contains(t, view, "http://localhost:45189/?token=...")
}

func TestStatusViewModel_Input_CustomPromptAndMaxLength(t *testing.T) {
	data := testStatusData()
	schema := testStatusInputSchema(false)
	schema.Status.Input.Prompt = "> "
	schema.Status.Input.MaxLength = 10

	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)
	require.NotNil(t, sv)
	assert.Equal(t, "> ", sv.Input.Prompt)
	assert.Equal(t, 10, sv.Input.CharLimit)
	assert.False(t, sv.IsMaskedInput())

	updated, _ := sv.Update(tea.PasteMsg{Content: "0123456789ABCDEF"})
	sv = updated.(*StatusViewModel)
	assert.Equal(t, "0123456789", sv.Input.Value(), "MaxLength must cap input runes")
}

func TestStatusViewModel_Input_BracketedPasteStripsCRLFWithoutAutoSubmit(t *testing.T) {
	data := testStatusData()
	schema := testStatusInputSchema(false)

	submitted := false
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 100, 24)
	sv.OnStatusInput = func(string) error {
		submitted = true
		return nil
	}

	updated, _ := sv.Update(tea.PasteMsg{Content: "http://localhost:45189/cb?token=abc\r\n"})
	sv = updated.(*StatusViewModel)

	assert.False(t, submitted, "PasteMsg must not auto-submit before Enter")
	assert.False(t, sv.InputSubmitted)
	assert.Equal(t, "http://localhost:45189/cb?token=abc", sv.Input.Value())
}

func TestModel_StatusInput_RawKeystrokePasteSafety_ThroughModelUpdate(t *testing.T) {
	restore := StubPlatformActions()
	defer restore()

	data := testStatusData()
	schema := testStatusInputSchema(false)

	m := InitialModel(data)
	m.Root = data
	m.DisplaySchema = schema
	m.KeyMode = KeyModeVim
	m.WinWidth = 100
	m.WinHeight = 24
	m.NoColor = true
	m.updateViewMode(data)
	require.NotNil(t, m.StatusViewState)
	require.True(t, m.StatusViewState.HasActiveInput())

	// Raw keystroke paste containing q, c, o, /, ?, f, e, v, :, y, Y.
	rawPaste := "http://localhost:45189/verify?req=foo&key=qcovef:yY"
	var cur tea.Model = &m
	for _, r := range rawPaste {
		var cmd tea.Cmd
		cur, cmd = cur.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		// None of the pasted characters may trigger tea.Quit or open popups/search.
		if cmd != nil {
			msg := cmd()
			_, isQuit := msg.(tea.QuitMsg)
			assert.False(t, isQuit, "character %q must not trigger quit during raw paste", string(r))
		}
	}

	mAfter := cur.(*Model)
	assert.Equal(t, "status", mAfter.ViewMode)
	assert.False(t, mAfter.HelpVisible, "'?' must not open help while input is active")
	assert.False(t, mAfter.AdvancedSearchActive, "'/' must not open search while input is active")
	assert.False(t, mAfter.MapFilterActive, "'f' must not open filter while input is active")
	assert.False(t, mAfter.ShowRawView, "'v' must not toggle raw view while input is active")
	assert.Empty(t, mAfter.StatusViewState.FlashMsg, "'c' and 'o' must not trigger copy/open actions")
	assert.Equal(t, rawPaste, mAfter.StatusViewState.Input.Value())

	// F1 still opens help modal when pressed explicitly.
	cur, _ = mAfter.Update(tea.KeyPressMsg{Code: tea.KeyF1})
	assert.True(t, cur.(*Model).HelpVisible, "F1 should still toggle help modal")
}

func TestModel_StatusInput_PasteMsgAndBlinkForwarded(t *testing.T) {
	data := testStatusData()
	schema := testStatusInputSchema(true)

	m := InitialModel(data)
	m.Root = data
	m.DisplaySchema = schema
	m.WinWidth = 100
	m.WinHeight = 24
	m.NoColor = true
	m.updateViewMode(data)

	// Forward tea.PasteMsg through Model.Update.
	res, _ := m.Update(tea.PasteMsg{Content: "secret-value\r\n"})
	m2 := res.(*Model)
	assert.Equal(t, "secret-value", m2.StatusViewState.Input.Value())

	// Forward textinput cursor blink message through Model.Update and verify next blink cmd is returned.
	initCmd := m2.Init()
	require.NotNil(t, initCmd)
	blinkMsg := textinput.Blink()
	res, blinkCmd := m2.Update(blinkMsg)
	assert.NotNil(t, res.(*Model).StatusViewState)
	assert.NotNil(t, blinkCmd, "cursor blink message should produce follow-up blink command")
}

func TestStatusViewModel_Input_ColorMode_CounterAdjacentToBullets(t *testing.T) {
	data := testStatusData()
	schema := testStatusInputSchema(true)

	// NoColor = false exercises ANSI-styled textinput rendering and ansi.Truncate clamping.
	sv := buildStatusViewModel(data, schema, KeyModeVim, false, nil, 100, 24)
	updated, _ := sv.Update(tea.PasteMsg{Content: "topsecret"})
	sv = updated.(*StatusViewModel)

	plain := ansi.Strip(sv.View())
	assert.NotContains(t, plain, "topsecret")
	// In color mode the 1-cell cursor sits right after the 9 bullets, followed immediately by " (9 chars)".
	assert.Contains(t, plain, "•••••••••  (9 chars)", "counter must sit adjacent to masked bullets/cursor without padding gap")
}

func TestModel_StatusInput_MaskedBulletEchoCounterAndSkippedDebugLogging(t *testing.T) {
	data := testStatusData()
	schema := testStatusInputSchema(true)

	m := InitialModel(data)
	m.Root = data
	m.DisplaySchema = schema
	m.DebugMode = true
	m.WinWidth = 100
	m.WinHeight = 24
	m.NoColor = true
	m.updateViewMode(data)

	// Type characters and press Enter/Esc while masked input is active.
	var cur tea.Model = &m
	for _, r := range "topsecret" {
		cur, _ = cur.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	m2 := cur.(*Model)

	view := m2.StatusViewState.View()
	assert.NotContains(t, view, "topsecret", "masked input must not echo raw characters")
	assert.Contains(t, view, "••••••••• (9 chars)", "masked input must render bullets and rune count")

	// Press Enter (no OnStatusInput set) and Esc (clears non-empty buffer): neither may log to DebugEvents.
	cur, _ = m2.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	cur, _ = cur.(*Model).Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m3 := cur.(*Model)

	var logged []string
	flushDebugEvents(m3, func(entry string) {
		logged = append(logged, entry)
	})
	assert.Empty(t, logged, "logKeyEvent must be skipped while masked status input is active")
	assert.Empty(t, m3.LastKey, "LastKey must not record keystrokes while masked input is active")
}

func TestStatusViewModel_Input_EscClearsThenQuits_CtrlCQuitsImmediately_CtrlUClears(t *testing.T) {
	data := testStatusData()
	schema := testStatusInputSchema(false)
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	sv.Input.SetValue("draft-token")

	// Ctrl+U clears buffer without quitting.
	updated, cmd := sv.Update(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	sv = updated.(*StatusViewModel)
	assert.Nil(t, cmd)
	assert.Empty(t, sv.Input.Value())

	// Populate buffer again; first Esc clears buffer and stays in TUI.
	sv.Input.SetValue("draft-token-2")
	updated, cmd = sv.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	sv = updated.(*StatusViewModel)
	assert.Nil(t, cmd, "Esc on non-empty input must clear buffer without quitting")
	assert.Empty(t, sv.Input.Value())

	// Second Esc on now-empty buffer quits.
	_, cmd = sv.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	require.NotNil(t, cmd, "Esc on empty input must return tea.Quit")
	_, isQuit := cmd().(tea.QuitMsg)
	assert.True(t, isQuit)

	// Ctrl+C quits immediately even when buffer is non-empty.
	sv.Input.SetValue("non-empty")
	_, cmd = sv.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	require.NotNil(t, cmd)
	_, isQuit = cmd().(tea.QuitMsg)
	assert.True(t, isQuit)
}

func TestModel_StatusInput_SubmitErrorRetryThenSuccessAndDoneTransition(t *testing.T) {
	data := testStatusData()
	schema := testStatusInputSchema(true)
	done := make(chan StatusResult, 1)

	var received []string
	m := InitialModel(data)
	m.Root = data
	m.DisplaySchema = schema
	m.DoneChan = done
	m.WinWidth = 100
	m.WinHeight = 24
	m.NoColor = true
	m.OnStatusInput = func(val string) error {
		received = append(received, val)
		if len(received) == 1 {
			return assert.AnError
		}
		return nil
	}
	m.updateViewMode(data)

	// 1. Paste invalid value with surrounding whitespace/newlines and press Enter.
	res, _ := m.Update(tea.PasteMsg{Content: "  bad-input\r\n"})
	m1 := res.(*Model)
	res, submitCmd := m1.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m1 = res.(*Model)
	require.NotNil(t, submitCmd)

	// Deliver async statusInputResultMsg (error case).
	res, _ = m1.Update(submitCmd())
	m1 = res.(*Model)
	assert.Equal(t, []string{"bad-input"}, received)
	assert.Empty(t, m1.StatusViewState.Input.Value(), "rejected input must clear buffer so retry paste does not concatenate")
	assert.True(t, m1.StatusViewState.Input.Focused(), "input must stay focused after error")
	assert.False(t, m1.StatusViewState.InputSubmitted)
	flash, isErr := m1.StatusViewState.FlashMessage()
	assert.True(t, isErr)
	assert.Contains(t, flash, "⚠")
	assert.Contains(t, flash, assert.AnError.Error())

	// 2. Paste valid value and press Enter.
	res, _ = m1.Update(tea.PasteMsg{Content: "  good-input  \n"})
	m2 := res.(*Model)
	res, submitCmd = m2.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2 = res.(*Model)
	require.NotNil(t, submitCmd)

	res, _ = m2.Update(submitCmd())
	m2 = res.(*Model)
	assert.Equal(t, []string{"bad-input", "good-input"}, received)
	assert.True(t, m2.StatusViewState.InputSubmitted)
	assert.False(t, m2.StatusViewState.HasActiveInput())
	assert.False(t, m2.StatusViewState.Input.Focused())
	assert.Empty(t, m2.StatusViewState.Input.Value(), "submitted input value must be cleared from memory")
	assert.Equal(t, "✓ Submitted", m2.StatusViewState.FlashMsg)
	assert.NotContains(t, m2.StatusViewState.View(), "Paste callback URL:", "input section must hide after successful submit")
	assert.Contains(t, m2.StatusViewState.View(), "Waiting for authentication...", "spinner wait state continues until Done")

	// 3. Async Done arrives, clears prior flash, and transitions screen to success.
	res, _ = m2.Update(statusDoneMsg{Message: "Completed"})
	m3 := res.(*Model)
	assert.Equal(t, statusPhaseSuccess, m3.StatusViewState.Phase)
	assert.Contains(t, m3.StatusViewState.View(), "✓ Completed")
	flash, isErr = m3.StatusViewState.FlashMessage()
	assert.Empty(t, flash, "completion must clear stale input flash message")
	assert.False(t, isErr)
}

func TestStatusViewModel_Input_DoneAndTimeoutClearFlashAndBuffer(t *testing.T) {
	data := testStatusData()
	schema := testStatusInputSchema(true)
	schema.Status.Timeout = "5s"

	// Case 1: Error flash + partial buffer cleared when Done arrives.
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 100, 24)
	sv.FlashMsg = "⚠ invalid callback URL"
	sv.Input.SetValue("secret-partial")

	updated, _ := sv.Update(statusDoneMsg{Message: "Finished"})
	sv = updated.(*StatusViewModel)
	flash, isErr := sv.FlashMessage()
	assert.Empty(t, flash)
	assert.False(t, isErr)
	assert.Empty(t, sv.Input.Value())
	assert.False(t, sv.Input.Focused())

	// Case 2: Timeout blurs/clears active input and flash.
	svTimeout := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 100, 24)
	svTimeout.FlashMsg = "⚠ retry"
	svTimeout.Input.SetValue("secret-timeout")

	updated, _ = svTimeout.Update(statusTimeoutMsg{})
	svTimeout = updated.(*StatusViewModel)
	assert.Equal(t, statusPhaseSuccess, svTimeout.Phase)
	assert.False(t, svTimeout.HasActiveInput())
	assert.False(t, svTimeout.Input.Focused())
	assert.Empty(t, svTimeout.Input.Value())
	assert.Empty(t, svTimeout.FlashMsg)
}

func TestStatusViewModel_InputActionKeys_ModesAndDistinctMnemonics(t *testing.T) {
	data := testStatusData()
	schema := testStatusInputSchema(false)
	schema.Status.Actions = []StatusActionConfig{
		{
			Label: "Copy value",
			Type:  "copy-value",
			Field: "code",
			Keys:  StatusKeyBindings{Vim: "y", Emacs: "alt+w", Function: "f5"},
		},
	}

	// Vim mode prefers F5 + Alt+y (derived from Vim 'y', not Emacs 'alt+w').
	svVim := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 100, 24)
	assert.Equal(t, []string{"f5", "alt+y"}, svVim.activeInputActionKeys(schema.Status.Actions[0].Keys))
	assert.Equal(t, "F5/Alt+y", svVim.displayInputActionKey(schema.Status.Actions[0].Keys))

	// Emacs mode formats modifier key as M-w and also accepts F5.
	svEmacs := buildStatusViewModel(data, schema, KeyModeEmacs, true, nil, 100, 24)
	assert.Equal(t, []string{"alt+w", "f5"}, svEmacs.activeInputActionKeys(schema.Status.Actions[0].Keys))
	assert.Equal(t, "M-w", svEmacs.displayInputActionKey(schema.Status.Actions[0].Keys))

	// Function mode uses only F5 and quits cleanly via F10 through Model.Update after submit.
	svFunc := buildStatusViewModel(data, schema, KeyModeFunction, true, nil, 100, 24)
	assert.Equal(t, []string{"f5"}, svFunc.activeInputActionKeys(schema.Status.Actions[0].Keys))
	assert.Equal(t, "F5", svFunc.displayInputActionKey(schema.Status.Actions[0].Keys))

	m := InitialModel(data)
	m.Root = data
	m.DisplaySchema = schema
	m.KeyMode = KeyModeFunction
	m.WinWidth = 100
	m.WinHeight = 24
	m.NoColor = true
	m.updateViewMode(data)
	m.StatusViewState.InputSubmitted = true
	res, quitCmd := m.Update(tea.KeyPressMsg{Code: tea.KeyF10})
	mAfter := res.(*Model)
	require.NotNil(t, quitCmd)
	assert.Empty(t, mAfter.PendingCLIExpr, "F10 in status view must quit cleanly without setting PendingCLIExpr")
}

func TestStatusViewModel_Input_DonePreemptsActiveInput(t *testing.T) {
	data := testStatusData()
	schema := testStatusInputSchema(true)
	done := make(chan StatusResult, 1)

	sv := buildStatusViewModel(data, schema, KeyModeVim, true, done, 100, 24)
	sv.Input.SetValue("partial")
	require.True(t, sv.HasActiveInput())

	updated, _ := sv.Update(statusDoneMsg{Message: "Finished via background event"})
	sv = updated.(*StatusViewModel)

	assert.Equal(t, statusPhaseSuccess, sv.Phase)
	assert.False(t, sv.HasActiveInput())
	assert.False(t, sv.Input.Focused())
	view := sv.View()
	assert.NotContains(t, view, "Paste callback URL:")
	assert.Contains(t, view, "✓ Finished via background event")
}

func TestStatusViewModel_ActionsCoexistenceAndCustomOpenURL(t *testing.T) {
	restore := StubPlatformActions()
	defer restore()

	data := testStatusData()
	schema := testStatusInputSchema(false)

	var opened []string
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 100, 24)
	sv.OpenURLFn = func(u string) error {
		opened = append(opened, u)
		return nil
	}

	// Bare 'o' goes into textinput rather than executing open-url.
	updated, _ := sv.Update(tea.KeyPressMsg{Code: 'o', Text: "o"})
	sv = updated.(*StatusViewModel)
	assert.Empty(t, opened)
	assert.Equal(t, "o", sv.Input.Value())

	// F3 triggers open-url via custom OpenURLFn without modifying textinput.
	updated, _ = sv.Update(tea.KeyPressMsg{Code: tea.KeyF3})
	sv = updated.(*StatusViewModel)
	assert.Equal(t, []string{"https://microsoft.com/devicelogin"}, opened)
	assert.Equal(t, "o", sv.Input.Value())
	assert.Equal(t, "✓ Opened in browser", sv.FlashMsg)

	// Alt+o also triggers open-url in vim mode when Keys.Vim is bare 'o'.
	updated, _ = sv.Update(tea.KeyPressMsg{Code: 'o', Mod: tea.ModAlt})
	sv = updated.(*StatusViewModel)
	assert.Len(t, opened, 2)

	// F2 triggers copy-value while input remains active.
	updated, _ = sv.Update(tea.KeyPressMsg{Code: tea.KeyF2})
	sv = updated.(*StatusViewModel)
	assert.Equal(t, "✓ Copied to clipboard", sv.FlashMsg)
}

func TestRenderModelSnapshot_StatusInputAndOpenURL(t *testing.T) {
	data := map[string]any{
		"title": "Verify Session",
		"url":   "https://example.com/verify",
	}
	schema := &DisplaySchema{
		Version: "v1",
		Status: &StatusDisplayConfig{
			TitleField: "title",
			DisplayFields: []StatusFieldDisplay{
				{Label: "Endpoint", Field: "url"},
			},
			Actions: []StatusActionConfig{
				{
					Label: "Open",
					Type:  "open-url",
					Field: "url",
					Keys:  StatusKeyBindings{Vim: "o", Emacs: "alt+o", Function: "f3"},
				},
			},
			Input: &StatusInputConfig{
				Label:  "Paste callback URL:",
				Masked: true,
			},
		},
	}

	var openedURL string
	snap := RenderModelSnapshot(data, ModelSnapshotConfig{
		Width:     100,
		Height:    20,
		NoColor:   true,
		AppName:   "kvx",
		StartKeys: []string{"http://localhost:8080/?q=1&c=2&o=3&f=4&e=5&v=6", "<F3>"},
		Configure: func(m *Model) {
			m.DisplaySchema = schema
			m.OpenURLFn = func(u string) error {
				openedURL = u
				return nil
			}
		},
	})

	assert.Equal(t, "https://example.com/verify", openedURL)
	assert.Contains(t, snap, "Paste callback URL:")
	assert.Contains(t, snap, "(46 chars)")
	assert.NotContains(t, snap, "localhost:8080")
	assert.Contains(t, snap, "Opened in browser")
}
