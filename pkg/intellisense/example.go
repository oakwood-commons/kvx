package intellisense

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Example demonstrates how to use the intellisense package in an interactive CLI application.
// This shows a simple REPL (Read-Eval-Print Loop) with completion support.
func Example() {
	// Sample data to work with
	data := map[string]interface{}{
		"users": []interface{}{
			map[string]interface{}{
				"name":   "Alice",
				"email":  "alice@example.com",
				"active": true,
				"age":    30,
			},
			map[string]interface{}{
				"name":   "Bob",
				"email":  "bob@example.com",
				"active": false,
				"age":    25,
			},
		},
		"metadata": map[string]interface{}{
			"version": "1.0",
			"count":   2,
		},
	}

	// Create CEL provider
	provider, err := NewCELProvider()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating provider: %v\n", err)
		return
	}

	fmt.Fprintln(os.Stdout, "Interactive Expression Mode (CEL)")
	fmt.Fprintln(os.Stdout, "Type expressions to evaluate. Press Ctrl+C to exit.")
	fmt.Fprintln(os.Stdout, "Examples:")
	fmt.Fprintln(os.Stdout, "  _.users")
	fmt.Fprintln(os.Stdout, "  _.users.filter(u, u.active)")
	fmt.Fprintln(os.Stdout, "  _.users.map(u, u.name)")
	fmt.Fprintln(os.Stdout)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Fprint(os.Stdout, "❯ ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Special commands
		if input == "help" {
			showHelp(provider)
			continue
		}
		if input == "functions" {
			showFunctions(provider)
			continue
		}
		if input == "exit" || input == "quit" {
			break
		}

		// Show completions if input ends with "."
		if strings.HasSuffix(input, ".") {
			showCompletions(provider, input, data)
			continue
		}

		// Evaluate expression
		result, err := provider.Evaluate(input, data)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Error: %v\n", err)
			continue
		}

		// Pretty-print result
		prettyPrint(result)
	}

	fmt.Fprintln(os.Stdout, "\nGoodbye!")
}

// showCompletions displays available completions for the current input
func showCompletions(provider Provider, input string, data interface{}) {
	ctx := CompletionContext{
		CurrentNode:  data,
		CurrentType:  "map",
		IsAfterDot:   true,
		PartialToken: "",
	}

	completions := provider.FilterCompletions(input, ctx)
	if len(completions) == 0 {
		fmt.Fprintln(os.Stdout, "No completions available")
		return
	}

	fmt.Fprintln(os.Stdout, "Available completions:")
	for i, c := range completions {
		if i >= 10 { // Limit to 10
			fmt.Fprintf(os.Stdout, "... and %d more\n", len(completions)-10)
			break
		}

		kindLabel := ""
		switch c.Kind {
		case CompletionField:
			kindLabel = "[field]"
		case CompletionFunction:
			kindLabel = "[function]"
		case CompletionIndex:
			kindLabel = "[index]"
		case CompletionKeyword:
			kindLabel = "[keyword]"
		case CompletionVariable:
			kindLabel = "[variable]"
		}

		fmt.Fprintf(os.Stdout, "  %s %-20s %s\n", kindLabel, c.Display, c.Detail)
	}
	fmt.Fprintln(os.Stdout)
}

// showFunctions displays all available functions
func showFunctions(provider Provider) {
	functions := provider.DiscoverFunctions()

	fmt.Fprintf(os.Stdout, "\nAvailable Functions (%d):\n\n", len(functions))

	// Group by category
	categories := make(map[string][]FunctionMetadata)
	for _, fn := range functions {
		cat := fn.Category
		if cat == "" {
			cat = "general"
		}
		categories[cat] = append(categories[cat], fn)
	}

	for cat, fns := range categories {
		fmt.Fprintf(os.Stdout, "=== %s ===\n", strings.ToUpper(cat))
		for _, fn := range fns {
			fmt.Fprintf(os.Stdout, "  %-30s %s\n", fn.Signature, fn.Description)
		}
		fmt.Fprintln(os.Stdout)
	}
}

// showHelp displays help information
func showHelp(_ Provider) {
	fmt.Fprintln(os.Stdout, "\nHelp:")
	fmt.Fprintln(os.Stdout, "  help        - Show this help")
	fmt.Fprintln(os.Stdout, "  functions   - List all available functions")
	fmt.Fprintln(os.Stdout, "  exit/quit   - Exit the REPL")
	fmt.Fprintln(os.Stdout, "  <expr>.     - Show completions (type a dot at the end)")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Expression Syntax:")
	fmt.Fprintln(os.Stdout, "  _           - Root data (always use _ as the base)")
	fmt.Fprintln(os.Stdout, "  _.field     - Access field")
	fmt.Fprintln(os.Stdout, "  _[0]        - Access array index")
	fmt.Fprintln(os.Stdout, "  _.field.nested - Navigate nested data")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Common Functions:")
	fmt.Fprintln(os.Stdout, "  filter(x, condition)  - Filter array elements")
	fmt.Fprintln(os.Stdout, "  map(x, expression)    - Transform array elements")
	fmt.Fprintln(os.Stdout, "  size()                - Get size of array/map/string")
	fmt.Fprintln(os.Stdout, "  contains(substring)   - Check if string contains substring")
	fmt.Fprintln(os.Stdout)
}

// prettyPrint outputs a value in a readable format
func prettyPrint(v interface{}) {
	// Try to marshal as JSON for pretty output
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stdout, "%v\n", v)
		return
	}
	fmt.Fprintln(os.Stdout, string(b))
}

// ExampleIntegration shows how to integrate intellisense into a custom application
func ExampleIntegration() {
	// 1. Create a provider
	provider, _ := NewCELProvider()

	// 2. Prepare your data
	myData := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"id": 1, "name": "Widget"},
			map[string]interface{}{"id": 2, "name": "Gadget"},
		},
	}

	// 3. Create completion context
	ctx := CompletionContext{
		CurrentNode:          myData,
		CurrentType:          "map",
		PartialToken:         "i", // User typed "_.i"
		IsAfterDot:           false,
		CursorPosition:       3,
		ExpressionResult:     nil,
		ExpressionResultType: "",
	}

	// 4. Get completions
	completions := provider.FilterCompletions("_.i", ctx)

	// 5. Display to user (in your UI)
	for _, c := range completions {
		fmt.Fprintf(os.Stdout, "Suggestion: %s (%v)\n", c.Display, c.Kind)
	}

	// 6. Evaluate when user hits Enter
	result, err := provider.Evaluate("_.items", myData)
	if err != nil {
		fmt.Fprintf(os.Stdout, "Error: %v\n", err)
	} else {
		fmt.Fprintf(os.Stdout, "Result: %v\n", result)
	}
}
