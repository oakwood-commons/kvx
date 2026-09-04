# Display Schema Example

This example demonstrates how to use **JSON Schema vendor extensions** (`x-kvx-*`)
to give the kvx interactive TUI a rich, opinionated layout for your data.

## The Problem

By default, `kvx -i` renders every object as a flat KEY/VALUE table.
For a catalog of scafctl workflow providers, you'd rather see:

- **List view** — scrollable cards showing name, description, category badge,
  version, and display name at a glance.
- **Detail view** — a sectioned layout with an inline header, paragraph
  description, capability tags, and a status table.

## How It Works

1. **providers.json** — an array of provider objects.
2. **provider_schema.json** — a standard JSON Schema with `x-kvx-*` extensions
   that describe the list and detail layouts.

### x-kvx Extensions Used

| Extension | Purpose |
|-----------|---------|
| `x-kvx-icon` | Emoji shown before the collection title |
| `x-kvx-collectionTitle` | Heading above the list view |
| `x-kvx-list` | Card-list configuration: title, subtitle, badges, secondary fields |
| `x-kvx-detail` | Sectioned detail view: inline, paragraph, tags, table layouts |

## Running

```bash
# Interactive TUI — card list + detail
go run ./examples/display_schema

# Non-interactive — prints the extracted schema info and a standard table
go run ./examples/display_schema --snapshot
```

### CLI (from data files)

```bash
# Use --schema to point at the JSON Schema with x-kvx-* extensions
kvx examples/display_schema/providers.json \
    --schema examples/display_schema/provider_schema.json -i
```

## Files

| File | Description |
|------|-------------|
| `main.go` | Entry point — loads data and schema, launches TUI |
| `providers.json` | Sample data: 15 scafctl workflow providers |
| `provider_schema.json` | JSON Schema with `x-kvx-*` display extensions |

## Copying while a schema is active

The schema-driven list and detail views intentionally hide the default
`y`/`Y` copy shortcuts because their keys (title, subtitle, badges,
sections) are pre-formatted for display rather than reflecting the raw
JSON path.

To copy paths or values from schema-rendered data, press `v` (vim),
`Alt+V` (emacs), or `F2` (function-key mode) to flip to the default
`KEY`/`VALUE` table for the current node. All standard bindings -- `y`
copy path, `Y` copy value, `/` search, `:` expression mode -- work as if
no schema were configured. Press the same key again to return to the
schema view; the previously selected item is preserved.

The toggle is only visible in the footer when a display schema is active
and does not apply to `x-kvx-status` screens (they expose their own
schema-defined copy actions).
