package search

import (
	"sort"
	"strings"

	"github.com/maskkiller/plum/internal/plugin"
	"github.com/sahilm/fuzzy"
)

// RankedPlugin wraps a plugin with its search score
type RankedPlugin struct {
	Plugin plugin.Plugin
	Score  int
}

// Search performs fuzzy search on plugins and returns ranked results
func Search(query string, plugins []plugin.Plugin) []RankedPlugin {
	if query == "" {
		// Return all plugins sorted by name when no query
		results := make([]RankedPlugin, len(plugins))
		for i, p := range plugins {
			results[i] = RankedPlugin{Plugin: p, Score: 0}
		}
		sort.Slice(results, func(i, j int) bool {
			// Installed plugins first, then by name
			if results[i].Plugin.Installed != results[j].Plugin.Installed {
				return results[i].Plugin.Installed
			}
			return results[i].Plugin.Name < results[j].Plugin.Name
		})
		return results
	}

	query = strings.ToLower(query)
	var results []RankedPlugin

	for _, p := range plugins {
		score := scorePlugin(query, p)
		if score > 0 {
			results = append(results, RankedPlugin{Plugin: p, Score: score})
		}
	}

	// Sort by score descending, then by installed status, then by name
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		if results[i].Plugin.Installed != results[j].Plugin.Installed {
			return results[i].Plugin.Installed
		}
		return results[i].Plugin.Name < results[j].Plugin.Name
	})

	return results
}

// scorePlugin calculates a relevance score for a plugin given a query
func scorePlugin(query string, p plugin.Plugin) int {
	score := 0
	lowerName := strings.ToLower(p.Name)
	lowerDesc := strings.ToLower(p.Description)
	lowerCategory := strings.ToLower(p.Category)

	// Exact name match: +100 points
	if lowerName == query {
		score += 100
	} else if strings.Contains(lowerName, query) {
		// Partial name match: +70 points
		score += 70
	} else {
		// Fuzzy name match
		nameMatches := fuzzy.Find(query, []string{lowerName})
		if len(nameMatches) > 0 {
			// Scale fuzzy score (0-100) to 0-50 points
			score += nameMatches[0].Score / 2
		}
	}

	// Keyword exact match: +30 per keyword
	for _, kw := range p.Keywords {
		lowerKw := strings.ToLower(kw)
		if lowerKw == query {
			score += 30
		} else if strings.Contains(lowerKw, query) {
			score += 20
		}
	}

	// Category match: +15 points
	if strings.Contains(lowerCategory, query) {
		score += 15
	}

	// Description fuzzy match: +20 * match score
	if strings.Contains(lowerDesc, query) {
		score += 25
	} else {
		descMatches := fuzzy.Find(query, []string{lowerDesc})
		if len(descMatches) > 0 {
			score += descMatches[0].Score / 5
		}
	}

	// Boost installed plugins slightly
	if p.Installed && score > 0 {
		score += 5
	}

	return score
}

// PluginSearchSource implements fuzzy.Source for plugins
type PluginSearchSource struct {
	Plugins []plugin.Plugin
}

// String returns the searchable string for item at index i
func (s PluginSearchSource) String(i int) string {
	p := s.Plugins[i]
	return p.Name + " " + p.Description + " " + strings.Join(p.Keywords, " ")
}

// Len returns the number of items
func (s PluginSearchSource) Len() int {
	return len(s.Plugins)
}
