package formatter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatAsMermaid_SimpleMap(t *testing.T) {
	data := map[string]interface{}{
		"name": "alice",
		"age":  30,
	}

	result := FormatAsMermaid(data, MermaidOptions{})

	assert.Contains(t, result, "graph TD")
	assert.Contains(t, result, `"root"`)
	assert.Contains(t, result, `"name: alice"`)
	assert.Contains(t, result, `"age: 30"`)
	assert.Contains(t, result, "-->")
}

func TestFormatAsMermaid_NestedMap(t *testing.T) {
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"name": "bob",
		},
	}

	result := FormatAsMermaid(data, MermaidOptions{})

	assert.Contains(t, result, "graph TD")
	assert.Contains(t, result, `"user"`)
	assert.Contains(t, result, `"name: bob"`)
	// Should have edges from root to user, and user to name
	lines := strings.Split(result, "\n")
	edgeCount := 0
	for _, line := range lines {
		if strings.Contains(line, "-->") {
			edgeCount++
		}
	}
	assert.Equal(t, 2, edgeCount)
}

func TestFormatAsMermaid_ArrayOfObjects(t *testing.T) {
	data := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"id": 1},
			map[string]interface{}{"id": 2},
		},
	}

	result := FormatAsMermaid(data, MermaidOptions{ArrayStyle: "index"})

	assert.Contains(t, result, `"items"`)
	assert.Contains(t, result, `"[0]"`)
	assert.Contains(t, result, `"[1]"`)
	assert.Contains(t, result, `"id: 1"`)
	assert.Contains(t, result, `"id: 2"`)
}

func TestFormatAsMermaid_ScalarArrayInline(t *testing.T) {
	data := map[string]interface{}{
		"tags": []interface{}{"a", "b", "c"},
	}

	result := FormatAsMermaid(data, MermaidOptions{MaxArrayInline: 3})

	assert.Contains(t, result, `"tags"`)
	assert.Contains(t, result, `"[a, b, c]"`)
}

func TestFormatAsMermaid_ScalarArraySummary(t *testing.T) {
	data := map[string]interface{}{
		"tags": []interface{}{"a", "b", "c", "d", "e"},
	}

	result := FormatAsMermaid(data, MermaidOptions{MaxArrayInline: 3})

	assert.Contains(t, result, `"tags"`)
	assert.Contains(t, result, `"[5 items]"`)
}

func TestFormatAsMermaid_NoValues(t *testing.T) {
	data := map[string]interface{}{
		"name": "alice",
		"age":  30,
	}

	result := FormatAsMermaid(data, MermaidOptions{NoValues: true})

	assert.Contains(t, result, `"name"`)
	assert.Contains(t, result, `"age"`)
	assert.NotContains(t, result, "alice")
	assert.NotContains(t, result, "30")
}

func TestFormatAsMermaid_MaxDepth(t *testing.T) {
	data := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": "deep",
			},
		},
	}

	result := FormatAsMermaid(data, MermaidOptions{MaxDepth: 2})

	assert.Contains(t, result, `"level1"`)
	assert.Contains(t, result, `"level2"`)
	assert.Contains(t, result, `"..."`)
	assert.NotContains(t, result, "level3")
	assert.NotContains(t, result, "deep")
}

func TestFormatAsMermaid_DirectionLR(t *testing.T) {
	data := map[string]interface{}{"key": "value"}

	result := FormatAsMermaid(data, MermaidOptions{Direction: "LR"})

	assert.Contains(t, result, "graph LR")
}

func TestFormatAsMermaid_DirectionBT(t *testing.T) {
	data := map[string]interface{}{"key": "value"}

	result := FormatAsMermaid(data, MermaidOptions{Direction: "BT"})

	assert.Contains(t, result, "graph BT")
}

func TestFormatAsMermaid_SpecialCharacterEscaping(t *testing.T) {
	data := map[string]interface{}{
		"message": `He said "hello"`,
	}

	result := FormatAsMermaid(data, MermaidOptions{})

	// Double quotes should be replaced with single quotes
	assert.Contains(t, result, `"message: He said 'hello'"`)
	assert.NotContains(t, result, `""`)
}

func TestFormatAsMermaid_EmptyMap(t *testing.T) {
	data := map[string]interface{}{}

	result := FormatAsMermaid(data, MermaidOptions{})

	assert.Contains(t, result, "graph TD")
	assert.Contains(t, result, `"root"`)
}

func TestFormatAsMermaid_ScalarRoot(t *testing.T) {
	// Scalar root values should be displayed as child nodes
	tests := []struct {
		name     string
		data     interface{}
		expected string
	}{
		{"string", "hello", `"hello"`},
		{"number", 42.0, `"42"`},
		{"boolean", true, `"true"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatAsMermaid(tt.data, MermaidOptions{})

			assert.Contains(t, result, "graph TD")
			assert.Contains(t, result, `"root"`)
			assert.Contains(t, result, tt.expected)
			assert.Contains(t, result, "-->") // Edge from root to value
		})
	}
}

func TestFormatAsMermaid_LongStringTruncation(t *testing.T) {
	data := map[string]interface{}{
		"url": "https://example.com/very/long/path/that/should/be/truncated",
	}

	result := FormatAsMermaid(data, MermaidOptions{MaxStringLen: 30})

	assert.Contains(t, result, "...")
	// The full URL should not appear
	assert.NotContains(t, result, "truncated")
}

func TestFormatAsMermaid_ExpandArrays(t *testing.T) {
	data := map[string]interface{}{
		"tags": []interface{}{"a", "b", "c", "d", "e"},
	}

	result := FormatAsMermaid(data, MermaidOptions{ExpandArrays: true, ArrayStyle: "index"})

	// With ExpandArrays, scalar elements show index with value
	assert.Contains(t, result, `"[0]: a"`)
	assert.Contains(t, result, `"[1]: b"`)
	assert.Contains(t, result, `"[4]: e"`)
	assert.NotContains(t, result, "[5 items]")
}

func TestFormatAsMermaid_ArrayStyleNone(t *testing.T) {
	// When array-style is "none", empty keys should show value-only or "(item)" labels
	t.Run("scalar array", func(t *testing.T) {
		data := map[string]interface{}{
			"tags": []interface{}{"a", "b", "c"},
		}
		result := FormatAsMermaid(data, MermaidOptions{ExpandArrays: true, ArrayStyle: "none"})

		// Should show just the values, not ": a" or empty labels
		assert.Contains(t, result, `"a"`)
		assert.Contains(t, result, `"b"`)
		assert.Contains(t, result, `"c"`)
		assert.NotContains(t, result, `: a`) // No leading colon
	})

	t.Run("object array", func(t *testing.T) {
		data := map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"name": "foo"},
			},
		}
		result := FormatAsMermaid(data, MermaidOptions{ExpandArrays: true, ArrayStyle: "none"})

		// Object array elements should show "(item)" placeholder
		assert.Contains(t, result, `"(item)"`)
		assert.Contains(t, result, `"name: foo"`)
	})
}

func TestSanitizeMermaidID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with spaces", "with_spaces"},
		{"with-dashes", "with_dashes"},
		{"with.dots", "with_dots"},
		{"with[brackets]", "with_brackets_"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := SanitizeMermaidID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
