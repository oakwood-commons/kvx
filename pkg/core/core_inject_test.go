package core

import "testing"

type fakeNavigator struct {
	setCalls   []SortOrder
	nodeAtRoot interface{}
	path       string
	nodeRows   [][]string
	nodeResult interface{}
	err        error
}

func (f *fakeNavigator) NodeAtPath(root interface{}, path string) (interface{}, error) {
	f.nodeAtRoot = root
	f.path = path
	return f.nodeResult, f.err
}

func (f *fakeNavigator) NodeToRows(node interface{}) [][]string {
	f.nodeAtRoot = node
	return f.nodeRows
}

func (f *fakeNavigator) SetSortOrder(order SortOrder) SortOrder {
	f.setCalls = append(f.setCalls, order)
	return SortNone
}

type fakeFormatter struct {
	renderInput  interface{}
	renderOut    string
	stringifyIn  interface{}
	stringifyOut string
}

func (f *fakeFormatter) RenderTable(node interface{}, noColor bool, keyColWidth, valueColWidth int) string {
	f.renderInput = node
	return f.renderOut
}

func (f *fakeFormatter) Stringify(node interface{}) string {
	f.stringifyIn = node
	return f.stringifyOut
}

func TestEngineUsesInjectedNavigatorForRows(t *testing.T) {
	nav := &fakeNavigator{nodeRows: [][]string{{"k", "v"}}}
	engine, err := New(WithNavigator(nav), WithSortOrder(SortDescending))
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	node := map[string]interface{}{"a": 1}
	rows := engine.Rows(node)
	if len(rows) != 1 || rows[0][0] != "k" {
		t.Fatalf("Rows returned %+v, want [[k v]]", rows)
	}
	if len(nav.setCalls) != 2 || nav.setCalls[0] != SortDescending || nav.setCalls[1] != SortNone {
		t.Fatalf("SetSortOrder calls = %+v, want [SortDescending SortNone]", nav.setCalls)
	}
}

func TestEngineUsesInjectedNavigatorNodeAtPath(t *testing.T) {
	expected := 42
	nav := &fakeNavigator{nodeResult: expected}
	engine, err := New(WithNavigator(nav))
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	out, err := engine.NodeAtPath(map[string]interface{}{"a": 1}, "a")
	if err != nil {
		t.Fatalf("NodeAtPath error: %v", err)
	}
	if out != expected {
		t.Fatalf("NodeAtPath = %v, want %v", out, expected)
	}
	if nav.path != "a" {
		t.Fatalf("navigator path = %q, want %q", nav.path, "a")
	}
}

func TestEngineUsesInjectedFormatter(t *testing.T) {
	fmtMock := &fakeFormatter{renderOut: "rendered", stringifyOut: "s"}
	engine, err := New(WithFormatter(fmtMock))
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	r := engine.RenderTable(123, false, 10, 10)
	if r != "rendered" {
		t.Fatalf("RenderTable = %q, want %q", r, "rendered")
	}
	if fmtMock.renderInput != 123 {
		t.Fatalf("render input = %v, want 123", fmtMock.renderInput)
	}

	s := engine.Stringify(456)
	if s != "s" {
		t.Fatalf("Stringify = %q, want %q", s, "s")
	}
	if fmtMock.stringifyIn != 456 {
		t.Fatalf("stringify input = %v, want 456", fmtMock.stringifyIn)
	}
}
