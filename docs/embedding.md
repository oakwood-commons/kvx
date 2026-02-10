# Embedding kvx in another app

Use the public `tui` and `core` packages to run kvx inside your Go program. This guide covers data loading, configuration, and common extension points.

## Data loading options

Pick the loader that matches your source:

- `core.LoadFile(path)` — read JSON/YAML/NDJSON from disk.
- `core.LoadRoot(input)` / `core.LoadRootBytes(data)` — parse strings/bytes with format auto-detection.
- `core.LoadObject(value)` — pass an already parsed Go value (map, slice, struct, etc.). Strings and byte slices are parsed through the same detection logic; nil inputs return an error.

## Minimal embed

```go
package main

import (
    "log"

    "github.com/oakwood-commons/kvx/pkg/core"
    "github.com/oakwood-commons/kvx/pkg/tui"
)

func main() {
    // Use an existing Go value (no re-serialization needed)
    data := map[string]any{
        "service": "kvx",
        "owners":  []any{"dev@acme.com", "ops@acme.com"},
    }

    root, err := core.LoadObject(data)
    if err != nil {
        log.Fatalf("load data: %v", err)
    }

    cfg := tui.DefaultConfig()

    if err := tui.Run(root, cfg); err != nil {
        log.Fatalf("tui error: %v", err)
    }
}
```

Swap `core.LoadObject` with `core.LoadFile` when you want to load from disk. For scripted renders (no interactive loop), use CLI-style flags with `--snapshot` via your own `os.Args` handling; the `tui` package itself always runs interactively.

## Customizing expression support

- Start from the built-in CEL provider: `tui.NewCELExpressionProvider(celEnv, exampleHints)`.
- Inject it with `tui.SetExpressionProvider(provider)` before calling `tui.Run`.
- See the extended example in [examples/embed-tui/main.go](../examples/embed-tui/main.go) for adding custom CEL functions and completion hints.

## Theming and layout

- `tui.DefaultConfig()` loads the bundled theme and key bindings from `internal/ui/default_config.yaml`.
- Modify the returned config (colors, borders, widths, key bindings) before calling `tui.Run`.
- To mirror CLI behavior, keep `cfg.Mode` at its default (interactive); snapshot/non-interactive runs are handled by the CLI front-end, not `tui.Run`.

## Navigation and rendering hooks

If you need custom navigation or table rendering outside the TUI, use the `core.Engine` directly:

```go
engine, _ := core.New()
rows := engine.Rows(root)        // table rows for your own renderer
out, _ := engine.Evaluate("_.path", root)
_ = rows
_ = out
```

This lets you reuse kvx navigation/formatting while embedding the TUI only where you need it.
