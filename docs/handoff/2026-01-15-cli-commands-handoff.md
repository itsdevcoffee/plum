# CLI Commands Feature Handoff

**Date:** 2026-01-15
**Branch:** v0.4.0
**Status:** Phase 2 Complete, Ready for Phase 3

---

## Context

Plum is a plugin manager for Claude Code. We're adding CLI subcommands to complement the existing TUI, transforming it into a full package manager.

## Key Documents

| Document | Location | Purpose |
|----------|----------|---------|
| Feature Plan | `docs/project/2026-01-14-cli-commands-feature.md` | Full spec with commands, flags, phases |
| Research | `docs/research/2026-01-14-cli-commands-research.md` | Claude Code plugin system findings |

**Read the feature plan first** - it contains the complete implementation spec.

---

## What's Done (Phase 1 + Phase 2)

### Phase 1: Foundation (Cobra Migration)

```
cmd/plum/
├── main.go          # Entry point (5 lines, calls Execute())
├── version.go       # Version logic + truncateCommitHash helper
├── version_test.go  # 5 tests for version handling
├── root.go          # Root command, launches TUI by default
├── root_test.go     # 5 tests for command structure
└── browse.go        # Browse subcommand (explicit TUI launch)
```

### Phase 2: Read Operations (NEW)

```
cmd/plum/
├── list.go             # List installed plugins (scope/status/version)
├── list_test.go        # Tests for list command
├── info.go             # Detailed plugin info
├── info_test.go        # Tests for info command
├── search.go           # Search plugins across marketplaces
├── search_test.go      # Tests for search command
├── marketplace.go      # Parent marketplace command + list subcommand
└── marketplace_test.go # Tests for marketplace commands

internal/settings/      # NEW package for 4-scope settings
├── scopes.go           # Scope types, paths, precedence
├── scopes_test.go      # Scope tests
├── settings.go         # Read/merge settings.json files
├── settings_test.go    # Settings tests
└── errors.go           # Package errors
```

### Working Commands

```bash
# Phase 1 (Foundation)
plum              # Launches TUI
plum browse       # Launches TUI
plum --version    # Shows version info
plum --help       # Shows help with subcommands
plum completion   # Shell completions

# Phase 2 (Read Operations)
plum list                        # List all plugins
plum list --scope=user           # Filter by scope
plum list --enabled              # Only enabled plugins
plum list --json                 # JSON output

plum info <plugin>               # Show plugin details
plum info memory --json          # JSON output

plum search <query>              # Search plugins
plum search memory --limit=10    # Limit results
plum search -m claude-code       # Filter by marketplace

plum marketplace list            # List all marketplaces
plum marketplace list --json     # JSON output
```

### Dependencies

- `github.com/spf13/cobra v1.10.2`

### Tests & Linting

All pass: `go test ./...`
Linter clean: `golangci-lint run --timeout=5m`

---

## What's Next (Phase 3: Write Operations)

### Tasks

1. `plum install <plugin>` — install and enable plugin
2. `plum remove <plugin>` — disable and remove plugin cache
3. `plum enable <plugin>` / `plum disable <plugin>` — toggle state
4. `plum marketplace add <source>` — add marketplace
5. `plum marketplace remove <name>` — remove marketplace

### Key Implementation Details

**Install flow:**
1. Resolve plugin from marketplace
2. Download to `~/.claude/plugins/cache/<marketplace>/<plugin>/<version>`
3. Add entry to `installed_plugins_v2.json`
4. Add `enabledPlugins` entry to appropriate `settings.json`

**Scope handling for writes:**
- Default: user scope (`~/.claude/settings.json`)
- `--scope=project`: `.claude/settings.json`
- `--scope=local`: `.claude/settings.local.json`
- `--scope=managed`: error (read-only)

**Settings write pattern:**
```go
// internal/settings/write.go
func SetPluginEnabled(fullName string, enabled bool, scope Scope, projectPath string) error
func SaveSettings(settings *Settings, scope Scope, projectPath string) error
```

### Suggested File Structure for Phase 3

```
cmd/plum/
├── install.go
├── install_test.go
├── remove.go
├── remove_test.go
├── enable.go
├── enable_test.go
├── disable.go
├── disable_test.go
└── marketplace_add.go     # marketplace add/remove subcommands
    marketplace_remove.go

internal/settings/
└── write.go               # NEW - write operations
    write_test.go
```

---

## Important Patterns

### 4-Scope System

```
Managed: /etc/claude-code/settings.json (read-only, IT-deployed)
User:    ~/.claude/settings.json (personal)
Project: .claude/settings.json (team, git-tracked)
Local:   .claude/settings.local.json (machine-specific, gitignored)
```

**Precedence:** Managed > Local > Project > User

### Cobra Pattern

```go
var installCmd = &cobra.Command{
    Use:   "install <plugin>",
    Short: "Install a plugin",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation
        return nil
    },
}

func init() {
    rootCmd.AddCommand(installCmd)
    installCmd.Flags().StringP("scope", "s", "user", "Installation scope")
}
```

### Error Handling

- Use `RunE` (returns error) instead of `Run`
- Cobra handles printing errors to stderr
- Return errors, don't `os.Exit` in handlers

---

## Pre-Push Checklist

Always run before pushing:
```bash
golangci-lint run --timeout=5m
go test ./...
go build -o ./plum ./cmd/plum
```

---

## Decisions Made (Phase 2)

1. **New `internal/settings/` package** — Cleaner separation from existing config
2. **Missing settings.json** — Returns empty settings (no error)
3. **JSON output** — Array of objects with all fields

---

## Git Status

- Branch: `v0.4.0`
- Phase 1 + Phase 2 complete
- Ready for commit
