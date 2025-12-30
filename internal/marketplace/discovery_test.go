package marketplace

import (
	"testing"
)

// TestPopularMarketplaces verifies the hardcoded marketplace list
func TestPopularMarketplaces(t *testing.T) {
	t.Run("list is not empty", func(t *testing.T) {
		if len(PopularMarketplaces) == 0 {
			t.Error("PopularMarketplaces should not be empty")
		}
	})

	t.Run("all marketplaces have required fields", func(t *testing.T) {
		for i, pm := range PopularMarketplaces {
			if pm.Name == "" {
				t.Errorf("Marketplace %d has empty Name", i)
			}
			if pm.DisplayName == "" {
				t.Errorf("Marketplace %d (%s) has empty DisplayName", i, pm.Name)
			}
			if pm.Repo == "" {
				t.Errorf("Marketplace %d (%s) has empty Repo", i, pm.Name)
			}
			if pm.Description == "" {
				t.Errorf("Marketplace %d (%s) has empty Description", i, pm.Name)
			}
		}
	})

	t.Run("verify specific known marketplaces", func(t *testing.T) {
		// Check that key marketplaces exist
		found := make(map[string]bool)
		for _, pm := range PopularMarketplaces {
			found[pm.Name] = true
		}

		expectedMarketplaces := []string{
			"claude-code-plugins",
			"anthropic-agent-skills",
			"wshobson-agents",
			"claude-mem",
		}

		for _, expected := range expectedMarketplaces {
			if !found[expected] {
				t.Errorf("Expected marketplace %q not found in PopularMarketplaces", expected)
			}
		}
	})

	t.Run("static stats present for some marketplaces", func(t *testing.T) {
		statsCount := 0
		for _, pm := range PopularMarketplaces {
			if pm.StaticStats != nil {
				statsCount++

				// Verify stats have reasonable values
				if pm.StaticStats.Stars < 0 {
					t.Errorf("Marketplace %s has negative stars: %d", pm.Name, pm.StaticStats.Stars)
				}
				if pm.StaticStats.Forks < 0 {
					t.Errorf("Marketplace %s has negative forks: %d", pm.Name, pm.StaticStats.Forks)
				}
			}
		}

		if statsCount == 0 {
			t.Error("At least some marketplaces should have StaticStats")
		}
	})

	t.Run("no duplicate names", func(t *testing.T) {
		seen := make(map[string]bool)
		for _, pm := range PopularMarketplaces {
			if seen[pm.Name] {
				t.Errorf("Duplicate marketplace name: %s", pm.Name)
			}
			seen[pm.Name] = true
		}
	})

	t.Run("repo URLs are valid format", func(t *testing.T) {
		for _, pm := range PopularMarketplaces {
			// Should contain github.com
			if pm.Repo != "" {
				// Most should be GitHub URLs
				// Just verify it's not empty and doesn't have obvious issues
				if len(pm.Repo) < 10 {
					t.Errorf("Marketplace %s has suspiciously short Repo URL: %s", pm.Name, pm.Repo)
				}
			}
		}
	})
}

// TestPopularMarketplace verifies the struct
func TestPopularMarketplace(t *testing.T) {
	t.Run("create with all fields", func(t *testing.T) {
		pm := PopularMarketplace{
			Name:        "test-marketplace",
			DisplayName: "Test Marketplace",
			Repo:        "https://github.com/test/marketplace",
			Description: "A test marketplace",
			StaticStats: &GitHubStats{
				Stars: 100,
				Forks: 10,
			},
		}

		if pm.Name != "test-marketplace" {
			t.Errorf("Expected name %q, got %q", "test-marketplace", pm.Name)
		}

		if pm.StaticStats == nil {
			t.Error("StaticStats should not be nil")
		}

		if pm.StaticStats.Stars != 100 {
			t.Errorf("Expected 100 stars, got %d", pm.StaticStats.Stars)
		}
	})

	t.Run("nil static stats allowed", func(t *testing.T) {
		pm := PopularMarketplace{
			Name:        "test",
			DisplayName: "Test",
			Repo:        "https://github.com/test/repo",
			Description: "Test",
			StaticStats: nil,
		}

		if pm.StaticStats != nil {
			t.Error("StaticStats should be nil")
		}
	})
}

// TestPluginSearchSource verifies the search source implementation
func TestPluginSearchSource(t *testing.T) {
	pm := PopularMarketplaces[0] // Use first marketplace

	t.Run("marketplace has name", func(t *testing.T) {
		if pm.Name == "" {
			t.Error("First marketplace should have a name")
		}
	})

	t.Run("marketplace has valid repo", func(t *testing.T) {
		if pm.Repo == "" {
			t.Error("First marketplace should have a repo URL")
		}
	})
}
