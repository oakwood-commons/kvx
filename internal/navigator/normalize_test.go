package navigator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizePath_Empty(t *testing.T) {
	assert.Equal(t, "", NormalizePath(""))
}

func TestNormalizePath_NoNumericSegments(t *testing.T) {
	assert.Equal(t, "user.name", NormalizePath("user.name"))
}

func TestNormalizePath_SingleIndex(t *testing.T) {
	assert.Equal(t, "items[0]", NormalizePath("items.0"))
}

func TestNormalizePath_MultipleIndices(t *testing.T) {
	assert.Equal(t, "items[0].tags[1]", NormalizePath("items.0.tags.1"))
}

func TestNormalizePath_MixedSegments(t *testing.T) {
	assert.Equal(t, "regions.asia.countries[1]", NormalizePath("regions.asia.countries.1"))
}

func TestNormalizePath_AlreadyBracketNotation(t *testing.T) {
	// Bracket notation should pass through unchanged
	result := NormalizePath("items[0].name")
	assert.Contains(t, result, "items")
	assert.Contains(t, result, "name")
}

func TestNormalizePath_TrailingNumeric(t *testing.T) {
	assert.Equal(t, "items[0]", NormalizePath("items.0"))
}

func TestNormalizePath_NestedNumeric(t *testing.T) {
	assert.Equal(t, "a[0].b[1].c[2]", NormalizePath("a.0.b.1.c.2"))
}

func TestNormalizePath_SingleField(t *testing.T) {
	assert.Equal(t, "name", NormalizePath("name"))
}
