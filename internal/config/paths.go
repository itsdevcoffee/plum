package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// ClaudeConfigDir returns the path to the Claude Code configuration directory
// Respects CLAUDE_CONFIG_DIR environment variable for custom locations
func ClaudeConfigDir() (string, error) {
	// 1. Check environment variable override
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return dir, nil
	}

	// 2. Get user home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}

	// 3. Platform-specific defaults
	if runtime.GOOS == "windows" {
		// Windows: %APPDATA%\ClaudeCode
		appdata := os.Getenv("APPDATA")
		if appdata != "" {
			return filepath.Join(appdata, "ClaudeCode"), nil
		}
		// Fallback to home\.claude on Windows
		return filepath.Join(home, ".claude"), nil
	}

	// Unix-like systems (Linux, macOS): ~/.claude
	return filepath.Join(home, ".claude"), nil
}

// ClaudePluginsDir returns the path to the Claude Code plugins directory
// This is where marketplaces and installed plugins are tracked
func ClaudePluginsDir() (string, error) {
	configDir, err := ClaudeConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "plugins"), nil
}

// KnownMarketplacesPath returns the path to known_marketplaces.json
func KnownMarketplacesPath() (string, error) {
	pluginsDir, err := ClaudePluginsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(pluginsDir, "known_marketplaces.json"), nil
}

// InstalledPluginsPath returns the path to installed_plugins_v2.json
func InstalledPluginsPath() (string, error) {
	pluginsDir, err := ClaudePluginsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(pluginsDir, "installed_plugins_v2.json"), nil
}
