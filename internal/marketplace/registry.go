package marketplace

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	// RegistryURL is the URL to the official marketplace registry
	RegistryURL = "https://raw.githubusercontent.com/itsdevcoffee/plum/main/marketplaces.json"

	// RegistryCacheName for storing the registry
	RegistryCacheName = "_registry"

	// RegistryCacheTTL is how often to check for registry updates (6 hours)
	RegistryCacheTTL = 6 * time.Hour
)

// MarketplaceRegistry represents the registry structure
type MarketplaceRegistry struct {
	Version      string               `json:"version"`
	LastUpdated  string               `json:"lastUpdated"`
	Description  string               `json:"description"`
	Marketplaces []PopularMarketplace `json:"marketplaces"`
}

// RegistryCacheEntry represents a cached registry with metadata
type RegistryCacheEntry struct {
	Registry  *MarketplaceRegistry `json:"registry"`
	FetchedAt time.Time            `json:"fetchedAt"`
}

// FetchRegistry fetches the marketplace registry from GitHub
// Falls back to hardcoded PopularMarketplaces on failure
func FetchRegistry() ([]PopularMarketplace, error) {
	// Try cache first (6-hour TTL for registry)
	cached, err := loadRegistryFromCache()
	if err == nil && cached != nil {
		return cached.Marketplaces, nil
	}

	// Cache miss or expired - fetch from GitHub
	registry, err := fetchRegistryFromGitHub()
	if err != nil {
		// Fallback to hardcoded list
		return PopularMarketplaces, nil
	}

	// Save to cache
	_ = saveRegistryToCache(registry)

	return registry.Marketplaces, nil
}

// FetchRegistryWithComparison fetches registry and compares with current
// Returns new marketplaces count and the full list
// Compares against CACHED registry if available, otherwise uses provided list
func FetchRegistryWithComparison(current []PopularMarketplace) ([]PopularMarketplace, int, error) {
	// IMPORTANT: Load cached registry BEFORE fetching new one for comparison
	cachedRegistry, err := loadRegistryFromCache()
	var compareList []PopularMarketplace
	if err == nil && cachedRegistry != nil {
		// Compare against cached registry (user already updated)
		compareList = cachedRegistry.Marketplaces
	} else {
		// No cached registry - compare against hardcoded list
		compareList = current
	}

	// Now fetch the latest registry (DON'T save to cache yet - only save on Shift+U)
	registry, err := fetchRegistryFromGitHub()
	if err != nil {
		// Return cached list if available, otherwise hardcoded
		if cachedRegistry != nil {
			return cachedRegistry.Marketplaces, 0, nil
		}
		return current, 0, err
	}

	updated := registry.Marketplaces

	// Count new marketplaces
	knownNames := make(map[string]bool)
	for _, m := range compareList {
		knownNames[m.Name] = true
	}

	newCount := 0
	for _, m := range updated {
		if !knownNames[m.Name] {
			newCount++
		}
	}

	return updated, newCount, nil
}

// fetchRegistryFromGitHub fetches the registry from GitHub
func fetchRegistryFromGitHub() (*MarketplaceRegistry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), HTTPTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, RegistryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "plum-marketplace-browser/0.2.0")

	client := httpClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub returned status %d for registry", resp.StatusCode)
	}

	// Limit response body size to prevent DoS
	limitedBody := io.LimitReader(resp.Body, MaxResponseBodySize)
	body, err := io.ReadAll(limitedBody)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}

	// Check if we hit the size limit
	if int64(len(body)) == MaxResponseBodySize {
		var oneByte [1]byte
		if n, _ := resp.Body.Read(oneByte[:]); n > 0 {
			return nil, fmt.Errorf("registry response exceeded %d bytes", MaxResponseBodySize)
		}
	}

	var registry MarketplaceRegistry
	if err := json.Unmarshal(body, &registry); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}

	return &registry, nil
}

// loadRegistryFromCache loads the registry from cache if valid
func loadRegistryFromCache() (*MarketplaceRegistry, error) {
	cacheDir, err := PlumCacheDir()
	if err != nil {
		return nil, err
	}

	cachePath := filepath.Join(cacheDir, RegistryCacheName+".json")

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var entry RegistryCacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}

	// Check if cache is still valid (6-hour TTL for registry)
	if time.Since(entry.FetchedAt) > RegistryCacheTTL {
		return nil, nil // Expired
	}

	return entry.Registry, nil
}

// saveRegistryToCache saves the registry to cache using atomic write
func saveRegistryToCache(registry *MarketplaceRegistry) error {
	cacheDir, err := PlumCacheDir()
	if err != nil {
		return err
	}

	// Create cache directory if it doesn't exist (user-only permissions)
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return err
	}

	entry := RegistryCacheEntry{
		Registry:  registry,
		FetchedAt: time.Now(),
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}

	cachePath := filepath.Join(cacheDir, RegistryCacheName+".json")

	// Atomic write: temp file + rename
	tmpFile, err := os.CreateTemp(cacheDir, ".tmp-"+RegistryCacheName+"-*.json")
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

	// Atomic rename (with Windows fallback)
	if err := atomicRename(tmpPath, cachePath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}
