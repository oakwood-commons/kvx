//revive:disable:exported
package completion

// Provider defines the interface for expression completion engines.
// Implementations provide language-specific completion, type inference, and evaluation.
type Provider interface {
	// DiscoverFunctions returns all available functions for this expression language.
	// Each provider determines its own discovery strategy (programmatic introspection,
	// static metadata, etc.).
	DiscoverFunctions() []FunctionMetadata

	// FilterCompletions returns completions based on current input and context.
	// Input is the current expression text, context contains type information
	// and other metadata needed for intelligent filtering.
	FilterCompletions(input string, context CompletionContext) []Completion

	// EvaluateType infers the result type of an expression without evaluating it.
	// Returns a string like "string", "list", "map", "bool", "int", etc.
	// Returns empty string if type cannot be inferred.
	EvaluateType(expr string, context CompletionContext) string

	// Evaluate executes the expression and returns the result.
	Evaluate(expr string, root interface{}) (interface{}, error)

	// IsExpression determines if the given string is a valid expression
	// for this language (vs. a simple path or key).
	IsExpression(expr string) bool
}

// FunctionMetadata describes a function available in the expression language.
type FunctionMetadata struct {
	Name        string   // Function name (e.g., "contains", "map", "filter")
	Signature   string   // Full signature (e.g., "contains(string, substring) -> bool")
	Description string   // Human-readable description
	Category    string   // Category for grouping (e.g., "string", "list", "math")
	IsMethod    bool     // True if this is a method (called on an object)
	ReturnType  string   // Return type (e.g., "bool", "string", "list")
	ParamTypes  []string // Parameter types
	Examples    []string // Usage examples (e.g., "[1,2,3].filter(x, x > 1) => [2,3]")
}

// Completion represents a single completion suggestion.
type Completion struct {
	Text        string            // The text to insert if selected
	Display     string            // Display text (may include formatting)
	Kind        CompletionKind    // Type of completion
	Detail      string            // Additional detail (e.g., function signature)
	Description string            // Longer description for help panel
	Score       int               // Relevance score for sorting (higher = more relevant)
	Function    *FunctionMetadata // Optional: full function metadata if Kind == CompletionFunction
}

// CompletionKind indicates the type of completion.
type CompletionKind int

const (
	CompletionField    CompletionKind = iota // Object field/key
	CompletionIndex                          // Array index
	CompletionFunction                       // Function
	CompletionKeyword                        // Language keyword
	CompletionVariable                       // Variable name
)

// CompletionContext provides context for intelligent completion filtering.
type CompletionContext struct {
	// CurrentNode is the data node currently being viewed/navigated
	CurrentNode interface{}

	// CurrentType is the type of the current node ("string", "list", "map", etc.)
	CurrentType string

	// CursorPosition is the position of the cursor in the input
	CursorPosition int

	// ExpressionResult holds the result of the last successful evaluation
	ExpressionResult interface{}

	// ExpressionResultType is the type of the last evaluation result
	ExpressionResultType string

	// PartialToken is the text being typed after the last operator (e.g., after ".")
	PartialToken string

	// IsAfterDot indicates if completion is happening after a "." operator
	IsAfterDot bool
}

// CompletionEngine wraps a Provider and adds common filtering/scoring logic.
type CompletionEngine struct {
	provider Provider
}

//revive:enable:exported

// NewEngine creates a new completion engine with the given provider.
func NewEngine(provider Provider) *CompletionEngine {
	return &CompletionEngine{
		provider: provider,
	}
}

// GetCompletions returns filtered and scored completions for the current input.
func (e *CompletionEngine) GetCompletions(input string, context CompletionContext) []Completion {
	return e.provider.FilterCompletions(input, context)
}

// GetFunctions returns all available functions.
func (e *CompletionEngine) GetFunctions() []FunctionMetadata {
	return e.provider.DiscoverFunctions()
}

// InferType returns the inferred type of the expression.
func (e *CompletionEngine) InferType(expr string, context CompletionContext) string {
	return e.provider.EvaluateType(expr, context)
}

// Evaluate executes the expression.
func (e *CompletionEngine) Evaluate(expr string, root interface{}) (interface{}, error) {
	return e.provider.Evaluate(expr, root)
}

// IsExpression checks if the input is an expression.
func (e *CompletionEngine) IsExpression(expr string) bool {
	return e.provider.IsExpression(expr)
}
