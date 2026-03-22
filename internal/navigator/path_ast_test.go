package navigator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsePath_Empty(t *testing.T) {
	nodes := ParsePath("")
	assert.Empty(t, nodes)
}

func TestParsePath_SimpleField(t *testing.T) {
	nodes := ParsePath("name")
	assert.Len(t, nodes, 1)
	assert.Equal(t, Field{Name: "name"}, nodes[0])
}

func TestParsePath_DottedPath(t *testing.T) {
	nodes := ParsePath("user.name")
	assert.Len(t, nodes, 2)
	assert.Equal(t, Field{Name: "user"}, nodes[0])
	assert.Equal(t, Field{Name: "name"}, nodes[1])
}

func TestParsePath_ArrayIndex(t *testing.T) {
	nodes := ParsePath("items[0]")
	assert.Len(t, nodes, 2)
	assert.Equal(t, Field{Name: "items"}, nodes[0])
	assert.Equal(t, ArrayIndex{Index: 0}, nodes[1])
}

func TestParsePath_QuotedKey(t *testing.T) {
	nodes := ParsePath(`city["postal-code"]`)
	assert.Len(t, nodes, 2)
	assert.Equal(t, Field{Name: "city"}, nodes[0])
	assert.Equal(t, QuotedKey{Name: "postal-code"}, nodes[1])
}

func TestParsePath_Complex(t *testing.T) {
	nodes := ParsePath(`regions.asia.countries[0].city["postal-code"]`)
	assert.Len(t, nodes, 6)
	assert.Equal(t, Field{Name: "regions"}, nodes[0])
	assert.Equal(t, Field{Name: "asia"}, nodes[1])
	assert.Equal(t, Field{Name: "countries"}, nodes[2])
	assert.Equal(t, ArrayIndex{Index: 0}, nodes[3])
	assert.Equal(t, Field{Name: "city"}, nodes[4])
	assert.Equal(t, QuotedKey{Name: "postal-code"}, nodes[5])
}

func TestParsePath_MultipleIndices(t *testing.T) {
	nodes := ParsePath("data[1][2]")
	assert.Len(t, nodes, 3)
	assert.Equal(t, Field{Name: "data"}, nodes[0])
	assert.Equal(t, ArrayIndex{Index: 1}, nodes[1])
	assert.Equal(t, ArrayIndex{Index: 2}, nodes[2])
}

func TestParsePath_IncompleteBracket(t *testing.T) {
	nodes := ParsePath("items[")
	assert.Len(t, nodes, 1)
	assert.Equal(t, Field{Name: "items"}, nodes[0])
}

func TestParsePath_DotAfterIndex(t *testing.T) {
	nodes := ParsePath("items[0].name")
	assert.Len(t, nodes, 3)
	assert.Equal(t, Field{Name: "items"}, nodes[0])
	assert.Equal(t, ArrayIndex{Index: 0}, nodes[1])
	assert.Equal(t, Field{Name: "name"}, nodes[2])
}

func TestParsePath_NonNumericBracket(t *testing.T) {
	nodes := ParsePath("data[key]")
	assert.Len(t, nodes, 2)
	assert.Equal(t, Field{Name: "data"}, nodes[0])
	assert.Equal(t, Field{Name: "key"}, nodes[1])
}

func TestReconstructPath_Empty(t *testing.T) {
	result := ReconstructPath(nil)
	assert.Empty(t, result)
}

func TestReconstructPath_SimpleField(t *testing.T) {
	nodes := []Node{Field{Name: "name"}}
	assert.Equal(t, "name", ReconstructPath(nodes))
}

func TestReconstructPath_DottedPath(t *testing.T) {
	nodes := []Node{Field{Name: "user"}, Field{Name: "name"}}
	assert.Equal(t, "user.name", ReconstructPath(nodes))
}

func TestReconstructPath_WithIndex(t *testing.T) {
	nodes := []Node{Field{Name: "items"}, ArrayIndex{Index: 0}, Field{Name: "name"}}
	assert.Equal(t, "items[0].name", ReconstructPath(nodes))
}

func TestReconstructPath_WithQuotedKey(t *testing.T) {
	nodes := []Node{Field{Name: "city"}, QuotedKey{Name: "postal-code"}}
	assert.Equal(t, `city["postal-code"]`, ReconstructPath(nodes))
}

func TestReconstructPath_CelExpr(t *testing.T) {
	nodes := []Node{Field{Name: "items"}, CelExpr{Expr: "(x > 1)"}}
	result := ReconstructPath(nodes)
	assert.Equal(t, "items.(x > 1)", result)
}

func TestReconstructPath_Roundtrip(t *testing.T) {
	tests := []string{
		"name",
		"user.name",
		"items[0].name",
		`city["postal-code"]`,
		"data[1][2]",
		"regions.asia.countries[0]",
	}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			nodes := ParsePath(input)
			result := ReconstructPath(nodes)
			assert.Equal(t, input, result)
		})
	}
}
