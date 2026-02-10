package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestFooterModel_Update(t *testing.T) {
	m := NewFooterModel()
	m.Width = 100
	updated, cmd := m.Update(tea.KeyPressMsg{})

	// Footer is passive, so it should return the same model (can't compare structs with slices directly)
	if updated.Width != m.Width {
		t.Errorf("Update should return model with same width")
	}
	if cmd != nil {
		t.Errorf("Update should return nil command")
	}
}

func TestFooterModel_View(t *testing.T) {
	m := NewFooterModel()
	m.Width = 100
	m.AllowEditInput = true
	m.KeyMode = KeyModeFunction // Explicitly test function mode

	view := m.View()
	if view == "" {
		t.Error("Footer View should not return empty string")
	}

	// Should contain at least one function key
	if !containsAny(view, []string{"F1", "F3", "F5", "F6", "F10"}) {
		t.Errorf("Footer should contain function keys, got: %q", view)
	}

	// Test with AllowEditInput = false
	m.AllowEditInput = false
	view2 := m.View()
	if view2 == "" {
		t.Error("Footer View should not return empty string even when AllowEditInput is false")
	}
}

func TestFooterModel_SetWidth(t *testing.T) {
	m := NewFooterModel()
	m.SetWidth(150)
	if m.Width != 150 {
		t.Errorf("SetWidth should set width to 150, got %d", m.Width)
	}
}

func TestDebugModel_Update(t *testing.T) {
	m := NewDebugModel()
	m.Width = 100
	updated, cmd := m.Update(tea.KeyPressMsg{})

	// Debug is passive, so it should return the same model
	if updated.Width != m.Width {
		t.Errorf("Update should return model with same width")
	}
	if cmd != nil {
		t.Errorf("Update should return nil command")
	}
}

func TestDebugModel_SetVisible(t *testing.T) {
	m := NewDebugModel()
	m.Visible = true
	m.LastDebugOutput = "test output"
	m.LastDebugValues = "test values"

	m.SetVisible(false)
	if m.Visible {
		t.Error("SetVisible(false) should set Visible to false")
	}
	if m.LastDebugOutput != "" {
		t.Error("SetVisible(false) should clear LastDebugOutput")
	}
	if m.LastDebugValues != "" {
		t.Error("SetVisible(false) should clear LastDebugValues")
	}

	m.SetVisible(true)
	if !m.Visible {
		t.Error("SetVisible(true) should set Visible to true")
	}
}

func TestHelpModel_Update(t *testing.T) {
	m := NewHelpModel()
	m.Width = 100
	updated, cmd := m.Update(tea.KeyPressMsg{})

	// Help is passive, so it should return the same model
	if updated.Width != m.Width {
		t.Errorf("Update should return model with same width")
	}
	if cmd != nil {
		t.Errorf("Update should return nil command")
	}
}

func TestHelpModel_SetVisible(t *testing.T) {
	m := NewHelpModel()
	m.SetVisible(true)
	if !m.Visible {
		t.Error("SetVisible(true) should set Visible to true")
	}

	m.SetVisible(false)
	if m.Visible {
		t.Error("SetVisible(false) should set Visible to false")
	}
}

func TestLayoutManager_GetWidth(t *testing.T) {
	lm := NewLayoutManager(100, 50)
	if lm.GetWidth() != 100 {
		t.Errorf("GetWidth() = %d, expected 100", lm.GetWidth())
	}

	lm.SetDimensions(150, 75)
	if lm.GetWidth() != 150 {
		t.Errorf("GetWidth() = %d, expected 150 after SetDimensions", lm.GetWidth())
	}
}

func TestLayoutManager_GetHeight(t *testing.T) {
	lm := NewLayoutManager(100, 50)
	if lm.GetHeight() != 50 {
		t.Errorf("GetHeight() = %d, expected 50", lm.GetHeight())
	}

	lm.SetDimensions(150, 75)
	if lm.GetHeight() != 75 {
		t.Errorf("GetHeight() = %d, expected 75 after SetDimensions", lm.GetHeight())
	}
}

func TestRegisterMenuAction(t *testing.T) {
	// Reset menu actions
	currentMenuActions = nil

	action := func(m *Model) tea.Cmd {
		return nil
	}

	RegisterMenuAction("test_action", action)

	actions := CurrentMenuActions()
	if actions["test_action"] == nil {
		t.Error("RegisterMenuAction should register the action")
	}

	// Clean up
	currentMenuActions = nil
}

func TestMenuFromConfig(t *testing.T) {
	cfg := MenuConfigYAML{
		F1: MenuItemConfig{
			Label:  "Custom Help",
			Action: "help",
		},
		F6: MenuItemConfig{
			Label:   "Custom Expr",
			Action:  "expr_toggle",
			Enabled: func() *bool { b := false; return &b }(),
		},
	}

	allowEditInput := true
	menu := MenuFromConfig(cfg, &allowEditInput)

	if menu.F1.Label != "Custom Help" {
		t.Errorf("MenuFromConfig should use custom label, got %q", menu.F1.Label)
	}

	if menu.F6.Enabled {
		t.Error("MenuFromConfig should respect Enabled flag")
	}

	// Test with nil allowEditInput
	menu2 := MenuFromConfig(cfg, nil)
	if menu2.F1.Label != "Custom Help" {
		t.Errorf("MenuFromConfig should work with nil allowEditInput, got %q", menu2.F1.Label)
	}
}

func TestSetExpressionProvider(t *testing.T) {
	// Ensure we reset after test to avoid polluting other tests
	defer ResetExpressionProvider()

	// Create a test provider
	testProvider := &testExpressionProvider{}
	SetExpressionProvider(testProvider)

	if exprProvider != testProvider {
		t.Error("SetExpressionProvider should set the provider")
	}

	// Test with nil (should not change)
	SetExpressionProvider(nil)
	if exprProvider != testProvider {
		t.Error("SetExpressionProvider(nil) should not change the provider")
	}
}

func TestDefaultExpressionProvider(t *testing.T) {
	provider := DefaultExpressionProvider()
	if provider == nil {
		t.Error("DefaultExpressionProvider should return a non-nil provider")
	}

	// Should be able to evaluate expressions
	result, err := provider.Evaluate("1 + 1", map[string]interface{}{})
	if err != nil {
		t.Errorf("DefaultExpressionProvider should evaluate expressions, got error: %v", err)
	}
	if result != int64(2) {
		t.Errorf("DefaultExpressionProvider evaluation result = %v, expected 2", result)
	}
}

// Helper functions
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if contains(s, substr) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			strings.Contains(s, substr))))
}

// Test expression provider for testing
type testExpressionProvider struct{}

func (t *testExpressionProvider) Evaluate(expr string, root interface{}) (interface{}, error) {
	return "test result", nil
}

func (t *testExpressionProvider) DiscoverSuggestions() []string {
	return []string{"test()"}
}

func (t *testExpressionProvider) IsExpression(expr string) bool {
	return expr == "test"
}
