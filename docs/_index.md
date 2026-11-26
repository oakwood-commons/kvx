---
title: "Documentation"
weight: 1
bookCollapseSection: true
---

# kvx Documentation

Welcome to the `kvx` documentation. Start with the guided tutorials to explore structured files and TUI workflows, or jump directly to the Go library and TUI reference guides.

## Tutorials

Step-by-step guides to get productive with `kvx` on the command line and in the interactive TUI:

- [Getting Started](tutorials/getting-started/): Installation, first run, `--where` filtering, and all output modes
- [Interactive Mode](tutorials/interactive-mode/): Tree navigation, search, `y`/`Y` copy actions, keymaps, and scripted snapshots
- [CEL Expressions](tutorials/expressions/): Dynamic querying, per-item `--where` predicates, ghost completion, and the `Ctrl+Space` Function Palette
- [Formats, JWTs & Decoding](tutorials/formats-and-decoding/): Auto-detected input formats, `Bearer` JWT inspection, and manual or `--auto-decode` string expansion
- [Schema-Driven Views](tutorials/schema-views/): JSON Schema column hints plus `x-kvx-list`, `x-kvx-detail`, and `x-kvx-status` TUI views
- [Configuration & Themes](tutorials/configuration/): Visual theme gallery, custom palettes, keymaps, and shell completion

## Reference

- [TUI Quick Guide](tui/): Panel layout, Vim/Emacs/Function keybindings, `--snapshot`, and `--debug`
- [Library Usage](library-usage/): Use `kvx` as a Go library (`pkg/core` and `pkg/tui`)
- [Embedding the TUI](embedding/): Embed the interactive viewer, custom CEL providers, and `ChildModel` components in Go
- [Development](development/): Go 1.27+ build setup, race testing, Hugo site preview, and TUI media generation
