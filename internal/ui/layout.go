package ui

// LayoutManager manages the layout calculations for the TUI
type LayoutManager struct {
	width  int
	height int
}

// AutoKeyWidthFunc computes a preferred key column width (e.g., based on visible data) capped by a provided maximum.
// If nil, CalculateColumnWidths will treat the provided keyColWidth as the fixed width/cap.
type AutoKeyWidthFunc func(maxPreset int) int

// ComponentHeights defines the height requirements for each component
type ComponentHeights struct {
	TableHeight      int // Calculated based on available space
	PathInputHeight  int // Always 1 line
	SuggestionHeight int // Variable: 0-3 for suggestions, 13 for help
	TitleHeight      int // Optional: panel title/header height
	StatusHeight     int // Always 1 line
	DebugHeight      int // 0 or 1 line
	FooterHeight     int // Always 1 line
}

// Constants for component heights
const (
	PathInputLineCount     = 1
	StatusLineCount        = 1
	FooterLineCount        = 1
	DefaultSuggestionLines = 3
	HelpBoxLines           = 13
	TitleLineCount         = 1
	TableHeaderLines       = 2 // Header + border
	MinTableHeight         = 2 // Minimum to show header + 1 row
	ColumnSeparatorWidth   = 3 // Table component adds internal spacing between columns
	PathInputPromptWidth   = 2 // Space for prompt characters
	MinValueColWidth       = 10
	MinInputWidth          = 20
	DefaultKeyColWidth     = 30
	DefaultValueColWidth   = 60
)

// NewLayoutManager creates a new layout manager
func NewLayoutManager(width, height int) *LayoutManager {
	return &LayoutManager{
		width:  width,
		height: height,
	}
}

// SetDimensions updates the layout manager dimensions
func (lm *LayoutManager) SetDimensions(width, height int) {
	lm.width = width
	lm.height = height
}

// CalculateHeights calculates component heights based on window size and UI state.
// The debug line is reserved whenever reserveDebugLine is true (we always reserve it to keep bars stable).
// showTitle reserves an extra line above the table for the panel title/header.
func (lm *LayoutManager) CalculateHeights(helpVisible bool, reserveDebugLine bool, showSuggestions bool, showTitle bool) ComponentHeights {
	heights := ComponentHeights{
		PathInputHeight: PathInputLineCount,
		StatusHeight:    StatusLineCount,
		FooterHeight:    FooterLineCount,
	}

	if showTitle {
		heights.TitleHeight = TitleLineCount
	}

	// Calculate suggestion/help height
	switch {
	case helpVisible:
		heights.SuggestionHeight = HelpBoxLines
	case showSuggestions:
		// Reserve space for suggestions when visible (up to 3 lines)
		heights.SuggestionHeight = DefaultSuggestionLines
	default:
		heights.SuggestionHeight = 0 // No suggestions or help
	}

	// Calculate debug height
	if reserveDebugLine {
		heights.DebugHeight = 1
	}

	// Fixed space that cannot be squeezed (table headers + bars below)
	nonTable := heights.PathInputHeight +
		heights.StatusHeight +
		heights.FooterHeight +
		heights.DebugHeight +
		1 + // Separator line between table and expression bar
		TableHeaderLines +
		heights.TitleHeight

	// First, trim suggestions/help if they would push us past the available height.
	remaining := lm.height - nonTable - heights.SuggestionHeight
	if remaining < 0 {
		heights.SuggestionHeight += remaining // remaining is negative; shrink suggestions
		if heights.SuggestionHeight < 0 {
			heights.SuggestionHeight = 0
		}
		remaining = lm.height - nonTable - heights.SuggestionHeight
	}

	// Allocate remaining space to the table body, preferring at least MinTableHeight when possible.
	if remaining < 0 {
		remaining = 0
	}
	// If we can hit the minimum by borrowing from suggestions, do it.
	if remaining < MinTableHeight {
		needed := MinTableHeight - remaining
		if heights.SuggestionHeight >= needed {
			heights.SuggestionHeight -= needed
			remaining += needed
		}
	}

	// Final guard: do not exceed the window height.
	maxTable := lm.height - nonTable - heights.SuggestionHeight
	if maxTable < 0 {
		maxTable = 0
	}
	if remaining > maxTable {
		remaining = maxTable
	}

	heights.TableHeight = remaining

	return heights
}

// CalculateColumnWidths calculates column widths based on window width
func (lm *LayoutManager) CalculateColumnWidths(keyColWidth int, configuredValueColWidth int, autoKey AutoKeyWidthFunc) (keyWidth, valueWidth int) {
	// Resolve key width: allow caller to auto-shrink based on data, capped by provided preset.
	preset := keyColWidth
	if preset <= 0 {
		preset = DefaultKeyColWidth
	}
	if autoKey != nil {
		keyWidth = autoKey(preset)
	}
	if keyWidth <= 0 {
		keyWidth = preset
	}

	// If value column width is configured, use it (but ensure it fits)
	if configuredValueColWidth > 0 {
		if lm.width > 0 {
			// Ensure configured width doesn't exceed available space
			maxValueWidth := lm.width - keyWidth - ColumnSeparatorWidth
			if configuredValueColWidth > maxValueWidth {
				valueWidth = maxValueWidth
			} else {
				valueWidth = configuredValueColWidth
			}
			if valueWidth < MinValueColWidth {
				valueWidth = MinValueColWidth
			}
		} else {
			valueWidth = configuredValueColWidth
		}
	} else {
		// Calculate value width from remaining space
		if lm.width > 0 {
			valueWidth = lm.width - keyWidth - ColumnSeparatorWidth
			if valueWidth < MinValueColWidth {
				valueWidth = MinValueColWidth
			}
		} else {
			valueWidth = DefaultValueColWidth
		}
	}

	return keyWidth, valueWidth
}

// CalculateInputWidth calculates the width for the path input
func (lm *LayoutManager) CalculateInputWidth() int {
	if lm.width > 0 {
		inputWidth := lm.width - PathInputPromptWidth
		if inputWidth < MinInputWidth {
			return MinInputWidth
		}
		return inputWidth
	}
	return 80 // Default width
}

// GetWidth returns the current width
func (lm *LayoutManager) GetWidth() int {
	return lm.width
}

// GetHeight returns the current height
func (lm *LayoutManager) GetHeight() int {
	return lm.height
}
