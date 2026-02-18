//nolint:forcetypeassert
package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sampleDataWithEncodedScalars returns a data structure containing serialized
// string values that can be decoded (JSON, JWT, etc.).
func sampleDataWithEncodedScalars() map[string]any {
	return map[string]any{
		"name":    "alice",
		"payload": `{"key":"value","nested":{"deep":true}}`,
		"jwt":     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
		"list":    `[1,2,3]`,
		"plain":   "hello world",
	}
}

// decodeTableModel returns a Model in table/navigation mode (not expr-focused)
// with data containing decodable scalars.
func decodeTableModel() *Model {
	node := sampleDataWithEncodedScalars()
	m := InitialModel(node)
	m.Root = node
	m.InputFocused = false
	m.KeyMode = KeyModeFunction
	m.WinWidth = 120
	m.WinHeight = 30
	m.applyLayout(true)
	m.Tbl.Focus()
	return &m
}

// pressKeyRight simulates pressing the right arrow key.
func pressKeyRight(m *Model) *Model {
	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	return result.(*Model)
}

// pressKeyLeft simulates pressing the left arrow key.
func pressKeyLeft(m *Model) *Model {
	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	return result.(*Model)
}

// pressKeyEnter simulates pressing the Enter key.
func pressKeyEnter(m *Model) *Model {
	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	return result.(*Model)
}

// moveCursorToKey positions the table cursor on the row whose key matches key.
func moveCursorToKey(t *testing.T, m *Model, key string) {
	t.Helper()
	for i, k := range m.AllRowKeys {
		if k == key {
			m.Tbl.SetCursor(i)
			return
		}
	}
	t.Fatalf("key %q not found in AllRowKeys: %v", key, m.AllRowKeys)
}

// ---------- tryDecodeAndNavigate ----------

func TestDecodeAndNavigate_JSONPayload(t *testing.T) {
	m := decodeTableModel()
	// Navigate into "payload" scalar
	moveCursorToKey(t, m, "payload")
	m = pressKeyRight(m) // enter payload → see (value)

	require.Equal(t, "_.payload", m.Path)
	// Now cursor should be on the scalar (value)
	selectedKey, ok := m.selectedRowKey()
	require.True(t, ok)
	require.Equal(t, "(value)", selectedKey)

	// Press Enter to decode; this calls tryDecodeAndNavigate
	m = pressKeyEnter(m)

	// After decode, the model should be at the same path but viewing the decoded map
	assert.Equal(t, "_.payload", m.Path)
	assert.True(t, m.DecodedActive, "DecodedActive should be true after decode")
	assert.Equal(t, "_.payload", m.DecodedPath, "DecodedPath should record where decode happened")

	// Decoded node should be a map with keys
	_, isMap := m.Node.(map[string]any)
	assert.True(t, isMap, "decoded node should be a map")
	assert.True(t, len(m.AllRowKeys) > 1, "should have multiple keys in decoded view, got: %v", m.AllRowKeys)
}

func TestDecodeAndNavigate_JWT(t *testing.T) {
	m := decodeTableModel()
	moveCursorToKey(t, m, "jwt")
	m = pressKeyRight(m) // enter jwt scalar

	require.Equal(t, "_.jwt", m.Path)
	selectedKey, _ := m.selectedRowKey()
	require.Equal(t, "(value)", selectedKey)

	m = pressKeyEnter(m) // decode

	assert.Equal(t, "_.jwt", m.Path)
	assert.True(t, m.DecodedActive)
	// JWT decodes into a map with header, payload, signature
	nodeMap, isMap := m.Node.(map[string]any)
	require.True(t, isMap, "decoded JWT should be a map")
	assert.Contains(t, nodeMap, "header")
	assert.Contains(t, nodeMap, "payload")
}

func TestDecodeAndNavigate_PlainStringNoOp(t *testing.T) {
	m := decodeTableModel()
	moveCursorToKey(t, m, "plain")
	m = pressKeyRight(m)

	require.Equal(t, "_.plain", m.Path)
	selectedKey, _ := m.selectedRowKey()
	require.Equal(t, "(value)", selectedKey)

	// Enter on a non-decodable scalar should NOT change DecodedActive
	m = pressKeyEnter(m)

	assert.False(t, m.DecodedActive, "non-decodable string should not trigger decode")
	assert.Equal(t, "_.plain", m.Path, "path should not change on failed decode")
}

func TestDecodeAndNavigate_JSONArray(t *testing.T) {
	m := decodeTableModel()
	moveCursorToKey(t, m, "list")
	m = pressKeyRight(m)

	require.Equal(t, "_.list", m.Path)
	m = pressKeyEnter(m) // decode JSON array

	assert.Equal(t, "_.list", m.Path)
	assert.True(t, m.DecodedActive)
	_, isSlice := m.Node.([]any)
	assert.True(t, isSlice, "decoded JSON array should be a slice")
}

// ---------- AllowDecode gating ----------

func TestDecodeBlockedWhenAllowDecodeFalse(t *testing.T) {
	m := decodeTableModel()
	m.AllowDecode = false

	moveCursorToKey(t, m, "payload")
	m = pressKeyRight(m) // enter payload scalar

	require.Equal(t, "_.payload", m.Path)
	m = pressKeyEnter(m) // attempt decode

	// Should NOT decode
	assert.False(t, m.DecodedActive, "decode should be blocked when AllowDecode=false")
	assert.Equal(t, "_.payload", m.Path)
	// Node should still be the string
	_, isStr := m.Node.(string)
	assert.True(t, isStr, "node should remain a string when decode is blocked")
}

// ---------- updateDecodedState / badge persistence ----------

func TestDecodedBadgePersistsInSubtree(t *testing.T) {
	m := decodeTableModel()

	// Navigate into payload and decode
	moveCursorToKey(t, m, "payload")
	m = pressKeyRight(m) // → (value)
	m = pressKeyEnter(m) // decode

	require.True(t, m.DecodedActive)
	require.Equal(t, "_.payload", m.DecodedPath)

	// Navigate deeper into decoded data (e.g., first key)
	m = pressKeyRight(m)

	// Should still be in decoded subtree
	assert.True(t, m.DecodedActive, "badge should persist when navigating deeper into decoded subtree")
	assert.True(t, strings.HasPrefix(m.Path, "_.payload"), "should still be under _.payload")
}

func TestDecodedBadgeClearsOnNavigateBack(t *testing.T) {
	m := decodeTableModel()

	// Navigate into payload and decode
	moveCursorToKey(t, m, "payload")
	m = pressKeyRight(m) // → (value)
	m = pressKeyEnter(m) // decode

	require.True(t, m.DecodedActive)
	require.Equal(t, "_.payload", m.Path)

	// Navigate back to root via left arrow
	m = pressKeyLeft(m)

	assert.Equal(t, "", m.Path, "should be back at root")
	assert.False(t, m.DecodedActive, "badge should be cleared after navigating out of decoded subtree")
	assert.Equal(t, "", m.DecodedPath, "DecodedPath should be cleared")
}

func TestDecodedBadgeClearsOnLeftFromRoot(t *testing.T) {
	// Decode JWT, navigate back to root, and verify badge gone on sibling keys
	m := decodeTableModel()

	moveCursorToKey(t, m, "jwt")
	m = pressKeyRight(m) // → (value)
	m = pressKeyEnter(m) // decode JWT

	require.True(t, m.DecodedActive)
	require.Equal(t, "_.jwt", m.Path)

	// Go back to root
	m = pressKeyLeft(m)

	assert.Equal(t, "", m.Path)
	assert.False(t, m.DecodedActive, "decoded badge should NOT show at root after leaving decoded subtree")

	// Move cursor to a different key (e.g., "name")
	moveCursorToKey(t, m, "name")
	// Force a sync to update status
	m.syncStatus()
	assert.False(t, m.DecodedActive, "badge should not appear on unrelated key")
}

func TestDecodedBadgeClearsOnVimBackNavigation(t *testing.T) {
	m := decodeTableModel()
	m.KeyMode = KeyModeVim

	moveCursorToKey(t, m, "payload")
	// vim: l to enter, then Enter to decode via (value)
	result, _ := m.Update(tea.KeyPressMsg{Code: 'l', Text: "l"})
	m = result.(*Model)
	m = pressKeyEnter(m) // decode

	require.True(t, m.DecodedActive)

	// vim: h to go back
	result, _ = m.Update(tea.KeyPressMsg{Code: 'h', Text: "h"})
	m = result.(*Model)

	assert.Equal(t, "", m.Path)
	assert.False(t, m.DecodedActive, "vim h should clear decoded badge when leaving subtree")
}

// ---------- decodeHintForSelectedRow ----------

func TestDecodeHintShowsOnValueRow(t *testing.T) {
	m := decodeTableModel()
	moveCursorToKey(t, m, "payload")
	m = pressKeyRight(m) // enter payload → (value)

	hint := m.decodeHintForSelectedRow()
	assert.Equal(t, "↵ decode", hint, "hint should show on (value) row with decodable content")
}

func TestDecodeHintNotShownOnMapKey(t *testing.T) {
	m := decodeTableModel()
	// Stay at root — cursor is on a map key like "name", not on (value)
	moveCursorToKey(t, m, "payload")

	hint := m.decodeHintForSelectedRow()
	assert.Equal(t, "", hint, "hint should NOT show on a parent map key")
}

func TestDecodeHintNotShownOnNonDecodableScalar(t *testing.T) {
	m := decodeTableModel()
	moveCursorToKey(t, m, "name")
	m = pressKeyRight(m) // enter "name" → (value) = "alice"

	hint := m.decodeHintForSelectedRow()
	assert.Equal(t, "", hint, "hint should NOT show for non-decodable string")
}

func TestDecodeHintNotShownWhenAllowDecodeFalse(t *testing.T) {
	m := decodeTableModel()
	m.AllowDecode = false

	moveCursorToKey(t, m, "payload")
	m = pressKeyRight(m) // enter payload → (value)

	hint := m.decodeHintForSelectedRow()
	assert.Equal(t, "", hint, "hint should NOT show when AllowDecode is false")
}

func TestDecodeHintNotShownInExprMode(t *testing.T) {
	m := decodeTableModel()
	moveCursorToKey(t, m, "payload")
	m = pressKeyRight(m)

	// Simulate being in expr/input mode
	m.InputFocused = true

	hint := m.decodeHintForSelectedRow()
	assert.Equal(t, "", hint, "hint should NOT show when InputFocused")
}

// ---------- Panel layout infoMessage ----------

func TestPanelLayoutInfoMessage_DecodedActive(t *testing.T) {
	m := decodeTableModel()
	moveCursorToKey(t, m, "payload")
	m = pressKeyRight(m)
	m = pressKeyEnter(m) // decode

	require.True(t, m.DecodedActive)

	state := panelLayoutStateFromModel(m, PanelLayoutModelOptions{})
	assert.Equal(t, "✓ decoded", state.InfoMessage, "info message should show ✓ decoded when inside decoded subtree")
}

func TestPanelLayoutInfoMessage_DecodeHint(t *testing.T) {
	m := decodeTableModel()
	moveCursorToKey(t, m, "payload")
	m = pressKeyRight(m) // at (value) with decodable content

	require.False(t, m.DecodedActive)

	state := panelLayoutStateFromModel(m, PanelLayoutModelOptions{})
	assert.Equal(t, "↵ decode", state.InfoMessage, "info message should show ↵ decode hint on decodable scalar")
}

func TestPanelLayoutInfoMessage_NoBadgeAfterLeavingSubtree(t *testing.T) {
	m := decodeTableModel()
	moveCursorToKey(t, m, "payload")
	m = pressKeyRight(m)
	m = pressKeyEnter(m) // decode
	m = pressKeyLeft(m)  // back to root

	require.False(t, m.DecodedActive)

	state := panelLayoutStateFromModel(m, PanelLayoutModelOptions{})
	assert.NotEqual(t, "✓ decoded", state.InfoMessage, "info message should NOT show ✓ decoded at root after leaving")
}

// ---------- AutoDecode lazy mode ----------

func TestAutoDecodeLazy_DecodesOnForwardNavigation(t *testing.T) {
	m := decodeTableModel()
	m.AutoDecode = "lazy"

	// Navigate forward into a decodable key (payload is a JSON string → map)
	moveCursorToKey(t, m, "payload")
	m = pressKeyRight(m) // navigateForward should auto-decode

	// With lazy decode, navigating to a scalar should auto-decode it
	// The node should now be structured
	if _, isStr := m.Node.(string); isStr {
		// If still string, we're at the (value) view — press enter to decode
		m = pressKeyEnter(m)
	}

	assert.True(t, m.DecodedActive, "lazy auto-decode should trigger DecodedActive")
}

// ---------- RightArrow triggers decode same as Enter ----------

func TestRightArrowTriggersDecodeOnValueRow(t *testing.T) {
	m := decodeTableModel()
	moveCursorToKey(t, m, "payload")
	m = pressKeyRight(m) // enter payload → (value)
	m = pressKeyRight(m) // right arrow on (value) should also decode

	assert.True(t, m.DecodedActive, "right arrow on (value) should trigger decode")
	_, isMap := m.Node.(map[string]any)
	assert.True(t, isMap, "decoded node should be a map")
}

// ---------- Multiple decode cycles ----------

func TestDecodeNavigateBackRedecodeWorks(t *testing.T) {
	m := decodeTableModel()

	// Decode payload
	moveCursorToKey(t, m, "payload")
	m = pressKeyRight(m)
	m = pressKeyEnter(m) // decode
	require.True(t, m.DecodedActive)

	// Go back
	m = pressKeyLeft(m)
	require.False(t, m.DecodedActive)

	// Decode jwt now
	moveCursorToKey(t, m, "jwt")
	m = pressKeyRight(m)
	m = pressKeyEnter(m) // decode jwt
	assert.True(t, m.DecodedActive)
	assert.Equal(t, "_.jwt", m.DecodedPath)

	// JWT has header, payload, signature
	nodeMap, isMap := m.Node.(map[string]any)
	require.True(t, isMap)
	assert.Contains(t, nodeMap, "header")
}

// ---------- CEL path accuracy ----------

func TestDecodedPathDoesNotContainDecodedSegment(t *testing.T) {
	m := decodeTableModel()
	moveCursorToKey(t, m, "payload")
	m = pressKeyRight(m)
	m = pressKeyEnter(m) // decode

	assert.Equal(t, "_.payload", m.Path, "path should remain clean without [decoded] segment")
	assert.NotContains(t, m.Path, "[decoded]")
	assert.NotContains(t, m.Path, "decoded")

	// Navigate deeper
	m = pressKeyRight(m) // into first child of decoded map
	assert.NotContains(t, m.Path, "[decoded]")
}

// ---------- updateDecodedState edge cases ----------

func TestUpdateDecodedState_AtDecodePointWithMap(t *testing.T) {
	m := decodeTableModel()
	m.DecodedActive = true
	m.DecodedPath = "_.payload"
	m.Path = "_.payload"
	m.Node = map[string]any{"key": "value"}

	m.updateDecodedState()

	assert.True(t, m.DecodedActive, "should remain active when at decode point with map node")
}

func TestUpdateDecodedState_AtDecodePointWithScalar(t *testing.T) {
	m := decodeTableModel()
	m.DecodedActive = true
	m.DecodedPath = "_.payload"
	m.Path = "_.payload"
	m.Node = "still a string" // Node is scalar — we're at parent level

	m.updateDecodedState()

	assert.False(t, m.DecodedActive, "should clear when at decode point but node is scalar (parent view)")
}

func TestUpdateDecodedState_DeeperInSubtree(t *testing.T) {
	m := decodeTableModel()
	m.DecodedActive = true
	m.DecodedPath = "_.payload"
	m.Path = "_.payload.nested.deep"
	m.Node = true

	m.updateDecodedState()

	assert.True(t, m.DecodedActive, "should remain active when deeper in decoded subtree")
}

func TestUpdateDecodedState_OutsideSubtree(t *testing.T) {
	m := decodeTableModel()
	m.DecodedActive = true
	m.DecodedPath = "_.payload"
	m.Path = "_.name"
	m.Node = "alice"

	m.updateDecodedState()

	assert.False(t, m.DecodedActive, "should clear when at a sibling path")
	assert.Equal(t, "", m.DecodedPath)
}

func TestUpdateDecodedState_DecodedAtRoot(t *testing.T) {
	m := decodeTableModel()
	m.DecodedActive = true
	m.DecodedPath = "" // decoded at root
	m.Path = "_.payload.nested"
	m.Node = map[string]any{"deep": true}

	m.updateDecodedState()

	assert.True(t, m.DecodedActive, "decoded at root means every path is within subtree")
}

func TestUpdateDecodedState_NotActive(t *testing.T) {
	m := decodeTableModel()
	m.DecodedActive = false
	m.Path = "_.payload"

	m.updateDecodedState()

	assert.False(t, m.DecodedActive, "should remain false when not active")
}

func TestUpdateDecodedState_BracketPath(t *testing.T) {
	m := decodeTableModel()
	m.DecodedActive = true
	m.DecodedPath = "_.list"
	m.Path = "_.list[0]"
	m.Node = float64(1)

	m.updateDecodedState()

	assert.True(t, m.DecodedActive, "should remain active with bracket notation under decoded path")
}

func TestDecodeReplacesQuotedKey(t *testing.T) {
	// Test that decode correctly replaces values under keys requiring bracket notation
	// (e.g., keys with dots like "a.b" → _["a.b"])
	node := map[string]any{
		"a.b": `{"nested": "decoded"}`,
	}
	m := InitialModel(node)
	m.Root = node
	m.InputFocused = false
	m.KeyMode = KeyModeFunction
	m.WinWidth = 120
	m.WinHeight = 30
	m.applyLayout(true)
	m.Tbl.Focus()

	// Navigate to the "a.b" key (first row) and into it
	moveCursorToKey(t, &m, "a.b")
	mp := pressKeyRight(&m) // enter the scalar → see (value)
	require.Equal(t, `_["a.b"]`, mp.Path, "path should use bracket notation for key with dot")
	require.Equal(t, `{"nested": "decoded"}`, mp.Node.(string), "should be on the serialized string value")

	// Now decode/navigate into it
	mp = pressKeyEnter(mp)

	// After decode, the Root should be mutated: the "a.b" key should now hold the decoded structure
	rootMap := mp.Root.(map[string]any)
	decoded, ok := rootMap["a.b"].(map[string]any)
	require.True(t, ok, "a.b should now be a map, not a string")
	assert.Equal(t, "decoded", decoded["nested"], "decoded structure should contain nested key")

	// Verify we're inside the decoded structure
	assert.True(t, mp.DecodedActive, "should be in decoded state")
}

func TestUnquoteSegment(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"a.b"`, `a.b`},
		{`"key with spaces"`, `key with spaces`},
		{`plain`, `plain`},
		{`0`, `0`},
		{`""`, ``}, // empty quoted string
		{`"`, `"`}, // malformed - not changed
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := unquoteSegment(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
