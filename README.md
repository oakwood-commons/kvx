# kvx

**kvx** is a terminal-based UI for exploring structured data like JSON, YAML, and TOML in an interactive, navigable way. It presents data as key value trees that you can expand, collapse, and inspect directly in the terminal, making it easy to understand complex or deeply nested structures. In addition to being a standalone CLI, kvx is designed as a reusable TUI component, allowing other applications to embed the viewer directly into their own terminal interfaces for consistent data inspection and visualization.

## Quick Start

```bash
task build
dist/kvx tests/sample.yaml       # table output
dist/kvx tests/sample.yaml -i    # interactive TUI
```

## Install

**Prerequisites:** Go 1.20+, [task](https://taskfile.dev/)

```bash
task build                    # recommended (builds to dist/kvx)
go build -o dist/kvx .        # alternative (always use -o dist/kvx)
```

**Note:** The binary is built to `dist/kvx`, not the root directory. Always use `-o dist/kvx` when building manually.

## Usage

```bash
dist/kvx <file> [flags]
```

### Flags

- `-i, --interactive` launch the TUI; `--snapshot` renders once and exits using the same layout as the TUI.
- `--press "<keys>"` script startup keys (e.g., `<F3>name<Enter>`); include `<F10>` to bypass the TUI and emit the non-interactive output.
- `-e, --expression <cel>` evaluate CEL against `_` (e.g., `_.items[0].name`, `type(_)`); dotted shorthand stays TUI-only.
- `--search <text>` search keys/values; seeds F3 in TUI and prints a bordered table in non-interactive runs.
- `-o, --output table|list|tree|yaml|json|toml|raw|csv` choose output format (default: `table`).
- `--limit N`, `--offset N`, `--tail N` apply record limiting after any expression; `--tail` ignores `--offset` and cannot combine with `--limit`.
- `--width N`, `--height N` override detected terminal size for TUI/snapshot/CLI bordered tables.
- `--theme <name>` select a theme (default from config, falls back to `midnight`); `--no-color` disables colors and box drawing.
- `--keymap vim|emacs|function` keybinding mode (default: `vim`); affects navigation keys in interactive mode.
- `--schema <path>` JSON Schema file for column display hints (title, maxLength, type, required, deprecated).
- `--config-file <path>` load config from a specific file; `--config` prints the merged config (defaults + XDG) in the chosen format.
- `--sort ascending|descending|none` pick map key ordering; `--debug` enable debug logging and `--debug-max-events N` cap stored debug events.

### Data formats and output

- Input auto-detects YAML/JSON (single or multi-doc), NDJSON, TOML (by extension or content), and CSV (by extension or stdin shape). If no input is provided, kvx shows help; with `--expression` but no input, it evaluates against an empty object `{}`.
- Non-interactive table output renders a bordered table with header/footer parity to the TUI; scalars print raw, and simple scalar arrays print one value per line.
- List output (`-o list`) displays data in a vertical format with each property on its own line. Arrays of objects show each element with an index header (`[0]`, `[1]`, etc.) and indented properties beneath. Maps display as `key: value` pairs, and scalars show as `value: <scalar>`.
- Tree output (`-o tree`) renders data as an ASCII tree structure using box-drawing characters. Nested objects become branches, arrays show indexed children, and scalar values appear inline. Options: `--tree-depth N` limits depth, `--tree-no-values` shows structure only, `--tree-expand-arrays` expands all array elements.
- Mermaid output (`-o mermaid`) generates Mermaid flowchart syntax for visualization in Markdown or diagram tools. Use `--mermaid-direction TD|LR|BT|RL` to set flow direction (default: TD for top-down).
- Array index style: `--array-style index|numbered|bullet|none` controls how array elements are labeled. Default is `index` (`[0]`, `[1]`); use `numbered` for `1, 2`, `bullet` for `•`, or `none` to hide indices (useful with `-o list`).
- CSV output is available for CLI/snapshot runs: arrays of objects become rows with merged headers, maps become key/value rows, other values emit a single `value` column.
- YAML output defaults to indent `2` and literal block strings; these options are configurable via `formatting.yaml.*` in the config.

### Logging Pattern
Uses the logr interface with zapr (zap adapter) for structured logging:
- `logger.Get(verbosity)` creates loggers with verbosity levels (negative numbers, e.g., `-1` for debug)
- Context-aware: `logger.WithLogger(ctx, lgr)` and `logger.FromContext(ctx)`
- Global keys defined in `logger/logger.go`: `RootCommandKey`, `CommitKey`, `VersionKey`, etc.
- Example: `lgr.V(1).Info("message", "key", value)` for verbose logging

## Config

Config merges built-in defaults with `~/.config/kvx/config.yaml` (or `XDG_CONFIG_HOME/kvx/config.yaml`); override with `--config-file`. Use `kvx --config` to print the merged view without reading input.

Manage and inspect configuration via the `config` command group:

```bash
# Show merged config (defaults + XDG config)
kvx config get

# As JSON/YAML/Table
kvx config get -o json
kvx config get -o yaml
kvx config get -o table

# Interactive TUI view of the config
kvx config get -i

# Use an explicit config file instead of XDG
kvx config get --config-file ~/.config/kvx/config.yaml
```

### Themes

- Themes are defined in your config (`ui.themes`) and selected via `ui.theme.default` (default: `midnight`). Built-ins: `midnight`, `dark`, `warm`, `cool`.
- List available themes from your merged config and see the default:

```bash
kvx config theme
# or with an explicit file:
kvx config theme --config-file ~/.config/kvx/config.yaml
```

- You can still pass a theme at runtime with `--theme <name>`; the default comes from your config.
- `--no-color` removes colors and the box-drawing borders for parity with plain terminals.

### Shell Completion

Generate shell completion scripts for bash, zsh, fish, or PowerShell:

```bash
# Bash (add to ~/.bashrc)
kvx completion bash > /etc/bash_completion.d/kvx

# Zsh (add to ~/.zshrc)
kvx completion zsh > "${fpath[1]}/_kvx"

# Fish
kvx completion fish > ~/.config/fish/completions/kvx.fish

# PowerShell
kvx completion powershell | Out-String | Invoke-Expression
```

### Shell Quoting

When using expressions with special characters, prefer single quotes around the entire expression:

- `'_.items[0]'`, `'type(_)'`, `'_.metadata["bad-key"]'`
- Double quotes require escaping internal quotes: `"_.metadata[\"bad-key\"]"`

```bash
# Print root as table
kvx tests/sample.yaml

# Navigate via CEL (CLI)
kvx tests/sample.yaml -e '_.items[0].name'

# Output as JSON
kvx tests/sample.yaml -e '_.metadata' -o json

# CEL functions
kvx tests/sample.yaml -e 'type(_)'

# Snapshot (render once) with scripted keys
kvx tests/sample.yaml --snapshot --press "<Right><Right>"

# Non-interactive search
kvx tests/sample.yaml --search status
```

### Path Syntax

- CLI (strict CEL): use `_` as the root variable
	- `_.metadata.created`, `_.items[0]`, `type(_)`
- TUI: dotted-path shorthand
	- `metadata.created`, `items[0]`, `items.0`, `items[0].name`

### Limiting Records

- `--limit N` shows the first N records after any `--expression` or filtering.
- `--offset N` skips the first N records before applying `--limit`.
- `--tail N` shows the last N records, ignores `--offset`, and cannot be combined with `--limit`.
- Applies to arrays by index order and maps by stable sorted key order (same as CLI table rendering).
- Non-interactive CLI output and snapshot TUI rendering use the same limiting rules; interactive scrolling does not change the applied limit.

## Interactive Mode (TUI)

**Keybinding modes:** kvx defaults to **vim** mode. Switch with `--keymap vim|emacs|function`.

### Vim mode (default)
| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `h` / `l` | Navigate back/forward (ascend/drill) |
| `/` | Search mode (live search keys/values) |
| `n` / `N` | Next/previous search match |
| `f` | Filter current map keys |
| `gg` / `G` | Go to top/bottom |
| `:` | Expression mode (CEL) |
| `y` | Copy current path/expression |
| `?` | Toggle help panel |
| `q` | Quit |
| `Esc` | Close input/help/search context (does not quit) |

### Emacs mode (`--keymap emacs`)
| Key | Action |
|-----|--------|
| `C-n` / `C-p` | Navigate up/down |
| `C-b` / `C-f` | Navigate back/forward |
| `C-s` | Search |
| `C-r` | Previous match |
| `M-<` / `M->` | Go to top/bottom |
| `M-x` | Expression mode |
| `M-w` | Copy path |
| `F1` | Toggle help |
| `C-g` | Cancel/clear |
| `C-q` | Quit |

### Function mode (`--keymap function`)
| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate up/down |
| `←` / `→` | Navigate back/forward |
| `Home` / `End` | Go to top/bottom |
| `F1` | Help |
| `F3` | Search |
| `F4` | Filter |
| `F5` | Copy |
| `F6` | Expression mode |
| `F10` | Quit |

**Panels (semantic names):**
- Data panel: main table view with path label and selection/total (`n/x`).
- Help panel (popup): F1 overlay with navigation help.
- Status bar: single-line info/error/status (right-aligned when input hidden).
- Input panel: single-line bordered input (Expression/Search titles); hidden until F3/F6.
- Footer: bottom line with `F1 - Help` on the left and `Rows/Cols` on the right.

**Filter & search:**
- With input hidden, typing filters rows by key prefix; backspace edits; `Esc` clears; filter clears when you drill/ascend.
- Search (F3) keeps input open; results update live; `Right/Enter` drill, `Left` backs up within search base; `Esc` restores the prior node.

**Expression (F6):**
- Starts with current path; Tab/Shift+Tab → keys/indices, Up/Down → CEL functions for the node type, Right accepts ghost completion.
- `Enter` evaluates and stays in expr; errors in red; non-navigable results fall back to your prior path when you exit.

## Embedding the TUI

kvx's TUI can be embedded into your own Go application using the public `tui` package:

```go
import (
  "log"
  "github.com/oakwood-commons/kvx/pkg/core"
  "github.com/oakwood-commons/kvx/pkg/tui"
)

func main() {
  // Load data from JSON/YAML/NDJSON (or build your own map).
  // Use LoadObject to pass already parsed Go values without re-serialization.
  data, err := core.LoadFile("data.yaml")
  if err != nil {
    log.Fatal(err)
  }

  cfg := tui.DefaultConfig()

  if err := tui.Run(data, cfg); err != nil {
    log.Fatal(err)
  }
}
```

See [docs/embedding.md](docs/embedding.md) for a deeper embed guide (custom CEL, themes, config, and data-loading options including `core.LoadObject`).

**Customization:**
- Extend CEL with custom functions via `tui.NewCELExpressionProvider()` and `tui.SetExpressionProvider()`
- Customize themes and UI settings via `tui.Config` (or start from `tui.DefaultConfig()`)
- Override navigation behavior with custom navigators

See [`examples/embed-tui/`](examples/embed-tui/) for a complete example with custom CEL functions.

### Core API (Load + Eval + Rows)

If you want to reuse kvx without the TUI, use the minimal core API:

```go
import (
  "fmt"
  "github.com/oakwood-commons/kvx/pkg/core"
)

func main() {
  root, err := core.LoadFile("data.yaml")
  if err != nil {
    panic(err)
  }

  engine, err := core.New(core.WithSortOrder(core.SortAscending))
  if err != nil {
    panic(err)
  }

  out, err := engine.Evaluate("_.items[0]", root)
  if err != nil {
    panic(err)
  }
  fmt.Println(out)
}
```

See [`examples/core-cli/`](examples/core-cli/) for a minimal core-only example.

## Development

```bash
task build          # Build binary
task run            # Run with sample data
task run-interactive # Run TUI
task test           # Run tests
go test ./...       # Direct test execution
```

### Project Structure

- `cmd/` - CLI commands (Cobra)
- `internal/navigator/` - Path/CEL navigation
- `internal/cel/` - CEL evaluator
- `internal/ui/` - TUI (Bubble Tea)
- `internal/formatter/` - Output formats
- `tests/`, `examples/data/` - Test fixtures

### Testing

Tests use fixture-based approach with deterministic Bubble Tea `Update()` calls:

```bash
task test
go test ./internal/ui -v
```

## Architecture

1. Parse YAML/JSON → `interface{}` (`map[string]any`, `[]any`)
2. CLI: evaluate `--expression` via CEL with `_` context
3. TUI: navigate dotted paths (sugar) or CEL in the path input
4. Output via formatters (non-interactive) or TUI (interactive)
