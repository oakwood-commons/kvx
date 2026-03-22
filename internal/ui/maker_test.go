package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
)

type testChildModel struct {
	id   string
	w, h int
}

func (m *testChildModel) Init() tea.Cmd                        { return nil }
func (m *testChildModel) Update(tea.Msg) (ChildModel, tea.Cmd) { return m, nil }
func (m *testChildModel) View() string                         { return m.id }
func (m *testChildModel) ID() string                           { return m.id }
func (m *testChildModel) Title() string                        { return m.id }
func (m *testChildModel) SetSize(w, h int)                     { m.w = w; m.h = h }

// Verify interface compliance.
var (
	_ ChildModel    = (*testChildModel)(nil)
	_ ModelWithID   = (*testChildModel)(nil)
	_ ModelWithSize = (*testChildModel)(nil)
)

func TestMakerFunc(t *testing.T) {
	maker := MakerFunc(func(id string, w, h int) (ChildModel, tea.Cmd) {
		return &testChildModel{id: id, w: w, h: h}, nil
	})
	model, cmd := maker.Make("test", 80, 24)
	assert.NotNil(t, model)
	assert.Nil(t, cmd)
	assert.Equal(t, "test", model.(*testChildModel).id)
}

func TestNewCachedMaker(t *testing.T) {
	maker := MakerFunc(func(id string, w, h int) (ChildModel, tea.Cmd) {
		return &testChildModel{id: id, w: w, h: h}, nil
	})
	cached := NewCachedMaker(maker)
	assert.NotNil(t, cached)
	assert.False(t, cached.Has("test"))
}

func TestCachedMaker_Make(t *testing.T) {
	callCount := 0
	maker := MakerFunc(func(id string, w, h int) (ChildModel, tea.Cmd) {
		callCount++
		return &testChildModel{id: id, w: w, h: h}, nil
	})
	cached := NewCachedMaker(maker)

	// First call creates
	m1, _ := cached.Make("test", 80, 24)
	assert.Equal(t, 1, callCount)
	assert.True(t, cached.Has("test"))

	// Second call returns cached
	m2, _ := cached.Make("test", 100, 30)
	assert.Equal(t, 1, callCount) // Not called again
	assert.Same(t, m1, m2)

	// Verify resize was applied via SetSize
	tm := m2.(*testChildModel)
	assert.Equal(t, 100, tm.w)
	assert.Equal(t, 30, tm.h)
}

func TestCachedMaker_Clear(t *testing.T) {
	maker := MakerFunc(func(id string, w, h int) (ChildModel, tea.Cmd) {
		return &testChildModel{id: id, w: w, h: h}, nil
	})
	cached := NewCachedMaker(maker)
	cached.Make("a", 80, 24)
	cached.Make("b", 80, 24)
	assert.True(t, cached.Has("a"))
	assert.True(t, cached.Has("b"))

	cached.Clear()
	assert.False(t, cached.Has("a"))
	assert.False(t, cached.Has("b"))
}

func TestCachedMaker_Remove(t *testing.T) {
	maker := MakerFunc(func(id string, w, h int) (ChildModel, tea.Cmd) {
		return &testChildModel{id: id, w: w, h: h}, nil
	})
	cached := NewCachedMaker(maker)
	cached.Make("a", 80, 24)
	assert.True(t, cached.Has("a"))

	cached.Remove("a")
	assert.False(t, cached.Has("a"))

	// Remove non-existing is a no-op
	cached.Remove("nonexistent")
}
