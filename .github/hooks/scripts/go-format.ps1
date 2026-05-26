# PostToolUse hook: auto-format .go files after edits
# Reads JSON from stdin, checks if a .go file was edited, runs goimports.

$ErrorActionPreference = 'Stop'

$hookInput = [System.Console]::In.ReadToEnd()

# Extract the file path from the tool input
if ($hookInput -match '"filePath"\s*:\s*"([^"]*)"') {
    $file = $Matches[1]
} else {
    exit 0
}

# Only proceed for .go files
if (-not $file.EndsWith('.go')) {
    exit 0
}

# Only format if the file exists
if (-not (Test-Path $file)) {
    exit 0
}

# Run goimports if available, fall back to gofmt
if (Get-Command goimports -ErrorAction SilentlyContinue) {
    & goimports -w $file
} elseif (Get-Command gofmt -ErrorAction SilentlyContinue) {
    & gofmt -w $file
}
