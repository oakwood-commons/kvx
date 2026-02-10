package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	runewidth "github.com/mattn/go-runewidth"

	"github.com/oakwood-commons/kvx/internal/formatter"
)

// PanelLayoutInput is the minimal interface needed to render the input panel.
type PanelLayoutInput interface {
	SetWidth(int)
	SetCursor(int)
	Position() int
	View() string
	Value() string
}

// PanelLayoutState captures the panel layout inputs without shell update logic.
// This allows other models to drive the same layout while sharing rendering logic.
type PanelLayoutState struct {
	WinWidth       int
	WinHeight      int
	NoColor        bool
	SnapshotHeader bool
	DebugEnabled   bool
	AllowEditInput bool
	HideFooter     bool // Hide the footer bar (for non-interactive display)
	KeyMode        KeyMode

	HelpVisible bool
	HelpTitle   string
	HelpText    string

	Title string

	InfoMessage string
	InfoError   bool

	InfoPopupText string
	HelpPopupText string

	InputVisible    bool
	Input           PanelLayoutInput
	InputPreferHead bool

	ExprMode bool
	ExprType string

	SearchActive    bool
	SearchResults   []searchHit
	MapFilterActive bool // 'f' key filter mode for maps

	DisplayNode interface{}
	Node        interface{}
	RowCount    int
	SelectedRow int
	PathLabel   string
	KeyColWidth int
}

// RenderPanelLayout renders the panel layout using precomputed state.
func RenderPanelLayout(state PanelLayoutState) string {
	if state.WinWidth <= 0 {
		state.WinWidth = 80
	}
	if state.WinHeight <= 0 {
		state.WinHeight = 24
	}
	// Clamp to a minimal canvas to avoid negative layouts
	if state.WinHeight < 6 {
		state.WinHeight = 6
	}
	if state.WinWidth < 4 {
		state.WinWidth = 4
	}

	// Layout: optional info popup, data panel (bordered), optional help panel, status panel (borderless), optional input, footer.
	bottomHeight := 1
	statusPanelWidth := state.WinWidth
	if statusPanelWidth < 1 {
		statusPanelWidth = 1
	}
	statusPanelHeight := 0
	// Show status panel in normal mode, or in snapshot mode if there's non-error info (like function help)
	showStatusPanel := !state.SnapshotHeader || (strings.TrimSpace(state.InfoMessage) != "" && !state.InfoError)
	if showStatusPanel {
		wrappedInfo := wrapPlainText(strings.TrimRight(state.InfoMessage, "\n"), statusPanelWidth)
		linesInfo := strings.Split(wrappedInfo, "\n")
		if len(linesInfo) == 0 {
			linesInfo = []string{""}
		}
		statusPanelHeight = len(linesInfo)
		if statusPanelHeight < 1 {
			statusPanelHeight = 1
		}
	}
	inputPanelHeight := 0
	if state.InputVisible {
		inputPanelHeight = 3 // border + single-line content
	}
	panelWidth := state.WinWidth
	if panelWidth < 4 {
		panelWidth = 4
	}
	helpContent := strings.TrimSpace(state.HelpText)
	popupContent := strings.TrimSpace(state.HelpPopupText)
	if popupContent != "" {
		helpContent = popupContent
	} else if helpContent == "" {
		helpContent = "<help is not configured>"
	}
	if state.HelpVisible {
		helpWidthLimit := panelWidth - 2
		if helpWidthLimit < 1 {
			helpWidthLimit = 1
		}
		helpContent = renderHelpMarkdown(helpContent, helpWidthLimit, state.NoColor)
	}

	infoContent := strings.TrimSpace(state.InfoPopupText)
	if infoContent != "" {
		infoContent = renderHelpMarkdown(infoContent, panelWidth-2, state.NoColor)
	}

	mainHeight := state.WinHeight - bottomHeight - statusPanelHeight - inputPanelHeight
	if mainHeight < 3 {
		mainHeight = 3
	}

	infoPanelHeight := 0
	if infoContent != "" {
		infoLines := len(strings.Split(infoContent, "\n"))
		infoPanelHeight = infoLines + 2
		if infoPanelHeight > mainHeight-3 {
			infoPanelHeight = intMax(3, mainHeight-3)
		}
	}
	remainingHeight := mainHeight - infoPanelHeight
	if remainingHeight < 3 {
		remainingHeight = 3
	}

	minDataPanelHeight := 3
	if state.HelpVisible {
		minDataPanelHeight = 1
	}

	helpPanelHeight := 0
	if state.HelpVisible {
		helpLines := len(strings.Split(helpContent, "\n"))
		helpPanelHeight = helpLines + 2 // borders + content
		if helpPanelHeight > remainingHeight-minDataPanelHeight {
			helpPanelHeight = intMax(3, remainingHeight-minDataPanelHeight)
		}
	}
	dataPanelHeight := remainingHeight - helpPanelHeight
	if dataPanelHeight > remainingHeight {
		dataPanelHeight = remainingHeight
	}
	if dataPanelHeight < minDataPanelHeight {
		dataPanelHeight = minDataPanelHeight
	}

	th := CurrentTheme()
	panelBorder := borderForTheme(th)
	highlightRows := true

	// Render a table of the root data for the first panel (clamped to panel width)
	keyColWidth := state.KeyColWidth
	if keyColWidth <= 0 {
		keyColWidth = DefaultKeyColWidth
	}
	// Reserve the separator width (2 spaces) between columns to align with formatter.RenderTable
	availableForValues := panelWidth - keyColWidth - 4 // allow space for padding without wrapping
	if availableForValues < 10 {
		// Narrow layouts: shrink key column to leave room for values
		keyColWidth = panelWidth / 2
		if keyColWidth < 8 {
			keyColWidth = 8
		}
		availableForValues = panelWidth - keyColWidth - 2
	}
	if availableForValues < 10 {
		availableForValues = 10
	}
	rowCount := state.RowCount
	selectedRow := state.SelectedRow
	if rowCount == 0 {
		selectedRow = 0
	}
	if rowCount > 0 && selectedRow >= rowCount {
		selectedRow = rowCount - 1
	}

	innerPanelWidth := panelWidth - 2
	if innerPanelWidth < 1 {
		innerPanelWidth = 1
	}

	var tableText string
	displayNode := state.DisplayNode
	switch {
	case !isCompositeNode(displayNode):
		tableText = renderScalarBlock(displayNode, innerPanelWidth+2, state.NoColor)
	case state.SearchActive:
		tableText = renderSearchTable(state.SearchResults, keyColWidth, availableForValues, state.NoColor)
		var windowSelected int
		tableText, windowSelected = windowTable(tableText, selectedRow, dataPanelHeight-2)
		if highlightRows {
			tableText = highlightTableRow(tableText, windowSelected, panelWidth-2, state.NoColor)
		}
		// Clamp after highlighting to ensure ANSI codes don't trigger wrapping
		tableText = clampANSITextWidth(tableText, innerPanelWidth+2)
	default:
		// Keep formatter table colors in sync with the current theme for the main view.
		th := CurrentTheme()
		formatter.SetTableTheme(formatter.TableColors{
			HeaderFG:       th.HeaderFG,
			HeaderBG:       th.HeaderBG,
			KeyColor:       th.KeyColor,
			ValueColor:     th.ValueColor,
			SeparatorColor: th.SeparatorColor,
		})
		tableText = formatter.RenderTable(displayNode, state.NoColor, keyColWidth, availableForValues)
		// Clamp to the inner content width (panel width minus borders) to prevent wrapping.
		// Clamp with +2 to preserve all three ellipsis dots that truncate() adds.
		tableText = clampANSITextWidth(tableText, innerPanelWidth+2)
		var windowSelected int
		tableText, windowSelected = windowTable(tableText, selectedRow, dataPanelHeight-2)
		if highlightRows {
			tableText = highlightTableRow(tableText, windowSelected, panelWidth-2, state.NoColor)
		}
		// Final clamp after highlighting so ANSI styling cannot cause wrapping
		tableText = clampANSITextWidth(tableText, innerPanelWidth+2)
	}
	// Clamp to the panel content height so the layout stays within the requested window size
	tableText = clampANSITextHeight(tableText, dataPanelHeight-2)
	// Compute counts for label
	totalRows := rowCount
	dataPanelTitle := strings.TrimSpace(state.Title)
	if dataPanelTitle == "" {
		dataPanelTitle = "kvx"
	}
	dataPanel := strings.TrimRight(panelWithTitle(dataPanelTitle, tableText, panelWidth, dataPanelHeight, panelBorder, state.NoColor), "\n")
	pathLabel := state.PathLabel
	if totalRows > 0 {
		selectedDisplay := selectedRow + 1
		if selectedDisplay < 1 {
			selectedDisplay = 1
		}
		if selectedDisplay > totalRows {
			selectedDisplay = totalRows
		}
		// Include node type in the label (e.g., "map: 1/10")
		typeStr := ""
		if state.Node != nil {
			typeStr = nodeTypeLabel(state.Node)
			if typeStr != "" && typeStr != "any" {
				typeStr += ": "
			} else {
				typeStr = ""
			}
		}
		label := fmt.Sprintf("%s%d/%d", typeStr, selectedDisplay, totalRows)
		dataPanel = addBottomLabel(dataPanel, strings.TrimSpace(pathLabel)+" ", label, state.WinWidth)
	}

	helpPanel := ""
	if state.HelpVisible {
		title := state.HelpTitle
		if strings.TrimSpace(title) == "" {
			title = "Help"
		}
		helpWidth := panelWidth
		if helpContent != "" {
			maxContentWidth := 0
			for _, line := range strings.Split(helpContent, "\n") {
				w := ansiVisibleWidth(line)
				if w > maxContentWidth {
					maxContentWidth = w
				}
			}
			candidate := maxContentWidth + 2          // borders
			titleWidth := ansiVisibleWidth(title) + 4 // borders + spaces
			if titleWidth > candidate {
				candidate = titleWidth
			}
			if candidate > 3 && candidate < helpWidth {
				helpWidth = candidate
			}
		}
		helpPanel = strings.TrimRight(panelWithTitle(title, helpContent, helpWidth, helpPanelHeight, panelBorder, state.NoColor), "\n")
	}
	infoPanel := ""
	if infoPanelHeight > 0 && infoContent != "" {
		infoPanel = strings.TrimRight(panelWithTitle("Info", infoContent, panelWidth, infoPanelHeight, panelBorder, state.NoColor), "\n")
	}

	statusPanel := ""
	if state.InputVisible {
		msg := wrapPlainText(strings.TrimRight(state.InfoMessage, "\n"), statusPanelWidth)
		lines := strings.Split(msg, "\n")
		if len(lines) < statusPanelHeight {
			for len(lines) < statusPanelHeight {
				lines = append(lines, "")
			}
		}
		style := lipgloss.NewStyle()
		if !state.NoColor {
			theme := CurrentTheme()
			if state.InfoError {
				style = style.Foreground(theme.StatusError)
			} else {
				style = style.Foreground(theme.StatusColor)
			}
		}
		for i, line := range lines {
			lines[i] = padANSIToWidth(style.Render(line), statusPanelWidth)
		}
		statusPanel = strings.Join(lines, "\n")
	} else {
		msg := wrapPlainText(strings.TrimRight(state.InfoMessage, "\n"), statusPanelWidth)
		lines := strings.Split(msg, "\n")
		if len(lines) < statusPanelHeight {
			for len(lines) < statusPanelHeight {
				lines = append(lines, "")
			}
		}
		style := lipgloss.NewStyle()
		if !state.NoColor {
			theme := CurrentTheme()
			if state.InfoError {
				style = style.Foreground(theme.StatusError)
			} else {
				style = style.Foreground(theme.StatusSuccess)
			}
		}
		for i, line := range lines {
			styled := style.Render(line)
			msgWidth := ansiVisibleWidth(styled)
			if msgWidth > statusPanelWidth {
				styled = runewidth.Truncate(styled, statusPanelWidth, "")
				msgWidth = ansiVisibleWidth(styled)
			}
			padding := statusPanelWidth - msgWidth
			if padding < 0 {
				padding = 0
			}
			lines[i] = strings.Repeat(" ", padding) + styled
		}
		statusPanel = strings.Join(lines, "\n")
	}

	// Input panel spans full width with a border; set input width to the inner content width.
	innerInputWidth := panelWidth
	if innerInputWidth < 1 {
		innerInputWidth = 1
	}
	contentInputWidth := innerInputWidth - 2 // account for borders inside panelWithTitle
	if contentInputWidth < 1 {
		contentInputWidth = 1
	}
	inputPanel := ""
	if state.InputVisible {
		inputTitle := "Input"
		switch {
		case state.ExprMode:
			inputTitle = "Expression"
		case state.SearchActive:
			inputTitle = "Search"
		case state.MapFilterActive:
			inputTitle = "Filter"
		}
		// Build input view with optional ghosted completion appended.
		targetContentWidth := contentInputWidth
		var inputView string
		var inputValue string
		prompt := "❯ "
		if state.SearchActive {
			prompt = searchPrompt(state.NoColor)
		} else if state.MapFilterActive {
			prompt = "🔍 " // Filter prompt
		}
		promptWidth := ansiVisibleWidth(prompt)
		if promptWidth < 1 {
			promptWidth = 0
		}
		bodyWidth := targetContentWidth - promptWidth
		if bodyWidth < 1 {
			bodyWidth = 1
		}
		if state.Input != nil {
			state.Input.SetWidth(bodyWidth)
			state.Input.SetCursor(state.Input.Position())
			inputValue = state.Input.Value()
			inputView = state.Input.View()
			if state.NoColor {
				inputView = ansiRegexp.ReplaceAllString(inputView, "")
			}
			if inputValue != "" {
				desiredWidth := runewidth.StringWidth(inputValue)
				if state.ExprMode || state.SearchActive || state.MapFilterActive {
					desiredWidth++
				}
				inputView = clampANSITextWidth(inputView, desiredWidth)
			}
			inputView = strings.TrimRight(inputView, " ")
		}

		// textinput.View() returns just the value (no prompt since we disabled it)
		body := inputView

		// Truncate body if needed
		if ansiVisibleWidth(body) > bodyWidth {
			// For empty search input, keep the hint intact (trim on the right only).
			if strings.TrimSpace(inputValue) == "" || state.InputPreferHead {
				body = clampANSITextWidth(body, bodyWidth)
			} else {
				body = leftTruncateANSI(body, bodyWidth)
			}
		}

		// Add prompt to the body
		inputView = prompt + body
		if ansiVisibleWidth(inputView) < targetContentWidth {
			inputView = padANSIToWidth(inputView, targetContentWidth)
		}
		inputPanel = strings.TrimRight(panelWithTitle(inputTitle, inputView, innerInputWidth, inputPanelHeight, panelBorder, state.NoColor), "\n")
	}

	// Combine all panels vertically
	normalizePanel := func(panel string, target int) string {
		if target <= 0 || strings.TrimSpace(panel) == "" {
			return ""
		}
		lines := strings.Split(strings.TrimRight(panel, "\n"), "\n")
		if len(lines) > target && target >= 2 {
			top := lines[0]
			bottom := lines[len(lines)-1]
			mid := lines[1 : len(lines)-1]
			if len(mid) > target-2 {
				headCount := target - 3
				if headCount < 0 {
					headCount = 0
				}
				head := mid
				if headCount < len(mid) {
					head = mid[:headCount]
				}
				tail := []string{}
				if len(mid) > 0 {
					tail = []string{mid[len(mid)-1]}
				}
				combined := make([]string, 0, len(head)+len(tail))
				combined = append(combined, head...)
				combined = append(combined, tail...)
				mid = combined
			}
			lines = append([]string{top}, append(mid, bottom)...)
		} else if len(lines) > target {
			lines = lines[:target]
		}
		for len(lines) < target {
			width := runewidth.StringWidth(lines[0])
			if width < 2 {
				width = 2
			}
			lines = append(lines, "│"+strings.Repeat(" ", width-2)+"│")
		}
		return strings.Join(lines, "\n")
	}

	dataPanel = normalizePanel(dataPanel, dataPanelHeight)
	helpPanel = normalizePanel(helpPanel, helpPanelHeight)
	infoPanel = normalizePanel(infoPanel, infoPanelHeight)
	// Status panel is borderless; ensure content is trimmed and optional.
	statusPanel = strings.TrimRight(statusPanel, "\n")
	if statusPanelHeight <= 0 {
		statusPanel = ""
	}
	inputPanel = normalizePanel(inputPanel, inputPanelHeight)

	var topLines []string
	split := func(block string) []string {
		if strings.TrimSpace(block) == "" {
			return []string{}
		}
		return strings.Split(strings.TrimRight(block, "\n"), "\n")
	}

	topLines = split(infoPanel)
	helpLines := split(helpPanel)
	dataLines := split(dataPanel)
	p3Lines := split(statusPanel)
	mainLines := append(append(helpLines, dataLines...), p3Lines...)
	inputLines := split(inputPanel)
	bottomWidth := state.WinWidth
	if bottomWidth < 1 {
		bottomWidth = 1
	}

	// Build footer unless hidden
	var bottomLines []string
	if !state.HideFooter {
		leftFooter := renderFooter(state.NoColor, state.AllowEditInput, bottomWidth, state.KeyMode)
		if strings.TrimSpace(leftFooter) == "" {
			leftFooter = "F1 help"
		}
		rightFooter := ""

		// Preserve debug dimensions on the right when enabled
		if state.DebugEnabled {
			rightFooter = fmt.Sprintf("Rows: %d  Cols: %d", state.WinHeight, state.WinWidth)
		}
		// Ensure the right footer remains visible by truncating the menu when space is tight.
		if rightFooter != "" {
			maxLeft := bottomWidth - ansiVisibleWidth(rightFooter) - 1
			if maxLeft < 0 {
				maxLeft = 0
			}
			if ansiVisibleWidth(leftFooter) > maxLeft {
				leftFooter = clampANSITextWidth(leftFooter, maxLeft)
			}
		}
		space := bottomWidth - ansiVisibleWidth(leftFooter) - ansiVisibleWidth(rightFooter)
		if space < 1 {
			space = 1
		}
		bottomLine := leftFooter + strings.Repeat(" ", space) + rightFooter
		if ansiVisibleWidth(bottomLine) < bottomWidth {
			bottomLine = padANSIToWidth(bottomLine, bottomWidth)
		} else if ansiVisibleWidth(bottomLine) > bottomWidth {
			bottomLine = runewidth.Truncate(bottomLine, bottomWidth, "")
		}
		footerStyle := lipgloss.NewStyle()
		if !state.NoColor {
			footerStyle = footerStyle.Foreground(CurrentTheme().FooterFG).Background(CurrentTheme().FooterBG)
		}
		bottomLines = []string{footerStyle.Render(bottomLine)}
	}

	headerLines := []string{}
	target := state.WinHeight
	if target <= 0 {
		target = len(topLines) + len(mainLines) + len(inputLines) + len(bottomLines)
	}

	// Ensure input and bottom are always kept; trim main if needed.
	remainingForMain := target - len(topLines) - len(inputLines) - len(bottomLines)
	if remainingForMain < 0 {
		remainingForMain = 0
	}
	if len(mainLines) > remainingForMain {
		keepTail := statusPanelHeight
		if keepTail > len(mainLines) {
			keepTail = len(mainLines)
		}
		if remainingForMain <= keepTail {
			mainLines = mainLines[len(mainLines)-keepTail:]
		} else {
			head := remainingForMain - keepTail
			mainLines = append(mainLines[:head], mainLines[len(mainLines)-keepTail:]...)
		}
	}

	lines := append([]string{}, topLines...)
	lines = append(lines, mainLines...)
	lines = append(lines, inputLines...)
	lines = append(lines, bottomLines...)
	if len(headerLines) > 0 {
		lines = append(headerLines, lines...)
	}
	// Pad or trim to match target height.
	for len(lines) < target {
		lines = append(lines, " ")
	}
	if len(lines) > target && target > 0 {
		lines = lines[:target]
	}

	return strings.Join(lines, "\n")
}
