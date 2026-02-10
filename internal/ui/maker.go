package ui

import tea "charm.land/bubbletea/v2"

// Maker creates child models on demand (lazy initialization pattern).
// This allows deferring model creation until needed, reducing memory usage
// and initialization time for large applications.
//
// Based on pug's Maker pattern, this is particularly useful for:
// - Navigation history (create models as user navigates)
// - Tab/pane systems (create panes only when accessed)
// - Dynamic content (create models based on data)
type Maker interface {
	// Make creates a new child model instance.
	// Parameters:
	//   id: unique identifier for the model (e.g., path, resource ID)
	//   width, height: available space for the model
	// Returns the created model and any initialization command.
	Make(id string, width, height int) (ChildModel, tea.Cmd)
}

// MakerFunc is a function adapter for Maker interface.
// Allows using simple functions as Makers without defining a struct.
type MakerFunc func(id string, width, height int) (ChildModel, tea.Cmd)

// Make implements Maker interface for MakerFunc.
func (f MakerFunc) Make(id string, width, height int) (ChildModel, tea.Cmd) {
	return f(id, width, height)
}

// CachedMaker wraps a Maker and caches created models to avoid recreating them.
// Useful for models that are expensive to initialize or need to maintain state
// across navigation (e.g., scroll position, expanded nodes).
type CachedMaker struct {
	maker Maker
	cache map[string]ChildModel
}

// NewCachedMaker creates a new CachedMaker that wraps the given maker.
func NewCachedMaker(maker Maker) *CachedMaker {
	return &CachedMaker{
		maker: maker,
		cache: make(map[string]ChildModel),
	}
}

// Make creates or retrieves a cached child model.
// If a model with the given id already exists in the cache, it is returned
// (after resizing if needed). Otherwise, the underlying maker is called.
func (c *CachedMaker) Make(id string, width, height int) (ChildModel, tea.Cmd) {
	if model, ok := c.cache[id]; ok {
		// Return cached model (resize if it implements ModelWithSize)
		if sized, ok := model.(ModelWithSize); ok {
			sized.SetSize(width, height)
		}
		return model, nil
	}

	// Create new model and cache it
	model, cmd := c.maker.Make(id, width, height)
	c.cache[id] = model
	return model, cmd
}

// Clear removes all cached models.
func (c *CachedMaker) Clear() {
	c.cache = make(map[string]ChildModel)
}

// Remove removes a specific model from the cache.
func (c *CachedMaker) Remove(id string) {
	delete(c.cache, id)
}

// Has returns true if a model with the given id is cached.
func (c *CachedMaker) Has(id string) bool {
	_, ok := c.cache[id]
	return ok
}
