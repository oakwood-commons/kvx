package ui

import (
	"strings"
	"testing"
)

// TestScalarDisplayFormat verifies that scalar values render without type headers
func TestScalarDisplayFormat(t *testing.T) {
	root := map[string]interface{}{
		"name":    "test-string",
		"count":   42,
		"enabled": true,
	}

	tests := []struct {
		name     string
		value    interface{}
		typeName string
	}{
		{"string scalar", root["name"], "string"},
		{"bool scalar", root["enabled"], "bool"},
		{"int scalar", root["count"], "int"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := RenderNodeTable(tt.value, false, 30, 60, 0)
			t.Logf("=== %s ===\n%s", tt.name, output)

			// Verify no type header like "BOOL", "STRING", "INT"
			lines := strings.Split(output, "\n")
			for _, line := range lines {
				upperLine := strings.ToUpper(line)
				if strings.Contains(upperLine, "BOOL") || strings.Contains(upperLine, "STRING") || strings.Contains(upperLine, "INT") {
					// Make sure it's not part of the actual value
					if !strings.Contains(line, "test-string") && !strings.Contains(line, "42") && !strings.Contains(line, "true") {
						t.Errorf("Found type header in output: %q", line)
					}
				}
			}
		})
	}
}
