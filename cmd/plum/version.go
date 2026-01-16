package main

import (
	"fmt"
	"runtime/debug"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// getVersion returns version info from build flags or debug info
func getVersion() (ver, cmt, bDate string) {
	// Try to get version from build info (works with go install)
	if info, ok := debug.ReadBuildInfo(); ok && version == "dev" {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			ver = info.Main.Version
		} else {
			ver = version
		}

		// Try to get commit from build settings
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" && commit == "none" {
				cmt = truncateCommitHash(setting.Value)
			}
			if setting.Key == "vcs.time" && date == "unknown" {
				bDate = setting.Value
			}
		}
	}

	// Use ldflags values if set (from GoReleaser)
	if ver == "" {
		ver = version
	}
	if cmt == "" {
		cmt = commit
	}
	if bDate == "" {
		bDate = date
	}

	return ver, cmt, bDate
}

// truncateCommitHash safely truncates a commit hash to 7 characters
func truncateCommitHash(hash string) string {
	if len(hash) >= 7 {
		return hash[:7]
	}
	return hash
}

// formatVersion returns a formatted version string for display
func formatVersion() string {
	ver, cmt, bDate := getVersion()
	result := fmt.Sprintf("plum version %s", ver)

	// Only show commit and build time if available
	if cmt != "none" && cmt != "" {
		result += fmt.Sprintf("\n  commit: %s", cmt)
	}
	if bDate != "unknown" && bDate != "" {
		result += fmt.Sprintf("\n  built: %s", bDate)
	}

	return result
}
