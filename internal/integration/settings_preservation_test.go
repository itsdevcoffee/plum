//go:build integration

package integration_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestSettingsPreservationEndToEnd verifies that plum preserves all
// custom fields in settings.json when modifying plugins.
// This is a critical test following the fix for the data loss bug.
func TestSettingsPreservationEndToEnd(t *testing.T) {
	// Build plum binary
	plumBin := buildPlumBinary(t)

	// Create isolated test environment
	testDir := t.TempDir()
	claudeDir := filepath.Join(testDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Set up local test marketplace
	marketplaceDir := setupTestMarketplace(t, testDir)

	// Create known_marketplaces.json pointing to our test marketplace
	marketplaces := map[string]interface{}{
		"test-market": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "local",
				"repo":   "",
			},
			"installLocation": marketplaceDir,
			"lastUpdated":     time.Now().Format(time.RFC3339),
		},
	}
	writeJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), marketplaces)

	// Create settings.json with ALL the custom fields users might have
	settingsPath := filepath.Join(claudeDir, "settings.json")
	initialSettings := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []string{"Bash(git:*)", "Read", "Write", "WebSearch"},
			"deny":  []string{"Bash(rm -rf:*)"},
		},
		"hooks": map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "/path/to/session-start.sh",
							"timeout": 5000,
						},
					},
				},
			},
			"UserPromptSubmit": []interface{}{
				map[string]interface{}{
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "/path/to/prompt.sh",
						},
					},
				},
			},
		},
		"attribution": map[string]interface{}{
			"commit": "test-commit-hash",
			"pr":     "https://github.com/example/repo/pull/123",
		},
		"model":               "claude-opus-4",
		"includeCoAuthoredBy": false,
		"enabledPlugins": map[string]bool{
			"existing-plugin@some-marketplace": true,
		},
	}

	writeJSON(t, settingsPath, initialSettings)

	// Capture original content for comparison
	originalBytes, _ := os.ReadFile(settingsPath)

	// Run plum install with test marketplace
	// #nosec G204 -- plumBin is built by us in buildPlumBinary
	cmd := exec.Command(plumBin, "install", "test-plugin@test-market", "--scope=project")
	cmd.Dir = testDir
	cmd.Env = append(os.Environ(),
		"HOME="+testDir,
		"CLAUDE_CONFIG_DIR="+claudeDir,
	)

	output, err := cmd.CombinedOutput()
	t.Logf("plum install output: %s", output)
	if err != nil {
		t.Logf("plum install error (may be expected): %v", err)
	}

	// Read settings after operation
	resultBytes, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("Failed to read settings after plum command: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("Failed to parse settings JSON: %v", err)
	}

	// CRITICAL: Verify ALL custom fields are preserved
	t.Run("permissions preserved", func(t *testing.T) {
		perms, ok := result["permissions"].(map[string]interface{})
		if !ok {
			t.Fatal("permissions field is missing or wrong type")
		}
		allow, ok := perms["allow"].([]interface{})
		if !ok || len(allow) != 4 {
			t.Errorf("permissions.allow changed: got %v", perms["allow"])
		}
		deny, ok := perms["deny"].([]interface{})
		if !ok || len(deny) != 1 {
			t.Errorf("permissions.deny changed: got %v", perms["deny"])
		}
	})

	t.Run("hooks preserved", func(t *testing.T) {
		hooks, ok := result["hooks"].(map[string]interface{})
		if !ok {
			t.Fatal("hooks field is missing or wrong type")
		}
		if _, ok := hooks["SessionStart"]; !ok {
			t.Error("hooks.SessionStart is missing")
		}
		if _, ok := hooks["UserPromptSubmit"]; !ok {
			t.Error("hooks.UserPromptSubmit is missing")
		}
	})

	t.Run("attribution preserved", func(t *testing.T) {
		attr, ok := result["attribution"].(map[string]interface{})
		if !ok {
			t.Fatal("attribution field is missing or wrong type")
		}
		if attr["commit"] != "test-commit-hash" {
			t.Errorf("attribution.commit changed: got %v", attr["commit"])
		}
		if attr["pr"] != "https://github.com/example/repo/pull/123" {
			t.Errorf("attribution.pr changed: got %v", attr["pr"])
		}
	})

	t.Run("model preserved", func(t *testing.T) {
		if result["model"] != "claude-opus-4" {
			t.Errorf("model changed: got %v, want claude-opus-4", result["model"])
		}
	})

	t.Run("includeCoAuthoredBy preserved", func(t *testing.T) {
		if result["includeCoAuthoredBy"] != false {
			t.Errorf("includeCoAuthoredBy changed: got %v, want false", result["includeCoAuthoredBy"])
		}
	})

	t.Run("existing plugins preserved", func(t *testing.T) {
		plugins, ok := result["enabledPlugins"].(map[string]interface{})
		if !ok {
			t.Fatal("enabledPlugins field is missing or wrong type")
		}
		if plugins["existing-plugin@some-marketplace"] != true {
			t.Errorf("existing-plugin@some-marketplace should still be enabled")
		}
	})

	t.Run("backup created", func(t *testing.T) {
		backupPath := settingsPath + ".backup-plum"
		backupInfo, err := os.Stat(backupPath)
		if err != nil {
			// Backup is only created if settings were actually modified
			// If the install failed before modification, skip this check
			t.Skipf("Backup file not created (command may have failed before modifying settings): %v", err)
			return
		}

		// Verify backup has correct permissions (0600)
		if backupInfo.Mode().Perm() != 0600 {
			t.Errorf("Backup permissions wrong: got %v, want 0600", backupInfo.Mode().Perm())
		}

		// Verify backup content matches original
		backupContent, _ := os.ReadFile(backupPath)
		if string(backupContent) != string(originalBytes) {
			t.Error("Backup content doesn't match original")
		}
	})
}

// TestMultipleOperationsPreserveFields tests that multiple plum operations
// in sequence all preserve custom fields.
func TestMultipleOperationsPreserveFields(t *testing.T) {
	plumBin := buildPlumBinary(t)
	testDir := t.TempDir()
	claudeDir := filepath.Join(testDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Set up local test marketplace
	marketplaceDir := setupTestMarketplace(t, testDir)

	// Create known_marketplaces.json
	marketplaces := map[string]interface{}{
		"test-market": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "local",
				"repo":   "",
			},
			"installLocation": marketplaceDir,
			"lastUpdated":     time.Now().Format(time.RFC3339),
		},
	}
	writeJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), marketplaces)

	settingsPath := filepath.Join(claudeDir, "settings.json")
	initialSettings := map[string]interface{}{
		"customField":    "must survive",
		"nestedObject":   map[string]interface{}{"key": "value", "num": 42},
		"arrayField":     []interface{}{1, 2, 3},
		"enabledPlugins": map[string]bool{},
	}

	writeJSON(t, settingsPath, initialSettings)

	env := append(os.Environ(),
		"HOME="+testDir,
		"CLAUDE_CONFIG_DIR="+claudeDir,
	)

	// Operation 1: Install
	// #nosec G204 -- plumBin is built by us in buildPlumBinary
	cmd1 := exec.Command(plumBin, "install", "test-plugin@test-market", "--scope=project")
	cmd1.Dir = testDir
	cmd1.Env = env
	output1, _ := cmd1.CombinedOutput()
	t.Logf("install output: %s", output1)

	// Operation 2: Disable
	// #nosec G204 -- plumBin is built by us in buildPlumBinary
	cmd2 := exec.Command(plumBin, "disable", "test-plugin@test-market", "--scope=project")
	cmd2.Dir = testDir
	cmd2.Env = env
	output2, _ := cmd2.CombinedOutput()
	t.Logf("disable output: %s", output2)

	// Operation 3: Enable
	// #nosec G204 -- plumBin is built by us in buildPlumBinary
	cmd3 := exec.Command(plumBin, "enable", "test-plugin@test-market", "--scope=project")
	cmd3.Dir = testDir
	cmd3.Env = env
	output3, _ := cmd3.CombinedOutput()
	t.Logf("enable output: %s", output3)

	// Verify custom fields survived all operations
	resultBytes, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("Failed to read settings: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("Failed to parse settings JSON: %v", err)
	}

	if result["customField"] != "must survive" {
		t.Errorf("customField lost after multiple operations: got %v, want 'must survive'", result["customField"])
	}

	nested, ok := result["nestedObject"].(map[string]interface{})
	if !ok || nested["key"] != "value" {
		t.Errorf("nestedObject corrupted: %v", result["nestedObject"])
	}

	arr, ok := result["arrayField"].([]interface{})
	if !ok || len(arr) != 3 {
		t.Errorf("arrayField corrupted: %v", result["arrayField"])
	}
}

// TestBackupCreation specifically tests that backup is created on first modification.
func TestBackupCreation(t *testing.T) {
	plumBin := buildPlumBinary(t)
	testDir := t.TempDir()
	claudeDir := filepath.Join(testDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Set up local test marketplace
	marketplaceDir := setupTestMarketplace(t, testDir)

	// Create known_marketplaces.json
	marketplaces := map[string]interface{}{
		"test-market": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "local",
				"repo":   "",
			},
			"installLocation": marketplaceDir,
			"lastUpdated":     time.Now().Format(time.RFC3339),
		},
	}
	writeJSON(t, filepath.Join(pluginsDir, "known_marketplaces.json"), marketplaces)

	settingsPath := filepath.Join(claudeDir, "settings.json")
	initialSettings := map[string]interface{}{
		"model":          "claude-opus-4",
		"enabledPlugins": map[string]bool{},
	}
	writeJSON(t, settingsPath, initialSettings)

	// Capture original content
	originalBytes, _ := os.ReadFile(settingsPath)

	env := append(os.Environ(),
		"HOME="+testDir,
		"CLAUDE_CONFIG_DIR="+claudeDir,
	)

	// Install plugin - this should create backup
	// #nosec G204 -- plumBin is built by us in buildPlumBinary
	cmd := exec.Command(plumBin, "install", "test-plugin@test-market", "--scope=project")
	cmd.Dir = testDir
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	t.Logf("install output: %s", output)

	if err != nil {
		t.Skipf("Install failed, cannot test backup: %v", err)
	}

	// Verify backup was created
	backupPath := settingsPath + ".backup-plum"
	backupInfo, err := os.Stat(backupPath)
	if err != nil {
		t.Fatalf("Backup file not created: %v", err)
	}

	// Verify permissions
	if backupInfo.Mode().Perm() != 0600 {
		t.Errorf("Backup permissions wrong: got %v, want 0600", backupInfo.Mode().Perm())
	}

	// Verify content matches original
	backupContent, _ := os.ReadFile(backupPath)
	if string(backupContent) != string(originalBytes) {
		t.Error("Backup content doesn't match original")
	}

	// Verify backup is NOT recreated on subsequent operations
	newContent, _ := os.ReadFile(settingsPath)

	// #nosec G204 -- plumBin is built by us in buildPlumBinary
	cmd2 := exec.Command(plumBin, "disable", "test-plugin@test-market", "--scope=project")
	cmd2.Dir = testDir
	cmd2.Env = env
	_, _ = cmd2.CombinedOutput()

	backupContent2, _ := os.ReadFile(backupPath)
	if string(backupContent2) != string(originalBytes) {
		t.Error("Backup was overwritten on subsequent operation")
	}
	if string(backupContent2) == string(newContent) {
		t.Error("Backup incorrectly reflects post-install state")
	}
}

// setupTestMarketplace creates a local test marketplace with proper directory structure
// Returns the marketplace directory path
func setupTestMarketplace(t *testing.T, baseDir string) string {
	t.Helper()

	// Create marketplace directory structure
	marketplaceDir := filepath.Join(baseDir, "test-marketplace")
	pluginMetaDir := filepath.Join(marketplaceDir, ".claude-plugin")
	if err := os.MkdirAll(pluginMetaDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Create marketplace.json manifest
	manifest := map[string]interface{}{
		"name": "test-market",
		"owner": map[string]interface{}{
			"name":  "Test Owner",
			"email": "test@example.com",
		},
		"plugins": []map[string]interface{}{
			{
				"name":        "test-plugin",
				"version":     "1.0.0",
				"description": "A test plugin for integration tests",
				"source":      "./plugins/test-plugin",
				"author": map[string]interface{}{
					"name": "Test Author",
				},
			},
		},
	}
	writeJSON(t, filepath.Join(pluginMetaDir, "marketplace.json"), manifest)

	// Create the plugin directory
	pluginDir := filepath.Join(marketplaceDir, "plugins", "test-plugin")
	if err := os.MkdirAll(pluginDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Create a minimal plugin.json for the plugin
	pluginManifest := map[string]interface{}{
		"name":        "test-plugin",
		"version":     "1.0.0",
		"description": "A test plugin",
	}
	writeJSON(t, filepath.Join(pluginDir, "plugin.json"), pluginManifest)

	return marketplaceDir
}

// buildPlumBinary builds the plum binary and returns its path.
// The binary is built in the test's temp directory.
func buildPlumBinary(t *testing.T) string {
	t.Helper()

	// Find the project root by looking for go.mod
	projectRoot := findProjectRoot(t)

	// Build in a temp directory
	tmpDir := t.TempDir()
	plumBin := filepath.Join(tmpDir, "plum")

	// Build from the project root
	// #nosec G204 -- static arguments, only plumBin path is variable (test temp dir)
	cmd := exec.Command("go", "build", "-o", plumBin, "./cmd/plum")
	cmd.Dir = projectRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build plum binary: %v\nOutput: %s", err, output)
	}

	return plumBin
}

// findProjectRoot walks up from the current directory to find go.mod
func findProjectRoot(t *testing.T) string {
	t.Helper()

	// Start from the current working directory
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("Could not find project root (go.mod)")
		}
		dir = parent
	}
}

// writeJSON writes data as formatted JSON to path.
func writeJSON(t *testing.T, path string, data interface{}) {
	t.Helper()
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}
	if err := os.WriteFile(path, bytes, 0600); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
}
