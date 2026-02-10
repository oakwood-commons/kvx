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

`LoadObject` automatically converts structs to maps via JSON serialization so they work with the expression engine:

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
    ColumnOrder:  []string{"name", "version"}, // preferred column order
    HiddenColumns: []string{"sha"},            // columns to omit
})
```

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
|---|---|
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

## Working Examples

| Example | Description | Run |
|---------|-------------|-----|
| [examples/embed-tui](../examples/embed-tui/) | Load a file, render a bordered table or launch the TUI | `go run ./examples/embed-tui sample.json` |
| [examples/custom_struct](../examples/custom_struct/) | Load Go structs, evaluate CEL expressions | `go run ./examples/custom_struct` |
| [examples/core-cli](../examples/core-cli/) | Headless CLI — load, evaluate, render (no TUI) | `go run ./examples/core-cli data.yaml "_.items[0]"` |

---

## API Quick Reference

### `pkg/core`

| Function / Method | Description |
|---|---|
| `core.LoadFile(path)` | Load JSON/YAML/NDJSON/JWT from disk |
| `core.LoadRoot(input)` | Parse a string |
| `core.LoadRootBytes(data)` | Parse bytes |
| `core.LoadObject(value)` | Wrap a Go value (map, slice, struct) |
| `core.New(opts...)` | Create an `Engine` with defaults |
| `engine.Evaluate(expr, root)` | Run a CEL expression |
| `engine.NodeAtPath(root, path)` | Navigate to a nested node |
| `engine.Rows(node)` | Convert a node to `[][]string` rows |
| `engine.RenderTable(node, noColor, keyW, valW)` | Render a plain KEY/VALUE table (no columnar detection — use `tui.RenderTable` for arrays) |
| `engine.Stringify(node)` | Render a scalar as a display string |

### `pkg/tui`

| Function | Description |
|---|---|
| `tui.Run(root, cfg, opts...)` | Launch the interactive TUI |
| `tui.Render(node, format, opts)` | Render using an `OutputFormat` (`FormatTable`, `FormatList`, `FormatTree`, `FormatMermaid`, `FormatYAML`, `FormatJSON`) |
| `tui.RenderTable(node, opts)` | Render a static table (bordered or plain; auto-detects columnar mode for arrays) |
| `tui.RenderList(node, noColor)` | Render a vertical list (properties stacked per object, like `-o list`) |
| `tui.RenderTree(node, opts)` | Render an ASCII tree structure (like `-o tree`) |
| `tui.RenderMermaid(node, opts)` | Render a Mermaid flowchart diagram (like `-o mermaid`) |
| `tui.RenderSnapshot(root, cfg)` | Render a full TUI frame as a string |
| `tui.DefaultConfig()` | Get baseline TUI configuration |
| `tui.DetectTerminalSize()` | Get terminal width and height |
| `tui.NewCELExpressionProvider(env, hints)` | Create an expression provider from a CEL env |
| `tui.SetExpressionProvider(p)` | Override the global expression provider |
| `tui.ResetExpressionProvider()` | Restore the default provider |
