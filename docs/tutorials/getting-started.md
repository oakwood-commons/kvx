---
title: "Getting Started"
weight: 1
---

# Getting Started with kvx

This tutorial walks through installing `kvx`, inspecting structured data from files or `stdin`, filtering lists with `--where`, evaluating CEL expressions, and choosing the right output format.

## Installation

{{< install-tabs "getting-started" "source" >}}

Verify your installation:

```bash
kvx version
```

## First Run

Create a sample file named `data.yaml`:

```yaml
metadata:
  name: platform-stack
  version: "2.4.1"
items:
  - name: api-gateway
    status: running
    region: us-east-1
    replicas: 4
  - name: billing-worker
    status: degraded
    region: eu-west-1
    replicas: 1
  - name: catalog-sync
    status: running
    region: eu-west-1
    replicas: 2
```

Render it directly:

```bash
kvx data.yaml
```

Maps render as `KEY`/`VALUE` rows, and homogeneous arrays of objects render as multi-column tables with field-name headers:

```bash
kvx data.yaml -e '_.items'
```

Control preferred key and column display order with `--column-order` across maps, columnar object tables, and vertical `-o list` output (specified keys appear first in exact order, and remaining keys follow alphabetically without dropping any fields):

```bash
# Reorder columns in a multi-column array table
kvx data.yaml -e '_.items' --column-order name,status,replicas,region

# Reorder keys in a single map or vertical list output
kvx data.yaml -e '_.metadata' --column-order version,name -o list
```

## Output Formats

Select an output mode with `-o` / `--output`. `kvx` supports `auto` (default) plus 9 explicit output formats:

| Format | Example | Behavior |
|--------|---------|----------|
| `auto` *(default)* | `kvx data.yaml -e '_.items'` | Renders a bordered columnar table when readable in the current terminal width; falls back to vertical `list` layout when columns would truncate |
| `table` | `kvx data.yaml -e '_.items' -o table` | Forces bordered `KEY`/`VALUE` or columnar table output |
| `list` | `kvx data.yaml -e '_.items' -o list` | Stacks each record vertically with indented `key: value` lines |
| `tree` | `kvx data.yaml -o tree --tree-depth 3` | Renders an ASCII box-drawing hierarchy (`--tree-depth`, `--tree-no-values`, `--tree-expand-arrays`, `--tree-max-string`) |
| `mermaid` | `kvx data.yaml -o mermaid --mermaid-direction LR` | Emits Mermaid flowchart syntax (`TD`, `LR`, `BT`, `RL`) ready to paste into Markdown |
| `yaml` | `kvx data.yaml -o yaml` | Emits indented YAML with literal block multiline strings |
| `json` | `kvx data.yaml -e '_.metadata' -o json` | Emits indented JSON |
| `toml` | `kvx data.yaml -e '_.metadata' -o toml` | Emits TOML tables and key/value pairs |
| `csv` | `kvx data.yaml -e '_.items' -o csv` | Emits RFC 4180 CSV rows with merged headers |
| `raw` | `kvx data.yaml -e '_.metadata.name' -o raw` | Alias for `-o yaml` (`-o auto` and `-o table` already emit bare unquoted scalars when `-e` resolves to a string, number, or boolean) |

### Tree and Mermaid Visualization

Generate an ASCII tree or a Mermaid diagram directly from any structured file:

```bash
# Structure-only tree (hides leaf scalar values)
kvx data.yaml -o tree --tree-no-values

# Left-to-right Mermaid diagram
kvx data.yaml -o mermaid --mermaid-direction LR
```

## Filtering Lists with `-w` / `--where`

When your root data is an array (such as JSON arrays, NDJSON streams, or CSV files), `-w` / `--where` evaluates a boolean CEL predicate against each item (`_` refers to the current element) before `-e` runs:

```bash
# Filter an array file per item
kvx examples/data/demo_services.json -w '_.status == "running" && _.replicas >= 3'

# Combine --where item filtering with --expression projection
kvx examples/data/demo_services.json \
  -w '_.region == "eu-west-1"' \
  -e '_.map(s, s.name + " (" + s.status + ")")' \
  -o yaml
```

## Querying with `-e` / `--expression`

Use `-e` / `--expression` to evaluate [CEL](https://cel.dev/) expressions against the full document (`_` is the root value):

```bash
# Access a nested field
kvx data.yaml -e '_.metadata.name'

# Filter and project inside an object's child array
kvx data.yaml -e '_.items.filter(x, x.status == "running").map(x, x.name)' -o json
```

If you pass `-e` without a file or piped `stdin`, `kvx` evaluates the expression against an empty map `{}`:

```bash
kvx -e '[1, 2, 3].map(x, x * 10)' -o json
```

## Launching Interactive Mode and Snapshots

Open the full-screen terminal UI with `-i` / `--interactive`:

```bash
kvx data.yaml -i
```

Render a single deterministic TUI frame and exit immediately with `--snapshot` (ideal for CI tests and documentation):

```bash
kvx data.yaml --snapshot --width 90 --height 20
```

## Auto-Detected Input Formats

`kvx` detects 6 input formats from files or piped `stdin`:

| Format | Detection Behavior |
|--------|--------------------|
| JSON | Content-based and `.json` extension |
| YAML | Single or multi-document (`---`) content and `.yaml`/`.yml` extensions |
| NDJSON | Newline-delimited JSON objects/values and `.ndjson`/`.jsonl` extensions |
| TOML | `.toml` extension or content fallback |
| CSV | `.csv` extension or multi-column `stdin` shape |
| JWT | Raw three-part base64url token or `Bearer <token>` header string |

See [Formats, JWTs & Decoding](../formats-and-decoding/) for details on inspecting JWT claims and decoding nested JSON/YAML strings inside leaf values.

## Limiting Records

Slice large arrays or sorted maps after `--where` and `--expression` evaluation:

```bash
# First 5 records
kvx data.yaml -e '_.items' --limit 5

# Skip first 2 records, then show next 5
kvx data.yaml -e '_.items' --offset 2 --limit 5

# Last 3 records (--tail ignores --offset and is mutually exclusive with --limit)
kvx data.yaml -e '_.items' --tail 3
```
