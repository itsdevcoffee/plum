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
// Stats snapshot: 2026-02-04 (updated with fresh data including timestamps)
var PopularMarketplaces = []PopularMarketplace{
	{
		Name:        "claude-code-plugins-plus",
		DisplayName: "Claude Code Plugins Plus",
		Repo:        "https://github.com/jeremylongshore/claude-code-plugins-plus-skills",
		Description: "The largest collection with 313 plugins and Agent Skills",
		StaticStats: &GitHubStats{
			Stars:        1242,
			Forks:        154,
			LastPushedAt: mustParseTime("2026-02-01T05:24:47Z"),
			OpenIssues:   0,
		},
	},
	{
		Name:        "claude-code-marketplace",
		DisplayName: "Claude Code Marketplace",
		Repo:        "https://github.com/ananddtyagi/cc-marketplace",
		Description: "Community-driven marketplace with 119 plugins",
		StaticStats: &GitHubStats{
			Stars:        634,
			Forks:        52,
			LastPushedAt: mustParseTime("2026-01-18T17:50:15Z"),
			OpenIssues:   6,
		},
	},
	{
		Name:        "claude-code-plugins",
		DisplayName: "Claude Code Plugins",
		Repo:        "https://github.com/anthropics/claude-code",
		Description: "Official Anthropic plugins with 13 plugins maintained by the Claude Code team",
		StaticStats: &GitHubStats{
			Stars:        63817,
			Forks:        4823,
			LastPushedAt: mustParseTime("2026-02-04T00:44:13Z"),
			OpenIssues:   6056,
		},
	},
	{
		Name:        "mag-claude-plugins",
		DisplayName: "MAG Claude Plugins",
		Repo:        "https://github.com/MadAppGang/claude-code",
		Description: "Battle-tested workflows with 8 specialized plugins",
		StaticStats: &GitHubStats{
			Stars:        221,
			Forks:        21,
			LastPushedAt: mustParseTime("2026-02-03T07:52:27Z"),
			OpenIssues:   2,
		},
	},
	{
		Name:        "dev-gom-plugins",
		DisplayName: "Dev-GOM Plugins",
		Repo:        "https://github.com/Dev-GOM/claude-code-marketplace",
		Description: "Automation-focused collection with 14 plugins",
		StaticStats: &GitHubStats{
			Stars:        58,
			Forks:        5,
			LastPushedAt: mustParseTime("2026-01-17T16:54:15Z"),
			OpenIssues:   0,
		},
	},
	{
		Name:        "feedmob-claude-plugins",
		DisplayName: "FeedMob Plugins",
		Repo:        "https://github.com/feed-mob/claude-code-marketplace",
		Description: "Productivity and workflow tools with 7 specialized plugins",
		StaticStats: &GitHubStats{
			Stars:        2,
			Forks:        1,
			LastPushedAt: mustParseTime("2025-12-22T09:15:58Z"),
			OpenIssues:   0,
		},
	},
	{
		Name:        "claude-plugins-official",
		DisplayName: "Claude Plugins Official",
		Repo:        "https://github.com/anthropics/claude-plugins-official",
		Description: "Official Anthropic marketplace with 51 plugins",
		StaticStats: &GitHubStats{
			Stars:        6397,
			Forks:        616,
			LastPushedAt: mustParseTime("2026-01-30T06:58:05Z"),
			OpenIssues:   171,
		},
	},
	{
		Name:        "anthropic-agent-skills",
		DisplayName: "Anthropic Agent Skills",
		Repo:        "https://github.com/anthropics/skills",
		Description: "Official Anthropic Agent Skills reference repository with 2 skills",
		StaticStats: &GitHubStats{
			Stars:        62390,
			Forks:        6119,
			LastPushedAt: mustParseTime("2026-02-04T02:20:55Z"),
			OpenIssues:   229,
		},
	},
	{
		Name:        "wshobson-agents",
		DisplayName: "Hobson's Agent Collection",
		Repo:        "https://github.com/wshobson/agents",
		Description: "Comprehensive production system with 71 plugins and multi-agent orchestration",
		StaticStats: &GitHubStats{
			Stars:        27707,
			Forks:        3055,
			LastPushedAt: mustParseTime("2026-02-02T02:06:12Z"),
			OpenIssues:   12,
		},
	},
	{
		Name:        "docker-plugins",
		DisplayName: "Docker Official Plugins",
		Repo:        "https://github.com/docker/claude-plugins",
		Description: "Official Docker Inc. marketplace with 1 plugin for Docker Desktop MCP",
		StaticStats: &GitHubStats{
			Stars:        13,
			Forks:        6,
			LastPushedAt: mustParseTime("2026-01-30T16:26:32Z"),
			OpenIssues:   1,
		},
	},
	{
		Name:        "claude-mem",
		DisplayName: "Claude-Mem",
		Repo:        "https://github.com/thedotmack/claude-mem",
		Description: "Persistent memory compression system for Claude Code with context preservation",
		StaticStats: &GitHubStats{
			Stars:        20789,
			Forks:        1404,
			LastPushedAt: mustParseTime("2026-02-04T02:09:03Z"),
			OpenIssues:   212,
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
