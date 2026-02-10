package ui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Mode represents the current UI mode/state.
// This follows pug's pattern of using modes to control message routing.
type Mode int

const (
	// NormalMode is the default browsing/navigation mode.
	NormalMode Mode = iota
	// ExprMode is expression input/editing mode.
	ExprMode
	// SearchMode is advanced search mode.
	SearchMode
	// HelpMode displays the help overlay.
	HelpMode
	// PromptMode displays a modal prompt/dialog.
	PromptMode
)

// RootModel is the root/top-level model that manages child models and routes messages.
// It follows the "tree of models" pattern from pug, where the root delegates to children.
//
// Key responsibilities:
// - Maintain current mode (normal, expr, search, help, prompt)
// - Route messages to appropriate child models
// - Handle window resize and layout
// - Coordinate global state (theme, debug, etc.)
type RootModel struct {
	// Current mode determines message routing
	mode Mode

	// Child models
	current ChildModel // Currently active child model
	help    ChildModel // Help overlay (rendered on top when visible)
	prompt  ChildModel // Modal prompt (rendered on top when visible)

	// Model management
	maker Maker        // Creates child models on demand
	stack []ChildModel // Navigation stack (for back navigation)

	// Global state
	width   int
	height  int
	debug   bool
	noColor bool
	root    interface{} // Root data node

	// Quit flag
	quitting bool
}

// NewRootModel creates a new root model with the given child model maker.
func NewRootModel(initialModel ChildModel, maker Maker) *RootModel {
	return &RootModel{
		mode:    NormalMode,
		current: initialModel,
		maker:   maker,
		stack:   []ChildModel{},
		width:   80,
		height:  24,
	}
}

// Init initializes the root model and its children.
func (m *RootModel) Init() tea.Cmd {
	var cmds []tea.Cmd

	// Initialize current model
	if m.current != nil {
		cmds = append(cmds, m.current.Init())
	}

	return tea.Batch(cmds...)
}

// Update handles messages and routes them to appropriate child models.
func (m *RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle global messages first
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Resize all child models
		if m.current != nil {
			if sized, ok := m.current.(ModelWithSize); ok {
				sized.SetSize(m.width, m.height)
			}
		}
		if m.help != nil {
			if sized, ok := m.help.(ModelWithSize); ok {
				sized.SetSize(m.width, m.height)
			}
		}
		if m.prompt != nil {
			if sized, ok := m.prompt.(ModelWithSize); ok {
				sized.SetSize(m.width, m.height)
			}
		}

		return m, nil

	case tea.KeyMsg:
		// Handle global quit
		keyStr := msg.String()
		// Check for Ctrl+C - handle both string form and raw control character (0x03)
		if keyStr == "ctrl+c" || msg.Key().Code == 0x03 {
			m.quitting = true
			return m, tea.Quit
		}

		// Handle mode-specific routing
		switch m.mode {
		case HelpMode:
			// Help mode intercepts all input
			if m.help != nil {
				var cmd tea.Cmd
				m.help, cmd = m.help.Update(msg)
				cmds = append(cmds, cmd)
			}
			// Check if help should close
			if msg.String() == "f1" || msg.String() == "esc" {
				m.mode = NormalMode
			}
			return m, tea.Batch(cmds...)

		case PromptMode:
			// Prompt mode intercepts all input
			if m.prompt != nil {
				var cmd tea.Cmd
				m.prompt, cmd = m.prompt.Update(msg)
				cmds = append(cmds, cmd)
			}
			// Prompt models should send a message when done
			return m, tea.Batch(cmds...)

		case ExprMode:
			// Expression mode - route to current model
			if m.current != nil {
				var cmd tea.Cmd
				m.current, cmd = m.current.Update(msg)
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		case SearchMode:
			// Search mode - route to current model
			if m.current != nil {
				var cmd tea.Cmd
				m.current, cmd = m.current.Update(msg)
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)

		case NormalMode:
			// Normal mode - route to current model
			if m.current != nil {
				var cmd tea.Cmd
				m.current, cmd = m.current.Update(msg)
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)

		default:
			return m, tea.Batch(cmds...)
		}
	}

	// Route message to current model
	if m.current != nil {
		var cmd tea.Cmd
		m.current, cmd = m.current.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the root model and its children.
func (m *RootModel) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}

	// Start with current model
	view := ""
	if m.current != nil {
		view = m.current.View()
	}

	// Overlay help if visible
	if m.mode == HelpMode && m.help != nil {
		helpView := m.help.View()
		if view == "" {
			view = helpView
		} else {
			// Render help above the current view to simulate an overlay while preserving the base
			view = helpView + "\n" + view
		}
	}

	// Overlay prompt if visible
	if m.mode == PromptMode && m.prompt != nil {
		// Render prompt centered on top of current view
		promptView := m.prompt.View()
		view = m.overlayPrompt(view, promptView)
	}

	v := tea.NewView(view)
	v.AltScreen = true
	// Enable keyboard enhancements for proper modifier key detection (e.g., Shift+Tab)
	v.KeyboardEnhancements.ReportEventTypes = true
	return v
}

// overlayPrompt renders a prompt centered on top of the base view.
func (m *RootModel) overlayPrompt(base, prompt string) string {
	// For now, just append prompt below base
	// (More sophisticated overlay requires line-by-line composition)
	// Future enhancement: Use lipgloss.Height/Width for centered overlay
	_ = lipgloss.Height(prompt)
	_ = lipgloss.Width(prompt)
	return base + "\n" + prompt
}

// SetMode changes the current UI mode.
func (m *RootModel) SetMode(mode Mode) {
	m.mode = mode
}

// Mode returns the current UI mode.
func (m *RootModel) Mode() Mode {
	return m.mode
}

// NavigateTo navigates to a new child model.
// Pushes the current model onto the navigation stack.
func (m *RootModel) NavigateTo(model ChildModel) tea.Cmd {
	// Blur the current model before navigating away
	if m.current != nil {
		if focusable, ok := m.current.(ModelWithFocus); ok {
			focusable.Blur()
		}
		m.stack = append(m.stack, m.current)
	}
	m.current = model

	// Initialize and focus new model
	var cmds []tea.Cmd
	if m.current != nil {
		cmds = append(cmds, m.current.Init())
		if focusable, ok := m.current.(ModelWithFocus); ok {
			cmds = append(cmds, focusable.Focus())
		}
	}
	return tea.Batch(cmds...)
}

// NavigateBack goes back to the previous model in the navigation stack.
func (m *RootModel) NavigateBack() bool {
	if len(m.stack) == 0 {
		return false
	}

	// Blur current before switching
	if m.current != nil {
		if focusable, ok := m.current.(ModelWithFocus); ok {
			focusable.Blur()
		}
	}

	// Pop from stack
	m.current = m.stack[len(m.stack)-1]
	m.stack = m.stack[:len(m.stack)-1]

	// Focus the restored model
	if m.current != nil {
		if focusable, ok := m.current.(ModelWithFocus); ok {
			focusable.Focus()
		}
	}

	return true
}

// CanNavigateBack returns true if there are models in the navigation stack.
func (m *RootModel) CanNavigateBack() bool {
	return len(m.stack) > 0
}

// SetHelp sets the help overlay model.
func (m *RootModel) SetHelp(help ChildModel) {
	m.help = help
}

// SetPrompt sets the prompt/dialog model.
func (m *RootModel) SetPrompt(prompt ChildModel) {
	m.prompt = prompt
}

// SetDebug enables/disables debug mode.
func (m *RootModel) SetDebug(debug bool) {
	m.debug = debug
}

// SetNoColor enables/disables color output.
func (m *RootModel) SetNoColor(noColor bool) {
	m.noColor = noColor
}

// SetRoot sets the root data node.
func (m *RootModel) SetRoot(root interface{}) {
	m.root = root
}

// Root returns the root data node.
func (m *RootModel) Root() interface{} {
	return m.root
}
