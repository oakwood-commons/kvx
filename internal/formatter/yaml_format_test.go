package formatter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatYAML_SimpleMap(t *testing.T) {
	data := map[string]any{"name": "test", "count": 42}
	result, err := FormatYAML(data, YAMLFormatOptions{Indent: 2})
	require.NoError(t, err)
	assert.Contains(t, result, "name: test")
	assert.Contains(t, result, "count: 42")
}

func TestFormatYAML_DefaultIndent(t *testing.T) {
	data := map[string]any{"a": map[string]any{"b": "c"}}
	result, err := FormatYAML(data, YAMLFormatOptions{})
	require.NoError(t, err)
	assert.Contains(t, result, "a:")
	assert.Contains(t, result, "b: c")
}

func TestFormatYAML_LiteralBlockStrings(t *testing.T) {
	data := map[string]any{"text": "line1\nline2\nline3"}
	result, err := FormatYAML(data, YAMLFormatOptions{LiteralBlockStrings: true})
	require.NoError(t, err)
	assert.Contains(t, result, "|")
}

func TestFormatYAML_NoLiteralForSingleLine(t *testing.T) {
	data := map[string]any{"text": "no newlines here"}
	result, err := FormatYAML(data, YAMLFormatOptions{LiteralBlockStrings: true})
	require.NoError(t, err)
	assert.NotContains(t, result, "|")
	assert.Contains(t, result, "no newlines here")
}

func TestFormatYAML_ExpandEscapedNewlines(t *testing.T) {
	data := map[string]any{"text": "line1\\nline2"}
	result, err := FormatYAML(data, YAMLFormatOptions{ExpandEscapedNewlines: true})
	require.NoError(t, err)
	assert.NotContains(t, result, "\\n")
}

func TestFormatYAML_BothOptions(t *testing.T) {
	data := map[string]any{"text": "line1\\nline2"}
	result, err := FormatYAML(data, YAMLFormatOptions{
		ExpandEscapedNewlines: true,
		LiteralBlockStrings:   true,
	})
	require.NoError(t, err)
	assert.Contains(t, result, "|")
	assert.NotContains(t, result, "\\n")
}

func TestFormatYAML_Array(t *testing.T) {
	data := []any{"a", "b", "c"}
	result, err := FormatYAML(data, YAMLFormatOptions{})
	require.NoError(t, err)
	assert.Contains(t, result, "- a")
	assert.Contains(t, result, "- b")
	assert.Contains(t, result, "- c")
}

func TestFormatYAML_Nested(t *testing.T) {
	data := map[string]any{
		"users": []any{
			map[string]any{"name": "alice"},
			map[string]any{"name": "bob"},
		},
	}
	result, err := FormatYAML(data, YAMLFormatOptions{Indent: 4})
	require.NoError(t, err)
	assert.Contains(t, result, "users:")
	assert.Contains(t, result, "name: alice")
	assert.Contains(t, result, "name: bob")
}

func TestFormatYAML_CustomIndent(t *testing.T) {
	data := map[string]any{"a": map[string]any{"b": "c"}}
	result, err := FormatYAML(data, YAMLFormatOptions{Indent: 4})
	require.NoError(t, err)
	assert.Contains(t, result, "    b: c")
}

func TestFormatYAML_EmptyMap(t *testing.T) {
	data := map[string]any{}
	result, err := FormatYAML(data, YAMLFormatOptions{})
	require.NoError(t, err)
	assert.Contains(t, result, "{}")
}

func TestFormatYAML_NilValue(t *testing.T) {
	data := map[string]any{"key": nil}
	result, err := FormatYAML(data, YAMLFormatOptions{})
	require.NoError(t, err)
	assert.Contains(t, result, "key: null")
}

func TestFormatYAML_BooleanValues(t *testing.T) {
	data := map[string]any{"active": true, "deleted": false}
	result, err := FormatYAML(data, YAMLFormatOptions{})
	require.NoError(t, err)
	assert.Contains(t, result, "active: true")
	assert.Contains(t, result, "deleted: false")
}
