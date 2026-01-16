package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/itsdevcoffee/plum/internal/config"
	"github.com/itsdevcoffee/plum/internal/settings"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed plugins",
	Long: `List all installed plugins across all scopes.

Shows plugin name, marketplace, scope, status (enabled/disabled), and version.

Examples:
  plum list                  # List all plugins
  plum list --scope=user     # List only user-scoped plugins
  plum list --enabled        # List only enabled plugins
  plum list --updates        # Show available updates inline
  plum list --json           # Output as JSON`,
	RunE: runList,
}

var (
	listScope    string
	listEnabled  bool
	listDisabled bool
	listUpdates  bool
	listJSON     bool
	listProject  string
)

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVarP(&listScope, "scope", "s", "", "Filter by scope (user, project, local)")
	listCmd.Flags().BoolVar(&listEnabled, "enabled", false, "Show only enabled plugins")
	listCmd.Flags().BoolVar(&listDisabled, "disabled", false, "Show only disabled plugins")
	listCmd.Flags().BoolVar(&listUpdates, "updates", false, "Show available updates inline")
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output as JSON")
	listCmd.Flags().StringVar(&listProject, "project", "", "Project path (default: current directory)")
}

// PluginListItem represents a plugin in the list output
type PluginListItem struct {
	Name          string `json:"name"`
	Marketplace   string `json:"marketplace"`
	Scope         string `json:"scope"`
	Status        string `json:"status"`
	Version       string `json:"version"`
	LatestVersion string `json:"latestVersion,omitempty"`
	UpdateAvail   bool   `json:"updateAvailable,omitempty"`
	Installed     bool   `json:"installed"`
}

func runList(cmd *cobra.Command, args []string) error {
	// Load installed plugins from Claude Code's registry
	installed, err := config.LoadInstalledPlugins()
	if err != nil {
		return fmt.Errorf("failed to load installed plugins: %w", err)
	}

	// Load plugin states from settings.json files
	states, err := settings.MergedPluginStates(listProject)
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	// Apply scope filter if specified
	if listScope != "" {
		scope, err := settings.ParseScope(listScope)
		if err != nil {
			return err
		}
		states = settings.FilterByScope(states, scope)
	}

	// Apply enabled/disabled filter
	if listEnabled {
		states = settings.FilterEnabled(states)
	} else if listDisabled {
		states = settings.FilterDisabled(states)
	}

	// Build lookup for latest versions if --updates flag is set
	latestVersions := make(map[string]string)
	if listUpdates {
		allPlugins, err := config.LoadAllPlugins()
		if err == nil {
			for _, p := range allPlugins {
				fullName := p.Name + "@" + p.Marketplace
				latestVersions[fullName] = p.Version
			}
		}
	}

	// Build list items
	items := make([]PluginListItem, 0, len(states))
	for _, state := range states {
		// Parse plugin@marketplace
		parts := strings.SplitN(state.FullName, "@", 2)
		name := parts[0]
		marketplace := ""
		if len(parts) > 1 {
			marketplace = parts[1]
		}

		// Get version from installed plugins registry
		version := ""
		isInstalled := false
		if installs, ok := installed.Plugins[state.FullName]; ok && len(installs) > 0 {
			version = installs[0].Version
			isInstalled = true
		}

		status := "disabled"
		if state.Enabled {
			status = "enabled"
		}

		item := PluginListItem{
			Name:        name,
			Marketplace: marketplace,
			Scope:       state.Scope.String(),
			Status:      status,
			Version:     version,
			Installed:   isInstalled,
		}

		// Check for updates if --updates flag is set
		if listUpdates && version != "" {
			if latest, ok := latestVersions[state.FullName]; ok && latest != "" {
				if isNewerVersion(latest, version) {
					item.LatestVersion = latest
					item.UpdateAvail = true
				}
			}
		}

		items = append(items, item)
	}

	// Output
	if listJSON {
		return outputJSON(items)
	}
	return outputTable(items)
}

func outputJSON(items []PluginListItem) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(items)
}

func outputTable(items []PluginListItem) error {
	if len(items) == 0 {
		fmt.Println("No plugins found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Header
	_, _ = fmt.Fprintln(w, "NAME\tMARKETPLACE\tSCOPE\tSTATUS\tVERSION")

	// Rows
	for _, item := range items {
		version := item.Version
		if version == "" {
			version = "-"
		}
		// Show update info inline if available
		if item.UpdateAvail && item.LatestVersion != "" {
			version = fmt.Sprintf("%s → %s available", version, item.LatestVersion)
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			item.Name,
			item.Marketplace,
			item.Scope,
			item.Status,
			version,
		)
	}

	return w.Flush()
}
