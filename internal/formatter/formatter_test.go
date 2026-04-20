package formatter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestSetTableTheme(t *testing.T) {
	// Should not panic with zero-value colors
	SetTableTheme(TableColors{})
}

func TestCalculateNaturalTableWidth(t *testing.T) {
	rows := [][]string{
		{"name", "Alice"},
		{"age", "30"},
	}
	w := CalculateNaturalTableWidth(rows)
	// Should be at least header widths: "KEY"(3) + sep(2) + "VALUE"(5) = 10
	if w < 10 {
		t.Fatalf("expected width >= 10, got %d", w)
	}
	// "Alice" is wider than "VALUE", so width should reflect "Alice"
	if w < 11 {
		t.Fatalf("expected width >= 11 (for 'Alice'), got %d", w)
	}
}

func TestCalculateNaturalTableWidth_Empty(t *testing.T) {
	w := CalculateNaturalTableWidth(nil)
	// Minimum: "KEY"(3) + sep(2) + "VALUE"(5) = 10
	if w != 10 {
		t.Fatalf("expected 10, got %d", w)
	}
}

func TestRenderTableFitContent_Basic(t *testing.T) {
	rows := [][]string{
		{"name", "Alice"},
		{"city", "NYC"},
	}
	result := RenderTableFitContent(rows, true, 0)
	if result == "" {
		t.Fatal("expected non-empty output")
	}
	if !strings.Contains(result, "KEY") {
		t.Fatal("expected KEY header")
	}
	if !strings.Contains(result, "VALUE") {
		t.Fatal("expected VALUE header")
	}
	if !strings.Contains(result, "Alice") {
		t.Fatal("expected Alice in output")
	}
}

func TestRenderTableFitContent_WithMaxWidth(t *testing.T) {
	rows := [][]string{
		{"name", "A very long value that should be truncated somehow"},
	}
	result := RenderTableFitContent(rows, true, 30)
	if result == "" {
		t.Fatal("expected non-empty output")
	}
}

func TestRenderRows_Basic(t *testing.T) {
	rows := [][]string{
		{"name", "Alice"},
		{"age", "30"},
	}
	result := RenderRows(rows, true, 20, 30)
	if result == "" {
		t.Fatal("expected non-empty output")
	}
	if !strings.Contains(result, "KEY") {
		t.Fatal("expected KEY header")
	}
	if !strings.Contains(result, "Alice") {
		t.Fatal("expected Alice in output")
	}
}

func TestRenderRows_DefaultWidths(t *testing.T) {
	rows := [][]string{{"key", "value"}}
	result := RenderRows(rows, true, 0, 0)
	if result == "" {
		t.Fatal("expected non-empty output")
	}
}

func TestRenderTable_Map(t *testing.T) {
	data := map[string]interface{}{"name": "test", "count": 42}
	result := RenderTable(data, true, 20, 40)
	if result == "" {
		t.Fatal("expected non-empty output")
	}
	if !strings.Contains(result, "name") {
		t.Fatal("expected 'name' in output")
	}
}

func TestRenderTable_Array(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{"name": "alice"},
		map[string]interface{}{"name": "bob"},
	}
	result := RenderTable(data, true, 20, 40)
	if result == "" {
		t.Fatal("expected non-empty output")
	}
}

func TestRenderMultilineRow_SplitsLines(t *testing.T) {
	orig := MaxValueLines()
	SetMaxValueLines(10)
	t.Cleanup(func() { SetMaxValueLines(orig) })

	node := map[string]any{"msg": "line1\nline2\nline3"}
	out := RenderTable(node, true, 10, 30)

	assert.Contains(t, out, "line1")
	assert.Contains(t, out, "line2")
	assert.Contains(t, out, "line3")

	// Continuation lines should not repeat the key
	lines := strings.Split(out, "\n")
	var keyCount int
	for _, l := range lines {
		if strings.Contains(l, "msg") {
			keyCount++
		}
	}
	assert.Equal(t, 1, keyCount, "key should appear once; continuation lines have blank key")
}

func TestRenderMultilineRow_DisabledEscapesNewlines(t *testing.T) {
	orig := MaxValueLines()
	SetMaxValueLines(0)
	t.Cleanup(func() { SetMaxValueLines(orig) })

	node := map[string]any{"msg": "line1\nline2"}
	out := RenderTable(node, true, 10, 40)

	// With multi-line disabled, value should be flattened with escaped newlines
	assert.Contains(t, out, `line1\nline2`)
	// No continuation row
	lines := strings.Split(out, "\n")
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		assert.NotEqual(t, "line2", trimmed, "line2 should not appear as a separate row")
	}
}

func TestRenderMultilineRow_TruncatesWithIndicator(t *testing.T) {
	orig := MaxValueLines()
	SetMaxValueLines(3)
	t.Cleanup(func() { SetMaxValueLines(orig) })

	node := map[string]any{"data": "a\nb\nc\nd\ne"}
	out := RenderTable(node, true, 10, 20)

	// First 3 lines should appear
	assert.Contains(t, out, "a")
	assert.Contains(t, out, "b")
	assert.Contains(t, out, "c")
	// Truncation indicator
	assert.Contains(t, out, "...")
	// Lines beyond cap should not appear as standalone rows
	lines := strings.Split(out, "\n")
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "d" || trimmed == "e" {
			t.Errorf("line %q should be truncated", trimmed)
		}
	}
}

func TestRenderMultilineRow_UnlimitedShowsAll(t *testing.T) {
	orig := MaxValueLines()
	SetMaxValueLines(-1)
	t.Cleanup(func() { SetMaxValueLines(orig) })

	node := map[string]any{"data": "a\nb\nc\nd\ne"}
	out := RenderTable(node, true, 10, 20)

	for _, ch := range []string{"a", "b", "c", "d", "e"} {
		assert.Contains(t, out, ch)
	}
	// No truncation indicator
	lines := strings.Split(out, "\n")
	for _, l := range lines {
		assert.NotEqual(t, "...", strings.TrimSpace(l))
	}
}

func TestNaturalValueWidth_FlattenedWhenDisabled(t *testing.T) {
	orig := MaxValueLines()
	SetMaxValueLines(0)
	t.Cleanup(func() { SetMaxValueLines(orig) })

	// "a\nb" flattened to "a\\nb" has width 4
	w := naturalValueWidth("a\nb")
	assert.Equal(t, 4, w)
}

func TestNaturalValueWidth_WidestLineWhenEnabled(t *testing.T) {
	orig := MaxValueLines()
	SetMaxValueLines(10)
	t.Cleanup(func() { SetMaxValueLines(orig) })

	// Widest line is "hello" (5)
	w := naturalValueWidth("hi\nhello\nbye")
	assert.Equal(t, 5, w)
}

func TestDefaultMaxValueLines(t *testing.T) {
	assert.Equal(t, 10, DefaultMaxValueLines())
}

func TestRenderMultilineRow_ColorBranches(t *testing.T) {
	// Exercise the noColor=false paths in renderMultilineRow
	orig := MaxValueLines()
	t.Cleanup(func() { SetMaxValueLines(orig) })

	// Enabled mode with color
	SetMaxValueLines(10)
	node := map[string]any{"key": "a\nb"}
	out := RenderTable(node, false, 10, 20)
	assert.Contains(t, out, "a")
	assert.Contains(t, out, "b")

	// Disabled mode with color
	SetMaxValueLines(0)
	out = RenderTable(node, false, 10, 30)
	assert.Contains(t, out, `a\nb`)

	// Truncation with color
	SetMaxValueLines(2)
	node2 := map[string]any{"k": "x\ny\nz\nw"}
	out = RenderTable(node2, false, 10, 20)
	assert.Contains(t, out, "...")
}

func TestRenderMultilineRow_TrailingEmptyLineTrimmed(t *testing.T) {
	orig := MaxValueLines()
	SetMaxValueLines(10)
	t.Cleanup(func() { SetMaxValueLines(orig) })

	// YAML block scalars often have a trailing newline -> trailing empty line
	node := map[string]any{"note": "line1\nline2\n"}
	out := RenderTable(node, true, 10, 20)
	// Should show 2 lines, not 3 (trailing empty trimmed)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	var valueLines int
	for _, l := range lines[2:] { // skip header + separator
		if strings.TrimSpace(l) != "" {
			valueLines++
		}
	}
	assert.Equal(t, 2, valueLines, "trailing empty line should be trimmed")
}

func TestRenderRows_ColorBranches(t *testing.T) {
	orig := MaxValueLines()
	SetMaxValueLines(10)
	t.Cleanup(func() { SetMaxValueLines(orig) })

	rows := [][]string{{"key", "a\nb"}}
	out := RenderRows(rows, false, 10, 20)
	assert.Contains(t, out, "a")
	assert.Contains(t, out, "b")
}

func TestRenderTableFitContent_ColorBranches(t *testing.T) {
	orig := MaxValueLines()
	SetMaxValueLines(10)
	t.Cleanup(func() { SetMaxValueLines(orig) })

	rows := [][]string{{"key", "a\nb"}}
	out := RenderTableFitContent(rows, false, 0)
	assert.Contains(t, out, "a")
	assert.Contains(t, out, "b")
}
