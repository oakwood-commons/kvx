package ui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	celhelper "github.com/oakwood-commons/kvx/internal/cel"
)

// Ensure embedded CEL examples stay valid as configuration evolves.
func TestEmbeddedDefaultCELExamplesEvaluate(t *testing.T) {
	cfg, err := EmbeddedDefaultConfig()
	require.NoError(t, err)

	exampleData := cfg.Help.CEL.ExampleData
	require.NotNil(t, exampleData, "example_data is required for validation")

	examples := cfg.Help.CEL.FunctionExamples
	require.NotEmpty(t, examples, "function_examples are required for validation")

	evaluator, err := celhelper.NewEvaluator()
	require.NoError(t, err)

	for fn, entry := range examples {
		for _, sample := range entry.Examples {
			expr := normalizeExampleExpr(sample)
			if expr == "" {
				continue
			}
			_, err := evaluator.Evaluate(expr, exampleData)
			assert.NoErrorf(t, err, "function %s example %q should evaluate", fn, sample)
		}
	}
}

func normalizeExampleExpr(sample string) string {
	expr := strings.TrimSpace(sample)
	if expr == "" {
		return ""
	}
	if arrow := strings.Index(expr, "=>"); arrow >= 0 {
		expr = expr[:arrow]
	}
	return strings.TrimSpace(expr)
}
