---
title: "Configuration & Themes"
weight: 6
---

# Configuration & Themes

`kvx` merges its embedded defaults with your user configuration at `~/.config/kvx/config.yaml` (or `$XDG_CONFIG_HOME/kvx/config.yaml`). Override the active config path on any run with `--config-file <path>`.

## Inspecting Configuration

Use the `kvx config` command group to view your active configuration file or explore the merged configuration:

```bash
# Print the active config file (or embedded defaults with comments) as YAML
kvx config get

# Unmarshal and render as a bordered table or JSON
kvx config get -o table
kvx config get -o json

# Explore the full merged configuration interactively in the TUI
kvx config get -i

# Inspect a specific config file instead of XDG defaults
kvx config get --config-file ./custom-config.yaml
```

## Visual Theme Gallery

`kvx` ships with four built-in 256-color/TrueColor themes (`midnight`, `dark`, `warm`, and `cool`) that style both the interactive TUI and non-interactive bordered CLI tables:

<div class="bp-theme-grid">
{{< svg-card src="images/tui/theme-midnight.svg" alt="kvx midnight theme" >}}**midnight** (default): deep navy chrome, cyan keys, and violet row highlight{{< /svg-card >}}
{{< svg-card src="images/tui/theme-dark.svg" alt="kvx dark theme" >}}**dark**: neutral charcoal surface with crisp cyan accents and steel selection{{< /svg-card >}}
{{< svg-card src="images/tui/theme-warm.svg" alt="kvx warm theme" >}}**warm**: amber, gold, and earth-toned retro palette with rounded borders{{< /svg-card >}}
{{< svg-card src="images/tui/theme-cool.svg" alt="kvx cool theme" >}}**cool**: sleek cobalt and frost-blue palette with crisp square borders{{< /svg-card >}}
</div>

### Selecting and Listing Themes

```bash
# List all built-in and custom themes from your merged config
kvx config theme

# Override the theme for a single run
kvx examples/data/demo_services.json -i --theme warm

# Disable ANSI color sequences while preserving borders and table layout
kvx examples/data/demo_services.json --no-color
```

Set your permanent default theme or define custom themes in `~/.config/kvx/config.yaml`:

```yaml
ui:
  theme:
    default: midnight
  themes:
    blueprint:
      key_color: "#00e9f2"
      value_color: "#eff7ff"
      header_fg: "#00e9f2"
      header_bg: "#071e3d"
      border_style: rounded
      selected_fg: "#05152a"
      selected_bg: "#00e9f2"
      separator_color: "#1b3d63"
      input_bg: "#05152a"
      input_fg: "#eff7ff"
      ghost_fg: "#7894af"
      status_color: "#00e9f2"
      status_error: "#ff8585"
      status_success: "#a4f6c5"
      footer_fg: "#9bafc4"
      footer_bg: "#071e3d"
      help_key: "#00e9f2"
      help_value: "#cbdae8"
```

## Keymaps (`vim`, `emacs`, `function`)

Switch navigation and shortcut bindings per run with `--keymap`, per shell session with `KVX_KEY_MODE`, or permanently in `config.yaml` (`--keymap` overrides `KVX_KEY_MODE`, which overrides `ui.features.key_mode`):

```bash
kvx data.yaml -i --keymap vim       # default
kvx data.yaml -i --keymap emacs
kvx data.yaml -i --keymap function
KVX_KEY_MODE=emacs kvx data.yaml -i
```

```yaml
ui:
  features:
    key_mode: emacs
```

Individual menu actions (`help`, `search`, `filter`, `copy`, `copy_value`, `view_toggle`, `expr`, `quit`) can also be rebound per mode under `ui.menu` in `config.yaml`.

## Column Ordering, Hidden Columns, and Schema Defaults

Reorder keys or hide noisy columns via CLI flags or persistent config settings under `ui.formatting`:

```bash
# CLI column order override (unlisted keys follow alphabetically)
kvx examples/data/demo_services.json --column-order name,status,region,replicas
```

```yaml
ui:
  formatting:
    table:
      schema_file: ~/.config/kvx/default-schema.json
      column_order: [name, status, region, replicas]
      hidden_columns: [authToken, internal_id]
    yaml:
      indent: 2
      literal_block_strings: true
    tree:
      max_depth: 0
      max_array_inline: 3
    mermaid:
      direction: TD
```

Schema resolution priority: CLI `--schema` overrides `ui.formatting.table.schema_file`, which overrides inline `ui.formatting.table.schema`. See [Schema-Driven Views](../schema-views/) for `x-kvx-*` extensions.

## Array Index Styles

Control how array elements are labeled across table and list outputs with `--array-style`:

```bash
kvx data.yaml -e '_.items' --array-style none      # default: clean rows without index column
kvx data.yaml -e '_.items' --array-style index     # [0], [1], [2]
kvx data.yaml -e '_.items' --array-style numbered  # 1, 2, 3
kvx data.yaml -e '_.items' --array-style bullet    # bullet prefix
```

## Shell Completion

Generate tab-completion scripts for Bash, Zsh, Fish, or PowerShell:

```bash
# Bash
kvx completion bash > ~/.local/share/bash-completion/completions/kvx

# Zsh (ensure compinit is enabled in ~/.zshrc)
kvx completion zsh > "${fpath[1]}/_kvx"

# Fish
kvx completion fish > ~/.config/fish/completions/kvx.fish
```

```powershell
# PowerShell
kvx completion powershell | Out-String | Invoke-Expression
```
