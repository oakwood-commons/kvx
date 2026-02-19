package completion

import (
	"sort"
	"strings"
)

// FunctionRegistry is the single source of truth for function metadata.
// It deduplicates functions by name and provides efficient lookup methods.
type FunctionRegistry struct {
	functions     map[string]FunctionMetadata // Deduped by name
	byCategory    map[string][]string         // category → function names
	categoryOrder []string                    // Display order
	allNames      []string                    // Sorted list of all function names
}

// categoryOrder defines the display order for function categories.
var defaultCategoryOrder = []string{
	"conversion",
	"string",
	"list",
	"map",
	"math",
	"encoding",
	"datetime",
	"regex",
	"general",
}

// NewFunctionRegistry creates an empty function registry.
func NewFunctionRegistry() *FunctionRegistry {
	return &FunctionRegistry{
		functions:     make(map[string]FunctionMetadata),
		byCategory:    make(map[string][]string),
		categoryOrder: defaultCategoryOrder,
	}
}

// LoadFromProvider populates the registry from a completion provider.
// Functions are deduplicated by name, keeping the entry with the best description.
func (r *FunctionRegistry) LoadFromProvider(p Provider) {
	if p == nil {
		return
	}
	rawFuncs := p.DiscoverFunctions()
	r.loadFunctions(rawFuncs)
}

// LoadFunctions populates the registry from a slice of function metadata.
// Functions are deduplicated by name, keeping the entry with the best description.
func (r *FunctionRegistry) LoadFunctions(funcs []FunctionMetadata) {
	r.loadFunctions(funcs)
}

// SupplementFromSuggestions adds functions discovered from a suggestion string list
// (the format returned by ExpressionProvider.DiscoverSuggestions) that are not
// already present in the registry. This bridges the ExpressionProvider path
// (used for tab-completion) with the palette so custom functions registered via
// SetExpressionProvider also appear in Ctrl+Space.
//
// Suggestion strings are expected to be of the form:
//
//	"name(args) - description"
//
// or plain function names / paths. Entries that cannot be parsed or are already
// in the registry are silently skipped.
func (r *FunctionRegistry) SupplementFromSuggestions(suggestions []string) {
	for _, s := range suggestions {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		// Parse: "name(args) - description" or "name(args)" or "name"
		sig, description, _ := strings.Cut(s, " - ")
		sig = strings.TrimSpace(sig)
		description = strings.TrimSpace(description)

		name := sig
		if idx := strings.Index(sig, "("); idx > 0 {
			name = strings.TrimSpace(sig[:idx])
		}
		if name == "" {
			continue
		}
		// Skip if already present with richer data.
		if existing, ok := r.functions[name]; ok && existing.Description != "" {
			continue
		}
		meta := FunctionMetadata{
			Name:        name,
			Signature:   sig,
			Description: description,
			Category:    categorizeFunction(name, description),
			IsMethod:    isKnownMethod(name),
			ReturnType:  inferReturnType(name, description),
			ParamTypes:  []string{},
		}
		r.addFunction(meta)
	}
}

// addFunction inserts or replaces a single function in the registry, updating
// the category index and sorted name list incrementally.
func (r *FunctionRegistry) addFunction(fn FunctionMetadata) {
	if _, exists := r.functions[fn.Name]; !exists {
		r.allNames = append(r.allNames, fn.Name)
		sort.Strings(r.allNames)
	}
	r.functions[fn.Name] = fn
	cat := fn.Category
	if cat == "" {
		cat = "general"
	}
	// Add to category list only if not already there.
	found := false
	for _, n := range r.byCategory[cat] {
		if n == fn.Name {
			found = true
			break
		}
	}
	if !found {
		r.byCategory[cat] = append(r.byCategory[cat], fn.Name)
		sort.Strings(r.byCategory[cat])
	}
}

func (r *FunctionRegistry) loadFunctions(funcs []FunctionMetadata) {
	// Clear existing data
	r.functions = make(map[string]FunctionMetadata)
	r.byCategory = make(map[string][]string)
	r.allNames = nil

	// Deduplicate by name - keep entry with best description/examples
	for i := range funcs {
		fn := funcs[i]
		existing, ok := r.functions[fn.Name]
		if !ok {
			r.functions[fn.Name] = fn
			continue
		}
		// Prefer entry with more examples, or longer description
		if len(fn.Examples) > len(existing.Examples) ||
			(len(fn.Examples) == len(existing.Examples) && len(fn.Description) > len(existing.Description)) {
			r.functions[fn.Name] = fn
		}
	}

	// Build category index and sorted name list
	for name, fn := range r.functions {
		cat := fn.Category
		if cat == "" {
			cat = "general"
		}
		r.byCategory[cat] = append(r.byCategory[cat], name)
		r.allNames = append(r.allNames, name)
	}

	// Sort names within each category
	for cat := range r.byCategory {
		sort.Strings(r.byCategory[cat])
	}
	sort.Strings(r.allNames)
}

// GetFunction returns metadata for a function by name, or nil if not found.
func (r *FunctionRegistry) GetFunction(name string) *FunctionMetadata {
	if fn, ok := r.functions[name]; ok {
		return &fn
	}
	return nil
}

// GetAll returns all functions sorted alphabetically by name.
func (r *FunctionRegistry) GetAll() []FunctionMetadata {
	result := make([]FunctionMetadata, 0, len(r.allNames))
	for _, name := range r.allNames {
		result = append(result, r.functions[name])
	}
	return result
}

// GetByCategory returns functions for a specific category, sorted alphabetically.
func (r *FunctionRegistry) GetByCategory(category string) []FunctionMetadata {
	names := r.byCategory[category]
	result := make([]FunctionMetadata, 0, len(names))
	for _, name := range names {
		if fn, ok := r.functions[name]; ok {
			result = append(result, fn)
		}
	}
	return result
}

// GetCategories returns all categories that contain functions, in display order.
func (r *FunctionRegistry) GetCategories() []string {
	result := make([]string, 0, len(r.categoryOrder))
	// First add categories in preferred order
	for _, cat := range r.categoryOrder {
		if len(r.byCategory[cat]) > 0 {
			result = append(result, cat)
		}
	}
	// Add any discovered categories not in the preferred order
	seen := make(map[string]bool, len(r.categoryOrder))
	for _, c := range r.categoryOrder {
		seen[c] = true
	}
	for cat := range r.byCategory {
		if !seen[cat] && len(r.byCategory[cat]) > 0 {
			result = append(result, cat)
		}
	}
	return result
}

// CategoryCount returns the number of functions in a category.
func (r *FunctionRegistry) CategoryCount(category string) int {
	return len(r.byCategory[category])
}

// Search returns functions matching the query (case-insensitive).
// Matches against name and description.
func (r *FunctionRegistry) Search(query string) []FunctionMetadata {
	if query == "" {
		return r.GetAll()
	}
	query = strings.ToLower(query)
	var result []FunctionMetadata
	for _, name := range r.allNames {
		fn := r.functions[name]
		if strings.Contains(strings.ToLower(fn.Name), query) ||
			strings.Contains(strings.ToLower(fn.Description), query) {
			result = append(result, fn)
		}
	}
	return result
}

// GetMethods returns only method functions (called on a value via dot notation).
func (r *FunctionRegistry) GetMethods() []FunctionMetadata {
	var result []FunctionMetadata
	for _, name := range r.allNames {
		fn := r.functions[name]
		if fn.IsMethod {
			result = append(result, fn)
		}
	}
	return result
}

// GetGlobals returns only global functions (called standalone).
func (r *FunctionRegistry) GetGlobals() []FunctionMetadata {
	var result []FunctionMetadata
	for _, name := range r.allNames {
		fn := r.functions[name]
		if !fn.IsMethod {
			result = append(result, fn)
		}
	}
	return result
}

// Size returns the total number of unique functions in the registry.
func (r *FunctionRegistry) Size() int {
	return len(r.functions)
}
