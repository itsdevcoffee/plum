package marketplace

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// GitHubStatsCacheTTL is how long cached GitHub stats remain valid (24 hours)
	GitHubStatsCacheTTL = 24 * time.Hour

	// GitHubAPIBase is the base URL for GitHub API v3
	GitHubAPIBase = "https://api.github.com"
)

// GitHubStats represents repository statistics from GitHub API
type GitHubStats struct {
	Stars        int       `json:"stargazers_count"`
	Forks        int       `json:"forks_count"`
	LastPushedAt time.Time `json:"pushed_at"`
	OpenIssues   int       `json:"open_issues_count"`
}

// GitHubStatsCacheEntry represents cached GitHub stats with metadata
type GitHubStatsCacheEntry struct {
	Stats     *GitHubStats `json:"stats"`
	FetchedAt time.Time    `json:"fetchedAt"`
	Repo      string       `json:"repo"`
}

// FetchGitHubStats fetches repository statistics from GitHub API v3
// repoURL format: "https://github.com/owner/repo" or "owner/repo"
// Returns nil (not error) on failure to allow graceful degradation
func FetchGitHubStats(repoURL string) (*GitHubStats, error) {
	owner, repo, err := extractOwnerRepo(repoURL)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), HTTPTimeout)
	defer cancel()

	url := fmt.Sprintf("%s/repos/%s/%s", GitHubAPIBase, owner, repo)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// GitHub API requires User-Agent and recommends Accept header
	req.Header.Set("User-Agent", "plum-marketplace-browser/0.2.0")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := httpClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch GitHub stats: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		// Non-fatal - allow graceful degradation
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	// Limit response size (same as marketplace manifests)
	limitedBody := io.LimitReader(resp.Body, MaxResponseBodySize)
	body, err := io.ReadAll(limitedBody)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var stats GitHubStats
	if err := json.Unmarshal(body, &stats); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub response: %w", err)
	}

	return &stats, nil
}

// LoadStatsFromCache loads GitHub stats from cache if valid
// Returns nil if cache miss or expired (not an error)
func LoadStatsFromCache(marketplaceName string) (*GitHubStats, error) {
	if err := validateMarketplaceName(marketplaceName); err != nil {
		return nil, err
	}

	cacheDir, err := PlumCacheDir()
	if err != nil {
		return nil, err
	}

	cachePath := filepath.Join(cacheDir, marketplaceName+"_stats.json")

	// #nosec G304 -- cachePath constructed from validated name
	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	var entry GitHubStatsCacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}

	// Check TTL
	if time.Since(entry.FetchedAt) > GitHubStatsCacheTTL {
		return nil, nil // Expired
	}

	return entry.Stats, nil
}

// SaveStatsToCache saves GitHub stats to cache with atomic write
func SaveStatsToCache(marketplaceName string, stats *GitHubStats) error {
	if err := validateMarketplaceName(marketplaceName); err != nil {
		return err
	}

	cacheDir, err := PlumCacheDir()
	if err != nil {
		return err
	}

	// Create cache directory if needed
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	entry := GitHubStatsCacheEntry{
		Stats:     stats,
		FetchedAt: time.Now(),
		Repo:      marketplaceName,
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}

	cachePath := filepath.Join(cacheDir, marketplaceName+"_stats.json")

	// Atomic write: temp file + rename
	tmpFile, err := os.CreateTemp(cacheDir, ".tmp-stats-"+marketplaceName+"-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Chmod(tmpPath, 0600); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	if err := atomicRename(tmpPath, cachePath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// extractOwnerRepo parses owner and repo from GitHub URL
// Supports: "https://github.com/owner/repo", "http://github.com/owner/repo", "owner/repo"
func extractOwnerRepo(repoURL string) (owner, repo string, err error) {
	// Remove protocol and domain
	repoURL = strings.TrimPrefix(repoURL, "https://github.com/")
	repoURL = strings.TrimPrefix(repoURL, "http://github.com/")

	// Remove trailing slashes and .git
	repoURL = strings.TrimSuffix(repoURL, "/")
	repoURL = strings.TrimSuffix(repoURL, ".git")

	parts := strings.Split(repoURL, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid GitHub repo URL: %s", repoURL)
	}

	return parts[0], parts[1], nil
}
