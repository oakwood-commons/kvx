package ui

import (
	"strings"
)

// PanelLayoutModelOptions configures how model state is mapped to the panel layout renderer.
type PanelLayoutModelOptions struct {
	AppName        string
	HelpTitle      string
	HelpText       string
	SnapshotHeader bool
	InputVisible   bool
	HideFooter     bool // Hide the footer bar (for non-interactive display)
}

func panelLayoutStateFromModel(m *Model, opts PanelLayoutModelOptions) PanelLayoutState {
	if m == nil {
		return PanelLayoutState{}
	}

	rowCount := len(m.AllRows)
	if m.AdvancedSearchActive && len(m.AdvancedSearchResults) > 0 {
		rowCount = len(m.AdvancedSearchResults)
	} else if rowCount == 0 && len(m.Tbl.Rows()) > 0 {
		rowCount = len(m.Tbl.Rows())
	}

	selected := m.Tbl.Cursor()
	if selected < 0 {
		selected = 0
	}

	displayNode := m.Node
	filteredCount := -1
	if !m.AdvancedSearchActive && m.FilterActive && strings.TrimSpace(m.FilterBuffer) != "" {
		if cur, ok := m.Node.(map[string]interface{}); ok {
			filtered := map[string]interface{}{}
			filterLower := strings.ToLower(strings.TrimSpace(m.FilterBuffer))
			for k, v := range cur {
				if strings.HasPrefix(strings.ToLower(k), filterLower) {
					filtered[k] = v
				}
			}
			displayNode = filtered
			filteredCount = len(filtered)
		}
	}
	// Handle 'f' key map filter mode
	if !m.AdvancedSearchActive && m.MapFilterActive {
		if cur, ok := m.Node.(map[string]interface{}); ok {
			if m.MapFilterQuery == "" {
				// Empty query shows all keys
				displayNode = cur
				filteredCount = len(cur)
			} else {
				filtered := map[string]interface{}{}
				queryLower := strings.ToLower(m.MapFilterQuery)
				for k := range cur {
					// Match key prefix (same logic as applyMapFilter)
					if strings.HasPrefix(strings.ToLower(k), queryLower) {
						filtered[k] = cur[k]
					}
				}
				displayNode = filtered
				filteredCount = len(filtered)
			}
		}
	}
	if filteredCount >= 0 {
		rowCount = filteredCount
	}

	pathLabel := ""
	if m.InputFocused {
		pathLabel = formatExprDisplay(m.ExprDisplay)
		if pathLabel == "" {
			pathLabel = formatExprDisplay(m.Path)
		}
	} else {
		pathLabel = formatPathForDisplay(strings.TrimSpace(m.selectedRowPath()))
	}
	if pathLabel == "" {
		pathLabel = "_"
	}

	searchResults := []searchHit{}
	if m.AdvancedSearchActive && len(m.AdvancedSearchResults) > 0 {
		searchResults = make([]searchHit, len(m.AdvancedSearchResults))
		for i, res := range m.AdvancedSearchResults {
			searchResults[i] = searchHit(res)
		}
	}

	input := m.PathInput
	var inputRef PanelLayoutInput = &input
	if m.AdvancedSearchActive {
		m.SearchInput.SetValue(m.AdvancedSearchQuery)
		m.SearchInput.SetCursor(len(m.AdvancedSearchQuery))
		inputRef = &m.SearchInput
	} else if m.MapFilterActive {
		m.MapFilterInput.SetValue(m.MapFilterQuery)
		m.MapFilterInput.SetCursor(len(m.MapFilterQuery))
		inputRef = &m.MapFilterInput
	}
	title := strings.TrimSpace(opts.AppName)
	if title == "" {
		title = "kvx"
	}

	helpText := strings.TrimSpace(opts.HelpText)
	popupText := strings.TrimSpace(helpPopupText(m))
	if m.HelpVisible && popupText != "" {
		helpText = ""
	}

	infoMessage := ""
	infoError := false
	if strings.TrimSpace(m.ErrMsg) != "" {
		infoMessage = m.ErrMsg
		infoError = m.StatusType == "error"
	} else if m.InputFocused {
		infoMessage = m.detectFunctionHelp()
		if infoMessage == "" && m.ShowSuggestionSummary && m.SuggestionSummary != "" {
			infoMessage = m.SuggestionSummary
		}
	} else if m.DecodedActive {
		infoMessage = "✓ decoded"
	} else if hint := m.decodeHintForSelectedRow(); hint != "" {
		infoMessage = hint
	}

	return PanelLayoutState{
		WinWidth:        m.WinWidth,
		WinHeight:       m.WinHeight,
		NoColor:         m.NoColor,
		SnapshotHeader:  opts.SnapshotHeader,
		DebugEnabled:    m.DebugMode,
		AllowEditInput:  m.AllowEditInput,
		HideFooter:      opts.HideFooter,
		KeyMode:         m.KeyMode,
		HelpVisible:     m.HelpVisible,
		HelpTitle:       opts.HelpTitle,
		HelpText:        helpText,
		Title:           title,
		InfoMessage:     infoMessage,
		InfoError:       infoError,
		InfoPopupText:   infoPopupText(m),
		HelpPopupText:   popupText,
		InputVisible:    opts.InputVisible,
		Input:           inputRef,
		InputPreferHead: false,
		ExprMode:        m.InputFocused,
		ExprType:        m.ExprType,
		SearchActive:    m.AdvancedSearchActive,
		SearchResults:   searchResults,
		MapFilterActive: m.MapFilterActive,
		DisplayNode:     displayNode,
		Node:            m.Node,
		RowCount:        rowCount,
		SelectedRow:     selected,
		PathLabel:       pathLabel,
		KeyColWidth:     m.KeyColWidth,
	}
}

func infoPopupText(m *Model) string {
	if m == nil {
		return ""
	}
	if !m.ShowInfoPopup || !m.InfoPopupEnabled {
		return ""
	}
	return strings.TrimSpace(m.InfoPopup)
}

func helpPopupText(m *Model) string {
	if m == nil {
		return ""
	}
	if !m.HelpVisible {
		return ""
	}
	return strings.TrimSpace(m.HelpPopupText)
}
