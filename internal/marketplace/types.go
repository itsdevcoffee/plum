package marketplace

import "encoding/json"

// MarketplaceManifest represents the marketplace.json structure
type MarketplaceManifest struct {
	Name     string              `json:"name"`
	Owner    MarketplaceOwner    `json:"owner"`
	Metadata MarketplaceMetadata `json:"metadata"`
	Plugins  []MarketplacePlugin `json:"plugins"`
}

// MarketplaceOwner represents the owner of a marketplace
type MarketplaceOwner struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Company string `json:"company"`
}

// MarketplaceMetadata represents marketplace metadata
type MarketplaceMetadata struct {
	Description string `json:"description"`
	Version     string `json:"version"`
	PluginRoot  string `json:"pluginRoot"`
}

// MarketplacePlugin represents a plugin entry in a marketplace manifest
type MarketplacePlugin struct {
	Name        string   `json:"name"`
	Source      string   `json:"source"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Author      Author   `json:"author"`
	Category    string   `json:"category"`
	Homepage    string   `json:"homepage"`
	Repository  string   `json:"repository"`
	License     string   `json:"license"`
	Keywords    []string `json:"keywords"`
	Tags        []string `json:"tags"`
	Strict      bool     `json:"strict"`

	// Installability tracking (set during unmarshaling or validation)
	HasLSPServers bool `json:"-"` // True if plugin has lspServers config (built into Claude Code)
	IsExternalURL bool `json:"-"` // True if source points to external Git repo
	IsIncomplete  bool `json:"-"` // True if plugin is missing required files (e.g., .claude-plugin/plugin.json)
}

// Installable returns true if the plugin can be installed via plum.
// Plugins with LSP servers, external URLs, or missing files require different installation methods.
func (mp *MarketplacePlugin) Installable() bool {
	return !mp.HasLSPServers && !mp.IsExternalURL && !mp.IsIncomplete
}

// InstallabilityReason returns a human-readable reason why the plugin is not installable.
// Returns empty string if the plugin is installable.
func (mp *MarketplacePlugin) InstallabilityReason() string {
	switch {
	case mp.HasLSPServers:
		return "LSP plugin (built into Claude Code)"
	case mp.IsExternalURL:
		return "external repository (requires manual installation)"
	case mp.IsIncomplete:
		return "incomplete plugin (missing .claude-plugin/plugin.json)"
	default:
		return ""
	}
}

// InstallabilityTag returns a short tag for display purposes.
// Returns empty string if the plugin is installable.
func (mp *MarketplacePlugin) InstallabilityTag() string {
	switch {
	case mp.HasLSPServers:
		return "[built-in]"
	case mp.IsExternalURL:
		return "[external]"
	case mp.IsIncomplete:
		return "[incomplete]"
	default:
		return ""
	}
}

// UnmarshalJSON implements custom JSON unmarshaling for MarketplacePlugin to handle
// the "source" field which can be either a string or an object with Git URL.
// Also detects LSP plugins and external URL sources for installability tracking.
// Required for claude-plugins-official compatibility (Atlassian, Figma, Vercel, etc.)
func (mp *MarketplacePlugin) UnmarshalJSON(data []byte) error {
	// Create alias type to avoid infinite recursion
	type MarketplacePluginAlias MarketplacePlugin

	// Use a temporary struct with source as RawMessage and lspServers detection
	var temp struct {
		*MarketplacePluginAlias
		SourceRaw  json.RawMessage `json:"source"`
		LSPServers json.RawMessage `json:"lspServers"` // Detect LSP plugins
	}
	temp.MarketplacePluginAlias = (*MarketplacePluginAlias)(mp)

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Check for LSP servers (indicates built-in Claude Code plugin)
	// Only mark as LSP if lspServers contains actual configuration (not null, {}, or [])
	if len(temp.LSPServers) > 0 {
		s := string(temp.LSPServers)
		if s != "null" && s != "{}" && s != "[]" {
			mp.HasLSPServers = true
		}
	}

	// Parse source field which can be string or object
	if len(temp.SourceRaw) > 0 {
		// Try to unmarshal as string first (most common)
		var sourceStr string
		if err := json.Unmarshal(temp.SourceRaw, &sourceStr); err == nil {
			mp.Source = sourceStr
		} else {
			// Try as object with URL (claude-plugins-official Git repos)
			var sourceObj struct {
				Source string `json:"source"`
				URL    string `json:"url"`
			}
			if err := json.Unmarshal(temp.SourceRaw, &sourceObj); err == nil {
				// Use the Git URL as the source
				if sourceObj.URL != "" {
					mp.Source = sourceObj.URL
					mp.IsExternalURL = true // Mark as external URL source
				}
			}
		}
	}

	return nil
}

// Author represents author information
type Author struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	URL     string `json:"url"`
	Company string `json:"company"`
}
