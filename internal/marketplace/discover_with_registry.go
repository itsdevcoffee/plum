package marketplace

import (
	"fmt"
	"os"
	"sync"
)

// DiscoverWithRegistry fetches marketplaces using the latest registry
// This is called when user presses Shift+U to update
func DiscoverWithRegistry() (map[string]*MarketplaceManifest, error) {
	// Fetch latest marketplace list from registry
	marketplaceList, err := FetchRegistry()
	if err != nil {
		// Fallback to hardcoded
		fmt.Fprintf(os.Stderr, "Warning: failed to fetch registry, using hardcoded list: %v\n", err)
		marketplaceList = PopularMarketplaces
	}

	var (
		manifests = make(map[string]*MarketplaceManifest)
		mu        sync.Mutex
		wg        sync.WaitGroup
		errs      []error
		sem       = make(chan struct{}, MaxConcurrentFetches) // Semaphore for concurrency limiting
	)

	// Fetch all marketplaces with concurrency limit
	for _, pm := range marketplaceList {
		wg.Add(1)
		go func(marketplace PopularMarketplace) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }() // Release semaphore

			// Skip cache - force fresh fetch from GitHub
			manifest, err := FetchManifestFromGitHub(marketplace.Repo)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("%s: %w", marketplace.Name, err))
				mu.Unlock()
				return
			}

			// Update manifest name to match registry
			manifest.Name = marketplace.Name

			// Save to cache (log error but don't fail the fetch)
			if err := SaveToCache(marketplace.Name, manifest); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save %s to cache: %v\n", marketplace.Name, err)
			}

			mu.Lock()
			manifests[marketplace.Name] = manifest
			mu.Unlock()
		}(pm)
	}

	wg.Wait()

	// If all fetches failed, return error
	if len(manifests) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("all marketplace fetches failed: %v", errs)
	}

	// Log partial failures
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}

	return manifests, nil
}
