---
title: "Development"
weight: 50
---

# Development

Guide for building, testing, linting, and maintaining documentation and TUI media in `kvx`.

## Prerequisites

- **Go 1.27+** ([go.dev/dl](https://go.dev/dl/))
- **[Task](https://taskfile.dev/)** (optional convenience runner for `taskfile.yaml`)
- **C toolchain** (`gcc` or `clang`) when running race-detector suites (`CGO_ENABLED=1 go test -race ./...`)
- **Hugo Extended** (`0.155+` / `0.158+`) when building or previewing the documentation site locally

## Building

Always output local binaries to `dist/kvx` (which is gitignored) rather than the repository root:

```bash
# Recommended via Task
task build

# Direct Go command
mkdir -p dist && go build -o dist/kvx .
```

## Testing & Linting

Tests exercise real loader, CEL, formatter, and deterministic Bubble Tea v2 `Update()` / `RenderSnapshot` state transitions without requiring an interactive TTY:

```bash
# Unit tests
go test ./...

# Race-detector suite (required before opening a PR on Linux/macOS)
CGO_ENABLED=1 go test -race ./...

# Strict golangci-lint suite (configured in .golangci.yaml, including gofumpt + goimports)
golangci-lint run ./...

# Full local CI pipeline (mod + lint + test + coverage + govulncheck)
task ci
```

> [!IMPORTANT]
> Never pass `-i` inside automated tests; use `--snapshot` (and optional `--press "<keys>"`) or `tui.RenderSnapshot` so test runs terminate deterministically.

## Documentation Site & TUI Media Generation

The documentation site (`https://kvx.oakwood-commons.org/`) uses [Hugo](https://gohugo.io/) (`site/hugo.yaml`) with the Oakwood Commons Blueprint theme (`assets/_custom.scss`). `site/go.mod` isolates the Hugo theme module from the root Go module, and Hugo builds into `site/public/` or `public/` (gitignored).

### Building and Previewing Docs

```bash
# Build minified static site into public/
task docs:build
# or directly:
(cd site && hugo --gc --minify --cleanDestinationDir -d ../public)

# Live-reload local preview server
task docs:serve
```

### Regenerating `.svg` Terminal Cards, `.cast` Recording, and `demo.gif`

All TUI visuals under `static/images/tui/*.svg`, `static/casts/demo.cast`, and `static/images/demo.gif` are generated deterministically from the live `dist/kvx` binary:

1. Install [Charmbracelet `freeze`](https://github.com/charmbracelet/freeze) (`go install github.com/charmbracelet/freeze@latest`) for vector `.svg` terminal snapshots, and install [`asciinema/agg`](https://github.com/asciinema/agg) in `PATH` to re-render `static/images/demo.gif` from `static/casts/demo.cast` (or set `SKIP_GIF=1` to update `.svg` cards and `demo.cast` only).
2. Run the media pipeline:
   ```bash
   task docs:media
   # or directly:
   ./scripts/render-docs-media.sh
   ```

`scripts/render-docs-media.sh` builds `dist/kvx`, pipes isolated `kvx ... --snapshot` frames through `freeze --language ansi` (stripping embedded base64 fonts and adding responsive `viewBox` and `<title>` metadata), runs `scripts/record_demo_cast.py` in a fixed `100x26` Python pseudo-terminal to emit `static/casts/demo.cast` with `"m"` chapter markers, and renders `static/images/demo.gif` via `agg`.

## Repository Structure

~~~text
.
|-- main.go                 # CLI entry point
|-- cmd/                    # Cobra root/config/version commands and CLI flag wiring
|-- pkg/
|   |-- core/               # Public headless loader, CEL Engine, and FilterWhere API
|   |-- tui/                # Public Bubble Tea v2 runner, RenderTable/Snapshot, and DisplaySchema API
|   |-- loader/             # Format auto-detection, JWT decoder, and RecursiveDecode
|   |-- intellisense/       # Public completion metadata and example helpers
|   |-- logger/             # Context-aware logr + zap structured logging
|   +-- settings/           # CLI run parameters and context helpers
|-- internal/
|   |-- cel/                # CEL environment setup, custom helpers, and expression evaluation
|   |-- completion/         # AST-aware CEL completion engine and categorized FunctionRegistry
|   |-- formatter/          # Table, columnar, list, tree, Mermaid, YAML, TOML, and CSV formatters
|   |-- limiter/            # --limit, --offset, and --tail record slicing
|   |-- navigator/          # Path AST, map/slice normalization, and row extraction
|   +-- ui/                 # Bubble Tea v2 models (ChildModel, RootModel, Card-List, Detail, Status, Palette)
|-- examples/               # Runnable Go library examples and sample data/schemas
|-- docs/ & home/           # Markdown content mounted by site/hugo.yaml
|-- assets/, layouts/, static/ # Blueprint SCSS, Hugo shortcodes, SVG snapshots, and hero cast/GIF
+-- scripts/                # Snapshot updater, release index generator, and docs media pipeline
~~~

## Logging Pattern

`kvx` uses `github.com/go-logr/logr` backed by `zapr`:

- `logger.Get(verbosity)` constructs structured loggers (negative verbosity values such as `-1` enable debug output).
- `logger.WithLogger(ctx, lgr)` and `logger.FromContext(ctx)` propagate loggers through `context.Context`.
