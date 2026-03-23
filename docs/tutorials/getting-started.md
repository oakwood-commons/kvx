---
title: "Getting Started"
weight: 1
---

# Getting Started with kvx

This tutorial walks you through installing kvx, loading your first data file, and exploring the output formats.

## Installation

### Homebrew (macOS / Linux)

```bash
brew install oakwood-commons/tap/kvx
```

### From Source

```bash
go install github.com/oakwood-commons/kvx@latest
```

### From Release Binaries

Download the latest binary for your platform from [GitHub Releases](https://github.com/oakwood-commons/kvx/releases), extract it, and add it to your `PATH`.

**macOS note:** You may need to remove the quarantine attribute:

```bash
xattr -dr 'com.apple.quarantine' /usr/local/bin/kvx
```

## First Run

Create a sample file `data.yaml`:

```yaml
metadata:
  name: my-app
  version: "2.1.0"
items:
  - name: api
    status: running
    replicas: 3
  - name: web
    status: running
    replicas: 2
  - name: worker
    status: stopped
    replicas: 0
```

Render it as a table:

```bash
kvx data.yaml
```

This produces a bordered table showing the top-level keys and values. Maps display as KEY/VALUE rows, and arrays of objects render as columnar tables with field-name headers.

## Output Formats

kvx supports multiple output formats via the `-o` flag:

### Table (default)

```bash
kvx data.yaml
kvx data.yaml -e '_.items' -o table
```

Tables auto-detect arrays of objects and render them as multi-column tables. Scalars print as raw values.

### JSON

```bash
kvx data.yaml -o json
kvx data.yaml -e '_.metadata' -o json
```

### YAML

```bash
kvx data.yaml -o yaml
```

### List

```bash
kvx data.yaml -e '_.items' -o list
```

List format displays each property on its own line. Arrays show each element with an index header (`[0]`, `[1]`, etc.) and indented properties.

### Tree

```bash
kvx data.yaml -o tree
```

Tree output renders data as an ASCII tree using box-drawing characters. Control depth with `--tree-depth N` and hide values with `--tree-no-values`.

### Mermaid

```bash
kvx data.yaml -o mermaid
```

Generates Mermaid flowchart syntax for visualization in Markdown. Use `--mermaid-direction TD|LR|BT|RL` to set flow direction.

### CSV

```bash
kvx data.yaml -e '_.items' -o csv
```

Arrays of objects become rows with merged headers.

## Basic Expressions

Use `-e` (or `--expression`) to evaluate [CEL](https://cel.dev/) expressions against your data. The root variable is `_`:

```bash
# Access a field
kvx data.yaml -e '_.metadata.name'

# Array index
kvx data.yaml -e '_.items[0]'

# Nested field
kvx data.yaml -e '_.items[0].name'

# Output as JSON
kvx data.yaml -e '_.metadata' -o json
```

See the [CEL Expressions tutorial](../expressions/) for advanced patterns like filtering, mapping, and type introspection.

## Interactive Mode

Launch the interactive TUI with `-i`:

```bash
kvx data.yaml -i
```

This opens a full-screen terminal interface where you can navigate, search, filter, and evaluate expressions. See the [Interactive Mode tutorial](../interactive-mode/) for a complete guide.

For a one-shot render of the TUI layout (useful for scripting or CI):

```bash
kvx data.yaml --snapshot
```

## Input Formats

kvx auto-detects the input format:

| Format | Detection |
|--------|-----------|
| JSON | Content-based |
| YAML | Content-based (single or multi-document) |
| NDJSON | Content-based (newline-delimited JSON) |
| TOML | By `.toml` extension or content |
| CSV | By `.csv` extension or stdin shape |
| JWT | Content-based (base64 dot-separated) |

If no input is provided, kvx shows help. With `--expression` but no input, it evaluates against an empty object `{}`.

## Limiting Records

Control how many records are displayed:

```bash
# First 5 records
kvx data.yaml -e '_.items' --limit 5

# Skip first 2, then show next 5
kvx data.yaml -e '_.items' --offset 2 --limit 5

# Last 3 records
kvx data.yaml -e '_.items' --tail 3
```

`--tail` ignores `--offset` and cannot be combined with `--limit`.

## Next Steps

- [Interactive Mode](../interactive-mode/) — Learn to navigate, search, and evaluate in the TUI
- [CEL Expressions](../expressions/) — Advanced querying patterns
- [Configuration & Themes](../configuration/) — Customize kvx
