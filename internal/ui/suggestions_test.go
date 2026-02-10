package ui

import (
	"testing"
)

// TestSuggestionsModelView tests the suggestions component rendering
// NOTE: Suggestions are now shown in the status bar, not as a dropdown
// This test verifies that the SuggestionsModel.View() returns empty string
func TestSuggestionsModelView(t *testing.T) {
	tests := []struct {
		name                string
		inputFocused        bool
		showSuggestions     bool
		filteredSuggestions []string
		selectedSuggestion  int
		inputValue          string
		expectedOutput      string
	}{
		{
			name:                "always returns empty (suggestions in status bar)",
			inputFocused:        true,
			showSuggestions:     true,
			filteredSuggestions: []string{"filter()", "map()"},
			selectedSuggestion:  0,
			inputValue:          "items.",
			expectedOutput:      "",
		},
		{
			name:                "returns empty when not focused",
			inputFocused:        false,
			showSuggestions:     true,
			filteredSuggestions: []string{"filter()", "map()"},
			expectedOutput:      "",
		},
		{
			name:                "returns empty when not shown",
			inputFocused:        true,
			showSuggestions:     false,
			filteredSuggestions: []string{"filter()", "map()"},
			expectedOutput:      "",
		},
		{
			name:                "returns empty when empty list",
			inputFocused:        true,
			showSuggestions:     true,
			filteredSuggestions: []string{},
			expectedOutput:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewSuggestionsModel()
			model.InputFocused = tt.inputFocused
			model.ShowSuggestions = tt.showSuggestions
			model.FilteredSuggestions = tt.filteredSuggestions
			model.SelectedSuggestion = tt.selectedSuggestion
			model.InputValue = tt.inputValue
			model.WinHeight = 24
			model.TableHeight = 10

			output := model.View("❯ ")

			if output != tt.expectedOutput {
				t.Errorf("expected output %q, got %q", tt.expectedOutput, output)
			}
		})
	}
}

// TestSuggestionsModelViewWithNoColor tests suggestions rendering without colors
// NOTE: Suggestions are now shown in the status bar, not as a dropdown
func TestSuggestionsModelViewWithNoColor(t *testing.T) {
	model := NewSuggestionsModel()
	model.InputFocused = true
	model.ShowSuggestions = true
	model.FilteredSuggestions = []string{"filter() - list.filter(x, expr) -> list [method]"}
	model.SelectedSuggestion = 0
	model.InputValue = "items."
	model.NoColor = true
	model.WinHeight = 24
	model.TableHeight = 10

	output := model.View("❯ ")

	// Should return empty (suggestions are in status bar)
	if output != "" {
		t.Errorf("expected empty output (suggestions in status bar), got %q", output)
	}
}

// TestSuggestionsModelViewWindowSizing tests that suggestions adapt to window size
// NOTE: Suggestions are now shown in the status bar, not as a dropdown
func TestSuggestionsModelViewWindowSizing(t *testing.T) {
	model := NewSuggestionsModel()
	model.InputFocused = true
	model.ShowSuggestions = true
	model.SelectedSuggestion = 0
	model.InputValue = "items."
	model.NoColor = true

	// Create many suggestions
	suggestions := make([]string, 10)
	for i := 0; i < 10; i++ {
		suggestions[i] = "filter() - list.filter(x, expr) -> list [method]"
	}
	model.FilteredSuggestions = suggestions

	// Small window
	model.WinHeight = 10
	model.TableHeight = 5
	outputSmall := model.View("❯ ")

	// Larger window
	model.WinHeight = 24
	model.TableHeight = 10
	outputLarge := model.View("❯ ")

	// Both should return empty (suggestions are in status bar)
	if outputSmall != "" {
		t.Errorf("expected empty output in small window (suggestions in status bar), got %q", outputSmall)
	}
	if outputLarge != "" {
		t.Errorf("expected empty output in large window (suggestions in status bar), got %q", outputLarge)
	}
}
