package main

import (
	"fmt"

	"github.com/itsdevcoffee/plum/internal/settings"
	"github.com/spf13/cobra"
)

var disableCmd = &cobra.Command{
	Use:   "disable <plugin>",
	Short: "Disable a plugin",
	Long: `Disable an enabled plugin.

This sets the plugin's enabled state to false in the specified scope's
settings.json file. The plugin files remain cached for quick re-enabling.

The plugin can be specified as:
  - plugin-name (uses first matching installed plugin)
  - plugin-name@marketplace (specific marketplace)

Examples:
  plum disable ralph-wiggum
  plum disable ralph-wiggum@claude-code-plugins
  plum disable memory --scope=project`,
	Args: cobra.ExactArgs(1),
	RunE: runDisable,
}

var (
	disableScope   string
	disableProject string
)

func init() {
	rootCmd.AddCommand(disableCmd)

	disableCmd.Flags().StringVarP(&disableScope, "scope", "s", "user", "Target scope (user, project, local)")
	disableCmd.Flags().StringVar(&disableProject, "project", "", "Project path (default: current directory)")
}

func runDisable(cmd *cobra.Command, args []string) error {
	pluginArg := args[0]

	// Parse scope
	scope, err := settings.ParseScope(disableScope)
	if err != nil {
		return err
	}

	// Validate scope is writable
	if !scope.IsWritable() {
		return fmt.Errorf("cannot write to %s scope (read-only)", scope)
	}

	// Resolve plugin full name (reuse from enable.go)
	fullName, err := resolvePluginFullName(pluginArg, disableProject)
	if err != nil {
		return err
	}

	// Disable the plugin
	if err := settings.SetPluginEnabled(fullName, false, scope, disableProject); err != nil {
		return fmt.Errorf("failed to disable plugin: %w", err)
	}

	fmt.Printf("Disabled %s in %s scope\n", fullName, scope)
	return nil
}
