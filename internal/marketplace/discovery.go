package marketplace

import (
	"fmt"
	"os"
	"sync"
)

// PopularMarketplace represents a hardcoded popular marketplace
type PopularMarketplace struct {
	Name        string
	DisplayName string
	GitHubRepo  string
	Description string
}

// PopularMarketplaces is the hardcoded list from README.md
var PopularMarketplaces = []PopularMarketplace{
	{
		Name:        "claude-code-plugins-plus",
		DisplayName: "Claude Code Plugins Plus",
		GitHubRepo:  "jeremylongshore/claude-code-plugins",
		Description: "The largest collection with 254 plugins and 185 Agent Skills",
	},
	{
		Name:        "claude-code-marketplace",
		DisplayName: "Claude Code Marketplace",
		GitHubRepo:  "ananddtyagi/claude-code-marketplace",
		Description: "Community-driven marketplace with granular installation",
	},
	{
		Name:        "claude-code-plugins",
		DisplayName: "Claude Code Plugins",
		GitHubRepo:  "anthropics/claude-code",
		Description: "Official Anthropic plugins maintained by the Claude Code team",
	},
	{
		Name:        "mag-claude-plugins",
		DisplayName: "MAG Claude Plugins",
		GitHubRepo:  "MadAppGang/claude-code",
		Description: "Battle-tested workflows with 4 specialized plugins",
	},
	{
		Name:        "dev-gom-plugins",
		DisplayName: "Dev-GOM Plugins",
		GitHubRepo:  "Dev-GOM/claude-code-marketplace",
		Description: "Automation-focused collection with 15 plugins",
	},
	{
		Name:        "feedmob-plugins",
		DisplayName: "FeedMob Plugins",
		GitHubRepo:  "feed-mob/claude-code-marketplace",
		Description: "Productivity and workflow tools with 6 specialized plugins",
	},
	{
		Name:        "anthropic-agent-skills",
		DisplayName: "Anthropic Agent Skills",
		GitHubRepo:  "anthropics/skills",
		Description: "Official Anthropic skills reference with document manipulation and examples",
	},
}

// DiscoverPopularMarketplaces fetches and returns manifests for popular marketplaces
// Uses HARDCODED list only - does not fetch registry
// Uses cache when available, fetches from GitHub otherwise
// Returns partial results on partial failures (best-effort)
func DiscoverPopularMarketplaces() (map[string]*MarketplaceManifest, error) {
	// Use hardcoded list - registry is only checked for notifications
	marketplaceList := PopularMarketplaces

	var (
		manifests = make(map[string]*MarketplaceManifest)
		mu        sync.Mutex
		wg        sync.WaitGroup
		errs      []error
	)

	// Fetch all marketplaces in parallel
	for _, pm := range marketplaceList {
		wg.Add(1)
		go func(marketplace PopularMarketplace) {
			defer wg.Done()

			manifest, err := fetchMarketplaceFromGitHub(marketplace)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", marketplace.Name, err))
				return
			}

			manifests[marketplace.Name] = manifest
		}(pm)
	}

	wg.Wait()

	// If all fetches failed, return error
	if len(manifests) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("all marketplace fetches failed: %v", errs)
	}

	// Log partial failures but continue
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}

	return manifests, nil
}

// fetchMarketplaceFromGitHub fetches a single marketplace with caching
func fetchMarketplaceFromGitHub(pm PopularMarketplace) (*MarketplaceManifest, error) {
	// Try cache first
	cached, err := LoadFromCache(pm.Name)
	if err == nil && cached != nil {
		return cached, nil
	}

	// Cache miss or expired - fetch from GitHub
	manifest, err := FetchManifestFromGitHub(pm.GitHubRepo)
	if err != nil {
		return nil, err
	}

	// Update manifest name to match our hardcoded name
	manifest.Name = pm.Name

	// Save to cache (best effort - don't fail if cache write fails)
	_ = SaveToCache(pm.Name, manifest)

	return manifest, nil
}
