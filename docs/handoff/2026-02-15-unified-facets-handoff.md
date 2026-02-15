# Unified Facets & Quick Actions Handoff

**Date:** 2026-02-15
**Branch:** `feature/unified-facets-and-quick-actions`
**Status:** Feature complete, needs merge to `main` + cleanup

---

## Context

Plum is a TUI plugin manager for Claude Code. This branch implements a comprehensive UX overhaul: unified facets (combined filter/sort tabs) and a context-aware quick action menu (Space key overlay). It builds on top of the TUI UX improvements already merged to main (marketplace autocomplete, dynamic filters — PR #9).

## Key Documents

| Document | Location | Purpose |
|----------|----------|---------|
| Implementation Log | `docs/buzzminson/2026-02-04-tui-ux-improvements-tasks-5-11.md` | Full implementation details, timeline, decisions |
| Previous Handoff | `docs/handoff/2026-01-15-cli-commands-handoff.md` | Prior CLI commands feature (for format reference) |

---

## What's Done (3 commits ahead of `main`)

### Commit 1: `4db6259` — feat: implement unified facets and quick action menu
The core feature. ~1000 lines added across 7 files.

### Commit 2: `51523a5` — refactor: maximus code review fixes
Post-review cleanup: division-by-zero guards on empty facets, URL validation before `exec.Command`, consolidated facet cycling with `cycleFacet()` helper, extracted marketplace filtering helpers, removed duplicate utilities.

### Commit 3: `b920b64` — fix: same-name plugins from different marketplaces
Changed dedup key from plugin name to `plugin@marketplace`, so plugins with the same name from different sources both appear.

### Features Implemented

**Unified Facets Model:**
- Plugin list facets: `All | Discover | Ready | Installed ║ ↑Name | ↑Updated | ↑Stars`
- Marketplace list facets: `↑Plugins | ↑Stars | ↑Name | ↑Updated`
- Tab/Shift+Tab cycles through all facets (filters AND sorts)
- Visual `║` separator between filter and sort sections

**Quick Action Menu (Space key):**
- Context-aware overlay with keyboard shortcut hints
- Plugin List: [m] Browse Marketplaces, [f] Filter by Marketplace, [s] Sort, [v] Toggle View, [u] Refresh
- Plugin Detail (Discoverable): [i] 2-Step Install, [m] Marketplace, [p] Plugin, [g] GitHub, [l] Link
- Plugin Detail (Installed): [o] Open Local, [p] Copy Path, [g] GitHub, [l] Link
- Marketplace List: [Enter] View, [f] Show Plugins, [i] Copy Install, [g] GitHub

**2-Step Install Copy (i key):**
- On discoverable plugins, copies formatted multi-step install commands to clipboard

**Marketplace Picker Enhancement (Shift+F):**
- Alternative trigger for `@` marketplace autocomplete from the plugin list

**Same-Name Plugin Fix:**
- `internal/config/config.go` — dedup key changed from `pluginName` to `pluginName@marketplaceName`

### Files Changed (vs `main`)

| File | Lines | What Changed |
|------|-------|-------------|
| `internal/ui/quick_menu.go` | +281 (new) | Complete quick menu system |
| `internal/ui/model.go` | +418/-341 | Facet types, navigation methods, sort logic |
| `internal/ui/view.go` | +228/-66 | Unified facet rendering, quick menu overlay |
| `internal/ui/update.go` | +182/-90 | Space/Shift+F/i key handlers, quick menu keys |
| `internal/ui/marketplace_view.go` | +61/-34 | Facet-based sort tab rendering |
| `internal/ui/help_view.go` | +17/-17 | Updated help text for new shortcuts |
| `internal/config/config.go` | +37/-37 | Plugin dedup fix (name@marketplace) |
| `docs/buzzminson/...` | +307 (new) | Implementation log |

---

## Current State (verified 2026-02-15)

| Check | Status |
|-------|--------|
| `go build ./cmd/plum` | Passes |
| `go test ./...` | All pass (8 packages) |
| `golangci-lint run` | 1 pre-existing warning only (gocyclo on `handleListKeys`, complexity 45 > 40) |
| Branch pushed to origin | Up to date |
| PR open | No PR created yet |

### Uncommitted Local Changes

1. **`.gitignore`** — Added `.mcp.json` entry (local MCP server config). This is a good change, should be committed.
2. **`cmd/debug-plugins/`** — A debug utility for listing loaded plugins and checking dedup. Used during the same-name bug fix. Can be committed if useful for debugging, or removed.

---

## What Needs to Be Done

### Required: Merge to Main

1. **Decide on uncommitted files:**
   - Commit `.gitignore` change (recommended — keeps `.mcp.json` out of repo)
   - Either commit `cmd/debug-plugins/` as a dev tool or delete it
2. **Create PR** from `feature/unified-facets-and-quick-actions` → `main`
3. **Merge PR**

### Recommended: Known Issues to Address

1. **`handleListKeys` complexity (gocyclo 45 > 40):**
   - Pre-existing but getting worse as features are added
   - Refactor into sub-handlers: `handleListNavigation()`, `handleListActions()`, `handleListInput()`
   - This would bring each handler well under the 40 threshold

2. **Plugin sort modes `↑Updated` and `↑Stars` are stubs:**
   - `applyPluginSort()` in `model.go:543-551` currently sorts ALL modes by name only
   - Need `UpdatedAt` and `Stars` fields on `plugin.Plugin` struct
   - Populate from marketplace manifest or GitHub API
   - Then implement real sort comparisons

3. **Quick menu doesn't dim background:**
   - `renderQuickMenuOverlay()` receives `baseView` but ignores it (`_ = baseView`)
   - Menu renders centered but the underlying view is not dimmed/blurred
   - Would need line-by-line overlay composition for proper overlay effect

### Optional: Future Enhancements (from backlog)

4. Marketplace filter facets (All | Installed | Cached | Available)
5. Integration tests for facet navigation and quick menu
6. README update with animated GIF demos of new features
7. Quick action icons/emojis for visual distinction

---

## Architecture Notes

### Facets System

```
Facet struct {
    Type            FacetType          // FacetFilter or FacetSort
    DisplayName     string             // e.g. "All", "↑Name"
    FilterMode      FilterMode         // For filter facets
    SortMode        PluginSortMode     // For plugin sort facets
    MarketplaceSort MarketplaceSortMode // For marketplace sort facets
    IsActive        bool
}
```

- `GetPluginFacets()` returns 7 facets (4 filters + 3 sorts)
- `GetMarketplaceFacets()` returns 4 sort facets
- `cycleFacet(direction)` handles Tab/Shift+Tab navigation
- Filters and sorts are independent — you can filter AND sort simultaneously

### Quick Menu System

- `ViewQuickMenu` state overlays on top of `quickMenuPreviousView`
- `GetQuickActionsForView()` dispatches based on what view the menu was opened from
- `ExecuteQuickMenuAction()` synthesizes a `tea.KeyMsg` to reuse existing key handlers
- Direct letter key presses in the menu execute immediately (no need to navigate + Enter)

### Plugin Dedup (config.go)

```
// Before (bug): plugins deduped by name → same-name plugins from different marketplaces hidden
seenPluginNames[mp.Name]

// After (fix): deduped by full qualified name → both shown
seenPluginFullNames[mp.Name + "@" + marketplaceName]
```

---

## Pre-Push Checklist

Always run before pushing:
```bash
golangci-lint run --timeout=5m
go test ./...
go build -o ./plum ./cmd/plum
```

---

## Quick Start for Buzzminson

```bash
# You're already on the right branch
git status

# Verify everything still builds and passes
go test ./... && go build -o ./plum ./cmd/plum

# Run the TUI to manually test features
./plum

# Key things to test:
# - Tab through facets in plugin list (should see filters ║ sorts)
# - Space to open quick menu in any view
# - Shift+F to open marketplace picker
# - 'i' on a discoverable plugin detail to copy 2-step install
```
