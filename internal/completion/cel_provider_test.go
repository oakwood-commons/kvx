package completion

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetFunctionExamples(t *testing.T) {
	original := globalFunctionExamples
	t.Cleanup(func() { globalFunctionExamples = original })

	examples := map[string][]string{
		"filter": {"_.items.filter(x, x.active)"},
		"map":    {"_.items.map(x, x.name)"},
	}
	SetFunctionExamples(examples)

	result := GetFunctionExamples()
	assert.Contains(t, result, "filter")
	assert.Contains(t, result, "map")
	assert.Equal(t, []string{"_.items.filter(x, x.active)"}, result["filter"])
}

func TestSetFunctionExamplesWithDescriptions(t *testing.T) {
	original := globalFunctionExamples
	t.Cleanup(func() { globalFunctionExamples = original })

	examples := map[string]FunctionExampleData{
		"filter": {Description: "Filter elements", Examples: []string{"ex1"}},
	}
	SetFunctionExamplesWithDescriptions(examples)

	data := GetFunctionExamplesData()
	assert.Equal(t, "Filter elements", data["filter"].Description)
	assert.Equal(t, []string{"ex1"}, data["filter"].Examples)
}

func TestGetFunctionExamples_Empty(t *testing.T) {
	original := globalFunctionExamples
	t.Cleanup(func() { globalFunctionExamples = original })

	globalFunctionExamples = nil
	result := GetFunctionExamples()
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestGetFunctionExamplesData_Empty(t *testing.T) {
	original := globalFunctionExamples
	t.Cleanup(func() { globalFunctionExamples = original })

	globalFunctionExamples = nil
	result := GetFunctionExamplesData()
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestInferGoType(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{nil, "null"},
		{"hello", "string"},
		{true, "bool"},
		{42, "int"},
		{int64(42), "int"},
		{uint(42), "uint"},
		{uint64(42), "uint"},
		{3.14, "double"},
		{float32(3.14), "double"},
		{[]interface{}{1, 2}, "list"},
		{map[string]interface{}{"a": 1}, "map"},
		{[]int{1, 2}, "list"},
		{map[int]string{1: "a"}, "map"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, inferGoType(tt.input))
		})
	}
}

func TestInferNodeType(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{nil, "null"},
		{map[string]interface{}{"a": 1}, "map"},
		{[]interface{}{1}, "list"},
		{"hello", "string"},
		{true, "bool"},
		{3.14, "double"},
		{42, "int"},
		{int64(42), "int"},
		{uint(42), "uint"},
		{uint64(42), "uint"},
		{struct{}{}, "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, inferNodeType(tt.input))
		})
	}
}

func TestCELProvider_EvaluateType(t *testing.T) {
	provider, err := NewCELProvider()
	require.NoError(t, err)

	ctx := CompletionContext{
		CurrentNode: map[string]interface{}{"name": "test", "items": []interface{}{1, 2}},
		CurrentType: "map",
	}

	assert.Equal(t, "string", provider.EvaluateType("_.name", ctx))
	assert.Equal(t, "list", provider.EvaluateType("_.items", ctx))
	assert.Equal(t, "map", provider.EvaluateType("_", ctx))
	assert.Equal(t, "map", provider.EvaluateType("  _  ", ctx))
	assert.Equal(t, "map", provider.EvaluateType("", ctx))
}

func TestCELProvider_EvaluateType_NilContext(t *testing.T) {
	provider, err := NewCELProvider()
	require.NoError(t, err)

	ctx := CompletionContext{CurrentNode: nil}
	assert.Equal(t, "", provider.EvaluateType("_", ctx))
	assert.Equal(t, "", provider.EvaluateType("_.name", ctx))
}

func TestCELProvider_EvaluateType_BadExpr(t *testing.T) {
	provider, err := NewCELProvider()
	require.NoError(t, err)

	ctx := CompletionContext{
		CurrentNode: map[string]interface{}{"a": 1},
		CurrentType: "map",
	}
	assert.Equal(t, "", provider.EvaluateType("_.nonexistent.deep.path", ctx))
}

func TestCELProvider_IsExpression(t *testing.T) {
	provider, err := NewCELProvider()
	require.NoError(t, err)

	assert.True(t, provider.IsExpression("_.items.filter(x, x > 1)"))
	assert.True(t, provider.IsExpression("_.items[0]"))
	assert.True(t, provider.IsExpression("a == b"))
	assert.False(t, provider.IsExpression("simple.path"))
}

func TestCELProvider_Evaluate(t *testing.T) {
	provider, err := NewCELProvider()
	require.NoError(t, err)

	root := map[string]interface{}{"name": "test", "count": int64(5)}
	result, err := provider.Evaluate("_.name", root)
	require.NoError(t, err)
	assert.Equal(t, "test", result)

	_, err = provider.Evaluate("_.nonexistent.deep", root)
	assert.Error(t, err)
}

func TestNormalizeFuncName(t *testing.T) {
	assert.Equal(t, "filter", normalizeFuncName("filter"))
	assert.Equal(t, "filter", normalizeFuncName("  filter  "))
	assert.Equal(t, "filter", normalizeFuncName("list.filter"))
	assert.Equal(t, "map", normalizeFuncName("list.map"))
}

func TestIsCompatibleWithType(t *testing.T) {
	provider, err := NewCELProvider()
	require.NoError(t, err)

	// type() is universal
	assert.True(t, provider.isCompatibleWithType("type", "map"))
	assert.True(t, provider.isCompatibleWithType("type", "list"))
	assert.True(t, provider.isCompatibleWithType("type", "string"))

	// Map functions
	assert.True(t, provider.isCompatibleWithType("keys", "map"))
	assert.True(t, provider.isCompatibleWithType("values", "map"))
	assert.True(t, provider.isCompatibleWithType("filter", "map"))

	// List functions
	assert.True(t, provider.isCompatibleWithType("size", "list"))
	assert.True(t, provider.isCompatibleWithType("filter", "list"))

	// String functions
	assert.True(t, provider.isCompatibleWithType("contains", "string"))
	assert.True(t, provider.isCompatibleWithType("startsWith", "string"))
}

func TestCELProvider_SplitPathSegments(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"_.name", []string{"_", "name"}},
		{"_.items[0]", []string{"_", "items", "0"}},
		{"_", []string{"_"}},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := splitPathSegments(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCELProvider_SplitPathSegments_Empty(t *testing.T) {
	got := splitPathSegments("")
	assert.Empty(t, got)
}

func TestCELProvider_IsValidCELIdentifier(t *testing.T) {
	assert.True(t, isValidCELIdentifier("name"))
	assert.True(t, isValidCELIdentifier("_field"))
	assert.True(t, isValidCELIdentifier("field123"))
	assert.False(t, isValidCELIdentifier(""))
	assert.False(t, isValidCELIdentifier("123field"))
	assert.False(t, isValidCELIdentifier("field-name"))
}

func TestCELProvider_AppendSegment(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		segment  string
		expected string
	}{
		{"dot notation", "_", "name", "_.name"},
		{"numeric bracket", "_", "0", "_[0]"},
		{"empty segment skipped", "_", "", "_"},
		{"root segment skipped", "", "_", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b strings.Builder
			b.WriteString(tt.initial)
			appendSegment(&b, tt.segment)
			assert.Equal(t, tt.expected, b.String())
		})
	}
}

func TestCELProvider_ListKeys(t *testing.T) {
	data := map[string]any{"b": 2, "a": 1, "c": 3}
	k := listKeys(data)
	assert.Len(t, k, 3)
	// Should be sorted
	assert.Equal(t, "a", k[0])
}

func TestCELProvider_ListKeys_NonMap(t *testing.T) {
	k := listKeys("not a map")
	assert.Empty(t, k)
}

func TestCELProvider_CategorizeFunction(t *testing.T) {
	cat := categorizeFunction("filter", "")
	assert.NotEmpty(t, cat)
}

func TestCELProvider_EnrichWithExamples(t *testing.T) {
	original := globalFunctionExamples
	t.Cleanup(func() { globalFunctionExamples = original })

	SetFunctionExamplesWithDescriptions(map[string]FunctionExampleData{
		"filter": {
			Description: "Filter elements",
			Examples:    []string{"_.items.filter(x, x > 0)"},
		},
	})

	funcs := []FunctionMetadata{{Name: "filter"}}
	enrichWithExamples(funcs)
	assert.Equal(t, "Filter elements", funcs[0].Description)
	assert.Contains(t, funcs[0].Examples, "_.items.filter(x, x > 0)")
}
