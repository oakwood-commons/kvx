---
title: "Interactive Mode"
weight: 2
---

# Interactive Mode

The interactive TUI (`-i` / `--interactive`) provides keyboard-driven tree navigation, instant prefix filtering, deep search, live CEL evaluation with a categorized Function Palette, clipboard copy actions, and in-place decoding of serialized strings.

## Launching the TUI

```bash
# Open a file in the interactive TUI
kvx examples/data/demo_services.json -i

# Attach a JSON Schema for Card-List and Detail views
kvx examples/data/demo_services.json --schema examples/data/demo_services_schema.json -i

# Pipe command output into the TUI
kubectl get pods -o json | kvx -i

# Override detected terminal dimensions
kvx examples/data/demo_services.json -i --width 110 --height 32
```

## Panel Layout

The TUI adapts dynamically to terminal resizes without leaving blank rows below the footer:

- **Data panel**: Displays the active `KEY`/`VALUE` table, schema Card-List, Sectioned Detail screen, or decoded scalar view. The bottom border shows the current `_`-rooted CEL path on the left and the node type plus `selection/total` count (such as `map: 1/8` or `list: 2/5`) on the right.
- **Function Palette overlay**: Opens above the data panel when you press `Ctrl+Space` inside Expression mode, grouping all CEL functions and methods by category.
- **Input panel**: Single-line bordered input bar titled `Expression` or `Search`. Hidden until `/` or `:` activates it.
- **Status / Info row**: Single-line feedback row showing copy confirmations, `decoded` indicators, CEL type/function hints, or evaluation errors.
- **Footer**: Persistent bottom shortcut bar listing context-aware keys for your active `--keymap`.
- **Help modal**: Toggled with `?` (Vim) or `F1` (Emacs/Function).

## Keybindings Across Modes

`kvx` defaults to `vim` keybindings and also supports `emacs` and `function` keymaps via `--keymap vim|emacs|function`, `KVX_KEY_MODE`, or `ui.features.key_mode` in `config.yaml`:

| Action | Vim (`default`) | Emacs (`--keymap emacs`) | Function (`--keymap function`) |
|--------|-----------------|--------------------------|--------------------------------|
| Move down / up | `j` / `k` or `Down` / `Up` | `Ctrl+N` / `Ctrl+P` or `Down` / `Up` | `Down` / `Up` |
| Drill in / decode scalar | `l`, `Enter`, or `Right` | `Ctrl+F`, `Enter`, or `Right` | `Enter` or `Right` |
| Ascend to parent | `h` or `Left` | `Ctrl+B` or `Left` | `Left` |
| Jump to top / bottom | `gg` / `G` (`Home` / `End`) | `Alt+<` / `Alt+>` (`Home` / `End`) | `Home` / `End` |
| Toggle help overlay | `?` or `F1` | `F1` | `F1` |
| Toggle schema view vs raw table | `v` | `Alt+v` | `F2` |
| Deep search (`keys` + `values`) | `/` (`n` / `N` next/prev) | `Ctrl+S` | `F3` |
| Filter current map keys | `f` | `Ctrl+L` | `F4` |
| Copy current path or expression | `y` | `Alt+w` | `F5` |
| Copy current value (scalar or JSON) | `Y` | `Alt+W` | *(unbound; set via `ui.menu.copy_value`)* |
| Open CEL Expression mode | `:` | `Alt+x` | `F6` |
| Open CEL Function Palette (in expr) | `Ctrl+Space` | `Ctrl+Space` | `Ctrl+Space` |
| Close active input / popup / search | `Esc` | `Esc` or `Ctrl+G` | `Esc` |
| Quit `kvx` | `q` or `Ctrl+C` | `Ctrl+Q` or `Ctrl+C` | `F10` or `Ctrl+C` |

## Copying Paths (`y`) vs Copying Values (`Y`)

`kvx` provides two context-aware clipboard actions:

- **`y` (`Alt+w` / `F5`)**: In table view, always copies the `_`-rooted CEL path of the **currently highlighted row** right from frame 1 without requiring a cursor move first (for example `Copied: _[1].config` or `Copied: _.metadata["bad-key"]`). While `:` Expression input is focused, pressing `F5` copies the typed expression string itself (typing `y` or `Y` inside the focused input inserts literal characters into the expression).
- **`Y` (`Shift+y` / `Alt+W`; unbound by default in `function` mode)**: Copies the **value** at the highlighted cursor position:
  - **Scalars** (`string`, `int`, `float`, `bool`, `nil`) copy as raw unquoted text (`Copied value: auth-issuer`).
  - **Maps and arrays** copy as pretty-printed, 2-space indented JSON (`Copied value (JSON, 148 bytes)`), ready to paste into an editor, fixture, or HTTP request.
  - **Evaluating & copying CEL results**: Press `Enter` inside `:` Expression mode to evaluate a navigable result, press `Esc` to focus the result table, and press `Y` (`Alt+W`) to copy it (or bind `ui.menu.copy_value.keys.function`, such as `f7`, to copy evaluated expression values directly while focused).

## Type-Ahead Row Filtering

When the input bar is closed in table view, typing printable characters (`a` through `z`, digits, or spaces) immediately filters the current node's rows by key prefix (case-insensitive):

- Press `Backspace` to edit the prefix filter.
- Press `Esc` to clear the filter.
- Drilling into a child node or ascending to a parent automatically resets the prefix filter.

## Deep Search (`/`)

Press `/` to search recursively across all nested keys and scalar values from the current node downward:

1. Type your query and press `Enter` to commit the deep search (up to `ui.performance.search_result_limit`, default `500`).
2. Use `Up` / `Down` arrow keys to move through matching paths while the Search input is open.
3. Press `Right` to drill into a match while preserving your search session, or `Enter` to jump into the matched node and close search.
4. Press `Esc` to cancel search and return to your starting node.

## Decoding Serialized Strings (`Enter`)

Many real-world payloads embed JSON, YAML, TOML, NDJSON, or `Bearer` JWT strings inside scalar fields (such as Kubernetes annotations, Terraform state blobs, or audit event payloads).

Highlight any string leaf and press `Enter` (or `l` / `Right`). If the string parses into structured data, `kvx` decodes it in place (`decoded` status badge) and lets you navigate its inner fields like any other map or list. Press `h` (`Left`) to ascend back to the parent object. See [Formats, JWTs & Decoding](../formats-and-decoding/) for details.

## Toggling Schema Views (`v`)

When launched with `--schema` containing `x-kvx-list` or `x-kvx-detail` extensions, `kvx` renders arrays as rich Card Lists and objects as Sectioned Detail views. Press `v` (`Alt+v` in Emacs, `F2` in Function mode) on either a Card-List or a Detail screen to switch between the schema-driven view and the underlying raw `KEY`/`VALUE` table:

- **Context-aware footer**: The `v view` shortcut only appears when the current node has a matching schema view (list or detail); drilling into a deeper nested field where no schema view applies hides `v view` automatically.
- **Copy integration**: While a schema Card-List or Detail view is active, `y` and `Y` copy hints stay hidden to keep the footer concise; pressing `v` switches into the raw table and immediately restores `y copy` and `Y copy value` so you can copy any field path, hidden property, or JSON subtree.

{{< svg-card src="images/tui/schema-card-list.svg" alt="kvx schema card-list view in interactive TUI" >}}Card-List view rendered from `x-kvx-list`; press `Enter` for Detail view or `v` for the raw table.{{< /svg-card >}}

## Scripted Snapshots (`--snapshot` and `--press`)

Combine `--snapshot` with `--press` to drive reproducible multi-step TUI interactions in scripts, tests, and documentation pipelines:

```bash
# Open Expression mode, launch the Ctrl+Space Function Palette, and switch to list functions
kvx examples/data/demo_services.json \
  --snapshot --width 94 --height 26 \
  --press ":<C-u>_<C-Space><Tab><Tab><Down><Down><Down><Down>"

# Drill into item 1 and decode its Bearer JWT field
kvx examples/data/demo_services.json \
  --snapshot --width 92 --height 20 \
  --press "jl<Enter><Enter>j"
```

Supported angle-bracket key tokens in `--press`:
`<Enter>`, `<Esc>`, `<Tab>`, `<Space>`, `<BS>`, `<Left>`, `<Right>`, `<Up>`, `<Down>`, `<Home>`, `<End>`, `<C-c>`, `<C-d>`, `<C-u>`, `<C-Space>`, and `<F1>` through `<F12>`. Including `<F10>` in `--press` bypasses the interactive event loop and emits non-interactive CLI output directly regardless of `--keymap`.

## Debug Event Capture

Pass `--debug` (and optional `--debug-max-events N`, default `200`) to buffer timestamped UI, key-routing, and navigation events in memory and emit them cleanly via structured logging upon exit:

```bash
kvx examples/data/demo_services.json -i --debug --debug-max-events 500
```
