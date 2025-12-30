package marketplace

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestExtractOwnerRepo verifies GitHub URL parsing
func TestExtractOwnerRepo(t *testing.T) {
	tests := []struct {
		name        string
		repoURL     string
		expectOwner string
		expectRepo  string
		expectError bool
	}{
		{
			name:        "https URL",
			repoURL:     "https://github.com/owner/repo",
			expectOwner: "owner",
			expectRepo:  "repo",
		},
		{
			name:        "http URL",
			repoURL:     "http://github.com/owner/repo",
			expectOwner: "owner",
			expectRepo:  "repo",
		},
		{
			name:        "owner/repo format",
			repoURL:     "owner/repo",
			expectOwner: "owner",
			expectRepo:  "repo",
		},
		{
			name:        "trailing slash",
			repoURL:     "https://github.com/owner/repo/",
			expectOwner: "owner",
			expectRepo:  "repo",
		},
		{
			name:        "trailing .git",
			repoURL:     "https://github.com/owner/repo.git",
			expectOwner: "owner",
			expectRepo:  "repo",
		},
		{
			name:        "trailing slash and .git",
			repoURL:     "https://github.com/owner/repo.git/",
			expectOwner: "owner",
			expectRepo:  "repo", // Slash removed first (repo.git), then .git removed (repo)
		},
		{
			name:        "nested path ignored",
			repoURL:     "https://github.com/owner/repo/tree/main/plugins",
			expectOwner: "owner",
			expectRepo:  "repo",
		},
		{
			name:        "invalid - no slash",
			repoURL:     "invalid",
			expectError: true,
		},
		{
			name:        "invalid - empty",
			repoURL:     "",
			expectError: true,
		},
		{
			name:        "invalid - only owner",
			repoURL:     "owner",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := extractOwnerRepo(tt.repoURL)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if owner != tt.expectOwner {
				t.Errorf("Expected owner %q, got %q", tt.expectOwner, owner)
			}

			if repo != tt.expectRepo {
				t.Errorf("Expected repo %q, got %q", tt.expectRepo, repo)
			}
		})
	}
}

// TestGitHubStatsCache verifies cache save/load functionality
func TestGitHubStatsCache(t *testing.T) {
	// Use temp directory for testing
	tmpDir := t.TempDir()
	originalPlumCacheDir := plumCacheDir
	plumCacheDir = func() (string, error) {
		return tmpDir, nil
	}
	defer func() { plumCacheDir = originalPlumCacheDir }()

	t.Run("save and load stats", func(t *testing.T) {
		stats := &GitHubStats{
			Stars:        1234,
			Forks:        567,
			LastPushedAt: time.Date(2025, 12, 30, 10, 0, 0, 0, time.UTC),
			OpenIssues:   42,
		}

		// Save stats
		err := SaveStatsToCache("test-marketplace", stats)
		if err != nil {
			t.Fatalf("SaveStatsToCache failed: %v", err)
		}

		// Verify file exists with correct permissions
		cachePath := filepath.Join(tmpDir, "test-marketplace_stats.json")
		info, err := os.Stat(cachePath)
		if err != nil {
			t.Fatalf("Cache file not created: %v", err)
		}

		if info.Mode().Perm() != 0600 {
			t.Errorf("Expected permissions 0600, got %o", info.Mode().Perm())
		}

		// Load stats
		loaded, err := LoadStatsFromCache("test-marketplace")
		if err != nil {
			t.Fatalf("LoadStatsFromCache failed: %v", err)
		}

		if loaded == nil {
			t.Fatal("LoadStatsFromCache returned nil")
		}

		if loaded.Stars != stats.Stars {
			t.Errorf("Expected %d stars, got %d", stats.Stars, loaded.Stars)
		}
		if loaded.Forks != stats.Forks {
			t.Errorf("Expected %d forks, got %d", stats.Forks, loaded.Forks)
		}
		if loaded.OpenIssues != stats.OpenIssues {
			t.Errorf("Expected %d issues, got %d", stats.OpenIssues, loaded.OpenIssues)
		}
	})

	t.Run("load non-existent cache", func(t *testing.T) {
		loaded, err := LoadStatsFromCache("nonexistent-marketplace")

		if err != nil {
			t.Errorf("Expected nil error for missing cache, got: %v", err)
		}

		if loaded != nil {
			t.Error("Expected nil for missing cache, got stats")
		}
	})

	t.Run("expired cache returns nil", func(t *testing.T) {
		stats := &GitHubStats{Stars: 100}

		// Save stats
		err := SaveStatsToCache("expired-test", stats)
		if err != nil {
			t.Fatalf("SaveStatsToCache failed: %v", err)
		}

		// Manually modify the cache file to have old timestamp
		cachePath := filepath.Join(tmpDir, "expired-test_stats.json")
		oldEntry := GitHubStatsCacheEntry{
			Stats:     stats,
			FetchedAt: time.Now().Add(-25 * time.Hour), // Older than 24h TTL
			Repo:      "expired-test",
		}

		// Write old entry
		data, _ := os.ReadFile(cachePath)
		oldData := []byte(`{"stats":{"stargazers_count":100,"forks_count":0,"pushed_at":"0001-01-01T00:00:00Z","open_issues_count":0},"fetchedAt":"2020-01-01T00:00:00Z","repo":"expired-test"}`)
		err = os.WriteFile(cachePath, oldData, 0600)
		if err != nil {
			t.Fatalf("Failed to write old cache: %v", err)
		}

		// Load should return nil (expired)
		loaded, err := LoadStatsFromCache("expired-test")
		if err != nil {
			t.Errorf("Expected nil error, got: %v", err)
		}

		if loaded != nil {
			t.Error("Expected nil for expired cache, got stats")
		}

		// Cleanup
		_ = data
		_ = oldEntry
	})

	t.Run("invalid marketplace name rejected", func(t *testing.T) {
		stats := &GitHubStats{Stars: 100}

		// Path traversal attempt
		err := SaveStatsToCache("../etc/passwd", stats)
		if err == nil {
			t.Error("SaveStatsToCache should reject path traversal")
		}

		// Load with invalid name
		_, err = LoadStatsFromCache("../etc/passwd")
		if err == nil {
			t.Error("LoadStatsFromCache should reject path traversal")
		}
	})

	t.Run("cache directory created with correct permissions", func(t *testing.T) {
		newTmpDir := filepath.Join(t.TempDir(), "new-cache-dir")
		plumCacheDir = func() (string, error) {
			return newTmpDir, nil
		}

		stats := &GitHubStats{Stars: 100}
		err := SaveStatsToCache("test", stats)
		if err != nil {
			t.Fatalf("SaveStatsToCache failed: %v", err)
		}

		// Verify directory permissions
		info, err := os.Stat(newTmpDir)
		if err != nil {
			t.Fatalf("Cache directory not created: %v", err)
		}

		if info.Mode().Perm() != 0700 {
			t.Errorf("Expected directory permissions 0700, got %o", info.Mode().Perm())
		}
	})
}

// TestGitHubStatsCacheEntry verifies the cache entry structure
func TestGitHubStatsCacheEntry(t *testing.T) {
	t.Run("cache entry fields", func(t *testing.T) {
		stats := &GitHubStats{
			Stars:      1000,
			Forks:      100,
			OpenIssues: 50,
		}

		entry := GitHubStatsCacheEntry{
			Stats:     stats,
			FetchedAt: time.Now(),
			Repo:      "test-repo",
		}

		if entry.Stats.Stars != 1000 {
			t.Errorf("Expected 1000 stars, got %d", entry.Stats.Stars)
		}

		if entry.Repo != "test-repo" {
			t.Errorf("Expected repo %q, got %q", "test-repo", entry.Repo)
		}

		if entry.FetchedAt.IsZero() {
			t.Error("FetchedAt should not be zero")
		}
	})
}

// TestGitHubStatsStruct verifies GitHubStats structure
func TestGitHubStatsStruct(t *testing.T) {
	t.Run("create stats with values", func(t *testing.T) {
		pushedAt := time.Date(2025, 12, 30, 10, 0, 0, 0, time.UTC)

		stats := GitHubStats{
			Stars:        5000,
			Forks:        1000,
			LastPushedAt: pushedAt,
			OpenIssues:   123,
		}

		if stats.Stars != 5000 {
			t.Errorf("Expected 5000 stars, got %d", stats.Stars)
		}

		if stats.Forks != 1000 {
			t.Errorf("Expected 1000 forks, got %d", stats.Forks)
		}

		if stats.OpenIssues != 123 {
			t.Errorf("Expected 123 issues, got %d", stats.OpenIssues)
		}

		if !stats.LastPushedAt.Equal(pushedAt) {
			t.Error("LastPushedAt mismatch")
		}
	})

	t.Run("zero values", func(t *testing.T) {
		stats := GitHubStats{}

		if stats.Stars != 0 {
			t.Error("Expected 0 stars by default")
		}

		if stats.LastPushedAt.IsZero() == false {
			t.Error("Expected zero time by default")
		}
	})
}
