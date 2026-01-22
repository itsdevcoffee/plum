# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.0] - 2026-01-22

### Added

- **Full CLI Interface** - Complete command-line interface for plugin management
  - `plum search <query>` - Search plugins across all marketplaces
  - `plum info <plugin>` - Show detailed plugin information
  - `plum install <plugin>` - Install plugins to user/project/local scope
  - `plum remove <plugin>` - Remove plugins (aliases: `uninstall`, `rm`)
  - `plum list` - List installed plugins with filters (`--enabled`, `--disabled`, `--scope`)
  - `plum enable/disable <plugin>` - Toggle plugin state
  - `plum update [plugin]` - Update plugins to latest versions
  - `plum doctor` - Health check for plugin installations
  - `plum marketplace list` - List available marketplaces
  - `plum marketplace add/remove` - Manage custom marketplaces
  - `plum marketplace refresh` - Fetch fresh data from GitHub
  - All commands support `--json` output for scripting
  - Scope support: `--scope=user|project|local`

- **Automatic Settings Backup** - Safety net for user configuration
  - Creates `settings.json.backup-plum` before first modification
  - One-time backup per settings file (idempotent)
  - Covers user and project scopes
  - Easy restore: `cp ~/.claude/settings.json.backup-plum ~/.claude/settings.json`

- **Settings Safety Documentation** - New README section explaining field preservation
  - Documents what plum manages vs preserves
  - Shows backup locations and restore instructions
  - Builds user confidence in plum safety

- **Integration Tests** - End-to-end testing with real plum binary
  - Verifies settings preservation in realistic scenarios
  - Tests multiple operations in sequence
  - Run with: `go test -tags=integration ./internal/integration/... -v`

### Fixed

- **CRITICAL: Settings.json Data Loss** - Fixed bug that destroyed user configuration
  - **Previous behavior**: Installing/removing plugins deleted `permissions`, `hooks`, `attribution`, `model`, and other custom fields
  - **New behavior**: All fields are preserved - plum only modifies `enabledPlugins` and `extraKnownMarketplaces`
  - Implemented custom JSON marshaling to preserve unknown fields
  - Added 4 comprehensive unit tests for field preservation
  - This was a production blocker that affected all users

### Security

- Addressed code review security issues in CLI commands
- Added input validation for plugin names and marketplace sources
- Atomic file writes with temp file + rename pattern
- Secure file permissions (0600) for settings and backups

### Changed

- Updated marketplace stats and plugin counts (2026-01-21 snapshot)

## [0.3.6] - 2026-01-11

### Changed
- **Cleaner Version Output** - Hide commit and build time when unavailable
  - `go install` users now see clean output: `plum version v0.3.6`
  - Homebrew/binary users still see full details with commit hash and build time
  - Removes clutter from "none" and "unknown" values

## [0.3.5] - 2026-01-11

### Fixed
- **Version Display for `go install`** - Now shows actual version instead of "dev"
  - Uses `runtime/debug.ReadBuildInfo()` to read version from Go module system
  - Displays correct version, commit hash, and build time
  - No more confusion about what version is installed
  - Works for both `go install` and pre-built binaries

**Before:**
```
plum version dev
  commit: none
  built: unknown
```

**After:**
```
plum version v0.3.5
  commit: 3d98de6
  built: 2026-01-11T07:37:42Z
```

## [0.3.4] - 2026-01-11

### Changed
- **Updated Marketplace Data** - Fresh sync with all 12 marketplaces (snapshot: 2026-01-11)
  - Total plugins: 600+ → **750+ plugins** (+25% growth)
  - claude-code-marketplace: 10 → 117 plugins (+107 🚀)
  - claude-plugins-official: 3 → 44 plugins (+41)
  - claude-code-plugins-plus: 254 → 280 plugins (+26)
  - claude-code-plugins: 5 → 13 plugins (+8)
  - mag-claude-plugins: 4 → 10 plugins (+6)
  - wshobson-agents: 65 → 68 plugins (+3)
  - Updated GitHub stats for all marketplaces (stars, forks, last push dates)
  - claude-code: 50,055 → 54,841 stars
  - claude-mem: 9,729 → 13,076 stars (+35%)
  - claude-plugins-official: 1,158 → 2,732 stars (+136%)
  - anthropic-agent-skills: 30,756 → 37,240 stars

### Added
- **claude-plugins-official marketplace** - Added missing link to README (44 plugins)
- **Plugin Count Verification Script** - `scripts/check-plugin-counts.go`
  - Fetches live plugin counts from all marketplace manifests
  - Validates accuracy of README and discovery.go
- **CLAUDE.md Maintenance Guide** - Project workflow documentation
  - Pre-push checklist (linting, tests, build)
  - Routine maintenance for marketplace data syncs
  - golangci-lint installation instructions

### Fixed
- **CI Linting** - Upgraded golangci-lint v1.64.8 → v2.8.0 for Go 1.24 support
  - Migrated config to v2 format
  - Using official golangci-lint-action@v9 in CI
  - Fixed all linting issues (0 issues now)
  - CI will no longer fail on lint step
- **Installation Documentation** - Improved clarity with all installation options
  - Homebrew, Windows/Manual, Go install methods clearly separated
  - Better troubleshooting guidance for PATH issues

## [0.3.3] - 2025-12-31

### Fixed
- **Git URL Source Parsing** - Critical fix for claude-plugins-official compatibility
  - Added custom JSON unmarshaling to handle source field as string OR object
  - Fixes parsing for Git-hosted plugins (Atlassian, Figma, Vercel, Notion, Sentry, etc.)
  - MarketplacePlugin now handles `{"source": "url", "url": "https://..."}` format
  - Plugin.Source now handles both `"./plugins/name"` and Git URL objects
- **Local Marketplace Loading** - Plugin counts now load from local installations when cache empty
  - Fixes "(? plugins)" display for installed marketplaces
  - claude-plugins-official now shows "(40 plugins)" correctly
  - Works for any locally installed marketplace

### Changed
- **Updated Moved Repo URLs** - Fixed broken marketplace links
  - claude-code-plugins-plus: Updated to new repo location (845 ⭐)
  - claude-code-marketplace: Updated to new repo location (577 ⭐)
- **Fresh Static Stats** - Updated all marketplace stats (snapshot: 2025-12-31)
  - claude-code: 50,055 stars (+245)
  - anthropic-agent-skills: 30,756 stars (+819)
  - wshobson-agents: 23,995 stars (+100)
  - claude-mem: 9,729 stars (+144)
  - claude-plugins-official: 1,158 stars (NEW!)
  - All 12 marketplaces now have timestamps (fixes "🕒 unknown" display)

### Added
- **Developer Script** - `scripts/update-marketplace-stats.sh`
  - Fetches fresh GitHub stats for all marketplaces
  - Outputs in Go code format for easy updates
  - Rate-limited, error handling for moved repos
  - Usage: `./scripts/update-marketplace-stats.sh`

## [0.3.2] - 2025-12-30

### Fixed
- **Marketplace Stats Display** - GitHub stats (⭐ stars, 🍴 forks, 🕒 updated) now display correctly in marketplace browser
  - Fixed static stats lookup when using remote registry
  - Stars and Updated sorting tabs now work as intended
  - Stats load from cache → static fallback → none (graceful degradation)

### Added
- **Expanded UI Test Suite** - 49 additional test cases (total: 166+ across all packages)
  - Copy functionality tests (install commands, marketplace commands)
  - Marketplace browser navigation tests
  - Display mode toggle and quit behavior tests
  - Helper method validation (counts, widths, filters)
  - Animation state and transition tests
  - UI coverage: 10.6% → 20.2%
- **Strict Linting** - Enhanced code quality enforcement
  - Added 6 new linters: revive, gocritic, gocyclo, unconvert, unparam, prealloc
  - Cyclomatic complexity monitoring (threshold: 40)
  - Style, performance, and diagnostic checks
  - Total: 11 linters active in CI/CD
- **Centralized Key Bindings** - keybindings.go for maintainability
  - 18 semantic actions (ActionQuit, ActionCopy, etc.)
  - 5 view-specific binding maps
  - Single source of truth for all key mappings
  - Prepares for future complexity reduction
- **TESTING.md Guide** - Comprehensive testing documentation
  - Quick start commands and coverage standards
  - 4 testing patterns with examples (table-driven, integration, fixtures, temp files)
  - Package-specific notes and best practices
  - Debugging guide and CI/CD info
- **Godoc Comments** - Improved API documentation
  - Model, Search, Plugin types documented
  - Scoring algorithm details
  - Usage patterns and thread-safety notes

### Changed
- **Reduced Complexity** - ApplyMarketplaceSort refactored (complexity: 21 → 5)
  - Replaced O(n²) bubble sort with O(n log n) sort.Slice
  - Extracted 4 dedicated comparison functions
  - 76% complexity reduction, improved performance
- **Refactoring TODOs** - Documented future improvements
  - handleDetailKeys (35) - planned sub-handler split
  - handleListKeys (36) - planned keybinding integration

## [0.3.1] - 2025-12-30

### Fixed
- **Detail View Scrolling** - Critical UX bug where long plugin descriptions (e.g., whimsy-injector) were cut off with no scroll capability
  - Added scrollable viewport with sticky header/footer (matches help menu pattern)
  - Header (plugin name, badge, metadata) stays pinned to top
  - Content (description, keywords, install instructions) scrolls with visual scrollbar
  - Footer (key bindings) stays pinned to bottom
  - Arrow key (↑↓) and mouse wheel scrolling support
  - Responsive window resize maintains proper layout
  - Scrollbar appears on right side when content overflows

### Added
- **Comprehensive Test Suite** - 117 new test cases across 6 test files
  - `internal/search`: 98.1% coverage (35 test cases) - fuzzy matching, scoring algorithms
  - `internal/plugin`: 100% coverage (37 test cases) - all methods and edge cases
  - `internal/ui`: 10.6% coverage (24 test cases) - core user flows, navigation, view transitions
  - `internal/marketplace`: 41.0% coverage (21 test cases) - GitHub stats, cache, refresh, discovery
- **Static GitHub Stats** - Hardcoded fallback stats for 9/12 marketplaces (snapshot: 2025-12-30)
  - Users see stats immediately on first run without API calls
  - Fallback mechanism: cache → static stats → none
  - Reduces GitHub API rate limit pressure
  - Stats: claude-code (49.8k★), anthropic-agent-skills (29.9k★), wshobson-agents (23.9k★), claude-mem (9.6k★), and 5 more
- **Development Documentation** - Comprehensive guide in README
  - Go 1.24+ requirements clearly documented
  - Test running commands with coverage options
  - Code formatting and linting instructions
  - Tooling version mismatch troubleshooting

### Changed
- Expanded PopularMarketplaces list from 8 to 12 entries (matches registry)
- Marketplace coverage improved from 29.3% to 41.0%
- Mouse wheel scrolling now works in both help and detail views

## [0.3.0] - 2025-12-30

### Added
- **Marketplace Browser** (`Shift+M`) - Dedicated view for browsing all 11 marketplaces
  - GitHub stats integration (stars, forks, last commit date)
  - Status badges: Installed/Cached/Available/New
  - Plugin count and your installed count per marketplace
  - Sort modes: by plugins, stars, name, or last updated
  - Detail view with full marketplace info
  - Filter plugins by marketplace with 'f' key
- **Plugin Source Access** - Quick access to plugin code
  - `g` - Open plugin on GitHub in browser
  - `o` - Open local directory (installed plugins)
  - `p` - Copy local path to clipboard
  - `l` - Copy GitHub link to clipboard
- **@marketplace Filter Syntax** - Type `@marketplace-name` to filter plugins by marketplace
- **Enhanced Refresh** - Shift+U improvements
  - Progress counter showing X/Y marketplaces
  - Shows current marketplace being fetched
  - Reduced timeout from 30s to 15s
  - Press Esc to cancel refresh
- **Scrollable Help Menu** - Help view now scrolls on small terminals
  - Sticky header and footer (always visible)
  - Visual scrollbar (plum-themed █ and ░)
  - Responsive to terminal resize
- **claude-mem Marketplace** - Added to registry (11 total marketplaces now)

### Changed
- **Key Binding Consistency** - All major commands now use Shift modifiers
  - `Ctrl+v` → `Shift+V` (toggle view mode)
  - `m` → `Shift+M` (marketplace browser) - avoids search conflicts
- **Help Menu Redesign**
  - Organized into logical sections with emoji headers
  - Fixed width (62 chars) for better readability
  - Removed clutter, added visual dividers
  - Context hints for detail-view-only commands
- **Status Bar Enhancement** - Keys now highlighted in orange for better visibility
- **Visual Feedback** - Specific flash messages for each action
  - 'g' → "✓ Opened!" (orange)
  - 'l' → "✓ Link Copied!" (green)
  - 'o' → "✓ Opened!" (orange)
  - 'p' → "✓ Path Copied!" (green)
  - Flash messages replace specific keys instead of adding clutter

### Fixed
- **Installed Tab** - Now correctly reads `installed_plugins.json` (was looking for `_v2` suffix)
- **CI/CD** - All linting issues resolved, tests passing
- **Cache Errors** - SaveToCache failures now logged instead of silently ignored
- **Help Menu Sizing** - Box height matches content, no unnecessary expansion
- **Header Clipping** - Help header stays visible on all terminal sizes

## [0.2.0] - 2025-12-13

### Added
- **Homebrew tap support** - Install with `brew install itsdevcoffee/plum/plum`
- **GoReleaser automation** - Automated multi-platform releases via GitHub Actions
- **Pre-built binaries** - macOS (Intel/ARM), Linux (amd64/arm64), Windows (amd64/arm64)
- **Version flag** - Check version with `plum --version` or `plum -v`

### Changed
- Release process now automated via GoReleaser
- Binaries include embedded version, commit hash, and build date
- Cross-platform distribution improved

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
