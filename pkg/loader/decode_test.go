package loader

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTryDecode_JSON(t *testing.T) {
	decoded, ok := TryDecode(`{"name":"alice","age":30}`)
	require.True(t, ok)
	m, ok := decoded.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "alice", m["name"])
	assert.Equal(t, float64(30), m["age"])
}

func TestTryDecode_JSONArray(t *testing.T) {
	decoded, ok := TryDecode(`[1,2,3]`)
	require.True(t, ok)
	arr, ok := decoded.([]any)
	require.True(t, ok)
	assert.Len(t, arr, 3)
}

func TestTryDecode_YAML(t *testing.T) {
	decoded, ok := TryDecode("name: bob\nage: 25\n")
	require.True(t, ok)
	m, ok := decoded.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "bob", m["name"])
	assert.Equal(t, 25, m["age"])
}

func TestTryDecode_JWT(t *testing.T) {
	jwt := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	decoded, ok := TryDecode(jwt)
	require.True(t, ok)
	m, ok := decoded.(map[string]any)
	require.True(t, ok)
	assert.Contains(t, m, "header")
	assert.Contains(t, m, "payload")
}

func TestTryDecode_PlainString(t *testing.T) {
	_, ok := TryDecode("hello world")
	assert.False(t, ok, "plain string should not decode")
}

func TestTryDecode_Number(t *testing.T) {
	_, ok := TryDecode("42")
	assert.False(t, ok, "bare number should not decode")
}

func TestTryDecode_Empty(t *testing.T) {
	_, ok := TryDecode("")
	assert.False(t, ok, "empty string should not decode")
}

func TestTryDecode_BooleanString(t *testing.T) {
	_, ok := TryDecode("true")
	assert.False(t, ok, "boolean string should not decode to structured data")
}

func TestRecursiveDecode_Flat(t *testing.T) {
	input := map[string]any{
		"name":    "alice",
		"payload": `{"key":"value"}`,
	}
	result := RecursiveDecode(input)
	m, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "alice", m["name"])
	inner, ok := m["payload"].(map[string]any)
	require.True(t, ok, "payload should be decoded to map")
	assert.Equal(t, "value", inner["key"])
}

func TestRecursiveDecode_Nested(t *testing.T) {
	// A JSON string containing another JSON string
	input := map[string]any{
		"outer": `{"inner":"{\"deep\":true}"}`,
	}
	result := RecursiveDecode(input)
	m := result.(map[string]any)
	outer := m["outer"].(map[string]any)
	inner := outer["inner"].(map[string]any)
	assert.Equal(t, true, inner["deep"])
}

func TestRecursiveDecode_Array(t *testing.T) {
	input := []any{
		`{"a":1}`,
		"plain",
		42,
	}
	result := RecursiveDecode(input)
	arr, ok := result.([]any)
	require.True(t, ok)
	decoded, ok := arr[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(1), decoded["a"])
	assert.Equal(t, "plain", arr[1])
	assert.Equal(t, 42, arr[2])
}

func TestRecursiveDecode_NoChange(t *testing.T) {
	input := map[string]any{
		"name": "alice",
		"age":  30,
	}
	result := RecursiveDecode(input)
	m := result.(map[string]any)
	assert.Equal(t, "alice", m["name"])
	assert.Equal(t, 30, m["age"])
}

func TestRecursiveDecode_TypedMapStringString(t *testing.T) {
	// map[string]string should also be traversed and strings decoded
	input := map[string]string{
		"payload": `{"key":"value"}`,
		"plain":   "hello",
	}
	result := RecursiveDecode(input)
	m, ok := result.(map[string]any)
	require.True(t, ok, "typed map should convert to map[string]any")

	decoded, ok := m["payload"].(map[string]any)
	require.True(t, ok, "payload should be decoded to map")
	assert.Equal(t, "value", decoded["key"])
	assert.Equal(t, "hello", m["plain"])
}

func TestRecursiveDecode_TypedSliceString(t *testing.T) {
	// []string should also be traversed and strings decoded
	input := []string{
		`{"a":1}`,
		"plain",
	}
	result := RecursiveDecode(input)
	arr, ok := result.([]any)
	require.True(t, ok, "typed slice should convert to []any")

	decoded, ok := arr[0].(map[string]any)
	require.True(t, ok, "first element should be decoded to map")
	assert.Equal(t, float64(1), decoded["a"])
	assert.Equal(t, "plain", arr[1])
}

func TestRecursiveDecode_NestedTypedMapInSlice(t *testing.T) {
	// []map[string]string - nested typed containers
	input := []map[string]string{
		{"payload": `{"nested":"decoded"}`},
		{"plain": "hello"},
	}
	result := RecursiveDecode(input)
	arr, ok := result.([]any)
	require.True(t, ok, "slice should convert to []any")
	require.Len(t, arr, 2)

	first, ok := arr[0].(map[string]any)
	require.True(t, ok, "first map should convert")
	decoded, ok := first["payload"].(map[string]any)
	require.True(t, ok, "payload should be decoded")
	assert.Equal(t, "decoded", decoded["nested"])

	second, ok := arr[1].(map[string]any)
	require.True(t, ok, "second map should convert")
	assert.Equal(t, "hello", second["plain"])
}

func TestIsStructured(t *testing.T) {
	assert.True(t, isStructured(map[string]any{"a": 1}))
	assert.True(t, isStructured([]any{1, 2}))
	assert.False(t, isStructured("hello"))
	assert.False(t, isStructured(42))
	assert.False(t, isStructured(nil))
	assert.False(t, isStructured(true))
}
