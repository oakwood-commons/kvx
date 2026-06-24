package main

import (
	"fmt"

	"github.com/oakwood-commons/kvx/pkg/tui"
)

func main() {
	data := map[string]any{
		"name":    "my-service",
		"version": "2.1.0",
		"healthy": true,
		"region":  "us-east-1",
		"port":    8080,
	}

	fmt.Println("=== RenderTable without ColumnOrder (alphabetical) ===") //nolint:forbidigo
	fmt.Print(tui.RenderTable(data, tui.TableOptions{                     //nolint:forbidigo
		NoColor: false,
		Width:   60,
	}))

	fmt.Println()                                                                //nolint:forbidigo
	fmt.Println("=== RenderTable with ColumnOrder: [name, version, region] ===") //nolint:forbidigo
	fmt.Print(tui.RenderTable(data, tui.TableOptions{                            //nolint:forbidigo
		NoColor:     false,
		Width:       60,
		ColumnOrder: []string{"name", "version", "region"},
	}))

	fmt.Println()                                                    //nolint:forbidigo
	fmt.Println("=== RenderList without ColumnOrder ===")            //nolint:forbidigo
	fmt.Print(tui.RenderList(data, tui.ListOptions{NoColor: false})) //nolint:forbidigo

	fmt.Println()                                                             //nolint:forbidigo
	fmt.Println("=== RenderList with ColumnOrder: [port, healthy, name] ===") //nolint:forbidigo
	fmt.Print(tui.RenderList(data, tui.ListOptions{                           //nolint:forbidigo
		NoColor:     false,
		ColumnOrder: []string{"port", "healthy", "name"},
	}))
}
