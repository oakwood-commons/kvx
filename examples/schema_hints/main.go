// Example: Using JSON Schema to customize table column display
//
// This example demonstrates how to use tui.ParseSchema and ColumnHints
// to control column headers, widths, alignment, and visibility when
// rendering tables with kvx as a library.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/oakwood-commons/kvx/pkg/core"
	"github.com/oakwood-commons/kvx/pkg/tui"
)

func main() {
	// Sample data: array of user objects
	data := []map[string]any{
		{"user_id": "u001", "full_name": "Alice Smith", "score": 95, "internal_code": "X123"},
		{"user_id": "u002", "full_name": "Bob Jones", "score": 87, "internal_code": "Y456"},
		{"user_id": "u003", "full_name": "Carol White", "score": 92, "internal_code": "Z789"},
	}

	root, err := core.LoadObject(data)
	if err != nil {
		log.Fatalf("failed to load data: %v", err)
	}

	// ====================================================================
	// Option 1: Parse hints from a JSON Schema
	// ====================================================================
	schemaJSON := []byte(`{
		"type": "array",
		"items": {
			"type": "object",
			"required": ["user_id", "full_name"],
			"properties": {
				"user_id": {
					"type": "string",
					"title": "ID",
					"maxLength": 8
				},
				"full_name": {
					"type": "string",
					"title": "Name"
				},
				"score": {
					"type": "integer"
				},
				"internal_code": {
					"type": "string",
					"deprecated": true
				}
			}
		}
	}`)

	hints, err := tui.ParseSchema(schemaJSON)
	if err != nil {
		log.Fatalf("failed to parse schema: %v", err)
	}

	fmt.Fprintln(os.Stdout, "=== Table with JSON Schema hints ===")                                       //nolint:forbidigo
	fmt.Fprintln(os.Stdout, "(internal_code hidden via 'deprecated: true', headers renamed via 'title')") //nolint:forbidigo
	fmt.Fprintln(os.Stdout)                                                                               //nolint:forbidigo
	output := tui.RenderTable(root, tui.TableOptions{
		AppName:     "schema-example",
		Path:        "_",
		Bordered:    true,
		NoColor:     true,
		ColumnOrder: []string{"full_name", "user_id", "score"}, // explicit order
		ColumnHints: hints,
	})
	fmt.Fprintln(os.Stdout, output) //nolint:forbidigo

	// ====================================================================
	// Option 2: Build hints programmatically (no schema file needed)
	// ====================================================================
	programmaticHints := map[string]tui.ColumnHint{
		"user_id":       {DisplayName: "User", MaxWidth: 6},
		"full_name":     {DisplayName: "Full Name"},
		"score":         {Align: "right"}, // right-align numeric column
		"internal_code": {Hidden: true},   // hide this column
	}

	fmt.Fprintln(os.Stdout, "=== Table with programmatic hints ===")                  //nolint:forbidigo
	fmt.Fprintln(os.Stdout, "(same result, built in Go code instead of JSON Schema)") //nolint:forbidigo
	fmt.Fprintln(os.Stdout)                                                           //nolint:forbidigo
	output2 := tui.RenderTable(root, tui.TableOptions{
		AppName:     "programmatic",
		Path:        "_",
		Bordered:    true,
		NoColor:     true,
		ColumnOrder: []string{"full_name", "user_id", "score"},
		ColumnHints: programmaticHints,
	})
	fmt.Fprintln(os.Stdout, output2) //nolint:forbidigo

	// ====================================================================
	// Without any hints (default behavior)
	// ====================================================================
	fmt.Fprintln(os.Stdout, "=== Table without hints (default) ===")   //nolint:forbidigo
	fmt.Fprintln(os.Stdout, "(all columns shown with original names)") //nolint:forbidigo
	fmt.Fprintln(os.Stdout)                                            //nolint:forbidigo
	output3 := tui.RenderTable(root, tui.TableOptions{
		AppName:  "default",
		Path:     "_",
		Bordered: true,
		NoColor:  true,
	})
	fmt.Fprintln(os.Stdout, output3) //nolint:forbidigo
}
