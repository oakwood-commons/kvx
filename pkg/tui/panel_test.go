package tui

import (
	"strings"
	"testing"
)

func TestRenderDataPanel(t *testing.T) {
	node := map[string]any{
		"name": "kvx",
	}

	out := RenderDataPanel(node, PanelOptions{
		NoColor:       true,
		KeyColWidth:   10,
		ValueColWidth: 10,
	})

	if !strings.Contains(out, "KEY") {
		t.Fatalf("expected header KEY, got output: %q", out)
	}
	if !strings.Contains(out, "VALUE") {
		t.Fatalf("expected header VALUE, got output: %q", out)
	}
	if !strings.Contains(out, "name") {
		t.Fatalf("expected key name, got output: %q", out)
	}
	if !strings.Contains(out, "kvx") {
		t.Fatalf("expected value kvx, got output: %q", out)
	}
}
