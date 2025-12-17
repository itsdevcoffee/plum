package marketplace

import (
	"fmt"
	"net/url"
	"strings"
)

// DeriveSource converts a repo URL to Claude Code CLI source format
// GitHub: https://github.com/owner/repo → owner/repo
// Others: https://gitlab.com/company/plugins → https://gitlab.com/company/plugins (full URL)
func DeriveSource(repoURL string) (string, error) {
	if repoURL == "" {
		return "", fmt.Errorf("empty repo URL")
	}

	// Parse the URL
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("invalid repo URL: %w", err)
	}

	// Ensure it's a valid HTTP/HTTPS URL with a host
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid repo URL: missing scheme or host")
	}

	// GitHub: extract owner/repo shorthand
	if u.Host == "github.com" {
		path := strings.TrimPrefix(u.Path, "/")
		path = strings.TrimSuffix(path, ".git")

		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			return fmt.Sprintf("%s/%s", parts[0], parts[1]), nil
		}
		return "", fmt.Errorf("invalid GitHub path: %s", u.Path)
	}

	// For non-GitHub (GitLab, Codeberg, etc.), use full URL
	return repoURL, nil
}

// IsGitHubRepo checks if a repo URL is from GitHub
func IsGitHubRepo(repoURL string) bool {
	u, err := url.Parse(repoURL)
	if err != nil {
		return false
	}
	return u.Host == "github.com"
}
