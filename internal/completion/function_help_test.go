package completion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatFunctionOneLiner(t *testing.T) {
	tests := []struct {
		name     string
		fn       FunctionMetadata
		expected string
	}{
		{
			name:     "signature and description",
			fn:       FunctionMetadata{Name: "filter", Signature: "list.filter(x, cond)", Description: "Filter list elements", Examples: []string{"[1,2].filter(x, x>1)"}},
			expected: "list.filter(x, cond) — Filter list elements",
		},
		{
			name:     "description only no signature",
			fn:       FunctionMetadata{Name: "int", Description: "Convert to integer"},
			expected: "int() — Convert to integer",
		},
		{
			name:     "examples ignored in one-liner",
			fn:       FunctionMetadata{Name: "abs", Examples: []string{"math.abs(-5)"}},
			expected: "abs()",
		},
		{
			name:     "no description or examples falls back to signature",
			fn:       FunctionMetadata{Name: "type", Signature: "type(value)"},
			expected: "type(value)",
		},
		{
			name:     "no metadata falls back to name()",
			fn:       FunctionMetadata{Name: "unknown"},
			expected: "unknown()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatFunctionOneLiner(tt.fn)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatFunctionSignature(t *testing.T) {
	fn := FunctionMetadata{Name: "filter", Signature: "list.filter(x, condition)"}
	assert.Equal(t, "list.filter(x, condition)", FormatFunctionSignature(fn))

	fn2 := FunctionMetadata{Name: "type"}
	assert.Equal(t, "type()", FormatFunctionSignature(fn2))
}

func TestFormatFunctionLines(t *testing.T) {
	fn := FunctionMetadata{
		Name:        "filter",
		Signature:   "list.filter(x, cond)",
		Description: "Filter elements",
		Examples:    []string{"[1,2].filter(x, x>1)", "[].filter(x, true)", "extra"},
	}

	lines := FormatFunctionLines(fn, 2)
	assert.Equal(t, []string{
		"list.filter(x, cond)",
		"Filter elements",
		"  [1,2].filter(x, x>1)",
		"  [].filter(x, true)",
	}, lines)
}

func TestFormatFunctionLinesNoExamples(t *testing.T) {
	fn := FunctionMetadata{
		Name:        "int",
		Description: "Convert to integer",
	}

	lines := FormatFunctionLines(fn, 2)
	assert.Equal(t, []string{
		"int()",
		"Convert to integer",
	}, lines)
}
