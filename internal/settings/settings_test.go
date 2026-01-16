package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewSettings(t *testing.T) {
	s := NewSettings()

	if s.EnabledPlugins == nil {
		t.Error("EnabledPlugins should not be nil")
	}
	if s.ExtraKnownMarketplaces == nil {
		t.Error("ExtraKnownMarketplaces should not be nil")
	}
	if len(s.EnabledPlugins) != 0 {
		t.Error("EnabledPlugins should be empty")
	}
	if len(s.ExtraKnownMarketplaces) != 0 {
		t.Error("ExtraKnownMarketplaces should be empty")
	}
}

func TestLoadSettingsFromPath_NonExistent(t *testing.T) {
	// Loading non-existent file should return empty settings, not error
	settings, err := LoadSettingsFromPath("/nonexistent/path/settings.json")
	if err != nil {
		t.Fatalf("expected no error for non-existent file, got %v", err)
	}

	if settings == nil {
		t.Fatal("expected non-nil settings")
	}
	if len(settings.EnabledPlugins) != 0 {
		t.Error("expected empty EnabledPlugins")
	}
}

func TestLoadSettingsFromPath_Valid(t *testing.T) {
	// Create a temporary settings file
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	content := `{
		"enabledPlugins": {
			"plugin1@marketplace1": true,
			"plugin2@marketplace1": false
		},
		"extraKnownMarketplaces": {
			"custom": {
				"source": {
					"source": "github",
					"repo": "owner/repo"
				}
			}
		}
	}`

	if err := os.WriteFile(settingsPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	settings, err := LoadSettingsFromPath(settingsPath)
	if err != nil {
		t.Fatalf("failed to load settings: %v", err)
	}

	// Check enabled plugins
	if len(settings.EnabledPlugins) != 2 {
		t.Errorf("expected 2 enabled plugins, got %d", len(settings.EnabledPlugins))
	}
	if !settings.EnabledPlugins["plugin1@marketplace1"] {
		t.Error("plugin1 should be enabled")
	}
	if settings.EnabledPlugins["plugin2@marketplace1"] {
		t.Error("plugin2 should be disabled")
	}

	// Check marketplaces
	if len(settings.ExtraKnownMarketplaces) != 1 {
		t.Errorf("expected 1 marketplace, got %d", len(settings.ExtraKnownMarketplaces))
	}
	if mp, ok := settings.ExtraKnownMarketplaces["custom"]; !ok {
		t.Error("expected custom marketplace")
	} else if mp.Source.Repo != "owner/repo" {
		t.Errorf("expected repo owner/repo, got %s", mp.Source.Repo)
	}
}

func TestLoadSettingsFromPath_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	if err := os.WriteFile(settingsPath, []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadSettingsFromPath(settingsPath)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadSettingsFromPath_EmptyMaps(t *testing.T) {
	// Test that empty/missing maps are initialized
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	if err := os.WriteFile(settingsPath, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	settings, err := LoadSettingsFromPath(settingsPath)
	if err != nil {
		t.Fatalf("failed to load settings: %v", err)
	}

	if settings.EnabledPlugins == nil {
		t.Error("EnabledPlugins should be initialized")
	}
	if settings.ExtraKnownMarketplaces == nil {
		t.Error("ExtraKnownMarketplaces should be initialized")
	}
}

func TestFilterByScope(t *testing.T) {
	states := []PluginState{
		{FullName: "p1@m", Enabled: true, Scope: ScopeUser},
		{FullName: "p2@m", Enabled: true, Scope: ScopeProject},
		{FullName: "p3@m", Enabled: false, Scope: ScopeUser},
		{FullName: "p4@m", Enabled: true, Scope: ScopeLocal},
	}

	userStates := FilterByScope(states, ScopeUser)
	if len(userStates) != 2 {
		t.Errorf("expected 2 user scope states, got %d", len(userStates))
	}

	projectStates := FilterByScope(states, ScopeProject)
	if len(projectStates) != 1 {
		t.Errorf("expected 1 project scope state, got %d", len(projectStates))
	}

	managedStates := FilterByScope(states, ScopeManaged)
	if len(managedStates) != 0 {
		t.Errorf("expected 0 managed scope states, got %d", len(managedStates))
	}
}

func TestFilterEnabled(t *testing.T) {
	states := []PluginState{
		{FullName: "p1@m", Enabled: true, Scope: ScopeUser},
		{FullName: "p2@m", Enabled: false, Scope: ScopeUser},
		{FullName: "p3@m", Enabled: true, Scope: ScopeProject},
	}

	enabled := FilterEnabled(states)
	if len(enabled) != 2 {
		t.Errorf("expected 2 enabled states, got %d", len(enabled))
	}

	for _, state := range enabled {
		if !state.Enabled {
			t.Error("FilterEnabled returned disabled state")
		}
	}
}

func TestFilterDisabled(t *testing.T) {
	states := []PluginState{
		{FullName: "p1@m", Enabled: true, Scope: ScopeUser},
		{FullName: "p2@m", Enabled: false, Scope: ScopeUser},
		{FullName: "p3@m", Enabled: false, Scope: ScopeProject},
	}

	disabled := FilterDisabled(states)
	if len(disabled) != 2 {
		t.Errorf("expected 2 disabled states, got %d", len(disabled))
	}

	for _, state := range disabled {
		if state.Enabled {
			t.Error("FilterDisabled returned enabled state")
		}
	}
}

func TestMergedPluginStates(t *testing.T) {
	// Create temp directories with settings files
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	claudeDir := filepath.Join(projectDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Override CLAUDE_CONFIG_DIR to isolate from real user settings
	userClaudeDir := filepath.Join(tmpDir, "user-claude")
	if err := os.MkdirAll(userClaudeDir, 0750); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", userClaudeDir)

	// Create project settings (lower precedence)
	projectSettings := `{
		"enabledPlugins": {
			"plugin1@market": true,
			"plugin2@market": false
		}
	}`
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(projectSettings), 0600); err != nil {
		t.Fatal(err)
	}

	// Create local settings (higher precedence) - should override plugin1
	localSettings := `{
		"enabledPlugins": {
			"plugin1@market": false,
			"plugin3@market": true
		}
	}`
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte(localSettings), 0600); err != nil {
		t.Fatal(err)
	}

	states, err := MergedPluginStates(projectDir)
	if err != nil {
		t.Fatalf("MergedPluginStates error = %v", err)
	}

	// Should have 3 unique plugins
	if len(states) != 3 {
		t.Errorf("expected 3 states, got %d", len(states))
	}

	// Check that local scope wins for plugin1
	for _, state := range states {
		if state.FullName == "plugin1@market" {
			if state.Enabled {
				t.Error("plugin1 should be disabled (local scope override)")
			}
			if state.Scope != ScopeLocal {
				t.Errorf("plugin1 scope should be local, got %s", state.Scope)
			}
		}
	}
}

func TestGetPluginState(t *testing.T) {
	// Create temp project with settings
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	claudeDir := filepath.Join(projectDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Create project settings
	projectSettings := `{
		"enabledPlugins": {
			"plugin1@market": true
		}
	}`
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(projectSettings), 0600); err != nil {
		t.Fatal(err)
	}

	// Test finding existing plugin
	state, err := GetPluginState("plugin1@market", projectDir)
	if err != nil {
		t.Fatalf("GetPluginState error = %v", err)
	}
	if state == nil {
		t.Fatal("expected non-nil state")
	}
	if !state.Enabled {
		t.Error("expected plugin to be enabled")
	}
	if state.Scope != ScopeProject {
		t.Errorf("expected project scope, got %s", state.Scope)
	}

	// Test non-existent plugin
	state, err = GetPluginState("nonexistent@market", projectDir)
	if err != nil {
		t.Fatalf("GetPluginState error = %v", err)
	}
	if state != nil {
		t.Error("expected nil state for non-existent plugin")
	}
}

func TestAllMarketplaces(t *testing.T) {
	// Create temp project with settings
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	claudeDir := filepath.Join(projectDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Create project settings with marketplace
	projectSettings := `{
		"extraKnownMarketplaces": {
			"team-plugins": {
				"source": {
					"source": "github",
					"repo": "team/plugins"
				}
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(projectSettings), 0600); err != nil {
		t.Fatal(err)
	}

	marketplaces, err := AllMarketplaces(projectDir)
	if err != nil {
		t.Fatalf("AllMarketplaces error = %v", err)
	}

	if len(marketplaces) != 1 {
		t.Errorf("expected 1 marketplace, got %d", len(marketplaces))
	}

	if mp, ok := marketplaces["team-plugins"]; !ok {
		t.Error("expected team-plugins marketplace")
	} else if mp.Source.Repo != "team/plugins" {
		t.Errorf("expected repo team/plugins, got %s", mp.Source.Repo)
	}
}
