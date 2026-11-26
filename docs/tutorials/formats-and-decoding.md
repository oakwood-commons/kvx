---
title: "Formats, JWTs & Decoding"
weight: 4
---

# Formats, JWTs & Serialized Decoding

Modern cloud, authentication, and CI/CD payloads rarely stay in a single flat format: JSON responses carry `Bearer` JWT strings, Kubernetes ConfigMaps embed YAML or TOML documents inside string values, and audit logs stream as NDJSON with serialized JSON blobs inside event fields. `kvx` handles both top-level multi-format detection and recursive leaf-string decoding out of the box.

## Six Auto-Detected Input Formats

Pass a file path or pipe bytes to `stdin`; `kvx` automatically parses any of six structured formats:

| Format | File Extensions | Stdin / Heuristic Detection | Root Shape in `kvx` |
|--------|-----------------|-----------------------------|---------------------|
| **JSON** | `.json` | Leading `{` or `[` | `map[string]any` or `[]any` |
| **YAML** | `.yaml`, `.yml` | Single document or `---` multi-document stream | Single doc value, or `[]any` for multi-doc YAML |
| **TOML** | `.toml` | Extension match or structured fallback parser | `map[string]any` |
| **NDJSON** | `.ndjson`, `.jsonl` | Multiple newline-delimited JSON values | `[]any` (one entry per non-empty line) |
| **CSV** | `.csv` | `.csv` extension or multi-column comma-delimited `stdin` when YAML unmarshaling does not match | `[]any` of row maps keyed by header names |
| **JWT** | `.jwt` or any text | Three base64url segments (`header.payload.signature`) where segments 1 and 2 decode to JSON objects, with optional `Bearer ` prefix stripped automatically | `map[string]any` containing `header`, `payload`, and `signature` |

### Examples Across Input Formats

```bash
# Multi-document YAML manifest (renders as a list of documents)
kvx tests/sample-multi.yaml

# Newline-delimited JSON event stream filtered per event
kvx tests/sample.ndjson -w '_.level == "error"'

# CSV file queried with CEL
kvx examples/data/sample_employees.csv -w 'int(_.salary) > 90000'

# TOML configuration converted to JSON
kvx tests/sample.toml -o json

# Convert a root JSON array to valid TOML (automatically wrapped under [[items]])
kvx examples/data/demo_services.json -w '_.status == "running"' -o toml
```

> [!NOTE]
> Standard TOML specifications do not permit bare arrays at the document root. When `-o toml` (or `tui.MarshalTOML` in Go) receives a slice or array, `kvx` automatically wraps the elements under an `items` table array (`[[items]]`) so list queries and `--where` filters serialize cleanly without error.

## First-Class JWT Inspection

You can inspect a raw JWT file or pipe an `Authorization: Bearer ...` header value directly into `kvx` without manual `cut` and `base64 -d` pipelines:

```bash
# Inspect a JWT file directly
kvx examples/data/sample.jwt

# Pipe a Bearer header value from an environment variable or CLI tool
printf '%s' "Bearer $ACCESS_TOKEN" | kvx -e '_.payload' -o yaml
```

`kvx` validates that the token has three non-empty base64url segments whose first two segments decode to JSON objects, strips any leading `Bearer ` prefix, and exposes:

- `header`: decoded JOSE header map (such as `alg`, `typ`, `kid`)
- `payload`: decoded claims map (such as `iss`, `sub`, `aud`, `scope`, `exp`)
- `signature`: raw base64url signature string preserved intact

> [!IMPORTANT]
> `kvx` decodes JWT headers and claims for inspection only; it does not verify cryptographic signatures.

## Manual Scalar Decoding in the TUI (`Enter`)

When browsing data interactively (`kvx <file> -i`), string leaves often contain embedded JSON, YAML, TOML, NDJSON, or `Bearer` JWT strings.

1. Highlight any string row (for example `authToken` or `config` in `examples/data/demo_services.json`).
2. Press **`Enter`** (or **`l`** / **`Right`**) to open the scalar leaf, and press **`Enter`** to decode it in place.
3. When the string parses into a map or slice, `kvx` decodes it on the fly, displays `decoded` in the status bar, and drills into the parsed structure.
4. Keep pressing `Enter` / `l` on nested serialized strings to decode deeper layers, or press `h` / `Left` to ascend back to the parent node.

{{< svg-card src="images/tui/jwt-decoded.svg" alt="kvx decoded Bearer JWT token inside the interactive TUI" >}}Pressing `Enter` on a `Bearer` JWT string leaf decodes `header`, `payload`, and `signature` directly into the navigation tree.{{< /svg-card >}}

Try it locally:

```bash
kvx examples/data/demo_services.json -i
# Select [1], press l, then press Enter on authToken or config
```

## Automatic Decoding (`--auto-decode`)

On the CLI (or when you want CEL expressions to query inside serialized strings without manual keypresses), use `--auto-decode`:

| Mode | Flag | When Decoding Runs | Best For |
|------|------|--------------------|----------|
| **Manual** *(default)* | *(omitted)* or `--auto-decode disabled` | Only when you press `Enter` / `l` on a string leaf in the TUI | Preserving exact raw string types until explicitly inspected |
| **Lazy** | `--auto-decode lazy` | Decodes the final target node when navigated to or returned by `-e` | Inspecting one specific serialized field (`-e '_.config' --auto-decode lazy`) without scanning the entire file |
| **Eager** | `--auto-decode eager` | Recursively walks the entire tree at load time (up to depth 20) before `-w` and `-e` run | Running CEL queries, filters, or exports across fields buried inside serialized JSON/YAML/JWT strings |

### Querying Inside Serialized Strings with `--auto-decode eager`

In `examples/data/sample_nested.json`, `lastEvent` is a JSON-encoded string whose inner `payload` contains `items`, and `jwt` is an encoded token string. With `--auto-decode eager`, CEL expressions traverse directly through those decoded structures:

```bash
# Lazy: decode just the expression result into a table
kvx examples/data/sample_nested.json -e '_.config' --auto-decode lazy

# Eager: query inside a JSON-encoded string and a JWT in a single CEL expression
kvx examples/data/sample_nested.json \
  --auto-decode eager \
  -e '{"orderId": _.lastEvent.payload.orderId, "jwtSub": _.jwt.payload.sub}' \
  -o yaml
```

## Programmatic Decoding in Go (`pkg/loader` & `pkg/tui`)

Library consumers can invoke the same decoding helpers directly from `github.com/oakwood-commons/kvx/pkg/loader` or configure TUI behavior on `tui.Config`:

```go
import (
    "github.com/oakwood-commons/kvx/pkg/loader"
    "github.com/oakwood-commons/kvx/pkg/tui"
)

// Decode a single string leaf if it contains structured data (JWT, JSON, YAML, TOML, NDJSON)
if decoded, ok := loader.TryDecode(rawString); ok {
    _ = decoded
}

// Recursively expand all serialized string leaves across an arbitrary tree (depth cap: 20)
expanded := loader.RecursiveDecode(rootData)

// Direct JWT detection and decoding (strips optional "Bearer " prefix)
if loader.IsJWT(tokenHeader) {
    parts, err := loader.DecodeJWT(tokenHeader)
    _, _ = parts, err
}

// Configure TUI decoding behavior
cfg := tui.DefaultConfig()
cfg.AutoDecode = "lazy" // "", "lazy", or "eager"
```
