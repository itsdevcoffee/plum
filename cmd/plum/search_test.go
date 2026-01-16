package main

import (
	"testing"
)

func TestSearchCommand_Structure(t *testing.T) {
	// Verify search command is registered
	cmd, _, err := rootCmd.Find([]string{"search"})
	if err != nil {
		t.Fatalf("search command not found: %v", err)
	}

	if cmd.Use != "search <query>" {
		t.Errorf("expected Use 'search <query>', got %s", cmd.Use)
	}

	// Should require exactly 1 argument
	if cmd.Args == nil {
		t.Error("expected Args validation to be set")
	}

	// Check flags exist
	flags := []string{"json", "marketplace", "category", "limit"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag --%s to exist", flag)
		}
	}
}

func TestSearchCommand_Flags(t *testing.T) {
	cmd, _, _ := rootCmd.Find([]string{"search"})

	// marketplace should have shorthand -m
	mpFlag := cmd.Flags().Lookup("marketplace")
	if mpFlag.Shorthand != "m" {
		t.Errorf("expected marketplace shorthand 'm', got %s", mpFlag.Shorthand)
	}

	// category should have shorthand -c
	catFlag := cmd.Flags().Lookup("category")
	if catFlag.Shorthand != "c" {
		t.Errorf("expected category shorthand 'c', got %s", catFlag.Shorthand)
	}

	// limit should have shorthand -n
	limitFlag := cmd.Flags().Lookup("limit")
	if limitFlag.Shorthand != "n" {
		t.Errorf("expected limit shorthand 'n', got %s", limitFlag.Shorthand)
	}
}

func TestSearchResult_Fields(t *testing.T) {
	r := SearchResult{
		Name:        "test-plugin",
		Marketplace: "test-market",
		Description: "A test plugin",
		Version:     "1.0.0",
		Category:    "productivity",
		Installed:   true,
		Score:       85,
	}

	if r.Name != "test-plugin" {
		t.Error("Name field missing")
	}
	if r.Marketplace != "test-market" {
		t.Error("Marketplace field missing")
	}
	if r.Description != "A test plugin" {
		t.Error("Description field missing")
	}
	if r.Score != 85 {
		t.Error("Score field missing")
	}
}
