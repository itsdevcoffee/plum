package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/itsdevcoffee/plum/internal/config"
	"github.com/itsdevcoffee/plum/internal/marketplace"
	"github.com/itsdevcoffee/plum/internal/settings"
	"github.com/spf13/cobra"
)

var marketplaceCmd = &cobra.Command{
	Use:   "marketplace",
	Short: "Manage plugin marketplaces",
	Long: `Manage plugin marketplaces.

Marketplaces are sources of plugins that Plum can search and install from.

Available subcommands:
  list     List all registered and discoverable marketplaces`,
}

var marketplaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all marketplaces",
	Long: `List all registered and discoverable marketplaces.

Shows marketplace name, source repository, plugin count, and installation status.

Examples:
  plum marketplace list
  plum marketplace list --json`,
	RunE: runMarketplaceList,
}

var (
	marketplaceListJSON    bool
	marketplaceListProject string
)

func init() {
	rootCmd.AddCommand(marketplaceCmd)
	marketplaceCmd.AddCommand(marketplaceListCmd)

	marketplaceListCmd.Flags().BoolVar(&marketplaceListJSON, "json", false, "Output as JSON")
	marketplaceListCmd.Flags().StringVar(&marketplaceListProject, "project", "", "Project path (default: current directory)")
}

// MarketplaceListItem represents a marketplace in the list output
type MarketplaceListItem struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName,omitempty"`
	Repo        string `json:"repo"`
	Description string `json:"description,omitempty"`
	PluginCount int    `json:"pluginCount"`
	Installed   bool   `json:"installed"`
	Source      string `json:"source,omitempty"`
	Stars       int    `json:"stars,omitempty"`
}

func runMarketplaceList(cmd *cobra.Command, args []string) error {
	// Load known marketplaces from Claude Code
	known, err := config.LoadKnownMarketplaces()
	if err != nil {
		// Not fatal - just means no marketplaces installed yet
		known = make(config.KnownMarketplaces)
	}

	// Load extra marketplaces from settings
	extra, _ := settings.AllMarketplaces(marketplaceListProject)

	// Build list of items from popular marketplaces (discoverable)
	items := make([]MarketplaceListItem, 0)
	seenNames := make(map[string]bool)

	// Add popular marketplaces
	for _, pm := range marketplace.PopularMarketplaces {
		_, isInstalled := known[pm.Name]

		item := MarketplaceListItem{
			Name:        pm.Name,
			DisplayName: pm.DisplayName,
			Repo:        pm.Repo,
			Description: pm.Description,
			Installed:   isInstalled,
		}

		// Add stats if available
		if pm.StaticStats != nil {
			item.Stars = pm.StaticStats.Stars
		}

		// Count plugins from cached manifest
		if cached, err := marketplace.LoadFromCache(pm.Name); err == nil && cached != nil {
			item.PluginCount = len(cached.Plugins)
		}

		items = append(items, item)
		seenNames[pm.Name] = true
	}

	// Add any installed marketplaces not in popular list
	for name, entry := range known {
		if seenNames[name] {
			continue
		}

		items = append(items, MarketplaceListItem{
			Name:      name,
			Repo:      entry.Source.Repo,
			Installed: true,
			Source:    entry.Source.Source,
		})
		seenNames[name] = true
	}

	// Add extra marketplaces from settings
	for name, em := range extra {
		if seenNames[name] {
			continue
		}

		items = append(items, MarketplaceListItem{
			Name:      name,
			Repo:      em.Source.Repo,
			Installed: false, // Extra marketplaces are config references
			Source:    em.Source.Source,
		})
	}

	// Output
	if marketplaceListJSON {
		return outputMarketplaceListJSON(items)
	}
	return outputMarketplaceListTable(items)
}

func outputMarketplaceListJSON(items []MarketplaceListItem) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(items)
}

func outputMarketplaceListTable(items []MarketplaceListItem) error {
	if len(items) == 0 {
		fmt.Println("No marketplaces found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Header
	_, _ = fmt.Fprintln(w, "NAME\tDESCRIPTION\tPLUGINS\tSTARS\tSTATUS")

	// Rows
	for _, item := range items {
		desc := item.Description
		if desc == "" && item.DisplayName != "" {
			desc = item.DisplayName
		}
		// Truncate long descriptions
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}

		plugins := "-"
		if item.PluginCount > 0 {
			plugins = fmt.Sprintf("%d", item.PluginCount)
		}

		stars := "-"
		if item.Stars > 0 {
			if item.Stars >= 1000 {
				stars = fmt.Sprintf("%.1fk", float64(item.Stars)/1000)
			} else {
				stars = fmt.Sprintf("%d", item.Stars)
			}
		}

		status := "discoverable"
		if item.Installed {
			status = "installed"
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			item.Name,
			desc,
			plugins,
			stars,
			status,
		)
	}

	return w.Flush()
}
