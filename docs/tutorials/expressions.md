---
title: "CEL Expressions"
weight: 3
---

# CEL Expressions

kvx uses the [Common Expression Language (CEL)](https://cel.dev/) for dynamic querying, filtering, and transformation. This tutorial covers the expression syntax and common patterns.

## Basics

The root variable is always `_`. Use `-e` on the CLI or `:` in the TUI to evaluate expressions.

```bash
# Access a field
kvx data.yaml -e '_.metadata.name'

# Array index
kvx data.yaml -e '_.items[0]'

# Nested field access
kvx data.yaml -e '_.items[0].name'
```

## Path Syntax

kvx supports two path styles:

### CLI (strict CEL)

Use `_` as the root variable:

```bash
kvx data.yaml -e '_.metadata.created'
kvx data.yaml -e '_.items[0]'
kvx data.yaml -e 'type(_)'
```

### TUI (dotted-path shorthand)

In the TUI expression mode, you can also use dotted-path shorthand:

```
metadata.created
items[0]
items.0
items[0].name
```

The TUI automatically resolves dotted paths. For CEL functions, use the full `_`-rooted syntax.

## Common Patterns

### Field access

```
_.name                            # top-level field
_.metadata.labels                 # nested field
_.items[0].name                   # array element field
```

### Array operations

```
_.items[0]                        # first element
_.items[2]                        # third element
_.items.size()                    # array length
```

### Filtering

```
# Keep items where status is "running"
_.items.filter(x, x.status == "running")

# Keep items with replicas > 0
_.items.filter(x, x.replicas > 0)

# Combine conditions
_.items.filter(x, x.status == "running" && x.replicas >= 2)
```

### Mapping / transformation

```
# Extract a single field from each item
_.items.map(x, x.name)

# Build a summary string
_.items.map(x, x.name + ": " + string(x.replicas))
```

### Existence and type checking

```
# Check if a field exists
has(_.metadata.labels)

# Inspect the type
type(_)
type(_.items)
type(_.items[0].replicas)
```

### String functions

```
_.name.startsWith("api")
_.name.endsWith("-svc")
_.name.contains("web")
_.description.matches("v[0-9]+")
```

### Conditional expressions

```
_.replicas > 5 ? "high" : "low"
has(_.labels) ? _.labels : {}
```

### Size and counting

```
_.items.size()                    # number of items
_.name.size()                     # string length
_.items.filter(x, x.status == "running").size()  # count matching
```

## Shell Quoting

When using expressions with special characters on the command line, prefer single quotes around the entire expression:

```bash
# Single quotes (recommended)
kvx data.yaml -e '_.items[0]'
kvx data.yaml -e 'type(_)'
kvx data.yaml -e '_.metadata["bad-key"]'

# Double quotes require escaping
kvx data.yaml -e "_.metadata[\"bad-key\"]"
```

### Tips

- Use single quotes to avoid shell expansion of `$`, `!`, `*`, etc.
- Bracket notation (`["key"]`) handles keys with hyphens, dots, or spaces.
- In PowerShell, use double quotes with backtick escaping or single quotes.

## Combining with Output Formats

Expressions work with any output format:

```bash
# Filter and output as JSON
kvx data.yaml -e '_.items.filter(x, x.status == "running")' -o json

# Map names and output as YAML
kvx data.yaml -e '_.items.map(x, x.name)' -o yaml

# First item as a list
kvx data.yaml -e '_.items[0]' -o list

# Tree view of metadata
kvx data.yaml -e '_.metadata' -o tree
```

## Expressions in the TUI

In interactive mode, press `:` to enter expression mode:

1. The input is prefilled with the current `_`-rooted path
2. **Tab/Shift+Tab** cycle through keys and indices as completions
3. **Up/Down** cycle through CEL functions valid for the current node type
4. **Right** accepts the ghosted completion
5. **Enter** evaluates and stays in expression mode
6. **Esc** exits; non-navigable results fall back to the prior path

## Next Steps

- [Configuration & Themes](../configuration/) — Customize kvx to your workflow
- [Library Usage](../../library-usage/) — Use CEL expressions programmatically with `core.Engine`
