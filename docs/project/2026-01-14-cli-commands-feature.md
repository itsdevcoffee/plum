# Plum CLI Commands Feature Plan

**Status:** Ready for Implementation

---

## Claude Code Plugin System Overview

### Plugin Installation Scopes

Claude Code uses a 4-tier scope hierarchy with strict precedence:

| Scope | Location | Shared? | Use Case |
|-------|----------|---------|----------|
| **Managed** | `/etc/claude-code/settings.json` | System-wide | IT-deployed enterprise plugins |
| **User** | `~/.claude/settings.json` | No | Personal productivity plugins |
| **Project** | `.claude/settings.json` | Git (team) | Team-shared workflows |
| **Local** | `.claude/settings.local.json` | No (gitignored) | Machine-specific overrides |

**Precedence:** Managed > Local > Project > User

### File Structure

```
~/.claude/
├── CLAUDE.md                      # User-level instructions
├── settings.json                  # User scope settings + enabledPlugins
└── plugins/
    ├── cache/                     # Installed plugin files
    │   ├── ralph-wiggum/
    │   └── memory/
    ├── known_marketplaces.json    # Registered marketplace sources
    └── installed_plugins_v2.json  # Installation registry

<project>/
├── .claude/
│   ├── CLAUDE.md                  # Project-level instructions
│   ├── settings.json              # Project scope (checked into git)
│   └── settings.local.json        # Local scope (gitignored)
```

### Plugin Activation State

Enabled/disabled is stored in `enabledPlugins` within scoped `settings.json`:

```json
{
  "enabledPlugins": {
    "ralph-wiggum@claude-code-plugins": true,
    "memory@claude-code-plugins": false
  },
  "extraKnownMarketplaces": {
    "team-tools": {
      "source": { "source": "github", "repo": "acme/plugins" }
    }
  }
}
```

### Key Observations

| Observation | Implication |
|-------------|-------------|
| 4-tier scope hierarchy | Must read/merge 4 config locations |
| `enabledPlugins` in settings.json | Separate from installation registry |
| Plugins cached to `~/.claude/plugins/cache/` | Not full repo clone |
| Full name format `plugin@marketplace` | Enables same plugin from multiple sources |
| Semver versions only | Implement semver comparison for updates |
| No plugin dependencies | Each plugin is self-contained |

### Config Files Plum Must Read/Write

| File | Purpose | Plum Operations |
|------|---------|-----------------|
| `~/.claude/settings.json` | User-scope enabled plugins | enable, disable, list |
| `.claude/settings.json` | Project-scope enabled plugins | enable, disable, list |
| `.claude/settings.local.json` | Local-scope enabled plugins | enable, disable, list |
| `~/.claude/plugins/cache/` | Plugin file storage | install, remove, update |
| `~/.claude/plugins/known_marketplaces.json` | Marketplace registry | marketplace add/remove |
| Remote manifests | Available plugins | search, install |

---

## CLI Framework

**Decision:** Cobra with Bubbletea

```go
var rootCmd = &cobra.Command{
    Use:   "plum",
    Short: "Plugin manager for Claude Code",
    Run: func(cmd *cobra.Command, args []string) {
        p := tea.NewProgram(ui.NewModel(), tea.WithAltScreen())
        p.Run()
    },
}
```

**Rationale:**
- Proven Bubbletea integration ("Charming Cobras" pattern)
- Auto-generated shell completions
- Industry standard (gh, kubectl, docker)
- Scalable subcommand structure

---

## Command Structure

```bash
plum [command] [args] [flags]

# No command = launch TUI (backward compatible)
plum                              # → TUI
plum browse                       # → TUI (explicit)
```

---

## Core Commands

### 1. `plum install`

```bash
# From marketplace (full syntax)
plum install <marketplace>:<plugin>
plum install anthropics/claude-code:ralph-wiggum

# With scope
plum install ralph-wiggum@claude-code-plugins --scope=user      # default
plum install ralph-wiggum@claude-code-plugins --scope=project
plum install ralph-wiggum@claude-code-plugins --scope=local

# Multiple plugins
plum install anthropics/claude-code:ralph-wiggum anthropics/claude-code:memory
```

**Behavior:**
1. Fetch plugin from marketplace source
2. Copy to `~/.claude/plugins/cache/<plugin-name>/`
3. Validate `.claude-plugin/plugin.json` exists
4. Add `enabledPlugins` entry to appropriate `settings.json`

---

### 2. `plum remove`

```bash
plum remove <plugin>
plum remove ralph-wiggum@claude-code-plugins

# Specific scope
plum remove ralph-wiggum --scope=project

# All scopes
plum remove ralph-wiggum --all
```

**Behavior:**
1. Remove from `enabledPlugins` in specified scope(s)
2. Delete from cache if no scopes remain

---

### 3. `plum list`

```bash
plum list
plum list --scope=user
plum list --scope=project
plum list --scope=local
plum list --enabled
plum list --disabled
plum list --updates          # Show available updates
plum list --json
```

**Output:**
```
NAME            MARKETPLACE           SCOPE    STATUS    VERSION
ralph-wiggum    claude-code-plugins   user     enabled   1.0.0
memory          claude-code-plugins   project  disabled  2.1.0 → 2.2.0 available
formatter       team-tools            local    enabled   1.5.0
```

---

### 4. `plum search`

```bash
plum search <query>
plum search "memory"
plum search --category=productivity
plum search --marketplace=anthropics/claude-code
```

**Output:**
```
NAME            MARKETPLACE           DESCRIPTION
memory          claude-code-plugins   Persistent memory for Claude
claude-mem      claude-mem            Memory compression system
```

---

### 5. `plum update`

```bash
plum update                              # All plugins
plum update ralph-wiggum                 # Specific plugin
plum update --dry-run                    # Check only
plum update --scope=project              # Project scope only
```

**Behavior:**
1. Compare installed version (semver) with marketplace version
2. Re-fetch and cache if newer version available
3. Display: `"1.0.0 → 1.1.0"`

---

### 6. `plum info`

```bash
plum info <plugin>
plum info ralph-wiggum
plum info ralph-wiggum@claude-code-plugins
```

**Output:**
```
Name:        ralph-wiggum
Version:     1.0.0
Description: Iterative AI development loop
Author:      Anthropic
License:     MIT
Marketplace: claude-code-plugins
Repository:  https://github.com/anthropics/claude-code

Installed:   Yes (user scope)
Status:      Enabled
Path:        ~/.claude/plugins/cache/ralph-wiggum

Components:
  Commands:  /ralph-loop, /cancel-ralph
  Hooks:     stop-hook.sh
  Skills:    None
```

---

### 7. `plum enable` / `plum disable`

```bash
plum enable <plugin>
plum enable ralph-wiggum

plum disable <plugin>
plum disable ralph-wiggum

# With scope
plum enable ralph-wiggum --scope=project
plum disable ralph-wiggum --scope=local
```

**Behavior:**
- Sets `enabledPlugins[plugin@marketplace]` to `true`/`false`
- Writes to appropriate scoped `settings.json`
- Does not remove files

---

### 8. `plum marketplace`

```bash
plum marketplace list
plum marketplace add <source>
plum marketplace add anthropics/claude-code
plum marketplace add anthropics/claude-code#v2.0.0    # Pin to version
plum marketplace add anthropics/claude-code#abc123    # Pin to commit
plum marketplace remove <name>
plum marketplace refresh                 # Refresh catalog (not plugins)
plum marketplace refresh --update        # Refresh + update all plugins
```

**Marketplace vs Plugin updates:**
- `refresh` → fetches latest catalog, plugins stay at current version
- `refresh --update` → fetches catalog AND updates installed plugins
- `plum update` → updates plugins without refreshing catalog

---

## Global Flags

| Flag | Description |
|------|-------------|
| `--scope=<user\|project\|local>` | Target scope (default: user) |
| `--project=<path>` | Specify project path (default: cwd) |
| `--json` | JSON output for scripting |
| `--quiet` | Minimal output |
| `--verbose` | Detailed output |
| `--dry-run` | Show what would happen |

**Note:** `managed` scope is read-only (admin-controlled).

---

## Implementation Phases

### Phase 1: Foundation (Cobra Migration)
- [ ] Add Cobra dependency
- [ ] Refactor `main.go` to use Cobra root command
- [ ] `plum` (no args) → launches TUI
- [ ] `plum browse` → launches TUI
- [ ] `plum --version` / `plum --help`

### Phase 2: Read Operations
- [ ] `plum list` — merge 4 scopes, show status
- [ ] `plum info <plugin>` — detailed plugin info
- [ ] `plum search <query>` — search across marketplaces
- [ ] `plum marketplace list` — show registered marketplaces
- [ ] `--json` flag support for all read commands

### Phase 3: Write Operations
- [ ] `plum install <marketplace>:<plugin>` — cache + enable
- [ ] `plum remove <plugin>` — disable + delete cache
- [ ] `plum enable <plugin>` / `plum disable <plugin>`
- [ ] `plum marketplace add/remove`

### Phase 4: Advanced
- [ ] `plum update` — semver comparison + re-fetch
- [ ] `plum list --updates` — show available updates
- [ ] Shell completions (Cobra auto-generation)
- [ ] `plum doctor` — validate plugin structure, check for issues

---

## Edge Case Recommendations

### 1. Same plugin in multiple marketplaces

**Recommendation:** Allow it, require full name for disambiguation.

```bash
# If "formatter" exists in multiple marketplaces:
plum install formatter                           # Error: ambiguous
plum install formatter@marketplace1              # OK
plum install marketplace1:formatter              # OK (alternate syntax)
```

Display warning when ambiguity detected during search.

### 2. Version pinning

**Recommendation:** Support at marketplace level, not plugin level.

Claude Code supports pinning marketplaces to git refs:
```bash
plum marketplace add anthropics/claude-code#v2.0.0
```

This pins ALL plugins from that marketplace to that point-in-time. Plum should:
- Support `#ref` syntax when adding marketplaces
- Store ref in `extraKnownMarketplaces` config
- Display pinned version in `plum marketplace list`
- Not support per-plugin version pinning (not how Claude Code works)

### 3. Plugin dependencies

**Recommendation:** Defer — not supported by Claude Code yet.

When Claude Code adds dependency support (Issue #9444), Plum can:
- Parse `dependencies` field in plugin.json
- Auto-install dependencies during `plum install`
- Warn on `plum remove` if other plugins depend on it

For now: Document that plugins must be self-contained.

### 4. Marketplace unavailable during install

**Recommendation:** Fail fast with clear error, offer cached fallback.

```bash
$ plum install anthropics/claude-code:memory
Error: Marketplace 'anthropics/claude-code' unreachable

Options:
  --offline    Use cached manifest (may be stale)

Run 'plum marketplace refresh' when online to update cache.
```

### 5. Local plugin development

**Recommendation:** Provide helper command, don't fake installation.

Claude Code uses `--plugin-dir` for session-only loading. Plum should:

```bash
# Print the command to run Claude with local plugin
plum dev /path/to/my-plugin
# Output: claude --plugin-dir /path/to/my-plugin

# Or create a local marketplace for persistent install
plum marketplace add --local /path/to/plugins
plum install my-plugin@local
```

---

## Interoperability Vision

- **Cross-agent compatibility** — Plugin format could work with other AI coding tools
- **Shareable dotfiles** — `plum install` commands in setup scripts
- **CI/CD integration** — Automated plugin deployment for teams
- **Plugin registries** — Centralized discovery beyond GitHub

---

## Research

See [CLI Commands Research](../research/2026-01-14-cli-commands-research.md) for detailed findings.

---

## Changelog

| Date | Change |
|------|--------|
| 2026-01-14 | Initial draft |
| 2026-01-14 | Added enable/disable commands, plugin states section |
| 2026-01-14 | Moved research to separate document |
| 2026-01-14 | Updated with research findings: 4-tier scopes, Cobra, settings.json, cache paths |
| 2026-01-14 | Added edge case recommendations |
| 2026-01-14 | Added marketplace-level version pinning (#ref syntax), clarified refresh vs update |
