package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
  list     List all registered and discoverable marketplaces
  add      Add a custom marketplace
  remove   Remove a custom marketplace`,
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

// marketplace add command
var marketplaceAddCmd = &cobra.Command{
	Use:   "add <repo>",
	Short: "Add a custom marketplace",
	Long: `Add a custom marketplace to your settings.

The marketplace is specified as a GitHub repository in the format owner/repo.
You can optionally pin to a specific version or commit using #ref syntax.

Custom marketplaces are stored in extraKnownMarketplaces in your settings.json.

Examples:
  plum marketplace add myorg/my-plugins
  plum marketplace add myorg/my-plugins#v2.0.0     # Pin to tag
  plum marketplace add myorg/my-plugins#abc123     # Pin to commit
  plum marketplace add myorg/my-plugins --scope=project`,
	Args: cobra.ExactArgs(1),
	RunE: runMarketplaceAdd,
}

var (
	marketplaceAddScope   string
	marketplaceAddProject string
)

func init() {
	marketplaceCmd.AddCommand(marketplaceAddCmd)

	marketplaceAddCmd.Flags().StringVarP(&marketplaceAddScope, "scope", "s", "user", "Settings scope (user, project, local)")
	marketplaceAddCmd.Flags().StringVar(&marketplaceAddProject, "project", "", "Project path (default: current directory)")
}

func runMarketplaceAdd(cmd *cobra.Command, args []string) error {
	repoArg := args[0]

	// Parse scope
	scope, err := settings.ParseScope(marketplaceAddScope)
	if err != nil {
		return err
	}

	// Validate scope is writable
	if !scope.IsWritable() {
		return fmt.Errorf("cannot write to %s scope (read-only)", scope)
	}

	// Parse repo and optional ref
	repo := repoArg
	ref := ""
	if idx := strings.LastIndex(repoArg, "#"); idx > 0 {
		repo = repoArg[:idx]
		ref = repoArg[idx+1:]
	}

	// Validate repo format (should be owner/repo)
	if !strings.Contains(repo, "/") {
		return fmt.Errorf("invalid repo format: expected owner/repo, got %s", repo)
	}

	// Derive marketplace name from repo
	parts := strings.Split(repo, "/")
	name := parts[len(parts)-1] // Use repo name as marketplace name

	// Build source
	source := settings.MarketplaceSource{
		Source: "github",
		Repo:   repo,
	}

	// Add ref to repo if specified
	if ref != "" {
		source.Repo = repo + "#" + ref
	}

	// Add to settings
	if err := settings.AddMarketplace(name, source, scope, marketplaceAddProject); err != nil {
		return fmt.Errorf("failed to add marketplace: %w", err)
	}

	fmt.Printf("Added marketplace '%s' (%s) to %s scope\n", name, repo, scope)
	if ref != "" {
		fmt.Printf("Pinned to: %s\n", ref)
	}

	return nil
}

// marketplace remove command
var marketplaceRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a custom marketplace",
	Long: `Remove a custom marketplace from your settings.

This removes the marketplace from extraKnownMarketplaces in your settings.json.
It does not affect any plugins you have installed from that marketplace.

Examples:
  plum marketplace remove my-plugins
  plum marketplace remove my-plugins --scope=project`,
	Args: cobra.ExactArgs(1),
	RunE: runMarketplaceRemove,
}

var (
	marketplaceRemoveScope   string
	marketplaceRemoveProject string
)

func init() {
	marketplaceCmd.AddCommand(marketplaceRemoveCmd)

	marketplaceRemoveCmd.Flags().StringVarP(&marketplaceRemoveScope, "scope", "s", "user", "Settings scope (user, project, local)")
	marketplaceRemoveCmd.Flags().StringVar(&marketplaceRemoveProject, "project", "", "Project path (default: current directory)")
}

// marketplace refresh command
var marketplaceRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh marketplace catalog",
	Long: `Fetch fresh marketplace data from GitHub.

This clears the local marketplace cache and fetches the latest plugin listings
from all known marketplaces. Use this to see newly added plugins or updated
versions.

By default, this only refreshes the catalog (plugin listings). Use --update
to also update all installed plugins to their latest versions.

Note: 'plum update' compares against cached marketplace data. Run 'plum marketplace
refresh' first to ensure you have the latest version information.

Examples:
  plum marketplace refresh              # Refresh catalog only
  plum marketplace refresh --update     # Refresh catalog and update all plugins`,
	RunE: runMarketplaceRefresh,
}

var (
	marketplaceRefreshUpdate  bool
	marketplaceRefreshProject string
)

func init() {
	marketplaceCmd.AddCommand(marketplaceRefreshCmd)

	marketplaceRefreshCmd.Flags().BoolVar(&marketplaceRefreshUpdate, "update", false, "Also update all installed plugins after refresh")
	marketplaceRefreshCmd.Flags().StringVar(&marketplaceRefreshProject, "project", "", "Project path for --update (default: current directory)")
}

func runMarketplaceRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Parse scope
	scope, err := settings.ParseScope(marketplaceRemoveScope)
	if err != nil {
		return err
	}

	// Validate scope is writable
	if !scope.IsWritable() {
		return fmt.Errorf("cannot write to %s scope (read-only)", scope)
	}

	// Check if marketplace exists in this scope
	existing, _ := settings.LoadSettings(scope, marketplaceRemoveProject)
	if existing == nil || existing.ExtraKnownMarketplaces == nil {
		return fmt.Errorf("marketplace '%s' not found in %s scope", name, scope)
	}
	if _, ok := existing.ExtraKnownMarketplaces[name]; !ok {
		return fmt.Errorf("marketplace '%s' not found in %s scope", name, scope)
	}

	// Remove from settings
	if err := settings.RemoveMarketplace(name, scope, marketplaceRemoveProject); err != nil {
		return fmt.Errorf("failed to remove marketplace: %w", err)
	}

	fmt.Printf("Removed marketplace '%s' from %s scope\n", name, scope)

	return nil
}

func runMarketplaceRefresh(cmd *cobra.Command, args []string) error {
	fmt.Println("Refreshing marketplace catalog...")

	// Use RefreshAll from marketplace package
	if err := marketplace.RefreshAll(); err != nil {
		return fmt.Errorf("failed to refresh marketplaces: %w", err)
	}

	// Count how many marketplaces were refreshed
	discovered, _ := marketplace.DiscoverPopularMarketplaces()
	fmt.Printf("Refreshed %d marketplace(s)\n", len(discovered))

	// If --update flag, also update plugins
	if marketplaceRefreshUpdate {
		fmt.Println("\nUpdating installed plugins...")

		// Get list of installed plugins
		states, err := settings.MergedPluginStates(marketplaceRefreshProject)
		if err != nil {
			return fmt.Errorf("failed to load plugin states: %w", err)
		}

		if len(states) == 0 {
			fmt.Println("No plugins installed")
			return nil
		}

		// Run update for all plugins using explicit options (no shared state)
		opts := updateOptions{
			Scope:   "",
			Project: marketplaceRefreshProject,
			DryRun:  false,
		}
		return performUpdate(cmd, []string{}, opts)
	}

	return nil
}
