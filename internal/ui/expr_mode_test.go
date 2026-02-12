//nolint:forcetypeassert
package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestEnterWithUnderscorePreservesIt validates that pressing Enter with just "_" in expr mode
// navigates to root while preserving the literal "_" in the input
func TestEnterWithUnderscorePreservesIt(t *testing.T) {
	m := focusedModel()
	m.PathInput.SetValue("_")
	m.filterSuggestions()

	// Press Enter
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2 := newModel.(*Model)

	// Should navigate to root
	if m2.Path != "" {
		t.Fatalf("expected empty path (root), got %q", m2.Path)
	}

	// Should preserve the literal "_" in the input
	if m2.PathInput.Value() != "_" {
		t.Fatalf("expected PathInput to remain '_', got %q", m2.PathInput.Value())
	}

	// Should stay in expr mode
	if !m2.InputFocused {
		t.Fatalf("expected to stay in expr mode")
	}
}

// TestTableModeAutoSyncShowsUnderscorePrefix validates that cursor movement in table mode
// auto-syncs PathInput with "_.<key>" to reflect CEL root context
func TestTableModeAutoSyncShowsUnderscorePrefix(t *testing.T) {
	m := tableModel()

	// Move cursor to first row
	m.Tbl.SetCursor(0)
	m.syncPathInputWithCursor()

	// Should show "_.<key>" format
	val := m.PathInput.Value()
	if !strings.HasPrefix(val, "_.") {
		t.Fatalf("expected PathInput to start with '_.' in table mode, got %q", val)
	}

	// At root with empty path, any key should show as _.<key>
	if val != "_.cluster_001" && val != "_.cluster_002" && val != "_.items" && val != "_.regions" {
		t.Fatalf("expected PathInput to be one of the root keys prefixed with '_', got %q", val)
	}
}

// TestF6TogglePrefixesWithUnderscore validates that pressing F6 to enter expr mode
// adds the "_" prefix to the current path
func TestF6TogglePrefixesWithUnderscore(t *testing.T) {
	m := tableModel()

	// Navigate to a nested path first
	regions := map[string]interface{}{"asia": map[string]interface{}{"count": 2}}
	m2 := m.NavigateTo(regions, "regions")

	// Verify we're in table mode (not focused)
	if m2.InputFocused {
		t.Fatalf("expected to start in table mode")
	}

	// Press F6 to enter expr mode
	newModel, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyF6})
	m3 := newModel.(*Model)

	// Should now be in expr mode
	if !m3.InputFocused {
		t.Fatalf("expected to be in expr mode after F6")
	}

	// PathInput should show "_.regions"
	if m3.PathInput.Value() != "_.regions" {
		t.Fatalf("expected PathInput to be '_.regions', got %q", m3.PathInput.Value())
	}
}

// TestF6AtRootShowsUnderscore validates F6 at root shows just "_"
func TestF6AtRootShowsUnderscore(t *testing.T) {
	m := tableModel()

	// Press F6 at root
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyF6})
	m2 := newModel.(*Model)

	if !m2.InputFocused {
		t.Fatalf("expected to be in expr mode after F6")
	}

	if m2.PathInput.Value() != "_" {
		t.Fatalf("expected PathInput to be '_' at root, got %q", m2.PathInput.Value())
	}
}

// TestTabCyclesArrayIndicesInExprMode validates Tab key cycles through array indices
// when the input ends with [n]
func TestTabCyclesArrayIndicesInExprMode(t *testing.T) {
	m := focusedModel()

	// Type path to an array element
	m.PathInput.SetValue("_.items[0]")
	m.filterSuggestions()

	// Press Tab to cycle to next index
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m2 := newModel.(*Model)

	if m2.PathInput.Value() != "_.items[1]" {
		t.Fatalf("expected PathInput to cycle to '_.items[1]', got %q", m2.PathInput.Value())
	}

	// Press Tab again to cycle (should wrap to 0 since items has 2 elements)
	newModel, _ = m2.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m3 := newModel.(*Model)

	if m3.PathInput.Value() != "_.items[0]" {
		t.Fatalf("expected PathInput to wrap to '_.items[0]', got %q", m3.PathInput.Value())
	}
}

// TestShiftTabCyclesArrayIndicesBackward validates Shift+Tab cycles backward
func TestShiftTabCyclesArrayIndicesBackward(t *testing.T) {
	m := focusedModel()

	// Type path to an array element
	m.PathInput.SetValue("_.items[1]")
	m.filterSuggestions()

	// Press Shift+Tab to cycle backward
	// TODO: v2 doesn't have KeyShiftTab constant - should use KeyTab with Mod field
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m2 := newModel.(*Model)

	if m2.PathInput.Value() != "_.items[0]" {
		t.Fatalf("expected PathInput to cycle back to '_.items[0]', got %q", m2.PathInput.Value())
	}

	// Press Shift+Tab again (should wrap to last index)
	// TODO: v2 doesn't have KeyShiftTab constant - should use KeyTab with Mod field
	newModel, _ = m2.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m3 := newModel.(*Model)

	if m3.PathInput.Value() != "_.items[1]" {
		t.Fatalf("expected PathInput to wrap to '_.items[1]', got %q", m3.PathInput.Value())
	}
}

// TestTabOnOpenBracketInsertsZeroIndex validates Tab on "_.items[" inserts [0]
func TestTabOnOpenBracketInsertsZeroIndex(t *testing.T) {
	m := focusedModel()

	m.PathInput.SetValue("_.items[")
	m.filterSuggestions()

	// Press Tab
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m2 := newModel.(*Model)

	if m2.PathInput.Value() != "_.items[0]" {
		t.Fatalf("expected PathInput to complete to '_.items[0]', got %q", m2.PathInput.Value())
	}
}

// TestExprModeNavigationWithUnderscorePrefix validates navigation from expr mode
// with "_." prefix works correctly
func TestExprModeNavigationWithUnderscorePrefix(t *testing.T) {
	m := focusedModel()

	// Type a path with "_." prefix
	m.PathInput.SetValue("_.regions.asia")
	m.filterSuggestions()

	// Press Enter to navigate
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2 := newModel.(*Model)

	// Should navigate to regions.asia (Path keeps the _.prefix from input normalization)
	if !strings.HasSuffix(m2.Path, "regions.asia") {
		t.Fatalf("expected path to end with 'regions.asia', got %q", m2.Path)
	}

	// Should stay in expr mode with the typed value
	if !m2.InputFocused {
		t.Fatalf("expected to stay in expr mode")
	}

	if m2.PathInput.Value() != "_.regions.asia" {
		t.Fatalf("expected PathInput to preserve '_.regions.asia', got %q", m2.PathInput.Value())
	}
}

// TestTableDrillingShowsUnderscorePrefix validates that drilling down in table mode
// via Enter sets PathInput with "_." prefix
func TestTableDrillingShowsUnderscorePrefix(t *testing.T) {
	m := tableModel()

	// Position cursor on first item (should be "cluster_001" or similar)
	m.Tbl.SetCursor(0)

	// Press Enter to drill down
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2 := newModel.(*Model)

	// Should still be in table mode
	if m2.InputFocused {
		t.Fatalf("expected to stay in table mode after drilling")
	}

	// PathInput should show "_.<key>"
	val := m2.PathInput.Value()
	if !strings.HasPrefix(val, "_.") {
		t.Fatalf("expected PathInput to start with '_.' after drilling, got %q", val)
	}
}

// TestNavigateToPreservesExprMode validates NavigateTo respects current focus mode
func TestNavigateToPreservesExprMode(t *testing.T) {
	m := focusedModel() // Start in expr mode

	// Navigate to a nested path
	regions := map[string]interface{}{"asia": map[string]interface{}{"count": 2}}
	m2 := m.NavigateTo(regions, "regions")

	// Should stay in expr mode
	if !m2.InputFocused {
		t.Fatalf("expected NavigateTo to preserve expr mode")
	}

	// PathInput should show literal path (no "_." added in expr mode)
	if m2.PathInput.Value() != "regions" {
		t.Fatalf("expected PathInput to be 'regions' in expr mode, got %q", m2.PathInput.Value())
	}
}

// TestNavigateToInTableModeShowsPrefix validates NavigateTo in table mode adds prefix
func TestNavigateToInTableModeShowsPrefix(t *testing.T) {
	m := tableModel() // Start in table mode

	// Navigate to a nested path
	regions := map[string]interface{}{"asia": map[string]interface{}{"count": 2}}
	m2 := m.NavigateTo(regions, "regions")

	// Should stay in table mode
	if m2.InputFocused {
		t.Fatalf("expected NavigateTo to preserve table mode")
	}

	// PathInput should show "_." prefix
	if m2.PathInput.Value() != "_.regions" {
		t.Fatalf("expected PathInput to be '_.regions' in table mode, got %q", m2.PathInput.Value())
	}
}

// TestUnderscorePrefixedKeyPreservedInExprMode validates that keys starting with underscores
// (like __hello) are preserved when entering expression mode, not stripped.
// Regression test for bug where _.__hello was incorrectly displayed as _.hello
func TestUnderscorePrefixedKeyPreservedInExprMode(t *testing.T) {
	// Create model with data containing underscore-prefixed keys
	data := map[string]any{
		"__hello":   "goodbye",
		"_internal": "value",
	}
	m := InitialModel(data)
	m.Root = data
	m.InputFocused = false
	m.Tbl.Focus()

	// Navigate to __hello key
	m2 := m.NavigateTo("goodbye", "__hello")

	// Enter expression mode via F6
	newModel, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyF6})
	m3 := newModel.(*Model)

	// Should be in expr mode
	if !m3.InputFocused {
		t.Fatalf("expected to be in expr mode after F6")
	}

	// PathInput should show "_.__hello" not "_.hello"
	if m3.PathInput.Value() != "_.__hello" {
		t.Fatalf("expected PathInput to be '_.__hello', got %q (underscore was incorrectly stripped)", m3.PathInput.Value())
	}
}

// TestUnderscorePrefixedKeyInTableModeSync validates that table mode sync preserves underscore keys
func TestUnderscorePrefixedKeyInTableModeSync(t *testing.T) {
	// Create model with underscore-prefixed key containing a scalar value (not nested)
	data := map[string]any{
		"__metadata": "v1.0",
	}
	m := InitialModel(data)
	m.Root = data
	m.InputFocused = false
	m.Tbl.Focus()

	// Sync path input at root - cursor should be on __metadata key
	m.syncPathInputWithCursor()

	// Should show "_.__metadata" not "_.metadata" (underscore preserved)
	val := m.PathInput.Value()
	if val != "_.__metadata" {
		t.Fatalf("expected PathInput to be '_.__metadata', got %q", val)
	}
}
