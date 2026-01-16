package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestListCommand_Structure(t *testing.T) {
	// Verify list command is registered
	cmd, _, err := rootCmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("list command not found: %v", err)
	}

	if cmd.Use != "list" {
		t.Errorf("expected Use 'list', got %s", cmd.Use)
	}

	// Check flags exist
	flags := []string{"scope", "enabled", "disabled", "json", "project"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag --%s to exist", flag)
		}
	}
}

func TestListCommand_ScopeFlag(t *testing.T) {
	// Test that scope flag accepts valid values
	cmd, _, _ := rootCmd.Find([]string{"list"})
	scopeFlag := cmd.Flags().Lookup("scope")

	if scopeFlag == nil {
		t.Fatal("scope flag not found")
	}

	if scopeFlag.Shorthand != "s" {
		t.Errorf("expected scope shorthand 's', got %s", scopeFlag.Shorthand)
	}
}

func TestListCommand_JSONOutput(t *testing.T) {
	// Create isolated test environment
	tmpDir := t.TempDir()

	// Set up isolated Claude config
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0750); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", claudeDir)

	// Create empty known_marketplaces.json
	if err := os.WriteFile(
		filepath.Join(pluginsDir, "known_marketplaces.json"),
		[]byte("{}"),
		0600,
	); err != nil {
		t.Fatal(err)
	}

	// Create user settings with a plugin
	userSettings := `{
		"enabledPlugins": {
			"test-plugin@test-market": true
		}
	}`
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(userSettings), 0600); err != nil {
		t.Fatal(err)
	}

	// Reset flags to default
	listJSON = true
	listScope = ""
	listEnabled = false
	listDisabled = false
	listProject = ""

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runList(listCmd, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runList failed: %v", err)
	}

	// Read output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Parse JSON output
	var items []PluginListItem
	if err := json.Unmarshal([]byte(output), &items); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, output)
	}

	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}

	if len(items) > 0 {
		if items[0].Name != "test-plugin" {
			t.Errorf("expected name 'test-plugin', got %s", items[0].Name)
		}
		if items[0].Marketplace != "test-market" {
			t.Errorf("expected marketplace 'test-market', got %s", items[0].Marketplace)
		}
		if items[0].Status != "enabled" {
			t.Errorf("expected status 'enabled', got %s", items[0].Status)
		}
	}

	// Reset
	listJSON = false
}

func TestPluginListItem_JSONSerialization(t *testing.T) {
	item := PluginListItem{
		Name:        "test-plugin",
		Marketplace: "test-market",
		Scope:       "user",
		Status:      "enabled",
		Version:     "1.0.0",
		Installed:   true,
	}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}

	var parsed PluginListItem
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}

	if parsed.Name != item.Name {
		t.Errorf("name mismatch: %s != %s", parsed.Name, item.Name)
	}
	if parsed.Version != item.Version {
		t.Errorf("version mismatch: %s != %s", parsed.Version, item.Version)
	}
}
