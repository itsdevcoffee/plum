package marketplace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// CacheTTL is how long cached marketplace data remains valid (24 hours)
	CacheTTL = 24 * time.Hour
)

// CacheEntry represents a cached marketplace manifest with metadata
type CacheEntry struct {
	Manifest  *MarketplaceManifest `json:"manifest"`
	FetchedAt time.Time            `json:"fetchedAt"`
	Source    string               `json:"source"`
}

// PlumCacheDir returns the path to plum's cache directory (~/.plum/cache/marketplaces/)
func PlumCacheDir() (string, error) {
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

// LoadFromCache loads a marketplace manifest from cache if valid
// Returns nil if cache miss or expired (no error)
func LoadFromCache(marketplaceName string) (*MarketplaceManifest, error) {
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

// SaveToCache saves a marketplace manifest to cache
func SaveToCache(marketplaceName string, manifest *MarketplaceManifest) error {
	cacheDir, err := PlumCacheDir()
	if err != nil {
		return err
	}

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
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
	return os.WriteFile(cachePath, data, 0644)
}

// isCacheValid checks if cache entry is still valid based on TTL
func isCacheValid(entry CacheEntry) bool {
	return time.Since(entry.FetchedAt) < CacheTTL
}
