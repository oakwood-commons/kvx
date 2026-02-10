package explorer

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/oakwood-commons/kvx/internal/navigator"
	"github.com/oakwood-commons/kvx/internal/ui"
	"github.com/oakwood-commons/kvx/internal/ui/table"
)

// Model represents the main data explorer view with table navigation.
// It wraps the generic table component and adds data-specific navigation logic.
type Model struct {
	// Table component for displaying rows
	table *table.Model[table.Row]

	// Navigation state
	root        interface{} // Root data node
	currentNode interface{} // Current node being displayed
	path        string      // Current path (e.g., "items.0.name")

	// UI state
	width   int
	height  int
	focused bool

	// Configuration
	noColor       bool
	allowFilter   bool
	keyColWidth   int
	valueColWidth int
}

// navigator.Row represents a table row (key-value pair).
// This is already defined in navigator package as []string.

// NewModel creates a new explorer model.
func NewModel(root interface{}) *Model {
	// Create columns for the table
	columns := []table.Column{
		{Title: "KEY", Width: 30},
		{Title: "VALUE", Width: 60},
	}

	// toRow function: bubble table.Row is []string, so identity function
	toRow := func(row table.Row) table.Row { return row }

	// keyFunc: extract key from row (first column)
	keyFunc := func(row table.Row) string {
		if len(row) > 0 {
			return row[0]
		}
		return ""
	}

	// Create table with table.Row type
	tbl := table.NewModel(columns, toRow, keyFunc)

	m := &Model{
		table:         tbl,
		root:          root,
		currentNode:   root,
		path:          "",
		width:         80,
		height:        24,
		keyColWidth:   30,
		valueColWidth: 60,
		allowFilter:   true,
	}

	// Load initial rows
	m.loadRows()

	return m
}

// loadRows converts the current node to table rows.
func (m *Model) loadRows() {
	stringRows := navigator.NodeToRows(m.currentNode)
	rows := make([]table.Row, len(stringRows))
	for i, r := range stringRows {
		rows[i] = table.Row(r)
	}
	m.table.SetRows(rows)
}

// Init initializes the explorer model.
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the explorer model.
func (m *Model) Update(msg tea.Msg) (ui.ChildModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle navigation keys
		switch msg.String() {
		case "enter", "right":
			// Navigate into selected row
			if row := m.table.SelectedRow(); row != nil {
				key := (*row)[0]

				// Skip scalar value indicator
				if key == "(value)" {
					return m, nil
				}

				// Build new path
				newPath := m.path
				if newPath != "" {
					newPath += "."
				}
				newPath += key

				// Navigate to new node
				node, err := navigator.NodeAtPath(m.root, newPath)
				if err != nil {
					// Show error (TODO: add status bar)
					return m, nil
				}

				m.currentNode = node
				m.path = newPath
				m.loadRows()
				m.table.SetCursor(0)
			}
			return m, nil

		case "left", "backspace":
			// Navigate back to parent
			if m.path == "" {
				// Already at root
				return m, nil
			}

			// Remove last segment from path
			lastDot := strings.LastIndex(m.path, ".")
			var parentPath string
			if lastDot >= 0 {
				parentPath = m.path[:lastDot]
			}

			// Navigate to parent
			var parentNode interface{}
			if parentPath == "" {
				parentNode = m.root
			} else {
				var err error
				parentNode, err = navigator.NodeAtPath(m.root, parentPath)
				if err != nil {
					return m, nil
				}
			}

			m.currentNode = parentNode
			m.path = parentPath
			m.loadRows()
			m.table.SetCursor(0)
			return m, nil

		case "esc":
			// Clear filter if active, otherwise do nothing
			if m.table.Filter() != "" {
				m.table.ClearFilter()
				return m, nil
			}
			return m, nil
		}

		// Handle alphanumeric input for filtering
		if m.allowFilter && len(msg.String()) == 1 {
			r := []rune(msg.String())[0]
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
				// Type-ahead filtering
				currentFilter := m.table.Filter()
				m.table.SetFilter(currentFilter + msg.String())
				return m, nil
			}
		}
	default:
		_, cmd := m.table.Update(msg)
		return m, cmd
	}

	// Delegate to table for cursor movement, etc.
	_, cmd := m.table.Update(msg)
	return m, cmd
}

// View renders the explorer model.
func (m *Model) View() string {
	var b strings.Builder

	// Breadcrumb / path display
	pathDisplay := m.path
	if pathDisplay == "" {
		pathDisplay = "_"
	}
	pathStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Bold(true)
	if m.noColor {
		b.WriteString("Path: " + pathDisplay + "\n")
	} else {
		b.WriteString(pathStyle.Render("Path: "+pathDisplay) + "\n")
	}

	// Separator
	separator := strings.Repeat("─", m.width)
	if !m.noColor {
		separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		separator = separatorStyle.Render(separator)
	}
	b.WriteString(separator + "\n")

	// Table
	b.WriteString(m.table.View())

	// Filter indicator
	if filter := m.table.Filter(); filter != "" {
		filterStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")).
			Bold(true)
		filterText := fmt.Sprintf("Filter: %s", filter)
		if m.noColor {
			b.WriteString("\n" + filterText)
		} else {
			b.WriteString("\n" + filterStyle.Render(filterText))
		}
	}

	return b.String()
}

// SetSize implements ModelWithSize.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	// Adjust table size
	// Reserve 3 lines for path, separator, and potential filter indicator
	tableHeight := height - 3
	if tableHeight < 5 {
		tableHeight = 5
	}

	// Adjust column widths based on available width
	keyWidth := m.keyColWidth
	valueWidth := width - keyWidth - 3 // -3 for borders/padding
	if valueWidth < 20 {
		valueWidth = 20
	}

	m.table.SetSize(width, tableHeight)

	// Update column widths
	m.table.SetColumns([]table.Column{
		{Title: "KEY", Width: keyWidth},
		{Title: "VALUE", Width: valueWidth},
	})
}

// Focus implements ModelWithFocus.
func (m *Model) Focus() tea.Cmd {
	m.focused = true
	m.table.Focus()
	return nil
}

// Blur implements ModelWithFocus.
func (m *Model) Blur() {
	m.focused = false
	m.table.Blur()
}

// Focused implements ModelWithFocus.
func (m *Model) Focused() bool {
	return m.focused
}

// Title implements ModelWithTitle.
func (m *Model) Title() string {
	return "Explorer"
}

// ID implements ModelWithID.
func (m *Model) ID() string {
	return "explorer:" + m.path
}

// SetNoColor sets the color mode.
func (m *Model) SetNoColor(noColor bool) {
	m.noColor = noColor
	m.table.SetNoColor(noColor)
}

// SetAllowFilter enables/disables type-ahead filtering.
func (m *Model) SetAllowFilter(allow bool) {
	m.allowFilter = allow
}

// Path returns the current path.
func (m *Model) Path() string {
	return m.path
}

// CurrentNode returns the current node.
func (m *Model) CurrentNode() interface{} {
	return m.currentNode
}
