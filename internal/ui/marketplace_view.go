package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// marketplaceListView renders the marketplace browser view
func (m Model) marketplaceListView() string {
	var b strings.Builder

	// Title
	title := TitleStyle.Render("🍑 plum - Marketplace Browser")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Sort tabs
	b.WriteString(m.renderMarketplaceSortTabs())
	b.WriteString("\n\n")

	// Marketplace list
	if len(m.marketplaceItems) == 0 {
		b.WriteString(DescriptionStyle.Render("No marketplaces found. Press Shift+U to refresh."))
	} else {
		visible := m.VisibleMarketplaceItems()
		offset := m.marketplaceScrollOffset

		for i, item := range visible {
			actualIdx := offset + i
			isSelected := actualIdx == m.marketplaceCursor
			b.WriteString(m.renderMarketplaceItem(item, isSelected))
			b.WriteString("\n")
		}
	}

	// Status bar
	b.WriteString("\n")
	b.WriteString(m.marketplaceStatusBar())

	return AppStyle.Render(b.String())
}

// renderMarketplaceItem renders a single marketplace entry
func (m Model) renderMarketplaceItem(item MarketplaceItem, selected bool) string {
	// Status indicator
	var indicator string
	switch item.Status {
	case MarketplaceInstalled:
		indicator = InstalledIndicator.String()
	case MarketplaceCached:
		indicator = "◆" // Diamond for cached
	case MarketplaceNew:
		indicator = "★" // Star for new
	default:
		indicator = AvailableIndicator.String()
	}

	// Selection prefix
	var prefix string
	if selected {
		prefix = HighlightBarFull.String()
	} else {
		prefix = "  "
	}

	// Name style
	var nameStyle lipgloss.Style
	if selected {
		nameStyle = PluginNameSelectedStyle
	} else {
		nameStyle = PluginNameStyle
	}

	name := nameStyle.Render(item.DisplayName)

	// Plugin count
	var pluginCountStr string
	if item.TotalPluginCount > 0 {
		if item.InstalledPluginCount > 0 {
			pluginCountStr = fmt.Sprintf("(%d/%d plugins)",
				item.InstalledPluginCount, item.TotalPluginCount)
		} else {
			pluginCountStr = fmt.Sprintf("(%d plugins)", item.TotalPluginCount)
		}
	} else {
		pluginCountStr = "(? plugins)"
	}

	// GitHub stats
	var statsStr string
	if item.GitHubStats != nil {
		stats := item.GitHubStats
		starsStr := formatNumber(stats.Stars)
		forksStr := formatNumber(stats.Forks)
		lastUpdated := formatRelativeTime(stats.LastPushedAt)
		statsStr = fmt.Sprintf("⭐ %s  🍴 %s  🕒 %s",
			starsStr, forksStr, lastUpdated)
	} else if item.StatsLoading {
		statsStr = "Loading stats..."
	} else if item.StatsError != nil {
		statsStr = "Stats unavailable"
	}

	// Create styles for text
	tertiaryStyle := lipgloss.NewStyle().Foreground(TextTertiary)
	mutedStyle := lipgloss.NewStyle().Foreground(TextMuted)

	pluginCount := tertiaryStyle.Render(pluginCountStr)
	stats := mutedStyle.Render(statsStr)

	return fmt.Sprintf("%s%s %s  %s  %s",
		prefix, indicator, name, pluginCount, stats)
}

// renderMarketplaceSortTabs renders sort mode tabs
func (m Model) renderMarketplaceSortTabs() string {
	// Tab styles (inline like renderFilterTabs)
	activeTab := lipgloss.NewStyle().
		Foreground(PlumBright).
		Bold(true).
		Padding(0, 1)

	inactiveTab := lipgloss.NewStyle().
		Foreground(TextTertiary).
		Padding(0, 1)

	var b strings.Builder

	tabs := []struct {
		name   string
		active bool
	}{
		{MarketplaceSortModeNames[SortByPluginCount], m.marketplaceSortMode == SortByPluginCount},
		{MarketplaceSortModeNames[SortByStars], m.marketplaceSortMode == SortByStars},
		{MarketplaceSortModeNames[SortByName], m.marketplaceSortMode == SortByName},
		{MarketplaceSortModeNames[SortByLastUpdated], m.marketplaceSortMode == SortByLastUpdated},
	}

	for i, tab := range tabs {
		if i > 0 {
			b.WriteString("  ")
		}

		if tab.active {
			b.WriteString(activeTab.Render(tab.name))
		} else {
			b.WriteString(inactiveTab.Render(tab.name))
		}
	}

	hint := HelpStyle.Render("  (Tab/← → to change sort)")
	return b.String() + hint
}

// marketplaceStatusBar renders the status bar for marketplace view
func (m Model) marketplaceStatusBar() string {
	var parts []string

	total := len(m.marketplaceItems)
	installed := 0
	for _, item := range m.marketplaceItems {
		if item.Status == MarketplaceInstalled {
			installed++
		}
	}

	parts = append(parts, fmt.Sprintf("%d marketplaces", total))
	parts = append(parts, fmt.Sprintf("%d installed", installed))
	parts = append(parts, KeyStyle.Render("esc")+" return to plugins")
	parts = append(parts, KeyStyle.Render("?")+" help")

	return StatusBarStyle.Render(strings.Join(parts, "  │  "))
}

// marketplaceDetailView renders detailed view of a marketplace
func (m Model) marketplaceDetailView() string {
	item := m.selectedMarketplace
	if item == nil {
		return AppStyle.Render("No marketplace selected")
	}

	contentWidth := m.ContentWidth() - 10
	if contentWidth < 40 {
		contentWidth = 40
	}

	var b strings.Builder

	// Header with name and status badge
	badge := item.StatusBadge()
	header := DetailTitleStyle.Render(item.DisplayName) + "  " + badge
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", contentWidth))
	b.WriteString("\n\n")

	// Details
	details := []struct {
		label string
		value string
	}{
		{"Name", item.Name},
		{"Repository", item.Repo},
		{"Plugins", fmt.Sprintf("%d total", item.TotalPluginCount)},
	}

	if item.InstalledPluginCount > 0 {
		details = append(details, struct {
			label string
			value string
		}{"Your Installs", fmt.Sprintf("%d plugins", item.InstalledPluginCount)})
	}

	// GitHub stats section
	if item.GitHubStats != nil {
		stats := item.GitHubStats
		details = append(details,
			struct {
				label string
				value string
			}{"Stars", formatNumber(stats.Stars)},
			struct {
				label string
				value string
			}{"Forks", formatNumber(stats.Forks)},
			struct {
				label string
				value string
			}{"Last Updated", stats.LastPushedAt.Format("2006-01-02")},
			struct {
				label string
				value string
			}{"Open Issues", fmt.Sprintf("%d", stats.OpenIssues)},
		)
	} else if item.StatsLoading {
		details = append(details, struct {
			label string
			value string
		}{"GitHub Stats", "Loading..."})
	}

	for _, d := range details {
		if d.value != "" {
			b.WriteString(DetailLabelStyle.Render(d.label+":") + " " + DetailValueStyle.Render(d.value))
			b.WriteString("\n")
		}
	}

	// Description
	b.WriteString("\n")
	b.WriteString(wrapText(item.Description, contentWidth))
	b.WriteString("\n")

	// Actions section
	if item.Status != MarketplaceInstalled {
		b.WriteString("\n")
		b.WriteString(strings.Repeat("─", contentWidth))
		b.WriteString("\n")
		b.WriteString(DetailLabelStyle.Render("Install:"))
		b.WriteString("\n")
		installCmd := fmt.Sprintf("/plugin marketplace add %s", extractMarketplaceSource(item.Repo))
		b.WriteString("  " + InstallCommandStyle.Render(installCmd))
		b.WriteString("  " + HelpStyle.Render("press 'c' to copy"))
		b.WriteString("\n")
	}

	// Footer
	b.WriteString("\n")
	var footerParts []string
	footerParts = append(footerParts, KeyStyle.Render("esc")+" back")

	// Flash messages
	if m.copiedFlash {
		successStyle := lipgloss.NewStyle().Foreground(Success).Bold(true)
		footerParts = append(footerParts, successStyle.Render("✓ Copied!"))
	} else if m.githubOpenedFlash {
		openedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF9500")).Bold(true)
		footerParts = append(footerParts, openedStyle.Render("✓ Opened!"))
	} else {
		if item.Status != MarketplaceInstalled {
			footerParts = append(footerParts, KeyStyle.Render("c")+" copy install")
		}
		footerParts = append(footerParts, KeyStyle.Render("f")+" filter plugins")
		footerParts = append(footerParts, KeyStyle.Render("g")+" github")
	}

	footerParts = append(footerParts, KeyStyle.Render("q")+" quit")
	b.WriteString(HelpStyle.Render(strings.Join(footerParts, "  │  ")))

	boxStyle := DetailBoxStyle.Width(contentWidth + 4)
	return AppStyle.Render(boxStyle.Render(b.String()))
}

// formatRelativeTime formats time.Time to human-readable relative time
func formatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}

	duration := time.Since(t)

	switch {
	case duration < time.Hour:
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	case duration < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(duration.Hours()))
	case duration < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(duration.Hours()/24))
	case duration < 30*24*time.Hour:
		return fmt.Sprintf("%dw ago", int(duration.Hours()/24/7))
	case duration < 365*24*time.Hour:
		return fmt.Sprintf("%dmo ago", int(duration.Hours()/24/30))
	default:
		return fmt.Sprintf("%dy ago", int(duration.Hours()/24/365))
	}
}

// formatNumber formats large numbers with k/M suffix
func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	} else if n < 1000000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	} else {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
}

// extractMarketplaceSource extracts owner/repo from GitHub URL
func extractMarketplaceSource(repoURL string) string {
	// Remove https://github.com/ prefix
	repoURL = strings.TrimPrefix(repoURL, "https://github.com/")
	repoURL = strings.TrimPrefix(repoURL, "http://github.com/")
	repoURL = strings.TrimSuffix(repoURL, "/")
	repoURL = strings.TrimSuffix(repoURL, ".git")
	return repoURL
}
