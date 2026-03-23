---
title: "Configuration & Themes"
weight: 4
---

# Configuration & Themes

This tutorial covers configuring kvx, selecting themes, using schema-driven column hints, and setting up shell completion.

## Config File

kvx merges built-in defaults with your config file at `~/.config/kvx/config.yaml` (or `$XDG_CONFIG_HOME/kvx/config.yaml`). Override the config path with `--config-file`.

### Viewing Configuration

```bash
# Show merged config (defaults + your overrides)
kvx config get

# Output as JSON, YAML, or table
kvx config get -o json
kvx config get -o yaml
kvx config get -o table

# Interactive TUI view
kvx config get -i

# Print merged config without reading input
kvx --config
```

### Using a Specific Config File

```bash
kvx config get --config-file ~/.config/kvx/config.yaml
kvx data.yaml --config-file /path/to/custom-config.yaml
```

## Themes

Themes control the colors, borders, and styling of both TUI and CLI output.

### Built-in Themes

kvx includes four built-in themes:

| Theme | Description |
|-------|-------------|
| `midnight` | Default theme, dark background |
| `dark` | Alternative dark theme |
| `warm` | Warm color palette |
| `cool` | Cool/blue color palette |

### Selecting a Theme

```bash
# Via CLI flag
kvx data.yaml -i --theme warm

# Via config file (ui.theme.default)
# In ~/.config/kvx/config.yaml:
#   ui:
#     theme:
#       default: warm
```

### Listing Available Themes

```bash
kvx config theme
kvx config theme --config-file ~/.config/kvx/config.yaml
```

### Disabling Colors

```bash
kvx data.yaml --no-color
```

This removes colors and box-drawing borders for plain terminals and piping.

## Schema-Driven Column Hints

Use `--schema` to provide a JSON Schema file that controls how table columns are displayed.

### How It Works

kvx reads standard JSON Schema properties and derives display hints:

| JSON Schema Field | Effect |
|-------------------|--------|
| `title` | Override column header text |
| `maxLength` | Cap column width (characters) |
| `enum` | Cap width to longest enum value |
| `format` | Auto-width for known formats (date=10, uuid=36, email=40, etc.) |
| `type: integer/number` | Right-align the column |
| `deprecated: true` | Hide the column |
| `required` array | Boost priority (+10) for required fields |
| Property order | Priority tiebreaker (first declared = highest) |

### Example Schema

```json
{
  "type": "array",
  "items": {
    "type": "object",
    "required": ["id", "name"],
    "properties": {
      "id": {
        "title": "ID",
        "type": "string",
        "maxLength": 8
      },
      "name": {
        "title": "Name",
        "type": "string"
      },
      "score": {
        "type": "integer"
      },
      "legacy_field": {
        "deprecated": true
      }
    }
  }
}
```

### Usage

```bash
# CLI with schema
kvx data.json --schema ./schema.json -o table

# Interactive TUI with schema
kvx data.json --schema ./schema.json -i
```

### Config File Schema Options

You can also specify schemas in your config file:

```yaml
formatting:
  table:
    # Path to external schema file
    schema_file: ./my-schema.json

    # Or inline schema
    schema:
      type: object
      required: [id, name]
      properties:
        id:
          title: ID
          maxLength: 8
        name:
          title: Name
```

**Priority**: CLI `--schema` > config `schema_file` > config inline `schema`.

### Related Column Options

```yaml
formatting:
  table:
    column_order: [name, id, status]     # reorder columns
    hidden_columns: [internal_id, hash]  # hide specific columns
```

## Keymaps

kvx supports three keybinding modes:

```bash
kvx data.yaml -i --keymap vim       # default
kvx data.yaml -i --keymap emacs
kvx data.yaml -i --keymap function
```

Set the default in your config:

```yaml
ui:
  features:
    key_mode: emacs
```

## Shell Completion

Generate shell completion scripts for your shell:

### Bash

```bash
kvx completion bash > /etc/bash_completion.d/kvx
```

### Zsh

```bash
kvx completion zsh > "${fpath[1]}/_kvx"
```

### Fish

```bash
kvx completion fish > ~/.config/fish/completions/kvx.fish
```

### PowerShell

```powershell
kvx completion powershell | Out-String | Invoke-Expression
```

Restart your shell (or `source` the file) for completions to take effect. Zsh users must have `compinit` loaded before the completion file is sourced.

## Array Index Style

Control how array elements are labeled in output:

```bash
kvx data.yaml -e '_.items' --array-style none        # no indices (default)
kvx data.yaml -e '_.items' --array-style index       # [0], [1]
kvx data.yaml -e '_.items' --array-style numbered    # 1, 2
kvx data.yaml -e '_.items' --array-style bullet      # •
```

## Next Steps

- [TUI Quick Guide](../../tui/) — Complete keybinding reference and panel details
- [Library Usage](../../library-usage/) — Use schemas and themes programmatically
