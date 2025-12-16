package marketplace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	// DefaultBranch to fetch from
	DefaultBranch = "main"

	// HTTPTimeout for fetching marketplace files
	HTTPTimeout = 30 * time.Second

	// MaxResponseBodySize limits HTTP response size to prevent DoS (10 MB)
	MaxResponseBodySize = 10 << 20

	// MaxRetries for transient network failures
	MaxRetries = 3
)

var (
	// GitHubRawBase is the base URL for GitHub raw content (variable for testing)
	GitHubRawBase = "https://raw.githubusercontent.com"
)

var (
	// Singleton HTTP client for connection reuse
	httpClientOnce sync.Once
	httpClientInst *http.Client
)

// httpStatusError wraps HTTP status code errors for retry logic
type httpStatusError struct {
	StatusCode int
	Message    string
}

func (e *httpStatusError) Error() string {
	return e.Message
}

// isRetryableError determines if an error should trigger a retry
// Only retries transient failures: network errors, timeouts, 5xx, and 429
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for HTTP status errors
	var statusErr *httpStatusError
	if errors.As(err, &statusErr) {
		// Retry on 5xx (server errors) and 429 (rate limiting)
		if statusErr.StatusCode >= 500 || statusErr.StatusCode == http.StatusTooManyRequests {
			return true
		}
		// Don't retry on 4xx client errors (except 429 handled above)
		if statusErr.StatusCode >= 400 && statusErr.StatusCode < 500 {
			return false
		}
	}

	// Check for network/timeout errors
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		// Network errors and timeouts are retryable
		return true
	}

	// Context timeout/cancellation
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}

	// For other errors (parsing, etc.), don't retry
	return false
}

// FetchManifestFromGitHub fetches marketplace.json from a GitHub repo with retries
// repo format: "owner/repo-name"
// Returns the parsed manifest or error
func FetchManifestFromGitHub(repo string) (*MarketplaceManifest, error) {
	var lastErr error

	// Retry with exponential backoff for transient failures
	for attempt := 0; attempt < MaxRetries; attempt++ {
		manifest, err := fetchManifestAttempt(repo)
		if err == nil {
			return manifest, nil
		}

		lastErr = err

		// Only retry transient failures (network errors, 5xx, 429)
		if !isRetryableError(err) {
			return nil, err
		}

		// Backoff before retry (except on last attempt): 1s, 2s, 4s
		if attempt < MaxRetries-1 {
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			time.Sleep(backoff)
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", MaxRetries, lastErr)
}

// fetchManifestAttempt performs a single fetch attempt
func fetchManifestAttempt(repo string) (*MarketplaceManifest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), HTTPTimeout)
	defer cancel()

	url := buildRawURL(repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add User-Agent header (GitHub best practice)
	req.Header.Set("User-Agent", "plum-marketplace-browser/0.2.0")

	client := httpClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from GitHub: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, &httpStatusError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("GitHub returned status %d for %s", resp.StatusCode, url),
		}
	}

	// Limit response body size to prevent DoS
	limitedBody := io.LimitReader(resp.Body, MaxResponseBodySize)
	body, err := io.ReadAll(limitedBody)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check if we hit the size limit
	if int64(len(body)) == MaxResponseBodySize {
		// Try reading one more byte to confirm truncation
		var oneByte [1]byte
		if n, _ := resp.Body.Read(oneByte[:]); n > 0 {
			return nil, fmt.Errorf("response body exceeded %d bytes", MaxResponseBodySize)
		}
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

// httpClient returns a singleton HTTP client for connection reuse
func httpClient() *http.Client {
	httpClientOnce.Do(func() {
		httpClientInst = &http.Client{
			Timeout: HTTPTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		}
	})
	return httpClientInst
}
