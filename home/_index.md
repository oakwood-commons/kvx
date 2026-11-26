---
title: "kvx"
description: "Interactive terminal UI, schema-driven explorer, and CEL query engine for structured data."
type: docs
---

<div class="bp-hero">
  <div class="bp-hero__eyebrow"><span class="bp-hero__dot" aria-hidden="true"></span><span>TERMINAL UI // STRUCTURED DATA EXPLORER</span></div>
  <img class="bp-hero__logo" src="images/kvx-logo.svg" alt="kvx" width="240" height="78">
  <h1 class="bp-hero__tagline">Explore, query, decode, and render structured data in your terminal.</h1>
  <p class="bp-hero__desc"><code>kvx</code> turns JSON, YAML, TOML, NDJSON, CSV, and JWT payloads into navigable trees, schema-driven card lists, sectioned detail views, and clean CLI output. Filter and transform data live with <a href="https://cel.dev/">CEL</a> and the <code>Ctrl+Space</code> Function Palette, decode nested JSON or JWT strings with <code>Enter</code>, or embed <code>pkg/tui</code> and <code>pkg/core</code> directly in your own Go CLI.</p>
  <div class="bp-hero__actions">
    <a class="bp-hero__btn bp-hero__btn--primary" href="docs/tutorials/getting-started/">Get Started</a>
    <a class="bp-hero__btn bp-hero__btn--secondary" href="docs/tutorials/interactive-mode/">Explore the TUI</a>
    <a class="bp-hero__btn bp-hero__btn--secondary" href="docs/library-usage/">Embed in Go</a>
  </div>
  <div class="bp-hero__badges">
    <a href="https://pkg.go.dev/github.com/oakwood-commons/kvx"><img src="https://pkg.go.dev/badge/github.com/oakwood-commons/kvx.svg" alt="Go Reference"></a>
    <a href="https://goreportcard.com/report/github.com/oakwood-commons/kvx"><img src="https://goreportcard.com/badge/github.com/oakwood-commons/kvx" alt="Go Report Card"></a>
    <a href="https://github.com/oakwood-commons/kvx/actions/workflows/ci.yml"><img src="https://github.com/oakwood-commons/kvx/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
    <a href="https://github.com/oakwood-commons/kvx/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-Apache_2.0-blue.svg" alt="License"></a>
    <a href="https://github.com/oakwood-commons/kvx/releases"><img src="https://img.shields.io/github/v/release/oakwood-commons/kvx" alt="Release"></a>
  </div>
  {{< asciinema cast="casts/demo.cast" title="kvx examples/data/demo_services.json --schema examples/data/demo_services_schema.json -i" cols=100 rows=26 >}}
</div>

## 30-Second Walkthrough

{{% steps %}}
1. **Install `kvx`**

   Install on macOS, Linux, or Windows (`x86_64` and `arm64` binaries are also published on [GitHub Releases](https://github.com/oakwood-commons/kvx/releases/latest)):

   {{< install-tabs "walkthrough" "source" >}}

2. **Inspect Any Structured File or Pipe**

   Pass a JSON, YAML, TOML, NDJSON, CSV, or JWT file, or pipe command output directly to `kvx`:

   ```bash
   # Auto-sized columnar table or readable list fallback
   kvx services.yaml

   # Launch the interactive TUI with schema-driven card and detail views
   kvx services.json --schema services.schema.json -i
   ```

3. **Filter with `--where`, Query with CEL, and Decode Nested Payloads**

   Combine per-item predicates (`-w`), full CEL expressions (`-e`), recursive scalar decoding (`--auto-decode`), and 10 output formats (`-o`):

   ```bash
   # Keep running services, decode embedded JSON/JWT strings, and emit JSON
   kvx services.json --auto-decode eager -w '_.status == "running"' -e '_.map(s, s.name)' -o json
   ```
{{% /steps %}}

---

## Core Capabilities

<div class="bp-pillars">
  <div class="bp-card">
    <div class="bp-card-head">
      <svg viewBox="0 0 24 24" fill="none" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="3" width="18" height="18" rx="2"/><path d="M3 9h18"/><path d="M9 21V9"/></svg>
      <h3>Schema-Driven TUI Views</h3>
    </div>
    <p>Attach standard JSON Schema hints and <code>x-kvx-*</code> extensions to render scrollable Card Lists, Sectioned Detail screens, and Async Status views with masked inline input. Press <code>v</code> anytime to toggle the raw table.</p>
  </div>
  <div class="bp-card">
    <div class="bp-card-head">
      <svg viewBox="0 0 24 24" fill="none" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
      <h3>CEL Querying &amp; Function Palette</h3>
    </div>
    <p>Filter lists per item with <code>-w/--where</code>, evaluate expressions with <code>-e/--expression</code> or <code>:</code> in the TUI, accept inline ghost completions, and open the categorized <code>Ctrl+Space</code> Function Palette.</p>
  </div>
  <div class="bp-card">
    <div class="bp-card-head">
      <svg viewBox="0 0 24 24" fill="none" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0110 0v4"/></svg>
      <h3>JWT &amp; Nested String Decoding</h3>
    </div>
    <p>Auto-detect JSON, YAML, TOML, NDJSON, CSV, and <code>Bearer</code> JWT tokens. Press <code>Enter</code> on serialized string leaves in the TUI or pass <code>--auto-decode lazy|eager</code> on the CLI.</p>
  </div>
  <div class="bp-card">
    <div class="bp-card-head">
      <svg viewBox="0 0 24 24" fill="none" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M21 16V8a2 2 0 00-1-1.73l-7-4a2 2 0 00-2 0l-7 4A2 2 0 003 8v8a2 2 0 001 1.73l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/></svg>
      <h3>Embeddable Go Library</h3>
    </div>
    <p>Import <code>pkg/core</code> for concurrency-safe loading, CEL evaluation, and <code>FilterWhere</code>, or <code>pkg/tui</code> for full Bubble Tea v2 interactive viewers, custom CEL providers, and deterministic snapshots.</p>
  </div>
</div>

---

## Output Formats at a Glance

Select an output mode with `-o` / `--output` (defaults to `auto`, which renders a columnar table when it fits your terminal width and falls back to `list` when columns would truncate):

| Mode | Flag | Best For |
|------|------|----------|
| **Auto** | `-o auto` *(default)* | Width-aware CLI tables that automatically switch to vertical lists on narrow terminals |
| **Table** | `-o table` | Bordered `KEY`/`VALUE` maps and multi-column object tables with `--schema` and `--column-order` |
| **List** | `-o list` | Stacked vertical properties per record when objects have many wide fields |
| **Tree** | `-o tree` | Box-drawing hierarchy inspection (`--tree-depth`, `--tree-no-values`, `--tree-expand-arrays`) |
| **Mermaid** | `-o mermaid` | Flowchart syntax for Markdown docs and architecture notes (`--mermaid-direction TD\|LR\|BT\|RL`) |
| **YAML / JSON / TOML** | `-o yaml`, `-o json`, `-o toml` | Format conversion and pipeline reshaping after `-w` or `-e` (`-o toml` automatically wraps root arrays under `[[items]]`) |
| **CSV / Raw** | `-o csv`, `-o raw` | Spreadsheet exports (`-o csv`) and YAML alias output (`-o raw`; scalar `-e` results print unquoted directly in `-o auto` and `-o table`) |

---

## Documentation

### Guided Tutorials
- [Getting Started](docs/tutorials/getting-started/): Install `kvx`, run your first queries, and tour all output modes
- [Interactive Mode](docs/tutorials/interactive-mode/): Navigate trees, search, copy paths or values, and script snapshots
- [CEL Expressions](docs/tutorials/expressions/): Query with `-e`, filter lists with `-w`, and browse the `Ctrl+Space` Function Palette
- [Formats, JWTs & Decoding](docs/tutorials/formats-and-decoding/): Inspect 6 input formats, `Bearer` JWTs, and nested serialized strings
- [Schema-Driven Views](docs/tutorials/schema-views/): Build Card-List (`x-kvx-list`), Detail (`x-kvx-detail`), and Status (`x-kvx-status`) UIs
- [Configuration & Themes](docs/tutorials/configuration/): Preview the 4 built-in themes, configure keymaps, and enable shell completion

### Reference & Go Library
- [TUI Quick Guide](docs/tui/): Panel layout, Vim/Emacs/Function keybindings, and `--press` automation
- [Library Usage](docs/library-usage/): Complete `pkg/core` and `pkg/tui` API guide with runnable examples
- [Embedding the TUI](docs/embedding/): Embed `kvx` inside your Go CLI, wire custom CEL functions, and compose `ChildModel` views
- [Development](docs/development/): Build with Go 1.27+, run race tests, build Hugo docs, and regenerate TUI media

---

## Links

- [GitHub Repository](https://github.com/oakwood-commons/kvx)
- [Go Package Reference (`pkg.go.dev`)](https://pkg.go.dev/github.com/oakwood-commons/kvx)
- [Releases](https://github.com/oakwood-commons/kvx/releases)
- [Discussions](https://github.com/oakwood-commons/kvx/discussions)
- [Contributing](https://github.com/oakwood-commons/kvx/blob/main/CONTRIBUTING.md)
