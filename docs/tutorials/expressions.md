---
title: "CEL Expressions"
weight: 3
---

# CEL Expressions & Function Palette

`kvx` embeds the [Common Expression Language (CEL)](https://cel.dev/) for fast, side-effect-free querying, filtering, and data shaping across both the non-interactive CLI (`-e` and `-w`) and the interactive TUI (`:` and `Ctrl+Space`).

## CLI (`-e`) vs Per-Item Filtering (`-w`)

`kvx` exposes two complementary CEL flags on the command line:

| Flag | Root Variable `_` | Purpose |
|------|-------------------|---------|
| `-w`, `--where <predicate>` | Bound to **each element** of a root list | Filters arrays/slices in place where the boolean predicate returns `true` |
| `-e`, `--expression <cel>` | Bound to the **entire document** (or filtered list after `-w`) | Navigates, projects, aggregates, or reshapes data into any value |

When both flags are passed, `--where` runs first and `--expression` receives the filtered list:

```bash
# 1. Filter services with --where (_ = each service object)
kvx examples/data/demo_services.json -w '_.status == "running" && _.replicas >= 3'

# 2. Chain --where into --expression (_ = filtered list of running services)
kvx examples/data/demo_services.json \
  -w '_.status == "running"' \
  -e '_.map(s, {"service": s.name, "region": s.region, "replicas": s.replicas})' \
  -o table
```

> [!TIP]
> On the CLI, `--where` requires a list or array input and exits with status `2` when given a top-level map or scalar. When embedding `pkg/core` in Go, `engine.FilterWhere` and `engine.EvaluateWhere` return a wrapped `core.ErrNotList` sentinel on non-lists so host CLIs can treat non-list commands as a safe no-op.

## Path Syntax: Strict CLI vs TUI Shorthand

- **CLI (`-e` and `-w`)**: Uses strict CEL syntax rooted at `_`:
  ```bash
  kvx data.yaml -e '_.metadata.name'
  kvx data.yaml -e '_.items[0]'
  kvx data.yaml -e '_.metadata["k8s.io/app"]'
  ```
- **TUI (`:` Expression mode)**: Accepts both `_`-rooted CEL and dotted-path shorthand (`metadata.name`, `items[0]`, `items.0`). For global CEL functions such as `type(_)` or `has(_.metadata.labels)`, include `_`.

## Interactive Expression Mode (`:`) & Ghost Completion

Press `:` (`Alt+x` in Emacs, `F6` in Function mode) to open the live CEL Expression bar:

1. **Prefilled context**: Starts at the `_`-rooted path of your currently highlighted node.
2. **Ghost text & Tab completion**: As you type `.` or partial field/method names, `kvx` shows dimmed inline ghost completions derived from the live runtime value and CEL environment.
   - Press `Right` to accept the inline ghost completion.
   - Press `Tab` / `Shift+Tab` to cycle through matching object keys, array indices, or completions.
   - Press `Up` / `Down` to cycle CEL functions valid for the current node's type.
3. **Live evaluation**: Press `Enter` to evaluate the expression immediately in the Data panel while staying in Expression mode. Invalid expressions show inline red diagnostics without losing your place.
4. **Exit behavior**: Press `Esc` to close Expression mode. If the result is a navigable map or list, the Data panel stays on it; scalar or error results restore your prior navigable path.

## Categorized Function Palette (`Ctrl+Space`)

While in Expression mode (`:`), press **`Ctrl+Space`** to open the interactive **Function Palette** overlay:

{{< svg-card src="images/tui/cel-function-palette.svg" alt="kvx CEL Expression Mode and Ctrl+Space Function Palette" >}}Pressing `Ctrl+Space` in Expression mode opens the categorized CEL Function Palette with signatures and runnable examples.{{< /svg-card >}}

Inside the Function Palette:

- **Category tabs**: Press `Tab` / `Shift+Tab` to switch across `conversion`, `string`, `list`, `map`, `math`, `encoding`, `datetime`, `regex`, and `general` categories.
- **Context-aware methods**: If your expression currently ends with a dot (for example `_.`), `Ctrl+Space` opens in **Methods** mode showing methods applicable to receiver values.
- **Type-to-filter**: Start typing any characters (for example `filter` or `base64`) to search function names and descriptions across all categories simultaneously.
- **Signature & live examples**: The bottom detail pane displays the selected function's signature, description, and evaluated examples from `ui.help.cel.function_examples`.
- **Insert at cursor**: Press `Enter` to insert `funcName(` (or `.methodName(`) directly into your expression input and continue typing arguments.

## High-Value CEL Patterns

### Filtering, Sorting, and Deduplicating Lists

```cel
// Filter objects matching multiple criteria
_.filter(s, s.status == "running" && s.replicas >= 2)

// Sort objects by a key or sort scalar lists
_.sortBy(s, s.replicas)
_.map(s, s.region).distinct().sort()

// Quantifiers: exists, all, exists_one
_.exists(s, s.status == "degraded")
_.all(s, s.replicas > 0)
```

### String Inspection and Formatting

```cel
_.name.startsWith("auth-")
_.description.lowerAscii().contains("jwt")
_.name.matches("^[a-z]+-[a-z]+$")
"%s (%d replicas)".format([_.name, _.replicas])
["us-east-1", "eu-west-1"].join(", ")
```

### Math, Timestamps, Durations, and Base64

```cel
math.greatest([1, 4, 3, 2])
timestamp("2026-02-18T10:30:00Z")
duration("1h30m")
string(base64.decode("aGVsbG8="))
```

### Type Introspection and Safe Field Checks

```cel
type(_)
has(_[0].authToken) ? "present" : "missing"
```

## Shell Quoting Tips

Wrap CLI expressions in single quotes so your shell does not expand `$`, `!`, `*`, or double quotes:

```bash
# Bash / Zsh / Fish: single quotes around the expression
kvx examples/data/demo_services.json -e '_[0].capabilities.contains("mtls")'

# Keys containing dots, slashes, or hyphens use bracket notation
kvx pod.json -e '_.metadata.annotations["kubectl.kubernetes.io/last-applied-configuration"]'
```
