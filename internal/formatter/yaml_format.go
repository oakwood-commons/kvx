package formatter

import (
	"bytes"
	"strings"

	"gopkg.in/yaml.v3"
)

// YAMLFormatOptions control YAML rendering.
type YAMLFormatOptions struct {
	Indent                int
	LiteralBlockStrings   bool
	ExpandEscapedNewlines bool
}

// FormatYAML renders an object to YAML using the provided options. Multi-line
// strings can be emitted as literal blocks ("|") to preserve newlines.
func FormatYAML(v interface{}, opts YAMLFormatOptions) (string, error) {
	var node yaml.Node
	if err := node.Encode(v); err != nil {
		return "", err
	}

	if opts.ExpandEscapedNewlines {
		expandEscapedNewlines(&node)
	}
	if opts.LiteralBlockStrings {
		applyLiteralStyle(&node)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	indent := opts.Indent
	if indent <= 0 {
		indent = 2
	}
	enc.SetIndent(indent)
	if err := enc.Encode(&node); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func applyLiteralStyle(n *yaml.Node) {
	if n == nil {
		return
	}
	if n.Kind == yaml.ScalarNode && n.Tag == "!!str" && strings.Contains(n.Value, "\n") {
		n.Style = yaml.LiteralStyle
	}
	for _, c := range n.Content {
		applyLiteralStyle(c)
	}
}

func expandEscapedNewlines(n *yaml.Node) {
	if n == nil {
		return
	}
	if n.Kind == yaml.ScalarNode && n.Tag == "!!str" && strings.Contains(n.Value, "\\n") {
		n.Value = strings.ReplaceAll(n.Value, "\\n", "\n")
	}
	for _, c := range n.Content {
		expandEscapedNewlines(c)
	}
}
