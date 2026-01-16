package main

import (
	"testing"

	"github.com/itsdevcoffee/plum/internal/plugin"
)

func TestInfoCommand_Structure(t *testing.T) {
	// Verify info command is registered
	cmd, _, err := rootCmd.Find([]string{"info"})
	if err != nil {
		t.Fatalf("info command not found: %v", err)
	}

	if cmd.Use != "info <plugin>" {
		t.Errorf("expected Use 'info <plugin>', got %s", cmd.Use)
	}

	// Should require exactly 1 argument
	if cmd.Args == nil {
		t.Error("expected Args validation to be set")
	}

	// Check flags exist
	flags := []string{"json", "project"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag --%s to exist", flag)
		}
	}
}

func TestBuildPluginInfo(t *testing.T) {
	p := plugin.Plugin{
		Name:            "test-plugin",
		Description:     "A test plugin",
		Version:         "1.0.0",
		Marketplace:     "test-market",
		MarketplaceRepo: "https://github.com/test/repo",
		License:         "MIT",
		Category:        "productivity",
		Keywords:        []string{"test", "example"},
		Tags:            []string{"official"},
		Author: plugin.Author{
			Name:  "Test Author",
			Email: "test@example.com",
		},
		Installed:   true,
		InstallPath: "/path/to/plugin",
	}

	info := buildPluginInfo(p)

	if info.Name != "test-plugin" {
		t.Errorf("expected name 'test-plugin', got %s", info.Name)
	}
	if info.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %s", info.Version)
	}
	if info.Description != "A test plugin" {
		t.Errorf("expected description 'A test plugin', got %s", info.Description)
	}
	if info.Author != "Test Author" {
		t.Errorf("expected author 'Test Author', got %s", info.Author)
	}
	if info.License != "MIT" {
		t.Errorf("expected license 'MIT', got %s", info.License)
	}
	if info.Marketplace != "test-market" {
		t.Errorf("expected marketplace 'test-market', got %s", info.Marketplace)
	}
	if info.Category != "productivity" {
		t.Errorf("expected category 'productivity', got %s", info.Category)
	}
	if len(info.Keywords) != 2 {
		t.Errorf("expected 2 keywords, got %d", len(info.Keywords))
	}
}

func TestBuildPluginInfo_UnknownAuthor(t *testing.T) {
	p := plugin.Plugin{
		Name:        "test-plugin",
		Marketplace: "test-market",
		// No author info set
	}

	info := buildPluginInfo(p)

	if info.Author != "Unknown" {
		t.Errorf("expected author 'Unknown', got %s", info.Author)
	}
}

func TestPluginInfo_JSONFields(t *testing.T) {
	info := PluginInfo{
		Name:        "test",
		Version:     "1.0.0",
		Description: "desc",
		Marketplace: "market",
		Installed:   true,
		Status:      "enabled",
		Scope:       "user",
	}

	// Verify all expected fields are present
	if info.Name != "test" {
		t.Error("Name field missing")
	}
	if info.Version != "1.0.0" {
		t.Error("Version field missing")
	}
	if info.Installed != true {
		t.Error("Installed field missing")
	}
}
