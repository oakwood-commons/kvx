package main

import (
	"fmt"
	"os"

	"github.com/oakwood-commons/kvx/pkg/core"
	"github.com/oakwood-commons/kvx/pkg/tui"
)

// This example demonstrates using kvx as a library with custom structs
// (like catalog.ArtifactListItem from the user's issue)

type ArtifactListItem struct {
	Name     string
	Version  string
	Sha      string
	Date     string
	Location string
}

func main() {
	// Create a custom artifact struct
	artifact := &ArtifactListItem{
		Name:     "complex-workflow",
		Version:  "1.0.0",
		Sha:      "d4bf894098c263b615514f08d2ecb92bdd854a3b27c588adf165fbbd294cd517",
		Date:     "2026-02-07 15:45:22",
		Location: "local",
	}

	// Load the custom struct using kvx's loader
	root, err := core.LoadObject(artifact)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load artifact: %v\n", err)
		os.Exit(1)
	}

	// Create an engine for evaluation
	engine, err := core.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init evaluator: %v\n", err)
		os.Exit(1)
	}

	// Before the fix, this would fail with:
	// "CEL evaluation error: eval error: unsupported conversion to ref.Val"
	// Now it works because custom structs are converted to maps for CEL compatibility
	fmt.Println("🔍 Original Artifact:")                //nolint:forbidigo
	fmt.Println(engine.RenderTable(root, true, 25, 0)) //nolint:forbidigo

	// Evaluate CEL expressions
	tests := []struct {
		name string
		expr string
	}{
		{"Get Name", "_.Name"},
		{"Get Version", "_.Version"},
		{"Check if Version is 1.0.0", "_.Version == '1.0.0'"},
		{"SHA starts with 'd4'", "_.Sha.startsWith('d4')"},
	}

	fmt.Println("\n📊 CEL Evaluation Examples:") //nolint:forbidigo
	for _, test := range tests {
		result, err := engine.Evaluate(test.expr, root)
		if err != nil {
			fmt.Printf("  ❌ %s: error evaluating %q: %v\n", test.name, test.expr, err) //nolint:forbidigo
		} else {
			fmt.Printf("  ✓ %s: %v\n", test.name, result) //nolint:forbidigo
		}
	}

	// Now demonstrate with a slice of artifacts
	artifacts := []any{
		&ArtifactListItem{
			Name:     "backend-service",
			Version:  "2.1.0",
			Sha:      "abc123def456",
			Date:     "2026-02-06",
			Location: "registry",
		},
		&ArtifactListItem{
			Name:     "frontend-app",
			Version:  "1.5.0",
			Sha:      "xyz789uvw012",
			Date:     "2026-02-05",
			Location: "registry",
		},
	}

	// Load the slice
	artifactList, err := core.LoadObject(artifacts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load artifacts: %v\n", err)
		os.Exit(1)
	}

	// Use tui.RenderTable for arrays — auto-detects columnar mode for
	// homogeneous arrays of objects, rendering field names as column headers.
	fmt.Println("\n📦 Multiple Artifacts:")                    //nolint:forbidigo
	fmt.Print(tui.RenderTable(artifactList, tui.TableOptions{ //nolint:forbidigo
		Bordered: true,
		AppName:  "artifacts",
		Path:     "_",
		NoColor:  true,
	}))

	// Filter using CEL, then render the filtered results as a columnar table.
	fmt.Println("\n🔎 Filter artifacts with Version >= '2.0.0':") //nolint:forbidigo
	filtered, err := engine.Evaluate(
		"_.filter(a, a.Version >= '2.0.0')",
		artifactList,
	)
	if err != nil {
		fmt.Printf("  Error: %v\n", err) //nolint:forbidigo
	} else {
		fmt.Print(tui.RenderTable(filtered, tui.TableOptions{ //nolint:forbidigo
			Bordered: true,
			AppName:  "filtered",
			Path:     `_.filter(a, a.Version >= "2.0.0")`,
			NoColor:  true,
		}))
	}
}
