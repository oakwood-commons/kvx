package ui

import (
	"strings"
	"testing"
)

func TestLayoutManager_CalculateHeights(t *testing.T) {
	tests := []struct {
		name               string
		width              int
		height             int
		helpVisible        bool
		reserveDebugLine   bool
		showSuggestions    bool
		showTitle          bool
		expectedTable      int
		expectedSuggestion int
		expectedDebug      int
	}{
		{
			name:               "normal window without suggestions",
			width:              100,
			height:             40,
			helpVisible:        false,
			reserveDebugLine:   false,
			showSuggestions:    false,
			showTitle:          true,
			expectedTable:      40 - 1 - 0 - 1 - 0 - 1 - 1 - 2 - TitleLineCount, // height - input - suggestions - status - debug - footer - separator - header - title
			expectedSuggestion: 0,
			expectedDebug:      0,
		},
		{
			name:               "normal window with suggestions",
			width:              100,
			height:             40,
			helpVisible:        false,
			reserveDebugLine:   false,
			showSuggestions:    true,
			showTitle:          true,
			expectedTable:      40 - 1 - 3 - 1 - 0 - 1 - 1 - 2 - TitleLineCount, // height - input - suggestions(3) - status - debug - footer - separator - header - title
			expectedSuggestion: 3,
			expectedDebug:      0,
		},
		{
			name:               "normal window with help",
			width:              100,
			height:             40,
			helpVisible:        true,
			reserveDebugLine:   false,
			showSuggestions:    false,
			showTitle:          true,
			expectedTable:      40 - 1 - 13 - 1 - 0 - 1 - 1 - 2 - TitleLineCount, // height - input - help(13) - status - debug - footer - separator - header - title
			expectedSuggestion: 13,
			expectedDebug:      0,
		},
		{
			name:               "normal window with debug",
			width:              100,
			height:             40,
			helpVisible:        false,
			reserveDebugLine:   true,
			showSuggestions:    false,
			showTitle:          true,
			expectedTable:      40 - 1 - 0 - 1 - 1 - 1 - 1 - 2 - TitleLineCount, // height - input - suggestions - status - debug(1) - footer - separator - header - title
			expectedSuggestion: 0,
			expectedDebug:      1,
		},
		{
			name:               "normal window with suggestions and debug",
			width:              100,
			height:             40,
			helpVisible:        false,
			reserveDebugLine:   true,
			showSuggestions:    true,
			showTitle:          true,
			expectedTable:      40 - 1 - 3 - 1 - 1 - 1 - 1 - 2 - TitleLineCount, // height - input - suggestions(3) - status - debug(1) - footer - separator - header - title
			expectedSuggestion: 3,
			expectedDebug:      1,
		},
		{
			name:               "small window minimum table height",
			width:              100,
			height:             10,
			helpVisible:        false,
			reserveDebugLine:   false,
			showSuggestions:    false,
			showTitle:          true,
			expectedTable:      MinTableHeight, // Should enforce minimum (actual calculation may be higher if space allows)
			expectedSuggestion: 0,
			expectedDebug:      0,
		},
		{
			name:               "small window with suggestions minimum table height",
			width:              100,
			height:             15,
			helpVisible:        false,
			reserveDebugLine:   false,
			showSuggestions:    true,
			showTitle:          true,
			expectedTable:      MinTableHeight, // Should enforce minimum even with suggestions (actual calculation may be higher if space allows)
			expectedSuggestion: 3,
			expectedDebug:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := NewLayoutManager(tt.width, tt.height)
			heights := lm.CalculateHeights(tt.helpVisible, tt.reserveDebugLine, tt.showSuggestions, tt.showTitle)
			// For small windows, just verify minimum is enforced
			if tt.height <= 15 {
				if heights.TableHeight < tt.expectedTable {
					t.Errorf("TableHeight = %d, expected at least %d", heights.TableHeight, tt.expectedTable)
				}
			} else {
				if heights.TableHeight != tt.expectedTable {
					t.Errorf("TableHeight = %d, expected %d", heights.TableHeight, tt.expectedTable)
				}
			}
			if heights.SuggestionHeight != tt.expectedSuggestion {
				t.Errorf("SuggestionHeight = %d, expected %d", heights.SuggestionHeight, tt.expectedSuggestion)
			}
			if heights.DebugHeight != tt.expectedDebug {
				t.Errorf("DebugHeight = %d, expected %d", heights.DebugHeight, tt.expectedDebug)
			}
			if heights.PathInputHeight != 1 {
				t.Errorf("PathInputHeight = %d, expected 1", heights.PathInputHeight)
			}
			if heights.StatusHeight != 1 {
				t.Errorf("StatusHeight = %d, expected 1", heights.StatusHeight)
			}
			if heights.FooterHeight != 1 {
				t.Errorf("FooterHeight = %d, expected 1", heights.FooterHeight)
			}

			// Verify total height calculation is correct
			totalUsed := heights.TableHeight + heights.TitleHeight + TableHeaderLines + heights.PathInputHeight + heights.SuggestionHeight + heights.StatusHeight + heights.DebugHeight + heights.FooterHeight + 1 // separator
			if totalUsed > tt.height {
				t.Errorf("Total height used (%d) exceeds window height (%d)", totalUsed, tt.height)
			}
		})
	}
}

func TestLayoutManager_CalculateColumnWidths(t *testing.T) {
	tests := []struct {
		name                    string
		width                   int
		keyColWidth             int
		configuredValueColWidth int
		expectedKeyWidth        int
		expectedValueWidth      int
	}{
		{
			name:                    "default widths",
			width:                   100,
			keyColWidth:             0,
			configuredValueColWidth: 0,
			expectedKeyWidth:        DefaultKeyColWidth,
			expectedValueWidth:      100 - DefaultKeyColWidth - ColumnSeparatorWidth,
		},
		{
			name:                    "custom key width",
			width:                   100,
			keyColWidth:             40,
			configuredValueColWidth: 0,
			expectedKeyWidth:        40,
			expectedValueWidth:      100 - 40 - ColumnSeparatorWidth,
		},
		{
			name:                    "custom value width",
			width:                   100,
			keyColWidth:             30,
			configuredValueColWidth: 50,
			expectedKeyWidth:        30,
			expectedValueWidth:      50,
		},
		{
			name:                    "value width exceeds available",
			width:                   100,
			keyColWidth:             30,
			configuredValueColWidth: 100, // Too large
			expectedKeyWidth:        30,
			expectedValueWidth:      100 - 30 - ColumnSeparatorWidth, // Should be clamped
		},
		{
			name:                    "value width below minimum",
			width:                   100,
			keyColWidth:             30,
			configuredValueColWidth: 5, // Too small
			expectedKeyWidth:        30,
			expectedValueWidth:      MinValueColWidth, // Should be clamped to minimum
		},
		{
			name:                    "zero width window",
			width:                   0,
			keyColWidth:             30,
			configuredValueColWidth: 0,
			expectedKeyWidth:        30,
			expectedValueWidth:      DefaultValueColWidth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := NewLayoutManager(tt.width, 24)
			keyW, valueW := lm.CalculateColumnWidths(tt.keyColWidth, tt.configuredValueColWidth, nil)
			if keyW != tt.expectedKeyWidth {
				t.Errorf("KeyWidth = %d, expected %d", keyW, tt.expectedKeyWidth)
			}
			if valueW != tt.expectedValueWidth {
				t.Errorf("ValueWidth = %d, expected %d", valueW, tt.expectedValueWidth)
			}

			// Verify total doesn't exceed window width (when width > 0)
			if tt.width > 0 {
				total := keyW + valueW + ColumnSeparatorWidth
				if total > tt.width {
					t.Errorf("Total width (%d) exceeds window width (%d)", total, tt.width)
				}
			}
		})
	}
}

func TestLayoutManager_CalculateInputWidth(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		expected int
	}{
		{
			name:     "normal width",
			width:    100,
			expected: 100 - PathInputPromptWidth,
		},
		{
			name:     "small width below minimum",
			width:    15,
			expected: MinInputWidth,
		},
		{
			name:     "zero width",
			width:    0,
			expected: 80, // Default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := NewLayoutManager(tt.width, 24)
			width := lm.CalculateInputWidth()

			if width != tt.expected {
				t.Errorf("InputWidth = %d, expected %d", width, tt.expected)
			}
		})
	}
}

func TestModel_ApplyLayout_ExpressionMode(t *testing.T) {
	// Test that entering expression mode (F6) reserves space for suggestions
	m := InitialModel(map[string]interface{}{"key": "value"})
	m.WinWidth = 100
	m.WinHeight = 40
	m.AllowIntellisense = true
	m.AllowSuggestions = true

	// Initial state: not in expression mode
	m.InputFocused = false
	m.ShowSuggestions = false
	m.applyLayout(true)

	initialTableHeight := m.TableHeight

	// Enter expression mode
	m.InputFocused = true
	m.ShowSuggestions = false
	m.applyLayout(true)

	afterF6TableHeight := m.TableHeight

	// Table height should not be reduced (suggestions are now in status bar, not a dropdown)
	if afterF6TableHeight != initialTableHeight {
		t.Errorf("After F6: TableHeight = %d, expected %d (no reduction - suggestions in status bar)", afterF6TableHeight, initialTableHeight)
	}
}

func TestModel_ApplyLayout_TopAnchoredPopup(t *testing.T) {
	// Test that top-anchored popups reduce table height
	m := InitialModel(map[string]interface{}{"key": "value"})
	m.WinWidth = 100
	m.WinHeight = 40
	m.HelpPopupText = "Test popup\nLine 2\nLine 3"
	m.HelpPopupAnchor = "top"
	m.HelpPopupJustify = "center"

	// Without popup
	m.HelpVisible = false
	m.applyLayout(true)
	heightWithoutPopup := m.TableHeight

	// With top-anchored popup
	m.HelpVisible = true
	m.applyLayout(true)
	heightWithPopup := m.TableHeight

	// Table height should be reduced when popup is visible
	if heightWithPopup >= heightWithoutPopup {
		t.Errorf("Table height with popup (%d) should be less than without (%d)", heightWithPopup, heightWithoutPopup)
	}

	// Table height should not go below minimum
	if heightWithPopup < MinTableHeight {
		t.Errorf("Table height with popup (%d) should be at least %d", heightWithPopup, MinTableHeight)
	}
}

func TestModel_ApplyLayout_SuggestionSpaceReservation(t *testing.T) {
	// Test that suggestion space is reserved when InputFocused is true, even if ShowSuggestions is false
	m := InitialModel(map[string]interface{}{"key": "value"})
	m.WinWidth = 100
	m.WinHeight = 40
	m.AllowIntellisense = true
	m.AllowSuggestions = true

	// Not in expression mode
	m.InputFocused = false
	m.ShowSuggestions = false
	m.applyLayout(true)
	heightNotInExpr := m.TableHeight

	// In expression mode but suggestions not showing
	m.InputFocused = true
	m.ShowSuggestions = false
	m.applyLayout(true)
	heightInExprNoSuggestions := m.TableHeight

	// In expression mode with suggestions showing
	m.InputFocused = true
	m.ShowSuggestions = true
	m.applyLayout(true)
	heightInExprWithSuggestions := m.TableHeight

	// Both expression mode states should reserve the same space (suggestions are in status bar, not dropdown)
	if heightInExprNoSuggestions != heightInExprWithSuggestions {
		t.Errorf("Table height should be same whether suggestions are showing or not: %d vs %d", heightInExprNoSuggestions, heightInExprWithSuggestions)
	}

	// Table height should be the same as not in expression mode (no space reserved for dropdown)
	if heightNotInExpr != heightInExprNoSuggestions {
		t.Errorf("Table height should be same in expression mode (suggestions in status bar): %d vs %d", heightNotInExpr, heightInExprNoSuggestions)
	}
}

func TestModel_ApplyLayout_MinimumTableHeight(t *testing.T) {
	// Test that minimum table height is enforced even with popups and suggestions
	m := InitialModel(map[string]interface{}{"key": "value"})
	m.WinWidth = 100
	m.WinHeight = 10 // Very small window
	m.HelpPopupText = "Large popup\nLine 1\nLine 2\nLine 3\nLine 4\nLine 5"
	m.HelpPopupAnchor = "top"
	m.HelpVisible = true
	m.InputFocused = true
	m.ShowSuggestions = true

	m.applyLayout(true)

	if m.TableHeight < MinTableHeight {
		t.Errorf("Table height (%d) should be at least %d even with large popup", m.TableHeight, MinTableHeight)
	}
}

func TestRenderPanelLayout_HelpPopupOverridesHelpText(t *testing.T) {
	state := PanelLayoutState{
		WinWidth:      80,
		WinHeight:     20,
		HelpVisible:   true,
		HelpText:      "base help should not render",
		HelpPopupText: "popup content only",
		Title:         "kvx",
		DisplayNode:   map[string]any{"key": "value"},
		RowCount:      1,
		SelectedRow:   0,
		PathLabel:     "_",
		KeyColWidth:   DefaultKeyColWidth,
	}

	output := RenderPanelLayout(state)
	if !strings.Contains(output, "popup content only") {
		t.Fatalf("expected popup content to render, got:\n%s", output)
	}
	if strings.Contains(output, "base help should not render") {
		t.Fatalf("expected base help text to be suppressed when popup is present")
	}
}
