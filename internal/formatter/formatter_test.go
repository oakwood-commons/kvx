package formatter

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestStringifyString(t *testing.T) {
	result := Stringify("hello")
	if result != "hello" {
		t.Fatalf("expected 'hello', got %q", result)
	}
}

func TestStringifyStringEscapesNewlines(t *testing.T) {
	input := "line1\nline2"
	result := Stringify(input)
	if result != "line1\\nline2" {
		t.Fatalf("expected escaped newlines, got %q", result)
	}
}

func TestStringifyPreserveNewlines(t *testing.T) {
	input := "line1\nline2"
	result := StringifyPreserveNewlines(input)
	if strings.Contains(result, "\\n") {
		t.Fatalf("expected real newlines, got %q", result)
	}
	lines := strings.Split(result, "\n")
	if len(lines) != 2 || lines[0] != "line1" || lines[1] != "line2" {
		t.Fatalf("expected lines split, got %#v", lines)
	}
}

func TestStringifyPreserveNewlinesExpandsEscaped(t *testing.T) {
	input := "line1\\nline2"
	result := StringifyPreserveNewlines(input)
	lines := strings.Split(result, "\n")
	if len(lines) != 2 || lines[0] != "line1" || lines[1] != "line2" {
		t.Fatalf("expected escaped newlines to expand, got %#v", lines)
	}
}

func TestStringifyNil(t *testing.T) {
	result := Stringify(nil)
	if result != "" {
		t.Fatalf("expected empty string for nil, got %q", result)
	}
}

func TestStringifyBool(t *testing.T) {
	result := Stringify(true)
	if result != "true" {
		t.Fatalf("expected 'true', got %q", result)
	}
}

func TestStringifyInt(t *testing.T) {
	result := Stringify(42)
	if result != "42" {
		t.Fatalf("expected '42', got %q", result)
	}
}

func TestStringifyInt64(t *testing.T) {
	result := Stringify(int64(123))
	if result != "123" {
		t.Fatalf("expected '123', got %q", result)
	}
}

func TestStringifyFloat(t *testing.T) {
	result := Stringify(3.14)
	if !strings.Contains(result, "3.14") {
		t.Fatalf("expected '3.14', got %q", result)
	}
}

func TestStringifyMap(t *testing.T) {
	data := map[string]any{"key": "value", "num": 42}
	result := Stringify(data)
	if !strings.Contains(result, "key") || !strings.Contains(result, "value") {
		t.Fatalf("expected map JSON representation, got %q", result)
	}
}

func TestStringifyArray(t *testing.T) {
	data := []any{1, 2, 3}
	result := Stringify(data)
	if !strings.Contains(result, "[") || !strings.Contains(result, "]") {
		t.Fatalf("expected array JSON representation, got %q", result)
	}
}

func TestStringifyMapUnmarshalError(t *testing.T) {
	// This would require a type that cannot be marshaled
	// For now, we test the fallback with a struct
	type customType struct {
		Field string
	}
	result := Stringify(customType{"test"})
	if result == "" {
		t.Fatalf("expected non-empty fallback representation")
	}
}

func TestStringifyStruct(t *testing.T) {
	// Verify structs are marshaled to JSON, not Go's default fmt representation
	type authStatus struct {
		Authenticated bool   `json:"authenticated"`
		DisplayName   string `json:"displayName"`
	}
	data := authStatus{Authenticated: true, DisplayName: "Test User"}
	result := Stringify(data)
	// Should be JSON like {"authenticated":true,"displayName":"Test User"}
	// NOT Go's fmt like {true Test User}
	if !strings.Contains(result, `"authenticated":true`) {
		t.Fatalf("expected JSON with authenticated field, got %q", result)
	}
	if !strings.Contains(result, `"displayName":"Test User"`) {
		t.Fatalf("expected JSON with displayName field, got %q", result)
	}
}

func TestStringifySliceOfMaps(t *testing.T) {
	// Verify []map[string]interface{} (not []any) renders as JSON
	data := []map[string]interface{}{
		{"name": "Alice", "active": true},
		{"name": "Bob", "active": false},
	}
	result := Stringify(data)
	// Should be JSON array, not Go's fmt representation
	if !strings.HasPrefix(result, "[") || !strings.HasSuffix(result, "]") {
		t.Fatalf("expected JSON array, got %q", result)
	}
	if strings.Contains(result, "map[") {
		t.Fatalf("should not contain Go map representation, got %q", result)
	}
}

func TestStringifyPointerToStruct(t *testing.T) {
	type info struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	data := &info{ID: 1, Name: "test"}
	result := Stringify(data)
	if !strings.Contains(result, `"id":1`) || !strings.Contains(result, `"name":"test"`) {
		t.Fatalf("expected JSON representation of pointer to struct, got %q", result)
	}
}

func TestTruncateNoTruncation(t *testing.T) {
	result := truncate("hello", 10)
	if result != "hello" {
		t.Fatalf("expected 'hello', got %q", result)
	}
}

func TestTruncateWithTruncation(t *testing.T) {
	result := truncate("hello world this is long", 10)
	if !strings.HasSuffix(result, "...") {
		t.Fatalf("expected truncation with '...', got %q", result)
	}
	if len(result) > 10 {
		t.Fatalf("expected length <= 10, got %d", len(result))
	}
}

func TestTruncateSmallMaxLen(t *testing.T) {
	result := truncate("hello", 2)
	// When maxLen < 3, truncate returns the string as-is up to maxLen
	if len(result) > 2 {
		t.Fatalf("expected length <= 2, got %q (len=%d)", result, len(result))
	}
}

func TestGetTerminalWidthDefault(t *testing.T) {
	width := getTerminalWidth()
	if width <= 0 {
		t.Fatalf("expected positive width, got %d", width)
	}
	if width != 120 {
		// In test environment, may not be a TTY, so default is 120
		t.Logf("terminal width: %d (likely default due to non-TTY test env)", width)
	}
}

func TestRenderTableEmpty(t *testing.T) {
	node := map[string]any{}
	result := RenderTable(node, true, 0, 0)
	if !strings.Contains(result, "KEY") || !strings.Contains(result, "VALUE") {
		t.Fatalf("expected header, got %q", result)
	}
}

func TestRenderTableSimpleMap(t *testing.T) {
	node := map[string]any{
		"name": "alice",
		"age":  30,
	}
	result := RenderTable(node, true, 0, 0)
	if !strings.Contains(result, "name") || !strings.Contains(result, "age") {
		t.Fatalf("expected keys in output, got %q", result)
	}
	if !strings.Contains(result, "alice") {
		t.Fatalf("expected 'alice' in output, got %q", result)
	}
}

func TestRenderTableArray(t *testing.T) {
	node := []any{"item1", "item2"}
	result := RenderTable(node, true, 0, 0)
	if !strings.Contains(result, "[0]") || !strings.Contains(result, "[1]") {
		t.Fatalf("expected array indices, got %q", result)
	}
	if !strings.Contains(result, "item1") {
		t.Fatalf("expected 'item1' in output, got %q", result)
	}
}

func TestRenderTableScalar(t *testing.T) {
	node := "simple value"
	result := RenderTable(node, true, 0, 0)
	// Check for "(value)" label - matches navigator.ScalarValueKey
	if !strings.Contains(result, "(value)") {
		t.Fatalf("expected '(value)' label, got %q", result)
	}
	if !strings.Contains(result, "simple value") {
		t.Fatalf("expected value in output, got %q", result)
	}
}

func TestRenderTableWithColor(t *testing.T) {
	node := map[string]any{"key": "value"}
	result := RenderTable(node, false, 0, 0)
	// With color enabled, lipgloss adds ANSI codes
	if !strings.Contains(result, "key") {
		t.Fatalf("expected 'key' in colored output, got %q", result)
	}
}

func TestRenderTableNoColor(t *testing.T) {
	node := map[string]any{"key": "value"}
	result := RenderTable(node, true, 0, 0)
	// Without color, output should be plain text
	if !strings.Contains(result, "key") {
		t.Fatalf("expected 'key' in no-color output, got %q", result)
	}
	// Should not contain ANSI escape codes
	if strings.Contains(result, "\x1b") {
		t.Fatalf("expected no ANSI codes in no-color output")
	}
}

func TestRenderTableTruncation(t *testing.T) {
	node := map[string]any{
		"very_long_key_name_that_exceeds_normal_width": "value",
	}
	result := RenderTable(node, true, 0, 0)
	// Key should be truncated or wrapped
	lines := strings.Split(result, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(lines))
	}
}

func TestRenderTableMultipleRows(t *testing.T) {
	node := map[string]any{
		"a": "value_a",
		"b": "value_b",
		"c": "value_c",
	}
	result := RenderTable(node, true, 0, 0)
	lines := strings.Split(result, "\n")
	// Header + separator + 3 rows + empty line
	if len(lines) < 5 {
		t.Fatalf("expected at least 5 lines (header, sep, 3 rows), got %d", len(lines))
	}
}

func TestFormatYAMLLiteralBlock(t *testing.T) {
	obj := map[string]any{
		"name":  "demo",
		"notes": "line1\nline2",
	}

	out, err := FormatYAML(obj, YAMLFormatOptions{Indent: 2, LiteralBlockStrings: true})
	if err != nil {
		t.Fatalf("format yaml: %v", err)
	}

	if !strings.Contains(out, "notes: |\n  line1\n  line2") && !strings.Contains(out, "notes: |-\n  line1\n  line2") {
		t.Fatalf("expected literal block for multiline string, got:\n%s", out)
	}

	var decoded map[string]any
	if err := yaml.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("unmarshal formatted yaml: %v", err)
	}
	if decoded["notes"] != "line1\nline2" {
		t.Fatalf("expected decoded notes to match, got %#v", decoded["notes"])
	}
}

func TestFormatYAMLExpandEscapedNewlines(t *testing.T) {
	obj := map[string]any{
		"note": "line1\\nline2",
	}

	out, err := FormatYAML(obj, YAMLFormatOptions{Indent: 2, LiteralBlockStrings: true, ExpandEscapedNewlines: true})
	if err != nil {
		t.Fatalf("format yaml: %v", err)
	}

	if !strings.Contains(out, "note: |\n  line1\n  line2") && !strings.Contains(out, "note: |-\n  line1\n  line2") {
		t.Fatalf("expected literal block with expanded newlines, got:\n%s", out)
	}

	var decoded map[string]any
	if err := yaml.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("unmarshal formatted yaml: %v", err)
	}
	if decoded["note"] != "line1\nline2" {
		t.Fatalf("expected decoded note to have newline, got %#v", decoded["note"])
	}
}

func TestPadRight(t *testing.T) {
	result := padRight("test", 10)
	if len(result) != 10 {
		t.Fatalf("expected length 10, got %d", len(result))
	}
	if !strings.HasPrefix(result, "test") {
		t.Fatalf("expected 'test' prefix, got %q", result)
	}
}

func TestPadRightAlreadyLong(t *testing.T) {
	result := padRight("very long string", 5)
	// padRight truncates if string is longer than width
	if len(result) != 5 {
		t.Fatalf("expected length 5, got %d: %q", len(result), result)
	}
	if result != "very " {
		t.Fatalf("expected 'very ', got %q", result)
	}
}

func TestPadRightExactLength(t *testing.T) {
	result := padRight("test", 4)
	if result != "test" {
		t.Fatalf("expected 'test', got %q", result)
	}
}
