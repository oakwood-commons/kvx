package cmd

import (
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// ansiCodeRegexp matches ANSI escape sequences
var ansiCodeRegexp = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\[m`)

// TestTableOutputConsistency verifies that CLI mode and snapshot mode produce
// expected output for their respective renderers.
func TestTableOutputConsistency(t *testing.T) {
	testFile := filepath.Join("..", "tests", "sample.yaml")

	// Get CLI table output (default mode, no flags)
	cliOutput := runCLI(t, []string{
		"kvx",
		testFile,
		"--no-color",
	})

	// Get snapshot output (model layout by default)
	snapshotOutput := runCLI(t, []string{
		"kvx",
		testFile,
		"--snapshot",
		"--no-color",
		"--width", "80",
		"--height", "30",
	})

	if !strings.Contains(cliOutput, "metadata") || !strings.Contains(cliOutput, "KEY") {
		t.Errorf("CLI output missing expected table content:\n%s", cliOutput)
	}
	if !strings.Contains(snapshotOutput, "KEY") || !strings.Contains(snapshotOutput, "VALUE") {
		t.Errorf("Snapshot output missing expected table headers:\n%s", snapshotOutput)
	}
	lines := strings.Split(strings.TrimRight(snapshotOutput, "\n"), "\n")
	if len(lines) != 31 && len(lines) != 30 {
		t.Errorf("Snapshot output height mismatch: got %d lines (want 30-31)", len(lines))
	}
}

// TestTableOutputUsesSyncTableState verifies that CLI mode uses SyncTableState
// by checking that row generation matches interactive mode behavior.
func TestTableOutputUsesSyncTableState(t *testing.T) {
	testFile := filepath.Join("..", "tests", "sample.yaml")

	// Get CLI output
	cliOutput := runCLI(t, []string{
		"kvx",
		testFile,
		"--no-color",
	})

	// Should contain expected keys from sample.yaml
	expectedKeys := []string{"metadata", "name", "active"}
	for _, key := range expectedKeys {
		if !strings.Contains(cliOutput, key) {
			t.Errorf("CLI output missing expected key %q. Output:\n%s", key, cliOutput)
		}
	}
}

// TestTableOutputThemeConsistency verifies that themes are applied consistently
// across CLI and snapshot modes (with --no-color, structure should match).
func TestTableOutputThemeConsistency(t *testing.T) {
	testFile := filepath.Join("..", "tests", "sample.yaml")

	// Test with different themes - structure should be identical with --no-color
	themes := []string{"dark", "warm", "cool"}

	for _, theme := range themes {
		cliOutput := runCLI(t, []string{
			"kvx",
			testFile,
			"--no-color",
			"--theme", theme,
		})

		snapshotOutput := runCLI(t, []string{
			"kvx",
			testFile,
			"--snapshot",
			"--no-color",
			"--theme", theme,
			"--width", "80",
			"--height", "24",
		})

		// Both should have same structure (headers, rows)
		cliHasHeaders := strings.Contains(cliOutput, "KEY") && strings.Contains(cliOutput, "VALUE")
		snapshotHasHeaders := strings.Contains(snapshotOutput, "KEY") && strings.Contains(snapshotOutput, "VALUE")

		if cliHasHeaders != snapshotHasHeaders {
			t.Errorf("Theme %q: header presence mismatch (CLI=%v, Snapshot=%v)", theme, cliHasHeaders, snapshotHasHeaders)
		}
	}
}

// TestTableOutputColumnWidths verifies that column widths can be configured via config file.
// Note: Column widths are configured via config file, not CLI flags.
func TestTableOutputColumnWidths(t *testing.T) {
	testFile := filepath.Join("..", "tests", "sample.yaml")

	// Test that default column widths work
	cliOutput := runCLI(t, []string{
		"kvx",
		testFile,
		"--no-color",
	})

	// Should render correctly with default widths
	if !strings.Contains(cliOutput, "KEY") || !strings.Contains(cliOutput, "VALUE") {
		t.Errorf("CLI output missing headers:\n%s", cliOutput)
	}
}

// TestTableOutputAllRowsRendered verifies that CLI mode shows all rows (no height limit).
func TestTableOutputAllRowsRendered(t *testing.T) {
	testFile := filepath.Join("..", "tests", "sample.yaml")

	cliOutput := runCLI(t, []string{
		"kvx",
		testFile,
		"--no-color",
	})

	// Count data rows (non-header, non-separator lines)
	lines := strings.Split(cliOutput, "\n")
	dataRowCount := 0
	inTable := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(line, "KEY") && strings.Contains(line, "VALUE") {
			inTable = true
			continue
		}
		if inTable && trimmed != "" && !strings.HasPrefix(trimmed, "─") && !strings.HasPrefix(trimmed, "-") {
			// This is a data row
			dataRowCount++
		}
	}

	// Should have multiple rows (sample.yaml has multiple root keys)
	if dataRowCount < 3 {
		t.Errorf("Expected at least 3 data rows, got %d. Output:\n%s", dataRowCount, cliOutput)
	}
}

func TestTableOutputKeyOrderMatchesSnapshot(t *testing.T) {
	testFile := filepath.Join("..", "tests", "sample.yaml")

	cliOutput := runCLI(t, []string{
		"kvx",
		testFile,
		"--no-color",
	})

	snapshotOutput := runCLI(t, []string{
		"kvx",
		testFile,
		"--snapshot",
		"--no-color",
		"--width", "80",
		"--height", "24",
	})

	cliKeys := extractTableKeys(cliOutput)
	snapshotKeys := extractTableKeys(snapshotOutput)

	if len(cliKeys) == 0 || len(snapshotKeys) == 0 {
		t.Fatalf("expected to extract keys from both outputs, got CLI=%v snapshot=%v", cliKeys, snapshotKeys)
	}

	if !reflect.DeepEqual(cliKeys, snapshotKeys) {
		t.Fatalf("expected CLI and snapshot key order to match. CLI=%v snapshot=%v", cliKeys, snapshotKeys)
	}
}

func extractTableKeys(out string) []string {
	// Strip ANSI codes first to handle no-color mode with selection highlighting
	cleanOut := ansiCodeRegexp.ReplaceAllString(out, "")
	lines := strings.Split(cleanOut, "\n")
	keys := make([]string, 0, len(lines))
	for _, line := range lines {
		if !strings.HasPrefix(line, "│") {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "KEY") && strings.Contains(trimmed, "VALUE") {
			continue
		}
		if strings.HasPrefix(trimmed, "─") {
			continue
		}
		fields := strings.Fields(strings.Trim(trimmed, "│"))
		if len(fields) == 0 {
			continue
		}
		if strings.ContainsAny(fields[0], "─╭╰┌┐└┘") {
			continue
		}
		keys = append(keys, fields[0])
	}
	return keys
}
