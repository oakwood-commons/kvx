package navigator

// Navigator is the interface for path resolution.
type Navigator interface {
	NodeAtPath(root interface{}, path string) (interface{}, error)
}

// Adapter implements Navigator using a provided implementation.
type Adapter struct {
	Navigator Navigator
}

func (a Adapter) NodeAtPath(root interface{}, path string) (interface{}, error) {
	return a.Navigator.NodeAtPath(root, path)
}

var currentNavigator Navigator = defaultNavigator{}

// SetNavigator overrides the global navigator used by UI.
func SetNavigator(n Navigator) {
	if n != nil {
		currentNavigator = n
	}
}

// DefaultNavigator returns the built-in navigator.
func DefaultNavigator() Navigator {
	return defaultNavigator{}
}

type defaultNavigator struct{}

func (defaultNavigator) NodeAtPath(root interface{}, path string) (interface{}, error) {
	return NodeAtPath(root, path)
}

// Resolve delegates to the current navigator.
func Resolve(root interface{}, path string) (interface{}, error) {
	return currentNavigator.NodeAtPath(root, path)
}
