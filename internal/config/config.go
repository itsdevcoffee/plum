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

	// #nosec G304 -- path is derived from known config dirs, not untrusted input
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("claude Code marketplaces not found at %s - please run Claude Code and configure at least one marketplace using the /plugin command", path)
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

	// #nosec G304 -- path is derived from known config dirs, not untrusted input
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
	// #nosec G304 -- manifestPath is constructed from validated local project path
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
	// Track ALL seen plugin names (across all sources) for global deduplication
	// Maps plugin name -> source marketplace (first one wins)
	seenPluginNames := make(map[string]string)

	// 1. Process installed marketplaces first
	for marketplaceName, entry := range marketplaces {
		processedMarketplaces[marketplaceName] = true

		manifest, err := LoadMarketplaceManifest(entry.InstallLocation)
		if err != nil {
			// Skip marketplaces we can't load
			continue
		}

		// Look up repo/source from PopularMarketplaces for known marketplaces
		var marketplaceRepo, marketplaceSource string
		for _, pm := range marketplace.PopularMarketplaces {
			if pm.Name == marketplaceName {
				marketplaceRepo = pm.Repo
				marketplaceSource, _ = marketplace.DeriveSource(pm.Repo)
				break
			}
		}

		// Track duplicates within this marketplace
		seenInThisMarketplace := make(map[string]bool)

		for _, mp := range manifest.Plugins {
			// Skip duplicates within this marketplace
			if seenInThisMarketplace[mp.Name] {
				continue
			}
			seenInThisMarketplace[mp.Name] = true

			// Skip if seen from a different marketplace
			if existingMarket, exists := seenPluginNames[mp.Name]; exists && existingMarket != marketplaceName {
				continue
			}
			seenPluginNames[mp.Name] = marketplaceName

			p := convertMarketplacePlugin(mp, marketplaceName, marketplaceRepo, marketplaceSource, false, installedSet)
			plugins = append(plugins, p)
		}
	}

	// 2. Discover popular marketplaces (best effort - don't fail if this fails)
	discovered, _ := marketplace.DiscoverPopularMarketplaces()
	for marketplaceName, disc := range discovered {
		// Skip if we already processed this marketplace from installed
		if processedMarketplaces[marketplaceName] {
			continue
		}

		// Track duplicates within this discovered marketplace
		seenInThisMarketplace := make(map[string]bool)

		for _, mp := range disc.Manifest.Plugins {
			// Skip duplicates within this marketplace
			if seenInThisMarketplace[mp.Name] {
				continue
			}
			seenInThisMarketplace[mp.Name] = true

			// Skip if seen from any previous source
			if _, exists := seenPluginNames[mp.Name]; exists {
				continue
			}
			seenPluginNames[mp.Name] = marketplaceName

			p := convertMarketplacePlugin(mp, marketplaceName, disc.Repo, disc.Source, true, installedSet)
			plugins = append(plugins, p)
		}
	}

	return plugins, nil
}

// convertMarketplacePlugin converts a MarketplacePlugin to a Plugin.
func convertMarketplacePlugin(
	mp marketplace.MarketplacePlugin,
	marketplaceName string,
	marketplaceRepo string,
	marketplaceSource string,
	isDiscoverable bool,
	installedSet map[string]PluginInstall,
) plugin.Plugin {
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
		Marketplace:       marketplaceName,
		MarketplaceRepo:   marketplaceRepo,
		MarketplaceSource: marketplaceSource,
		Installed:         isInstalled,
		IsDiscoverable:    isDiscoverable,
		Source:            mp.Source,
		Homepage:          mp.Homepage,
		Repository:        mp.Repository,
		License:           mp.License,
		Tags:              mp.Tags,
		HasLSPServers:     mp.HasLSPServers,
		IsExternalURL:     mp.IsExternalURL,
	}

	if isInstalled {
		p.InstallPath = install.InstallPath
	}

	return p
}
