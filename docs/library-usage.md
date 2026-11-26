---
title: "Library Usage"
weight: 20
---

# Using kvx as a Library

Embed `kvx` in your Go CLI or service using two public packages: `pkg/core` for headless loading, CEL evaluation, and per-item filtering, and `pkg/tui` for bordered tables, multi-format rendering, schema-driven views, and the interactive Bubble Tea v2 TUI.

## Install

```bash
go get github.com/oakwood-commons/kvx@latest
```

Requires Go 1.27 or later.

## Quickstart

| Package | Import | Use When You Want |
|---------|--------|-------------------|
| `pkg/core` | `github.com/oakwood-commons/kvx/pkg/core` | Load structured data, evaluate CEL expressions, filter typed slices (`FilterWhere`), and extract rows with zero terminal UI dependency |
| `pkg/tui` | `github.com/oakwood-commons/kvx/pkg/tui` | Render bordered tables, trees, lists, or Mermaid diagrams, run deterministic TUI snapshots, or launch the interactive TUI |

### Render a Bordered Table (Headless)

```go
package main

import (
	"fmt"

	"github.com/oakwood-commons/kvx/pkg/core"
	"github.com/oakwood-commons/kvx/pkg/tui"
)

func main() {
	data := map[string]any{
		"name":    "auth-issuer",
		"version": "2.4.1",
		"healthy": true,
	}

	root, _ := core.LoadObject(data)

	fmt.Print(tui.RenderTable(root, tui.TableOptions{
		AppName:     "my-cli",
		Path:        "_",
		Bordered:    true,
		ColumnOrder: []string{"name", "version", "healthy"},
	}))
}
```

### Launch the Interactive TUI

```go
root, err := core.LoadFile("services.yaml")
if err != nil {
	log.Fatal(err)
}

cfg := tui.DefaultConfig()
cfg.AppName = "my-cli"

if err := tui.Run(root, cfg); err != nil {
	log.Fatal(err)
}
```

---

## Loading Data & Serialized Decoding

All `pkg/core` loaders normalize inputs into `map[string]any`, `[]any`, or scalars with automatic format detection (JSON, YAML, NDJSON, TOML, and `Bearer` JWTs):

~~~go
// From disk
root, err := core.LoadFile("config.yaml")

// From a raw string or byte slice (HTTP body, embedded asset, JWT token)
root, err := core.LoadRoot(`{"key": "value"}`)
root, err := core.LoadRootBytes(bodyBytes)

// From any Go value (map, slice, struct, or pointer to struct; honors json/yaml tags)
root, err := core.LoadObject(myStructSlice)
~~~

To recursively expand embedded JSON, YAML, TOML, NDJSON, or JWT strings inside leaf fields before querying, use `pkg/loader`:

```go
import "github.com/oakwood-commons/kvx/pkg/loader"

expanded := loader.RecursiveDecode(root)
```

---

## Evaluating CEL & Filtering Lists (`FilterWhere`)

`core.Engine` provides thread-safe CEL evaluation, path navigation, and per-item list filtering:

~~~go
engine, err := core.New()
if err != nil {
	log.Fatal(err)
}

// Evaluate any CEL expression with '_' bound to root
result, err := engine.Evaluate(`_.items.filter(x, x.active).map(x, x.name)`, root)

// Navigate a dotted/indexed path directly
node, err := engine.NodeAtPath(root, "metadata.labels")

// Convert a node into [][]string KEY/VALUE rows
rows := engine.Rows(root)
~~~

### Per-Item Slice Filtering (`FilterWhere` & `ErrNotList`)

Use `engine.FilterWhere(expr, root)` to evaluate a boolean CEL predicate (`_` bound to each item) across any Go slice or array, including typed struct slices (`[]Summary`, `[]*Summary`), `[]map[string]any`, and `[]any`:

~~~go
type Summary struct {
	Name       string `json:"name" yaml:"name"`
	Deprecated bool   `json:"deprecated" yaml:"deprecated"`
}

providers := []Summary{
	{Name: "http", Deprecated: false},
	{Name: "legacy", Deprecated: true},
}

engine, _ := core.New()

filtered, err := engine.FilterWhere(`_.name == "http" && !_.deprecated`, providers)
if errors.Is(err, core.ErrNotList) {
	// Non-list input (map, struct, scalar, []byte, nil): safe no-op for uniform --where flags
	filtered = providers
	err = nil
}
if err != nil {
	log.Fatal(err)
}
// filtered is a non-nil []any (marshals to "[]" in JSON when 0 items match, never "null")
~~~

---

## Rendering Tables, Trees, Lists, & Diagrams

### Column Ordering, Array Styles, & Vertical Lists

Homogeneous arrays of objects automatically render as multi-column tables, and `ColumnOrder` + `HiddenColumns` work consistently across both `tui.RenderTable` (maps and columnar arrays) and `tui.RenderList` (vertical property stacks):

```go
// Columnar or KEY/VALUE table with custom field ordering
output := tui.RenderTable(root, tui.TableOptions{
	AppName:       "my-cli",
	Path:          "_",
	Bordered:      true,
	ColumnarMode:  tui.ColumnarModeAuto,                 // "auto" (default), "always", or "never"
	ArrayStyle:    tui.ArrayStyleNumbered,               // "numbered", "index", "bullet", or "none"
	ColumnOrder:   []string{"name", "status", "region"}, // unlisted fields follow alphabetically
	HiddenColumns: []string{"authToken"},
})

// Vertical list layout honoring the same ColumnOrder and HiddenColumns
listOut := tui.RenderList(root, tui.ListOptions{
	ColumnOrder:   []string{"name", "status", "region"},
	HiddenColumns: []string{"authToken"},
})
```

### Root-Safe TOML Marshaling (`tui.MarshalTOML`)

Because the TOML specification forbids bare arrays at the document root, `tui.MarshalTOML` unwraps pointers/interfaces and automatically wraps root slices or arrays under `[[items]]` while marshaling maps directly:

```go
tomlText, err := tui.MarshalTOML(filteredItems)
```

### Column Display Hints & JSON Schema (`ParseSchemaWithDisplay`)

Extract both non-interactive `ColumnHint` rules and interactive `DisplaySchema` views (`x-kvx-list`, `x-kvx-detail`, `x-kvx-status`) from a single JSON Schema document:

```go
hints, displaySchema, err := tui.ParseSchemaWithDisplay(schemaBytes)
if err != nil {
	log.Fatal(err)
}

// Non-interactive CLI output with schema column hints and derived column order
fmt.Print(tui.RenderTable(root, tui.TableOptions{
	AppName:     "my-cli",
	Bordered:    true,
	ColumnHints: hints,
	Schema:      displaySchema,
}))

// Or launch the interactive TUI with Card-List and Sectioned Detail views
cfg := tui.DefaultConfig()
cfg.AppName = "my-cli"
cfg.DisplaySchema = displaySchema
_ = tui.Run(root, cfg)
```

Standard JSON Schema keywords mapped by `ParseSchema` / `ParseSchemaWithDisplay`:

| JSON Schema Keyword | `tui.ColumnHint` Effect |
|---------------------|-------------------------|
| `title` | `DisplayName` (column header override) |
| `maxLength`, `enum`, `format` | `MaxWidth` cap (never narrower than header text) |
| `type: "integer"` or `"number"` | `Align: "right"` |
| `deprecated: true` | `Hidden: true` |
| `required` | `Priority` boost (`+10`) |
| Unconstrained visible field | `Flex: true` (absorbs remaining terminal width) |

### Unified Multi-Format Renderer (`tui.Render`)

Let callers select any output format at runtime via `tui.OutputFormat`:

```go
output := tui.Render(root, tui.FormatAuto, tui.TableOptions{
	Bordered:    true,
	ColumnOrder: []string{"name", "version"},
})
```

Supported `OutputFormat` constants:
`tui.FormatAuto` (table when readable, falling back to list), `tui.FormatTable`, `tui.FormatList`, `tui.FormatTree`, `tui.FormatMermaid`, `tui.FormatYAML`, `tui.FormatJSON`, `tui.FormatTOML`, and `tui.FormatCSV`. Dedicated helpers `tui.RenderList`, `tui.RenderTree`, and `tui.RenderMermaid` are also available when passing format-specific options (`TreeOptions`, `MermaidOptions`).

---

## Interactive TUI, Snapshots, & Status Screens

### Snapshot Rendering (`tui.RenderSnapshot`)

Render a deterministic full-screen TUI frame (including scripted startup keys) without entering an interactive loop:

```go
root, _ := core.LoadObject(mySlice) // always normalize typed Go slices via LoadObject first
cfg := tui.DefaultConfig()
cfg.Width = 92
cfg.Height = 24
cfg.ThemeName = "midnight"
cfg.DisplaySchema = displaySchema
cfg.StartKeys = []string{"j", "<Enter>"}

frame := tui.RenderSnapshot(root, cfg)
fmt.Print(frame)
```

### Status Screens & Masked Inline Input (`x-kvx-status`)

Drive device-code login flows or async background operations with `StatusDisplayConfig`, hotkey actions, and frame-1 focused inline input:

```go
done := make(chan tui.StatusResult, 1)

cfg := tui.DefaultConfig()
cfg.AppName = "my-cli"
cfg.Done = done
cfg.DisplaySchema = &tui.DisplaySchema{
	Version: "v1",
	Status: &tui.StatusDisplayConfig{
		TitleField:  "title",
		WaitMessage: "Waiting for callback...",
		DisplayFields: []tui.StatusFieldDisplay{
			{Label: "Verification URL", Field: "url"},
		},
		Actions: []tui.StatusActionConfig{
			{
				Label: "Copy URL",
				Type:  "copy-value",
				Field: "url",
				Keys:  tui.StatusKeyBindings{Vim: "c", Emacs: "alt+c", Function: "f2"},
			},
			{
				Label: "Open URL",
				Type:  "open-url",
				Field: "url",
				Keys:  tui.StatusKeyBindings{Vim: "o", Emacs: "alt+o", Function: "f3"},
			},
		},
		Input: &tui.StatusInputConfig{
			Label:       "Paste callback token:",
			Placeholder: "http://localhost:8080/callback?token=...",
			Masked:      true,
		},
	},
}
cfg.OnStatusInput = func(value string) error {
	// Return a non-nil error to clear the buffer and display a warning for inline retry.
	// Return nil to mark submitted while continuing to await Done.
	return processCallbackToken(value)
}
cfg.OpenURL = tui.OpenURL
```

---

## Custom CEL Functions

Register application-specific CEL functions and autocomplete examples via `tui.NewCELExpressionProvider`:

```go
env, _ := cel.NewEnv(
	cel.Variable("_", cel.DynType),
	cel.Function("double",
		cel.Overload("double_int",
			[]*cel.Type{cel.IntType}, cel.IntType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				return args[0].(types.Int) * 2
			}),
		),
	),
)

cfg := tui.DefaultConfig()
cfg.ExpressionProvider = tui.NewCELExpressionProvider(env, map[string]string{
	"double": "double(5) => 10",
})
_ = tui.Run(root, cfg)
```

---

## Concurrency Guarantees

- `tui.RenderTable`, `tui.RenderList`, `tui.RenderTree`, `tui.RenderMermaid`, and `core.Engine` methods (`Evaluate`, `Rows`, `RenderTable`, `FilterWhere`, `EvaluateWhere`) are safe to call concurrently across goroutines with distinct per-call options (`TableOptions.MaxValueLines`, `ColumnHints`, `ColumnOrder`).
- Package-level setters (`tui.SetMaxValueLines`, `navigator.SetSortOrder`, `tui.SetExpressionProvider`) mutate process-wide defaults; prefer per-call `TableOptions` or per-run `tui.Config` fields when rendering concurrently.

## Runnable Examples in This Repository

| Example | Description | Command |
|---------|-------------|---------|
| [`examples/display_schema`](../examples/display_schema) | JSON Schema `x-kvx-list` Card-List and `x-kvx-detail` Sectioned Detail explorer | `go run ./examples/display_schema` |
| [`examples/schema_hints`](../examples/schema_hints) | JSON Schema and programmatic `ColumnHint` table customization | `go run ./examples/schema_hints` |
| [`examples/column_order`](../examples/column_order) | Custom key and column ordering in `RenderTable` and `RenderList` | `go run ./examples/column_order` |
| [`examples/status-screen`](../examples/status-screen) | Async waiting screen with hotkeys and optional masked inline input | `go run ./examples/status-screen -with-input` |
| [`examples/custom_struct`](../examples/custom_struct) | Normalizing Go structs and evaluating CEL expressions | `go run ./examples/custom_struct` |
| [`examples/core-cli`](../examples/core-cli) | Minimal headless CLI built on `pkg/core` | `go run ./examples/core-cli tests/sample.yaml "_.items[0]"` |

---

## API Quick Reference

### `pkg/core`

| Symbol | Description |
|--------|-------------|
| `core.LoadFile(path)` / `LoadRoot(s)` / `LoadRootBytes(b)` | Parse JSON, YAML, NDJSON, TOML, or JWT input |
| `core.LoadObject(v)` | Normalize an in-memory Go struct, slice, or map |
| `core.New(opts...)` | Construct a CEL-backed `Engine` |
| `engine.Evaluate(expr, root)` | Evaluate a CEL expression with `_` bound to `root` |
| `engine.FilterWhere(expr, root)` / `EvaluateWhere(expr, root)` | Filter any slice/array per item via CEL; returns `[]any{}` on zero matches and wraps `core.ErrNotList` on non-lists |
| `engine.NodeAtPath(root, path)` | Resolve a dotted/indexed path |
| `engine.Rows(node)` | Extract `[][]string` key/value rows |

### `pkg/tui`

| Symbol | Description |
|--------|-------------|
| `tui.Run(root, cfg, opts...)` | Launch the interactive Bubble Tea v2 TUI |
| `tui.RenderSnapshot(root, cfg)` | Render a single TUI frame (supports `cfg.StartKeys`, `cfg.DisplaySchema`, `cfg.AutoDecode`) |
| `tui.Render(node, format, opts)` | Render using `FormatAuto`, `FormatTable`, `FormatList`, `FormatTree`, `FormatMermaid`, `FormatYAML`, `FormatJSON`, `FormatTOML`, or `FormatCSV` |
| `tui.RenderTable` / `RenderList` / `RenderTree` / `RenderMermaid` | Format-specific static renderers (`RenderTable` and `RenderList` both honor `ColumnOrder` and `HiddenColumns`) |
| `tui.RenderSchemaView` / `RenderCardList` / `RenderDetailView` | Static schema-driven card-list and sectioned detail renderers (`x-kvx-detail` preserves paragraph newlines and expands typed slice tags into badges) |
| `tui.MarshalTOML(node)` | Serialize maps or root slices/arrays to TOML (automatically wraps bare slices under `[[items]]`) |
| `tui.ParseSchema(json)` / `ParseSchemaWithDisplay(json)` | Extract `ColumnHint` map and optional `DisplaySchema` (`x-kvx-*`) from JSON Schema |
| `tui.ParseDisplaySchema(json)` | Parse a standalone `displaySchema: "v1"` document |
| `tui.NewCELExpressionProvider(env, hints)` | Build an `ExpressionProvider` from a custom `*cel.Env` |
