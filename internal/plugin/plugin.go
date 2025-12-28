package plugin

import (
	"strings"
)

// Plugin represents a Claude Code plugin from any marketplace
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
