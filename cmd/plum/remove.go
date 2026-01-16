package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/itsdevcoffee/plum/internal/config"
	"github.com/itsdevcoffee/plum/internal/settings"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove <plugin>",
	Aliases: []string{"uninstall", "rm"},
	Short:   "Remove a plugin",
	Long: `Remove/uninstall a plugin.

This removes the plugin from the specified scope's settings.json and optionally
deletes the cached plugin files.

The plugin can be specified as:
  - plugin-name (uses first matching installed plugin)
  - plugin-name@marketplace (specific marketplace)

Examples:
  plum remove ralph-wiggum
  plum remove ralph-wiggum@claude-code-plugins
  plum remove memory --scope=project
  plum remove memory --all         # Remove from all scopes`,
	Args: cobra.ExactArgs(1),
	RunE: runRemove,
}

var (
	removeScope     string
	removeProject   string
	removeAll       bool
	removeKeepCache bool
)

func init() {
	rootCmd.AddCommand(removeCmd)

	removeCmd.Flags().StringVarP(&removeScope, "scope", "s", "user", "Target scope (user, project, local)")
	removeCmd.Flags().StringVar(&removeProject, "project", "", "Project path (default: current directory)")
	removeCmd.Flags().BoolVar(&removeAll, "all", false, "Remove from all scopes")
	removeCmd.Flags().BoolVar(&removeKeepCache, "keep-cache", false, "Keep cached plugin files")
}

func runRemove(cmd *cobra.Command, args []string) error {
	pluginArg := args[0]

	// Resolve plugin full name
	fullName, err := resolvePluginFullName(pluginArg, removeProject)
	if err != nil {
		return err
	}

	if removeAll {
		// Remove from all writable scopes
		var removedCount int
		var failedScopes []string
		for _, scope := range settings.WritableScopes() {
			// Check if plugin exists in this scope before attempting removal
			scopeSettings, loadErr := settings.LoadSettings(scope, removeProject)
			if loadErr != nil {
				failedScopes = append(failedScopes, fmt.Sprintf("%s: failed to load settings: %v", scope, loadErr))
				continue
			}
			if _, exists := scopeSettings.EnabledPlugins[fullName]; !exists {
				// Plugin not in this scope - skip silently (expected)
				continue
			}

			// Plugin exists in this scope, attempt removal
			if err := removePluginFromScope(fullName, scope, removeProject); err != nil {
				failedScopes = append(failedScopes, fmt.Sprintf("%s: %v", scope, err))
				continue
			}
			fmt.Printf("Removed %s from %s scope\n", fullName, scope)
			removedCount++
		}

		// Report any real failures
		if len(failedScopes) > 0 {
			return fmt.Errorf("removal failed in some scopes:\n  %s", strings.Join(failedScopes, "\n  "))
		}

		if removedCount == 0 {
			fmt.Printf("Plugin %s was not found in any writable scope\n", fullName)
		}
	} else {
		// Parse scope
		scope, err := settings.ParseScope(removeScope)
		if err != nil {
			return err
		}

		// Validate scope is writable
		if !scope.IsWritable() {
			return fmt.Errorf("cannot write to %s scope (read-only)", scope)
		}

		// Remove from the specified scope
		if err := removePluginFromScope(fullName, scope, removeProject); err != nil {
			return err
		}
		fmt.Printf("Removed %s from %s scope\n", fullName, scope)
	}

	// Check if plugin is still in any scope
	stillInstalled := false
	states, err := settings.MergedPluginStates(removeProject)
	if err == nil {
		for _, state := range states {
			if state.FullName == fullName {
				stillInstalled = true
				break
			}
		}
	}

	// Delete cache if not still installed and --keep-cache not specified
	if !stillInstalled && !removeKeepCache {
		if err := deletePluginCache(fullName); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to delete cache: %v\n", err)
		} else {
			fmt.Println("Deleted cached plugin files")
		}

		// Remove from installed_plugins_v2.json
		if err := unregisterInstalledPlugin(fullName); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update install registry: %v\n", err)
		}
	}

	return nil
}

// removePluginFromScope removes a plugin from a specific scope's settings
func removePluginFromScope(fullName string, scope settings.Scope, projectPath string) error {
	return settings.RemovePluginFromScope(fullName, scope, projectPath)
}

// deletePluginCache removes the cached plugin files
func deletePluginCache(fullName string) error {
	// Parse plugin@marketplace
	parts := strings.SplitN(fullName, "@", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid plugin name format: %s", fullName)
	}
	pluginName := parts[0]
	marketplace := parts[1]

	// Get cache directory
	pluginsDir, err := config.ClaudePluginsDir()
	if err != nil {
		return err
	}

	cachePath := filepath.Join(pluginsDir, "cache", marketplace, pluginName)

	// Check if exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return nil // Already gone
	}

	return os.RemoveAll(cachePath)
}

// unregisterInstalledPlugin removes the plugin from installed_plugins_v2.json
func unregisterInstalledPlugin(fullName string) error {
	installed, err := config.LoadInstalledPlugins()
	if err != nil {
		return err
	}

	// Remove the plugin entry
	delete(installed.Plugins, fullName)

	// Write back to file
	path, err := config.InstalledPluginsPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	// #nosec G301 -- Plugin directory needs to be readable by Claude Code
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(installed, "", "  ")
	if err != nil {
		return err
	}

	// Write atomically
	tmpFile, err := os.CreateTemp(dir, ".installed-*.json")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if _, err := tmpFile.WriteString("\n"); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	// #nosec G302 -- Config files need to be readable by Claude Code
	if err := os.Chmod(tmpPath, 0644); err != nil {
		return err
	}

	return settings.AtomicRename(tmpPath, path)
}
