# TUI UX Improvement Options

**Date:** 2026-02-04
**Status:** Analysis & Recommendations
**Context:** User feedback that plugin search vs marketplace browser feels clunky and inconsistent

## Executive Summary

The current TUI has **5 views** (plugin list, plugin detail, marketplace list, marketplace detail, help) with good separation of concerns but suffers from:

1. **Inconsistent keyboard patterns** - Tab key means different things in different contexts
2. **Hidden features** - `@marketplace` filter syntax not discoverable
3. **Multi-step workflows** - Common tasks require 4-7 navigation steps
4. **Context loss** - Active filters/search state unclear when switching views
5. **Asymmetric navigation** - Some cross-view jumps feel unintuitive

## Current Issues Deep Dive

### Issue 1: Inconsistent Tab Behavior

**Current State:**
- Plugin List: `Tab` cycles filters (All → Discover → Ready → Installed)
- Marketplace List: `Tab` cycles sort modes (Plugins → Stars → Name → Updated)

**Problem:** Same key performs conceptually different actions. Users build muscle memory in one view that doesn't transfer.

**User Impact:** Medium - Can be confusing but learnable

---

### Issue 2: Hidden Marketplace Filter Syntax

**Current State:**
- Plugin list supports `@marketplace-name` to filter by marketplace
- Only documented in help view, no visual hints

**Problem:** Users unlikely to discover this powerful feature without reading help.

**User Impact:** High - Many users probably never discover this

---

### Issue 3: Multi-Step Workflows

**Example 1: Install Discoverable Plugin**
1. View plugin details
2. Copy marketplace command (`c`)
3. Exit TUI
4. Install marketplace
5. Re-enter TUI
6. Search plugin again
7. Copy plugin command (`y`)

**7 steps total**

**Example 2: Browse Marketplace Plugins**
1. Open marketplace browser (`Shift+M`)
2. Navigate to marketplace
3. Press `Enter` for details
4. Press `f` to filter plugins

**4 steps total**

**Problem:** Common tasks require too many navigation steps and context switches.

**User Impact:** High - Primary workflows feel tedious

---

### Issue 4: Context Loss Between Views

**Current State:**
- When switching to marketplace browser, active filter (All/Discover/Ready/Installed) is preserved but invisible
- Search query remains active but hidden
- Returning to plugin list, users forget what filter/search was active

**Problem:** Users lose orientation when switching between views.

**User Impact:** Medium - Causes occasional confusion

---

### Issue 5: Asymmetric Navigation

**Current Pattern:**
- Most views: `Enter` forward, `Esc` back (symmetric)
- Marketplace detail: `f` goes to plugin list with filter (asymmetric cross-view jump)

**Problem:** The `f` key breaks expected navigation patterns - it's not a pure "back", it's a contextual action that changes views and state.

**User Impact:** Low - Works but feels inconsistent

---

### Issue 6: Display Mode Only in Plugin List

**Current State:**
- Plugin list has Card/Slim toggle (`Shift+V`)
- Marketplace list has only one display mode

**Problem:** Users might try `Shift+V` in marketplace view, expecting similar behavior.

**User Impact:** Low - Minor inconsistency

---

## Improvement Options

I'm presenting **4 options** ranging from conservative fixes to ambitious redesign. Each option builds on the previous.

---

## Option 1: Conservative Polish (Low Risk)

**Philosophy:** Fix obvious inconsistencies without changing navigation model

### Changes

#### 1.1 Unify Tab Terminology
- Rename "filters" → "views" in plugin list
- Rename "sort modes" → "order" in marketplace list
- Update status bar to clearly show: `[Tab: Next View]` vs `[Tab: Next Order]`

#### 1.2 Make `@marketplace` Filter Discoverable
- Add hint to search placeholder: `Search plugins (or @marketplace to filter)...`
- Add autocomplete: When user types `@`, show list of marketplace names
- Show active marketplace filter in status bar: `@claude-code-plugins (12 results)`

#### 1.3 Add Context Breadcrumbs
- Show active filter in status bar when returning from marketplace: `Installed | 25 plugins`
- Show previous view context when switching: `← from Marketplace Browser`

#### 1.4 Contextual Help View
- Filter help shortcuts based on current view
- Mark unavailable shortcuts as grayed out with context note

### Pros
- Low implementation risk
- Fixes most annoying UX issues
- Doesn't break existing muscle memory
- Can be done incrementally

### Cons
- Doesn't address multi-step workflow issues
- Navigation model still feels a bit clunky
- Tab behavior still conceptually different between views

### Implementation Complexity
**Effort:** 2-3 days
**Files Changed:** 5-7 files
**Risk:** Low

---

## Option 2: Unified Filter/Sort Bar (Medium Risk)

**Philosophy:** Make Tab behavior consistent by unifying filters and sort into one model

### Changes

All of **Option 1**, plus:

#### 2.1 Unified Filter Bar Model
Replace separate "filters" and "sort modes" with unified "facets":

**Plugin List Facets:**
```
All | Discover | Ready | Installed | ↑Name | ↑Updated | ↑Stars
```

**Marketplace List Facets:**
```
All | Installed | Cached | ↑Plugins | ↑Stars | ↑Name | ↑Updated
```

- First N facets are filters (mutually exclusive)
- Remaining facets are sort orders (cycle through with subsequent Tab)
- Visual separator: `|` between filters and sorts

#### 2.2 Consistent Tab Behavior
- `Tab` always cycles to next facet
- Filters are mutually exclusive (radio button model)
- Sort facets show direction arrows (↑↓)

### Pros
- Tab key now means the same thing everywhere
- More powerful: Can filter AND sort simultaneously
- Clearer mental model for users
- Marketplace list gets filtering capability

### Cons
- More complex status bar rendering
- Wider status bar required (might wrap on small terminals)
- Requires rethinking current filter/sort state management
- Could feel "busier" visually

### Implementation Complexity
**Effort:** 4-6 days
**Files Changed:** 8-12 files
**Risk:** Medium - requires refactoring filter/sort logic

---

## Option 3: Quick Action Menu (Medium-High Risk)

**Philosophy:** Reduce multi-step workflows with contextual quick actions

### Changes

All of **Option 1**, plus:

#### 3.1 Quick Action Menu (Press `Space`)

**From Plugin List:**
```
┌─ Quick Actions ────────────────┐
│ [m] Browse Marketplaces        │
│ [f] Filter by Marketplace...   │ ← Opens marketplace picker
│ [s] Sort by...                 │ ← Opens sort picker
│ [v] Toggle View Mode           │
│ [u] Refresh Cache              │
└────────────────────────────────┘
```

**From Plugin Detail (Discoverable):**
```
┌─ Quick Actions ────────────────┐
│ [i] Copy 2-Step Install        │ ← Copies both commands
│ [m] Copy Marketplace Install   │
│ [p] Copy Plugin Install        │
│ [g] Open on GitHub             │
│ [l] Copy GitHub Link           │
└────────────────────────────────┘
```

**From Marketplace List:**
```
┌─ Quick Actions ────────────────┐
│ [Enter] View Details           │
│ [f] Show Plugins from This     │ ← Direct filter, no detail view
│ [i] Copy Install Command       │
│ [g] Open on GitHub             │
└────────────────────────────────┘
```

#### 3.2 Marketplace Picker (Triggered by `f` in quick menu or `Shift+F`)
```
┌─ Filter by Marketplace ────────┐
│ > claude-code-plugins-plus     │ ← Fuzzy search enabled
│   claude-code-marketplace      │
│   anthropic-agent-skills       │
│   ...                          │
└────────────────────────────────┘
```

Select marketplace → immediately returns to plugin list with `@marketplace` filter applied.

**Reduces workflow from 4 steps to 2 steps.**

#### 3.3 Copy 2-Step Install
For discoverable plugins, `i` key copies both commands to clipboard:
```
# Step 1: Install marketplace
/plugin marketplace add feed-mob/claude-code-marketplace

# Step 2: Install plugin
/plugin install csv-parser@feedmob-claude-plugins
```

**Reduces workflow from 7 steps to 2 steps** (copy + paste).

### Pros
- Dramatically reduces multi-step workflows
- Quick actions are contextual (only show relevant options)
- Marketplace picker makes filtering discoverable
- Power users can still use direct key shortcuts
- 2-step install copy is huge UX win for discoverable plugins

### Cons
- Adds new concept (quick action menu) to learn
- `Space` key conflicts if used for page down (currently `Ctrl+d`)
- More UI complexity to maintain
- Marketplace picker is essentially a third list view

### Implementation Complexity
**Effort:** 6-8 days
**Files Changed:** 12-18 files
**Risk:** Medium-High - new UI component (menu overlay) + picker view

---

## Option 4: Unified Dual-Pane View (High Risk)

**Philosophy:** Rethink the navigation model entirely - side-by-side instead of view switching

### Changes

All of **Option 1**, plus:

#### 4.1 Dual-Pane Layout

**Wide Terminals (>140 cols):**
```
┌─────────────────────────────────────────────────────────────────┐
│ Plum - Plugin Manager                                           │
├────────────── Plugins ─────────┬──────── Marketplaces ──────────┤
│ > csv-parser                   │   claude-code-plugins-plus     │
│   claude-commit                │   claude-code-marketplace      │
│   ai-writer                    │ > anthropic-agent-skills       │
│   ...                          │   wshobson-agents              │
│                                │   ...                          │
│                                │                                │
│ @anthropic-agent-skills (2)    │ ● Installed | 2/2 plugins      │
│ All | Card View | ↑Name        │ ★ 62,390 | Updated 2h ago     │
└────────────────────────────────┴────────────────────────────────┘
```

**Narrow Terminals (<140 cols):**
Falls back to current view-switching model.

#### 4.2 Navigation
- `Tab` switches focus between panes (plugin list ↔ marketplace list)
- `Enter` on plugin → plugin detail (full screen overlay)
- `Enter` on marketplace → marketplace detail (full screen overlay)
- `f` on marketplace → filter plugin pane by selected marketplace
- Active pane has highlighted border

#### 4.3 Synchronized Filtering
- Selecting marketplace in right pane filters plugins in left pane
- Filtering plugins in left pane highlights relevant marketplace in right pane
- Real-time visual connection between panes

### Pros
- See both plugins AND marketplaces simultaneously
- No more context switching between views
- Filter relationships are visually obvious
- Modern, professional feel (like Vim split windows or VS Code panels)
- Tab key has one clear meaning: switch pane focus

### Cons
- **Major redesign** - complete rewrite of view layer
- Only works well on wide terminals (>140 cols)
- Requires fallback to single-pane on small terminals
- More complex state management (two active cursors)
- Animation system needs refactoring for pane transitions
- Accessibility concerns (more cognitive load)

### Implementation Complexity
**Effort:** 12-15 days
**Files Changed:** 20+ files (nearly all of internal/ui/)
**Risk:** High - fundamental architecture change

---

## Comparison Matrix

| Criteria | Option 1 | Option 2 | Option 3 | Option 4 |
|----------|----------|----------|----------|----------|
| **Fixes Tab inconsistency** | Partial | ✅ Full | Partial | ✅ Full |
| **Makes @filter discoverable** | ✅ Yes | ✅ Yes | ✅ Yes | ✅ Yes |
| **Reduces multi-step workflows** | ❌ No | ❌ No | ✅ Yes | ✅ Yes |
| **Fixes context loss** | ✅ Yes | ✅ Yes | ✅ Yes | ✅ Yes |
| **Implementation time** | 2-3 days | 4-6 days | 6-8 days | 12-15 days |
| **Risk level** | Low | Medium | Med-High | High |
| **Breaking changes** | None | Minor | Minor | Major |
| **Cognitive load** | Low | Low | Medium | Medium-High |
| **Terminal size requirement** | Any | Any | Any | >140 cols (with fallback) |

---

## Recommendations

### For Quick Win: **Option 1 (Conservative Polish)**
- Safe, incremental improvements
- Fixes the most annoying issues (hidden @filter, context loss, tab terminology)
- Can ship quickly
- Low risk of regression

**Recommended if:** You want immediate improvement with minimal risk

---

### For Best Balance: **Option 3 (Quick Action Menu)**
- Addresses all major issues including multi-step workflows
- Introduces one new concept (quick action menu) but familiar from other TUIs
- 2-step install copy is a killer feature for discoverability
- Marketplace picker makes filtering actually discoverable
- Medium risk but high reward

**Recommended if:** You want to meaningfully improve UX and have ~1 week to invest

---

### For Maximum Impact: **Option 2 + Option 3 (Unified Facets + Quick Actions)**
- Combine unified filter/sort bar (Option 2) with quick action menu (Option 3)
- Tab behavior is consistent everywhere
- Multi-step workflows are fast
- Best of both approaches

**Recommended if:** You want comprehensive UX overhaul with acceptable risk

---

### For Long-Term Vision: **Option 4 (Dual-Pane)**
- Complete rethink of navigation model
- Modern, professional appearance
- Only worth it if:
  - You plan to add more features that benefit from split view
  - Most users have wide terminals
  - You're willing to maintain fallback mode for narrow terminals

**Recommended if:** You want to set Plum apart visually and have 2-3 weeks to invest

---

## Implementation Strategy

### Phased Approach (Recommended)

**Phase 1 (v0.5.0):** Ship Option 1 (Conservative Polish)
- Low risk, immediate value
- Gets user feedback on terminology changes
- Estimated: 2-3 days

**Phase 2 (v0.6.0):** Add Option 3 (Quick Action Menu)
- Build on Option 1 improvements
- Focus on reducing multi-step workflows
- Estimated: 6-8 days (including testing)

**Phase 3 (v0.7.0):** Add Option 2 (Unified Facets) OR Option 4 (Dual-Pane)
- Choose based on Phase 2 user feedback
- Option 2 if users want more power in current model
- Option 4 if users want visual overhaul
- Estimated: 4-6 days (Option 2) or 12-15 days (Option 4)

### All-At-Once Approach

**Target Release: v0.5.0** - Ship Option 2 + Option 3 together
- Takes ~10-12 days total
- Comprehensive UX upgrade in one release
- Higher risk but bigger splash

---

## Next Steps

1. **Decide on approach:**
   - Conservative (Option 1)
   - Balanced (Option 2 + 3)
   - Ambitious (Option 4)
   - Phased (1 → 3 → 2 or 4)

2. **Create detailed design docs:**
   - Keyboard shortcut mapping
   - State machine transitions
   - Visual mockups for new views

3. **Prototype:**
   - Build quick action menu in isolation
   - Test unified facet bar rendering
   - Validate dual-pane layout on different terminal sizes

4. **User testing:**
   - Share prototype with early users
   - Gather feedback on navigation flow
   - Validate terminology choices

Would you like me to:
- Create detailed design docs for a specific option?
- Build a prototype for quick action menu?
- Create visual mockups?
- Start implementation on Option 1?
