# Contributing to kvx

Thank you for your interest in contributing to `kvx`! This document provides guidelines and instructions for building, testing, and submitting changes.

## Development Setup

### Prerequisites

- **Go 1.27+**: [Download Go](https://go.dev/dl/)
- **Task** (optional but recommended): [Install Task](https://taskfile.dev/installation/)

### Building from Source

```bash
# Using task (recommended)
task build

# Or using go directly
mkdir -p dist && go build -o dist/kvx .
```

The binary is built to `dist/kvx`.

### Running Tests

```bash
# Run all unit tests
go test ./...

# Run tests with the race detector (Linux/macOS; requires CGO)
CGO_ENABLED=1 go test -race ./...

# Run tests for a specific package
go test ./internal/ui -v
```

### Linting

`kvx` uses `golangci-lint` with strict rules configured in `.golangci.yaml`:

```bash
# Run linter
golangci-lint run ./...

# Auto-fix formatting and import issues where possible
golangci-lint run --fix ./...
```

## Code Style

- **Formatting**: Use `gofumpt` (stricter than `gofmt`) and keep imports `goimports`-clean.
- **Modern Go**: Use `any` instead of `interface{}`, and prefer standard library helpers (`maps`, `slices`).
- **Error Handling**: Return errors rather than panicking outside initialization. Wrap errors with context via `fmt.Errorf("context: %w", err)`.
- **Linting**: All changes must pass `golangci-lint run ./...` before merging.

## Submitting Changes

### Pull Request Process

1. **Fork the repository** and create your branch from `main`.
2. **Make your changes** following the code style guidelines.
3. **Add tests** covering new behavior and edge cases.
4. **Run the test suite** (`CGO_ENABLED=1 go test -race ./...`) and ensure all tests pass.
5. **Run the linter** (`golangci-lint run ./...`) and resolve any findings.
6. **Submit a pull request** with a Conventional Commit title and clear summary.

### Pull Request Checklist

- [ ] Tests pass (`CGO_ENABLED=1 go test -race ./...`)
- [ ] Linter passes (`golangci-lint run ./...`)
- [ ] Code follows project style guidelines
- [ ] Documentation (and `task docs:build` if docs changed) updated as needed
- [ ] Commits are signed (`git commit -s -S`) and use Conventional Commits

### Commit Messages

Use Conventional Commit prefixes (`feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `chore:`, `perf:`, `ci:`):

```text
feat(tui): add tree output format with depth control

- Add --tree-depth flag to limit expansion depth
- Add --tree-no-values flag to show structure only
- Include comprehensive tests for edge cases
```

## Restricted Terms Check

This repository runs an automated CI check ("Restricted Terms Check") that scans new content and commit messages for a maintainer-configured list of disallowed terms and patterns. The list itself is never committed to this repository or printed in CI logs.

If your PR fails this check, a maintainer will inspect the flagged file path(s) during review and help sanitize the content. Repeat contributors can also configure the optional local git hooks under `.githooks/` (see `.githooks/README.md`).

## Reporting Issues

### Bug Reports

When filing a bug report, please include:

- **kvx version** (`kvx version`)
- **Operating system** and version
- **Go version** (if building from source)
- **Steps to reproduce**
- **Expected behavior** vs **actual behavior**
- **Sample input data** (anonymized)

### Feature Requests

Please describe the use case or problem you want to solve, your proposed approach, and any alternatives considered.
