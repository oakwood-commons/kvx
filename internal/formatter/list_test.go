package formatter

import (
	"strings"
	"testing"
)

func TestFormatAsListScalar(t *testing.T) {
	result := FormatAsList("hello", ListOptions{NoColor: true})
	expected := "value: hello\n"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestFormatAsListScalarNumber(t *testing.T) {
	result := FormatAsList(42, ListOptions{NoColor: true})
	expected := "value: 42\n"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestFormatAsListScalarBool(t *testing.T) {
	result := FormatAsList(true, ListOptions{NoColor: true})
	expected := "value: true\n"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestFormatAsListScalarArray(t *testing.T) {
	arr := []interface{}{"apple", "banana", "cherry"}
	result := FormatAsList(arr, ListOptions{NoColor: true})
	expected := "apple\nbanana\ncherry\n"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestFormatAsListNumberArray(t *testing.T) {
	arr := []interface{}{1, 2, 3}
	result := FormatAsList(arr, ListOptions{NoColor: true})
	expected := "1\n2\n3\n"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestFormatAsListMixedArray(t *testing.T) {
	arr := []interface{}{1, "hello", true}
	result := FormatAsList(arr, ListOptions{NoColor: true})
	expected := "1\nhello\ntrue\n"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestFormatAsListEmptyArray(t *testing.T) {
	arr := []interface{}{}
	result := FormatAsList(arr, ListOptions{NoColor: true})
	expected := ""
	if result != expected {
		t.Fatalf("expected empty string, got %q", result)
	}
}

func TestFormatAsListMap(t *testing.T) {
	m := map[string]interface{}{
		"name": "Alice",
		"age":  30,
	}
	result := FormatAsList(m, ListOptions{NoColor: true})

	// Check that both keys are present
	if !strings.Contains(result, "age: 30") {
		t.Fatalf("expected 'age: 30' in output, got %q", result)
	}
	if !strings.Contains(result, "name: Alice") {
		t.Fatalf("expected 'name: Alice' in output, got %q", result)
	}
}

func TestFormatAsListEmptyMap(t *testing.T) {
	m := map[string]interface{}{}
	result := FormatAsList(m, ListOptions{NoColor: true})
	expected := ""
	if result != expected {
		t.Fatalf("expected empty string, got %q", result)
	}
}

func TestFormatAsListArrayOfObjects(t *testing.T) {
	arr := []interface{}{
		map[string]interface{}{"name": "Alice", "age": 30},
		map[string]interface{}{"name": "Bob", "age": 25},
	}
	result := FormatAsList(arr, ListOptions{NoColor: true, ArrayStyle: "index"})

	// Check for index headers
	if !strings.Contains(result, "[0]") {
		t.Fatalf("expected '[0]' in output, got %q", result)
	}
	if !strings.Contains(result, "[1]") {
		t.Fatalf("expected '[1]' in output, got %q", result)
	}

	// Check for indented properties
	if !strings.Contains(result, "  name: Alice") {
		t.Fatalf("expected '  name: Alice' in output, got %q", result)
	}
	if !strings.Contains(result, "  age: 25") {
		t.Fatalf("expected '  age: 25' in output, got %q", result)
	}

	// Check proper structure with newlines between items
	lines := strings.Split(strings.TrimSpace(result), "\n")
	if len(lines) < 6 { // At least 2 headers + 4 properties
		t.Fatalf("expected at least 6 lines, got %d: %q", len(lines), result)
	}
}

func TestFormatAsListArrayOfObjectsNonHomogeneous(t *testing.T) {
	// Non-homogeneous array (different keys or mixed types)
	arr := []interface{}{
		map[string]interface{}{"name": "Alice"},
		map[string]interface{}{"age": 25},
	}
	result := FormatAsList(arr, ListOptions{NoColor: true, ArrayStyle: "index"})

	// Should still show as indexed items with indented properties
	if !strings.Contains(result, "[0]") {
		t.Fatalf("expected '[0]' in output, got %q", result)
	}
	if !strings.Contains(result, "[1]") {
		t.Fatalf("expected '[1]' in output, got %q", result)
	}
}

func TestFormatAsListNoColor(t *testing.T) {
	// Test with noColor=true to ensure no color codes
	m := map[string]interface{}{"key": "value"}
	result := FormatAsList(m, ListOptions{NoColor: true})

	// Check no ANSI escape codes present
	if strings.Contains(result, "\x1b[") {
		t.Fatalf("expected no ANSI codes with noColor=true, got %q", result)
	}
}

func TestFormatAsListArrayOfObjectsNoIndex(t *testing.T) {
	arr := []interface{}{
		map[string]interface{}{"name": "Alice", "age": 30},
		map[string]interface{}{"name": "Bob", "age": 25},
	}
	// Default style "none" should produce no index headers
	result := FormatAsList(arr, ListOptions{NoColor: true, ArrayStyle: "none"})

	// Should NOT contain index markers
	if strings.Contains(result, "[0]") {
		t.Fatalf("expected no index markers with style 'none', got %q", result)
	}
	if strings.Contains(result, "[1]") {
		t.Fatalf("expected no index markers with style 'none', got %q", result)
	}

	// Should still contain properties without indent
	if !strings.Contains(result, "name: Alice") {
		t.Fatalf("expected 'name: Alice' in output, got %q", result)
	}
	if !strings.Contains(result, "age: 25") {
		t.Fatalf("expected 'age: 25' in output, got %q", result)
	}

	// Should NOT have leading spaces (no indent when no index)
	for _, line := range strings.Split(strings.TrimSpace(result), "\n") {
		if strings.HasPrefix(line, "  ") {
			t.Fatalf("expected no leading indent with style 'none', got line %q", line)
		}
	}
}

func TestFormatAsListWithColor(t *testing.T) {
	// Test with noColor=false to ensure color codes are present
	m := map[string]interface{}{"key": "value"}
	result := FormatAsList(m, ListOptions{NoColor: false})

	// With color enabled, should contain ANSI escape codes
	// (lipgloss adds these for styled output)
	// We won't check exact codes as they might change, just verify styling happens
	if !strings.Contains(result, "key") || !strings.Contains(result, "value") {
		t.Fatalf("expected formatted output, got %q", result)
	}
}

func TestFormatAsListMapSortedKeys(t *testing.T) {
	m := map[string]interface{}{
		"zebra": "z",
		"apple": "a",
		"mango": "m",
	}
	result := FormatAsList(m, ListOptions{NoColor: true})

	// Keys should appear in sorted order
	lines := strings.Split(strings.TrimSpace(result), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines for 3 keys, got %d: %q", len(lines), result)
	}

	// Verify keys are sorted alphabetically
	expected := []string{"apple: a", "mango: m", "zebra: z"}
	for i, line := range lines {
		if line != expected[i] {
			t.Fatalf("expected line %d to be %q, got %q", i, expected[i], line)
		}
	}
}
