package main

import (
	"fmt"
	"os"

	"github.com/oakwood-commons/kvx/pkg/core"
	"github.com/oakwood-commons/kvx/pkg/tui"
)

func main() {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <data-file> [expression]\n", os.Args[0])
		os.Exit(1)
	}

	root, err := core.LoadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load data: %v\n", err)
		os.Exit(1)
	}

	engine, err := core.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init evaluator: %v\n", err)
		os.Exit(1)
	}

	node := root
	if len(os.Args) == 3 {
		node, err = engine.Evaluate(os.Args[2], root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "evaluate error: %v\n", err)
			os.Exit(1)
		}
	}

	// Use tui.RenderTable for bordered output with auto-detected columnar mode.
	// For arrays of objects this renders a proper columnar table;
	// for maps it renders a KEY/VALUE table; for scalars it prints the raw value.
	fmt.Print(tui.RenderTable(node, tui.TableOptions{ //nolint:forbidigo
		Bordered: true,
		AppName:  "core-cli",
		Path:     "_",
		NoColor:  true,
	}))
}
