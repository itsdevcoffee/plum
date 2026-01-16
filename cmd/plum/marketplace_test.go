package main

import (
	"testing"
)

func TestMarketplaceCommand_Structure(t *testing.T) {
	// Verify marketplace command is registered
	cmd, _, err := rootCmd.Find([]string{"marketplace"})
	if err != nil {
		t.Fatalf("marketplace command not found: %v", err)
	}

	if cmd.Use != "marketplace" {
		t.Errorf("expected Use 'marketplace', got %s", cmd.Use)
	}
}

func TestMarketplaceListCommand_Structure(t *testing.T) {
	// Verify marketplace list subcommand is registered
	cmd, _, err := rootCmd.Find([]string{"marketplace", "list"})
	if err != nil {
		t.Fatalf("marketplace list command not found: %v", err)
	}

	if cmd.Use != "list" {
		t.Errorf("expected Use 'list', got %s", cmd.Use)
	}

	// Check flags exist
	flags := []string{"json", "project"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag --%s to exist", flag)
		}
	}
}

func TestMarketplaceListItem_Fields(t *testing.T) {
	item := MarketplaceListItem{
		Name:        "test-marketplace",
		DisplayName: "Test Marketplace",
		Repo:        "https://github.com/test/marketplace",
		Description: "A test marketplace",
		PluginCount: 42,
		Installed:   true,
		Source:      "github",
		Stars:       1234,
	}

	if item.Name != "test-marketplace" {
		t.Error("Name field incorrect")
	}
	if item.DisplayName != "Test Marketplace" {
		t.Error("DisplayName field incorrect")
	}
	if item.PluginCount != 42 {
		t.Error("PluginCount field incorrect")
	}
	if item.Stars != 1234 {
		t.Error("Stars field incorrect")
	}
	if !item.Installed {
		t.Error("Installed field incorrect")
	}
}

func TestMarketplaceCommand_HasListSubcommand(t *testing.T) {
	cmd, _, _ := rootCmd.Find([]string{"marketplace"})

	// Check that 'list' is a subcommand
	found := false
	for _, sub := range cmd.Commands() {
		if sub.Use == "list" {
			found = true
			break
		}
	}

	if !found {
		t.Error("marketplace command should have 'list' subcommand")
	}
}
