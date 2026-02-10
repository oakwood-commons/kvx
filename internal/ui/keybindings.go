package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/oakwood-commons/kvx/internal/navigator"
)

// KeyMode represents the keybinding mode for the UI.
type KeyMode string

const (
	// KeyModeVim enables vim-style keybindings (j/k/h/l navigation, / search).
	KeyModeVim KeyMode = "vim"
	// KeyModeEmacs enables emacs-style keybindings (future implementation).
	KeyModeEmacs KeyMode = "emacs"
	// KeyModeFunction disables single-key shortcuts, uses function keys only.
	KeyModeFunction KeyMode = "function"
)

// DefaultKeyMode is the default keybinding mode.
const DefaultKeyMode = KeyModeVim

// ValidKeyModes lists all valid key modes for validation.
var ValidKeyModes = []KeyMode{KeyModeVim, KeyModeEmacs, KeyModeFunction}

// IsValidKeyMode checks if a key mode string is valid.
func IsValidKeyMode(mode string) bool {
	for _, m := range ValidKeyModes {
		if string(m) == mode {
			return true
		}
	}
	return false
}

// VimAction represents an action triggered by a vim-style keybinding.
type VimAction string

const (
	VimActionNone        VimAction = ""
	VimActionDown        VimAction = "down"
	VimActionUp          VimAction = "up"
	VimActionBack        VimAction = "back"
	VimActionForward     VimAction = "forward"
	VimActionSearch      VimAction = "search"
	VimActionNextMatch   VimAction = "next_match"
	VimActionPrevMatch   VimAction = "prev_match"
	VimActionTop         VimAction = "top"
	VimActionBottom      VimAction = "bottom"
	VimActionHelp        VimAction = "help"
	VimActionCopy        VimAction = "copy"
	VimActionExpr        VimAction = "expr"
	VimActionQuit        VimAction = "quit"
	VimActionPendingG    VimAction = "pending_g" // Waiting for second key in gg sequence
	VimActionClearSearch VimAction = "clear_search"
	VimActionEnter       VimAction = "enter"
	VimActionFilter      VimAction = "filter" // Map filter mode ('f' key)
)

// VimKeyBindings maps keys to actions for vim mode.
// This is the default mapping; it can be overridden by config.
var VimKeyBindings = map[string]VimAction{
	"j":     VimActionDown,
	"k":     VimActionUp,
	"h":     VimActionBack,
	"l":     VimActionForward,
	"/":     VimActionSearch,
	"f":     VimActionFilter, // Map filter mode
	"n":     VimActionNextMatch,
	"N":     VimActionPrevMatch,
	"g":     VimActionPendingG,
	"G":     VimActionBottom,
	"?":     VimActionHelp,
	"y":     VimActionCopy,
	":":     VimActionExpr,
	"q":     VimActionQuit,
	"enter": VimActionEnter,
}

// EmacsKeyBindings maps keys to actions for emacs mode.
// Uses standard emacs keybindings with ctrl modifiers.
var EmacsKeyBindings = map[string]VimAction{
	"ctrl+n": VimActionDown,
	"ctrl+p": VimActionUp,
	"ctrl+b": VimActionBack,
	"ctrl+f": VimActionForward,
	"ctrl+s": VimActionSearch,
	"ctrl+l": VimActionFilter,    // Filter current map
	"ctrl+r": VimActionPrevMatch, // Reverse search in emacs
	"alt+<":  VimActionTop,
	"alt+>":  VimActionBottom,
	"f1":     VimActionHelp, // Use F1 for help (ctrl+h is backspace in terminals)
	"alt+w":  VimActionCopy,
	"alt+x":  VimActionExpr,
	"ctrl+g": VimActionClearSearch, // Cancel in emacs
	"ctrl+q": VimActionQuit,        // Quit
	"enter":  VimActionEnter,
}

// actionToVimAction maps config action names to VimAction constants.
var actionToVimAction = map[string]VimAction{
	"help":        VimActionHelp,
	"search":      VimActionSearch,
	"filter":      VimActionFilter,
	"copy":        VimActionCopy,
	"expr":        VimActionExpr,
	"expr_toggle": VimActionExpr,
	"quit":        VimActionQuit,
}

// UpdateKeyBindingsFromConfig rebuilds vim/emacs keybindings from menu config.
// Called after config is loaded to apply custom key mappings.
func UpdateKeyBindingsFromConfig(menu MenuConfig) {
	// Update Vim keybindings from config
	for _, item := range menu.Items {
		if !item.Enabled {
			continue
		}
		action := item.Action
		if action == "" {
			continue
		}
		vimAction, ok := actionToVimAction[action]
		if !ok {
			continue
		}
		// Update vim binding if config specifies a different key
		if item.Keys.Vim != "" {
			// Remove old bindings for this action
			for k, v := range VimKeyBindings {
				if v == vimAction {
					delete(VimKeyBindings, k)
				}
			}
			VimKeyBindings[item.Keys.Vim] = vimAction
		}
		// Update emacs binding if config specifies a different key
		if item.Keys.Emacs != "" {
			// Remove old bindings for this action
			for k, v := range EmacsKeyBindings {
				if v == vimAction {
					delete(EmacsKeyBindings, k)
				}
			}
			EmacsKeyBindings[item.Keys.Emacs] = vimAction
		}
	}
}

// handleVimKey processes a key press in vim mode and returns the action to take.
// Returns VimActionNone if the key is not a vim binding.
func (m *Model) handleVimKey(keyStr string) VimAction {
	// Don't process vim keys when in input mode (expr mode or search mode with focus)
	if m.InputFocused {
		return VimActionNone
	}

	// Check for pending 'g' key (for gg sequence)
	if m.PendingVimKey == "g" {
		m.PendingVimKey = ""
		if keyStr == "g" {
			return VimActionTop
		}
		// Not 'g', so the pending 'g' is consumed without action
		// Fall through to check if this key has its own binding
	}

	action, ok := VimKeyBindings[keyStr]
	if !ok {
		return VimActionNone
	}

	// Handle pending key sequences
	if action == VimActionPendingG {
		m.PendingVimKey = "g"
		return VimActionNone
	}

	return action
}

// handleEmacsKey processes a key press in emacs mode and returns the action to take.
// Returns VimActionNone if the key is not an emacs binding.
func (m *Model) handleEmacsKey(keyStr string) VimAction {
	// Don't process emacs keys when in input mode (expr mode or search mode with focus)
	if m.InputFocused {
		return VimActionNone
	}

	action, ok := EmacsKeyBindings[keyStr]
	if !ok {
		return VimActionNone
	}

	return action
}

// executeVimAction executes a vim action and returns the updated model and command.
func (m *Model) executeVimAction(action VimAction) (tea.Model, tea.Cmd) {
	switch action {
	case VimActionNone, VimActionPendingG:
		// These are handled by handleVimKey, shouldn't reach here
		return m, nil
	case VimActionDown:
		return m.vimNavigateDown()
	case VimActionUp:
		return m.vimNavigateUp()
	case VimActionBack:
		return m.vimNavigateBack()
	case VimActionForward, VimActionEnter:
		return m.vimNavigateForward()
	case VimActionSearch:
		return m.vimEnterSearch()
	case VimActionFilter:
		return m.vimEnterMapFilter()
	case VimActionNextMatch:
		return m.vimNextMatch()
	case VimActionPrevMatch:
		return m.vimPrevMatch()
	case VimActionTop:
		return m.vimGoToTop()
	case VimActionBottom:
		return m.vimGoToBottom()
	case VimActionHelp:
		return m.vimToggleHelp()
	case VimActionCopy:
		return m.vimCopy()
	case VimActionExpr:
		return m.vimEnterExpr()
	case VimActionQuit:
		return m.vimQuit()
	case VimActionClearSearch:
		return m.vimClearSearch()
	}
	return m, nil
}

// vimNavigateDown moves cursor down one row.
func (m *Model) vimNavigateDown() (tea.Model, tea.Cmd) {
	m.Tbl.MoveDown(1)
	m.clearErrorUnlessSticky()
	m.SyncTableState()
	m.syncPathInputWithCursor()
	return m, nil
}

// vimNavigateUp moves cursor up one row.
func (m *Model) vimNavigateUp() (tea.Model, tea.Cmd) {
	m.Tbl.MoveUp(1)
	m.clearErrorUnlessSticky()
	m.SyncTableState()
	m.syncPathInputWithCursor()
	return m, nil
}

// vimNavigateBack goes back one level (same as left arrow).
func (m *Model) vimNavigateBack() (tea.Model, tea.Cmd) {
	// Simulate left arrow key press by delegating to existing navigation logic
	// This is handled by returning a special marker that the Update function will process
	return m, nil // Placeholder - actual implementation will be in Update
}

// vimNavigateForward drills into the selected item (same as right arrow/enter).
func (m *Model) vimNavigateForward() (tea.Model, tea.Cmd) {
	// Placeholder - actual implementation will be in Update
	return m, nil
}

// vimEnterSearch activates search mode (same as F3).
func (m *Model) vimEnterSearch() (tea.Model, tea.Cmd) {
	// Use existing search action
	return m, menuActionSearch(m)
}

// vimEnterMapFilter activates map filter mode ('f' key).
// Only works when current node is a map - does nothing for arrays.
func (m *Model) vimEnterMapFilter() (tea.Model, tea.Cmd) {
	cmd := menuActionFilter(m)
	if cmd != nil {
		// Apply empty filter (shows all rows) when entering filter mode
		m.applyMapFilter()
	}
	return m, cmd
}

// vimNextMatch moves to the next search match.
func (m *Model) vimNextMatch() (tea.Model, tea.Cmd) {
	// Only works if we have committed search results
	if !m.hasCommittedSearch() {
		return m, nil
	}
	// Move down in results
	m.Tbl.MoveDown(1)
	m.clearErrorUnlessSticky()
	m.SyncTableState()
	m.syncPathInputWithCursor()
	return m, nil
}

// vimPrevMatch moves to the previous search match.
func (m *Model) vimPrevMatch() (tea.Model, tea.Cmd) {
	// Only works if we have committed search results
	if !m.hasCommittedSearch() {
		return m, nil
	}
	// Move up in results
	m.Tbl.MoveUp(1)
	m.clearErrorUnlessSticky()
	m.SyncTableState()
	m.syncPathInputWithCursor()
	return m, nil
}

// hasCommittedSearch returns true if there's an active committed search.
func (m *Model) hasCommittedSearch() bool {
	return m.SearchContextActive || (m.AdvancedSearchActive && len(m.AdvancedSearchResults) > 0)
}

// vimGoToTop moves cursor to the first row.
func (m *Model) vimGoToTop() (tea.Model, tea.Cmd) {
	m.Tbl.GotoTop()
	m.clearErrorUnlessSticky()
	m.SyncTableState()
	m.syncPathInputWithCursor()
	return m, nil
}

// vimGoToBottom moves cursor to the last row.
func (m *Model) vimGoToBottom() (tea.Model, tea.Cmd) {
	m.Tbl.GotoBottom()
	m.clearErrorUnlessSticky()
	m.SyncTableState()
	m.syncPathInputWithCursor()
	return m, nil
}

// vimToggleHelp toggles the help overlay (same as F1).
func (m *Model) vimToggleHelp() (tea.Model, tea.Cmd) {
	return m, menuActionHelp(m)
}

// vimCopy copies current path/expression (same as F5).
func (m *Model) vimCopy() (tea.Model, tea.Cmd) {
	return m, menuActionCopy(m)
}

// vimEnterExpr enters expression mode (same as F6).
func (m *Model) vimEnterExpr() (tea.Model, tea.Cmd) {
	return m, menuActionExprToggle(m)
}

// vimQuit exits the application (same as F10).
func (m *Model) vimQuit() (tea.Model, tea.Cmd) {
	return m, menuActionQuit(m)
}

// vimClearSearch clears any active search.
func (m *Model) vimClearSearch() (tea.Model, tea.Cmd) {
	if m.AdvancedSearchActive || m.SearchContextActive {
		m.clearSearchState()
		m.SearchContextActive = false
		m.SearchContextResults = nil
		m.SearchContextQuery = ""
		m.SearchContextBasePath = ""
		m.applyLayout(true)
	}
	return m, nil
}

// handleVimBackNavigation handles the 'h' key for back navigation.
// This method exists because back navigation is complex and handled inline in Update.
// We delegate to the caller by returning a special command that signals "process as left arrow".
func (m *Model) handleVimBackNavigation() (tea.Model, tea.Cmd) {
	// Navigate back - same logic as left arrow
	return m.navigateBack()
}

// handleVimForwardNavigation handles the 'l' and 'enter' keys for forward navigation.
func (m *Model) handleVimForwardNavigation() (tea.Model, tea.Cmd) {
	// Handle navigation in search results first
	if m.AdvancedSearchActive && len(m.AdvancedSearchResults) > 0 {
		return m.navigateForwardFromSearch()
	}
	// Regular navigation
	return m.navigateForward()
}

// navigateForwardFromSearch handles forward navigation when in search results.
func (m *Model) navigateForwardFromSearch() (tea.Model, tea.Cmd) {
	// Ensure cursor is within bounds
	m.SyncTableState()
	cur := m.Tbl.Cursor()
	if cur < 0 || cur >= len(m.AdvancedSearchResults) {
		return m, nil
	}

	result := m.AdvancedSearchResults[cur]
	// Combine base path with result path (result path is relative to search base)
	var fullPath string
	if m.AdvancedSearchBasePath == "" {
		fullPath = result.FullPath
	} else {
		// Check if result path starts with bracket notation (array index)
		if len(result.FullPath) > 0 && result.FullPath[0] == '[' {
			fullPath = m.AdvancedSearchBasePath + result.FullPath
		} else {
			fullPath = m.AdvancedSearchBasePath + "." + result.FullPath
		}
	}

	// Navigate to the combined path
	newNode, err := navigator.Resolve(m.Root, fullPath)
	if err != nil {
		m.ErrMsg = "Error: " + err.Error()
		m.StatusType = "error"
		return m, nil
	}

	// Use NavigateTo which updates existing model to avoid flicker
	newModel := m.NavigateTo(newNode, navigator.NormalizePath(fullPath))

	// Save search context for navigation back (like right arrow does)
	newModel.AdvancedSearchActive = false
	newModel.SearchContextActive = true
	newModel.SearchContextResults = make([]SearchResult, len(m.AdvancedSearchResults))
	copy(newModel.SearchContextResults, m.AdvancedSearchResults)
	newModel.SearchContextQuery = m.AdvancedSearchQuery
	newModel.SearchContextBasePath = m.AdvancedSearchBasePath

	newModel.SearchInput.Blur()
	newModel.applyLayout(true)
	return newModel, nil
}
