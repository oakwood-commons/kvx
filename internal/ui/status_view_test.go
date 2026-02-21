package ui

import (
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
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
