package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestEnableCommandRegistered(t *testing.T) {
	// Enable command should be registered as a subcommand
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "enable <plugin>" {
			found = true
			break
		}
	}

	if !found {
		t.Error("enable command should be registered as a subcommand")
	}
}

func TestEnableCommandStructure(t *testing.T) {
	if enableCmd.Use != "enable <plugin>" {
		t.Errorf("enableCmd.Use = %q, want %q", enableCmd.Use, "enable <plugin>")
	}

	if enableCmd.Short == "" {
		t.Error("enableCmd.Short should not be empty")
	}

	if enableCmd.RunE == nil {
		t.Error("enableCmd.RunE should not be nil")
	}
}

func TestEnableCommandFlags(t *testing.T) {
	// Check scope flag exists
	scopeFlag := enableCmd.Flags().Lookup("scope")
	if scopeFlag == nil {
		t.Error("enable command should have --scope flag")
	} else {
		if scopeFlag.Shorthand != "s" {
			t.Errorf("--scope shorthand = %q, want %q", scopeFlag.Shorthand, "s")
		}
		if scopeFlag.DefValue != "user" {
			t.Errorf("--scope default = %q, want %q", scopeFlag.DefValue, "user")
		}
	}

	// Check project flag exists
	projectFlag := enableCmd.Flags().Lookup("project")
	if projectFlag == nil {
		t.Error("enable command should have --project flag")
	}
}

func TestEnableCommandHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	enableCmd.SetOut(buf)
	enableCmd.SetErr(buf)

	defer func() {
		enableCmd.SetOut(nil)
		enableCmd.SetErr(nil)
	}()

	err := enableCmd.Help()
	if err != nil {
		t.Fatalf("enableCmd.Help() failed: %v", err)
	}

	output := buf.String()

	// Verify help contains expected content
	expectedStrings := []string{
		"enable",
		"plugin",
		"--scope",
	}

	lowercaseOutput := strings.ToLower(output)
	for _, expected := range expectedStrings {
		if !strings.Contains(lowercaseOutput, strings.ToLower(expected)) {
			t.Errorf("Help output should contain %q", expected)
		}
	}
}

func TestEnableCommandRequiresArg(t *testing.T) {
	// Enable command requires exactly 1 argument
	if enableCmd.Args == nil {
		t.Error("enableCmd.Args should not be nil")
	}
}
