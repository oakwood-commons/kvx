package core

import (
	"os"
	"testing"

	"github.com/go-logr/logr"
)

func TestEngineEvaluate(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	root := map[string]interface{}{
		"items": []interface{}{map[string]interface{}{"name": "a"}},
	}
	out, err := engine.Evaluate("_.items[0].name", root)
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}
	if out != "a" {
		t.Fatalf("Evaluate output = %v, want %v", out, "a")
	}
}

func TestRowsSortOrder(t *testing.T) {
	engine, err := New(WithSortOrder(SortAscending))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	node := map[string]interface{}{
		"b": 2,
		"a": 1,
	}
	rows := engine.Rows(node)
	if len(rows) < 2 {
		t.Fatalf("expected rows, got %d", len(rows))
	}
	if rows[0][0] != "a" {
		t.Fatalf("first row key = %q, want %q", rows[0][0], "a")
	}
}

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/data.yaml"
	if err := os.WriteFile(path, []byte("name: test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
	root, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile error: %v", err)
	}
	m, ok := root.(map[string]interface{})
	if !ok {
		t.Fatalf("LoadFile type = %T, want map", root)
	}
	if m["name"] != "test" {
		t.Fatalf("LoadFile name = %v, want %v", m["name"], "test")
	}
}

func TestLoadObject(t *testing.T) {
	obj := map[string]any{"name": "test"}
	root, err := LoadObject(obj)
	if err != nil {
		t.Fatalf("LoadObject error: %v", err)
	}
	rootMap, ok := root.(map[string]any)
	if !ok {
		t.Fatalf("LoadObject root type = %T, want map[string]any", root)
	}
	rootMap["role"] = "admin"
	if obj["role"] != "admin" {
		t.Fatalf("LoadObject root pointer changed")
	}

	if _, err := LoadObject(nil); err == nil {
		t.Fatalf("LoadObject nil should error")
	}
}

func TestLoadRoot(t *testing.T) {
	root, err := LoadRoot(`{"name":"test"}`)
	if err != nil {
		t.Fatalf("LoadRoot error: %v", err)
	}
	m, ok := root.(map[string]interface{})
	if !ok {
		t.Fatalf("LoadRoot type = %T, want map", root)
	}
	if m["name"] != "test" {
		t.Fatalf("LoadRoot name = %v, want test", m["name"])
	}
}

func TestLoadRootBytes(t *testing.T) {
	root, err := LoadRootBytes([]byte(`{"count":42}`))
	if err != nil {
		t.Fatalf("LoadRootBytes error: %v", err)
	}
	m, ok := root.(map[string]interface{})
	if !ok {
		t.Fatalf("LoadRootBytes type = %T, want map", root)
	}
	if m["count"] != float64(42) {
		t.Fatalf("LoadRootBytes count = %v, want 42", m["count"])
	}
}

func TestEngineWithEvaluator(t *testing.T) {
	// WithEvaluator(nil) still gets a default evaluator from New()
	engine, err := New(WithEvaluator(nil))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	result, err := engine.Evaluate("_.name", map[string]interface{}{"name": "test"})
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}
	if result != "test" {
		t.Fatalf("Evaluate = %v, want test", result)
	}
}

func TestEngineRenderTable(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	node := map[string]interface{}{"key": "value"}
	result := engine.RenderTable(node, true, 20, 40)
	if result == "" {
		t.Fatal("RenderTable returned empty string")
	}
}

func TestEngineStringify(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	result := engine.Stringify("hello")
	if result != "hello" {
		t.Fatalf("Stringify = %q, want hello", result)
	}
}

func TestEngineNodeAtPath(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	root := map[string]interface{}{
		"user": map[string]interface{}{"name": "alice"},
	}
	result, err := engine.NodeAtPath(root, "user.name")
	if err != nil {
		t.Fatalf("NodeAtPath error: %v", err)
	}
	if result != "alice" {
		t.Fatalf("NodeAtPath = %v, want alice", result)
	}
}

func TestToNavigatorSort(t *testing.T) {
	tests := []struct {
		input SortOrder
	}{
		{SortAscending},
		{SortDescending},
		{SortNone},
		{SortOrder("invalid")},
	}
	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			// Roundtrip: toNavigatorSort then fromNavigatorSort should preserve known values
			nav := toNavigatorSort(tt.input)
			back := fromNavigatorSort(nav)
			if tt.input == SortAscending || tt.input == SortDescending || tt.input == SortNone {
				if back != tt.input {
					t.Fatalf("roundtrip failed: %q -> %q", tt.input, back)
				}
			}
		})
	}
}

func TestEngineRowsNilNavigator(t *testing.T) {
	engine := &Engine{Navigator: nil}
	rows := engine.Rows(map[string]interface{}{"a": 1})
	// ensureNavigator fills it in, so we should get rows
	if rows == nil {
		t.Fatal("expected non-nil rows after ensureNavigator")
	}
}

func TestEngineRenderTableNil(t *testing.T) {
	engine := &Engine{}
	result := engine.RenderTable(map[string]interface{}{"a": 1}, true, 20, 40)
	// ensureFormatter fills it in
	if result == "" {
		t.Fatal("expected non-empty after ensureFormatter")
	}
}

func TestEngineStringifyNil(t *testing.T) {
	engine := &Engine{}
	result := engine.Stringify("test")
	// ensureFormatter fills it in
	if result != "test" {
		t.Fatalf("Stringify = %q, want test", result)
	}
}

func TestLoadFileWithLogger(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/data.json"
	if err := os.WriteFile(path, []byte(`{"ok":true}`), 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
	root, err := LoadFileWithLogger(path, testLogger())
	if err != nil {
		t.Fatalf("LoadFileWithLogger error: %v", err)
	}
	m, ok := root.(map[string]interface{})
	if !ok {
		t.Fatalf("type = %T, want map", root)
	}
	if m["ok"] != true {
		t.Fatalf("ok = %v, want true", m["ok"])
	}
}

func TestLoadRootBytesWithLogger(t *testing.T) {
	root, err := LoadRootBytesWithLogger([]byte(`name: test`), testLogger())
	if err != nil {
		t.Fatalf("LoadRootBytesWithLogger error: %v", err)
	}
	m, ok := root.(map[string]interface{})
	if !ok {
		t.Fatalf("type = %T, want map", root)
	}
	if m["name"] != "test" {
		t.Fatalf("name = %v, want test", m["name"])
	}
}

func testLogger() logr.Logger {
	return logr.Discard()
}
