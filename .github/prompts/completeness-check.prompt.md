---
description: "kvx: Check if staged changes have corresponding docs, tutorials, examples, and tests."
agent: "agent"
argument-hint: "Optional: specific area to check"
---
Review staged changes and check if supporting artifacts exist:

1. Run `git diff --cached --stat` to identify staged changes
2. If nothing is staged, fall back to `git log origin/main..HEAD --stat` to check pushed commits on the branch
3. For each feature or command change, verify:
   - Docs in `docs/`
   - Tutorials in `docs/tutorials/` for user-facing features
   - Examples in `examples/`
   - Unit tests colocated with implementation (`*_test.go`)
   - Snapshot tests for TUI behavior changes (`--snapshot --press`)
4. Report present vs missing as a checklist
5. Do not create anything, just report the gaps
