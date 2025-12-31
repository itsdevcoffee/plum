package plugin

import (
	"encoding/json"
	"testing"
)

// TestFullName verifies plugin identifier format
func TestFullName(t *testing.T) {
	tests := []struct {
		name        string
		plugin      Plugin
		expectValue string
	}{
		{
			name:        "standard plugin",
			plugin:      Plugin{Name: "test-plugin", Marketplace: "test-marketplace"},
			expectValue: "test-plugin@test-marketplace",
		},
		{
			name:        "plugin with hyphens",
			plugin:      Plugin{Name: "my-cool-plugin", Marketplace: "awesome-marketplace"},
			expectValue: "my-cool-plugin@awesome-marketplace",
		},
		{
			name:        "empty marketplace",
			plugin:      Plugin{Name: "plugin", Marketplace: ""},
			expectValue: "plugin@",
		},
		{
			name:        "empty name",
			plugin:      Plugin{Name: "", Marketplace: "marketplace"},
			expectValue: "@marketplace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.plugin.FullName()
			if result != tt.expectValue {
				t.Errorf("Expected %q, got %q", tt.expectValue, result)
			}
		})
	}
}

// TestInstallCommand verifies install command format
func TestInstallCommand(t *testing.T) {
	tests := []struct {
		name        string
		plugin      Plugin
		expectValue string
	}{
		{
			name:        "standard plugin",
			plugin:      Plugin{Name: "test-plugin", Marketplace: "test-marketplace"},
			expectValue: "/plugin install test-plugin@test-marketplace",
		},
		{
			name:        "plugin with special characters",
			plugin:      Plugin{Name: "my_plugin-v2", Marketplace: "marketplace-2024"},
			expectValue: "/plugin install my_plugin-v2@marketplace-2024",
		},
		{
			name:        "discoverable plugin",
			plugin:      Plugin{Name: "plugin", Marketplace: "uninstalled-marketplace", IsDiscoverable: true},
			expectValue: "/plugin install plugin@uninstalled-marketplace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.plugin.InstallCommand()
			if result != tt.expectValue {
				t.Errorf("Expected %q, got %q", tt.expectValue, result)
			}
		})
	}
}

// TestFilterValue verifies searchable text generation
func TestFilterValue(t *testing.T) {
	tests := []struct {
		name        string
		plugin      Plugin
		expectValue string
	}{
		{
			name:        "with name and description",
			plugin:      Plugin{Name: "test-plugin", Description: "A test plugin"},
			expectValue: "test-plugin A test plugin",
		},
		{
			name:        "description only",
			plugin:      Plugin{Name: "", Description: "Some description"},
			expectValue: " Some description",
		},
		{
			name:        "name only",
			plugin:      Plugin{Name: "plugin-name", Description: ""},
			expectValue: "plugin-name ",
		},
		{
			name:        "both empty",
			plugin:      Plugin{Name: "", Description: ""},
			expectValue: " ",
		},
		{
			name:        "with special characters",
			plugin:      Plugin{Name: "test™", Description: "A plugin with emoji 🚀"},
			expectValue: "test™ A plugin with emoji 🚀",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.plugin.FilterValue()
			if result != tt.expectValue {
				t.Errorf("Expected %q, got %q", tt.expectValue, result)
			}
		})
	}
}

// TestTitle verifies title passthrough
func TestTitle(t *testing.T) {
	tests := []struct {
		name        string
		plugin      Plugin
		expectValue string
	}{
		{
			name:        "standard name",
			plugin:      Plugin{Name: "test-plugin"},
			expectValue: "test-plugin",
		},
		{
			name:        "empty name",
			plugin:      Plugin{Name: ""},
			expectValue: "",
		},
		{
			name:        "name with spaces",
			plugin:      Plugin{Name: "test plugin name"},
			expectValue: "test plugin name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.plugin.Title()
			if result != tt.expectValue {
				t.Errorf("Expected %q, got %q", tt.expectValue, result)
			}
		})
	}
}

// TestAuthorName verifies author name resolution logic
func TestAuthorName(t *testing.T) {
	tests := []struct {
		name        string
		plugin      Plugin
		expectValue string
	}{
		{
			name: "name set",
			plugin: Plugin{
				Author: Author{Name: "John Doe"},
			},
			expectValue: "John Doe",
		},
		{
			name: "company set, name empty",
			plugin: Plugin{
				Author: Author{Name: "", Company: "Acme Corp"},
			},
			expectValue: "Acme Corp",
		},
		{
			name: "both name and company set",
			plugin: Plugin{
				Author: Author{Name: "John Doe", Company: "Acme Corp"},
			},
			expectValue: "John Doe", // Name takes precedence
		},
		{
			name: "neither set",
			plugin: Plugin{
				Author: Author{Name: "", Company: ""},
			},
			expectValue: "Unknown",
		},
		{
			name:        "empty author struct",
			plugin:      Plugin{},
			expectValue: "Unknown",
		},
		{
			name: "whitespace only name",
			plugin: Plugin{
				Author: Author{Name: "   ", Company: "Acme Corp"},
			},
			expectValue: "   ", // Whitespace is technically not empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.plugin.AuthorName()
			if result != tt.expectValue {
				t.Errorf("Expected %q, got %q", tt.expectValue, result)
			}
		})
	}
}

// TestGitHubURL verifies GitHub URL construction
func TestGitHubURL(t *testing.T) {
	tests := []struct {
		name        string
		plugin      Plugin
		expectValue string
	}{
		{
			name: "standard source path",
			plugin: Plugin{
				MarketplaceRepo: "https://github.com/owner/repo",
				Source:          "plugins/test-plugin",
			},
			expectValue: "https://github.com/owner/repo/tree/main/plugins/test-plugin",
		},
		{
			name: "source with leading ./",
			plugin: Plugin{
				MarketplaceRepo: "https://github.com/owner/repo",
				Source:          "./plugins/test-plugin",
			},
			expectValue: "https://github.com/owner/repo/tree/main/plugins/test-plugin",
		},
		{
			name: "empty source defaults to plugin name",
			plugin: Plugin{
				Name:            "my-plugin",
				MarketplaceRepo: "https://github.com/owner/repo",
				Source:          "",
			},
			expectValue: "https://github.com/owner/repo/tree/main/plugins/my-plugin",
		},
		{
			name: "source is dot",
			plugin: Plugin{
				Name:            "my-plugin",
				MarketplaceRepo: "https://github.com/owner/repo",
				Source:          ".",
			},
			expectValue: "https://github.com/owner/repo/tree/main/plugins/my-plugin",
		},
		{
			name: "empty marketplace repo",
			plugin: Plugin{
				MarketplaceRepo: "",
				Source:          "plugins/test",
			},
			expectValue: "",
		},
		{
			name: "nested source path",
			plugin: Plugin{
				MarketplaceRepo: "https://github.com/owner/repo",
				Source:          "dir/subdir/plugins/test",
			},
			expectValue: "https://github.com/owner/repo/tree/main/dir/subdir/plugins/test",
		},
		{
			name: "marketplace repo without https",
			plugin: Plugin{
				MarketplaceRepo: "github.com/owner/repo",
				Source:          "plugins/test",
			},
			expectValue: "github.com/owner/repo/tree/main/plugins/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.plugin.GitHubURL()
			if result != tt.expectValue {
				t.Errorf("Expected %q, got %q", tt.expectValue, result)
			}
		})
	}
}

// TestPluginStruct verifies the Plugin struct can be created and fields accessed
func TestPluginStruct(t *testing.T) {
	t.Run("create plugin with all fields", func(t *testing.T) {
		p := Plugin{
			Name:              "test-plugin",
			Description:       "A test plugin",
			Version:           "1.0.0",
			Keywords:          []string{"test", "example"},
			Category:          "Testing",
			Author:            Author{Name: "Test Author", Email: "test@example.com"},
			Marketplace:       "test-marketplace",
			MarketplaceRepo:   "https://github.com/test/marketplace",
			MarketplaceSource: "test/marketplace",
			Installed:         true,
			IsDiscoverable:    false,
			InstallPath:       "/path/to/plugin",
			Source:            "plugins/test-plugin",
			Homepage:          "https://example.com",
			Repository:        "https://github.com/test/plugin",
			License:           "MIT",
			Tags:              []string{"automation", "testing"},
		}

		if p.Name != "test-plugin" {
			t.Errorf("Expected Name %q, got %q", "test-plugin", p.Name)
		}
		if p.Version != "1.0.0" {
			t.Errorf("Expected Version %q, got %q", "1.0.0", p.Version)
		}
		if len(p.Keywords) != 2 {
			t.Errorf("Expected 2 keywords, got %d", len(p.Keywords))
		}
		if !p.Installed {
			t.Error("Expected Installed=true")
		}
	})

	t.Run("default values", func(t *testing.T) {
		p := Plugin{}

		if p.Name != "" {
			t.Error("Expected empty Name by default")
		}
		if p.Installed {
			t.Error("Expected Installed=false by default")
		}
		if p.IsDiscoverable {
			t.Error("Expected IsDiscoverable=false by default")
		}
	})
}

// TestAuthorStruct verifies Author struct
func TestAuthorStruct(t *testing.T) {
	t.Run("create author with all fields", func(t *testing.T) {
		a := Author{
			Name:    "John Doe",
			Email:   "john@example.com",
			URL:     "https://johndoe.com",
			Company: "Acme Corp",
		}

		if a.Name != "John Doe" {
			t.Errorf("Expected Name %q, got %q", "John Doe", a.Name)
		}
		if a.Email != "john@example.com" {
			t.Errorf("Expected Email %q, got %q", "john@example.com", a.Email)
		}
	})

	t.Run("default values", func(t *testing.T) {
		a := Author{}

		if a.Name != "" {
			t.Error("Expected empty Name by default")
		}
		if a.Company != "" {
			t.Error("Expected empty Company by default")
		}
	})
}

// TestPluginUnmarshalJSON verifies custom JSON unmarshaling for source field
func TestPluginUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name         string
		jsonData     string
		expectSource string
		expectName   string
	}{
		{
			name: "string source path - internal plugins",
			jsonData: `{
				"name": "test-plugin",
				"description": "A test plugin",
				"source": "./plugins/test-plugin"
			}`,
			expectSource: "./plugins/test-plugin",
			expectName:   "test-plugin",
		},
		{
			name: "string source path - external plugins",
			jsonData: `{
				"name": "github-plugin",
				"source": "./external_plugins/github"
			}`,
			expectSource: "./external_plugins/github",
			expectName:   "github-plugin",
		},
		{
			name: "git URL object format - claude-plugins-official pattern",
			jsonData: `{
				"name": "atlassian-plugin",
				"source": {
					"source": "url",
					"url": "https://github.com/atlassian/atlassian-mcp-server.git"
				}
			}`,
			expectSource: "https://github.com/atlassian/atlassian-mcp-server.git",
			expectName:   "atlassian-plugin",
		},
		{
			name: "git URL object with extra fields",
			jsonData: `{
				"name": "vercel-plugin",
				"source": {
					"source": "url",
					"url": "https://github.com/vercel/mcp-server.git",
					"ref": "main"
				}
			}`,
			expectSource: "https://github.com/vercel/mcp-server.git",
			expectName:   "vercel-plugin",
		},
		{
			name: "empty source string",
			jsonData: `{
				"name": "empty-source",
				"source": ""
			}`,
			expectSource: "",
			expectName:   "empty-source",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p Plugin
			err := json.Unmarshal([]byte(tt.jsonData), &p)
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if p.Source != tt.expectSource {
				t.Errorf("Expected Source %q, got %q", tt.expectSource, p.Source)
			}

			if p.Name != tt.expectName {
				t.Errorf("Expected Name %q, got %q", tt.expectName, p.Name)
			}
		})
	}
}
