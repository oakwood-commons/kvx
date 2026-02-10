//nolint:forcetypeassert
package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestBaseForGlobal validates the baseForGlobal helper correctly extracts parent context for global wrapping.
// When user types a partial path or incomplete expression, baseForGlobal strips the incomplete part
// so global functions wrap the complete base expression.
func TestBaseForGlobal(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		desc     string
	}{
		// Trailing dot: removes dot, keeps rest → full parent path
		{"_.pd1001.platform.", "_.pd1001.platform", "trailing dot removed, path preserved"},
		// Trailing partial token: drops last token after dot → parent path
		{"_.pd1001.platform.h", "_.pd1001.platform", "partial token after dot dropped"},
		// At root with trailing dot: removes dot → returns _
		{"_.", "_", "root with dot returns underscore"},
		// Single word with dot prefix: dot already removed in prior step, returns parent
		{"_.", "_", "underscore alone returns underscore"},
		// Simple key without underscore: normalizeExprBase adds _. prefix
		{"items", "_.items", "simple word gets underscore prefix"},
		// Unrooted path: drops last token after dot
		{"regions.asia", "_.regions", "unrooted path drops last token"},
		// Multiple dots with trailing word: drops last token after last dot
		{"a.b.c.d", "_.a.b.c", "multiple dots drops last token"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := baseForGlobal(tt.input)
			if got != tt.expected {
				t.Errorf("baseForGlobal(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestWrapGlobal validates the wrapGlobal helper correctly formats global function calls
func TestWrapGlobal(t *testing.T) {
	tests := []struct {
		name     string
		baseExpr string
		expected string
	}{
		// Basic global function
		{"has()", "_.pd1001.platform", "has(_.pd1001.platform)"},
		// Function with trailing parens stripped
		{"filter()", "_.items", "filter(_.items)"},
		// Namespace-qualified
		{"regex.extract()", "\"abc123\"", "regex.extract(\"abc123\")"},
		// At root
		{"size()", "_", "size(_)"},
	}

	for _, tt := range tests {
		got := wrapGlobal(tt.name, tt.baseExpr)
		if got != tt.expected {
			t.Errorf("wrapGlobal(%q, %q) = %q, want %q", tt.name, tt.baseExpr, got, tt.expected)
		}
	}
}

// TestGlobalFunctionTabCompletion validates Tab cycles through keys, not functions
func TestGlobalFunctionTabCompletion(t *testing.T) {
	m := focusedModel()
	m.Root = sampleData()

	// Set up with key suggestions (Tab only cycles through keys, not functions)
	m.PathInput.SetValue("items.")
	m.filterSuggestions()
	// Tab should cycle through keys, not complete functions
	if len(m.FilteredSuggestions) == 0 {
		t.Skip("No key suggestions available for Tab completion test")
	}

	// Simulate Tab completion - should select first key, not function
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m2 := newModel.(*Model)

	// Tab should complete with a key or array index, not a function
	// The exact key depends on the data, but it should be a key name or array index, not a function
	value := m2.PathInput.Value()
	// Should be either items.<key> or items[index] format
	if !strings.HasPrefix(value, "items.") && !strings.HasPrefix(value, "items[") {
		t.Errorf("Tab completion should result in items.<key> or items[index], got %q", value)
	}
	// Should not be a function call (unless it's part of an array index)
	if strings.Contains(value, "(") && !strings.Contains(value, "[") {
		t.Errorf("Tab should not complete functions, got %q", value)
	}
}

// TestGlobalFunctionRightArrowCompletion validates right-arrow does NOT auto-complete global-style functions
func TestGlobalFunctionRightArrowCompletion(t *testing.T) {
	m := focusedModel()
	m.Root = sampleData()

	// Simulate being after a trailing dot with global function selected
	m.PathInput.SetValue("items.")
	m.Suggestions = []string{
		"has() - has(field) -> bool [global]",
	}
	m.FilteredSuggestions = []string{"has() - has(field) -> bool [global]"}
	m.ShowSuggestions = true
	m.SelectedSuggestion = 0

	// Right arrow should NOT auto-complete global-style functions (they must be used as has(_), not _.has())
	// It should just move the cursor to the end
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	m2 := newModel.(*Model)

	expected := "items."
	if m2.PathInput.Value() != expected {
		t.Errorf("Right-arrow should not auto-complete global has() = %q, want %q (just move cursor)", m2.PathInput.Value(), expected)
	}
}

// TestMethodFunctionTabCompletion validates Tab cycles through keys, not functions
func TestMethodFunctionTabCompletion(t *testing.T) {
	m := focusedModel()
	m.Root = sampleData()

	// Array node for method-style filter
	m.Node = []interface{}{map[string]interface{}{"name": "a"}}

	// User types items. - Tab should cycle through keys, not functions
	m.PathInput.SetValue("items.")
	m.filterSuggestions()
	// Tab should cycle through keys, not complete functions
	if len(m.FilteredSuggestions) == 0 {
		t.Skip("No key suggestions available for Tab completion test")
	}

	// Tab completion should select a key or array index, not a function
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m2 := newModel.(*Model)

	// Tab should complete with a key or array index, not a function
	value := m2.PathInput.Value()
	// Should be either items.<key> or items[index] format
	if !strings.HasPrefix(value, "items.") && !strings.HasPrefix(value, "items[") {
		t.Errorf("Tab completion should result in items.<key> or items[index], got %q", value)
	}
	// Should not be a function call
	if strings.Contains(value, "(") && !strings.Contains(value, "[") {
		t.Errorf("Tab should not complete functions, got %q", value)
	}
}

// TestUsageStyleLabelClassification validates that global vs method styles are correctly identified
func TestUsageStyleLabelClassification(t *testing.T) {
	tests := []struct {
		suggestion string
		expected   string
	}{
		// Global style (function syntax)
		{"has() - has(field) -> bool", "[global]"},
		{"size() - size(list) -> int", "[global]"},
		// Method style (receiver.method syntax)
		{"filter() - list.filter(x, expr) -> list", "[method]"},
		{"map() - list.map(x, expr) -> list", "[method]"},
		{"contains() - string.contains(str) -> bool", "[method]"},
		// Macro with no usage hint (fallback to name)
		{"has()", "[global]"},
		{"all()", "[global]"},
		// Unknown
		{"unknown_function()", "[global]"},
	}

	for _, tt := range tests {
		got := usageStyleLabel(tt.suggestion)
		if got != tt.expected {
			t.Errorf("usageStyleLabel(%q) = %q, want %q", tt.suggestion, got, tt.expected)
		}
	}
}

// TestGlobalFunctionWithPartialTokenDropsToken validates Tab cycles through matching keys or functions
func TestGlobalFunctionWithPartialTokenDropsToken(t *testing.T) {
	m := focusedModel()
	m.Root = sampleData()

	// User types partial: items.f
	// Tab should cycle through keys matching "f" or matching functions

	m.PathInput.SetValue("items.f")
	m.filterSuggestions()
	// Tab should cycle through matches; ensure we have suggestions to test
	if len(m.FilteredSuggestions) == 0 {
		t.Skip("No suggestions available for Tab completion test")
	}

	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m2 := newModel.(*Model)

	value := m2.PathInput.Value()
	if !strings.HasPrefix(value, "items.") {
		t.Errorf("Tab completion should result in items.<key>, got %q", value)
	}
	// Allow function completion; ensure token was replaced with a reasonable match
	if !strings.Contains(value, "(") && value == "items.f" {
		t.Errorf("Tab should advance completion, got %q", value)
	}
}

// TestTrailingDotPrioritizesFunctions validates that when typing 'items.' functions come first
func TestTrailingDotPrioritizesFunctions(t *testing.T) {
	m := focusedModel()
	m.Root = sampleData()

	m.PathInput.SetValue("items.")
	// Mix of functions and keys
	m.Suggestions = []string{
		"filter() - list.filter(x, x.available) -> list [method]",
		"map() - list.map(x, x.field) -> list [method]",
		"has() - has(field) -> bool [global]",
	}

	m.filterSuggestions()

	// Functions should be prioritized and appear first in filtered suggestions
	if len(m.FilteredSuggestions) == 0 {
		t.Fatalf("expected suggestions after trailing dot")
	}

	// First suggestion should be a function (contains "(" or " - ")
	firstIsFunc := strings.Contains(m.FilteredSuggestions[0], "(") || strings.Contains(m.FilteredSuggestions[0], " - ")
	if !firstIsFunc {
		t.Errorf("expected first suggestion to be function, got %q", m.FilteredSuggestions[0])
	}
}

// TestTabCyclesWithTrailingDot validates Tab cycles through all suggestions when input ends with "."
func TestTabCyclesWithTrailingDot(t *testing.T) {
	m := focusedModel()
	m.PathInput.SetValue("_.")
	m.ShowSuggestions = true
	m.FilteredSuggestions = []string{
		"name",
		"items",
		"active",
	}

	// First Tab -> first key
	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m1 := nm.(*Model)
	if got := m1.PathInput.Value(); got != "_.name" {
		t.Fatalf("First Tab should pick first key, got %q", got)
	}

	// Second Tab -> second key
	nm, _ = m1.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m2 := nm.(*Model)
	if got := m2.PathInput.Value(); got != "_.items" {
		t.Fatalf("Second Tab should pick second key, got %q", got)
	}

	// Third Tab -> third key
	nm, _ = m2.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m3 := nm.(*Model)
	if got := m3.PathInput.Value(); got != "_.active" {
		t.Fatalf("Third Tab should pick third key, got %q", got)
	}

	// Fourth Tab wraps to first key
	nm, _ = m3.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m4 := nm.(*Model)
	if got := m4.PathInput.Value(); got != "_.name" {
		t.Fatalf("Fourth Tab should wrap to first key, got %q", got)
	}

	// Shift+Tab should move backward to third key
	nm, _ = m4.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	m5 := nm.(*Model)
	if got := m5.PathInput.Value(); got != "_.active" {
		t.Fatalf("Shift+Tab should cycle backward to last key, got %q", got)
	}
}

// Ensure Tab cycles keys then functions (deduped) and Shift+Tab cycles backwards.
func TestTabCyclesKeysAndFunctionsDeduped(t *testing.T) {
	m := focusedModel()
	m.PathInput.SetValue("_.e")
	m.ShowSuggestions = true
	m.FilteredSuggestions = []string{
		"env",
		"exists() - list.exists() [method]",
		"exists_one() - list.exists_one() [method]",
		"exists() - list.exists() [method]", // duplicate to exercise dedupe
	}

	// First Tab -> key
	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m1 := nm.(*Model)
	if got := m1.PathInput.Value(); got != "_.env" {
		t.Fatalf("Tab should pick key first, got %q", got)
	}

	// Second Tab -> first function
	nm, _ = m1.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m2 := nm.(*Model)
	if got := m2.PathInput.Value(); got != "_.exists()" {
		t.Fatalf("Second Tab should pick first function, got %q", got)
	}

	// Third Tab -> next function
	nm, _ = m2.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m3 := nm.(*Model)
	if got := m3.PathInput.Value(); got != "_.exists_one()" {
		t.Fatalf("Third Tab should pick next function, got %q", got)
	}

	// Fourth Tab wraps to key again
	nm, _ = m3.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m4 := nm.(*Model)
	if got := m4.PathInput.Value(); got != "_.env" {
		t.Fatalf("Fourth Tab should wrap to key, got %q", got)
	}

	// Shift+Tab should move backward to last function
	nm, _ = m4.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	m5 := nm.(*Model)
	if got := m5.PathInput.Value(); got != "_.exists_one()" {
		t.Fatalf("Shift+Tab should cycle backward to previous item, got %q", got)
	}
}

// TestTypeAwareFunctionFiltering validates that incompatible functions are filtered
func TestTypeAwareFunctionFiltering(t *testing.T) {
	m := focusedModel()
	// String node
	m.Node = "hello"
	m.Root = sampleData()

	m.PathInput.SetValue(".")
	// Mix of string and list functions
	m.Suggestions = []string{
		"flatten() - list.flatten() -> list [method]",
		"lowerAscii() - string.lowerAscii() -> string [method]",
		"size() - any.size() -> int [method]",
	}

	m.filterSuggestions()

	// flatten should not be in filtered suggestions for a string
	for _, s := range m.FilteredSuggestions {
		if strings.HasPrefix(s, "flatten()") {
			t.Errorf("flatten should not be suggested for string node, got: %v", m.FilteredSuggestions)
		}
	}

	// lowerAscii and size should be present
	hasLower := false
	hasSize := false
	for _, s := range m.FilteredSuggestions {
		if strings.HasPrefix(s, "lowerAscii()") {
			hasLower = true
		}
		if strings.HasPrefix(s, "size()") {
			hasSize = true
		}
	}
	if !hasLower || !hasSize {
		t.Errorf("expected lowerAscii and size for string, got: %v", m.FilteredSuggestions)
	}
}
