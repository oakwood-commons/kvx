---
description: "kvx: Verify development environment is ready -- check build, tools, and dependencies."
mode: "agent"
---

# Startup Check

Verify the kvx development environment is ready.

## Steps

1. Check Go version: `go version`
2. Build kvx: `go build ./...`
3. Run vet: `go vet ./...`
4. Check linter is available: `golangci-lint version`
5. Check goimports is available: `which goimports || where goimports`
6. Verify tests pass: `go test ./... -count=1 -short`
7. Verify kvx runs: `go run . --help`

## Report

Summarize the results as a checklist:
- [ ] Go version (minimum 1.22)
- [ ] Build succeeds
- [ ] Vet passes
- [ ] golangci-lint available
- [ ] goimports available
- [ ] Tests pass
- [ ] kvx runs

Flag any failures and suggest fixes.
