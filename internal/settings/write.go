package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SaveSettings saves settings to a specific scope
// Creates the necessary directories if they don't exist
func SaveSettings(s *Settings, scope Scope, projectPath string) error {
	// Validate scope is writable
	if !scope.IsWritable() {
		return ErrManagedReadOnly
	}

	// Get path for this scope
	path, err := ScopePath(scope, projectPath)
	if err != nil {
		return err
	}

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	// #nosec G301 -- Settings directory needs to be readable by Claude Code
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Load existing settings to preserve other fields
	existing, err := LoadSettingsFromPath(path)
	if err != nil {
		return fmt.Errorf("failed to load existing settings: %w", err)
	}

	// Merge: update enabledPlugins and extraKnownMarketplaces from s
	for k, v := range s.EnabledPlugins {
		existing.EnabledPlugins[k] = v
	}
	for k, v := range s.ExtraKnownMarketplaces {
		existing.ExtraKnownMarketplaces[k] = v
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Write atomically using temp file + rename
	tmpFile, err := os.CreateTemp(dir, ".settings-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }() // Cleanup on failure

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Add trailing newline
	if _, err := tmpFile.WriteString("\n"); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write newline: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Set permissions (user read/write, group/other read)
	// #nosec G302 -- Settings files need to be readable by Claude Code
	if err := os.Chmod(tmpPath, 0644); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Atomic rename
	if err := AtomicRename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// SetPluginEnabled sets the enabled state for a plugin in the specified scope
func SetPluginEnabled(fullName string, enabled bool, scope Scope, projectPath string) error {
	// Validate scope is writable
	if !scope.IsWritable() {
		return ErrManagedReadOnly
	}

	// Load existing settings for this scope
	path, err := ScopePath(scope, projectPath)
	if err != nil {
		return err
	}

	settings, err := LoadSettingsFromPath(path)
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	// Update the plugin state
	settings.EnabledPlugins[fullName] = enabled

	// Save settings
	return saveSettingsDirect(settings, path)
}

// RemovePluginFromScope removes a plugin entry from a specific scope
func RemovePluginFromScope(fullName string, scope Scope, projectPath string) error {
	// Validate scope is writable
	if !scope.IsWritable() {
		return ErrManagedReadOnly
	}

	// Load existing settings for this scope
	path, err := ScopePath(scope, projectPath)
	if err != nil {
		return err
	}

	settings, err := LoadSettingsFromPath(path)
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	// Remove the plugin entry
	delete(settings.EnabledPlugins, fullName)

	// Save settings
	return saveSettingsDirect(settings, path)
}

// AddMarketplace adds a marketplace to the specified scope
func AddMarketplace(name string, source MarketplaceSource, scope Scope, projectPath string) error {
	// Validate scope is writable
	if !scope.IsWritable() {
		return ErrManagedReadOnly
	}

	// Load existing settings for this scope
	path, err := ScopePath(scope, projectPath)
	if err != nil {
		return err
	}

	settings, err := LoadSettingsFromPath(path)
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	// Add the marketplace
	settings.ExtraKnownMarketplaces[name] = ExtraMarketplace{
		Source: source,
	}

	// Save settings
	return saveSettingsDirect(settings, path)
}

// RemoveMarketplace removes a marketplace from the specified scope
func RemoveMarketplace(name string, scope Scope, projectPath string) error {
	// Validate scope is writable
	if !scope.IsWritable() {
		return ErrManagedReadOnly
	}

	// Load existing settings for this scope
	path, err := ScopePath(scope, projectPath)
	if err != nil {
		return err
	}

	settings, err := LoadSettingsFromPath(path)
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	// Remove the marketplace entry
	delete(settings.ExtraKnownMarketplaces, name)

	// Save settings
	return saveSettingsDirect(settings, path)
}

// saveSettingsDirect saves settings directly to a path without merging
func saveSettingsDirect(s *Settings, path string) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	// #nosec G301 -- Settings directory needs to be readable by Claude Code
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Write atomically using temp file + rename
	tmpFile, err := os.CreateTemp(dir, ".settings-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }() // Cleanup on failure

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Add trailing newline
	if _, err := tmpFile.WriteString("\n"); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write newline: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Set permissions (user read/write, group/other read)
	// #nosec G302 -- Settings files need to be readable by Claude Code
	if err := os.Chmod(tmpPath, 0644); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Atomic rename
	if err := AtomicRename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// AtomicRename performs an atomic rename with Windows fallback
// Exported for use by other packages (install.go, remove.go)
func AtomicRename(tmpPath, finalPath string) error {
	err := os.Rename(tmpPath, finalPath)
	if err == nil {
		return nil
	}

	// Windows fallback: remove destination if it exists, then retry
	// We don't check if file exists first to avoid TOCTOU race condition
	_ = os.Remove(finalPath) // Ignore error - file may not exist
	if retryErr := os.Rename(tmpPath, finalPath); retryErr != nil {
		return fmt.Errorf("failed to rename: %w", retryErr)
	}
	return nil
}
