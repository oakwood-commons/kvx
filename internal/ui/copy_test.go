//nolint:forcetypeassert
package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/oakwood-commons/kvx/internal/navigator"
)

// withCapturedClipboard swaps copyToClipboardFn for the duration of a test and
// returns a pointer to a string that records the most recent copied payload.
func withCapturedClipboard(t *testing.T) *string {
	t.Helper()
	var captured string
	orig := copyToClipboardFn
	copyToClipboardFn = func(s string) error {
		captured = s
		return nil
	}
	t.Cleanup(func() { copyToClipboardFn = orig })
	return &captured
}

func newCopyTestModel(t *testing.T, node interface{}) *Model {
	t.Helper()
	m := InitialModel(node)
	m.Root = node
	m.KeyMode = KeyModeVim
	m.InputFocused = false
	m.WinWidth = 80
	m.WinHeight = 24
	m.Tbl.Focus()
	m.applyLayout(true)
	return &m
}

// Regression: pressing "copy path" on the very first render, before any nav
// event has fired, must copy the highlighted key's path -- not the root "_".
func TestMenuActionCopy_InitialRenderCopiesHighlightedPath(t *testing.T) {
	captured := withCapturedClipboard(t)
	m := newCopyTestModel(t, map[string]any{
		"env":     map[string]any{"TASK_USE_BASH": true},
		"version": "3",
	})

	menuActionCopy(m)

	// Cursor starts at row 0. NodeToRows preserves map insertion order in tests,
	// but any highlighted key is acceptable -- what matters is that the copied
	// value is not the bare root.
	if *captured == "_" || *captured == "" {
		t.Fatalf("expected highlighted path to be copied, got %q", *captured)
	}
	if !strings.HasPrefix(*captured, "_.") {
		t.Errorf("expected copied path to start with %q, got %q", "_.", *captured)
	}
}

func TestMenuActionCopy_AfterNavigationCopiesNewPath(t *testing.T) {
	captured := withCapturedClipboard(t)
	m := newCopyTestModel(t, map[string]any{
		"alpha": 1,
		"beta":  2,
	})

	// Simulate 'j' (down) so cursor moves to row 1.
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	m2 := updated.(*Model)
	menuActionCopy(m2)

	if *captured == "_" || *captured == "" {
		t.Fatalf("expected a path to be copied after navigation, got %q", *captured)
	}
	if !strings.HasPrefix(*captured, "_.") {
		t.Errorf("expected copied path to start with %q, got %q", "_.", *captured)
	}
}

func TestMenuActionCopy_InputFocusedPreservesTypedExpression(t *testing.T) {
	captured := withCapturedClipboard(t)
	m := newCopyTestModel(t, map[string]any{"a": 1})
	m.InputFocused = true
	m.PathInput.SetValue("_.items.filter(x, x > 1)")

	menuActionCopy(m)

	// The typed expression should be preserved verbatim (may be wrapped for CLI safety).
	if !strings.Contains(*captured, "_.items.filter(x, x > 1)") {
		t.Errorf("expected typed expression to be copied, got %q", *captured)
	}
}

func TestMenuActionCopyValue_StringScalarCopiedRaw(t *testing.T) {
	captured := withCapturedClipboard(t)
	// Terminal scalar view: Node is the scalar, cursor is on the (value) row.
	root := map[string]any{"name": "kvx"}
	m := newCopyTestModel(t, root)
	m.Node = "hello world"
	m.Path = "_.name"
	// Rebuild rows so the scalar view is active (single "(value)" row).
	m.AllRows = styleRows(navigator.NodeToRows(m.Node))
	m.AllRowKeys = extractRowKeys(navigator.NodeToRows(m.Node))
	m.Tbl.SetRows(m.AllRows)
	m.applyLayout(true)

	menuActionCopyValue(m)

	if *captured != "hello world" {
		t.Errorf("expected raw string %q, got %q", "hello world", *captured)
	}
	if !strings.HasPrefix(m.ErrMsg, "Copied value:") {
		t.Errorf("expected scalar status message, got %q", m.ErrMsg)
	}
}

func TestMenuActionCopyValue_BoolScalarCopied(t *testing.T) {
	captured := withCapturedClipboard(t)
	root := map[string]any{"flag": true}
	m := newCopyTestModel(t, root)
	m.Node = true
	m.Path = "_.flag"
	m.AllRows = styleRows(navigator.NodeToRows(m.Node))
	m.AllRowKeys = extractRowKeys(navigator.NodeToRows(m.Node))
	m.Tbl.SetRows(m.AllRows)
	m.applyLayout(true)

	menuActionCopyValue(m)

	if *captured != "true" {
		t.Errorf("expected %q, got %q", "true", *captured)
	}
}

func TestMenuActionCopyValue_NumericScalarCopied(t *testing.T) {
	captured := withCapturedClipboard(t)
	root := map[string]any{"n": 42}
	m := newCopyTestModel(t, root)
	m.Node = 42
	m.Path = "_.n"
	m.AllRows = styleRows(navigator.NodeToRows(m.Node))
	m.AllRowKeys = extractRowKeys(navigator.NodeToRows(m.Node))
	m.Tbl.SetRows(m.AllRows)
	m.applyLayout(true)

	menuActionCopyValue(m)

	if *captured != "42" {
		t.Errorf("expected %q, got %q", "42", *captured)
	}
}

func TestMenuActionCopyValue_NilScalarCopiedAsNull(t *testing.T) {
	captured := withCapturedClipboard(t)
	root := map[string]any{"x": nil}
	m := newCopyTestModel(t, root)
	m.Node = nil
	m.Path = "_.x"
	m.AllRows = styleRows(navigator.NodeToRows(m.Node))
	m.AllRowKeys = extractRowKeys(navigator.NodeToRows(m.Node))
	m.Tbl.SetRows(m.AllRows)
	m.applyLayout(true)

	menuActionCopyValue(m)

	if *captured != "null" {
		t.Errorf("expected %q for nil, got %q", "null", *captured)
	}
	if !strings.Contains(m.ErrMsg, "Copied value: null") {
		t.Errorf("expected status to match payload, got %q", m.ErrMsg)
	}
}

func TestMenuActionCopyValue_MapNodeCopiedAsJSON(t *testing.T) {
	captured := withCapturedClipboard(t)
	// Cursor on a map row at the parent view: value copy resolves that row
	// through navigator.Resolve and returns the underlying map, which we
	// serialize as pretty JSON.
	root := map[string]any{
		"env": map[string]any{"TASK_USE_BASH": "true"},
	}
	m := newCopyTestModel(t, root)

	menuActionCopyValue(m)

	if !strings.Contains(*captured, "TASK_USE_BASH") {
		t.Errorf("expected copied JSON to include key, got %q", *captured)
	}
	if !strings.Contains(*captured, "\n") {
		t.Errorf("expected pretty-printed JSON with newlines, got %q", *captured)
	}
	if !strings.HasPrefix(m.ErrMsg, "Copied value (JSON,") {
		t.Errorf("expected JSON status message, got %q", m.ErrMsg)
	}
}

func TestMenuActionCopyValue_InputFocusedEvaluatesExpression(t *testing.T) {
	captured := withCapturedClipboard(t)
	root := map[string]any{"n": 7}
	m := newCopyTestModel(t, root)
	m.InputFocused = true
	m.PathInput.SetValue("_.n")

	menuActionCopyValue(m)

	if *captured != "7" {
		t.Errorf("expected evaluated result %q, got %q", "7", *captured)
	}
}

// The 'Y' binding must be recognized in vim mode.
func TestHandleVimKey_YCopiesValue(t *testing.T) {
	m := testKeyModeModel(KeyModeVim)
	if got := m.handleVimKey("Y"); got != VimActionCopyValue {
		t.Errorf("Y should map to VimActionCopyValue, got %v", got)
	}
}

// withFailingClipboard swaps copyToClipboardFn to always return an error.
func withFailingClipboard(t *testing.T) {
	t.Helper()
	orig := copyToClipboardFn
	copyToClipboardFn = func(string) error { return errStubClipboard }
	t.Cleanup(func() { copyToClipboardFn = orig })
}

var errStubClipboard = &stubClipboardErr{}

type stubClipboardErr struct{}

func (*stubClipboardErr) Error() string { return "clipboard unavailable" }

func TestMenuActionCopy_ClipboardErrorSetsErrorStatus(t *testing.T) {
	withFailingClipboard(t)
	m := newCopyTestModel(t, map[string]any{"a": 1})

	menuActionCopy(m)

	if m.StatusType != "error" {
		t.Errorf("expected error status, got %q", m.StatusType)
	}
	if !strings.HasPrefix(m.ErrMsg, "Clipboard unavailable:") {
		t.Errorf("expected clipboard error message, got %q", m.ErrMsg)
	}
}

func TestMenuActionCopyValue_ClipboardErrorSetsErrorStatus(t *testing.T) {
	withFailingClipboard(t)
	m := newCopyTestModel(t, map[string]any{"a": 1})

	menuActionCopyValue(m)

	if m.StatusType != "error" {
		t.Errorf("expected error status, got %q", m.StatusType)
	}
	if !strings.HasPrefix(m.ErrMsg, "Clipboard unavailable:") {
		t.Errorf("expected clipboard error message, got %q", m.ErrMsg)
	}
}

func TestMenuActionCopyValue_ResolveErrorSetsErrorStatus(t *testing.T) {
	captured := withCapturedClipboard(t)
	m := newCopyTestModel(t, map[string]any{"a": 1})
	m.InputFocused = true
	// Invalid CEL expression -- evaluateExpression should fail.
	m.PathInput.SetValue("_.missing.nested.path")

	menuActionCopyValue(m)

	if m.StatusType != "error" {
		t.Errorf("expected error status, got %q; captured=%q", m.StatusType, *captured)
	}
	if !strings.HasPrefix(m.ErrMsg, "Copy value error:") {
		t.Errorf("expected copy value error message, got %q", m.ErrMsg)
	}
}

func TestMenuActionCopy_InputFocusedEmptyExprCopiesUnderscore(t *testing.T) {
	captured := withCapturedClipboard(t)
	m := newCopyTestModel(t, map[string]any{"a": 1})
	m.InputFocused = true
	m.PathInput.SetValue("   ")

	menuActionCopy(m)

	if *captured != "_" {
		t.Errorf("expected root %q, got %q", "_", *captured)
	}
}

func TestMenuActionCopyValue_InputFocusedEmptyExprResolvesRoot(t *testing.T) {
	captured := withCapturedClipboard(t)
	root := map[string]any{"a": 1}
	m := newCopyTestModel(t, root)
	m.InputFocused = true
	m.PathInput.SetValue("")

	menuActionCopyValue(m)

	if !strings.Contains(*captured, `"a"`) {
		t.Errorf("expected JSON containing root map, got %q", *captured)
	}
}

func TestResolveCopyValue_NoRowsReturnsRoot(t *testing.T) {
	root := map[string]any{"a": 1}
	m := newCopyTestModel(t, root)
	// Force the no-selected-row branch: clear rows and keys so selectedRowKey
	// returns not-ok, then selectedRowPath returns the empty current path.
	m.AllRows = nil
	m.AllRowKeys = nil
	m.Tbl.SetRows(nil)
	m.Path = ""

	got, err := m.resolveCopyValue()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rootMap, ok := got.(map[string]any)
	if !ok || rootMap["a"] != 1 {
		t.Errorf("expected root map with a=1, got %T (%v)", got, got)
	}
}

func TestFormatCopyValue_LongStringTruncatesInSummary(t *testing.T) {
	long := strings.Repeat("x", 200)
	payload, summary, isJSON := formatCopyValue(long)
	if isJSON {
		t.Error("string value should not be JSON")
	}
	if payload != long {
		t.Errorf("payload should be raw string, got len=%d", len(payload))
	}
	if !strings.HasSuffix(summary, "...") {
		t.Errorf("expected truncated summary, got %q", summary)
	}
	if len(summary) >= len(long) {
		t.Errorf("expected summary shorter than payload, got %d >= %d", len(summary), len(long))
	}
}

func TestFormatCopyValue_StringSummaryReplacesNewlines(t *testing.T) {
	payload, summary, _ := formatCopyValue("line1\nline2")
	if payload != "line1\nline2" {
		t.Errorf("payload should preserve newlines, got %q", payload)
	}
	if strings.Contains(summary, "\n") {
		t.Errorf("summary should not contain newlines, got %q", summary)
	}
}

// Defensive fallback: values that cannot be JSON-encoded (channels, funcs)
// fall through to formatter.Stringify. This should never occur in real data
// flow but the fallback keeps the action robust.
func TestFormatCopyValue_UnencodableFallsBackToStringify(t *testing.T) {
	payload, _, isJSON := formatCopyValue(func() {})
	if isJSON {
		t.Error("func value should not be JSON")
	}
	if payload == "" {
		t.Error("expected non-empty fallback payload")
	}
}

// vimCopyValue is a trivial delegator, but exercising it ensures the dispatch
// wiring from executeVimAction to the copy value action stays intact.
func TestVimCopyValue_Dispatches(t *testing.T) {
	captured := withCapturedClipboard(t)
	m := testKeyModeModel(KeyModeVim)
	_, _ = m.vimCopyValue()

	if *captured == "" {
		t.Error("expected vimCopyValue to copy something")
	}
}

func TestVimCopy_Dispatches(t *testing.T) {
	captured := withCapturedClipboard(t)
	m := testKeyModeModel(KeyModeVim)
	_, _ = m.vimCopy()

	if *captured == "" {
		t.Error("expected vimCopy to copy something")
	}
}
