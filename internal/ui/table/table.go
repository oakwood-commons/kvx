package table

import (
	"fmt"
	"image/color"

	bubtable "charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Re-export common table types so callers can construct columns/rows without
// importing bubbles directly. This keeps the generic wrapper ergonomic.
type Column = bubtable.Column
type Row = bubtable.Row

// Model is a generic table component that can display any type of row data.
// It wraps the bubbles table component and provides additional functionality
// like filtering, custom styling, and type-safe row handling.
//
// Type parameter V represents the row data type (e.g., SearchResult, FileInfo).
type Model[V any] struct {
	table    bubtable.Model
	styles   bubtable.Styles
	rows     []V    // Original unfiltered rows
	filter   string // Current filter text
	filtered []V    // Filtered rows
	columns  []Column

	// Rendering functions
	toRow   func(V) Row    // Convert value to table row
	keyFunc func(V) string // Extract display key from value (for filtering)

	// Display settings
	width   int
	height  int
	focused bool
	noColor bool

	// Theme colors (optional)
	headerFG   color.Color
	headerBG   color.Color
	selectedFG color.Color
	selectedBG color.Color
}

// NewModel creates a new generic table model.
// Parameters:
//
//	columns: table column definitions
//	toRow: function to convert value V to table.Row
//	keyFunc: function to extract searchable key from value V
func NewModel[V any](
	columns []Column,
	toRow func(V) Row,
	keyFunc func(V) string,
) *Model[V] {
	t := bubtable.New(
		bubtable.WithColumns(columns),
		bubtable.WithFocused(true),
		bubtable.WithHeight(5),
	)

	// Apply default styles
	s := bubtable.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false).
		Bold(true).
		Align(lipgloss.Left).
		PaddingLeft(0).
		PaddingRight(0)
	s.Selected = s.Selected.
		PaddingLeft(0).
		PaddingRight(0)
	s.Cell = lipgloss.NewStyle().
		Align(lipgloss.Left).
		PaddingLeft(0).
		PaddingRight(0)
	t.SetStyles(s)

	return &Model[V]{
		table:    t,
		styles:   s,
		rows:     []V{},
		filtered: []V{},
		columns:  columns,
		toRow:    toRow,
		keyFunc:  keyFunc,
		width:    80,
		height:   10,
		focused:  true,
	}
}

// SetRows updates the table with new row data.
func (m *Model[V]) SetRows(rows []V) {
	m.rows = rows
	m.applyFilter()
}

// SetColumns updates the table columns and reapplies styles.
func (m *Model[V]) SetColumns(columns []Column) {
	m.columns = columns
	m.table.SetColumns(columns)
	m.applyColorScheme()
}

// Rows returns the current filtered rows.
func (m *Model[V]) Rows() []V {
	return m.filtered
}

// AllRows returns all unfiltered rows.
func (m *Model[V]) AllRows() []V {
	return m.rows
}

// SetFilter sets the filter text and reapplies filtering.
func (m *Model[V]) SetFilter(filter string) {
	m.filter = filter
	m.applyFilter()
}

// Filter returns the current filter text.
func (m *Model[V]) Filter() string {
	return m.filter
}

// ClearFilter removes the filter and shows all rows.
func (m *Model[V]) ClearFilter() {
	m.filter = ""
	m.applyFilter()
}

// applyFilter filters rows based on the current filter text.
func (m *Model[V]) applyFilter() {
	if m.filter == "" {
		m.filtered = m.rows
	} else {
		m.filtered = []V{}
		for _, row := range m.rows {
			key := m.keyFunc(row)
			// Simple prefix matching (can be extended to fuzzy matching)
			if len(key) >= len(m.filter) && key[:len(m.filter)] == m.filter {
				m.filtered = append(m.filtered, row)
			}
		}
	}

	// Update table rows
	tableRows := make([]Row, len(m.filtered))
	for i, row := range m.filtered {
		tableRows[i] = m.toRow(row)
	}
	m.table.SetRows(tableRows)

	// Reset cursor if out of bounds
	if m.Cursor() >= len(m.filtered) && len(m.filtered) > 0 {
		m.SetCursor(0)
	}
}

// Cursor returns the current cursor position.
func (m *Model[V]) Cursor() int {
	return m.table.Cursor()
}

// SetCursor sets the cursor position.
func (m *Model[V]) SetCursor(pos int) {
	m.table.SetCursor(pos)
}

// SelectedRow returns the currently selected row value, or nil if no rows.
func (m *Model[V]) SelectedRow() *V {
	if len(m.filtered) == 0 {
		return nil
	}
	cursor := m.Cursor()
	if cursor < 0 || cursor >= len(m.filtered) {
		return nil
	}
	return &m.filtered[cursor]
}

// SetSize sets the table dimensions.
func (m *Model[V]) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.table.SetHeight(height)
	// Recalculate column widths if needed
	// (This is where you'd implement dynamic column width calculation)
}

// SetHeight updates only the table height, preserving current width.
func (m *Model[V]) SetHeight(height int) {
	m.SetSize(m.width, height)
}

// Focus sets the table focus state.
func (m *Model[V]) Focus() {
	m.focused = true
	m.table.Focus()
}

// Blur removes focus from the table.
func (m *Model[V]) Blur() {
	m.focused = false
	m.table.Blur()
}

// Focused returns true if the table has focus.
func (m *Model[V]) Focused() bool {
	return m.focused
}

// SetNoColor enables/disables color output.
func (m *Model[V]) SetNoColor(noColor bool) {
	m.noColor = noColor
	m.applyColorScheme()
}

// SetColors sets custom theme colors.
func (m *Model[V]) SetColors(headerFG, headerBG, selectedFG, selectedBG color.Color) {
	m.headerFG = headerFG
	m.headerBG = headerBG
	m.selectedFG = selectedFG
	m.selectedBG = selectedBG
	m.applyColorScheme()
}

// applyColorScheme applies the current color scheme to table styles.
func (m *Model[V]) applyColorScheme() {
	s := m.styles

	if m.noColor {
		s.Header = s.Header.UnsetForeground().UnsetBackground()
		s.Selected = s.Selected.UnsetForeground().UnsetBackground().Reverse(true)
		s.Cell = s.Cell.UnsetForeground().UnsetBackground()
	} else {
		if m.headerFG != nil {
			s.Header = s.Header.Foreground(m.headerFG)
		}
		if m.headerBG != nil {
			s.Header = s.Header.Background(m.headerBG)
		}
		if m.selectedFG != nil {
			s.Selected = s.Selected.Foreground(m.selectedFG)
		}
		if m.selectedBG != nil {
			s.Selected = s.Selected.Background(m.selectedBG)
		}
	}

	m.table.SetStyles(s)
	m.styles = s
}

// Update handles messages and updates the table state.
func (m *Model[V]) Update(msg tea.Msg) (*Model[V], tea.Cmd) {
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View renders the table to a string.
func (m *Model[V]) View() string {
	return m.table.View()
}

// Height returns the rendered height of the table (including header).
func (m *Model[V]) Height() int {
	return lipgloss.Height(m.View())
}

// Width returns the rendered width of the table.
func (m *Model[V]) Width() int {
	return lipgloss.Width(m.View())
}

// String returns a string representation for debugging.
func (m *Model[V]) String() string {
	return fmt.Sprintf("Table[rows=%d, filtered=%d, cursor=%d, filter=%q]",
		len(m.rows), len(m.filtered), m.Cursor(), m.filter)
}
