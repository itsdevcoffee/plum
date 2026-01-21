package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// setEnvForTest is a helper to set environment variables for testing
func setEnvForTest(t *testing.T, key, value string) func() {
	t.Helper()
	original := os.Getenv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("Failed to set %s: %v", key, err)
	}
	return func() {
		if err := os.Setenv(key, original); err != nil {
			t.Logf("Warning: failed to restore %s: %v", key, err)
		}
	}
}

func TestSetPluginEnabled(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		fullName    string
		enabled     bool
		scope       Scope
		projectPath string
		wantErr     bool
	}{
		{
			name:        "enable plugin in user scope",
			fullName:    "test-plugin@test-marketplace",
			enabled:     true,
			scope:       ScopeUser,
			projectPath: tmpDir,
			wantErr:     false,
		},
		{
			name:        "disable plugin in project scope",
			fullName:    "another-plugin@marketplace",
			enabled:     false,
			scope:       ScopeProject,
			projectPath: tmpDir,
			wantErr:     false,
		},
		{
			name:        "managed scope should fail",
			fullName:    "test-plugin@test-marketplace",
			enabled:     true,
			scope:       ScopeManaged,
			projectPath: tmpDir,
			wantErr:     true,
		},
	}

	// Override CLAUDE_CONFIG_DIR for testing
	cleanup := setEnvForTest(t, "CLAUDE_CONFIG_DIR", tmpDir)
	defer cleanup()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SetPluginEnabled(tt.fullName, tt.enabled, tt.scope, tt.projectPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("SetPluginEnabled() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify the plugin was set correctly
				settings, err := LoadSettings(tt.scope, tt.projectPath)
				if err != nil {
					t.Errorf("Failed to load settings: %v", err)
					return
				}

				if got, ok := settings.EnabledPlugins[tt.fullName]; !ok || got != tt.enabled {
					t.Errorf("Plugin state = %v, want %v", got, tt.enabled)
				}
			}
		})
	}
}

func TestRemovePluginFromScope(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Override CLAUDE_CONFIG_DIR for testing
	cleanup := setEnvForTest(t, "CLAUDE_CONFIG_DIR", tmpDir)
	defer cleanup()

	fullName := "test-plugin@test-marketplace"

	// First, add a plugin
	err := SetPluginEnabled(fullName, true, ScopeUser, tmpDir)
	if err != nil {
		t.Fatalf("Failed to set plugin enabled: %v", err)
	}

	// Verify it was added
	settings, err := LoadSettings(ScopeUser, tmpDir)
	if err != nil {
		t.Fatalf("Failed to load settings: %v", err)
	}
	if _, ok := settings.EnabledPlugins[fullName]; !ok {
		t.Fatal("Plugin was not added")
	}

	// Now remove it
	err = RemovePluginFromScope(fullName, ScopeUser, tmpDir)
	if err != nil {
		t.Fatalf("Failed to remove plugin: %v", err)
	}

	// Verify it was removed
	settings, err = LoadSettings(ScopeUser, tmpDir)
	if err != nil {
		t.Fatalf("Failed to load settings after removal: %v", err)
	}
	if _, ok := settings.EnabledPlugins[fullName]; ok {
		t.Error("Plugin was not removed")
	}
}

func TestSaveSettings(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Override CLAUDE_CONFIG_DIR for testing
	cleanup := setEnvForTest(t, "CLAUDE_CONFIG_DIR", tmpDir)
	defer cleanup()

	// Create settings to save
	s := NewSettings()
	s.EnabledPlugins["plugin1@marketplace1"] = true
	s.EnabledPlugins["plugin2@marketplace2"] = false

	// Save settings
	err := SaveSettings(s, ScopeUser, tmpDir)
	if err != nil {
		t.Fatalf("Failed to save settings: %v", err)
	}

	// Load and verify
	loaded, err := LoadSettings(ScopeUser, tmpDir)
	if err != nil {
		t.Fatalf("Failed to load settings: %v", err)
	}

	if len(loaded.EnabledPlugins) != 2 {
		t.Errorf("Expected 2 plugins, got %d", len(loaded.EnabledPlugins))
	}

	if !loaded.EnabledPlugins["plugin1@marketplace1"] {
		t.Error("plugin1@marketplace1 should be enabled")
	}

	if loaded.EnabledPlugins["plugin2@marketplace2"] {
		t.Error("plugin2@marketplace2 should be disabled")
	}
}

func TestSaveSettingsPreservesExistingData(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Override CLAUDE_CONFIG_DIR for testing
	cleanup := setEnvForTest(t, "CLAUDE_CONFIG_DIR", tmpDir)
	defer cleanup()

	// Create initial settings with one plugin
	initial := NewSettings()
	initial.EnabledPlugins["existing-plugin@market"] = true

	err := SaveSettings(initial, ScopeUser, tmpDir)
	if err != nil {
		t.Fatalf("Failed to save initial settings: %v", err)
	}

	// Save new settings with different plugin
	newSettings := NewSettings()
	newSettings.EnabledPlugins["new-plugin@market"] = true

	err = SaveSettings(newSettings, ScopeUser, tmpDir)
	if err != nil {
		t.Fatalf("Failed to save new settings: %v", err)
	}

	// Load and verify both plugins exist
	loaded, err := LoadSettings(ScopeUser, tmpDir)
	if err != nil {
		t.Fatalf("Failed to load settings: %v", err)
	}

	if len(loaded.EnabledPlugins) != 2 {
		t.Errorf("Expected 2 plugins (merged), got %d", len(loaded.EnabledPlugins))
	}

	if !loaded.EnabledPlugins["existing-plugin@market"] {
		t.Error("existing-plugin should still exist")
	}

	if !loaded.EnabledPlugins["new-plugin@market"] {
		t.Error("new-plugin should exist")
	}
}

func TestAddMarketplace(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Override CLAUDE_CONFIG_DIR for testing
	cleanup := setEnvForTest(t, "CLAUDE_CONFIG_DIR", tmpDir)
	defer cleanup()

	source := MarketplaceSource{
		Source: "github",
		Repo:   "owner/repo",
	}

	err := AddMarketplace("test-marketplace", source, ScopeUser, tmpDir)
	if err != nil {
		t.Fatalf("Failed to add marketplace: %v", err)
	}

	// Load and verify
	loaded, err := LoadSettings(ScopeUser, tmpDir)
	if err != nil {
		t.Fatalf("Failed to load settings: %v", err)
	}

	mp, ok := loaded.ExtraKnownMarketplaces["test-marketplace"]
	if !ok {
		t.Fatal("Marketplace was not added")
	}

	if mp.Source.Repo != "owner/repo" {
		t.Errorf("Expected repo 'owner/repo', got '%s'", mp.Source.Repo)
	}
}

func TestRemoveMarketplace(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Override CLAUDE_CONFIG_DIR for testing
	cleanup := setEnvForTest(t, "CLAUDE_CONFIG_DIR", tmpDir)
	defer cleanup()

	// Add a marketplace first
	source := MarketplaceSource{
		Source: "github",
		Repo:   "owner/repo",
	}
	err := AddMarketplace("test-marketplace", source, ScopeUser, tmpDir)
	if err != nil {
		t.Fatalf("Failed to add marketplace: %v", err)
	}

	// Remove it
	err = RemoveMarketplace("test-marketplace", ScopeUser, tmpDir)
	if err != nil {
		t.Fatalf("Failed to remove marketplace: %v", err)
	}

	// Verify it was removed
	loaded, err := LoadSettings(ScopeUser, tmpDir)
	if err != nil {
		t.Fatalf("Failed to load settings: %v", err)
	}

	if _, ok := loaded.ExtraKnownMarketplaces["test-marketplace"]; ok {
		t.Error("Marketplace was not removed")
	}
}

func TestSettingsFileFormat(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Override CLAUDE_CONFIG_DIR for testing
	cleanup := setEnvForTest(t, "CLAUDE_CONFIG_DIR", tmpDir)
	defer cleanup()

	// Create and save settings
	s := NewSettings()
	s.EnabledPlugins["test-plugin@market"] = true

	err := SaveSettings(s, ScopeUser, tmpDir)
	if err != nil {
		t.Fatalf("Failed to save settings: %v", err)
	}

	// Read raw file and verify JSON structure
	path, _ := ScopePath(ScopeUser, tmpDir)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read settings file: %v", err)
	}

	// Verify it's valid JSON
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Settings file is not valid JSON: %v", err)
	}

	// Verify structure
	if _, ok := raw["enabledPlugins"]; !ok {
		t.Error("Missing enabledPlugins key")
	}

	// Verify file ends with newline
	if data[len(data)-1] != '\n' {
		t.Error("Settings file should end with newline")
	}
}

func TestManagedScopeIsReadOnly(t *testing.T) {
	tests := []struct {
		name string
		fn   func() error
	}{
		{
			name: "SetPluginEnabled",
			fn: func() error {
				return SetPluginEnabled("test@market", true, ScopeManaged, "")
			},
		},
		{
			name: "RemovePluginFromScope",
			fn: func() error {
				return RemovePluginFromScope("test@market", ScopeManaged, "")
			},
		},
		{
			name: "AddMarketplace",
			fn: func() error {
				return AddMarketplace("test", MarketplaceSource{}, ScopeManaged, "")
			},
		},
		{
			name: "RemoveMarketplace",
			fn: func() error {
				return RemoveMarketplace("test", ScopeManaged, "")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err != ErrManagedReadOnly {
				t.Errorf("Expected ErrManagedReadOnly, got %v", err)
			}
		})
	}
}

func TestSaveSettingsCreatesDirectory(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Use a nested path that doesn't exist
	nestedPath := filepath.Join(tmpDir, "nested", "path")

	// Override CLAUDE_CONFIG_DIR for testing
	cleanup := setEnvForTest(t, "CLAUDE_CONFIG_DIR", nestedPath)
	defer cleanup()

	// Create and save settings - should create the directory
	s := NewSettings()
	s.EnabledPlugins["test@market"] = true

	err := SaveSettings(s, ScopeUser, nestedPath)
	if err != nil {
		t.Fatalf("Failed to save settings: %v", err)
	}

	// Verify directory was created
	path, _ := ScopePath(ScopeUser, nestedPath)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("Settings file was not created")
	}
}

func TestSaveSettingsPreservesUnknownTopLevelFields(t *testing.T) {
	// This is a CRITICAL test - it verifies that plum doesn't destroy user settings
	// that it doesn't manage (e.g., permissions, hooks, attribution, model, etc.)
	tmpDir := t.TempDir()

	// Override CLAUDE_CONFIG_DIR for testing
	cleanup := setEnvForTest(t, "CLAUDE_CONFIG_DIR", tmpDir)
	defer cleanup()

	// Create a settings.json with many fields that plum doesn't manage
	// This simulates a real user's settings.json with various configurations
	initialJSON := `{
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
  "attribution": {
    "commit": "abc123",
    "pr": "https://github.com/example/repo/pull/1"
  },
  "includeCoAuthoredBy": false,
  "model": "claude-opus-4",
  "enabledPlugins": {
    "existing-plugin@market": true,
    "another-plugin@market": false
  }
}`

	// Write the initial settings file
	path, _ := ScopePath(ScopeUser, tmpDir)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(initialJSON), 0600); err != nil {
		t.Fatalf("Failed to write initial settings: %v", err)
	}

	// Now use SetPluginEnabled to modify a plugin (this is what plum install does)
	err := SetPluginEnabled("new-plugin@market", true, ScopeUser, tmpDir)
	if err != nil {
		t.Fatalf("SetPluginEnabled failed: %v", err)
	}

	// Read the raw file and verify ALL fields are preserved
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read settings file: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to parse settings JSON: %v", err)
	}

	// Verify permissions field is preserved with correct structure
	permissions, ok := result["permissions"].(map[string]any)
	if !ok {
		t.Fatal("permissions field was lost or corrupted")
	}
	allowList, ok := permissions["allow"].([]any)
	if !ok || len(allowList) != 4 {
		t.Errorf("permissions.allow was corrupted: %v", permissions["allow"])
	}

	// Verify hooks field is preserved
	hooks, ok := result["hooks"].(map[string]any)
	if !ok {
		t.Fatal("hooks field was lost")
	}
	if _, ok := hooks["SessionStart"]; !ok {
		t.Error("hooks.SessionStart was lost")
	}

	// Verify attribution field is preserved
	attribution, ok := result["attribution"].(map[string]any)
	if !ok {
		t.Fatal("attribution field was lost")
	}
	if attribution["commit"] != "abc123" {
		t.Errorf("attribution.commit was corrupted: %v", attribution["commit"])
	}
	if attribution["pr"] != "https://github.com/example/repo/pull/1" {
		t.Errorf("attribution.pr was corrupted: %v", attribution["pr"])
	}

	// Verify includeCoAuthoredBy field is preserved
	includeCoAuthoredBy, ok := result["includeCoAuthoredBy"].(bool)
	if !ok {
		t.Fatal("includeCoAuthoredBy field was lost")
	}
	if includeCoAuthoredBy != false {
		t.Error("includeCoAuthoredBy value was corrupted")
	}

	// Verify model field is preserved
	model, ok := result["model"].(string)
	if !ok {
		t.Fatal("model field was lost")
	}
	if model != "claude-opus-4" {
		t.Errorf("model was corrupted: %v", model)
	}

	// Verify enabledPlugins contains both old and new plugins
	plugins, ok := result["enabledPlugins"].(map[string]any)
	if !ok {
		t.Fatal("enabledPlugins field was lost")
	}
	if plugins["existing-plugin@market"] != true {
		t.Error("existing-plugin@market was lost or changed")
	}
	if plugins["another-plugin@market"] != false {
		t.Error("another-plugin@market was lost or changed")
	}
	if plugins["new-plugin@market"] != true {
		t.Error("new-plugin@market was not added")
	}

	// Count total top-level fields - should be 6
	expectedFields := []string{"permissions", "hooks", "attribution", "includeCoAuthoredBy", "model", "enabledPlugins"}
	for _, field := range expectedFields {
		if _, ok := result[field]; !ok {
			t.Errorf("Field %q was lost", field)
		}
	}
}

func TestSetPluginEnabledPreservesUnknownFields(t *testing.T) {
	// Test SetPluginEnabled specifically as it's the most common operation
	tmpDir := t.TempDir()

	cleanup := setEnvForTest(t, "CLAUDE_CONFIG_DIR", tmpDir)
	defer cleanup()

	// Create settings with unknown fields
	initialJSON := `{
  "customField": "custom value",
  "nestedObject": {"key": "value", "number": 42},
  "arrayField": [1, 2, 3],
  "enabledPlugins": {"old@market": true}
}`

	path, _ := ScopePath(ScopeUser, tmpDir)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(initialJSON), 0600); err != nil {
		t.Fatalf("Failed to write initial settings: %v", err)
	}

	// Enable a new plugin
	if err := SetPluginEnabled("new@market", true, ScopeUser, tmpDir); err != nil {
		t.Fatalf("SetPluginEnabled failed: %v", err)
	}

	// Read and verify
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read settings: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify all fields are preserved
	if result["customField"] != "custom value" {
		t.Errorf("customField lost: %v", result["customField"])
	}
	nested, ok := result["nestedObject"].(map[string]any)
	if !ok || nested["key"] != "value" || nested["number"] != float64(42) {
		t.Errorf("nestedObject corrupted: %v", result["nestedObject"])
	}
	arr, ok := result["arrayField"].([]any)
	if !ok || len(arr) != 3 {
		t.Errorf("arrayField corrupted: %v", result["arrayField"])
	}

	plugins := result["enabledPlugins"].(map[string]any)
	if plugins["old@market"] != true {
		t.Error("old plugin was lost")
	}
	if plugins["new@market"] != true {
		t.Error("new plugin was not added")
	}
}

func TestRemovePluginFromScopePreservesUnknownFields(t *testing.T) {
	tmpDir := t.TempDir()

	cleanup := setEnvForTest(t, "CLAUDE_CONFIG_DIR", tmpDir)
	defer cleanup()

	initialJSON := `{
  "permissions": {"allow": ["Read"]},
  "enabledPlugins": {"plugin-to-remove@market": true, "keep-this@market": false}
}`

	path, _ := ScopePath(ScopeUser, tmpDir)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(initialJSON), 0600); err != nil {
		t.Fatalf("Failed to write initial settings: %v", err)
	}

	// Remove a plugin
	if err := RemovePluginFromScope("plugin-to-remove@market", ScopeUser, tmpDir); err != nil {
		t.Fatalf("RemovePluginFromScope failed: %v", err)
	}

	// Read and verify
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read settings: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify permissions field is preserved
	if _, ok := result["permissions"]; !ok {
		t.Fatal("permissions field was lost")
	}

	plugins := result["enabledPlugins"].(map[string]any)
	if _, ok := plugins["plugin-to-remove@market"]; ok {
		t.Error("plugin-to-remove@market should have been removed")
	}
	if plugins["keep-this@market"] != false {
		t.Error("keep-this@market was corrupted")
	}
}

func TestAddMarketplacePreservesUnknownFields(t *testing.T) {
	tmpDir := t.TempDir()

	cleanup := setEnvForTest(t, "CLAUDE_CONFIG_DIR", tmpDir)
	defer cleanup()

	initialJSON := `{
  "model": "claude-sonnet-4",
  "enabledPlugins": {"existing@market": true}
}`

	path, _ := ScopePath(ScopeUser, tmpDir)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(initialJSON), 0600); err != nil {
		t.Fatalf("Failed to write initial settings: %v", err)
	}

	// Add a marketplace
	source := MarketplaceSource{Source: "github", Repo: "owner/repo"}
	if err := AddMarketplace("new-marketplace", source, ScopeUser, tmpDir); err != nil {
		t.Fatalf("AddMarketplace failed: %v", err)
	}

	// Read and verify
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read settings: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify model field is preserved
	if result["model"] != "claude-sonnet-4" {
		t.Errorf("model field was corrupted: %v", result["model"])
	}

	// Verify plugins are preserved
	plugins := result["enabledPlugins"].(map[string]any)
	if plugins["existing@market"] != true {
		t.Error("existing plugin was lost")
	}

	// Verify marketplace was added
	marketplaces, ok := result["extraKnownMarketplaces"].(map[string]any)
	if !ok {
		t.Fatal("extraKnownMarketplaces not found")
	}
	if _, ok := marketplaces["new-marketplace"]; !ok {
		t.Error("new marketplace was not added")
	}
}
