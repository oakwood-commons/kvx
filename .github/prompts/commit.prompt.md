---
description: "kvx: Generate a conventional commit message from staged or recent changes. Outputs the message only -- does not run git commit."
agent: "commit-message"
---
Analyze the current changes and generate a conventional commit message. **Output the message only -- do not commit.**

1. Check `git diff --cached --stat` for staged changes (fall back to `git diff --stat`)
2. Read the actual diff to understand what changed
3. Run `gh issue list --state open --limit 50 --json number,title` to find open issues
4. Match issues to changes -- include `Closes #NNN` for resolved issues
5. If asked to amend, check `git log -1` for the last commit
6. Generate a message: `<type>(<scope>): <description>` + body with bullet points + issue references
7. Output in a code block for the user to copy

Always include a body unless the change is a single trivial file edit.
Types: feat, fix, docs, perf, refactor, style, test, chore, ci, revert
Add `!` and `BREAKING CHANGE:` footer for breaking changes.