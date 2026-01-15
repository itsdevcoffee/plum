# CLI Commands Feature Handoff

**Date:** 2026-01-15
**Branch:** v0.4.0
**Status:** Phase 1 Complete, Ready for Phase 2

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

## What's Done (Phase 1)

### Files Created

```
cmd/plum/
├── main.go          # Entry point (5 lines, calls Execute())
├── version.go       # Version logic + truncateCommitHash helper
├── version_test.go  # 5 tests for version handling
├── root.go          # Root command, launches TUI by default
├── root_test.go     # 5 tests for command structure
└── browse.go        # Browse subcommand (explicit TUI launch)
```

### Working Commands

```bash
plum              # Launches TUI
plum browse       # Launches TUI
plum --version    # Shows version info
plum --help       # Shows help with subcommands
plum completion   # Shell completions (auto-added by Cobra)
```

### Dependencies Added

- `github.com/spf13/cobra v1.10.2`

### Tests

All pass: `go test ./...`
Linter clean: `golangci-lint run --timeout=5m`

---

## What's Next (Phase 2: Read Operations)

### Tasks

1. Create config package for reading scoped `settings.json` files
2. Implement 4-scope merge logic (Managed > Local > Project > User)
3. `plum list` — show installed plugins with scope/status/version
4. `plum list --json` — machine-readable output
5. `plum info <plugin>` — detailed plugin metadata
6. `plum search <query>` — search across marketplaces
7. `plum marketplace list` — show registered marketplaces

### Key Implementation Details

**4-tier scope system:**
```
Managed: /etc/claude-code/settings.json (read-only)
User:    ~/.claude/settings.json
Project: .claude/settings.json
Local:   .claude/settings.local.json
```

**Precedence:** Managed > Local > Project > User

**Settings.json structure:**
```json
{
  "enabledPlugins": {
    "plugin@marketplace": true
  },
  "extraKnownMarketplaces": {
    "name": {
      "source": { "source": "github", "repo": "owner/repo" }
    }
  }
}
```

**Plugin cache location:** `~/.claude/plugins/cache/`

### Suggested File Structure for Phase 2

```
cmd/plum/
├── list.go
├── list_test.go
├── info.go
├── info_test.go
├── search.go
├── search_test.go
└── marketplace/
    ├── marketplace.go
    ├── list.go
    └── list_test.go

internal/
├── settings/           # NEW - read/write settings.json
│   ├── settings.go
│   ├── settings_test.go
│   ├── scopes.go       # 4-tier merge logic
│   └── scopes_test.go
└── config/             # Existing - may need updates
```

---

## Important Patterns

### Test Organization

Each command file gets a corresponding test file:
- `list.go` → `list_test.go`
- Use table-driven tests for edge cases
- Test command structure + output format

### Cobra Pattern

```go
var listCmd = &cobra.Command{
    Use:   "list",
    Short: "List installed plugins",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation
        return nil
    },
}

func init() {
    rootCmd.AddCommand(listCmd)
    listCmd.Flags().StringP("scope", "s", "", "Filter by scope")
    listCmd.Flags().Bool("json", false, "Output as JSON")
}
```

### Error Handling

- Use `RunE` (returns error) instead of `Run` for commands that can fail
- Cobra handles printing errors to stderr
- Just return the error, don't os.Exit in command handlers

---

## Pre-Push Checklist

Always run before pushing:
```bash
golangci-lint run --timeout=5m
go test ./...
go build -o ./plum ./cmd/plum
```

---

## Open Decisions

1. Should `internal/settings/` be new package or extend `internal/config/`?
2. How to handle missing settings.json files (create empty or error)?
3. JSON output format for `plum list --json`

---

## Git Status

- Branch: `v0.4.0`
- Last commit: Phase 1 Cobra migration (not yet committed - working tree dirty)
- Recommend committing Phase 1 before starting Phase 2
