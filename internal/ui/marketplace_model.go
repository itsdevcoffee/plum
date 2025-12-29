package ui

import (
	"github.com/itsdevcoffee/plum/internal/marketplace"
)

// MarketplaceStatus represents the installation status of a marketplace
type MarketplaceStatus int

const (
	MarketplaceInstalled MarketplaceStatus = iota // Added to Claude Code
	MarketplaceCached                             // Fetched but not installed
	MarketplaceAvailable                          // In registry, not cached
	MarketplaceNew                                // Recently added to registry
)

// MarketplaceItem represents a marketplace with enriched display data
type MarketplaceItem struct {
	Name                 string                   // Internal name
	DisplayName          string                   // User-facing name
	Repo                 string                   // GitHub repo URL
	Description          string                   // One-line description
	Status               MarketplaceStatus        // Installation status
	InstalledPluginCount int                      // Plugins you have installed
	TotalPluginCount     int                      // Total plugins available
	GitHubStats          *marketplace.GitHubStats // GitHub repo stats (may be nil)
	StatsLoading         bool                     // True while fetching stats
	StatsError           error                    // Stats fetch error if any
}

// MarketplaceSortMode represents sorting options for marketplaces
type MarketplaceSortMode int

const (
	SortByPluginCount MarketplaceSortMode = iota // Most plugins first
	SortByStars                                  // Most stars first
	SortByName                                   // Alphabetical
	SortByLastUpdated                            // Most recently updated first
)

// MarketplaceSortModeNames for display
var MarketplaceSortModeNames = []string{"Plugins", "Stars", "Name", "Updated"}

// StatusBadge returns a display badge for marketplace status
func (m MarketplaceItem) StatusBadge() string {
	switch m.Status {
	case MarketplaceInstalled:
		return InstalledBadge.String()
	case MarketplaceCached:
		return "[Cached]"
	case MarketplaceNew:
		return "[New]"
	default:
		return AvailableBadge.String()
	}
}
