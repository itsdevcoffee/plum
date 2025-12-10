# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2025-12-10

### Added
- Initial release of Plum 🍑
- Fuzzy search across all Claude Code plugin marketplaces
- Filter plugins by All, Available, or Installed status
- Two display modes: Card view (default) and Slim view
- Responsive UI with 4-tier breakpoint system adapting to terminal width
- Plugin detail view with comprehensive information
- One-click copy of plugin install commands to clipboard
- Built-in help menu with keyboard shortcuts reference
- Three transition styles: Instant, Zoom, and Slide Vertical
- Cross-platform support (Linux, macOS, Windows)
- Respects `CLAUDE_CONFIG_DIR` environment variable
- fzf-style keyboard navigation (Ctrl+j/k, Ctrl+p/n, etc.)
- Visual indicators for installed (●) vs available (○) plugins
- Spring-based smooth animations using Harmonica
- Support for multiple plugin marketplaces
- Metadata display: version, author, description, keywords, category, license, repository

### Technical
- Built with Bubble Tea TUI framework
- Styled with Lip Gloss
- Dynamic configuration reading from `~/.claude/plugins/`
- Reads `known_marketplaces.json` and `installed_plugins_v2.json`
- No hardcoded paths or plugin names - works for any Claude Code installation

[Unreleased]: https://github.com/itsdevcoffee/plum/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/itsdevcoffee/plum/releases/tag/v0.1.0
