package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/oakwood-commons/kvx/internal/formatter"
	"github.com/oakwood-commons/kvx/internal/navigator"
	"github.com/oakwood-commons/kvx/internal/ui"
	"github.com/oakwood-commons/kvx/pkg/core"
	"gopkg.in/yaml.v3"
)

// ArrayStyle constants control how array indices are displayed.
const (
	ArrayStyleIndex    = "index"    // [0], [1], [2]
	ArrayStyleNumbered = "numbered" // 1, 2, 3 (1-based, default)
	ArrayStyleBullet   = "bullet"   // •
	ArrayStyleNone     = "none"     // no index column
)

// ColumnarMode constants control when arrays render as multi-column tables.
const (
	ColumnarModeAuto   = "auto"   // detect homogeneous arrays (default)
	ColumnarModeAlways = "always" // always attempt columnar
	ColumnarModeNever  = "never"  // always use KEY/VALUE format
)

// OutputFormat controls the top-level rendering style.
type OutputFormat string

const (
	// FormatTable renders data as a columnar table (auto-detects arrays of objects).
	FormatTable OutputFormat = "table"
	// FormatList renders data as a vertical property list (one section per object).
	FormatList OutputFormat = "list"
	// FormatYAML renders data as YAML.
	FormatYAML OutputFormat = "yaml"
	// FormatJSON renders data as indented JSON.
	FormatJSON OutputFormat = "json"
	// FormatTree renders data as an ASCII tree structure.
	FormatTree OutputFormat = "tree"
	// FormatMermaid renders data as a Mermaid flowchart diagram.
	FormatMermaid OutputFormat = "mermaid"
)

// PanelOptions configures data panel rendering.
//
// Deprecated: Use TableOptions instead.
type PanelOptions struct {
	NoColor       bool
	KeyColWidth   int
	ValueColWidth int
	Width         int // Total width. If 0, auto-detect terminal width.
}

// RenderDataPanel renders a two-column key/value table for the given node.
//
// Deprecated: Use RenderTable with Bordered: false instead.
func RenderDataPanel(node any, opts PanelOptions) string {
	return RenderTable(node, TableOptions{
		NoColor:       opts.NoColor,
		KeyColWidth:   opts.KeyColWidth,
		ValueColWidth: opts.ValueColWidth,
		Width:         opts.Width,
		Bordered:      false,
	})
}

// BorderedTableOptions configures bordered table rendering.
//
// Deprecated: Use TableOptions with Bordered: true instead.
type BorderedTableOptions struct {
	AppName       string
	Path          string
	Width         int
	NoColor       bool
	KeyColWidth   int
	ValueColWidth int
}

// RenderBorderedTable renders a bordered table view like the kvx CLI.
//
// Deprecated: Use RenderTable with Bordered: true instead.
func RenderBorderedTable(node any, opts BorderedTableOptions) string {
	return RenderTable(node, TableOptions{
		AppName:       opts.AppName,
		Path:          opts.Path,
		Width:         opts.Width,
		NoColor:       opts.NoColor,
		KeyColWidth:   opts.KeyColWidth,
		ValueColWidth: opts.ValueColWidth,
		Bordered:      true,
	})
}

// TableOptions configures table rendering.
type TableOptions struct {
	// Bordered adds box borders around the table (top/bottom/sides).
	// When true, renders like the kvx CLI with title and footer.
	// When false, renders just the KEY/VALUE table content.
	Bordered bool

	// AppName is shown in the top border when Bordered is true.
	// Defaults to "kvx" if empty.
	AppName string

	// Path is shown in the bottom border when Bordered is true.
	// Defaults to "_" if empty.
	Path string

	// Width is the total table width. If 0, auto-detect terminal width.
	Width int

	// NoColor disables color output.
	NoColor bool

	// KeyColWidth sets the key column width. If 0, auto-calculate.
	KeyColWidth int

	// ValueColWidth sets the value column width. If 0, auto-calculate.
	ValueColWidth int

	// ArrayStyle controls how array indices are displayed:
	//   "index"    - [0], [1], [2] (default for non-object arrays)
	//   "numbered" - 1, 2, 3 (1-based numbering, default overall)
	//   "bullet"   - •
	//   "none"     - no index/key column for arrays
	ArrayStyle string

	// ColumnarMode controls when arrays of objects render as multi-column tables:
	//   "auto"   - detect homogeneous arrays and render as columns (default)
	//   "always" - always attempt columnar rendering for arrays
	//   "never"  - always use KEY/VALUE format
	ColumnarMode string

	// ColumnOrder specifies the preferred order of columns for columnar rendering.
	// Fields not in this list are appended in alphabetical order.
	ColumnOrder []string

	// HiddenColumns specifies field names to omit from columnar rendering.
	HiddenColumns []string

	// ColumnHints provides per-column display hints (max width, priority, alignment, etc.).
	// Keys are the original field names in the data. Use [ParseSchema] to derive hints
	// from a JSON Schema, or construct directly for programmatic control.
	ColumnHints map[string]ColumnHint
}

// RenderTable renders a two-column key/value table for the given node.
// Set Bordered: true for a boxed table with title and footer (like kvx CLI output).
// Set Bordered: false for just the table content without borders.
//
// For arrays of objects, if ColumnarMode is "auto" (default) or "always", the table
// will render with field names as column headers and values as rows.
//
// If Width is 0, terminal size is auto-detected.
// When Bordered is true, the table shrinks to fit its content width rather
// than always expanding to the full terminal width.
func RenderTable(node any, opts TableOptions) string {
	th := ui.CurrentTheme()
	formatter.SetTableTheme(formatter.TableColors{
		HeaderFG:       th.HeaderFG,
		HeaderBG:       th.HeaderBG,
		KeyColor:       th.KeyColor,
		ValueColor:     th.ValueColor,
		SeparatorColor: th.SeparatorColor,
	})

	// Auto-detect terminal width if not specified
	termWidth := opts.Width
	if termWidth <= 0 {
		termWidth, _ = DetectTerminalSize()
	}

	// Check for columnar rendering
	columnarMode := opts.ColumnarMode
	if columnarMode == "" {
		columnarMode = ColumnarModeAuto
	}

	if shouldUseColumnarRendering(node, columnarMode) {
		return renderColumnarTable(node, opts, termWidth)
	}

	// Standard KEY/VALUE table rendering
	// Create a minimal model for proper column width calculation

	// When bordered, calculate the natural content width and shrink to fit
	// so the table doesn't needlessly expand to the full terminal width.
	tableWidth := termWidth
	fitContent := false
	if opts.Bordered {
		rowOpts := navigator.DefaultRowOptions()
		if opts.ArrayStyle != "" {
			rowOpts.ArrayStyle = opts.ArrayStyle
		}
		rows := navigator.NodeToRowsWithOptions(node, rowOpts)
		naturalContentWidth := formatter.CalculateNaturalTableWidth(rows)
		naturalTableWidth := naturalContentWidth + 2 // +2 for side borders
		if naturalTableWidth < termWidth {
			tableWidth = naturalTableWidth
			fitContent = true
		}

		// Ensure minimum width for title/footer visibility
		appName := opts.AppName
		if appName == "" {
			appName = "kvx"
		}
		path := opts.Path
		if path == "" {
			path = "_"
		}
		typeLabel := ""
		switch node.(type) {
		case map[string]any:
			typeLabel = "map: "
		case []any:
			typeLabel = "list: "
		}
		visibleRows := len(rows)
		if visibleRows == 0 {
			visibleRows = 1
		}
		pathWithSpace := fmt.Sprintf(" %s ", path)
		countWithSpace := fmt.Sprintf(" %s1/%d ", typeLabel, visibleRows)
		minFooterWidth := lipgloss.Width(pathWithSpace) + lipgloss.Width(countWithSpace) + 3
		minTableWidth := lipgloss.Width(appName) + 6
		if minFooterWidth > minTableWidth {
			minTableWidth = minFooterWidth
		}
		if tableWidth < minTableWidth {
			tableWidth = minTableWidth
		}
	}

	m := ui.InitialModel(node)
	m.NoColor = opts.NoColor
	m.Root = node
	m.Node = node
	m.WinWidth = tableWidth
	m.Layout.SetDimensions(tableWidth, 24)

	// Calculate column widths using the layout manager
	keyColWidth := opts.KeyColWidth
	if keyColWidth <= 0 {
		// Shrink key column when keys are short
		maxKey := 0
		if n, ok := node.(map[string]any); ok {
			for k := range n {
				if w := lipgloss.Width(k); w > maxKey {
					maxKey = w
				}
			}
		}
		if maxKey > 0 {
			minKey := 8
			maxKeyWidth := 30 // default max
			keyColWidth = maxKey + 2
			if keyColWidth < minKey {
				keyColWidth = minKey
			}
			if keyColWidth > maxKeyWidth {
				keyColWidth = maxKeyWidth
			}
		}
	}
	m.KeyColWidth = keyColWidth
	m.ConfiguredKeyColWidth = keyColWidth
	m.ConfiguredValueColWidth = opts.ValueColWidth
	keyW, valueW := m.Layout.CalculateColumnWidths(m.ConfiguredKeyColWidth, m.ConfiguredValueColWidth, m.AutoKeyColumnWidth)
	if valueW > 1 {
		valueW--
	}

	// Render the table content
	var tableView string
	if fitContent {
		rowOpts := navigator.DefaultRowOptions()
		if opts.ArrayStyle != "" {
			rowOpts.ArrayStyle = opts.ArrayStyle
		}
		rows := navigator.NodeToRowsWithOptions(node, rowOpts)
		tableView = formatter.RenderTableFitContent(rows, opts.NoColor, tableWidth-2)
	} else {
		engine := &core.Engine{}
		tableView = engine.RenderTable(node, opts.NoColor, keyW, valueW)
	}

	// If not bordered, return just the table content
	if !opts.Bordered {
		return tableView
	}

	// Add borders for bordered mode
	appName := opts.AppName
	if appName == "" {
		appName = "kvx"
	}
	path := opts.Path
	if path == "" {
		path = "_"
	}

	// Create top border - centered "╭─ appName ─────────────────────╮"
	borderChar := "─"
	availableWidth := tableWidth - 4 // -4 for "╭─" and "─╮"
	titleText := appName
	leftDashes := (availableWidth - len(titleText)) / 2
	rightDashes := availableWidth - len(titleText) - leftDashes
	topBorderLine := fmt.Sprintf("╭%s %s %s╮",
		strings.Repeat(borderChar, leftDashes),
		titleText,
		strings.Repeat(borderChar, rightDashes))

	// Apply theme colors to top border
	if !opts.NoColor && th.SeparatorColor != nil {
		borderStyle := lipgloss.NewStyle().Foreground(th.SeparatorColor)
		topBorderLine = borderStyle.Render(topBorderLine)
	}

	// Calculate row count for footer
	visibleRows := len(navigator.NodeToRows(node))
	if visibleRows == 0 {
		visibleRows = 1
	}

	// Determine node type label
	typeLabel := ""
	switch node.(type) {
	case map[string]any:
		typeLabel = "map"
	case []any:
		typeLabel = "list"
	}
	if typeLabel != "" {
		typeLabel += ": "
	}

	// Create footer: "╰ path ───────────────────── type: 1/N ╯"
	pathWithSpace := fmt.Sprintf(" %s ", path)
	countWithSpace := fmt.Sprintf(" %s1/%d ", typeLabel, visibleRows)
	// Recalculate dashes using tableWidth (which may be shrunk to fit content)
	dashes := tableWidth - lipgloss.Width(pathWithSpace) - lipgloss.Width(countWithSpace) - 2
	if dashes < 0 {
		dashes = 0
	}

	// Build bottom border with theme colors - foreground only, no background
	// to match CLI output styling
	var bottomBorderLine string
	if !opts.NoColor {
		leftText := pathWithSpace
		rightText := countWithSpace
		if th.StatusColor != nil {
			leftText = lipgloss.NewStyle().Foreground(th.StatusColor).Render(leftText)
		}
		if th.StatusSuccess != nil {
			rightText = lipgloss.NewStyle().Foreground(th.StatusSuccess).Render(rightText)
		}
		leftCorner := "╰"
		rightCorner := "╯"
		dashedLine := strings.Repeat(borderChar, dashes)
		if th.SeparatorColor != nil {
			borderStyle := lipgloss.NewStyle().Foreground(th.SeparatorColor)
			leftCorner = borderStyle.Render(leftCorner)
			rightCorner = borderStyle.Render(rightCorner)
			dashedLine = borderStyle.Render(dashedLine)
		}
		bottomBorderLine = leftCorner + leftText + dashedLine + rightText + rightCorner
	} else {
		bottomBorderLine = fmt.Sprintf("╰%s%s%s╯", pathWithSpace, strings.Repeat(borderChar, dashes), countWithSpace)
	}

	// Pad table lines to tableWidth (which may be shrunk to fit content)
	tableLines := strings.Split(tableView, "\n")
	paddedLines := make([]string, 0, len(tableLines))
	for i, line := range tableLines {
		// Skip trailing empty line
		if strings.TrimSpace(line) == "" && i == len(tableLines)-1 {
			continue
		}
		lineWidth := lipgloss.Width(line)
		padding := tableWidth - lineWidth - 2 // -2 for borders
		if padding < 0 {
			padding = 0
		}
		paddedLine := "│" + line + strings.Repeat(" ", padding) + "│"
		if !opts.NoColor && th.SeparatorColor != nil {
			borderStyle := lipgloss.NewStyle().Foreground(th.SeparatorColor)
			paddedLine = borderStyle.Render("│") + line + strings.Repeat(" ", padding) + borderStyle.Render("│")
		}
		paddedLines = append(paddedLines, paddedLine)
	}

	return topBorderLine + "\n" + strings.Join(paddedLines, "\n") + "\n" + bottomBorderLine + "\n"
}

// shouldUseColumnarRendering determines if columnar rendering should be used.
func shouldUseColumnarRendering(node any, mode string) bool {
	switch mode {
	case ColumnarModeNever:
		return false
	case ColumnarModeAlways:
		switch node.(type) {
		case []interface{}:
			return true
		default:
			return false
		}
	default: // ColumnarModeAuto
		isHomogeneous, _ := navigator.IsHomogeneousArray(node)
		return isHomogeneous
	}
}

// renderColumnarTable renders a homogeneous array as a multi-column table.
func renderColumnarTable(node any, opts TableOptions, termWidth int) string {
	columns, rows := navigator.ExtractColumnarData(node, opts.ColumnOrder)
	if columns == nil {
		// Fall back to standard rendering
		return renderStandardTable(node, opts, termWidth)
	}

	th := ui.CurrentTheme()

	// Determine row number style
	rowNumStyle := opts.ArrayStyle
	if rowNumStyle == "" {
		rowNumStyle = ArrayStyleNumbered
	}

	// When bordered, shrink to fit content width if it's narrower than the terminal.
	tableWidth := termWidth
	if opts.Bordered {
		showRowNum := rowNumStyle != ArrayStyleNone
		naturalContentWidth := formatter.CalculateNaturalColumnarWidth(columns, rows, showRowNum, len(rows))
		naturalTableWidth := naturalContentWidth + 2 // +2 for side borders
		if naturalTableWidth < termWidth {
			tableWidth = naturalTableWidth
		}

		// Ensure minimum width for title/footer visibility
		appName := opts.AppName
		if appName == "" {
			appName = "kvx"
		}
		path := opts.Path
		if path == "" {
			path = "_"
		}
		typeLabel := "list: "
		pathWithSpace := fmt.Sprintf(" %s ", path)
		countWithSpace := fmt.Sprintf(" %s1/%d ", typeLabel, len(rows))
		minFooterWidth := lipgloss.Width(pathWithSpace) + lipgloss.Width(countWithSpace) + 3
		minTableWidth := lipgloss.Width(appName) + 6
		if minFooterWidth > minTableWidth {
			minTableWidth = minFooterWidth
		}
		if tableWidth < minTableWidth {
			tableWidth = minTableWidth
		}
	}

	// Calculate content width (accounting for borders if needed)
	contentWidth := tableWidth
	if opts.Bordered {
		contentWidth = tableWidth - 2
	}

	// Merge hidden columns from hints
	hiddenCols := opts.HiddenColumns
	for name, hint := range opts.ColumnHints {
		if hint.Hidden {
			hiddenCols = append(hiddenCols, name)
		}
	}

	// Apply display name overrides to column headers
	displayColumns := make([]string, len(columns))
	copy(displayColumns, columns)
	if len(opts.ColumnHints) > 0 {
		for i, col := range columns {
			if hint, ok := opts.ColumnHints[col]; ok && hint.DisplayName != "" {
				displayColumns[i] = hint.DisplayName
			}
		}
	}

	// Build formatter-level column hints keyed by original column name
	var fmtHints map[string]formatter.ColumnHint
	if len(opts.ColumnHints) > 0 {
		fmtHints = make(map[string]formatter.ColumnHint, len(opts.ColumnHints))
		for name, h := range opts.ColumnHints {
			fmtHints[name] = formatter.ColumnHint{
				MaxWidth: h.MaxWidth,
				Priority: h.Priority,
				Align:    h.Align,
			}
		}
	}

	// Render columnar content
	tableView := formatter.RenderColumnarTable(displayColumns, rows, formatter.ColumnarOptions{
		NoColor:        opts.NoColor,
		TotalWidth:     contentWidth,
		RowNumberStyle: rowNumStyle,
		ColumnOrder:    opts.ColumnOrder,
		HiddenColumns:  hiddenCols,
		ColumnHints:    fmtHints,
		OriginalNames:  columns,
	})

	if !opts.Bordered {
		return tableView
	}

	// Add borders
	appName := opts.AppName
	if appName == "" {
		appName = "kvx"
	}
	path := opts.Path
	if path == "" {
		path = "_"
	}

	borderChar := "─"
	availableWidth := tableWidth - 4
	titleText := appName
	leftDashes := (availableWidth - len(titleText)) / 2
	rightDashes := availableWidth - len(titleText) - leftDashes
	topBorderLine := fmt.Sprintf("╭%s %s %s╮",
		strings.Repeat(borderChar, leftDashes),
		titleText,
		strings.Repeat(borderChar, rightDashes))

	if !opts.NoColor && th.SeparatorColor != nil {
		borderStyle := lipgloss.NewStyle().Foreground(th.SeparatorColor)
		topBorderLine = borderStyle.Render(topBorderLine)
	}

	// Footer
	visibleRows := len(rows)
	typeLabel := "list: "
	pathWithSpace := fmt.Sprintf(" %s ", path)
	countWithSpace := fmt.Sprintf(" %s1/%d ", typeLabel, visibleRows)
	dashes := tableWidth - lipgloss.Width(pathWithSpace) - lipgloss.Width(countWithSpace) - 2
	if dashes < 0 {
		dashes = 0
	}

	var bottomBorderLine string
	if !opts.NoColor {
		leftText := pathWithSpace
		rightText := countWithSpace
		if th.StatusColor != nil {
			leftText = lipgloss.NewStyle().Foreground(th.StatusColor).Render(leftText)
		}
		if th.StatusSuccess != nil {
			rightText = lipgloss.NewStyle().Foreground(th.StatusSuccess).Render(rightText)
		}
		leftCorner := "╰"
		rightCorner := "╯"
		dashedLine := strings.Repeat(borderChar, dashes)
		if th.SeparatorColor != nil {
			borderStyle := lipgloss.NewStyle().Foreground(th.SeparatorColor)
			leftCorner = borderStyle.Render(leftCorner)
			rightCorner = borderStyle.Render(rightCorner)
			dashedLine = borderStyle.Render(dashedLine)
		}
		bottomBorderLine = leftCorner + leftText + dashedLine + rightText + rightCorner
	} else {
		bottomBorderLine = fmt.Sprintf("╰%s%s%s╯", pathWithSpace, strings.Repeat(borderChar, dashes), countWithSpace)
	}

	// Add side borders to table lines
	tableLines := strings.Split(tableView, "\n")
	paddedLines := make([]string, 0, len(tableLines))
	for i, line := range tableLines {
		if strings.TrimSpace(line) == "" && i == len(tableLines)-1 {
			continue
		}
		lineWidth := lipgloss.Width(line)
		padding := tableWidth - lineWidth - 2
		if padding < 0 {
			padding = 0
		}
		paddedLine := "│" + line + strings.Repeat(" ", padding) + "│"
		if !opts.NoColor && th.SeparatorColor != nil {
			borderStyle := lipgloss.NewStyle().Foreground(th.SeparatorColor)
			paddedLine = borderStyle.Render("│") + line + strings.Repeat(" ", padding) + borderStyle.Render("│")
		}
		paddedLines = append(paddedLines, paddedLine)
	}

	return topBorderLine + "\n" + strings.Join(paddedLines, "\n") + "\n" + bottomBorderLine + "\n"
}

// renderStandardTable renders a standard KEY/VALUE table.
func renderStandardTable(node any, opts TableOptions, termWidth int) string {
	m := ui.InitialModel(node)
	m.NoColor = opts.NoColor
	m.Root = node
	m.Node = node
	m.WinWidth = termWidth
	m.Layout.SetDimensions(termWidth, 24)

	keyColWidth := opts.KeyColWidth
	if keyColWidth <= 0 {
		maxKey := 0
		if n, ok := node.(map[string]any); ok {
			for k := range n {
				if w := lipgloss.Width(k); w > maxKey {
					maxKey = w
				}
			}
		}
		if maxKey > 0 {
			minKey := 8
			maxKeyWidth := 30
			keyColWidth = maxKey + 2
			if keyColWidth < minKey {
				keyColWidth = minKey
			}
			if keyColWidth > maxKeyWidth {
				keyColWidth = maxKeyWidth
			}
		}
	}
	m.KeyColWidth = keyColWidth
	m.ConfiguredKeyColWidth = keyColWidth
	m.ConfiguredValueColWidth = opts.ValueColWidth
	keyW, valueW := m.Layout.CalculateColumnWidths(m.ConfiguredKeyColWidth, m.ConfiguredValueColWidth, m.AutoKeyColumnWidth)
	if valueW > 1 {
		valueW--
	}

	engine := &core.Engine{}
	return engine.RenderTable(node, opts.NoColor, keyW, valueW)
}

// RenderList renders data in a vertical list format.
// Arrays of objects display each element with an index header and indented properties.
// Maps display as key/value pairs. Scalars display as "value: <v>".
//
// This is the counterpart to RenderTable: use RenderTable for columnar output
// and RenderList for vertical per-object output.
//
//	fmt.Print(tui.RenderList(data, false))
func RenderList(node any, noColor bool) string {
	return formatter.FormatAsList(node, formatter.ListOptions{NoColor: noColor})
}

// ListOptions controls list output formatting.
type ListOptions = formatter.ListOptions

// TreeOptions controls ASCII tree output formatting.
type TreeOptions = formatter.TreeOptions

// MermaidOptions controls Mermaid diagram output formatting.
type MermaidOptions = formatter.MermaidOptions

// RenderTree renders data as an ASCII tree structure.
// Maps become branches with keys as labels, arrays show indexed children,
// and scalar values are displayed inline at leaves.
//
//	fmt.Print(tui.RenderTree(data, tui.TreeOptions{}))
func RenderTree(node any, opts TreeOptions) string {
	return formatter.FormatAsTree(node, opts)
}

// RenderMermaid renders data as a Mermaid flowchart diagram.
// Maps become nodes with edges to child nodes, arrays show indexed children,
// and scalar values are displayed as node labels.
//
//	fmt.Print(tui.RenderMermaid(data, tui.MermaidOptions{}))
func RenderMermaid(node any, opts MermaidOptions) string {
	return formatter.FormatAsMermaid(node, opts)
}

// Render formats data according to the given OutputFormat.
//
// FormatTable and FormatList accept TableOptions for fine-tuning (borders, columnar mode, etc.).
// FormatYAML and FormatJSON ignore opts and render plain serialized output.
//
// Example:
//
//	fmt.Print(tui.Render(data, tui.FormatTable, tui.TableOptions{Bordered: true}))
//	fmt.Print(tui.Render(data, tui.FormatList, tui.TableOptions{NoColor: true}))
//	fmt.Print(tui.Render(data, tui.FormatJSON, tui.TableOptions{}))
func Render(node any, format OutputFormat, opts TableOptions) string {
	switch format {
	case FormatTable:
		return RenderTable(node, opts)
	case FormatList:
		return RenderList(node, opts.NoColor)
	case FormatYAML:
		return renderYAML(node)
	case FormatJSON:
		return renderJSON(node)
	case FormatTree:
		return RenderTree(node, TreeOptions{})
	case FormatMermaid:
		return RenderMermaid(node, MermaidOptions{})
	default:
		return RenderTable(node, opts)
	}
}

func renderYAML(node any) string {
	b, err := yaml.Marshal(node)
	if err != nil {
		return fmt.Sprintf("error: %v\n", err)
	}
	return string(b)
}

func renderJSON(node any) string {
	b, err := json.MarshalIndent(node, "", "  ")
	if err != nil {
		return fmt.Sprintf("error: %v\n", err)
	}
	return string(b) + "\n"
}
