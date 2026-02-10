package core

import (
	"os"
	"testing"
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
