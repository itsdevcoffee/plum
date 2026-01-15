# CLI Commands Research

**Status:** ✅ Complete
**Related:** [CLI Commands Feature Plan](../project/2026-01-14-cli-commands-feature.md)

---

## Research Tasks

| # | Task | Priority | Status | Findings |
|---|------|----------|--------|----------|
| 1 | Enabled/disabled storage | Critical | ✅ Done | `enabledPlugins` in scoped `settings.json` |
| 2 | Plugin installation mechanics | Critical | ✅ Done | Cached to `~/.claude/plugins/cache/` |
| 3 | Local plugin registration | Critical | ✅ Done | `--plugin-dir` flag, session-only |
| 4 | Project-scoped plugin storage | Medium | ✅ Done | 4-tier scope hierarchy |
| 5 | Version/update detection | Medium | ✅ Done | Semver in manifest, `/plugin update` command |
| 6 | Marketplace manifest schema | Low | ✅ Done | Full schema documented |
| 7 | CLI framework selection | Decision | ✅ Done | Cobra recommended |

---

## 1. Enabled/Disabled Storage

**Question:** Where does Claude Code store plugin activation state?

**Findings:** ✅ RESOLVED

Plugin activation is stored in `enabledPlugins` object within scoped `settings.json` files:

| Scope | File | Shared? |
|-------|------|---------|
| Managed | `/etc/claude-code/settings.json` | System-wide |
| User | `~/.claude/settings.json` | Personal |
| Project | `.claude/settings.json` | Git (team) |
| Local | `.claude/settings.local.json` | Gitignored |

**Format:**
```json
{
  "enabledPlugins": {
    "plugin-name@marketplace-name": true,
    "another-plugin@marketplace": false
  }
}
```

**Precedence:** Managed > Local > Project > User

**Impact on Plum:**
- `plum enable/disable` must write to appropriate `settings.json`
- `plum list` must read and merge all scopes
- Need to handle precedence correctly

**Source:** [Claude Code Settings - Plugin Configuration](https://docs.anthropic.com/en/docs/claude-code)

---

## 2. Plugin Installation Mechanics

**Questions:**
- What files/dirs are created during install?
- Clone whole repo or sparse checkout?
- What validation on `.claude-plugin/` structure?
- Auto-enabled after install?

**Findings:** ✅ RESOLVED

**Installation process:**
1. Marketplace lookup — queries specified marketplace
2. Plugin caching — **copies only the specific plugin** (not whole repo)
3. Scope selection — user chooses where to install
4. Settings update — adds `enabledPlugins: true` entry

**Cache location:** `~/.claude/plugins/cache/`

**What gets copied:**
- For `"source": "./plugins/my-plugin"` — entire directory structure
- For plugins with `.claude-plugin/plugin.json` — implicit root directory
- Symlinks within plugin directory are honored
- Plugins cannot reference files outside their directory (security)

**Environment variable:** `${CLAUDE_PLUGIN_ROOT}` — absolute path to plugin directory

**Impact on Plum:**
- `plum install` must replicate the caching behavior
- Write to `~/.claude/plugins/cache/<plugin-name>/`
- Update `enabledPlugins` in `settings.json`
- Validate `.claude-plugin/plugin.json` exists

**Source:** [Plugins Reference - Plugin Caching](https://docs.anthropic.com/en/docs/claude-code)

---

## 3. Local Plugin Registration

**Questions:**
- How are local plugins referenced?
- Can plugin be anywhere?
- How does Claude Code discover local plugins at runtime?

**Findings:** ✅ RESOLVED

**Key insight:** Local plugins use `--plugin-dir` flag, NOT `--local`:

```bash
claude --plugin-dir ./my-plugin
claude --plugin-dir ./plugin-one --plugin-dir ./plugin-two
```

**Behavior:**
- Loads plugin directly from filesystem **without installation**
- **Session-only** — no persistent registration
- Does NOT modify any settings files
- Changes picked up on Claude Code restart
- Reads `.claude-plugin/plugin.json` from specified directory

**For persistent local registration:**
- Must add to a marketplace (local path, GitHub, URL)
- Then install via `/plugin install plugin-name@marketplace-name`

**Impact on Plum:**
- `plum install --local` is a misnomer — should clarify behavior
- Option A: Mimic `--plugin-dir` (session-only, just print the command)
- Option B: Create a "local" marketplace entry and install from it
- Option C: Skip local support initially, focus on marketplace installs

**Source:** [Create Plugins - Test Locally](https://docs.anthropic.com/en/docs/claude-code)

---

## 4. Project-Scoped Plugin Storage

**Questions:**
- Structure of `<project>/.claude/plugins/`?
- Project-level `installed_plugins.json`?
- How are global + project plugins merged?
- Conflict resolution (same plugin, different scopes)?

**Findings:** ✅ RESOLVED

**4-tier scope hierarchy:**

| Scope | Location | Shared? | Use case |
|-------|----------|---------|----------|
| Managed | `/etc/claude-code/settings.json` | System | IT-deployed |
| User | `~/.claude/settings.json` | No | Personal tools |
| Project | `.claude/settings.json` | Git | Team plugins |
| Local | `.claude/settings.local.json` | No | Machine-specific |

**Project scope config:**
```json
// .claude/settings.json (checked into git)
{
  "enabledPlugins": {
    "code-formatter@team-tools": true
  },
  "extraKnownMarketplaces": {
    "team-tools": {
      "source": {
        "source": "github",
        "repo": "acme-corp/claude-plugins"
      }
    }
  }
}
```

**Precedence when same plugin in multiple scopes:**
1. Managed (highest) — cannot be overridden
2. Local — overrides project and user
3. Project — overrides user
4. User (lowest)

**Impact on Plum:**
- `plum list` must merge all 4 scopes with correct precedence
- `plum install --scope=project` writes to `.claude/settings.json`
- `plum install --scope=user` writes to `~/.claude/settings.json`
- Need to detect and display effective state after merging

**Source:** [Settings - Configuration Scopes](https://docs.anthropic.com/en/docs/claude-code)

---

## 5. Version/Update Detection

**Questions:**
- How to detect newer version in marketplace?
- Version format: semver? git tags? commit SHA?
- Does manifest include version history?

**Findings:** ✅ RESOLVED

**Version format:** Strict semver only (`MAJOR.MINOR.PATCH`)
- Example: `"2.1.0"`, `"1.0.0"`, `"2.0.0-beta.1"`
- NOT git tags, commit SHAs, or arbitrary strings

**Update commands:**
```bash
/plugin marketplace update marketplace-name  # Refresh listings
/plugin update plugin-name@marketplace       # Update specific plugin
```

**Auto-update behavior:**
- Official Anthropic marketplaces: auto-update enabled by default
- Third-party marketplaces: auto-update disabled by default
- Env vars: `DISABLE_AUTOUPDATER`, `FORCE_AUTOUPDATE_PLUGINS=true`

**Version pinning:** Via git ref in marketplace source
```json
{
  "source": {
    "source": "github",
    "repo": "owner/repo",
    "ref": "v1.0.0"  // or commit SHA
  }
}
```

**Limitation:** No version history in manifest — only current version stored

**Impact on Plum:**
- Implement semver comparison for "update available" detection
- Display: `"1.0.0 → 1.1.0 available"`
- Consider caching previous versions for rollback (optional)

**Source:** [Plugins Reference - Version Management](https://docs.anthropic.com/en/docs/claude-code)

---

## 6. Marketplace Manifest Schema

**Questions:**
- Full schema for `marketplace.json`?
- Required vs optional fields?
- Plugin directory structure?

**Findings:** ✅ RESOLVED

**Plugin directory structure:**
```
my-plugin/
├── .claude-plugin/
│   └── plugin.json          # ONLY this file here
├── commands/                # Slash commands (*.md)
├── agents/                  # Subagent definitions (*.md)
├── skills/                  # Agent skills (dirs with SKILL.md)
├── hooks/
│   └── hooks.json           # Event handlers
├── .mcp.json                # MCP server configs
├── .lsp.json                # LSP server configs
└── scripts/                 # Supporting scripts
```

**Critical:** Only `plugin.json` goes in `.claude-plugin/`. Everything else at root.

**plugin.json schema:**
```json
{
  "name": "my-plugin",              // Required, kebab-case
  "version": "1.2.0",               // Semver
  "description": "What it does",
  "author": {
    "name": "Name",
    "email": "email@example.com",
    "url": "https://github.com/user"
  },
  "homepage": "https://docs.example.com",
  "repository": "https://github.com/user/plugin",
  "license": "MIT",
  "keywords": ["keyword1", "keyword2"],
  "commands": ["./custom/commands/special.md"],
  "agents": "./custom/agents/",
  "skills": "./custom/skills/",
  "hooks": "./config/hooks.json",
  "mcpServers": "./mcp-config.json",
  "outputStyles": "./styles/",
  "lspServers": "./.lsp.json"
}
```

**Impact on Plum:**
- Validation: check `.claude-plugin/plugin.json` exists
- Display: parse name, version, description, author
- `plum info` can show all metadata

**Source:** [Plugins Reference - Directory Structure](https://docs.anthropic.com/en/docs/claude-code)

---

## 7. CLI Framework Selection

**Question:** Cobra vs urfave/cli for Plum?

**Decision:** ✅ **Cobra recommended**

### Comparison

| Aspect | Cobra | urfave/cli |
|--------|-------|------------|
| GitHub stars | 35k+ | 20k+ |
| Bubbletea integration | Proven patterns | Limited examples |
| Shell completions | Built-in, auto-generated | Manual config |
| Nested subcommands | Excellent | Basic |
| Plugin manager usage | Common (gh, kubectl) | Less common |

### Why Cobra

1. **Bubbletea integration is proven** — "Charming Cobras with Bubbletea" pattern well-documented
2. **Shell completions** — Auto-generation for install, remove, list, search
3. **Scalability** — Nested subcommands support future expansion
4. **Precedent** — Go plugin managers predominantly use Cobra

### Integration Pattern

```go
var rootCmd = &cobra.Command{
    Use:   "plum",
    Short: "Plugin manager for Claude Code",
    Run: func(cmd *cobra.Command, args []string) {
        // Default: launch TUI
        p := tea.NewProgram(ui.NewModel(), tea.WithAltScreen())
        p.Run()
    },
}

var installCmd = &cobra.Command{
    Use:   "install <marketplace>:<plugin>",
    Short: "Install a plugin",
    Args:  cobra.MinimumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        scope, _ := cmd.Flags().GetString("scope")
        return installPlugin(args[0], scope)
    },
}
```

**Key gotcha:** Bubbletea requires exclusive terminal control via `tea.WithAltScreen()`. Don't mix TUI and CLI output.

**Source:** [Charming Cobras with Bubbletea](https://elewis.dev/charming-cobras-with-bubbletea-part-1)

---

## 8. Edge Cases

### Same plugin in multiple marketplaces

**Behavior:** Allowed — plugins identified by full name `plugin@marketplace`

```json
{
  "enabledPlugins": {
    "formatter@marketplace1": true,
    "formatter@marketplace2": false
  }
}
```

Both can be installed simultaneously. Install command must specify marketplace.

### Different versions in different scopes

**Partially supported** — version pinning is at marketplace level, not scope level.

Use git ref in marketplace source:
```json
{ "ref": "v1.0.0" }  // or commit SHA
```

No explicit "install version X.Y.Z" mechanism.

### Plugin dependencies

**NOT IMPLEMENTED** — active feature request ([Issue #9444](https://github.com/anthropics/claude-code/issues/9444))

Current workaround: Bundle everything into single plugin or use MCP servers.

### Marketplace unavailable during install

**Fails gracefully** — clear error message, no partial cache operations.

Recommendation: Pre-cache marketplace data.

### Version pinning (marketplace-level)

**Supported** via git ref when adding marketplace:
```bash
/plugin marketplace add anthropics/claude-code#v2.0.0
```

Or in settings:
```json
{
  "extraKnownMarketplaces": {
    "stable": {
      "source": {
        "source": "github",
        "repo": "owner/repo",
        "ref": "v2.0.0"  // tag, branch, or commit SHA
      }
    }
  }
}
```

**Effect:** Pins ALL plugins in that marketplace to that point-in-time snapshot.

**Limitation:** Cannot pin individual plugins — version control is marketplace-level only.

### Marketplace update vs Plugin update

| Operation | Effect |
|-----------|--------|
| `/plugin marketplace update` | Refreshes catalog only, installed plugins unchanged |
| `/plugin update plugin@marketplace` | Updates specific installed plugin |
| Auto-update enabled | Both catalog AND plugins update automatically |

---

## Summary of Findings

| Task | Key Finding | Impact on Design |
|------|-------------|------------------|
| 1 | `enabledPlugins` in scoped `settings.json` | Must read/write 4 config locations |
| 2 | Plugins cached to `~/.claude/plugins/cache/` | Replicate caching, not full repo clone |
| 3 | `--plugin-dir` is session-only | Reconsider `--local` flag design |
| 4 | 4-tier scope hierarchy with precedence | Merge logic needed for `plum list` |
| 5 | Semver versions, `/plugin update` command | Implement semver comparison |
| 6 | Full plugin.json schema documented | Validation and display logic ready |
| 7 | Cobra recommended for Bubbletea integration | Refactor main.go to use Cobra |

## Key Design Changes Needed

Based on research, the feature plan needs these updates:

1. **Scope model:** Change from 2 scopes to 4 scopes (managed/user/project/local)
2. **Config files:** Read/write `settings.json` instead of `installed_plugins_v2.json` for enable/disable
3. **Cache path:** Install to `~/.claude/plugins/cache/` not `~/.claude/plugins/`
4. **Local plugins:** Either drop `--local` or redesign as marketplace registration
5. **Precedence:** Implement Managed > Local > Project > User merge logic
6. **CLI framework:** Use Cobra for subcommands with Bubbletea for TUI
7. **Version comparison:** Implement semver parsing for update detection
8. **Plugin namespacing:** Full name format `plugin@marketplace` for disambiguation

---

## Changelog

| Date | Change |
|------|--------|
| 2026-01-14 | Initial research document |
| 2026-01-14 | Completed research tasks 1-4, 6 via claude-code-guide agent |
| 2026-01-14 | Completed tasks 5, 7 + edge cases; all research complete |
| 2026-01-14 | Added marketplace-level version pinning, clarified refresh vs update distinction |
