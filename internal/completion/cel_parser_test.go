package completion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsOperator(t *testing.T) {
	assert.True(t, isOperator("_&&_"))
	assert.True(t, isOperator("_||_"))
	assert.True(t, isOperator("_==_"))
	assert.True(t, isOperator("_!=_"))
	assert.True(t, isOperator("_<_"))
	assert.True(t, isOperator("_>_"))
	assert.True(t, isOperator("_<=_"))
	assert.True(t, isOperator("_>=_"))
	assert.True(t, isOperator("_+_"))
	assert.True(t, isOperator("_-_"))
	assert.True(t, isOperator("!_"))
	assert.True(t, isOperator("-_"))
	assert.True(t, isOperator("_?_:_"))
	assert.True(t, isOperator("@in"))
	assert.True(t, isOperator("_[_]"))
	assert.True(t, isOperator("_*_"))
	assert.True(t, isOperator("_/_"))
	assert.True(t, isOperator("_%_"))
	assert.True(t, isOperator("_in_"))
	assert.False(t, isOperator("filter"))
	assert.False(t, isOperator("map"))
	assert.False(t, isOperator("size"))
	assert.False(t, isOperator(""))
}
