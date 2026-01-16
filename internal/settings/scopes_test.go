package settings

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAllScopes(t *testing.T) {
	scopes := AllScopes()

	// Should return 4 scopes in precedence order
	if len(scopes) != 4 {
		t.Errorf("expected 4 scopes, got %d", len(scopes))
	}

	// Precedence order: Managed > Local > Project > User
	expected := []Scope{ScopeManaged, ScopeLocal, ScopeProject, ScopeUser}
	for i, scope := range scopes {
		if scope != expected[i] {
			t.Errorf("scope %d: expected %s, got %s", i, expected[i], scope)
		}
	}
}

func TestWritableScopes(t *testing.T) {
	scopes := WritableScopes()

	// Should return 3 writable scopes (excludes managed)
	if len(scopes) != 3 {
		t.Errorf("expected 3 writable scopes, got %d", len(scopes))
	}

	// Should not include managed
	for _, scope := range scopes {
		if scope == ScopeManaged {
			t.Error("writable scopes should not include managed")
		}
	}
}

func TestScopeIsWritable(t *testing.T) {
	tests := []struct {
		scope    Scope
		writable bool
	}{
		{ScopeManaged, false},
		{ScopeUser, true},
		{ScopeProject, true},
		{ScopeLocal, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.scope), func(t *testing.T) {
			if got := tt.scope.IsWritable(); got != tt.writable {
				t.Errorf("Scope(%s).IsWritable() = %v, want %v", tt.scope, got, tt.writable)
			}
		})
	}
}

func TestParseScope(t *testing.T) {
	tests := []struct {
		input   string
		want    Scope
		wantErr bool
	}{
		{"managed", ScopeManaged, false},
		{"user", ScopeUser, false},
		{"project", ScopeProject, false},
		{"local", ScopeLocal, false},
		{"invalid", "", true},
		{"", "", true},
		{"MANAGED", "", true}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseScope(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseScope(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseScope(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestManagedSettingsPath(t *testing.T) {
	path, err := ManagedSettingsPath()
	if err != nil {
		t.Fatalf("ManagedSettingsPath() error = %v", err)
	}

	if runtime.GOOS == "windows" {
		if filepath.Base(path) != "settings.json" {
			t.Errorf("expected settings.json, got %s", filepath.Base(path))
		}
	} else {
		expected := "/etc/claude-code/settings.json"
		if path != expected {
			t.Errorf("expected %s, got %s", expected, path)
		}
	}
}

func TestUserSettingsPath(t *testing.T) {
	path, err := UserSettingsPath()
	if err != nil {
		t.Fatalf("UserSettingsPath() error = %v", err)
	}

	// Should end with settings.json
	if filepath.Base(path) != "settings.json" {
		t.Errorf("expected settings.json, got %s", filepath.Base(path))
	}

	// Parent dir should be .claude
	if filepath.Base(filepath.Dir(path)) != ".claude" {
		t.Errorf("expected parent dir .claude, got %s", filepath.Base(filepath.Dir(path)))
	}
}

func TestProjectSettingsPath(t *testing.T) {
	// Test with explicit project path
	projectPath := "/tmp/test-project"
	path, err := ProjectSettingsPath(projectPath)
	if err != nil {
		t.Fatalf("ProjectSettingsPath() error = %v", err)
	}

	expected := filepath.Join(projectPath, ".claude", "settings.json")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestLocalSettingsPath(t *testing.T) {
	// Test with explicit project path
	projectPath := "/tmp/test-project"
	path, err := LocalSettingsPath(projectPath)
	if err != nil {
		t.Fatalf("LocalSettingsPath() error = %v", err)
	}

	expected := filepath.Join(projectPath, ".claude", "settings.local.json")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestScopePath(t *testing.T) {
	projectPath := "/tmp/test-project"

	tests := []struct {
		scope    Scope
		wantErr  bool
		contains string // path should contain this
	}{
		{ScopeManaged, false, "settings.json"},
		{ScopeUser, false, "settings.json"},
		{ScopeProject, false, "settings.json"},
		{ScopeLocal, false, "settings.local.json"},
		{Scope("invalid"), true, ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.scope), func(t *testing.T) {
			path, err := ScopePath(tt.scope, projectPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ScopePath(%s) error = %v, wantErr %v", tt.scope, err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.contains != "" {
				if filepath.Base(path) != tt.contains {
					t.Errorf("ScopePath(%s) = %s, want to contain %s", tt.scope, path, tt.contains)
				}
			}
		})
	}
}

func TestProjectSettingsPathDefaultsToCwd(t *testing.T) {
	// Test that empty project path defaults to cwd
	path, err := ProjectSettingsPath("")
	if err != nil {
		t.Fatalf("ProjectSettingsPath(\"\") error = %v", err)
	}

	cwd, _ := os.Getwd()
	expected := filepath.Join(cwd, ".claude", "settings.json")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestNormalizeProjectPath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantAbs bool // should result be absolute
	}{
		{
			name:    "empty defaults to cwd",
			input:   "",
			wantAbs: true,
		},
		{
			name:    "absolute path stays absolute",
			input:   "/tmp/test-project",
			wantAbs: true,
		},
		{
			name:    "relative path becomes absolute",
			input:   "relative/path",
			wantAbs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizeProjectPath(tt.input)
			if err != nil {
				t.Fatalf("normalizeProjectPath(%q) error = %v", tt.input, err)
			}

			if tt.wantAbs && !filepath.IsAbs(result) {
				t.Errorf("normalizeProjectPath(%q) = %s, want absolute path", tt.input, result)
			}
		})
	}
}

func TestNormalizeProjectPathCleansPath(t *testing.T) {
	// Test that path is cleaned (redundant slashes, dots removed)
	input := "/tmp/../tmp/test-project/./subdir/../"
	result, err := normalizeProjectPath(input)
	if err != nil {
		t.Fatalf("normalizeProjectPath error = %v", err)
	}

	// Path should be cleaned - no .. or . or trailing slash
	if filepath.Base(result) == ".." || filepath.Base(result) == "." {
		t.Errorf("path not properly cleaned: %s", result)
	}
}

func TestProjectSettingsPathWithRelativePath(t *testing.T) {
	// Create a temp dir and use a relative path to it
	tmpDir := t.TempDir()

	// Get relative path from cwd to tmpDir
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	relPath, err := filepath.Rel(cwd, tmpDir)
	if err != nil {
		// Can't create relative path (different drives on Windows), skip
		t.Skip("cannot create relative path")
	}

	path, err := ProjectSettingsPath(relPath)
	if err != nil {
		t.Fatalf("ProjectSettingsPath(%q) error = %v", relPath, err)
	}

	// Result should be absolute
	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got %s", path)
	}

	// Should end with settings.json
	if filepath.Base(path) != "settings.json" {
		t.Errorf("expected settings.json, got %s", filepath.Base(path))
	}
}
