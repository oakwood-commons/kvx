package core

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/oakwood-commons/kvx/internal/cel"
	"github.com/oakwood-commons/kvx/internal/formatter"
	"github.com/oakwood-commons/kvx/internal/navigator"
	"github.com/oakwood-commons/kvx/pkg/loader"
)

// Evaluator evaluates expressions against a root node.
type Evaluator interface {
	Evaluate(expr string, root interface{}) (interface{}, error)
}

// Navigator defines navigation and row conversion behavior.
type Navigator interface {
	NodeAtPath(root interface{}, path string) (interface{}, error)
	NodeToRows(node interface{}) [][]string
	SetSortOrder(order SortOrder) SortOrder
}

// Formatter defines rendering and stringify behavior.
type Formatter interface {
	RenderTable(node interface{}, noColor bool, keyColWidth, valueColWidth int) string
	Stringify(node interface{}) string
}

// SortOrder controls map key ordering for row rendering.
type SortOrder string

const (
	SortNone       SortOrder = "none"
	SortAscending  SortOrder = "ascending"
	SortDescending SortOrder = "descending"
)

// Engine provides a minimal shared API for loading, evaluating, and rendering data.
type Engine struct {
	Evaluator Evaluator
	Navigator Navigator
	Formatter Formatter
	SortOrder SortOrder
}

// Option configures the Engine.
type Option func(*Engine)

// WithEvaluator sets a custom evaluator.
func WithEvaluator(e Evaluator) Option {
	return func(c *Engine) {
		c.Evaluator = e
	}
}

// WithNavigator sets a custom navigator.
func WithNavigator(n Navigator) Option {
	return func(c *Engine) {
		c.Navigator = n
	}
}

// WithFormatter sets a custom formatter.
func WithFormatter(f Formatter) Option {
	return func(c *Engine) {
		c.Formatter = f
	}
}

// WithSortOrder sets the row sort order.
func WithSortOrder(order SortOrder) Option {
	return func(c *Engine) {
		c.SortOrder = order
	}
}

// New creates an Engine with defaults.
func New(opts ...Option) (*Engine, error) {
	engine := &Engine{
		SortOrder: SortNone,
	}
	for _, opt := range opts {
		opt(engine)
	}
	if engine.Evaluator == nil {
		eval, err := cel.NewEvaluator()
		if err != nil {
			return nil, err
		}
		engine.Evaluator = eval
	}
	if engine.Navigator == nil {
		engine.Navigator = defaultNavigator{}
	}
	if engine.Formatter == nil {
		engine.Formatter = defaultFormatter{}
	}
	return engine, nil
}

// LoadRoot parses input into a single root node; multi-doc inputs return a slice.
func LoadRoot(input string) (interface{}, error) {
	return loader.LoadRoot(input)
}

// LoadRootBytes parses input bytes into a single root node.
func LoadRootBytes(data []byte) (interface{}, error) {
	return loader.LoadRootBytes(data)
}

// LoadRootBytesWithLogger is like LoadRootBytes but accepts a logger for
// recording fallback parse attempts.
func LoadRootBytesWithLogger(data []byte, lgr logr.Logger) (interface{}, error) {
	return loader.LoadRootBytesWithLogger(data, lgr)
}

// LoadFile reads a file and parses it into a single root node.
func LoadFile(path string) (interface{}, error) {
	return loader.LoadFile(path)
}

// LoadFileWithLogger is like LoadFile but accepts a logger for recording
// fallback parse attempts and extension-based dispatch.
func LoadFileWithLogger(path string, lgr logr.Logger) (interface{}, error) {
	return loader.LoadFileWithLogger(path, lgr)
}

// LoadObject accepts an already parsed object and returns it directly.
// Strings and byte slices are parsed using the shared loader to preserve auto-detection.
func LoadObject(value interface{}) (interface{}, error) {
	return loader.LoadObject(value)
}

// Evaluate runs the evaluator against the provided root node.
func (e *Engine) Evaluate(expr string, root interface{}) (interface{}, error) {
	if e == nil || e.Evaluator == nil {
		return nil, fmt.Errorf("evaluator is not configured")
	}
	return e.Evaluator.Evaluate(expr, root)
}

// NodeAtPath navigates a path into the root using navigator rules.
func (e *Engine) NodeAtPath(root interface{}, path string) (interface{}, error) {
	e.ensureNavigator()
	if e == nil || e.Navigator == nil {
		return nil, fmt.Errorf("navigator is not configured")
	}
	return e.Navigator.NodeAtPath(root, path)
}

// Rows converts a node into table rows, honoring the Engine sort order.
func (e *Engine) Rows(node interface{}) [][]string {
	e.ensureNavigator()
	if e == nil || e.Navigator == nil {
		return nil
	}
	prev := e.Navigator.SetSortOrder(e.SortOrder)
	defer e.Navigator.SetSortOrder(prev)
	return e.Navigator.NodeToRows(node)
}

// RenderTable renders a two-column table for the node.
func (e *Engine) RenderTable(node interface{}, noColor bool, keyColWidth, valueColWidth int) string {
	e.ensureFormatter()
	if e == nil || e.Formatter == nil {
		return ""
	}
	return e.Formatter.RenderTable(node, noColor, keyColWidth, valueColWidth)
}

// Stringify renders a node into a display string.
func (e *Engine) Stringify(node interface{}) string {
	e.ensureFormatter()
	if e == nil || e.Formatter == nil {
		return ""
	}
	return e.Formatter.Stringify(node)
}

func toNavigatorSort(order SortOrder) navigator.SortOrder {
	switch order {
	case SortAscending:
		return navigator.SortAscending
	case SortDescending:
		return navigator.SortDescending
	case SortNone:
		return navigator.SortNone
	default:
		return navigator.SortNone
	}
}

func fromNavigatorSort(order navigator.SortOrder) SortOrder {
	switch order {
	case navigator.SortAscending:
		return SortAscending
	case navigator.SortDescending:
		return SortDescending
	case navigator.SortNone:
		return SortNone
	default:
		return SortNone
	}
}

type defaultNavigator struct{}

func (defaultNavigator) NodeAtPath(root interface{}, path string) (interface{}, error) {
	return navigator.NodeAtPath(root, path)
}

func (defaultNavigator) NodeToRows(node interface{}) [][]string {
	return navigator.NodeToRows(node)
}

func (defaultNavigator) SetSortOrder(order SortOrder) SortOrder {
	prev := navigator.SetSortOrder(toNavigatorSort(order))
	return fromNavigatorSort(prev)
}

type defaultFormatter struct{}

func (defaultFormatter) RenderTable(node interface{}, noColor bool, keyColWidth, valueColWidth int) string {
	return formatter.RenderTable(node, noColor, keyColWidth, valueColWidth)
}

func (defaultFormatter) Stringify(node interface{}) string {
	return formatter.Stringify(node)
}

func (e *Engine) ensureNavigator() {
	if e == nil {
		return
	}
	if e.Navigator == nil {
		e.Navigator = defaultNavigator{}
	}
}

func (e *Engine) ensureFormatter() {
	if e == nil {
		return
	}
	if e.Formatter == nil {
		e.Formatter = defaultFormatter{}
	}
}
