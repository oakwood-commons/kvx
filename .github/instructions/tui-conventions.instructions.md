---
description: "TUI conventions for kvx: bubbletea model patterns, lipgloss styling, navigator usage, terminal state management, and interactive mode best practices. Use when writing or editing TUI code."
applyTo: "pkg/tui/**/*.go,internal/navigator/**/*.go,internal/ui/**/*.go"
---

# TUI Conventions

## Testing TUI Changes

Interactive UI is hard to test. **Always verify TUI changes with snapshot tests** before considering them done.

### Snapshot Testing

Snapshot tests capture the rendered TUI output and compare it against a known-good baseline in `tests/snapshots/`. This is the primary verification method for visual correctness.

```bash
# Run kvx in snapshot mode (renders TUI and exits)
go run . tests/sample.yaml --snapshot --no-color --width 80 --height 30

# With key presses to test a specific state
go run . tests/sample.yaml --snapshot --no-color --width 80 --height 30 --press "<F1>"
```

- Snapshot baselines live in `tests/snapshots/` (e.g., `default-80x30.txt`, `f1-80x30.txt`)
- Update all snapshots: `scripts/update_snapshots.sh`
- Use `--no-color` and fixed `--width`/`--height` for deterministic output
- Use `--press "..."` to simulate key sequences and test specific UI states
- **Never hand-edit snapshot files** -- always regenerate with the script
- When adding a new TUI feature or key binding, add a corresponding snapshot test

### When to Add Snapshots

- New panel or layout change
- New key binding that alters the display
- Changes to table rendering, borders, or styling
- Changes to status bar, breadcrumbs, or help overlay

### Verification Workflow

After any TUI change:
1. Run existing snapshot comparisons to check for regressions
2. Visually inspect with `--snapshot --press "..."` for the affected states
3. Update baselines if the change is intentional: `scripts/update_snapshots.sh`
4. Commit updated snapshot files alongside the code change

## Bubbletea Model Pattern

All TUI components follow the bubbletea `Model` interface:

```go
type Model struct {
    // immutable config (set at construction)
    // mutable state (updated in Update)
}

func (m Model) Init() tea.Cmd    { /* initial commands */ }
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { /* handle messages */ }
func (m Model) View() string     { /* render to string */ }
```

### Rules

- `Update` must return a new model value (copy-on-write) -- never mutate and return the same pointer
  - **Note**: the existing `internal/ui.Model` uses pointer receivers and mutates in place for historical reasons. This rule applies to **new** bubbletea components; do not refactor the existing Model to match
- `Update` must always return a `tea.Cmd` (use `nil` only when no command is needed)
- `View` must be pure -- no side effects, no state mutation
- Use `tea.Batch()` to combine multiple commands
- Use custom `tea.Msg` types for component communication

## Lipgloss Styling

- Use lipgloss for ALL terminal styling -- never raw ANSI escape codes
- Define styles as package-level variables or in a theme struct
- Account for border and padding widths when setting `Width()` or `MaxWidth()`:

```go
// WRONG: content will be clipped by border width
style := lipgloss.NewStyle().Width(80).Border(lipgloss.RoundedBorder())

// RIGHT: subtract border width from content width
borderWidth := 2 // left + right
style := lipgloss.NewStyle().Width(80 - borderWidth).Border(lipgloss.RoundedBorder())
```

- Use `lipgloss.Place()` for alignment, not manual padding

## Navigator State

The navigator tracks cursor position and drill-down state for data exploration.

### Rules

- All navigator state mutations go through bubbletea's `Update` message loop -- never mutate from goroutines
- Cursor position must stay in bounds after data changes (filter, drill-down, drill-up)
- Breadcrumb path must stay in sync with the current data view
- Use functional options (`navigator.WithXxx()`) for navigator configuration

## Terminal State Cleanup

- Always restore terminal state on exit (bubbletea handles this, but custom code must not bypass it)
- `defer cancel()` immediately after context creation, before any early returns
- Clean up goroutines on quit -- use `context.Context` for cancellation
- Never leave orphan goroutines after the TUI exits

## Key Bindings

- Define key bindings in a `keyMap` struct implementing `help.KeyMap`
- Use `key.NewBinding()` with both `key.WithKeys()` and `key.WithHelp()`
- Group related bindings (navigation, search, mode switching)
- Document all bindings in the help overlay

## Panels and Layout

- Each panel (data, status, input) is a separate component with its own `View()`
- Layout composition happens in the top-level model's `View()`
- Use lipgloss `JoinHorizontal()` / `JoinVertical()` for panel composition
- Panels must handle zero-width/zero-height gracefully (terminal resize edge case)
