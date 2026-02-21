package cmd

import (
	"os"
	"testing"

	"github.com/oakwood-commons/kvx/internal/ui"
)

// TestMain stubs platform actions (clipboard, browser) so that no test in the
// cmd package can accidentally trigger real side effects like opening a browser.
func TestMain(m *testing.M) {
	restore := ui.StubPlatformActions()
	code := m.Run()
	restore()
	os.Exit(code)
}
