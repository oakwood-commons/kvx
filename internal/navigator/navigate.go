package navigator

// Navigate takes a root node and a path expression (dotted, CEL, or mixed) and returns the target node.
// It serves as a unified interface for both CLI and GUI navigation.
func Navigate(root interface{}, expr string) (interface{}, error) {
	return NodeAtPath(root, expr)
}
