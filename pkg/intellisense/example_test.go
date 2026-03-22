package intellisense

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShowHelp_NoPanic(t *testing.T) {
	provider, err := NewCELProvider()
	require.NoError(t, err)
	// Just verify it doesn't panic
	showHelp(provider)
}

func TestPrettyPrint_NoPanic(t *testing.T) {
	prettyPrint(map[string]any{"key": "value"})
	prettyPrint("scalar")
	prettyPrint(42)
	prettyPrint(nil)
	prettyPrint([]any{1, 2, 3})
}

func TestShowFunctions_NoPanic(t *testing.T) {
	provider, err := NewCELProvider()
	require.NoError(t, err)
	showFunctions(provider)
}

func TestShowCompletions_NoPanic(t *testing.T) {
	provider, err := NewCELProvider()
	require.NoError(t, err)
	data := map[string]any{"name": "test", "items": []any{1, 2}}
	showCompletions(provider, "_.", data)
}

func TestShowCompletions_Empty(t *testing.T) {
	provider, err := NewCELProvider()
	require.NoError(t, err)
	// An empty input should produce no completions
	showCompletions(provider, "zzzzzznotreal.", nil)
}
