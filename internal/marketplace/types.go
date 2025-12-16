package marketplace

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

// Author represents author information
type Author struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	URL     string `json:"url"`
	Company string `json:"company"`
}
