package main

import (
	"fmt"
	"os"
	"runtime/debug"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/itsdevcoffee/plum/internal/ui"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

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
				cmt = setting.Value[:7] // Short commit hash
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

func main() {
	// Handle --version flag
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		ver, cmt, bDate := getVersion()
		fmt.Printf("plum version %s\n", ver)
		fmt.Printf("  commit: %s\n", cmt)
		fmt.Printf("  built: %s\n", bDate)
		os.Exit(0)
	}

	p := tea.NewProgram(
		ui.NewModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running plum: %v\n", err)
		os.Exit(1)
	}
}
