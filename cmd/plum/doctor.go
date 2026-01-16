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

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check plugin installation health",
	Long: `Validate plugin structure and check for common issues.

Performs the following checks:
  - Missing plugin.json files in cached plugins
  - Invalid JSON in plugin manifests
  - Orphaned cache entries (cache files with no registry entry)
  - Missing cache files for registered plugins
  - Enabled plugins that aren't installed

Examples:
  plum doctor
  plum doctor --json`,
	RunE: runDoctor,
}

var (
	doctorJSON    bool
	doctorProject string
)

func init() {
	rootCmd.AddCommand(doctorCmd)

	doctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "Output as JSON")
	doctorCmd.Flags().StringVar(&doctorProject, "project", "", "Project path (default: current directory)")
}

// DoctorIssue represents a health check issue
type DoctorIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"` // "error", "warning", "info"
	Plugin      string `json:"plugin,omitempty"`
	Path        string `json:"path,omitempty"`
	Description string `json:"description"`
}

// DoctorResult holds the results of the health check
type DoctorResult struct {
	Healthy bool          `json:"healthy"`
	Issues  []DoctorIssue `json:"issues"`
	Summary DoctorSummary `json:"summary"`
}

// DoctorSummary provides counts of different issue types
type DoctorSummary struct {
	CachedPlugins     int `json:"cachedPlugins"`
	RegisteredPlugins int `json:"registeredPlugins"`
	EnabledPlugins    int `json:"enabledPlugins"`
	Errors            int `json:"errors"`
	Warnings          int `json:"warnings"`
}

func runDoctor(cmd *cobra.Command, args []string) error {
	result := DoctorResult{
		Healthy: true,
		Issues:  make([]DoctorIssue, 0),
	}

	// Get plugins directory
	pluginsDir, err := config.ClaudePluginsDir()
	if err != nil {
		return fmt.Errorf("failed to get plugins directory: %w", err)
	}
	cacheDir := filepath.Join(pluginsDir, "cache")

	// Load installed plugins registry
	installed, err := config.LoadInstalledPlugins()
	if err != nil {
		// Not fatal - might just not exist yet
		installed = &config.InstalledPluginsV2{Plugins: make(map[string][]config.PluginInstall)}
	}
	result.Summary.RegisteredPlugins = len(installed.Plugins)

	// Load enabled plugins from settings
	states, err := settings.MergedPluginStates(doctorProject)
	if err != nil {
		// Not fatal
		states = nil
	}
	for _, s := range states {
		if s.Enabled {
			result.Summary.EnabledPlugins++
		}
	}

	// Build set of registered plugins for lookup
	registeredPaths := make(map[string]string) // path -> fullName
	for fullName, installs := range installed.Plugins {
		for _, install := range installs {
			if install.InstallPath != "" {
				registeredPaths[install.InstallPath] = fullName
			}
		}
	}

	// Check 1: Scan cache directory for plugin directories
	cachedPlugins := make(map[string]bool) // path -> exists
	if _, err := os.Stat(cacheDir); err == nil {
		err := filepath.WalkDir(cacheDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil // Skip errors
			}

			// Look for .claude-plugin directories
			if d.IsDir() && d.Name() == ".claude-plugin" {
				pluginDir := filepath.Dir(path)
				cachedPlugins[pluginDir] = true
				result.Summary.CachedPlugins++

				// Check for plugin.json
				pluginJSONPath := filepath.Join(path, "plugin.json")
				if _, statErr := os.Stat(pluginJSONPath); os.IsNotExist(statErr) {
					result.Issues = append(result.Issues, DoctorIssue{
						Type:        "missing_plugin_json",
						Severity:    "error",
						Path:        pluginDir,
						Description: "Missing plugin.json file",
					})
					result.Summary.Errors++
				} else if statErr == nil {
					// Validate JSON
					if jsonErr := validatePluginJSON(pluginJSONPath); jsonErr != nil {
						result.Issues = append(result.Issues, DoctorIssue{
							Type:        "invalid_json",
							Severity:    "error",
							Path:        pluginJSONPath,
							Description: fmt.Sprintf("Invalid plugin.json: %v", jsonErr),
						})
						result.Summary.Errors++
					}
				}

				// Check if this cached plugin is registered
				if _, registered := registeredPaths[pluginDir]; !registered {
					// Extract plugin name from path for the message
					relPath, _ := filepath.Rel(cacheDir, pluginDir)
					result.Issues = append(result.Issues, DoctorIssue{
						Type:        "orphaned_cache",
						Severity:    "warning",
						Path:        pluginDir,
						Description: fmt.Sprintf("Cached plugin '%s' not in registry", relPath),
					})
					result.Summary.Warnings++
				}
			}
			return nil
		})
		if err != nil {
			// Log but don't fail
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: error scanning cache: %v\n", err)
		}
	}

	// Check 2: Verify registered plugins have cache files
	for fullName, installs := range installed.Plugins {
		for _, install := range installs {
			if install.InstallPath != "" {
				pluginJSONPath := filepath.Join(install.InstallPath, ".claude-plugin", "plugin.json")
				if _, err := os.Stat(pluginJSONPath); os.IsNotExist(err) {
					result.Issues = append(result.Issues, DoctorIssue{
						Type:        "missing_cache",
						Severity:    "error",
						Plugin:      fullName,
						Path:        install.InstallPath,
						Description: "Registered plugin missing from cache",
					})
					result.Summary.Errors++
				}
			}
		}
	}

	// Check 3: Verify enabled plugins are installed
	for _, state := range states {
		if state.Enabled {
			if _, registered := installed.Plugins[state.FullName]; !registered {
				result.Issues = append(result.Issues, DoctorIssue{
					Type:        "enabled_not_installed",
					Severity:    "warning",
					Plugin:      state.FullName,
					Description: fmt.Sprintf("Plugin enabled in %s scope but not installed", state.Scope),
				})
				result.Summary.Warnings++
			}
		}
	}

	// Determine overall health
	result.Healthy = result.Summary.Errors == 0

	// Output
	if doctorJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	return outputDoctorResult(result)
}

func validatePluginJSON(path string) error {
	// #nosec G304 -- path is constructed from known cache directory
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return err
	}

	// Check required fields
	if _, ok := manifest["name"]; !ok {
		return fmt.Errorf("missing required field 'name'")
	}

	return nil
}

func outputDoctorResult(result DoctorResult) error {
	// Summary header
	if result.Healthy {
		fmt.Println("✓ Plugin installation is healthy")
	} else {
		fmt.Println("✗ Issues found with plugin installation")
	}
	fmt.Println()

	// Stats
	fmt.Printf("Plugins:\n")
	fmt.Printf("  Cached:     %d\n", result.Summary.CachedPlugins)
	fmt.Printf("  Registered: %d\n", result.Summary.RegisteredPlugins)
	fmt.Printf("  Enabled:    %d\n", result.Summary.EnabledPlugins)
	fmt.Println()

	if len(result.Issues) == 0 {
		fmt.Println("No issues found")
		return nil
	}

	// Group issues by severity
	var errors, warnings []DoctorIssue
	for _, issue := range result.Issues {
		switch issue.Severity {
		case "error":
			errors = append(errors, issue)
		case "warning":
			warnings = append(warnings, issue)
		}
	}

	// Print errors first
	if len(errors) > 0 {
		fmt.Printf("Errors (%d):\n", len(errors))
		for _, issue := range errors {
			printIssue(issue)
		}
		fmt.Println()
	}

	// Then warnings
	if len(warnings) > 0 {
		fmt.Printf("Warnings (%d):\n", len(warnings))
		for _, issue := range warnings {
			printIssue(issue)
		}
		fmt.Println()
	}

	// Suggestions
	if result.Summary.Errors > 0 {
		fmt.Println("Run 'plum install <plugin>' to reinstall missing plugins")
	}

	return nil
}

func printIssue(issue DoctorIssue) {
	prefix := "  "
	switch issue.Severity {
	case "error":
		prefix = "  ✗"
	case "warning":
		prefix = "  !"
	}

	desc := issue.Description
	if issue.Plugin != "" {
		desc = issue.Plugin + ": " + desc
	} else if issue.Path != "" {
		// Shorten path for display
		short := shortenPath(issue.Path)
		desc = short + ": " + desc
	}

	fmt.Printf("%s %s\n", prefix, desc)
}

func shortenPath(path string) string {
	// Try to shorten to ~/.claude/...
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}
