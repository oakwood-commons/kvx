package ui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

// updateViewMode determines whether the current node should be rendered as a
// list view, detail view, or the default table view based on the DisplaySchema.
func (m *Model) updateViewMode(node interface{}) {
	if m.DisplaySchema == nil {
		m.ViewMode = ""
		m.ListViewState = nil
		m.DetailViewState = nil
		m.StatusViewState = nil
		return
	}

	// Status view takes priority — it's a top-level screen, not a drill-down view.
	if m.DisplaySchema.Status != nil && m.DisplaySchema.Status.TitleField != "" {
		m.ViewMode = "status"
		m.StatusViewState = buildStatusViewModel(
			node, m.DisplaySchema, m.KeyMode, m.NoColor, m.DoneChan,
			m.WinWidth, m.WinHeight,
		)
		m.ListViewState = nil
		m.DetailViewState = nil
		return
	}

	// If we're coming from a list view drill-in, check for detail view
	if m.ViewMode == "list" || m.ViewMode == "detail" {
		// If the node is a single object and we have detail config, show detail
		if _, isObj := node.(map[string]interface{}); isObj {
			if m.DisplaySchema.Detail != nil {
				m.ViewMode = "detail"
				m.DetailViewState = buildDetailViewModel(node, m.DisplaySchema, m.WinWidth, m.WinHeight)
				m.ListViewState = nil
				return
			}
		}
	}

	// Check if node is a homogeneous array of objects for list view
	if m.DisplaySchema.List != nil && m.DisplaySchema.List.TitleField != "" && isHomogeneousObjectArray(node) {
		m.ViewMode = "list"
		m.ListViewState = buildListViewModel(node, m.DisplaySchema, m.WinWidth, m.WinHeight)
		m.DetailViewState = nil
		return
	}

	// Default: table view
	m.ViewMode = ""
	m.ListViewState = nil
	m.DetailViewState = nil
}

// activeCustomView returns the active CustomView for the current ViewMode,
// or nil when the default table view is active.
func (m *Model) activeCustomView() CustomView {
	switch m.ViewMode {
	case "list":
		if m.ListViewState != nil {
			return m.ListViewState
		}
	case "detail":
		if m.DetailViewState != nil {
			return m.DetailViewState
		}
	case "status":
		if m.StatusViewState != nil {
			return m.StatusViewState
		}
	}
	return nil
}

// resolveViewAction maps a raw keyStr to a VimAction, respecting the current KeyMode.
// Arrow keys and universal keys (ctrl+c, enter, esc, f1, backspace) are always
// resolved regardless of mode.  Mode-specific letter keys (j/k/h/l in vim,
// ctrl+n/ctrl+p/ctrl+b/ctrl+f in emacs) are resolved through the binding maps.
// In function mode only arrow keys, function keys, and universal keys are active.
func (m *Model) resolveViewAction(keyStr string) VimAction {
	// Universal keys first — always active in every mode.
	switch keyStr {
	case "up":
		return VimActionUp
	case "down":
		return VimActionDown
	case "left":
		return VimActionBack
	case "right":
		return VimActionForward
	case "enter":
		return VimActionEnter
	case "home":
		return VimActionTop
	case "end":
		return VimActionBottom
	case "ctrl+c":
		return VimActionQuit
	case "f1":
		return VimActionHelp
	}

	// Mode-specific bindings.
	switch m.KeyMode {
	case KeyModeVim:
		if a, ok := VimKeyBindings[keyStr]; ok {
			// Map pending-g to Top for single-press in list view
			if a == VimActionPendingG {
				return VimActionTop
			}
			return a
		}
	case KeyModeEmacs:
		if a, ok := EmacsKeyBindings[keyStr]; ok {
			return a
		}
	case KeyModeFunction:
		// Function mode: no single-letter shortcuts.
		// Only F-keys are resolved (f1 already handled above).
	}

	return VimActionNone
}

// handleListViewKey handles key events when in list view mode.
// Returns (handled bool, model, cmd).
func (m *Model) handleListViewKey(keyStr string) (bool, tea.Model, tea.Cmd) {
	if m.ViewMode != "list" || m.ListViewState == nil {
		return false, m, nil
	}

	lv := m.ListViewState

	// When the search panel is active, intercept navigation keys while
	// letting text input fall through to handleSearchInput.
	if m.AdvancedSearchActive {
		// In filter mode, sync list filter in real-time from the query.
		// In search mode, leave lv unchanged until Enter commits.
		if m.ListPanelMode == ListPanelModeFilter {
			lv.Filter = m.AdvancedSearchQuery
			lv.Selected = 0
			lv.ScrollTop = 0
		}
		items := filterListItems(lv)

		switch keyStr {
		case "esc":
			// Close panel and discard the pending query.
			m.AdvancedSearchActive = false
			m.AdvancedSearchQuery = ""
			m.SearchInput.SetValue("")
			m.SearchInput.Blur()
			if m.ListPanelMode == ListPanelModeFilter {
				lv.Filter = ""
			}
			lv.Selected = 0
			lv.ScrollTop = 0
			m.ListPanelMode = ""
			return true, m, nil
		case "enter":
			// Commit and close the panel.
			m.AdvancedSearchActive = false
			m.SearchInput.Blur()
			if m.ListPanelMode == ListPanelModeSearch {
				// Deep search: commit query across all fields.
				lv.SearchQuery = m.AdvancedSearchQuery
				lv.Selected = 0
				lv.ScrollTop = 0
			}
			// In filter mode, lv.Filter is already set above.
			m.ListPanelMode = ""
			return true, m, nil
		case "up":
			if lv.Selected > 0 {
				lv.Selected--
			}
			return true, m, nil
		case "down":
			if len(items) > 0 && lv.Selected < len(items)-1 {
				lv.Selected++
			}
			return true, m, nil
		case "right":
			// Drill into selected item and close search
			if len(items) > 0 && lv.Selected < len(items) {
				m.AdvancedSearchActive = false
				m.SearchInput.Blur()
				item := items[lv.Selected]
				m.DetailSourcePath = m.Path
				m.storeCursorForPath(m.Path)
				newPath := buildPathWithKey(m.Path, fmt.Sprintf("[%d]", item.Index))
				arr, ok := m.Node.([]interface{})
				if !ok || item.Index >= len(arr) {
					return true, m, nil
				}
				m.ViewMode = "detail"
				newModel := m.NavigateTo(arr[item.Index], normalizePathForModel(newPath))
				newModel.PathKeys = parsePathKeys(newModel.Path)
				newModel.applyLayout(true)
				return true, newModel, nil
			}
			return true, m, nil
		case "ctrl+c":
			return true, m, tea.Quit
		default:
			// Let all other keys (printable chars, backspace, etc.) fall
			// through to handleSearchInput which manages the text input.
			return false, m, nil
		}
	}

	items := filterListItems(lv)

	// Resolve the key through the keymap system.
	action := m.resolveViewAction(keyStr)

	// When the list is empty, only allow quit, help, esc (clear filter), and back.
	if len(items) == 0 {
		switch action { //nolint:exhaustive // only subset applies when list is empty
		case VimActionQuit:
			return true, m, tea.Quit
		case VimActionHelp:
			m.HelpVisible = !m.HelpVisible
			m.applyLayout(true)
			return true, m, nil
		case VimActionBack:
			if lv.Filter != "" {
				lv.Filter = ""
				lv.Selected = 0
				lv.ScrollTop = 0
				return true, m, nil
			}
			if lv.SearchQuery != "" {
				lv.SearchQuery = ""
				lv.Selected = 0
				lv.ScrollTop = 0
				return true, m, nil
			}
			result, cmd := m.navigateBack()
			return true, result, cmd
		default:
			// other actions are not applicable when the list is empty
		}
		if keyStr == "esc" && (lv.Filter != "" || lv.SearchQuery != "") {
			if lv.Filter != "" {
				lv.Filter = ""
			} else {
				lv.SearchQuery = ""
			}
			lv.Selected = 0
			lv.ScrollTop = 0
			return true, m, nil
		}
		return false, m, nil
	}

	switch action { //nolint:exhaustive // unhandled actions fall through to type-ahead filter
	case VimActionUp:
		if lv.Selected > 0 {
			lv.Selected--
		}
		return true, m, nil

	case VimActionDown:
		if lv.Selected < len(items)-1 {
			lv.Selected++
		}
		return true, m, nil

	case VimActionTop:
		lv.Selected = 0
		lv.ScrollTop = 0
		return true, m, nil

	case VimActionBottom:
		lv.Selected = len(items) - 1
		return true, m, nil

	case VimActionForward, VimActionEnter:
		// Drill into the selected item
		if lv.Selected < len(items) {
			item := items[lv.Selected]
			m.DetailSourcePath = m.Path
			m.storeCursorForPath(m.Path)

			newPath := buildPathWithKey(m.Path, fmt.Sprintf("[%d]", item.Index))
			arr, ok := m.Node.([]interface{})
			if !ok || item.Index >= len(arr) {
				return true, m, nil
			}
			childNode := arr[item.Index]

			// Force detail view mode before NavigateTo
			m.ViewMode = "detail"
			newModel := m.NavigateTo(childNode, normalizePathForModel(newPath))
			newModel.PathKeys = parsePathKeys(newModel.Path)
			newModel.applyLayout(true)
			return true, newModel, nil
		}
		return true, m, nil

	case VimActionBack:
		// Clear filter first, then search, then navigate back.
		if lv.Filter != "" {
			lv.Filter = ""
			lv.Selected = 0
			lv.ScrollTop = 0
			return true, m, nil
		}
		if lv.SearchQuery != "" {
			lv.SearchQuery = ""
			lv.Selected = 0
			lv.ScrollTop = 0
			return true, m, nil
		}
		// Navigate back to parent
		result, cmd := m.navigateBack()
		return true, result, cmd

	case VimActionSearch:
		// Open the search panel in search mode (deep search on Enter).
		m.ListPanelMode = ListPanelModeSearch
		m.AdvancedSearchActive = true
		m.AdvancedSearchCommitted = false
		m.AdvancedSearchQuery = ""
		m.SearchInput.SetValue("")
		m.SearchInput.SetCursor(0)
		return true, m, m.SearchInput.Focus()

	case VimActionFilter:
		// Open the search panel in filter mode (real-time title+subtitle).
		m.ListPanelMode = ListPanelModeFilter
		m.AdvancedSearchActive = true
		m.AdvancedSearchCommitted = false
		m.AdvancedSearchQuery = lv.Filter
		m.SearchInput.SetValue(lv.Filter)
		m.SearchInput.SetCursor(len(lv.Filter))
		return true, m, m.SearchInput.Focus()

	case VimActionCopy, VimActionNextMatch, VimActionPrevMatch, VimActionClearSearch:
		// No meaningful use in list view; consume to prevent fallthrough.
		return true, m, nil

	case VimActionQuit:
		return true, m, tea.Quit

	case VimActionHelp:
		m.HelpVisible = !m.HelpVisible
		m.applyLayout(true)
		return true, m, nil
	}

	// esc always clears filter/search or navigates back, regardless of mode.
	if keyStr == "esc" {
		if lv.Filter != "" {
			lv.Filter = ""
			lv.Selected = 0
			lv.ScrollTop = 0
			return true, m, nil
		}
		if lv.SearchQuery != "" {
			lv.SearchQuery = ""
			lv.Selected = 0
			lv.ScrollTop = 0
			return true, m, nil
		}
		result, cmd := m.navigateBack()
		return true, result, cmd
	}

	// Type-ahead filter: single printable rune (only when action is unresolved).
	if action == VimActionNone {
		if len(keyStr) == 1 && keyStr >= " " && keyStr <= "~" {
			lv.Filter += keyStr
			lv.Selected = 0
			lv.ScrollTop = 0
			return true, m, nil
		}
		// Backspace to remove filter character
		if keyStr == "backspace" && lv.Filter != "" {
			lv.Filter = lv.Filter[:len(lv.Filter)-1]
			lv.Selected = 0
			lv.ScrollTop = 0
			return true, m, nil
		}
	}

	return false, m, nil
}

// handleDetailViewKey handles key events when in detail view mode.
// Returns (handled bool, model, cmd).
func (m *Model) handleDetailViewKey(keyStr string) (bool, tea.Model, tea.Cmd) {
	if m.ViewMode != "detail" || m.DetailViewState == nil {
		return false, m, nil
	}

	dv := m.DetailViewState
	action := m.resolveViewAction(keyStr)

	switch action { //nolint:exhaustive // only navigation subset applies in detail view
	case VimActionUp:
		if dv.ScrollTop > 0 {
			dv.ScrollTop--
		}
		return true, m, nil

	case VimActionDown:
		dv.ScrollTop++
		return true, m, nil

	case VimActionForward, VimActionEnter:
		// Consume forward/enter as no-ops; the detail view is a read-only
		// sectioned display — drilling deeper into raw fields is not useful.
		return true, m, nil

	case VimActionBack:
		result, cmd := m.navigateBack()
		return true, result, cmd

	case VimActionTop:
		dv.ScrollTop = 0
		return true, m, nil

	case VimActionSearch, VimActionFilter, VimActionNextMatch, VimActionPrevMatch, VimActionClearSearch:
		// Not applicable in detail view; consume to prevent fallthrough.
		return true, m, nil

	case VimActionCopy:
		// Copy not applicable in display schema detail view; consume.
		return true, m, nil

	case VimActionQuit:
		return true, m, tea.Quit

	case VimActionHelp:
		m.HelpVisible = !m.HelpVisible
		m.applyLayout(true)
		return true, m, nil
	}

	// esc always navigates back, regardless of mode.
	if keyStr == "esc" {
		result, cmd := m.navigateBack()
		return true, result, cmd
	}

	return false, m, nil
}

// renderCustomViewContent renders the content for custom view modes (list, detail, status).
// Returns the rendered string and true if a custom view was rendered, or ("", false) for default.
func (m *Model) renderCustomViewContent() (string, bool) {
	cv := m.activeCustomView()
	if cv == nil {
		return "", false
	}

	// Sync filter state before rendering/counting.
	m.syncCustomViewFilter()

	width := m.WinWidth
	height := m.WinHeight - 6 // account for borders, footer, status
	content := cv.Render(width, height, m.NoColor)
	return content, true
}

// customViewRowCount returns the item count for the footer label in custom view modes.
func (m *Model) customViewRowCount() (count int, selected int, label string) {
	cv := m.activeCustomView()
	if cv == nil {
		return 0, 0, ""
	}

	// Sync filter state before counting.
	m.syncCustomViewFilter()

	return cv.RowCount()
}

// syncCustomViewFilter syncs the search panel query into the list view filter
// when the list panel mode is "filter" and the search panel is active.
func (m *Model) syncCustomViewFilter() {
	if m.ViewMode == "list" && m.ListViewState != nil &&
		m.AdvancedSearchActive && m.ListPanelMode == ListPanelModeFilter {
		m.ListViewState.Filter = m.AdvancedSearchQuery
	}
}

// customSearchTitle returns the panel title override for the search/filter panel
// when a custom view is active. Returns "" when no override is needed
// (i.e. the default "Search" title should be used).
func (m *Model) customSearchTitle() string {
	cv := m.activeCustomView()
	if cv == nil {
		return ""
	}
	// The view itself may provide a title override.
	if t := cv.SearchTitle(); t != "" {
		return t
	}
	// Model-level overrides (e.g. list panel filter mode).
	if m.ViewMode == "list" && m.ListPanelMode == ListPanelModeFilter {
		return "Filter"
	}
	return "" // default "Search"
}

// handleStatusViewKey handles key events when in status view mode.
// Returns (handled bool, model, cmd).
func (m *Model) handleStatusViewKey(msg tea.KeyPressMsg) (bool, tea.Model, tea.Cmd) {
	if m.ViewMode != "status" || m.StatusViewState == nil {
		return false, m, nil
	}

	sv, cmd := m.StatusViewState.Update(msg)
	if updated, ok := sv.(*StatusViewModel); ok {
		m.StatusViewState = updated
	}
	return true, m, cmd
}
