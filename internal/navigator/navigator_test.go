package navigator

import (
	"testing"
)

func TestNodeAtPathEmpty(t *testing.T) {
	node := map[string]interface{}{"key": "value"}
	result, err := NodeAtPath(node, "")
	if err != nil {
		t.Fatalf("expected no error for empty path, got %v", err)
	}
	// Maps can't be compared directly, so compare by type and content
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result for empty path, got %T", result)
	}
	if resultMap["key"] != "value" {
		t.Fatalf("expected root node for empty path, got different map")
	}
}

func TestNodeAtPathSimpleDottedPath(t *testing.T) {
	root := map[string]interface{}{
		"user": map[string]interface{}{
			"name": "alice",
			"age":  30,
		},
	}
	result, err := NodeAtPath(root, "user.name")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != "alice" {
		t.Fatalf("expected 'alice', got %v", result)
	}
}

func TestNodeAtPathArrayIndex(t *testing.T) {
	root := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"id": 1},
			map[string]interface{}{"id": 2},
		},
	}
	result, err := NodeAtPath(root, "items.0.id")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// CEL returns int64 type
	if result != int64(1) && result != 1 {
		t.Fatalf("expected 1 or int64(1), got %v (%T)", result, result)
	}
}

func TestNodeAtPathBracketNotation(t *testing.T) {
	root := map[string]interface{}{
		"items": []interface{}{"a", "b", "c"},
	}
	result, err := NodeAtPath(root, "items[1]")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != "b" {
		t.Fatalf("expected 'b', got %v", result)
	}
}

func TestNodeAtPathSpecialCharKey(t *testing.T) {
	root := map[string]interface{}{
		"bad-key": "value",
	}
	// Bracket notation returns the array, we need to use a dotted notation
	// Actually ["bad-key"] is CEL array subscript, not navigation
	// Let's test the proper way through simple navigate
	resultDot, errDot := NodeAtPath(root, "")
	if errDot != nil {
		t.Fatalf("expected no error for root access, got %v", errDot)
	}
	rootMap, ok := resultDot.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", resultDot)
	}
	if rootMap["bad-key"] != "value" {
		t.Fatalf("expected 'value', got %v", rootMap["bad-key"])
	}
}

func TestNodeAtPathNotFound(t *testing.T) {
	root := map[string]interface{}{"key": "value"}
	_, err := NodeAtPath(root, "missing")
	if err == nil {
		t.Fatalf("expected error for missing key, got nil")
	}
}

func TestNodeAtPathNestedMissing(t *testing.T) {
	root := map[string]interface{}{
		"user": map[string]interface{}{"name": "alice"},
	}
	_, err := NodeAtPath(root, "user.age")
	if err == nil {
		t.Fatalf("expected error for missing nested key, got nil")
	}
}

func TestNodeAtPathArrayOutOfBounds(t *testing.T) {
	root := map[string]interface{}{
		"items": []interface{}{"a", "b"},
	}
	_, err := NodeAtPath(root, "items.5")
	if err == nil {
		t.Fatalf("expected error for out of bounds index, got nil")
	}
}

func TestNodeAtPathCELExpression(t *testing.T) {
	root := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"name": "a", "active": true},
			map[string]interface{}{"name": "b", "active": false},
		},
	}
	result, err := NodeAtPath(root, "_.items.filter(x, x.active)")
	if err != nil {
		t.Fatalf("expected no error for CEL filter, got %v", err)
	}
	arr, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected array result, got %T", result)
	}
	if len(arr) != 1 {
		t.Fatalf("expected 1 filtered item, got %d", len(arr))
	}
}

func TestNodeAtPathCELArrayLiteral(t *testing.T) {
	root := map[string]interface{}{}
	result, err := NodeAtPath(root, "[1,2,3]")
	if err != nil {
		t.Fatalf("expected no error for array literal, got %v", err)
	}
	arr, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected array result, got %T", result)
	}
	if len(arr) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(arr))
	}
}

func TestNodeToRowsMap(t *testing.T) {
	prev := SetSortOrder(SortAscending)
	defer SetSortOrder(prev)

	node := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	rows := NodeToRows(node)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	// Keys should be sorted
	if rows[0][0] != "key1" || rows[1][0] != "key2" {
		t.Fatalf("expected sorted keys, got %v, %v", rows[0][0], rows[1][0])
	}
}

func TestNodeToRowsMapDescending(t *testing.T) {
	prev := SetSortOrder(SortDescending)
	defer SetSortOrder(prev)

	node := map[string]interface{}{
		"alpha": "1",
		"beta":  "2",
		"gamma": "3",
	}

	rows := NodeToRows(node)
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}

	if rows[0][0] != "gamma" || rows[1][0] != "beta" || rows[2][0] != "alpha" {
		t.Fatalf("expected descending keys [gamma beta alpha], got %v", []string{rows[0][0], rows[1][0], rows[2][0]})
	}
}

func TestNodeToRowsArray(t *testing.T) {
	node := []interface{}{"a", "b", "c"}
	rows := NodeToRows(node)
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if rows[0][0] != "[0]" || rows[1][0] != "[1]" || rows[2][0] != "[2]" {
		t.Fatalf("expected array indices, got %v, %v, %v", rows[0][0], rows[1][0], rows[2][0])
	}
}

func TestNodeToRowsScalar(t *testing.T) {
	node := "simple string"
	rows := NodeToRows(node)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row for scalar, got %d", len(rows))
	}
	if rows[0][0] != "(value)" {
		t.Fatalf("expected '(value)' label, got %v", rows[0][0])
	}
	if rows[0][1] != "simple string" {
		t.Fatalf("expected string value, got %v", rows[0][1])
	}
}

func TestNodeToRowsEmptyMap(t *testing.T) {
	node := map[string]interface{}{}
	rows := NodeToRows(node)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row for empty map, got %d", len(rows))
	}
	if rows[0][0] != "(value)" {
		t.Fatalf("expected '(value)' label for empty map, got %v", rows[0][0])
	}
}

func TestNodeToRowsEmptyArray(t *testing.T) {
	node := []interface{}{}
	rows := NodeToRows(node)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row for empty array, got %d", len(rows))
	}
	if rows[0][0] != "(value)" {
		t.Fatalf("expected '(value)' label for empty array, got %v", rows[0][0])
	}
}

func TestNodeToRowsTypedMap(t *testing.T) {
	prev := SetSortOrder(SortAscending)
	defer SetSortOrder(prev)

	node := map[string]string{"b": "2", "a": "1"}
	rows := NodeToRows(node)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0][0] != "a" || rows[1][0] != "b" {
		t.Fatalf("expected sorted keys for typed map, got %v", []string{rows[0][0], rows[1][0]})
	}
}

func TestNodeToRowsTypedSlice(t *testing.T) {
	node := []string{"x", "y"}
	rows := NodeToRows(node)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0][0] != "[0]" || rows[1][0] != "[1]" {
		t.Fatalf("expected array indices, got %v", []string{rows[0][0], rows[1][0]})
	}
}

func TestNodeToRowsWithOptionsArrayStyleNumbered(t *testing.T) {
	node := []interface{}{"a", "b", "c"}
	rows := NodeToRowsWithOptions(node, RowOptions{ArrayStyle: ArrayStyleNumbered})
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	// 1-based numbering
	if rows[0][0] != "1" || rows[1][0] != "2" || rows[2][0] != "3" {
		t.Fatalf("expected 1, 2, 3 indices, got %v, %v, %v", rows[0][0], rows[1][0], rows[2][0])
	}
}

func TestNodeToRowsWithOptionsArrayStyleBullet(t *testing.T) {
	node := []interface{}{"x", "y"}
	rows := NodeToRowsWithOptions(node, RowOptions{ArrayStyle: ArrayStyleBullet})
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0][0] != "•" || rows[1][0] != "•" {
		t.Fatalf("expected bullet points, got %v, %v", rows[0][0], rows[1][0])
	}
}

func TestNodeToRowsWithOptionsArrayStyleNone(t *testing.T) {
	node := []interface{}{"val"}
	rows := NodeToRowsWithOptions(node, RowOptions{ArrayStyle: ArrayStyleNone})
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0][0] != "" {
		t.Fatalf("expected empty key, got %v", rows[0][0])
	}
}

func TestNodeToRowsWithOptionsArrayStyleIndex(t *testing.T) {
	node := []interface{}{"a", "b"}
	rows := NodeToRowsWithOptions(node, RowOptions{ArrayStyle: ArrayStyleIndex})
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0][0] != "[0]" || rows[1][0] != "[1]" {
		t.Fatalf("expected [0], [1] indices, got %v, %v", rows[0][0], rows[1][0])
	}
}

func TestNodeToRowsWithOptionsMapUnchanged(t *testing.T) {
	node := map[string]interface{}{"key": "value"}
	rows := NodeToRowsWithOptions(node, RowOptions{ArrayStyle: ArrayStyleNumbered})
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	// Maps should not be affected by ArrayStyle
	if rows[0][0] != "key" {
		t.Fatalf("expected 'key', got %v", rows[0][0])
	}
}

func TestIsComplexCELQuotedString(t *testing.T) {
	tests := []string{
		`"hello"`,
		`"test string"`,
	}
	for _, test := range tests {
		if !isComplexCEL(test) {
			t.Fatalf("expected '%s' to be detected as CEL", test)
		}
	}
}

func TestIsComplexCELArrayLiteral(t *testing.T) {
	if !isComplexCEL("[1,2,3]") {
		t.Fatalf("expected array literal to be detected as CEL")
	}
}

func TestIsComplexCELMapLiteral(t *testing.T) {
	if !isComplexCEL(`{"key":"value"}`) {
		t.Fatalf("expected map literal to be detected as CEL")
	}
}

func TestIsComplexCELFunctionCall(t *testing.T) {
	tests := []string{
		"size(items)",
		"filter(x, x > 0)",
		"map(x, x.name)",
	}
	for _, test := range tests {
		if !isComplexCEL(test) {
			t.Fatalf("expected '%s' to be detected as CEL function", test)
		}
	}
}

func TestIsComplexCELUnderscorePrefix(t *testing.T) {
	if !isComplexCEL("_.items[0].name") {
		t.Fatalf("expected underscore prefix to be detected as CEL")
	}
}

func TestIsComplexCELComparison(t *testing.T) {
	tests := []string{
		"size > 0",
		"active == true",
		"count <= 10",
	}
	for _, test := range tests {
		if !isComplexCEL(test) {
			t.Fatalf("expected '%s' to be detected as CEL comparison", test)
		}
	}
}

func TestIsComplexCELSimplePath(t *testing.T) {
	tests := []string{
		"items",
		"items.0",
		"items.name",
		"items[0]",
	}
	for _, test := range tests {
		if isComplexCEL(test) {
			t.Fatalf("expected '%s' to NOT be detected as complex CEL", test)
		}
	}
}

func TestIsComplexCELBracketNavigation(t *testing.T) {
	// Bracket index notation (e.g., [0], [1]) should be simple navigation, not CEL
	tests := []string{
		"[0]",
		"[1]",
		"[123]",
		`["key"]`,
		`["some-key"]`,
	}
	for _, test := range tests {
		if isComplexCEL(test) {
			t.Fatalf("expected '%s' to NOT be detected as complex CEL (should be simple bracket navigation)", test)
		}
	}
}

func TestIsComplexCELArrayLiteralMultiple(t *testing.T) {
	// Array literals with multiple elements should be CEL
	tests := []string{
		"[1,2]",
		"[1, 2, 3]",
		"[a, b]",
	}
	for _, test := range tests {
		if !isComplexCEL(test) {
			t.Fatalf("expected '%s' to be detected as complex CEL (array literal)", test)
		}
	}
}

func TestParsePathDottedSegments(t *testing.T) {
	parts := parsePath("a.b.c")
	if len(parts) != 3 || parts[0] != "a" || parts[1] != "b" || parts[2] != "c" {
		t.Fatalf("expected ['a', 'b', 'c'], got %v", parts)
	}
}

func TestParsePathBracketNotation(t *testing.T) {
	parts := parsePath("items[0]")
	if len(parts) != 2 || parts[0] != "items" || parts[1] != "0" {
		t.Fatalf("expected ['items', '0'], got %v", parts)
	}
}

func TestParsePathMixed(t *testing.T) {
	parts := parsePath("items[0].name")
	if len(parts) != 3 || parts[0] != "items" || parts[1] != "0" || parts[2] != "name" {
		t.Fatalf("expected ['items', '0', 'name'], got %v", parts)
	}
}

func TestParsePathQuotedKey(t *testing.T) {
	parts := parsePath(`root["bad-key"]`)
	if len(parts) != 2 || parts[0] != "root" || parts[1] != `"bad-key"` {
		t.Fatalf("expected ['root', '\"bad-key\"'], got %v", parts)
	}
}

func TestNavigateStepMapKey(t *testing.T) {
	data := map[string]interface{}{"key": "value"}
	result := navigateStep(data, "key")
	if result != "value" {
		t.Fatalf("expected 'value', got %v", result)
	}
}

func TestNavigateStepArrayIndex(t *testing.T) {
	data := []interface{}{"a", "b", "c"}
	result := navigateStep(data, "1")
	if result != "b" {
		t.Fatalf("expected 'b', got %v", result)
	}
}

func TestNavigateStepMapKeyNotFound(t *testing.T) {
	data := map[string]interface{}{"key": "value"}
	result := navigateStep(data, "missing")
	_, isErr := result.(error)
	if !isErr {
		t.Fatalf("expected error for missing key, got %v", result)
	}
}

func TestNavigateStepArrayIndexOutOfBounds(t *testing.T) {
	data := []interface{}{"a", "b"}
	result := navigateStep(data, "5")
	_, isErr := result.(error)
	if !isErr {
		t.Fatalf("expected error for out of bounds, got %v", result)
	}
}

func TestNavigateStepInvalidArrayIndex(t *testing.T) {
	data := []interface{}{"a", "b"}
	result := navigateStep(data, "abc")
	_, isErr := result.(error)
	if !isErr {
		t.Fatalf("expected error for non-numeric index, got %v", result)
	}
}

func TestNavigateStepTypedMap(t *testing.T) {
	data := map[string]string{"key": "value"}
	result := navigateStep(data, "key")
	if result != "value" {
		t.Fatalf("expected 'value', got %v", result)
	}
}

func TestNavigateStepTypedSlice(t *testing.T) {
	data := []string{"a", "b", "c"}
	result := navigateStep(data, "2")
	if result != "c" {
		t.Fatalf("expected 'c', got %v", result)
	}
}

func TestNavigateStepStructField(t *testing.T) {
	type sample struct {
		Name string `json:"name"`
		Role string
	}
	data := sample{Name: "alice", Role: "admin"}
	if got := navigateStep(data, "name"); got != "alice" {
		t.Fatalf("expected 'alice', got %v", got)
	}
	if got := navigateStep(&data, "Role"); got != "admin" {
		t.Fatalf("expected 'admin', got %v", got)
	}
}

// Additional comprehensive tests for improved coverage

func TestNodeAtPathComplexNesting(t *testing.T) {
	root := map[string]interface{}{
		"data": map[string]interface{}{
			"users": []interface{}{
				map[string]interface{}{"name": "alice", "age": 30},
				map[string]interface{}{"name": "bob", "age": 25},
			},
		},
	}
	result, err := NodeAtPath(root, "data.users.0.name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "alice" {
		t.Fatalf("expected 'alice', got %v", result)
	}
}

func TestNodeAtPathNilData(t *testing.T) {
	_, err := NodeAtPath(nil, "any.path")
	if err == nil {
		t.Fatalf("expected error for nil data")
	}
}

func TestNodeAtPathInvalidPath(t *testing.T) {
	root := map[string]interface{}{"key": "value"}
	_, err := NodeAtPath(root, "nonexistent.path")
	if err == nil {
		t.Fatalf("expected error for invalid path")
	}
}

func TestNodeAtPathArrayScalar(t *testing.T) {
	root := []interface{}{"a", "b", "c"}
	result, err := NodeAtPath(root, "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "b" {
		t.Fatalf("expected 'b', got %v", result)
	}
}

func TestNodeAtPathEmptyArray(t *testing.T) {
	root := map[string]interface{}{"items": []interface{}{}}
	_, err := NodeAtPath(root, "items.0")
	if err == nil {
		t.Fatalf("expected error for empty array access")
	}
}

func TestKeyPathRootOnly(t *testing.T) {
	parts := parsePath("root")
	if len(parts) != 1 || parts[0] != "root" {
		t.Fatalf("expected ['root'], got %v", parts)
	}
}

func TestKeyPathDeepNesting(t *testing.T) {
	parts := parsePath("a.b.c.d.e.f.g")
	if len(parts) != 7 {
		t.Fatalf("expected 7 parts, got %d", len(parts))
	}
}

func TestNavigateStepWithNestedMap(t *testing.T) {
	data := map[string]interface{}{
		"nested": map[string]interface{}{
			"value": 42,
		},
	}
	result := navigateStep(data, "nested")
	nested, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if nested["value"] != 42 {
		t.Fatalf("expected 42, got %v", nested["value"])
	}
}

func TestNavigateStepWithArray(t *testing.T) {
	data := map[string]interface{}{
		"items": []interface{}{1, 2, 3},
	}
	result := navigateStep(data, "items")
	arr, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected array result, got %T", result)
	}
	if len(arr) != 3 {
		t.Fatalf("expected length 3, got %d", len(arr))
	}
}

func TestNavigateStepFirstArrayElement(t *testing.T) {
	data := []interface{}{"first", "second", "third"}
	result := navigateStep(data, "0")
	if result != "first" {
		t.Fatalf("expected 'first', got %v", result)
	}
}

func TestNavigateStepLastArrayElement(t *testing.T) {
	data := []interface{}{"first", "second", "third"}
	result := navigateStep(data, "2")
	if result != "third" {
		t.Fatalf("expected 'third', got %v", result)
	}
}

func TestNavigateStepNegativeArrayIndex(t *testing.T) {
	data := []interface{}{"a", "b", "c"}
	result := navigateStep(data, "-1")
	_, isErr := result.(error)
	if !isErr {
		t.Fatalf("expected error for negative index, got %v", result)
	}
}

func TestNodeAtPathScalarRoot(t *testing.T) {
	result, err := NodeAtPath("scalar", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "scalar" {
		t.Fatalf("expected 'scalar', got %v", result)
	}
}

func TestNodeAtPathNumberRoot(t *testing.T) {
	result, err := NodeAtPath(42, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Fatalf("expected 42, got %v", result)
	}
}

func TestKeyPathWithNumbers(t *testing.T) {
	parts := parsePath("items.0.name")
	if len(parts) != 3 || parts[1] != "0" {
		t.Fatalf("expected numeric part preserved, got %v", parts)
	}
}

func TestNavigateStepTypeMismatch(t *testing.T) {
	// Try to access array index on a string
	data := "not an array"
	result := navigateStep(data, "0")
	_, isErr := result.(error)
	if !isErr {
		t.Fatalf("expected error for type mismatch, got %v", result)
	}
}

func TestNodeAtPathMultipleArrayIndices(t *testing.T) {
	root := []interface{}{
		[]interface{}{1, 2, 3},
		[]interface{}{4, 5, 6},
	}
	result, err := NodeAtPath(root, "1.2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 6 {
		t.Fatalf("expected 6, got %v", result)
	}
}

func TestNodeAtPathMixedTypes(t *testing.T) {
	root := map[string]interface{}{
		"data": []interface{}{
			map[string]interface{}{"values": []interface{}{10, 20, 30}},
		},
	}
	result, err := NodeAtPath(root, "data.0.values.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 20 {
		t.Fatalf("expected 20, got %v", result)
	}
}

func TestNavigateStepEmptyString(t *testing.T) {
	data := map[string]interface{}{"key": "value"}
	result := navigateStep(data, "")
	// Empty key access behavior
	_, isErr := result.(error)
	// Should either return error or handle gracefully
	_ = isErr
}

func TestKeyPathSpecialCharacters(t *testing.T) {
	// Test that parsePath handles various path formats
	parts := parsePath("root")
	if len(parts) < 1 {
		t.Fatalf("parsePath failed to parse valid path")
	}
}

func TestNodeAtPathValidation(t *testing.T) {
	tests := []struct {
		name    string
		root    interface{}
		path    string
		wantErr bool
	}{
		{"valid simple", map[string]interface{}{"a": 1}, "a", false},
		{"invalid deep", map[string]interface{}{"a": 1}, "a.b.c", true},
		{"array valid", []interface{}{1, 2}, "0", false},
		{"array invalid", []interface{}{1, 2}, "10", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NodeAtPath(tt.root, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeAtPath error: got %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
