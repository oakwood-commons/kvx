# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

<!-- To generate changelog automatically, install git-cliff and run: -->
<!-- git-cliff --unreleased -o CHANGELOG.md -->

## [0.1.8] - 2026

### Added
- Interactive TUI for exploring JSON, YAML, NDJSON, and CSV data
- CEL expression evaluation with `--expression` flag
- Multiple output formats: table, list, tree, yaml, json, raw, csv, mermaid
- Search functionality with `--search` and F3 in TUI
- Record limiting with `--limit`, `--offset`, and `--tail`
- Themeable UI with built-in themes (midnight, dark, warm, cool)
- Keymap modes: vim, emacs, function
- JSON Schema support for column display hints
- Shell completion for bash, zsh, fish, PowerShell
- Embeddable TUI and core API for library usage

### Documentation
- Comprehensive README with usage examples
- Library embedding guide (`docs/embedding.md`)
- Full library usage documentation (`docs/library-usage.md`)
- TUI documentation (`docs/tui.md`)

[Unreleased]: https://github.com/oakwood-commons/kvx/compare/v0.1.8...HEAD
[0.1.8]: https://github.com/oakwood-commons/kvx/releases/tag/v0.1.8
