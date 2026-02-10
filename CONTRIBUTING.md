# Contributing to kvx

Thank you for your interest in contributing to kvx! This document provides guidelines and instructions for contributing.

## Development Setup

### Prerequisites

- **Go 1.20+** - [Download Go](https://go.dev/dl/)
- **Task** (optional but recommended) - [Install Taskfile](https://taskfile.dev/installation/)

### Building from Source

```bash
# Using task (recommended)
task build

# Or using go directly
go build -ldflags "-s -w -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X main.BuildVersion=dev -X main.Commit=$(git rev-parse HEAD)" -o dist/kvx ./cmd/kvx/kvx.go
```

The binary is built to `dist/kvx`.

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test ./internal/ui -v
```

### Linting

We use `golangci-lint` with strict rules configured in `.golangci.yml`:

```bash
# Run linter
golangci-lint run

# Auto-fix issues where possible
golangci-lint run --fix
```

## Code Style

- **Formatting**: Use `gofumpt` (stricter than `gofmt`) and keep imports `goimports`-clean
- **Modern Go**: Use `any` instead of `interface{}`, use `maps.Copy()` over manual loops
- **Error Handling**: Return errors, don't panic. Use `fmt.Errorf("context: %w", err)` for wrapping
- **Linting**: All code must pass `golangci-lint` before merging

## Submitting Changes

### Pull Request Process

1. **Fork the repository** and create your branch from `main`
2. **Make your changes** following the code style guidelines
3. **Add tests** for any new functionality
4. **Run the test suite** and ensure all tests pass
5. **Run the linter** and fix any issues
6. **Submit a pull request** with a clear description of your changes

### Pull Request Checklist

- [ ] Tests pass (`go test ./...`)
- [ ] Linter passes (`golangci-lint run`)
- [ ] Code follows project style guidelines
- [ ] Documentation updated if needed
- [ ] Commit messages are clear and descriptive

### Commit Messages

Write clear, concise commit messages that explain the "what" and "why":

```
feat: add tree output format with depth control

- Add --tree-depth flag to limit expansion depth
- Add --tree-no-values flag to show structure only
- Include comprehensive tests for edge cases
```

Use conventional commit prefixes when appropriate:
- `feat:` - New features
- `fix:` - Bug fixes
- `docs:` - Documentation changes
- `test:` - Test additions or modifications
- `refactor:` - Code refactoring without functional changes
- `chore:` - Maintenance tasks

## Reporting Issues

### Bug Reports

When filing a bug report, please include:

- **kvx version** (`kvx --version`)
- **Operating system** and version
- **Go version** (if building from source)
- **Steps to reproduce** the issue
- **Expected behavior** vs **actual behavior**
- **Sample input data** (if applicable, anonymized)

### Feature Requests

Feature requests are welcome! Please describe:

- The use case or problem you're trying to solve
- Your proposed solution (if any)
- Any alternatives you've considered

## Questions?

If you have questions about contributing, feel free to open a discussion or issue on GitHub.
