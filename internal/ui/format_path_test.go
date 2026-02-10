package ui

import "testing"

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
