package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/itsdevcoffee/plum/internal/marketplace"
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
	indicator := m.marketplaceIndicator(item.Status)
	prefix := m.selectionPrefix(selected)
	nameStyle := m.nameStyle(selected)
	name := nameStyle.Render(item.DisplayName)

	pluginCountStr := formatPluginCount(item.InstalledPluginCount, item.TotalPluginCount)
	statsStr := formatGitHubStats(item.GitHubStats, item.StatsLoading, item.StatsError)

	tertiaryStyle := lipgloss.NewStyle().Foreground(TextTertiary)
	mutedStyle := lipgloss.NewStyle().Foreground(TextMuted)

	return fmt.Sprintf("%s%s %s  %s  %s",
		prefix, indicator, name,
		tertiaryStyle.Render(pluginCountStr),
		mutedStyle.Render(statsStr))
}

func (m Model) marketplaceIndicator(status MarketplaceStatus) string {
	switch status {
	case MarketplaceInstalled:
		return InstalledIndicator.String()
	case MarketplaceCached:
		return "◆"
	case MarketplaceNew:
		return "★"
	default:
		return AvailableIndicator.String()
	}
}

func (m Model) selectionPrefix(selected bool) string {
	if selected {
		return HighlightBarFull.String()
	}
	return "  "
}

func (m Model) nameStyle(selected bool) lipgloss.Style {
	if selected {
		return PluginNameSelectedStyle
	}
	return PluginNameStyle
}

func formatPluginCount(installed, total int) string {
	if total > 0 {
		if installed > 0 {
			return fmt.Sprintf("(%d/%d plugins)", installed, total)
		}
		return fmt.Sprintf("(%d plugins)", total)
	}
	return "(? plugins)"
}

func formatGitHubStats(stats *marketplace.GitHubStats, loading bool, err error) string {
	if stats != nil {
		return fmt.Sprintf("⭐ %s  🍴 %s  🕒 %s",
			formatNumber(stats.Stars),
			formatNumber(stats.Forks),
			formatRelativeTime(stats.LastPushedAt))
	}
	if loading {
		return "Loading stats..."
	}
	if err != nil {
		return "Stats unavailable"
	}
	return ""
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

	hint := HelpStyle.Render("  (Tab/← → to change order)")
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
	hours := duration.Hours()

	if hours < 1 {
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	}
	if hours < 24 {
		return fmt.Sprintf("%dh ago", int(hours))
	}
	if hours < 168 {
		return fmt.Sprintf("%dd ago", int(hours/24))
	}
	if hours < 720 {
		return fmt.Sprintf("%dw ago", int(hours/24/7))
	}
	if hours < 8760 {
		return fmt.Sprintf("%dmo ago", int(hours/24/30))
	}
	return fmt.Sprintf("%dy ago", int(hours/24/365))
}

// formatNumber formats large numbers with k/M suffix
func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1000000)
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
