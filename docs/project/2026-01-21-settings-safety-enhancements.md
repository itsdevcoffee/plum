# Settings Safety Enhancements Plan

**Status:** Planning
**Created:** 2026-01-21
**Context:** Post-fix enhancements for critical settings preservation bug

---

## Background

After fixing the critical bug where plum destroyed user settings.json fields (commit 38d0ff4), an independent code review recommended three enhancements to improve user confidence and safety:

1. **Automatic Backup** - Create backup before first modification
2. **Integration Test** - End-to-end test with real plum binary
3. **User Documentation** - Explain field preservation to users

This document plans implementation in phases to avoid overwhelm.

---

## Phase 1: Automatic Backup

### Objective

Create automatic backup of settings.json before plum modifies it for the first time, giving users a recovery option if anything goes wrong.

### Design Questions

#### Q1: When to create backup?

**Options:**
- A. Before every write (safest, but creates many backup files)
- B. Before first write only (one backup per settings file)
- C. Timestamped backups with rotation (keep last N)

**Recommendation:** B - Before first write only
- Simple, clean (one backup file)
- Covers 99% of user needs (recovery from first-time corruption)
- Can enhance later with rotation if needed

#### Q2: Which scopes to backup?

**Options:**
- A. User scope only (`~/.claude/settings.json`)
- B. User + Project scopes
- C. All scopes (user, project, local)

**Recommendation:** B - User + Project
- User scope: Critical (has permissions, hooks, attribution)
- Project scope: Important (shared team config)
- Local scope: Skip (personal overrides, less critical)

#### Q3: Backup file naming?

**Options:**
- A. `settings.json.backup-plum` (simple, single backup)
- B. `settings.json.backup-20260121-135500` (timestamped)
- C. `settings.json.bak` (generic)

**Recommendation:** A - `settings.json.backup-plum`
- Clear it's from plum (user knows who made it)
- Simple to reference in docs
- Can detect if backup already exists

#### Q4: What if backup fails?

**Options:**
- A. Block write operation (fail safely)
- B. Warn and continue (don't block on backup)
- C. Silent failure

**Recommendation:** B - Warn and continue
- Backup is nice-to-have, not critical
- Shouldn't prevent plum from working
- Log warning so user knows

#### Q5: Add restore command?

**Options:**
- A. Yes: `plum settings restore --scope=user`
- B. No: Just document manual `cp` command
- C. Later: Add in Phase 3

**Recommendation:** C - Add in Phase 3
- Manual restore works fine for now
- Keep Phase 1 simple and focused
- Can add convenience command later

### Implementation Plan

**File:** `internal/settings/backup.go` (new file)

```go
package settings

import (
    "fmt"
    "os"
)

// ensureBackup creates a backup of settings.json if one doesn't exist
// Returns error only for logging - should not block writes
func ensureBackup(path string) error {
    backupPath := path + ".backup-plum"

    // Check if backup already exists
    if _, err := os.Stat(backupPath); err == nil {
        return nil // Backup already exists, nothing to do
    }

    // Check if original exists
    data, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return nil // No original file, nothing to backup
        }
        return fmt.Errorf("failed to read original: %w", err)
    }

    // Create backup
    if err := os.WriteFile(backupPath, data, 0600); err != nil {
        return fmt.Errorf("failed to write backup: %w", err)
    }

    return nil
}
```

**File:** `internal/settings/write.go` (modify existing)

```go
// In saveSettingsDirect(), add before atomic write:

// Create backup before first modification (best-effort, don't block on failure)
if err := ensureBackup(path); err != nil {
    // Log warning but continue - backup is nice-to-have, not critical
    // TODO: Add proper logging when plum has logging infrastructure
    _ = err // Ignore for now
}
```

**File:** `internal/settings/backup_test.go` (new file)

```go
package settings

import (
    "os"
    "path/filepath"
    "testing"
)

func TestEnsureBackup(t *testing.T) {
    tests := []struct {
        name           string
        setupFile      bool
        setupBackup    bool
        expectBackup   bool
        expectError    bool
    }{
        {
            name:         "creates backup when none exists",
            setupFile:    true,
            setupBackup:  false,
            expectBackup: true,
        },
        {
            name:         "skips when backup exists",
            setupFile:    true,
            setupBackup:  true,
            expectBackup: true, // Should still exist
        },
        {
            name:         "skips when no original file",
            setupFile:    false,
            setupBackup:  false,
            expectBackup: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tmpDir := t.TempDir()
            settingsPath := filepath.Join(tmpDir, "settings.json")
            backupPath := settingsPath + ".backup-plum"

            // Setup
            if tt.setupFile {
                os.WriteFile(settingsPath, []byte(`{"test": true}`), 0600)
            }
            if tt.setupBackup {
                os.WriteFile(backupPath, []byte(`{"old": true}`), 0600)
            }

            // Test
            err := ensureBackup(settingsPath)

            // Verify
            if tt.expectError && err == nil {
                t.Error("expected error, got nil")
            }

            _, err = os.Stat(backupPath)
            backupExists := err == nil

            if backupExists != tt.expectBackup {
                t.Errorf("backup exists = %v, want %v", backupExists, tt.expectBackup)
            }
        })
    }
}

func TestBackupPreservesContent(t *testing.T) {
    tmpDir := t.TempDir()
    settingsPath := filepath.Join(tmpDir, "settings.json")
    backupPath := settingsPath + ".backup-plum"

    originalContent := `{"permissions": {"allow": ["Read"]}, "model": "opus"}`
    os.WriteFile(settingsPath, []byte(originalContent), 0600)

    err := ensureBackup(settingsPath)
    if err != nil {
        t.Fatalf("ensureBackup failed: %v", err)
    }

    backupContent, _ := os.ReadFile(backupPath)
    if string(backupContent) != originalContent {
        t.Errorf("backup content mismatch:\ngot:  %s\nwant: %s", backupContent, originalContent)
    }
}
```

### Success Criteria

- ✅ Backup created on first write to settings.json
- ✅ Backup not recreated if already exists
- ✅ Backup has correct permissions (0600)
- ✅ Write operation continues even if backup fails
- ✅ Tests pass for all scenarios

### Files to Create/Modify

- Create: `internal/settings/backup.go`
- Create: `internal/settings/backup_test.go`
- Modify: `internal/settings/write.go` (add ensureBackup call)

---

## Phase 2: User Documentation

### Objective

Clearly communicate to users that their settings are safe and explain what plum manages vs preserves.

### Design Questions

#### Q1: Where to document?

**Options:**
- A. README.md only (most visible)
- B. Separate docs/settings-safety.md (detailed)
- C. Both (brief in README, detailed in separate doc)

**Recommendation:** A - README.md only (for now)
- Most users read README first
- Can move to separate doc if it gets too long
- Keep it simple for v0.4.0 release

#### Q2: What sections to add?

**Recommendation:**
1. "Settings Safety" section in README
2. Explain field preservation
3. Show backup location
4. List what plum manages

#### Q3: Tone and detail level?

**Recommendation:**
- Clear, confident tone
- Bullet points for scannability
- Technical details but not overwhelming
- Show file paths for clarity

### Implementation Plan

**File:** `README.md` (add new section)

**Location:** After "Features" section, before "Installation"

**Content:**

```markdown
## Settings Safety

Plum **preserves all fields** in your `settings.json` files. Your custom configuration is safe:

- ✅ `permissions.allow` arrays (bash command permissions)
- ✅ `hooks` (SessionStart, UserPromptSubmit, PreToolUse, etc.)
- ✅ `attribution` (commit/PR attribution settings)
- ✅ `model` preferences
- ✅ `includeCoAuthoredBy` flags
- ✅ Any other custom fields

### Automatic Backups

Before modifying settings for the first time, plum creates a backup:

```bash
~/.claude/settings.json.backup-plum  # User scope
.claude/settings.json.backup-plum    # Project scope
```

To restore from backup:
```bash
cp ~/.claude/settings.json.backup-plum ~/.claude/settings.json
```

### What Plum Manages

Plum only modifies these two fields:
- `enabledPlugins` - Plugin enable/disable states
- `extraKnownMarketplaces` - Custom marketplace sources

**Everything else in your settings.json remains untouched.**
```

### Success Criteria

- ✅ Section added to README.md
- ✅ Clear explanation of field preservation
- ✅ Backup location documented
- ✅ Restore instructions provided
- ✅ User confidence in plum safety

### Files to Modify

- Modify: `README.md`

---

## Phase 3: Integration Test

### Objective

Add end-to-end test that runs actual plum binary to verify settings preservation in realistic scenarios.

### Design Questions

#### Q1: Test infrastructure?

**Options:**
- A. New package: `internal/integration/`
- B. Add to existing: `internal/settings/`
- C. Separate: `test/integration/`

**Recommendation:** A - `internal/integration/`
- Keeps integration tests separate from unit tests
- Can add more integration tests later
- Standard Go convention

#### Q2: Build tag?

**Options:**
- A. Use `//go:build integration` tag
- B. No tag, always run
- C. Use `-short` flag to skip

**Recommendation:** A - Build tag
- Integration tests are slower (build binary)
- Allow `go test ./...` to run quickly
- Run in CI with `go test -tags=integration`

#### Q3: What scenarios to test?

**Recommendation:**
1. Install plugin with custom fields → verify preservation
2. Remove plugin with custom fields → verify preservation
3. Enable/disable with custom fields → verify preservation
4. Add marketplace with custom fields → verify preservation

#### Q4: Run in CI?

**Options:**
- A. Yes, in GitHub Actions
- B. No, manual only
- C. Optional (can be enabled/disabled)

**Recommendation:** C - Optional
- Add to pre-push checklist in CLAUDE.md
- Can enable in CI later if desired
- Keep CI fast for now

### Implementation Plan

**File:** `internal/integration/settings_preservation_test.go` (new file)

```go
//go:build integration
// +build integration

package integration_test

import (
    "encoding/json"
    "os"
    "os/exec"
    "path/filepath"
    "testing"
)

func TestSettingsPreservationEndToEnd(t *testing.T) {
    // Build plum binary
    plumBin := buildPlumBinary(t)
    defer os.Remove(plumBin)

    // Create isolated test environment
    testDir := t.TempDir()
    claudeDir := filepath.Join(testDir, ".claude")
    if err := os.MkdirAll(claudeDir, 0755); err != nil {
        t.Fatal(err)
    }

    // Create settings.json with all custom fields
    settingsPath := filepath.Join(claudeDir, "settings.json")
    initialSettings := map[string]interface{}{
        "permissions": map[string]interface{}{
            "allow": []string{"Bash(git:*)", "Read", "Write"},
        },
        "hooks": map[string]interface{}{
            "SessionStart": []interface{}{
                map[string]interface{}{
                    "hooks": []interface{}{
                        map[string]interface{}{
                            "type":    "command",
                            "command": "/path/to/script.sh",
                        },
                    },
                },
            },
        },
        "attribution": map[string]interface{}{
            "commit": "test-commit",
            "pr":     "test-pr-url",
        },
        "model":               "claude-opus-4",
        "includeCoAuthoredBy": false,
        "enabledPlugins": map[string]bool{
            "existing@market": true,
        },
    }

    writeJSON(t, settingsPath, initialSettings)

    // Run plum install (will fail due to marketplace, but should preserve fields)
    cmd := exec.Command(plumBin, "install", "ralph-wiggum", "--scope=user")
    cmd.Env = append(os.Environ(),
        "CLAUDE_CONFIG_DIR="+testDir,
        "HOME="+testDir,
    )

    output, _ := cmd.CombinedOutput()
    t.Logf("plum output: %s", output)
    // Note: Install may fail (marketplace issues), but fields should be preserved

    // Read settings after operation
    var result map[string]interface{}
    data, err := os.ReadFile(settingsPath)
    if err != nil {
        t.Fatalf("Failed to read settings: %v", err)
    }
    if err := json.Unmarshal(data, &result); err != nil {
        t.Fatalf("Failed to parse settings: %v", err)
    }

    // Verify ALL custom fields preserved
    assertFieldExists(t, result, "permissions", "permissions field lost")
    assertFieldExists(t, result, "hooks", "hooks field lost")
    assertFieldExists(t, result, "attribution", "attribution field lost")
    assertFieldExists(t, result, "model", "model field lost")
    assertFieldExists(t, result, "includeCoAuthoredBy", "includeCoAuthoredBy field lost")
    assertFieldExists(t, result, "enabledPlugins", "enabledPlugins field lost")

    // Verify values unchanged
    if result["model"] != "claude-opus-4" {
        t.Errorf("model value changed: got %v, want claude-opus-4", result["model"])
    }

    permissions := result["permissions"].(map[string]interface{})
    allow := permissions["allow"].([]interface{})
    if len(allow) != 3 {
        t.Errorf("permissions.allow changed: got %d items, want 3", len(allow))
    }

    // Verify backup was created
    backupPath := settingsPath + ".backup-plum"
    if _, err := os.Stat(backupPath); err != nil {
        t.Errorf("Backup not created at %s", backupPath)
    }
}

func buildPlumBinary(t *testing.T) string {
    t.Helper()
    tmpBin := filepath.Join(t.TempDir(), "plum")
    cmd := exec.Command("go", "build", "-o", tmpBin, "./cmd/plum")
    if output, err := cmd.CombinedOutput(); err != nil {
        t.Fatalf("Failed to build plum: %v\nOutput: %s", err, output)
    }
    return tmpBin
}

func writeJSON(t *testing.T, path string, data interface{}) {
    t.Helper()
    bytes, err := json.MarshalIndent(data, "", "  ")
    if err != nil {
        t.Fatal(err)
    }
    if err := os.WriteFile(path, bytes, 0600); err != nil {
        t.Fatal(err)
    }
}

func assertFieldExists(t *testing.T, m map[string]interface{}, field, message string) {
    t.Helper()
    if _, ok := m[field]; !ok {
        t.Error(message)
    }
}
```

**File:** `CLAUDE.md` (update pre-push checklist)

Add to pre-push checklist:
```markdown
# 4. Run integration tests (optional but recommended)
go test -tags=integration ./internal/integration/... -v
```

### Success Criteria

- ✅ Integration test passes
- ✅ Test builds plum binary
- ✅ Test verifies all custom fields preserved
- ✅ Test runs in isolation (temp directories)
- ✅ Test can be run manually or in CI

### Files to Create/Modify

- Create: `internal/integration/settings_preservation_test.go`
- Modify: `CLAUDE.md` (update pre-push checklist)

---

## Implementation Order

### Recommended Sequence

1. **Phase 2 first** (Documentation) - Quickest win, immediate user value
2. **Phase 1 second** (Automatic Backup) - Safety feature, moderate complexity
3. **Phase 3 third** (Integration Test) - Validation, most complex

### Rationale

- Documentation is easiest and provides immediate confidence
- Backup provides actual safety feature users can rely on
- Integration test validates everything works end-to-end

---

## Phase-by-Phase Questions

### For Phase 2 (Documentation)

**Questions to answer before implementation:**

1. Should we mention the critical bug fix in the safety section?
2. Should we add a "What happened?" section explaining the bug?
3. Should we include version info (e.g., "Fixed in v0.4.0")?
4. Should we add screenshots/examples?

### For Phase 1 (Automatic Backup)

**Questions to answer before implementation:**

1. Final decision on backup file naming?
2. Should backup have same permissions as original (0600)?
3. Should we log backup creation to stdout?
4. Should backup be created even if settings.json is empty?

### For Phase 3 (Integration Test)

**Questions to answer before implementation:**

1. Should test clean up plum binary after build?
2. Should test use actual marketplace or mock?
3. Should test verify backup creation too?
4. Should test include multiple operations in sequence?

---

## Success Metrics

After all phases complete:

- ✅ Users have automatic backups
- ✅ Users know their settings are safe (documented)
- ✅ Integration test validates preservation
- ✅ Code review recommendations addressed
- ✅ User confidence in plum increased

---

## Next Steps

**Immediate:**
1. Review this plan
2. Answer questions for Phase 2 (Documentation)
3. Implement Phase 2
4. Move to Phase 1 questions and implementation

**Per Phase:**
- Answer phase-specific questions
- Implement changes
- Test locally
- Commit
- Move to next phase
