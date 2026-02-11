package cel

import (
	"strings"
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

// Ensure DiscoverCELFunctions returns a rich set when extension libs are loaded.
func TestDiscoverCELFunctionsIncludesExtensions(t *testing.T) {
	funcs, err := DiscoverCELFunctions()
	if err != nil {
		t.Fatalf("DiscoverCELFunctions error: %v", err)
	}
	if len(funcs) < 10 {
		t.Fatalf("expected at least 10 CEL functions, got %d: %v", len(funcs), funcs)
	}
}

func TestNewEvaluator_CreatesValidEnvironment(t *testing.T) {
	eval, err := NewEvaluator()
	if err != nil {
		t.Fatalf("NewEvaluator failed: %v", err)
	}
	if eval == nil {
		t.Fatal("NewEvaluator returned nil")
	}
	if eval.GetEnvironment() == nil {
		t.Fatal("GetEnvironment returned nil")
	}
}

func TestEvaluate_SimpleExpressions(t *testing.T) {
	eval, err := NewEvaluator()
	if err != nil {
		t.Fatalf("NewEvaluator failed: %v", err)
	}

	tests := []struct {
		name     string
		expr     string
		data     interface{}
		expected interface{}
	}{
		{"access field", "_.name", map[string]interface{}{"name": "test"}, "test"},
		{"access number", "_.count", map[string]interface{}{"count": 42}, int64(42)},
		{"array index", "_[0]", []interface{}{"first", "second"}, "first"},
		{"boolean", "_.active", map[string]interface{}{"active": true}, true},
		{"nested field", "_.user.email", map[string]interface{}{"user": map[string]interface{}{"email": "test@example.com"}}, "test@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eval.Evaluate(tt.expr, tt.data)
			if err != nil {
				t.Fatalf("Evaluate failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvaluate_CELOperators(t *testing.T) {
	eval, err := NewEvaluator()
	if err != nil {
		t.Fatalf("NewEvaluator failed: %v", err)
	}

	tests := []struct {
		name     string
		expr     string
		data     interface{}
		expected interface{}
	}{
		{"equality", "_.x == 10", map[string]interface{}{"x": 10}, true},
		{"inequality", "_.x != 5", map[string]interface{}{"x": 10}, true},
		{"greater than", "_.x > 5", map[string]interface{}{"x": 10}, true},
		{"less than", "_.x < 20", map[string]interface{}{"x": 10}, true},
		{"and operator", "_.x > 5 && _.x < 20", map[string]interface{}{"x": 10}, true},
		{"or operator", "_.x < 5 || _.x > 20", map[string]interface{}{"x": 10}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eval.Evaluate(tt.expr, tt.data)
			if err != nil {
				t.Fatalf("Evaluate failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvaluate_FilterFunction(t *testing.T) {
	eval, err := NewEvaluator()
	if err != nil {
		t.Fatalf("NewEvaluator failed: %v", err)
	}

	data := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"name": "item1", "available": true},
			map[string]interface{}{"name": "item2", "available": false},
			map[string]interface{}{"name": "item3", "available": true},
		},
	}

	result, err := eval.Evaluate("_.items.filter(x, x.available)", data)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	resultSlice, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected slice, got %T", result)
	}
	if len(resultSlice) != 2 {
		t.Errorf("expected 2 items, got %d", len(resultSlice))
	}
}

func TestEvaluate_MapFunction(t *testing.T) {
	eval, err := NewEvaluator()
	if err != nil {
		t.Fatalf("NewEvaluator failed: %v", err)
	}

	data := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"name": "item1", "price": 10},
			map[string]interface{}{"name": "item2", "price": 20},
		},
	}

	result, err := eval.Evaluate("_.items.map(x, x.price)", data)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	resultSlice, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected slice, got %T", result)
	}
	if len(resultSlice) != 2 {
		t.Errorf("expected 2 items, got %d", len(resultSlice))
	}
}

func TestToGo_PrimitiveTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    ref.Val
		expected interface{}
	}{
		{"bool true", types.Bool(true), true},
		{"bool false", types.Bool(false), false},
		{"int", types.Int(42), int64(42)},
		{"uint", types.Uint(100), uint64(100)},
		{"double", types.Double(3.14), float64(3.14)},
		{"string", types.String("hello"), "hello"},
		{"nil", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToGo(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v (%T), got %v (%T)", tt.expected, tt.expected, result, result)
			}
		})
	}
}

func TestToGo_BytesType(t *testing.T) {
	input := types.Bytes([]byte("data"))
	result := ToGo(input)
	resultBytes, ok := result.([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", result)
	}
	expected := []byte("data")
	if string(resultBytes) != string(expected) {
		t.Errorf("expected %q, got %q", expected, resultBytes)
	}
}

func TestIsCELExpression_DetectsBrackets(t *testing.T) {
	tests := []struct {
		expr     string
		expected bool
	}{
		{"items[0]", true},
		{"items[0].name", true},
		{"[0]", true},
		{"simple.path", false},
		{"name", false},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result := IsCELExpression(tt.expr)
			if result != tt.expected {
				t.Errorf("IsCELExpression(%q) = %v, want %v", tt.expr, result, tt.expected)
			}
		})
	}
}

func TestIsCELExpression_DetectsFunctions(t *testing.T) {
	tests := []struct {
		expr     string
		expected bool
	}{
		{"items.filter(x, x.active)", true},
		{"items.map(x, x.name)", true},
		{"items.all(x, x.valid)", true},
		{"items.exists(x, x.id == 1)", true},
		{"items.exists_one(x, x.primary)", true},
		{"simple.path", false},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result := IsCELExpression(tt.expr)
			if result != tt.expected {
				t.Errorf("IsCELExpression(%q) = %v, want %v", tt.expr, result, tt.expected)
			}
		})
	}
}

func TestIsCELExpression_DetectsOperators(t *testing.T) {
	tests := []struct {
		expr     string
		expected bool
	}{
		{"x == 10", true},
		{"x != 5", true},
		{"x >= 10", true},
		{"x <= 10", true},
		{"x > 5", true},
		{"x < 20", true},
		{"a && b", true},
		{"a || b", true},
		{"!valid", true},
		{"simple.path", false},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result := IsCELExpression(tt.expr)
			if result != tt.expected {
				t.Errorf("IsCELExpression(%q) = %v, want %v", tt.expr, result, tt.expected)
			}
		})
	}
}

func TestParseCEL_SimplePaths(t *testing.T) {
	tests := []struct {
		expr     string
		expected []string
	}{
		{"a.b.c", []string{"a", "b", "c"}},
		{"items[0]", []string{"items", "0"}},
		{"items[0].name", []string{"items", "0", "name"}},
		{"a", []string{"a"}},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := ParseCEL(tt.expr)
			if err != nil {
				t.Fatalf("ParseCEL failed: %v", err)
			}
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d steps, got %d: %v", len(tt.expected), len(result), result)
				return
			}
			for i, exp := range tt.expected {
				if result[i] != exp {
					t.Errorf("step %d: expected %q, got %q", i, exp, result[i])
				}
			}
		})
	}
}

func TestParseCEL_InvalidExpression(t *testing.T) {
	_, err := ParseCEL("")
	if err == nil {
		t.Error("expected error for empty expression")
	}
}

func TestExtractPathAndIndex_ValidSyntax(t *testing.T) {
	tests := []struct {
		expr         string
		expectedPath string
		expectedIdx  string
	}{
		{"items[0]", "items", "0"},
		{"regions.asia.countries[5]", "regions.asia.countries", "5"},
		{"data[key]", "data", "key"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			path, idx, err := ExtractPathAndIndex(tt.expr)
			if err != nil {
				t.Fatalf("ExtractPathAndIndex failed: %v", err)
			}
			if path != tt.expectedPath {
				t.Errorf("path: expected %q, got %q", tt.expectedPath, path)
			}
			if idx != tt.expectedIdx {
				t.Errorf("index: expected %q, got %q", tt.expectedIdx, idx)
			}
		})
	}
}

func TestExtractPathAndIndex_InvalidSyntax(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"no brackets", "items"},
		{"empty path", "[0]"},
		{"empty index", "items[]"},
		{"mismatched", "items[0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ExtractPathAndIndex(tt.expr)
			if err == nil {
				t.Errorf("expected error for %q", tt.expr)
			}
		})
	}
}

func TestGetAvailableFunctions_ReturnsDynamicFunctions(t *testing.T) {
	funcs := GetAvailableFunctions()
	if len(funcs) == 0 {
		t.Fatal("expected discovered functions from CEL environment")
	}
	// Verify at least one common function is discovered
	found := false
	for _, fn := range funcs {
		if strings.Contains(fn, "filter") || strings.Contains(fn, "map") || strings.Contains(fn, "size") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected common functions like filter, map, or size")
	}
}

func TestDiscoverCELFunctionDocs_ReturnsFormattedSuggestions(t *testing.T) {
	docs, err := DiscoverCELFunctionDocs()
	if err != nil {
		t.Fatalf("DiscoverCELFunctionDocs failed: %v", err)
	}
	if len(docs) < 10 {
		t.Errorf("expected at least 10 function docs, got %d", len(docs))
	}
	// Verify format: should contain function names with "() - " notation
	for _, doc := range docs {
		if strings.Contains(doc, "() - ") {
			return // At least one valid function found
		}
	}
	t.Error("expected function docs to contain name() - description notation")
}

func TestDiscoverFunctionsFromEnv_WithCustomEnvironment(t *testing.T) {
	env, err := cel.NewEnv(
		cel.Variable("_", cel.DynType),
		cel.Function("customFunc",
			cel.Overload("custom_func_overload",
				[]*cel.Type{cel.StringType},
				cel.StringType,
			),
		),
	)
	if err != nil {
		t.Fatalf("failed to create custom env: %v", err)
	}

	hints := map[string]string{
		"customFunc": "e.g. customFunc(\"test\")",
	}

	funcs := DiscoverFunctionsFromEnv(env, hints)
	found := false
	for _, fn := range funcs {
		if len(fn) >= 10 && fn[0:10] == "customFunc" {
			found = true
			if !strings.Contains(fn, "e.g.") {
				t.Error("expected hint to be included")
			}
			break
		}
	}
	if !found {
		t.Error("expected customFunc to be discovered")
	}
}

func TestEvaluateExpressionWithEnv_ErrorHandling(t *testing.T) {
	env, err := newStandardCELEnv()
	if err != nil {
		t.Fatalf("newStandardCELEnv failed: %v", err)
	}

	tests := []struct {
		name string
		expr string
		data interface{}
	}{
		{"invalid syntax", "_.name[", map[string]interface{}{}},
		{"undefined field", "_.nonexistent.deep.field", map[string]interface{}{}},
		{"type error", "_.count + \"string\"", map[string]interface{}{"count": 10}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EvaluateExpressionWithEnv(env, tt.expr, tt.data)
			if err == nil {
				t.Error("expected error for invalid expression")
			}
		})
	}
}

func TestContains_BasicFunctionality(t *testing.T) {
	if !contains("hello world", "world") {
		t.Error("expected contains to find substring")
	}
	if contains("hello", "xyz") {
		t.Error("expected contains to return false for missing substring")
	}
	if contains("", "x") {
		t.Error("expected contains to return false for empty string")
	}
	if contains("test", "") {
		t.Error("expected contains to return false for empty substring")
	}
}

func TestIsOperator_FiltersInternalNames(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"@in", true},
		{"_+_", true},
		{"_-_", true},
		{"_==_", true},
		{"@index", true},
		{"_internal_", true},
		{"filter", false},
		{"map", false},
		{"size", false},
		{"customFunc", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isOperator(tt.name)
			if result != tt.expected {
				t.Errorf("isOperator(%q) = %v, want %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestGetCommonPatterns_ReturnsExamples(t *testing.T) {
	patterns := GetCommonPatterns()
	if len(patterns) == 0 {
		t.Fatal("expected common patterns")
	}
	// Verify at least one pattern contains filter
	found := false
	for _, p := range patterns {
		if strings.Contains(p, "filter") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected filter pattern in common patterns")
	}
}
