# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2025-12-13

### Added
- **Marketplace Discovery System** - Browse and discover plugins from popular marketplaces even if not installed locally
- **Dynamic Registry** - Auto-fetches marketplace list from GitHub (`marketplaces.json`)
- **Auto-Update Notifications** - Shows alert when new marketplaces are available: "⚡ X new marketplace - Shift+U"
- **Discover Filter Tab** - New filter to show only plugins from uninstalled marketplaces with `[Discover]` badge
- **Ready Filter** - Renamed "Available" to "Ready" for plugins that are one command away from installation
- **Manual Refresh** - Press `Shift+U` to refresh marketplace registry and re-fetch all data
- **Arrow Key Navigation** - Use `←` and `→` for filter tab navigation (in addition to Tab/Shift+Tab)
- **Dual Copy Commands** - Press `c` to copy marketplace add command, `y` to copy plugin install command (for discoverable plugins)
- **8 Popular Marketplaces** in registry:
  - claude-code-plugins-plus (254 plugins)
  - claude-code-marketplace (10+ plugins)
  - claude-code-plugins (official)
  - mag-claude-plugins (4 plugins)
  - dev-gom-plugins (15 plugins)
  - feedmob-plugins (6 plugins)
  - anthropic-agent-skills (official)
  - wshobson-agents (65 plugins)

### Changed
- Filter order: All → Discover → Ready → Installed (was All → Available → Installed)
- Detail view for discoverable plugins shows 2-step install instructions
- Notification displays in styled box aligned to right of header
- Registry cache refreshes every 6 hours, marketplace cache every 24 hours

### Design
- Orange/peach themed color palette with warm, earthy aesthetic
- Semantic color naming (PlumBright, TextSecondary, Success, etc.)
- Updated title: "Claude Plugin Manager"
- Professional screenshots showcasing discovery features

### Technical
- New package: `internal/marketplace/` with discovery, registry, cache, and GitHub fetching
- Plugin struct adds `IsDiscoverable` field
- Cache system: `~/.plum/cache/marketplaces/` with TTL-based invalidation
- Parallel fetching of marketplace data for fast startup
- Graceful fallback to hardcoded list if registry unavailable
- Smart comparison: compares against cached registry after first update
- Cross-platform path resolution with `CLAUDE_CONFIG_DIR` support

### Initial Features (from development)
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
