package marketplace

import (
	"encoding/json"
	"testing"
)

func TestMarketplacePlugin_UnmarshalJSON_StringSource(t *testing.T) {
	jsonData := `{
		"name": "test-plugin",
		"source": "./plugins/test-plugin",
		"description": "A test plugin"
	}`

	var plugin MarketplacePlugin
	if err := json.Unmarshal([]byte(jsonData), &plugin); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if plugin.Name != "test-plugin" {
		t.Errorf("expected name 'test-plugin', got %q", plugin.Name)
	}
	if plugin.Source != "./plugins/test-plugin" {
		t.Errorf("expected source './plugins/test-plugin', got %q", plugin.Source)
	}
	if plugin.HasLSPServers {
		t.Error("expected HasLSPServers to be false")
	}
	if plugin.IsExternalURL {
		t.Error("expected IsExternalURL to be false")
	}
	if !plugin.Installable() {
		t.Error("expected plugin to be installable")
	}
}

func TestMarketplacePlugin_UnmarshalJSON_ExternalURLSource(t *testing.T) {
	jsonData := `{
		"name": "atlassian",
		"source": {
			"source": "url",
			"url": "https://github.com/atlassian/atlassian-mcp-server.git"
		},
		"description": "Atlassian integration"
	}`

	var plugin MarketplacePlugin
	if err := json.Unmarshal([]byte(jsonData), &plugin); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if plugin.Name != "atlassian" {
		t.Errorf("expected name 'atlassian', got %q", plugin.Name)
	}
	if plugin.Source != "https://github.com/atlassian/atlassian-mcp-server.git" {
		t.Errorf("expected source URL, got %q", plugin.Source)
	}
	if !plugin.IsExternalURL {
		t.Error("expected IsExternalURL to be true")
	}
	if plugin.Installable() {
		t.Error("expected plugin to NOT be installable (external URL)")
	}
	if plugin.InstallabilityReason() != "external repository (requires manual installation)" {
		t.Errorf("unexpected installability reason: %q", plugin.InstallabilityReason())
	}
}

func TestMarketplacePlugin_UnmarshalJSON_LSPPlugin(t *testing.T) {
	jsonData := `{
		"name": "gopls-lsp",
		"source": "./plugins/gopls-lsp",
		"description": "Go language server",
		"lspServers": {
			"gopls": {
				"command": "gopls",
				"extensionToLanguage": {
					".go": "go"
				}
			}
		}
	}`

	var plugin MarketplacePlugin
	if err := json.Unmarshal([]byte(jsonData), &plugin); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if plugin.Name != "gopls-lsp" {
		t.Errorf("expected name 'gopls-lsp', got %q", plugin.Name)
	}
	if !plugin.HasLSPServers {
		t.Error("expected HasLSPServers to be true")
	}
	if plugin.Installable() {
		t.Error("expected plugin to NOT be installable (LSP plugin)")
	}
	if plugin.InstallabilityReason() != "LSP plugin (built into Claude Code)" {
		t.Errorf("unexpected installability reason: %q", plugin.InstallabilityReason())
	}
}

func TestMarketplacePlugin_UnmarshalJSON_RegularPlugin(t *testing.T) {
	jsonData := `{
		"name": "code-simplifier",
		"source": "./plugins/code-simplifier",
		"description": "Code simplification agent",
		"version": "1.0.0"
	}`

	var plugin MarketplacePlugin
	if err := json.Unmarshal([]byte(jsonData), &plugin); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if !plugin.Installable() {
		t.Error("expected plugin to be installable")
	}
	if plugin.InstallabilityReason() != "" {
		t.Errorf("expected empty installability reason, got %q", plugin.InstallabilityReason())
	}
}

func TestMarketplacePlugin_UnmarshalJSON_EmptyLSPServersObject(t *testing.T) {
	// Edge case: empty lspServers object should NOT mark as LSP plugin
	jsonData := `{
		"name": "plugin-with-empty-lsp",
		"source": "./plugins/test",
		"description": "Plugin with empty lspServers",
		"lspServers": {}
	}`

	var plugin MarketplacePlugin
	if err := json.Unmarshal([]byte(jsonData), &plugin); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if plugin.HasLSPServers {
		t.Error("expected HasLSPServers to be false for empty object")
	}
	if !plugin.Installable() {
		t.Error("expected plugin to be installable (empty lspServers)")
	}
}

func TestMarketplacePlugin_UnmarshalJSON_EmptyLSPServersArray(t *testing.T) {
	// Edge case: empty lspServers array should NOT mark as LSP plugin
	jsonData := `{
		"name": "plugin-with-empty-lsp-array",
		"source": "./plugins/test",
		"description": "Plugin with empty lspServers array",
		"lspServers": []
	}`

	var plugin MarketplacePlugin
	if err := json.Unmarshal([]byte(jsonData), &plugin); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if plugin.HasLSPServers {
		t.Error("expected HasLSPServers to be false for empty array")
	}
	if !plugin.Installable() {
		t.Error("expected plugin to be installable (empty lspServers array)")
	}
}

func TestMarketplacePlugin_UnmarshalJSON_NullLSPServers(t *testing.T) {
	// Edge case: null lspServers should NOT mark as LSP plugin
	jsonData := `{
		"name": "plugin-with-null-lsp",
		"source": "./plugins/test",
		"description": "Plugin with null lspServers",
		"lspServers": null
	}`

	var plugin MarketplacePlugin
	if err := json.Unmarshal([]byte(jsonData), &plugin); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if plugin.HasLSPServers {
		t.Error("expected HasLSPServers to be false for null")
	}
	if !plugin.Installable() {
		t.Error("expected plugin to be installable (null lspServers)")
	}
}

func TestMarketplaceManifest_UnmarshalJSON(t *testing.T) {
	jsonData := `{
		"name": "test-marketplace",
		"plugins": [
			{
				"name": "regular-plugin",
				"source": "./plugins/regular",
				"description": "Regular plugin"
			},
			{
				"name": "lsp-plugin",
				"source": "./plugins/lsp",
				"description": "LSP plugin",
				"lspServers": {"test": {}}
			},
			{
				"name": "external-plugin",
				"source": {"source": "url", "url": "https://example.com/repo.git"},
				"description": "External plugin"
			}
		]
	}`

	var manifest MarketplaceManifest
	if err := json.Unmarshal([]byte(jsonData), &manifest); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}

	if len(manifest.Plugins) != 3 {
		t.Fatalf("expected 3 plugins, got %d", len(manifest.Plugins))
	}

	// Check installability counts
	installable := 0
	for _, p := range manifest.Plugins {
		if p.Installable() {
			installable++
		}
	}
	if installable != 1 {
		t.Errorf("expected 1 installable plugin, got %d", installable)
	}
}
