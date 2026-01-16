package settings

import (
	"encoding/json"
	"os"
)

// Settings represents the Claude Code settings.json structure
type Settings struct {
	// EnabledPlugins maps plugin@marketplace to enabled/disabled state
	EnabledPlugins map[string]bool `json:"enabledPlugins,omitempty"`

	// ExtraKnownMarketplaces stores additional marketplaces configured in this scope
	ExtraKnownMarketplaces map[string]ExtraMarketplace `json:"extraKnownMarketplaces,omitempty"`
}

// ExtraMarketplace represents a marketplace entry in settings
type ExtraMarketplace struct {
	Source MarketplaceSource `json:"source"`
}

// MarketplaceSource represents the source of a marketplace
type MarketplaceSource struct {
	Source string `json:"source"` // e.g., "github"
	Repo   string `json:"repo"`   // e.g., "owner/repo"
}

// PluginState represents the enabled/disabled state of a plugin with its scope
type PluginState struct {
	FullName string // plugin@marketplace
	Enabled  bool
	Scope    Scope
}

// NewSettings creates an empty Settings instance
func NewSettings() *Settings {
	return &Settings{
		EnabledPlugins:         make(map[string]bool),
		ExtraKnownMarketplaces: make(map[string]ExtraMarketplace),
	}
}

// LoadSettings loads settings from a specific scope
// Returns empty settings (not error) if file doesn't exist
func LoadSettings(scope Scope, projectPath string) (*Settings, error) {
	path, err := ScopePath(scope, projectPath)
	if err != nil {
		return nil, err
	}

	return LoadSettingsFromPath(path)
}

// LoadSettingsFromPath loads settings from a specific file path
// Returns empty settings (not error) if file doesn't exist
func LoadSettingsFromPath(path string) (*Settings, error) {
	// #nosec G304 -- path is derived from known config dirs, not untrusted input
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewSettings(), nil
		}
		return nil, err
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	// Initialize maps if nil
	if settings.EnabledPlugins == nil {
		settings.EnabledPlugins = make(map[string]bool)
	}
	if settings.ExtraKnownMarketplaces == nil {
		settings.ExtraKnownMarketplaces = make(map[string]ExtraMarketplace)
	}

	return &settings, nil
}

// MergedPluginStates loads all scopes and returns plugin states
// with scope information, respecting precedence order
// Precedence: Managed > Local > Project > User
func MergedPluginStates(projectPath string) ([]PluginState, error) {
	// Track which plugins we've seen (first one wins due to precedence)
	seen := make(map[string]bool)
	var states []PluginState

	for _, scope := range AllScopes() {
		settings, err := LoadSettings(scope, projectPath)
		if err != nil {
			// Skip scopes we can't read (e.g., managed may not exist)
			continue
		}

		for fullName, enabled := range settings.EnabledPlugins {
			if seen[fullName] {
				continue // Higher precedence scope already set this
			}
			seen[fullName] = true

			states = append(states, PluginState{
				FullName: fullName,
				Enabled:  enabled,
				Scope:    scope,
			})
		}
	}

	return states, nil
}

// GetPluginState returns the effective state for a specific plugin
// Returns the state from the highest precedence scope that has it
func GetPluginState(pluginFullName string, projectPath string) (*PluginState, error) {
	for _, scope := range AllScopes() {
		settings, err := LoadSettings(scope, projectPath)
		if err != nil {
			continue
		}

		if enabled, ok := settings.EnabledPlugins[pluginFullName]; ok {
			return &PluginState{
				FullName: pluginFullName,
				Enabled:  enabled,
				Scope:    scope,
			}, nil
		}
	}

	// Plugin not found in any scope
	return nil, nil
}

// AllMarketplaces returns all extra marketplaces from all scopes merged
// Precedence order applies (higher precedence scope wins on conflicts)
func AllMarketplaces(projectPath string) (map[string]ExtraMarketplace, error) {
	result := make(map[string]ExtraMarketplace)

	// Load in reverse precedence order so higher precedence overwrites
	scopes := AllScopes()
	for i := len(scopes) - 1; i >= 0; i-- {
		settings, err := LoadSettings(scopes[i], projectPath)
		if err != nil {
			continue
		}

		for name, marketplace := range settings.ExtraKnownMarketplaces {
			result[name] = marketplace
		}
	}

	return result, nil
}

// FilterByScope returns plugin states filtered to a specific scope
func FilterByScope(states []PluginState, scope Scope) []PluginState {
	var filtered []PluginState
	for _, state := range states {
		if state.Scope == scope {
			filtered = append(filtered, state)
		}
	}
	return filtered
}

// FilterEnabled returns only enabled plugin states
func FilterEnabled(states []PluginState) []PluginState {
	var filtered []PluginState
	for _, state := range states {
		if state.Enabled {
			filtered = append(filtered, state)
		}
	}
	return filtered
}

// FilterDisabled returns only disabled plugin states
func FilterDisabled(states []PluginState) []PluginState {
	var filtered []PluginState
	for _, state := range states {
		if !state.Enabled {
			filtered = append(filtered, state)
		}
	}
	return filtered
}
