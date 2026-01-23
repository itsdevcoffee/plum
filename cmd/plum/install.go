package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/itsdevcoffee/plum/internal/config"
	"github.com/itsdevcoffee/plum/internal/marketplace"
	"github.com/itsdevcoffee/plum/internal/settings"
	"github.com/spf13/cobra"
)

// validatePathComponent checks if a path component is safe (no path traversal)
func validatePathComponent(name, componentType string) error {
	if name == "" {
		return fmt.Errorf("%s cannot be empty", componentType)
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("%s contains invalid path traversal: %s", componentType, name)
	}
	if strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("%s contains invalid path separator: %s", componentType, name)
	}
	if name == "." {
		return fmt.Errorf("%s cannot be current directory", componentType)
	}
	return nil
}

// validatePluginFilePath validates a file path from plugin manifest is safe
// Returns cleaned path relative to cacheDir, or error if path escapes
func validatePluginFilePath(filePath, cacheDir string) (string, error) {
	// Reject absolute paths
	if filepath.IsAbs(filePath) {
		return "", fmt.Errorf("absolute paths not allowed: %s", filePath)
	}

	// Reject path traversal attempts
	if strings.Contains(filePath, "..") {
		return "", fmt.Errorf("path traversal not allowed: %s", filePath)
	}

	// Clean the path
	cleanPath := filepath.Clean(filePath)

	// Construct full path and verify it's under cacheDir
	fullPath := filepath.Join(cacheDir, cleanPath)
	absCache, err := filepath.Abs(cacheDir)
	if err != nil {
		return "", err
	}
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}

	// Ensure the path is under the cache directory
	if !strings.HasPrefix(absPath, absCache+string(filepath.Separator)) && absPath != absCache {
		return "", fmt.Errorf("path escapes cache directory: %s", filePath)
	}

	return fullPath, nil
}

var installCmd = &cobra.Command{
	Use:   "install <plugin>",
	Short: "Install a plugin",
	Long: `Install a plugin from a marketplace.

The plugin can be specified as:
  - plugin-name (searches all known marketplaces)
  - plugin-name@marketplace (specific marketplace)

Installation downloads plugin files to the Claude Code cache and enables
the plugin in the specified scope.

Examples:
  plum install ralph-wiggum
  plum install ralph-wiggum@claude-code-plugins
  plum install memory --scope=project`,
	Args: cobra.MinimumNArgs(1),
	RunE: runInstall,
}

var (
	installScope   string
	installProject string
)

func init() {
	rootCmd.AddCommand(installCmd)

	installCmd.Flags().StringVarP(&installScope, "scope", "s", "user", "Installation scope (user, project, local)")
	installCmd.Flags().StringVar(&installProject, "project", "", "Project path (default: current directory)")
}

func runInstall(cmd *cobra.Command, args []string) error {
	// Parse scope
	scope, err := settings.ParseScope(installScope)
	if err != nil {
		return err
	}

	// Validate scope is writable
	if !scope.IsWritable() {
		return fmt.Errorf("cannot write to %s scope (read-only)", scope)
	}

	// Install each plugin
	for _, pluginArg := range args {
		if err := installPlugin(pluginArg, scope, installProject); err != nil {
			return fmt.Errorf("failed to install %s: %w", pluginArg, err)
		}
	}

	return nil
}

func installPlugin(pluginArg string, scope settings.Scope, projectPath string) error {
	// Parse plugin name and marketplace filter
	pluginName := pluginArg
	marketplaceFilter := ""
	if idx := strings.LastIndex(pluginArg, "@"); idx > 0 {
		pluginName = pluginArg[:idx]
		marketplaceFilter = pluginArg[idx+1:]
	}

	// Find the plugin in marketplaces
	pluginInfo, err := findPluginInMarketplaces(pluginName, marketplaceFilter)
	if err != nil {
		return err
	}

	fullName := pluginInfo.Name + "@" + pluginInfo.Marketplace

	// Check if plugin is installable via plum
	if !pluginInfo.Installable {
		fmt.Printf("Cannot install %s: %s\n\n", fullName, pluginInfo.InstallabilityReason)
		fmt.Println("This plugin requires a different installation method.")
		fmt.Println("Check the plugin's homepage for installation instructions.")
		return fmt.Errorf("plugin not installable via plum")
	}

	fmt.Printf("Installing %s...\n", fullName)

	// Get cache directory
	cacheDir, err := pluginCacheDir(pluginInfo.Marketplace, pluginInfo.Name)
	if err != nil {
		return fmt.Errorf("failed to get cache directory: %w", err)
	}

	// Download plugin files to cache
	if err := downloadPluginToCache(pluginInfo, cacheDir); err != nil {
		return fmt.Errorf("failed to download plugin: %w", err)
	}

	// Register in installed_plugins_v2.json
	if err := registerInstalledPlugin(fullName, cacheDir, pluginInfo.Version, scope, projectPath); err != nil {
		return fmt.Errorf("failed to register plugin: %w", err)
	}

	// Enable in settings.json
	if err := settings.SetPluginEnabled(fullName, true, scope, projectPath); err != nil {
		return fmt.Errorf("failed to enable plugin: %w", err)
	}

	fmt.Printf("Installed %s (v%s) in %s scope\n", fullName, pluginInfo.Version, scope)
	return nil
}

// pluginSearchResult holds plugin info needed for installation
type pluginSearchResult struct {
	Name                 string
	Marketplace          string
	MarketplaceRepo      string
	Version              string
	Source               string // Path within marketplace
	Installable          bool   // Whether plum can install this plugin
	InstallabilityReason string // Human-readable reason if not installable
}

// findPluginInMarketplaces searches for a plugin across all known marketplaces
func findPluginInMarketplaces(pluginName, marketplaceFilter string) (*pluginSearchResult, error) {
	// Load all plugins
	plugins, err := config.LoadAllPlugins()
	if err != nil {
		return nil, fmt.Errorf("failed to load plugins: %w", err)
	}

	var matches []*pluginSearchResult
	for _, p := range plugins {
		if p.Name == pluginName {
			// If marketplace filter specified, must match
			if marketplaceFilter != "" && p.Marketplace != marketplaceFilter {
				continue
			}
			matches = append(matches, &pluginSearchResult{
				Name:                 p.Name,
				Marketplace:          p.Marketplace,
				MarketplaceRepo:      p.MarketplaceRepo,
				Version:              p.Version,
				Source:               p.Source,
				Installable:          p.Installable(),
				InstallabilityReason: p.InstallabilityReason(),
			})
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("plugin '%s' not found in any marketplace", pluginName)
	}

	if len(matches) > 1 && marketplaceFilter == "" {
		var names []string
		for _, m := range matches {
			names = append(names, m.Name+"@"+m.Marketplace)
		}
		return nil, fmt.Errorf("plugin '%s' found in multiple marketplaces:\n  %s\nSpecify with: plum install %s@<marketplace>",
			pluginName, strings.Join(names, "\n  "), pluginName)
	}

	return matches[0], nil
}

// pluginCacheDir returns the path to cache a plugin
// Path: ~/.claude/plugins/cache/<marketplace>/<plugin>/
func pluginCacheDir(marketplaceName, pluginName string) (string, error) {
	// Validate marketplace and plugin names to prevent path traversal
	if err := validatePathComponent(marketplaceName, "marketplace name"); err != nil {
		return "", err
	}
	if err := validatePathComponent(pluginName, "plugin name"); err != nil {
		return "", err
	}

	pluginsDir, err := config.ClaudePluginsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(pluginsDir, "cache", marketplaceName, pluginName), nil
}

// maxTotalDownloadSize is the maximum total download size per plugin (50 MB)
const maxTotalDownloadSize = 50 << 20

// downloadPluginToCache downloads plugin files from GitHub to the cache directory
func downloadPluginToCache(plugin *pluginSearchResult, cacheDir string) error {
	// Extract owner/repo from marketplace repo URL
	source, err := marketplace.DeriveSource(plugin.MarketplaceRepo)
	if err != nil {
		return fmt.Errorf("failed to derive source from repo: %w", err)
	}

	// Normalize source path (remove leading ./ if present)
	sourcePath := strings.TrimPrefix(plugin.Source, "./")
	if sourcePath == "" || sourcePath == "." {
		sourcePath = "plugins/" + plugin.Name
	}

	// Create cache directory
	// #nosec G301 -- Plugin cache needs to be readable by Claude Code
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Track total download size to prevent DoS
	var totalDownloaded int64

	// downloadWithLimit downloads a file and tracks total size
	downloadWithLimit := func(url string) ([]byte, error) {
		data, err := downloadFile(url)
		if err != nil {
			return nil, err
		}
		totalDownloaded += int64(len(data))
		if totalDownloaded > maxTotalDownloadSize {
			return nil, fmt.Errorf("plugin download size exceeded limit (%d MB)", maxTotalDownloadSize>>20)
		}
		return data, nil
	}

	// Download plugin.json to verify the plugin structure
	pluginJSONURL := fmt.Sprintf("%s/%s/%s/%s/.claude-plugin/plugin.json",
		marketplace.GitHubRawBase, source, marketplace.DefaultBranch, sourcePath)

	pluginJSON, err := downloadWithLimit(pluginJSONURL)
	if err != nil {
		return fmt.Errorf("failed to download plugin.json: %w", err)
	}

	// Create .claude-plugin directory in cache
	claudePluginDir := filepath.Join(cacheDir, ".claude-plugin")
	// #nosec G301 -- Plugin directory needs to be readable by Claude Code
	if err := os.MkdirAll(claudePluginDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude-plugin directory: %w", err)
	}

	// Write plugin.json
	pluginJSONPath := filepath.Join(claudePluginDir, "plugin.json")
	// #nosec G306 -- Plugin files need to be readable by Claude Code
	if err := os.WriteFile(pluginJSONPath, pluginJSON, 0644); err != nil {
		return fmt.Errorf("failed to write plugin.json: %w", err)
	}

	// Parse plugin.json to get file list
	var pluginManifest struct {
		Name        string   `json:"name"`
		Version     string   `json:"version"`
		Description string   `json:"description"`
		Commands    []string `json:"commands"`
		Hooks       []string `json:"hooks"`
	}
	if err := json.Unmarshal(pluginJSON, &pluginManifest); err != nil {
		// Not a fatal error - we have the plugin.json at least
		fmt.Fprintf(os.Stderr, "Warning: failed to parse plugin.json: %v\n", err)
	}

	// Download commands (non-executable)
	downloadPluginFiles(pluginManifest.Commands, "command", cacheDir, source, sourcePath, downloadWithLimit, 0644)

	// Download hooks (executable)
	downloadPluginFiles(pluginManifest.Hooks, "hook", cacheDir, source, sourcePath, downloadWithLimit, 0755)

	return nil
}

// downloadPluginFiles downloads a list of plugin files to the cache directory.
// fileType is used for warning messages (e.g., "command" or "hook").
// perm specifies the file permissions (e.g., 0644 for commands, 0755 for hooks).
func downloadPluginFiles(
	files []string,
	fileType string,
	cacheDir string,
	source string,
	sourcePath string,
	downloadWithLimit func(string) ([]byte, error),
	perm os.FileMode,
) {
	for _, file := range files {
		// Validate path to prevent path traversal attacks
		filePath, err := validatePluginFilePath(file, cacheDir)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: skipping invalid %s path %s: %v\n", fileType, file, err)
			continue
		}

		fileURL := fmt.Sprintf("%s/%s/%s/%s/%s",
			marketplace.GitHubRawBase, source, marketplace.DefaultBranch, sourcePath, file)

		content, err := downloadWithLimit(fileURL)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to download %s %s: %v\n", fileType, file, err)
			continue
		}

		fileDir := filepath.Dir(filePath)
		// #nosec G301 -- Plugin directory needs to be readable by Claude Code
		if err := os.MkdirAll(fileDir, 0755); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to create directory for %s: %v\n", file, err)
			continue
		}

		// #nosec G306 -- Plugin files need appropriate permissions
		if err := os.WriteFile(filePath, content, perm); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to write %s: %v\n", file, err)
		}
	}
}

// downloadFile downloads a file from a URL
func downloadFile(url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "plum/0.4.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
	}

	// Limit response size
	limitedBody := io.LimitReader(resp.Body, 10<<20) // 10 MB limit
	return io.ReadAll(limitedBody)
}

// registerInstalledPlugin adds the plugin to installed_plugins_v2.json
func registerInstalledPlugin(fullName, installPath, version string, scope settings.Scope, projectPath string) error {
	// Get registry path for locking
	registryPath, err := config.InstalledPluginsPath()
	if err != nil {
		return err
	}

	// Use file locking to prevent race conditions
	return settings.WithLock(registryPath, func() error {
		installed, err := config.LoadInstalledPlugins()
		if err != nil {
			return err
		}

		// Create install entry
		install := config.PluginInstall{
			Scope:        scope.String(),
			InstallPath:  installPath,
			Version:      version,
			InstalledAt:  time.Now().UTC().Format(time.RFC3339),
			LastUpdated:  time.Now().UTC().Format(time.RFC3339),
			GitCommitSha: "", // We don't track commit SHA for now
			IsLocal:      false,
		}

		// Add project path for project/local scopes
		if scope == settings.ScopeProject || scope == settings.ScopeLocal {
			if projectPath == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}
				projectPath = cwd
			}
			install.ProjectPath = projectPath
		}

		// Check if already installed
		existing, ok := installed.Plugins[fullName]
		if ok {
			// Update existing entry for this scope
			found := false
			for i, e := range existing {
				if e.Scope == scope.String() {
					existing[i] = install
					found = true
					break
				}
			}
			if !found {
				existing = append(existing, install)
			}
			installed.Plugins[fullName] = existing
		} else {
			installed.Plugins[fullName] = []config.PluginInstall{install}
		}

		// Write back to file
		return saveInstalledPlugins(installed)
	})
}

// saveInstalledPlugins writes the installed plugins registry
func saveInstalledPlugins(installed *config.InstalledPluginsV2) error {
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
