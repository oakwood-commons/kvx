package ui

import (
	"strings"
	"testing"

	"github.com/oakwood-commons/kvx/internal/completion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testFunctions returns a sample set of FunctionMetadata for testing.
func testFunctions() []completion.FunctionMetadata {
	return []completion.FunctionMetadata{
		{Name: "type", Category: "conversion", IsMethod: false, Description: "Returns the type of a value", Signature: "type(value)", Examples: []string{"type(1) => int"}},
		{Name: "int", Category: "conversion", IsMethod: false, Description: "Convert to integer", Signature: "int(value)"},
		{Name: "string", Category: "conversion", IsMethod: false, Description: "Convert to string", Signature: "string(value)"},
		{Name: "filter", Category: "list", IsMethod: true, Description: "Filter list elements", Signature: "list.filter(x, condition)", Examples: []string{"[1,2,3].filter(x, x > 1)"}},
		{Name: "map", Category: "list", IsMethod: true, Description: "Transform list elements", Signature: "list.map(x, expr)"},
		{Name: "sort", Category: "list", IsMethod: true, Description: "Sort list elements", Signature: "list.sort()"},
		{Name: "contains", Category: "string", IsMethod: true, Description: "Check if string contains substring", Signature: "string.contains(substr)"},
		{Name: "has", Category: "map", IsMethod: false, Description: "Check if field exists", Signature: "has(obj.field)"},
		{Name: "math.abs", Category: "math", IsMethod: false, Description: "Absolute value", Signature: "math.abs(number)"},
		{Name: "base64.encode", Category: "encoding", IsMethod: false, Description: "Encode to base64", Signature: "base64.encode(bytes)"},
		{Name: "timestamp", Category: "datetime", IsMethod: false, Description: "Parse timestamp", Signature: "timestamp(value)"},
	}
}

// newTestPalette creates a palette preloaded with test functions via the registry.
func newTestPalette() FunctionPaletteModel {
	p := NewFunctionPaletteModel()
	registry := completion.NewFunctionRegistry()
	registry.LoadFunctions(testFunctions())
	p.LoadFunctions(registry)
	p.Width = 80
	p.Height = 24
	p.NoColor = true
	return p
}

func TestPaletteNewHasNoCategories(t *testing.T) {
	p := NewFunctionPaletteModel()
	assert.False(t, p.Visible)
	assert.Empty(t, p.Categories)
}

func TestPaletteLoadCategories(t *testing.T) {
	p := newTestPalette()
	require.NotEmpty(t, p.Categories)

	// Should include known categories in order.
	assert.Contains(t, p.Categories, "conversion")
	assert.Contains(t, p.Categories, "list")
	assert.Contains(t, p.Categories, "string")
	assert.Contains(t, p.Categories, "map")
	assert.Contains(t, p.Categories, "math")
	assert.Contains(t, p.Categories, "encoding")
	assert.Contains(t, p.Categories, "datetime")

	// conversion should come before string (per categoryOrder).
	convIdx := indexOf(p.Categories, "conversion")
	strIdx := indexOf(p.Categories, "string")
	assert.Less(t, convIdx, strIdx, "conversion should precede string in category order")
}

func TestPaletteToggle(t *testing.T) {
	p := newTestPalette()
	assert.False(t, p.Visible)

	p.Toggle()
	assert.True(t, p.Visible)

	p.Toggle()
	assert.False(t, p.Visible)
}

func TestPaletteClose(t *testing.T) {
	p := newTestPalette()
	p.Toggle()
	assert.True(t, p.Visible)

	p.Close()
	assert.False(t, p.Visible)
	assert.Empty(t, p.SearchQuery)
	assert.Equal(t, 0, p.SelectedIndex)
}

func TestPaletteCategoryNavigation(t *testing.T) {
	p := newTestPalette()
	p.Toggle()

	initialCat := p.SelectedCategory
	p.NextCategory()
	assert.NotEqual(t, initialCat, p.SelectedCategory)
	assert.Equal(t, 0, p.SelectedIndex, "moving categories should reset selection")

	// Go back.
	p.PrevCategory()
	assert.Equal(t, initialCat, p.SelectedCategory)
}

func TestPaletteCategoryWrapAround(t *testing.T) {
	p := newTestPalette()
	p.Toggle()

	// Wrap forward.
	for range p.Categories {
		p.NextCategory()
	}
	assert.Equal(t, 0, p.SelectedCategory, "should wrap around to first category")

	// Wrap backward.
	p.PrevCategory()
	assert.Equal(t, len(p.Categories)-1, p.SelectedCategory, "should wrap to last category")
}

func TestPaletteFunctionNavigation(t *testing.T) {
	p := newTestPalette()
	p.Toggle()

	// Select "conversion" which has 3 functions.
	funcs := p.allFilteredFunctions()
	require.NotEmpty(t, funcs)

	assert.Equal(t, 0, p.SelectedIndex)

	p.MoveDown()
	assert.Equal(t, 1, p.SelectedIndex)

	p.MoveUp()
	assert.Equal(t, 0, p.SelectedIndex)

	// Wrap around up.
	p.MoveUp()
	assert.Equal(t, len(funcs)-1, p.SelectedIndex, "should wrap to bottom")
}

func TestPaletteSearchFilter(t *testing.T) {
	p := newTestPalette()
	p.Toggle()

	// Type "ty" to filter.
	p.HandleSearchKey("t")
	p.HandleSearchKey("y")
	assert.Equal(t, "ty", p.SearchQuery)

	funcs := p.allFilteredFunctions()
	require.Len(t, funcs, 1)
	assert.Equal(t, "type", funcs[0].Name)
}

func TestPaletteSearchCrossCategory(t *testing.T) {
	p := newTestPalette()
	p.Toggle()

	// Search for "a" should find functions across categories.
	p.HandleSearchKey("a")
	funcs := p.allFilteredFunctions()
	require.NotEmpty(t, funcs)

	names := make([]string, len(funcs))
	for i, fn := range funcs {
		names[i] = fn.Name
	}
	// Should find math.abs, base64.encode, has, timestamp, etc.
	assert.True(t, len(funcs) > 1, "search should find across categories")
}

func TestPaletteSearchBackspace(t *testing.T) {
	p := newTestPalette()
	p.Toggle()

	p.HandleSearchKey("t")
	p.HandleSearchKey("y")
	assert.Equal(t, "ty", p.SearchQuery)

	p.HandleSearchKey("backspace")
	assert.Equal(t, "t", p.SearchQuery)

	p.HandleSearchKey("backspace")
	assert.Equal(t, "", p.SearchQuery)

	// Backspace on empty should not panic.
	p.HandleSearchKey("backspace")
	assert.Equal(t, "", p.SearchQuery)
}

func TestPaletteSelectedFunction(t *testing.T) {
	p := newTestPalette()
	p.Toggle()

	fn := p.SelectedFunction()
	require.NotNil(t, fn)
	// First category is "conversion", first function alphabetically.
	assert.NotEmpty(t, fn.Name)
}

func TestPaletteSelectedFunctionEmpty(t *testing.T) {
	p := newTestPalette()
	p.Toggle()

	// Filter to no results.
	p.SearchQuery = "zzzzz"
	fn := p.SelectedFunction()
	assert.Nil(t, fn)
}

func TestInsertTextGlobal(t *testing.T) {
	fn := &completion.FunctionMetadata{
		Name:     "type",
		IsMethod: false,
	}
	assert.Equal(t, "type(", InsertText(fn))
}

func TestInsertTextMethod(t *testing.T) {
	fn := &completion.FunctionMetadata{
		Name:     "filter",
		IsMethod: true,
	}
	assert.Equal(t, ".filter(", InsertText(fn))
}

func TestInsertTextNamespaced(t *testing.T) {
	fn := &completion.FunctionMetadata{
		Name:     "math.abs",
		IsMethod: false,
	}
	assert.Equal(t, "math.abs(", InsertText(fn))
}

func TestInsertTextNil(t *testing.T) {
	assert.Equal(t, "", InsertText(nil))
}

func TestPaletteViewNotVisibleReturnsEmpty(t *testing.T) {
	p := newTestPalette()
	assert.Equal(t, "", p.View())
}

func TestPaletteViewVisible(t *testing.T) {
	p := newTestPalette()
	p.Toggle()

	view := p.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Functions")
	assert.Contains(t, view, "Esc close")
}

func TestPaletteViewShowsCategories(t *testing.T) {
	p := newTestPalette()
	p.Toggle()

	view := p.View()
	// Should show category tabs.
	assert.Contains(t, view, "conversion")
	assert.Contains(t, view, "list")
}

func TestPaletteViewShowsFunctionDetail(t *testing.T) {
	p := newTestPalette()
	p.Toggle()

	view := p.View()
	fn := p.SelectedFunction()
	require.NotNil(t, fn)
	// Should show the selected function's signature or name.
	assert.Contains(t, view, fn.Signature)
}

func TestInsertPaletteFunction(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		text     string
		isMethod bool
		expected string
	}{
		{
			name:     "global function with root",
			current:  "_",
			text:     "type(",
			isMethod: false,
			expected: "type(",
		},
		{
			name:     "global function with expression",
			current:  "_.mykey",
			text:     "type(",
			isMethod: false,
			expected: "type(_.mykey",
		},
		{
			name:     "method on expression",
			current:  "_.items",
			text:     ".filter(",
			isMethod: true,
			expected: "_.items.filter(",
		},
		{
			name:     "method on trailing dot",
			current:  "_.items.",
			text:     ".sort(",
			isMethod: true,
			expected: "_.items.sort(",
		},
		{
			name:     "method replaces partial token",
			current:  "_.items.filt",
			text:     ".filter(",
			isMethod: true,
			expected: "_.items.filter(",
		},
		{
			name:     "method replaces partial token short",
			current:  "_.items.ma",
			text:     ".map(",
			isMethod: true,
			expected: "_.items.map(",
		},
		{
			name:     "method on complete call keeps expression intact",
			current:  "_.items.filter(x, x.active)",
			text:     ".map(",
			isMethod: true,
			expected: "_.items.filter(x, x.active).map(",
		},
		{
			name:     "global function empty input",
			current:  "",
			text:     "int(",
			isMethod: false,
			expected: "int(",
		},
		{
			name:     "namespaced global wraps expression",
			current:  "_.value",
			text:     "math.abs(",
			isMethod: false,
			expected: "math.abs(_.value",
		},
		{
			name:     "namespace trailing dot completes function name",
			current:  "math.",
			text:     "math.abs(",
			isMethod: false,
			expected: "math.abs(",
		},
		{
			name:     "partial namespace token completes function name",
			current:  "math.sq",
			text:     "math.sqrt(",
			isMethod: false,
			expected: "math.sqrt(",
		},
		{
			name:     "base64 namespace trailing dot completes function name",
			current:  "base64.",
			text:     "base64.encode(",
			isMethod: false,
			expected: "base64.encode(",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := InitialModel(map[string]any{"key": "value"})
			m.PathInput.SetValue(tc.current)
			m.insertPaletteFunction(tc.text, tc.isMethod)
			assert.Equal(t, tc.expected, m.PathInput.Value())
		})
	}
}

func TestCategorizeFunctionConversion(t *testing.T) {
	// Verify categorization picks up conversion and datetime.
	p := newTestPalette()
	convFuncs := p.FuncsByCategory["conversion"]
	require.NotEmpty(t, convFuncs)
	names := make([]string, len(convFuncs))
	for i, fn := range convFuncs {
		names[i] = fn.Name
	}
	assert.Contains(t, names, "type")
	assert.Contains(t, names, "int")
	assert.Contains(t, names, "string")
}

func TestCategorizeFunctionDatetime(t *testing.T) {
	p := newTestPalette()
	dtFuncs := p.FuncsByCategory["datetime"]
	require.NotEmpty(t, dtFuncs)
	names := make([]string, len(dtFuncs))
	for i, fn := range dtFuncs {
		names[i] = fn.Name
	}
	assert.Contains(t, names, "timestamp")
}

// indexOf returns the index of s in slice, or -1 if not found.
func indexOf(slice []string, s string) int {
	for i, v := range slice {
		if v == s {
			return i
		}
	}
	return -1
}

func TestPaletteSearchResetIndex(t *testing.T) {
	p := newTestPalette()
	p.Toggle()

	// Navigate down.
	p.MoveDown()
	p.MoveDown()
	assert.Greater(t, p.SelectedIndex, 0)

	// Searching should reset index.
	p.HandleSearchKey("f")
	assert.Equal(t, 0, p.SelectedIndex)
}

// TestPaletteIgnoresNonPrintable ensures non-printable keys don't add to search.
func TestPaletteIgnoresNonPrintable(t *testing.T) {
	p := newTestPalette()
	p.Toggle()

	p.HandleSearchKey("ctrl+a")
	assert.Equal(t, "", p.SearchQuery, "non-printable key should be ignored")
}

// TestPaletteViewWithSearch shows how search affects the display.
func TestPaletteViewWithSearch(t *testing.T) {
	p := newTestPalette()
	p.Toggle()

	p.HandleSearchKey("f")
	view := p.View()

	assert.Contains(t, view, "🔍 f", "should show search indicator")
	assert.Contains(t, view, "filter", "should show matching function")

	// Confirm matched functions.
	funcs := p.allFilteredFunctions()
	for _, fn := range funcs {
		lowerName := strings.ToLower(fn.Name)
		lowerDesc := strings.ToLower(fn.Description)
		assert.True(t,
			strings.Contains(lowerName, "f") || strings.Contains(lowerDesc, "f"),
			"all results should match filter: %s", fn.Name)
	}
}

func TestPaletteMethodsOnlyMode(t *testing.T) {
	p := newTestPalette()
	p.OpenMethodsOnly()
	assert.True(t, p.Visible)
	assert.True(t, p.MethodsOnly)

	// Should only show methods (filter, map, sort, contains are methods in test data)
	funcs := p.filteredFunctions()
	require.NotEmpty(t, funcs)
	for _, fn := range funcs {
		assert.True(t, fn.IsMethod, "expected only methods, got %s (IsMethod=%v)", fn.Name, fn.IsMethod)
	}

	// Verify non-methods are excluded
	for _, fn := range funcs {
		assert.NotEqual(t, "int", fn.Name, "int is not a method")
		assert.NotEqual(t, "type", fn.Name, "type is not a method")
	}

	// Search within methods-only mode
	p.SearchQuery = "filt"
	filtered := p.filteredFunctions()
	require.Len(t, filtered, 1)
	assert.Equal(t, "filter", filtered[0].Name)

	// Close resets methods-only
	p.Close()
	assert.False(t, p.MethodsOnly)
}

func TestPaletteMethodsOnlyView(t *testing.T) {
	p := newTestPalette()
	p.OpenMethodsOnly()
	view := p.View()
	assert.Contains(t, view, "Methods", "should show Methods header")
	// Should show method functions
	assert.Contains(t, view, "filter")
}

func TestPaletteSelectFunction(t *testing.T) {
	p := newTestPalette()
	p.Toggle()

	// Select a function that exists in "list" category
	found := p.SelectFunction("filter")
	assert.True(t, found)
	sel := p.SelectedFunction()
	require.NotNil(t, sel)
	assert.Equal(t, "filter", sel.Name)

	// Verify category switched to "list"
	assert.Equal(t, "list", p.Categories[p.SelectedCategory])

	// Select function in another category
	found = p.SelectFunction("math.abs")
	assert.True(t, found)
	assert.Equal(t, "math", p.Categories[p.SelectedCategory])

	// Non-existent function returns false
	found = p.SelectFunction("bogus")
	assert.False(t, found)
}

func TestPaletteSelectFunctionMethodsOnly(t *testing.T) {
	p := newTestPalette()
	p.OpenMethodsOnly()

	found := p.SelectFunction("filter")
	assert.True(t, found)
	sel := p.SelectedFunction()
	require.NotNil(t, sel)
	assert.Equal(t, "filter", sel.Name)

	// Non-method should not be found in methods-only mode
	found = p.SelectFunction("int")
	assert.False(t, found)
}

func TestFunctionAtCursor(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"_.items.exists()", "exists"},
		{"_.items.exists(x, x > 1)", "exists"},
		{"_.items.map(x, x.name)", "map"},
		{"int(_.value)", "int"},
		{"_.items", ""},
		{"", ""},
		{"_.items.filter(x, x.size() > 0)", "filter"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, functionAtCursor(tt.input))
		})
	}
}

func TestPartialFunctionAtCursor(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"_.items.exist", "exist"},
		{"_.items.ma", "ma"},
		{"_.items.", ""}, // trailing dot, not a partial
		{"_.items", ""},  // no dot-prefix partial
		{"", ""},
		{"_.items.filter(", ""}, // has parens, not a partial
		{"_.items.filt", "filt"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, partialFunctionAtCursor(tt.input))
		})
	}
}
