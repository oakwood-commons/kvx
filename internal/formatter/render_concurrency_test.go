package formatter

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRenderTableWith_ConcurrentDifferentMaxValueLines is a regression test
// for issue #83. Before the fix, RenderTable callers overrode the
// package-level MaxValueLines global with a set/defer-restore pattern
// (see historical pkg/tui/panel.go and internal/ui/panel_layout.go). Under
// concurrent rendering, one goroutine's setting could leak into another
// goroutine's render mid-flight, silently corrupting output (returning the
// wrong number of value lines) even with the race detector disabled.
//
// RenderTableWith takes MaxValueLines explicitly via RenderStyles and reads
// no global state, so concurrent renders with different limits must never
// observe each other's setting.
func TestRenderTableWith_ConcurrentDifferentMaxValueLines(t *testing.T) {
	multilineValue := "l1\nl2\nl3\nl4\nl5\nl6\nl7\nl8\nl9\nl10\nl11\nl12"
	node := map[string]any{"msg": multilineValue}

	const iterations = 300
	var wg sync.WaitGroup
	errs := make(chan string, iterations*2)

	render := func(maxLines, wantLines int) {
		defer wg.Done()
		styles := CurrentRenderStyles()
		styles.MaxValueLines = maxLines
		out := RenderTableWith(node, true, 10, 40, nil, styles)

		got := countValueContinuationLines(out)
		if got != wantLines {
			errs <- fmt.Sprintf("maxLines=%d: expected %d rendered value lines, got %d\noutput:\n%s", maxLines, wantLines, got, out)
		}
	}

	for i := 0; i < iterations; i++ {
		wg.Add(2)
		go render(1, 1)
		go render(9, 9)
	}
	wg.Wait()
	close(errs)

	var failures []string
	for e := range errs {
		failures = append(failures, e)
	}
	if len(failures) > 0 {
		t.Fatalf("%d/%d concurrent renders returned the wrong line count (cross-goroutine leakage):\n%s",
			len(failures), iterations*2, strings.Join(failures[:min(5, len(failures))], "\n---\n"))
	}
}

// countValueContinuationLines counts how many "l<N>" value lines appear in
// the rendered table output.
func countValueContinuationLines(out string) int {
	count := 0
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Continuation lines: padded key column is blank, value column holds "lN".
		fields := strings.Fields(trimmed)
		if len(fields) == 1 {
			if _, err := strconv.Atoi(strings.TrimPrefix(fields[0], "l")); err == nil {
				count++
				continue
			}
		}
		// First row: "msg       l1"
		if len(fields) == 2 && fields[0] == "msg" && strings.HasPrefix(fields[1], "l") {
			count++
		}
	}
	return count
}

// TestRenderStyles_RaceAcrossGoroutines hammers the legacy global
// SetTableTheme/SetMaxValueLines accessors alongside RenderTable from many
// goroutines. It does not assert on output (the legacy globals are, by
// design, shared process-wide state), only that -race finds nothing.
func TestRenderStyles_RaceAcrossGoroutines(t *testing.T) {
	node := map[string]any{"a": "line1\nline2", "b": "value"}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			SetTableTheme(TableColors{})
		}()
		go func() {
			defer wg.Done()
			SetMaxValueLines(i % 5)
		}()
		go func() {
			defer wg.Done()
			_ = RenderTable(node, true, 10, 30, nil)
		}()
	}
	wg.Wait()
}

// TestRenderTableWith_ConcurrentNoSharedMutation hammers RenderTableWith
// itself (the explicit-parameter path) from many goroutines with distinct
// RenderStyles snapshots to confirm the *With family touches no shared
// mutable state.
func TestRenderTableWith_ConcurrentNoSharedMutation(t *testing.T) {
	node := map[string]any{"a": "line1\nline2\nline3", "b": "value"}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			styles := RenderStyles{MaxValueLines: i%3 + 1}
			_ = RenderTableWith(node, true, 10, 30, nil, styles)
		}()
	}
	wg.Wait()
	assert.True(t, true, "no race/panic across concurrent RenderTableWith calls")
}
