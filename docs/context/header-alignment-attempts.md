# Header Alignment Debug Session - Slim Mode Issue

**Status:** Blocked - Multiple approaches failed in slim mode

## Problem

Header with notification box breaks in slim mode, works in verbose mode.

**Desired layout:**
```
🍑 plum - Plugin Search              ╭─────────────────────────────────╮
                                     │ ⚡ 1 new marketplace - Shift+U │
                                     ╰─────────────────────────────────╯
```

**Actual behavior in slim mode:**
```
🍑 plum - Plugin Search              ╰─────────────────────────────────
```
(Only top-right corner of notification box visible, rest wraps/breaks)

## Key Constraints

- Notification box: 3 lines tall (rounded border + content)
- Must work in both DisplaySlim and DisplayCard modes
- ContentWidth() caps at maxContentWidth = 120
- AppStyle adds Padding(1, 2) to entire view
- windowWidth starts at 80, updated by tea.WindowSizeMsg
- Title must bottom-align with notification box

## Solutions Attempted (All Failed in Slim Mode)

### Attempt 1: lipgloss.JoinHorizontal(lipgloss.Top)
**File:** `internal/ui/view.go:221`
**Result:** Works in verbose, breaks in slim (box corner wraps)

### Attempt 2: lipgloss.JoinHorizontal(lipgloss.Bottom) + No Margin
**Code:**
```go
headerTitleStyle := lipgloss.NewStyle().Foreground(Peach).Bold(true)
titleRendered := headerTitleStyle.Render(title)
spacer := lipgloss.NewStyle().Width(spacerWidth).Render("")
header := lipgloss.JoinHorizontal(lipgloss.Bottom, titleRendered, spacer, notice)
```
**Result:** Works in verbose, breaks in slim (box corner wraps)

### Attempt 3: Conservative Width Calculation
**Code:**
```go
availableWidth := contentWidth - 4  // Account for AppStyle padding
minSpacer := 4
requiredWidth := titleWidth + noticeWidth + minSpacer
if requiredWidth > availableWidth { /* stack vertically */ }
```
**Result:** Works in verbose, breaks in slim (box corner wraps)

### Attempt 4: Padding-Based Alignment + windowWidth
**Code:**
```go
headerTitleStyle := lipgloss.NewStyle().
    Foreground(Peach).Bold(true).
    PaddingTop(noticeHeight - 1)  // Align to bottom

actualWidth := m.windowWidth - 8
spacer := strings.Repeat(" ", spacerWidth)
b.WriteString(titleRendered + spacer + notice)
```
**Result:** Works in verbose, breaks in slim (box corner wraps)

## Observations

| Aspect | Verbose Mode | Slim Mode |
|--------|--------------|-----------|
| Header render | ✅ Perfect | ❌ Box wraps |
| Toggle to mode | ✅ Stable | ❌ Breaks immediately |
| Start in mode | ✅ Works | ❌ Broken on init |
| Width calc seems correct | Yes | Yes (no obvious error) |

**Pattern:** Every approach works in verbose, fails identically in slim

## Hypothesis: Root Cause

Not a width calculation issue - all approaches calculate correctly but fail identically in slim mode.

**Possible causes:**
1. Display mode affects some rendering context/state
2. Slim mode renders earlier in update cycle (before windowWidth set?)
3. Something in renderPluginItemSlim() interferes with header
4. AppStyle.Render() behaves differently based on content below header
5. Terminal control codes different between modes

## Code Locations

- Header rendering: `internal/ui/view.go:185-233`
- Display mode toggle: `internal/ui/model.go:307-311`
- Display mode init: `internal/ui/model.go:134` (defaults to DisplaySlim)
- Slim item render: `internal/ui/view.go:280` → renderPluginItemSlim()
- Window size update: `internal/ui/update.go:58-61`

## Current Code State

```go
// internal/ui/view.go:197-233
noticeHeight := lipgloss.Height(notice)
noticeWidth := lipgloss.Width(notice)

headerTitleStyle := lipgloss.NewStyle().
    Foreground(Peach).Bold(true).
    PaddingTop(noticeHeight - 1)

titleRendered := headerTitleStyle.Render(title)
titleWidth := lipgloss.Width(titleRendered)

actualWidth := m.windowWidth - 8
if actualWidth > contentWidth {
    actualWidth = contentWidth
}

minSpacer := 4
requiredWidth := titleWidth + noticeWidth + minSpacer

if requiredWidth > actualWidth || actualWidth < 60 {
    // Stack vertically
} else {
    spacerWidth := actualWidth - titleWidth - noticeWidth
    spacer := strings.Repeat(" ", spacerWidth)
    b.WriteString(titleRendered + spacer + notice)
}
```

## Next Steps to Try

1. **Add debug output** - Log actual widths in both modes to terminal
2. **Check renderPluginItemSlim** - See if it affects header somehow
3. **Force verbose mode temporarily** - Confirm workaround still works
4. **Test without AppStyle.Render** - Isolate if padding is the issue
5. **Check if first render vs subsequent** - WindowSizeMsg timing
6. **Try lipgloss.Place** - Explicit positioning in fixed container
7. **Render notification without border** - Test if border calculation wrong
8. **Check for race condition** - Display mode vs window size initialization

## Files Changed

- `internal/ui/view.go` (lines 185-233) - 4 iterations of header rendering
- Binary rebuilt after each change: `go build -o plum ./cmd/plum`

## Test Command

```bash
./plum
# Press Ctrl+v to toggle between modes
# Observe header in both DisplaySlim and DisplayCard
```
