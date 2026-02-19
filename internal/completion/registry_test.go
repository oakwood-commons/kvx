package completion

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryDeduplication(t *testing.T) {
	registry := NewFunctionRegistry()

	// Load functions with duplicates (simulating overloads)
	funcs := []FunctionMetadata{
		{Name: "int", Signature: "int(string) -> int", Description: "Convert string to int", Category: "conversion"},
		{Name: "int", Signature: "int(double) -> int", Description: "Convert double to int", Category: "conversion"},
		{Name: "int", Signature: "int(timestamp) -> int", Description: "Convert to int with examples", Category: "conversion", Examples: []string{"int('42')"}},
		{Name: "filter", Signature: "list.filter(x, cond)", Description: "Filter list elements", Category: "list", IsMethod: true},
	}

	registry.LoadFunctions(funcs)

	// Should have only 2 unique functions
	assert.Equal(t, 2, registry.Size())

	// int should have the version with examples (best metadata)
	intFn := registry.GetFunction("int")
	require.NotNil(t, intFn)
	assert.Equal(t, "int", intFn.Name)
	assert.Len(t, intFn.Examples, 1)
	assert.Equal(t, "int('42')", intFn.Examples[0])

	// filter should be present
	filterFn := registry.GetFunction("filter")
	require.NotNil(t, filterFn)
	assert.True(t, filterFn.IsMethod)
}

func TestRegistryCategoryLookup(t *testing.T) {
	registry := NewFunctionRegistry()
	funcs := []FunctionMetadata{
		{Name: "int", Category: "conversion"},
		{Name: "double", Category: "conversion"},
		{Name: "filter", Category: "list", IsMethod: true},
		{Name: "map", Category: "list", IsMethod: true},
		{Name: "size", Category: "list", IsMethod: true},
		{Name: "abs", Category: "math"},
	}
	registry.LoadFunctions(funcs)

	// Get by category
	convFuncs := registry.GetByCategory("conversion")
	assert.Len(t, convFuncs, 2)

	listFuncs := registry.GetByCategory("list")
	assert.Len(t, listFuncs, 3)

	// Category count
	assert.Equal(t, 2, registry.CategoryCount("conversion"))
	assert.Equal(t, 3, registry.CategoryCount("list"))
	assert.Equal(t, 1, registry.CategoryCount("math"))
	assert.Equal(t, 0, registry.CategoryCount("nonexistent"))

	// Categories in order
	cats := registry.GetCategories()
	assert.Contains(t, cats, "conversion")
	assert.Contains(t, cats, "list")
	assert.Contains(t, cats, "math")
}

func TestRegistrySearch(t *testing.T) {
	registry := NewFunctionRegistry()
	funcs := []FunctionMetadata{
		{Name: "filter", Description: "Filter list elements by condition", Category: "list"},
		{Name: "map", Description: "Transform each element", Category: "list"},
		{Name: "contains", Description: "Check if string contains substring", Category: "string"},
		{Name: "startsWith", Description: "Check if string starts with prefix", Category: "string"},
	}
	registry.LoadFunctions(funcs)

	// Search by name
	results := registry.Search("filt")
	assert.Len(t, results, 1)
	assert.Equal(t, "filter", results[0].Name)

	// Search by description
	results = registry.Search("string")
	assert.Len(t, results, 2) // contains and startsWith both mention "string"

	// Case insensitive
	results = registry.Search("FILTER")
	assert.Len(t, results, 1)

	// Empty query returns all
	results = registry.Search("")
	assert.Len(t, results, 4)
}

func TestRegistryMethodsAndGlobals(t *testing.T) {
	registry := NewFunctionRegistry()
	funcs := []FunctionMetadata{
		{Name: "int", Category: "conversion", IsMethod: false},
		{Name: "type", Category: "conversion", IsMethod: false},
		{Name: "filter", Category: "list", IsMethod: true},
		{Name: "map", Category: "list", IsMethod: true},
	}
	registry.LoadFunctions(funcs)

	methods := registry.GetMethods()
	assert.Len(t, methods, 2)
	for _, m := range methods {
		assert.True(t, m.IsMethod)
	}

	globals := registry.GetGlobals()
	assert.Len(t, globals, 2)
	for _, g := range globals {
		assert.False(t, g.IsMethod)
	}
}

func TestRegistryGetAll(t *testing.T) {
	registry := NewFunctionRegistry()
	funcs := []FunctionMetadata{
		{Name: "zebra", Category: "general"},
		{Name: "apple", Category: "general"},
		{Name: "mango", Category: "general"},
	}
	registry.LoadFunctions(funcs)

	all := registry.GetAll()
	require.Len(t, all, 3)
	// Should be sorted alphabetically
	assert.Equal(t, "apple", all[0].Name)
	assert.Equal(t, "mango", all[1].Name)
	assert.Equal(t, "zebra", all[2].Name)
}

func TestRegistryEmptyCategory(t *testing.T) {
	registry := NewFunctionRegistry()
	funcs := []FunctionMetadata{
		{Name: "test", Category: ""}, // Empty category should become "general"
	}
	registry.LoadFunctions(funcs)

	cats := registry.GetCategories()
	assert.Contains(t, cats, "general")

	generalFuncs := registry.GetByCategory("general")
	assert.Len(t, generalFuncs, 1)
}

func TestRegistrySupplementFromSuggestions(t *testing.T) {
	registry := NewFunctionRegistry()
	// Pre-load one function with rich metadata.
	registry.LoadFunctions([]FunctionMetadata{
		{Name: "filter", Category: "list", IsMethod: true, Description: "Filter list elements", Signature: "list.filter(x, cond)"},
	})

	suggestions := []string{
		"customFunc(arg) - My custom function",    // new: should be added
		"filter(x, condition) - Filter elements",  // existing with description: should be skipped
		"anotherFunc() - Another custom function", // new: should be added
		"",         // blank: should be skipped
		"noDesc()", // new, no description: should be added
	}
	registry.SupplementFromSuggestions(suggestions)

	// Custom functions should now be present.
	cf := registry.GetFunction("customFunc")
	require.NotNil(t, cf)
	assert.Equal(t, "customFunc(arg)", cf.Signature)
	assert.Equal(t, "My custom function", cf.Description)

	af := registry.GetFunction("anotherFunc")
	require.NotNil(t, af)

	nd := registry.GetFunction("noDesc")
	require.NotNil(t, nd)

	// The existing rich entry should be unchanged.
	ff := registry.GetFunction("filter")
	require.NotNil(t, ff)
	assert.Equal(t, "Filter list elements", ff.Description, "existing rich entry should not be overwritten")
	assert.Equal(t, "list.filter(x, cond)", ff.Signature)
}

func TestRegistrySupplementDoesNotOverwriteRichEntries(t *testing.T) {
	registry := NewFunctionRegistry()
	registry.LoadFunctions([]FunctionMetadata{
		{Name: "map", Category: "list", IsMethod: true, Description: "Transform list elements", Examples: []string{"[1,2].map(x, x*2)"}},
	})

	// Supplement with a weaker suggestion for the same function.
	registry.SupplementFromSuggestions([]string{"map(x, expr) - weak description"})

	fn := registry.GetFunction("map")
	require.NotNil(t, fn)
	assert.Equal(t, "Transform list elements", fn.Description, "rich entry should win")
	assert.Len(t, fn.Examples, 1, "examples should be preserved")
}
