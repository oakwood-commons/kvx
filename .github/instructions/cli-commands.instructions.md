---
description: "CLI command layer rules for kvx. Commands are thin wiring -- no business logic. Use cobra for flags, pkg/core for engine logic, pkg/tui for interactive mode. Use when editing CLI command files."
applyTo: "cmd/**/*.go"
---

# CLI Command Layer

Commands are **thin wiring only** -- they parse flags, call domain packages, and render output.

## Rules

- **No business logic** -- delegate to `pkg/core` (engine), `pkg/tui` (interactive mode), `pkg/loader` (data loading), and `internal/` packages
- Use `cobra.Command` for command definition and flag binding
- CLI flags are defined in `cmd/root.go`; keep all flag wiring there
- Data loading flows through `pkg/loader.LoadData()` or `pkg/core.LoadFile()`/`pkg/core.LoadObject()`
- Interactive mode launches via `pkg/tui.Run()`; non-interactive renders via `pkg/tui.RenderTable()` or output formatters
- Use `pkg/logger` for structured logging, never `fmt.Println` for debug output
