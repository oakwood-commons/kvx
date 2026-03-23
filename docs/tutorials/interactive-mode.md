---
title: "Interactive Mode"
weight: 2
---

# Interactive Mode

The interactive TUI lets you navigate, search, filter, and evaluate expressions against your data in a full-screen terminal interface.

## Launching the TUI

```bash
kvx data.yaml -i
```

Override terminal size detection:

```bash
kvx data.yaml -i --width 120 --height 40
```

## Panels

The TUI is organized into panels:

- **Data panel** — Main table view. The bottom border shows the current CEL path (tail is kept when the path is long). The lower-right corner shows `n/x` for selection/visible rows.
- **Input panel** — Single-line bordered input. Title reflects the current mode (Expression or Search). Hidden until activated.
- **Info panel** — One row, borderless. Right-justified when input is hidden. Left-justified and recolored when input is shown.
- **Footer** — Always visible. Left shows `? - Help`, right shows rows/cols.
- **Help popup** — Hidden by default. Press `?` to toggle help content.

## Navigation

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

### Emacs keybindings

```bash
kvx data.yaml -i --keymap emacs
```

### Function-key keybindings

```bash
kvx data.yaml -i --keymap function
```

## Filter (Type-Ahead)

With input hidden, typing letters, digits, or spaces filters rows by key prefix (case-insensitive):

- **Backspace** edits the filter
- **Esc** clears the filter
- Navigation works on filtered rows
- The `n/x` indicator reflects filtered counts
- Filter clears automatically when you drill into or ascend from a node

## Search

Press `/` to enter search mode:

- Type your query in the input bar — the query updates as you type but the deep search executes when you press **Enter**
- **Enter** commits the search and shows matching results (searches both keys and values)
- **Up/Down** move within results
- **Right** drills into a result but keeps search context
- **Left** backs up but stops at the search base path
- **Esc** exits search and restores the prior node

The search query stays visible until you exit search mode.

## Expression Mode

Press `:` to enter expression mode:

- Starts with the current path (prefilled with `_`-rooted CEL path)
- **Tab / Shift+Tab** — cycle through keys/indices
- **Up / Down** — cycle through CEL functions valid for the current node type
- **Right** — accept ghosted completion
- **Enter** — evaluate the expression and stay in expression mode; errors show in red
- **Esc** — exit expression mode; non-navigable results fall back to the path you started from
- While in expression mode, **y** copies the current expression

### Expression examples

```
_.metadata                        # navigate to metadata
_.items[0]                        # first item
_.items.filter(x, x.status == "running")  # filter running items
_.items.map(x, x.name)           # extract names
type(_)                           # inspect root type
```

## Snapshot and Scripted Runs

### Snapshot

Render the TUI layout once and exit — useful for CI or scripted output:

```bash
kvx data.yaml --snapshot
```

Honors `--width`, `--height`, and `--theme`.

### Scripted Keys (--press)

Feed a sequence of key presses for reproducible end states:

```bash
# Search for "api"
kvx data.yaml --snapshot --press "/api<Enter>"

# Navigate with expression
kvx data.yaml --snapshot --press ":_.items[0]<Enter>"

# Open help then close
kvx data.yaml --snapshot --press "?<Esc>"

# Multiple operations: search then filter
kvx data.yaml --snapshot --press "/test<Enter><Esc>me"
```

Special keys use angle brackets: `<Enter>`, `<Esc>`, `<Tab>`, `<Space>`, `<BS>` (backspace), `<Left>`, `<Right>`, `<Up>`, `<Down>`, `<Home>`, `<End>`, `<C-c>`, `<C-d>`, `<C-u>`, `<C-Space>`, `<F1>`–`<F12>`.

Include `<F10>` in `--press` to bypass the interactive loop and emit non-interactive output directly (works regardless of `--keymap`).

## Debug

```bash
kvx data.yaml -i --debug
```

Buffers recent debug events and prints them on exit. Adjust the buffer size with `--debug-max-events N` (default: 200).

## Next Steps

- [CEL Expressions](../expressions/) — Advanced querying patterns
- [Configuration & Themes](../configuration/) — Customize themes, keymaps, and schemas
