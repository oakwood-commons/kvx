# TUI quick guide

## Launch & sizing

- Interactive: `kvx <file> -i`; one-shot renders use `--snapshot` (no `-x` alias). Add `--press` to script startup keys.
- Size: `--width/--height` override terminal size; otherwise we detect and react to resizes.
- No blank rows below the footer; panels scale proportionally when resizing.

## Panels

- Footer: always visible; left shows `F1 - Help`, right shows rows/cols.
- Data panel: fills remaining space; bottom border shows current CEL path (tail is kept when long); lower-right of the data panel shows `n/x` for selection/visible rows (footer still shows rows/cols).
- Input panel: one-line bordered; title reflects mode (Expression/Search); hidden by default.
- Info panel: one row, borderless; right-justified when input is hidden, left-justified and recolored when input is shown.
- Popup: hidden by default; `F1` toggles help content.

## Navigation basics

kvx defaults to **vim** keybinding mode. Switch with `--keymap vim|emacs|function`.

### Vim mode (default)
- `j`/`k`: navigate up/down; `h`/`l`: ascend/drill into selection.
- `/`: search/filter; `n`/`N`: next/prev match; `f`: filter map keys.
- `gg`/`G`: go to top/bottom.
- `:`: expression mode; `y`: copy path; `?`: toggle help; `q`: quit.
- `Esc`: close open contexts (input/search/popup) but do not exit.

### Emacs mode (`--keymap emacs`)
- `C-n`/`C-p`: up/down; `C-b`/`C-f`: back/forward.
- `C-s`: search; `C-r`: prev match; `M-<`/`M->`: top/bottom.
- `M-x`: expression; `M-w`: copy; `F1`: help; `C-g`: cancel; `C-q`: quit.

### Function mode (`--keymap function`)
- Arrows: navigate table; `Left` ascends; `Right/Enter` drill.
- `Home`/`End`: top/bottom.
- `F1` Help, `F3` Search, `F4` Filter, `F5` Copy, `F6` Expression, `F10` Quit.

## Filter (type-ahead)

- With input hidden, typing letters/digits/space filters rows by key prefix (case-insensitive); backspace edits; `Esc` clears.
- Navigation works on filtered rows; `n/x` reflects filtered counts; filter clears when you drill/ascend.

## Search (F3)

- Opens input with search prompt; results update live (keys and values).
- While active: up/down/left/right move within results; `Right` drills but keeps search context; `Left` stops at search base path.
- `Enter` drills and exits search; `Esc` exits and restores prior node; query stays visible until exit.

## Expression (F6)

- Toggles expression mode; starts at current path (prefilled with `_`-rooted path).
- Tab/Shift+Tab cycle keys/indices; Up/Down cycle CEL functions valid for the current node type; Right accepts ghosted completion.
- `Enter` evaluates the expression and stays in expr mode; errors show in red; results render in the data panel.
- `Esc` exits expr mode; non-navigable results fall back to the path you started from.
- While in expr mode, `F5` copies the current expression.

## Snapshot & scripted runs

- `--snapshot` renders the view once and exits; honors width/height and themes. No short alias.
- `--press` runs the TUI and feeds a sequence of key presses for reproducible end states.
  - Special keys use angle brackets: `<F1>`, `<F3>`, `<F6>`, `<Enter>`, `<Esc>`, `<Tab>`, `<Space>`, etc.
  - Literal text is typed normally without brackets
  - Examples:
    - Search for "pd1030": `--press "<F3>pd1030"`
    - Navigate with expression: `--press "<F6>_.items[0]"`
    - Open help then close: `--press "<F1><Esc>"`
    - Multiple operations: `--press "<F3>test<Esc>me"`
  - Available special keys: `<F1>` through `<F12>`, `<Enter>`, `<Esc>`, `<Tab>`, `<Space>`, `<BS>` (backspace), `<Left>`, `<Right>`, `<Up>`, `<Down>`, `<Home>`, `<End>`, `<C-c>`, `<C-d>`
- Include `<F10>` in `--press` when you want to bypass the interactive loop and emit the non-interactive output directly (useful for scripted parity checks).

## Debug

- `--debug` buffers recent debug events and prints them on exit; adjust the cap with `--debug-max-events` (default 200).

## Themes

- Built-ins: midnight (default), dark, warm, cool — loaded from `internal/ui/default_config.yaml`.
- Select with `--theme <name>`; `--no-color` disables colors/box drawing.
- User config merges with defaults; custom themes can be added via config; `--config` prints the merged config for editing.
