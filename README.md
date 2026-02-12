# kvx

**kvx** is a terminal-based UI for exploring structured data like JSON, YAML, and TOML in an interactive, navigable way. It presents data as key value trees that you can expand, collapse, and inspect directly in the terminal, making it easy to understand complex or deeply nested structures. In addition to being a standalone CLI, kvx is designed as a reusable TUI component, allowing other applications to embed the viewer directly into their own terminal interfaces for consistent data inspection and visualization.

## Usage

```bash
kvx <file> [flags]
kvx tests/sample.yaml           # table output
kvx tests/sample.yaml -i        # interactive TUI
```

See [docs/development.md](docs/development.md) for build instructions and project setup.

### Flags

- `-i, --interactive` launch the TUI; `--snapshot` renders once and exits using the same layout as the TUI.
- `--press "<keys>"` script startup keys (e.g., `/name<Enter>`); include `<F10>` to bypass the TUI and emit non-interactive output (works regardless of `--keymap`).
- `-e, --expression <cel>` evaluate CEL against `_` (e.g., `_.items[0].name`, `type(_)`); dotted shorthand stays TUI-only.
- `--search <text>` search keys/values; seeds search mode in TUI and prints a bordered table in non-interactive runs.
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

kvx defaults to **vim** keybindings:

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

Prefer **emacs** or **function-key** style bindings? Use `--keymap emacs` or `--keymap function`.

**Panels:**
- Data panel: main table view with path label and selection/total (`n/x`).
- Help panel: overlay with navigation help (`?` to toggle).
- Status bar: single-line info/error/status (right-aligned when input hidden).
- Input panel: single-line bordered input (Expression/Search titles); hidden until activated.
- Footer: bottom line with Help hint on the left and Rows/Cols on the right.

**Filter & search:**
- With input hidden, typing filters rows by key prefix; backspace edits; `Esc` clears; filter clears when you drill/ascend.
- Search (`/`) keeps input open; results update live; `Right/Enter` drill, `Left` backs up within search base; `Esc` restores the prior node.

**Expression (`:`):**
- Starts with current path; Tab/Shift+Tab → keys/indices, Up/Down → CEL functions for the node type, Right accepts ghost completion.
- `Enter` evaluates and stays in expr; errors in red; non-navigable results fall back to your prior path when you exit.

## Embedding the TUI

kvx's TUI can be embedded into your own Go application. See [docs/embedding.md](docs/embedding.md) for the full guide covering data loading, custom CEL functions, themes, and configuration. For a working example, see [`examples/embed-tui/`](examples/embed-tui/).
