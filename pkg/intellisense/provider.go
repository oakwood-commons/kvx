// Package intellisense provides a reusable completion engine for expression languages.
//
// This package exports a clean API for integrating intelligent completion and type-aware
// suggestions into CLI and TUI applications. It supports pluggable expression engines,
// starting with CEL (Common Expression Language), with the ability to add jq, JSONPath,
// and other expression languages.
//
// # Basic Usage
//
// Create a CEL provider and use it for completions:
//
//	provider, err := intellisense.NewCELProvider()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	ctx := intellisense.CompletionContext{
//		CurrentNode: myData,
//		CurrentType: "map",
//	}
//
//	completions := provider.FilterCompletions("_.items.", ctx)
//	for _, c := range completions {
//		fmt.Printf("%s - %s\n", c.Display, c.Detail)
//	}
//
// # Interactive Mode Example
//
// See the example.go file for a complete interactive REPL implementation.
package intellisense

import (
	"strings"

	"github.com/oakwood-commons/kvx/internal/completion"
)

// Provider defines the interface for expression completion engines.
// Different expression languages (CEL, jq, etc.) implement this interface
// to provide language-specific completion, type inference, and evaluation.
type Provider interface {
	// DiscoverFunctions returns all available functions for this expression language.
	DiscoverFunctions() []FunctionMetadata

	// FilterCompletions returns completions based on current input and context.
	FilterCompletions(input string, context CompletionContext) []Completion

	// EvaluateType infers the result type of an expression without evaluating it.
	EvaluateType(expr string, context CompletionContext) string

	// Evaluate executes the expression and returns the result.
	Evaluate(expr string, root interface{}) (interface{}, error)

	// IsExpression determines if the given string is a valid expression.
	IsExpression(expr string) bool
}

// FunctionMetadata describes a function available in the expression language.
type FunctionMetadata = completion.FunctionMetadata

// Completion represents a single completion suggestion.
type Completion = completion.Completion

// CompletionKind indicates the type of completion.
type CompletionKind = completion.CompletionKind

// Completion kinds
const (
	CompletionField    = completion.CompletionField
	CompletionIndex    = completion.CompletionIndex
	CompletionFunction = completion.CompletionFunction
	CompletionKeyword  = completion.CompletionKeyword
	CompletionVariable = completion.CompletionVariable
)

// CompletionContext provides context for intelligent completion filtering.
type CompletionContext = completion.CompletionContext

// NewCELProvider creates a new CEL expression completion provider.
// The provider automatically discovers CEL functions and provides type-aware completions.
//
// Example:
//
//	provider, err := intellisense.NewCELProvider()
//	if err != nil {
//		return err
//	}
//	// Use provider for completions
func NewCELProvider() (Provider, error) {
	return completion.NewCELProvider()
}

// NewJQProvider creates a new jq completion provider (stub).
func NewJQProvider() (Provider, error) {
	return completion.NewJQProvider()
}

var defaultLanguage = "cel"
var customProvider Provider

// SetDefaultLanguage sets the default intellisense provider language (e.g., "cel", "jq").
func SetDefaultLanguage(lang string) {
	l := strings.ToLower(strings.TrimSpace(lang))
	if l == "jq" || l == "cel" {
		defaultLanguage = l
	}
}

// SetProvider allows host applications to inject a custom Provider implementation
// (e.g., an MQL provider). When set, NewProvider() will return this provider
// instead of the built-in factory.
func SetProvider(p Provider) {
	customProvider = p
}

// NewProvider returns a provider for the configured default language.
// Defaults to CEL; set via SetDefaultLanguage("jq") to use jq.
func NewProvider() (Provider, error) {
	if customProvider != nil {
		return customProvider, nil
	}
	switch defaultLanguage {
	case "jq":
		return NewJQProvider()
	case "cel":
		fallthrough
	default:
		return NewCELProvider()
	}
}

// SearchOptions configures completion behavior.
type SearchOptions struct {
	// MaxResults limits the number of completion results returned.
	// 0 means no limit.
	MaxResults int

	// CaseSensitive controls whether completion matching is case-sensitive.
	CaseSensitive bool

	// FuzzyMatch enables fuzzy/substring matching instead of prefix-only.
	FuzzyMatch bool

	// ShowDescriptions includes detailed descriptions in completion results.
	ShowDescriptions bool

	// TypeAwareFiltering filters completions based on the current data type.
	TypeAwareFiltering bool
}

// DefaultSearchOptions returns sensible defaults for search options.
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		MaxResults:         10,
		CaseSensitive:      false,
		FuzzyMatch:         false,
		ShowDescriptions:   true,
		TypeAwareFiltering: true,
	}
}
