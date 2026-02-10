package completion

import (
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// ParsedExpression represents a parsed CEL expression with extracted navigation info
type ParsedExpression struct {
	// The original input expression
	Input string

	// The base expression (up to the last complete navigation step)
	BaseExpr string

	// The partial token being completed (if any)
	Partial string

	// Whether the input has a root reference (_)
	HasRoot bool

	// Whether the expression contains function calls
	HasFunctionCalls bool

	// The segments for simple navigation paths (when no functions)
	Segments []string
}

// ParseCELExpression uses the CEL parser to intelligently parse an expression
// Note: This will fail for incomplete expressions (e.g., "_.items." or "_.items[")
// which is expected during tab completion - caller should handle with string-based fallback
func ParseCELExpression(input string, env *cel.Env) (*ParsedExpression, error) {
	result := &ParsedExpression{
		Input:   input,
		HasRoot: strings.HasPrefix(strings.TrimSpace(input), "_"),
	}

	// Handle trailing dot or bracket - these indicate completion mode
	trimmed := strings.TrimSpace(input)
	endsWithDot := strings.HasSuffix(trimmed, ".")
	endsWithBracket := strings.HasSuffix(trimmed, "[")

	// Extract partial token if input doesn't end with . or [
	if !endsWithDot && !endsWithBracket && len(trimmed) > 0 {
		// Find the last . or [ to extract partial
		lastDot := strings.LastIndex(trimmed, ".")
		lastBracket := strings.LastIndex(trimmed, "[")
		lastSep := lastDot
		if lastBracket > lastDot {
			lastSep = lastBracket
		}

		if lastSep >= 0 && lastSep < len(trimmed)-1 {
			result.Partial = trimmed[lastSep+1:]
			trimmed = trimmed[:lastSep+1]
		}
	}

	// Remove trailing . or [ for parsing
	baseForParsing := strings.TrimSuffix(strings.TrimSuffix(trimmed, "."), "[")

	// Try to parse the expression - this will fail for incomplete syntax
	if baseForParsing != "" && baseForParsing != "_" {
		ast, issues := env.Parse(baseForParsing)
		if issues == nil || issues.Err() == nil {
			// Successfully parsed - analyze the AST
			parsed, _ := cel.AstToParsedExpr(ast)
			expr := parsed.GetExpr()
			result.HasFunctionCalls = expr != nil && hasFunctionCallsInProto(expr)

			// If no function calls, extract simple navigation path
			if !result.HasFunctionCalls && expr != nil {
				result.Segments = extractSegmentsFromProto(expr)
			}
		} else {
			// Parse failed (incomplete expression) - use string analysis as hint
			result.HasFunctionCalls = strings.Contains(baseForParsing, "(")
		}
	}

	// Set base expression
	switch {
	case endsWithDot || endsWithBracket:
		result.BaseExpr = strings.TrimSuffix(strings.TrimSuffix(trimmed, "."), "[")
	case result.Partial != "":
		// Ensure partial doesn't exceed trimmed length before slicing
		partialLen := len(result.Partial)
		if partialLen <= len(trimmed) {
			result.BaseExpr = strings.TrimSuffix(trimmed[:len(trimmed)-partialLen], ".")
		} else {
			// Partial is longer than input (edge case), use base for parsing
			result.BaseExpr = baseForParsing
			result.Partial = "" // Clear invalid partial
		}
	default:
		result.BaseExpr = baseForParsing
	}

	// If we don't have segments from AST, fall back to simple splitting
	if len(result.Segments) == 0 && result.BaseExpr != "" {
		result.Segments = splitPathSegments(result.BaseExpr)
	}

	return result, nil
}

// hasFunctionCallsInProto checks if expression contains function calls using proto inspection
func hasFunctionCallsInProto(expr *exprpb.Expr) bool {
	if expr == nil {
		return false
	}

	switch expr.ExprKind.(type) {
	case *exprpb.Expr_CallExpr:
		call := expr.GetCallExpr()
		if call != nil && !isOperator(call.Function) {
			return true
		}
		// Check args recursively
		for _, arg := range call.Args {
			if hasFunctionCallsInProto(arg) {
				return true
			}
		}
		if call.Target != nil && hasFunctionCallsInProto(call.Target) {
			return true
		}

	case *exprpb.Expr_SelectExpr:
		sel := expr.GetSelectExpr()
		if sel != nil && sel.Operand != nil {
			return hasFunctionCallsInProto(sel.Operand)
		}

	case *exprpb.Expr_ComprehensionExpr:
		// Comprehensions contain function-like behavior
		return true

	case *exprpb.Expr_ListExpr:
		list := expr.GetListExpr()
		for _, elem := range list.Elements {
			if hasFunctionCallsInProto(elem) {
				return true
			}
		}

	case *exprpb.Expr_StructExpr:
		str := expr.GetStructExpr()
		for _, entry := range str.Entries {
			if hasFunctionCallsInProto(entry.GetMapKey()) || hasFunctionCallsInProto(entry.GetValue()) {
				return true
			}
		}
	}

	return false
}

// extractSegmentsFromProto extracts navigation segments from proto expression
// Returns segments like ["_", "items", "0", "name"]
func extractSegmentsFromProto(expr *exprpb.Expr) []string {
	if expr == nil {
		return nil
	}

	var segments []string

	// Walk the expression tree to extract the navigation chain
	var extract func(*exprpb.Expr)
	extract = func(e *exprpb.Expr) {
		if e == nil {
			return
		}

		switch e.ExprKind.(type) {
		case *exprpb.Expr_IdentExpr:
			ident := e.GetIdentExpr()
			if ident != nil {
				segments = append([]string{ident.Name}, segments...)
			}

		case *exprpb.Expr_SelectExpr:
			sel := e.GetSelectExpr()
			if sel != nil {
				segments = append([]string{sel.Field}, segments...)
				extract(sel.Operand)
			}

		case *exprpb.Expr_CallExpr:
			call := e.GetCallExpr()
			if call != nil {
				// Handle index operator _[_]
				if call.Function == "_[_]" && len(call.Args) == 2 {
					// Get the index value if it's a constant
					if literal := call.Args[1].GetConstExpr(); literal != nil {
						val := fmt.Sprintf("%v", protoConstToValue(literal))
						segments = append([]string{val}, segments...)
						extract(call.Args[0])
						return
					}
				}
				// For other calls, extract target
				if call.Target != nil {
					extract(call.Target)
				}
			}
		}
	}

	extract(expr)
	return segments
}

// protoConstToValue converts a proto constant to a Go value
func protoConstToValue(c *exprpb.Constant) interface{} {
	if c == nil {
		return nil
	}

	switch c.ConstantKind.(type) {
	case *exprpb.Constant_BoolValue:
		return c.GetBoolValue()
	case *exprpb.Constant_Int64Value:
		return c.GetInt64Value()
	case *exprpb.Constant_Uint64Value:
		return c.GetUint64Value()
	case *exprpb.Constant_DoubleValue:
		return c.GetDoubleValue()
	case *exprpb.Constant_StringValue:
		return c.GetStringValue()
	case *exprpb.Constant_BytesValue:
		return c.GetBytesValue()
	default:
		return nil
	}
}

// isOperator checks if a function name is an operator vs a real function call
func isOperator(name string) bool {
	operators := map[string]bool{
		"_[_]":  true, // index
		"_==_":  true,
		"_!=_":  true,
		"_<_":   true,
		"_<=_":  true,
		"_>_":   true,
		"_>=_":  true,
		"_+_":   true,
		"_-_":   true,
		"_*_":   true,
		"_/_":   true,
		"_%_":   true,
		"_&&_":  true,
		"_||_":  true,
		"!_":    true,
		"-_":    true,
		"_?_:_": true, // ternary
		"_in_":  true,
		"@in":   true,
	}
	return operators[name]
}
