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

	fmt.Println("=== RenderTable without ColumnOrder (alphabetical) ===")
	fmt.Print(tui.RenderTable(data, tui.TableOptions{
		NoColor: false,
		Width:   60,
	}))

	fmt.Println()
	fmt.Println("=== RenderTable with ColumnOrder: [name, version, region] ===")
	fmt.Print(tui.RenderTable(data, tui.TableOptions{
		NoColor:     false,
		Width:       60,
		ColumnOrder: []string{"name", "version", "region"},
	}))

	fmt.Println()
	fmt.Println("=== RenderList without ColumnOrder ===")
	fmt.Print(tui.RenderList(data, tui.ListOptions{NoColor: false}))

	fmt.Println()
	fmt.Println("=== RenderList with ColumnOrder: [port, healthy, name] ===")
	fmt.Print(tui.RenderList(data, tui.ListOptions{
		NoColor:     false,
		ColumnOrder: []string{"port", "healthy", "name"},
	}))
}
