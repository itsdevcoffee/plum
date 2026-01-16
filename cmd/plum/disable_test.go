package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestDisableCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "disable <plugin>" {
			found = true
			break
		}
	}

	if !found {
		t.Error("disable command should be registered as a subcommand")
	}
}

func TestDisableCommandStructure(t *testing.T) {
	if disableCmd.Use != "disable <plugin>" {
		t.Errorf("disableCmd.Use = %q, want %q", disableCmd.Use, "disable <plugin>")
	}

	if disableCmd.Short == "" {
		t.Error("disableCmd.Short should not be empty")
	}

	if disableCmd.RunE == nil {
		t.Error("disableCmd.RunE should not be nil")
	}
}

func TestDisableCommandFlags(t *testing.T) {
	scopeFlag := disableCmd.Flags().Lookup("scope")
	if scopeFlag == nil {
		t.Error("disable command should have --scope flag")
	} else {
		if scopeFlag.Shorthand != "s" {
			t.Errorf("--scope shorthand = %q, want %q", scopeFlag.Shorthand, "s")
		}
		if scopeFlag.DefValue != "user" {
			t.Errorf("--scope default = %q, want %q", scopeFlag.DefValue, "user")
		}
	}

	projectFlag := disableCmd.Flags().Lookup("project")
	if projectFlag == nil {
		t.Error("disable command should have --project flag")
	}
}

func TestDisableCommandHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	disableCmd.SetOut(buf)
	disableCmd.SetErr(buf)

	defer func() {
		disableCmd.SetOut(nil)
		disableCmd.SetErr(nil)
	}()

	err := disableCmd.Help()
	if err != nil {
		t.Fatalf("disableCmd.Help() failed: %v", err)
	}

	output := buf.String()

	expectedStrings := []string{
		"disable",
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
