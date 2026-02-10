package cel

import (
	"strings"
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

// TestNewStandardCELEnv_WithOpts tests that newStandardCELEnv accepts and applies additional options.
// This verifies that the helper function supports extensibility.
func TestNewStandardCELEnv_WithOpts(t *testing.T) {
	// Create a custom function option
	customFunc := cel.Function("testCustomFunc",
		cel.Overload("testCustomFunc_string",
			[]*cel.Type{cel.StringType},
			cel.StringType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				s, ok := args[0].(types.String)
				if !ok {
					return types.NewErr("testCustomFunc requires string argument")
				}
				return s
			}),
		),
	)

	// Call newStandardCELEnv with the custom function
	env, err := newStandardCELEnv(customFunc)
	if err != nil {
		t.Fatalf("newStandardCELEnv with opts failed: %v", err)
	}

	// Verify the environment has the standard extensions AND the custom function
	// Check that standard functions are present (strings extension)
	foundStrings := false
	for _, fn := range env.Functions() {
		if fn.Name() == "contains" { // strings extension function
			foundStrings = true
		}
		if fn.Name() == "testCustomFunc" {
			// Custom function should be present
			return // Success - both standard and custom are present
		}
	}

	if !foundStrings {
		t.Error("newStandardCELEnv should include standard extensions (strings)")
	}
	t.Error("newStandardCELEnv with opts should include custom function")
}

// TestNewStandardCELEnv_WithoutOpts tests that newStandardCELEnv works without additional options.
func TestNewStandardCELEnv_WithoutOpts(t *testing.T) {
	env, err := newStandardCELEnv()
	if err != nil {
		t.Fatalf("newStandardCELEnv without opts failed: %v", err)
	}

	// Verify standard extensions are present
	foundStrings := false
	for _, fn := range env.Functions() {
		if fn.Name() == "contains" { // strings extension function
			foundStrings = true
			break
		}
	}

	if !foundStrings {
		t.Error("newStandardCELEnv should include standard extensions (strings)")
	}
}

// TestDiscoverFunctionsFromEnv_WithCustomFunctions tests that DiscoverFunctionsFromEnv
// discovers functions from a custom CEL environment including custom functions.
func TestDiscoverFunctionsFromEnv_WithCustomFunctions(t *testing.T) {
	// Create a custom CEL environment with a custom function
	env, err := cel.NewEnv(
		cel.Variable("_", cel.DynType),
		cel.Function("customFunc",
			cel.Overload("customFunc_string",
				[]*cel.Type{cel.StringType},
				cel.StringType,
				cel.FunctionBinding(func(args ...ref.Val) ref.Val {
					s, ok := args[0].(types.String)
					if !ok {
						return types.NewErr("customFunc requires string argument")
					}
					return types.String(strings.ToUpper(string(s)))
				}),
			),
		),
	)
	if err != nil {
		t.Fatalf("failed to create custom CEL environment: %v", err)
	}

	// Discover functions
	suggestions := DiscoverFunctionsFromEnv(env, nil)

	// Should include the custom function
	found := false
	for _, s := range suggestions {
		if strings.Contains(s, "customFunc()") {
			found = true
			// Should have usage information
			if !strings.Contains(s, " - ") {
				t.Errorf("customFunc suggestion should include usage info, got: %q", s)
			}
		}
	}
	if !found {
		t.Errorf("customFunc should be discovered, got suggestions: %v", suggestions)
	}
}

// TestDiscoverFunctionsFromEnv_WithExampleHints tests that example hints are included in suggestions.
func TestDiscoverFunctionsFromEnv_WithExampleHints(t *testing.T) {
	// Create a minimal CEL environment
	env, err := cel.NewEnv(
		cel.Variable("_", cel.DynType),
		cel.Function("testFunc",
			cel.Overload("testFunc_string",
				[]*cel.Type{cel.StringType},
				cel.StringType,
				cel.FunctionBinding(func(args ...ref.Val) ref.Val {
					return args[0]
				}),
			),
		),
	)
	if err != nil {
		t.Fatalf("failed to create CEL environment: %v", err)
	}

	exampleHints := map[string]string{
		"testFunc": "e.g. testFunc(arg)",
	}

	suggestions := DiscoverFunctionsFromEnv(env, exampleHints)

	// Find testFunc suggestion
	found := false
	for _, s := range suggestions {
		if strings.Contains(s, "testFunc()") {
			found = true
			if !strings.Contains(s, "e.g. testFunc(arg)") {
				t.Errorf("testFunc suggestion should include example hint, got: %q", s)
			}
		}
	}
	if !found {
		t.Errorf("testFunc should be discovered, got suggestions: %v", suggestions)
	}
}

// TestDiscoverFunctionsFromEnv_FiltersOperators tests that internal operators are filtered out.
func TestDiscoverFunctionsFromEnv_FiltersOperators(t *testing.T) {
	// Create a standard CEL environment (includes operators)
	env, err := cel.NewEnv(
		cel.Variable("_", cel.DynType),
	)
	if err != nil {
		t.Fatalf("failed to create CEL environment: %v", err)
	}

	suggestions := DiscoverFunctionsFromEnv(env, nil)

	// Should not contain internal operators (those starting with _ and ending with _)
	for _, s := range suggestions {
		funcName := s
		if idx := strings.Index(s, "()"); idx >= 0 {
			funcName = s[:idx]
		}
		if strings.HasPrefix(funcName, "_") && strings.HasSuffix(funcName, "_") {
			t.Errorf("operator-style function should be filtered: %q", s)
		}
		if strings.HasPrefix(funcName, "@") {
			t.Errorf("macro-style operator should be filtered: %q", s)
		}
	}
}
