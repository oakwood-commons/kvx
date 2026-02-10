package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// Mock ChildModel for testing
type mockChild struct {
	id          string
	title       string
	initCalled  bool
	updateCalls int
	viewCalls   int
	lastMsg     tea.Msg
	focused     bool
	width       int
	height      int
}

func newMockChild(id, title string) *mockChild {
	return &mockChild{
		id:    id,
		title: title,
	}
}

func (m *mockChild) Init() tea.Cmd {
	m.initCalled = true
	return nil
}

func (m *mockChild) Update(msg tea.Msg) (ChildModel, tea.Cmd) {
	m.updateCalls++
	m.lastMsg = msg
	return m, nil
}

func (m *mockChild) View() string {
	m.viewCalls++
	return m.title + " view"
}

func (m *mockChild) ID() string {
	return m.id
}

func (m *mockChild) Title() string {
	return m.title
}

func (m *mockChild) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *mockChild) Focus() tea.Cmd {
	m.focused = true
	return nil
}

func (m *mockChild) Blur() {
	m.focused = false
}

func (m *mockChild) Focused() bool {
	return m.focused
}

// Mock Maker for testing
type mockMaker struct {
	created map[string]*mockChild
}

func newMockMaker() *mockMaker {
	return &mockMaker{
		created: make(map[string]*mockChild),
	}
}

func (m *mockMaker) Make(id string, width, height int) (ChildModel, tea.Cmd) {
	child := newMockChild(id, "Mock "+id)
	child.SetSize(width, height)
	m.created[id] = child
	return child, child.Init()
}

// TestRootModelCreation tests basic RootModel initialization
func TestRootModelCreation(t *testing.T) {
	maker := newMockMaker()
	initialChild := newMockChild("initial", "Initial")
	m := NewRootModel(initialChild, maker)

	if m.mode != NormalMode {
		t.Errorf("expected NormalMode, got %v", m.mode)
	}

	if m.current != initialChild {
		t.Errorf("expected initialChild as current, got %v", m.current)
	}

	if len(m.stack) != 0 {
		t.Errorf("expected empty navigation stack, got %d items", len(m.stack))
	}
}

// TestRootModelInit tests Init() initialization
func TestRootModelInit(t *testing.T) {
	maker := newMockMaker()
	initialChild := newMockChild("initial", "Initial")
	m := NewRootModel(initialChild, maker)

	cmd := m.Init()
	if cmd != nil {
		t.Errorf("expected nil cmd from Init, got %v", cmd)
	}
}

// TestModeTransitions tests mode switching
func TestModeTransitions(t *testing.T) {
	tests := []struct {
		name     string
		fromMode Mode
		toMode   Mode
	}{
		{"Normal to Expr", NormalMode, ExprMode},
		{"Expr to Normal", ExprMode, NormalMode},
		{"Normal to Search", NormalMode, SearchMode},
		{"Search to Normal", SearchMode, NormalMode},
		{"Normal to Help", NormalMode, HelpMode},
		{"Help to Normal", HelpMode, NormalMode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maker := newMockMaker()
			initialChild := newMockChild("initial", "Initial")
			m := NewRootModel(initialChild, maker)
			m.SetMode(tt.fromMode)

			if m.Mode() != tt.fromMode {
				t.Errorf("failed to set initial mode: expected %v, got %v", tt.fromMode, m.Mode())
			}

			m.SetMode(tt.toMode)

			if m.Mode() != tt.toMode {
				t.Errorf("failed to transition: expected %v, got %v", tt.toMode, m.Mode())
			}
		})
	}
}

// TestNavigationStack tests NavigateTo and NavigateBack
func TestNavigationStack(t *testing.T) {
	maker := newMockMaker()
	initialChild := newMockChild("initial", "Initial")
	m := NewRootModel(initialChild, maker)

	// Initially, can't navigate back
	if m.CanNavigateBack() {
		t.Error("expected CanNavigateBack to be false initially")
	}

	// Navigate to first child
	child1 := newMockChild("child1", "Child 1")
	m.NavigateTo(child1)

	if m.current != child1 {
		t.Error("expected current to be child1")
	}

	if !m.CanNavigateBack() {
		t.Error("expected CanNavigateBack to be true after navigation")
	}

	if len(m.stack) != 1 {
		t.Errorf("expected stack length 1, got %d", len(m.stack))
	}

	// Navigate to second child
	child2 := newMockChild("child2", "Child 2")
	m.NavigateTo(child2)

	if m.current != child2 {
		t.Error("expected current to be child2")
	}

	if len(m.stack) != 2 {
		t.Errorf("expected stack length 2, got %d", len(m.stack))
	}

	// Navigate back
	m.NavigateBack()

	if m.current != child1 {
		t.Error("expected current to be child1 after NavigateBack")
	}

	if len(m.stack) != 1 {
		t.Errorf("expected stack length 1 after back, got %d", len(m.stack))
	}

	// Navigate back again
	m.NavigateBack()

	if m.current != initialChild {
		t.Error("expected current to be initialChild after second NavigateBack")
	}

	if len(m.stack) != 0 {
		t.Errorf("expected empty stack, got %d", len(m.stack))
	}

	if m.CanNavigateBack() {
		t.Error("expected CanNavigateBack to be false after clearing stack")
	}
}

// TestMessageRoutingNormalMode tests that messages route to current child in NormalMode
func TestMessageRoutingNormalMode(t *testing.T) {
	maker := newMockMaker()
	initialChild := newMockChild("initial", "Initial")
	m := NewRootModel(initialChild, maker)
	m.SetMode(NormalMode)

	child := newMockChild("test", "Test Child")
	m.NavigateTo(child)

	// Send a key message
	keyMsg := tea.KeyPressMsg{Code: 'a', Text: "a"}
	_, _ = m.Update(keyMsg)

	if child.updateCalls != 1 {
		t.Errorf("expected 1 update call to child, got %d", child.updateCalls)
	}

	// Verify message was received (can't directly compare tea.KeyMsg due to []rune)
	if child.lastMsg == nil {
		t.Error("child did not receive any message")
	}
}

// TestMessageRoutingHelpMode tests that Help overlays and intercepts messages
func TestMessageRoutingHelpMode(t *testing.T) {
	maker := newMockMaker()
	initialChild := newMockChild("initial", "Initial")
	m := NewRootModel(initialChild, maker)

	child := newMockChild("test", "Test Child")
	m.NavigateTo(child)

	helpChild := newMockChild("help", "Help")
	m.SetHelp(helpChild)
	m.SetMode(HelpMode)

	// Send a key message - should go to help, not current
	keyMsg := tea.KeyPressMsg{Code: 'a', Text: "a"}
	_, _ = m.Update(keyMsg)

	if helpChild.updateCalls != 1 {
		t.Errorf("expected 1 update call to help, got %d", helpChild.updateCalls)
	}

	if child.updateCalls != 0 {
		t.Errorf("expected 0 update calls to current child in help mode, got %d", child.updateCalls)
	}
}

// TestMessageRoutingPromptMode tests that Prompt overlays and intercepts messages
func TestMessageRoutingPromptMode(t *testing.T) {
	maker := newMockMaker()
	initialChild := newMockChild("initial", "Initial")
	m := NewRootModel(initialChild, maker)

	child := newMockChild("test", "Test Child")
	m.NavigateTo(child)

	promptChild := newMockChild("prompt", "Prompt")
	m.SetPrompt(promptChild)
	m.SetMode(PromptMode)

	// Send a key message - should go to prompt, not current
	keyMsg := tea.KeyPressMsg{Code: 'a', Text: "a"}
	_, _ = m.Update(keyMsg)

	if promptChild.updateCalls != 1 {
		t.Errorf("expected 1 update call to prompt, got %d", promptChild.updateCalls)
	}

	if child.updateCalls != 0 {
		t.Errorf("expected 0 update calls to current child in prompt mode, got %d", child.updateCalls)
	}
}

// TestWindowResize tests that window resize messages propagate to all children
func TestWindowResize(t *testing.T) {
	maker := newMockMaker()
	initialChild := newMockChild("initial", "Initial")
	m := NewRootModel(initialChild, maker)

	child := newMockChild("test", "Test Child")
	m.NavigateTo(child)

	helpChild := newMockChild("help", "Help")
	m.SetHelp(helpChild)

	promptChild := newMockChild("prompt", "Prompt")
	m.SetPrompt(promptChild)

	// Send window resize
	resizeMsg := tea.WindowSizeMsg{Width: 100, Height: 50}
	_, _ = m.Update(resizeMsg)

	if m.width != 100 || m.height != 50 {
		t.Errorf("expected dimensions 100x50, got %dx%d", m.width, m.height)
	}

	if child.width != 100 || child.height != 50 {
		t.Errorf("expected child dimensions 100x50, got %dx%d", child.width, child.height)
	}

	if helpChild.width != 100 || helpChild.height != 50 {
		t.Errorf("expected help dimensions 100x50, got %dx%d", helpChild.width, helpChild.height)
	}

	if promptChild.width != 100 || promptChild.height != 50 {
		t.Errorf("expected prompt dimensions 100x50, got %dx%d", promptChild.width, promptChild.height)
	}
}

// TestViewRendering tests View() output in different modes
func TestViewRendering(t *testing.T) {
	maker := newMockMaker()
	initialChild := newMockChild("initial", "Initial")
	m := NewRootModel(initialChild, maker)

	child := newMockChild("test", "Test Child")
	m.NavigateTo(child)

	// Normal mode - should show current child
	m.SetMode(NormalMode)
	view := m.View()
	if !strings.Contains(fmt.Sprint(view.Content), "Test Child view") {
		t.Error("normal mode view should contain child view")
	}

	// Help mode - should show help overlay on top
	helpChild := newMockChild("help", "Help")
	m.SetHelp(helpChild)
	m.SetMode(HelpMode)
	view = m.View()
	if !strings.Contains(fmt.Sprint(view.Content), "Help view") {
		t.Error("help mode view should contain help view")
	}
	if !strings.Contains(fmt.Sprint(view.Content), "Test Child view") {
		t.Error("help mode view should still contain underlying child view")
	}

	// Prompt mode - should show prompt overlay on top
	promptChild := newMockChild("prompt", "Prompt")
	m.SetPrompt(promptChild)
	m.SetMode(PromptMode)
	view = m.View()
	if !strings.Contains(fmt.Sprint(view.Content), "Prompt view") {
		t.Error("prompt mode view should contain prompt view")
	}
}

// TestGlobalShortcuts tests Ctrl+C quit behavior
func TestGlobalShortcuts(t *testing.T) {
	maker := newMockMaker()
	initialChild := newMockChild("initial", "Initial")
	m := NewRootModel(initialChild, maker)

	child := newMockChild("test", "Test Child")
	m.NavigateTo(child)

	// Send Ctrl+C (0x03 is the control character)
	quitMsg := tea.KeyPressMsg{Code: 0x03}
	_, cmd := m.Update(quitMsg)

	if cmd == nil {
		t.Error("expected quit command from Ctrl+C")
	}

	if !m.quitting {
		t.Error("expected quitting flag to be set")
	}
}

// TestNoColorMode tests SetNoColor propagation
func TestNoColorMode(t *testing.T) {
	maker := newMockMaker()
	initialChild := newMockChild("initial", "Initial")
	m := NewRootModel(initialChild, maker)

	if m.noColor {
		t.Error("expected noColor to be false initially")
	}

	m.SetNoColor(true)

	if !m.noColor {
		t.Error("expected noColor to be true after SetNoColor(true)")
	}
}

// TestDebugMode tests SetDebug flag
func TestDebugMode(t *testing.T) {
	maker := newMockMaker()
	initialChild := newMockChild("initial", "Initial")
	m := NewRootModel(initialChild, maker)

	if m.debug {
		t.Error("expected debug to be false initially")
	}

	m.SetDebug(true)

	if !m.debug {
		t.Error("expected debug to be true after SetDebug(true)")
	}
}

// TestRootDataManagement tests SetRoot and Root methods
func TestRootDataManagement(t *testing.T) {
	maker := newMockMaker()
	initialChild := newMockChild("initial", "Initial")
	m := NewRootModel(initialChild, maker)

	testData := map[string]interface{}{
		"key": "value",
	}

	m.SetRoot(testData)

	retrieved := m.Root()
	if retrieved == nil {
		t.Error("expected root data to be set")
	}

	if rootMap, ok := retrieved.(map[string]interface{}); ok {
		if rootMap["key"] != "value" {
			t.Error("root data does not match")
		}
	} else {
		t.Error("root data type mismatch")
	}
}

// TestNavigateToNilChild tests that NavigateTo(nil) clears current
func TestNavigateToNilChild(t *testing.T) {
	maker := newMockMaker()
	initialChild := newMockChild("initial", "Initial")
	m := NewRootModel(initialChild, maker)

	child := newMockChild("test", "Test Child")
	m.NavigateTo(child)

	if m.current != child {
		t.Error("expected current to be child")
	}

	m.NavigateTo(nil)

	if m.current != nil {
		t.Error("expected current to be nil after NavigateTo(nil)")
	}
}

// TestMultipleBackNavigations tests navigation stack with multiple items
func TestMultipleBackNavigations(t *testing.T) {
	maker := newMockMaker()
	initialChild := newMockChild("initial", "Initial")
	m := NewRootModel(initialChild, maker)

	// Build a stack: child1 -> child2 -> child3
	child1 := newMockChild("child1", "Child 1")
	child2 := newMockChild("child2", "Child 2")
	child3 := newMockChild("child3", "Child 3")

	m.NavigateTo(child1)
	m.NavigateTo(child2)
	m.NavigateTo(child3)

	// Verify stack depth
	if len(m.stack) != 3 {
		t.Errorf("expected stack length 3, got %d", len(m.stack))
	}

	// Navigate back through the stack
	if m.current != child3 {
		t.Error("expected current to be child3")
	}

	m.NavigateBack()
	if m.current != child2 {
		t.Error("expected current to be child2")
	}

	m.NavigateBack()
	if m.current != child1 {
		t.Error("expected current to be child1")
	}

	m.NavigateBack()
	if m.current != initialChild {
		t.Error("expected current to be initialChild")
	}

	if m.CanNavigateBack() {
		t.Error("expected CanNavigateBack to be false")
	}
}

// TestFocusManagement tests focus handling during navigation
func TestFocusManagement(t *testing.T) {
	maker := newMockMaker()
	initialChild := newMockChild("initial", "Initial")
	m := NewRootModel(initialChild, maker)

	child1 := newMockChild("child1", "Child 1")
	child2 := newMockChild("child2", "Child 2")

	// Navigate to child1
	m.NavigateTo(child1)

	if !child1.focused {
		t.Error("expected child1 to be focused after navigation")
	}

	// Navigate to child2
	m.NavigateTo(child2)

	if child1.focused {
		t.Error("expected child1 to be blurred after navigating away")
	}

	if !child2.focused {
		t.Error("expected child2 to be focused")
	}

	// Navigate back
	m.NavigateBack()

	if child2.focused {
		t.Error("expected child2 to be blurred after navigating back")
	}

	if !child1.focused {
		t.Error("expected child1 to be focused after navigating back")
	}
}
