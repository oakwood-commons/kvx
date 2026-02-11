package cel

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	celext "github.com/google/cel-go/ext"
)

// Evaluator compiles and evaluates CEL expressions.
type Evaluator struct {
	env *cel.Env
}

// NewEvaluator creates a new CEL evaluator with standard library functions.
func NewEvaluator() (*Evaluator, error) {
	env, err := newStandardCELEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}
	return &Evaluator{env: env}, nil
}

// GetEnvironment returns the CEL environment for introspection
func (e *Evaluator) GetEnvironment() *cel.Env {
	return e.env
}

// newStandardCELEnv creates a standard CEL environment with common extensions.
// Additional options can be provided to extend the environment (e.g., custom functions).
func newStandardCELEnv(opts ...cel.EnvOption) (*cel.Env, error) {
	allOpts := make([]cel.EnvOption, 0, 5+len(opts))
	allOpts = append(allOpts,
		cel.Variable("_", cel.DynType),
		// Enable common extension libraries so discovery surfaces richer functions
		celext.Strings(),
		celext.Encoders(),
		celext.Lists(),
		celext.Math(),
		// Note: Maps/Sets/Bytes extensions not available in our cel-go version
	)
	allOpts = append(allOpts, opts...)
	return cel.NewEnv(allOpts...)
}

// EvaluateExpressionWithEnv evaluates a CEL expression using the given environment.
// It handles compilation, program creation, evaluation, and result conversion.
// This is a shared helper used by both Evaluator and celEnvProvider for consistency.
func EvaluateExpressionWithEnv(env *cel.Env, expr string, data interface{}) (interface{}, error) {
	// Compile the expression (parse + type check)
	ast, issues := env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("compilation error: %w", issues.Err())
	}

	// Create program
	prg, err := env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("program error: %w", err)
	}

	// Evaluate with data bound to the '_' variable
	result, _, err := prg.Eval(map[string]interface{}{
		"_": data,
	})
	if err != nil {
		return nil, fmt.Errorf("eval error: %w", err)
	}

	// Convert CEL result back to Go types
	converted := ToGo(result)

	// Final fallback: if we still have a ref.Val after conversion, use Value()
	if refVal, ok := converted.(ref.Val); ok {
		if valFunc, ok := refVal.(interface{ Value() interface{} }); ok {
			converted = valFunc.Value()
		}
	}

	return converted, nil
}

// Evaluate evaluates a CEL expression against data.
// The expression can reference the data with the variable name "_".
// Example: "_.items[0]" or "_.items.filter(x, x.available == true)"
func (e *Evaluator) Evaluate(expr string, data interface{}) (interface{}, error) {
	return EvaluateExpressionWithEnv(e.env, expr, data)
}

// ToGo converts CEL types to Go native types recursively.
// Handles both CEL primitive types and collection types (List, Map).
func ToGo(val ref.Val) interface{} {
	if val == nil {
		return nil
	}

	// Try native Go types first
	switch v := val.(type) {
	case types.Bool:
		return bool(v)
	case types.Int:
		return int64(v)
	case types.Uint:
		return uint64(v)
	case types.Double:
		return float64(v)
	case types.String:
		return string(v)
	case types.Bytes:
		return []byte(v)
	}

	// Handle CEL collections - try to extract using Value() method
	if valuer, ok := val.(interface{ Value() interface{} }); ok {
		innerVal := valuer.Value()

		// If Value() returns a slice of ref.Val, recursively convert elements
		if refSlice, ok := innerVal.([]ref.Val); ok {
			result := make([]interface{}, len(refSlice))
			for i, elem := range refSlice {
				result[i] = ToGo(elem)
			}
			return result
		}

		// If Value() returns a slice, recursively convert elements
		if slice, ok := innerVal.([]interface{}); ok {
			result := make([]interface{}, len(slice))
			for i, elem := range slice {
				// Recursively convert each element
				if refVal, ok := elem.(ref.Val); ok {
					result[i] = ToGo(refVal)
				} else if elemMap, ok := elem.(map[string]interface{}); ok {
					// Recursively convert map elements
					result[i] = convertMapValues(elemMap)
				} else {
					result[i] = elem
				}
			}
			return result
		}

		// If Value() returns a map[string]interface{}, recursively convert values
		if m, ok := innerVal.(map[string]interface{}); ok {
			return convertMapValues(m)
		}

		// If Value() returns a map[ref.Val]ref.Val (CEL map literal), convert both keys and values
		if m, ok := innerVal.(map[ref.Val]ref.Val); ok {
			result := make(map[string]interface{})
			for k, v := range m {
				// Convert key to string
				keyStr := ""
				if keyVal, ok := k.(interface{ Value() interface{} }); ok {
					keyStr = fmt.Sprintf("%v", keyVal.Value())
				} else {
					keyStr = fmt.Sprintf("%v", k)
				}
				// Recursively convert value
				result[keyStr] = ToGo(v)
			}
			return result
		}

		// Return the inner value as-is
		return innerVal
	}

	// As last resort, return the CEL value directly
	return val
}

// convertMapValues recursively converts map values from CEL types
func convertMapValues(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		if refVal, ok := v.(ref.Val); ok {
			result[k] = ToGo(refVal)
		} else if innerMap, ok := v.(map[string]interface{}); ok {
			result[k] = convertMapValues(innerMap)
		} else if slice, ok := v.([]interface{}); ok {
			converted := make([]interface{}, len(slice))
			for i, elem := range slice {
				if refVal, ok := elem.(ref.Val); ok {
					converted[i] = ToGo(refVal)
				} else {
					converted[i] = elem
				}
			}
			result[k] = converted
		} else {
			result[k] = v
		}
	}
	return result
}

// IsCELExpression detects if a string contains CEL operators or functions.
func IsCELExpression(expr string) bool {
	// Check for brackets (array indexing)
	if contains(expr, "[") && contains(expr, "]") {
		return true
	}
	// Check for CEL function calls like map, filter, etc.
	celFunctions := []string{"map", "filter", "all", "any", "exists", "exists_one", "dyn"}
	for _, fn := range celFunctions {
		if matched, _ := regexp.MatchString(`\b`+fn+`\s*\(`, expr); matched {
			return true
		}
	}
	// Check for CEL operators
	celOps := []string{"==", "!=", "<=", ">=", "<", ">", "&&", "||", "!"}
	for _, op := range celOps {
		if contains(expr, op) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && regexp.MustCompile(regexp.QuoteMeta(substr)).MatchString(s)
}

// ParseCEL breaks down a CEL expression into navigation steps.
// For simple paths like "a.b[0].c", returns the steps.
// For complex expressions, returns the full expression wrapped for CEL evaluation.
func ParseCEL(expr string) ([]string, error) {
	var steps []string

	// Use regex to split by dots and brackets
	re := regexp.MustCompile(`([a-zA-Z0-9_]+|\[([^\]]+)\])`)
	matches := re.FindAllStringSubmatch(expr, -1)

	if len(matches) == 0 {
		return nil, fmt.Errorf("invalid expression: %s", expr)
	}

	for _, match := range matches {
		if match[2] != "" {
			steps = append(steps, match[2])
		} else {
			steps = append(steps, match[1])
		}
	}

	return steps, nil
}

// ExtractPathAndIndex parses expressions like "regions.asia.countries[0]"
// Returns (path, index, error)
//
// Deprecated: Use ParseCEL instead for better support of chained expressions.
func ExtractPathAndIndex(expr string) (string, string, error) {
	// Find the last [ and ]
	openIdx := -1
	closeIdx := -1
	for i := len(expr) - 1; i >= 0; i-- {
		if expr[i] == ']' && closeIdx == -1 {
			closeIdx = i
		} else if expr[i] == '[' && openIdx == -1 {
			openIdx = i
			break
		}
	}

	if openIdx == -1 || closeIdx == -1 || closeIdx < openIdx {
		return "", "", fmt.Errorf("invalid CEL syntax: mismatched brackets")
	}

	path := expr[:openIdx]
	index := expr[openIdx+1 : closeIdx]

	if path == "" {
		return "", "", fmt.Errorf("invalid CEL syntax: empty path before bracket")
	}
	if index == "" {
		return "", "", fmt.Errorf("invalid CEL syntax: empty index")
	}

	return path, index, nil
}

// GetAvailableFunctions returns a list of CEL functions discovered from the environment.
// Functions are returned in "name() - description" format.
func GetAvailableFunctions() []string {
	funcs, err := DiscoverCELFunctions()
	if err != nil || len(funcs) == 0 {
		return nil
	}
	return funcs
}

// exampleHints holds example usage hints set from config at startup.
// These hints are used in suggestions to help users understand how to use functions.
var exampleHints map[string]string

// SetExampleHints sets the example hints used by DiscoverCELFunctionDocs.
// These should be derived from the config file's function_examples section.
func SetExampleHints(hints map[string]string) {
	exampleHints = hints
}

// GetExampleHints returns the current example hints, or nil if none have been set.
func GetExampleHints() map[string]string {
	return exampleHints
}

// DiscoverCELFunctions builds a CEL environment and returns discovered function names.
// This mirrors the actual CEL environment so newly added functions automatically surface
// in the UI suggestions without manual updates.
func DiscoverCELFunctions() ([]string, error) {
	env, err := newStandardCELEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	// Collect functions and macros, skipping operator-style internals.
	seen := make(map[string]bool)
	for _, fn := range env.Functions() {
		if isOperator(fn.Name()) {
			continue
		}
		seen[fn.Name()] = true
	}
	for _, m := range env.Macros() {
		if isOperator(m.Function()) {
			continue
		}
		seen[m.Function()] = true
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)

	// Format for UI display (function() - CEL function)
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = n + "() - CEL function"
	}
	return out, nil
}

// isOperator filters out internal operator-style declarations that shouldn't be shown in UI.
func isOperator(name string) bool {
	// Filter out internal macros and operator-style names.
	if strings.HasPrefix(name, "@") {
		return true
	}
	if strings.HasPrefix(name, "_") && strings.HasSuffix(name, "_") {
		return true
	}
	operators := map[string]bool{
		"!_": true, "-_": true, "@in": true,
		"_!=_": true, "_%_": true, "_&&_": true,
		"_*_": true, "_+_": true, "_-_": true,
		"_/_": true, "_<=_": true, "_<_": true,
		"_==_": true, "_>=_": true, "_>_": true,
		"_?_:_": true, "_[_]": true, "_||_": true,
		"_in_": true,
	}
	return operators[name]
}

// GetCommonPatterns returns example CEL patterns
func GetCommonPatterns() []string {
	return []string{
		"filter(x, x.field == value)",
		"map(x, x.property)",
		"filter(x, x.available == true)",
		"size() > 0",
		"has(field)",
	}
}

// DiscoverCELFunctionDocs returns function suggestions with usage hints (method vs global).
// Hints are loaded from config via SetExampleHints; if none are set, suggestions omit hints.
// The returned strings keep the bare name up front for insertion, and append usage after " - ".
func DiscoverCELFunctionDocs() ([]string, error) {
	env, err := newStandardCELEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	return DiscoverFunctionsFromEnv(env, exampleHints), nil
}
func typeLabel(t *types.Type) string {
	if t == nil {
		return "any"
	}
	if name := t.DeclaredTypeName(); name != "" {
		return name
	}
	if name := t.TypeName(); name != "" {
		return name
	}
	return "any"
}

func formatParams(params []*types.Type) string {
	if len(params) == 0 {
		return ""
	}
	parts := make([]string, len(params))
	for i, p := range params {
		parts[i] = typeLabel(p)
	}
	return strings.Join(parts, ", ")
}

func appendResult(call string, result *types.Type) string {
	if result == nil {
		return call
	}
	return call + " -> " + typeLabel(result)
}

// usageFromOverload builds a human-readable usage string from a function overload.
func usageFromOverload(name string, o *decls.OverloadDecl) string {
	params := o.ArgTypes()
	if len(params) == 0 {
		return appendResult(name+"()", o.ResultType())
	}
	if o.IsMemberFunction() {
		recv := typeLabel(params[0])
		args := params[1:]
		return appendResult(recv+"."+name+"("+formatParams(args)+")", o.ResultType())
	}
	return appendResult(name+"("+formatParams(params)+")", o.ResultType())
}

// DiscoverFunctionsFromEnv discovers functions from the given CEL environment and returns
// formatted suggestions with usage hints (method vs global). The returned strings keep the
// bare name up front for insertion, and append usage after " - ".
//
// This function can be used with any CEL environment, including custom environments with
// extended functions. Example hints can be provided to add helpful usage examples.
func DiscoverFunctionsFromEnv(env *cel.Env, exampleHints map[string]string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, 100)

	for _, fn := range env.Functions() {
		if isOperator(fn.Name()) {
			continue
		}
		for _, o := range fn.OverloadDecls() {
			usage := usageFromOverload(fn.Name(), o)
			entry := fn.Name() + "()"
			if usage != "" {
				entry += " - " + usage
			} else {
				entry += " - CEL function"
			}
			if hint, ok := exampleHints[fn.Name()]; ok {
				entry += " | " + hint
			}
			if seen[entry] {
				continue
			}
			seen[entry] = true
			out = append(out, entry)
		}
	}

	// Include macros as global-style functions for discoverability
	for _, m := range env.Macros() {
		name := m.Function()
		if isOperator(name) {
			continue
		}
		entry := name + "() - CEL function"
		if hint, ok := exampleHints[name]; ok {
			entry = entry + " | " + hint
		}
		if seen[entry] {
			continue
		}
		seen[entry] = true
		out = append(out, entry)
	}

	sort.Strings(out)
	return out
}
