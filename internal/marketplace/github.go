package marketplace

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// GitHubRawBase is the base URL for GitHub raw content
	GitHubRawBase = "https://raw.githubusercontent.com"

	// DefaultBranch to fetch from
	DefaultBranch = "main"

	// HTTPTimeout for fetching marketplace files
	HTTPTimeout = 30 * time.Second
)

// FetchManifestFromGitHub fetches marketplace.json from a GitHub repo
// repo format: "owner/repo-name"
// Returns the parsed manifest or error
func FetchManifestFromGitHub(repo string) (*MarketplaceManifest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), HTTPTimeout)
	defer cancel()

	url := buildRawURL(repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add User-Agent header (GitHub best practice)
	req.Header.Set("User-Agent", "plum-marketplace-browser/0.1.0")

	client := httpClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub returned status %d for %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var manifest MarketplaceManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse marketplace.json: %w", err)
	}

	return &manifest, nil
}

// buildRawURL constructs the raw GitHub URL for marketplace.json
// Example: https://raw.githubusercontent.com/owner/repo/main/.claude-plugin/marketplace.json
func buildRawURL(repo string) string {
	return fmt.Sprintf("%s/%s/%s/.claude-plugin/marketplace.json",
		GitHubRawBase, repo, DefaultBranch)
}

// httpClient returns a configured HTTP client with timeout and sensible defaults
func httpClient() *http.Client {
	return &http.Client{
		Timeout: HTTPTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 5,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}
}
