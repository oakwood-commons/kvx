package tui

import (
	"strings"
	"testing"
)

func TestRenderDataPanel(t *testing.T) {
	node := map[string]any{
		"name": "kvx",
	}

	out := RenderDataPanel(node, PanelOptions{
		NoColor:       true,
		KeyColWidth:   10,
		ValueColWidth: 10,
	})

	if !strings.Contains(out, "KEY") {
		t.Fatalf("expected header KEY, got output: %q", out)
	}
	if !strings.Contains(out, "VALUE") {
		t.Fatalf("expected header VALUE, got output: %q", out)
	}
	if !strings.Contains(out, "name") {
		t.Fatalf("expected key name, got output: %q", out)
	}
	if !strings.Contains(out, "kvx") {
		t.Fatalf("expected value kvx, got output: %q", out)
	}
}

func TestRenderTable_ColumnarWithDisplayHints(t *testing.T) {
	// Array of homogeneous objects — triggers columnar rendering
	data := []any{
		map[string]any{"user_id": "u001", "full_name": "Alice", "internal_code": "X123", "score": 95},
		map[string]any{"user_id": "u002", "full_name": "Bob", "internal_code": "Y456", "score": 87},
	}

	hints := map[string]ColumnHint{
		"user_id":       {DisplayName: "ID"},
		"full_name":     {DisplayName: "Name"},
		"internal_code": {Hidden: true},
		"score":         {Align: "right"},
	}

	out := RenderTable(data, TableOptions{
		NoColor:      true,
		Width:        100,
		ColumnarMode: "always",
		ColumnHints:  hints,
	})

	t.Run("display names in header", func(t *testing.T) {
		if !strings.Contains(out, "ID") {
			t.Errorf("expected renamed header 'ID', got:\n%s", out)
		}
		if !strings.Contains(out, "Name") {
			t.Errorf("expected renamed header 'Name', got:\n%s", out)
		}
	})

	t.Run("hidden column excluded", func(t *testing.T) {
		if strings.Contains(out, "internal_code") || strings.Contains(out, "X123") || strings.Contains(out, "Y456") {
			t.Errorf("expected hidden column internal_code to be excluded, got:\n%s", out)
		}
	})

	t.Run("data values present", func(t *testing.T) {
		if !strings.Contains(out, "Alice") || !strings.Contains(out, "Bob") {
			t.Errorf("expected data values, got:\n%s", out)
		}
	})

	t.Run("right-aligned score", func(t *testing.T) {
		lines := strings.Split(out, "\n")
		found := false
		for _, line := range lines {
			if strings.Contains(line, "Bob") && strings.Contains(line, "87") {
				idx := strings.Index(line, "87")
				if idx > 0 && line[idx-1] == ' ' {
					found = true
				}
			}
		}
		if !found {
			t.Errorf("expected right-aligned score column, got:\n%s", out)
		}
	})
}

func TestRenderTable_ColumnarMaxWidthWithDisplayName(t *testing.T) {
	data := []any{
		map[string]any{"name": "Very Long Name That Should Be Capped", "val": "ok"},
		map[string]any{"name": "Another Long Name Here Too", "val": "ok"},
	}

	hints := map[string]ColumnHint{
		"name": {DisplayName: "Full Name", MaxWidth: 10},
	}

	out := RenderTable(data, TableOptions{
		NoColor:      true,
		Width:        100,
		ColumnarMode: "always",
		ColumnHints:  hints,
	})

	// Header should use display name
	if !strings.Contains(out, "Full Name") {
		t.Errorf("expected 'Full Name' header, got:\n%s", out)
	}
	// Long values should be truncated by MaxWidth
	if strings.Contains(out, "Very Long Name That Should Be Capped") {
		t.Errorf("expected name to be capped by MaxWidth, got:\n%s", out)
	}
}
