package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRemoveCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "remove <plugin>" {
			found = true
			break
		}
	}

	if !found {
		t.Error("remove command should be registered as a subcommand")
	}
}

func TestRemoveCommandStructure(t *testing.T) {
	if removeCmd.Use != "remove <plugin>" {
		t.Errorf("removeCmd.Use = %q, want %q", removeCmd.Use, "remove <plugin>")
	}

	if removeCmd.Short == "" {
		t.Error("removeCmd.Short should not be empty")
	}

	if removeCmd.RunE == nil {
		t.Error("removeCmd.RunE should not be nil")
	}
}

func TestRemoveCommandAliases(t *testing.T) {
	// Remove should have aliases
	hasUninstall := false
	hasRm := false

	for _, alias := range removeCmd.Aliases {
		if alias == "uninstall" {
			hasUninstall = true
		}
		if alias == "rm" {
			hasRm = true
		}
	}

	if !hasUninstall {
		t.Error("remove command should have 'uninstall' alias")
	}
	if !hasRm {
		t.Error("remove command should have 'rm' alias")
	}
}

func TestRemoveCommandFlags(t *testing.T) {
	scopeFlag := removeCmd.Flags().Lookup("scope")
	if scopeFlag == nil {
		t.Error("remove command should have --scope flag")
	} else {
		if scopeFlag.Shorthand != "s" {
			t.Errorf("--scope shorthand = %q, want %q", scopeFlag.Shorthand, "s")
		}
		if scopeFlag.DefValue != "user" {
			t.Errorf("--scope default = %q, want %q", scopeFlag.DefValue, "user")
		}
	}

	projectFlag := removeCmd.Flags().Lookup("project")
	if projectFlag == nil {
		t.Error("remove command should have --project flag")
	}

	allFlag := removeCmd.Flags().Lookup("all")
	if allFlag == nil {
		t.Error("remove command should have --all flag")
	}

	keepCacheFlag := removeCmd.Flags().Lookup("keep-cache")
	if keepCacheFlag == nil {
		t.Error("remove command should have --keep-cache flag")
	}
}

func TestRemoveCommandHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	removeCmd.SetOut(buf)
	removeCmd.SetErr(buf)

	defer func() {
		removeCmd.SetOut(nil)
		removeCmd.SetErr(nil)
	}()

	err := removeCmd.Help()
	if err != nil {
		t.Fatalf("removeCmd.Help() failed: %v", err)
	}

	output := buf.String()

	expectedStrings := []string{
		"remove",
		"uninstall",
		"plugin",
		"--scope",
		"--all",
	}

	lowercaseOutput := strings.ToLower(output)
	for _, expected := range expectedStrings {
		if !strings.Contains(lowercaseOutput, strings.ToLower(expected)) {
			t.Errorf("Help output should contain %q", expected)
		}
	}
}
