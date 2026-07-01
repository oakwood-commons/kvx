package formatter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatTOML_SingleMap(t *testing.T) {
	data := map[string]any{"name": "alice"}
	out, err := FormatTOML(data)
	require.NoError(t, err)
	assert.Regexp(t, `(?m)^name\s*=\s*["']alice["']\s*$`, out)
	assert.NotContains(t, out, "items")
}

func TestFormatTOML_Slice(t *testing.T) {
	data := []map[string]any{
		{"name": "alice"},
		{"name": "bob"},
	}
	out, err := FormatTOML(data)
	require.NoError(t, err)
	assert.Contains(t, out, "[[items]]")
	assert.Regexp(t, `(?m)^name\s*=\s*["']alice["']\s*$`, out)
	assert.Regexp(t, `(?m)^name\s*=\s*["']bob["']\s*$`, out)
}

func TestFormatTOML_EmptySlice(t *testing.T) {
	data := []any{}
	out, err := FormatTOML(data)
	require.NoError(t, err)
	assert.NotEmpty(t, out)
}

func TestFormatTOML_SliceOfScalars(t *testing.T) {
	data := []any{"a", "b", "c"}
	out, err := FormatTOML(data)
	require.NoError(t, err)
	assert.Contains(t, out, "items")
}

func TestFormatTOML_Nil(t *testing.T) {
	out, err := FormatTOML(nil)
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestFormatTOML_TypedNilPointer(t *testing.T) {
	var data *[]map[string]any
	out, err := FormatTOML(data)
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestFormatTOML_ScalarRoot(t *testing.T) {
	out, err := FormatTOML("alice")
	require.NoError(t, err)
	assert.Contains(t, out, "alice")
}

func TestFormatTOML_PointerToSlice(t *testing.T) {
	data := []map[string]any{{"name": "alice"}}
	out, err := FormatTOML(&data)
	require.NoError(t, err)
	assert.Contains(t, out, "[[items]]")
	assert.Regexp(t, `(?m)^name\s*=\s*["']alice["']\s*$`, out)
}
