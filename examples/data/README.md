# Example Data Files

Sample data files for testing kvx features.

## Schema Example: users.json + users_schema.json

Demonstrates the `--schema` flag for customizing table column display.

### Without schema (default)

```bash
kvx examples/data/users.json -o table
```

Output shows all columns with original field names:
```
╭─────────────────────────── kvx ────────────────────────────╮
│#    department   full_name    internal_code  score  user_id│
│────────────────────────────────────────────────────────────│
│1    Engineering  Alice Smith  X123           95     u001   │
│2    Sales        Bob Jones    Y456           87     u002   │
│3    Engineering  Carol White  Z789           92     u003   │
│4    Marketing    David Lee    W012           78     u004   │
│5    Engineering  Eva Brown    V345           99     u005   │
╰ _ ────────────────────────────────────────────── list: 1/5 ╯
```

### With schema

```bash
kvx examples/data/users.json --schema examples/data/users_schema.json -o table
```

Output with schema-driven display:
```
╭─────────────────────────── kvx ────────────────────────────╮
│#    Dept         Name         Score  ID                    │
│──────────────────────────────────────────                  │
│1    Engineering  Alice Smith     95  u001                  │
│2    Sales        Bob Jones       87  u002                  │
│3    Engineering  Carol White     92  u003                  │
│4    Marketing    David Lee       78  u004                  │
│5    Engineering  Eva Brown       99  u005                  │
╰ _ ────────────────────────────────────────────── list: 1/5 ╯
```

### What the schema does

The [users_schema.json](users_schema.json) file applies these hints:

| Field | Schema Setting | Effect |
|-------|----------------|--------|
| `user_id` | `"title": "ID"` | Column header renamed |
| `full_name` | `"title": "Name"` | Column header renamed |
| `department` | `"title": "Dept"` | Column header renamed |
| `score` | `"type": "integer"` | Right-aligned |
| `internal_code` | `"deprecated": true` | Column hidden |

### Interactive mode

```bash
kvx examples/data/users.json --schema examples/data/users_schema.json -i
```

### With column ordering

Combine `--schema` with config file column ordering:

```yaml
# ~/.config/kvx/config.yaml
formatting:
  table:
    column_order: [full_name, user_id, score, department]
```

## Other data files

- `sample_cluster.json` - Large nested cluster configuration
- `sample_employees.csv` - CSV format example  
- `sample_nested.json` - Nested JSON with encoded strings for auto-decode testing
- `sample.jwt` - JWT token for decoding
