package settings

import (
	"os"
	"path/filepath"

	"github.com/itsdevcoffee/plum/internal/config"
)

// Scope represents a Claude Code settings scope
type Scope string

const (
	// ScopeManaged is the system-wide managed scope (read-only)
	// Location: /etc/claude-code/settings.json (Unix) or C:\ProgramData\ClaudeCode\settings.json (Windows)
	ScopeManaged Scope = "managed"

	// ScopeUser is the user-level scope
	// Location: ~/.claude/settings.json
	ScopeUser Scope = "user"

	// ScopeProject is the project-level scope (checked into git)
	// Location: <project>/.claude/settings.json
	ScopeProject Scope = "project"

	// ScopeLocal is the local project scope (gitignored)
	// Location: <project>/.claude/settings.local.json
	ScopeLocal Scope = "local"
)

// AllScopes returns all scopes in precedence order (highest to lowest)
func AllScopes() []Scope {
	return []Scope{ScopeManaged, ScopeLocal, ScopeProject, ScopeUser}
}

// WritableScopes returns scopes that can be written to
func WritableScopes() []Scope {
	return []Scope{ScopeLocal, ScopeProject, ScopeUser}
}

// String returns the string representation of the scope
func (s Scope) String() string {
	return string(s)
}

// IsWritable returns true if the scope can be written to
func (s Scope) IsWritable() bool {
	return s != ScopeManaged
}

// ScopePath returns the settings.json path for a given scope
// For project/local scopes, projectPath must be provided
func ScopePath(scope Scope, projectPath string) (string, error) {
	switch scope {
	case ScopeManaged:
		return ManagedSettingsPath()
	case ScopeUser:
		return UserSettingsPath()
	case ScopeProject:
		return ProjectSettingsPath(projectPath)
	case ScopeLocal:
		return LocalSettingsPath(projectPath)
	default:
		return "", ErrInvalidScope
	}
}

// ManagedSettingsPath returns the path to managed settings.json
func ManagedSettingsPath() (string, error) {
	// Unix: /etc/claude-code/settings.json
	// Windows: C:\ProgramData\ClaudeCode\settings.json
	if os.PathSeparator == '\\' {
		// Windows
		programData := os.Getenv("PROGRAMDATA")
		if programData == "" {
			programData = `C:\ProgramData`
		}
		return filepath.Join(programData, "ClaudeCode", "settings.json"), nil
	}
	// Unix
	return "/etc/claude-code/settings.json", nil
}

// UserSettingsPath returns the path to user settings.json
func UserSettingsPath() (string, error) {
	configDir, err := config.ClaudeConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "settings.json"), nil
}

// ProjectSettingsPath returns the path to project settings.json
func ProjectSettingsPath(projectPath string) (string, error) {
	projectPath, err := normalizeProjectPath(projectPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(projectPath, ".claude", "settings.json"), nil
}

// LocalSettingsPath returns the path to local settings.json
func LocalSettingsPath(projectPath string) (string, error) {
	projectPath, err := normalizeProjectPath(projectPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(projectPath, ".claude", "settings.local.json"), nil
}

// normalizeProjectPath validates and normalizes a project path
// Returns absolute, cleaned path; defaults to cwd if empty
func normalizeProjectPath(projectPath string) (string, error) {
	if projectPath == "" {
		return os.Getwd()
	}

	// Resolve to absolute path and clean it
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return "", err
	}

	return filepath.Clean(absPath), nil
}

// ParseScope parses a string into a Scope
func ParseScope(s string) (Scope, error) {
	switch s {
	case "managed":
		return ScopeManaged, nil
	case "user":
		return ScopeUser, nil
	case "project":
		return ScopeProject, nil
	case "local":
		return ScopeLocal, nil
	default:
		return "", ErrInvalidScope
	}
}
