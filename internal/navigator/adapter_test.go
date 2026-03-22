package navigator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdapter_NodeAtPath(t *testing.T) {
	root := map[string]any{"a": map[string]any{"b": 42}}
	adapter := Adapter{Navigator: defaultNavigator{}}
	result, err := adapter.NodeAtPath(root, "a.b")
	require.NoError(t, err)
	assert.Equal(t, 42, result)
}

func TestSetNavigator_And_Resolve(t *testing.T) {
	original := currentNavigator
	t.Cleanup(func() { currentNavigator = original })

	root := map[string]any{"key": "value"}

	// Default navigator should work
	result, err := Resolve(root, "key")
	require.NoError(t, err)
	assert.Equal(t, "value", result)
}

func TestSetNavigator_Nil(t *testing.T) {
	original := currentNavigator
	t.Cleanup(func() { currentNavigator = original })

	// Setting nil should not change the navigator
	SetNavigator(nil)
	assert.NotNil(t, currentNavigator)
}

func TestSetNavigator_Custom(t *testing.T) {
	original := currentNavigator
	t.Cleanup(func() { currentNavigator = original })

	custom := &mockNavigator{result: "custom"}
	SetNavigator(custom)
	result, err := Resolve(nil, "any")
	require.NoError(t, err)
	assert.Equal(t, "custom", result)
}

func TestDefaultNavigator_Func(t *testing.T) {
	nav := DefaultNavigator()
	assert.NotNil(t, nav)
	root := map[string]any{"x": 1}
	result, err := nav.NodeAtPath(root, "x")
	require.NoError(t, err)
	assert.Equal(t, 1, result)
}

func TestNavigate(t *testing.T) {
	root := map[string]any{"a": []any{1, 2, 3}}
	result, err := Navigate(root, "a")
	require.NoError(t, err)
	assert.Equal(t, []any{1, 2, 3}, result)
}

func TestNavigate_EmptyPath(t *testing.T) {
	root := map[string]any{"a": 1}
	result, err := Navigate(root, "")
	require.NoError(t, err)
	assert.Equal(t, root, result)
}

type mockNavigator struct {
	result any
}

func (m *mockNavigator) NodeAtPath(_ any, _ string) (any, error) {
	return m.result, nil
}
