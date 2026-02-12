# TUI quick guide

## Launch & sizing

- Interactive: `kvx <file> -i`; one-shot renders use `--snapshot` (no `-x` alias). Add `--press` to script startup keys.
- Size: `--width/--height` override terminal size; otherwise we detect and react to resizes.
- No blank rows below the footer; panels scale proportionally when resizing.

## Panels

- Footer: always visible; left shows `? - Help`, right shows rows/cols.
- Data panel: fills remaining space; bottom border shows current CEL path (tail is kept when long); lower-right of the data panel shows `n/x` for selection/visible rows (footer still shows rows/cols).
- Input panel: one-line bordered; title reflects mode (Expression/Search); hidden by default.
- Info panel: one row, borderless; right-justified when input is hidden, left-justified and recolored when input is shown.
- Popup: hidden by default; `?` toggles help content.

## Navigation basics

kvx defaults to **vim** keybinding mode:

- `j`/`k`: navigate up/down; `h`/`l`: ascend/drill into selection.
- `/`: search/filter; `n`/`N`: next/prev match; `f`: filter map keys.
- `gg`/`G`: go to top/bottom.
- `:`: expression mode; `y`: copy path; `?`: toggle help; `q`: quit.
- `Esc`: close open contexts (input/search/popup) but do not exit.

Prefer **emacs** or **function-key** style bindings? Use `--keymap emacs` or `--keymap function`.

## Filter (type-ahead)

- With input hidden, typing letters/digits/space filters rows by key prefix (case-insensitive); backspace edits; `Esc` clears.
- Navigation works on filtered rows; `n/x` reflects filtered counts; filter clears when you drill/ascend.

## Search (/)

- Opens input with search prompt; results update live (keys and values).
- While active: up/down/left/right move within results; `Right` drills but keeps search context; `Left` stops at search base path.
- `Enter` drills and exits search; `Esc` exits and restores prior node; query stays visible until exit.

## Expression (:)

- Toggles expression mode; starts at current path (prefilled with `_`-rooted path).
- Tab/Shift+Tab cycle keys/indices; Up/Down cycle CEL functions valid for the current node type; Right accepts ghosted completion.
- `Enter` evaluates the expression and stays in expr mode; errors show in red; results render in the data panel.
- `Esc` exits expr mode; non-navigable results fall back to the path you started from.
- While in expr mode, `y` copies the current expression.

## Snapshot & scripted runs

- `--snapshot` renders the view once and exits; honors width/height and themes. No short alias.
- `--press` runs the TUI and feeds a sequence of key presses for reproducible end states.
  - Special keys use angle brackets: `<Enter>`, `<Esc>`, `<Tab>`, `<Space>`, etc.
  - Literal text is typed normally without brackets
  - Examples:
    - Search for "pd1030": `--press "/pd1030"`
    - Navigate with expression: `--press ":_.items[0]"`
    - Open help then close: `--press "?<Esc>"`
    - Multiple operations: `--press "/test<Esc>me"`
  - Available special keys: `<Enter>`, `<Esc>`, `<Tab>`, `<Space>`, `<BS>` (backspace), `<Left>`, `<Right>`, `<Up>`, `<Down>`, `<Home>`, `<End>`, `<C-c>`, `<C-d>`, `<F1>`–`<F12>`
- Include `<F10>` in `--press` to bypass the interactive loop and emit the non-interactive output directly (works regardless of `--keymap`).

## Debug

- `--debug` buffers recent debug events and prints them on exit; adjust the cap with `--debug-max-events` (default 200).

## Themes

- Built-ins: midnight (default), dark, warm, cool — loaded from `internal/ui/default_config.yaml`.
- Select with `--theme <name>`; `--no-color` disables colors/box drawing.
- User config merges with defaults; custom themes can be added via config; `--config` prints the merged config for editing.
