package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const (
	// maxSettingsFileSize is the maximum allowed size for settings.json files (10MB)
	// This prevents DoS attacks from maliciously large files in project-scoped settings
	maxSettingsFileSize = 10 * 1024 * 1024

	// maxPluginEntries is the maximum number of plugins allowed in enabledPlugins
	maxPluginEntries = 10000

	// maxMarketplaceEntries is the maximum number of marketplaces allowed
	maxMarketplaceEntries = 1000
)

// Settings represents the Claude Code settings.json structure
// It preserves all unknown fields when reading/writing to avoid destroying
// user configuration that plum doesn't manage (e.g., permissions, hooks, model)
type Settings struct {
	// EnabledPlugins maps plugin@marketplace to enabled/disabled state
	EnabledPlugins map[string]bool `json:"enabledPlugins,omitempty"`

	// ExtraKnownMarketplaces stores additional marketplaces configured in this scope
	ExtraKnownMarketplaces map[string]ExtraMarketplace `json:"extraKnownMarketplaces,omitempty"`

	// otherFields preserves all unknown top-level fields from the JSON file
	// This ensures plum doesn't destroy user settings it doesn't manage
	otherFields map[string]json.RawMessage
}

// UnmarshalJSON implements custom JSON unmarshaling to preserve unknown fields
func (s *Settings) UnmarshalJSON(data []byte) error {
	// First, unmarshal all fields into a raw map
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Extract known fields
	if rawPlugins, ok := raw["enabledPlugins"]; ok {
		if err := json.Unmarshal(rawPlugins, &s.EnabledPlugins); err != nil {
			return err
		}
		delete(raw, "enabledPlugins")
	}

	if rawMarketplaces, ok := raw["extraKnownMarketplaces"]; ok {
		if err := json.Unmarshal(rawMarketplaces, &s.ExtraKnownMarketplaces); err != nil {
			return err
		}
		delete(raw, "extraKnownMarketplaces")
	}

	// Store all remaining unknown fields
	if len(raw) > 0 {
		s.otherFields = raw
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling to preserve unknown fields
func (s *Settings) MarshalJSON() ([]byte, error) {
	// Build the output map
	result := make(map[string]any)

	// First, add all preserved unknown fields
	for key, value := range s.otherFields {
		// Unmarshal to any to avoid double-encoding
		var v any
		if err := json.Unmarshal(value, &v); err != nil {
			return nil, err
		}
		result[key] = v
	}

	// Then add known fields (these take precedence over any preserved fields with same name)
	if len(s.EnabledPlugins) > 0 {
		result["enabledPlugins"] = s.EnabledPlugins
	}
	if len(s.ExtraKnownMarketplaces) > 0 {
		result["extraKnownMarketplaces"] = s.ExtraKnownMarketplaces
	}

	return json.Marshal(result)
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
		otherFields:            make(map[string]json.RawMessage),
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
	// Check file size before reading to prevent DoS from large files
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewSettings(), nil
		}
		return nil, err
	}
	if stat.Size() > maxSettingsFileSize {
		return nil, fmt.Errorf("settings file too large: %d bytes (max %d)", stat.Size(), maxSettingsFileSize)
	}

	// #nosec G304 -- path is validated via ScopePath which uses known config dirs
	data, err := os.ReadFile(path)
	if err != nil {
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
	if settings.otherFields == nil {
		settings.otherFields = make(map[string]json.RawMessage)
	}

	// Validate structure to prevent resource exhaustion
	if err := validateSettings(&settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

// validateSettings validates the settings structure
func validateSettings(s *Settings) error {
	// Limit total entries to prevent memory exhaustion
	if len(s.EnabledPlugins) > maxPluginEntries {
		return fmt.Errorf("too many enabled plugins: %d (max %d)", len(s.EnabledPlugins), maxPluginEntries)
	}
	if len(s.ExtraKnownMarketplaces) > maxMarketplaceEntries {
		return fmt.Errorf("too many marketplaces: %d (max %d)", len(s.ExtraKnownMarketplaces), maxMarketplaceEntries)
	}

	// Validate plugin key format (must be plugin@marketplace)
	for key := range s.EnabledPlugins {
		if !strings.Contains(key, "@") {
			return fmt.Errorf("invalid plugin key format (missing @): %s", key)
		}
	}

	return nil
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
	seen := make(map[string]bool)

	// Iterate in precedence order (highest first wins)
	for _, scope := range AllScopes() {
		settings, err := LoadSettings(scope, projectPath)
		if err != nil {
			continue
		}

		for name, marketplace := range settings.ExtraKnownMarketplaces {
			if seen[name] {
				continue // Higher precedence scope already set this
			}
			seen[name] = true
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
