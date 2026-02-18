# Schema Hints Example

Demonstrates how to customize table column display using JSON Schema or programmatic hints.

## Features shown

- Rename column headers (`title` → `DisplayName`)
- Hide columns (`deprecated: true` → `Hidden`)
- Cap column width (`maxLength` → `MaxWidth`)
- Right-align numeric columns (`type: integer` → `Align: "right"`)
- Explicit column ordering (`ColumnOrder`)

## Run

```bash
go run ./examples/schema_hints
```

## Output

```
=== Table with JSON Schema hints ===
(internal_code hidden via 'deprecated: true', headers renamed via 'title')

╭─────────────── schema-example ───────────────╮
│#    Name          ID      score              │
│──────────────────────────────                │
│1    Alice Smith   u001       95              │
│2    Bob Jones     u002       87              │
│3    Carol White   u003       92              │
╰ _ ───────────────────────────────── list: 1/3╯
```

## Ways to provide hints

### 1. JSON Schema file (CLI)

```bash
kvx data.json --schema schema.json
```

### 2. JSON Schema in code

```go
hints, _ := tui.ParseSchema(schemaJSON)
tui.RenderTable(root, tui.TableOptions{ColumnHints: hints})
```

### 3. Programmatic hints

```go
hints := map[string]tui.ColumnHint{
    "user_id": {DisplayName: "ID", MaxWidth: 10},
    "score":   {Align: "right"},
    "legacy":  {Hidden: true},
}
```

## JSON Schema → ColumnHint mapping

| JSON Schema Field | ColumnHint Field | Effect |
|-------------------|------------------|--------|
| `title` | `DisplayName` | Column header text |
| `maxLength` | `MaxWidth` | Cap column width |
| `enum` | `MaxWidth` | Longest enum value |
| `format` (date, uuid, etc.) | `MaxWidth` | Auto-calculated width |
| `type: integer/number` | `Align: "right"` | Right-align column |
| `deprecated: true` | `Hidden: true` | Hide column |
| `required` array | `Priority` | Boost shrink priority |
