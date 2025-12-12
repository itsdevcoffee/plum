# Header Layout Issue - Update Notification Alignment

## Current Problem

The header bar with update notification has layout/rendering issues when switching between display modes (slim/verbose).

### What We're Trying to Achieve

```
🍑 plum - Plugin Search              ╭─────────────────────────────────╮
                                     │ ⚡ 1 new marketplace - Shift+U │
                                     ╰─────────────────────────────────╯
```

**Requirements:**
1. Title on left: `🍑 plum - Plugin Search`
2. Notification box on right: `╭─ ⚡ 1 new marketplace - Shift+U ─╮`
3. Both on same line
4. Notification aligned to the right edge
5. **Must work in BOTH slim and verbose display modes**
6. Must respect `ContentWidth()` max width constraint
7. Title and notification should be **bottom-aligned** (sit on same baseline)

## The Bug

When switching between DisplaySlim and DisplayCard modes (via Ctrl+v):
- **Verbose mode (DisplayCard):** Header renders correctly
- **Slim mode (DisplaySlim):** Header breaks - either cuts off title, misaligns, or wraps incorrectly

## Code Location

**File:** `internal/ui/view.go`
**Function:** `listView()`
**Lines:** ~185-220

**Relevant code:**
```go
title := "🍑 plum - Plugin Search"
notice := UpdateNotificationStyle.Render(noticeText) // Has rounded border, padding

// Current approach: string concatenation
b.WriteString(TitleStyle.Render(title) + spacer + notice)
```

## Attempted Solutions (DO NOT REPEAT)

### ❌ Attempt 1: `lipgloss.JoinHorizontal(lipgloss.Top, ...)`
**Result:** Title and notification on same line, but title sits higher than notification box

### ❌ Attempt 2: `lipgloss.JoinHorizontal(lipgloss.Bottom, ...)`
**Result:** Title disappears or gets cut off at top of terminal

### ❌ Attempt 3: `lipgloss.JoinHorizontal(lipgloss.Center, ...)`
**Result:** Similar issues, title positioning breaks

### ❌ Attempt 4: Simple string concatenation (`title + spacer + notice`)
**Result:** Works in verbose mode, breaks in slim mode (notification wraps to new line, creates empty box on right)

## Root Cause Hypothesis

The issue appears to be related to:
1. **Window size initialization:** `windowWidth: 80` on init, but actual terminal might be different
2. **Display mode affecting layout:** Slim vs verbose modes have different item heights, might affect header rendering
3. **Lipgloss border rendering:** The `UpdateNotificationStyle` has rounded borders and padding, making it multi-line, which breaks horizontal alignment

## What Works

**Verbose mode with `lipgloss.JoinHorizontal(lipgloss.Top)`:**
- Title visible
- Notification aligned right
- Both on same line
- Layout stable

## What Doesn't Work

**Slim mode with ANY of the above approaches:**
- Layout breaks on first render OR when toggling from verbose

## Constraints

- Cannot change `UpdateNotificationStyle` significantly (user wants rounded box)
- Must work on window sizes from 80 to 150+ columns
- Must handle notification text of varying lengths (1-9 new marketplaces)
- Must respect `ContentWidth()` which caps at `maxContentWidth = 120`

## Potential Solutions to Try

1. **Force a minimum height for title rendering** to match notification box height
2. **Use `lipgloss.Place` to explicitly position both elements**
3. **Render title and notification in separate containers with fixed heights**
4. **Pre-calculate the exact height of notification box and pad title accordingly**
5. **Use a table-like layout with explicit column widths**
6. **Investigate if slim mode is setting some state that affects rendering**

## Testing Checklist

Any solution must pass:
- [ ] Start in verbose mode - header looks good
- [ ] Switch to slim mode (Ctrl+v) - header still good
- [ ] Switch back to verbose - header still good
- [ ] Start fresh in slim mode - header looks good from init
- [ ] Works on narrow terminals (80 cols)
- [ ] Works on wide terminals (120+ cols)

## Current State

- User wants **slim mode as default**
- Currently defaulting to **verbose mode** as workaround
- Header works fine in verbose, breaks in slim

**⚠️ UPDATE:** See `docs/context/header-alignment-attempts.md` for detailed debug session with all attempted solutions (4+ approaches, all failed in slim mode)

## Files Involved

- `internal/ui/view.go` - listView() function (~line 185)
- `internal/ui/model.go` - displayMode field initialization (~line 134)
- `internal/ui/styles.go` - UpdateNotificationStyle (~line 30)
