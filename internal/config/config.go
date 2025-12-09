package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/maskkiller/plum/internal/plugin"
)

// ClaudePluginsDir returns the path to the Claude plugins directory
func ClaudePluginsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "plugins")
}

// KnownMarketplaces represents the known_marketplaces.json structure
type KnownMarketplaces map[string]MarketplaceEntry

// MarketplaceEntry represents a single marketplace entry
type MarketplaceEntry struct {
	Source          MarketplaceSource `json:"source"`
	InstallLocation string            `json:"installLocation"`
	LastUpdated     string            `json:"lastUpdated"`
}

// MarketplaceSource represents the source of a marketplace
type MarketplaceSource struct {
	Source string `json:"source"`
	Repo   string `json:"repo"`
}

// InstalledPluginsV2 represents the installed_plugins_v2.json structure
type InstalledPluginsV2 struct {
	Version int                        `json:"version"`
	Plugins map[string][]PluginInstall `json:"plugins"`
}

// PluginInstall represents a single plugin installation entry
type PluginInstall struct {
	Scope        string `json:"scope"`
	InstallPath  string `json:"installPath"`
	Version      string `json:"version"`
	InstalledAt  string `json:"installedAt"`
	LastUpdated  string `json:"lastUpdated"`
	GitCommitSha string `json:"gitCommitSha"`
	IsLocal      bool   `json:"isLocal"`
	ProjectPath  string `json:"projectPath,omitempty"`
}

// MarketplaceManifest represents the marketplace.json structure
type MarketplaceManifest struct {
	Name     string               `json:"name"`
	Owner    MarketplaceOwner     `json:"owner"`
	Metadata MarketplaceMetadata  `json:"metadata"`
	Plugins  []MarketplacePlugin  `json:"plugins"`
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
	Keywords    []string `json:"keywords"`
	Strict      bool     `json:"strict"`
}

// Author represents author information
type Author struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	URL     string `json:"url"`
	Company string `json:"company"`
}

// LoadKnownMarketplaces loads the known_marketplaces.json file
func LoadKnownMarketplaces() (KnownMarketplaces, error) {
	path := filepath.Join(ClaudePluginsDir(), "known_marketplaces.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var marketplaces KnownMarketplaces
	if err := json.Unmarshal(data, &marketplaces); err != nil {
		return nil, err
	}

	return marketplaces, nil
}

// LoadInstalledPlugins loads the installed_plugins_v2.json file
func LoadInstalledPlugins() (*InstalledPluginsV2, error) {
	path := filepath.Join(ClaudePluginsDir(), "installed_plugins_v2.json")
	data, err := os.ReadFile(path)
	if err != nil {
		// Return empty if file doesn't exist
		if os.IsNotExist(err) {
			return &InstalledPluginsV2{Version: 2, Plugins: make(map[string][]PluginInstall)}, nil
		}
		return nil, err
	}

	var installed InstalledPluginsV2
	if err := json.Unmarshal(data, &installed); err != nil {
		return nil, err
	}

	return &installed, nil
}

// LoadMarketplaceManifest loads a marketplace.json file from a marketplace directory
func LoadMarketplaceManifest(marketplacePath string) (*MarketplaceManifest, error) {
	manifestPath := filepath.Join(marketplacePath, ".claude-plugin", "marketplace.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var manifest MarketplaceManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// LoadAllPlugins loads all plugins from all known marketplaces
func LoadAllPlugins() ([]plugin.Plugin, error) {
	marketplaces, err := LoadKnownMarketplaces()
	if err != nil {
		return nil, err
	}

	installed, err := LoadInstalledPlugins()
	if err != nil {
		return nil, err
	}

	// Build a set of installed plugin names for quick lookup
	installedSet := make(map[string]PluginInstall)
	for fullName, installs := range installed.Plugins {
		if len(installs) > 0 {
			installedSet[fullName] = installs[0]
		}
	}

	var plugins []plugin.Plugin

	for marketplaceName, entry := range marketplaces {
		manifest, err := LoadMarketplaceManifest(entry.InstallLocation)
		if err != nil {
			// Skip marketplaces we can't load
			continue
		}

		for _, mp := range manifest.Plugins {
			fullName := mp.Name + "@" + marketplaceName
			install, isInstalled := installedSet[fullName]

			p := plugin.Plugin{
				Name:        mp.Name,
				Description: mp.Description,
				Version:     mp.Version,
				Keywords:    mp.Keywords,
				Category:    mp.Category,
				Author: plugin.Author{
					Name:    mp.Author.Name,
					Email:   mp.Author.Email,
					URL:     mp.Author.URL,
					Company: mp.Author.Company,
				},
				Marketplace: marketplaceName,
				Installed:   isInstalled,
				Source:      mp.Source,
				Homepage:    mp.Homepage,
			}

			if isInstalled {
				p.InstallPath = install.InstallPath
			}

			plugins = append(plugins, p)
		}
	}

	return plugins, nil
}
