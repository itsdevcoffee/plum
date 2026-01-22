package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureBackup_CreatesBackupWhenNoneExists(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")
	backupPath := settingsPath + ".backup-plum"

	// Create original settings file
	originalContent := `{"permissions": {"allow": ["Read"]}, "model": "opus"}`
	if err := os.WriteFile(settingsPath, []byte(originalContent), 0600); err != nil {
		t.Fatalf("Failed to create original file: %v", err)
	}

	// Call ensureBackup
	err := ensureBackup(settingsPath)
	if err != nil {
		t.Fatalf("ensureBackup failed: %v", err)
	}

	// Verify backup was created
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Fatal("Backup file was not created")
	}

	// Verify content matches
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("Failed to read backup: %v", err)
	}

	if string(backupContent) != originalContent {
		t.Errorf("Backup content doesn't match original.\nGot: %s\nWant: %s", backupContent, originalContent)
	}
}

func TestEnsureBackup_SkipsWhenBackupExists(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")
	backupPath := settingsPath + ".backup-plum"

	// Create original settings file
	originalContent := `{"model": "opus"}`
	if err := os.WriteFile(settingsPath, []byte(originalContent), 0600); err != nil {
		t.Fatalf("Failed to create original file: %v", err)
	}

	// Create existing backup with different content
	existingBackupContent := `{"model": "sonnet"}`
	if err := os.WriteFile(backupPath, []byte(existingBackupContent), 0600); err != nil {
		t.Fatalf("Failed to create existing backup: %v", err)
	}

	// Call ensureBackup
	err := ensureBackup(settingsPath)
	if err != nil {
		t.Fatalf("ensureBackup failed: %v", err)
	}

	// Verify backup content was NOT overwritten
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("Failed to read backup: %v", err)
	}

	if string(backupContent) != existingBackupContent {
		t.Errorf("Existing backup was overwritten.\nGot: %s\nWant: %s", backupContent, existingBackupContent)
	}
}

func TestEnsureBackup_SkipsWhenNoOriginalFile(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")
	backupPath := settingsPath + ".backup-plum"

	// Don't create any files - call ensureBackup on non-existent file
	err := ensureBackup(settingsPath)
	if err != nil {
		t.Fatalf("ensureBackup should not fail when original doesn't exist: %v", err)
	}

	// Verify no backup was created
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Error("Backup file should not have been created when original doesn't exist")
	}
}

func TestEnsureBackup_PreservesContent(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")
	backupPath := settingsPath + ".backup-plum"

	// Create a complex original settings file
	originalContent := `{
  "permissions": {
    "allow": ["Bash(git:*)", "Read", "WebSearch", "Write"],
    "deny": ["Bash(rm -rf:*)"]
  },
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {"type": "command", "command": "echo Starting session"}
        ]
      }
    ]
  },
  "model": "claude-opus-4",
  "enabledPlugins": {
    "existing-plugin@market": true
  }
}`
	if err := os.WriteFile(settingsPath, []byte(originalContent), 0600); err != nil {
		t.Fatalf("Failed to create original file: %v", err)
	}

	// Call ensureBackup
	err := ensureBackup(settingsPath)
	if err != nil {
		t.Fatalf("ensureBackup failed: %v", err)
	}

	// Verify backup content is byte-for-byte identical
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("Failed to read backup: %v", err)
	}

	if string(backupContent) != originalContent {
		t.Errorf("Backup content doesn't match original exactly.\nGot length: %d\nWant length: %d",
			len(backupContent), len(originalContent))
	}
}

func TestEnsureBackup_SetsCorrectPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")
	backupPath := settingsPath + ".backup-plum"

	// Create original settings file
	originalContent := `{"model": "opus"}`
	if err := os.WriteFile(settingsPath, []byte(originalContent), 0600); err != nil {
		t.Fatalf("Failed to create original file: %v", err)
	}

	// Call ensureBackup
	err := ensureBackup(settingsPath)
	if err != nil {
		t.Fatalf("ensureBackup failed: %v", err)
	}

	// Check permissions
	info, err := os.Stat(backupPath)
	if err != nil {
		t.Fatalf("Failed to stat backup file: %v", err)
	}

	// On Unix, verify permissions are 0600 (owner read/write only)
	mode := info.Mode().Perm()
	expectedMode := os.FileMode(0600)
	if mode != expectedMode {
		t.Errorf("Backup permissions are %o, want %o", mode, expectedMode)
	}
}

func TestEnsureBackup_IsIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")
	backupPath := settingsPath + ".backup-plum"

	// Create original settings file
	originalContent := `{"model": "opus"}`
	if err := os.WriteFile(settingsPath, []byte(originalContent), 0600); err != nil {
		t.Fatalf("Failed to create original file: %v", err)
	}

	// Call ensureBackup multiple times
	for i := 0; i < 3; i++ {
		err := ensureBackup(settingsPath)
		if err != nil {
			t.Fatalf("ensureBackup call %d failed: %v", i+1, err)
		}
	}

	// Modify the original file
	modifiedContent := `{"model": "sonnet", "new": "field"}`
	if err := os.WriteFile(settingsPath, []byte(modifiedContent), 0600); err != nil {
		t.Fatalf("Failed to modify original file: %v", err)
	}

	// Call ensureBackup again - should NOT update backup
	err := ensureBackup(settingsPath)
	if err != nil {
		t.Fatalf("ensureBackup after modification failed: %v", err)
	}

	// Verify backup still has original content
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("Failed to read backup: %v", err)
	}

	if string(backupContent) != originalContent {
		t.Errorf("Backup was updated when it shouldn't have been.\nGot: %s\nWant: %s",
			backupContent, originalContent)
	}
}

func TestEnsureBackup_IntegrationWithSetPluginEnabled(t *testing.T) {
	// Integration test: verify backup is created when using SetPluginEnabled
	tmpDir := t.TempDir()

	// Override CLAUDE_CONFIG_DIR for testing
	cleanup := setEnvForTest(t, "CLAUDE_CONFIG_DIR", tmpDir)
	defer cleanup()

	// Create initial settings with various fields
	initialJSON := `{
  "permissions": {"allow": ["Read"]},
  "model": "opus",
  "enabledPlugins": {}
}`

	path, _ := ScopePath(ScopeUser, tmpDir)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(initialJSON), 0600); err != nil {
		t.Fatalf("Failed to write initial settings: %v", err)
	}

	// Use SetPluginEnabled which should trigger backup
	err := SetPluginEnabled("new-plugin@market", true, ScopeUser, tmpDir)
	if err != nil {
		t.Fatalf("SetPluginEnabled failed: %v", err)
	}

	// Verify backup was created
	backupPath := path + ".backup-plum"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Fatal("Backup was not created during SetPluginEnabled")
	}

	// Verify backup has original content (before modification)
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("Failed to read backup: %v", err)
	}

	if string(backupContent) != initialJSON {
		t.Errorf("Backup content doesn't match original.\nGot: %s\nWant: %s",
			backupContent, initialJSON)
	}
}
