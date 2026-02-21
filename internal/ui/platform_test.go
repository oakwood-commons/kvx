package ui

import (
	"os"
	"testing"
)

// TestMain stubs platform actions (clipboard, browser) so that no test in the
// ui package can accidentally trigger real side effects.
func TestMain(m *testing.M) {
	restore := StubPlatformActions()
	code := m.Run()
	restore()
	os.Exit(code)
}
