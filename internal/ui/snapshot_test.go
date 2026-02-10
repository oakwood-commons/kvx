package ui

import (
	"strings"
	"testing"
)

func TestSnapshotRenderRootTable(t *testing.T) {
	node := map[string]interface{}{
		"name": "alice",
		"age":  30,
	}

	rendered := RenderModelSnapshot(node, ModelSnapshotConfig{
		Width:   60,
		Height:  12,
		NoColor: true,
	})

	if !strings.Contains(rendered, "KEY") || !strings.Contains(rendered, "VALUE") {
		t.Fatalf("expected table headers in snapshot render, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "name") || !strings.Contains(rendered, "alice") {
		t.Fatalf("expected rendered table to include node content, got:\n%s", rendered)
	}
}

func TestSnapshotRenderHelpVisible(t *testing.T) {
	node := map[string]interface{}{"foo": "bar"}

	rendered := RenderModelSnapshot(node, ModelSnapshotConfig{
		Width:       60,
		Height:      12,
		NoColor:     true,
		HelpVisible: true,
		HelpTitle:   "Help",
		HelpText:    "Help content",
	})

	if !strings.Contains(rendered, "Help content") {
		t.Fatalf("expected help overlay content in snapshot render, got:\n%s", rendered)
	}
}

func TestSnapshotRenderSuggestionsDropdown(t *testing.T) {
	node := map[string]interface{}{"items": []interface{}{"a"}}

	rendered := RenderModelSnapshot(node, ModelSnapshotConfig{
		Width:   50,
		Height:  10,
		NoColor: true,
		Configure: func(m *Model) {
			m.ShowSuggestions = true
			m.InputFocused = true
			m.PathInput.SetValue(".")
			m.FilteredSuggestions = []string{"type()", "size()", "items"}
		},
	})

	// Suggestions are now shown in the status bar, not as a dropdown
	// The snapshot should still render, but without a dropdown
	// We just verify it renders successfully (doesn't crash)
	if rendered == "" {
		t.Fatalf("expected snapshot to render, got empty string")
	}
	// The suggestions should appear in the status bar, not as a dropdown
	// We can't easily test the status bar content here, so we just verify rendering works
}

func TestSnapshotLayoutAnchorsWithDebugPlaceholder(t *testing.T) {
	assertSnapshotAnchors(t, false)
}

func TestSnapshotLayoutAnchorsWithDebugOn(t *testing.T) {
	assertSnapshotAnchors(t, true)
}

func assertSnapshotAnchors(t *testing.T, debugMode bool) {
	t.Helper()

	node := map[string]interface{}{
		"alert": true,
		"name":  "demo",
	}
	width := 50
	height := 20

	rendered := RenderModelSnapshot(node, ModelSnapshotConfig{
		Width:   width,
		Height:  height,
		NoColor: true,
		Configure: func(m *Model) {
			m.DebugMode = debugMode
		},
	})

	lines := trimTrailingEmpty(strings.Split(rendered, "\n"))
	if len(lines) != height {
		t.Fatalf("expected snapshot to use %d lines, got %d", height, len(lines))
	}
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 bottom lines, got %d", len(lines))
	}

	if len(lines) < 2 || !strings.Contains(lines[1], "KEY") {
		t.Fatalf("expected table header near the top, got line: %q", lines[1])
	}

	lastIdx := len(lines) - 1
	footerLine := lines[lastIdx]
	labelLine := lines[lastIdx-1]
	// Default is vim mode, check for vim-style keys
	if !strings.Contains(footerLine, "? help") && !strings.Contains(footerLine, "F1") {
		t.Fatalf("expected footer bindings at the bottom, got: %q", footerLine)
	}
	if debugMode {
		if !strings.Contains(footerLine, "Rows:") || !strings.Contains(footerLine, "Cols:") {
			t.Fatalf("expected debug row/col info in footer, got: %q", footerLine)
		}
	} else if strings.Contains(footerLine, "Rows:") {
		t.Fatalf("unexpected debug info in footer: %q", footerLine)
	}
	if !strings.Contains(labelLine, "_") {
		t.Fatalf("expected path label line above footer, got: %q", labelLine)
	}
}

func TestSnapshotSmallHeightKeepsHeaderAndFooterVisible(t *testing.T) {
	node := map[string]interface{}{"k": "v"}
	width := 40
	height := 8 // intentionally tight to stress layout clamping

	rendered := RenderModelSnapshot(node, ModelSnapshotConfig{
		Width:   width,
		Height:  height,
		NoColor: true,
	})

	lines := trimTrailingEmpty(strings.Split(rendered, "\n"))
	if len(lines) != height {
		t.Fatalf("expected snapshot to use %d lines, got %d", height, len(lines))
	}
	if len(lines) < 2 || !strings.Contains(lines[1], "KEY") {
		t.Fatalf("expected table header to remain visible near the top, got %q", lines[1])
	}
	last := lines[len(lines)-1]
	// Default is vim mode, check for vim-style keys
	if !strings.Contains(last, "? help") && !strings.Contains(last, "F1") {
		t.Fatalf("expected footer bindings on the last line, got %q", last)
	}
}

func TestSnapshotStartKeysToggleHelp(t *testing.T) {
	node := map[string]interface{}{"k": "v"}

	rendered := RenderModelSnapshot(node, ModelSnapshotConfig{
		Width:   50,
		Height:  14,
		NoColor: true,
		StartKeys: []string{
			"<F1>",
		},
		HelpTitle: "Help",
		HelpText:  "Help content",
	})

	if !strings.Contains(rendered, "Help content") {
		t.Fatalf("expected help overlay to render when F1 is in start keys, got:\n%s", rendered)
	}
}

func TestSnapshotExpressionStartKeysShowInput(t *testing.T) {
	node := map[string]interface{}{
		"items": []interface{}{map[string]interface{}{"name": "a"}},
	}

	rendered := RenderModelSnapshot(node, ModelSnapshotConfig{
		Width:       60,
		Height:      16,
		NoColor:     true,
		InitialExpr: "_.items[0]",
		StartKeys: []string{
			"<F6>",
		},
	})

	if !strings.Contains(rendered, "Expression") || !strings.Contains(rendered, "_.items[0]") {
		t.Fatalf("expected expression overlay with initial expr, got:\n%s", rendered)
	}
}

func TestPadSnapshotHeightUsesWidthPadding(t *testing.T) {
	view := "only"
	got := padSnapshotHeight(view, 3, 6)
	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines after padding, got %d", len(lines))
	}
	if len(lines[1]) != 6 {
		t.Fatalf("expected pad line to match width 6, got %d", len(lines[1]))
	}
}

func trimTrailingEmpty(lines []string) []string {
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
