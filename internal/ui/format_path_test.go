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

	got := buildPathWithKey("_.tasks", "build-windows")
	if got != "_.tasks[\"build-windows\"]" {
		t.Fatalf("buildPathWithKey did not quote invalid identifier: got %q", got)
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
