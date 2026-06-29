package ui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractNestedTable_HomogeneousArray(t *testing.T) {
	tests := []struct {
		name        string
		input       interface{}
		columnOrder []string
		wantRows    int
		wantCols    []string
	}{
		{
			name: "basic array of maps",
			input: []interface{}{
				map[string]interface{}{"name": "a", "status": "ok"},
				map[string]interface{}{"name": "b", "status": "fail"},
			},
			wantRows: 2,
			wantCols: []string{"name", "status"},
		},
		{
			name: "column order respected",
			input: []interface{}{
				map[string]interface{}{"name": "a", "status": "ok", "flow": "x"},
			},
			columnOrder: []string{"status", "name"},
			wantRows:    1,
			wantCols:    []string{"status", "name", "flow"},
		},
		{
			name: "column order with missing keys ignored",
			input: []interface{}{
				map[string]interface{}{"name": "a", "status": "ok"},
			},
			columnOrder: []string{"missing", "status"},
			wantRows:    1,
			wantCols:    []string{"status", "name"},
		},
		{
			name:     "nil input",
			input:    nil,
			wantRows: 0,
			wantCols: nil,
		},
		{
			name:     "empty array",
			input:    []interface{}{},
			wantRows: 0,
			wantCols: nil,
		},
		{
			name:     "non-array input",
			input:    "just a string",
			wantRows: 0,
			wantCols: nil,
		},
		{
			name: "heterogeneous array (mixed types)",
			input: []interface{}{
				map[string]interface{}{"name": "a"},
				"not a map",
			},
			wantRows: 0,
			wantCols: nil,
		},
		{
			name: "superset keys across rows",
			input: []interface{}{
				map[string]interface{}{"a": 1},
				map[string]interface{}{"a": 2, "b": 3},
			},
			wantRows: 2,
			wantCols: []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, cols := extractNestedTable(tt.input, tt.columnOrder)
			if tt.wantRows == 0 {
				assert.Nil(t, rows)
				assert.Nil(t, cols)
			} else {
				require.Len(t, rows, tt.wantRows)
				assert.Equal(t, tt.wantCols, cols)
			}
		})
	}
}

func TestRenderInlineColumnarTable(t *testing.T) {
	th := CurrentTheme()

	t.Run("basic rendering", func(t *testing.T) {
		cols := []string{"name", "status"}
		rows := []map[string]interface{}{
			{"name": "work", "status": "authenticated"},
			{"name": "personal", "status": "not authenticated"},
		}
		lines := renderInlineColumnarTable(cols, rows, 80, &th)
		require.Len(t, lines, 3) // header + 2 rows
		assert.Contains(t, stripANSI(lines[0]), "name")
		assert.Contains(t, stripANSI(lines[0]), "status")
		assert.Contains(t, stripANSI(lines[1]), "work")
		assert.Contains(t, stripANSI(lines[2]), "personal")
	})

	t.Run("empty input", func(t *testing.T) {
		lines := renderInlineColumnarTable(nil, nil, 80, &th)
		assert.Nil(t, lines)
	})

	t.Run("empty cols", func(t *testing.T) {
		lines := renderInlineColumnarTable([]string{}, []map[string]interface{}{{"a": 1}}, 80, &th)
		assert.Nil(t, lines)
	})

	t.Run("narrow width triggers shrink", func(t *testing.T) {
		cols := []string{"name", "very_long_status_column"}
		rows := []map[string]interface{}{
			{"name": "x", "very_long_status_column": "active"},
		}
		lines := renderInlineColumnarTable(cols, rows, 20, &th)
		require.NotNil(t, lines)
		for _, line := range lines {
			// All lines should be rendered (may be truncated but still present)
			assert.NotEmpty(t, stripANSI(line))
		}
	})

	t.Run("nil values in rows", func(t *testing.T) {
		cols := []string{"name", "status"}
		rows := []map[string]interface{}{
			{"name": "test", "status": nil},
		}
		lines := renderInlineColumnarTable(cols, rows, 80, &th)
		require.Len(t, lines, 2) // header + 1 row
	})
}

func TestBuildDetailViewModel(t *testing.T) {
	schema := &DisplaySchema{
		Detail: &DetailDisplayConfig{
			TitleField: "displayName",
			Sections: []DetailSection{
				{Title: "Info", Fields: []string{"status"}, Layout: DisplayLayoutTable},
				{Title: "Tags", Fields: []string{"tags"}, Layout: DisplayLayoutTags},
			},
			HiddenFields: []string{"internal"},
		},
	}

	t.Run("basic object", func(t *testing.T) {
		obj := map[string]interface{}{
			"displayName": "Test Item",
			"status":      "active",
			"tags":        []interface{}{"a", "b"},
			"internal":    "should-be-hidden",
		}
		dv := buildDetailViewModel(obj, schema, 80, 40)
		require.NotNil(t, dv)
		assert.Equal(t, "Test Item", dv.TitleText)
		assert.NotEmpty(t, dv.Sections)
	})

	t.Run("nil object", func(t *testing.T) {
		dv := buildDetailViewModel(nil, schema, 80, 40)
		assert.Nil(t, dv)
	})

	t.Run("nil schema", func(t *testing.T) {
		obj := map[string]interface{}{"name": "test"}
		dv := buildDetailViewModel(obj, nil, 80, 40)
		assert.Nil(t, dv)
	})

	t.Run("nil detail config", func(t *testing.T) {
		obj := map[string]interface{}{"name": "test"}
		s := &DisplaySchema{Detail: nil}
		dv := buildDetailViewModel(obj, s, 80, 40)
		assert.Nil(t, dv)
	})

	t.Run("non-map input", func(t *testing.T) {
		dv := buildDetailViewModel("not a map", schema, 80, 40)
		assert.Nil(t, dv)
	})

	t.Run("with nested table field", func(t *testing.T) {
		nestedSchema := &DisplaySchema{
			Detail: &DetailDisplayConfig{
				TitleField: "name",
				Sections: []DetailSection{
					{
						Title:       "Profiles",
						Fields:      []string{"profiles"},
						Layout:      DisplayLayoutTable,
						ColumnOrder: []string{"name", "status"},
					},
				},
			},
		}
		obj := map[string]interface{}{
			"name": "GitHub",
			"profiles": []interface{}{
				map[string]interface{}{"name": "work", "status": "auth"},
				map[string]interface{}{"name": "personal", "status": "none"},
			},
		}
		dv := buildDetailViewModel(obj, nestedSchema, 80, 40)
		require.NotNil(t, dv)
		assert.Equal(t, "GitHub", dv.TitleText)
		// The profiles section should have lines containing column data
		require.NotEmpty(t, dv.Sections)
		profileSection := dv.Sections[0]
		assert.Equal(t, "Profiles", profileSection.Title)
		require.NotEmpty(t, profileSection.Lines)
		// Header + 2 rows
		assert.GreaterOrEqual(t, len(profileSection.Lines), 3)
	})

	t.Run("uncovered fields go to Other section", func(t *testing.T) {
		minSchema := &DisplaySchema{
			Detail: &DetailDisplayConfig{
				TitleField: "name",
				Sections:   []DetailSection{},
			},
		}
		obj := map[string]interface{}{
			"name":  "Test",
			"extra": "uncovered",
		}
		dv := buildDetailViewModel(obj, minSchema, 80, 40)
		require.NotNil(t, dv)
		// extra should appear in a generated section
		assert.NotEmpty(t, dv.Sections)
	})
}

func TestRenderDetailView_Output(t *testing.T) {
	schema := &DisplaySchema{
		Detail: &DetailDisplayConfig{
			TitleField: "name",
			Sections: []DetailSection{
				{Title: "", Fields: []string{"status", "version"}, Layout: DisplayLayoutTable},
				{Title: "Description", Fields: []string{"desc"}, Layout: DisplayLayoutParagraph},
			},
		},
	}

	obj := map[string]interface{}{
		"name":    "GitHub",
		"status":  "authenticated",
		"version": "1.0",
		"desc":    "This is a description paragraph.",
	}

	dv := buildDetailViewModel(obj, schema, 80, 100)
	require.NotNil(t, dv)

	output := dv.Render(80, 100, true)
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "status")
	assert.Contains(t, output, "authenticated")
	assert.Contains(t, output, "Description")
	assert.Contains(t, output, "This is a description paragraph.")
}

func TestRenderDetailView_NilModel(t *testing.T) {
	output := renderDetailView(nil, nil, false)
	assert.Equal(t, "  (no data)", output)
}

func TestRenderInlineSection(t *testing.T) {
	obj := map[string]interface{}{
		"a": "hello",
		"b": "world",
		"c": nil,
	}
	hidden := map[string]bool{}

	lines := renderInlineSection(obj, []string{"a", "b", "c"}, 80, hidden)
	require.Len(t, lines, 1)
	plain := stripANSI(lines[0])
	assert.Contains(t, plain, "hello")
	assert.Contains(t, plain, "world")
}

func TestRenderInlineSection_HiddenFields(t *testing.T) {
	obj := map[string]interface{}{
		"a": "visible",
		"b": "hidden",
	}
	hidden := map[string]bool{"b": true}

	lines := renderInlineSection(obj, []string{"a", "b"}, 80, hidden)
	require.Len(t, lines, 1)
	plain := stripANSI(lines[0])
	assert.Contains(t, plain, "visible")
	assert.NotContains(t, plain, "hidden")
}

func TestRenderInlineSection_AllNil(t *testing.T) {
	obj := map[string]interface{}{"a": nil}
	lines := renderInlineSection(obj, []string{"a"}, 80, map[string]bool{})
	assert.Nil(t, lines)
}

func TestRenderParagraphSection(t *testing.T) {
	obj := map[string]interface{}{
		"desc": "Short paragraph.",
	}
	lines := renderParagraphSection(obj, []string{"desc"}, 80, map[string]bool{})
	require.NotEmpty(t, lines)
	assert.Contains(t, lines[0], "Short paragraph.")
}

func TestRenderParagraphSection_MultiLineNewlines(t *testing.T) {
	obj := map[string]interface{}{
		"desc": "Line one\nLine two\nLine three",
	}
	lines := renderParagraphSection(obj, []string{"desc"}, 80, map[string]bool{})
	require.Len(t, lines, 3)
	assert.Equal(t, "Line one", lines[0])
	assert.Equal(t, "Line two", lines[1])
	assert.Equal(t, "Line three", lines[2])
}

func TestRenderParagraphSection_EscapedNewlines(t *testing.T) {
	obj := map[string]interface{}{
		"desc": "First line\\nSecond line",
	}
	lines := renderParagraphSection(obj, []string{"desc"}, 80, map[string]bool{})
	require.Len(t, lines, 2)
	assert.Equal(t, "First line", lines[0])
	assert.Equal(t, "Second line", lines[1])
}

func TestRenderTagsSection(t *testing.T) {
	obj := map[string]interface{}{
		"tags": []interface{}{"alpha", "beta", "gamma"},
	}
	lines := renderTagsSection(obj, []string{"tags"}, 200, map[string]bool{})
	require.NotEmpty(t, lines)
	plain := stripANSI(strings.Join(lines, " "))
	assert.Contains(t, plain, "alpha")
	assert.Contains(t, plain, "beta")
	assert.Contains(t, plain, "gamma")
}

func TestRenderTagsSection_StringValue(t *testing.T) {
	obj := map[string]interface{}{
		"label": "single",
	}
	lines := renderTagsSection(obj, []string{"label"}, 80, map[string]bool{})
	require.NotEmpty(t, lines)
	assert.Contains(t, stripANSI(strings.Join(lines, " ")), "single")
}

func TestRenderTagsSection_Hidden(t *testing.T) {
	obj := map[string]interface{}{
		"tags": []interface{}{"a"},
	}
	lines := renderTagsSection(obj, []string{"tags"}, 80, map[string]bool{"tags": true})
	assert.Nil(t, lines)
}

type testFlow string

func TestRenderTagsSection_TypedSlices(t *testing.T) {
	tests := []struct {
		name       string
		val        interface{}
		wantCount  int
		wantTexts  []string // each text must appear individually in the stripped output
		wantAbsent string   // must NOT appear (e.g. JSON-stringified array)
	}{
		{
			name:      "[]interface{} renders per-element badges",
			val:       []interface{}{"a", "b", "c"},
			wantCount: 3,
			wantTexts: []string{"a", "b", "c"},
		},
		{
			name:       "[]string renders per-element badges",
			val:        []string{"a", "b", "c"},
			wantCount:  3,
			wantTexts:  []string{"a", "b", "c"},
			wantAbsent: `["a","b","c"]`,
		},
		{
			name:       "named string-kind slice renders per-element badges",
			val:        []testFlow{"alpha", "beta"},
			wantCount:  2,
			wantTexts:  []string{"alpha", "beta"},
			wantAbsent: `["alpha","beta"]`,
		},
		{
			name:      "single string renders one badge",
			val:       "solo",
			wantCount: 1,
			wantTexts: []string{"solo"},
		},
		{
			name:      "scalar int renders one badge",
			val:       42,
			wantCount: 1,
			wantTexts: []string{"42"},
		},
		{
			name:      "scalar bool renders one badge",
			val:       true,
			wantCount: 1,
			wantTexts: []string{"true"},
		},
		{
			name:      "nil renders zero badges",
			val:       nil,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := map[string]interface{}{"field": tt.val}
			lines := renderTagsSection(obj, []string{"field"}, 200, map[string]bool{})

			if tt.wantCount == 0 {
				assert.Nil(t, lines)
				return
			}

			require.NotNil(t, lines)
			plain := stripANSI(strings.Join(lines, " "))

			gotCount := 0
			for _, text := range tt.wantTexts {
				c := strings.Count(plain, " "+text+" ")
				assert.Greater(t, c, 0, "expected badge %q in output", text)
				gotCount += c
			}
			assert.Equal(t, tt.wantCount, gotCount)

			if tt.wantAbsent != "" {
				assert.NotContains(t, plain, tt.wantAbsent)
			}
		})
	}
}

func TestRenderTableSection_WithNestedTable(t *testing.T) {
	obj := map[string]interface{}{
		"profiles": []interface{}{
			map[string]interface{}{"name": "work", "status": "active"},
			map[string]interface{}{"name": "home", "status": "inactive"},
		},
	}
	lines := renderTableSection(obj, []string{"profiles"}, 80, map[string]bool{}, []string{"name", "status"})
	require.NotEmpty(t, lines)
	// Should have header + 2 data rows
	assert.GreaterOrEqual(t, len(lines), 3)
	plain := stripANSI(strings.Join(lines, "\n"))
	assert.Contains(t, plain, "name")
	assert.Contains(t, plain, "status")
	assert.Contains(t, plain, "work")
}

func TestRenderTableSection_ScalarFields(t *testing.T) {
	obj := map[string]interface{}{
		"name":   "test",
		"status": "ok",
	}
	lines := renderTableSection(obj, []string{"name", "status"}, 80, map[string]bool{}, nil)
	require.Len(t, lines, 2)
	plain := stripANSI(strings.Join(lines, "\n"))
	assert.Contains(t, plain, "name")
	assert.Contains(t, plain, "test")
	assert.Contains(t, plain, "status")
	assert.Contains(t, plain, "ok")
}

func TestStringifyValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		maxWidth int
		want     string
	}{
		{"string", "hello", 80, "hello"},
		{"int", 42, 80, "42"},
		{"bool", true, 80, "true"},
		{"nil", nil, 80, ""},
		{"array", []interface{}{"a", "b"}, 80, "[a, b]"},
		{"map", map[string]interface{}{"a": 1, "b": 2}, 80, "{2 keys}"},
		{"truncated string", "very long string that should be truncated", 10, "very lo..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stringifyValue(tt.input, tt.maxWidth)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildDetailView(t *testing.T) {
	schema := &DisplaySchema{
		Detail: &DetailDisplayConfig{
			TitleField: "name",
			Sections: []DetailSection{
				{Fields: []string{"status"}, Layout: DisplayLayoutTable},
			},
		},
	}
	obj := map[string]interface{}{
		"name":   "Test",
		"status": "ok",
	}
	dv := BuildDetailView(obj, schema, 80, 40)
	require.NotNil(t, dv)
	assert.Equal(t, "Test", dv.TitleText)
}
