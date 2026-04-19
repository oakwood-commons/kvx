package formatter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatAsCSV_ArrayOfMaps(t *testing.T) {
	data := []any{
		map[string]any{"name": "alice", "age": 30},
		map[string]any{"name": "bob", "age": 25},
	}
	result := FormatAsCSV(data)
	assert.Contains(t, result, "age,name")
	assert.Contains(t, result, "alice")
	assert.Contains(t, result, "bob")
}

func TestFormatAsCSV_SimpleArray(t *testing.T) {
	data := []any{"a", "b", "c"}
	result := FormatAsCSV(data)
	assert.Contains(t, result, "value")
	assert.Contains(t, result, "a\n")
	assert.Contains(t, result, "b\n")
	assert.Contains(t, result, "c\n")
}

func TestFormatAsCSV_Map(t *testing.T) {
	data := map[string]any{"name": "test", "count": 42}
	result := FormatAsCSV(data)
	assert.Contains(t, result, "key,value")
	assert.Contains(t, result, "name,test")
}

func TestFormatAsCSV_Scalar(t *testing.T) {
	result := FormatAsCSV("hello")
	assert.Contains(t, result, "value")
	assert.Contains(t, result, "hello")
}

func TestFormatAsCSV_EmptyArray(t *testing.T) {
	result := FormatAsCSV([]any{})
	assert.Empty(t, result)
}

func TestEscapeCSVField(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "simple", input: "hello", want: "hello"},
		{name: "comma", input: "a,b", want: `"a,b"`},
		{name: "quotes", input: `say "hi"`, want: `"say ""hi"""`},
		{name: "newline", input: "line1\nline2", want: "\"line1\nline2\""},
		{name: "space", input: "hello world", want: `"hello world"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, escapeCSVField(tt.input))
		})
	}
}
