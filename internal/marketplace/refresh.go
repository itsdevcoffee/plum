package marketplace

import (
	"fmt"
	"os"
)

// ClearCache removes all cached marketplace data
func ClearCache() error {
	cacheDir, err := PlumCacheDir()
	if err != nil {
		return err
	}

	// Check if cache directory exists
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return nil // Nothing to clear
	}

	// Remove entire cache directory
	if err := os.RemoveAll(cacheDir); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	return nil
}

// RefreshAll clears cache and re-fetches all marketplaces using latest registry
func RefreshAll() error {
	// Clear existing cache
	if err := ClearCache(); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	// Fetch fresh data from registry (this will repopulate cache with ALL marketplaces)
	_, err := DiscoverWithRegistry()
	if err != nil {
		return fmt.Errorf("failed to refresh marketplaces: %w", err)
	}

	return nil
}
