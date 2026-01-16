package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommandStructure(t *testing.T) {
	// Verify root command is properly configured
	if rootCmd.Use != "plum" {
		t.Errorf("rootCmd.Use = %q, want %q", rootCmd.Use, "plum")
	}

	if rootCmd.Short == "" {
		t.Error("rootCmd.Short should not be empty")
	}

	if rootCmd.Long == "" {
		t.Error("rootCmd.Long should not be empty")
	}

	if rootCmd.Run == nil {
		t.Error("rootCmd.Run should not be nil")
	}
}

func TestRootCommandVersion(t *testing.T) {
	// Version should be set
	if rootCmd.Version == "" {
		t.Error("rootCmd.Version should not be empty")
	}
}

func TestBrowseCommandRegistered(t *testing.T) {
	// Browse command should be registered as a subcommand
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "browse" {
			found = true
			break
		}
	}

	if !found {
		t.Error("browse command should be registered as a subcommand")
	}
}

func TestBrowseCommandStructure(t *testing.T) {
	if browseCmd.Use != "browse" {
		t.Errorf("browseCmd.Use = %q, want %q", browseCmd.Use, "browse")
	}

	if browseCmd.Short == "" {
		t.Error("browseCmd.Short should not be empty")
	}

	if browseCmd.Run == nil {
		t.Error("browseCmd.Run should not be nil")
	}
}

func TestHelpOutput(t *testing.T) {
	// Capture help output
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--help"})

	// Reset command state
	defer func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("rootCmd.Execute() with --help failed: %v", err)
	}

	output := buf.String()

	// Verify help contains expected sections (case-insensitive for "plugin manager")
	expectedStrings := []string{
		"plum",
		"plugin manager", // lowercase to match actual output
		"browse",
		"--help",
		"--version",
	}

	lowercaseOutput := strings.ToLower(output)
	for _, expected := range expectedStrings {
		if !strings.Contains(lowercaseOutput, strings.ToLower(expected)) {
			t.Errorf("Help output should contain %q (case-insensitive)", expected)
		}
	}
}

func TestVersionOutput(t *testing.T) {
	// Test the version string directly rather than through Cobra execution
	// This avoids state issues with Cobra's command execution
	output := formatVersion()

	// Should contain version info
	if !strings.Contains(output, "plum version") {
		t.Errorf("Version output should contain 'plum version', got: %s", output)
	}
}
