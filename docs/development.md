# Development

## Prerequisites

- Go 1.20+
- [task](https://taskfile.dev/) (optional but recommended)

## Building

```bash
task build                    # recommended (builds to dist/kvx)
go build -o dist/kvx .        # alternative (always use -o dist/kvx)
```

**Note:** The binary is built to `dist/kvx`, not the root directory. Always use `-o dist/kvx` when building manually.

## Task Commands

```bash
task build           # Build binary
task run             # Run with sample data
task run-interactive # Run TUI
task test            # Run tests
go test ./...        # Direct test execution
```

## Testing

Tests use fixture-based approach with deterministic Bubble Tea `Update()` calls:

```bash
task test
go test ./internal/ui -v
```

## Project Structure

- `cmd/` - CLI commands (Cobra)
- `internal/navigator/` - Path/CEL navigation
- `internal/cel/` - CEL evaluator
- `internal/ui/` - TUI (Bubble Tea)
- `internal/formatter/` - Output formats
- `tests/`, `examples/data/` - Test fixtures

## Architecture

1. Parse YAML/JSON → `interface{}` (`map[string]any`, `[]any`)
2. CLI: evaluate `--expression` via CEL with `_` context
3. TUI: navigate dotted paths (sugar) or CEL in the path input
4. Output via formatters (non-interactive) or TUI (interactive)

## Logging Pattern

Uses the logr interface with zapr (zap adapter) for structured logging:

- `logger.Get(verbosity)` creates loggers with verbosity levels (negative numbers, e.g., `-1` for debug)
- Context-aware: `logger.WithLogger(ctx, lgr)` and `logger.FromContext(ctx)`
- Global keys defined in `logger/logger.go`: `RootCommandKey`, `CommitKey`, `VersionKey`, etc.
- Example: `lgr.V(1).Info("message", "key", value)` for verbose logging
