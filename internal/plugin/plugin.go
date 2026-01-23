package plugin

import (
	"encoding/json"
	"strings"
)

// Plugin represents a Claude Code plugin from any marketplace.
// Contains metadata, installation state, and marketplace source information.
// Used for search, display, and installation command generation.
type Plugin struct {
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Version           string   `json:"version"`
	Keywords          []string `json:"keywords"`
	Category          string   `json:"category"`
	Author            Author   `json:"author"`
	Marketplace       string   `json:"-"`      // Friendly marketplace name (e.g., "feedmob-plugins")
	MarketplaceRepo   string   `json:"-"`      // Full repo URL for display (e.g., "https://github.com/feed-mob/claude-code-marketplace")
	MarketplaceSource string   `json:"-"`      // CLI source format (e.g., "feed-mob/claude-code-marketplace" for GitHub)
	Installed         bool     `json:"-"`      // Whether this plugin is currently installed
	IsDiscoverable    bool     `json:"-"`      // Whether from a discoverable (not installed) marketplace
	InstallPath       string   `json:"-"`      // Path if installed
	Source            string   `json:"source"` // Source path within marketplace
	Homepage          string   `json:"homepage"`
	Repository        string   `json:"repository"` // Source repository URL
	License           string   `json:"license"`    // License identifier (e.g., "MIT")
	Tags              []string `json:"tags"`       // Categorization tags

	// Installability tracking
	HasLSPServers bool `json:"-"` // True if plugin has lspServers config (built into Claude Code)
	IsExternalURL bool `json:"-"` // True if source points to external Git repo
	IsIncomplete  bool `json:"-"` // True if plugin is missing required files (e.g., .claude-plugin/plugin.json)
}

// Installable returns true if the plugin can be installed via plum.
// Plugins with LSP servers, external URLs, or missing files require different installation methods.
func (p Plugin) Installable() bool {
	return !p.HasLSPServers && !p.IsExternalURL && !p.IsIncomplete
}

// InstallabilityReason returns a human-readable reason why the plugin is not installable.
// Returns empty string if the plugin is installable.
func (p Plugin) InstallabilityReason() string {
	switch {
	case p.HasLSPServers:
		return "LSP plugin (built into Claude Code)"
	case p.IsExternalURL:
		return "external repository (requires manual installation)"
	case p.IsIncomplete:
		return "incomplete plugin (missing .claude-plugin/plugin.json)"
	default:
		return ""
	}
}

// InstallabilityTag returns a short tag for display purposes.
// Returns empty string if the plugin is installable.
func (p Plugin) InstallabilityTag() string {
	switch {
	case p.HasLSPServers:
		return "[built-in]"
	case p.IsExternalURL:
		return "[external]"
	case p.IsIncomplete:
		return "[incomplete]"
	default:
		return ""
	}
}

// Author represents plugin author information
type Author struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	URL     string `json:"url"`
	Company string `json:"company"`
}

// FullName returns the plugin identifier in format "name@marketplace"
func (p Plugin) FullName() string {
	return p.Name + "@" + p.Marketplace
}

// InstallCommand returns the command to install this plugin
func (p Plugin) InstallCommand() string {
	return "/plugin install " + p.FullName()
}

// FilterValue implements the list.Item interface for bubbles/list
func (p Plugin) FilterValue() string {
	return p.Name + " " + p.Description
}

// Title implements the list.DefaultItem interface
func (p Plugin) Title() string {
	return p.Name
}

// AuthorName returns the author's name or "Unknown" if not set
func (p Plugin) AuthorName() string {
	if p.Author.Name != "" {
		return p.Author.Name
	}
	if p.Author.Company != "" {
		return p.Author.Company
	}
	return "Unknown"
}

// UnmarshalJSON implements custom JSON unmarshaling for Plugin to handle
// the "source" field which can be either a string or an object with Git URL.
func (p *Plugin) UnmarshalJSON(data []byte) error {
	// Create alias type to avoid infinite recursion
	type PluginAlias Plugin

	// Use a temporary struct with source as RawMessage
	var temp struct {
		*PluginAlias
		SourceRaw json.RawMessage `json:"source"`
	}
	temp.PluginAlias = (*PluginAlias)(p)

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Parse source field which can be string or object
	if len(temp.SourceRaw) > 0 {
		// Try to unmarshal as string first
		var sourceStr string
		if err := json.Unmarshal(temp.SourceRaw, &sourceStr); err == nil {
			p.Source = sourceStr
		} else {
			// Try as object with URL
			var sourceObj struct {
				Source string `json:"source"`
				URL    string `json:"url"`
			}
			if err := json.Unmarshal(temp.SourceRaw, &sourceObj); err == nil {
				// Use the Git URL as the source
				if sourceObj.URL != "" {
					p.Source = sourceObj.URL
				}
			}
		}
	}

	return nil
}

// GitHubURL returns the GitHub URL for this plugin's source code
// Constructs URL from MarketplaceRepo + Source path
// Example: https://github.com/owner/repo/tree/main/plugins/plugin-name
func (p Plugin) GitHubURL() string {
	if p.MarketplaceRepo == "" {
		return ""
	}

	// Normalize source path (remove leading ./ if present)
	sourcePath := strings.TrimPrefix(p.Source, "./")

	// If source is empty or not a relative path, default to plugin name
	if sourcePath == "" || sourcePath == "." {
		sourcePath = "plugins/" + p.Name
	}

	// Construct GitHub tree URL
	return p.MarketplaceRepo + "/tree/main/" + sourcePath
}
