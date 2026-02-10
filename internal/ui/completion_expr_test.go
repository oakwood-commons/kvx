package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/oakwood-commons/kvx/internal/completion"
)

func TestExprDotUsesEvaluatedType(t *testing.T) {
	root := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"name": "abc"},
		},
	}

	m := InitialModel(root)
	m.Root = root
	m.Node = root
	m.AllowIntellisense = true
	m.AllowSuggestions = true
	if provider, err := completion.NewCELProvider(); err == nil {
		m.CompletionEngine = completion.NewEngine(provider)
	}
	m.InputFocused = true
	m.PathInput.SetValue("_.items.map(x,x.name)[0]")
	m.PathInput.SetCursor(len(m.PathInput.Value()))

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	nm := updated.(*Model)
	provider, err := completion.NewCELProvider()
	if err != nil {
		t.Fatalf("failed to create CEL provider: %v", err)
	}
	nm.CompletionEngine = completion.NewEngine(provider)
	if nm.ExprType == "" {
		t.Fatalf("ExprType empty after enter (PathInput=%q, Node=%T)", nm.PathInput.Value(), nm.Node)
	}

	nm.PathInput.SetValue(nm.PathInput.Value() + ".")
	nm.PathInput.SetCursor(len(nm.PathInput.Value()))

	ctx := completion.CompletionContext{
		CurrentNode:          nm.Root,
		CurrentType:          nm.ExprType,
		ExpressionResultType: nm.ExprType,
		IsAfterDot:           true,
	}
	nm.Status.Completions = provider.FilterCompletions(nm.PathInput.Value(), ctx)

	if nm.ExprType != "string" {
		t.Fatalf("expected ExprType to be string, got %q (PathInput=%q, Node=%T)", nm.ExprType, nm.PathInput.Value(), nm.Node)
	}

	foundStringFn := false
	for _, c := range nm.Status.Completions {
		if strings.Contains(strings.ToLower(c.Detail), "string.") {
			foundStringFn = true
			break
		}
	}

	if !foundStringFn {
		t.Fatalf("expected string functions after trailing dot, got completions: %#v", nm.Status.Completions)
	}
}

func TestExprDotAfterSizeShowsNumberFuncs(t *testing.T) {
	root := map[string]interface{}{
		"items": []interface{}{1, 2, 3},
	}

	m := InitialModel(root)
	m.Root = root
	m.Node = root
	m.AllowIntellisense = true
	m.AllowSuggestions = true
	if provider, err := completion.NewCELProvider(); err == nil {
		m.CompletionEngine = completion.NewEngine(provider)
	}
	m.InputFocused = true
	m.PathInput.SetValue("_.items.size()")
	m.PathInput.SetCursor(len(m.PathInput.Value()))

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	nm := updated.(*Model)
	provider, err := completion.NewCELProvider()
	if err != nil {
		t.Fatalf("failed to create CEL provider: %v", err)
	}
	nm.CompletionEngine = completion.NewEngine(provider)
	if nm.ExprType == "" {
		t.Fatalf("ExprType empty after enter (PathInput=%q, Node=%T)", nm.PathInput.Value(), nm.Node)
	}

	nm.PathInput.SetValue(nm.PathInput.Value() + ".")
	nm.PathInput.SetCursor(len(nm.PathInput.Value()))
	ctx := completion.CompletionContext{
		CurrentNode:          nm.Root,
		CurrentType:          nm.ExprType,
		ExpressionResultType: nm.ExprType,
		IsAfterDot:           true,
	}
	nm.Status.Completions = provider.FilterCompletions(nm.PathInput.Value(), ctx)

	if nm.ExprType != "int" {
		t.Fatalf("expected ExprType to be int, got %q (PathInput=%q, Node=%T)", nm.ExprType, nm.PathInput.Value(), nm.Node)
	}

	foundNumberFn := false
	for _, c := range nm.Status.Completions {
		if c.Function != nil && strings.Contains(strings.ToLower(c.Function.Name), "abs") {
			foundNumberFn = true
			break
		}
	}

	if !foundNumberFn {
		t.Fatalf("expected numeric functions after trailing dot, got completions: %#v (ExprType=%q PathInput=%q)", nm.Status.Completions, nm.ExprType, nm.PathInput.Value())
	}
}

func TestExprDotAfterRootSizeShowsNumberFuncs(t *testing.T) {
	root := map[string]interface{}{
		"a": 1,
	}

	m := InitialModel(root)
	m.Root = root
	m.Node = root
	m.AllowIntellisense = true
	m.AllowSuggestions = true
	if provider, err := completion.NewCELProvider(); err == nil {
		m.CompletionEngine = completion.NewEngine(provider)
	}
	m.InputFocused = true
	m.PathInput.SetValue("_.size()")
	m.PathInput.SetCursor(len(m.PathInput.Value()))

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	nm := updated.(*Model)
	provider, err := completion.NewCELProvider()
	if err != nil {
		t.Fatalf("failed to create CEL provider: %v", err)
	}
	nm.CompletionEngine = completion.NewEngine(provider)
	if nm.ExprType == "" {
		t.Fatalf("ExprType empty after enter (PathInput=%q, Node=%T)", nm.PathInput.Value(), nm.Node)
	}

	nm.PathInput.SetValue(nm.PathInput.Value() + ".")
	nm.PathInput.SetCursor(len(nm.PathInput.Value()))
	ctx := completion.CompletionContext{
		CurrentNode:          nm.Root,
		CurrentType:          nm.ExprType,
		ExpressionResultType: nm.ExprType,
		IsAfterDot:           true,
	}
	nm.Status.Completions = provider.FilterCompletions(nm.PathInput.Value(), ctx)

	if nm.ExprType != "int" {
		t.Fatalf("expected ExprType to be int, got %q (PathInput=%q, Node=%T)", nm.ExprType, nm.PathInput.Value(), nm.Node)
	}

	foundNumberFn := false
	for _, c := range nm.Status.Completions {
		if c.Function != nil && strings.Contains(strings.ToLower(c.Function.Name), "abs") {
			foundNumberFn = true
			break
		}
	}

	if !foundNumberFn {
		t.Fatalf("expected numeric functions after trailing dot on root size, got completions: %#v", m.Status.Completions)
	}
}
