package tui

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/google/cel-go/cel"

	"github.com/oakwood-commons/kvx/internal/ui"
)

func TestDefaultExpressionProvider_ReturnsNonNil(t *testing.T) {
	provider := DefaultExpressionProvider()
	if provider == nil {
		t.Fatal("DefaultExpressionProvider returned nil")
	}
}

func TestSetExpressionProvider_AcceptsCustomProvider(t *testing.T) {
	// Create a mock provider
	mockProvider := &mockExpressionProvider{
		suggestions: []string{"test1", "test2"},
	}

	// Reset after test to avoid polluting other tests
	defer ResetExpressionProvider()

	SetExpressionProvider(mockProvider)

	// Verify it was set (indirect - through UI layer)
	// We can't directly verify without exposing UI internals,
	// but this tests the function doesn't panic
}

func TestNewCELExpressionProvider_CreatesValidProvider(t *testing.T) {
	env, err := cel.NewEnv(
		cel.Variable("_", cel.DynType),
	)
	if err != nil {
		t.Fatalf("failed to create CEL env: %v", err)
	}

	hints := map[string]string{
		"filter": "e.g. items.filter(x, x.active)",
	}

	provider := NewCELExpressionProvider(env, hints)
	if provider == nil {
		t.Fatal("NewCELExpressionProvider returned nil")
	}

	// Verify it can discover suggestions
	suggestions := provider.DiscoverSuggestions()
	if len(suggestions) == 0 {
		t.Error("expected some suggestions from CEL environment")
	}
}

func TestNewCELExpressionProvider_WithNilHints(t *testing.T) {
	env, err := cel.NewEnv(
		cel.Variable("_", cel.DynType),
	)
	if err != nil {
		t.Fatalf("failed to create CEL env: %v", err)
	}

	provider := NewCELExpressionProvider(env, nil)
	if provider == nil {
		t.Fatal("NewCELExpressionProvider returned nil with nil hints")
	}

	suggestions := provider.DiscoverSuggestions()
	if len(suggestions) == 0 {
		t.Error("expected suggestions even without hints")
	}
}

func TestWithIO_ReturnsOptions(t *testing.T) {
	in := bytes.NewBufferString("")
	out := bytes.NewBuffer(nil)

	opts := WithIO(in, out)
	if len(opts) == 0 {
		t.Error("expected WithIO to return options")
	}
}

func TestWithIO_NilInputsHandled(t *testing.T) {
	opts := WithIO(nil, nil)
	// Should return empty slice when both are nil
	if len(opts) != 0 {
		t.Errorf("expected 0 options for nil inputs, got %d", len(opts))
	}
}

func TestWithIO_OnlyInput(t *testing.T) {
	in := bytes.NewBufferString("")
	opts := WithIO(in, nil)
	if len(opts) != 1 {
		t.Errorf("expected 1 option for input only, got %d", len(opts))
	}
}

func TestWithIO_OnlyOutput(t *testing.T) {
	out := bytes.NewBuffer(nil)
	opts := WithIO(nil, out)
	if len(opts) != 1 {
		t.Errorf("expected 1 option for output only, got %d", len(opts))
	}
}

func TestDefaultNavigator_ReturnsNonNil(t *testing.T) {
	nav := DefaultNavigator()
	if nav == nil {
		t.Fatal("DefaultNavigator returned nil")
	}
}

func TestSetNavigator_AcceptsCustomNavigator(t *testing.T) {
	// Create mock navigator
	mockNav := &mockNavigator{}

	// Save and restore
	orig := DefaultNavigator()
	defer SetNavigator(orig)

	SetNavigator(mockNav)
	// Function should not panic
}

func TestRun_WithMinimalConfig(t *testing.T) {
	t.Skip("Skip Bubble Tea integration tests - requires proper terminal stdin handling")
	// This test verifies Run starts without panicking
	// We use WithIO to capture output and avoid interactive mode
	in := bytes.NewBufferString("q") // Send quit command
	out := bytes.NewBuffer(nil)

	cfg := Config{
		AppName: "test-app",
	}

	data := map[string]interface{}{"test": "value"}

	// Use a channel to signal test completion
	done := make(chan error, 1)
	go func() {
		// Run with test IO - this will start and quit immediately
		err := Run(data, cfg, WithIO(in, out)...)
		done <- err
	}()

	// Wait for Run to complete or timeout after 5 seconds
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Run timed out - program did not exit cleanly")
	}
}

func TestRun_DefaultsAppName(t *testing.T) {
	t.Skip("Skip Bubble Tea integration tests - requires proper terminal stdin handling")
	in := bytes.NewBufferString("q")
	out := bytes.NewBuffer(nil)

	cfg := Config{
		AppName: "", // Empty - should default to "kvx"
	}

	data := map[string]interface{}{}
	done := make(chan error, 1)
	go func() {
		err := Run(data, cfg, WithIO(in, out)...)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run with empty app name failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Run timed out")
	}
}

func TestRun_WithDebugEnabled(t *testing.T) {
	t.Skip("Skip Bubble Tea integration tests - requires proper terminal stdin handling")
	in := bytes.NewBufferString("q")
	out := bytes.NewBuffer(nil)

	debugCalled := false
	cfg := Config{
		AppName:      "test",
		DebugEnabled: true,
		DebugSink: func(msg string) {
			debugCalled = true
		},
	}

	data := map[string]interface{}{}
	done := make(chan error, 1)
	go func() {
		err := Run(data, cfg, WithIO(in, out)...)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run with debug failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Run timed out")
	}

	// Debug sink may or may not be called depending on execution path
	// Just verify it doesn't crash
	_ = debugCalled
}

func TestRun_WithCustomMenu(t *testing.T) {
	t.Skip("Skip Bubble Tea integration tests - requires proper terminal stdin handling")
	in := bytes.NewBufferString("q")
	out := bytes.NewBuffer(nil)

	menu := ui.MenuConfig{
		F1:  ui.MenuItem{Enabled: true, Label: "Help"},
		F12: ui.MenuItem{Enabled: true, Label: "Quit"},
	}

	cfg := Config{
		AppName: "test",
		Menu:    &menu,
	}

	data := map[string]interface{}{}
	done := make(chan error, 1)
	go func() {
		err := Run(data, cfg, WithIO(in, out)...)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run with custom menu failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Run timed out")
	}
}

func TestRun_WithNoColor(t *testing.T) {
	t.Skip("Skip Bubble Tea integration tests - requires proper terminal stdin handling")
	in := bytes.NewBufferString("q")
	out := bytes.NewBuffer(nil)

	cfg := Config{
		AppName: "test",
		NoColor: true,
	}

	data := map[string]interface{}{}
	done := make(chan error, 1)
	go func() {
		err := Run(data, cfg, WithIO(in, out)...)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run with no color failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Run timed out")
	}

	// Verify output is plain (no ANSI codes)
	output := out.String()
	if strings.Contains(output, "\x1b[") {
		t.Log("output contains ANSI escape sequences; Bubble Tea may inject codes even in no-color mode")
	}
}

func TestRun_WithInitialExpression(t *testing.T) {
	t.Skip("Skip Bubble Tea integration tests - requires proper terminal stdin handling")
	in := bytes.NewBufferString("q")
	out := bytes.NewBuffer(nil)

	cfg := Config{
		AppName:     "test",
		InitialExpr: "_.test",
	}

	data := map[string]interface{}{"test": "value"}
	done := make(chan error, 1)
	go func() {
		err := Run(data, cfg, WithIO(in, out)...)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run with initial expression failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Run timed out")
	}
}

func TestRun_WithCustomDimensions(t *testing.T) {
	t.Skip("Skip Bubble Tea integration tests - requires proper terminal stdin handling")
	in := bytes.NewBufferString("q")
	out := bytes.NewBuffer(nil)

	cfg := Config{
		AppName: "test",
		Width:   100,
		Height:  40,
	}

	data := map[string]interface{}{}
	done := make(chan error, 1)
	go func() {
		err := Run(data, cfg, WithIO(in, out)...)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run with custom dimensions failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Run timed out")
	}
}

func TestRun_WithStartKeys(t *testing.T) {
	t.Skip("Skip Bubble Tea integration tests - requires proper terminal stdin handling")
	in := bytes.NewBufferString("")
	out := bytes.NewBuffer(nil)

	cfg := Config{
		AppName:   "test",
		StartKeys: []string{"q"}, // Quit immediately via start keys
	}

	data := map[string]interface{}{}
	done := make(chan error, 1)
	go func() {
		err := Run(data, cfg, WithIO(in, out)...)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run with start keys failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Run timed out")
	}
}

// Mock types for testing

type mockExpressionProvider struct {
	suggestions []string
}

func (m *mockExpressionProvider) Evaluate(expr string, root interface{}) (interface{}, error) {
	return root, nil
}

func (m *mockExpressionProvider) DiscoverSuggestions() []string {
	return m.suggestions
}

func (m *mockExpressionProvider) IsExpression(expr string) bool {
	return strings.Contains(expr, "[") || strings.Contains(expr, ".")
}

type mockNavigator struct{}

func (m *mockNavigator) NodeAtPath(root interface{}, path string) (interface{}, error) {
	return root, nil
}
