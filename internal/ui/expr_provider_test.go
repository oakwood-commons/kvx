package ui

import (
	"strings"
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

// TestNewCELExpressionProvider_NilEnv tests that nil environment returns default provider.
func TestNewCELExpressionProvider_NilEnv(t *testing.T) {
	provider := NewCELExpressionProvider(nil, nil)
	if provider == nil {
		t.Fatal("NewCELExpressionProvider(nil, nil) should return default provider, not nil")
	}

	// Should be able to evaluate standard expressions
	result, err := provider.Evaluate("1 + 1", map[string]interface{}{})
	if err != nil {
		t.Errorf("Default provider should evaluate expressions, got error: %v", err)
	}
	if result != int64(2) {
		t.Errorf("Evaluation result = %v, expected 2", result)
	}
}

// TestNewCELExpressionProvider_WithCustomFunctions tests that custom functions are discovered.
func TestNewCELExpressionProvider_WithCustomFunctions(t *testing.T) {
	// Create a custom CEL environment with a custom function
	env, err := cel.NewEnv(
		cel.Variable("_", cel.DynType),
		cel.Function("myCustomFunc",
			cel.Overload("myCustomFunc_string",
				[]*cel.Type{cel.StringType},
				cel.StringType,
				cel.FunctionBinding(func(args ...ref.Val) ref.Val {
					s, ok := args[0].(types.String)
					if !ok {
						return types.NewErr("myCustomFunc requires string argument")
					}
					return types.String(strings.ToUpper(string(s)))
				}),
			),
		),
	)
	if err != nil {
		t.Fatalf("failed to create custom CEL environment: %v", err)
	}

	exampleHints := map[string]string{
		"myCustomFunc": "e.g. myCustomFunc(\"test\")",
	}

	provider := NewCELExpressionProvider(env, exampleHints)

	// Test that custom function is discovered
	suggestions := provider.DiscoverSuggestions()
	found := false
	for _, s := range suggestions {
		if strings.Contains(s, "myCustomFunc()") {
			found = true
			if !strings.Contains(s, "e.g. myCustomFunc(\"test\")") {
				t.Errorf("myCustomFunc suggestion should include example hint, got: %q", s)
			}
		}
	}
	if !found {
		t.Errorf("myCustomFunc should be discovered, got suggestions: %v", suggestions)
	}

	// Test that custom function can be evaluated (though this requires the function to be properly registered)
	// Note: This test verifies discovery, not evaluation, as evaluation requires proper CEL program compilation
}

// TestNewCELExpressionProvider_EvaluatesExpressions tests that the provider can evaluate expressions.
func TestNewCELExpressionProvider_EvaluatesExpressions(t *testing.T) {
	// Create a standard CEL environment
	env, err := cel.NewEnv(
		cel.Variable("_", cel.DynType),
	)
	if err != nil {
		t.Fatalf("failed to create CEL environment: %v", err)
	}

	provider := NewCELExpressionProvider(env, nil)

	// Test evaluation with simple expression
	result, err := provider.Evaluate("1 + 2", map[string]interface{}{})
	if err != nil {
		t.Errorf("Should evaluate simple expressions, got error: %v", err)
	}
	if result != int64(3) {
		t.Errorf("Evaluation result = %v, expected 3", result)
	}

	// Test evaluation with data
	data := map[string]interface{}{
		"value": 42,
	}
	result, err = provider.Evaluate("_.value", data)
	if err != nil {
		t.Errorf("Should evaluate expressions with data, got error: %v", err)
	}
	if result != int64(42) {
		t.Errorf("Evaluation result = %v, expected 42", result)
	}
}

// TestNewCELExpressionProvider_IsExpression tests that IsExpression works correctly.
func TestNewCELExpressionProvider_IsExpression(t *testing.T) {
	env, err := cel.NewEnv(
		cel.Variable("_", cel.DynType),
	)
	if err != nil {
		t.Fatalf("failed to create CEL environment: %v", err)
	}

	provider := NewCELExpressionProvider(env, nil)

	tests := []struct {
		expr     string
		expected bool
	}{
		{"_.items[0]", true},   // Has brackets
		{"items[0]", true},     // Has brackets
		{"items", false},       // Simple identifier
		{"filter(x, x)", true}, // Has filter function (in the function list)
		{"_.field", false},     // Doesn't match any pattern
		{"1 == 1", true},       // Has == operator
		{"simple text", false},
	}

	for _, tt := range tests {
		result := provider.IsExpression(tt.expr)
		if result != tt.expected {
			t.Errorf("IsExpression(%q) = %v, expected %v", tt.expr, result, tt.expected)
		}
	}
}
