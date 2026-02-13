package ui

import (
	"reflect"
	"testing"
)

func TestSplitPathSegments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want []string
	}{
		{name: "empty", in: "", want: nil},
		{name: "root only", in: "_", want: nil},
		{name: "simple key", in: "_.items", want: []string{"items"}},
		{name: "nested keys", in: "_.regions.asia", want: []string{"regions", "asia"}},
		{name: "with array index", in: "_.items[0]", want: []string{"items", "0"}},
		{name: "bracket at root", in: "_[0]", want: []string{"0"}},
		{name: "quoted key", in: `_.tasks["build-windows"]`, want: []string{"tasks", `"build-windows"`}},
		// Underscore-prefixed keys (regression tests for stripping bug)
		{name: "double underscore key", in: "_.__hello", want: []string{"__hello"}},
		{name: "single underscore key", in: "_._internal", want: []string{"_internal"}},
		{name: "nested underscore keys", in: "_._meta._version", want: []string{"_meta", "_version"}},
		{name: "triple underscore key", in: "_.___test", want: []string{"___test"}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := splitPathSegments(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("splitPathSegments(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestFormatPathForDisplay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty becomes underscore", in: "", want: "_"},
		{name: "keeps underscore", in: "_", want: "_"},
		{name: "keeps quoted literal", in: "\"hi\"", want: "\"hi\""},
		{name: "keeps array literal", in: "[1,2]", want: "[1,2]"},
		{name: "adds prefix for plain path", in: "regions.asia", want: "_.regions.asia"},
		{name: "keeps prefixed path", in: "_.items[0]", want: "_.items[0]"},
		{name: "quotes invalid key segment", in: "_.tasks.build-windows", want: "_.tasks[\"build-windows\"]"},
		{name: "quotes invalid key without prefix", in: "tasks.build-windows", want: "_.tasks[\"build-windows\"]"},
		// Underscore-prefixed keys (regression tests for stripping bug)
		{name: "preserves double underscore key", in: "_.__hello", want: "_.__hello"},
		{name: "adds prefix to underscore key", in: "__hello", want: "_.__hello"},
		{name: "preserves internal underscore key", in: "_._internal", want: "_._internal"},
		{name: "preserves nested underscore keys", in: "_._meta._version", want: "_._meta._version"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := formatPathForDisplay(tt.in); got != tt.want {
				t.Fatalf("formatPathForDisplay(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestBuildPathWithKeyInvalidIdentifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		basePath string
		key      string
		want     string
	}{
		{name: "dash in key", basePath: "_.tasks", key: "build-windows", want: `_.tasks["build-windows"]`},
		{name: "dot in key", basePath: "_", key: "foo.bar", want: `_["foo.bar"]`},
		{name: "slash in key", basePath: "_", key: "api/v1", want: `_["api/v1"]`},
		{name: "colon in key", basePath: "_", key: "key:value", want: `_["key:value"]`},
		{name: "space in key", basePath: "_", key: "my key", want: `_["my key"]`},
		{name: "valid identifier uses dot notation", basePath: "_", key: "validKey", want: "_.validKey"},
		{name: "underscore key uses dot notation", basePath: "_", key: "_internal", want: "_._internal"},
		{name: "numeric suffix valid", basePath: "_", key: "item1", want: "_.item1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildPathWithKey(tt.basePath, tt.key)
			if got != tt.want {
				t.Fatalf("buildPathWithKey(%q, %q) = %q, want %q", tt.basePath, tt.key, got, tt.want)
			}
		})
	}
}

func TestIsValidCELIdentifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Valid identifiers
		{name: "simple word", input: "items", want: true},
		{name: "underscore prefix", input: "_internal", want: true},
		{name: "double underscore", input: "__hello", want: true},
		{name: "with digits", input: "item1", want: true},
		{name: "mixed case", input: "myVariable", want: true},
		{name: "single underscore", input: "_", want: true},

		// Invalid identifiers
		{name: "empty string", input: "", want: false},
		{name: "contains dot", input: "foo.bar", want: false},
		{name: "contains dash", input: "build-windows", want: false},
		{name: "contains slash", input: "api/v1", want: false},
		{name: "contains colon", input: "key:value", want: false},
		{name: "contains space", input: "my key", want: false},
		{name: "starts with digit", input: "123start", want: false},
		{name: "contains at symbol", input: "user@domain", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isValidCELIdentifier(tt.input)
			if got != tt.want {
				t.Fatalf("isValidCELIdentifier(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizePathForModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "root underscore", in: "_", want: ""},
		{name: "simple", in: "items", want: "_.items"},
		{name: "numeric segment", in: "items.0", want: "_.items[0]"},
		{name: "already prefixed", in: "_.items[0]", want: "_.items[0]"},
		{name: "invalid identifier", in: "tasks.build-windows", want: "_.tasks[\"build-windows\"]"},
		// Underscore-prefixed keys (regression tests for stripping bug)
		{name: "underscore prefixed key", in: "__hello", want: "_.__hello"},
		{name: "internal underscore", in: "_internal.value", want: "_._internal.value"},
		{name: "already prefixed underscore key", in: "_.__hello", want: "_.__hello"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := normalizePathForModel(tt.in); got != tt.want {
				t.Fatalf("normalizePathForModel(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRemoveLastSegment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "simple key", in: "items", want: ""},
		{name: "dot notation", in: "items.tags", want: "items"},
		{name: "nested dot notation", in: "items.tags.name", want: "items.tags"},
		{name: "bracket index", in: "items[0]", want: "items"},
		{name: "bracket after dot", in: "items.tags[1]", want: "items.tags"},
		{name: "multiple brackets", in: "items[0].tags[1]", want: "items[0].tags"},
		// Quoted keys with dots inside (regression test for navigation bug)
		{name: "quoted key with dot", in: `_["foo.bar"]`, want: "_"},
		{name: "nested quoted key with dot", in: `_.items["foo.bar"]`, want: "_.items"},
		{name: "quoted key after bracket", in: `_[0]["foo.bar"]`, want: "_[0]"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := removeLastSegment(tt.in)
			if got != tt.want {
				t.Fatalf("removeLastSegment(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
func TestLastDotOutsideBrackets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want int
	}{
		{name: "empty", in: "", want: -1},
		{name: "no dots", in: "items", want: -1},
		{name: "simple dot", in: "items.name", want: 5},
		{name: "multiple dots", in: "items.tags.name", want: 10},
		{name: "dot inside brackets", in: `_["foo.bar"]`, want: -1},
		{name: "dot outside with dot inside", in: `_.items["foo.bar"]`, want: 1},
		{name: "multiple dots outside", in: `_.items.tags["foo.bar"]`, want: 7},
		{name: "dot after bracket", in: `_["foo.bar"].name`, want: 12},
		{name: "nested brackets with dots", in: `_["a.b"]["c.d"].x`, want: 15},
		{name: "no closing quote", in: `_["foo.bar`, want: -1},
		{name: "bracket index then dot", in: `_[0].items`, want: 4},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := lastDotOutsideBrackets(tt.in)
			if got != tt.want {
				t.Fatalf("lastDotOutsideBrackets(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestIsCompletePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "empty string", input: "", want: false},
		{name: "underscore only", input: "_", want: false},
		{name: "underscore with dot", input: "_.", want: false},
		{name: "simple key", input: "_.items", want: false},
		{name: "nested key", input: "_.items.name", want: false},
		{name: "bracket key", input: `_["foo"]`, want: true},
		{name: "bracket key with dot", input: `_["foo.bar"]`, want: true},
		{name: "nested bracket key", input: `_.items["foo.bar"]`, want: true},
		{name: "nested then dot", input: `_.items["foo.bar"].`, want: false},
		{name: "function call", input: "_.all()", want: false},
		{name: "partial after dot", input: "_.ite", want: false},
		{name: "after bracket waiting", input: `_["foo.bar"].`, want: false},
		{name: "deep nested complete", input: `_.a.b["c.d"].e`, want: false},
		{name: "array index", input: "_[0]", want: true},
		{name: "nested array", input: "_.items[0]", want: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isCompletePath(tt.input)
			if got != tt.want {
				t.Fatalf("isCompletePath(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
