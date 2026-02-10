package completion_test

import (
	"testing"

	"github.com/oakwood-commons/kvx/internal/completion"
)

// TestNewCompletionEngine verifies the completion engine can be created and used
func TestNewCompletionEngine(t *testing.T) {
	// Create CEL provider
	provider, err := completion.NewCELProvider()
	if err != nil {
		t.Fatalf("Failed to create CEL provider: %v", err)
	}

	// Create engine
	engine := completion.NewEngine(provider)
	if engine == nil {
		t.Fatal("Engine should not be nil")
	}

	// Test function discovery
	functions := engine.GetFunctions()
	if len(functions) == 0 {
		t.Error("Expected to discover some functions")
	}

	t.Logf("Discovered %d functions", len(functions))
	for i, fn := range functions {
		if i < 5 { // Show first 5
			t.Logf("  - %s: %s", fn.Name, fn.Signature)
		}
	}
}

// TestCompletionFiltering verifies completion filtering works
func TestCompletionFiltering(t *testing.T) {
	provider, err := completion.NewCELProvider()
	if err != nil {
		t.Fatalf("Failed to create CEL provider: %v", err)
	}

	engine := completion.NewEngine(provider)

	// Test data
	data := map[string]interface{}{
		"users": []interface{}{
			map[string]interface{}{"name": "Alice", "active": true},
			map[string]interface{}{"name": "Bob", "active": false},
		},
		"count": 42,
	}

	// Test completions for "_.u"
	ctx := completion.CompletionContext{
		CurrentNode:  data,
		CurrentType:  "map",
		PartialToken: "u",
		IsAfterDot:   false,
	}

	completions := engine.GetCompletions("_.u", ctx)

	// Should include "users" field
	found := false
	for _, c := range completions {
		if c.Display == "users" {
			found = true
			t.Logf("Found completion: %s (%s)", c.Display, c.Detail)
		}
	}

	if !found {
		t.Error("Expected to find 'users' in completions")
	}

	t.Logf("Total completions for '_.u': %d", len(completions))
}

// TestCompletionContextAwareness verifies type-aware filtering
func TestCompletionContextAwareness(t *testing.T) {
	provider, err := completion.NewCELProvider()
	if err != nil {
		t.Fatalf("Failed to create CEL provider: %v", err)
	}

	engine := completion.NewEngine(provider)

	// Test with string type - should suggest string functions
	ctx := completion.CompletionContext{
		CurrentNode:  "test string",
		CurrentType:  "string",
		PartialToken: "",
		IsAfterDot:   true,
	}

	completions := engine.GetCompletions("_.name.", ctx)

	// Should include string functions like "contains", "startsWith", etc.
	stringFunctions := 0
	for _, c := range completions {
		if c.Kind == completion.CompletionFunction {
			stringFunctions++
			t.Logf("String function: %s - %s", c.Display, c.Detail)
		}
	}

	if stringFunctions == 0 {
		t.Error("Expected to find string functions for string type")
	}

	t.Logf("Found %d function completions for string type", stringFunctions)
}
