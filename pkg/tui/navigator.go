package tui

import "github.com/oakwood-commons/kvx/internal/navigator"

// Navigator lets host apps plug in their own path resolution.
type Navigator interface {
	NodeAtPath(root interface{}, path string) (interface{}, error)
}

// SetNavigator overrides the global navigator used by the UI.
func SetNavigator(n Navigator) {
	if n != nil {
		navigator.SetNavigator(navigator.Adapter{Navigator: n})
	}
}

// DefaultNavigator returns the built-in navigator.
func DefaultNavigator() Navigator {
	return navigator.DefaultNavigator()
}
