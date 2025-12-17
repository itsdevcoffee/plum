package marketplace

import (
	"fmt"
	"os"
	"sync"
)

const (
	// MaxConcurrentFetches limits parallel marketplace downloads
	MaxConcurrentFetches = 5
)

// PopularMarketplace represents a hardcoded popular marketplace
type PopularMarketplace struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Repo        string `json:"repo"` // Full repo URL (e.g., https://github.com/owner/repo)
	Description string `json:"description"`
}

// DiscoveredMarketplace contains a marketplace manifest with source information
type DiscoveredMarketplace struct {
	Manifest *MarketplaceManifest
	Repo     string // Full repo URL for display
	Source   string // Derived CLI source (owner/repo for GitHub, full URL for others)
}

// PopularMarketplaces is the hardcoded list from README.md
var PopularMarketplaces = []PopularMarketplace{
	{
		Name:        "claude-code-plugins-plus",
		DisplayName: "Claude Code Plugins Plus",
		Repo:        "https://github.com/jeremylongshore/claude-code-plugins",
		Description: "The largest collection with 254 plugins and 185 Agent Skills",
	},
	{
		Name:        "claude-code-marketplace",
		DisplayName: "Claude Code Marketplace",
		Repo:        "https://github.com/ananddtyagi/claude-code-marketplace",
		Description: "Community-driven marketplace with granular installation",
	},
	{
		Name:        "claude-code-plugins",
		DisplayName: "Claude Code Plugins",
		Repo:        "https://github.com/anthropics/claude-code",
		Description: "Official Anthropic plugins maintained by the Claude Code team",
	},
	{
		Name:        "mag-claude-plugins",
		DisplayName: "MAG Claude Plugins",
		Repo:        "https://github.com/MadAppGang/claude-code",
		Description: "Battle-tested workflows with 4 specialized plugins",
	},
	{
		Name:        "dev-gom-plugins",
		DisplayName: "Dev-GOM Plugins",
		Repo:        "https://github.com/Dev-GOM/claude-code-marketplace",
		Description: "Automation-focused collection with 15 plugins",
	},
	{
		Name:        "feedmob-claude-plugins",
		DisplayName: "FeedMob Plugins",
		Repo:        "https://github.com/feed-mob/claude-code-marketplace",
		Description: "Productivity and workflow tools with 6 specialized plugins",
	},
	{
		Name:        "claude-plugins-official",
		DisplayName: "Claude Plugins Official",
		Repo:        "https://github.com/anthropics/claude-plugins-official",
		Description: "Official Anthropic plugins for Claude Code",
	},
	{
		Name:        "anthropic-agent-skills",
		DisplayName: "Anthropic Agent Skills",
		Repo:        "https://github.com/anthropics/skills",
		Description: "Official Anthropic skills reference with document manipulation and examples",
	},
}

// DiscoverPopularMarketplaces fetches and returns manifests for popular marketplaces
// Uses cached registry if available (from Shift+U), otherwise hardcoded list
// Uses cache when available, fetches from GitHub otherwise
// Returns partial results on partial failures (best-effort)
func DiscoverPopularMarketplaces() (map[string]*DiscoveredMarketplace, error) {
	// Check if user has updated the registry (via Shift+U)
	marketplaceList := PopularMarketplaces
	if cachedRegistry, err := loadRegistryFromCache(); err == nil && cachedRegistry != nil {
		// User pressed Shift+U before - use their updated registry
		marketplaceList = cachedRegistry.Marketplaces
	}

	var (
		discovered = make(map[string]*DiscoveredMarketplace)
		mu         sync.Mutex
		wg         sync.WaitGroup
		errs       []error
		sem        = make(chan struct{}, MaxConcurrentFetches) // Semaphore for concurrency limiting
	)

	// Fetch all marketplaces with concurrency limit
	for _, pm := range marketplaceList {
		wg.Add(1)
		go func(marketplace PopularMarketplace) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }() // Release semaphore

			disc, err := fetchMarketplaceFromGitHub(marketplace)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", marketplace.Name, err))
				return
			}

			discovered[marketplace.Name] = disc
		}(pm)
	}

	wg.Wait()

	// If all fetches failed, return error
	if len(discovered) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("all marketplace fetches failed: %v", errs)
	}

	// Log partial failures but continue
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}

	return discovered, nil
}

// fetchMarketplaceFromGitHub fetches a single marketplace with caching
func fetchMarketplaceFromGitHub(pm PopularMarketplace) (*DiscoveredMarketplace, error) {
	// Derive CLI source from repo URL
	source, err := DeriveSource(pm.Repo)
	if err != nil {
		return nil, fmt.Errorf("failed to derive source: %w", err)
	}

	// Try cache first
	cached, err := LoadFromCache(pm.Name)
	if err == nil && cached != nil {
		return &DiscoveredMarketplace{
			Manifest: cached,
			Repo:     pm.Repo,
			Source:   source,
		}, nil
	}

	// Cache miss or expired - fetch from GitHub
	manifest, err := FetchManifestFromGitHub(pm.Repo)
	if err != nil {
		return nil, err
	}

	// Update manifest name to match our hardcoded name
	manifest.Name = pm.Name

	// Save to cache (best effort - don't fail if cache write fails)
	_ = SaveToCache(pm.Name, manifest)

	return &DiscoveredMarketplace{
		Manifest: manifest,
		Repo:     pm.Repo,
		Source:   source,
	}, nil
}
