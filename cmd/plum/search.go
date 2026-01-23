package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/itsdevcoffee/plum/internal/config"
	"github.com/itsdevcoffee/plum/internal/plugin"
	"github.com/itsdevcoffee/plum/internal/search"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for plugins across marketplaces",
	Long: `Search for plugins across all registered and discoverable marketplaces.

Uses fuzzy matching on plugin names, descriptions, and keywords.
Results are ranked by relevance.

Examples:
  plum search memory
  plum search "code review"
  plum search formatting --marketplace=claude-code-plugins
  plum search --json memory`,
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
}

var (
	searchJSON        bool
	searchMarketplace string
	searchCategory    string
	searchLimit       int
)

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "Output as JSON")
	searchCmd.Flags().StringVarP(&searchMarketplace, "marketplace", "m", "", "Filter by marketplace")
	searchCmd.Flags().StringVarP(&searchCategory, "category", "c", "", "Filter by category")
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "n", 20, "Maximum number of results")
}

// SearchResult represents a search result
type SearchResult struct {
	Name              string `json:"name"`
	Marketplace       string `json:"marketplace"`
	Description       string `json:"description"`
	Version           string `json:"version"`
	Category          string `json:"category,omitempty"`
	Installed         bool   `json:"installed"`
	Score             int    `json:"score,omitempty"`
	Installable       bool   `json:"installable"`
	InstallabilityTag string `json:"installabilityTag,omitempty"`
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	// Load all plugins
	plugins, err := config.LoadAllPlugins()
	if err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	// Apply filters before search
	plugins = filterPlugins(plugins, searchMarketplace, searchCategory)

	// Perform search
	ranked := search.Search(query, plugins)

	// Apply limit
	if searchLimit > 0 && len(ranked) > searchLimit {
		ranked = ranked[:searchLimit]
	}

	// Build results
	results := make([]SearchResult, len(ranked))
	for i, r := range ranked {
		results[i] = SearchResult{
			Name:              r.Plugin.Name,
			Marketplace:       r.Plugin.Marketplace,
			Description:       r.Plugin.Description,
			Version:           r.Plugin.Version,
			Category:          r.Plugin.Category,
			Installed:         r.Plugin.Installed,
			Score:             r.Score,
			Installable:       r.Plugin.Installable(),
			InstallabilityTag: r.Plugin.InstallabilityTag(),
		}
	}

	// Output
	if searchJSON {
		return outputSearchJSON(results)
	}
	return outputSearchTable(results, query)
}

func outputSearchJSON(results []SearchResult) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}

func outputSearchTable(results []SearchResult, query string) error {
	if len(results) == 0 {
		fmt.Printf("No plugins found matching '%s'\n", query)
		return nil
	}

	fmt.Printf("Found %d plugin(s) matching '%s':\n\n", len(results), query)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Header
	_, _ = fmt.Fprintln(w, "NAME\tMARKETPLACE\tDESCRIPTION")

	// Track if we have any special indicators to explain in legend
	hasInstalled := false
	hasBuiltIn := false
	hasExternal := false
	hasIncomplete := false

	// Rows
	for _, r := range results {
		desc := r.Description
		// Truncate long descriptions
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}

		// Add indicators to name
		name := r.Name
		if r.Installed {
			name += " *"
			hasInstalled = true
		}
		switch r.InstallabilityTag {
		case "[built-in]":
			name += " " + r.InstallabilityTag
			hasBuiltIn = true
		case "[external]":
			name += " " + r.InstallabilityTag
			hasExternal = true
		case "[incomplete]":
			name += " " + r.InstallabilityTag
			hasIncomplete = true
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", name, r.Marketplace, desc)
	}

	// Print legend
	_, _ = fmt.Fprintln(w)
	if hasInstalled {
		_, _ = fmt.Fprintln(w, "* = installed")
	}
	if hasBuiltIn {
		_, _ = fmt.Fprintln(w, "[built-in] = LSP plugin handled by Claude Code")
	}
	if hasExternal {
		_, _ = fmt.Fprintln(w, "[external] = external repo (install manually)")
	}
	if hasIncomplete {
		_, _ = fmt.Fprintln(w, "[incomplete] = missing plugin.json (not installable)")
	}

	return w.Flush()
}

// filterPlugins applies marketplace and category filters to a plugin list.
func filterPlugins(plugins []plugin.Plugin, marketplace, category string) []plugin.Plugin {
	if marketplace == "" && category == "" {
		return plugins
	}

	var filtered []plugin.Plugin
	for _, p := range plugins {
		if marketplace != "" && p.Marketplace != marketplace {
			continue
		}
		if category != "" && p.Category != category {
			continue
		}
		filtered = append(filtered, p)
	}
	return filtered
}
