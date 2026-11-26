---
title: "TUI Quick Guide"
weight: 10
---

# TUI Quick Guide

Fast lookup reference for `kvx` interactive panels, keymaps, CLI flags, scripted snapshots, and schema-driven display controls.

## Launch & Sizing

- **Interactive TUI**: `kvx <file> -i` (or pipe `stdin`: `cat data.json | kvx -i`).
- **One-shot TUI frame**: `kvx <file> --snapshot` renders the exact TUI layout once and exits. Combine with `--press "<keys>"` for deterministic UI automation.
- **Dimensions**: `--width N` and `--height N` override auto-detected terminal dimensions across TUI, `--snapshot`, and bordered CLI tables.
- **Schema views**: `--schema <file.json>` enables standard JSON Schema column hints plus `x-kvx-list`, `x-kvx-detail`, and `x-kvx-status` views.

## Panels

- **Data panel**: Displays the current table, Card-List (`x-kvx-list`), Sectioned Detail (`x-kvx-detail`), Status screen (`x-kvx-status`), or decoded scalar tree. Bottom border shows the `_`-rooted CEL path on the left and `type: selection/total` on the right.
- **Function Palette (`Ctrl+Space`)**: Categorized overlay inside Expression mode showing signatures, descriptions, type-to-filter search, and live CEL examples.
- **Input panel**: Single-line bordered input (`Expression` or `Search`), shown only while active.
- **Info / Status row**: Displays copy confirmations, `decoded` badges, CEL completion hints, and evaluation errors.
- **Footer**: Context-aware shortcut bar at the bottom of the screen.
- **Help modal**: Toggled via `?` (Vim) or `F1` (Emacs/Function).

## Keybinding Reference (`vim`, `emacs`, `function`)

Select a mode via `--keymap vim|emacs|function` or `KVX_KEY_MODE` (defaults to `vim`):

| Action | Vim (`default`) | Emacs | Function |
|--------|-----------------|-------|----------|
| Move down / up | `j` / `k`, `Down` / `Up` | `Ctrl+N` / `Ctrl+P`, `Down` / `Up` | `Down` / `Up` |
| Drill into child / decode scalar | `l`, `Enter`, `Right` | `Ctrl+F`, `Enter`, `Right` | `Enter`, `Right` |
| Ascend to parent | `h`, `Left` | `Ctrl+B`, `Left` | `Left` |
| Top / bottom row | `gg` / `G`, `Home` / `End` | `Alt+<` / `Alt+>`, `Home` / `End` | `Home` / `End` |
| Toggle help modal | `?`, `F1` | `F1` | `F1` |
| Toggle schema view vs raw table | `v` | `Alt+v` | `F2` |
| Deep search (`keys` + `values`) | `/` (`n` / `N` next/prev) | `Ctrl+S` | `F3` |
| Filter current map keys | `f` | `Ctrl+L` | `F4` |
| Copy current path or expression | `y` | `Alt+w` | `F5` |
| Copy current value (raw or JSON) | `Y` | `Alt+W` | *(unbound; set via `ui.menu.copy_value`)* |
| Toggle CEL Expression mode | `:` | `Alt+x` | `F6` |
| Toggle CEL Function Palette | `Ctrl+Space` *(in expr mode)* | `Ctrl+Space` *(in expr mode)* | `Ctrl+Space` *(in expr mode)* |
| Close input / popup / search | `Esc` | `Esc`, `Ctrl+G` | `Esc` |
| Quit | `q`, `Ctrl+C` | `Ctrl+Q`, `Ctrl+C` | `F10`, `Ctrl+C` |

## Modes & Workflows

### Prefix Filter (Type-Ahead)
- When the input bar is hidden in table view, typing letters, digits, or spaces filters current rows by key prefix (case-insensitive).
- `Backspace` edits; `Esc` clears; drilling or ascending resets the filter automatically.

### Deep Search (`/`)
- Searches nested keys and values recursively from the current node when you press `Enter`.
- `Up` / `Down` move through matching paths; `Right` drills into a result while keeping search active; `Left` backs up to the search base path; `Esc` restores the prior node.

### CEL Expression Mode (`:`) & Function Palette (`Ctrl+Space`)
- Prefills the current `_`-rooted path (`dotted.path` shorthand also works interactively).
- `Tab` / `Shift+Tab` cycle keys and array indices; `Up` / `Down` cycle type-compatible CEL functions; `Right` accepts dimmed ghost completion.
- `Ctrl+Space` opens the categorized Function Palette (`Tab` switches categories, typing filters across all categories, `Enter` inserts the selected function or method).
- `Enter` evaluates live in the Data panel; `F5` copies the expression string while focused; `Esc` closes the input bar so `y` / `Y` act on the resulting table (falling back to the prior path if the result is non-navigable).

### Serialized Scalar Decoding (`Enter` and `--auto-decode`)
- Pressing `Enter` or `l` on a string leaf containing JSON, YAML, TOML, NDJSON, or a `Bearer` JWT decodes it in place (`decoded` status badge) so you can navigate its inner structure.
- On the CLI or at startup, `--auto-decode lazy` decodes target scalar results on navigation, and `--auto-decode eager` recursively decodes all string leaves (up to depth 20) before `--where` and `--expression` run.

## CLI Flag Summary

| Flag | Description |
|------|-------------|
| `-i, --interactive` | Launch the interactive TUI |
| `-o, --output <fmt>` | Output mode: `auto` (default), `table`, `list`, `tree`, `mermaid`, `yaml`, `json`, `toml`, `csv`, `raw` |
| `-w, --where <cel>` | Per-item boolean CEL filter for list data (`_` = current item) |
| `-e, --expression <cel>` | Evaluate CEL expression against root `_` |
| `--auto-decode <mode>` | Decode serialized string leaves: `lazy`, `eager`, or `disabled` (default manual via `Enter`) |
| `--schema <path>` | JSON Schema file for column hints and `x-kvx-*` TUI views (`x-kvx-list`, `x-kvx-detail`, `x-kvx-status`) |
| `--config-file <path>` | Load configuration from a custom YAML file instead of `~/.config/kvx/config.yaml` |
| `--column-order <keys>` | Comma-separated preferred key/column order |
| `--array-style <style>` | Array index format: `none` (default), `index`, `numbered`, `bullet` |
| `--search <query>` | Case-insensitive deep search across keys and values |
| `--sort <order>` | Map key order: `ascending`, `descending`, `none` |
| `--limit N`, `--offset N`, `--tail N` | Slice records after `--where` and `--expression` (`--tail` is mutually exclusive with `--limit`) |
| `--theme <name>` | Theme preset (`midnight`, `dark`, `warm`, `cool`, or custom config theme); `--no-color` disables ANSI color codes while preserving borders and layout |
| `--keymap <mode>` | Keybinding preset: `vim` (default), `emacs`, `function` (also configurable via `KVX_KEY_MODE`) |
| `--snapshot` | Render one TUI frame and exit; pair with `--press "<keys>"`, `--width`, and `--height` |
| `--tree-depth N`, `--tree-no-values`, `--tree-expand-arrays`, `--tree-max-string N` | Customize `-o tree` output |
| `--mermaid-direction <dir>` | Set `-o mermaid` flow direction: `TD` (default), `LR`, `BT`, `RL` |
| `--debug`, `--debug-max-events N` | Buffer and emit structured debug events on exit |

## Schema-Driven Column Hints & Extensions

`--schema` derives table `ColumnHint` settings from standard JSON Schema keywords (`title`, `maxLength`, `enum`, `format`, `type`, `deprecated`, `required`, plus automatic `Flex` sizing for unconstrained columns) and activates rich TUI views when `x-kvx-*` vendor extensions are present:

```json
{
  "type": "array",
  "x-kvx-collectionTitle": "Providers",
  "x-kvx-list": {
    "titleField": "name",
    "subtitleField": "description",
    "badgeFields": ["category", "version"]
  },
  "items": {
    "type": "object",
    "required": ["name", "version"],
    "properties": {
      "name": { "title": "Name", "type": "string", "maxLength": 20 },
      "version": { "title": "Version", "type": "string", "maxLength": 12 },
      "description": { "title": "Description", "type": "string" },
      "legacy_field": { "deprecated": true }
    }
  }
}
```

See the [Schema-Driven Views tutorial](/docs/tutorials/schema-views/) and [Library Usage](/docs/library-usage/#column-display-hints--json-schema-parseschemawithdisplay) for full examples covering `x-kvx-list`, `x-kvx-detail`, and `x-kvx-status`.
