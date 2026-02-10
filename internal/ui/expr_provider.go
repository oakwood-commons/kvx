package ui

import (
	"github.com/google/cel-go/cel"

	celhelper "github.com/oakwood-commons/kvx/internal/cel"
	"github.com/oakwood-commons/kvx/internal/navigator"
)

// ExpressionProvider allows pluggable evaluation and suggestion sources for the TUI.
type ExpressionProvider interface {
	Evaluate(expr string, root interface{}) (interface{}, error)
	DiscoverSuggestions() []string
	IsExpression(expr string) bool
}

type celExpressionProvider struct{}

func (celExpressionProvider) Evaluate(expr string, root interface{}) (interface{}, error) {
	evaluator, err := celhelper.NewEvaluator()
	if err != nil {
		return nil, err
	}
	return evaluator.Evaluate(expr, root)
}

func (celExpressionProvider) DiscoverSuggestions() []string {
	if list, err := celhelper.DiscoverCELFunctionDocs(); err == nil && len(list) > 0 {
		return list
	}
	if list, err := celhelper.DiscoverCELFunctions(); err == nil && len(list) > 0 {
		return list
	}
	return celhelper.GetAvailableFunctions()
}

func (celExpressionProvider) IsExpression(expr string) bool {
	return celhelper.IsCELExpression(expr)
}

var exprProvider ExpressionProvider = celExpressionProvider{}

// SetExpressionProvider overrides the default expression provider (global).
// This also configures the navigator to use the same evaluator for complex CEL expressions.
func SetExpressionProvider(p ExpressionProvider) {
	if p != nil {
		exprProvider = p
		// Also wire up the navigator to use the same evaluator
		navigator.SetEvaluator(p.Evaluate)
	}
}

// ResetExpressionProvider restores the default CEL expression provider.
// This also resets the navigator to use its default evaluator.
// Useful for test cleanup.
func ResetExpressionProvider() {
	exprProvider = celExpressionProvider{}
	navigator.SetEvaluator(nil)
}

// DefaultExpressionProvider returns the built-in CEL provider.
func DefaultExpressionProvider() ExpressionProvider {
	return celExpressionProvider{}
}

// EvaluateExpression evaluates expr using the configured provider.
func EvaluateExpression(expr string, root interface{}) (interface{}, error) {
	return exprProvider.Evaluate(expr, root)
}

// DiscoverExpressions returns suggestions from the configured provider.
func DiscoverExpressions() []string {
	return exprProvider.DiscoverSuggestions()
}

// IsExpression delegates to the configured provider to detect expression syntax.
func IsExpression(expr string) bool {
	return exprProvider.IsExpression(expr)
}

// celEnvProvider wraps a CEL environment to implement ExpressionProvider.
// This makes it easy to use a custom CEL environment (with extended functions)
// with the TUI without implementing the full ExpressionProvider interface.
type celEnvProvider struct {
	env          *cel.Env
	exampleHints map[string]string
}

// Evaluate evaluates expressions using the wrapped CEL environment.
func (p *celEnvProvider) Evaluate(expr string, root interface{}) (interface{}, error) {
	return celhelper.EvaluateExpressionWithEnv(p.env, expr, root)
}

// DiscoverSuggestions discovers functions from the wrapped CEL environment.
func (p *celEnvProvider) DiscoverSuggestions() []string {
	return celhelper.DiscoverFunctionsFromEnv(p.env, p.exampleHints)
}

// IsExpression detects if a string contains CEL expression syntax.
func (p *celEnvProvider) IsExpression(expr string) bool {
	return celhelper.IsCELExpression(expr)
}

// NewCELExpressionProvider creates an ExpressionProvider from a CEL environment.
// This helper makes it easy to use custom CEL environments (with extended functions)
// with the TUI. The environment should include the "_" variable for root data access.
//
// exampleHints is optional and can provide helpful usage examples for functions.
// For example: map[string]string{"myFunc": "e.g. myFunc(arg)"}
//
// Example:
//
//	env, _ := cel.NewEnv(
//	    cel.Variable("_", cel.DynType),
//	    cel.Function("myFunc", ...),
//	)
//	provider := ui.NewCELExpressionProvider(env, nil)
//	ui.SetExpressionProvider(provider)
func NewCELExpressionProvider(env *cel.Env, exampleHints map[string]string) ExpressionProvider {
	if env == nil {
		return DefaultExpressionProvider()
	}
	return &celEnvProvider{
		env:          env,
		exampleHints: exampleHints,
	}
}
