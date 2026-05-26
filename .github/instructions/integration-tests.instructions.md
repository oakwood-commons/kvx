---
description: "Integration and acceptance test conventions for kvx. Defines test scope boundaries across CLI, TUI, renderer, loader, and CEL layers. Use when writing integration or end-to-end tests."
applyTo: "tests/**,**/*_integration_test.go"
---

# Integration Test Conventions

## Test Scope Boundaries

Each layer has its own test scope. Do not mix concerns across boundaries.

| Layer | Package | Tests cover | Do NOT test here |
|-------|---------|-------------|-----------------|
| CLI commands | `cmd/` | Flag parsing, cobra wiring, output format selection | Business logic, data loading |
| TUI | `pkg/tui/` | Panel layout, navigation, key handling, interactive rendering | Data parsing, CEL evaluation |
| Core engine | `pkg/core/` | Data pipeline, option wiring, engine lifecycle | CLI flags, terminal rendering |
| Formatter | `internal/formatter/` | Table rendering, CSV output, column hints | Navigation, data loading |
| Navigator | `internal/navigator/` | Cursor movement, breadcrumbs, drill-down/up | Rendering, CLI flags |
| CEL | `internal/cel/` | Expression evaluation, filtering, where clauses | Data loading, output formatting |
| Loader | `pkg/loader/` | File parsing (JSON, YAML, TOML, CSV, NDJSON, JWT) | CEL expressions, rendering |

## Snapshot Tests

Snapshot tests live in `tests/snapshots/`. Use `--snapshot` mode to capture TUI output:

```bash
go run . tests/sample.yaml --snapshot --press "<Right><Right>"
```

- Update snapshots with `scripts/update_snapshots.sh`
- Never hand-edit snapshot files

## CLI Integration Tests

CLI tests verify that flags, arguments, and output formats work end-to-end:

```go
func TestCLI_TableOutput(t *testing.T) {
    // Run kvx as a subprocess or call root command directly
    // Assert on stdout content and exit code
}
```

- Use `tests/*.yaml`, `tests/*.json` etc. as test fixtures
- Test both interactive (`--snapshot`) and non-interactive output
- Test expression evaluation (`-e "_.field"`) end-to-end

## Rules

- Every new CLI command or flag must have a CLI-level test
- Every new TUI feature (key binding, panel, mode) must have a TUI-level test
- Snapshot tests are the source of truth for visual output correctness
- Use `t.Parallel()` for independent test cases
- Use `-short` flag to skip slow integration tests in unit test runs
