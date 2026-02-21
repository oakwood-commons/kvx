# Display Schema Example

This example demonstrates how to use **JSON Schema vendor extensions** (`x-kvx-*`)
to give the kvx interactive TUI a rich, opinionated layout for your data.

## The Problem

By default, `kvx -i` renders every object as a flat KEY/VALUE table.
For a catalog of infrastructure providers (the "scafctl" use case), you'd
rather see:

- **List view** — scrollable cards showing name, description, status badge,
  version, and maintainer at a glance.
- **Detail view** — a sectioned layout with an inline header, paragraph
  description, colored tag pills, and a details table.

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
| `providers.json` | Sample data: 6 infrastructure providers |
| `provider_schema.json` | JSON Schema with `x-kvx-*` display extensions |
