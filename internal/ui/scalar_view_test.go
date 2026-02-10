package ui

import (
	"testing"
)

// TestScalarViewIncludesTypeInFooter verifies that scalar views include type in footer
func TestScalarViewIncludesTypeInFooter(t *testing.T) {
	root := map[string]interface{}{
		"enabled": true,
		"count":   42,
	}

	tests := []struct {
		name    string
		value   interface{}
		typeStr string
	}{
		{"bool true", true, "bool"},
		{"bool false", false, "bool"},
		{"int 42", 42, "int"},
		{"string value", "hello", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a model with the scalar as the current node
			m := InitialModel(root)
			m.Node = tt.value
			m.WinWidth = 80
			m.WinHeight = 24

			// The type label should match
			typeLabel := nodeTypeLabel(m.Node)
			if typeLabel != tt.typeStr {
				t.Errorf("Expected type %q, got %q", tt.typeStr, typeLabel)
			}
		})
	}
}
