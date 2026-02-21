package formatter

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
)

// TableFormatOptions configures overall table rendering behavior.
type TableFormatOptions struct {
	// ArrayStyle controls how array indices are displayed:
	//   "index" = [0], [1], [2]; "numbered" = 1, 2, 3 (default); "bullet" = •; "none" = no index
	ArrayStyle string

	// ColumnarMode controls when arrays render as multi-column tables:
	//   "auto" = detect homogeneous arrays (default); "always" = force; "never" = KEY/VALUE only
	ColumnarMode string

	// ColumnOrder specifies preferred column order for columnar tables.
	ColumnOrder []string

	// HiddenColumns specifies columns to omit from columnar tables.
	HiddenColumns []string

	// SelectColumns, when non-empty, means "show only these columns".
	// Any column not in this list is automatically hidden.
	// The column order is also derived from this list.
	SelectColumns []string

	// ColumnHints provides per-column display hints derived from a JSON Schema.
	// Keys are the original JSON field names.
	ColumnHints map[string]ColumnHint
}

// DefaultTableFormatOptions returns sensible defaults for table formatting.
func DefaultTableFormatOptions() TableFormatOptions {
	return TableFormatOptions{
		ArrayStyle:   "numbered",
		ColumnarMode: "auto",
	}
}

// ApplySelectColumns resolves SelectColumns against the actual column names.
// When SelectColumns is non-empty, any column not in the set is appended to
// HiddenColumns and ColumnOrder is set to SelectColumns.
func (opts *TableFormatOptions) ApplySelectColumns(columns []string) {
	if len(opts.SelectColumns) == 0 {
		return
	}
	selected := make(map[string]bool, len(opts.SelectColumns))
	for _, c := range opts.SelectColumns {
		selected[c] = true
	}
	hidden := make(map[string]bool, len(opts.HiddenColumns))
	for _, c := range opts.HiddenColumns {
		hidden[c] = true
	}
	for _, c := range columns {
		if !selected[c] && !hidden[c] {
			opts.HiddenColumns = append(opts.HiddenColumns, c)
		}
	}
	opts.ColumnOrder = opts.SelectColumns
}

// EffectiveColumnOrder returns SelectColumns if set, otherwise ColumnOrder.
func (opts *TableFormatOptions) EffectiveColumnOrder() []string {
	if len(opts.SelectColumns) > 0 {
		return opts.SelectColumns
	}
	return opts.ColumnOrder
}

// CalculateNaturalColumnarWidth calculates the natural width needed for a columnar table
// without truncation. Returns the width needed for all columns including separators.
func CalculateNaturalColumnarWidth(columns []string, rows [][]string, showRowNum bool, numRows int) int {
	return CalculateNaturalColumnarWidthWithHints(columns, rows, showRowNum, numRows, nil, nil)
}

// CalculateNaturalColumnarWidthWithHints is like CalculateNaturalColumnarWidth but
// accounts for hidden columns, display-name overrides, and MaxWidth caps from hints.
func CalculateNaturalColumnarWidthWithHints(columns []string, rows [][]string, showRowNum bool, numRows int, hints map[string]ColumnHint, hiddenColumns []string) int {
	if len(columns) == 0 {
		return 0
	}

	// Filter out hidden columns
	visCols, visRows := filterColumns(columns, rows, hiddenColumns)
	if len(visCols) == 0 {
		return 0
	}

	sepWidth := 2

	// Calculate row number column width
	rowNumWidth := 0
	if showRowNum {
		rowNumWidth = len(fmt.Sprintf("%d", numRows)) + 2 // padding
	}

	// Calculate natural width for each column (header or display name + max data)
	colWidths := make([]int, len(visCols))
	for i, col := range visCols {
		header := col
		if h, ok := hints[col]; ok && h.DisplayName != "" {
			header = h.DisplayName
		}
		colWidths[i] = lipgloss.Width(header)
	}
	for _, row := range visRows {
		for i, val := range row {
			if i < len(colWidths) {
				w := lipgloss.Width(val)
				if w > colWidths[i] {
					colWidths[i] = w
				}
			}
		}
	}

	// Apply MaxWidth caps from hints
	for i, col := range visCols {
		if h, ok := hints[col]; ok && h.MaxWidth > 0 && colWidths[i] > h.MaxWidth {
			colWidths[i] = h.MaxWidth
		}
	}

	// Sum up total width: rowNum + sep + col1 + sep + col2 + sep + ...
	totalWidth := 0
	if showRowNum {
		totalWidth += rowNumWidth + sepWidth
	}
	for i, w := range colWidths {
		totalWidth += w
		if i < len(colWidths)-1 {
			totalWidth += sepWidth
		}
	}

	return totalWidth
}

// ColumnarOptions configures columnar table rendering.
type ColumnarOptions struct {
	// NoColor disables color output
	NoColor bool

	// TotalWidth is the total available width. If 0, uses terminal width.
	TotalWidth int

	// RowNumberStyle controls how row numbers are displayed:
	//   "numbered" - 1, 2, 3 (default)
	//   "index"    - [0], [1], [2]
	//   "bullet"   - •
	//   "none"     - no row number column
	RowNumberStyle string

	// ColumnOrder specifies preferred column order. Unspecified columns are appended.
	ColumnOrder []string

	// HiddenColumns specifies columns to omit from output.
	HiddenColumns []string

	// ColumnHints provides per-column display hints for width, priority, and alignment.
	// Keys are the original field names (before any display name remapping).
	ColumnHints map[string]ColumnHint
}

// RenderColumnarTable renders data as a multi-column table with field names as headers.
// columns: the field names (column headers)
// rows: the data rows (each row has values corresponding to columns)
func RenderColumnarTable(columns []string, rows [][]string, opts ColumnarOptions) string {
	if len(columns) == 0 || len(rows) == 0 {
		return ""
	}

	// Filter hidden columns
	visibleCols, visibleRows := filterColumns(columns, rows, opts.HiddenColumns)
	if len(visibleCols) == 0 {
		return ""
	}

	// Apply DisplayName overrides to visible columns for headers.
	// Keep track of original names for hint lookup.
	displayCols := make([]string, len(visibleCols))
	for i, col := range visibleCols {
		displayCols[i] = col
		if h, ok := opts.ColumnHints[col]; ok && h.DisplayName != "" {
			displayCols[i] = h.DisplayName
		}
	}

	// Build per-column alignment lookup.
	// visibleCols are original field names, matching ColumnHints keys.
	colAligns := make([]string, len(visibleCols))
	if len(opts.ColumnHints) > 0 {
		for i, col := range visibleCols {
			if h, ok := opts.ColumnHints[col]; ok {
				colAligns[i] = h.Align
			}
		}
	}

	// Determine total width
	totalWidth := opts.TotalWidth
	if totalWidth <= 0 {
		totalWidth = getTerminalWidth()
	}

	// Calculate whether we need a row number column
	showRowNum := opts.RowNumberStyle != "none"
	rowNumWidth := 0
	if showRowNum {
		// Width based on number of rows
		maxRowNum := len(rows)
		rowNumWidth = len(fmt.Sprintf("%d", maxRowNum)) + 2 // padding
		if opts.RowNumberStyle == "bullet" {
			rowNumWidth = 3 // "• " plus padding
		}
	}

	// Calculate column widths
	sepWidth := 2
	availableWidth := totalWidth - rowNumWidth
	if showRowNum {
		availableWidth -= sepWidth
	}
	colWidths := calculateColumnWidths(displayCols, visibleRows, availableWidth, resolveHints(visibleCols, opts))

	var b strings.Builder

	// Render header
	headerRow := renderHeader(displayCols, colWidths, sepWidth, rowNumWidth, showRowNum, opts.NoColor)
	b.WriteString(headerRow + "\n")

	// Separator line - width needs to match header width including row number separator
	totalHeaderWidth := rowNumWidth
	if showRowNum && len(colWidths) > 0 {
		totalHeaderWidth += sepWidth // separator between row number and first column
	}
	for i, w := range colWidths {
		totalHeaderWidth += w
		if i < len(colWidths)-1 {
			totalHeaderWidth += sepWidth
		}
	}
	separator := strings.Repeat("─", totalHeaderWidth)
	if !opts.NoColor {
		separator = separatorStyle.Render(separator)
	}
	b.WriteString(separator + "\n")

	// Render data rows
	for i, row := range visibleRows {
		rowStr := renderDataRow(i, row, colWidths, sepWidth, rowNumWidth, opts.RowNumberStyle, opts.NoColor, colAligns)
		b.WriteString(rowStr + "\n")
	}

	return b.String()
}

func filterColumns(columns []string, rows [][]string, hidden []string) ([]string, [][]string) {
	if len(hidden) == 0 {
		return columns, rows
	}

	hiddenSet := make(map[string]bool, len(hidden))
	for _, h := range hidden {
		hiddenSet[h] = true
	}

	// Find indices of visible columns
	visibleIndices := make([]int, 0, len(columns))
	visibleCols := make([]string, 0, len(columns))
	for i, col := range columns {
		if !hiddenSet[col] {
			visibleIndices = append(visibleIndices, i)
			visibleCols = append(visibleCols, col)
		}
	}

	// Filter rows to only include visible columns
	visibleRows := make([][]string, len(rows))
	for i, row := range rows {
		newRow := make([]string, len(visibleIndices))
		for j, idx := range visibleIndices {
			if idx < len(row) {
				newRow[j] = row[idx]
			}
		}
		visibleRows[i] = newRow
	}

	return visibleCols, visibleRows
}

func calculateColumnWidths(columns []string, rows [][]string, availableWidth int, hints []ColumnHint) []int {
	numCols := len(columns)
	if numCols == 0 {
		return nil
	}

	const sepWidth = 2
	const minColWidth = 3
	widths := make([]int, numCols)
	for i, col := range columns {
		widths[i] = lipgloss.Width(col)
	}

	// Expand to fit data
	for _, row := range rows {
		for i, val := range row {
			if i < numCols {
				w := lipgloss.Width(val)
				if w > widths[i] {
					widths[i] = w
				}
			}
		}
	}

	// Apply MaxWidth caps from hints before any shrinking
	for i := range columns {
		if i < len(hints) && hints[i].MaxWidth > 0 && widths[i] > hints[i].MaxWidth {
			widths[i] = hints[i].MaxWidth
		}
	}

	// Calculate space for separators and determine if we need to shrink
	totalSeps := (numCols - 1) * sepWidth
	usableWidth := availableWidth - totalSeps

	// Calculate total needed
	totalNeeded := 0
	for _, w := range widths {
		totalNeeded += w
	}

	// Only apply constraints if we exceed available space
	if totalNeeded > usableWidth && usableWidth > 0 {
		if len(hints) > 0 {
			// Priority-based shrinking: shrink lowest-priority columns first
			widths = shrinkByPriority(widths, usableWidth, hints)
		} else {
			// Original behavior: cap then proportional shrink
			maxColWidth := 40
			for i := range widths {
				if widths[i] > maxColWidth {
					widths[i] = maxColWidth
				}
			}

			totalNeeded = 0
			for _, w := range widths {
				totalNeeded += w
			}

			if totalNeeded > usableWidth {
				totalOriginal := 0
				for _, w := range widths {
					totalOriginal += w
				}

				for i := range widths {
					proportion := float64(widths[i]) / float64(totalOriginal)
					newWidth := int(proportion * float64(usableWidth))
					if newWidth < minColWidth {
						newWidth = minColWidth
					}
					widths[i] = newWidth
				}

				// Final adjustment: ensure total doesn't exceed usableWidth
				for {
					total := 0
					for _, w := range widths {
						total += w
					}
					if total <= usableWidth {
						break
					}
					maxIdx := 0
					for i := 1; i < numCols; i++ {
						if widths[i] > widths[maxIdx] {
							maxIdx = i
						}
					}
					if widths[maxIdx] > minColWidth {
						widths[maxIdx]--
					} else {
						break
					}
				}
			}
		}
	}

	return widths
}

func renderHeader(columns []string, widths []int, sepWidth, rowNumWidth int, showRowNum, noColor bool) string {
	sep := strings.Repeat(" ", sepWidth)
	parts := make([]string, 0, len(columns)+1)

	// Row number header
	if showRowNum {
		header := padRight("#", rowNumWidth)
		if !noColor {
			header = headerStyle.Render(header)
		}
		parts = append(parts, header)
	}

	// Column headers
	for i, col := range columns {
		w := widths[i]
		header := padRight(truncate(col, w), w)
		if !noColor {
			header = headerStyle.Render(header)
		}
		parts = append(parts, header)
	}

	return strings.Join(parts, sep)
}

func renderDataRow(rowIndex int, values []string, widths []int, sepWidth, rowNumWidth int, rowNumStyle string, noColor bool, colAligns []string) string {
	sep := strings.Repeat(" ", sepWidth)
	parts := make([]string, 0, len(values)+1)

	// Row number
	if rowNumStyle != "none" {
		var numStr string
		switch rowNumStyle {
		case "index":
			numStr = fmt.Sprintf("[%d]", rowIndex)
		case "bullet":
			numStr = "•"
		default: // "numbered"
			numStr = fmt.Sprintf("%d", rowIndex+1)
		}
		numStr = padRight(numStr, rowNumWidth)
		if !noColor {
			numStr = keyStyle.Render(numStr)
		}
		parts = append(parts, numStr)
	}

	// Values
	for i, val := range values {
		if i >= len(widths) {
			break
		}
		w := widths[i]
		var valStr string
		if i < len(colAligns) && colAligns[i] == "right" {
			valStr = padLeft(truncate(val, w), w)
		} else {
			valStr = padRight(truncate(val, w), w)
		}
		if !noColor {
			valStr = valueStyle.Render(valStr)
		}
		parts = append(parts, valStr)
	}

	return strings.Join(parts, sep)
}

// resolveHints builds a per-column hint slice so that calculateColumnWidths
// and shrinkByPriority can look up hints by index, avoiding collisions when
// multiple columns share the same display name.
//
// visibleCols: original field names after hidden-column filtering.
// opts:        ColumnarOptions carrying the ColumnHints map (keyed by original field name).
func resolveHints(visibleCols []string, opts ColumnarOptions) []ColumnHint {
	if len(opts.ColumnHints) == 0 {
		return nil
	}

	result := make([]ColumnHint, len(visibleCols))
	for i, vc := range visibleCols {
		if h, ok := opts.ColumnHints[vc]; ok {
			result[i] = h
		}
	}
	return result
}

// shrinkByPriority reduces column widths to fit within usableWidth by shrinking
// lowest-priority columns first. Higher Priority values mean the column is more
// important and will be shrunk last.
func shrinkByPriority(widths []int, usableWidth int, hints []ColumnHint) []int {
	const minColWidth = 3
	total := 0
	for _, w := range widths {
		total += w
	}
	excess := total - usableWidth
	if excess <= 0 {
		return widths
	}

	// Build column indices sorted by priority ascending (lowest priority first)
	type colPri struct {
		idx      int
		priority int
	}
	cols := make([]colPri, len(widths))
	for i := range widths {
		pri := 0
		if i < len(hints) {
			pri = hints[i].Priority
		}
		cols[i] = colPri{idx: i, priority: pri}
	}
	sort.Slice(cols, func(a, b int) bool {
		return cols[a].priority < cols[b].priority
	})

	// Shrink columns starting from lowest priority
	for _, cp := range cols {
		if excess <= 0 {
			break
		}
		shrinkable := widths[cp.idx] - minColWidth
		if shrinkable <= 0 {
			continue
		}
		shrink := shrinkable
		if shrink > excess {
			shrink = excess
		}
		widths[cp.idx] -= shrink
		excess -= shrink
	}

	return widths
}

// minReadableWidth is the minimum column width before data becomes unreadable.
// Below this, columns show mostly ellipsis (e.g. "ab...") which is useless.
const minReadableWidth = 8

// IsColumnarReadableOpts configures the readability check to match the renderer.
type IsColumnarReadableOpts struct {
	HiddenColumns  []string // columns to exclude (same as ColumnarOptions.HiddenColumns)
	RowNumberStyle string   // "none" means no row number column; otherwise accounts for its width
}

// IsColumnarReadable checks whether a columnar table can be rendered readably
// within the given available width. It returns false if any column whose natural
// width exceeds minReadableWidth would be shrunk below that threshold, meaning
// the table data would be effectively truncated to the point of being unusable.
//
// This function applies the same transformations as RenderColumnarTable:
// filtering hidden columns and accounting for the row-number column width.
func IsColumnarReadable(columns []string, rows [][]string, availableWidth int, hints map[string]ColumnHint, opts IsColumnarReadableOpts) bool {
	if len(columns) == 0 {
		return true
	}

	// Filter hidden columns (same as the renderer does)
	visibleCols, visibleRows := filterColumns(columns, rows, opts.HiddenColumns)
	if len(visibleCols) == 0 {
		return true
	}

	// Account for row number column (same calculation as RenderColumnarTable)
	showRowNum := opts.RowNumberStyle != "none" && opts.RowNumberStyle != ""
	rowNumWidth := 0
	if showRowNum {
		maxRowNum := len(visibleRows)
		if maxRowNum == 0 {
			maxRowNum = 1
		}
		rowNumWidth = len(fmt.Sprintf("%d", maxRowNum)) + 2 // padding
		if opts.RowNumberStyle == "bullet" {
			rowNumWidth = 3 // "• " plus padding
		}
	}

	const sepWidth = 2
	effectiveWidth := availableWidth - rowNumWidth
	if showRowNum {
		effectiveWidth -= sepWidth
	}

	if effectiveWidth < minReadableWidth {
		return false
	}

	// Calculate natural widths to know each column's unconstrained size
	naturalWidths := make([]int, len(visibleCols))
	for i, col := range visibleCols {
		naturalWidths[i] = lipgloss.Width(col)
	}
	for _, row := range visibleRows {
		for i, val := range row {
			if i < len(naturalWidths) {
				if w := lipgloss.Width(val); w > naturalWidths[i] {
					naturalWidths[i] = w
				}
			}
		}
	}

	// Convert hints map to slice for calculateColumnWidths
	var hintSlice []ColumnHint
	if len(hints) > 0 {
		hintSlice = make([]ColumnHint, len(visibleCols))
		for i, col := range visibleCols {
			if h, ok := hints[col]; ok {
				hintSlice[i] = h
			}
		}
	}

	// Calculate assigned widths using the same algorithm the renderer uses
	assigned := calculateColumnWidths(visibleCols, visibleRows, effectiveWidth, hintSlice)

	// Check if any column that naturally needs more than minReadableWidth
	// was shrunk below that threshold
	for i, w := range assigned {
		if naturalWidths[i] >= minReadableWidth && w < minReadableWidth {
			return false
		}
	}

	return true
}

// padLeft right-aligns s within the given width, padding with spaces on the left.
func padLeft(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return truncate(s, width)
	}
	return strings.Repeat(" ", width-w) + s
}
