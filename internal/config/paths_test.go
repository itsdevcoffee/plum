package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestClaudeConfigDir(t *testing.T) {
	t.Run("with CLAUDE_CONFIG_DIR override", func(t *testing.T) {
		customDir := "/custom/claude/config"
		t.Setenv("CLAUDE_CONFIG_DIR", customDir)

		got, err := ClaudeConfigDir()
		if err != nil {
			t.Fatalf("ClaudeConfigDir() error = %v", err)
		}
		if got != customDir {
			t.Errorf("ClaudeConfigDir() = %q, want %q", got, customDir)
		}
	})

	t.Run("default path on unix", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping Unix test on Windows")
		}

		t.Setenv("CLAUDE_CONFIG_DIR", "")

		got, err := ClaudeConfigDir()
		if err != nil {
			t.Fatalf("ClaudeConfigDir() error = %v", err)
		}

		home, _ := os.UserHomeDir()
		want := filepath.Join(home, ".claude")
		if got != want {
			t.Errorf("ClaudeConfigDir() = %q, want %q", got, want)
		}
	})

	t.Run("default path on windows with APPDATA", func(t *testing.T) {
		if runtime.GOOS != "windows" {
			t.Skip("Skipping Windows test on Unix")
		}

		t.Setenv("CLAUDE_CONFIG_DIR", "")
		appdata := "C:\\Users\\TestUser\\AppData\\Roaming"
		t.Setenv("APPDATA", appdata)

		got, err := ClaudeConfigDir()
		if err != nil {
			t.Fatalf("ClaudeConfigDir() error = %v", err)
		}

		want := filepath.Join(appdata, "ClaudeCode")
		if got != want {
			t.Errorf("ClaudeConfigDir() = %q, want %q", got, want)
		}
	})

	t.Run("default path on windows without APPDATA", func(t *testing.T) {
		if runtime.GOOS != "windows" {
			t.Skip("Skipping Windows test on Unix")
		}

		t.Setenv("CLAUDE_CONFIG_DIR", "")
		t.Setenv("APPDATA", "")

		got, err := ClaudeConfigDir()
		if err != nil {
			t.Fatalf("ClaudeConfigDir() error = %v", err)
		}

		home, _ := os.UserHomeDir()
		want := filepath.Join(home, ".claude")
		if got != want {
			t.Errorf("ClaudeConfigDir() = %q, want %q", got, want)
		}
	})

	t.Run("path format validation", func(t *testing.T) {
		t.Setenv("CLAUDE_CONFIG_DIR", "")

		got, err := ClaudeConfigDir()
		if err != nil {
			t.Fatalf("ClaudeConfigDir() error = %v", err)
		}

		// Should be an absolute path
		if !filepath.IsAbs(got) {
			t.Errorf("ClaudeConfigDir() = %q, want absolute path", got)
		}

		// Should end with .claude or ClaudeCode
		if !strings.HasSuffix(got, ".claude") && !strings.HasSuffix(got, "ClaudeCode") {
			t.Errorf("ClaudeConfigDir() = %q, want path ending with .claude or ClaudeCode", got)
		}
	})
}

func TestClaudePluginsDir(t *testing.T) {
	t.Run("derived from config dir", func(t *testing.T) {
		customDir := "/custom/claude/config"
		t.Setenv("CLAUDE_CONFIG_DIR", customDir)

		got, err := ClaudePluginsDir()
		if err != nil {
			t.Fatalf("ClaudePluginsDir() error = %v", err)
		}

		want := filepath.Join(customDir, "plugins")
		if got != want {
			t.Errorf("ClaudePluginsDir() = %q, want %q", got, want)
		}
	})

	t.Run("absolute path", func(t *testing.T) {
		t.Setenv("CLAUDE_CONFIG_DIR", "")

		got, err := ClaudePluginsDir()
		if err != nil {
			t.Fatalf("ClaudePluginsDir() error = %v", err)
		}

		if !filepath.IsAbs(got) {
			t.Errorf("ClaudePluginsDir() = %q, want absolute path", got)
		}

		if !strings.HasSuffix(got, "plugins") {
			t.Errorf("ClaudePluginsDir() = %q, want path ending with plugins", got)
		}
	})
}

func TestKnownMarketplacesPath(t *testing.T) {
	t.Run("correct file path", func(t *testing.T) {
		customDir := "/custom/claude/config"
		t.Setenv("CLAUDE_CONFIG_DIR", customDir)

		got, err := KnownMarketplacesPath()
		if err != nil {
			t.Fatalf("KnownMarketplacesPath() error = %v", err)
		}

		want := filepath.Join(customDir, "plugins", "known_marketplaces.json")
		if got != want {
			t.Errorf("KnownMarketplacesPath() = %q, want %q", got, want)
		}
	})

	t.Run("absolute path with json extension", func(t *testing.T) {
		t.Setenv("CLAUDE_CONFIG_DIR", "")

		got, err := KnownMarketplacesPath()
		if err != nil {
			t.Fatalf("KnownMarketplacesPath() error = %v", err)
		}

		if !filepath.IsAbs(got) {
			t.Errorf("KnownMarketplacesPath() = %q, want absolute path", got)
		}

		if !strings.HasSuffix(got, "known_marketplaces.json") {
			t.Errorf("KnownMarketplacesPath() = %q, want path ending with known_marketplaces.json", got)
		}
	})
}

func TestInstalledPluginsPath(t *testing.T) {
	t.Run("correct file path", func(t *testing.T) {
		customDir := "/custom/claude/config"
		t.Setenv("CLAUDE_CONFIG_DIR", customDir)

		got, err := InstalledPluginsPath()
		if err != nil {
			t.Fatalf("InstalledPluginsPath() error = %v", err)
		}

		want := filepath.Join(customDir, "plugins", "installed_plugins.json")
		if got != want {
			t.Errorf("InstalledPluginsPath() = %q, want %q", got, want)
		}
	})

	t.Run("absolute path with json extension", func(t *testing.T) {
		t.Setenv("CLAUDE_CONFIG_DIR", "")

		got, err := InstalledPluginsPath()
		if err != nil {
			t.Fatalf("InstalledPluginsPath() error = %v", err)
		}

		if !filepath.IsAbs(got) {
			t.Errorf("InstalledPluginsPath() = %q, want absolute path", got)
		}

		if !strings.HasSuffix(got, "installed_plugins.json") {
			t.Errorf("InstalledPluginsPath() = %q, want path ending with installed_plugins.json", got)
		}
	})
}
