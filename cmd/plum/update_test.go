package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestUpdateCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "update [plugin]" {
			found = true
			break
		}
	}

	if !found {
		t.Error("update command should be registered as a subcommand")
	}
}

func TestUpdateCommandStructure(t *testing.T) {
	if updateCmd.Use != "update [plugin]" {
		t.Errorf("updateCmd.Use = %q, want %q", updateCmd.Use, "update [plugin]")
	}

	if updateCmd.Short == "" {
		t.Error("updateCmd.Short should not be empty")
	}

	if updateCmd.RunE == nil {
		t.Error("updateCmd.RunE should not be nil")
	}
}

func TestUpdateCommandFlags(t *testing.T) {
	scopeFlag := updateCmd.Flags().Lookup("scope")
	if scopeFlag == nil {
		t.Error("update command should have --scope flag")
	} else if scopeFlag.Shorthand != "s" {
		t.Errorf("--scope shorthand = %q, want %q", scopeFlag.Shorthand, "s")
	}

	projectFlag := updateCmd.Flags().Lookup("project")
	if projectFlag == nil {
		t.Error("update command should have --project flag")
	}

	dryRunFlag := updateCmd.Flags().Lookup("dry-run")
	if dryRunFlag == nil {
		t.Error("update command should have --dry-run flag")
	}
}

func TestUpdateCommandHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	updateCmd.SetOut(buf)
	updateCmd.SetErr(buf)

	defer func() {
		updateCmd.SetOut(nil)
		updateCmd.SetErr(nil)
	}()

	err := updateCmd.Help()
	if err != nil {
		t.Fatalf("updateCmd.Help() failed: %v", err)
	}

	output := buf.String()

	expectedStrings := []string{
		"update",
		"plugin",
		"--dry-run",
	}

	lowercaseOutput := strings.ToLower(output)
	for _, expected := range expectedStrings {
		if !strings.Contains(lowercaseOutput, strings.ToLower(expected)) {
			t.Errorf("Help output should contain %q", expected)
		}
	}
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected bool
	}{
		{"1.1.0", "1.0.0", true},
		{"2.0.0", "1.9.9", true},
		{"1.0.1", "1.0.0", true},
		{"1.0.0", "1.0.0", false},
		{"1.0.0", "1.1.0", false},
		{"1.0.0", "2.0.0", false},
		{"v1.1.0", "v1.0.0", true},
		{"1.1.0", "v1.0.0", true},
		{"v1.0.0", "1.0.0", false},
		{"1.2.3-alpha", "1.2.2", true},
		{"1.2.3-beta", "1.2.3-alpha", true},
	}

	for _, tt := range tests {
		t.Run(tt.v1+"_vs_"+tt.v2, func(t *testing.T) {
			result := isNewerVersion(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("isNewerVersion(%q, %q) = %v, want %v", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}
