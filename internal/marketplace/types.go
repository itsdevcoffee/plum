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
}

// UnmarshalJSON implements custom JSON unmarshaling for MarketplacePlugin to handle
// the "source" field which can be either a string or an object with Git URL.
// Required for claude-plugins-official compatibility (Atlassian, Figma, Vercel, etc.)
func (mp *MarketplacePlugin) UnmarshalJSON(data []byte) error {
	// Create alias type to avoid infinite recursion
	type MarketplacePluginAlias MarketplacePlugin

	// Use a temporary struct with source as RawMessage
	var temp struct {
		*MarketplacePluginAlias
		SourceRaw json.RawMessage `json:"source"`
	}
	temp.MarketplacePluginAlias = (*MarketplacePluginAlias)(mp)

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
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
