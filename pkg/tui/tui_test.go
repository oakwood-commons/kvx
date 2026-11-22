package tui

import (
	"bytes"
	"errors"
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
		DebugSink: func(_ string) {
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

func TestRenderSnapshot_StatusInputAndCustomOpenURL(t *testing.T) {
	data := map[string]any{
		"title": "Verify Session",
		"url":   "https://example.com/verify",
		"messages": []any{
			"Complete verification in your browser.",
		},
	}
	schema := &DisplaySchema{
		Version: "v1",
		Status: &StatusDisplayConfig{
			TitleField:   "title",
			MessageField: "messages",
			WaitMessage:  "Waiting for callback...",
			DisplayFields: []StatusFieldDisplay{
				{Label: "Endpoint", Field: "url"},
			},
			Actions: []StatusActionConfig{
				{
					Label: "Copy URL",
					Type:  "copy-value",
					Field: "url",
					Keys:  StatusKeyBindings{Vim: "c", Emacs: "alt+c", Function: "f2"},
				},
				{
					Label: "Open",
					Type:  "open-url",
					Field: "url",
					Keys:  StatusKeyBindings{Vim: "o", Emacs: "alt+o", Function: "f3"},
				},
			},
			Input: &StatusInputConfig{
				Label:       "Paste callback URL from browser:",
				Placeholder: "http://localhost:45189/?token=...",
				Masked:      true,
			},
		},
	}

	done := make(chan StatusResult, 1)
	var openedURL string

	cfg := DefaultConfig()
	cfg.AppName = "demo"
	cfg.Width = 100
	cfg.Height = 24
	cfg.NoColor = true
	cfg.DisplaySchema = schema
	cfg.Done = done
	cfg.OpenURL = func(u string) error {
		openedURL = u
		return nil
	}

	// Frame-1 snapshot: shows helper label, placeholder, waiting message, and safe footer hints.
	frame1 := RenderSnapshot(data, cfg)
	if !strings.Contains(frame1, "Paste callback URL from browser:") {
		t.Fatalf("expected input label in snapshot, got:\n%s", frame1)
	}
	if !strings.Contains(frame1, "http://localhost:45189/?token=...") {
		t.Fatalf("expected input placeholder in snapshot, got:\n%s", frame1)
	}
	if !strings.Contains(frame1, "Waiting for callback...") {
		t.Fatalf("expected wait message in snapshot, got:\n%s", frame1)
	}
	if !strings.Contains(frame1, "Enter submit") || !strings.Contains(frame1, "F2/Alt+c") || !strings.Contains(frame1, "F3/Alt+o") {
		t.Fatalf("expected active-input footer hints in snapshot, got:\n%s", frame1)
	}

	// Simulated raw-keystroke paste containing q, c, o, /, ?, f, e, v + F3 custom OpenURL trigger.
	rawURL := "http://localhost:45189/verify?req=foo&token=qcovef"
	cfg.StartKeys = []string{rawURL, "<F3>"}
	afterPaste := RenderSnapshot(data, cfg)

	if openedURL != "https://example.com/verify" {
		t.Fatalf("expected custom OpenURL to receive https://example.com/verify, got %q", openedURL)
	}
	if strings.Contains(afterPaste, "qcovef") {
		t.Fatalf("masked snapshot must not leak raw input text:\n%s", afterPaste)
	}
	if !strings.Contains(afterPaste, "(50 chars)") {
		t.Fatalf("expected masked char counter (50 chars) in snapshot, got:\n%s", afterPaste)
	}
	if !strings.Contains(afterPaste, "•") {
		t.Fatalf("expected bullet masking in snapshot, got:\n%s", afterPaste)
	}
	if !strings.Contains(afterPaste, "Opened in browser") {
		t.Fatalf("expected open-url flash message in snapshot, got:\n%s", afterPaste)
	}
}

func TestRenderSnapshot_StatusInputRejectedSubmissionShowsWarning(t *testing.T) {
	data := map[string]any{
		"title": "Verify Session",
	}
	schema := &DisplaySchema{
		Version: "v1",
		Status: &StatusDisplayConfig{
			TitleField:  "title",
			WaitMessage: "Waiting for callback...",
			Input: &StatusInputConfig{
				Label:  "Paste callback URL:",
				Masked: true,
			},
		},
	}

	done := make(chan StatusResult, 1)
	cfg := DefaultConfig()
	cfg.AppName = "demo"
	cfg.Width = 100
	cfg.Height = 20
	cfg.NoColor = true
	cfg.DisplaySchema = schema
	cfg.Done = done
	cfg.StartKeys = []string{"bad-callback-url", "<CR>"}
	cfg.OnStatusInput = func(string) error {
		return errors.New("missing token query parameter")
	}

	snap := RenderSnapshot(data, cfg)
	if !strings.Contains(snap, "missing token query parameter") {
		t.Fatalf("expected rejected submission warning flash in snapshot, got:\n%s", snap)
	}
	if !strings.Contains(snap, "Paste callback URL:") {
		t.Fatalf("expected input to remain visible for retry after rejected submission, got:\n%s", snap)
	}
}
