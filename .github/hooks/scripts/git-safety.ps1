# PreToolUse hook: block git commit/push/amend unless user explicitly approves.
# Reads JSON from stdin, checks if the command is a git write operation.

$ErrorActionPreference = 'Stop'

$hookInput = [System.Console]::In.ReadToEnd()

# Extract the command being run
if ($hookInput -match '"command"\s*:\s*"([^"]*)"') {
    $cmd = $Matches[1]
} else {
    exit 0
}

# Check for git write operations
if ($cmd -match 'git\s+(commit|push|amend|reset\s+--hard|rebase|force-push)') {
    @'
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "ask",
    "permissionDecisionReason": "Git write operation detected. This project requires explicit user approval before committing, pushing, or rewriting history."
  }
}
'@
    exit 0
}
