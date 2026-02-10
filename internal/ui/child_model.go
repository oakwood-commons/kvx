package ui

import tea "charm.land/bubbletea/v2"

// ChildModel defines the interface that all child models must implement.
// This pattern allows building a tree of models where the root model
// routes messages to child models.
//
// Inspired by pug's architecture, this enables:
// - Separation of concerns (each model handles its own domain)
// - Easier testing (test child models in isolation)
// - Better maintainability (smaller, focused models)
// - Lazy loading (create models only when needed via Maker pattern)
type ChildModel interface {
	// Init initializes the child model and returns any initial commands.
	// Called when the child model is first created.
	Init() tea.Cmd

	// Update handles messages and updates the child model state.
	// Returns the updated model and any commands to run.
	Update(msg tea.Msg) (ChildModel, tea.Cmd)

	// View renders the child model to a string.
	View() string
}

// ModelWithID is an optional interface that child models can implement
// to provide an identifier for debugging and logging.
type ModelWithID interface {
	// ID returns a unique identifier for this model instance.
	ID() string
}

// ModelWithTitle is an optional interface that child models can implement
// to provide a title for display in navigation breadcrumbs or headers.
type ModelWithTitle interface {
	// Title returns a human-readable title for this model.
	Title() string
}

// ModelWithBorder is an optional interface that child models can implement
// to customize their border appearance when rendered in a pane.
type ModelWithBorder interface {
	// BorderText returns the text to display in the border (e.g., title, status).
	BorderText() string
}

// ModelWithSize is an optional interface that child models can implement
// to respond to resize events.
type ModelWithSize interface {
	// SetSize sets the available width and height for the model.
	SetSize(width, height int)
}

// ModelWithFocus is an optional interface that child models can implement
// to handle focus/blur events when navigating between panes.
type ModelWithFocus interface {
	// Focus is called when the model gains focus.
	Focus() tea.Cmd

	// Blur is called when the model loses focus.
	Blur()

	// Focused returns true if the model currently has focus.
	Focused() bool
}
