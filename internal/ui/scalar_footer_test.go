package ui

import (
	"testing"
)

// TestScalarFooterShowsType verifies the footer displays the type when viewing a scalar
func TestScalarFooterShowsType(t *testing.T) {
	root := map[string]interface{}{
		"active":   true,
		"disabled": false,
		"count":    42,
		"name":     "test",
	}

	// Test viewing a bool scalar
	m := InitialModel(root)
	m.Node = root["active"]

	// Get the type label
	typeStr := nodeTypeLabel(m.Node)
	if typeStr != "bool" {
		t.Errorf("Expected type 'bool' for true, got %q", typeStr)
	}

	// Now test with string
	m.Node = root["name"]
	typeStr = nodeTypeLabel(m.Node)
	if typeStr != "string" {
		t.Errorf("Expected type 'string' for string value, got %q", typeStr)
	}
}
