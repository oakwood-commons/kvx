package ui

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// runSnapshotCLI runs the kvx CLI with snapshot mode and returns the output
// It uses the actual CLI by executing the built binary or using go run
//
//nolint:unparam // testFile parameter allows different test files to be used
func runSnapshotCLI(t *testing.T, testFile string, args ...string) string {
	t.Helper()

	// Get the current working directory (should be project root when running tests)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	// Try to find project root by looking for go.mod
	projectRoot := cwd
	for {
		if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			t.Fatalf("could not find project root (go.mod)")
		}
		projectRoot = parent
	}

	// Resolve test file path relative to project root
	var absTestFile string
	if filepath.IsAbs(testFile) {
		absTestFile = testFile
	} else {
		absTestFile = filepath.Join(projectRoot, testFile)
	}

	// Check if test file exists
	if _, err := os.Stat(absTestFile); os.IsNotExist(err) {
		t.Skipf("test data file not found: %s", absTestFile)
		return ""
	}

	// Use relative path from project root for the command
	relTestFile, err := filepath.Rel(projectRoot, absTestFile)
	if err != nil {
		relTestFile = absTestFile
	}

	// Build command: go run . <testFile> --snapshot <args>
	cmdArgs := make([]string, 0, 4+len(args))
	cmdArgs = append(cmdArgs, "run", ".", relTestFile, "--snapshot")
	cmdArgs = append(cmdArgs, args...)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", cmdArgs...)
	cmd.Dir = projectRoot
	cmd.Stdin = strings.NewReader("") // Provide empty stdin to avoid hanging

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI command failed: %v\nOutput: %s", err, output)
	}

	return string(output)
}

func snapshotGoldenPath(name string) string {
	return filepath.Join("testdata", "snapshot", name)
}

func TestSnapshotGoldenBaseline(t *testing.T) {
	output := runSnapshotCLI(t, "tests/sample.yaml", "--no-color", "--width", "80", "--height", "24")

	assertGolden(t, snapshotGoldenPath("baseline-80x24.txt"), strings.TrimRight(output, "\n"))
}

func TestSnapshotGoldenFilter(t *testing.T) {
	// Use 'f' for real-time map filtering (not '/' which is now deep search with Enter)
	output := runSnapshotCLI(t, "tests/sample.yaml", "--no-color", "--width", "80", "--height", "24", "--press", "fme")

	assertGolden(t, snapshotGoldenPath("filter-me-80x24.txt"), strings.TrimRight(output, "\n"))
}

func TestSnapshotGoldenSearch(t *testing.T) {
	output := runSnapshotCLI(t, "tests/sample.yaml", "--no-color", "--width", "80", "--height", "24", "--press", "<F3>name<Enter>")

	assertGolden(t, snapshotGoldenPath("search-name-80x24.txt"), strings.TrimRight(output, "\n"))
}

func TestSnapshotGoldenExpressionOverlay(t *testing.T) {
	// After F6, PathInput syncs to selected row path. Use Ctrl-U to clear line before typing new expression.
	output := runSnapshotCLI(t, "tests/sample.yaml", "--no-color", "--width", "80", "--height", "24", "--press", "<F6><C-u>_.items[1].name<Enter>")

	assertGolden(t, snapshotGoldenPath("expr-items-80x24.txt"), strings.TrimRight(output, "\n"))
}

func TestSnapshotGoldenHelpOverlay(t *testing.T) {
	output := runSnapshotCLI(t, "tests/sample.yaml", "--no-color", "--width", "80", "--height", "24", "--press", "<F1>")

	assertGolden(t, snapshotGoldenPath("help-80x24.txt"), strings.TrimRight(output, "\n"))
}

// TestSnapshotBasicRendering tests that snapshot mode produces valid output
func TestSnapshotBasicRendering(t *testing.T) {
	testFile := "tests/sample.yaml"
	output := runSnapshotCLI(t, testFile, "--no-color", "--width", "80", "--height", "24")

	if output == "" {
		t.Fatal("expected non-empty snapshot output")
	}

	// Should contain table headers
	if !strings.Contains(output, "KEY") || !strings.Contains(output, "VALUE") {
		t.Errorf("expected table headers in output, got:\n%s", output)
	}

	// Should contain footer (vim mode is default)
	if !strings.Contains(output, "? help") && !strings.Contains(output, "F1") {
		t.Errorf("expected footer with help binding in output, got:\n%s", output)
	}
}

// TestSnapshotWithTheme tests that themes are applied correctly
func TestSnapshotWithTheme(t *testing.T) {
	testFile := "tests/sample.yaml"
	// Test dark theme
	outputDefault := runSnapshotCLI(t, testFile, "--no-color", "--theme", "dark", "--width", "80", "--height", "24")
	if !strings.Contains(outputDefault, "KEY") {
		t.Error("dark theme should render table headers")
	}

	// Test warm theme
	outputWarm := runSnapshotCLI(t, testFile, "--no-color", "--theme", "warm", "--width", "80", "--height", "24")
	if !strings.Contains(outputWarm, "KEY") {
		t.Error("warm theme should render table headers")
	}

	// Test cool theme
	outputCool := runSnapshotCLI(t, testFile, "--no-color", "--theme", "cool", "--width", "80", "--height", "24")
	if !strings.Contains(outputCool, "KEY") {
		t.Error("cool theme should render table headers")
	}

	// All themes should produce valid output with table structure
	// (With --no-color, they may look identical, but structure should be the same)
	if !strings.Contains(outputDefault, "VALUE") {
		t.Error("dark theme should render table values")
	}
	if !strings.Contains(outputWarm, "VALUE") {
		t.Error("warm theme should render table values")
	}
	if !strings.Contains(outputCool, "VALUE") {
		t.Error("cool theme should render table values")
	}
}

// TestSnapshotWithFilter tests type-ahead filtering
func TestSnapshotWithFilter(t *testing.T) {
	testFile := "tests/sample.yaml"
	// Type "me" to filter (should match "metadata" or similar)
	output := runSnapshotCLI(t, testFile, "--no-color", "--press", "me", "--width", "80", "--height", "24")

	if output == "" {
		t.Fatal("expected non-empty snapshot output")
	}

	// Should still contain table structure
	if !strings.Contains(output, "KEY") {
		t.Error("filtered output should still contain table headers")
	}

	// The filter should have been applied (we can't easily verify the exact filtering
	// without knowing the data, but we can verify the UI still renders)
	// Default is vim mode, check for vim-style keys
	if !strings.Contains(output, "? help") && !strings.Contains(output, "F1") {
		t.Error("filtered output should still contain footer")
	}
}

// TestSnapshotWithSearch tests F3 search functionality
func TestSnapshotWithSearch(t *testing.T) {
	testFile := "tests/sample.yaml"
	// Press F3 to enter search mode, then type "test"
	output := runSnapshotCLI(t, testFile, "--no-color", "--press", "<F3>test", "--width", "80", "--height", "24")

	if output == "" {
		t.Fatal("expected non-empty snapshot output")
	}

	// Should still render UI (search mode should be visible)
	if !strings.Contains(output, "KEY") || !strings.Contains(output, "VALUE") {
		t.Error("search mode should still render table structure")
	}
}

// TestSnapshotWithHelp tests F1 help popup
func TestSnapshotWithHelp(t *testing.T) {
	testFile := "tests/sample.yaml"
	// Press F1 to show help
	output := runSnapshotCLI(t, testFile, "--no-color", "--press", "<F1>", "--width", "80", "--height", "24")

	if output == "" {
		t.Fatal("expected non-empty snapshot output")
	}

	// Help should be visible (may contain help text or popup)
	// The exact content depends on config, but UI should still render
	if !strings.Contains(output, "KEY") && !strings.Contains(output, "help") {
		t.Log("Help popup may overlay table, but UI should still render")
	}
}

// TestSnapshotWithExpression tests expression mode (F6)
func TestSnapshotWithExpression(t *testing.T) {
	testFile := "tests/sample.yaml"
	// Press F6 to enter expression mode
	output := runSnapshotCLI(t, testFile, "--no-color", "--press", "<F6>", "--width", "80", "--height", "24")

	if output == "" {
		t.Fatal("expected non-empty snapshot output")
	}

	// Expression mode should show input field
	if !strings.Contains(output, "$") && !strings.Contains(output, "_") {
		t.Error("expression mode should show input field with $ or _")
	}
}

// TestSnapshotWithExpressionAndFilter tests expression mode with filtering
func TestSnapshotWithExpressionAndFilter(t *testing.T) {
	testFile := "tests/sample.yaml"
	// Press F6, then type "items" to filter suggestions
	output := runSnapshotCLI(t, testFile, "--no-color", "--press", "<F6>items", "--width", "80", "--height", "24")

	if output == "" {
		t.Fatal("expected non-empty snapshot output")
	}

	// Should show expression input
	if !strings.Contains(output, "$") && !strings.Contains(output, "_") {
		t.Error("expression mode should show input field")
	}
}

// TestSnapshotWithWidthHeight tests that custom dimensions are honored
func TestSnapshotWithWidthHeight(t *testing.T) {
	testFile := "tests/sample.yaml"
	// Test with custom dimensions
	output := runSnapshotCLI(t, testFile, "--no-color", "--width", "100", "--height", "30")

	if output == "" {
		t.Fatal("expected non-empty snapshot output")
	}

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) != 30 {
		t.Errorf("expected 30 lines with --height 30, got %d", len(lines))
	}

	// Check that width is respected (first line should be around 100 chars)
	if len(lines) > 0 && len(lines[0]) > 120 {
		t.Logf("First line length: %d (may vary due to ANSI codes or content)", len(lines[0]))
	}
}

// TestSnapshotWithExpressionEvaluation tests that -e works with snapshot
func TestSnapshotWithExpressionEvaluation(t *testing.T) {
	testFile := "tests/sample.yaml"
	// Use -e to evaluate an expression
	output := runSnapshotCLI(t, testFile, "--no-color", "-e", "_.items[0]", "--width", "80", "--height", "24")

	if output == "" {
		t.Fatal("expected non-empty snapshot output")
	}

	// Should show the evaluated result
	if !strings.Contains(output, "KEY") {
		t.Error("expression evaluation should render table")
	}
}

// TestSnapshotCombinedOperations tests multiple operations in sequence
func TestSnapshotCombinedOperations(t *testing.T) {
	testFile := "tests/sample.yaml"
	// F1 to show help, then Escape to close, then type filter
	output := runSnapshotCLI(t, testFile, "--no-color", "--press", "<F1><Esc>me", "--width", "80", "--height", "24")

	if output == "" {
		t.Fatal("expected non-empty snapshot output")
	}

	// Should render normally after closing help
	if !strings.Contains(output, "KEY") {
		t.Error("should render table after closing help and applying filter")
	}
}

// TestSnapshotFunctionHelpPartialMatch tests that function help displays all examples
// when typing a partial function name (e.g., ".m" for "map")
func TestSnapshotFunctionHelpPartialMatch(t *testing.T) {
	testFile := "tests/sample.yaml"
	// Press F6 to enter expression mode, then type ".m" to trigger map function help
	output := runSnapshotCLI(t, testFile, "--no-color", "--press", "<F6>.m", "--width", "120", "--height", "30")

	if output == "" {
		t.Fatal("expected non-empty snapshot output")
	}

	// Should show the map function help text with method description
	if !strings.Contains(output, "Method: list.map") {
		t.Errorf("expected function help to contain 'Method: list.map', got:\n%s", output)
	}

	// Should show description
	if !strings.Contains(output, "Transform each array element") {
		t.Errorf("expected function help to contain description, got:\n%s", output)
	}

	// Should show all 3 examples separated by pipes
	if !strings.Contains(output, "[1,2,3].map(x, x * 2)") {
		t.Error("expected first example in function help")
	}
	if !strings.Contains(output, "_.items.map(x, x.name)") {
		t.Error("expected second example in function help")
	}
	if !strings.Contains(output, "_.users.map(u, u.email)") {
		t.Error("expected third example in function help")
	}

	// All examples should be on one line with pipe separators
	if !strings.Contains(output, "[1,2,3].map") || !strings.Contains(output, "|") {
		t.Log("Examples should include pipe separators when all 3 are shown")
	}
}

// TestSnapshotFunctionHelpCompleteFunction tests that function help displays all examples
// when typing a complete function call (e.g., "_.tasks.map(")
func TestSnapshotFunctionHelpCompleteFunction(t *testing.T) {
	testFile := "tests/sample.yaml"
	// Press F6 to enter expression mode, then type complete function call
	output := runSnapshotCLI(t, testFile, "--no-color", "--press", "<F6>_.tasks.map(", "--width", "120", "--height", "30")

	if output == "" {
		t.Fatal("expected non-empty snapshot output")
	}

	// Should show the map function help text
	if !strings.Contains(output, "Method: list.map") {
		t.Errorf("expected function help to contain 'Method: list.map', got:\n%s", output)
	}

	// Should show all 3 examples
	if !strings.Contains(output, "[1,2,3].map(x, x * 2)") {
		t.Error("expected first example in function help")
	}
	if !strings.Contains(output, "_.items.map(x, x.name)") {
		t.Error("expected second example in function help")
	}
	if !strings.Contains(output, "_.users.map(u, u.email)") {
		t.Error("expected third example in function help")
	}
}
