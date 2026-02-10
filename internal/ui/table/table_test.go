package table

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Item represents a simple key/value used for testing generic table
type Item struct {
	Key   string
	Value string
}

func makeModel() *Model[Item] {
	cols := []Column{{Title: "KEY", Width: 10}, {Title: "VALUE", Width: 20}}
	toRow := func(v Item) Row { return Row{v.Key, v.Value} }
	keyFn := func(v Item) string { return v.Key }
	return NewModel[Item](cols, toRow, keyFn)
}

func TestTable_SetRowsAndFilter(t *testing.T) {
	m := makeModel()
	m.SetRows([]Item{{"apple", "red"}, {"banana", "yellow"}, {"apricot", "orange"}})

	if got := len(m.Rows()); got != 3 {
		t.Fatalf("expected 3 filtered rows initially, got %d", got)
	}

	m.SetFilter("ap")
	rows := m.Rows()
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows after filter prefix 'ap', got %d", len(rows))
	}
	if rows[0].Key != "apple" || rows[1].Key != "apricot" {
		t.Fatalf("unexpected filter order: %+v", rows)
	}

	m.ClearFilter()
	if len(m.Rows()) != 3 {
		t.Fatalf("expected 3 rows after clear filter, got %d", len(m.Rows()))
	}
}

func TestTable_CursorSelection(t *testing.T) {
	m := makeModel()
	m.SetRows([]Item{{"apple", "red"}, {"banana", "yellow"}})

	// Default cursor at 0
	sel := m.SelectedRow()
	if sel == nil || sel.Key != "apple" {
		t.Fatalf("expected first row selected, got %+v", sel)
	}

	m.SetCursor(1)
	sel = m.SelectedRow()
	if sel == nil || sel.Key != "banana" {
		t.Fatalf("expected second row selected, got %+v", sel)
	}

	// Move via Update to ensure bubbles path runs
	m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	// Cursor shouldn't exceed bounds when only 2 rows
	if m.Cursor() > 1 {
		t.Fatalf("cursor out of bounds: %d", m.Cursor())
	}
}

func TestTable_SizeHeightFocus(t *testing.T) {
	m := makeModel()
	m.SetRows([]Item{{"k", "v"}})

	m.SetSize(40, 8)
	if m.Height() <= 0 || m.Width() <= 0 {
		t.Fatalf("expected non-zero dimensions, got h=%d w=%d", m.Height(), m.Width())
	}

	m.SetHeight(12)
	if m.Height() <= 0 {
		t.Fatalf("expected non-zero height after SetHeight, got %d", m.Height())
	}

	if !m.Focused() { // default true
		t.Fatalf("expected model focused by default")
	}
	m.Blur()
	if m.Focused() {
		t.Fatalf("expected model to be unfocused after Blur")
	}
	m.Focus()
	if !m.Focused() {
		t.Fatalf("expected model to be focused after Focus")
	}
}

func TestTable_ColorScheme(t *testing.T) {
	m := makeModel()
	m.SetRows([]Item{{"k", "v"}})

	// Toggle no color
	m.SetNoColor(true)
	// Set custom colors and ensure no panic
	m.SetColors(lipgloss.Color("12"), lipgloss.Color("0"), lipgloss.Color("15"), lipgloss.Color("8"))

	// Render
	_ = m.View()
}

func TestTable_StringDebug(t *testing.T) {
	m := makeModel()
	m.SetRows([]Item{{"k", "v"}})
	s := m.String()
	if s == "" {
		t.Fatalf("expected non-empty debug string")
	}
}
