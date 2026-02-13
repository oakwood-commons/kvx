package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/go-logr/logr"

	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"
	runewidth "github.com/mattn/go-runewidth"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/oakwood-commons/kvx/internal/formatter"
	ui "github.com/oakwood-commons/kvx/internal/ui"
)

//nolint:gochecknoinits // test setup to initialize theme presets
func init() {
	// Initialize themes with basic configuration for testing.
	// This populates ui.ThemePresets which tests reference.
	cfg := &ui.ThemeConfigFile{
		Themes: map[string]ui.ThemeConfig{
			"dark": {
				KeyColor:       "14",
				ValueColor:     "248",
				HeaderFG:       "15",
				HeaderBG:       "236",
				SelectedFG:     "16",
				SelectedBG:     "240",
				SeparatorColor: "240",
				InputBG:        "236",
				InputFG:        "15",
				StatusColor:    "240",
				StatusError:    "196",
				StatusSuccess:  "46",
				DebugColor:     "246",
				FooterFG:       "15",
				FooterBG:       "240",
				HelpKey:        "14",
				HelpValue:      "248",
			},
			"warm": {
				KeyColor:       "215",
				ValueColor:     "223",
				HeaderFG:       "16",
				HeaderBG:       "94",
				SelectedFG:     "16",
				SelectedBG:     "130",
				SeparatorColor: "130",
				InputBG:        "94",
				InputFG:        "16",
				StatusColor:    "130",
				StatusError:    "196",
				StatusSuccess:  "76",
				DebugColor:     "173",
				FooterFG:       "16",
				FooterBG:       "130",
				HelpKey:        "215",
				HelpValue:      "223",
			},
			"cool": {
				KeyColor:       "39",
				ValueColor:     "51",
				HeaderFG:       "16",
				HeaderBG:       "24",
				SelectedFG:     "16",
				SelectedBG:     "27",
				SeparatorColor: "27",
				InputBG:        "24",
				InputFG:        "16",
				StatusColor:    "27",
				StatusError:    "196",
				StatusSuccess:  "51",
				DebugColor:     "45",
				FooterFG:       "16",
				FooterBG:       "27",
				HelpKey:        "39",
				HelpValue:      "51",
			},
		},
	}
	_ = ui.InitializeThemes(cfg)
}

// captureOutput runs fn while capturing stdout into a string.
func captureOutput(t *testing.T, fn func()) string {
	t.Helper()
	// Save original stdout
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	// Run function
	fn()
	// Restore stdout and close writer
	_ = w.Close()
	os.Stdout = orig
	// Read captured output
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("copy: %v", err)
	}
	_ = r.Close()
	return buf.String()
}

func resetRootCmdState() {
	interactive = false
	output = "table"
	expression = ""
	searchTerm = ""
	debug = false
	noColor = false
	renderSnapshot = false
	helpInteractive = false
	configMode = false
	startKeys = nil
	snapshotWidth = 0
	snapshotHeight = 0
	configFile = ""
	themeName = ""
	schemaFile = ""
	ui.SetMenuConfig(ui.DefaultMenuConfig())

	rootCmd.SetArgs(nil)
	rootCmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
	})
}

func runCLI(t *testing.T, args []string) string {
	t.Helper()
	resetRootCmdState()
	// Isolate from user config by pointing XDG_CONFIG_HOME to a temp dir.
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	tmpDir := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Cleanup(func() {
		if origXDG == "" {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		}
	})
	os.Args = args
	return captureOutput(t, func() {
		if err := Execute(); err != nil {
			t.Fatalf("Execute error: %v", err)
		}
	})
}

func TestCLI_TablePrintsScalarPlain(t *testing.T) {
	// kvx tests/sample.yaml --no-color -e '_.name'
	out := runCLI(t, []string{"kvx", filepath.Join("..", "tests", "sample.yaml"), "--no-color", "-e", "_.name"})
	expected := "kvx-catalog\n"
	if out != expected {
		t.Fatalf("expected %q, got %q", expected, out)
	}
}

func TestCLI_TablePrintsSimpleArrayLines(t *testing.T) {
	// kvx tests/sample.yaml --no-color -e '_.items[0].tags'
	out := runCLI(t, []string{"kvx", filepath.Join("..", "tests", "sample.yaml"), "--no-color", "-e", "_.items[0].tags"})
	expected := "herbal\ncalming\ncaffeine-free\n"
	if out != expected {
		t.Fatalf("expected %q, got %q", expected, out)
	}
}

func TestRenderBorderedTableWithOptionsColumnarNever(t *testing.T) {
	node := []interface{}{
		map[string]interface{}{"name": "Alice", "age": 30},
	}

	out := renderBorderedTableWithOptions(node, true, 0, 0, 80, "kvx", "_", formatter.TableFormatOptions{
		ColumnarMode: "never",
		ArrayStyle:   "index",
	})
	if !strings.Contains(out, "{\"") {
		t.Fatalf("expected key/value JSON string in output, got %q", out)
	}
}

func TestCLI_JSONScalarIsValid(t *testing.T) {
	// kvx tests/sample.yaml -o json -e '_.name'
	out := runCLI(t, []string{"kvx", filepath.Join("..", "tests", "sample.yaml"), "-o", "json", "-e", "_.name"})
	expected := "\"kvx-catalog\"\n"
	if out != expected {
		t.Fatalf("expected %q, got %q", expected, out)
	}
}

func TestCLI_YAMLArrayIsValid(t *testing.T) {
	// kvx tests/sample.yaml -o yaml -e '_.items[0].tags'
	out := runCLI(t, []string{"kvx", filepath.Join("..", "tests", "sample.yaml"), "-o", "yaml", "-e", "_.items[0].tags"})
	expected := "- herbal\n- calming\n- caffeine-free\n"
	if out != expected {
		t.Fatalf("expected %q, got %q", expected, out)
	}
}

func TestCLI_EvaluatesArrayLiteralExpression(t *testing.T) {
	// kvx --no-color -e '[1,2][0]'
	out := runCLI(t, []string{"kvx", "--no-color", "-e", "[1,2][0]"})
	expected := "1\n"
	if out != expected {
		t.Fatalf("expected %q, got %q", expected, out)
	}
}

func TestCLI_EvaluatesFunctionCallExpression(t *testing.T) {
	// kvx tests/sample.yaml --no-color -e 'string(_.active)'
	out := runCLI(t, []string{"kvx", filepath.Join("..", "tests", "sample.yaml"), "--no-color", "-e", "string(_.active)"})
	expected := "true\n"
	if out != expected {
		t.Fatalf("expected %q, got %q", expected, out)
	}
}

func TestCLI_EvaluatesComplexFunctionCallExpression(t *testing.T) {
	// kvx tests/sample.yaml --no-color -e 'string(_.items[0].name.contains("ch"))'
	out := runCLI(t, []string{"kvx", filepath.Join("..", "tests", "sample.yaml"), "--no-color", "-e", "string(_.items[0].name.contains(\"ch\"))"})
	expected := "true\n"
	if out != expected {
		t.Fatalf("expected %q, got %q", expected, out)
	}
}

func TestCLI_EvaluatesMapLiteralExpression(t *testing.T) {
	// kvx --no-color -e '{"a":1,"b":2}.a'
	out := runCLI(t, []string{"kvx", "--no-color", "-e", "{\"a\":1,\"b\":2}.a"})
	expected := "1\n"
	if out != expected {
		t.Fatalf("expected %q, got %q", expected, out)
	}
}

func TestTerminalDeviceNames(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		in  string
		out string
	}{
		"windows": {in: "CONIN$", out: "CONOUT$"},
		"linux":   {in: "/dev/tty", out: "/dev/tty"},
		"darwin":  {in: "/dev/tty", out: "/dev/tty"},
		"freebsd": {in: "/dev/tty", out: "/dev/tty"},
	}

	for goos, expected := range tests {
		goos := goos
		expected := expected
		t.Run(goos, func(t *testing.T) {
			t.Parallel()

			in, out := terminalDeviceNames(goos)
			require.Equal(t, expected.in, in)
			require.Equal(t, expected.out, out)
		})
	}
}

func TestConfigGetTableAllowsDuplicateKeys(t *testing.T) {
	// Prepare a config with a duplicate key to ensure lenient decoding doesn't error
	dupCfg := `app:
  about:
    name: test-kvx
    description: test app
ui:
  theme:
    default: dark
  themes: {}
  help:
    function_examples:
      string.toUpper:
        description: first
        examples: ["one"]
      string.toUpper:
        description: second
        examples: ["two"]
`
	root := t.TempDir()
	cfgDir := filepath.Join(root, "kvx")
	require.NoError(t, os.MkdirAll(cfgDir, 0o755))
	cfgPath := filepath.Join(cfgDir, "config.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(dupCfg), 0o644))
	// Point XDG_CONFIG_HOME at the temp dir so config get picks up this file
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	require.NoError(t, os.Setenv("XDG_CONFIG_HOME", root))
	t.Cleanup(func() {
		if origXDG == "" {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		}
	})

	out := runCLI(t, []string{"kvx", "config", "get"})
	if out == "" {
		t.Fatalf("expected non-empty output for config get")
	}
	if strings.Contains(out, "failed to decode config") {
		t.Fatalf("unexpected decode error in output: %q", out)
	}
}

func TestConfigGetCustomConfigRawPreservesComments(t *testing.T) {
	// Write a custom config with a distinctive comment and value, ensure raw output matches
	cfg := `# custom kvx config
app:
  about:
    name: custom-kvx
    description: custom desc
ui:
  theme:
    default: dark
  themes: {}
`
	root := t.TempDir()
	cfgDir := filepath.Join(root, "kvx")
	require.NoError(t, os.MkdirAll(cfgDir, 0o755))
	cfgPath := filepath.Join(cfgDir, "config.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(cfg), 0o644))

	origXDG := os.Getenv("XDG_CONFIG_HOME")
	require.NoError(t, os.Setenv("XDG_CONFIG_HOME", root))
	t.Cleanup(func() {
		if origXDG == "" {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		}
	})

	out := runCLI(t, []string{"kvx", "config", "get", "-o", "yaml", "--config-file", cfgPath})
	if !strings.Contains(out, "# custom kvx config") {
		t.Fatalf("expected custom comment preserved, got: %q", out)
	}
	if !strings.Contains(out, "custom-kvx") {
		t.Fatalf("expected custom name in output, got: %q", out)
	}
}

func TestCLI_EvaluatesMapLiteralWithJSONOutput(t *testing.T) {
	// kvx -e '{"foo":"bar"}' -o json
	out := runCLI(t, []string{"kvx", "-e", "{\"foo\":\"bar\"}", "-o", "json"})
	// JSON output should contain the map structure
	if !strings.Contains(out, "\"foo\"") || !strings.Contains(out, "\"bar\"") {
		t.Fatalf("expected JSON output with foo:bar, got %q", out)
	}
}

func TestGetProgramOptions_PipedUsesTTYAndCleansUp(t *testing.T) {
	origIsPiped := stdinIsPiped
	origOpenTTY := openTerminalIOFn
	stdinIsPiped = func() bool { return true }

	// Create distinct temp files to stand in for CONIN$/CONOUT$
	inFile, err := os.CreateTemp(t.TempDir(), "tty-in-*")
	require.NoError(t, err)
	outFile, err := os.CreateTemp(t.TempDir(), "tty-out-*")
	require.NoError(t, err)

	openTerminalIOFn = func() (*os.File, *os.File, error) {
		return inFile, outFile, nil
	}

	defer func() {
		stdinIsPiped = origIsPiped
		openTerminalIOFn = origOpenTTY
	}()

	opts, cleanup := getProgramOptions()
	require.NotNil(t, cleanup)
	require.GreaterOrEqual(t, len(opts), 1)

	// Cleanup should close both handles; second close should error
	cleanup()
	require.Error(t, inFile.Close())
	require.Error(t, outFile.Close())
}

func TestGetProgramOptions_NotPipedUsesDefaults(t *testing.T) {
	origIsPiped := stdinIsPiped
	origOpenTTY := openTerminalIOFn
	stdinIsPiped = func() bool { return false }
	openTerminalIOFn = func() (*os.File, *os.File, error) {
		return nil, nil, fmt.Errorf("should not be called")
	}
	defer func() {
		stdinIsPiped = origIsPiped
		openTerminalIOFn = origOpenTTY
	}()

	opts, cleanup := getProgramOptions()
	require.NotNil(t, cleanup)
	require.Nil(t, opts)

	// Cleanup should be a no-op
	require.NotPanics(t, cleanup)
}

// Verify resize watcher emits WindowSizeMsg on size change when stdin is piped.
func TestWithTTYResizeWatcherSendsOnSizeChange(t *testing.T) {
	origTermGetSize := termGetSize
	origTicker := newResizeTicker
	origSend := sendWindowSize
	termCalls := atomic.Int32{}

	termGetSize = func(_ int) (int, int, error) {
		switch termCalls.Add(1) {
		case 1:
			return 80, 24, nil
		default:
			return 81, 24, nil
		}
	}

	ticks := make(chan time.Time, 2)
	newResizeTicker = func(time.Duration) resizeTicker {
		return &fakeResizeTicker{ch: ticks}
	}

	msgs := make(chan tea.WindowSizeMsg, 2)
	sendWindowSize = func(_ *tea.Program, msg tea.WindowSizeMsg) {
		msgs <- msg
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer func() {
		termGetSize = origTermGetSize
		newResizeTicker = origTicker
		sendWindowSize = origSend
	}()

	_, out := makePipe(t)
	opt := withTTYResizeWatcher(ctx, out)
	var p tea.Program
	opt(&p)

	// Trigger two ticks: first sets baseline, second should emit change
	ticks <- time.Now()
	ticks <- time.Now()

	recv := func() tea.WindowSizeMsg {
		select {
		case m := <-msgs:
			return m
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("timed out waiting for resize message")
			return tea.WindowSizeMsg{}
		}
	}

	first := recv()
	if first.Width != 80 || first.Height != 24 {
		t.Fatalf("unexpected first size: %+v", first)
	}
	second := recv()
	if second.Width != 81 || second.Height != 24 {
		t.Fatalf("expected width change to 81, got %+v", second)
	}
}

func TestWithTTYResizeWatcherSkipsUnchangedSize(t *testing.T) {
	origTermGetSize := termGetSize
	origTicker := newResizeTicker
	origSend := sendWindowSize
	termCalls := atomic.Int32{}

	termGetSize = func(_ int) (int, int, error) {
		switch termCalls.Add(1) {
		case 1, 2:
			return 80, 24, nil
		default:
			return 81, 24, nil
		}
	}

	ticks := make(chan time.Time, 3)
	newResizeTicker = func(time.Duration) resizeTicker {
		return &fakeResizeTicker{ch: ticks}
	}

	msgs := make(chan tea.WindowSizeMsg, 2)
	sendWindowSize = func(_ *tea.Program, msg tea.WindowSizeMsg) {
		msgs <- msg
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer func() {
		termGetSize = origTermGetSize
		newResizeTicker = origTicker
		sendWindowSize = origSend
	}()

	_, out := makePipe(t)
	opt := withTTYResizeWatcher(ctx, out)
	var p tea.Program
	opt(&p)

	recv := func() tea.WindowSizeMsg {
		select {
		case m := <-msgs:
			return m
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("timed out waiting for resize message")
			return tea.WindowSizeMsg{}
		}
	}

	ticks <- time.Now()
	first := recv()
	if first.Width != 80 || first.Height != 24 {
		t.Fatalf("unexpected first size: %+v", first)
	}

	ticks <- time.Now()
	select {
	case m := <-msgs:
		t.Fatalf("unexpected resize message on unchanged size: %+v", m)
	case <-time.After(150 * time.Millisecond):
	}

	ticks <- time.Now()
	second := recv()
	if second.Width != 81 || second.Height != 24 {
		t.Fatalf("expected width change to 81 after size change, got %+v", second)
	}
}

type fakeResizeTicker struct {
	ch <-chan time.Time
}

func (f *fakeResizeTicker) C() <-chan time.Time { return f.ch }
func (f *fakeResizeTicker) Stop()               {}

func makePipe(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	return r, w
}

// Unit tests for suggestion logic (avoid testing os.Exit error path directly)
func TestSuggestion_NameKey(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "tests", "sample.yaml"))
	if err != nil {
		t.Fatalf("read sample: %v", err)
	}
	var root interface{}
	if err := yaml.Unmarshal(data, &root); err != nil {
		t.Fatalf("yaml: %v", err)
	}
	hint := buildSuggestion("name", root)
	if hint == "" || !bytes.Contains([]byte(hint), []byte("_.name")) {
		t.Fatalf("expected hint for _.name, got %q", hint)
	}
}

func TestSuggestion_ItemsNumericIndex(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "tests", "sample.yaml"))
	if err != nil {
		t.Fatalf("read sample: %v", err)
	}
	var root interface{}
	if err := yaml.Unmarshal(data, &root); err != nil {
		t.Fatalf("yaml: %v", err)
	}
	hint := buildSuggestion("items.0.name", root)
	if hint == "" || !bytes.Contains([]byte(hint), []byte("_.items[0].name")) {
		t.Fatalf("expected hint for _.items[0].name, got %q", hint)
	}
}

func TestValidateLimitingFlagsConflicts(t *testing.T) {
	resetRootCmdState()
	limitRecords = 1
	tailRecords = 1
	err := validateLimitingFlags()
	require.Error(t, err)
	require.Contains(t, err.Error(), "mutually exclusive")
}

func TestValidateLimitingFlagsNegative(t *testing.T) {
	resetRootCmdState()
	limitRecords = -1
	err := validateLimitingFlags()
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-negative")
}

func TestCLI_LimitingJSON(t *testing.T) {
	// limit applies after expression
	out := runCLI(t, []string{"kvx", filepath.Join("..", "tests", "sample.yaml"), "-o", "json", "-e", "_.items", "--limit", "2"})
	var arr []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &arr))
	require.Len(t, arr, 2)
	require.Equal(t, "chamomile", arr[0]["name"])
	require.Equal(t, "earl-grey", arr[1]["name"])
}

func TestCLI_LimitingOffsetJSON(t *testing.T) {
	out := runCLI(t, []string{"kvx", filepath.Join("..", "tests", "sample.yaml"), "-o", "json", "-e", "_.items", "--offset", "1", "--limit", "1"})
	var arr []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &arr))
	require.Len(t, arr, 1)
	require.Equal(t, "earl-grey", arr[0]["name"])
}

func TestCLI_LimitingTailIgnoresOffsetJSON(t *testing.T) {
	out := runCLI(t, []string{"kvx", filepath.Join("..", "tests", "sample.yaml"), "-o", "json", "-e", "_.items", "--tail", "1", "--offset", "2"})
	var arr []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &arr))
	require.Len(t, arr, 1)
	require.Equal(t, "matcha", arr[0]["name"])
}

func TestSuggestion_SpecialCharKey(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "tests", "sample.yaml"))
	if err != nil {
		t.Fatalf("read sample: %v", err)
	}
	var root interface{}
	if err := yaml.Unmarshal(data, &root); err != nil {
		t.Fatalf("yaml: %v", err)
	}
	hint := buildSuggestion("metadata.bad-key", root)
	if hint == "" || !bytes.Contains([]byte(hint), []byte("_.metadata[\"bad-key\"]")) {
		t.Fatalf("expected hint for _.metadata[\"bad-key\"], got %q", hint)
	}
}

func TestSetThemeExposedForExternalUse(t *testing.T) {
	orig := ui.CurrentTheme()
	defer ui.SetTheme(orig)

	lightTheme, _ := ui.GetTheme("light")
	ui.SetTheme(lightTheme)
	th := ui.CurrentTheme()
	if th.HeaderFG != lightTheme.HeaderFG {
		t.Fatalf("expected CurrentTheme to reflect SetTheme for external consumers")
	}
}

func TestCLI_NoInputShowsHelp(t *testing.T) {
	out := runCLI(t, []string{"kvx"})
	// With no input and no expression, kvx should show help text
	if !strings.Contains(out, "Usage:") || !strings.Contains(out, "kvx [file]") {
		t.Fatalf("expected help output, got %q", out)
	}
	// Should NOT show an empty table
	if strings.Contains(out, "(value)") && strings.Contains(out, "{}") {
		t.Fatalf("should not show empty table, got %q", out)
	}
	// Should contain command examples
	if !strings.Contains(out, "Examples:") {
		t.Fatalf("expected Examples section in help, got %q", out)
	}
	// Should contain flags
	if !strings.Contains(out, "Flags:") {
		t.Fatalf("expected Flags section in help, got %q", out)
	}
}

func TestCLI_NoInputWithExpressionUsesEmptyObject(t *testing.T) {
	// With expression but no input, should evaluate against empty object
	out := runCLI(t, []string{"kvx", "-o", "json", "-e", "type(_)"})
	expected := "\"map\"\n"
	if out != expected {
		t.Fatalf("expected %q, got %q", expected, out)
	}
}

func TestCLI_NoInputWithExpressionEmptySize(t *testing.T) {
	// size({}) should return 0
	out := runCLI(t, []string{"kvx", "-o", "json", "-e", "size(_)"})
	expected := "0\n"
	if out != expected {
		t.Fatalf("expected %q, got %q", expected, out)
	}
}

func TestCLI_SnapshotNoColorHasNoANSI(t *testing.T) {
	out := runCLI(t, []string{
		"kvx",
		filepath.Join("..", "tests", "sample.yaml"),
		"--snapshot",
		"--no-color",
		"--width", "40",
		"--height", "12",
	})
	// Strip inverse video codes (\x1b[7m, \x1b[m) which are allowed for highlight bar
	outNoInverse := strings.ReplaceAll(out, "\x1b[7m", "")
	outNoInverse = strings.ReplaceAll(outNoInverse, "\x1b[m", "")
	if strings.Contains(outNoInverse, "\x1b[") {
		t.Fatalf("expected no ANSI color codes (except inverse video) in snapshot output with --no-color, got:\n%s", out)
	}
	if !strings.Contains(out, "KEY") || !strings.Contains(out, "VALUE") {
		t.Fatalf("expected snapshot output to include table headers, got:\n%s", out)
	}
}

func TestConfigOutputsMenuPopup(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "cfg.yaml")
	content := `
default_theme: default
menu:
  f12:
    label: "custom"
    action: "custom"
    enabled: true
    popup:
      text: "Hello modal"
      anchor: top
      modal: true
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	mergedCfg, err := loadMergedConfig(cfgPath)
	if err != nil {
		t.Fatalf("load merged config: %v", err)
	}
	sanitized := sanitizeConfig(mergedCfg)
	outBytes, err := yaml.Marshal(sanitized)
	if err != nil {
		t.Fatalf("marshal sanitized config: %v", err)
	}
	out := string(outBytes)
	if !strings.Contains(out, "popup:") || !strings.Contains(out, "Hello modal") {
		t.Fatalf("expected popup object in config output, got: %s", out)
	}
}

func TestLoadMergedConfigUsesEmbeddedDefaults(t *testing.T) {
	cfg, err := loadMergedConfig("")
	require.NoError(t, err)

	require.NotEmpty(t, cfg.Themes, "embedded default config should include themes")
	if cfg.Theme.Default != "midnight" {
		t.Fatalf("expected default theme to come from embedded config (midnight), got %q", cfg.Theme.Default)
	}
	if _, ok := cfg.Themes["midnight"]; !ok {
		t.Fatalf("expected embedded themes to include midnight theme, got keys: %v", keys(cfg.Themes))
	}
}

func TestCLI_RenderSnapshotFlag(t *testing.T) {
	// kvx tests/sample.yaml --snapshot --no-color --width 40 --height 12
	out := runCLI(t, []string{
		"kvx",
		filepath.Join("..", "tests", "sample.yaml"),
		"--snapshot",
		"--no-color",
		"--width", "40",
		"--height", "12",
	})

	// Strip inverse video codes (\x1b[7m, \x1b[m) which are allowed for highlight bar
	outNoInverse := strings.ReplaceAll(out, "\x1b[7m", "")
	outNoInverse = strings.ReplaceAll(outNoInverse, "\x1b[m", "")
	if strings.Contains(outNoInverse, "\x1b[") {
		t.Fatalf("expected snapshot output to be colorless (except inverse video), got:\n%s", out)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 12 {
		t.Fatalf("expected snapshot output to use 12 lines, got %d\n%s", len(lines), out)
	}
	if !strings.Contains(out, "KEY") || !strings.Contains(out, "VALUE") {
		t.Fatalf("expected snapshot output to include data panel headers, got:\n%s", out)
	}
}

func TestSnapshotModeUsesMenuConfigFromDefaultConfig(t *testing.T) {
	out := runCLI(t, []string{
		"kvx",
		filepath.Join("..", "tests", "sample.yaml"),
		"--snapshot",
		"--width", "80",
		"--height", "30",
		"--no-color",
	})

	if !strings.Contains(out, "KEY") || !strings.Contains(out, "VALUE") {
		t.Fatalf("expected snapshot to include table headers, got:\n%s", out)
	}
	// Default is vim mode, check for vim-style keys or F-keys
	if !strings.Contains(strings.ToLower(out), "help") {
		t.Fatalf("expected footer to include help hint, got:\n%s", out)
	}
}

func TestSnapshotWidthHeightRespectFlags(t *testing.T) {
	out := runCLI(t, []string{
		"kvx",
		filepath.Join("..", "tests", "sample.yaml"),
		"--snapshot",
		"--no-color",
		"--width", "40",
		"--height", "8",
	})

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	require.Equal(t, 8, len(lines))

	maxWidth := 0
	for _, line := range lines {
		// Strip ANSI escape codes before measuring width
		cleanLine := ansiStripRe.ReplaceAllString(line, "")
		if w := runewidth.StringWidth(cleanLine); w > maxWidth {
			maxWidth = w
		}
	}
	require.LessOrEqual(t, maxWidth, 40, "snapshot lines should not exceed requested width")
}

func TestSnapshotAndInteractiveUseSameMenuConfig(t *testing.T) {
	out := runCLI(t, []string{
		"kvx",
		filepath.Join("..", "tests", "sample.yaml"),
		"--snapshot",
		"--width", "80",
		"--height", "30",
		"--press", "<f1>",
		"--no-color",
	})

	if !strings.Contains(strings.ToLower(out), "help") {
		t.Fatalf("expected help panel in snapshot output, got:\n%s", out)
	}
	if !strings.Contains(out, "KEY") || !strings.Contains(out, "VALUE") {
		t.Fatalf("expected data panel headers in snapshot output, got:\n%s", out)
	}
}

func TestSnapshotStartKeyF1ShowsHelp(t *testing.T) {
	out := runCLI(t, []string{
		"kvx",
		filepath.Join("..", "tests", "sample.yaml"),
		"--snapshot",
		"--width", "80",
		"--height", "24",
		"--press", "<F1>",
		"--no-color",
	})

	if !strings.Contains(strings.ToLower(out), "help") {
		t.Fatalf("expected help text when start key includes F1, got:\n%s", out)
	}
	// Help overlay may occupy the main panel; no strict header requirement here.
}

// ansiStripRe strips ANSI escape sequences for width measurements.
var ansiStripRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\[m`)

func TestConfigSnapshotRespectsWidthHeight(t *testing.T) {
	out := runCLI(t, []string{
		"kvx",
		"--config",
		"--snapshot",
		"--no-color",
		"--width", "50",
		"--height", "8",
	})

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	require.Equal(t, 8, len(lines))

	maxWidth := 0
	for _, line := range lines {
		// Strip ANSI escape codes before measuring width
		cleanLine := ansiStripRe.ReplaceAllString(line, "")
		if w := runewidth.StringWidth(cleanLine); w > maxWidth {
			maxWidth = w
		}
	}
	require.LessOrEqual(t, maxWidth, 50, "config snapshot lines should not exceed requested width")
	require.Contains(t, out, "app")
	require.Contains(t, out, "ui")
}

func TestStartKeysF10BypassesTUI_TableOutput(t *testing.T) {
	out := runCLI(t, []string{
		"kvx",
		filepath.Join("..", "tests", "sample.yaml"),
		"--press", "<F10>",
		"--no-color",
		"-e", "_.name",
	})

	require.Equal(t, "kvx-catalog\n", out)
}

func TestStartKeysF10RespectsJSONOutput(t *testing.T) {
	out := runCLI(t, []string{
		"kvx",
		filepath.Join("..", "tests", "sample.yaml"),
		"--press", "<F10>",
		"--no-color",
		"-o", "json",
		"-e", "_.name",
	})

	require.Equal(t, "\"kvx-catalog\"\n", out)
}

func TestVersionCommand(t *testing.T) {
	rootCmd.SetArgs([]string{"version"})
	out := captureOutput(t, func() {
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Execute error: %v", err)
		}
	})
	if !strings.Contains(out, "kvx") {
		t.Fatalf("expected version output to contain kvx, got %q", out)
	}
}

func TestRootFlagVersion(t *testing.T) {
	rootCmd.SetArgs([]string{"--version"})
	out := captureOutput(t, func() {
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Execute error: %v", err)
		}
	})
	if !strings.Contains(out, "kvx") {
		t.Fatalf("expected --version output to contain kvx, got %q", out)
	}
}

func TestClassifyNode(t *testing.T) {
	tests := []struct {
		name           string
		value          interface{}
		wantCollection bool
		wantSimpleArr  bool
	}{
		{name: "map", value: map[string]interface{}{"a": 1}, wantCollection: true},
		{name: "empty array", value: []interface{}{}, wantCollection: true},
		{name: "scalar array", value: []interface{}{1, 2}, wantSimpleArr: true},
		{name: "mixed array", value: []interface{}{1, map[string]interface{}{"a": 1}}, wantCollection: true},
		{name: "scalar", value: "x", wantCollection: false, wantSimpleArr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCollection, gotSimple := classifyNode(tt.value)
			if gotCollection != tt.wantCollection || gotSimple != tt.wantSimpleArr {
				t.Fatalf("classifyNode(%T) = (%v,%v), want (%v,%v)", tt.value, gotCollection, gotSimple, tt.wantCollection, tt.wantSimpleArr)
			}
		})
	}
}

func TestCLI_CSVFileParsing(t *testing.T) {
	// Test that CSV files are parsed correctly as array of objects
	out := runCLI(t, []string{"kvx", filepath.Join("..", "examples", "data", "sample_employees.csv"), "--no-color", "-e", "_[0].name"})
	expected := "Alice Johnson\n"
	if out != expected {
		t.Fatalf("expected first row name to be %q, got %q", expected, out)
	}
}

func TestCLI_CSVFileAccessingFields(t *testing.T) {
	// Test accessing fields from CSV row objects
	out := runCLI(t, []string{"kvx", filepath.Join("..", "examples", "data", "sample_employees.csv"), "--no-color", "-e", "_[0].department"})
	expected := "Engineering\n"
	if out != expected {
		t.Fatalf("expected first row department to be %q, got %q", expected, out)
	}
}

func TestCLI_CSVFileArrayLength(t *testing.T) {
	// Test that CSV file produces an array with correct length
	out := runCLI(t, []string{"kvx", filepath.Join("..", "examples", "data", "sample_employees.csv"), "--no-color", "-e", "size(_)"})
	expected := "8\n"
	if out != expected {
		t.Fatalf("expected CSV array length to be 8, got %q", out)
	}
}

func TestParseCSV(t *testing.T) {
	csvData := []byte("name,age,city\nAlice,30,New York\nBob,25,London")
	root, err := parseCSV(csvData)
	if err != nil {
		t.Fatalf("parseCSV failed: %v", err)
	}
	rows, ok := root.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", root)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	row1, ok := rows[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %T", rows[0])
	}
	if row1["name"] != "Alice" {
		t.Fatalf("expected name=Alice, got %v", row1["name"])
	}
	if row1["age"] != "30" {
		t.Fatalf("expected age=30, got %v", row1["age"])
	}
	if row1["city"] != "New York" {
		t.Fatalf("expected city=New York, got %v", row1["city"])
	}
}

func TestIsCSVFile(t *testing.T) {
	tests := []struct {
		filePath string
		want     bool
	}{
		{"test.csv", true},
		{"test.CSV", true},
		{"data.csv", true},
		{"test.yaml", false},
		{"test.json", false},
		{"test", false},
		{"/path/to/file.csv", true},
		{"/path/to/file.txt", false},
	}
	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			got := isCSVFile(tt.filePath)
			if got != tt.want {
				t.Fatalf("isCSVFile(%q) = %v, want %v", tt.filePath, got, tt.want)
			}
		})
	}
}

// TestIsTOMLFile and TestParseTOML removed — TOML extension/parsing
// is now handled inside the loader package with fallback support.

// Test debug collector basic functionality
func TestDebugCollector_Record(t *testing.T) {
	dc := newDebugCollector(true, 100)
	dc.Printf("test message %d", 1)
	dc.Println("test message 2")
	dc.Append("test message 3")

	if len(dc.events) != 3 {
		t.Errorf("expected 3 events, got %d", len(dc.events))
	}
}

func TestDebugCollector_Disabled(t *testing.T) {
	dc := newDebugCollector(false, 100)
	dc.Printf("test")
	dc.Println("test")
	dc.Append("test")

	if len(dc.events) != 0 {
		t.Errorf("expected 0 events when disabled, got %d", len(dc.events))
	}
}

func TestDebugCollector_MaxEvents(t *testing.T) {
	dc := newDebugCollector(true, 5)

	// Add 10 events
	for i := 0; i < 10; i++ {
		dc.Printf("event %d", i)
	}

	// Should only keep last 5
	if len(dc.events) != 5 {
		t.Errorf("expected 5 events (max), got %d", len(dc.events))
	}

	// Should keep the most recent ones (5-9)
	if !strings.Contains(dc.events[0].Message, "event 5") {
		t.Errorf("expected oldest kept event to be event 5, got %q", dc.events[0].Message)
	}
}

func TestDebugCollector_Writer(t *testing.T) {
	dc := newDebugCollector(true, 100)
	w := dc.Writer()

	n, err := w.Write([]byte("test message"))
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if n != 12 {
		t.Errorf("expected 12 bytes written, got %d", n)
	}
	if len(dc.events) != 1 {
		t.Errorf("expected 1 event, got %d", len(dc.events))
	}
}

func TestDebugCollector_WriterDisabled(t *testing.T) {
	dc := newDebugCollector(false, 100)
	w := dc.Writer()

	n, err := w.Write([]byte("test"))
	if err != nil {
		t.Errorf("Write should not fail when disabled: %v", err)
	}
	if n != 4 {
		t.Errorf("expected 4 bytes reported, got %d", n)
	}
	if len(dc.events) != 0 {
		t.Errorf("expected 0 events when disabled, got %d", len(dc.events))
	}
}

// Test isAlphaNumOrUnderscore helper
func TestIsAlphaNumOrUnderscore(t *testing.T) {
	tests := []struct {
		r    rune
		want bool
	}{
		{'a', true},
		{'Z', true},
		{'0', true},
		{'9', true},
		{'_', true},
		{'-', false},
		{'.', false},
		{' ', false},
		{'@', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.r), func(t *testing.T) {
			got := isAlphaNumOrUnderscore(tt.r)
			if got != tt.want {
				t.Errorf("isAlphaNumOrUnderscore(%q) = %v, want %v", tt.r, got, tt.want)
			}
		})
	}
}

// Test loadInputData with no args and no stdin returns errShowHelp
func TestLoadInputData_NoInputShowsHelp(t *testing.T) {
	dc := newDebugCollector(false, 100)
	_, _, err := loadInputData([]string{}, "", false, dc, logr.Discard())
	if !errors.Is(err, errShowHelp) {
		t.Errorf("expected errShowHelp, got %v", err)
	}
}

// Test loadInputData with file argument
func TestLoadInputData_WithFile(t *testing.T) {
	// Create temp JSON file
	tmpFile, err := os.CreateTemp("", "test-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	testData := `{"test": "value"}`
	if _, err := tmpFile.WriteString(testData); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	dc := newDebugCollector(false, 100)
	root, fromStdin, err := loadInputData([]string{tmpFile.Name()}, "", false, dc, logr.Discard())
	if err != nil {
		t.Errorf("loadInputData with file failed: %v", err)
	}
	if fromStdin {
		t.Error("expected fromStdin to be false for file input")
	}
	m, ok := root.(map[string]interface{})
	if !ok {
		t.Errorf("expected map, got %T", root)
	}
	if m["test"] != "value" {
		t.Errorf("expected test=value, got %v", m["test"])
	}
}

// Test loadInputData with CSV file
func TestLoadInputData_WithCSVFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	csvData := "name,age\nAlice,30\nBob,25"
	if _, err := tmpFile.WriteString(csvData); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	dc := newDebugCollector(false, 100)
	root, fromStdin, err := loadInputData([]string{tmpFile.Name()}, "", false, dc, logr.Discard())
	if err != nil {
		t.Errorf("loadInputData with CSV failed: %v", err)
	}
	if fromStdin {
		t.Error("expected fromStdin to be false")
	}

	// Should be parsed as array of maps
	arr, ok := root.([]interface{})
	if !ok {
		t.Fatalf("expected array for CSV, got %T", root)
	}
	if len(arr) != 2 {
		t.Errorf("expected 2 rows, got %d", len(arr))
	}
}

// Test loadInputData with expression but no input
func TestLoadInputData_WithExpressionNoInput(t *testing.T) {
	dc := newDebugCollector(false, 100)
	root, fromStdin, err := loadInputData([]string{}, "_.test", false, dc, logr.Discard())
	if err != nil {
		t.Errorf("loadInputData with expression failed: %v", err)
	}
	if fromStdin {
		t.Error("expected fromStdin to be false")
	}
	// Should default to empty object
	m, ok := root.(map[string]interface{})
	if !ok {
		t.Errorf("expected map, got %T", root)
	}
	if len(m) != 0 {
		t.Errorf("expected empty map, got %d entries", len(m))
	}
}

// Test bracket notation on multi-doc YAML
func TestCLI_BracketNotationOnMultiDocYAML(t *testing.T) {
	// kvx tests/sample-multi.yaml --no-color -e '_[0].title'
	out := runCLI(t, []string{"kvx", filepath.Join("..", "tests", "sample-multi.yaml"), "--no-color", "-e", "_[0].title"})
	expected := "Log Entry 1\n"
	if out != expected {
		t.Fatalf("expected %q, got %q", expected, out)
	}
}

// Test bracket notation on array element in multi-doc YAML
func TestCLI_AccessArrayElementByIndex(t *testing.T) {
	// kvx tests/sample-multi.yaml --no-color -e '_[1].message'
	out := runCLI(t, []string{"kvx", filepath.Join("..", "tests", "sample-multi.yaml"), "--no-color", "-e", "_[1].message"})
	expected := "High memory usage detected\n"
	if out != expected {
		t.Fatalf("expected %q, got %q", expected, out)
	}
}

// Test that arrow keys work in expression input mode (requires F6 to activate)
// Note: -e evaluates an expression and navigates to it, but does not enter expression input mode.
// To edit the expression, press F6 first. Without F6, arrow keys are not captured by the input
// widget and terminal escape sequences will be echoed.
func TestSnapshotExpressionArrowKeysRequireF6(t *testing.T) {
	out := runCLI(t, []string{
		"kvx",
		filepath.Join("..", "tests", "sample-multi.yaml"),
		"--snapshot",
		"--no-color",
		"--width", "80",
		"--height", "30",
		"--press", "<F6><Home><Left><Left>",
	})

	// After F6 (enter expression mode), Home (go to start), and Left arrow twice,
	// the expression should show the modified input without arrow key escape sequences.
	// Note: We allow \x1b[7m (reverse video) and \x1b[m (reset) for selection highlighting.
	// We specifically check for arrow key escape codes that would indicate unhandled input.
	arrowCodes := []string{"^[[A", "^[[B", "^[[C", "^[[D", "\x1b[A", "\x1b[B", "\x1b[C", "\x1b[D"}
	for _, code := range arrowCodes {
		if strings.Contains(out, code) {
			t.Fatalf("expected no arrow key escape sequences in expression input after F6, found %q", code)
		}
	}
}

// List format tests
func TestCLI_ListOutputScalarValue(t *testing.T) {
	// kvx tests/sample.yaml --no-color -o list -e '_.name'
	out := runCLI(t, []string{"kvx", filepath.Join("..", "tests", "sample.yaml"), "--no-color", "-o", "list", "-e", "_.name"})
	expected := "value: kvx-catalog\n"
	if out != expected {
		t.Fatalf("expected %q, got %q", expected, out)
	}
}

func TestCLI_ListOutputScalarNumber(t *testing.T) {
	// kvx tests/sample.yaml --no-color -o list -e '_.items[0].price'
	out := runCLI(t, []string{"kvx", filepath.Join("..", "tests", "sample.yaml"), "--no-color", "-o", "list", "-e", "_.items[0].price"})
	expected := "value: 12.99\n"
	if out != expected {
		t.Fatalf("expected %q, got %q", expected, out)
	}
}

func TestCLI_ListOutputScalarArray(t *testing.T) {
	// kvx tests/sample.yaml --no-color -o list -e '_.items[0].tags'
	out := runCLI(t, []string{"kvx", filepath.Join("..", "tests", "sample.yaml"), "--no-color", "-o", "list", "-e", "_.items[0].tags"})
	// Scalar arrays should be one per line (same as table)
	expected := "herbal\ncalming\ncaffeine-free\n"
	if out != expected {
		t.Fatalf("expected %q, got %q", expected, out)
	}
}

func TestCLI_ListOutputMap(t *testing.T) {
	// kvx tests/sample.yaml --no-color -o list -e '_.metadata'
	out := runCLI(t, []string{"kvx", filepath.Join("..", "tests", "sample.yaml"), "--no-color", "-o", "list", "-e", "_.metadata"})

	// Map should have indented key/value pairs
	if !strings.Contains(out, "author: abaker9") {
		t.Fatalf("expected 'author: abaker9' in output, got %q", out)
	}
	if !strings.Contains(out, "created: 2025-12-10") {
		t.Fatalf("expected 'created: 2025-12-10' in output, got %q", out)
	}
	if !strings.Contains(out, "verified: true") {
		t.Fatalf("expected 'verified: true' in output, got %q", out)
	}
}

func TestCLI_ListOutputArrayOfObjects(t *testing.T) {
	// kvx tests/sample.yaml --no-color -o list --array-style index -e '_.items'
	out := runCLI(t, []string{"kvx", filepath.Join("..", "tests", "sample.yaml"), "--no-color", "-o", "list", "--array-style", "index", "-e", "_.items"})

	// Should have index headers
	if !strings.Contains(out, "[0]") {
		t.Fatalf("expected '[0]' in output, got %q", out)
	}
	if !strings.Contains(out, "[1]") {
		t.Fatalf("expected '[1]' in output, got %q", out)
	}

	// Should have indented properties
	if !strings.Contains(out, "  name: chamomile") {
		t.Fatalf("expected '  name: chamomile' in output, got %q", out)
	}
	if !strings.Contains(out, "  price: 12.99") {
		t.Fatalf("expected '  price: 12.99' in output, got %q", out)
	}
	if !strings.Contains(out, "  origin: egypt") {
		t.Fatalf("expected '  origin: egypt' in output, got %q", out)
	}
}

func TestCLI_ListOutputEmptyArray(t *testing.T) {
	// kvx --no-color -o list -e '[]'
	out := runCLI(t, []string{"kvx", "--no-color", "-o", "list", "-e", "[]"})
	expected := ""
	if out != expected {
		t.Fatalf("expected empty output for empty array, got %q", out)
	}
}

func TestCLI_ListOutputEmptyMap(t *testing.T) {
	// kvx --no-color -o list -e '{}'
	out := runCLI(t, []string{"kvx", "--no-color", "-o", "list", "-e", "{}"})
	expected := ""
	if out != expected {
		t.Fatalf("expected empty output for empty map, got %q", out)
	}
}

func TestCLI_ListOutputSameAsTableForScalarArray(t *testing.T) {
	// For scalar arrays, list and table should produce identical output
	tableOut := runCLI(t, []string{"kvx", filepath.Join("..", "tests", "sample.yaml"), "--no-color", "-o", "table", "-e", "_.items[0].tags"})
	listOut := runCLI(t, []string{"kvx", filepath.Join("..", "tests", "sample.yaml"), "--no-color", "-o", "list", "-e", "_.items[0].tags"})

	if tableOut != listOut {
		t.Fatalf("expected list output to match table output for scalar array, table=%q, list=%q", tableOut, listOut)
	}
}

// Tree format tests
func TestCLI_TreeOutputBasic(t *testing.T) {
	out := runCLI(t, []string{"kvx", filepath.Join("..", "tests", "sample.yaml"), "-o", "tree"})
	// Should have tree structure markers
	if !strings.Contains(out, "├──") && !strings.Contains(out, "└──") {
		t.Fatalf("expected tree box-drawing characters, got %q", out)
	}
	// Should contain key-value data
	if !strings.Contains(out, "name: kvx-catalog") {
		t.Fatalf("expected 'name: kvx-catalog' in tree output, got %q", out)
	}
}

func TestCLI_TreeOutputNested(t *testing.T) {
	out := runCLI(t, []string{"kvx", filepath.Join("..", "tests", "sample.yaml"), "-o", "tree", "-e", "_.metadata"})
	// Should show metadata keys
	if !strings.Contains(out, "author: abaker9") {
		t.Fatalf("expected 'author: abaker9', got %q", out)
	}
	if !strings.Contains(out, "verified: true") {
		t.Fatalf("expected 'verified: true', got %q", out)
	}
}

func TestCLI_TreeOutputNoValues(t *testing.T) {
	out := runCLI(t, []string{"kvx", filepath.Join("..", "tests", "sample.yaml"), "-o", "tree", "--tree-no-values", "-e", "_.metadata"})
	// Should have keys but not values
	if !strings.Contains(out, "author") {
		t.Fatalf("expected 'author' key, got %q", out)
	}
	// Should NOT have the value
	if strings.Contains(out, "abaker9") {
		t.Fatalf("should not contain value 'abaker9' with --tree-no-values, got %q", out)
	}
}

func TestCLI_TreeOutputDepth(t *testing.T) {
	out := runCLI(t, []string{"kvx", filepath.Join("..", "tests", "sample.yaml"), "-o", "tree", "--tree-depth", "1"})
	// At depth 1, should truncate children with ...
	if !strings.Contains(out, "...") {
		t.Fatalf("expected '...' truncation at depth 1, got %q", out)
	}
}

func TestCLI_TreeOutputArrays(t *testing.T) {
	out := runCLI(t, []string{"kvx", filepath.Join("..", "tests", "sample.yaml"), "-o", "tree", "--array-style", "index", "-e", "_.items"})
	// Should have indexed array entries
	if !strings.Contains(out, "[0]") {
		t.Fatalf("expected '[0]' array index, got %q", out)
	}
	if !strings.Contains(out, "[1]") {
		t.Fatalf("expected '[1]' array index, got %q", out)
	}
}

func TestCLI_TreeOutputScalarArrayInline(t *testing.T) {
	out := runCLI(t, []string{"kvx", filepath.Join("..", "tests", "sample.yaml"), "-o", "tree", "-e", "_.regions.asia"})
	// Short scalar arrays should be inline
	if !strings.Contains(out, "countries: [japan, india, china]") {
		t.Fatalf("expected inline array 'countries: [japan, india, china]', got %q", out)
	}
}
