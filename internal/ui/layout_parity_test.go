package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oakwood-commons/kvx/internal/navigator"
)

func TestPanelLayoutBaseline(t *testing.T) {
	t.Parallel()
	node := sampleLayoutNode()
	width := 60
	height := 16
	got := renderInteractiveView(t, node, width, height)
	goldenPath := filepath.Join("testdata", "layout", "panel-interactive-60x16.txt")
	assertGolden(t, goldenPath, got)
}

func TestPanelLayoutBaselineSearch(t *testing.T) {
	t.Parallel()
	node := sampleLayoutNode()
	width := 60
	height := 16
	got := renderInteractiveViewWith(t, node, width, height, configureSearchScenario)
	goldenPath := filepath.Join("testdata", "layout", "panel-search-60x16.txt")
	assertGolden(t, goldenPath, got)
}

func TestPanelLayoutBaselineFilter(t *testing.T) {
	t.Parallel()
	node := sampleLayoutNode()
	width := 60
	height := 16
	got := renderInteractiveViewWith(t, node, width, height, configureFilterScenario)
	goldenPath := filepath.Join("testdata", "layout", "panel-filter-60x16.txt")
	assertGolden(t, goldenPath, got)
}

func TestPanelLayoutBaselineInput(t *testing.T) {
	t.Parallel()
	node := sampleLayoutNode()
	width := 60
	height := 16
	got := renderInteractiveViewWith(t, node, width, height, configureInputScenario)
	goldenPath := filepath.Join("testdata", "layout", "panel-input-60x16.txt")
	assertGolden(t, goldenPath, got)
}

func TestPanelLayoutBaselineHelp(t *testing.T) {
	t.Parallel()
	node := sampleLayoutNode()
	width := 60
	height := 16
	got := renderInteractiveViewWith(t, node, width, height, configureHelpScenario)
	goldenPath := filepath.Join("testdata", "layout", "panel-help-60x16.txt")
	assertGolden(t, goldenPath, got)
}

func renderInteractiveView(t *testing.T, node interface{}, width, height int) string {
	return renderInteractiveViewWith(t, node, width, height, nil)
}

func renderInteractiveViewWith(t *testing.T, node interface{}, width, height int, configure func(*Model)) string {
	t.Helper()
	prevTheme := CurrentTheme()
	prevSort := navigator.SetSortOrder(navigator.SortAscending)
	defer func() {
		SetTheme(prevTheme)
		navigator.SetSortOrder(prevSort)
	}()
	SetTheme(DefaultTheme())

	m := InitialModel(node)
	m.Root = node
	m.Node = node
	m.NoColor = true
	m.WinWidth = width
	m.WinHeight = height
	m.AppName = "kvx"
	m.HelpTitle = "Help"
	m.HelpText = "Help content"
	if configure != nil {
		configure(&m)
	}
	m.ApplyColorScheme()
	m.applyLayout(true)
	m.syncAllComponents()

	viewObj := m.View()
	view := fmt.Sprint(viewObj.Content)
	return strings.TrimRight(view, "\n")
}

func sampleLayoutNode() map[string]interface{} {
	return map[string]interface{}{
		"active": true,
		"count":  12,
		"name":   "alice",
		"nested": map[string]interface{}{
			"value": "hello world",
		},
	}
}

func configureSearchScenario(m *Model) {
	m.AdvancedSearchActive = true
	m.AdvancedSearchCommitted = true // Commit to deep search for testing
	m.AdvancedSearchQuery = "hello"
	m.applyAdvancedSearch()
}

func configureFilterScenario(m *Model) {
	m.FilterBuffer = "na"
	m.applyTypeAheadFilter()
}

func configureInputScenario(m *Model) {
	m.InputFocused = true
	m.PathInput.SetValue("_.nested")
	m.PathInput.SetCursor(len(m.PathInput.Value()))
	m.PathInput.Focus()
	m.ExprDisplay = m.PathInput.Value()
}

func configureHelpScenario(m *Model) {
	m.HelpVisible = true
	m.HelpPopupText = "Tip: use F10 to quit."
}

func assertGolden(t *testing.T, path string, got string) {
	t.Helper()
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("failed to create snapshot dir: %v", err)
		}
		if err := os.WriteFile(path, []byte(got+"\n"), 0o644); err != nil {
			t.Fatalf("failed to write snapshot: %v", err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read snapshot: %v", err)
	}
	if got != strings.TrimRight(string(want), "\n") {
		t.Fatalf("snapshot mismatch:\n%s", got)
	}
}
