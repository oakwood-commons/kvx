---
title: "kvx"
type: docs
---

# kvx

**Explore structured data interactively in your terminal.**

kvx is a terminal-based UI for exploring JSON, YAML, TOML, NDJSON, CSV, and JWT data in an interactive, navigable way. It presents data as key-value trees that you can expand, collapse, and inspect directly in the terminal.

[![Go Report Card](https://goreportcard.com/badge/github.com/oakwood-commons/kvx)](https://goreportcard.com/report/github.com/oakwood-commons/kvx)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://github.com/oakwood-commons/kvx/blob/main/LICENSE)
[![Release](https://img.shields.io/github/v/release/oakwood-commons/kvx)](https://github.com/oakwood-commons/kvx/releases)

---

## Quick Install

```bash
brew install oakwood-commons/tap/kvx
```

Or install from source:

```bash
go install github.com/oakwood-commons/kvx@latest
```

Or download a binary from [GitHub Releases](https://github.com/oakwood-commons/kvx/releases).

---

## 30-Second Example

```bash
# Render a YAML file as a table
kvx data.yaml

# Explore interactively
kvx data.yaml -i

# Evaluate a CEL expression
kvx data.yaml -e '_.items[0].name'

# Output as JSON
kvx data.yaml -e '_.metadata' -o json
```

---

## Key Features

- **Multi-format** — Auto-detects JSON, YAML, TOML, NDJSON, CSV, and JWT
- **Interactive TUI** — Navigate, search, filter, and evaluate expressions in the terminal
- **CEL Expressions** — Use [Common Expression Language](https://cel.dev/) for dynamic querying and filtering
- **Multiple Output Formats** — Table, list, tree, Mermaid diagrams, CSV, YAML, JSON
- **Themes** — Built-in themes (midnight, dark, warm, cool) with full customization
- **Schema Hints** — JSON Schema-driven column display (headers, widths, alignment)
- **Embeddable** — Use kvx as a Go library in your own applications
- **Shell Completion** — Bash, Zsh, Fish, and PowerShell support

---

## Documentation

### Tutorials
- [Getting Started](docs/tutorials/getting-started/) — Install and run your first exploration
- [Interactive Mode](docs/tutorials/interactive-mode/) — Navigate, search, and evaluate in the TUI
- [CEL Expressions](docs/tutorials/expressions/) — Dynamic querying with CEL
- [Configuration & Themes](docs/tutorials/configuration/) — Customize kvx to your workflow

### Reference
- [TUI Quick Guide](docs/tui/) — Panels, keybindings, and keyboard reference
- [Library Usage](docs/library-usage/) — Use kvx as a Go library (pkg/core + pkg/tui)
- [Embedding the TUI](docs/embedding/) — Embed the interactive viewer in your app
- [Development](docs/development/) — Build, test, and contribute

---

## Links

- [GitHub Repository](https://github.com/oakwood-commons/kvx)
- [Releases](https://github.com/oakwood-commons/kvx/releases)
- [Discussions](https://github.com/oakwood-commons/kvx/discussions)
- [Contributing](https://github.com/oakwood-commons/kvx/blob/main/CONTRIBUTING.md)
