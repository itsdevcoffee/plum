package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/itsdevcoffee/plum/internal/config"
	"github.com/itsdevcoffee/plum/internal/plugin"
	"github.com/itsdevcoffee/plum/internal/settings"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info <plugin>",
	Short: "Show detailed plugin information",
	Long: `Display detailed information about a plugin.

The plugin can be specified as:
  - plugin-name (searches all marketplaces)
  - plugin-name@marketplace (specific marketplace)

Examples:
  plum info ralph-wiggum
  plum info ralph-wiggum@claude-code-plugins
  plum info memory --json`,
	Args: cobra.ExactArgs(1),
	RunE: runInfo,
}

var (
	infoJSON    bool
	infoProject string
)

func init() {
	rootCmd.AddCommand(infoCmd)

	infoCmd.Flags().BoolVar(&infoJSON, "json", false, "Output as JSON")
	infoCmd.Flags().StringVar(&infoProject, "project", "", "Project path (default: current directory)")
}

// PluginInfo represents detailed plugin information
type PluginInfo struct {
	Name             string   `json:"name"`
	Version          string   `json:"version"`
	Description      string   `json:"description"`
	Author           string   `json:"author"`
	License          string   `json:"license"`
	Marketplace      string   `json:"marketplace"`
	MarketplaceRepo  string   `json:"marketplaceRepo,omitempty"`
	Repository       string   `json:"repository,omitempty"`
	Homepage         string   `json:"homepage,omitempty"`
	Category         string   `json:"category,omitempty"`
	Keywords         []string `json:"keywords,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	Installed        bool     `json:"installed"`
	Status           string   `json:"status,omitempty"`
	Scope            string   `json:"scope,omitempty"`
	InstallPath      string   `json:"installPath,omitempty"`
	InstalledVersion string   `json:"installedVersion,omitempty"`
	InstalledAt      string   `json:"installedAt,omitempty"`
	IsLocal          bool     `json:"isLocal,omitempty"`
	GitHubURL        string   `json:"githubUrl,omitempty"`
}

func runInfo(cmd *cobra.Command, args []string) error {
	query := args[0]

	// Parse plugin@marketplace format
	pluginName := query
	marketplaceFilter := ""
	if idx := strings.LastIndex(query, "@"); idx > 0 {
		pluginName = query[:idx]
		marketplaceFilter = query[idx+1:]
	}

	// Load all plugins
	plugins, err := config.LoadAllPlugins()
	if err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	// Find matching plugin
	var found *PluginInfo
	for _, p := range plugins {
		if p.Name == pluginName {
			// If marketplace filter specified, must match
			if marketplaceFilter != "" && p.Marketplace != marketplaceFilter {
				continue
			}
			found = buildPluginInfo(p)
			break
		}
	}

	if found == nil {
		return fmt.Errorf("plugin '%s' not found", query)
	}

	// Get additional state from settings
	fullName := found.Name + "@" + found.Marketplace
	state, err := settings.GetPluginState(fullName, infoProject)
	if err == nil && state != nil {
		found.Scope = state.Scope.String()
		if state.Enabled {
			found.Status = "enabled"
		} else {
			found.Status = "disabled"
		}
	}

	// Get installation details
	installed, _ := config.LoadInstalledPlugins()
	if installs, ok := installed.Plugins[fullName]; ok && len(installs) > 0 {
		install := installs[0]
		found.Installed = true
		found.InstallPath = install.InstallPath
		found.InstalledVersion = install.Version
		found.InstalledAt = install.InstalledAt
		found.IsLocal = install.IsLocal
	}

	// Output
	if infoJSON {
		return outputInfoJSON(found)
	}
	return outputInfoFormatted(found)
}

func buildPluginInfo(p plugin.Plugin) *PluginInfo {
	return &PluginInfo{
		Name:            p.Name,
		Version:         p.Version,
		Description:     p.Description,
		Author:          p.AuthorName(),
		License:         p.License,
		Marketplace:     p.Marketplace,
		MarketplaceRepo: p.MarketplaceRepo,
		Repository:      p.Repository,
		Homepage:        p.Homepage,
		Category:        p.Category,
		Keywords:        p.Keywords,
		Tags:            p.Tags,
		Installed:       p.Installed,
		InstallPath:     p.InstallPath,
		GitHubURL:       p.GitHubURL(),
	}
}

func outputInfoJSON(info *PluginInfo) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(info)
}

func outputInfoFormatted(info *PluginInfo) error {
	// Name and version
	fmt.Printf("Name:        %s\n", info.Name)
	if info.Version != "" {
		fmt.Printf("Version:     %s\n", info.Version)
	}
	if info.Description != "" {
		fmt.Printf("Description: %s\n", info.Description)
	}
	if info.Author != "" && info.Author != "Unknown" {
		fmt.Printf("Author:      %s\n", info.Author)
	}
	if info.License != "" {
		fmt.Printf("License:     %s\n", info.License)
	}
	fmt.Printf("Marketplace: %s\n", info.Marketplace)
	if info.MarketplaceRepo != "" {
		fmt.Printf("Repository:  %s\n", info.MarketplaceRepo)
	}
	if info.Category != "" {
		fmt.Printf("Category:    %s\n", info.Category)
	}
	if len(info.Keywords) > 0 {
		fmt.Printf("Keywords:    %s\n", strings.Join(info.Keywords, ", "))
	}

	fmt.Println()

	// Installation status
	if info.Installed {
		fmt.Printf("Installed:   Yes (%s scope)\n", info.Scope)
		fmt.Printf("Status:      %s\n", info.Status)
		if info.InstallPath != "" {
			fmt.Printf("Path:        %s\n", info.InstallPath)
		}
		if info.InstalledVersion != "" {
			fmt.Printf("Inst. Ver:   %s\n", info.InstalledVersion)
		}
	} else {
		fmt.Println("Installed:   No")
	}

	if info.GitHubURL != "" {
		fmt.Printf("\nSource:      %s\n", info.GitHubURL)
	}

	return nil
}
