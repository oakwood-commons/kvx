package intellisense

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCELProvider(t *testing.T) {
	provider, err := NewCELProvider()
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestNewProvider_DefaultReturnsCEL(t *testing.T) {
	original := customProvider
	t.Cleanup(func() { customProvider = original })

	customProvider = nil
	provider, err := NewProvider()
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestSetProvider_CustomProviderUsed(t *testing.T) {
	original := customProvider
	t.Cleanup(func() { customProvider = original })

	mock := &mockProvider{}
	SetProvider(mock)
	provider, err := NewProvider()
	require.NoError(t, err)
	assert.Equal(t, mock, provider)
}

func TestSetProvider_NilResetsToDefault(t *testing.T) {
	original := customProvider
	t.Cleanup(func() { customProvider = original })

	SetProvider(&mockProvider{})
	SetProvider(nil)
	provider, err := NewProvider()
	require.NoError(t, err)
	assert.NotNil(t, provider)
	_, isMock := provider.(*mockProvider)
	assert.False(t, isMock)
}

func TestDefaultSearchOptions(t *testing.T) {
	opts := DefaultSearchOptions()
	assert.Equal(t, 10, opts.MaxResults)
	assert.False(t, opts.CaseSensitive)
	assert.False(t, opts.FuzzyMatch)
	assert.True(t, opts.ShowDescriptions)
	assert.True(t, opts.TypeAwareFiltering)
}

func TestCELProvider_DiscoverFunctions(t *testing.T) {
	provider, err := NewCELProvider()
	require.NoError(t, err)
	funcs := provider.DiscoverFunctions()
	assert.NotEmpty(t, funcs)
}

func TestCELProvider_FilterCompletions(t *testing.T) {
	provider, err := NewCELProvider()
	require.NoError(t, err)

	data := map[string]any{"name": "test", "items": []any{1, 2}}
	ctx := CompletionContext{
		CurrentNode: data,
		CurrentType: "map",
		IsAfterDot:  true,
	}

	completions := provider.FilterCompletions("_.", ctx)
	assert.NotEmpty(t, completions)
}

func TestCELProvider_IsExpression(t *testing.T) {
	provider, err := NewCELProvider()
	require.NoError(t, err)

	assert.True(t, provider.IsExpression("_.items.filter(x, x > 1)"))
	assert.True(t, provider.IsExpression("_.items[0]"))
	assert.False(t, provider.IsExpression("_.name"))
	assert.False(t, provider.IsExpression("simple"))
}

func TestCELProvider_Evaluate(t *testing.T) {
	provider, err := NewCELProvider()
	require.NoError(t, err)

	data := map[string]any{"name": "test"}
	result, err := provider.Evaluate("_.name", data)
	require.NoError(t, err)
	assert.Equal(t, "test", result)
}

func TestCELProvider_EvaluateType(t *testing.T) {
	provider, err := NewCELProvider()
	require.NoError(t, err)

	ctx := CompletionContext{
		CurrentNode: map[string]any{"name": "test"},
		CurrentType: "map",
	}
	typeStr := provider.EvaluateType("_.name", ctx)
	assert.NotEmpty(t, typeStr)
}

type mockProvider struct{}

func (m *mockProvider) DiscoverFunctions() []FunctionMetadata                    { return nil }
func (m *mockProvider) FilterCompletions(string, CompletionContext) []Completion { return nil }
func (m *mockProvider) EvaluateType(string, CompletionContext) string            { return "mock" }
func (m *mockProvider) Evaluate(string, interface{}) (interface{}, error)        { return nil, nil }
func (m *mockProvider) IsExpression(string) bool                                 { return false }
