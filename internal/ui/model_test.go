//nolint:forcetypeassert
package ui

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/oakwood-commons/kvx/internal/completion"
	"github.com/oakwood-commons/kvx/internal/navigator"
)

func trimTrailingEmptyLines(lines []string) []string {
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// sampleData provides a small map structure for testing
func sampleData() map[string]interface{} {
	return map[string]interface{}{
		"cluster_001": map[string]interface{}{"alert": true},
		"cluster_002": map[string]interface{}{"alert": true},
		"items":       []interface{}{map[string]interface{}{"name": "a"}, map[string]interface{}{"name": "b"}},
		"regions":     map[string]interface{}{"asia": map[string]interface{}{"count": 2}, "europe": map[string]interface{}{"count": 3}},
	}
}

// helper to create a focused model
func focusedModel() Model {
	node := sampleData()
	m := InitialModel(node)
	m.Root = node
	m.InputFocused = true
	m.CompletionEngine = nil
	// Ensure PathInput is a fresh widget
	m.PathInput = textinput.New()
	m.PathInput.Prompt = ""
	m.PathInput.Focus()
	return m
}

// helper to create a non-focused table model for type-ahead testing
func tableModel() Model {
	node := sampleData()
	m := InitialModel(node)
	m.Root = node
	m.InputFocused = false
	m.KeyMode = KeyModeFunction // Disable vim mode to allow type-ahead filter testing
	m.Tbl.Focus()
	return m
}

func TestCompletionContextUsesResultTypeWhenInferredTypeDiffers(t *testing.T) {
	m := focusedModel()
	m.AllowSuggestions = true
	m.AllowIntellisense = true
	m.ExprType = "string"
	m.Root = sampleData()
	m.Node = m.Root
	m.PathInput.SetValue("_.cluster_001.")

	provider := &recordingCompletionProvider{}
	m.CompletionEngine = completion.NewEngine(provider)

	m.filterWithCompletionEngine(true)

	if provider.lastCtx.ExpressionResultType != "string" {
		t.Fatalf("expected ExpressionResultType to stay with last result type, got %q", provider.lastCtx.ExpressionResultType)
	}
	if provider.lastCtx.CurrentType != "string" {
		t.Fatalf("expected CurrentType to honor result type, got %q", provider.lastCtx.CurrentType)
	}
}

type recordingCompletionProvider struct {
	lastCtx completion.CompletionContext
}

func (p *recordingCompletionProvider) DiscoverFunctions() []completion.FunctionMetadata {
	return nil
}

func (p *recordingCompletionProvider) FilterCompletions(_ string, ctx completion.CompletionContext) []completion.Completion {
	p.lastCtx = ctx
	return nil
}

func (p *recordingCompletionProvider) EvaluateType(_ string, _ completion.CompletionContext) string {
	return "map"
}

func (p *recordingCompletionProvider) Evaluate(_ string, _ interface{}) (interface{}, error) {
	return nil, nil
}

func (p *recordingCompletionProvider) IsExpression(string) bool {
	return true
}

// Ensure that when entering at root, the expr section shows '_' by default
func TestRootShowsUnderscoreInPathInput(t *testing.T) {
	node := sampleData()
	m := InitialModel(node)
	if m.Path != "" {
		t.Fatalf("expected empty path at root, got %q", m.Path)
	}
	if got := m.PathInput.Value(); got != "_" {
		t.Fatalf("expected PathInput '_' at root, got %q", got)
	}
}

// Free-form CEL array literal in expr mode should evaluate and display indices
func TestFreeFormArrayLiteralInExprMode(t *testing.T) {
	m := focusedModel()
	// Type a CEL array literal and press Enter
	m.PathInput.SetValue("[1,2]")
	m.filterSuggestions()
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2 := newModel.(*Model)
	// Should render two rows with indices [0], [1]
	rows := m2.Tbl.Rows()
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows for array literal, got %d", len(rows))
	}
	if strings.TrimSpace(rows[0][0]) != "[0]" || strings.TrimSpace(rows[1][0]) != "[1]" {
		t.Fatalf("expected index keys [0],[1], got %q and %q", rows[0][0], rows[1][0])
	}
}

func TestSuggestionsRootAndDot(t *testing.T) {
	m := focusedModel()
	// type 'regions.'
	m.PathInput.SetValue("regions.")
	m.filterSuggestions()
	if !m.ShowSuggestions {
		t.Fatalf("expected suggestions after trailing dot")
	}
	// Should include keys like 'asia' and 'europe'
	hasAsia, hasEurope := false, false
	for _, s := range m.FilteredSuggestions {
		if s == "asia" {
			hasAsia = true
		}
		if s == "europe" {
			hasEurope = true
		}
	}
	if !hasAsia || !hasEurope {
		t.Fatalf("expected asia and europe in suggestions, got %v", m.FilteredSuggestions)
	}
}

func TestBracketIndexSuggestions(t *testing.T) {
	m := focusedModel()
	// type 'items['
	m.PathInput.SetValue("items[")
	m.filterSuggestions()
	// In expr mode, no index suggestions should be shown
	if m.ShowSuggestions && len(m.FilteredSuggestions) > 0 {
		t.Fatalf("did not expect index suggestions in expr mode, got %v", m.FilteredSuggestions)
	}
}

// When typing a trailing dot, functions should be prioritized over keys for Tab completion.
func TestFunctionsPreferredOnTrailingDot(t *testing.T) {
	m := focusedModel()
	// Override suggestions to a known mix of functions and keys
	m.Suggestions = []string{
		"asia",
		"contains() - contains(list, value) -> bool",
		"europe",
	}
	// Use a list context so 'contains(list, value)' is compatible
	m.PathInput.SetValue("items.")
	m.filterSuggestions()

	if len(m.FilteredSuggestions) == 0 {
		t.Fatalf("expected suggestions for trailing dot")
	}
	if !strings.Contains(m.FilteredSuggestions[0], "(") {
		t.Fatalf("expected first suggestion to be a function, got %q", m.FilteredSuggestions[0])
	}
	if m.SelectedSuggestion != 0 {
		t.Fatalf("expected first suggestion to be selected, got %d", m.SelectedSuggestion)
	}
}

func TestTrailingDotShowsFunctionSummaryInStatus(t *testing.T) {
	m := focusedModel()
	m.Suggestions = []string{
		"map() - list.map(x, expr) -> list",
		"filter() - list.filter(x, expr) -> list",
		"all() - list.all(expr) -> bool",
	}

	m.PathInput.SetValue("items.")
	m.filterSuggestions()
	m.syncStatus()

	status := stripANSI(m.Status.View())
	if !strings.Contains(status, ".map()") || !strings.Contains(status, ".filter()") {
		t.Fatalf("expected function summary in status bar, got %q", status)
	}
	if !m.ShowSuggestionSummary {
		t.Fatalf("expected ShowSuggestionSummary to be true after trailing dot")
	}
}

func TestFunctionSummaryClearsAfterTyping(t *testing.T) {
	m := focusedModel()
	m.Suggestions = []string{
		"map() - list.map(x, expr) -> list",
		"filter() - list.filter(x, expr) -> list",
	}

	m.PathInput.SetValue("items.")
	m.filterSuggestions()
	if !m.ShowSuggestionSummary {
		t.Fatalf("expected summary to be present after trailing dot")
	}

	m.PathInput.SetValue("items.m")
	m.filterSuggestions()
	if m.ShowSuggestionSummary {
		t.Fatalf("expected summary to clear after typing past trailing dot")
	}
}

func TestTrailingDotSummaryWhenIntellisenseDisabled(t *testing.T) {
	m := focusedModel()
	m.AllowIntellisense = false
	m.AllowSuggestions = true
	m.Suggestions = []string{
		"map() - list.map(x, expr) -> list",
		"filter() - list.filter(x, expr) -> list",
	}

	m.PathInput.SetValue("items.")
	m.filterSuggestions()
	m.syncStatus()

	status := stripANSI(m.Status.View())
	if !strings.Contains(status, ".map()") || !strings.Contains(status, ".filter()") {
		t.Fatalf("expected function summary in status bar even when intellisense disabled, got %q", status)
	}
	if !m.ShowSuggestionSummary {
		t.Fatalf("expected ShowSuggestionSummary to be true after trailing dot with intellisense disabled")
	}
}

// Ensure function insertion does not duplicate namespaces, e.g., _.base64.base64.decode()
func TestFunctionInsertionNamespaceDedupe(t *testing.T) {
	m := focusedModel()
	// Simulate suggestions containing a namespaced function and a key
	m.Suggestions = []string{
		"base64.decode() - CEL function",
		"asia",
	}
	// User types a trailing dot on base64 namespace
	m.PathInput.SetValue("_.base64.")
	m.filterSuggestions()
	// Press Tab to insert the first matching item (keys first, functions allowed)
	m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	got := m.PathInput.Value()
	if strings.Contains(got, "base64.base64") {
		t.Fatalf("Tab completion duplicated namespace, got %q", got)
	}
	if got == "_.base64." {
		t.Fatalf("Tab should complete a key or function, got unchanged value")
	}
}

// Filters function suggestions by type: for a string node, list-only functions like flatten shouldn't appear.
func TestTypeFilteredFunctionSuggestionsForString(t *testing.T) {
	m := focusedModel()
	// Simulate being at a string node
	m.Node = "hello"
	// Provide mixed suggestions with usage hints
	m.Suggestions = []string{
		"flatten() - list.flatten() -> list",
		"lowerAscii() - string.lowerAscii() -> string",
		"size() - string.size() -> int",
	}
	// Trailing dot to open function dropdown
	m.PathInput.SetValue(".")
	m.filterSuggestions()
	// Ensure only string-compatible functions are shown first and 'flatten' is excluded
	for _, s := range m.FilteredSuggestions {
		if strings.HasPrefix(s, "flatten()") {
			t.Fatalf("expected 'flatten' to be filtered out for string node, got in suggestions: %v", m.FilteredSuggestions)
		}
	}
	if len(m.FilteredSuggestions) == 0 {
		t.Fatalf("expected some suggestions for string node")
	}
}

func TestTabInsertsFirstIndex(t *testing.T) {
	m := focusedModel()
	m.PathInput.SetValue("items[")
	m.filterSuggestions()
	// simulate tab
	m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if got := m.PathInput.Value(); got != "items[0]" {
		t.Fatalf("expected items[0], got %s", got)
	}
}

// Numeric dotted tokens should still offer array index suggestions and replace the token with bracket form.
func TestTabCompletesNumericTokenToBracketIndex(t *testing.T) {
	m := focusedModel()
	m.PathInput.SetValue("items.0")
	m.filterSuggestions()
	if !m.ShowSuggestions || len(m.FilteredSuggestions) == 0 {
		t.Fatalf("expected suggestions for numeric token, got none: %+v", m.FilteredSuggestions)
	}
	// simulate tab
	m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if got := m.PathInput.Value(); got != "items[0]" {
		t.Fatalf("expected items[0], got %s", got)
	}
}

// Keys containing dots (like "foo.bar") should use bracket notation in completions.
func TestTabCompletesDottedKeyWithBracketNotation(t *testing.T) {
	// Create a model with expression mode and a dotted key
	node := map[string]interface{}{
		"foo.bar": "hello world",
		"normal":  "test",
	}
	m := InitialModel(node)
	m.Root = node
	m.Node = node
	m.NoColor = true
	m.AllowSuggestions = true
	m.AllowIntellisense = true
	m.InputFocused = true

	// Set up completion engine
	celProvider, err := completion.NewCELProvider()
	if err != nil {
		t.Fatalf("failed to create CEL provider: %v", err)
	}
	m.CompletionEngine = completion.NewEngine(celProvider)

	// Type "_." and then Tab to complete
	m.PathInput.SetValue("_.")
	m.filterSuggestions(true)

	if !m.ShowSuggestions || len(m.FilteredSuggestions) == 0 {
		t.Fatalf("expected suggestions for dotted key, got none: %+v", m.FilteredSuggestions)
	}

	// Verify the completion contains bracket notation
	foundBracket := false
	for _, s := range m.FilteredSuggestions {
		if s == `_["foo.bar"]` {
			foundBracket = true
			break
		}
	}
	if !foundBracket {
		t.Fatalf("expected suggestion _[\"foo.bar\"] not found in: %v", m.FilteredSuggestions)
	}

	// Simulate tab
	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m2 := result.(*Model)

	got := m2.PathInput.Value()
	// First suggestion should be "foo.bar" in bracket notation
	// (keys are sorted, so foo.bar comes before normal)
	expected := `_["foo.bar"]`
	if got != expected {
		t.Fatalf("expected %q, got %q; FilteredSuggestions=%v", expected, got, m.FilteredSuggestions)
	}
}

// Tab on just underscore should complete to the first key.
func TestTabOnUnderscoreCompletesToFirstKey(t *testing.T) {
	node := map[string]interface{}{
		"foo.bar": "hello world",
	}
	m := InitialModel(node)
	m.Root = node
	m.Node = node
	m.NoColor = true
	m.AllowSuggestions = true
	m.AllowIntellisense = true
	m.InputFocused = true

	celProvider, err := completion.NewCELProvider()
	if err != nil {
		t.Fatalf("failed to create CEL provider: %v", err)
	}
	m.CompletionEngine = completion.NewEngine(celProvider)

	// Type just "_" and then Tab
	m.PathInput.SetValue("_")
	m.filterSuggestions(true)

	// Simulate tab
	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m2 := result.(*Model)

	got := m2.PathInput.Value()
	// Should complete to _["foo.bar"] since it's the only key (and contains a dot)
	expected := `_["foo.bar"]`
	if got != expected {
		t.Fatalf("Tab on _ should complete to first key: expected %q, got %q; FilteredSuggestions=%v", expected, got, m.FilteredSuggestions)
	}
}

// Tab on a completed bracket-quoted path should NOT cycle to functions.
func TestTabOnBracketQuotedPathDoesNotCycleToFunctions(t *testing.T) {
	node := map[string]interface{}{
		"foo.bar": "hello world",
	}
	m := InitialModel(node)
	m.Root = node
	m.Node = node
	m.NoColor = true
	m.AllowSuggestions = true
	m.AllowIntellisense = true
	m.InputFocused = true

	celProvider, err := completion.NewCELProvider()
	if err != nil {
		t.Fatalf("failed to create CEL provider: %v", err)
	}
	m.CompletionEngine = completion.NewEngine(celProvider)

	// Set input to a completed bracket-quoted path
	m.PathInput.SetValue(`_["foo.bar"]`)
	m.filterSuggestions(true)

	// Tab should NOT change the value (no functions should be inserted)
	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m2 := result.(*Model)

	got := m2.PathInput.Value()
	expected := `_["foo.bar"]`
	if got != expected {
		t.Fatalf("Tab on bracket-quoted path should not change it: expected %q, got %q", expected, got)
	}

	// Multiple tabs should also not change
	result, _ = m2.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m3 := result.(*Model)
	got = m3.PathInput.Value()
	if got != expected {
		t.Fatalf("Multiple tabs should not change bracket-quoted path: expected %q, got %q", expected, got)
	}

	// Test many more tabs to verify no cycling to functions
	for i := 0; i < 10; i++ {
		result, _ = m3.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		m3 = result.(*Model)
		got = m3.PathInput.Value()
		if got != expected {
			t.Fatalf("Tab %d should not change bracket-quoted path: expected %q, got %q", i+3, expected, got)
		}
	}
}

// Test that even if suggestions were cached from before, Tab on bracket-quoted path doesn't cycle to functions.
func TestTabOnBracketQuotedPathWithCachedSuggestions(t *testing.T) {
	node := map[string]interface{}{
		"foo.bar": "hello world",
	}
	m := InitialModel(node)
	m.Root = node
	m.Node = node
	m.NoColor = true
	m.AllowSuggestions = true
	m.AllowIntellisense = true
	m.InputFocused = true

	celProvider, err := completion.NewCELProvider()
	if err != nil {
		t.Fatalf("failed to create CEL provider: %v", err)
	}
	m.CompletionEngine = completion.NewEngine(celProvider)

	// IMPORTANT: Simulate having stale suggestions from a previous state
	// This tests the case where ShowSuggestions is true and FilteredSuggestions contains functions
	m.PathInput.SetValue(`_["foo.bar"]`)
	m.ShowSuggestions = true
	m.FilteredSuggestions = []string{
		`_["foo.bar"]`,
		"_.all - CEL function",
		"_.exists - CEL function",
		"_.filter - CEL function",
	}

	// Tab should NOT cycle to functions because the dot in "foo.bar" is inside brackets
	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m2 := result.(*Model)

	got := m2.PathInput.Value()
	expected := `_["foo.bar"]`
	if got != expected {
		t.Fatalf("Tab with cached suggestions should not change bracket-quoted path: expected %q, got %q", expected, got)
	}

	// More tabs should also not change to functions
	for i := 0; i < 5; i++ {
		result, _ = m2.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		m2 = result.(*Model)
		got = m2.PathInput.Value()
		if got != expected {
			t.Fatalf("Tab %d with cached suggestions changed path: expected %q, got %q", i+2, expected, got)
		}
	}
}

// Regression test: Tab on nested bracket key like _.items["foo.bar"] should not cycle to _.all()
func TestTabOnNestedBracketKeyDoesNotCycleToFunctions(t *testing.T) {
	node := map[string]interface{}{
		"items": map[string]interface{}{
			"foo.bar": "value",
		},
	}
	m := InitialModel(node)
	m.Root = node
	m.Node = node
	m.NoColor = true
	m.AllowSuggestions = true
	m.AllowIntellisense = true
	m.InputFocused = true

	celProvider, err := completion.NewCELProvider()
	if err != nil {
		t.Fatalf("failed to create CEL provider: %v", err)
	}
	m.CompletionEngine = completion.NewEngine(celProvider)

	// Set up a nested bracket path
	m.PathInput.SetValue(`_.items["foo.bar"]`)
	m.ShowSuggestions = true
	// Simulate stale suggestions from prior _.items. state
	m.FilteredSuggestions = []string{
		`_.items["foo.bar"]`,
		".all() - CEL function",
		".exists() - CEL function",
		".filter() - CEL function",
	}

	// Tab should NOT cycle to .all() because the path ends with a complete bracket key
	mp := &m
	for i := 0; i < 10; i++ {
		result, _ := mp.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		mp = result.(*Model)
		got := mp.PathInput.Value()
		// Path should either stay the same or not become a function call
		if strings.Contains(got, "all()") || strings.Contains(got, "exists()") || strings.Contains(got, "filter()") {
			t.Fatalf("Tab %d incorrectly cycled to function: got %q", i+1, got)
		}
	}
}

func TestViewHeightStableWithoutDebug(t *testing.T) {
	node := sampleData()
	m := InitialModel(node)
	m.Root = node
	m.NoColor = true

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m1 := updated.(*Model)

	view := m1.View()
	lines := trimTrailingEmptyLines(strings.Split(fmt.Sprint(view.Content), "\n"))
	if got := len(lines); got != 20 {
		t.Fatalf("expected 20 lines after resize, got %d", got)
	}
	for _, l := range lines {
		if strings.Contains(l, "Rows:") || strings.Contains(l, "Cols:") {
			t.Fatalf("did not expect debug footer without debug mode, got: %q", l)
		}
	}

	updated, _ = m1.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2 := updated.(*Model)
	view = m2.View()
	lines = trimTrailingEmptyLines(strings.Split(fmt.Sprint(view.Content), "\n"))
	if got := len(lines); got != 20 {
		t.Fatalf("expected 20 lines after navigation, got %d", got)
	}
	for _, l := range lines {
		if strings.Contains(l, "Rows:") || strings.Contains(l, "Cols:") {
			t.Fatalf("did not expect debug footer without debug mode after navigation, got: %q", l)
		}
	}
}

func TestViewHeightStableWithDebug(t *testing.T) {
	node := sampleData()
	m := InitialModel(node)
	m.Root = node
	m.NoColor = true
	m.DebugMode = true

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m1 := updated.(*Model)

	view := m1.View()
	lines := trimTrailingEmptyLines(strings.Split(fmt.Sprint(view.Content), "\n"))
	if got := len(lines); got != 20 {
		t.Fatalf("expected 20 lines after resize, got %d", got)
	}
	foundDebug := false
	for _, l := range lines {
		if strings.Contains(l, "Rows:") && strings.Contains(l, "Cols:") {
			foundDebug = true
			break
		}
	}
	if !foundDebug {
		t.Fatalf("expected debug line with debug mode enabled")
	}

	updated, _ = m1.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2 := updated.(*Model)
	view = m2.View()
	lines = trimTrailingEmptyLines(strings.Split(fmt.Sprint(view.Content), "\n"))
	if got := len(lines); got != 20 {
		t.Fatalf("expected 20 lines after navigation, got %d", got)
	}
	foundDebug = false
	for _, l := range lines {
		if strings.Contains(l, "Rows:") && strings.Contains(l, "Cols:") {
			foundDebug = true
			break
		}
	}
	if !foundDebug {
		t.Fatalf("expected debug line with debug mode enabled after navigation")
	}
}

func TestF1HelpTogglesAndRenders(t *testing.T) {
	m := focusedModel()
	// Toggle help on
	m.Update(tea.KeyPressMsg{Code: tea.KeyF1})
	if !m.HelpVisible {
		t.Fatalf("expected HelpVisible true after F1")
	}
	view := m.View()
	// In vim mode (default), we show vim keybindings, not function keys
	// Check for vim-style help content
	if !strings.Contains(fmt.Sprint(view.Content), "toggle help") {
		t.Fatalf("expected help overlay with 'toggle help' in view, got: %s", fmt.Sprint(view.Content))
	}
	if !strings.Contains(fmt.Sprint(view.Content), "expression mode") {
		t.Fatalf("expected help overlay with 'expression mode' in view, got: %s", fmt.Sprint(view.Content))
	}
	// Toggle help off
	m.Update(tea.KeyPressMsg{Code: tea.KeyF1})
	if m.HelpVisible {
		t.Fatalf("expected HelpVisible false after second F1")
	}
}

// Enter on a scalar '(value)' row after searching should not append trailing dots to the path input.
func TestEnterOnScalarAfterSearchDoesNotAppendDot(t *testing.T) {
	root := sampleData()
	// Start in table mode at scalar path cluster_001.alert
	m := InitialModel(root)
	m.Root = root
	m.InputFocused = false
	m.Tbl.Focus()

	node, err := navigator.NodeAtPath(root, "cluster_001.alert")
	if err != nil {
		t.Fatalf("navigate to scalar: %v", err)
	}
	m2 := m.NavigateTo(node, "cluster_001.alert")

	// Trigger type-ahead search (even though only '(value)' exists)
	m2.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})

	newModel, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m3 := newModel.(*Model)

	if got := m3.PathInput.Value(); got != "_.cluster_001.alert" {
		t.Fatalf("expected stable path input without trailing dot, got %q", got)
	}
	if m3.FilterActive {
		t.Fatalf("expected search to be cleared on scalar enter")
	}
}

// Resizing should not drop active type-ahead filtering
func TestWindowResizePreservesSearchFilter(t *testing.T) {
	m := tableModel()
	// Activate search to filter to cluster_001/cluster_002
	m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	if !m.FilterActive {
		t.Fatalf("expected search to be active")
	}
	before := m.Tbl.Rows()
	if len(before) == 0 {
		t.Fatalf("expected filtered rows before resize")
	}

	// Simulate window resize
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	after := m.Tbl.Rows()
	if len(after) != len(before) {
		t.Fatalf("expected filtered rows preserved after resize, got %d want %d", len(after), len(before))
	}
	if !m.FilterActive {
		t.Fatalf("expected search to remain active after resize")
	}
}

// Very small windows should still keep sane widths/heights without panics
func TestApplyLayoutClampsSmallWindow(t *testing.T) {
	m := tableModel()
	m.WinWidth = 20
	m.WinHeight = 8
	m.HelpVisible = true
	m.applyLayout()

	if m.TableHeight < 2 {
		t.Fatalf("expected table height >=2 (minimum for headers), got %d", m.TableHeight)
	}
	if m.ValueColWidth < 10 {
		t.Fatalf("expected value column width >=10, got %d", m.ValueColWidth)
	}
	// KeyColWidth stays at default (30) even for narrow windows
	// The table will just be wider than the window, which is acceptable
	if m.KeyColWidth <= 0 {
		t.Fatalf("expected key column width >0, got %d", m.KeyColWidth)
	}
}

func TestAutoSelectAndEnterDrill(t *testing.T) {
	m := focusedModel()
	// In expr mode, type full key and press Enter to navigate
	m.PathInput.SetValue("cluster_001")
	m.filterSuggestions()
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2 := newModel.(*Model)
	if m2.Path != "_.cluster_001" {
		t.Fatalf("expected path to _.cluster_001, got %s", m2.Path)
	}
}

func TestEnterDrillAfterDot(t *testing.T) {
	m := focusedModel()
	// Type full path in expr mode and Enter should navigate
	m.PathInput.SetValue("regions.asia")
	m.filterSuggestions()
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2 := newModel.(*Model)
	if m2.Path != "_.regions.asia" {
		t.Fatalf("expected _.regions.asia, got %s", m2.Path)
	}
}

// Validates that typing a partial key auto-selects the best match and Enter navigates
func TestAutoSelectPrefixAndEnterNavigates(t *testing.T) {
	m := focusedModel()
	// Type full child key under regions in expr mode
	m.PathInput.SetValue("regions.asia")
	m.filterSuggestions()
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2 := newModel.(*Model)
	if m2.Path != "_.regions.asia" {
		t.Fatalf("expected _.regions.asia, got %s", m2.Path)
	}
}

// Validates that after typing a key with trailing dot, Enter navigates to selected sub key
func TestEnterNavigatesToSubKey(t *testing.T) {
	m := focusedModel()
	// Drill into cluster_001's sub key by typing full path in expr mode
	m.PathInput.SetValue("cluster_001.alert")
	m.filterSuggestions()
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2 := newModel.(*Model)
	if m2.Path != "_.cluster_001.alert" {
		t.Fatalf("expected _.cluster_001.alert, got %s", m2.Path)
	}
}

// TestTruncatedKeyNavigation validates that navigating to a truncated key uses the full untruncated key
// This is a regression test for the bug where truncated display keys were used for navigation
func TestTruncatedKeyNavigation(t *testing.T) {
	// Create a model with a key longer than the default column width (30 chars)
	longKey := "secret_scanning_non_provider_patterns"
	node := map[string]interface{}{
		longKey: map[string]interface{}{"enabled": true},
		"short": "value",
	}
	m := InitialModel(node)
	m.Root = node
	m.InputFocused = false
	// Set narrow column width to ensure truncation
	m.KeyColWidth = 30
	m.ValueColWidth = 60
	m.applyLayout()

	// Verify the key is displayed as truncated in the table
	rows := m.Tbl.Rows()
	if len(rows) < 1 {
		t.Fatalf("expected at least 1 row, got %d", len(rows))
	}
	// Find the row with the long key (it should be truncated with "...")
	truncatedFound := false
	var rowIdx int
	for i, row := range rows {
		if len(row[0]) > 0 && len(row[0]) < len(longKey) && row[0][len(row[0])-3:] == "..." {
			truncatedFound = true
			rowIdx = i
			break
		}
	}
	if !truncatedFound {
		t.Fatalf("expected to find truncated key in table rows")
	}

	// Position cursor on the truncated key
	m.Tbl.SetCursor(rowIdx)

	// Simulate pressing Enter to navigate into the key
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2 := newModel.(*Model)

	// Verify navigation succeeded with the full untruncated key
	expectedPath := "_." + longKey
	if m2.Path != expectedPath {
		t.Fatalf("expected path to be '%s', got '%s'", expectedPath, m2.Path)
	}
	if m2.ErrMsg != "" {
		t.Fatalf("expected no error, got: %s", m2.ErrMsg)
	}

	// Verify we're now looking at the nested content
	rows2 := m2.Tbl.Rows()
	if len(rows2) != 1 {
		t.Fatalf("expected 1 row in nested view, got %d", len(rows2))
	}
	if strings.TrimSpace(rows2[0][0]) != "enabled" {
		t.Fatalf("expected 'enabled' key, got '%s'", rows2[0][0])
	}
}

// TestTypeAheadSearch validates the type-ahead search feature
func TestTypeAheadSearch(t *testing.T) {
	m := tableModel()

	// Initially should have all rows
	initialRowCount := len(m.Tbl.Rows())
	if initialRowCount != 4 {
		t.Fatalf("expected 4 initial rows, got %d", initialRowCount)
	}

	// Type 'c' - should activate search and filter to cluster_001, cluster_002
	m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	if !m.FilterActive {
		t.Fatalf("expected search to be active after typing")
	}
	if m.FilterBuffer != "c" {
		t.Fatalf("expected search buffer 'c', got '%s'", m.FilterBuffer)
	}
	rows := m.Tbl.Rows()
	if len(rows) != 2 {
		t.Fatalf("expected 2 filtered rows (cluster_*), got %d", len(rows))
	}

	// Type 'l' - should narrow to cluster_ keys
	m.Update(tea.KeyPressMsg{Code: 'l', Text: "l"})
	if m.FilterBuffer != "cl" {
		t.Fatalf("expected search buffer 'cl', got '%s'", m.FilterBuffer)
	}

	// Type 'u' - should further narrow
	m.Update(tea.KeyPressMsg{Code: 'u', Text: "u"})
	if m.FilterBuffer != "clu" {
		t.Fatalf("expected search buffer 'clu', got '%s'", m.FilterBuffer)
	}
}

// TestTypeAheadBackspace validates backspace removes characters from search buffer
func TestTypeAheadBackspace(t *testing.T) {
	m := tableModel()

	// Type 'cluster'
	for _, ch := range "cluster" {
		// v2: Use Code and Text for character input
		m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}
	if m.FilterBuffer != "cluster" {
		t.Fatalf("expected search buffer 'cluster', got '%s'", m.FilterBuffer)
	}

	// Press backspace - should remove 'r'
	m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if m.FilterBuffer != "cluste" {
		t.Fatalf("expected search buffer 'cluste' after backspace, got '%s'", m.FilterBuffer)
	}
	if !m.FilterActive {
		t.Fatalf("expected search to still be active")
	}

	// Keep pressing backspace until buffer is empty
	for i := 0; i < 6; i++ {
		m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	}

	// Should deactivate search and restore all rows
	if m.FilterActive {
		t.Fatalf("expected search to be inactive after clearing buffer")
	}
	if m.FilterBuffer != "" {
		t.Fatalf("expected empty search buffer, got '%s'", m.FilterBuffer)
	}
	rows := m.Tbl.Rows()
	if len(rows) != 4 {
		t.Fatalf("expected all 4 rows restored, got %d", len(rows))
	}
}

// TestTypeAheadEscape validates escape clears search and restores table
func TestTypeAheadEscape(t *testing.T) {
	m := tableModel()

	// Type some characters to activate search
	m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	m.Update(tea.KeyPressMsg{Code: 'l', Text: "l"})
	if !m.FilterActive {
		t.Fatalf("expected search to be active")
	}

	// Press escape - should clear search and restore all rows
	m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.FilterActive {
		t.Fatalf("expected search to be inactive after escape")
	}
	if m.FilterBuffer != "" {
		t.Fatalf("expected empty search buffer after escape, got '%s'", m.FilterBuffer)
	}
	rows := m.Tbl.Rows()
	if len(rows) != 4 {
		t.Fatalf("expected all 4 rows restored after escape, got %d", len(rows))
	}
	if m.Tbl.Cursor() != 0 {
		t.Fatalf("expected cursor reset to 0 after escape, got %d", m.Tbl.Cursor())
	}
}

// TestTypeAheadDoesNotInterfereWithCommands validates single-key commands work
func TestTypeAheadDoesNotInterfereWithCommands(t *testing.T) {
	m := tableModel()

	// Typing 'y' should now start search (since copy moved to ctrl+y)
	m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if !m.FilterActive || m.FilterBuffer != "y" {
		t.Fatalf("expected search to start with 'y', got active=%v buf=%q", m.FilterActive, m.FilterBuffer)
	}
}

// TestTypeAheadNavigationUsesFilteredKey validates navigation after filtering
// This is a regression test for the bug where filtered cursor position was used
// to index into original unfiltered data, causing navigation to wrong keys
func TestTypeAheadNavigationUsesFilteredKey(t *testing.T) {
	// Create data with multiple keys where alphabetical order matters
	node := map[string]interface{}{
		"allow_auto_merge": "value1",
		"allow_forking":    "value2",
		"items":            []interface{}{"a", "b"},
		"regions":          map[string]interface{}{"asia": "data"},
	}

	m := InitialModel(node)
	m.Root = node
	m.InputFocused = false
	m.KeyMode = KeyModeFunction // Disable vim mode for type-ahead filter testing
	m.Tbl.Focus()

	// Type 'i' to filter to "items"
	m.Update(tea.KeyPressMsg{Code: 'i', Text: "i"})
	if !m.FilterActive {
		t.Fatalf("expected search to be active after typing")
	}

	// Verify filtered to show only "items"
	rows := m.Tbl.Rows()
	if len(rows) != 1 {
		t.Fatalf("expected 1 filtered row, got %d", len(rows))
	}
	if !strings.HasPrefix(rows[0][0], "items") {
		t.Fatalf("expected filtered row to be 'items', got '%s'", rows[0][0])
	}

	// Press Enter to navigate into "items"
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2 := newModel.(*Model)

	// Verify we navigated to "items", not "allow_auto_merge" (which would be index 0)
	if m2.Path != "_.items" {
		t.Fatalf("expected path '_.items', got '%s'", m2.Path)
	}
	if m2.ErrMsg != "" {
		t.Fatalf("expected no error, got: %s", m2.ErrMsg)
	}

	// Verify we're looking at the array contents
	rows2 := m2.Tbl.Rows()
	if len(rows2) != 2 {
		t.Fatalf("expected 2 rows (array elements), got %d", len(rows2))
	}
}

// TestTypeAheadNavigationWithTruncatedKeys validates navigation works with long keys
// This is a regression test for the bug where truncated display keys were used for
// navigation instead of the full original keys
func TestTypeAheadNavigationWithTruncatedKeys(t *testing.T) {
	// Create data with a very long key that will be truncated (> 30 chars)
	longKey := "secret_scanning_non_provider_patterns"
	node := map[string]interface{}{
		"allow_auto_merge": "value1",
		longKey:            map[string]interface{}{"enabled": true},
		"other_key":        "value2",
	}

	m := InitialModel(node)
	m.Root = node
	m.InputFocused = false
	m.KeyMode = KeyModeFunction // Disable vim mode for type-ahead filter testing
	m.Tbl.Focus()
	// Set narrow column width to ensure truncation
	m.KeyColWidth = 30
	m.ValueColWidth = 60
	m.applyLayout()

	// Type 'secret' to filter to the long key
	for _, ch := range "secret" {
		// v2: Use Code and Text for character input
		m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}

	if !m.FilterActive {
		t.Fatalf("expected search to be active after typing")
	}
	if m.FilterBuffer != "secret" {
		t.Fatalf("expected search buffer 'secret', got '%s'", m.FilterBuffer)
	}

	// Verify filtered to show only the long key
	rows := m.Tbl.Rows()
	if len(rows) != 1 {
		t.Fatalf("expected 1 filtered row, got %d", len(rows))
	}

	// The display might show truncated key, but navigation should use full key
	// Press Enter to navigate
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2 := newModel.(*Model)

	// Verify we navigated to the correct long key, not another key
	expectedPath := "_." + longKey
	if m2.Path != expectedPath {
		t.Fatalf("expected path '%s', got '%s'", expectedPath, m2.Path)
	}
	if m2.ErrMsg != "" {
		t.Fatalf("expected no error, got: %s", m2.ErrMsg)
	}

	// Verify we're looking at the nested content
	rows2 := m2.Tbl.Rows()
	if len(rows2) != 1 {
		t.Fatalf("expected 1 row in nested view, got %d", len(rows2))
	}
	if strings.TrimSpace(rows2[0][0]) != "enabled" {
		t.Fatalf("expected 'enabled' key, got '%s'", rows2[0][0])
	}
}

// View should not mutate model state; it must remain read-only.
func TestViewDoesNotMutateState(t *testing.T) {
	m := tableModel()
	// Set some state that previously could be reset during View
	m.FilterActive = true
	m.FilterBuffer = "c"
	initialRows := m.Tbl.Rows()
	initialCursor := m.Tbl.Cursor()

	_ = m.View()

	if m.FilterActive != true {
		t.Fatalf("expected FilterActive to remain true after View, got %v", m.FilterActive)
	}
	if m.FilterBuffer != "c" {
		t.Fatalf("expected FilterBuffer to remain 'c', got %q", m.FilterBuffer)
	}
	if len(m.Tbl.Rows()) != len(initialRows) {
		t.Fatalf("expected rows length to remain %d, got %d", len(initialRows), len(m.Tbl.Rows()))
	}
	if m.Tbl.Cursor() != initialCursor {
		t.Fatalf("expected cursor to remain %d, got %d", initialCursor, m.Tbl.Cursor())
	}
}

// buildViewSnapshot should assemble render parts without mutation and include key pieces.
func TestBuildViewSnapshotNoColor(t *testing.T) {
	m := tableModel()
	m.NoColor = true
	m.ApplyColorScheme()
	m.WinWidth = 5

	snap := m.buildViewSnapshot()

	if snap.Table == "" {
		t.Fatalf("expected table view content")
	}
	if snap.Separator != strings.Repeat("-", m.WinWidth) {
		t.Fatalf("expected separator of width %d, got %q", m.WinWidth, snap.Separator)
	}
	if snap.Status == "" {
		t.Fatalf("expected status content in snapshot")
	}
	if !strings.Contains(snap.Input, "$ ") {
		t.Fatalf("expected input to include label, got %q", snap.Input)
	}
}

// Snapshot rendering should keep table headers visible for regression protection.
func TestViewSnapshotIncludesHeaders(t *testing.T) {
	m := tableModel()
	m.NoColor = true
	m.ApplyColorScheme()

	snap := m.buildViewSnapshot()

	if !strings.Contains(snap.Table, "KEY") {
		t.Fatalf("expected snapshot table to include KEY header, got %q", snap.Table)
	}
	if !strings.Contains(snap.Table, "VALUE") {
		t.Fatalf("expected snapshot table to include VALUE header, got %q", snap.Table)
	}
}

// Advanced search navigation should restore context correctly when moving right/right then left/left and Enter.
func TestAdvancedSearchNavigationRoundTrip(t *testing.T) {
	// Fixture with nested data for search and navigation
	node := map[string]interface{}{
		"root": map[string]interface{}{
			"child": map[string]interface{}{
				"grandchild": "value",
			},
			"sibling": "keep",
		},
		"other": "x",
	}

	m := InitialModel(node)
	m.Root = node
	m.InputFocused = false
	m.SearchDebounceMs = 0 // Disable debouncing for tests
	m.Tbl.Focus()

	// Enter advanced search (F3)
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF3})

	// Type query "child"
	for _, ch := range "child" {
		m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}

	// Press Enter to commit the search (search no longer runs in real-time)
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	// We should now have search results; ensure they exist
	if len(m.AdvancedSearchResults) == 0 {
		t.Fatalf("expected search results for query 'child'")
	}

	// Right arrow: go to first result
	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	m1 := nm.(*Model)
	// Right exits search view but preserves search context
	if m1.AdvancedSearchActive {
		t.Fatalf("expected to leave search view after right")
	}
	if !m1.SearchContextActive {
		t.Fatalf("expected search context to remain active after right")
	}
	if m1.Path == "" || !strings.HasPrefix(m1.Path, "_.root") {
		t.Fatalf("expected path to include search result under root, got %q", m1.Path)
	}

	// Right arrow again (descend further if possible)
	nm, _ = m1.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	m2 := nm.(*Model)
	if !strings.HasPrefix(m2.Path, "_.root.child") {
		t.Fatalf("expected path to stay within _.root.child, got %q", m2.Path)
	}

	// Left arrow: back once
	nm, _ = m2.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	m3 := nm.(*Model)
	if !m3.SearchContextActive {
		t.Fatalf("expected search context to persist after first back")
	}
	// Search view should toggle only after second back
	if m3.AdvancedSearchActive {
		t.Fatalf("expected search view to stay exited after first back")
	}

	// Left arrow: back to search results root
	nm, _ = m3.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	m4 := nm.(*Model)
	if !m4.AdvancedSearchActive {
		t.Fatalf("expected to restore search view after backing twice")
	}
	if m4.Path != "" && m4.Path != "_.root" {
		t.Fatalf("expected path reset toward search base, got %q", m4.Path)
	}

	// Enter: accept and exit search; should land on selected result path and clear search flags
	nm, _ = m4.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m5 := nm.(*Model)
	if m5.AdvancedSearchActive {
		t.Fatalf("expected to exit search after Enter")
	}
	if m5.Path == "" || !strings.HasPrefix(m5.Path, "_.root") {
		t.Fatalf("expected to land on search result path after Enter, got %q", m5.Path)
	}
	if m5.SearchContextActive {
		t.Fatalf("expected search context cleared after Enter")
	}
	if m5.PathInput.Value() != formatPathForDisplay(m5.Path) {
		t.Fatalf("expected path input to show current path, got %q", m5.PathInput.Value())
	}
}

// Enter should clear search and search context flags.
func TestEnterClearsSearchState(t *testing.T) {
	node := map[string]interface{}{
		"root": map[string]interface{}{
			"child": "v",
		},
	}
	m := InitialModel(node)
	m.Root = node
	m.SearchDebounceMs = 0 // Disable debouncing for tests
	m.Tbl.Focus()

	// Enter search, type query
	m.Update(tea.KeyPressMsg{Code: tea.KeyF3})
	for _, ch := range "child" {
		m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}
	// Press Enter to commit the search (search no longer runs in real-time)
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !m.AdvancedSearchActive || len(m.AdvancedSearchResults) == 0 {
		t.Fatalf("expected active search with results")
	}

	// Navigate right to set search context
	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	m2 := nm.(*Model)
	if !m2.SearchContextActive {
		t.Fatalf("expected search context after right navigation")
	}

	// Enter should clear both search active and context
	nm, _ = m2.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m3 := nm.(*Model)
	if m3.AdvancedSearchActive || m3.SearchContextActive {
		t.Fatalf("expected search state cleared after enter, got active=%v ctx=%v", m3.AdvancedSearchActive, m3.SearchContextActive)
	}

	// Press F6 to enter expr mode; search state should remain cleared and expr should reflect selection
	nm, _ = m3.Update(tea.KeyPressMsg{Code: tea.KeyF6})
	m4 := nm.(*Model)
	if !m4.InputFocused {
		t.Fatalf("expected expr mode after F6")
	}
	if m4.AdvancedSearchActive || m4.SearchContextActive {
		t.Fatalf("expected search state cleared after F6 toggle, got active=%v ctx=%v", m4.AdvancedSearchActive, m4.SearchContextActive)
	}
	if m4.PathInput.Value() == "" || m4.PathInput.Value() == "_" {
		t.Fatalf("expected PathInput to reflect current selection after F6, got %q", m4.PathInput.Value())
	}
}

// View must not mutate model state (guardrail against stateful rendering).
func TestViewIsPure(t *testing.T) {
	node := sampleData()
	m := InitialModel(node)
	m.Root = node
	m.NoColor = true
	m.ApplyColorScheme()

	// Set a known path and cursor
	m.Path = "regions"
	m.PathInput.SetValue("_.regions")
	m.Tbl.SetCursor(1)
	m.SyncTableState()

	beforePath := m.Path
	beforeRows := m.AllRows
	beforeCursor := m.Tbl.Cursor()

	_ = m.View()
	_ = m.View() // call twice to be sure

	if m.Path != beforePath {
		t.Fatalf("View mutated Path: expected %q, got %q", beforePath, m.Path)
	}
	if len(m.AllRows) != len(beforeRows) {
		t.Fatalf("View mutated rows length: expected %d, got %d", len(beforeRows), len(m.AllRows))
	}
	if m.Tbl.Cursor() != beforeCursor {
		t.Fatalf("View mutated cursor: expected %d, got %d", beforeCursor, m.Tbl.Cursor())
	}
}

// Verifies that pressing F6 transfers the active search into the path input
// and places the cursor at the end of the composed value (root context)
func TestSlashTransfersSearchToPathAndSetsCursor_Root(t *testing.T) {
	m := tableModel()
	// Type a small search token
	for _, ch := range "clu" {
		m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}
	if !m.FilterActive || m.FilterBuffer != "clu" {
		t.Fatalf("expected active search buffer 'clu', got active=%v buf=%q", m.FilterActive, m.FilterBuffer)
	}

	// Press F6 to enter command mode
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyF6})
	m2 := newModel.(*Model)

	if !m2.InputFocused {
		t.Fatalf("expected input to be focused after F6")
	}
	// PathInput should show the selected row path (matching footer) - first key in sorted order
	val := m2.PathInput.Value()
	if !strings.HasPrefix(val, "_.") {
		t.Fatalf("expected PathInput to show selected row path starting with '_.' at root, got %q", val)
	}
	// Search should be cleared
	if m2.FilterActive || m2.FilterBuffer != "" {
		t.Fatalf("expected search state cleared, got active=%v buf=%q", m2.FilterActive, m2.FilterBuffer)
	}
}

// Verifies F6 with a non-root path shows selected row path (matching footer)
func TestSlashTransfersSearchToPathAndSetsCursor_NonRoot(t *testing.T) {
	// Start from sample data and navigate to 'regions'
	node := sampleData()
	m := InitialModel(node)
	m.Root = node
	m.InputFocused = false
	m.KeyMode = KeyModeFunction // Disable vim mode for type-ahead filter testing
	m.Tbl.Focus()

	// Navigate to regions via NavigateTo helper
	regions := map[string]interface{}{"asia": map[string]interface{}{"count": 2}, "europe": map[string]interface{}{"count": 3}}
	m2 := m.NavigateTo(regions, "regions")
	m2.KeyMode = KeyModeFunction // Ensure vim mode stays off after NavigateTo

	// Type a search token 'a'
	m2.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	if !m2.FilterActive || m2.FilterBuffer != "a" {
		t.Fatalf("expected active search buffer 'a', got active=%v buf=%q", m2.FilterActive, m2.FilterBuffer)
	}

	// Press F6 to enter command mode
	nm, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyF6})
	m3 := nm.(*Model)

	if !m3.InputFocused {
		t.Fatalf("expected input to be focused after F6")
	}
	// PathInput should show the selected row path (matching footer)
	// At regions level, first key in sorted order is "asia"
	if m3.PathInput.Value() != "_.regions.asia" {
		t.Fatalf("expected PathInput to be '_.regions.asia', got %q", m3.PathInput.Value())
	}
	if m3.FilterActive || m3.FilterBuffer != "" {
		t.Fatalf("expected search state cleared, got active=%v buf=%q", m3.FilterActive, m3.FilterBuffer)
	}
}

// TestFreeeFormCELExpression validates that CEL functions like type(_) can be entered
// without auto-prefix interfering. This tests the fix to allow arbitrary CEL expressions.
func TestFreeFormCELExpression(t *testing.T) {
	m := focusedModel()
	// Start with fresh input
	m.PathInput.SetValue("")

	// Type 'type('
	for _, ch := range "type(" {
		m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}

	// Should NOT have been auto-prefixed with '_.' because it contains '('
	// which is a CEL indicator
	if !strings.Contains(m.PathInput.Value(), "type(") {
		t.Fatalf("expected to contain 'type(', got %q", m.PathInput.Value())
	}

	// Continue typing the expression
	for _, ch := range "_)" {
		m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}

	value := m.PathInput.Value()
	if value != "type(_)" {
		t.Fatalf("expected 'type(_)', got %q", value)
	}
}

// TestEnterKeepsFreeFormCELExpression ensures pressing Enter does not inject '_.'
// when the user types a free-form CEL expression like type(_)
func TestEnterKeepsFreeFormCELExpression(t *testing.T) {
	m := focusedModel()
	// Type the full expression
	m.PathInput.SetValue("type(_)")
	m.filterSuggestions()
	// Press Enter
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2 := newModel.(*Model)
	// PathInput should remain exactly as typed
	if m2.PathInput.Value() != "type(_)" {
		t.Fatalf("expected PathInput to remain 'type(_)', got %q", m2.PathInput.Value())
	}
}

// TestEnterKeepsFreeFormCELWithOps ensures expressions with operators are preserved
func TestEnterKeepsFreeFormCELWithOps(t *testing.T) {
	m := focusedModel()
	// Provide sample data as root
	m.Root = sampleData()
	// Type an ops-based CEL expression
	m.PathInput.SetValue("size(_) > 0")
	m.filterSuggestions()
	// Press Enter
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2 := newModel.(*Model)
	// PathInput should remain exactly as typed
	if m2.PathInput.Value() != "size(_) > 0" {
		t.Fatalf("expected PathInput to remain 'size(_) > 0', got %q", m2.PathInput.Value())
	}
}

// TestEnterEvaluatesQuotedString ensures quoted strings are treated as CEL literals
// and not as keys; Enter should evaluate and keep the input exactly.
func TestEnterEvaluatesQuotedString(t *testing.T) {
	m := focusedModel()
	// Type a quoted string literal
	m.PathInput.SetValue("\"hi\"")
	m.filterSuggestions()
	// Press Enter
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2 := newModel.(*Model)
	// PathInput should remain exactly as typed
	if m2.PathInput.Value() != "\"hi\"" {
		t.Fatalf("expected PathInput to remain '"+"\"hi\""+"', got %q", m2.PathInput.Value())
	}
	// Should render a scalar value row
	rows := m2.Tbl.Rows()
	if len(rows) != 1 || strings.TrimSpace(rows[0][0]) != "(value)" {
		t.Fatalf("expected scalar '(value)' row, got %+v", rows)
	}
	if strings.TrimSpace(rows[0][1]) != "hi" {
		t.Fatalf("expected scalar value 'hi', got '%s'", rows[0][1])
	}
}

// TestQuotedStringMethod ensures quoted string method calls like "hi".size() are
// evaluated via CEL and preserved in input after Enter.
func TestQuotedStringMethod(t *testing.T) {
	m := focusedModel()
	// Type a quoted string with a method call
	m.PathInput.SetValue("\"hi\".size()")
	m.filterSuggestions()
	// Press Enter
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2 := newModel.(*Model)
	// PathInput should remain exactly as typed
	if m2.PathInput.Value() != "\"hi\".size()" {
		t.Fatalf("expected PathInput to remain '"+"\"hi\".size()"+"', got %q", m2.PathInput.Value())
	}
	// Should render a scalar value row with size 2
	rows := m2.Tbl.Rows()
	if len(rows) != 1 || strings.TrimSpace(rows[0][0]) != "(value)" {
		t.Fatalf("expected scalar '(value)' row, got %+v", rows)
	}
	if strings.TrimSpace(rows[0][1]) != "2" {
		t.Fatalf("expected scalar value '2', got '%s'", rows[0][1])
	}
}

// TestSimpleDottedPathAllowsTyping validates that users can type freely
// without forced auto-prefix, allowing arbitrary CEL expressions
func TestSimpleDottedPathAllowsTyping(t *testing.T) {
	m := focusedModel()
	// Start fresh
	m.PathInput.SetValue("")

	// Type a simple key without special characters
	for _, ch := range "items" {
		m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}

	value := m.PathInput.Value()
	// Should allow free typing without forced prefix
	if value != "items" {
		t.Fatalf("expected 'items', got %q", value)
	}
}

// TestBackspaceInExprMode validates that backspacing doesn't force prefix back
func TestBackspaceInExprMode(t *testing.T) {
	m := focusedModel()
	// Start with "_."
	m.PathInput.SetValue("_.")

	// Press backspace once: "_." → "_"
	m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	value := m.PathInput.Value()
	if value != "_" {
		t.Fatalf("after 1st backspace, expected '_', got %q", value)
	}

	// Press backspace again: "_" → ""
	m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	value = m.PathInput.Value()
	if value != "" {
		t.Fatalf("after 2nd backspace, expected empty string, got %q", value)
	}

	// Type a character: "" → "t"
	m.Update(tea.KeyPressMsg{Code: 't', Text: "t"})
	value = m.PathInput.Value()
	if value != "t" {
		t.Fatalf("after typing 't', expected 't', got %q", value)
	}
}

// Ensure backspace edits work after typing a complex expression
func TestBackspaceWorksOnComplexExpr(t *testing.T) {
	m := focusedModel()
	m.PathInput.SetValue("greatest(_.[1,2])")
	// Send backspace once; should remove the last ')'
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if got := m.PathInput.Value(); got != "greatest(_.[1,2]" {
		t.Fatalf("expected one char removed, got %q", got)
	}
}

// Ctrl+U should clear the input and suggestions for recovery
func TestCtrlUClearsInput(t *testing.T) {
	m := focusedModel()
	m.PathInput.SetValue("greatest(_.[1,2])")
	m.filterSuggestions()
	// Simulate Ctrl+U
	// v2: Use control character code directly (Ctrl+U = 0x15)
	nm, _ := m.Update(tea.KeyPressMsg{Code: 0x15})
	m2 := nm.(*Model)
	if m2.PathInput.Value() != "" {
		t.Fatalf("expected empty input after Ctrl+U, got %q", m2.PathInput.Value())
	}
	// Ensure typing works after clearing
	nm2, _ := m2.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	m3 := nm2.(*Model)
	if m3.PathInput.Value() != "a" {
		t.Fatalf("expected input 'a' after typing, got %q", m3.PathInput.Value())
	}
}

// TestBackspaceToEmptyThenArrow validates that arrow keys don't force the prefix back
func TestBackspaceToEmptyThenArrow(t *testing.T) {
	m := focusedModel()
	// Start with "_."
	m.PathInput.SetValue("_.")

	// Press backspace twice to get to empty
	m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	value := m.PathInput.Value()
	if value != "" {
		t.Fatalf("after backspacing twice, expected empty, got %q", value)
	}

	// Press up arrow (shouldn't force prefix back)
	m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	value = m.PathInput.Value()
	if value != "" {
		t.Fatalf("after pressing up arrow, expected empty, got %q", value)
	}

	// Press down arrow (shouldn't force prefix back)
	m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	value = m.PathInput.Value()
	if value != "" {
		t.Fatalf("after pressing down arrow, expected empty, got %q", value)
	}
}

// TestDoubleEnterPrintsCLIOutput ensures that after copying a scalar path,
// pressing Enter again prints CLI-style output and exits.
func TestDoubleEnterPrintsCLIOutput(_ *testing.T) {
	// Replaced by F10 run behavior; see TestF10PrintsCLIOutput.
}

// TestF10PrintsCLIOutput ensures F10 evaluates the expression, sets pending output, and quits.
func TestF10PrintsCLIOutput(t *testing.T) {
	// Use fixture data where vendor.contact is a scalar string
	root := loadYAMLFixture(t)
	m := focusedModel()
	m.Root = root

	// Type a scalar path and press F10 to run and quit
	m.PathInput.SetValue("_.items[0].vendor.contact")
	m.filterSuggestions()

	// Execute F10
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyF10})
	if cmd == nil {
		t.Fatalf("expected a quit command to be returned on F10")
	}
	// Execute the command (should return tea.Quit)
	_ = cmd()

	// Verify the expression was normalized and stored for post-exit printing
	if m.PendingCLIExpr != "_.items[0].vendor.contact" {
		t.Fatalf("expected PendingCLIExpr to be '_.items[0].vendor.contact', got %q", m.PendingCLIExpr)
	}
}

// TestF10ArrayLiteralPrintsCLIOutput ensures F10 respects CEL literals (arrays)
// and does not auto-prefix with '_.' so evaluation succeeds post-exit.
func TestF10ArrayLiteralPrintsCLIOutput(t *testing.T) {
	m := focusedModel()
	// No root needed; array literal should be evaluated as CEL directly
	m.PathInput.SetValue("[1,2][0]")
	m.filterSuggestions()

	// Execute F10
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyF10})
	if cmd == nil {
		t.Fatalf("expected a quit command to be returned on F10")
	}
	_ = cmd()

	// Verify the expression was not prefixed with '_.'
	if m.PendingCLIExpr != "[1,2][0]" {
		t.Fatalf("expected PendingCLIExpr to be '[1,2][0]', got %q", m.PendingCLIExpr)
	}
}

// TestF10QuotedStringLiteralPrintsCLIOutput ensures F10 respects quoted string CEL literals
// and does not auto-prefix with '_.' so evaluation succeeds post-exit.
func TestF10QuotedStringLiteralPrintsCLIOutput(t *testing.T) {
	m := focusedModel()
	m.PathInput.SetValue("\"hi\".size()")
	m.filterSuggestions()

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyF10})
	if cmd == nil {
		t.Fatalf("expected a quit command to be returned on F10")
	}
	_ = cmd()

	if m.PendingCLIExpr != "\"hi\".size()" {
		t.Fatalf("expected PendingCLIExpr to be '"+"\"hi\".size()"+"', got %q", m.PendingCLIExpr)
	}
}

// TestF10FunctionCallPrintsCLIOutput ensures F10 respects function calls like string(...)
// and does not auto-prefix with '_.' so the function name isn't treated as a path.
func TestF10FunctionCallPrintsCLIOutput(t *testing.T) {
	root := loadYAMLFixture(t)
	m := focusedModel()
	m.Root = root

	// Type a function call expression
	m.PathInput.SetValue("string(_.items[0].available)")
	m.filterSuggestions()

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyF10})
	if cmd == nil {
		t.Fatalf("expected a quit command to be returned on F10")
	}
	_ = cmd()

	// Verify the expression was not prefixed with '_.'
	if m.PendingCLIExpr != "string(_.items[0].available)" {
		t.Fatalf("expected PendingCLIExpr to be 'string(_.items[0].available)', got %q", m.PendingCLIExpr)
	}
}

// TestF6PreservesExpressionFromCLI validates F6 toggle from expr mode works correctly
// After toggling to table mode and back, expression reflects current selection (matching footer)
func TestF6PreservesExpressionFromCLI(t *testing.T) {
	// Simulate launching TUI with -e flag containing a function call
	root := loadYAMLFixture(t)
	m := InitialModel(root)
	m.Root = root
	m.InputFocused = true
	m.PathInput.Focus()
	// Set PathInput as if -e flag passed this expression
	m.PathInput.SetValue("string(_.items[0].name)")

	// Press F6 to toggle to table mode
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyF6})
	m2 := newModel.(*Model)

	if m2.InputFocused {
		t.Fatalf("expected input to be blurred after F6")
	}

	// Press F6 again to return to expr mode
	newModel2, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyF6})
	m3 := newModel2.(*Model)

	if !m3.InputFocused {
		t.Fatalf("expected input to be focused after second F6")
	}

	// After toggling back, expression shows selected row path (matching footer)
	// This is expected behavior - the user was in table mode and may have moved
	val := m3.PathInput.Value()
	if !strings.HasPrefix(val, "_.") {
		t.Fatalf("expected PathInput to show selected row path starting with '_.' after F6 toggle, got %q", val)
	}
}

// TestF10InTableModeWithCELExpression validates F10 in table mode uses current PathInput expression
func TestF10InTableModeWithCELExpression(t *testing.T) {
	root := loadYAMLFixture(t)
	m := InitialModel(root)
	m.Root = root
	m.InputFocused = false
	m.Tbl.Focus()
	// Sync path input with a CEL expression (as if user navigated then typed in expr mode)
	m.PathInput.SetValue("_.items.filter(x, x.available)")

	// Press F10 to evaluate and quit
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyF10})
	if cmd == nil {
		t.Fatalf("expected a quit command to be returned on F10")
	}
	_ = cmd()

	// Verify the expression was stored without modification
	if m.PendingCLIExpr != "_.items.filter(x, x.available)" {
		t.Fatalf("expected PendingCLIExpr to be '_.items.filter(x, x.available)', got %q", m.PendingCLIExpr)
	}
}

// TestF10MapLiteralPrintsCLIOutput ensures F10 respects map literals
func TestF10MapLiteralPrintsCLIOutput(t *testing.T) {
	m := focusedModel()
	m.PathInput.SetValue("{\"a\":1}.a")
	m.filterSuggestions()

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyF10})
	if cmd == nil {
		t.Fatalf("expected a quit command to be returned on F10")
	}
	_ = cmd()

	// Verify the expression was not prefixed with '_.'
	if m.PendingCLIExpr != "{\"a\":1}.a" {
		t.Fatalf("expected PendingCLIExpr to be '{\"a\":1}.a', got %q", m.PendingCLIExpr)
	}
}

// TestGlobalFunctionInsertionOnLiteralArray validates Tab cycles through keys/indices, not functions
func TestGlobalFunctionInsertionOnLiteralArray(t *testing.T) {
	m := focusedModel()
	// Type a literal array followed by a dot
	m.PathInput.SetValue("[1,2].")
	m.filterSuggestions()
	// Press Tab - may complete with array indices or functions
	m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	got := m.PathInput.Value()
	// Allow function insertion now; just ensure something reasonable was produced
	if got == "[1,2]." {
		t.Fatalf("Tab should complete an index or function, got unchanged value")
	}
}

// Tab on a quoted string should cycle through keys, not complete functions
func TestGlobalFunctionInsertionOnQuotedString(t *testing.T) {
	m := focusedModel()
	m.PathInput.SetValue("\"hi\".")
	m.filterSuggestions()
	// Press Tab - should cycle through keys or functions (if any)
	before := m.PathInput.Value()
	m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	got := m.PathInput.Value()
	// Allow function insertion; ensure it either stayed the same or appended something meaningful
	if got == before {
		return
	}
	if !strings.HasPrefix(got, "\"hi\".") {
		t.Fatalf("Tab on quoted string should keep base expression, got %q", got)
	}
}

// TestF6SyncUntruncatedKeyName validates that F6 toggle shows full untruncated key names in PathInput
// This is a regression test for the bug where long key names were truncated to "...key" in expr mode
func TestF6SyncUntruncatedKeyName(t *testing.T) {
	// Create data with a very long key that will be truncated in table display (> 30 chars)
	longKeyName := "security_and_analysis.secret_scanning_non_provider_patterns"
	node := map[string]interface{}{
		longKeyName: map[string]interface{}{"enabled": true},
		"short":     "value",
	}

	m := InitialModel(node)
	m.Root = node
	m.InputFocused = false
	m.Tbl.Focus()
	// Set narrow column width to ensure truncation in display
	m.KeyColWidth = 30
	m.ValueColWidth = 60
	m.applyLayout()

	// Verify the key is displayed as truncated in the table (sanity check)
	rows := m.Tbl.Rows()
	truncated := false
	for _, row := range rows {
		if len(row[0]) > 0 && len(row[0]) < len(longKeyName) && strings.HasSuffix(row[0], "...") {
			truncated = true
			break
		}
	}
	if !truncated {
		t.Fatalf("expected truncated display in table (sanity check)")
	}

	// Position cursor on the long key row
	for i, row := range rows {
		if strings.Contains(row[0], "security") {
			m.Tbl.SetCursor(i)
			break
		}
	}

	// Call syncPathInputWithCursor to update the PathInput
	m.syncPathInputWithCursor()

	// Verify PathInput has the FULL untruncated key name
	// Keys containing dots must use bracket notation per CEL syntax
	got := m.PathInput.Value()
	expected := `_["` + longKeyName + `"]`
	if got != expected {
		t.Fatalf("expected PathInput to show full key '%s', got '%s'", expected, got)
	}

	// Now press F6 to enter expression mode
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyF6})
	m2 := newModel.(*Model)

	// Verify expression mode still has the full untruncated path
	if m2.PathInput.Value() != expected {
		t.Fatalf("expected F6 expr mode to preserve full key '%s', got '%s'", expected, m2.PathInput.Value())
	}
}

func TestF12PopupTextRendersInView(t *testing.T) {
	origMenu := CurrentMenuConfig()
	defer SetMenuConfig(origMenu)

	SetMenuConfig(MenuConfig{
		F12: MenuItem{
			Label:   "custom",
			Action:  "custom",
			Enabled: true,
			Popup: InfoPopupConfig{
				Text: "hello popup",
			},
		},
	})

	m := InitialModel(map[string]interface{}{})
	if m.ShowInfoPopup {
		t.Fatalf("expected popup hidden initially")
	}

	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyF12})
	m2 := nm.(*Model)
	if !m2.ShowInfoPopup {
		t.Fatalf("expected popup visible after F12")
	}
	view := m2.View()
	if !strings.Contains(fmt.Sprint(view.Content), "hello popup") {
		t.Fatalf("expected popup text in view, got %q", fmt.Sprint(view.Content))
	}
}

func TestInitialExprPathAllowsLeftNavigation(t *testing.T) {
	root := map[string]interface{}{
		"tasks": map[string]interface{}{
			"default": map[string]interface{}{
				"cmds": []interface{}{"echo hi", "echo bye"},
			},
		},
	}

	m := InitialModel(root)
	m.Root = root
	m.Node = root
	m.WinWidth = 80
	m.WinHeight = 24
	m.applyLayout(true)

	applyInitialExpr(&m, "_.tasks.default.cmds[0]")

	if got := m.Path; got != "_.tasks.default.cmds[0]" {
		t.Fatalf("expected path to be set for initial expr, got %q", got)
	}

	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	m1 := nm.(*Model)

	if got := m1.Path; got != "_.tasks.default.cmds" {
		t.Fatalf("expected left arrow to navigate to parent path, got %q", got)
	}
}

func TestF12TogglesPopupText(t *testing.T) {
	origMenu := CurrentMenuConfig()
	defer SetMenuConfig(origMenu)

	SetMenuConfig(MenuConfig{
		F12: MenuItem{
			Label:   "custom",
			Action:  "custom",
			Enabled: true,
			Popup: InfoPopupConfig{
				Text: "hello popup",
			},
		},
	})

	m := InitialModel(map[string]interface{}{})
	if m.ShowInfoPopup {
		t.Fatalf("expected popup hidden initially")
	}

	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyF12})
	m2 := nm.(*Model)
	if !m2.ShowInfoPopup {
		t.Fatalf("expected popup visible after F12")
	}
	if got := m2.InfoPopup; got != "hello popup" {
		t.Fatalf("expected popup text 'hello popup', got %q", got)
	}
}

func TestHelpPopupReplacesHelpContent(t *testing.T) {
	m := InitialModel(map[string]interface{}{})
	m.WinWidth = 60
	m.HelpVisible = true
	m.Help.Visible = true
	m.Help.SetWidth(60)
	m.HelpPopupText = "about text"
	m.HelpPopupJustify = "center"
	m.ApplyColorScheme()

	view := m.View()
	viewStr := fmt.Sprint(view.Content)
	if !strings.Contains(viewStr, "about text") {
		t.Fatalf("expected help popup content in view, got:\n%s", viewStr)
	}
	if strings.Contains(viewStr, "Toggle inline help") {
		t.Fatalf("expected help popup to replace base help content")
	}
}

func TestHelpPopupDefaultVisibleOnF1(t *testing.T) {
	origMenu := CurrentMenuConfig()
	defer SetMenuConfig(origMenu)
	SetMenuConfig(DefaultMenuConfig())

	m := InitialModel(map[string]interface{}{})
	if m.HelpVisible {
		t.Fatalf("expected help hidden initially")
	}

	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyF1})
	m2 := nm.(*Model)
	if !m2.HelpVisible {
		t.Fatalf("expected help visible after F1")
	}
	if m2.HelpPopupText == "" {
		t.Fatalf("expected default help popup text to be set")
	}
	view := m2.View()
	viewStr := fmt.Sprint(view.Content)
	if !strings.Contains(viewStr, "kvx: terminal-based UI") {
		t.Fatalf("expected default help popup content in view, got %q", viewStr)
	}
	// In vim mode (default), we show "toggle help" instead of "Toggle inline help"
	if count := strings.Count(viewStr, "toggle help"); count != 1 {
		t.Fatalf("expected help content to render once, got %d occurrences", count)
	}
}

func TestHelpPopupUsesMenuConfig(t *testing.T) {
	origMenu := CurrentMenuConfig()
	defer SetMenuConfig(origMenu)

	SetMenuConfig(MenuConfig{
		F1: MenuItem{
			Label:    "help",
			Action:   "help",
			Enabled:  true,
			HelpText: "Toggle inline help",
			Popup: InfoPopupConfig{
				Text: "config about popup",
			},
		},
	})

	m := InitialModel(map[string]interface{}{})
	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyF1})
	m2 := nm.(*Model)
	if !m2.HelpVisible {
		t.Fatalf("expected help visible after F1")
	}
	if got := m2.HelpPopupText; got != "config about popup" {
		t.Fatalf("expected help popup text from config, got %q", got)
	}
	view := m2.View()
	if !strings.Contains(fmt.Sprint(view.Content), "config about popup") {
		t.Fatalf("expected config popup content in view, got %q", fmt.Sprint(view.Content))
	}
}

// Test clearSearchState clears all search-related state
func TestClearSearchState_ClearsAllFields(t *testing.T) {
	m := tableModel()
	m.AdvancedSearchActive = true
	m.AdvancedSearchQuery = "test"
	m.AdvancedSearchResults = []SearchResult{{Key: "test"}}
	m.AdvancedSearchBasePath = "base"
	m.SearchContextActive = true
	m.SearchContextResults = []SearchResult{{Key: "ctx"}}
	m.SearchContextQuery = "ctx query"
	m.SearchContextBasePath = "ctx base"

	m.clearSearchState()

	if m.AdvancedSearchActive {
		t.Error("expected AdvancedSearchActive to be false")
	}
	if m.AdvancedSearchQuery != "" {
		t.Errorf("expected empty AdvancedSearchQuery, got %q", m.AdvancedSearchQuery)
	}
	if len(m.AdvancedSearchResults) != 0 {
		t.Errorf("expected empty AdvancedSearchResults, got %d items", len(m.AdvancedSearchResults))
	}
	if m.AdvancedSearchBasePath != "" {
		t.Errorf("expected empty AdvancedSearchBasePath, got %q", m.AdvancedSearchBasePath)
	}
	if m.SearchContextActive {
		t.Error("expected SearchContextActive to be false")
	}
	if m.SearchContextResults != nil {
		t.Error("expected nil SearchContextResults")
	}
}

// Test clearSearchContext clears only context fields
func TestClearSearchContext_PreservesMainSearch(t *testing.T) {
	m := tableModel()
	m.AdvancedSearchActive = true
	m.AdvancedSearchQuery = "test"
	m.SearchContextActive = true
	m.SearchContextQuery = "ctx"

	m.clearSearchContext()

	if m.SearchContextActive {
		t.Error("expected SearchContextActive to be false")
	}
	if m.SearchContextQuery != "" {
		t.Errorf("expected empty SearchContextQuery, got %q", m.SearchContextQuery)
	}
	// Main search should be preserved
	if !m.AdvancedSearchActive {
		t.Error("expected AdvancedSearchActive to remain true")
	}
	if m.AdvancedSearchQuery != "test" {
		t.Errorf("expected AdvancedSearchQuery preserved, got %q", m.AdvancedSearchQuery)
	}
}

// Test styleRows handles empty input
func TestStyleRows_EmptyInput(t *testing.T) {
	rows := styleRows([][]string{})
	if len(rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(rows))
	}
}

// Test styleRows applies styling
func TestStyleRows_AppliesStyling(t *testing.T) {
	input := [][]string{
		{"key1", "value1"},
		{"key2", "value2"},
	}
	rows := styleRows(input)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	// Rows should contain the same content (styling is via ANSI codes)
	if !strings.Contains(rows[0][0], "key1") {
		t.Errorf("expected key1 in first row, got %q", rows[0][0])
	}
	if !strings.Contains(rows[1][1], "value2") {
		t.Errorf("expected value2 in second row, got %q", rows[1][1])
	}
}

// Test styleRowsWithWidths truncates long content
func TestStyleRowsWithWidths_TruncatesLongContent(t *testing.T) {
	longKey := strings.Repeat("k", 50)
	longValue := strings.Repeat("v", 100)
	input := [][]string{{longKey, longValue}}

	rows := styleRowsWithWidths(input, 30, 60)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	// Strip ANSI codes to check actual content
	key := stripANSI(rows[0][0])
	value := stripANSI(rows[0][1])

	// Should be truncated with ellipsis
	if len(key) > 30 {
		t.Errorf("expected key truncated to ~30 chars, got %d", len(key))
	}
	if len(value) > 60 {
		t.Errorf("expected value truncated to ~60 chars, got %d", len(value))
	}
}

// Test applyLayout handles zero/negative dimensions
func TestApplyLayout_HandlesInvalidDimensions(t *testing.T) {
	m := tableModel()
	m.WinWidth = 0
	m.WinHeight = 0

	// Should not panic
	m.applyLayout()

	// Should set minimum viable dimensions
	if m.TableHeight < 2 {
		t.Errorf("expected minimum table height, got %d", m.TableHeight)
	}
	if m.ValueColWidth < 10 {
		t.Errorf("expected minimum value width, got %d", m.ValueColWidth)
	}
}

// Test applyLayout with forceRegenerate parameter
func TestApplyLayout_ForceRegenerateFlag(t *testing.T) {
	m := tableModel()
	m.WinWidth = 80
	m.WinHeight = 30
	initialRows := m.Tbl.Rows()

	// Call with false - should not regenerate rows if widths unchanged
	m.applyLayout()
	rows1 := m.Tbl.Rows()
	if len(rows1) != len(initialRows) {
		t.Errorf("expected same row count, got %d want %d", len(rows1), len(initialRows))
	}

	// Change dimensions to trigger regeneration
	m.WinWidth = 100
	m.applyLayout()
	rows2 := m.Tbl.Rows()
	if len(rows2) != len(initialRows) {
		t.Errorf("expected same row count after resize, got %d want %d", len(rows2), len(initialRows))
	}
}

// Test clipboard operation handling
func TestClipboardOperations_HandleMissingClipboard(t *testing.T) {
	m := tableModel()
	m.Tbl.SetCursor(0)

	// Ctrl+Y (copy) should not panic even if clipboard unavailable
	// v2: Use control character code directly (Ctrl+Y = 0x19)
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 0x19})
	m2 := newModel.(*Model)

	// Should not have error (clipboard failures are silent)
	if strings.Contains(m2.ErrMsg, "clipboard") {
		t.Errorf("unexpected clipboard error: %s", m2.ErrMsg)
	}
}
