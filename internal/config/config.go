package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/itsdevcoffee/plum/internal/marketplace"
	"github.com/itsdevcoffee/plum/internal/plugin"
)

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

// LoadKnownMarketplaces loads the known_marketplaces.json file
func LoadKnownMarketplaces() (KnownMarketplaces, error) {
	path, err := KnownMarketplacesPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("Claude Code marketplaces not found at %s.\n\nPlease run Claude Code and configure at least one marketplace using the /plugin command.", path)
		}
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
	path, err := InstalledPluginsPath()
	if err != nil {
		return nil, err
	}

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
func LoadMarketplaceManifest(marketplacePath string) (*marketplace.MarketplaceManifest, error) {
	manifestPath := filepath.Join(marketplacePath, ".claude-plugin", "marketplace.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var manifest marketplace.MarketplaceManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// LoadAllPlugins loads all plugins from all known marketplaces
// Also discovers plugins from popular marketplaces not yet installed
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

	// Track which marketplaces we've processed to avoid duplicates
	processedMarketplaces := make(map[string]bool)

	// 1. Process installed marketplaces first
	for marketplaceName, entry := range marketplaces {
		processedMarketplaces[marketplaceName] = true

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
				Marketplace:    marketplaceName,
				Installed:      isInstalled,
				IsDiscoverable: false, // From installed marketplace
				Source:         mp.Source,
				Homepage:       mp.Homepage,
				Repository:     mp.Repository,
				License:        mp.License,
				Tags:           mp.Tags,
			}

			if isInstalled {
				p.InstallPath = install.InstallPath
			}

			plugins = append(plugins, p)
		}
	}

	// 2. Discover popular marketplaces (best effort - don't fail if this fails)
	discovered, err := marketplace.DiscoverPopularMarketplaces()
	if err != nil {
		// Log warning but continue with installed marketplaces only
		fmt.Fprintf(os.Stderr, "Warning: marketplace discovery failed: %v\n", err)
		fmt.Fprintf(os.Stderr, "Continuing with installed marketplaces only.\n")
	} else {
		// Process discovered marketplaces
		for marketplaceName, manifest := range discovered {
			// Skip if we already processed this marketplace from installed
			if processedMarketplaces[marketplaceName] {
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
					Marketplace:    marketplaceName,
					Installed:      isInstalled,
					IsDiscoverable: true, // From discovered marketplace
					Source:         mp.Source,
					Homepage:       mp.Homepage,
					Repository:     mp.Repository,
					License:        mp.License,
					Tags:           mp.Tags,
				}

				if isInstalled {
					p.InstallPath = install.InstallPath
				}

				plugins = append(plugins, p)
			}
		}
	}

	return plugins, nil
}
