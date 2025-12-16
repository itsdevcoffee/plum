package marketplace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// CacheTTL is how long cached marketplace data remains valid (24 hours)
	CacheTTL = 24 * time.Hour

	// MaxMarketplaceNameLength limits marketplace name length for security
	MaxMarketplaceNameLength = 100
)

// CacheEntry represents a cached marketplace manifest with metadata
type CacheEntry struct {
	Manifest  *MarketplaceManifest `json:"manifest"`
	FetchedAt time.Time            `json:"fetchedAt"`
	Source    string               `json:"source"`
}

// validateMarketplaceName ensures marketplace name is safe for filesystem use
// Prevents path traversal and injection attacks
func validateMarketplaceName(name string) error {
	if name == "" {
		return fmt.Errorf("marketplace name cannot be empty")
	}

	// Reject path traversal attempts
	if strings.Contains(name, "..") {
		return fmt.Errorf("marketplace name contains path traversal: %q", name)
	}

	// Reject path separators (both Unix and Windows)
	if strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("marketplace name contains path separator: %q", name)
	}

	// Only allow safe characters: alphanumeric, dash, underscore, dot
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.') {
			return fmt.Errorf("marketplace name contains invalid character %q", r)
		}
	}

	// Enforce length limit
	if len(name) > MaxMarketplaceNameLength {
		return fmt.Errorf("marketplace name too long (max %d characters): %d", MaxMarketplaceNameLength, len(name))
	}

	return nil
}

// plumCacheDir is a variable to allow testing with a custom directory
var plumCacheDir = defaultPlumCacheDir

// defaultPlumCacheDir returns the default path to plum's cache directory
func defaultPlumCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}

	// Check for CLAUDE_CONFIG_DIR override (user might want all plum data there)
	if configDir := os.Getenv("CLAUDE_CONFIG_DIR"); configDir != "" {
		return filepath.Join(configDir, "plum", "cache", "marketplaces"), nil
	}

	return filepath.Join(home, ".plum", "cache", "marketplaces"), nil
}

// PlumCacheDir returns the path to plum's cache directory (~/.plum/cache/marketplaces/)
func PlumCacheDir() (string, error) {
	return plumCacheDir()
}

// LoadFromCache loads a marketplace manifest from cache if valid
// Returns nil if cache miss or expired (no error)
func LoadFromCache(marketplaceName string) (*MarketplaceManifest, error) {
	// Validate marketplace name for security
	if err := validateMarketplaceName(marketplaceName); err != nil {
		return nil, err
	}

	cacheDir, err := PlumCacheDir()
	if err != nil {
		return nil, err
	}

	cachePath := filepath.Join(cacheDir, marketplaceName+".json")

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Cache miss - not an error
		}
		return nil, err
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}

	// Check if cache is still valid
	if !isCacheValid(entry) {
		return nil, nil // Expired - not an error
	}

	return entry.Manifest, nil
}

// SaveToCache saves a marketplace manifest to cache using atomic write
func SaveToCache(marketplaceName string, manifest *MarketplaceManifest) error {
	// Validate marketplace name for security
	if err := validateMarketplaceName(marketplaceName); err != nil {
		return err
	}

	cacheDir, err := PlumCacheDir()
	if err != nil {
		return err
	}

	// Create cache directory if it doesn't exist (user-only permissions)
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	entry := CacheEntry{
		Manifest:  manifest,
		FetchedAt: time.Now(),
		Source:    marketplaceName,
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}

	cachePath := filepath.Join(cacheDir, marketplaceName+".json")

	// Atomic write: temp file + rename
	tmpFile, err := os.CreateTemp(cacheDir, ".tmp-"+marketplaceName+"-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // Cleanup on failure

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Set restrictive permissions (user-only read/write)
	if err := os.Chmod(tmpPath, 0600); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Atomic rename (POSIX guarantee)
	if err := os.Rename(tmpPath, cachePath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// isCacheValid checks if cache entry is still valid based on TTL
func isCacheValid(entry CacheEntry) bool {
	return time.Since(entry.FetchedAt) < CacheTTL
}
