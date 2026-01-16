package main

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/itsdevcoffee/plum/internal/config"
	"github.com/itsdevcoffee/plum/internal/settings"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update [plugin]",
	Short: "Update plugins",
	Long: `Update installed plugins to their latest versions.

Without arguments, updates all installed plugins. Optionally specify one
or more plugins to update only those.

The plugin can be specified as:
  - plugin-name (updates first matching installed plugin)
  - plugin-name@marketplace (specific marketplace)

Examples:
  plum update                      # Update all plugins
  plum update ralph-wiggum         # Update specific plugin
  plum update --dry-run            # Check for updates without installing
  plum update --scope=project      # Only update project-scoped plugins`,
	RunE: runUpdate,
}

var (
	updateScope   string
	updateProject string
	updateDryRun  bool
)

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().StringVarP(&updateScope, "scope", "s", "", "Filter by scope (user, project, local)")
	updateCmd.Flags().StringVar(&updateProject, "project", "", "Project path (default: current directory)")
	updateCmd.Flags().BoolVar(&updateDryRun, "dry-run", false, "Check for updates without installing")
}

// updateInfo holds information about an available update
type updateInfo struct {
	FullName       string
	CurrentVersion string
	LatestVersion  string
	Scope          settings.Scope
}

func runUpdate(cmd *cobra.Command, args []string) error {
	// Get list of plugins to update
	var pluginsToCheck []string

	if len(args) > 0 {
		// Specific plugins
		for _, arg := range args {
			fullName, err := resolvePluginFullName(arg, updateProject)
			if err != nil {
				return err
			}
			pluginsToCheck = append(pluginsToCheck, fullName)
		}
	} else {
		// All installed plugins
		states, err := settings.MergedPluginStates(updateProject)
		if err != nil {
			return fmt.Errorf("failed to load plugin states: %w", err)
		}

		// Apply scope filter if specified
		if updateScope != "" {
			scope, err := settings.ParseScope(updateScope)
			if err != nil {
				return err
			}
			states = settings.FilterByScope(states, scope)
		}

		for _, state := range states {
			pluginsToCheck = append(pluginsToCheck, state.FullName)
		}
	}

	if len(pluginsToCheck) == 0 {
		fmt.Println("No plugins to update")
		return nil
	}

	// Load installed plugins registry for current versions
	installed, err := config.LoadInstalledPlugins()
	if err != nil {
		return fmt.Errorf("failed to load installed plugins: %w", err)
	}

	// Load all available plugins to get latest versions
	allPlugins, err := config.LoadAllPlugins()
	if err != nil {
		return fmt.Errorf("failed to load available plugins: %w", err)
	}

	// Build lookup map for latest versions
	latestVersions := make(map[string]string)
	for _, p := range allPlugins {
		fullName := p.Name + "@" + p.Marketplace
		latestVersions[fullName] = p.Version
	}

	// Check each plugin for updates
	var updates []updateInfo
	for _, fullName := range pluginsToCheck {
		// Get current version from installed registry
		currentVersion := ""
		if installs, ok := installed.Plugins[fullName]; ok && len(installs) > 0 {
			currentVersion = installs[0].Version
		}

		// Get latest version from marketplace
		latestVersion, ok := latestVersions[fullName]
		if !ok {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %s not found in any marketplace\n", fullName)
			continue
		}

		// Compare versions using semver
		if currentVersion == "" || isNewerVersion(latestVersion, currentVersion) {
			// Determine scope for update
			scope := settings.ScopeUser
			if installs, ok := installed.Plugins[fullName]; ok && len(installs) > 0 {
				parsedScope, _ := settings.ParseScope(installs[0].Scope)
				scope = parsedScope
			}

			updates = append(updates, updateInfo{
				FullName:       fullName,
				CurrentVersion: currentVersion,
				LatestVersion:  latestVersion,
				Scope:          scope,
			})
		}
	}

	if len(updates) == 0 {
		fmt.Println("All plugins are up to date")
		return nil
	}

	// Print available updates
	fmt.Printf("Found %d update(s):\n\n", len(updates))
	for _, u := range updates {
		if u.CurrentVersion == "" {
			fmt.Printf("  %s: (not installed) → %s\n", u.FullName, u.LatestVersion)
		} else {
			fmt.Printf("  %s: %s → %s\n", u.FullName, u.CurrentVersion, u.LatestVersion)
		}
	}

	if updateDryRun {
		fmt.Println("\nRun without --dry-run to install updates")
		return nil
	}

	fmt.Println()

	// Perform updates
	for _, u := range updates {
		fmt.Printf("Updating %s...\n", u.FullName)

		// Parse plugin name and marketplace
		parts := strings.SplitN(u.FullName, "@", 2)
		if len(parts) != 2 {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Error: invalid plugin name format: %s\n", u.FullName)
			continue
		}

		// Reinstall the plugin to update it
		if err := installPlugin(u.FullName, u.Scope, updateProject); err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Error updating %s: %v\n", u.FullName, err)
			continue
		}
	}

	fmt.Println("\nUpdate complete")
	return nil
}

// isNewerVersion returns true if v1 is newer than v2 using semver comparison
func isNewerVersion(v1, v2 string) bool {
	// Clean version strings (remove 'v' prefix if present)
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	// Parse versions
	ver1, err1 := semver.NewVersion(v1)
	ver2, err2 := semver.NewVersion(v2)

	// If either version is invalid, fall back to string comparison
	if err1 != nil || err2 != nil {
		return v1 > v2
	}

	return ver1.GreaterThan(ver2)
}
