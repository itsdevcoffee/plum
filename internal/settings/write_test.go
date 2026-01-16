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
