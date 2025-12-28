package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadKnownMarketplaces(t *testing.T) {
	t.Run("valid known_marketplaces.json", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("CLAUDE_CONFIG_DIR", tmpDir)

		// Create test file
		pluginsDir := filepath.Join(tmpDir, "plugins")
		if err := os.MkdirAll(pluginsDir, 0755); err != nil {
			t.Fatal(err)
		}

		marketplacesFile := filepath.Join(pluginsDir, "known_marketplaces.json")
		testData := `{
			"test-marketplace": {
				"source": {
					"source": "github",
					"repo": "owner/repo"
				},
				"installLocation": "/path/to/marketplace",
				"lastUpdated": "2025-12-17T00:00:00.000Z"
			}
		}`

		if err := os.WriteFile(marketplacesFile, []byte(testData), 0644); err != nil {
			t.Fatal(err)
		}

		marketplaces, err := LoadKnownMarketplaces()
		if err != nil {
			t.Fatalf("LoadKnownMarketplaces() error = %v", err)
		}

		if len(marketplaces) != 1 {
			t.Errorf("LoadKnownMarketplaces() returned %d marketplaces, want 1", len(marketplaces))
		}

		entry, ok := marketplaces["test-marketplace"]
		if !ok {
			t.Fatal("test-marketplace not found in loaded marketplaces")
		}

		if entry.Source.Source != "github" {
			t.Errorf("Source.Source = %q, want %q", entry.Source.Source, "github")
		}
		if entry.Source.Repo != "owner/repo" {
			t.Errorf("Source.Repo = %q, want %q", entry.Source.Repo, "owner/repo")
		}
		if entry.InstallLocation != "/path/to/marketplace" {
			t.Errorf("InstallLocation = %q, want %q", entry.InstallLocation, "/path/to/marketplace")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		// Use empty temp dir (no file)
		emptyDir := t.TempDir()
		t.Setenv("CLAUDE_CONFIG_DIR", emptyDir)

		_, err := LoadKnownMarketplaces()
		if err == nil {
			t.Error("LoadKnownMarketplaces() expected error for missing file, got nil")
		}

		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("LoadKnownMarketplaces() error = %q, want error containing 'not found'", err.Error())
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("CLAUDE_CONFIG_DIR", tmpDir)

		// Create invalid JSON file
		pluginsDir := filepath.Join(tmpDir, "plugins")
		if err := os.MkdirAll(pluginsDir, 0755); err != nil {
			t.Fatal(err)
		}

		marketplacesFile := filepath.Join(pluginsDir, "known_marketplaces.json")
		if err := os.WriteFile(marketplacesFile, []byte("invalid json {"), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := LoadKnownMarketplaces()
		if err == nil {
			t.Error("LoadKnownMarketplaces() expected error for invalid JSON, got nil")
		}
	})

	t.Run("empty JSON object", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("CLAUDE_CONFIG_DIR", tmpDir)

		pluginsDir := filepath.Join(tmpDir, "plugins")
		if err := os.MkdirAll(pluginsDir, 0755); err != nil {
			t.Fatal(err)
		}

		marketplacesFile := filepath.Join(pluginsDir, "known_marketplaces.json")
		if err := os.WriteFile(marketplacesFile, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}

		marketplaces, err := LoadKnownMarketplaces()
		if err != nil {
			t.Fatalf("LoadKnownMarketplaces() error = %v", err)
		}

		if len(marketplaces) != 0 {
			t.Errorf("LoadKnownMarketplaces() returned %d marketplaces, want 0", len(marketplaces))
		}
	})
}

func TestLoadInstalledPlugins(t *testing.T) {
	t.Run("valid installed_plugins_v2.json", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("CLAUDE_CONFIG_DIR", tmpDir)

		pluginsDir := filepath.Join(tmpDir, "plugins")
		if err := os.MkdirAll(pluginsDir, 0755); err != nil {
			t.Fatal(err)
		}

		installedFile := filepath.Join(pluginsDir, "installed_plugins_v2.json")
		testData := `{
			"version": 2,
			"plugins": {
				"test-plugin@test-marketplace": [{
					"scope": "global",
					"installPath": "/path/to/plugin",
					"version": "1.0.0",
					"installedAt": "2025-12-17T00:00:00.000Z",
					"lastUpdated": "2025-12-17T00:00:00.000Z",
					"gitCommitSha": "abc123",
					"isLocal": false
				}]
			}
		}`

		if err := os.WriteFile(installedFile, []byte(testData), 0644); err != nil {
			t.Fatal(err)
		}

		installed, err := LoadInstalledPlugins()
		if err != nil {
			t.Fatalf("LoadInstalledPlugins() error = %v", err)
		}

		if installed.Version != 2 {
			t.Errorf("Version = %d, want 2", installed.Version)
		}

		if len(installed.Plugins) != 1 {
			t.Errorf("Plugins count = %d, want 1", len(installed.Plugins))
		}

		pluginInstalls, ok := installed.Plugins["test-plugin@test-marketplace"]
		if !ok {
			t.Fatal("test-plugin@test-marketplace not found in plugins")
		}

		if len(pluginInstalls) != 1 {
			t.Fatalf("Plugin installs count = %d, want 1", len(pluginInstalls))
		}

		install := pluginInstalls[0]
		if install.Scope != "global" {
			t.Errorf("Scope = %q, want %q", install.Scope, "global")
		}
		if install.Version != "1.0.0" {
			t.Errorf("Version = %q, want %q", install.Version, "1.0.0")
		}
		if install.IsLocal {
			t.Error("IsLocal = true, want false")
		}
	})

	t.Run("file not found returns empty", func(t *testing.T) {
		emptyDir := t.TempDir()
		t.Setenv("CLAUDE_CONFIG_DIR", emptyDir)

		installed, err := LoadInstalledPlugins()
		if err != nil {
			t.Fatalf("LoadInstalledPlugins() error = %v, want nil", err)
		}

		if installed.Version != 2 {
			t.Errorf("Version = %d, want 2", installed.Version)
		}

		if len(installed.Plugins) != 0 {
			t.Errorf("Plugins count = %d, want 0", len(installed.Plugins))
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("CLAUDE_CONFIG_DIR", tmpDir)

		pluginsDir := filepath.Join(tmpDir, "plugins")
		if err := os.MkdirAll(pluginsDir, 0755); err != nil {
			t.Fatal(err)
		}

		installedFile := filepath.Join(pluginsDir, "installed_plugins_v2.json")
		if err := os.WriteFile(installedFile, []byte("invalid json {"), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := LoadInstalledPlugins()
		if err == nil {
			t.Error("LoadInstalledPlugins() expected error for invalid JSON, got nil")
		}
	})
}

func TestLoadMarketplaceManifest(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("valid marketplace.json", func(t *testing.T) {
		// Create marketplace directory structure
		marketplaceDir := filepath.Join(tmpDir, "test-marketplace")
		pluginDir := filepath.Join(marketplaceDir, ".claude-plugin")
		if err := os.MkdirAll(pluginDir, 0755); err != nil {
			t.Fatal(err)
		}

		manifestFile := filepath.Join(pluginDir, "marketplace.json")
		testData := `{
			"name": "test-marketplace",
			"owner": {
				"name": "Test Owner",
				"email": "test@example.com"
			},
			"plugins": [
				{
					"name": "test-plugin",
					"description": "A test plugin",
					"version": "1.0.0",
					"source": "./plugins/test-plugin",
					"author": {
						"name": "Plugin Author"
					}
				}
			]
		}`

		if err := os.WriteFile(manifestFile, []byte(testData), 0644); err != nil {
			t.Fatal(err)
		}

		manifest, err := LoadMarketplaceManifest(marketplaceDir)
		if err != nil {
			t.Fatalf("LoadMarketplaceManifest() error = %v", err)
		}

		if manifest.Name != "test-marketplace" {
			t.Errorf("Name = %q, want %q", manifest.Name, "test-marketplace")
		}

		if manifest.Owner.Name != "Test Owner" {
			t.Errorf("Owner.Name = %q, want %q", manifest.Owner.Name, "Test Owner")
		}

		if len(manifest.Plugins) != 1 {
			t.Fatalf("Plugins count = %d, want 1", len(manifest.Plugins))
		}

		plugin := manifest.Plugins[0]
		if plugin.Name != "test-plugin" {
			t.Errorf("Plugin.Name = %q, want %q", plugin.Name, "test-plugin")
		}
		if plugin.Version != "1.0.0" {
			t.Errorf("Plugin.Version = %q, want %q", plugin.Version, "1.0.0")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		nonexistentDir := filepath.Join(tmpDir, "nonexistent")

		_, err := LoadMarketplaceManifest(nonexistentDir)
		if err == nil {
			t.Error("LoadMarketplaceManifest() expected error for missing file, got nil")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		marketplaceDir := filepath.Join(tmpDir, "invalid-marketplace")
		pluginDir := filepath.Join(marketplaceDir, ".claude-plugin")
		if err := os.MkdirAll(pluginDir, 0755); err != nil {
			t.Fatal(err)
		}

		manifestFile := filepath.Join(pluginDir, "marketplace.json")
		if err := os.WriteFile(manifestFile, []byte("invalid json {"), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := LoadMarketplaceManifest(marketplaceDir)
		if err == nil {
			t.Error("LoadMarketplaceManifest() expected error for invalid JSON, got nil")
		}
	})
}

func TestMarketplaceEntryJSON(t *testing.T) {
	t.Run("marshal and unmarshal", func(t *testing.T) {
		original := MarketplaceEntry{
			Source: MarketplaceSource{
				Source: "github",
				Repo:   "owner/repo",
			},
			InstallLocation: "/path/to/marketplace",
			LastUpdated:     "2025-12-17T00:00:00.000Z",
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}

		var unmarshaled MarketplaceEntry
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}

		if unmarshaled.Source.Source != original.Source.Source {
			t.Errorf("Source.Source = %q, want %q", unmarshaled.Source.Source, original.Source.Source)
		}
		if unmarshaled.Source.Repo != original.Source.Repo {
			t.Errorf("Source.Repo = %q, want %q", unmarshaled.Source.Repo, original.Source.Repo)
		}
		if unmarshaled.InstallLocation != original.InstallLocation {
			t.Errorf("InstallLocation = %q, want %q", unmarshaled.InstallLocation, original.InstallLocation)
		}
	})
}

func TestPluginInstallJSON(t *testing.T) {
	t.Run("marshal and unmarshal with optional fields", func(t *testing.T) {
		original := PluginInstall{
			Scope:        "project",
			InstallPath:  "/path/to/plugin",
			Version:      "2.0.0",
			InstalledAt:  "2025-12-17T00:00:00.000Z",
			LastUpdated:  "2025-12-17T12:00:00.000Z",
			GitCommitSha: "def456",
			IsLocal:      true,
			ProjectPath:  "/path/to/project",
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}

		var unmarshaled PluginInstall
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}

		if unmarshaled.Scope != original.Scope {
			t.Errorf("Scope = %q, want %q", unmarshaled.Scope, original.Scope)
		}
		if unmarshaled.ProjectPath != original.ProjectPath {
			t.Errorf("ProjectPath = %q, want %q", unmarshaled.ProjectPath, original.ProjectPath)
		}
		if !unmarshaled.IsLocal {
			t.Error("IsLocal = false, want true")
		}
	})
}
