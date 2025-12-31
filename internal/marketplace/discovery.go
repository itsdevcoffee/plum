package marketplace

import (
	"fmt"
	"os"
	"sync"
	"time"
)

const (
	// MaxConcurrentFetches limits parallel marketplace downloads
	MaxConcurrentFetches = 5
)

// PopularMarketplace represents a hardcoded popular marketplace
type PopularMarketplace struct {
	Name        string       `json:"name"`
	DisplayName string       `json:"displayName"`
	Repo        string       `json:"repo"` // Full repo URL (e.g., https://github.com/owner/repo)
	Description string       `json:"description"`
	StaticStats *GitHubStats `json:"staticStats,omitempty"` // Static GitHub stats snapshot (fallback if cache empty)
}

// DiscoveredMarketplace contains a marketplace manifest with source information
type DiscoveredMarketplace struct {
	Manifest *MarketplaceManifest
	Repo     string // Full repo URL for display
	Source   string // Derived CLI source (owner/repo for GitHub, full URL for others)
}

// PopularMarketplaces is the hardcoded list from README.md with static GitHub stats
// Stats snapshot: 2025-12-31 (updated with fresh data including timestamps)
var PopularMarketplaces = []PopularMarketplace{
	{
		Name:        "claude-code-plugins-plus",
		DisplayName: "Claude Code Plugins Plus",
		Repo:        "https://github.com/jeremylongshore/claude-code-plugins-plus-skills",
		Description: "The largest collection with 254 plugins and 185 Agent Skills",
		StaticStats: &GitHubStats{
			Stars:        845,
			Forks:        96,
			LastPushedAt: mustParseTime("2025-12-31T06:30:55Z"),
			OpenIssues:   3,
		},
	},
	{
		Name:        "claude-code-marketplace",
		DisplayName: "Claude Code Marketplace",
		Repo:        "https://github.com/ananddtyagi/cc-marketplace",
		Description: "Community-driven marketplace with granular installation",
		StaticStats: &GitHubStats{
			Stars:        577,
			Forks:        49,
			LastPushedAt: mustParseTime("2025-12-14T22:31:07Z"),
			OpenIssues:   5,
		},
	},
	{
		Name:        "claude-code-plugins",
		DisplayName: "Claude Code Plugins",
		Repo:        "https://github.com/anthropics/claude-code",
		Description: "Official Anthropic plugins maintained by the Claude Code team",
		StaticStats: &GitHubStats{
			Stars:        50055,
			Forks:        3548,
			LastPushedAt: mustParseTime("2025-12-20T19:00:03Z"),
			OpenIssues:   6573,
		},
	},
	{
		Name:        "mag-claude-plugins",
		DisplayName: "MAG Claude Plugins",
		Repo:        "https://github.com/MadAppGang/claude-code",
		Description: "Battle-tested workflows with 4 specialized plugins",
		StaticStats: &GitHubStats{
			Stars:        192,
			Forks:        17,
			LastPushedAt: mustParseTime("2025-12-30T12:23:11Z"),
			OpenIssues:   1,
		},
	},
	{
		Name:        "dev-gom-plugins",
		DisplayName: "Dev-GOM Plugins",
		Repo:        "https://github.com/Dev-GOM/claude-code-marketplace",
		Description: "Automation-focused collection with 15 plugins",
		StaticStats: &GitHubStats{
			Stars:        41,
			Forks:        5,
			LastPushedAt: mustParseTime("2025-12-02T03:56:32Z"),
			OpenIssues:   0,
		},
	},
	{
		Name:        "feedmob-claude-plugins",
		DisplayName: "FeedMob Plugins",
		Repo:        "https://github.com/feed-mob/claude-code-marketplace",
		Description: "Productivity and workflow tools with 6 specialized plugins",
		StaticStats: &GitHubStats{
			Stars:        2,
			Forks:        1,
			LastPushedAt: mustParseTime("2025-12-22T09:15:58Z"),
			OpenIssues:   1,
		},
	},
	{
		Name:        "claude-plugins-official",
		DisplayName: "Claude Plugins Official",
		Repo:        "https://github.com/anthropics/claude-plugins-official",
		Description: "Official Anthropic plugins for Claude Code",
		StaticStats: &GitHubStats{
			Stars:        1158,
			Forks:        127,
			LastPushedAt: mustParseTime("2025-12-26T06:00:12Z"),
			OpenIssues:   69,
		},
	},
	{
		Name:        "anthropic-agent-skills",
		DisplayName: "Anthropic Agent Skills",
		Repo:        "https://github.com/anthropics/skills",
		Description: "Official Anthropic Agent Skills reference repository",
		StaticStats: &GitHubStats{
			Stars:        30756,
			Forks:        2802,
			LastPushedAt: mustParseTime("2025-12-20T18:09:45Z"),
			OpenIssues:   118,
		},
	},
	{
		Name:        "wshobson-agents",
		DisplayName: "Hobson's Agent Collection",
		Repo:        "https://github.com/wshobson/agents",
		Description: "Comprehensive production system with 65 plugins and multi-agent orchestration",
		StaticStats: &GitHubStats{
			Stars:        23995,
			Forks:        2669,
			LastPushedAt: mustParseTime("2025-12-30T21:40:12Z"),
			OpenIssues:   10,
		},
	},
	{
		Name:        "docker-plugins",
		DisplayName: "Docker Official Plugins",
		Repo:        "https://github.com/docker/claude-plugins",
		Description: "Official Docker Inc. marketplace with Docker Desktop MCP Toolkit integration",
		StaticStats: &GitHubStats{
			Stars:        11,
			Forks:        3,
			LastPushedAt: mustParseTime("2025-12-19T19:10:46Z"),
			OpenIssues:   0,
		},
	},
	{
		Name:        "ccplugins-marketplace",
		DisplayName: "CC Plugins Curated",
		Repo:        "https://github.com/ccplugins/marketplace",
		Description: "Curated collection of 200 plugins across 13 categories",
		StaticStats: &GitHubStats{
			Stars:        10,
			Forks:        7,
			LastPushedAt: mustParseTime("2025-10-14T03:38:20Z"),
			OpenIssues:   2,
		},
	},
	{
		Name:        "claude-mem",
		DisplayName: "Claude-Mem",
		Repo:        "https://github.com/thedotmack/claude-mem",
		Description: "Persistent memory compression system for Claude Code with context preservation",
		StaticStats: &GitHubStats{
			Stars:        9729,
			Forks:        587,
			LastPushedAt: mustParseTime("2025-12-31T03:01:45Z"),
			OpenIssues:   21,
		},
	},
}

// mustParseTime parses RFC3339 timestamp or panics (for static data only)
func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic("invalid timestamp in static data: " + s)
	}
	return t
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

	// Save to cache (log error but don't fail)
	if err := SaveToCache(pm.Name, manifest); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save %s to cache: %v\n", pm.Name, err)
	}

	return &DiscoveredMarketplace{
		Manifest: manifest,
		Repo:     pm.Repo,
		Source:   source,
	}, nil
}
