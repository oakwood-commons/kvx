package cmd

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	rdebug "runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/go-logr/logr"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"

	"github.com/oakwood-commons/kvx/internal/formatter"
	"github.com/oakwood-commons/kvx/internal/limiter"
	"github.com/oakwood-commons/kvx/internal/navigator"
	"github.com/oakwood-commons/kvx/internal/ui"
	"github.com/oakwood-commons/kvx/pkg/core"
	"github.com/oakwood-commons/kvx/pkg/loader"
	"github.com/oakwood-commons/kvx/pkg/logger"
	"github.com/oakwood-commons/kvx/pkg/tui"
)

// errShowHelp is returned by loadInputData when no input is provided and help should be shown.
var errShowHelp = errors.New("no input provided")

func init() {
	// Custom help: prepend about text; if interactive + --help, open TUI help.
	defaultHelp := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		// Only run custom help when help was explicitly requested (flag or help command).
		helpFlag := cmd.Flags().Lookup("help")
		helpRequested := (helpFlag != nil && helpFlag.Changed) || cmd.CalledAs() == "help" || cmd.Name() == "help"
		if !helpRequested {
			defaultHelp(cmd, args)
			return
		}
		// Check if interactive flag is set by looking at the flag value directly
		// This works even if flags haven't been fully parsed yet
		// Also check raw args in case --help comes before -i in the command line
		isInteractive := false
		// First, check if -i or --interactive appears in the raw args
		rawArgs := os.Args
		for i, arg := range rawArgs {
			if arg == "-i" || arg == "--interactive" {
				isInteractive = true
				break
			}
			// Also check for short form combined with other flags (e.g., "-ih" becomes "-i -h")
			if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && len(arg) > 1 {
				if strings.Contains(arg, "i") {
					// Check if 'i' is a standalone flag (not part of another flag)
					// This handles cases like "-ih" where i and h are separate flags
					for _, char := range arg[1:] {
						if char == 'i' {
							isInteractive = true
							break
						}
					}
				}
			}
			// Stop checking once we hit the command name or file argument
			if i > 0 && !strings.HasPrefix(arg, "-") {
				break
			}
		}
		// Fallback: check the flag value if flags have been parsed
		if !isInteractive {
			interactiveFlag := cmd.Flags().Lookup("interactive")
			if interactiveFlag != nil {
				if val, err := cmd.Flags().GetBool("interactive"); err == nil {
					isInteractive = val
				}
			}
		}
		// Also check the package variable as final fallback
		if !isInteractive {
			isInteractive = interactive
		}
		if isInteractive {
			// Defer to normal interactive flow but seed the F1 help key.
			startKeys = append([]string{"<F1>"}, startKeys...)
			if cmd.Run != nil {
				limitRecords = 0
				offsetRecords = 0
				tailRecords = 0
				cmd.Run(cmd, cmd.Flags().Args())
			}
			return
		}
		fmt.Fprint(cmd.OutOrStdout(), helpAboutHeader())
		defaultHelp(cmd, args)
	})
}

type snapshotSize struct {
	Width          int
	Height         int
	DetectedWidth  int
	DetectedHeight int
}

func resolveSnapshotSize(flagWidth, flagHeight, detectedWidth, detectedHeight int) snapshotSize {
	width := flagWidth
	height := flagHeight
	usedDetectW := detectedWidth
	usedDetectH := detectedHeight

	if width <= 0 || height <= 0 {
		if usedDetectW <= 0 || usedDetectH <= 0 {
			if w, h := detectTerminalSize(); w > 0 || h > 0 {
				if usedDetectW <= 0 {
					usedDetectW = w
				}
				if usedDetectH <= 0 {
					usedDetectH = h
				}
			}
		}
		if width <= 0 && usedDetectW > 0 {
			width = usedDetectW
		}
		if height <= 0 && usedDetectH > 0 {
			height = usedDetectH
		}
	}

	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}

	return snapshotSize{
		Width:          width,
		Height:         height,
		DetectedWidth:  usedDetectW,
		DetectedHeight: usedDetectH,
	}
}

func renderSnapshotView(renderRoot interface{}, root interface{}, appName, helpTitle, helpText string, startKeys []string, initialExpr string, noColor bool, sizing snapshotSize, configure func(*ui.Model)) string {
	helpVisible := snapshotHelpVisible(startKeys)
	return ui.RenderModelSnapshot(renderRoot, ui.ModelSnapshotConfig{
		Width:       sizing.Width,
		Height:      sizing.Height,
		NoColor:     noColor,
		HelpVisible: helpVisible,
		StartKeys:   startKeys,
		InitialExpr: initialExpr,
		Configure:   configure,
		Root:        root,
		AppName:     appName,
		HelpTitle:   helpTitle,
		HelpText:    helpText,
	})
}

func snapshotHelpVisible(keys []string) bool {
	for _, k := range keys {
		if strings.EqualFold(k, "<f1>") || strings.EqualFold(k, "f1") {
			return true
		}
	}
	return false
}

var (
	interactive     bool
	output          string // for rootCmd (default: table)
	configOutput    string // for configCmd (default: yaml)
	expression      string
	searchTerm      string
	themeName       string
	configFile      string
	configMode      bool
	debug           bool
	noColor         bool
	arrayStyle      string // index, numbered, bullet, none
	renderSnapshot  bool
	helpInteractive bool //nolint:unused // preserved for tests
	startKeys       []string
	snapshotWidth   int
	snapshotHeight  int
	debugMaxEvents  int
	limitRecords    int
	offsetRecords   int
	tailRecords     int
	sortOrder       string
	schemaFile      string
	keyMode         string // empty = use config, "vim"/"emacs"/"function" = override

	// Tree output options
	treeNoValues     bool
	treeMaxDepth     int
	treeExpandArrays bool
	treeMaxStringLen int // 0 = auto; -1 = explicit unlimited; >0 = explicit limit

	// Mermaid output options
	mermaidDirection string

	// Decode options
	autoDecode string // "" = manual only, "lazy" = on navigate, "eager" = at load
)

var (
	stdinIsPiped     = func() bool { stat, _ := os.Stdin.Stat(); return (stat.Mode() & os.ModeCharDevice) == 0 }
	stdoutIsPiped    = func() bool { stat, _ := os.Stdout.Stat(); return (stat.Mode() & os.ModeCharDevice) == 0 }
	openTerminalIOFn = openTerminalIO
	termGetSize      = term.GetSize
	newResizeTicker  = func(d time.Duration) resizeTicker { return realResizeTicker{Ticker: time.NewTicker(d)} }
	sendWindowSize   = func(p *tea.Program, msg tea.WindowSizeMsg) { p.Send(msg) }
)

type resizeTicker interface {
	C() <-chan time.Time
	Stop()
}

type realResizeTicker struct {
	*time.Ticker
}

func (t realResizeTicker) C() <-chan time.Time { return t.Ticker.C }

type debugCollector struct {
	enabled   bool
	events    []ui.DebugEvent
	maxEvents int // Maximum number of events to keep (0 = unlimited, but should be set)
}

func newDebugCollector(enabled bool, maxEvents int) *debugCollector {
	if maxEvents <= 0 {
		maxEvents = 200 // Default to 200 if not specified
	}
	return &debugCollector{
		enabled:   enabled,
		maxEvents: maxEvents,
	}
}

func (d *debugCollector) Printf(format string, args ...interface{}) {
	if !d.enabled {
		return
	}
	d.record(fmt.Sprintf(format, args...))
}

func (d *debugCollector) Println(args ...interface{}) {
	if !d.enabled {
		return
	}
	d.record(fmt.Sprintln(args...))
}

// Append records a preformatted debug message (without emitting immediately).
func (d *debugCollector) Append(msg string) {
	if !d.enabled {
		return
	}
	d.record(msg)
}

func (d *debugCollector) Writer() io.Writer {
	if !d.enabled {
		return io.Discard
	}
	return d
}

func (d *debugCollector) Flush() {
	if !d.enabled {
		return
	}
	// no-op: events are printed via printDebugEvents for ordering
}

// Write implements io.Writer so components can log into the collector.
func (d *debugCollector) Write(p []byte) (int, error) {
	if !d.enabled {
		return len(p), nil
	}
	d.record(string(p))
	return len(p), nil
}

func (d *debugCollector) record(msg string) {
	msg = strings.TrimRight(msg, "\n")
	d.events = append(d.events, ui.DebugEvent{Time: time.Now(), Message: msg})
	// Keep a reasonable cap to avoid unbounded growth (keep last maxEvents)
	if d.maxEvents > 0 && len(d.events) > d.maxEvents {
		d.events = d.events[len(d.events)-d.maxEvents:]
	}
}

func isAlphaNumOrUnderscore(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '_'
}

var rootCtx = context.Background()

func printDebugEvents(events []ui.DebugEvent) {
	if len(events) == 0 {
		return
	}
	sorted := append([]ui.DebugEvent(nil), events...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Time.Before(sorted[j].Time)
	})
	lgr := logger.FromContext(rootCtx)
	for i, ev := range sorted {
		lgr.Info(ev.Message, "debug_index", i+1, "event_time", ev.Time.Format(time.RFC3339Nano))
	}
}

// parseCSV converts CSV data into an array of objects, where each row is an object
// with column headers as keys. This makes CSV data explorable as key-value pairs.
// The root context is `_` which contains the array of row objects.
func parseCSV(data []byte) (interface{}, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}
	if len(records) == 0 {
		return []interface{}{}, nil
	}
	// First row contains headers
	headers := records[0]
	// Convert each data row to a map with column headers as keys
	rows := make([]interface{}, 0, len(records)-1)
	for i := 1; i < len(records); i++ {
		row := make(map[string]interface{})
		for j, header := range headers {
			value := ""
			if j < len(records[i]) {
				value = records[i][j]
			}
			row[header] = value
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// isCSVFile checks if a file path appears to be a CSV file based on extension.
func isCSVFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".csv"
}

// loadInputData reads input data from a file or stdin (or defaults to "{}").
// It returns the parsed root object and whether stdin was used.
// The logger is forwarded to the loader so fallback parse attempts are logged.
func loadInputData(args []string, expr string, debugLog bool, dc *debugCollector, lgr logr.Logger) (interface{}, bool, error) {
	var data []byte
	var fromStdin bool
	var err error

	if len(args) == 0 {
		// No file argument - check if stdin is piped or if expression provided
		stat, _ := os.Stdin.Stat()
		isPiped := (stat.Mode() & os.ModeCharDevice) == 0

		switch {
		case !isPiped && expr != "":
			// If expression provided but no stdin piped, allow evaluation with empty data
			if debugLog {
				dc.Println("DBG: No input data, evaluating expression with empty context")
			}
			data = []byte("{}")
		case !isPiped && expr == "":
			// No input and no expression: signal to show help.
			if debugLog {
				dc.Println("DBG: No input provided; showing help")
			}
			return nil, false, errShowHelp
		default:
			// Read from stdin
			if debugLog {
				dc.Println("DBG: Reading from stdin...")
			}
			data, err = io.ReadAll(os.Stdin)
			if err != nil {
				return nil, false, fmt.Errorf("failed to read from stdin: %w", err)
			}
			if len(data) == 0 {
				if debugLog {
					dc.Println("DBG: No input provided; defaulting to empty object")
				}
				data = []byte("{}")
			}
			fromStdin = true
			if debugLog {
				dc.Printf("DBG: Read %d bytes from stdin\n", len(data))
			}
		}
	} else {
		// File argument provided
		filePath := args[0]
		if debugLog {
			dc.Printf("DBG: Reading file: %s\n", filePath)
		}
		data, err = os.ReadFile(filePath)
		if err != nil {
			return nil, false, fmt.Errorf("failed to read file %s: %w", filePath, err)
		}
		if debugLog {
			dc.Printf("DBG: Read %d bytes from file\n", len(data))
		}
	}

	var root interface{}
	var filePath string
	if len(args) > 0 {
		filePath = args[0]
	}

	// Check if this is a CSV file (by extension for files, or by content for stdin)
	isCSV := false
	if filePath != "" {
		isCSV = isCSVFile(filePath)
	} else if fromStdin {
		// For stdin, try to detect CSV by attempting to parse it
		// CSV typically has comma-separated values with multiple columns
		reader := csv.NewReader(bytes.NewReader(data))
		firstRow, err := reader.Read()
		if err == nil && len(firstRow) > 1 {
			// If first row has multiple columns, it's likely CSV
			// Verify by checking if YAML parsing would fail
			var testYAML interface{}
			if yaml.Unmarshal(data, &testYAML) != nil {
				// YAML parsing failed, so it's likely CSV
				isCSV = true
			} else {
				// YAML parsing succeeded, prefer YAML/JSON
				isCSV = false
			}
		}
	}

	if isCSV {
		if debugLog {
			dc.Println("DBG: Parsing as CSV...")
		}
		var err error
		root, err = parseCSV(data)
		if err != nil {
			return nil, fromStdin, fmt.Errorf("failed to parse CSV: %w", err)
		}
		if debugLog {
			dc.Printf("DBG: Parsed successfully, root type: %T\n", root)
		}
		return root, fromStdin, nil
	}

	// Parse using the loader which honours file extensions, applies
	// content heuristics, and falls through remaining parsers on failure.
	if debugLog {
		dc.Println("DBG: Parsing input (auto-detect with fallback)...")
	}
	if filePath != "" {
		root, err = core.LoadFileWithLogger(filePath, lgr)
	} else {
		root, err = core.LoadRootBytesWithLogger(data, lgr)
	}
	if err != nil {
		return nil, fromStdin, fmt.Errorf("failed to parse input: %w", err)
	}
	if debugLog {
		switch v := root.(type) {
		case []interface{}:
			dc.Printf("DBG: Parsed %d document(s)\n", len(v))
		default:
			dc.Println("DBG: Parsed 1 document")
		}
	}
	return root, fromStdin, nil
}

// escapeCSVField escapes a CSV field according to RFC 4180.
// Fields are quoted if they contain:
//   - Commas (required by RFC 4180)
//   - Double quotes (required by RFC 4180)
//   - Line breaks (newlines/carriage returns, required by RFC 4180)
//   - Spaces (common practice for readability, not required by RFC 4180)
//
// When a field is quoted, any double quotes inside are escaped by doubling them.
func escapeCSVField(field string) string {
	needsQuoting := strings.Contains(field, ",") ||
		strings.Contains(field, "\"") ||
		strings.Contains(field, "\n") ||
		strings.Contains(field, "\r") ||
		strings.Contains(field, " ") // Quote spaces for readability (not RFC required)

	if needsQuoting {
		// RFC 4180: Escape double quotes by doubling them
		escaped := strings.ReplaceAll(field, `"`, `""`)
		return `"` + escaped + `"`
	}
	return field
}

// formatAsCSV converts data to CSV format
func formatAsCSV(node interface{}) string {
	var buf bytes.Buffer
	// We'll write CSV manually to have full control over quoting

	writeCSVRow := func(fields []string) {
		for i, field := range fields {
			if i > 0 {
				buf.WriteString(",")
			}
			buf.WriteString(escapeCSVField(field))
		}
		buf.WriteString("\n")
	}

	switch v := node.(type) {
	case []interface{}:
		if len(v) == 0 {
			return ""
		}
		// Check if it's an array of objects (maps)
		firstElem := v[0]
		if _, ok := firstElem.(map[string]interface{}); ok {
			// Array of objects: use object keys as headers
			// Get all unique keys from all objects
			keySet := make(map[string]bool)
			for _, elem := range v {
				if obj, ok := elem.(map[string]interface{}); ok {
					for k := range obj {
						keySet[k] = true
					}
				}
			}
			// Sort keys for consistent output
			keys := make([]string, 0, len(keySet))
			for k := range keySet {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			// Write header
			writeCSVRow(keys)

			// Write rows
			for _, elem := range v {
				if obj, ok := elem.(map[string]interface{}); ok {
					row := make([]string, len(keys))
					for i, key := range keys {
						val := ""
						if v, ok := obj[key]; ok {
							val = formatter.Stringify(v)
						}
						row[i] = val
					}
					writeCSVRow(row)
				}
			}
		} else {
			// Simple array: single column
			writeCSVRow([]string{"value"})
			for _, elem := range v {
				val := formatter.Stringify(elem)
				writeCSVRow([]string{val})
			}
		}
	case map[string]interface{}:
		// Map: output as key,value pairs
		writeCSVRow([]string{"key", "value"})
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			val := formatter.Stringify(v[k])
			writeCSVRow([]string{k, val})
		}
	default:
		// Scalar: single value
		writeCSVRow([]string{"value"})
		val := formatter.Stringify(node)
		writeCSVRow([]string{val})
	}

	return buf.String()
}

// renderBorderedTable creates a bordered table view without the shell panels.
// Shows just the table with top border (title), data, and bottom border (footer).
// widthHint allows callers to respect --width flags when stdout size is not the desired width.
// path shows the current CEL path in the footer (left side)
func renderBorderedTable(node interface{}, noColor bool, keyColWidth, valueColWidth int, widthHint int, appName string, path string) string {
	return renderBorderedTableWithOptions(node, noColor, keyColWidth, valueColWidth, widthHint, appName, path, formatter.DefaultTableFormatOptions())
}

// renderBorderedTableWithOptions renders a KEY/VALUE table with array style and columnar options.
func renderBorderedTableWithOptions(node interface{}, noColor bool, keyColWidth, valueColWidth int, widthHint int, appName string, path string, tableOpts formatter.TableFormatOptions) string {
	termWidth := widthHint
	if termWidth <= 0 {
		w, _ := detectTerminalSize()
		if w <= 0 {
			termWidth = defaultFallbackTermWidth // fallback to generous width
		} else {
			termWidth = w
		}
	}

	// Get rows to calculate natural width
	rowOpts := navigator.DefaultRowOptions()
	if tableOpts.ArrayStyle != "" {
		rowOpts.ArrayStyle = tableOpts.ArrayStyle
	}
	rows := navigator.NodeToRowsWithOptions(node, rowOpts)

	// Calculate natural content width (key + sep + value)
	naturalContentWidth := formatter.CalculateNaturalTableWidth(rows)
	// Add 2 for side borders
	naturalTableWidth := naturalContentWidth + 2

	// Use the smaller of natural width or terminal width
	tableWidth := termWidth
	fitContent := false
	if naturalTableWidth < termWidth {
		tableWidth = naturalTableWidth
		fitContent = true
	}

	// Calculate footer width requirements: ╰ path ─ type: count ╯
	typeLabel := cliNodeTypeLabel(node)
	if typeLabel == "any" {
		typeLabel = ""
	} else if typeLabel != "" {
		typeLabel += ": "
	}
	visibleRows := len(rows)
	selectedRow := 1
	pathWithSpace := fmt.Sprintf(" %s ", path)
	countWithSpace := fmt.Sprintf(" %s%d/%d ", typeLabel, selectedRow, visibleRows)
	minFooterWidth := lipgloss.Width(pathWithSpace) + lipgloss.Width(countWithSpace) + 3 // +2 corners +1 min dash

	// Ensure minimum width for title/footer visibility
	minTableWidth := lipgloss.Width(appName) + 6 // app name + corners + padding
	if minFooterWidth > minTableWidth {
		minTableWidth = minFooterWidth
	}
	if tableWidth < minTableWidth {
		tableWidth = minTableWidth
	}

	// Render table content with the fitted width
	var tableView string
	if fitContent {
		// Use fit-to-content rendering for narrow data
		th := ui.CurrentTheme()
		formatter.SetTableTheme(formatter.TableColors{
			HeaderFG:       th.HeaderFG,
			HeaderBG:       th.HeaderBG,
			KeyColor:       th.KeyColor,
			ValueColor:     th.ValueColor,
			SeparatorColor: th.SeparatorColor,
		})
		tableView = formatter.RenderTableFitContent(rows, noColor, tableWidth-2)
	} else {
		// Use standard layout-based rendering for wide data
		tableView = renderTableFromNode(node, noColor, keyColWidth, valueColWidth, tableWidth, tableOpts)
	}

	// Get theme colors for header and footer
	theme := ui.CurrentTheme()

	// Create title for top border - centered "╭─ appName ─────────────────────╮"
	borderChar := "─"

	// Center the app name
	availableWidth := tableWidth - 4 // -4 for "╭─" and "─╮"
	titleText := appName
	leftDashes := (availableWidth - len(titleText)) / 2
	rightDashes := availableWidth - len(titleText) - leftDashes
	topBorderLine := fmt.Sprintf("╭%s %s %s╮",
		strings.Repeat(borderChar, leftDashes),
		titleText,
		strings.Repeat(borderChar, rightDashes))

	// Apply theme colors to top border (foreground only, no background)
	var borderStyle lipgloss.Style
	if !noColor && theme.SeparatorColor != nil {
		borderStyle = lipgloss.NewStyle().
			Foreground(theme.SeparatorColor)
		topBorderLine = borderStyle.Render(topBorderLine)
	}

	// Calculate dashes needed between path and count (footer vars already computed above)
	dashes := tableWidth - lipgloss.Width(pathWithSpace) - lipgloss.Width(countWithSpace) - 2 // -2 for corners
	if dashes < 0 {
		dashes = 0
	}

	// Build bottom border with theme colors applied to match the TUI footer label styling.
	var bottomBorderLine string
	if !noColor {
		leftText := pathWithSpace
		rightText := countWithSpace
		if theme.StatusColor != nil {
			leftText = lipgloss.NewStyle().Foreground(theme.StatusColor).Render(leftText)
		}
		if theme.StatusSuccess != nil {
			rightText = lipgloss.NewStyle().Foreground(theme.StatusSuccess).Render(rightText)
		}
		leftCorner := "╰"
		rightCorner := "╯"
		dashedLine := strings.Repeat(borderChar, dashes)
		if theme.SeparatorColor != nil {
			leftCorner = borderStyle.Render(leftCorner)
			rightCorner = borderStyle.Render(rightCorner)
			dashedLine = borderStyle.Render(dashedLine)
		}
		bottomBorderLine = leftCorner + leftText + dashedLine + rightText + rightCorner
	} else {
		bottomBorderLine = fmt.Sprintf("╰%s%s%s╯", pathWithSpace, strings.Repeat(borderChar, dashes), countWithSpace)
	}

	// Build the output with borders
	lines := strings.Split(tableView, "\n")
	output := topBorderLine + "\n"

	// Create style for side borders to match the TUI border color (foreground only).
	if !noColor && theme.SeparatorColor != nil {
		borderStyle = lipgloss.NewStyle().
			Foreground(theme.SeparatorColor)
	}

	for i, line := range lines {
		if strings.TrimSpace(line) == "" && i == len(lines)-1 {
			// Skip trailing empty line
			continue
		}
		// Ensure line is exactly tableWidth by padding or trimming
		lineLen := lipgloss.Width(line)
		if lineLen < tableWidth-2 {
			// Pad with spaces to fill the width
			line += strings.Repeat(" ", tableWidth-2-lineLen)
		}

		// Add side borders with theme colors
		borderedLine := "│" + line + "│"
		if !noColor && theme.SeparatorColor != nil {
			// Apply theme color only to the border characters
			leftBorder := borderStyle.Render("│")
			rightBorder := borderStyle.Render("│")
			borderedLine = leftBorder + line + rightBorder
		}
		output += borderedLine + "\n"
	}
	output += bottomBorderLine

	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}
	return output
}

func autoKeyWidthFromRows(rows [][]string, maxPreset int) int {
	if maxPreset <= 0 {
		maxPreset = ui.DefaultKeyColWidth
	}
	maxKey := 0
	for _, row := range rows {
		if len(row) == 0 {
			continue
		}
		w := lipgloss.Width(strings.TrimSpace(row[0]))
		if w > maxKey {
			maxKey = w
		}
	}
	if maxKey <= 0 {
		return maxPreset
	}
	if maxKey > maxPreset {
		maxKey = maxPreset
	}
	if maxKey < 8 {
		maxKey = 8
	}
	return maxKey
}

func renderTableFromRows(rows [][]string, noColor bool, keyColWidth, valueColWidth int, widthHint int) string {
	termWidth := widthHint
	if termWidth <= 0 {
		w, _ := detectTerminalSize()
		if w <= 0 {
			termWidth = defaultFallbackTermWidth // fallback to generous width
		} else {
			termWidth = w
		}
	}
	layout := ui.NewLayoutManager(termWidth, 24)
	autoKey := func(maxPreset int) int {
		return autoKeyWidthFromRows(rows, maxPreset)
	}
	keyW, valueW := layout.CalculateColumnWidths(keyColWidth, valueColWidth, autoKey)
	if valueW > 1 {
		// Formatter tables use a smaller column separator than the TUI table.
		// Shrink value width by 1 to keep bordered CLI output aligned.
		valueW--
	}
	th := ui.CurrentTheme()
	formatter.SetTableTheme(formatter.TableColors{
		HeaderFG:       th.HeaderFG,
		HeaderBG:       th.HeaderBG,
		KeyColor:       th.KeyColor,
		ValueColor:     th.ValueColor,
		SeparatorColor: th.SeparatorColor,
	})
	output := formatter.RenderRows(rows, noColor, keyW, valueW)
	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}
	return output
}

func renderBorderedTableRows(rows [][]string, noColor bool, keyColWidth, valueColWidth int, widthHint int, appName string, path string, nodeForType interface{}) string {
	termWidth := widthHint
	if termWidth <= 0 {
		w, _ := detectTerminalSize()
		if w <= 0 {
			termWidth = defaultFallbackTermWidth // fallback to generous width
		} else {
			termWidth = w
		}
	}

	// Calculate natural content width (key + sep + value)
	naturalContentWidth := formatter.CalculateNaturalTableWidth(rows)
	// Add 2 for side borders
	naturalTableWidth := naturalContentWidth + 2

	// Use the smaller of natural width or terminal width
	tableWidth := termWidth
	if naturalTableWidth < termWidth {
		tableWidth = naturalTableWidth
	}

	// Ensure minimum width for title/footer visibility
	minTableWidth := lipgloss.Width(appName) + 6 // app name + corners + padding
	if tableWidth < minTableWidth {
		tableWidth = minTableWidth
	}

	tableView := renderTableFromRows(rows, noColor, keyColWidth, valueColWidth, tableWidth)

	// Get theme colors for header and footer
	theme := ui.CurrentTheme()

	borderChar := "─"

	availableWidth := tableWidth - 4 // -4 for "╭─" and "─╮"
	titleText := appName
	leftDashes := (availableWidth - len(titleText)) / 2
	rightDashes := availableWidth - len(titleText) - leftDashes
	topBorderLine := fmt.Sprintf("╭%s %s %s╮",
		strings.Repeat(borderChar, leftDashes),
		titleText,
		strings.Repeat(borderChar, rightDashes))

	var borderStyle lipgloss.Style
	if !noColor && theme.SeparatorColor != nil {
		borderStyle = lipgloss.NewStyle().
			Foreground(theme.SeparatorColor)
		topBorderLine = borderStyle.Render(topBorderLine)
	}

	visibleRows := len(rows)
	selectedRow := 0
	if visibleRows > 0 {
		selectedRow = 1
	}

	pathWithSpace := fmt.Sprintf(" %s ", path)
	typeLabel := cliNodeTypeLabel(nodeForType)
	if typeLabel == "any" {
		typeLabel = ""
	} else if typeLabel != "" {
		typeLabel += ": "
	}
	countWithSpace := fmt.Sprintf(" %s%d/%d ", typeLabel, selectedRow, visibleRows)

	dashes := tableWidth - lipgloss.Width(pathWithSpace) - lipgloss.Width(countWithSpace) - 2 // -2 for corners
	if dashes < 0 {
		dashes = 0
	}

	var bottomBorderLine string
	if !noColor {
		leftText := pathWithSpace
		rightText := countWithSpace
		if theme.StatusColor != nil {
			leftText = lipgloss.NewStyle().Foreground(theme.StatusColor).Render(leftText)
		}
		if theme.StatusSuccess != nil {
			rightText = lipgloss.NewStyle().Foreground(theme.StatusSuccess).Render(rightText)
		}
		leftCorner := "╰"
		rightCorner := "╯"
		dashedLine := strings.Repeat(borderChar, dashes)
		if theme.SeparatorColor != nil {
			leftCorner = borderStyle.Render(leftCorner)
			rightCorner = borderStyle.Render(rightCorner)
			dashedLine = borderStyle.Render(dashedLine)
		}
		bottomBorderLine = leftCorner + leftText + dashedLine + rightText + rightCorner
	} else {
		bottomBorderLine = fmt.Sprintf("╰%s%s%s╯", pathWithSpace, strings.Repeat(borderChar, dashes), countWithSpace)
	}

	lines := strings.Split(tableView, "\n")
	output := topBorderLine + "\n"

	if !noColor && theme.SeparatorColor != nil {
		borderStyle = lipgloss.NewStyle().
			Foreground(theme.SeparatorColor)
	}

	for i, line := range lines {
		if strings.TrimSpace(line) == "" && i == len(lines)-1 {
			continue
		}
		lineLen := lipgloss.Width(line)
		if lineLen < tableWidth-2 {
			line += strings.Repeat(" ", tableWidth-2-lineLen)
		}

		borderedLine := "│" + line + "│"
		if !noColor && theme.SeparatorColor != nil {
			leftBorder := borderStyle.Render("│")
			rightBorder := borderStyle.Render("│")
			borderedLine = leftBorder + line + rightBorder
		}
		output += borderedLine + "\n"
	}
	output += bottomBorderLine

	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}
	return output
}

// cliNodeTypeLabel maps a node to a simple CEL-like type label for CLI footer display.
func cliNodeTypeLabel(node interface{}) string {
	switch node.(type) {
	case []interface{}:
		return "list"
	case map[string]interface{}:
		return "map"
	case string:
		return "string"
	case bool:
		return "bool"
	case int, int64:
		return "int"
	case uint, uint64:
		return "uint"
	case float32, float64:
		return "double"
	default:
		return "any"
	}
}

// renderColumnarBorderedTable renders a homogeneous array as a multi-column table with borders.
func renderColumnarBorderedTable(node interface{}, noColor bool, widthHint int, appName string, path string, tableOpts formatter.TableFormatOptions) string {
	termWidth := widthHint
	if termWidth <= 0 {
		w, _ := detectTerminalSize()
		if w <= 0 {
			termWidth = defaultFallbackTermWidth
		} else {
			termWidth = w
		}
	}

	// Extract columnar data
	columns, rows := navigator.ExtractColumnarData(node, tableOpts.ColumnOrder)
	if columns == nil {
		// Fall back to regular table rendering
		return renderBorderedTable(node, noColor, 0, 0, termWidth, appName, path)
	}

	// Calculate natural content width accounting for hidden columns and display name overrides
	showRowNum := tableOpts.ArrayStyle != "none"
	naturalContentWidth := formatter.CalculateNaturalColumnarWidthWithHints(columns, rows, showRowNum, len(rows), tableOpts.ColumnHints, tableOpts.HiddenColumns)

	// Table width: use natural width if it fits, otherwise use terminal width
	tableWidth := termWidth
	if naturalContentWidth+2 < termWidth {
		// Content fits naturally - use fit-to-content width
		tableWidth = naturalContentWidth + 2
	}

	// Ensure minimum width for title/footer visibility
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

	// Set theme colors
	th := ui.CurrentTheme()
	formatter.SetTableTheme(formatter.TableColors{
		HeaderFG:       th.HeaderFG,
		HeaderBG:       th.HeaderBG,
		KeyColor:       th.KeyColor,
		ValueColor:     th.ValueColor,
		SeparatorColor: th.SeparatorColor,
	})

	// Render columnar table (content only, we add borders)
	tableView := formatter.RenderColumnarTable(columns, rows, formatter.ColumnarOptions{
		NoColor:        noColor,
		TotalWidth:     tableWidth - 2, // Content width (accounting for borders)
		RowNumberStyle: tableOpts.ArrayStyle,
		ColumnOrder:    tableOpts.ColumnOrder,
		HiddenColumns:  tableOpts.HiddenColumns,
		ColumnHints:    tableOpts.ColumnHints,
	})

	// Add borders
	theme := ui.CurrentTheme()
	borderChar := "─"

	// Split table view into lines for border wrapping
	lines := strings.Split(tableView, "\n")

	// Top border with app name
	availableWidth := tableWidth - 4
	titleText := appName
	leftDashes := (availableWidth - len(titleText)) / 2
	rightDashes := availableWidth - len(titleText) - leftDashes
	topBorderLine := fmt.Sprintf("╭%s %s %s╮",
		strings.Repeat(borderChar, leftDashes),
		titleText,
		strings.Repeat(borderChar, rightDashes))

	var borderStyle lipgloss.Style
	if !noColor && theme.SeparatorColor != nil {
		borderStyle = lipgloss.NewStyle().Foreground(theme.SeparatorColor)
		topBorderLine = borderStyle.Render(topBorderLine)
	}

	// Footer with path and count (reuse variables computed earlier)
	dashes := tableWidth - lipgloss.Width(pathWithSpace) - lipgloss.Width(countWithSpace) - 2
	if dashes < 0 {
		dashes = 0
	}

	var bottomBorderLine string
	if !noColor {
		leftText := pathWithSpace
		rightText := countWithSpace
		if theme.StatusColor != nil {
			leftText = lipgloss.NewStyle().Foreground(theme.StatusColor).Render(leftText)
		}
		if theme.StatusSuccess != nil {
			rightText = lipgloss.NewStyle().Foreground(theme.StatusSuccess).Render(rightText)
		}
		leftCorner := "╰"
		rightCorner := "╯"
		dashedLine := strings.Repeat(borderChar, dashes)
		if theme.SeparatorColor != nil {
			leftCorner = borderStyle.Render(leftCorner)
			rightCorner = borderStyle.Render(rightCorner)
			dashedLine = borderStyle.Render(dashedLine)
		}
		bottomBorderLine = leftCorner + leftText + dashedLine + rightText + rightCorner
	} else {
		bottomBorderLine = fmt.Sprintf("╰%s%s%s╯", pathWithSpace, strings.Repeat(borderChar, dashes), countWithSpace)
	}

	// Build output with bordered lines (lines already split above)
	output := topBorderLine + "\n"

	for i, line := range lines {
		if strings.TrimSpace(line) == "" && i == len(lines)-1 {
			continue
		}
		lineLen := lipgloss.Width(line)
		if lineLen < tableWidth-2 {
			line += strings.Repeat(" ", tableWidth-2-lineLen)
		}

		borderedLine := "│" + line + "│"
		if !noColor && theme.SeparatorColor != nil {
			leftBorder := borderStyle.Render("│")
			rightBorder := borderStyle.Render("│")
			borderedLine = leftBorder + line + rightBorder
		}
		output += borderedLine + "\n"
	}
	output += bottomBorderLine

	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}
	return output
}

// renderTableFromNode creates a minimal TUI model and renders just the table component.
// This ensures CLI table output uses the same rendering code as snapshot mode, honoring themes.
// widthHint allows callers to respect --width flags when stdout size is not the desired width.
func renderTableFromNode(node interface{}, noColor bool, keyColWidth, valueColWidth int, widthHint int, tableOpts formatter.TableFormatOptions) string {
	// Create a minimal model for table rendering
	m := ui.InitialModel(node)
	m.NoColor = noColor
	m.Root = node
	m.Node = node

	// Set window dimensions first (needed for column width calculation)
	termWidth := widthHint
	if termWidth <= 0 {
		w, _ := detectTerminalSize()
		if w <= 0 {
			termWidth = defaultFallbackTermWidth // fallback to generous width
		} else {
			termWidth = w
		}
	}
	m.WinWidth = termWidth
	m.Layout.SetDimensions(termWidth, 24) // Height doesn't matter for CLI

	// Configure requested widths and calculate actual widths using the layout manager
	if keyColWidth <= 0 {
		// Shrink key column when keys are short to free space for values in CLI output.
		maxKey := 0
		if n, ok := node.(map[string]interface{}); ok {
			for k := range n {
				if w := lipgloss.Width(k); w > maxKey {
					maxKey = w
				}
			}
		}
		if maxKey > 0 {
			minKey := 8
			maxKeyWidth := ui.DefaultKeyColWidth
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
	m.ConfiguredValueColWidth = valueColWidth
	keyW, valueW := m.Layout.CalculateColumnWidths(m.ConfiguredKeyColWidth, m.ConfiguredValueColWidth, m.AutoKeyColumnWidth)
	if valueW > 1 {
		// Formatter tables use a smaller column separator than the TUI table.
		// Shrink value width by 1 to keep bordered CLI output aligned.
		valueW--
	}
	output := tui.RenderTable(node, tui.TableOptions{
		NoColor:       noColor,
		KeyColWidth:   keyW,
		ValueColWidth: valueW,
		Width:         termWidth,
		Bordered:      false,
		ArrayStyle:    tableOpts.ArrayStyle,
		ColumnarMode:  tableOpts.ColumnarMode,
	})
	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}
	return output
}

// effectiveKeyMode returns the KeyMode to use based on CLI flag > env var > config > default.
func effectiveKeyMode(cfg ui.ThemeConfigFile) ui.KeyMode {
	if keyMode != "" && ui.IsValidKeyMode(keyMode) {
		return ui.KeyMode(keyMode)
	}
	if envVal := os.Getenv("KVX_KEY_MODE"); envVal != "" && ui.IsValidKeyMode(envVal) {
		return ui.KeyMode(envVal)
	}
	if cfg.Features.KeyMode != nil && ui.IsValidKeyMode(*cfg.Features.KeyMode) {
		return ui.KeyMode(*cfg.Features.KeyMode)
	}
	return ui.DefaultKeyMode
}

func applySnapshotConfigToModel(m *ui.Model, cfg ui.ThemeConfigFile) {
	if m == nil {
		return
	}
	if cfg.Features.AllowEditInput != nil {
		m.AllowEditInput = *cfg.Features.AllowEditInput
	}
	if cfg.Features.AllowFilter != nil {
		m.AllowFilter = *cfg.Features.AllowFilter
	}
	if cfg.Features.AllowSuggestions != nil {
		m.AllowSuggestions = *cfg.Features.AllowSuggestions
	}
	if cfg.Features.AllowIntellisense != nil {
		m.AllowIntellisense = *cfg.Features.AllowIntellisense
	}
	// Apply key mode: CLI flag > env var > config > default (vim)
	if keyMode != "" && ui.IsValidKeyMode(keyMode) {
		m.KeyMode = ui.KeyMode(keyMode)
	} else if envVal := os.Getenv("KVX_KEY_MODE"); envVal != "" && ui.IsValidKeyMode(envVal) {
		m.KeyMode = ui.KeyMode(envVal)
	} else if cfg.Features.KeyMode != nil && ui.IsValidKeyMode(*cfg.Features.KeyMode) {
		m.KeyMode = ui.KeyMode(*cfg.Features.KeyMode)
	}
	// Note: default is already KeyModeVim from InitialModel
	if cfg.Display.KeyColWidth != nil {
		m.ConfiguredKeyColWidth = *cfg.Display.KeyColWidth
	}
	if len(cfg.Help.FunctionHelp) > 0 {
		m.FunctionHelpOverrides = cfg.Help.FunctionHelp
	}
	if len(cfg.Help.CEL.FunctionExamples) > 0 {
		m.FunctionExamples = cfg.Help.CEL.FunctionExamples
	} else if len(cfg.Help.FunctionExamples) > 0 { // legacy
		m.FunctionExamples = cfg.Help.FunctionExamples
	}
	// Apply performance config
	if cfg.Performance.FilterDebounceMs != nil {
		m.SearchDebounceMs = *cfg.Performance.FilterDebounceMs
	}
	if cfg.Performance.SearchResultLimit != nil {
		m.SearchResultLimit = *cfg.Performance.SearchResultLimit
	}
	if cfg.Performance.ScrollBufferRows != nil {
		m.ScrollBufferRows = *cfg.Performance.ScrollBufferRows
	}
	if cfg.Performance.VirtualScrolling != nil {
		m.VirtualScrolling = *cfg.Performance.VirtualScrolling
	}
	// Apply auto-decode setting from CLI flag
	if autoDecode != "" {
		m.AutoDecode = autoDecode
	}
}

func printEvalResult(node interface{}, output string, noColor bool, keyColWidth, valueColWidth int, _ int, width int, appName string, path string, yamlOpts formatter.YAMLFormatOptions, tableOpts formatter.TableFormatOptions, treeOpts formatter.TreeOptions, mermaidOpts formatter.MermaidOptions) {
	switch output {
	case "table":
		// For scalar values, print the raw value instead of a table
		isCollection, isSimpleArray := classifyNode(node)

		switch {
		case isSimpleArray:
			// Print each scalar element on its own line
			for _, elem := range node.([]interface{}) { //nolint:forcetypeassert
				fmt.Println(formatter.StringifyPreserveNewlines(elem)) //nolint:forbidigo
			}
		case !isCollection:
			fmt.Println(formatter.StringifyPreserveNewlines(node)) //nolint:forbidigo
		default:
			// Check if we should use columnar rendering for homogeneous arrays
			if shouldUseColumnar(node, tableOpts.ColumnarMode) {
				fmt.Print(renderColumnarBorderedTable(node, noColor, width, appName, path, tableOpts)) //nolint:forbidigo
			} else {
				// Non-interactive mode: render bordered table with header and footer
				fmt.Print(renderBorderedTableWithOptions(node, noColor, keyColWidth, valueColWidth, width, appName, path, tableOpts)) //nolint:forbidigo
			}
		}
	case "csv":
		csvOutput := formatAsCSV(node)
		fmt.Print(csvOutput) //nolint:forbidigo
	case "yaml", "raw":
		if s, err := formatter.FormatYAML(node, yamlOpts); err == nil {
			fmt.Print(s) //nolint:forbidigo
		} else {
			fmt.Fprintf(os.Stderr, "failed to marshal yaml: %v\n", err)
			os.Exit(1)
		}
	case "json":
		if b, err := json.MarshalIndent(node, "", "  "); err == nil {
			fmt.Print(string(b) + "\n") //nolint:forbidigo
		} else {
			fmt.Fprintf(os.Stderr, "failed to marshal json: %v\n", err)
			os.Exit(1)
		}
	case "toml":
		if b, err := toml.Marshal(node); err == nil {
			fmt.Print(string(b)) //nolint:forbidigo
		} else {
			fmt.Fprintf(os.Stderr, "failed to marshal toml: %v\n", err)
			os.Exit(1)
		}
	case "auto":
		// Auto mode: use table when readable, fall back to list when columns are too narrow
		isCollection, isSimpleArray := classifyNode(node)

		switch {
		case isSimpleArray:
			for _, elem := range node.([]interface{}) { //nolint:forcetypeassert
				fmt.Println(formatter.StringifyPreserveNewlines(elem)) //nolint:forbidigo
			}
		case !isCollection:
			fmt.Println(formatter.StringifyPreserveNewlines(node)) //nolint:forbidigo
		default:
			if shouldUseColumnar(node, tableOpts.ColumnarMode) {
				// Check if columnar table is readable at current terminal width
				termWidth := width
				if termWidth <= 0 {
					w, _ := detectTerminalSize()
					if w <= 0 {
						termWidth = defaultFallbackTermWidth
					} else {
						termWidth = w
					}
				}

				columns, rows := navigator.ExtractColumnarData(node, tableOpts.ColumnOrder)
				readableOpts := formatter.IsColumnarReadableOpts{
					HiddenColumns:  tableOpts.HiddenColumns,
					RowNumberStyle: tableOpts.ArrayStyle,
				}
				if columns != nil && !formatter.IsColumnarReadable(columns, rows, termWidth-2, tableOpts.ColumnHints, readableOpts) {
					// Table would be unreadable — fall back to list view
					listOpts := formatter.ListOptions{
						NoColor:    noColor,
						ArrayStyle: arrayStyle,
					}
					fmt.Print(formatter.FormatAsList(node, listOpts)) //nolint:forbidigo
				} else {
					fmt.Print(renderColumnarBorderedTable(node, noColor, width, appName, path, tableOpts)) //nolint:forbidigo
				}
			} else {
				fmt.Print(renderBorderedTableWithOptions(node, noColor, keyColWidth, valueColWidth, width, appName, path, tableOpts)) //nolint:forbidigo
			}
		}
	case "list":
		listOpts := formatter.ListOptions{
			NoColor:    noColor,
			ArrayStyle: arrayStyle,
		}
		fmt.Print(formatter.FormatAsList(node, listOpts)) //nolint:forbidigo
	case "tree":
		fmt.Print(formatter.FormatAsTree(node, treeOpts)) //nolint:forbidigo
	case "mermaid":
		fmt.Print(formatter.FormatAsMermaid(node, mermaidOpts)) //nolint:forbidigo
	default:
		fmt.Fprintf(os.Stderr, "invalid output: %s\n", output)
		os.Exit(2)
	}
}

// shouldUseColumnar determines if columnar rendering should be used for the node.
func shouldUseColumnar(node interface{}, mode string) bool {
	switch mode {
	case "never":
		return false
	case "always":
		// Only for arrays
		switch node.(type) {
		case []interface{}:
			return true
		default:
			return false
		}
	default: // "auto"
		// Use columnar only for homogeneous arrays of objects
		isHomogeneous, _ := navigator.IsHomogeneousArray(node)
		return isHomogeneous
	}
}

func yamlFormatOptionsFromConfig(cfg ui.ThemeConfigFile) formatter.YAMLFormatOptions {
	indent := 2
	literal := true
	expandEscaped := false
	if cfg.Formatting.YAML.Indent != nil && *cfg.Formatting.YAML.Indent > 0 {
		indent = *cfg.Formatting.YAML.Indent
	}
	if cfg.Formatting.YAML.LiteralBlockStrings != nil {
		literal = *cfg.Formatting.YAML.LiteralBlockStrings
	}
	if cfg.Formatting.YAML.ExpandEscapedNewlines != nil {
		expandEscaped = *cfg.Formatting.YAML.ExpandEscapedNewlines
	}
	return formatter.YAMLFormatOptions{
		Indent:                indent,
		LiteralBlockStrings:   literal,
		ExpandEscapedNewlines: expandEscaped,
	}
}

func tableFormatOptionsFromConfig(cfg ui.ThemeConfigFile) formatter.TableFormatOptions {
	opts := formatter.DefaultTableFormatOptions()
	if cfg.Formatting.Table.ArrayStyle != nil {
		opts.ArrayStyle = *cfg.Formatting.Table.ArrayStyle
	}
	if cfg.Formatting.Table.ColumnarMode != nil {
		opts.ColumnarMode = *cfg.Formatting.Table.ColumnarMode
	}
	if len(cfg.Formatting.Table.ColumnOrder) > 0 {
		opts.ColumnOrder = cfg.Formatting.Table.ColumnOrder
	}
	if len(cfg.Formatting.Table.HiddenColumns) > 0 {
		opts.HiddenColumns = cfg.Formatting.Table.HiddenColumns
	}

	// Load JSON Schema for column display hints.
	// Priority: CLI --schema flag > config schema_file > config inline schema.
	var schemaHints map[string]tui.ColumnHint
	schemaPath := schemaFile
	if schemaPath == "" && cfg.Formatting.Table.SchemaFile != nil {
		schemaPath = *cfg.Formatting.Table.SchemaFile
	}
	switch {
	case schemaPath != "":
		data, err := os.ReadFile(schemaPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: cannot read schema file %s: %v\n", schemaPath, err)
		} else {
			schemaHints, err = tui.ParseSchema(data)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: cannot parse schema file %s: %v\n", schemaPath, err)
			}
		}
	case len(cfg.Formatting.Table.Schema) > 0:
		data, err := json.Marshal(cfg.Formatting.Table.Schema)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: cannot serialize inline schema: %v\n", err)
		} else {
			schemaHints, err = tui.ParseSchema(data)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: cannot parse inline schema: %v\n", err)
			}
		}
	}
	if len(schemaHints) > 0 {
		opts.ColumnHints = make(map[string]formatter.ColumnHint, len(schemaHints))
		for k, h := range schemaHints {
			opts.ColumnHints[k] = formatter.ColumnHint{
				MaxWidth:    h.MaxWidth,
				Priority:    h.Priority,
				Align:       h.Align,
				DisplayName: h.DisplayName,
			}
			if h.Hidden {
				opts.HiddenColumns = append(opts.HiddenColumns, k)
			}
		}
	}

	return opts
}

// treeFormatOptionsFromConfig builds TreeOptions from config with priority:
// CLI flag > TTY auto-detect > config file > defaults.
// termWidth is passed for TTY-based sizing (0 = use default).
// piped indicates whether stdout is piped (disables truncation by default).
func treeFormatOptionsFromConfig(cfg ui.ThemeConfigFile, termWidth int, piped bool) formatter.TreeOptions {
	opts := formatter.TreeOptions{
		NoValues:       treeNoValues,
		MaxDepth:       treeMaxDepth,
		ExpandArrays:   treeExpandArrays,
		MaxArrayInline: 3,
		MaxStringLen:   0, // 0 = unlimited initially
	}
	// Apply config values for tree (dereference pointers)
	if cfg.Formatting.Tree.MaxDepth != nil && *cfg.Formatting.Tree.MaxDepth > 0 {
		opts.MaxDepth = *cfg.Formatting.Tree.MaxDepth
	}
	if cfg.Formatting.Tree.MaxArrayInline != nil && *cfg.Formatting.Tree.MaxArrayInline > 0 {
		opts.MaxArrayInline = *cfg.Formatting.Tree.MaxArrayInline
	}
	if cfg.Formatting.Tree.ExpandArrays != nil && *cfg.Formatting.Tree.ExpandArrays {
		opts.ExpandArrays = true
	}
	if cfg.Formatting.Tree.NoValues != nil && *cfg.Formatting.Tree.NoValues {
		opts.NoValues = true
	}

	// String length priority: CLI flag > TTY auto > config > default
	configMaxString := 0
	if cfg.Formatting.Tree.MaxStringLength != nil {
		configMaxString = *cfg.Formatting.Tree.MaxStringLength
	}
	switch {
	case treeMaxStringLen != 0:
		// Explicit CLI flag: -1=unlimited, >0=explicit limit
		if treeMaxStringLen > 0 {
			opts.MaxStringLen = treeMaxStringLen
		}
		// -1 or other negative = leave at 0 (unlimited)
	case piped:
		// Piped output: no truncation
		opts.MaxStringLen = 0
	case configMaxString > 0:
		// Config specifies explicit limit
		opts.MaxStringLen = configMaxString
	case termWidth > 0:
		// Terminal mode: MaxStringLen is for VALUE only.
		// Total line = prefix (~4×depth) + key (~15 avg) + ": " + value
		// For typical nesting (depth 2-3), overhead is ~20-30 chars.
		// Use 60% of terminal width for value to avoid wrapping.
		opts.MaxStringLen = (termWidth * 60) / 100
		if opts.MaxStringLen < 40 {
			opts.MaxStringLen = 40
		}
		if opts.MaxStringLen > 200 {
			opts.MaxStringLen = 200
		}
	default:
		// Fallback for interactive without terminal size
		opts.MaxStringLen = 80
	}

	// CLI flags override config
	if treeNoValues {
		opts.NoValues = true
	}
	if treeMaxDepth > 0 {
		opts.MaxDepth = treeMaxDepth
	}
	if treeExpandArrays {
		opts.ExpandArrays = true
	}

	// Load JSON Schema for field hints (reuse schemaFile or config).
	var fieldHints map[string]int
	schemaPath := schemaFile
	if schemaPath == "" && cfg.Formatting.Table.SchemaFile != nil {
		schemaPath = *cfg.Formatting.Table.SchemaFile
	}
	if schemaPath != "" {
		data, err := os.ReadFile(schemaPath)
		if err == nil {
			schemaHints, err := tui.ParseSchema(data)
			if err == nil {
				fieldHints = make(map[string]int, len(schemaHints))
				for k, h := range schemaHints {
					if h.MaxWidth > 0 {
						fieldHints[k] = h.MaxWidth
					}
				}
			}
		}
	}
	if len(fieldHints) > 0 {
		opts.FieldHints = fieldHints
	}

	// Array style from CLI flag
	if arrayStyle != "" {
		opts.ArrayStyle = arrayStyle
	}

	return opts
}

// mermaidFormatOptionsFromConfig builds MermaidOptions from config.
// Reuses tree options for shared settings (depth, no-values, expand-arrays, max-string).
func mermaidFormatOptionsFromConfig(cfg ui.ThemeConfigFile) formatter.MermaidOptions {
	opts := formatter.MermaidOptions{
		Direction:      mermaidDirection,
		NoValues:       treeNoValues,
		MaxDepth:       treeMaxDepth,
		ExpandArrays:   treeExpandArrays,
		MaxArrayInline: 3,
		MaxStringLen:   0, // 0 = unlimited for mermaid (diagrams should have full data by default)
	}

	// Apply config values for mermaid
	if cfg.Formatting.Mermaid.Direction != nil && *cfg.Formatting.Mermaid.Direction != "" {
		opts.Direction = *cfg.Formatting.Mermaid.Direction
	}
	if cfg.Formatting.Mermaid.MaxDepth != nil && *cfg.Formatting.Mermaid.MaxDepth > 0 {
		opts.MaxDepth = *cfg.Formatting.Mermaid.MaxDepth
	}
	if cfg.Formatting.Mermaid.MaxArrayInline != nil && *cfg.Formatting.Mermaid.MaxArrayInline > 0 {
		opts.MaxArrayInline = *cfg.Formatting.Mermaid.MaxArrayInline
	}
	if cfg.Formatting.Mermaid.ExpandArrays != nil && *cfg.Formatting.Mermaid.ExpandArrays {
		opts.ExpandArrays = true
	}
	if cfg.Formatting.Mermaid.NoValues != nil && *cfg.Formatting.Mermaid.NoValues {
		opts.NoValues = true
	}
	if cfg.Formatting.Mermaid.MaxStringLength != nil && *cfg.Formatting.Mermaid.MaxStringLength > 0 {
		opts.MaxStringLen = *cfg.Formatting.Mermaid.MaxStringLength
	}

	// CLI flags override config
	if mermaidDirection != "TD" && mermaidDirection != "" {
		opts.Direction = mermaidDirection
	}
	if treeNoValues {
		opts.NoValues = true
	}
	if treeMaxDepth > 0 {
		opts.MaxDepth = treeMaxDepth
	}
	if treeExpandArrays {
		opts.ExpandArrays = true
	}
	if treeMaxStringLen != 0 {
		if treeMaxStringLen > 0 {
			opts.MaxStringLen = treeMaxStringLen
		}
		// -1 or other negative = leave at 0 (unlimited)
	}

	// Load JSON Schema for field hints (reuse schemaFile or config).
	var fieldHints map[string]int
	schemaPath := schemaFile
	if schemaPath == "" && cfg.Formatting.Table.SchemaFile != nil {
		schemaPath = *cfg.Formatting.Table.SchemaFile
	}
	if schemaPath != "" {
		data, err := os.ReadFile(schemaPath)
		if err == nil {
			schemaHints, err := tui.ParseSchema(data)
			if err == nil {
				fieldHints = make(map[string]int, len(schemaHints))
				for k, h := range schemaHints {
					if h.MaxWidth > 0 {
						fieldHints[k] = h.MaxWidth
					}
				}
			}
		}
	}
	if len(fieldHints) > 0 {
		opts.FieldHints = fieldHints
	}

	// Array style from CLI flag
	if arrayStyle != "" {
		opts.ArrayStyle = arrayStyle
	}

	return opts
}

// classifyNode determines how to render the evaluated node.
// isCollection is true for maps/slices; isSimpleArray is true for []interface{} of scalars.
func classifyNode(node interface{}) (isCollection bool, isSimpleArray bool) {
	switch v := node.(type) {
	case map[string]interface{}:
		return true, false
	case []interface{}:
		if len(v) == 0 {
			return true, false
		}
		allScalars := true
		for _, elem := range v {
			rv := reflect.ValueOf(elem)
			if rv.IsValid() {
				if rv.Kind() == reflect.Map || rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
					allScalars = false
					break
				}
			}
		}
		if allScalars {
			return false, true
		}
		return true, false
	default:
		rv := reflect.ValueOf(node)
		if rv.IsValid() {
			if rv.Kind() == reflect.Map || rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
				return true, false
			}
		}
		return false, false
	}
}

// buildSuggestion returns a helpful hint for CLI --expression errors when the user
// omits the required '_' root variable but the first path token exists at the root.
func buildSuggestion(expr string, root interface{}) string {
	// Only suggest when '_' is missing and the expression looks like a simple path
	if strings.Contains(expr, "_") {
		return ""
	}
	// Special case: root is an array and user started with an index like "[0]"
	if strings.HasPrefix(expr, "[") {
		if _, ok := root.([]interface{}); ok {
			return fmt.Sprintf("Hint: CLI --expression uses '_' as root. Did you mean '_%s'?", expr)
		}
	}
	// Extract first token before dot or bracket
	token := expr
	if idx := strings.IndexAny(expr, ".[\""); idx >= 0 {
		token = expr[:idx]
	}
	if token == "" {
		return ""
	}
	if m, ok := root.(map[string]interface{}); ok {
		if _, exists := m[token]; exists {
			// Preserve rest of the expression after the first token
			suffix := ""
			if len(expr) > len(token) {
				suffix = expr[len(token):]
			}
			// Normalize dotted numeric segments to CEL index notation: .0 -> [0]
			// Also handle chained numeric segments like .0.tags.1 -> [0].tags[1]
			re := regexp.MustCompile(`\.(\d+)`)
			suffix = re.ReplaceAllString(suffix, "[$1]")
			// Convert dotted tokens with special characters to bracket notation: .bad-key -> ["bad-key"]
			// Leave existing bracket segments intact
			if strings.Contains(suffix, ".") {
				var b strings.Builder
				i := 0
				for i < len(suffix) {
					ch := suffix[i]
					if ch == '.' {
						// start of a dot segment
						j := i + 1
						for j < len(suffix) && suffix[j] != '.' && suffix[j] != '[' {
							j++
						}
						seg := suffix[i+1 : j]
						// If segment contains non [A-Za-z0-9_] chars, wrap with ["..."]
						needsBracket := false
						for _, rc := range seg {
							if !isAlphaNumOrUnderscore(rc) {
								needsBracket = true
								break
							}
						}
						if needsBracket {
							b.WriteString("[")
							b.WriteString("\"")
							b.WriteString(seg)
							b.WriteString("\"")
							b.WriteString("]")
						} else {
							b.WriteString(".")
							b.WriteString(seg)
						}
						i = j
					} else {
						// copy other chars verbatim (including '[' segments)
						b.WriteByte(ch)
						i++
					}
				}
				suffix = b.String()
			}
			// If the target is an array, provide an index example and length context
			if arr, ok := m[token].([]interface{}); ok {
				// If user opened a bracket without index, suggest [0]
				if suffix == "[" || suffix == "" {
					return fmt.Sprintf("Hint: CLI --expression uses '_' as root. Did you mean '_.%s[0]'? (array length: %d)", token, len(arr))
				}
				// Otherwise, keep suffix and add length info
				return fmt.Sprintf("Hint: CLI --expression uses '_' as root. Did you mean '_.%s%s'? (array length: %d)", token, suffix, len(arr))
			}
			// Suggest the explicit '_' root variable
			return fmt.Sprintf("Hint: CLI --expression uses '_' as root. Did you mean '_.%s%s'?", token, suffix)
		}
	}
	return ""
}

const defaultFallbackTermWidth = 120

// validateLimitingFlags checks that limiting flags are not in conflict and returns an error if they are.
func validateLimitingFlags() error {
	cfg := limiter.Config{
		Limit:  limitRecords,
		Offset: offsetRecords,
		Tail:   tailRecords,
	}
	return cfg.Validate()
}

// applyLimiting applies the record-limiting configuration to data.
func applyLimiting(data interface{}) interface{} {
	cfg := limiter.Config{
		Limit:  limitRecords,
		Offset: offsetRecords,
		Tail:   tailRecords,
	}
	if !cfg.IsActive() {
		return data
	}
	return cfg.Apply(data)
}

func parseSortOrder(value string) (navigator.SortOrder, error) {
	s := strings.ToLower(strings.TrimSpace(value))
	switch s {
	case "", "none":
		return navigator.SortNone, nil
	case "asc", "ascending":
		return navigator.SortAscending, nil
	case "desc", "descending":
		return navigator.SortDescending, nil
	default:
		return navigator.SortNone, fmt.Errorf("invalid sort order %q (expected ascending, descending, or none)", value)
	}
}

func resolveSortOrder(cfg ui.ThemeConfigFile) (navigator.SortOrder, error) {
	// CLI flag takes precedence when provided
	if strings.TrimSpace(sortOrder) != "" {
		return parseSortOrder(sortOrder)
	}

	// Fall back to config value
	if cfg.Display.Sort != nil {
		return parseSortOrder(*cfg.Display.Sort)
	}

	// Default to none if not set
	return navigator.SortNone, nil
}

// buildVersionData collects version and build information for templating.
func buildVersionData(cfg *ui.ThemeConfigFile) map[string]interface{} {
	info, ok := rdebug.ReadBuildInfo()

	version := "dev"
	goVersion := runtime.Version()
	buildOS := runtime.GOOS
	buildArch := runtime.GOARCH
	gitCommit := ""

	if ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = info.Main.Version
		} else {
			for _, s := range info.Settings {
				if s.Key == "vcs.revision" && len(s.Value) >= 7 {
					gitCommit = s.Value[:7]
					version = gitCommit
					break
				}
			}
		}
		if info.GoVersion != "" {
			goVersion = info.GoVersion
		}
		// Extract build settings
		for _, s := range info.Settings {
			switch s.Key {
			case "GOOS":
				buildOS = s.Value
			case "GOARCH":
				buildArch = s.Value
			}
		}
	}

	name := "kvx"
	if cfg != nil {
		name = cfg.About.Name
	}

	return map[string]interface{}{
		"Version":   version,
		"GoVersion": goVersion,
		"BuildOS":   buildOS,
		"BuildArch": buildArch,
		"GitCommit": gitCommit,
		"Name":      name,
	}
}

// searchCLI recursively searches keys and values for a query (case-insensitive) and returns matches keyed by path.
func searchCLI(node interface{}, query string) map[string]interface{} {
	hits := make(map[string]interface{})
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return hits
	}
	var walk func(n interface{}, path string)
	walk = func(n interface{}, path string) {
		switch cur := n.(type) {
		case map[string]interface{}:
			keys := make([]string, 0, len(cur))
			for k := range cur {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				child := cur[k]
				childPath := joinSearchPath(path, k)
				valStr := formatter.Stringify(child)
				if strings.Contains(strings.ToLower(k), q) || strings.Contains(strings.ToLower(valStr), q) {
					hits[childPath] = child
				}
				walk(child, childPath)
			}
		case []interface{}:
			for i, child := range cur {
				childPath := fmt.Sprintf("%s[%d]", path, i)
				valStr := formatter.Stringify(child)
				if strings.Contains(strings.ToLower(valStr), q) {
					hits[childPath] = child
				}
				walk(child, childPath)
			}
		default:
			// scalars handled by parent
		}
	}
	walk(node, "")
	return hits
}

func joinSearchPath(prefix, seg string) string {
	if prefix == "" {
		return seg
	}
	if strings.HasPrefix(seg, "[") {
		return prefix + seg
	}
	return prefix + "." + seg
}

// detectTerminalSize returns the best-effort terminal width/height by probing
// stdout, stderr, and stdin, then falling back to $COLUMNS.
func detectTerminalSize() (int, int) {
	fds := []uintptr{os.Stdout.Fd(), os.Stderr.Fd(), os.Stdin.Fd()}
	for _, fd := range fds {
		if w, h, err := term.GetSize(int(fd)); err == nil && (w > 0 || h > 0) {
			return w, h
		}
	}
	if col := os.Getenv("COLUMNS"); col != "" {
		if w, err := strconv.Atoi(col); err == nil && w > 0 {
			return w, 0
		}
	}
	// Use a generous fallback to avoid overly narrow truncation when size cannot be detected (e.g., CI, PowerShell).
	return defaultFallbackTermWidth, 0
}

// cliVersionString builds a human-readable version string for CLI output and Cobra's --version flag.
func cliVersionString() string {
	// Load config to get about information
	configFile := resolveConfigPath("")
	cfg, _ := loadMergedConfig(configFile)

	// Use individual fields to build version string
	name := cfg.About.Name
	if name == "" {
		name = "kvx"
	}
	version := cfg.About.Version
	if version == "" {
		version = "dev"
	}
	goVersion := cfg.About.GoVersion
	if goVersion == "" {
		goVersion = runtime.Version()
	}

	return fmt.Sprintf("%s %s (go %s)", name, version, goVersion)
}

// getCLIShortHelp returns the short help text from config
func getCLIShortHelp() string {
	configFile := resolveConfigPath("")
	cfg, _ := loadMergedConfig(configFile)

	name := cfg.About.Name
	short := fmt.Sprintf("%s - YAML/JSON data explorer", name)
	return short
}

// getCLILongHelp returns the long help text from config
func getCLILongHelp() string {
	configFile := resolveConfigPath("")
	cfg, _ := loadMergedConfig(configFile)

	name := cfg.About.Name
	description := cfg.About.Description

	// Build template data for processing help text
	buildData := buildVersionData(&cfg)
	templateData := map[string]interface{}{
		"config": map[string]interface{}{
			"app": map[string]interface{}{
				"about": map[string]interface{}{
					"name":              cfg.About.Name,
					"description":       cfg.About.Description,
					"version":           cfg.About.Version,
					"go_version":        cfg.About.GoVersion,
					"build_os":          cfg.About.BuildOS,
					"build_arch":        cfg.About.BuildArch,
					"git_commit":        cfg.About.GitCommit,
					"license":           cfg.About.License,
					"repository_url":    cfg.About.RepositoryURL,
					"documentation_url": cfg.About.DocumentationURL,
					"author":            cfg.About.Author,
					"details":           cfg.About.Details,
				},
				"cli": map[string]interface{}{
					"help_header_template": cfg.CLI.HelpHeaderTemplate,
					"help_description":     cfg.CLI.HelpDescription,
					"help_usage":           cfg.CLI.HelpUsage,
				},
			},
		},
		"build": buildData,
	}

	var long strings.Builder

	// Use config description (always present from default config)
	helpDescription := processTemplateString(cfg.CLI.HelpDescription, templateData)
	long.WriteString(fmt.Sprintf("%s is %s. %s\n\n", name, description, helpDescription))

	// Add details (always present from default config)
	for _, detail := range cfg.About.Details {
		processedDetail := processTemplateString(detail, templateData)
		long.WriteString(processedDetail)
		long.WriteString("\n")
	}
	long.WriteString("\n")

	// Use config usage instructions (always present from default config)
	helpUsage := processTemplateString(cfg.CLI.HelpUsage, templateData)
	long.WriteString(helpUsage)

	return long.String()
}

func helpAboutHeader() string {
	// Load config to get about information
	configFile := resolveConfigPath("")
	cfg, _ := loadMergedConfig(configFile)

	// Use config header template (always present from default config)
	headerTemplate := cfg.CLI.HelpHeaderTemplate

	// Process template
	buildData := buildVersionData(&cfg)
	templateData := map[string]interface{}{
		"config": map[string]interface{}{
			"app": map[string]interface{}{
				"about": map[string]interface{}{
					"name":              cfg.About.Name,
					"description":       cfg.About.Description,
					"version":           cfg.About.Version,
					"go_version":        cfg.About.GoVersion,
					"build_os":          cfg.About.BuildOS,
					"build_arch":        cfg.About.BuildArch,
					"git_commit":        cfg.About.GitCommit,
					"license":           cfg.About.License,
					"repository_url":    cfg.About.RepositoryURL,
					"documentation_url": cfg.About.DocumentationURL,
					"author":            cfg.About.Author,
				},
			},
		},
		"build": buildData,
	}
	return processTemplateString(headerTemplate, templateData) + "\n"
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print kvx version",
	RunE: func(_ *cobra.Command, _ []string) error {
		fmt.Println(cliVersionString()) //nolint:forbidigo
		return nil
	},
}

// configCmd groups configuration-related subcommands similar to gh-style CLIs.
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage kvx configuration",
	RunE: func(cmd *cobra.Command, _ []string) error {
		// Show help when invoked without a subcommand (gh-style UX)
		return cmd.Help()
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Show merged configuration",
	RunE: func(cmd *cobra.Command, _ []string) error {
		// Interactive flag for TUI view of the config
		interactiveFlag, _ := cmd.Flags().GetBool("interactive")
		if interactiveFlag {
			return runConfigGetInteractive(cmd)
		}
		return runConfigView(cmd)
	},
}

var configThemesCmd = &cobra.Command{
	Use:   "themes",
	Short: "List available themes",
	RunE: func(_ *cobra.Command, _ []string) error {
		return runThemesList()
	},
}

var configThemeCmd = &cobra.Command{
	Use:   "theme",
	Short: "List available themes",
	RunE: func(_ *cobra.Command, _ []string) error {
		return runThemesList()
	},
}

var themesCmd = &cobra.Command{
	Use:   "themes",
	Short: "List available themes",
	RunE: func(_ *cobra.Command, _ []string) error {
		return runThemesList()
	},
}

// runThemesList prints the available themes from merged configuration
func runThemesList() error {
	resolvedPath := resolveConfigPath(configFile)
	merged, _ := loadMergedConfig(resolvedPath)
	names := keys(merged.Themes)
	sort.Strings(names)
	def := merged.Theme.Default
	if def == "" {
		def = "dark"
	}
	fmt.Printf("Available themes (default: %s):\n", def) //nolint:forbidigo
	for _, name := range names {
		fmt.Printf(" - %s\n", name) //nolint:forbidigo
	}
	return nil
}

// runConfigView prints the configuration honoring --output. It prefers the user's config file
// (verbatim, preserving comments). If no user config is found, it falls back to the default
// config file (also verbatim with comments). Only table/json outputs require unmarshaling.
func runConfigView(_ *cobra.Command) error {
	resolved := resolveConfigPath(configFile)
	var raw []byte
	var err error
	if resolved != "" {
		raw, err = os.ReadFile(resolved)
		if err != nil {
			return fmt.Errorf("failed to read config file %s: %w", resolved, err)
		}
	} else {
		raw, err = loadDefaultConfigRaw()
		if err != nil {
			return fmt.Errorf("failed to read default config: %w", err)
		}
	}

	printRaw := func(data []byte) {
		// Preserve existing newline if present; append one if missing for clean CLI output.
		if len(data) == 0 {
			fmt.Print("\n") //nolint:forbidigo
			return
		}
		if data[len(data)-1] == '\n' {
			fmt.Print(string(data)) //nolint:forbidigo
			return
		}
		fmt.Printf("%s\n", string(data)) //nolint:forbidigo
	}

	switch configOutput {
	case "yaml", "raw":
		printRaw(raw)
		return nil
	case "table", "json":
		obj, err := decodeYAMLLenient(raw)
		if err != nil {
			return fmt.Errorf("failed to decode config for %s view: %w", configOutput, err)
		}
		if configOutput == "json" {
			data, err := json.MarshalIndent(obj, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal config: %w", err)
			}
			fmt.Print(string(data) + "\n") //nolint:forbidigo
			return nil
		}

		// Table view
		outputWidth := snapshotWidth
		if outputWidth <= 0 {
			if w, _ := detectTerminalSize(); w > 0 {
				outputWidth = w
			}
		}
		keyW := 30
		valueW := 0 // auto
		appName := "kvx"
		if rootMap, ok := obj.(map[string]interface{}); ok {
			if app, ok := rootMap["app"].(map[string]interface{}); ok {
				if about, ok := app["about"].(map[string]interface{}); ok {
					if name, ok := about["name"].(string); ok && strings.TrimSpace(name) != "" {
						appName = name
					}
				}
			}
		}
		fmt.Print(renderBorderedTable(obj, noColor, keyW, valueW, outputWidth, appName, "_")) //nolint:forbidigo
		return nil
	default:
		return fmt.Errorf("invalid output for config: %s (use yaml|json|table|raw)", configOutput)
	}
}

// runConfigGetInteractive renders the merged config in the interactive TUI
func runConfigGetInteractive(_ *cobra.Command) error {
	resolved := resolveConfigPath(configFile)
	mergedCfg, err := loadMergedConfig(resolved)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	// Initialize themes and apply default
	_ = ui.InitializeThemes(&mergedCfg)
	selectedTheme := mergedCfg.Theme.Default
	if th, ok := mergedCfg.Themes[selectedTheme]; ok {
		ui.SetTheme(ui.ThemeFromConfig(th))
	} else if th, ok := ui.GetTheme(selectedTheme); ok {
		ui.SetTheme(th)
	}

	// Prepare sanitized config data as root node
	var root interface{}
	sanitized := sanitizeConfig(mergedCfg)
	b, _ := yaml.Marshal(sanitized)
	_ = yaml.Unmarshal(b, &root)

	appName := mergedCfg.About.Name
	if strings.TrimSpace(appName) == "" {
		appName = "kvx"
	}

	// Load help content from config
	helpTitle, helpText := loadHelp(resolved, effectiveKeyMode(mergedCfg))

	sink := func(string) {}
	// Use terminal-detected size unless explicit width/height are set at root flags
	runW := 0
	runH := 0
	if f := rootCmd.Flags().Lookup("width"); f != nil && f.Changed {
		runW = snapshotWidth
	}
	if f := rootCmd.Flags().Lookup("height"); f != nil && f.Changed {
		runH = snapshotHeight
	}
	opts, cleanup := getProgramOptions()
	defer cleanup()

	if err := ui.RunModel(appName, root, helpTitle, helpText, false, sink, "", runW, runH, nil, false, "", nil, func(m *ui.Model) {
		applySnapshotConfigToModel(m, mergedCfg)
	}, opts...); err != nil {
		return err
	}
	return nil
}

// getProgramOptions handles piped stdin by reopening the terminal for interactive input/output.
// This allows Bubble Tea to work properly with piped data while still receiving keyboard input
// and resize events on platforms like Windows.
// Returns tea.ProgramOption values (plus a cleanup) that should be passed to tea.NewProgram.
func getProgramOptions() ([]tea.ProgramOption, func()) {
	isPiped := stdinIsPiped()
	cleanup := func() {}

	if !isPiped {
		// Normal terminal input - use default behavior
		return nil, cleanup
	}

	// Piped input detected: open real terminal devices for interactive control
	ttyIn, ttyOut, err := openTerminalIOFn()
	if err != nil {
		// /dev/tty not available (e.g., in some CI environments)
		// Silently fall back to piped stdin - TUI will work but arrow keys/resize won't
		return nil, cleanup
	}
	cleanup = func() {
		_ = ttyIn.Close()
		if ttyOut != nil && ttyOut != ttyIn {
			_ = ttyOut.Close()
		}
	}

	// Return WithInput/WithOutput options to use the real terminal instead of stdin.
	// Import tea at the top of the file includes this
	ctx, cancel := context.WithCancel(context.Background())
	opts := []tea.ProgramOption{tea.WithContext(ctx), tea.WithInput(ttyIn)}
	if ttyOut != nil {
		opts = append(opts, tea.WithOutput(ttyOut), withTTYResizeWatcher(ctx, ttyOut))
	}

	return opts, func() {
		cancel()
		cleanup()
	}
}

func openTerminalIO() (*os.File, *os.File, error) {
	in, out := terminalDeviceNames(runtime.GOOS)

	input, err := os.OpenFile(in, os.O_RDWR, 0)
	if err != nil {
		return nil, nil, err
	}

	if out == "" || out == in {
		return input, input, nil
	}

	output, err := os.OpenFile(out, os.O_RDWR, 0)
	if err != nil {
		return input, nil, err
	}

	return input, output, nil
}

func terminalDeviceNames(goos string) (input string, output string) {
	if goos == "windows" {
		return "CONIN$", "CONOUT$"
	}

	return "/dev/tty", "/dev/tty"
}

// withTTYResizeWatcher polls terminal size and sends resize messages when signals are unreliable
// (e.g., piped stdin on Windows). This is best-effort and stops when the context is canceled.
func withTTYResizeWatcher(ctx context.Context, out *os.File) tea.ProgramOption {
	return func(p *tea.Program) {
		if ctx == nil || out == nil {
			return
		}

		go func() {
			t := newResizeTicker(250 * time.Millisecond)
			defer t.Stop()

			lastW, lastH := 0, 0
			for {
				select {
				case <-ctx.Done():
					return
				case <-t.C():
					w, h, err := termGetSize(int(out.Fd()))
					if err != nil {
						continue
					}
					if w == lastW && h == lastH {
						continue
					}
					lastW, lastH = w, h
					sendWindowSize(p, tea.WindowSizeMsg{Width: w, Height: h})
				}
			}
		}()
	}
}

var rootCmd = &cobra.Command{
	Use:     "kvx [file]",
	Short:   getCLIShortHelp(),
	Long:    getCLILongHelp(),
	Example: "\n  kvx tests/sample.yaml\n  kvx tests/sample.yaml -e '_.items[0].name'\n  kvx tests/sample.yaml -e 'type(_)'\n  kvx tests/sample.yaml -e '_.metadata[\"bad-key\"]'\n  type tests/sample.yaml | kvx -e '_.items.filter(x, x.available)'\n",
	Args:    cobra.MaximumNArgs(1),
	PersistentPreRun: func(cmd *cobra.Command, _ []string) {
		// Initialize structured logger with JSON output
		// Map CLI debug flag to log level: debug => zap.DebugLevel (-1), else zap.InfoLevel (0)
		var level int8 = 0
		if debug {
			level = -1
		}
		lgr := logger.Get(level)
		// Attach basic context about the command
		lgr = logger.WithValues(lgr, logger.RootCommandKey, "kvx", logger.SubCommandKey, cmd.Name())
		rootCtx = logger.WithLogger(context.Background(), lgr)
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Validate record-limiting flags first
		if err := validateLimitingFlags(); err != nil {
			fmt.Fprintf(os.Stderr, "record limiting error: %v\n", err)
			os.Exit(2)
		}

		// Validate array-style flag
		if err := formatter.ValidateArrayStyle(arrayStyle); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(2)
		}

		// Validate auto-decode flag
		if autoDecode != "" && autoDecode != "lazy" && autoDecode != "eager" && autoDecode != "disabled" {
			fmt.Fprintf(os.Stderr, "Error: invalid --auto-decode value %q (expected 'lazy', 'eager', or 'disabled')\n", autoDecode)
			os.Exit(2)
		}

		limitCfg := limiter.Config{
			Limit:  limitRecords,
			Offset: offsetRecords,
			Tail:   tailRecords,
		}
		ui.SetLimiterConfig(limitCfg)

		// Snapshot rendering is handled separately.
		if renderSnapshot {
			interactive = false
		}

		themeFlagSet := cmd.Flags().Changed("theme")
		debugLog := debug
		if interactive && strings.TrimSpace(searchTerm) != "" {
			startKeys = append([]string{"<F3>", strings.TrimSpace(searchTerm)}, startKeys...)
		}

		// Load config early to get DebugMaxEvents value (CLI flag takes precedence if set)
		configFile = resolveConfigPath(configFile)
		maxEvents := debugMaxEvents // Default to CLI flag value
		if configFile != "" {
			cfgFile, err := loadMergedConfig(configFile)
			if err == nil && cfgFile.DebugMaxEvents != nil {
				// Use config value only if CLI flag wasn't explicitly set (default is 200)
				// If user set --debug-max-events, it takes precedence
				debugMaxEventsFlag := cmd.Flags().Lookup("debug-max-events")
				if debugMaxEventsFlag == nil || !debugMaxEventsFlag.Changed {
					maxEvents = *cfgFile.DebugMaxEvents
				}
			}
		}

		dc := newDebugCollector(debugLog, maxEvents)
		defer dc.Flush()
		navigator.DebugWriter = dc.Writer()

		detectedTermWidth, detectedTermHeight := detectTerminalSize()

		// Prefer explicit --config-file, else check XDG default locations.
		if debugLog {
			if configFile != "" {
				dc.Printf("DBG: --config-file provided: %s\n", configFile)
			} else {
				dc.Println("DBG: No --config-file provided; checking XDG_CONFIG_HOME and ~/.config/kvx/config.yaml")
			}
		}
		if debugLog {
			if configFile != "" {
				dc.Printf("DBG: Using config file: %s\n", configFile)
			} else {
				dc.Println("DBG: No config file found (using built-in defaults)")
			}
			if detectedTermWidth > 0 || detectedTermHeight > 0 {
				dc.Printf("DBG: Detected terminal size before run: width=%d height=%d\n", detectedTermWidth, detectedTermHeight)
			} else {
				dc.Println("DBG: Detected terminal size before run: unavailable (falling back to defaults if needed)")
			}
		}

		if !configMode && (interactive || renderSnapshot) {
			cfg, _ := loadMergedConfig(configFile)
			applyFunctionExamples(cfg, true)
			// Initialize themes from configuration (must be done before theme selection)
			if err := ui.InitializeThemes(&cfg); err != nil {
				// Log warning but continue - will use fallback dark theme
				fmt.Fprintf(os.Stderr, "Warning: Failed to initialize themes: %v\n", err)
			}
			if err := applyThemeFromConfig(cfg, themeName, themeFlagSet); err != nil {
				printThemeSelectionError(os.Stderr, err)
				os.Exit(2)
			}
			order, err := resolveSortOrder(cfg)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
			navigator.SetSortOrder(order)
			if menuHasData(cfg.Menu) {
				ui.SetMenuConfig(ui.MenuFromConfig(cfg.Menu, cfg.Features.AllowEditInput))
			}
			appName := cfg.About.Name
			if appName == "" {
				appName = "kvx"
			}
			rootData, _, err := loadInputData(args, expression, debugLog, dc, *logger.FromContext(rootCtx))
			if err != nil {
				if errors.Is(err, errShowHelp) {
					_ = cmd.Help()
					return
				}
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
			// Eager auto-decode: recursively decode all serialized scalars before
			// expression evaluation so that CEL can see the decoded structures.
			if autoDecode == "eager" {
				rootData = loader.RecursiveDecode(rootData)
			}

			// Evaluate expression for snapshot parity with CLI limiting
			snapshotNode := rootData
			if expression != "" {
				if debug {
					dc.Printf("DBG: Evaluating expression for snapshot: %s\n", expression)
				}
				engine, err := core.New()
				if err != nil {
					fmt.Fprintf(os.Stderr, "failed to init evaluator: %v\n", err)
					os.Exit(1)
				}
				n, err := engine.Evaluate(expression, rootData)
				if err != nil {
					fmt.Fprintf(os.Stderr, "explore expression error: %v\n", err)
					if hint := buildSuggestion(expression, rootData); hint != "" {
						fmt.Fprintln(os.Stderr, hint)
					}
					os.Exit(2)
				}
				if debug {
					dc.Printf("DBG: Snapshot expression result type: %T\n", n)
				}
				snapshotNode = n
			}

			// Lazy auto-decode: decode the expression result if it's a serialized scalar
			if autoDecode == "lazy" {
				if s, ok := snapshotNode.(string); ok {
					if decoded, ok := loader.TryDecode(s); ok {
						snapshotNode = decoded
					}
				}
			}

			// Apply limiting after expression for snapshot mode
			snapshotNode = applyLimiting(snapshotNode)

			// Support snapshot rendering
			if renderSnapshot {
				view := renderSnapshotOutput(cfg, snapshotNode, snapshotNode, appName, startKeys, expression, noColor, snapshotWidth, snapshotHeight, detectedTermWidth, detectedTermHeight, configFile, debugLog, dc, "", effectiveKeyMode(cfg))
				fmt.Print(view) //nolint:forbidigo
				if debugLog {
					printDebugEvents(dc.events)
				}
				return
			}

			sink := func(msg string) {
				if debugLog {
					dc.Append(msg)
				}
			}
			widthFlagSet := cmd.Flags().Lookup("width").Changed
			heightFlagSet := cmd.Flags().Lookup("height").Changed
			runW := 0
			runH := 0
			if widthFlagSet {
				runW = snapshotWidth
			}
			if heightFlagSet {
				runH = snapshotHeight
			}
			// If startup keys include F10, bypass TUI and emit the non-interactive output immediately.
			if hasF10(startKeys) {
				navigator.Debug = debugLog
				node := rootData
				if expression != "" {
					if debug {
						dc.Printf("DBG: Evaluating expression: %s\n", expression)
					}
					engine, err := core.New()
					if err != nil {
						fmt.Fprintf(os.Stderr, "failed to init evaluator: %v\n", err)
						os.Exit(1)
					}
					n, err := engine.Evaluate(expression, rootData)
					if err != nil {
						fmt.Fprintf(os.Stderr, "explore expression error: %v\n", err)
						if hint := buildSuggestion(expression, rootData); hint != "" {
							fmt.Fprintln(os.Stderr, hint)
						}
						os.Exit(2)
					}
					if debug {
						dc.Printf("DBG: Expression result type: %T\n", n)
					}
					node = n
				} else if debug {
					dc.Println("DBG: No expression provided, using root")
				}

				// Apply record limiting after expression evaluation
				node = applyLimiting(node)

				// Get column widths from config (or use defaults)
				keyW := 30
				valueW := 0 // 0 means auto-calculate
				if cfg.Display.KeyColWidth != nil && *cfg.Display.KeyColWidth > 0 {
					keyW = *cfg.Display.KeyColWidth
				}
				// Get height: use --height flag if set, otherwise auto-detect from terminal
				outputHeight := snapshotHeight
				usedDetectH := detectedTermHeight
				if outputHeight <= 0 {
					if usedDetectH <= 0 {
						if _, h := detectTerminalSize(); h > 0 {
							usedDetectH = h
						}
					}
					if usedDetectH > 0 {
						outputHeight = usedDetectH
					} else {
						outputHeight = 24 // fallback
					}
				}
				if debugLog {
					dc.Printf("DBG: CLI output sizing (F10 bypass): widthHint=%d detectedWidth=%d height=%d (detectedHeight=%d, flagHeight=%d)\n",
						snapshotWidth, detectedTermWidth, outputHeight, usedDetectH, snapshotHeight)
				}
				if searchTerm != "" {
					if debug {
						dc.Printf("DBG: Running CLI search for %q\n", searchTerm)
					}
					rows := ui.SearchRows(node, searchTerm)
					if len(rows) == 0 {
						fmt.Println("No matches found.") //nolint:forbidigo
					} else {
						// Non-interactive search: render bordered table
						outputWidth := snapshotWidth
						if outputWidth <= 0 {
							if detectedTermWidth > 0 {
								outputWidth = detectedTermWidth
							}
						}
						fmt.Print(renderBorderedTableRows(rows, noColor, keyW, valueW, outputWidth, appName, "_", node)) //nolint:forbidigo
					}
					if debugLog && len(dc.events) > 0 {
						printDebugEvents(dc.events)
					}
				}
				// Determine width for rendering
				outputWidth := snapshotWidth
				if outputWidth <= 0 {
					if detectedTermWidth > 0 {
						outputWidth = detectedTermWidth
					}
				}
				yamlOpts := yamlFormatOptionsFromConfig(cfg)
				tableOpts := tableFormatOptionsFromConfig(cfg)
				treeOpts := treeFormatOptionsFromConfig(cfg, outputWidth, stdoutIsPiped())
				mermaidOpts := mermaidFormatOptionsFromConfig(cfg)
				printEvalResult(node, output, noColor, keyW, valueW, outputHeight, outputWidth, appName, "_", yamlOpts, tableOpts, treeOpts, mermaidOpts)
				if debugLog && len(dc.events) > 0 {
					printDebugEvents(dc.events)
				}
			}
			helpTitle, helpText := loadHelp(configFile, effectiveKeyMode(cfg))
			opts, cleanup := getProgramOptions()
			defer cleanup()
			if err := ui.RunModel(appName, rootData, helpTitle, helpText, debugLog, sink, expression, runW, runH, startKeys, noColor, "", nil, func(m *ui.Model) {
				applySnapshotConfigToModel(m, cfg)
			}, opts...); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			if debugLog {
				printDebugEvents(dc.events)
			}
			return
		}

		if configMode {
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				fmt.Fprintln(os.Stderr, "--config cannot be used with stdin input")
				os.Exit(2)
			}
			if len(args) > 0 {
				fmt.Fprintln(os.Stderr, "--config is mutually exclusive with file input")
				os.Exit(2)
			}
			mergedCfg, err := loadMergedConfig(configFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
				os.Exit(2)
			}
			if err := ui.InitializeThemes(&mergedCfg); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to initialize themes: %v\n", err)
			}
			if err := applyThemeFromConfig(mergedCfg, themeName, themeFlagSet); err != nil {
				printThemeSelectionError(os.Stderr, err)
				os.Exit(2)
			}
			order, err := resolveSortOrder(mergedCfg)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
			navigator.SetSortOrder(order)
			if menuHasData(mergedCfg.Menu) {
				ui.SetMenuConfig(ui.MenuFromConfig(mergedCfg.Menu, mergedCfg.AllowEditInput))
			}

			allowEditInput := true
			if mergedCfg.Features.AllowEditInput != nil {
				allowEditInput = *mergedCfg.Features.AllowEditInput
			}
			menuCfg := ui.MenuFromConfig(mergedCfg.Menu, &allowEditInput)
			baseHelp := strings.TrimSpace(ui.GenerateHelpText(menuCfg, allowEditInput, nil, ui.DefaultKeyMode))
			if baseHelp == "" {
				defMenu := ui.DefaultMenuConfig()
				baseHelp = strings.TrimSpace(ui.GenerateHelpText(defMenu, true, nil, ui.DefaultKeyMode))
			}
			helpText := baseHelp
			helpTitle := "Help"
			if popup := mergedCfg.Menu.F1.Popup; (popup.Enabled == nil || *popup.Enabled) && strings.TrimSpace(popup.Text) != "" {
				helpText = strings.TrimSpace(popup.Text)
				if strings.TrimSpace(popup.Title) != "" {
					helpTitle = strings.TrimSpace(popup.Title)
				}
			}
			if strings.TrimSpace(mergedCfg.Menu.F1.Popup.Title) != "" {
				helpTitle = strings.TrimSpace(mergedCfg.Menu.F1.Popup.Title)
			}

			appName := mergedCfg.About.Name
			if appName == "" {
				appName = "kvx"
			}

			// Respect explicit width/height flags for parity with normal -i usage.
			widthFlagSet := cmd.Flags().Lookup("width").Changed
			heightFlagSet := cmd.Flags().Lookup("height").Changed
			runW := 0
			runH := 0
			if widthFlagSet {
				runW = snapshotWidth
			}
			if heightFlagSet {
				runH = snapshotHeight
			}

			// Prepare sanitized config data once for all modes.
			var root interface{}
			sanitized := sanitizeConfig(mergedCfg)
			b, _ := yaml.Marshal(sanitized)
			_ = yaml.Unmarshal(b, &root)

			if renderSnapshot {
				limitedRoot := applyLimiting(root)
				view := renderSnapshotOutput(mergedCfg, limitedRoot, root, appName, startKeys, expression, noColor, snapshotWidth, snapshotHeight, detectedTermWidth, detectedTermHeight, configFile, debugLog, dc, "config mode", effectiveKeyMode(mergedCfg))
				fmt.Print(view) //nolint:forbidigo
				if debugLog {
					printDebugEvents(dc.events)
				}
				return
			}

			if interactive {
				sink := func(msg string) {
					if debugLog {
						dc.Append(msg)
					}
				}
				opts, cleanup := getProgramOptions()
				defer cleanup()
				if err := ui.RunModel(appName, root, helpTitle, helpText, debugLog, sink, expression, runW, runH, startKeys, noColor, "", nil, func(m *ui.Model) {
					applySnapshotConfigToModel(m, mergedCfg)
				}, opts...); err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				if debugLog {
					printDebugEvents(dc.events)
				}
			} else {
				switch output {
				case "yaml", "raw":
					var buf bytes.Buffer
					enc := yaml.NewEncoder(&buf)
					enc.SetIndent(2)
					if err := enc.Encode(sanitized); err != nil {
						fmt.Fprintf(os.Stderr, "failed to marshal config: %v\n", err)
						os.Exit(2)
					}
					fmt.Print(addConfigComments(buf.String())) //nolint:forbidigo
				case "table":
					var obj interface{}
					data, err := yaml.Marshal(sanitized)
					if err != nil {
						fmt.Fprintf(os.Stderr, "failed to marshal config: %v\n", err)
						os.Exit(2)
					}
					if err := yaml.Unmarshal(data, &obj); err != nil {
						fmt.Fprintf(os.Stderr, "failed to decode config for table view: %v\n", err)
						os.Exit(2)
					}
					// Determine width for rendering (honor --width when set; else detect terminal width)
					outputWidth := snapshotWidth
					if outputWidth <= 0 {
						if detectedTermWidth > 0 {
							outputWidth = detectedTermWidth
						}
					}
					// Use default column widths for config output
					keyW := 30
					valueW := 0 // auto
					appName := mergedCfg.About.Name
					if strings.TrimSpace(appName) == "" {
						appName = "kvx"
					}
					// Render bordered table with footer parity (includes type label)
					fmt.Print(renderBorderedTable(obj, noColor, keyW, valueW, outputWidth, appName, "_")) //nolint:forbidigo
				case "json":
					data, err := json.MarshalIndent(sanitized, "", "  ")
					if err != nil {
						fmt.Fprintf(os.Stderr, "failed to marshal config: %v\n", err)
						os.Exit(2)
					}
					fmt.Print(string(data) + "\n") //nolint:forbidigo
				default:
					fmt.Fprintf(os.Stderr, "invalid output for --config: %s (use yaml|json)\n", output)
					os.Exit(2)
				}
			}
			return
		}

		var root interface{}

		var cfg ui.ThemeConfigFile
		var err error
		var keyColWidthPtr *int
		var valueColWidthPtr *int
		if configFile != "" {
			cfgFile, err := loadConfigState(configFile, themeName, themeFlagSet, true, true, true)
			if err != nil {
				if errors.As(err, new(themeSelectionError)) {
					printThemeSelectionError(os.Stderr, err)
				} else {
					fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
				}
				os.Exit(2)
			}
			cfg = cfgFile
			keyColWidthPtr = cfgFile.Display.KeyColWidth
		} else {
			// No config file - load default config and initialize themes
			cfgFile, err := loadConfigState("", themeName, themeFlagSet, true, true, false)
			if err != nil {
				if errors.As(err, new(themeSelectionError)) {
					printThemeSelectionError(os.Stderr, err)
				} else {
					fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
				}
				os.Exit(2)
			}
			cfg = cfgFile
		}

		root, _, err = loadInputData(args, expression, debugLog, dc, *logger.FromContext(rootCtx))
		if err != nil {
			if errors.Is(err, errShowHelp) {
				_ = cmd.Help()
				return
			}
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}

		// Eager auto-decode: recursively decode all serialized scalars before
		// expression evaluation so that CEL can see the decoded structures.
		if autoDecode == "eager" {
			root = loader.RecursiveDecode(root)
		}

		// All interactive and snapshot flows are handled earlier; reaching here should
		// always mean non-interactive CLI output.
		if interactive || renderSnapshot {
			fmt.Fprintln(os.Stderr, "interactive path should have been handled earlier")
			os.Exit(1)
		}

		// Wire CLI debug flag to navigator troubleshooting logs
		navigator.Debug = debugLog
		node := root
		if expression != "" {
			if debug {
				dc.Printf("DBG: Evaluating expression: %s\n", expression)
			}
			// Strict CLI mode: evaluate explore as CEL; require explicit '_' or valid CEL
			engine, err := core.New()
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to init evaluator: %v\n", err)
				os.Exit(1)
			}
			n, err := engine.Evaluate(expression, root)
			if err != nil {
				fmt.Fprintf(os.Stderr, "explore expression error: %v\n", err)
				if hint := buildSuggestion(expression, root); hint != "" {
					fmt.Fprintln(os.Stderr, hint)
				}
				os.Exit(2)
			}
			if debug {
				dc.Printf("DBG: Expression result type: %T\n", n)
			}
			node = n
		} else if debug {
			dc.Println("DBG: No expression provided, using root")
		}

		// Lazy auto-decode: decode the expression result if it's a serialized scalar
		if autoDecode == "lazy" {
			if s, ok := node.(string); ok {
				if decoded, ok := loader.TryDecode(s); ok {
					node = decoded
				}
			}
		}

		// Apply record limiting after expression evaluation
		node = applyLimiting(node)

		if debug {
			dc.Printf("DBG: Output format: %s\n", output)
		}
		// Get column widths from config (or use defaults)
		keyW := 30
		valueW := 0 // 0 means auto-calculate
		if keyColWidthPtr != nil && *keyColWidthPtr > 0 {
			keyW = *keyColWidthPtr
		}
		if valueColWidthPtr != nil && *valueColWidthPtr > 0 {
			valueW = *valueColWidthPtr
		}
		// Get height: use --height flag if set, otherwise auto-detect from terminal
		outputHeight := snapshotHeight
		usedDetectH := detectedTermHeight
		if outputHeight <= 0 {
			if usedDetectH <= 0 {
				if _, h := detectTerminalSize(); h > 0 {
					usedDetectH = h
				}
			}
			if usedDetectH > 0 {
				outputHeight = usedDetectH
			} else {
				outputHeight = 24 // fallback
			}
		}
		if debugLog {
			dc.Printf("DBG: CLI output sizing: widthHint=%d detectedWidth=%d height=%d (detectedHeight=%d, flagHeight=%d)\n",
				snapshotWidth, detectedTermWidth, outputHeight, usedDetectH, snapshotHeight)
		}
		// Determine width for rendering
		outputWidth := snapshotWidth
		if outputWidth <= 0 {
			if detectedTermWidth > 0 {
				outputWidth = detectedTermWidth
			}
		}
		appNameVal := cfg.About.Name
		if appNameVal == "" {
			appNameVal = "kvx"
		}
		if searchTerm != "" {
			if debug {
				dc.Printf("DBG: Running CLI search for %q\n", searchTerm)
			}
			if output == "table" {
				rows := ui.SearchRows(node, searchTerm)
				if len(rows) == 0 {
					fmt.Println("No matches found.") //nolint:forbidigo
					return
				}
				fmt.Print(renderBorderedTableRows(rows, noColor, keyW, valueW, outputWidth, appNameVal, "_", node)) //nolint:forbidigo
				if debugLog && len(dc.events) > 0 {
					printDebugEvents(dc.events)
				}
				return
			}
			results := searchCLI(node, searchTerm)
			if len(results) == 0 {
				fmt.Println("No matches found.") //nolint:forbidigo
				return
			}
			// Replace node with search results so output formatting (yaml/json/table) is honored
			node = results
		}
		yamlOpts := yamlFormatOptionsFromConfig(cfg)
		tableOpts := tableFormatOptionsFromConfig(cfg)
		treeOpts := treeFormatOptionsFromConfig(cfg, outputWidth, stdoutIsPiped())
		mermaidOpts := mermaidFormatOptionsFromConfig(cfg)
		printEvalResult(node, output, noColor, keyW, valueW, outputHeight, outputWidth, appNameVal, "_", yamlOpts, tableOpts, treeOpts, mermaidOpts)
		if debugLog && len(dc.events) > 0 {
			printDebugEvents(dc.events)
		}
	},
}

func init() { //nolint:gochecknoinits
	rootCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "start interactive TUI")
	rootCmd.Flags().StringVarP(&output, "output", "o", "auto", "output format: auto|table|list|tree|mermaid|yaml|json|toml|csv|raw")
	rootCmd.Flags().StringVarP(&expression, "expression", "e", "", "CEL expression using '_' as root. Examples: '_.items[0].name', 'type(_)'. For special keys use bracket notation: '_.metadata[\"bad-key\"]'.")
	rootCmd.Flags().StringVar(&searchTerm, "search", "", "Search keys and values (case-insensitive) and display matches")
	// --sort requires a value; default comes from config (or none)
	rootCmd.Flags().StringVar(&sortOrder, "sort", "", "Sort map keys: ascending|asc|descending|desc|none (default from config or none)")
	// No static default here so help doesn't misstate it; default comes from config
	rootCmd.Flags().StringVar(&themeName, "theme", "", "theme name (default from config; see 'kvx themes')")
	rootCmd.Flags().StringVar(&configFile, "config-file", "", "path to a YAML config file (themes, settings)")
	rootCmd.Flags().BoolVar(&configMode, "config", false, "output the merged config (or view in TUI with -i)")
	rootCmd.Flags().BoolVar(&debug, "debug", false, "show debug info in status bar")
	rootCmd.Flags().IntVar(&debugMaxEvents, "debug-max-events", 200, "maximum number of debug events to keep (default: 200)")
	rootCmd.Flags().BoolVar(&noColor, "no-color", false, "disable color output")
	rootCmd.Flags().StringVar(&arrayStyle, "array-style", "none", "Array index style: none, index, numbered, bullet")
	rootCmd.Flags().BoolVar(&renderSnapshot, "snapshot", false, "render a single TUI snapshot and exit (dev/test); honors --width/--height")
	rootCmd.Flags().StringVar(&keyMode, "keymap", "", "keybinding mode: vim (default), emacs, or function")
	rootCmd.Flags().StringArrayVar(&startKeys, "press", nil, "Simulate keys on startup. Use <Key> for special keys (e.g. <F3>, <F6>, <Enter>, <Esc>, <Tab>). Literal text types normally. Examples: --press \"<F3>search\" or --press \"<F6>_.items[0]\"")
	rootCmd.Flags().IntVar(&snapshotWidth, "width", 0, "Output width in columns (affects formatting and TUI layout)")
	rootCmd.Flags().IntVar(&snapshotHeight, "height", 0, "Output height in rows (affects formatting and TUI layout)")
	rootCmd.Flags().IntVar(&limitRecords, "limit", 0, "Limit total number of records displayed")
	rootCmd.Flags().IntVar(&offsetRecords, "offset", 0, "Skip the first N records")
	rootCmd.Flags().IntVar(&tailRecords, "tail", 0, "Show the last N records (mutually exclusive with --limit; ignores --offset)")
	rootCmd.Flags().StringVar(&schemaFile, "schema", "", "path to a JSON Schema file for column display hints (title, maxLength, type, required, deprecated)")
	// Tree output options
	rootCmd.Flags().BoolVar(&treeNoValues, "tree-no-values", false, "Show structure only (hide values) in tree output")
	rootCmd.Flags().IntVar(&treeMaxDepth, "tree-depth", 0, "Limit tree depth (0 = unlimited)")
	rootCmd.Flags().BoolVar(&treeExpandArrays, "tree-expand-arrays", false, "Expand all array elements instead of showing inline/summary")
	rootCmd.Flags().IntVar(&treeMaxStringLen, "tree-max-string", 0, "Max string length in tree output (0=auto, -1=unlimited)")
	// Mermaid output options
	rootCmd.Flags().StringVar(&mermaidDirection, "mermaid-direction", "TD", "Mermaid diagram direction: TD, LR, BT, RL")
	rootCmd.Flags().StringVar(&autoDecode, "auto-decode", "", "Auto-decode serialized scalars: 'lazy' (on navigate), 'eager' (at load), or 'disabled' (default, manual via Enter)")
	_ = rootCmd.Flags().MarkHidden("snapshot-width")
	_ = rootCmd.Flags().MarkHidden("snapshot-height")
	// Hide legacy --config flag from help; use `kvx config` instead
	_ = rootCmd.Flags().MarkHidden("config")
	rootCmd.Version = cliVersionString()
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	rootCmd.AddCommand(versionCmd)
	// Wire config command group
	// Provide --config-file for config commands
	configCmd.PersistentFlags().StringVar(&configFile, "config-file", "", "path to a YAML config file (themes, settings)")
	// Provide output format specifically for `kvx config` (default yaml)
	configCmd.PersistentFlags().StringVarP(&configOutput, "output", "o", "table", "output interface: yaml|json|table|raw")
	// `kvx config get` supports -i
	configGetCmd.Flags().BoolP("interactive", "i", false, "view config in interactive TUI")
	configCmd.AddCommand(configGetCmd)
	// Add both 'theme' and 'themes' (hidden) for listing
	configCmd.AddCommand(configThemeCmd)
	configThemesCmd.Hidden = true
	configCmd.AddCommand(configThemesCmd)
	rootCmd.AddCommand(configCmd)
	// Keep top-level themes for backward compatibility but hide it
	themesCmd.Hidden = true
	rootCmd.AddCommand(themesCmd)
}

func Execute() error {
	return rootCmd.Execute()
}

func hasF10(keys []string) bool {
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if strings.EqualFold(k, "<f10>") || strings.EqualFold(k, "f10") {
			return true
		}
	}
	return false
}

// resolveConfigPath returns the explicit configFile if set, otherwise the XDG path
// ($XDG_CONFIG_HOME/kvx/config.yaml) or ~/.config/kvx/config.yaml if present.
func resolveConfigPath(explicit string) string {
	if explicit != "" {
		return explicit
	}
	xdg := os.Getenv("XDG_CONFIG_HOME")
	candidate := ""
	if xdg != "" {
		candidate = filepath.Join(xdg, "kvx", "config.yaml")
	} else if home, err := os.UserHomeDir(); err == nil {
		candidate = filepath.Join(home, ".config", "kvx", "config.yaml")
	}
	if candidate != "" {
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			return candidate
		}
	}
	return ""
}

func keys(m map[string]ui.ThemeConfig) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// decodeYAMLLenient decodes YAML into a generic interface while allowing duplicate keys by
// keeping the last occurrence. This is used for table/json rendering of configs so we can
// display user files verbatim even if they contain duplicates that would normally fail
// strict unmarshaling.
func decodeYAMLLenient(raw []byte) (interface{}, error) {
	var doc yaml.Node
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(&doc); err != nil {
		return nil, err
	}
	if len(doc.Content) == 0 {
		return nil, nil
	}
	return yamlNodeToInterface(doc.Content[0]), nil
}

func yamlNodeToInterface(n *yaml.Node) interface{} {
	switch n.Kind {
	case yaml.DocumentNode:
		if len(n.Content) > 0 {
			return yamlNodeToInterface(n.Content[0])
		}
		return nil
	case yaml.MappingNode:
		m := make(map[string]interface{}, len(n.Content)/2)
		for i := 0; i < len(n.Content)-1; i += 2 {
			keyNode := n.Content[i]
			valNode := n.Content[i+1]
			key := fmt.Sprint(yamlNodeToInterface(keyNode))
			var val interface{}
			if err := valNode.Decode(&val); err != nil {
				val = yamlNodeToInterface(valNode)
			}
			m[key] = val // last one wins on duplicate keys
		}
		return m
	case yaml.SequenceNode:
		arr := make([]interface{}, 0, len(n.Content))
		for _, c := range n.Content {
			var val interface{}
			if err := c.Decode(&val); err != nil {
				val = yamlNodeToInterface(c)
			}
			arr = append(arr, val)
		}
		return arr
	case yaml.ScalarNode:
		var val interface{}
		if err := n.Decode(&val); err != nil {
			return n.Value
		}
		return val
	case yaml.AliasNode:
		if n.Alias != nil {
			return yamlNodeToInterface(n.Alias)
		}
		return nil
	default:
		return nil
	}
}

func loadHelp(configFile string, km ui.KeyMode) (string, string) {
	helpTitle := "Help"
	helpText := ""
	if km == "" {
		km = ui.DefaultKeyMode
	}
	if cfg, err := loadMergedConfig(configFile); err == nil {
		allowEditInput := true
		if cfg.Features.AllowEditInput != nil {
			allowEditInput = *cfg.Features.AllowEditInput
		}
		menuCfg := ui.MenuFromConfig(cfg.Menu, &allowEditInput)
		baseHelp := strings.TrimSpace(ui.GenerateHelpText(menuCfg, allowEditInput, nil, km))
		if baseHelp == "" {
			defMenu := ui.DefaultMenuConfig()
			baseHelp = strings.TrimSpace(ui.GenerateHelpText(defMenu, true, nil, km))
		}
		helpText = baseHelp
		// Use popup title if configured, but don't use popup text since it may have
		// pre-expanded help with wrong KeyMode from template processing
		if popup := cfg.Menu.F1.Popup; popup.Enabled == nil || *popup.Enabled {
			if strings.TrimSpace(popup.Title) != "" {
				helpTitle = strings.TrimSpace(popup.Title)
			}
		}
	}
	return helpTitle, helpText
}
