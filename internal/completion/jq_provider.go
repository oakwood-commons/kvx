package completion

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// JQProvider implements Provider for a minimal subset of jq-like completions.
// This is a lightweight, modular example showing how another expression
// language could be plugged into the intellisense engine. It does not execute
// jq or fully parse jq expressions; it provides basic key/index suggestions
// and a few common functions.
type JQProvider struct{}

// NewJQProvider creates a new jq completion provider (stub).
func NewJQProvider() (*JQProvider, error) { return &JQProvider{}, nil }

// DiscoverFunctions returns a small set of common jq functions for help.
func (p *JQProvider) DiscoverFunctions() []FunctionMetadata {
	return []FunctionMetadata{
		{Name: "length", Signature: "length() -> int", Description: "Return length of the current value", Category: "general", ReturnType: "int"},
		{Name: "keys", Signature: "keys() -> list", Description: "Return object keys", Category: "map", ReturnType: "list"},
		{Name: "map", Signature: "map(expr) -> list", Description: "Transform array elements", Category: "list", ReturnType: "list"},
		{Name: "select", Signature: "select(cond) -> any", Description: "Filter by condition", Category: "list", ReturnType: "any"},
	}
}

// FilterCompletions returns completions based on current input and context using jq-style paths (root '.')
func (p *JQProvider) FilterCompletions(input string, context CompletionContext) []Completion {
	in := strings.TrimSpace(input)
	if in == "" {
		in = "."
	}

	// Determine partial token
	partial := in
	base := ""
	if idx := strings.LastIndex(in, "."); idx >= 0 {
		partial = in[idx+1:]
		base = in[:idx]
	}

	completions := []Completion{}

	// Suggest keys/indices from current node
	switch n := context.CurrentNode.(type) {
	case map[string]interface{}:
		for k := range n {
			if partial == "" || strings.HasPrefix(strings.ToLower(k), strings.ToLower(partial)) {
				display := k
				text := base
				if text == "" {
					text = "."
				}
				// jq concatenation uses .key
				if !strings.HasSuffix(text, ".") {
					text += "."
				}
				text += k
				completions = append(completions, Completion{Text: text, Display: display, Kind: CompletionField, Detail: fmt.Sprintf("field: %s", k), Score: 100})
			}
		}
	case []interface{}:
		for i := range n {
			s := strconv.Itoa(i)
			if partial == "" || strings.HasPrefix(s, partial) {
				text := base
				if text == "" {
					text = "."
				}
				// jq indices typically .[i]
				if strings.HasSuffix(text, ".") {
					text += "[" + s + "]"
				} else {
					text += ".[" + s + "]"
				}
				completions = append(completions, Completion{Text: text, Display: "[" + s + "]", Kind: CompletionIndex, Detail: "index", Score: 100})
			}
		}
	}

	// Suggest a minimal set of jq functions
	fnList := []FunctionMetadata{
		{Name: "length"}, {Name: "keys"}, {Name: "map"}, {Name: "select"},
	}
	partLower := strings.ToLower(partial)
	for _, meta := range fnList {
		if partLower == "" || strings.HasPrefix(strings.ToLower(meta.Name), partLower) {
			display := meta.Name + "()"
			completions = append(completions, Completion{Text: meta.Name, Display: display, Kind: CompletionFunction, Detail: meta.Signature, Description: meta.Description, Score: 50, Function: &meta})
		}
	}

	sort.Slice(completions, func(i, j int) bool {
		if completions[i].Score != completions[j].Score {
			return completions[i].Score > completions[j].Score
		}
		return completions[i].Display < completions[j].Display
	})

	return completions
}

// EvaluateType is not implemented for jq in this stub; returns empty.
func (p *JQProvider) EvaluateType(_ string, _ CompletionContext) string { return "" }

// Evaluate is not implemented for jq in this stub.
func (p *JQProvider) Evaluate(_ string, _ interface{}) (interface{}, error) {
	return nil, fmt.Errorf("jq evaluate not implemented")
}

// IsExpression performs a basic heuristic for jq syntax.
func (p *JQProvider) IsExpression(expr string) bool {
	e := strings.TrimSpace(expr)
	if e == "" {
		return false
	}
	return strings.HasPrefix(e, ".") || strings.Contains(e, "|")
}
