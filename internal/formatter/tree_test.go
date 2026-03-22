package formatter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatAsTree_SimpleMap(t *testing.T) {
	data := map[string]interface{}{
		"name":   "alice",
		"active": true,
		"count":  float64(42),
	}

	result := FormatAsTree(data, TreeOptions{})

	// Check root marker
	if !strings.HasPrefix(result, ".") {
		t.Error("expected tree to start with root marker '.'")
	}

	// Check key-value pairs
	if !strings.Contains(result, "name: alice") {
		t.Errorf("expected 'name: alice' in output, got:\n%s", result)
	}
	if !strings.Contains(result, "active: true") {
		t.Errorf("expected 'active: true' in output, got:\n%s", result)
	}
	if !strings.Contains(result, "count: 42") {
		t.Errorf("expected 'count: 42' in output, got:\n%s", result)
	}
}

func TestFormatAsTree_NestedMap(t *testing.T) {
	data := map[string]interface{}{
		"server": map[string]interface{}{
			"host": "localhost",
			"port": float64(8080),
		},
	}

	result := FormatAsTree(data, TreeOptions{})

	// Check branch structure
	if !strings.Contains(result, "server") {
		t.Errorf("expected 'server' branch, got:\n%s", result)
	}
	if !strings.Contains(result, "host: localhost") {
		t.Errorf("expected 'host: localhost' in output, got:\n%s", result)
	}
	if !strings.Contains(result, "port: 8080") {
		t.Errorf("expected 'port: 8080' in output, got:\n%s", result)
	}
}

func TestFormatAsTree_ArrayOfObjects(t *testing.T) {
	data := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"name": "item1"},
			map[string]interface{}{"name": "item2"},
		},
	}

	result := FormatAsTree(data, TreeOptions{ArrayStyle: "index"})

	// Check indexed array entries
	if !strings.Contains(result, "[0]") {
		t.Errorf("expected '[0]' index, got:\n%s", result)
	}
	if !strings.Contains(result, "[1]") {
		t.Errorf("expected '[1]' index, got:\n%s", result)
	}
	if !strings.Contains(result, "name: item1") {
		t.Errorf("expected 'name: item1', got:\n%s", result)
	}
	if !strings.Contains(result, "name: item2") {
		t.Errorf("expected 'name: item2', got:\n%s", result)
	}
}

func TestFormatAsTree_ScalarArrayInline(t *testing.T) {
	data := map[string]interface{}{
		"tags": []interface{}{"a", "b", "c"},
	}

	result := FormatAsTree(data, TreeOptions{})

	// Short scalar arrays should be inline
	if !strings.Contains(result, "tags: [a, b, c]") {
		t.Errorf("expected inline array 'tags: [a, b, c]', got:\n%s", result)
	}
}

func TestFormatAsTree_ScalarArraySummary(t *testing.T) {
	data := map[string]interface{}{
		"numbers": []interface{}{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
	}

	result := FormatAsTree(data, TreeOptions{})

	// Long scalar arrays should be summarized
	if !strings.Contains(result, "numbers: [10 items]") {
		t.Errorf("expected summary 'numbers: [10 items]', got:\n%s", result)
	}
}

func TestFormatAsTree_ScalarArrayExpand(t *testing.T) {
	data := map[string]interface{}{
		"numbers": []interface{}{1, 2, 3, 4, 5},
	}

	result := FormatAsTree(data, TreeOptions{ExpandArrays: true, ArrayStyle: "index"})

	// With ExpandArrays, should show indexed entries
	if !strings.Contains(result, "[0]") {
		t.Errorf("expected '[0]' when expanding arrays, got:\n%s", result)
	}
	if !strings.Contains(result, "[4]") {
		t.Errorf("expected '[4]' when expanding arrays, got:\n%s", result)
	}
}

func TestFormatAsTree_NoValues(t *testing.T) {
	data := map[string]interface{}{
		"name": "alice",
		"age":  float64(30),
	}

	result := FormatAsTree(data, TreeOptions{NoValues: true})

	// Should show keys only, no values
	if strings.Contains(result, "alice") {
		t.Errorf("should not contain value 'alice' with NoValues, got:\n%s", result)
	}
	if strings.Contains(result, "30") {
		t.Errorf("should not contain value '30' with NoValues, got:\n%s", result)
	}
	// But should still have the key names
	if !strings.Contains(result, "name") {
		t.Errorf("expected key 'name', got:\n%s", result)
	}
	if !strings.Contains(result, "age") {
		t.Errorf("expected key 'age', got:\n%s", result)
	}
}

func TestFormatAsTree_MaxDepth(t *testing.T) {
	data := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": "deep",
			},
		},
	}

	result := FormatAsTree(data, TreeOptions{MaxDepth: 2})

	// At depth 2, should show level1 and level2, but truncate level3's value
	if !strings.Contains(result, "level1") {
		t.Errorf("expected 'level1', got:\n%s", result)
	}
	if !strings.Contains(result, "level2") {
		t.Errorf("expected 'level2', got:\n%s", result)
	}
	// Should truncate at level3 with "..."
	if !strings.Contains(result, "...") {
		t.Errorf("expected truncation '...' at max depth, got:\n%s", result)
	}
	// Should not fully show level3's value "deep"
	if strings.Contains(result, "deep") {
		t.Errorf("should not contain 'deep' at max depth 2, got:\n%s", result)
	}
}

func TestFormatAsTree_EmptyMap(t *testing.T) {
	data := map[string]interface{}{
		"empty": map[string]interface{}{},
	}

	result := FormatAsTree(data, TreeOptions{})

	if !strings.Contains(result, "empty: {}") {
		t.Errorf("expected 'empty: {}', got:\n%s", result)
	}
}

func TestFormatAsTree_EmptyArray(t *testing.T) {
	data := map[string]interface{}{
		"empty": []interface{}{},
	}

	result := FormatAsTree(data, TreeOptions{})

	if !strings.Contains(result, "empty: []") {
		t.Errorf("expected 'empty: []', got:\n%s", result)
	}
}

func TestFormatAsTree_NullValue(t *testing.T) {
	data := map[string]interface{}{
		"value": nil,
	}

	result := FormatAsTree(data, TreeOptions{})

	if !strings.Contains(result, "value: null") {
		t.Errorf("expected 'value: null', got:\n%s", result)
	}
}

func TestFormatAsTree_LongStringTruncation(t *testing.T) {
	data := map[string]interface{}{
		"description": "This is a very long description that should be truncated for display",
	}

	result := FormatAsTree(data, TreeOptions{MaxStringLen: 30})

	// Should be truncated with ellipsis
	if !strings.Contains(result, "...") {
		t.Errorf("expected truncated string with '...', got:\n%s", result)
	}
	// Should not contain the full string
	if strings.Contains(result, "truncated for display") {
		t.Errorf("should have truncated the string, got:\n%s", result)
	}
}

func TestFormatAsTree_BoxDrawingChars(t *testing.T) {
	data := map[string]interface{}{
		"a": "1",
		"b": "2",
	}

	result := FormatAsTree(data, TreeOptions{})

	// Check for tree drawing characters
	if !strings.Contains(result, "├") && !strings.Contains(result, "└") {
		t.Errorf("expected tree box-drawing characters, got:\n%s", result)
	}
	if !strings.Contains(result, "──") {
		t.Errorf("expected horizontal line characters, got:\n%s", result)
	}
}

func TestFormatAsTree_ArrayStyleNone(t *testing.T) {
	// When array-style is "none", empty keys should show value-only or "(item)" labels
	t.Run("scalar array", func(t *testing.T) {
		data := map[string]interface{}{
			"tags": []interface{}{"a", "b", "c"},
		}
		result := FormatAsTree(data, TreeOptions{ExpandArrays: true, ArrayStyle: "none"})

		// Should show just the values, not ": a" or empty nodes
		if !strings.Contains(result, "a") {
			t.Errorf("expected value 'a' in output, got:\n%s", result)
		}
		if !strings.Contains(result, "b") {
			t.Errorf("expected value 'b' in output, got:\n%s", result)
		}
		if !strings.Contains(result, "c") {
			t.Errorf("expected value 'c' in output, got:\n%s", result)
		}
		if strings.Contains(result, ": a") {
			t.Errorf("expected no colon before value, got:\n%s", result)
		}
	})

	t.Run("object array", func(t *testing.T) {
		data := map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"name": "foo"},
			},
		}
		result := FormatAsTree(data, TreeOptions{ExpandArrays: true, ArrayStyle: "none"})

		// Object array elements should show "(item)" placeholder
		if !strings.Contains(result, "(item)") {
			t.Errorf("expected '(item)' placeholder in output, got:\n%s", result)
		}
		if !strings.Contains(result, "name: foo") {
			t.Errorf("expected 'name: foo' in output, got:\n%s", result)
		}
	})
}

func TestValidateArrayStyle(t *testing.T) {
	// Valid styles
	for _, style := range []string{"index", "numbered", "bullet", "none", ""} {
		if err := ValidateArrayStyle(style); err != nil {
			t.Errorf("ValidateArrayStyle(%q) should be valid, got %v", style, err)
		}
	}
	// Invalid styles
	for _, style := range []string{"invalid", "fancy", "roman"} {
		if err := ValidateArrayStyle(style); err == nil {
			t.Errorf("ValidateArrayStyle(%q) should be invalid", style)
		}
	}
}

func TestFormatScalarValue_NoTruncation(t *testing.T) {
	opts := TreeOptions{MaxStringLen: 0}
	result := formatScalarValue("hello world", opts)
	if result != "hello world" {
		t.Fatalf("expected 'hello world', got %q", result)
	}
}

func TestFormatScalarValue_Truncation(t *testing.T) {
	opts := TreeOptions{MaxStringLen: 10}
	result := formatScalarValue("a very long string", opts)
	if len(result) > 10 {
		t.Fatalf("expected max length 10, got %d: %q", len(result), result)
	}
	if !strings.HasSuffix(result, "...") {
		t.Fatalf("expected trailing '...', got %q", result)
	}
}

func TestFormatScalarValue_ShortMaxLen(t *testing.T) {
	opts := TreeOptions{MaxStringLen: 3}
	result := formatScalarValue("hello", opts)
	if result != "..." {
		t.Fatalf("expected '...', got %q", result)
	}
}

func TestFormatScalarValue_NilValue(t *testing.T) {
	opts := TreeOptions{}
	result := formatScalarValue(nil, opts)
	if result != "null" {
		t.Fatalf("expected 'null', got %q", result)
	}
}

func TestFormatScalarValue_Boolean(t *testing.T) {
	opts := TreeOptions{}
	if formatScalarValue(true, opts) != "true" {
		t.Fatal("expected 'true'")
	}
	if formatScalarValue(false, opts) != "false" {
		t.Fatal("expected 'false'")
	}
}

func TestFormatScalarValue_Numbers(t *testing.T) {
	opts := TreeOptions{}
	if formatScalarValue(float64(42), opts) != "42" {
		t.Fatalf("expected '42', got %q", formatScalarValue(float64(42), opts))
	}
	if formatScalarValue(3.14, opts) != "3.14" {
		t.Fatalf("expected '3.14', got %q", formatScalarValue(3.14, opts))
	}
}

func TestFormatScalarValueWithKey_FieldHint(t *testing.T) {
	opts := TreeOptions{
		MaxStringLen: 20,
		FieldHints:   map[string]int{"name": 5},
	}
	result := formatScalarValueWithKey("name", "Alice Johnson", opts)
	if len(result) > 5 {
		t.Fatalf("expected max length 5, got %d: %q", len(result), result)
	}
}

func TestFormatAsTree_ScalarRoot(t *testing.T) {
	out := FormatAsTree("hello", TreeOptions{})
	assert.Contains(t, out, "hello")
}

func TestFormatAsTree_MaxDepthLimit(t *testing.T) {
	node := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": "deep",
			},
		},
	}
	out := FormatAsTree(node, TreeOptions{MaxDepth: 1})
	assert.Contains(t, out, "...")
}

func TestFormatAsTree_ExpandArrays(t *testing.T) {
	node := map[string]interface{}{
		"tags": []interface{}{"go", "rust"},
	}
	out := FormatAsTree(node, TreeOptions{ExpandArrays: true})
	assert.Contains(t, out, "go")
	assert.Contains(t, out, "rust")
}

func TestFormatArrayIndex_AllStyles(t *testing.T) {
	assert.Equal(t, "[0]", FormatArrayIndex(0, "index"))
	assert.Equal(t, "1", FormatArrayIndex(0, "numbered"))
	assert.Equal(t, "•", FormatArrayIndex(0, "bullet"))
	assert.Equal(t, "", FormatArrayIndex(0, "none"))
	assert.Equal(t, "[0]", FormatArrayIndex(0, ""))
}
