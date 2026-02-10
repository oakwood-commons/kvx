package completion

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	celhelper "github.com/oakwood-commons/kvx/internal/cel"

	"github.com/google/cel-go/cel"
)

// FunctionExampleData holds both description and examples for a function
type FunctionExampleData struct {
	Description string
	Examples    []string
}

// globalFunctionExamples holds function examples loaded from config.
// This is set by SetFunctionExamples() during initialization.
var globalFunctionExamples map[string]FunctionExampleData

// SetFunctionExamples sets the global function examples database from config.
// Accepts legacy format (map[string][]string) for backwards compatibility.
func SetFunctionExamples(examples map[string][]string) {
	// Convert legacy format to new format
	globalFunctionExamples = make(map[string]FunctionExampleData)
	for name, exs := range examples {
		globalFunctionExamples[name] = FunctionExampleData{
			Examples: exs,
		}
	}
}

// SetFunctionExamplesWithDescriptions sets function examples with descriptions.
func SetFunctionExamplesWithDescriptions(examples map[string]FunctionExampleData) {
	globalFunctionExamples = examples
}

// GetFunctionExamples returns the current function examples database in legacy format.
func GetFunctionExamples() map[string][]string {
	examplesData := GetFunctionExamplesData()
	result := make(map[string][]string)
	for name, data := range examplesData {
		result[name] = data.Examples
	}
	return result
}

// GetFunctionExamplesData returns the function examples with descriptions.
func GetFunctionExamplesData() map[string]FunctionExampleData {
	if globalFunctionExamples != nil {
		return globalFunctionExamples
	}
	return getDefaultFunctionExamplesData()
}

// CELProvider implements Provider for the CEL expression language.
type CELProvider struct {
	functions     []FunctionMetadata
	functionDocs  map[string]string
	functionCache []string // cached function list for UI
}

// NewCELProvider creates a new CEL completion provider.
// It automatically discovers functions from the CEL environment.
func NewCELProvider() (*CELProvider, error) {
	p := &CELProvider{
		functionDocs: make(map[string]string),
	}

	// Discover functions programmatically from CEL environment
	p.discoverFunctions()

	return p, nil
}

// newCELEnv creates a CEL environment for parsing
func newCELEnv() (*cel.Env, error) {
	eval, err := celhelper.NewEvaluator()
	if err != nil {
		return nil, err
	}
	return eval.GetEnvironment(), nil
}

// discoverFunctions programmatically discovers CEL functions using the evaluator.
func (p *CELProvider) discoverFunctions() {
	// Get function documentation (includes examples and signatures)
	docs, err := celhelper.DiscoverCELFunctionDocs()
	if err == nil && len(docs) > 0 {
		p.functionCache = docs
		// Parse docs into metadata
		for _, doc := range docs {
			meta := p.parseFunctionDoc(doc)
			if meta.Name != "" {
				p.functions = append(p.functions, meta)
				p.functionDocs[normalizeFuncName(meta.Name)] = meta.Description
			}
		}
	} else {
		// Fallback: discover function names only
		names, err := celhelper.DiscoverCELFunctions()
		if err == nil && len(names) > 0 {
			p.functionCache = names
			for _, name := range names {
				meta := p.parseFunctionDoc(name)
				if meta.Name != "" {
					p.functions = append(p.functions, meta)
				}
			}
		} else {
			// Last resort: use static list
			p.functionCache = celhelper.GetAvailableFunctions()
			for _, fn := range p.functionCache {
				meta := p.parseFunctionDoc(fn)
				if meta.Name != "" {
					p.functions = append(p.functions, meta)
				}
			}
		}
	}

	// Enrich functions with usage examples
	enrichWithExamples(p.functions)
}

// parseFunctionDoc parses a function string like "filter(x, condition) - Filter array elements"
// into FunctionMetadata.
func (p *CELProvider) parseFunctionDoc(doc string) FunctionMetadata {
	parts := strings.SplitN(doc, " - ", 2)
	signature := strings.TrimSpace(parts[0])
	description := ""
	if len(parts) > 1 {
		description = strings.TrimSpace(parts[1])
	}

	// Extract function name
	name := signature
	if idx := strings.Index(signature, "("); idx > 0 {
		name = signature[:idx]
	}
	name = strings.TrimSpace(name)

	// Determine if this is a method (appears after a dot)
	isMethod := strings.Contains(description, "method") || strings.Contains(strings.ToLower(description), "called on")

	// Categorize based on common patterns
	category := categorizeFunction(name, description)

	return FunctionMetadata{
		Name:        name,
		Signature:   signature,
		Description: description,
		Category:    category,
		IsMethod:    isMethod,
		ReturnType:  inferReturnType(name, description),
		ParamTypes:  []string{}, // CEL doesn't expose detailed param types easily
	}
}

// DiscoverFunctions returns all available CEL functions.
func (p *CELProvider) DiscoverFunctions() []FunctionMetadata {
	return p.functions
}

// FilterCompletions returns completions based on current input and context.
func (p *CELProvider) FilterCompletions(input string, context CompletionContext) []Completion {
	input = strings.TrimSpace(input)
	if input == "" {
		input = "_"
	}

	// Parse the input to extract completion context
	// For incomplete expressions (e.g., "_.items." or "_.items["), the CEL parser will fail
	// which is expected - we handle both complete and incomplete expressions
	var hasRoot bool
	var partial string
	var baseExpr string
	var containsFunctionCall bool
	var segs []string

	// When context.PartialToken is provided, it means the user is typing a partial token
	// In this case, use string-based parsing to ensure accurate segment extraction
	// because AST parsing on an incomplete path might fail or return unexpected results
	useStringParsing := context.PartialToken != ""

	// Try AST-based parsing for the base expression (before trailing . or [)
	// This gives us accurate function call detection for complete sub-expressions
	if !useStringParsing {
		env, _ := newCELEnv()
		if env != nil {
			parsed, _ := ParseCELExpression(input, env)
			if parsed != nil {
				hasRoot = parsed.HasRoot
				partial = parsed.Partial
				baseExpr = parsed.BaseExpr
				containsFunctionCall = parsed.HasFunctionCalls
				segs = parsed.Segments
			}
		}
	}

	// If AST parsing didn't populate the fields (likely due to incomplete syntax),
	// fall back to string-based analysis - this is the normal case during typing
	if baseExpr == "" {
		hasRoot = strings.HasPrefix(input, "_")
		containsFunctionCall = strings.Contains(input, "(")
		segs = splitPathSegments(input)

		if strings.HasSuffix(input, ".") || strings.HasSuffix(input, "[") {
			partial = ""
		} else if len(segs) > 0 {
			lastSeg := segs[len(segs)-1]
			if _, err := strconv.Atoi(lastSeg); err != nil {
				partial = lastSeg
				segs = segs[:len(segs)-1]
			}
		}

		if partial == "" && !containsFunctionCall {
			baseExpr = buildCompletion("", segs, hasRoot)
		} else {
			baseExpr = strings.TrimSpace(input)
			switch {
			case strings.HasSuffix(baseExpr, "."):
				baseExpr = strings.TrimSuffix(baseExpr, ".")
			case strings.HasSuffix(baseExpr, "["):
				baseExpr = strings.TrimSuffix(baseExpr, "[")
			case partial != "":
				baseExpr = strings.TrimSuffix(baseExpr, partial)
				baseExpr = strings.TrimSuffix(baseExpr, ".")
			}
		}
	}

	// Evaluate the base expression to get the actual node we're completing
	currentNode := context.CurrentNode
	currentNodeType := ""
	evaluationFailed := false

	if baseExpr != "" && baseExpr != "_" {
		// Try to evaluate the expression to get the actual node
		// If expression doesn't start with _, add it for CEL evaluation
		evalExpr := baseExpr
		if !strings.HasPrefix(evalExpr, "_") {
			evalExpr = "_." + strings.TrimPrefix(evalExpr, ".")
		}
		evaluated, err := p.Evaluate(evalExpr, context.CurrentNode)
		if err == nil && evaluated != nil {
			currentNode = evaluated
			currentNodeType = inferNodeType(evaluated)
		} else {
			// Evaluation failed - the path is invalid
			evaluationFailed = true
		}
	}

	// If the base expression evaluation failed and we're not at the root:
	// - If context.CurrentType is provided, use it for type-aware completions
	// - Otherwise, return empty completions
	if evaluationFailed && baseExpr != "" && baseExpr != "_" {
		if context.CurrentType == "" {
			return []Completion{} // Return empty list for invalid paths with no type info
		}
		// Use the provided type from context for completions
		currentNodeType = context.CurrentType
	}

	// Determine node type for filtering
	nodeType := context.ExpressionResultType
	if nodeType == "" {
		nodeType = context.CurrentType
	}
	if nodeType == "" {
		if currentNodeType != "" {
			nodeType = currentNodeType
		} else {
			nodeType = inferNodeType(currentNode)
		}
	}

	// Use context.PartialToken if provided (more accurate for nested paths)
	if context.PartialToken != "" {
		partial = context.PartialToken
	}
	partialLower := strings.ToLower(partial)
	completions := []Completion{}

	// If partial is empty and we have numeric indices, include the base expression itself
	// This handles cases like "items.0" where the user typed a numeric index without trailing dot
	if partial == "" && len(segs) > 0 && !strings.HasSuffix(input, ".") && !strings.HasSuffix(input, "[") {
		lastSeg := segs[len(segs)-1]
		if _, err := strconv.Atoi(lastSeg); err == nil {
			// Last segment is numeric - include base expression as a completion
			baseExpr := buildCompletion("", segs, hasRoot)
			if baseExpr != "" {
				completions = append(completions, Completion{
					Text:    baseExpr,
					Display: baseExpr,
					Kind:    CompletionField,
					Detail:  "array index",
					Score:   200, // High priority for exact matches
				})
			}
		}
	}

	// Add field/key completions
	keys := listKeys(currentNode)
	for _, key := range keys {
		if partialLower == "" || strings.HasPrefix(strings.ToLower(key), partialLower) {
			// For old model mode (no underscore), just return the key name
			// For expression mode (with underscore), return the full path
			var completionText string
			if hasRoot {
				if containsFunctionCall {
					// When there's a function call, append to baseExpr instead of rebuilding
					// Use bracket notation for numeric indices
					if _, err := strconv.Atoi(key); err == nil {
						completionText = baseExpr + "[" + key + "]"
					} else {
						completionText = baseExpr + "." + key
					}
				} else {
					// Normal path - rebuild from segments
					completionText = buildCompletion(key, segs, hasRoot)
				}
			} else {
				completionText = key
			}
			completions = append(completions, Completion{
				Text:    completionText,
				Display: key,
				Kind:    CompletionField,
				Detail:  fmt.Sprintf("field: %s", key),
				Score:   100, // Fields get higher priority
			})
		}
	}

	// Add function completions
	funcPartial := strings.ToLower(partial)
	seenFn := make(map[string]bool)

	for _, meta := range p.functions {
		if !p.isCompatibleWithType(meta.Name, nodeType) {
			continue
		}

		fnLower := strings.ToLower(meta.Name)
		if funcPartial == "" || strings.HasPrefix(fnLower, funcPartial) {
			norm := normalizeFuncName(meta.Name)
			if seenFn[norm] {
				continue
			}
			seenFn[norm] = true

			display := meta.Name
			if !strings.Contains(display, "(") {
				display += "()"
			}

			// Build detailed help: description + up to 2 compact examples
			var detailParts []string

			// Add description first (most important)
			if meta.Description != "" {
				detailParts = append(detailParts, meta.Description)
			}

			// Add compact examples (max 2) on same line or next line
			if len(meta.Examples) > 0 {
				maxExamples := 2
				if len(meta.Examples) < maxExamples {
					maxExamples = len(meta.Examples)
				}
				exampleLines := make([]string, 0, maxExamples)
				for i := 0; i < maxExamples; i++ {
					exampleLines = append(exampleLines, meta.Examples[i])
				}
				detailParts = append(detailParts, "e.g. "+strings.Join(exampleLines, " | "))
			}

			detail := strings.Join(detailParts, "\n")

			// For old model mode (no underscore), just return the function name
			// For expression mode (with underscore), return the full path
			var completionText string
			if hasRoot {
				if containsFunctionCall {
					// When there's a function call, append to baseExpr instead of rebuilding
					completionText = baseExpr + "." + meta.Name
				} else {
					// Normal path - rebuild from segments
					completionText = buildCompletion(meta.Name, segs, hasRoot)
				}
			} else {
				completionText = meta.Name
			}

			// Boost score for better prefix matches
			score := 50 // Base score for functions
			if funcPartial != "" && strings.HasPrefix(fnLower, funcPartial) {
				// Exact prefix match - boost significantly
				if len(funcPartial) > 0 {
					score += len(funcPartial) * 10 // More typed chars = higher score
				}
			}

			completions = append(completions, Completion{
				Text:        completionText,
				Display:     display,
				Kind:        CompletionFunction,
				Detail:      detail,
				Description: meta.Description,
				Score:       score,
				Function:    &meta,
			})
		}
	}

	// Sort by score (descending) then alphabetically
	sort.Slice(completions, func(i, j int) bool {
		if completions[i].Score != completions[j].Score {
			return completions[i].Score > completions[j].Score
		}
		return completions[i].Display < completions[j].Display
	})

	return completions
}

// EvaluateType infers the result type of an expression.
func (p *CELProvider) EvaluateType(expr string, context CompletionContext) string {
	expr = strings.TrimSpace(expr)
	if expr == "" || expr == "_" {
		// Root node type
		if context.CurrentNode != nil {
			return inferGoType(context.CurrentNode)
		}
		return ""
	}

	// Try to evaluate the expression to get its result type
	evaluator, err := celhelper.NewEvaluator()
	if err != nil {
		return ""
	}

	// Use context.CurrentNode as the root if available
	root := context.CurrentNode
	if root == nil {
		return ""
	}

	result, err := evaluator.Evaluate(expr, root)
	if err != nil {
		return ""
	}

	return inferGoType(result)
}

// inferGoType infers the type name from a Go value
func inferGoType(value interface{}) string {
	if value == nil {
		return "null"
	}

	switch v := value.(type) {
	case string:
		return "string"
	case bool:
		return "bool"
	case int, int8, int16, int32, int64:
		return "int"
	case uint, uint8, uint16, uint32, uint64:
		return "uint"
	case float32, float64:
		return "double"
	case []interface{}:
		return "list"
	case map[string]interface{}:
		return "map"
	default:
		rt := reflect.TypeOf(v)
		if rt == nil {
			return "unknown"
		}

		k := rt.Kind()
		if k == reflect.Slice || k == reflect.Array {
			return "list"
		}
		if k == reflect.Map {
			return "map"
		}
		return k.String()
	}
}

// Evaluate executes the expression.
func (p *CELProvider) Evaluate(expr string, root interface{}) (interface{}, error) {
	evaluator, err := celhelper.NewEvaluator()
	if err != nil {
		return nil, err
	}
	return evaluator.Evaluate(expr, root)
}

// IsExpression checks if the input is a CEL expression.
func (p *CELProvider) IsExpression(expr string) bool {
	return celhelper.IsCELExpression(expr)
}

// isCompatibleWithType checks if a function is compatible with the given node type.
func (p *CELProvider) isCompatibleWithType(fnName, nodeType string) bool {
	fn := strings.ToLower(strings.TrimSpace(fnName))
	fn = strings.TrimSuffix(strings.TrimSuffix(fn, "()"), "(")
	if idx := strings.LastIndex(fn, "."); idx >= 0 {
		fn = fn[idx+1:]
	}

	// Universal helpers
	if fn == "type" {
		return true
	}

	switch nodeType {
	case "map":
		switch fn {
		case "keys", "values", "filter", "map", "all", "exists", "exists_one", "size", "has":
			return true
		}
	case "list":
		switch fn {
		case "filter", "map", "all", "exists", "exists_one", "size", "flatten", "slice", "sort":
			return true
		}
	case "string":
		switch fn {
		case "contains", "startswith", "endswith", "matches", "lowerascii", "upperascii", "size":
			return true
		}
	case "double", "int", "uint":
		switch fn {
		case "abs", "ceil", "floor", "round", "sqrt":
			return true
		}
	case "bool":
		// No specialized functions beyond universal ones
		return fn == "type"
	}

	// Default: allow if we can't determine incompatibility
	return false
}

// Helper functions

func splitPathSegments(path string) []string {
	var segments []string
	var current strings.Builder
	inBracket := false

	for i := 0; i < len(path); i++ {
		ch := path[i]
		switch ch {
		case '.':
			if !inBracket && current.Len() > 0 {
				segments = append(segments, current.String())
				current.Reset()
			} else {
				current.WriteByte(ch)
			}
		case '[':
			if current.Len() > 0 {
				segments = append(segments, current.String())
				current.Reset()
			}
			inBracket = true
		case ']':
			if inBracket && current.Len() > 0 {
				segments = append(segments, current.String())
				current.Reset()
			}
			inBracket = false
		default:
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		segments = append(segments, current.String())
	}

	return segments
}

func listKeys(node interface{}) []string {
	switch n := node.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(n))
		for k := range n {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return keys
	case []interface{}:
		keys := make([]string, len(n))
		for i := range n {
			keys[i] = strconv.Itoa(i)
		}
		return keys
	default:
		return nil
	}
}

func buildCompletion(seg string, baseSegs []string, hasRoot bool) string {
	// Filter out "_" from baseSegs since it's the root marker
	filteredSegs := make([]string, 0, len(baseSegs))
	for _, s := range baseSegs {
		if s != "_" {
			filteredSegs = append(filteredSegs, s)
		}
	}

	var b strings.Builder
	if hasRoot {
		b.WriteString("_")
	}

	for _, s := range filteredSegs {
		if _, err := strconv.Atoi(s); err == nil {
			// Numeric segment - use bracket notation
			b.WriteString("[")
			b.WriteString(s)
			b.WriteString("]")
		} else {
			// Non-numeric segment - use dot notation
			// But only add dot if we already have content (i.e., not the first segment without root)
			if b.Len() > 0 {
				b.WriteString(".")
			}
			b.WriteString(s)
		}
	}

	// Append the new segment
	if seg != "" && seg != "_" {
		if _, err := strconv.Atoi(seg); err == nil {
			b.WriteString("[")
			b.WriteString(seg)
			b.WriteString("]")
		} else {
			if b.Len() > 0 {
				b.WriteString(".")
			}
			b.WriteString(seg)
		}
	}

	return b.String()
}

func inferNodeType(node interface{}) string {
	if node == nil {
		return "null"
	}
	switch node.(type) {
	case map[string]interface{}:
		return "map"
	case []interface{}:
		return "list"
	case string:
		return "string"
	case bool:
		return "bool"
	case float64:
		return "double"
	case int, int64:
		return "int"
	case uint, uint64:
		return "uint"
	default:
		return "unknown"
	}
}

func normalizeFuncName(name string) string {
	n := strings.TrimSpace(name)
	if idx := strings.LastIndex(n, "."); idx >= 0 {
		n = n[idx+1:]
	}
	n = strings.TrimSuffix(strings.TrimSuffix(n, "()"), "(")
	n = strings.ToLower(strings.TrimSpace(n))
	return n
}

func categorizeFunction(name, desc string) string {
	nameLower := strings.ToLower(name)
	descLower := strings.ToLower(desc)

	if strings.Contains(descLower, "string") ||
		contains(nameLower, "string", "upper", "lower", "trim", "split", "join") {
		return "string"
	}
	if strings.Contains(descLower, "array") || strings.Contains(descLower, "list") ||
		contains(nameLower, "filter", "map", "all", "exists", "flatten", "slice") {
		return "list"
	}
	if strings.Contains(descLower, "math") ||
		contains(nameLower, "abs", "ceil", "floor", "round", "sqrt", "min", "max") {
		return "math"
	}
	if contains(nameLower, "regex", "matches") {
		return "regex"
	}
	if contains(nameLower, "base64", "encode", "decode") {
		return "encoding"
	}
	if contains(nameLower, "keys", "values", "has") {
		return "map"
	}

	return "general"
}

func contains(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

func inferReturnType(_ string, desc string) string {
	descLower := strings.ToLower(desc)

	if strings.Contains(descLower, "bool") || strings.Contains(descLower, "check") {
		return "bool"
	}
	if strings.Contains(descLower, "string") {
		return "string"
	}
	if strings.Contains(descLower, "array") || strings.Contains(descLower, "list") {
		return "list"
	}
	if strings.Contains(descLower, "map") {
		return "map"
	}
	if strings.Contains(descLower, "number") || strings.Contains(descLower, "int") {
		return "int"
	}

	return "any"
}

// enrichWithExamples adds usage examples and descriptions to function metadata.
func enrichWithExamples(functions []FunctionMetadata) {
	// Use examples from config (populated via SetFunctionExamples* from config loader).
	// If none are provided, fall back to hardcoded defaults.
	examplesDB := globalFunctionExamples
	if examplesDB == nil {
		examplesDB = getDefaultFunctionExamplesData()
	}

	// Apply examples and descriptions to functions
	for i := range functions {
		fn := &functions[i]
		if examples, ok := examplesDB[fn.Name]; ok {
			fn.Examples = examples.Examples
			// Only override description if not already set or if we have a better one from examples DB
			if fn.Description == "" || examples.Description != "" {
				fn.Description = examples.Description
			}
		}
	}
}

// getDefaultFunctionExamplesData returns the hardcoded default function examples with descriptions.
// This is used when no config is provided.
func getDefaultFunctionExamplesData() map[string]FunctionExampleData {
	return map[string]FunctionExampleData{
		// List/Array methods
		"filter": {
			Description: "Method: list.filter(x, condition). Filter array elements based on a condition.",
			Examples: []string{
				"[1,2,3,4,5].filter(x, x > 2) => [3,4,5]",
				"_.items.filter(x, x.status == 'active')",
				"_.users.filter(u, u.age >= 18)",
			},
		},
		"map": {
			Description: "Method: list.map(x, expr). Transform each array element.",
			Examples: []string{
				"[1,2,3].map(x, x * 2) => [2,4,6]",
				"_.items.map(x, x.name)",
				"_.users.map(u, u.email)",
			},
		},
		"exists": {
			Description: "Method: list.exists(x, condition). Check if any element matches.",
			Examples: []string{
				"[1,2,3].exists(x, x > 5) => false",
				"_.items.exists(x, x.id == 'target')",
				"_.users.exists(u, u.role == 'admin')",
			},
		},
		"all": {
			Description: "Method: list.all(x, condition). Check if all elements match.",
			Examples: []string{
				"[1,2,3].all(x, x > 0) => true",
				"_.items.all(x, x.valid == true)",
				"_.users.all(u, u.email != '')",
			},
		},
		"exists_one": {
			Description: "Method: list.exists_one(x, condition). Check if exactly one element matches.",
			Examples: []string{
				"[1,2,3].exists_one(x, x == 2) => true",
				"_.items.exists_one(x, x.primary)",
			},
		},
		"size": {
			Description: "Method: value.size(). Get length of string, array, or map.",
			Examples: []string{
				"[1,2,3].size() => 3",
				"_.items.size()",
				"'hello'.size() => 5",
			},
		},

		// String methods
		"contains": {
			Description: "Method: string.contains(substring). Check if string contains substring.",
			Examples: []string{
				"'hello world'.contains('world') => true",
				"_.name.contains('test')",
				"_.email.contains('@example.com')",
			},
		},
		"startsWith": {
			Description: "Method: string.startsWith(prefix). Check if string starts with prefix.",
			Examples: []string{
				"'hello'.startsWith('he') => true",
				"_.filename.startsWith('test_')",
				"_.url.startsWith('https://')",
			},
		},
		"endsWith": {
			Description: "Method: string.endsWith(suffix). Check if string ends with suffix.",
			Examples: []string{
				"'hello'.endsWith('lo') => true",
				"_.filename.endsWith('.json')",
				"_.email.endsWith('@gmail.com')",
			},
		},
		"matches": {
			Description: "Method: string.matches(pattern). Test string against regex pattern.",
			Examples: []string{
				"'test123'.matches('[a-z]+[0-9]+') => true",
				"_.email.matches('[a-zA-Z0-9]+@.+')",
			},
		},
		"split": {
			Description: "Method: string.split(delimiter). Split string into array.",
			Examples: []string{
				"'a,b,c'.split(',') => ['a','b','c']",
				"_.path.split('/')",
				"_.csv.split(',')",
			},
		},
		"replace": {
			Description: "Method: string.replace(old, new). Replace all occurrences of substring.",
			Examples: []string{
				"'hello'.replace('l', 'L') => 'heLLo'",
				"_.text.replace('old', 'new')",
			},
		},
		"substring": {
			Description: "Method: string.substring(start, end). Extract portion of string by index.",
			Examples: []string{
				"'hello'.substring(1, 4) => 'ell'",
				"_.name.substring(0, 5)",
			},
		},
		"abs": {
			Description: "Global: math.abs(number). Absolute value of a number.",
			Examples: []string{
				"math.abs(-3) => 3",
				"math.abs(_.delta)",
			},
		},
		"ceil": {
			Description: "Global: math.ceil(number). Round up to the nearest integer.",
			Examples: []string{
				"math.ceil(1.2) => 2",
				"math.ceil(_.ratio)",
			},
		},
		"floor": {
			Description: "Global: math.floor(number). Round down to the nearest integer.",
			Examples: []string{
				"math.floor(1.8) => 1",
				"math.floor(_.ratio)",
			},
		},
		"round": {
			Description: "Global: math.round(number). Round to the nearest integer.",
			Examples: []string{
				"math.round(1.5) => 2",
				"math.round(_.ratio)",
			},
		},
		"int": {
			Description: "Global: int(value). Convert value to integer (truncates decimals).",
			Examples: []string{
				"int('42') => 42",
				"int(3.14) => 3",
				"int(_.stringValue)",
			},
		},
		"double": {
			Description: "Global: double(value). Convert value to floating-point.",
			Examples: []string{
				"double('3.14') => 3.14",
				"double(42) => 42.0",
				"double(_.value)",
			},
		},
		"string": {
			Description: "Global: string(value). Convert value to string.",
			Examples: []string{
				"string(42) => '42'",
				"string(true) => 'true'",
				"string(_.number)",
			},
		},
		"bytes": {
			Description: "Global: bytes(string). Convert string to bytes.",
			Examples: []string{
				"bytes('hello')",
				"string(bytes('hello')) => 'hello'",
			},
		},

		// Map methods
		"has": {
			Description: "Global: has(obj.field). Check if a field exists.",
			Examples: []string{
				"has({\"a\": 1}.a) => true",
				"has(_.config.debug)",
				"has(_.user.email)",
			},
		},

		// Comparison and logic
		"in": {
			Description: "Operator: value in collection. Check if value exists in a collection.",
			Examples: []string{
				"'a' in ['a', 'b', 'c'] => true",
				"5 in [1,2,3,4,5] => true",
				"_.status in ['active', 'pending']",
			},
		},

		// Math
		// Date/Time (if available)
		"timestamp": {
			Description: "Global: timestamp(value). Parse timestamp from string.",
			Examples: []string{
				"timestamp('2024-01-01T00:00:00Z')",
				"timestamp(_.createdAt)",
			},
		},
		"duration": {
			Description: "Global: duration(value). Parse duration from string.",
			Examples: []string{
				"duration('1h30m')",
				"duration('5s')",
			},
		},
		// Base64 helpers (global)
		"base64.encode": {
			Description: "Global: base64.encode(bytes). Encode bytes to base64.",
			Examples: []string{
				"base64.encode(b\"hello\") => 'aGVsbG8='",
			},
		},
		"base64.decode": {
			Description: "Global: base64.decode(string). Decode base64 to bytes.",
			Examples: []string{
				"string(base64.decode('aGVsbG8=')) => 'hello'",
			},
		},
		// List helpers (method)
		"flatten": {
			Description: "Method: list.flatten(). Flatten a list of lists.",
			Examples: []string{
				"[[1,2],[3]].flatten() => [1,2,3]",
			},
		},
		"slice": {
			Description: "Method: list.slice(start, end). Slice a list by index.",
			Examples: []string{
				"[1,2,3,4].slice(1, 3) => [2,3]",
			},
		},
		// Math helpers (global)
		"math.greatest": {
			Description: "Global: math.greatest(list). Get the greatest numeric value.",
			Examples: []string{
				"math.greatest([1,2,3]) => 3",
			},
		},
		"math.least": {
			Description: "Global: math.least(list). Get the least numeric value.",
			Examples: []string{
				"math.least([1,2,3]) => 1",
			},
		},
		"math.sqrt": {
			Description: "Global: math.sqrt(value). Compute square root.",
			Examples: []string{
				"math.sqrt(9) => 3",
			},
		},
	}
}
