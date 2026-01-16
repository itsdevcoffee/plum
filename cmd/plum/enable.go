package main

import (
	"fmt"
	"strings"

	"github.com/itsdevcoffee/plum/internal/config"
	"github.com/itsdevcoffee/plum/internal/settings"
	"github.com/spf13/cobra"
)

var enableCmd = &cobra.Command{
	Use:   "enable <plugin>",
	Short: "Enable a plugin",
	Long: `Enable a disabled plugin.

The plugin must already be installed. This sets the plugin's enabled state
to true in the specified scope's settings.json file.

The plugin can be specified as:
  - plugin-name (uses first matching installed plugin)
  - plugin-name@marketplace (specific marketplace)

Examples:
  plum enable ralph-wiggum
  plum enable ralph-wiggum@claude-code-plugins
  plum enable memory --scope=project`,
	Args: cobra.ExactArgs(1),
	RunE: runEnable,
}

var (
	enableScope   string
	enableProject string
)

func init() {
	rootCmd.AddCommand(enableCmd)

	enableCmd.Flags().StringVarP(&enableScope, "scope", "s", "user", "Target scope (user, project, local)")
	enableCmd.Flags().StringVar(&enableProject, "project", "", "Project path (default: current directory)")
}

func runEnable(cmd *cobra.Command, args []string) error {
	pluginArg := args[0]

	// Parse scope
	scope, err := settings.ParseScope(enableScope)
	if err != nil {
		return err
	}

	// Validate scope is writable
	if !scope.IsWritable() {
		return fmt.Errorf("cannot write to %s scope (read-only)", scope)
	}

	// Resolve plugin full name
	fullName, err := resolvePluginFullName(pluginArg, enableProject)
	if err != nil {
		return err
	}

	// Enable the plugin
	if err := settings.SetPluginEnabled(fullName, true, scope, enableProject); err != nil {
		return fmt.Errorf("failed to enable plugin: %w", err)
	}

	fmt.Printf("Enabled %s in %s scope\n", fullName, scope)
	return nil
}

// resolvePluginFullName resolves a plugin argument to its full name (plugin@marketplace)
// If the argument already contains @, it's returned as-is after validation
// Otherwise, it searches installed plugins and settings for a match
func resolvePluginFullName(pluginArg string, projectPath string) (string, error) {
	// If already has @marketplace, validate and return
	if strings.Contains(pluginArg, "@") {
		// Validate format
		parts := strings.SplitN(pluginArg, "@", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", fmt.Errorf("invalid plugin format: %s (expected: plugin-name@marketplace)", pluginArg)
		}
		return pluginArg, nil
	}

	// Search for the plugin in installed plugins
	installed, err := config.LoadInstalledPlugins()
	if err != nil {
		return "", fmt.Errorf("failed to load installed plugins: %w", err)
	}

	// Look for exact name match
	var matches []string
	for fullName := range installed.Plugins {
		name := strings.SplitN(fullName, "@", 2)[0]
		if name == pluginArg {
			matches = append(matches, fullName)
		}
	}

	// Also check settings for enabled plugins that might not be in installed registry
	states, err := settings.MergedPluginStates(projectPath)
	if err == nil {
		for _, state := range states {
			name := strings.SplitN(state.FullName, "@", 2)[0]
			if name == pluginArg {
				// Check if we already have this match
				found := false
				for _, m := range matches {
					if m == state.FullName {
						found = true
						break
					}
				}
				if !found {
					matches = append(matches, state.FullName)
				}
			}
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("plugin '%s' not found - specify full name (plugin@marketplace)", pluginArg)
	}

	if len(matches) > 1 {
		return "", fmt.Errorf("plugin '%s' is ambiguous, found in multiple marketplaces:\n  %s\nSpecify full name (e.g., %s)",
			pluginArg, strings.Join(matches, "\n  "), matches[0])
	}

	return matches[0], nil
}
