package navigator

import (
	"strconv"
	"strings"
)

// Node represents a parsed segment of a path input.
// Path example: regions.asia.countries[0].city["postal-code"]
// CEL segments are treated as opaque for now.
type Node interface{}

// Field represents a simple dotted field name.
type Field struct {
	Name string
}

// QuotedKey represents a field accessed via bracket-quoted key: ["key"]
type QuotedKey struct {
	Name string
}

// ArrayIndex represents an array index like [0]
type ArrayIndex struct {
	Index int
}

// CelExpr represents an opaque CEL expression segment
// We keep it simple and detect by presence of '(' for now.
type CelExpr struct {
	Expr string
}

// ParsePath parses a path string into a slice of nodes.
// It supports dots, bracket indices, and bracket quoted keys. CEL is detected heuristically.
func ParsePath(input string) []Node {
	var nodes []Node
	if input == "" {
		return nodes
	}

	// We iterate through the string and parse tokens.
	// This is a lightweight parser intended to stabilize completion logic.
	i := 0
	for i < len(input) {
		ch := input[i]
		if ch == '.' {
			// skip dot
			i++
			continue
		}
		if ch == '[' {
			// Could be index or quoted key
			end := strings.IndexByte(input[i:], ']')
			if end == -1 {
				// incomplete bracket, treat as pending and stop
				break
			}
			segment := input[i+1 : i+end]
			if strings.HasPrefix(segment, "\"") && strings.HasSuffix(segment, "\"") {
				name := strings.Trim(segment, "\"")
				nodes = append(nodes, QuotedKey{Name: name})
			} else {
				// try index
				if n, err := strconv.Atoi(segment); err == nil {
					nodes = append(nodes, ArrayIndex{Index: n})
				} else {
					// fallback: treat as field-like inside brackets
					nodes = append(nodes, Field{Name: segment})
				}
			}
			i += end + 1 + 0 // move past ']'
			continue
		}
		// CEL detection: presence of '(' implies CEL expression from here
		if idx := strings.IndexByte(input[i:], '('); idx == 0 {
			// start of CEL at current position
			nodes = append(nodes, CelExpr{Expr: input[i:]})
			break
		}
		// parse a dotted identifier until next '.' or '['
		j := i
		for j < len(input) && input[j] != '.' && input[j] != '[' {
			j++
		}
		name := input[i:j]
		if name != "" {
			nodes = append(nodes, Field{Name: name})
		}
		i = j
	}
	return nodes
}

// ReconstructPath rebuilds a path string from nodes.
func ReconstructPath(nodes []Node) string {
	var b strings.Builder
	for idx, n := range nodes {
		switch v := n.(type) {
		case Field:
			if idx > 0 {
				b.WriteByte('.')
			}
			b.WriteString(v.Name)
		case QuotedKey:
			b.WriteString("[\"")
			b.WriteString(v.Name)
			b.WriteString("\"]")
		case ArrayIndex:
			b.WriteByte('[')
			b.WriteString(strconv.Itoa(v.Index))
			b.WriteByte(']')
		case CelExpr:
			// Append as-is, assumes it's terminal
			if idx > 0 {
				b.WriteByte('.')
			}
			b.WriteString(v.Expr)
		}
	}
	return b.String()
}
