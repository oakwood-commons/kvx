package ui

import (
	tea "charm.land/bubbletea/v2"
)

// SuggestionsModel represents the Bubble Tea UI model for the suggestions dropdown
type SuggestionsModel struct {
	FilteredSuggestions []string
	SelectedSuggestion  int
	ShowSuggestions     bool
	InputValue          string
	InputFocused        bool
	NoColor             bool
	WinHeight           int
	TableHeight         int
}

// NewSuggestionsModel creates a new instance of the SuggestionsModel
func NewSuggestionsModel() SuggestionsModel {
	return SuggestionsModel{
		FilteredSuggestions: []string{},
		SelectedSuggestion:  0,
		ShowSuggestions:     false,
	}
}

// Update handles messages for the suggestions dropdown (currently no-op)
func (m SuggestionsModel) Update(_ tea.Msg) (SuggestionsModel, tea.Cmd) {
	return m, nil
}

// View renders the suggestions dropdown
// NOTE: Suggestions are now shown in the status bar, not as a dropdown
//
//nolint:unparam // _ parameter is required by interface
func (m SuggestionsModel) View(_ string) string {
	// Always return empty - suggestions are shown in status bar instead
	return ""
}
