# Plum CLI Commands - Testing Guide

## Scopes Overview

| Scope | Location | Use Case |
|-------|----------|----------|
| `user` | `~/.claude/settings.json` | Personal plugins (default) |
| `project` | `.claude/settings.json` | Shared with team via git |
| `local` | `.claude/settings.local.json` | Personal overrides for project |

---

## 1. `plum` / `plum browse`

**What:** Launches interactive TUI

```bash
plum
plum browse
```

**Expected:** TUI opens with plugin browser

---

## 2. `plum search <query>`

**What:** Search plugins across marketplaces

```bash
plum search memory                           # Basic search
plum search "code review"                    # Multi-word
plum search memory --limit=5                 # Limit results
plum search memory --marketplace=claude-code-plugins  # Filter marketplace
plum search memory --json                    # JSON output
```

**Expected:** List of matching plugins with name, marketplace, description

---

## 3. `plum info <plugin>`

**What:** Show detailed plugin information

```bash
plum info ralph-wiggum                       # By name (first match)
plum info ralph-wiggum@claude-code-plugins   # Specific marketplace
plum info ralph-wiggum --json                # JSON output
```

**Expected:** Plugin details (name, version, description, author, commands, etc.)

---

## 4. `plum install <plugin>`

**What:** Install a plugin

```bash
plum install ralph-wiggum                    # Default (user scope)
plum install ralph-wiggum --scope=user       # Explicit user scope
plum install ralph-wiggum --scope=project    # Project scope
plum install ralph-wiggum --scope=local      # Local scope
plum install memory@claude-code-plugins      # Specific marketplace
```

**Expected:**
- Plugin files downloaded to `~/.plum/cache/plugins/<marketplace>/<plugin>/`
- Entry added to appropriate `settings.json` with `enabled: true`
- Entry added to `~/.claude/plugins.json` registry

**Verify:**
```bash
plum list
cat ~/.claude/settings.json | jq '.plugins'
cat .claude/settings.json | jq '.plugins'        # for project scope
```

---

## 5. `plum list`

**What:** List installed plugins

```bash
plum list                                    # All plugins
plum list --scope=user                       # User scope only
plum list --scope=project                    # Project scope only
plum list --enabled                          # Enabled only
plum list --disabled                         # Disabled only
plum list --updates                          # Show available updates
plum list --json                             # JSON output
```

**Expected:** Table with NAME, MARKETPLACE, SCOPE, STATUS, VERSION columns

---

## 6. `plum enable <plugin>`

**What:** Enable a disabled plugin

```bash
plum enable ralph-wiggum                     # Default (user scope)
plum enable ralph-wiggum --scope=project     # Project scope
plum enable ralph-wiggum@claude-code-plugins # Specific marketplace
```

**Expected:** Plugin's `enabled` field set to `true` in settings.json

**Verify:**
```bash
plum list --enabled
```

---

## 7. `plum disable <plugin>`

**What:** Disable a plugin (keeps files)

```bash
plum disable ralph-wiggum                    # Default (user scope)
plum disable ralph-wiggum --scope=project    # Project scope
```

**Expected:** Plugin's `enabled` field set to `false` in settings.json

**Verify:**
```bash
plum list --disabled
```

---

## 8. `plum remove <plugin>`

**What:** Remove/uninstall a plugin

```bash
plum remove ralph-wiggum                     # Default (user scope)
plum remove ralph-wiggum --scope=project     # Project scope
plum remove ralph-wiggum --all               # All scopes
plum remove ralph-wiggum --keep-cache        # Keep cached files
```

**Aliases:** `plum uninstall`, `plum rm`

**Expected:**
- Plugin removed from settings.json
- Cache files deleted (unless `--keep-cache`)
- Registry entry removed if no other scopes reference it

**Verify:**
```bash
plum list
ls ~/.plum/cache/plugins/                    # Should be gone unless --keep-cache
```

---

## 9. `plum update [plugin]`

**What:** Update plugins to latest versions

```bash
plum update                                  # Update all
plum update ralph-wiggum                     # Specific plugin
plum update --dry-run                        # Check only, don't install
plum update --scope=project                  # Only project-scoped plugins
```

**Expected:** Compares installed versions against cached marketplace data, downloads newer versions

**Note:** Run `plum marketplace refresh` first to get latest version info from GitHub

---

## 10. `plum marketplace list`

**What:** List available marketplaces

```bash
plum marketplace list                        # Table output
plum marketplace list --json                 # JSON output
```

**Expected:** Table with NAME, DESCRIPTION, PLUGINS, STARS, STATUS columns

---

## 11. `plum marketplace add <repo>`

**What:** Add a custom marketplace

```bash
plum marketplace add myorg/my-plugins                    # User scope
plum marketplace add myorg/my-plugins --scope=project   # Project scope
plum marketplace add myorg/my-plugins#v2.0.0            # Pin to tag
plum marketplace add myorg/my-plugins#abc123            # Pin to commit
```

**Expected:** Marketplace added to `extraKnownMarketplaces` in settings.json

**Verify:**
```bash
plum marketplace list
```

---

## 12. `plum marketplace remove <name>`

**What:** Remove a custom marketplace

```bash
plum marketplace remove my-plugins                      # User scope
plum marketplace remove my-plugins --scope=project     # Project scope
```

**Expected:** Marketplace removed from `extraKnownMarketplaces`

---

## 13. `plum marketplace refresh`

**What:** Fetch fresh marketplace data from GitHub

```bash
plum marketplace refresh                     # Refresh catalog only
plum marketplace refresh --update            # Refresh + update all plugins
```

**Expected:**
- Clears `~/.plum/cache/marketplaces/`
- Fetches fresh manifests from all marketplaces
- With `--update`: also updates all installed plugins

---

## 14. `plum doctor`

**What:** Check plugin health

```bash
plum doctor                                  # Table output
plum doctor --json                           # JSON output
```

**Expected:** Health report showing:
- Missing plugin.json files
- Invalid JSON
- Orphaned cache entries
- Missing cache files
- Enabled plugins not installed

---

## Full Test Sequence

```bash
# 1. Start fresh
plum marketplace refresh

# 2. Search and explore
plum search memory
plum info memory-compressor@claude-code-plugins-plus

# 3. Install to different scopes
plum install ralph-wiggum --scope=user
plum install memory-compressor --scope=project

# 4. Verify
plum list
plum list --scope=user
plum list --scope=project

# 5. Disable/Enable
plum disable ralph-wiggum
plum list --disabled
plum enable ralph-wiggum
plum list --enabled

# 6. Update
plum update --dry-run
plum update

# 7. Remove
plum remove ralph-wiggum
plum remove memory-compressor --scope=project

# 8. Health check
plum doctor
```
