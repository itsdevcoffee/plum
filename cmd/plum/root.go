package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/itsdevcoffee/plum/internal/ui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "plum",
	Short: "Plugin manager for Claude Code",
	Long: `Plum is an interactive plugin manager for Claude Code.

Run without arguments to browse and manage plugins interactively,
or use subcommands for specific operations.`,
	// When run without subcommand, launch the TUI
	Run: func(cmd *cobra.Command, args []string) {
		runTUI()
	},
}

func init() {
	// Set version for --version flag
	ver, _, _ := getVersion()
	rootCmd.Version = ver

	// Customize version template to show full version info
	rootCmd.SetVersionTemplate(formatVersion() + "\n")
}

// Execute runs the root command
func Execute() {
	// Cobra prints errors to stderr automatically, just handle exit code
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// runTUI launches the Bubbletea TUI
func runTUI() {
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
