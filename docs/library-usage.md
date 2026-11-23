---
title: "Library Usage"
weight: 20
---

# Using kvx as a Library

This guide covers everything you need to embed kvx in your own Go application — from a five-line quickstart to advanced customization.

## Install

```bash
go get github.com/oakwood-commons/kvx@latest
```

## Quickstart

There are two public packages. Pick the one that fits your use case:

| Package | Import | Use when you want… |
|---------|--------|--------------------|
| `pkg/core` | `github.com/oakwood-commons/kvx/pkg/core` | Load data, evaluate expressions, render plain text tables — **no terminal UI** |
| `pkg/tui` | `github.com/oakwood-commons/kvx/pkg/tui` | Launch the full interactive TUI or render bordered tables |

### Minimal example — render a table (no TUI)

```go
package main

import (
    "fmt"

    "github.com/oakwood-commons/kvx/pkg/core"
    "github.com/oakwood-commons/kvx/pkg/tui"
)

func main() {
    data := map[string]any{
        "name":    "my-service",
        "version": "2.1.0",
        "healthy": true,
    }

    root, _ := core.LoadObject(data)

    // Bordered table, just like the kvx CLI
    fmt.Print(tui.RenderTable(root, tui.TableOptions{
        AppName:  "my-app",
        Path:     "_",
        Bordered: true,
    }))
}
```

### Minimal example — launch the interactive TUI

```go
root, _ := core.LoadFile("data.yaml")

cfg := tui.DefaultConfig()
cfg.AppName = "my-app"

if err := tui.Run(root, cfg); err != nil {
    log.Fatal(err)
}
```

That's it. Everything else below is optional.

---

## Loading Data

All loaders return a generic `interface{}` (map, slice, or scalar) that every other kvx function accepts. Format is auto-detected.

```go
// From a file (JSON, YAML, NDJSON, JWT — auto-detected)
root, err := core.LoadFile("config.yaml")

// From a string
root, err := core.LoadRoot(`{"key": "value"}`)

// From bytes (useful for HTTP responses, embedded data, etc.)
root, err := core.LoadRootBytes(bodyBytes)

// From an existing Go value — map, slice, or struct
root, err := core.LoadObject(myStruct)
```

### Custom structs

`LoadObject` automatically normalizes structs to maps (respecting `json` and `yaml` struct tags) so they work seamlessly with the expression engine:

```go
type Artifact struct {
    Name    string `json:"name"`
    Version string `json:"version"`
}

artifact := &Artifact{Name: "api", Version: "1.0.0"}
root, _ := core.LoadObject(artifact)
// root is now map[string]any{"name": "api", "version": "1.0.0"}
```

Slices of structs work the same way:

```go
items := []any{
    &Artifact{Name: "api", Version: "1.0.0"},
    &Artifact{Name: "web", Version: "2.3.1"},
}
root, _ := core.LoadObject(items)
```

---

## Evaluating Expressions

The `core.Engine` provides CEL expression evaluation, navigation, and rendering:

```go
engine, _ := core.New()

// Evaluate a CEL expression
result, err := engine.Evaluate("_.name", root)

// Navigate to a nested path
node, err := engine.NodeAtPath(root, "metadata.labels")

// Get table rows ([][]string) for your own renderer
rows := engine.Rows(root)

// Render a scalar value as a string
s := engine.Stringify(result)
```

> **Note:** `engine.RenderTable()` renders a simple two-column KEY/VALUE table.
> For arrays of objects, use `tui.RenderTable()` instead — it auto-detects
> homogeneous arrays and renders them as columnar tables with field-name headers.
> See [Rendering Tables](#rendering-tables) below.

### Common CEL expressions

```
_.fieldName                          # access a field
_.items[0]                           # array index
_.items.filter(x, x.active)         # filter a list
_.items.map(x, x.name)              # map/transform
_.items.size()                       # list length
_.name.startsWith("api")            # string functions
_.count > 10 ? "high" : "low"       # ternary
```

### Filtering Lists (`FilterWhere` and `ErrNotList`)

Use `engine.FilterWhere(expr, root)` to apply a per-item CEL boolean predicate (`_` bound to each item) across any slice or array -- including typed Go struct slices (`[]Summary`, `[]*Summary`), `[]map[string]any`, and `[]any`:

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
    // Non-list input (map, struct, scalar, []byte, nil): safe no-op for uniform --where flags.
    filtered = providers
    err = nil
}
if err != nil {
    log.Fatal(err)
}
// filtered is a non-nil []any (marshals to "[]" via encoding/json when 0 items match, never "null")
~~~

Key behaviors:

- Accepts any Go slice or array except `[]byte` (byte slices are treated as scalar bytes and return `core.ErrNotList`).
- Typed `nil` slices (for example `var items []Summary`) are treated as empty lists and return `[]any{}` without error.
- Struct and pointer-to-struct elements are normalized to `map[string]any` respecting `json` and `yaml` struct tags, so predicates like `_.name == "http"` behave identically across `[]Summary`, `[]*Summary`, and `[]map[string]any`.
- Zero matches always return a non-nil `[]any{}` slice so `json.Marshal` produces `[]`, never `null`.
- Non-list inputs return an error wrapping `core.ErrNotList` (`errors.Is(err, core.ErrNotList)`).

---

## Rendering Tables

### Bordered table (matches kvx CLI output)

```go
output := tui.RenderTable(root, tui.TableOptions{
    AppName:  "my-app",   // shown in the top border
    Path:     "_",        // shown in the bottom border
    Bordered: true,
    // Width: 0,          // 0 = auto-detect terminal width
    // NoColor: false,    // set true for CI/tests
})
fmt.Print(output)
```

### Plain table (no borders)

```go
output := tui.RenderTable(root, tui.TableOptions{
    Bordered: false,
    NoColor:  true,
})
```

### Array formatting

Arrays of objects automatically render as multi-column tables. Control this with:

```go
tui.RenderTable(root, tui.TableOptions{
    Bordered:     true,
    ColumnarMode: tui.ColumnarModeAuto,       // "auto" (default), "always", or "never"
    ArrayStyle:   tui.ArrayStyleNumbered,      // "numbered" (default), "index", "bullet", "none"
    ColumnOrder:  []string{"name", "version"}, // preferred column order (arrays and single maps)
    HiddenColumns: []string{"sha"},            // columns to omit
})
```

### Column display hints (schema)

Control column headers, widths, alignment, and visibility using `ColumnHints`.
You can build hints programmatically or parse them from a JSON Schema:

```go
// Option 1: Build hints directly
hints := map[string]tui.ColumnHint{
    "user_id":   {DisplayName: "ID", MaxWidth: 10},
    "full_name": {DisplayName: "Name"},
    "score":     {Align: "right"},           // right-align numbers
    "internal":  {Hidden: true},             // hide this column
}

output := tui.RenderTable(root, tui.TableOptions{
    Bordered:    true,
    ColumnOrder: []string{"full_name", "user_id", "score"},
    ColumnHints: hints,
})
```

```go
// Option 2: Parse from JSON Schema
schemaJSON := []byte(`{
    "type": "array",
    "items": {
        "type": "object",
        "required": ["id", "name"],
        "properties": {
            "id":    {"title": "ID", "type": "string", "maxLength": 10},
            "name":  {"title": "Name", "type": "string"},
            "price": {"type": "number"},
            "legacy": {"deprecated": true}
        }
    }
}`)

hints, err := tui.ParseSchema(schemaJSON)
if err != nil {
    log.Fatal(err)
}

output := tui.RenderTable(root, tui.TableOptions{
    Bordered:    true,
    ColumnHints: hints,
})
```

`ParseSchema` derives hints from standard JSON Schema fields:

| JSON Schema Field | ColumnHint Effect |
| ------------------- | ------------------- |
| `title` | `DisplayName` — column header text |
| `maxLength` | `MaxWidth` — cap column width |
| `enum` | `MaxWidth` — longest enum value |
| `format` (date, uuid, etc.) | `MaxWidth` — auto-calculated |
| `type: integer/number` | `Align: "right"` |
| `deprecated: true` | `Hidden: true` |
| `required` array | `Priority` boost |

### Flex columns (fill terminal width)

By default, bordered tables shrink to fit the natural content width. When you want
one or more columns to absorb remaining terminal space (e.g. a "message" or
"description" column), mark them with `Flex: true`:

```go
hints := map[string]tui.ColumnHint{
    "severity": {MaxWidth: 8, Priority: 10},
    "message":  {Flex: true},  // absorbs remaining terminal width
}

output := tui.RenderTable(root, tui.TableOptions{
    Bordered:    true,
    ColumnHints: hints,
})
```

Key behaviors:

- When any column has `Flex: true`, the bordered table fills the full terminal width
  instead of shrinking to content.
- Surplus space is distributed evenly among flex columns.
- `MaxWidth` acts as a minimum guarantee for flex columns during initial sizing,
  not a cap on expansion. Flex columns start at `MaxWidth` (if set) and grow to
  fill remaining space.
- Non-flex columns keep their natural (content-fitted) width.
- If no column is marked `Flex`, behavior is unchanged (fit-to-content).

### Calculating table overhead

In most cases you do not need to calculate overhead manually -- use `Flex: true`
and let kvx handle the layout math internally.

For advanced use cases where you need to compute available content width (e.g.
to set a dynamic `MaxWidth`), the overhead formula is:

~~~
overhead = 2 (borders) + (numCols - 1) * 2 (separators)
~~~

Add the row-number column width if row numbers are enabled.

### Unified `Render` function

If you want to let the caller choose the output format at runtime (table, list, YAML, JSON),
use `tui.Render` with an `OutputFormat`:

```go
// format could come from a flag, config, etc.
format := tui.FormatTable // or FormatList, FormatYAML, FormatJSON

output := tui.Render(root, format, tui.TableOptions{
    Bordered: true,
    NoColor:  true,
})
fmt.Print(output)
```

Available formats:

| Constant | Output |
| --- | --- |
| `tui.FormatTable` | Columnar table (default) |
| `tui.FormatList` | Vertical property list |
| `tui.FormatYAML` | YAML |
| `tui.FormatJSON` | Indented JSON |

---

## Interactive TUI

### Basic launch

```go
cfg := tui.DefaultConfig()
cfg.AppName = "my-app"

if err := tui.Run(root, cfg); err != nil {
    log.Fatal(err)
}
```

### Configuration options

`tui.DefaultConfig()` returns sensible defaults. Override only what you need:

```go
cfg := tui.DefaultConfig()

// Appearance
cfg.AppName  = "my-app"     // header title
cfg.ThemeName = "warm"       // built-in themes: "dark", "warm", "cool"
cfg.NoColor   = true         // disable colors

// Behavior
cfg.InitialExpr = "_.items"  // start with an expression pre-evaluated

// UI text
cfg.KeyHeader        = "FIELD"           // table header (default: "KEY")
cfg.ValueHeader      = "DATA"            // table header (default: "VALUE")
cfg.InputPlaceholder = "Type a path…"    // input bar placeholder

// Feature toggles (pass *bool)
f := false
cfg.AllowFilter      = &f  // disable search/filter (F3)
cfg.AllowSuggestions = &f  // disable autocomplete
```

See [`tui.Config`](../pkg/tui/config.go) for the full list of fields.

### Snapshot mode (non-interactive)

Render exactly what the TUI would show, then exit — useful for CI or scripted output:

```go
output := tui.RenderSnapshot(root, cfg)
fmt.Print(output)
```

### Status screens and inline input

Use `StatusDisplayConfig` (`DisplaySchema.Status` or JSON Schema `x-kvx-status`) to render an async waiting screen with display fields, hotkey actions, and an optional frame-1 focused inline text input (`StatusInputConfig`):

```go
done := make(chan tui.StatusResult, 1)

cfg := tui.DefaultConfig()
cfg.AppName = "my-app"
cfg.Done = done
cfg.DisplaySchema = &tui.DisplaySchema{
    Version: "v1",
    Status: &tui.StatusDisplayConfig{
        TitleField:   "title",
        WaitMessage:  "Waiting for callback...",
        DisplayFields: []tui.StatusFieldDisplay{
            {Label: "Endpoint", Field: "url"},
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
            Label:       "Paste callback URL:",
            Placeholder: "http://localhost:8080/callback?token=...",
            Masked:      true,
        },
    },
}
cfg.OnStatusInput = func(value string) error {
    // Return a non-nil error to clear the buffer and display a retry warning.
    // Return nil to mark submitted while continuing to await Done.
    return processCallbackURL(value)
}
cfg.OpenURL = func(url string) error {
    // Optional custom URL opener for "open-url" status actions.
    return tui.OpenURL(url)
}
```

You can also declare status input settings via JSON Schema vendor extensions (`x-kvx-status.input`) parsed with `tui.ParseSchemaWithDisplay` or standalone display schemas parsed with `tui.ParseDisplaySchema`:

```json
{
  "type": "object",
  "x-kvx-status": {
    "titleField": "title",
    "waitMessage": "Waiting for callback...",
    "input": {
      "label": "Paste callback URL:",
      "placeholder": "http://localhost:8080/callback?token=...",
      "prompt": "> ",
      "masked": true,
      "maxLength": 4096
    }
  }
}
```

`StatusInputConfig` fields:

| Field | Type | Description |
| --- | --- | --- |
| `Label` | `string` | Optional helper text rendered above the input prompt |
| `Placeholder` | `string` | Placeholder shown inside the input when empty |
| `Prompt` | `string` | Prefix shown before the text input (default: `"> "` / theme prompt) |
| `Masked` | `bool` | Echoes bullet characters, renders `(N chars)` when non-empty, and skips keystroke debug logging |
| `MaxLength` | `int` | Maximum accepted input length in runes (`0` uses the default `8192`; negative values fail schema validation) |

---

## Custom CEL Functions

Add your own functions to the expression engine so they appear in TUI suggestions:

```go
import (
    "github.com/google/cel-go/cel"
    "github.com/google/cel-go/common/types"
    "github.com/google/cel-go/common/types/ref"
    "github.com/oakwood-commons/kvx/pkg/tui"
)

// 1. Build a CEL environment with your functions.
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

// 2. Create a provider (optional hints show in autocomplete).
provider := tui.NewCELExpressionProvider(env, map[string]string{
    "double": "e.g. double(5) → 10",
})

// 3. Plug it in before launching the TUI.
tui.SetExpressionProvider(provider)

cfg := tui.DefaultConfig()
tui.Run(root, cfg)
```

---

## Concurrency

`tui.RenderTable`/`tui.RenderList` and `core.Engine.Rows`/`RenderTable`/`FilterWhere`/`EvaluateWhere` are
safe to call from multiple goroutines concurrently, including with different
per-call options (theme, `MaxValueLines`): each call builds its own render
state instead of mutating shared package globals. Sort order is per-call-safe
through `core.Engine.Rows` (which threads an explicit `SortOrder` down), but
`tui.RenderTable`/`RenderList` have no per-call sort-order option yet -- they
still read the process-global default via `navigator.SetSortOrder`, so
concurrent callers wanting different orders through `pkg/tui` specifically
cannot get them (see the global-defaults caveat below).

The package-level configuration setters — `tui.SetMaxValueLines`,
`ui.SetTheme`/`SetThemeByName`, `navigator.SetSortOrder`, and the legacy
`formatter.SetTableTheme`/`SetMaxValueLines` — still change process-wide
defaults. They are internally synchronized (no data races or panics), but
concurrent callers changing these defaults will race for "whose setting wins"
for anything that reads the *global* default rather than an explicit per-call
option. If you render concurrently with different settings per call, pass
them explicitly (e.g. `TableOptions.MaxValueLines`) rather than relying on
these global setters.

## Working Examples

| Example | Description | Run |
| --------- | ------------- | ----- |
| [examples/embed-tui](../examples/embed-tui/) | Load a file, render a bordered table or launch the TUI | `go run ./examples/embed-tui sample.json` |
| [examples/custom_struct](../examples/custom_struct/) | Load Go structs, evaluate CEL expressions | `go run ./examples/custom_struct` |
| [examples/core-cli](../examples/core-cli/) | Headless CLI — load, evaluate, render (no TUI) | `go run ./examples/core-cli data.yaml "_.items[0]"` |
| [examples/status-screen](../examples/status-screen/) | Async status screen with actions and optional inline input | `go run ./examples/status-screen -with-input` |

---

## API Quick Reference

### `pkg/core`

| Function / Method | Description |
| --- | --- |
| `core.LoadFile(path)` | Load JSON/YAML/NDJSON/JWT from disk |
| `core.LoadRoot(input)` | Parse a string |
| `core.LoadRootBytes(data)` | Parse bytes |
| `core.LoadObject(value)` | Wrap a Go value (map, slice, struct) |
| `core.New(opts...)` | Create an `Engine` with defaults |
| `engine.Evaluate(expr, root)` | Run a CEL expression |
| `engine.FilterWhere(expr, root)` | Filter a slice/array per item via CEL (`_` = item); returns non-nil `[]any{}` on 0 matches, wraps `core.ErrNotList` on non-list input |
| `engine.EvaluateWhere(expr, root)` | Typed `([]any, error)` variant of `FilterWhere`; wraps `core.ErrNotList` on non-list input |
| `core.ErrNotList` | Sentinel error matched via `errors.Is(err, core.ErrNotList)` when `FilterWhere`/`EvaluateWhere` receives non-list data |
| `engine.NodeAtPath(root, path)` | Navigate to a nested node |
| `engine.Rows(node)` | Convert a node to `[][]string` rows |
| `engine.RenderTable(node, noColor, keyW, valW, columnOrder)` | Render a plain KEY/VALUE table (no columnar detection -- use `tui.RenderTable` for arrays) |
| `engine.Stringify(node)` | Render a scalar as a display string |

### `pkg/tui`

| Function | Description |
| --- | --- |
| `tui.Run(root, cfg, opts...)` | Launch the interactive TUI |
| `tui.Render(node, format, opts)` | Render using an `OutputFormat` (`FormatTable`, `FormatList`, `FormatTree`, `FormatMermaid`, `FormatYAML`, `FormatJSON`) |
| `tui.RenderTable(node, opts)` | Render a static table (bordered or plain; auto-detects columnar mode for arrays) |
| `tui.RenderList(node, opts)` | Render a vertical list (properties stacked per object, like `-o list`) |
| `tui.RenderTree(node, opts)` | Render an ASCII tree structure (like `-o tree`) |
| `tui.RenderMermaid(node, opts)` | Render a Mermaid flowchart diagram (like `-o mermaid`) |
| `tui.RenderSnapshot(root, cfg)` | Render a full TUI frame as a string |
| `tui.DefaultConfig()` | Get baseline TUI configuration |
| `tui.DetectTerminalSize()` | Get terminal width and height |
| `tui.NewCELExpressionProvider(env, hints)` | Create an expression provider from a CEL env |
| `tui.SetExpressionProvider(p)` | Override the global expression provider |
| `tui.ResetExpressionProvider()` | Restore the default provider |
| `tui.ParseDisplaySchema(data)` | Parse a standalone `DisplaySchema` JSON document |
| `tui.ParseSchemaWithDisplay(schemaJSON)` | Extract `ColumnHint` values and `x-kvx-*` `DisplaySchema` from JSON Schema |
| `tui.OpenURL(url)` | Open a URL in the default system browser |

### `tui.ColumnHint` fields

| Field | Type | Description |
| --- | --- | --- |
| `MaxWidth` | `int` | Cap column width (0 = no cap). For flex columns, acts as a minimum guarantee — the column starts at MaxWidth but can expand beyond it |
| `Priority` | `int` | Column importance during shrinking (higher = resists more) |
| `DisplayName` | `string` | Override column header text |
| `Align` | `string` | `"right"` or `"left"` (default) |
| `Hidden` | `bool` | Omit column from output |
| `Flex` | `bool` | Absorb remaining terminal width after fixed columns. Auto-set by `ParseSchema` for columns without `maxLength`/`enum`/`format` constraints |

### `internal/formatter` (advanced)

| Function | Description |
| --- | --- |
| `formatter.ColumnarOverhead(numCols, showRowNum, numRows)` | Calculate fixed overhead (borders + separators + row numbers) |
| `formatter.HasFlexColumn(hints)` | Check if any hint has `Flex: true` |
