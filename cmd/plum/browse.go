package main

import (
	"github.com/spf13/cobra"
)

var browseCmd = &cobra.Command{
	Use:   "browse",
	Short: "Browse and manage plugins interactively",
	Long: `Opens the interactive TUI for browsing, installing, and managing
Claude Code plugins from the marketplace and local sources.`,
	Run: func(cmd *cobra.Command, args []string) {
		runTUI()
	},
}

func init() {
	rootCmd.AddCommand(browseCmd)
}
