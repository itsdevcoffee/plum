package main

import (
	"strings"
	"testing"
)

func TestFormatVersion(t *testing.T) {
	result := formatVersion()

	// Should always start with "plum version"
	if !strings.HasPrefix(result, "plum version ") {
		t.Errorf("formatVersion() should start with 'plum version ', got: %s", result)
	}

	// In dev/test mode, version should be present
	if !strings.Contains(result, "version") {
		t.Errorf("formatVersion() should contain version info, got: %s", result)
	}
}

func TestFormatVersionStructure(t *testing.T) {
	result := formatVersion()
	lines := strings.Split(result, "\n")

	// First line should be version
	if len(lines) < 1 {
		t.Fatal("formatVersion() should have at least one line")
	}

	if !strings.HasPrefix(lines[0], "plum version ") {
		t.Errorf("First line should be version, got: %s", lines[0])
	}

	// Additional lines (if present) should be indented with 2 spaces
	for i := 1; i < len(lines); i++ {
		if lines[i] != "" && !strings.HasPrefix(lines[i], "  ") {
			t.Errorf("Line %d should be indented with 2 spaces, got: %s", i+1, lines[i])
		}
	}
}

func TestGetVersion(t *testing.T) {
	ver, cmt, bDate := getVersion()

	// Version should never be empty
	if ver == "" {
		t.Error("getVersion() version should not be empty")
	}

	// Commit should never be empty (defaults to "none")
	if cmt == "" {
		t.Error("getVersion() commit should not be empty")
	}

	// Build date should never be empty (defaults to "unknown")
	if bDate == "" {
		t.Error("getVersion() build date should not be empty")
	}
}

func TestTruncateCommitHash(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "full SHA",
			input:    "abc123def456789",
			expected: "abc123d",
		},
		{
			name:     "exactly 7 chars",
			input:    "abc123d",
			expected: "abc123d",
		},
		{
			name:     "short hash",
			input:    "abc",
			expected: "abc",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "6 chars",
			input:    "abc123",
			expected: "abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateCommitHash(tt.input)
			if result != tt.expected {
				t.Errorf("truncateCommitHash(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
