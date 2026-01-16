package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestInstallCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "install <plugin>" {
			found = true
			break
		}
	}

	if !found {
		t.Error("install command should be registered as a subcommand")
	}
}

func TestInstallCommandStructure(t *testing.T) {
	if installCmd.Use != "install <plugin>" {
		t.Errorf("installCmd.Use = %q, want %q", installCmd.Use, "install <plugin>")
	}

	if installCmd.Short == "" {
		t.Error("installCmd.Short should not be empty")
	}

	if installCmd.RunE == nil {
		t.Error("installCmd.RunE should not be nil")
	}
}

func TestInstallCommandFlags(t *testing.T) {
	scopeFlag := installCmd.Flags().Lookup("scope")
	if scopeFlag == nil {
		t.Error("install command should have --scope flag")
	} else {
		if scopeFlag.Shorthand != "s" {
			t.Errorf("--scope shorthand = %q, want %q", scopeFlag.Shorthand, "s")
		}
		if scopeFlag.DefValue != "user" {
			t.Errorf("--scope default = %q, want %q", scopeFlag.DefValue, "user")
		}
	}

	projectFlag := installCmd.Flags().Lookup("project")
	if projectFlag == nil {
		t.Error("install command should have --project flag")
	}
}

func TestInstallCommandHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	installCmd.SetOut(buf)
	installCmd.SetErr(buf)

	defer func() {
		installCmd.SetOut(nil)
		installCmd.SetErr(nil)
	}()

	err := installCmd.Help()
	if err != nil {
		t.Fatalf("installCmd.Help() failed: %v", err)
	}

	output := buf.String()

	expectedStrings := []string{
		"install",
		"plugin",
		"marketplace",
		"--scope",
	}

	lowercaseOutput := strings.ToLower(output)
	for _, expected := range expectedStrings {
		if !strings.Contains(lowercaseOutput, strings.ToLower(expected)) {
			t.Errorf("Help output should contain %q", expected)
		}
	}
}
