---
description: "kvx: Run Go code review on recent changes. Checks for idiomatic Go, security, error handling, concurrency, and kvx conventions."
agent: "go-reviewer"
---
Review the current Go code changes for:
- Security vulnerabilities (command injection, path traversal, hardcoded secrets)
- Error handling (ignored errors, missing wrapping, panics for recoverable errors)
- Concurrency issues (goroutine leaks, race conditions, deadlocks)
- Code quality (function length, nesting depth, idiomatic patterns)
- kvx conventions (bubbletea patterns, lipgloss styling, core engine usage, functional options)

Run `go vet ./...` and `golangci-lint run` first, then review the code.
