# kvx - AI Agent Instructions

## Project Overview
`kvx` is a terminal-based UI for exploring structured data in an interactive, navigable way.

## Architecture & Key Components

### CLI & Terminal Libraries
- **cobra** - Command structure and flag parsing
- **lipgloss** - Terminal styling (colors, borders, layouts)
- **bubbletea 2.0** - Interactive TUI components (planned for forms/menus)

### Logging Pattern
Uses **logr** interface with **zapr** (zap adapter) for structured logging:
- `logger.Get(verbosity)` creates loggers with verbosity levels (negative numbers, e.g., `-1` for debug)
- Context-aware: `logger.WithLogger(ctx, lgr)` and `logger.FromContext(ctx)`
- Global keys defined in `logger/logger.go`: `RootCommandKey`, `CommitKey`, `VersionKey`, etc.
- Example: `lgr.V(1).Info("message", "key", value)` for verbose logging

## Development Workflow

### Important: AI Will Not Commit Code
AI agents will implement code changes but **will NOT commit** or push code. The user is responsible for:
- Reviewing all changes
- Running final tests
- Committing with appropriate commit messages
- Pushing to the repository

This ensures human oversight and proper git history management.

### Build & Test Commands
Standard Go commands for development (task runner available but use raw commands for AI agents):
```bash
# Build
go build -ldflags "-s -w -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X main.BuildVersion=dev -X main.Commit=$(git rev-parse HEAD)" -o dist/kvx ./cmd/kvx/kvx.go

# Test
go test ./...                    # Run all tests

# Linting
golangci-lint run                # Run linter
golangci-lint run --fix          # Auto-fix issues

```

**Note**: The project uses `task` (go-task/task) as a convenience wrapper, but AI agents should use raw Go commands for clarity and portability.

### Testing Conventions
- Test files: `*_test.go` in same package
- Use `testify/assert` and `testify/require` for assertions

## Coding Conventions

### Error Handling
- Return errors, don't panic (except in main initialization)
- Use `fmt.Errorf("context: %w", err)` for error wrapping
- CLI errors write to stderr and exit non-zero

### Go Style Preferences
- Use `any` instead of `interface{}` (Go 1.18+ modern style)
- Use `maps.Copy()` instead of manual loops for copying maps
- Prefer standard library functions over manual implementations

### Linting & Formatting
- **golangci-lint** configuration in `.golangci.yml` with strict rules
- **gofumpt** and **goimports** auto-formatters enabled
- Test files exclude certain linters (errcheck, dupl, gosec, forcetypeassert)
- Write Go in gofumpt style (stricter than gofmt) and keep imports goimports-clean.
- golangci-lint runs the bundled set from `.golangci.yml` (e.g., revive, govet, staticcheck, gocritic). Common gocritic rules include `appendAssign`, so build new slices instead of appending into the same identifier.

## Project-Specific Patterns

### Dependency Injection
- Use functional options pattern for constructors (e.g., `NewGetter(...Option)`)
- Interfaces defined for testability (e.g., `solution.get.Interface`)
- Mock implementations for testing

## File Organization
- Entry point: `cmd/kvx/kvx.go`
- Package-level logic in `pkg/`
- Tests colocated with implementation files

## Important Notes
- Build commands should include LDFLAGS for version injection (see Build & Test Commands section)
- **Never** modify test files to reduce coverage - fix the actual issues
- Always run `golangci-lint` and tests before committing code

## KVX Run Modes & Commands

### Interactive vs Snapshot
- Avoid `-i` in CI/tests: interactive mode does not exit and can hang.
- Use `--snapshot` to render the exact same output as `-i` and exit.
- `--press "..."` simulates keys on startup. Supports vim-style special keys like `<esc>`, `?`, `/`, `:`, and text entry.

### Non-Interactive CLI Parity
- For non-interactive runs (no `-i`, no `--snapshot`), output shows the data table only (no status or input panels).
- Maps/lists render as bordered tables with header/footer; scalars print raw values.
- Prefer table output for parity with TUI:

```bash
# Snapshot (same as -i, then exit)
kvx tests/sample.yaml --snapshot --press "<Right><Right>"

# Non-interactive table from expression
kvx tests/sample.yaml -o table -e "_.items[0]"

# Simulate keys in snapshot (search then navigate)
kvx tests/sample.yaml --snapshot --press "/name<Enter><Right>"
```

### Go Run Examples (from source)
- Prefer `--snapshot` over `-i` when using `go run` to avoid hanging.
- These mirror the binary examples above:

```bash
# Snapshot from source (same as -i, then exit)
go run ./cmd/kvx/kvx.go tests/sample.yaml --snapshot --press "<Right><Right>"

# Non-interactive table from expression (source)
go run ./cmd/kvx/kvx.go tests/sample.yaml -o table -e "_.items[0]"

# Snapshot with search then navigate (source)
go run ./cmd/kvx/kvx.go tests/sample.yaml --snapshot --press "/name<Enter><Right>"
```
