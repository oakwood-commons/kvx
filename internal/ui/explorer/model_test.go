package explorer

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func sampleRoot() map[string]interface{} {
	return map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"name": "alpha", "id": 1},
			map[string]interface{}{"name": "beta", "id": 2},
		},
		"meta":  map[string]interface{}{"version": "v1"},
		"plain": "scalar",
	}
}

func TestExplorer_NewModelAndView(t *testing.T) {
	m := NewModel(sampleRoot())
	v := m.View()
	if !strings.Contains(v, "Path:") {
		t.Fatalf("expected path breadcrumb in view, got: %q", v)
	}
}

func TestExplorer_FilterTypingAndClear(t *testing.T) {
	m := NewModel(sampleRoot())

	// Type 'm' to filter to meta
	m.Update(tea.KeyPressMsg{Text: "m"})
	if m.table.Filter() == "" {
		t.Fatalf("expected filter to be set after typing")
	}

	// Esc clears filter
	m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if m.table.Filter() != "" {
		t.Fatalf("expected filter to be cleared after Esc")
	}
}

func TestExplorer_EnterNavigateAndBack(t *testing.T) {
	m := NewModel(sampleRoot())

	// Ensure cursor at 0 and enter navigates
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.Path() == "" {
		t.Fatalf("expected path to change after enter; got empty")
	}

	// Navigate back using left
	m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if m.Path() != "" {
		t.Fatalf("expected path to return to root after left, got %q", m.Path())
	}
}

func TestExplorer_SetSize_NoColor_Focus(t *testing.T) {
	m := NewModel(sampleRoot())
	m.SetSize(80, 20)
	m.SetNoColor(true)
	m.Focus()
	if !m.Focused() {
		t.Fatalf("expected explorer focused after Focus()")
	}
	m.Blur()
	if m.Focused() {
		t.Fatalf("expected explorer unfocused after Blur()")
	}
}
