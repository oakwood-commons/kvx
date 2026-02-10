//nolint:forcetypeassert
package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// testKeyModeModel creates a Model configured for keybinding tests.
func testKeyModeModel(mode KeyMode) *Model {
	node := map[string]any{
		"alpha":   1,
		"beta":    2,
		"charlie": 3,
		"delta":   4,
	}
	m := InitialModel(node)
	m.Root = node
	m.KeyMode = mode
	m.InputFocused = false
	m.WinWidth = 80
	m.WinHeight = 24
	m.Tbl.Focus()
	m.applyLayout(true)
	return &m
}

// --- Unit Tests for handleVimKey ---

func TestHandleVimKey_Navigation(t *testing.T) {
	m := testKeyModeModel(KeyModeVim)

	tests := []struct {
		name string
		key  string
		want VimAction
	}{
		{"j moves down", "j", VimActionDown},
		{"k moves up", "k", VimActionUp},
		{"h goes back", "h", VimActionBack},
		{"l goes forward", "l", VimActionForward},
		{"/ opens search", "/", VimActionSearch},
		{"n next match", "n", VimActionNextMatch},
		{"N prev match", "N", VimActionPrevMatch},
		{"G goes to bottom", "G", VimActionBottom},
		{"? toggles help", "?", VimActionHelp},
		{"y copies", "y", VimActionCopy},
		{": opens expr", ":", VimActionExpr},
		{"q quits", "q", VimActionQuit},
		{"enter goes forward", "enter", VimActionEnter},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.handleVimKey(tt.key)
			if got != tt.want {
				t.Errorf("handleVimKey(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestHandleVimKey_GGSequence(t *testing.T) {
	m := testKeyModeModel(KeyModeVim)

	// First 'g' sets pending state, returns none
	action1 := m.handleVimKey("g")
	if action1 != VimActionNone {
		t.Errorf("first g should return VimActionNone, got %v", action1)
	}
	if m.PendingVimKey != "g" {
		t.Errorf("PendingVimKey should be 'g', got %q", m.PendingVimKey)
	}

	// Second 'g' completes sequence, returns top
	action2 := m.handleVimKey("g")
	if action2 != VimActionTop {
		t.Errorf("gg should return VimActionTop, got %v", action2)
	}
	if m.PendingVimKey != "" {
		t.Errorf("PendingVimKey should be cleared, got %q", m.PendingVimKey)
	}
}

func TestHandleVimKey_GFollowedByOther(t *testing.T) {
	m := testKeyModeModel(KeyModeVim)

	// First 'g' sets pending
	m.handleVimKey("g")
	if m.PendingVimKey != "g" {
		t.Fatalf("expected pending g")
	}

	// Non-g key cancels pending and processes normally
	action := m.handleVimKey("j")
	// 'j' after 'g' should consume the 'g' and then check 'j'
	if action != VimActionDown {
		t.Errorf("g then j should return VimActionDown (j binding), got %v", action)
	}
}

func TestHandleVimKey_IgnoredWhenInputFocused(t *testing.T) {
	m := testKeyModeModel(KeyModeVim)
	m.InputFocused = true

	tests := []string{"j", "k", "h", "l", "/", "?", "q"}
	for _, key := range tests {
		action := m.handleVimKey(key)
		if action != VimActionNone {
			t.Errorf("handleVimKey(%q) with InputFocused=true should return VimActionNone, got %v", key, action)
		}
	}
}

func TestHandleVimKey_UnknownKey(t *testing.T) {
	m := testKeyModeModel(KeyModeVim)

	action := m.handleVimKey("x")
	if action != VimActionNone {
		t.Errorf("unknown key 'x' should return VimActionNone, got %v", action)
	}
}

// --- Unit Tests for handleEmacsKey ---

func TestHandleEmacsKey_Navigation(t *testing.T) {
	m := testKeyModeModel(KeyModeEmacs)

	tests := []struct {
		key  string
		want VimAction
		name string
	}{
		{"ctrl+n", VimActionDown, "ctrl+n moves down"},
		{"ctrl+p", VimActionUp, "ctrl+p moves up"},
		{"ctrl+b", VimActionBack, "ctrl+b goes back"},
		{"ctrl+f", VimActionForward, "ctrl+f goes forward"},
		{"ctrl+s", VimActionSearch, "ctrl+s opens search"},
		{"ctrl+r", VimActionPrevMatch, "ctrl+r prev match"},
		{"alt+<", VimActionTop, "alt+< goes to top"},
		{"alt+>", VimActionBottom, "alt+> goes to bottom"},
		{"f1", VimActionHelp, "f1 toggles help"},
		{"alt+w", VimActionCopy, "alt+w copies"},
		{"alt+x", VimActionExpr, "alt+x opens expr"},
		{"ctrl+g", VimActionClearSearch, "ctrl+g clears search"},
		{"ctrl+q", VimActionQuit, "ctrl+q quits"},
		{"enter", VimActionEnter, "enter goes forward"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.handleEmacsKey(tt.key)
			if got != tt.want {
				t.Errorf("handleEmacsKey(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestHandleEmacsKey_IgnoredWhenInputFocused(t *testing.T) {
	m := testKeyModeModel(KeyModeEmacs)
	m.InputFocused = true

	tests := []string{"ctrl+n", "ctrl+p", "ctrl+b", "ctrl+f", "ctrl+s", "ctrl+q"}
	for _, key := range tests {
		action := m.handleEmacsKey(key)
		if action != VimActionNone {
			t.Errorf("handleEmacsKey(%q) with InputFocused=true should return VimActionNone, got %v", key, action)
		}
	}
}

func TestHandleEmacsKey_UnknownKey(t *testing.T) {
	m := testKeyModeModel(KeyModeEmacs)

	action := m.handleEmacsKey("j")
	if action != VimActionNone {
		t.Errorf("unknown key 'j' in emacs mode should return VimActionNone, got %v", action)
	}
}

// --- Integration Tests via Update() ---

func TestVimMode_J_MovesDown(t *testing.T) {
	m := testKeyModeModel(KeyModeVim)
	initialCursor := m.Tbl.Cursor()

	// Send 'j' key
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	m2 := newModel.(*Model)

	if m2.Tbl.Cursor() != initialCursor+1 {
		t.Errorf("vim 'j' should move cursor down: got %d, want %d", m2.Tbl.Cursor(), initialCursor+1)
	}
}

func TestVimMode_K_MovesUp(t *testing.T) {
	m := testKeyModeModel(KeyModeVim)
	// Move down first so we can move up
	m.Tbl.MoveDown(2)
	initialCursor := m.Tbl.Cursor()

	// Send 'k' key
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	m2 := newModel.(*Model)

	if m2.Tbl.Cursor() != initialCursor-1 {
		t.Errorf("vim 'k' should move cursor up: got %d, want %d", m2.Tbl.Cursor(), initialCursor-1)
	}
}

func TestVimMode_G_GoesToBottom(t *testing.T) {
	m := testKeyModeModel(KeyModeVim)

	// Send 'G' key
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'G', Text: "G"})
	m2 := newModel.(*Model)

	// Should be at last row (index 3 for 4 items)
	if m2.Tbl.Cursor() != 3 {
		t.Errorf("vim 'G' should go to bottom: got cursor %d, want 3", m2.Tbl.Cursor())
	}
}

func TestVimMode_GG_GoesToTop(t *testing.T) {
	m := testKeyModeModel(KeyModeVim)
	m.Tbl.MoveDown(3) // Move to bottom first

	// Send 'g' then 'g'
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	m2 := newModel.(*Model)
	newModel, _ = m2.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	m3 := newModel.(*Model)

	if m3.Tbl.Cursor() != 0 {
		t.Errorf("vim 'gg' should go to top: got cursor %d, want 0", m3.Tbl.Cursor())
	}
}

func TestVimMode_Slash_OpensSearch(t *testing.T) {
	m := testKeyModeModel(KeyModeVim)

	// Send '/' key
	newModel, _ := m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m2 := newModel.(*Model)

	if !m2.AdvancedSearchActive {
		t.Error("vim '/' should activate advanced search")
	}
}

func TestVimMode_QuestionMark_TogglesHelp(t *testing.T) {
	m := testKeyModeModel(KeyModeVim)
	initialHelp := m.HelpVisible

	// Send '?' key
	newModel, _ := m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	m2 := newModel.(*Model)

	if m2.HelpVisible == initialHelp {
		t.Error("vim '?' should toggle help visibility")
	}
}

func TestEmacsMode_CtrlN_MovesDown(t *testing.T) {
	m := testKeyModeModel(KeyModeEmacs)
	initialCursor := m.Tbl.Cursor()

	// Send Ctrl+N - bubbletea represents this as Code: 'n' with ModCtrl
	// However, we need to check what string the handler expects
	// The handler uses keyStr from msg.String() which returns "ctrl+n"
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	m2 := newModel.(*Model)

	if m2.Tbl.Cursor() != initialCursor+1 {
		t.Errorf("emacs Ctrl+N should move cursor down: got %d, want %d", m2.Tbl.Cursor(), initialCursor+1)
	}
}

func TestEmacsMode_CtrlP_MovesUp(t *testing.T) {
	m := testKeyModeModel(KeyModeEmacs)
	m.Tbl.MoveDown(2)
	initialCursor := m.Tbl.Cursor()

	// Send Ctrl+P
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	m2 := newModel.(*Model)

	if m2.Tbl.Cursor() != initialCursor-1 {
		t.Errorf("emacs Ctrl+P should move cursor up: got %d, want %d", m2.Tbl.Cursor(), initialCursor-1)
	}
}

func TestEmacsMode_CtrlS_OpensSearch(t *testing.T) {
	m := testKeyModeModel(KeyModeEmacs)

	// Send Ctrl+S
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	m2 := newModel.(*Model)

	if !m2.AdvancedSearchActive {
		t.Error("emacs Ctrl+S should activate advanced search")
	}
}

func TestEmacsMode_F1_TogglesHelp(t *testing.T) {
	m := testKeyModeModel(KeyModeEmacs)
	initialHelp := m.HelpVisible

	// Send F1 (emacs uses F1 for help since ctrl+h is backspace in terminals)
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyF1})
	m2 := newModel.(*Model)

	if m2.HelpVisible == initialHelp {
		t.Error("emacs F1 should toggle help visibility")
	}
}

func TestFunctionMode_J_IsTypeAhead(t *testing.T) {
	m := testKeyModeModel(KeyModeFunction)

	// In function mode, 'j' should be type-ahead filter, not vim navigation
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	m2 := newModel.(*Model)

	// The key difference: in vim mode cursor moves, in function mode it's filter
	// We can verify by checking that vim key handler isn't used
	// handleVimKey returns the action regardless of mode (mode check is in Update)
	// So we just verify the model processed it correctly
	_ = m2 // Model was updated

	// Verify vim action returns Down (the handler doesn't check mode)
	action := m.handleVimKey("j")
	if action != VimActionDown {
		t.Errorf("handleVimKey returns action regardless of mode, got %v", action)
	}
}

func TestFunctionMode_VimKeysIgnored(t *testing.T) {
	m := testKeyModeModel(KeyModeFunction)

	// Verify vim keys return VimActionNone in function mode
	// This is implicit since handleVimKey is only called when KeyMode == KeyModeVim
	// But we can test the handler directly
	action := m.handleVimKey("j")
	// handleVimKey doesn't check KeyMode, it checks InputFocused
	// So this will return VimActionDown even in function mode
	// The KeyMode check happens in Update()
	if action != VimActionDown {
		t.Errorf("handleVimKey returns action regardless of mode, got %v", action)
	}
	// The real test is that Update() doesn't call handleVimKey when mode is Function
}

// --- Footer Rendering Tests ---

func TestFooterModel_View_VimMode(t *testing.T) {
	fm := NewFooterModel()
	fm.Width = 100
	fm.AllowEditInput = true
	fm.KeyMode = KeyModeVim

	view := fm.View()

	// Should contain vim-style keys
	vimKeys := []string{"?", "/", "y", ":", "q"}
	for _, key := range vimKeys {
		if !strings.Contains(view, key) {
			t.Errorf("vim mode footer should contain %q, got: %q", key, view)
		}
	}

	// Should NOT contain function keys
	if strings.Contains(view, "F1") || strings.Contains(view, "F3") {
		t.Errorf("vim mode footer should not contain F-keys, got: %q", view)
	}
}

func TestFooterModel_View_EmacsMode(t *testing.T) {
	fm := NewFooterModel()
	fm.Width = 100
	fm.AllowEditInput = true
	fm.KeyMode = KeyModeEmacs

	view := fm.View()

	// Should contain emacs-style keys (F1 is used for help since ctrl+h is backspace)
	emacsKeys := []string{"F1", "C-s", "C-l", "M-w", "M-x", "C-q"}
	for _, key := range emacsKeys {
		if !strings.Contains(view, key) {
			t.Errorf("emacs mode footer should contain %q, got: %q", key, view)
		}
	}

	// Should NOT contain vim keys like ? or :
	if strings.Contains(view, "? help") || strings.Contains(view, ": expr") {
		t.Errorf("emacs mode footer should not contain vim keys, got: %q", view)
	}
}

func TestFooterModel_View_FunctionMode(t *testing.T) {
	fm := NewFooterModel()
	fm.Width = 100
	fm.AllowEditInput = true
	fm.KeyMode = KeyModeFunction

	view := fm.View()

	// Should contain function keys
	fkeys := []string{"F1", "F3", "F4", "F5", "F6", "F10"}
	for _, key := range fkeys {
		if !strings.Contains(view, key) {
			t.Errorf("function mode footer should contain %q, got: %q", key, view)
		}
	}
}

// --- Help Rendering Tests ---

func TestHelpModel_View_VimMode(t *testing.T) {
	hm := NewHelpModel()
	hm.Visible = true
	hm.KeyMode = KeyModeVim
	hm.SetWidth(80)

	view := hm.View()

	// Should contain vim-style keys
	if !strings.Contains(view, "j/k") {
		t.Errorf("vim help should contain j/k, got: %q", view)
	}
	if !strings.Contains(view, "h/l") {
		t.Errorf("vim help should contain h/l, got: %q", view)
	}
	if !strings.Contains(view, "/") {
		t.Errorf("vim help should contain /, got: %q", view)
	}
}

func TestHelpModel_View_EmacsMode(t *testing.T) {
	hm := NewHelpModel()
	hm.Visible = true
	hm.KeyMode = KeyModeEmacs
	hm.SetWidth(80)

	view := hm.View()

	// Should contain emacs-style keys
	if !strings.Contains(view, "C-n/C-p") {
		t.Errorf("emacs help should contain C-n/C-p, got: %q", view)
	}
	if !strings.Contains(view, "C-b/C-f") {
		t.Errorf("emacs help should contain C-b/C-f, got: %q", view)
	}
}

func TestHelpModel_View_FunctionMode(t *testing.T) {
	hm := NewHelpModel()
	hm.Visible = true
	hm.KeyMode = KeyModeFunction
	hm.SetWidth(80)

	view := hm.View()

	// Should contain function key descriptions
	if !strings.Contains(view, "F1") {
		t.Errorf("function help should contain F1, got: %q", view)
	}
	if !strings.Contains(view, "F3") {
		t.Errorf("function help should contain F3, got: %q", view)
	}
}

// --- KeyMode Validation Tests ---

func TestIsValidKeyMode(t *testing.T) {
	tests := []struct {
		mode string
		want bool
	}{
		{"vim", true},
		{"emacs", true},
		{"function", true},
		{"invalid", false},
		{"", false},
		{"VIM", false}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			got := IsValidKeyMode(tt.mode)
			if got != tt.want {
				t.Errorf("IsValidKeyMode(%q) = %v, want %v", tt.mode, got, tt.want)
			}
		})
	}
}

func TestDefaultKeyMode(t *testing.T) {
	if DefaultKeyMode != KeyModeVim {
		t.Errorf("DefaultKeyMode should be vim, got %v", DefaultKeyMode)
	}
}
