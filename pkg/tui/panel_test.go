package tui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestRenderBorderedTable(t *testing.T) {
	node := map[string]any{"name": "test", "value": 42}
	out := RenderBorderedTable(node, BorderedTableOptions{
		Width:   80,
		NoColor: true,
	})
	if out == "" {
		t.Fatal("expected non-empty output")
	}
	if !strings.Contains(out, "name") {
		t.Errorf("expected key 'name' in output, got:\n%s", out)
	}
}

func TestRenderList(t *testing.T) {
	node := map[string]any{"name": "test", "items": []any{1, 2, 3}}
	out := RenderList(node, true)
	if out == "" {
		t.Fatal("expected non-empty output")
	}
	if !strings.Contains(out, "name") {
		t.Errorf("expected 'name' in list output, got:\n%s", out)
	}
}

func TestRenderTree(t *testing.T) {
	node := map[string]any{"a": map[string]any{"b": 1}, "c": 2}
	out := RenderTree(node, TreeOptions{})
	if out == "" {
		t.Fatal("expected non-empty output")
	}
}

func TestRenderMermaid(t *testing.T) {
	node := map[string]any{"root": map[string]any{"child": "value"}}
	out := RenderMermaid(node, MermaidOptions{})
	if out == "" {
		t.Fatal("expected non-empty output")
	}
	if !strings.Contains(out, "graph") && !strings.Contains(out, "flowchart") {
		t.Errorf("expected mermaid diagram, got:\n%s", out)
	}
}

func TestRenderYAMLFormat(t *testing.T) {
	node := map[string]any{"key": "value"}
	out := Render(node, FormatYAML, TableOptions{})
	if !strings.Contains(out, "key") || !strings.Contains(out, "value") {
		t.Errorf("expected YAML output, got:\n%s", out)
	}
}

func TestRenderJSONFormat(t *testing.T) {
	node := map[string]any{"key": "value"}
	out := Render(node, FormatJSON, TableOptions{})
	if !strings.Contains(out, "\"key\"") || !strings.Contains(out, "\"value\"") {
		t.Errorf("expected JSON output, got:\n%s", out)
	}
}

func TestRenderTableFormat(t *testing.T) {
	node := map[string]any{"name": "test"}
	out := Render(node, FormatTable, TableOptions{NoColor: true, Width: 80})
	if !strings.Contains(out, "name") {
		t.Errorf("expected table output with 'name', got:\n%s", out)
	}
}

func TestRenderListFormat(t *testing.T) {
	node := []any{
		map[string]any{"id": 1, "name": "a"},
		map[string]any{"id": 2, "name": "b"},
	}
	out := Render(node, FormatList, TableOptions{NoColor: true})
	if !strings.Contains(out, "name") {
		t.Errorf("expected list output with 'name', got:\n%s", out)
	}
}

func TestRenderTreeFormat(t *testing.T) {
	node := map[string]any{"a": 1}
	out := Render(node, FormatTree, TableOptions{})
	if out == "" {
		t.Fatal("expected non-empty tree output")
	}
}

func TestRenderMermaidFormat(t *testing.T) {
	node := map[string]any{"a": 1}
	out := Render(node, FormatMermaid, TableOptions{})
	if out == "" {
		t.Fatal("expected non-empty mermaid output")
	}
}

func TestRenderDefaultFormat(t *testing.T) {
	node := map[string]any{"x": 1}
	out := Render(node, OutputFormat("unknown"), TableOptions{NoColor: true, Width: 80})
	if out == "" {
		t.Fatal("expected non-empty default output")
	}
}

func TestRenderTable_StandardTable(t *testing.T) {
	// Non-homogeneous data forces standard (non-columnar) rendering
	node := map[string]any{
		"name":   "test",
		"nested": map[string]any{"a": 1},
	}
	out := RenderTable(node, TableOptions{
		NoColor:      true,
		Width:        80,
		ColumnarMode: "never",
	})
	if !strings.Contains(out, "name") {
		t.Errorf("expected 'name' in output, got:\n%s", out)
	}
}

func TestRenderTable_Bordered(t *testing.T) {
	node := map[string]any{"name": "test"}
	out := RenderTable(node, TableOptions{
		Bordered: true,
		Width:    80,
		NoColor:  true,
	})
	if out == "" {
		t.Fatal("expected non-empty bordered output")
	}
}

func TestRenderTable_ScalarValue(t *testing.T) {
	out := RenderTable("just a string", TableOptions{
		NoColor: true,
		Width:   80,
	})
	if out == "" {
		t.Fatal("expected non-empty output for scalar")
	}
	if !strings.Contains(out, "just a string") {
		t.Errorf("expected scalar value in output, got:\n%s", out)
	}
}

func TestRenderTable_Array(t *testing.T) {
	node := []any{"alpha", "beta", "gamma"}
	out := RenderTable(node, TableOptions{
		NoColor: true,
		Width:   80,
	})
	if !strings.Contains(out, "alpha") {
		t.Errorf("expected 'alpha' in output, got:\n%s", out)
	}
}

func TestRenderTable_NestedMap(t *testing.T) {
	node := map[string]any{
		"user": map[string]any{
			"name": "test",
			"age":  42,
		},
	}
	out := RenderTable(node, TableOptions{
		NoColor:      true,
		Width:        80,
		ColumnarMode: "never",
	})
	if !strings.Contains(out, "user") {
		t.Errorf("expected 'user' in output, got:\n%s", out)
	}
}

func TestRenderTable_ArrayOfObjects_ColumnarAlways(t *testing.T) {
	node := []any{
		map[string]any{"id": 1, "name": "a"},
		map[string]any{"id": 2, "name": "b"},
	}
	out := RenderTable(node, TableOptions{
		NoColor:      true,
		Width:        80,
		ColumnarMode: "always",
	})
	if !strings.Contains(out, "a") || !strings.Contains(out, "b") {
		t.Errorf("expected data values in columnar output, got:\n%s", out)
	}
}

func TestRenderTable_WithColumnOrder(t *testing.T) {
	node := []any{
		map[string]any{"z": 1, "a": 2, "m": 3},
	}
	out := RenderTable(node, TableOptions{
		NoColor:      true,
		Width:        80,
		ColumnarMode: "always",
		ColumnOrder:  []string{"m", "a", "z"},
	})
	if out == "" {
		t.Fatal("expected non-empty output")
	}
}

func TestRenderTable_WithHiddenColumns(t *testing.T) {
	node := []any{
		map[string]any{"visible": "yes", "hidden": "no"},
	}
	out := RenderTable(node, TableOptions{
		NoColor:       true,
		Width:         80,
		ColumnarMode:  "always",
		HiddenColumns: []string{"hidden"},
	})
	if strings.Contains(out, "hidden") {
		t.Errorf("hidden column should not appear in output, got:\n%s", out)
	}
}

func TestRenderTable_EmptyMap(t *testing.T) {
	node := map[string]any{}
	out := RenderTable(node, TableOptions{
		NoColor: true,
		Width:   80,
	})
	assert.NotEmpty(t, out)
}

func TestRenderTable_NilNode(t *testing.T) {
	out := RenderTable(nil, TableOptions{
		NoColor: true,
		Width:   80,
	})
	assert.NotEmpty(t, out)
}

func TestRenderTable_IntegerValue(t *testing.T) {
	out := RenderTable(42, TableOptions{
		NoColor: true,
		Width:   80,
	})
	assert.Contains(t, out, "42")
}

func TestRenderTable_ArrayOfMixed(t *testing.T) {
	node := []any{
		map[string]any{"name": "a"},
		"scalar",
		42,
	}
	out := RenderTable(node, TableOptions{
		NoColor:      true,
		Width:        80,
		ColumnarMode: "never",
	})
	assert.NotEmpty(t, out)
}

func TestRenderBorderedTable_WithAppNameAndPath(t *testing.T) {
	node := map[string]any{"key": "value"}
	out := RenderBorderedTable(node, BorderedTableOptions{
		AppName: "myapp",
		Path:    "root.items",
		Width:   80,
		NoColor: true,
	})
	assert.Contains(t, out, "myapp")
}

func TestRenderTable_ColumnarBordered(t *testing.T) {
	node := []any{
		map[string]any{"id": 1, "name": "x"},
		map[string]any{"id": 2, "name": "y"},
	}
	out := RenderTable(node, TableOptions{
		Bordered:     true,
		NoColor:      true,
		Width:        80,
		ColumnarMode: "always",
		AppName:      "test",
		Path:         "items",
	})
	assert.Contains(t, out, "test")
}

func TestRenderTable_ArrayStyleVariants(t *testing.T) {
	node := []any{
		map[string]any{"id": 1, "name": "a"},
		map[string]any{"id": 2, "name": "b"},
	}

	for _, style := range []string{"index", "numbered", "bullet", "none"} {
		t.Run(style, func(t *testing.T) {
			out := RenderTable(node, TableOptions{
				NoColor:      true,
				Width:        80,
				ColumnarMode: "always",
				ArrayStyle:   style,
			})
			assert.NotEmpty(t, out)
		})
	}
}
func TestRenderTable_ColumnarAlwaysFallbackToStandard(t *testing.T) {
	// Non-homogeneous array with ColumnarMode "always" triggers
	// renderColumnarTable → ExtractColumnarData returns nil → renderStandardTable fallback
	node := []any{"scalar1", 42, true}
	out := RenderTable(node, TableOptions{
		NoColor:      true,
		Width:        80,
		ColumnarMode: "always",
	})
	assert.NotEmpty(t, out)
}

func TestRenderTable_ColumnarAlwaysFallbackBordered(t *testing.T) {
	// Same but with bordered mode to cover more of renderStandardTable
	node := []any{"a", "b", "c"}
	out := RenderTable(node, TableOptions{
		NoColor:      true,
		Width:        80,
		Bordered:     true,
		ColumnarMode: "always",
		AppName:      "test",
	})
	assert.NotEmpty(t, out)
}

func TestRenderTable_MapBordered(t *testing.T) {
	node := map[string]any{
		"name":    "test",
		"version": "1.0",
		"active":  true,
	}
	out := RenderTable(node, TableOptions{
		Bordered: true,
		NoColor:  true,
		Width:    80,
		AppName:  "app",
		Path:     "root",
	})
	assert.Contains(t, out, "app")
	assert.Contains(t, out, "name")
}

func TestRenderTable_MapBorderedFitContent(t *testing.T) {
	// Small map with wide terminal - should shrink to fit content
	node := map[string]any{"a": "b"}
	out := RenderTable(node, TableOptions{
		Bordered: true,
		NoColor:  true,
		Width:    200,
	})
	assert.NotEmpty(t, out)
	assert.Contains(t, out, "╭")
}

func TestRenderTable_EmptyArray(t *testing.T) {
	out := RenderTable([]any{}, TableOptions{
		NoColor: true,
		Width:   80,
	})
	assert.NotEmpty(t, out)
}

func TestRenderTable_LargeColumnarBordered(t *testing.T) {
	// Larger array to cover more columnar bordered rendering paths
	node := []any{
		map[string]any{"id": 1, "name": "Alice", "email": "alice@example.com", "active": true},
		map[string]any{"id": 2, "name": "Bob", "email": "bob@example.com", "active": false},
		map[string]any{"id": 3, "name": "Carol", "email": "carol@example.com", "active": true},
	}
	out := RenderTable(node, TableOptions{
		Bordered:     true,
		NoColor:      true,
		Width:        120,
		ColumnarMode: "always",
		AppName:      "myapp",
		Path:         "users",
	})
	assert.Contains(t, out, "myapp")
	assert.Contains(t, out, "Alice")
	assert.Contains(t, out, "users")
}

func TestRenderTable_ColumnarWithColor(t *testing.T) {
	node := []any{
		map[string]any{"name": "x", "val": 1},
		map[string]any{"name": "y", "val": 2},
	}
	out := RenderTable(node, TableOptions{
		NoColor:      false,
		Width:        80,
		ColumnarMode: "always",
	})
	assert.NotEmpty(t, out)
}

func TestRenderTable_StandardWithColor(t *testing.T) {
	node := map[string]any{"key": "value"}
	out := RenderTable(node, TableOptions{
		NoColor: false,
		Width:   80,
	})
	assert.NotEmpty(t, out)
}

func TestRenderTable_BorderedWithColor(t *testing.T) {
	node := map[string]any{"key": "value"}
	out := RenderTable(node, TableOptions{
		Bordered: true,
		NoColor:  false,
		Width:    80,
	})
	assert.Contains(t, out, "╭")
}
