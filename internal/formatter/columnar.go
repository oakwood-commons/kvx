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

// CalculateNaturalColumnarWidth calculates the natural width needed for a columnar table
// without truncation. Returns the width needed for all columns including separators.
func CalculateNaturalColumnarWidth(columns []string, rows [][]string, showRowNum bool, numRows int) int {
	if len(columns) == 0 {
		return 0
	}

	sepWidth := 2

	// Calculate row number column width
	rowNumWidth := 0
	if showRowNum {
		rowNumWidth = len(fmt.Sprintf("%d", numRows)) + 2 // padding
	}

	// Calculate natural width for each column (header + max data)
	colWidths := make([]int, len(columns))
	for i, col := range columns {
		colWidths[i] = lipgloss.Width(col)
	}
	for _, row := range rows {
		for i, val := range row {
			if i < len(colWidths) {
				w := lipgloss.Width(val)
				if w > colWidths[i] {
					colWidths[i] = w
				}
			}
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

	// OriginalNames maps display column names back to original field names.
	// Used to look up ColumnHints when display names differ from field names.
	// If nil, column names are used as-is for hint lookup.
	OriginalNames []string
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

	// Build per-column hint lookup for visible columns.
	// OriginalNames maps display names → original field names for hint lookup.
	colAligns := make([]string, len(visibleCols))
	if len(opts.ColumnHints) > 0 {
		for i, col := range visibleCols {
			origName := col
			if opts.OriginalNames != nil {
				// Find the original name for this visible column
				for oi, oc := range columns {
					if oc == col || (oi < len(opts.OriginalNames) && opts.OriginalNames[oi] == col) {
						if oi < len(opts.OriginalNames) {
							origName = opts.OriginalNames[oi]
						}
						break
					}
				}
			}
			if h, ok := opts.ColumnHints[origName]; ok {
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
	colWidths := calculateColumnWidths(visibleCols, visibleRows, availableWidth, resolveHints(visibleCols, columns, opts))

	var b strings.Builder

	// Render header
	headerRow := renderHeader(visibleCols, colWidths, sepWidth, rowNumWidth, showRowNum, opts.NoColor)
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

func calculateColumnWidths(columns []string, rows [][]string, availableWidth int, hints map[string]ColumnHint) []int {
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
	for i, col := range columns {
		if h, ok := hints[col]; ok && h.MaxWidth > 0 && widths[i] > h.MaxWidth {
			widths[i] = h.MaxWidth
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
			widths = shrinkByPriority(columns, widths, usableWidth, hints)
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

// resolveHints maps visible (display) column names to their ColumnHint using
// OriginalNames for lookup when display names differ from field names.
func resolveHints(visibleCols, allCols []string, opts ColumnarOptions) map[string]ColumnHint {
	if len(opts.ColumnHints) == 0 {
		return nil
	}

	result := make(map[string]ColumnHint, len(visibleCols))
	for _, vc := range visibleCols {
		origName := vc
		if opts.OriginalNames != nil {
			for oi, ac := range allCols {
				if ac == vc || (oi < len(opts.OriginalNames) && opts.OriginalNames[oi] == vc) {
					if oi < len(opts.OriginalNames) {
						origName = opts.OriginalNames[oi]
					}
					break
				}
			}
		}
		if h, ok := opts.ColumnHints[origName]; ok {
			result[vc] = h
		}
	}
	return result
}

// shrinkByPriority reduces column widths to fit within usableWidth by shrinking
// lowest-priority columns first. Higher Priority values mean the column is more
// important and will be shrunk last.
func shrinkByPriority(columns []string, widths []int, usableWidth int, hints map[string]ColumnHint) []int {
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
	cols := make([]colPri, len(columns))
	for i, col := range columns {
		pri := 0
		if h, ok := hints[col]; ok {
			pri = h.Priority
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

// padLeft right-aligns s within the given width, padding with spaces on the left.
func padLeft(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return truncate(s, width)
	}
	return strings.Repeat(" ", width-w) + s
}
