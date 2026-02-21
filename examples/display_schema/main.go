// Example: Display schema for rich card-list and detail views
//
// This example demonstrates the "scafctl" use case — an infra-provider
// catalog rendered with a scrollable card list and sectioned detail view
// instead of the default flat KEY/VALUE table.
//
// Run interactively:
//
//	go run ./examples/display_schema
//
// Run as a snapshot (CI-safe):
//
//	go run ./examples/display_schema --snapshot
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/oakwood-commons/kvx/pkg/core"
	"github.com/oakwood-commons/kvx/pkg/tui"
)

func main() {
	// 1. Load the provider data
	dataBytes, err := os.ReadFile("examples/display_schema/providers.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read providers.json: %v\n", err)
		os.Exit(1)
	}

	var data any
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse providers.json: %v\n", err)
		os.Exit(1)
	}

	root, err := core.LoadObject(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load data: %v\n", err)
		os.Exit(1)
	}

	// 2. Load JSON Schema with x-kvx-* extensions
	schemaBytes, err := os.ReadFile("examples/display_schema/provider_schema.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read provider_schema.json: %v\n", err)
		os.Exit(1)
	}

	hints, displaySchema, err := tui.ParseSchemaWithDisplay(schemaBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse schema: %v\n", err)
		os.Exit(1)
	}

	// 3. Show what was extracted
	fmt.Println("=== Display Schema Extracted ===") //nolint:forbidigo
	if displaySchema != nil {
		fmt.Printf("  Icon:            %s\n", displaySchema.Icon)            //nolint:forbidigo
		fmt.Printf("  CollectionTitle: %s\n", displaySchema.CollectionTitle) //nolint:forbidigo
		if displaySchema.List != nil {
			fmt.Printf("  List.TitleField: %s\n", displaySchema.List.TitleField)       //nolint:forbidigo
			fmt.Printf("  List.SubtitleField: %s\n", displaySchema.List.SubtitleField) //nolint:forbidigo
			fmt.Printf("  List.BadgeFields: %v\n", displaySchema.List.BadgeFields)     //nolint:forbidigo
		}
		if displaySchema.Detail != nil {
			fmt.Printf("  Detail.TitleField: %s\n", displaySchema.Detail.TitleField)  //nolint:forbidigo
			fmt.Printf("  Detail sections: %d\n", len(displaySchema.Detail.Sections)) //nolint:forbidigo
		}
		fmt.Println() //nolint:forbidigo
	}

	fmt.Printf("Column hints: %d fields recognized\n\n", len(hints)) //nolint:forbidigo

	// 4. Render the standard table (for comparison)
	fmt.Println("=== Standard Table Output ===") //nolint:forbidigo
	tableOut := tui.RenderTable(root, tui.TableOptions{
		AppName:     "display-schema-example",
		Path:        "_",
		Bordered:    true,
		NoColor:     true,
		ColumnHints: hints,
	})
	fmt.Println(tableOut) //nolint:forbidigo

	// 5. Launch TUI with display schema (interactive)
	//
	// To run interactively:
	//   go run ./examples/display_schema
	//
	// The TUI will show a card list with title, subtitle, and badges
	// for each provider. Press Right/Enter to see the detail view
	// with sectioned layout (inline header, description paragraph,
	// tags, details table, dependencies).
	//
	// To run as a snapshot (non-interactive):
	//   go run ./examples/display_schema --snapshot
	cfg := tui.Config{
		DisplaySchema: displaySchema,
	}
	if err := tui.Run(root, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
