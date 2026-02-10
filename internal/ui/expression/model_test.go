//go:build ignore
// +build ignore

package expression

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func expRoot() map[string]interface{} {
	return map[string]interface{}{
		"user": map[string]interface{}{
			"name":    "alice",
			"age":     30,
			"bad-key": "needs bracket",
		},
		"items": []interface{}{"a", "b", "c"},
	}
}

func defaultSuggestions() []Suggestion {
	return []Suggestion{
		{Name: "string.toUpper", Description: "String function: toUpper", IsFunction: true},
		{Name: "size", Description: "list size", IsFunction: true},
		{Name: "user", Description: "Key", IsFunction: false},
	}
}

func TestExpression_SubmitCancelAndResult(t *testing.T) {
	m := NewModel(expRoot(), "", expRoot())
	m.SetSuggestions(defaultSuggestions())

	// Type an expression and submit
	for _, r := range ".user.name" { // initial value is "_", so just append suffix
		m.Update(tea.KeyPressMsg{Text: string(r)})
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !m.Submitted() || m.Result() != "_.user.name" {
		t.Fatalf("expected submitted with result; submitted=%v result=%q", m.Submitted(), m.Result())
	}

	// Reset and cancel
	m.Reset()
	m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if !m.Cancelled() {
		t.Fatalf("expected cancelled after Esc")
	}
}

func TestExpression_SuggestionsOnDotAndNavigation(t *testing.T) {
	m := NewModel(expRoot(), "_", expRoot())
	m.SetSuggestions(defaultSuggestions())

	// Type trailing dot to trigger suggestions
	for _, r := range "_." {
		m.Update(tea.KeyPressMsg{Text: string(r)})
	}
	v := m.View()
	if !strings.Contains(v, "Suggestions:") {
		t.Fatalf("expected suggestions section in view, got: %q", v)
	}

	// Navigate suggestions with up/down
	m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
}

func TestExpression_TabCyclesKeysAndRightCompletesFunction(t *testing.T) {
	m := NewModel(expRoot(), "_", expRoot())
	m.SetSuggestions(defaultSuggestions())

	// Show suggestions for keys/functions
	for _, r := range "_.u" { // start typing 'u' to filter to 'user'
		m.Update(tea.KeyPressMsg{Text: string(r)})
	}

	// Tab cycles keys
	m.Update(tea.KeyPressMsg{Code: tea.KeyTab})

	// Move cursor to end and complete a function using right
	m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	_ = m.View()
}

func TestExpression_SizeFocusNoColor(t *testing.T) {
	m := NewModel(expRoot(), "_", expRoot())
	m.SetSize(80, 20)
	m.Focus()
	if !m.Focused() {
		t.Fatalf("expected focused after Focus()")
	}
	m.Blur()
	if m.Focused() {
		t.Fatalf("expected unfocused after Blur()")
	}
	m.SetNoColor(true)
	_ = m.View()
}

func TestExpression_HelperFunctions(t *testing.T) {
	// needsBracketNotation
	if !needsBracketNotation("bad-key") || needsBracketNotation("good_key") {
		t.Fatalf("needsBracketNotation failed")
	}

	// getKeysFromNode
	keys := getKeysFromNode(map[string]interface{}{"bad-key": 1, "ok": 2})
	// Expect bracket for bad-key and plain for ok
	hasBracket := false
	hasOk := false
	for _, k := range keys {
		if strings.HasPrefix(k, "[") {
			hasBracket = true
		}
		if k == "ok" {
			hasOk = true
		}
	}
	if !hasBracket || !hasOk {
		t.Fatalf("getKeysFromNode did not include expected forms: %v", keys)
	}

	// isSuggestionCompatibleWithNode
	s := Suggestion{Name: "string.toUpper", Description: "String function", IsFunction: true}
	if !isSuggestionCompatibleWithNode(s, "abc") {
		t.Fatalf("string function should be compatible with string node")
	}
	if !isSuggestionCompatibleWithNode(s, nil) { // nil treated as any
		t.Fatalf("function should be compatible with nil node")
	}
}
