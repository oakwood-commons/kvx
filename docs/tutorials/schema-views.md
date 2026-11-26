---
title: "Schema-Driven Views"
weight: 5
---

# Schema-Driven Views

With `--schema <path>` on the CLI or `tui.ParseSchemaWithDisplay` in Go, a single JSON Schema document can drive both **non-interactive columnar table layout** and **rich interactive TUI views**: scrollable Card Lists (`x-kvx-list`), Sectioned Detail screens (`x-kvx-detail`), and Async Status & Waiting screens (`x-kvx-status`) with hotkey actions and masked inline input.

## Part 1: Standard JSON Schema Column Hints

Even without vendor extensions, `kvx` reads standard JSON Schema keywords from `properties` (for objects) or `items.properties` (for arrays of objects) to format columnar tables:

| JSON Schema Keyword | `ColumnHint` Effect |
|---------------------|---------------------|
| `title` | Overrides the column header label (`DisplayName`) |
| `maxLength` | Caps column width in characters (`MaxWidth`), never narrower than the header |
| `enum` | Caps width to the longest enum value |
| `format` | Applies sensible width caps (`date`: 10, `time`: 15, `ipv4`: 15, `date-time`: 26, `uuid`: 36, `ipv6`: 39, `email`: 40, `uri`/`iri`: 60) |
| `type: "integer"` or `"number"` | Right-aligns numeric values (`Align: "right"`) |
| `deprecated: true` | Hides the column (`Hidden: true`) |
| `required` | Boosts column shrink-resistance priority (`+10`) when terminal space is tight |
| Unconstrained width (`MaxWidth == 0`) | Marks visible unconstrained columns as `Flex: true` so bordered tables expand to fill available width |

Run with a schema on the CLI:

```bash
kvx examples/data/users.json --schema examples/data/users_schema.json -o table
```

## Part 2: Card-List (`x-kvx-list`) and Sectioned Detail (`x-kvx-detail`)

Adding `x-kvx-*` vendor extension keys to your JSON Schema upgrades `kvx -i` (and `--snapshot`) from a flat `KEY`/`VALUE` grid into a domain-tailored catalog explorer.

### Card-List View (`x-kvx-list`)

When `x-kvx-list` is defined on an array schema, `kvx` renders each object as a compact card showing a bold title, inline pill badges, optional secondary metadata, and a wrapped/truncated subtitle:

{{< svg-card src="images/tui/schema-card-list.svg" alt="kvx Card-List view configured via x-kvx-list" >}}Card-List view for `examples/data/demo_services.json` using `x-kvx-icon`, `x-kvx-collectionTitle`, and `x-kvx-list`.{{< /svg-card >}}

| `x-kvx-list` Field | Type | Description |
|--------------------|------|-------------|
| `titleField` | `string` *(required)* | Object property rendered as the bold card title |
| `subtitleField` | `string` | Object property rendered beneath the title |
| `subtitleMaxLines` | `int` | Maximum subtitle lines per card (default: `1`) |
| `badgeFields` | `[]string` | Properties rendered as inline colored pill badges beside the title (array values expand into multiple badges) |
| `secondaryFields` | `[]string` | Additional metadata keys rendered beneath the subtitle |
| `arrayStyle` | `string` | Optional per-list index override (`none`, `index`, `numbered`, `bullet`) |

Top-level schema metadata keys:
- `x-kvx-icon`: Optional symbol shown before the collection header.
- `x-kvx-collectionTitle`: Collection heading displayed in the top border (for example `"Services"` or `"Providers"`).

### Sectioned Detail View (`x-kvx-detail`)

Pressing `Enter` or `l` (`Right`) on a card drills into its **Sectioned Detail** view:

{{< svg-card src="images/tui/schema-detail-view.svg" alt="kvx Sectioned Detail view configured via x-kvx-detail" >}}Sectioned Detail view combining inline metadata, a wrapped Summary paragraph, Capabilities tag pills, and a Runtime table.{{< /svg-card >}}

Each entry in `x-kvx-detail.sections` groups one or more `fields` under an optional `title` using one of four layouts:

| Section `layout` | Rendering Style |
|------------------|-----------------|
| `inline` | Compact single-line header values separated by middle dots (for example `api-gateway / 2.4.1 / us-east-1 / running`) |
| `paragraph` | Full-width word-wrapped prose block that preserves embedded `\n` line and paragraph breaks instead of collapsing or escaping them |
| `tags` | Individual colored pill badges for string slices (`[]string`, named Go slice types, and `[]any` all render element-by-element rather than as a raw JSON array string) |
| `table` *(default)* | Two-column `KEY`/`VALUE` rows (or columnar sub-tables using optional `columnOrder` when a field holds an array of objects) |

Any object properties not listed in `sections` (and not excluded via `hiddenFields`) are automatically collected into a trailing `Other` table section so no data is hidden accidentally.

### Complete `x-kvx-list` + `x-kvx-detail` Schema Example

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "array",
  "x-kvx-collectionTitle": "Services",
  "x-kvx-list": {
    "titleField": "name",
    "subtitleField": "description",
    "subtitleMaxLines": 1,
    "badgeFields": ["status", "region", "version"]
  },
  "x-kvx-detail": {
    "titleField": "displayName",
    "hiddenFields": ["authToken"],
    "sections": [
      { "fields": ["name", "version", "region", "status"], "layout": "inline" },
      { "title": "Summary", "fields": ["description"], "layout": "paragraph" },
      { "title": "Capabilities", "fields": ["capabilities"], "layout": "tags" },
      { "title": "Runtime", "fields": ["replicas", "config"], "layout": "table" }
    ]
  },
  "items": {
    "type": "object",
    "required": ["name", "status"],
    "properties": {
      "name": { "type": "string", "title": "Service", "maxLength": 18 },
      "status": { "type": "string", "title": "Status" },
      "replicas": { "type": "integer", "title": "Replicas" }
    }
  }
}
```

Try it interactively:

```bash
kvx examples/data/demo_services.json --schema examples/data/demo_services_schema.json -i
```

### Toggling Back to the Raw Table (`v` / `Alt+v` / `F2`)

Need to inspect `hiddenFields` (such as `authToken`), copy raw CEL paths (`y`), or copy a field's JSON value (`Y`)? Press **`v`** in Vim mode (**`Alt+v`** in Emacs mode, **`F2`** in Function mode) from **either** the Card-List or the Sectioned Detail view to switch directly into the raw `KEY`/`VALUE` table for that exact node:

{{< svg-card src="images/tui/schema-view-toggle-raw.svg" alt="kvx raw table view after pressing v inside a schema detail screen" >}}Pressing `v` on the Auth Token Issuer detail screen reveals the underlying `KEY`/`VALUE` table (including `hiddenFields`) and restores `y`/`Y` copy actions.{{< /svg-card >}}

`kvx` manages footer hints automatically across transitions:
- `v view` appears in the footer **only** on nodes where a schema Card-List or Detail view applies; if you drill deeper into a child map or array without a schema view, `v view` hides automatically.
- `y copy` and `Y copy value` hide while the formatted Card-List or Detail view is displayed and reappear as soon as you press `v` to enter raw mode.

## Part 3: Async Status & Waiting Screens (`x-kvx-status`)

For device-code OAuth flows, remote callback handshakes, or long-running provisioning steps, `x-kvx-status` renders a dedicated status panel with labeled fields, a live spinner, per-keymap hotkeys (`copy-value` and `open-url`), and an optional **auto-focused inline input** that supports bullet masking:

{{< svg-card src="images/tui/status-screen.svg" alt="kvx async status screen with hotkey actions and masked inline input" >}}`x-kvx-status` screen showing verification fields, `c`/`o` hotkey actions, and a masked inline callback token input with character count.{{< /svg-card >}}

### Declaring `x-kvx-status` in JSON Schema

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "x-kvx-status": {
    "titleField": "title",
    "messageField": "messages",
    "waitMessage": "Waiting for browser verification or callback token...",
    "successMessage": "Session authenticated",
    "timeout": "2m",
    "doneBehavior": "exit-after-delay",
    "doneDelay": "2s",
    "displayFields": [
      { "label": "Verification URL", "field": "url" },
      { "label": "One-Time Code", "field": "code" }
    ],
    "actions": [
      {
        "label": "Copy code",
        "type": "copy-value",
        "field": "code",
        "keys": { "vim": "c", "emacs": "alt+c", "function": "f2" }
      },
      {
        "label": "Open browser",
        "type": "open-url",
        "field": "url",
        "keys": { "vim": "o", "emacs": "alt+o", "function": "f3" }
      }
    ],
    "input": {
      "label": "Paste callback token (masked):",
      "placeholder": "https://localhost:8400/callback?token=...",
      "prompt": "> ",
      "masked": true,
      "maxLength": 4096
    }
  }
}
```

Key behaviors:

- **Inline input focus & action routing**: When `input` is configured, the text prompt is focused on frame 1 so users can paste or type immediately. Single-character action keys (such as `c` or `o`) insert into the text buffer while typing; when `input` is omitted, `c` and `o` trigger their status actions directly (or use function/modifier bindings such as `F2`/`F3` or `Alt+c`/`Alt+o`).
- **Secret masking (`masked: true`)**: Echoes masked bullet characters, displays a live `(N chars)` length indicator when non-empty, and suppresses keystroke debug logging so pasted tokens never leak into `--debug` logs or terminal scrollback.
- **Non-interactive CLI fallback**: Running `kvx status.json --schema status_schema.json` without `-i` renders a clean plain-text summary of the title, messages, and display fields for CI logs, while explicit `-o json` or `-o yaml` flags still emit machine-readable data.

## Standalone Display Schemas (`displaySchema: "v1"`)

If you do not need JSON Schema validation keywords, you can also author a standalone display schema JSON document parsed via `tui.ParseDisplaySchema`:

```json
{
  "displaySchema": "v1",
  "collectionTitle": "Providers",
  "list": {
    "titleField": "name",
    "subtitleField": "description",
    "badgeFields": ["category", "version"]
  }
}
```

See [Library Usage](/docs/library-usage/#status-screens--masked-inline-input-x-kvx-status) and the runnable [`examples/display_schema`](../../examples/display_schema) and [`examples/status-screen`](../../examples/status-screen) programs for Go integration details.
