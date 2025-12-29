package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/itsdevcoffee/plum/internal/plugin"
)

// View renders the current view
func (m Model) View() string {
	if m.err != nil {
		return AppStyle.Render(fmt.Sprintf("Error loading plugins: %v\n\nPress q to quit.", m.err))
	}

	// Get the current view content
	var content string
	switch m.viewState {
	case ViewDetail:
		content = m.detailView()
	case ViewHelp:
		content = m.helpView()
	case ViewMarketplaceList:
		content = m.marketplaceListView()
	case ViewMarketplaceDetail:
		content = m.marketplaceDetailView()
	default:
		content = m.listView()
	}

	// Apply transition effect if animating (skip for instant)
	if m.IsViewTransitioning() && m.transitionStyle != TransitionInstant {
		switch m.transitionStyle {
		case TransitionZoom:
			content = m.applyZoomTransition(content)
		case TransitionSlideV:
			content = m.applySlideVTransition(content)
		}
	}

	return content
}

// applyZoomTransition creates a center-expand/contract effect
func (m Model) applyZoomTransition(content string) string {
	progress := m.transitionProgress
	if progress >= 1.0 {
		return content
	}
	if progress < 0 {
		progress = 0
	}

	lines := strings.Split(content, "\n")
	totalLines := len(lines)
	if totalLines == 0 {
		return content
	}

	// Calculate how many lines to show based on progress
	visibleLines := int(float64(totalLines) * progress)
	if visibleLines < 1 {
		visibleLines = 1
	}
	if visibleLines > totalLines {
		visibleLines = totalLines
	}

	// Calculate start/end to center the visible portion
	hiddenLines := totalLines - visibleLines
	startLine := hiddenLines / 2
	endLine := startLine + visibleLines

	// Build result with blank lines for hidden portions
	var result strings.Builder
	for i := 0; i < totalLines; i++ {
		if i > 0 {
			result.WriteString("\n")
		}
		if i >= startLine && i < endLine {
			result.WriteString(lines[i])
		}
	}

	return result.String()
}

// applySlideVTransition creates a vertical slide (push) effect
func (m Model) applySlideVTransition(content string) string {
	progress := m.transitionProgress
	if progress >= 1.0 {
		return content
	}
	if progress < 0 {
		progress = 0
	}

	lines := strings.Split(content, "\n")
	totalLines := len(lines)
	if totalLines == 0 {
		return content
	}

	// Calculate vertical offset based on progress and direction
	remaining := 1.0 - progress
	offsetLines := int(remaining * float64(totalLines))

	var result strings.Builder

	if m.transitionDirection > 0 {
		// Forward: slide up from bottom
		// Show blank lines at top, content slides up from bottom
		for i := 0; i < offsetLines; i++ {
			if i > 0 {
				result.WriteString("\n")
			}
			result.WriteString("")
		}
		for i := 0; i < totalLines-offsetLines; i++ {
			if i > 0 || offsetLines > 0 {
				result.WriteString("\n")
			}
			result.WriteString(lines[i])
		}
	} else {
		// Back: slide down from top
		// Content visible at top, blank lines fill from bottom
		for i := 0; i < totalLines-offsetLines; i++ {
			if i > 0 {
				result.WriteString("\n")
			}
			result.WriteString(lines[i+offsetLines])
		}
		for i := 0; i < offsetLines; i++ {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// renderFilterTabs renders the filter tab bar
func (m Model) renderFilterTabs() string {
	// Tab styles
	activeTab := lipgloss.NewStyle().
		Foreground(PlumBright).
		Bold(true).
		Padding(0, 1)

	inactiveTab := lipgloss.NewStyle().
		Foreground(TextTertiary).
		Padding(0, 1)

	// Build tabs with counts
	allCount := len(m.allPlugins)
	discoverCount := m.DiscoverableCount()
	readyCount := m.ReadyCount()
	installCount := m.InstalledCount()

	tabs := []struct {
		name   string
		count  int
		active bool
	}{
		{"All", allCount, m.filterMode == FilterAll},
		{"Discover", discoverCount, m.filterMode == FilterDiscover},
		{"Ready", readyCount, m.filterMode == FilterReady},
		{"Installed", installCount, m.filterMode == FilterInstalled},
	}

	var parts []string
	for _, tab := range tabs {
		label := fmt.Sprintf("%s (%d)", tab.name, tab.count)
		if tab.active {
			parts = append(parts, activeTab.Render(label))
		} else {
			parts = append(parts, inactiveTab.Render(label))
		}
	}

	return strings.Join(parts, DimSeparator.Render("│"))
}

// listView renders the main list view
func (m Model) listView() string {
	var b strings.Builder

	// Header - Title with optional inline notification
	title := "🍑 plum - Claude Plugin Manager"

	if m.newMarketplacesCount > 0 {
		plural := ""
		if m.newMarketplacesCount > 1 {
			plural = "s"
		}
		title = fmt.Sprintf("%s | ⚡ %d new marketplace%s - Shift+U", title, m.newMarketplacesCount, plural)
	}

	b.WriteString(TitleStyle.Render(title))
	b.WriteString("\n\n")

	// Search input
	b.WriteString(m.textInput.View())
	b.WriteString("\n")

	// Filter tabs
	b.WriteString(m.renderFilterTabs())
	b.WriteString("\n\n")

	// Results
	if m.loading {
		b.WriteString(m.spinner.View())
		b.WriteString(" ")
		b.WriteString(DescriptionStyle.Render("Loading plugins..."))
	} else if m.refreshing {
		b.WriteString(m.spinner.View())
		b.WriteString(" ")
		refreshStyle := lipgloss.NewStyle().Foreground(PeachSoft).Bold(true)
		if m.refreshTotal > 0 {
			progressText := fmt.Sprintf("Refreshing marketplaces (%d/%d)", m.refreshProgress, m.refreshTotal)
			if m.refreshCurrent != "" {
				progressText += fmt.Sprintf(" - %s", m.refreshCurrent)
			}
			b.WriteString(refreshStyle.Render(progressText))
		} else {
			b.WriteString(refreshStyle.Render("Refreshing marketplace data from GitHub..."))
		}
	} else if len(m.allPlugins) == 0 {
		b.WriteString(DescriptionStyle.Render("No plugins found."))
	} else if len(m.results) == 0 {
		b.WriteString(DescriptionStyle.Render("No plugins found matching your search."))
	} else {
		visible := m.VisibleResults()
		offset := m.ScrollOffset()

		for i, rp := range visible {
			actualIdx := offset + i
			isSelected := actualIdx == m.cursor
			b.WriteString(m.renderPluginItem(rp.Plugin, isSelected))
			b.WriteString("\n")
		}
	}

	// Status bar
	b.WriteString("\n")
	b.WriteString(m.statusBar())

	return AppStyle.Render(b.String())
}

// renderPluginItem renders a single plugin item based on display mode
func (m Model) renderPluginItem(p plugin.Plugin, selected bool) string {
	if m.displayMode == DisplaySlim {
		return m.renderPluginItemSlim(p, selected)
	}
	return m.renderPluginItemCard(p, selected)
}

// renderPluginItemSlim renders a compact one-line plugin item
func (m Model) renderPluginItemSlim(p plugin.Plugin, selected bool) string {
	// Indicator
	var indicator string
	if p.Installed {
		indicator = InstalledIndicator.String()
	} else {
		indicator = AvailableIndicator.String()
		// Add [Discover] badge for plugins from uninstalled marketplaces
		if p.IsDiscoverable {
			indicator += " " + DiscoverBadge.String()
		}
	}

	// Name style based on selection
	var nameStyle lipgloss.Style
	if selected {
		nameStyle = PluginNameSelectedStyle
	} else {
		nameStyle = PluginNameStyle
	}

	// Selection prefix
	var prefix string
	if selected {
		prefix = HighlightBarFull.String()
	} else {
		prefix = "  "
	}

	// Format: [prefix][indicator] name v[version]
	name := nameStyle.Render(p.Name)
	version := VersionStyle.Render("v" + p.Version)

	return fmt.Sprintf("%s%s %s %s", prefix, indicator, name, version)
}

// renderPluginItemCard renders a plugin item as a card with border
func (m Model) renderPluginItemCard(p plugin.Plugin, selected bool) string {
	// Card width (account for app padding and card border)
	cardWidth := m.ContentWidth() - 6
	if cardWidth < 40 {
		cardWidth = 40
	}
	innerWidth := cardWidth - 4 // Account for card padding and border

	// Indicator
	var indicator string
	if p.Installed {
		indicator = InstalledIndicator.String()
	} else {
		indicator = AvailableIndicator.String()
		// Add [Discover] badge for plugins from uninstalled marketplaces
		if p.IsDiscoverable {
			indicator += " " + DiscoverBadge.String()
		}
	}

	// Name style based on selection
	var nameStyle lipgloss.Style
	if selected {
		nameStyle = PluginNameSelectedStyle
	} else {
		nameStyle = PluginNameStyle
	}

	// Row 1: [indicator] Name v[version]                    @marketplace
	name := nameStyle.Render(p.Name)
	version := VersionStyle.Render("v" + p.Version)
	marketplace := MarketplaceStyle.Render("@" + p.Marketplace)

	leftPart := fmt.Sprintf("%s %s %s", indicator, name, version)
	leftLen := lipgloss.Width(leftPart)
	rightLen := lipgloss.Width(marketplace)

	// Calculate spacing for right-aligned marketplace
	spacerLen := innerWidth - leftLen - rightLen
	if spacerLen < 1 {
		spacerLen = 1
	}
	row1 := leftPart + strings.Repeat(" ", spacerLen) + marketplace

	// Row 2: Description (truncated to fit)
	maxDescLen := innerWidth - 2
	if maxDescLen < 20 {
		maxDescLen = 20
	}
	truncDesc := p.Description
	if len(truncDesc) > maxDescLen {
		truncDesc = truncDesc[:maxDescLen-3] + "..."
	}
	row2 := "  " + DescriptionStyle.Render(truncDesc)

	// Combine rows (2 rows now)
	content := row1 + "\n" + row2

	// Apply card style
	var cardStyle lipgloss.Style
	if selected {
		cardStyle = PluginCardSelectedStyle.Width(cardWidth)
	} else {
		cardStyle = PluginCardStyle.Width(cardWidth)
	}

	return cardStyle.Render(content)
}

// statusBar renders the status bar (responsive to terminal width)
func (m Model) statusBar() string {
	var parts []string

	// Position in current filtered results
	var position string
	if len(m.results) > 0 {
		position = fmt.Sprintf("%d/%d", m.cursor+1, len(m.results))
	} else {
		position = "0/0"
	}

	// Opposite view mode name for the toggle hint
	var oppositeView string
	if m.displayMode == DisplaySlim {
		oppositeView = "verbose"
	} else {
		oppositeView = "slim"
	}

	width := m.ContentWidth()

	// In slim mode, skip the verbose breakpoint (use standard instead)
	useVerbose := width >= 100 && m.displayMode == DisplayCard

	switch {
	case useVerbose:
		// Verbose: full descriptions (only in card/verbose mode)
		parts = append(parts, position+" "+m.FilterModeName())
		parts = append(parts, KeyStyle.Render("↑↓/ctrl+jk")+" navigate")
		parts = append(parts, KeyStyle.Render("tab")+" filter")
		parts = append(parts, KeyStyle.Render("Shift+V")+" "+oppositeView)
		parts = append(parts, KeyStyle.Render("enter")+" details")
		parts = append(parts, KeyStyle.Render("?"))

	case width >= 70:
		// Standard: concise but complete
		parts = append(parts, position)
		parts = append(parts, KeyStyle.Render("↑↓")+" nav")
		parts = append(parts, KeyStyle.Render("tab")+" filter")
		parts = append(parts, KeyStyle.Render("Shift+M")+" marketplaces")
		parts = append(parts, KeyStyle.Render("Shift+V")+" "+oppositeView)
		parts = append(parts, KeyStyle.Render("?")+" help")

	case width >= 50:
		// Compact: essentials only
		parts = append(parts, position)
		parts = append(parts, KeyStyle.Render("↑↓")+" nav")
		parts = append(parts, KeyStyle.Render("tab")+" filter")
		parts = append(parts, KeyStyle.Render("?")+" help")

	default:
		// Minimal: bare minimum
		parts = append(parts, position)
		parts = append(parts, KeyStyle.Render("?")+"=help")
	}

	return StatusBarStyle.Render(strings.Join(parts, "  │  "))
}

// detailView renders the detail view for the selected plugin
func (m Model) detailView() string {
	p := m.SelectedPlugin()
	if p == nil {
		return AppStyle.Render("No plugin selected")
	}

	// Calculate content width (account for borders and padding)
	contentWidth := m.ContentWidth() - 10
	if contentWidth < 40 {
		contentWidth = 40
	}

	var b strings.Builder

	// Header with name and status badge
	var badge string
	if p.Installed {
		badge = InstalledBadge.String()
	} else {
		badge = AvailableBadge.String()
	}
	header := DetailTitleStyle.Render(p.Name) + "  " + badge
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", contentWidth))
	b.WriteString("\n\n")

	// Details
	details := []struct {
		label string
		value string
	}{
		{"Version", p.Version},
		{"Author", p.AuthorName()},
		{"Marketplace", p.Marketplace},
		{"Category", p.Category},
	}

	for _, d := range details {
		if d.value != "" {
			b.WriteString(DetailLabelStyle.Render(d.label+":") + " " + DetailValueStyle.Render(d.value))
			b.WriteString("\n")
		}
	}

	// Install path (only for installed plugins)
	if p.Installed && p.InstallPath != "" {
		b.WriteString(DetailLabelStyle.Render("Install Path:") + " " + DetailValueStyle.Render(p.InstallPath))
		b.WriteString("\n")
		b.WriteString(HelpStyle.Render("              Press 'o' to open in file manager"))
		b.WriteString("\n")
	}

	// Description (word-wrapped)
	b.WriteString("\n")
	b.WriteString(wrapText(p.Description, contentWidth))
	b.WriteString("\n")

	// Keywords (word-wrapped)
	if len(p.Keywords) > 0 {
		b.WriteString("\n")
		keywordsText := strings.Join(p.Keywords, ", ")
		b.WriteString(DetailLabelStyle.Render("Keywords:") + " ")
		b.WriteString(wrapText(keywordsText, contentWidth-12))
		b.WriteString("\n")
	}

	// Install command (only for non-installed plugins)
	if !p.Installed {
		b.WriteString("\n")
		b.WriteString(strings.Repeat("─", contentWidth))
		b.WriteString("\n")

		if p.IsDiscoverable {
			// Marketplace not installed - show 2-step instructions
			b.WriteString(DiscoverMessageStyle.Render("⚠ This marketplace is not installed yet"))
			b.WriteString("\n\n")

			b.WriteString(DetailLabelStyle.Render("Step 1:") + " Install the marketplace")
			b.WriteString("\n")
			installMarketplace := fmt.Sprintf("/plugin marketplace add %s", p.MarketplaceSource)
			b.WriteString("  " + InstallCommandStyle.Render(installMarketplace))
			b.WriteString("  " + HelpStyle.Render("press 'c' to copy"))
			b.WriteString("\n\n")

			b.WriteString(DetailLabelStyle.Render("Step 2:") + " Install the plugin")
			b.WriteString("\n")
			b.WriteString("  " + InstallCommandStyle.Render(p.InstallCommand()))
			b.WriteString("  " + HelpStyle.Render("press 'y' to copy"))
			b.WriteString("\n")
		} else {
			// Marketplace installed - show normal install command
			b.WriteString(DetailLabelStyle.Render("Install:") + " " + InstallCommandStyle.Render(p.InstallCommand()))
			b.WriteString("\n")
		}
	}

	// Footer - build with flash message replacements
	b.WriteString("\n")
	var footerParts []string

	// Define styles for flash messages
	successStyle := lipgloss.NewStyle().Foreground(Success).Bold(true)
	openedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF9500")).Bold(true)
	errorStyle := lipgloss.NewStyle().Foreground(Error).Bold(true)

	// Always show esc
	footerParts = append(footerParts, KeyStyle.Render("esc")+" back")

	// Show install commands for non-installed plugins (or flash message)
	if !p.Installed {
		if m.copiedFlash {
			footerParts = append(footerParts, successStyle.Render("✓ Copied!"))
		} else if m.clipboardErrorFlash {
			footerParts = append(footerParts, errorStyle.Render("✗ Clipboard error"))
		} else {
			if p.IsDiscoverable {
				footerParts = append(footerParts, KeyStyle.Render("c")+" copy marketplace")
				footerParts = append(footerParts, KeyStyle.Render("y")+" copy plugin")
			} else {
				footerParts = append(footerParts, KeyStyle.Render("c")+" copy install command")
			}
		}
	}

	// GitHub link (with flash replacement)
	if m.githubOpenedFlash {
		footerParts = append(footerParts, openedStyle.Render("✓ Opened!"))
	} else {
		footerParts = append(footerParts, KeyStyle.Render("g")+" github")
	}

	// Copy link (with flash replacement)
	if m.linkCopiedFlash {
		footerParts = append(footerParts, successStyle.Render("✓ Link Copied!"))
	} else if m.clipboardErrorFlash && !m.copiedFlash {
		footerParts = append(footerParts, errorStyle.Render("✗ Clipboard error"))
	} else {
		footerParts = append(footerParts, KeyStyle.Render("l")+" copy link")
	}

	// Local directory actions (only for installed)
	if p.Installed && p.InstallPath != "" {
		// Open local (with flash replacement)
		if m.localOpenedFlash {
			footerParts = append(footerParts, openedStyle.Render("✓ Opened!"))
		} else {
			footerParts = append(footerParts, KeyStyle.Render("o")+" open local")
		}

		// Copy path (with flash replacement)
		if m.pathCopiedFlash {
			footerParts = append(footerParts, successStyle.Render("✓ Path Copied!"))
		} else if m.clipboardErrorFlash && !m.copiedFlash && !m.linkCopiedFlash {
			footerParts = append(footerParts, errorStyle.Render("✗ Clipboard error"))
		} else {
			footerParts = append(footerParts, KeyStyle.Render("p")+" copy path")
		}
	}

	// Always show quit
	footerParts = append(footerParts, KeyStyle.Render("q")+" quit")
	b.WriteString(HelpStyle.Render(strings.Join(footerParts, "  │  ")))

	// Apply box style with full width
	boxStyle := DetailBoxStyle.Width(contentWidth + 4)
	return AppStyle.Render(boxStyle.Render(b.String()))
}

// wrapText wraps text to fit within maxWidth characters
func wrapText(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return text
	}

	var result strings.Builder
	words := strings.Fields(text)
	lineLen := 0

	for i, word := range words {
		wordLen := len(word)

		if lineLen+wordLen+1 > maxWidth && lineLen > 0 {
			result.WriteString("\n")
			lineLen = 0
		}

		if lineLen > 0 {
			result.WriteString(" ")
			lineLen++
		}

		// Handle words longer than maxWidth
		if wordLen > maxWidth {
			for len(word) > maxWidth {
				if lineLen > 0 {
					result.WriteString("\n")
					lineLen = 0
				}
				result.WriteString(word[:maxWidth])
				word = word[maxWidth:]
				result.WriteString("\n")
			}
			if len(word) > 0 {
				result.WriteString(word)
				lineLen = len(word)
			}
		} else {
			result.WriteString(word)
			lineLen += wordLen
		}

		_ = i // suppress unused warning
	}

	return result.String()
}

// generateHelpContent generates the help menu content without box wrapper
func (m Model) generateHelpContent() string {
	var b strings.Builder

	// Context hint style for detail-view-only commands
	contextStyle := lipgloss.NewStyle().Foreground(TextMuted).Italic(true)
	installedOnlyStyle := lipgloss.NewStyle().Foreground(Success)
	dividerStyle := lipgloss.NewStyle().Foreground(BorderSubtle)

	// Header with legend on the right using JoinHorizontal
	contentWidth := 56

	title := DetailTitleStyle.Render("🍑 plum Help")

	legendText := installedOnlyStyle.Render("🟢") + " = installed only"
	legendStyle := lipgloss.NewStyle().
		Foreground(TextMuted).
		Align(lipgloss.Right).
		Width(contentWidth - lipgloss.Width(title))
	legend := legendStyle.Render(legendText)

	// Join horizontally along top edge
	headerLine := lipgloss.JoinHorizontal(lipgloss.Top, title, legend)

	b.WriteString(headerLine)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", contentWidth))
	b.WriteString("\n\n")

	// Navigation section
	b.WriteString(HelpSectionStyle.Render("  🧭 Navigation"))
	b.WriteString("\n")
	navKeys := []struct{ key, desc string }{
		{"↑ Ctrl+k/p", "Move up"},
		{"↓ Ctrl+j/n", "Move down"},
		{"Ctrl+u PgUp", "Page up"},
		{"Ctrl+d PgDn", "Page down"},
		{"Home / End", "Jump to edges"},
	}
	for _, h := range navKeys {
		b.WriteString(fmt.Sprintf("    %s  %s\n", KeyStyle.Width(16).Render(h.key), HelpTextStyle.Render(h.desc)))
	}
	b.WriteString(dividerStyle.Render("  " + strings.Repeat("─", 56)))
	b.WriteString("\n")

	// Views & Browsing section
	b.WriteString(HelpSectionStyle.Render("  👁️  Views & Browsing"))
	b.WriteString("\n")
	viewKeys := []struct{ key, desc string }{
		{"Enter", "View details"},
		{"Shift+M", "Marketplace browser"},
		{"?", "Toggle help"},
	}
	for _, h := range viewKeys {
		b.WriteString(fmt.Sprintf("    %s  %s\n", KeyStyle.Width(16).Render(h.key), HelpTextStyle.Render(h.desc)))
	}
	b.WriteString(dividerStyle.Render("  " + strings.Repeat("─", 56)))
	b.WriteString("\n")

	// Plugin Actions section (detail view)
	b.WriteString(HelpSectionStyle.Render("  📦 Plugin Actions ") + contextStyle.Render("(detail view)"))
	b.WriteString("\n")
	pluginKeys := []struct{ key, desc, suffix string }{
		{"c", "Copy install command", ""},
		{"g", "Open on GitHub", ""},
		{"o", "Open local directory", " 🟢"},
		{"p", "Copy local path", " 🟢"},
		{"l", "Copy GitHub link", ""},
		{"f", "Filter by marketplace", ""},
	}
	for _, h := range pluginKeys {
		desc := HelpTextStyle.Render(h.desc)
		if h.suffix != "" {
			desc += installedOnlyStyle.Render(h.suffix)
		}
		b.WriteString(fmt.Sprintf("    %s  %s\n", KeyStyle.Width(16).Render(h.key), desc))
	}
	b.WriteString(dividerStyle.Render("  " + strings.Repeat("─", 56)))
	b.WriteString("\n")

	// Display & Filters section
	b.WriteString(HelpSectionStyle.Render("  🎨 Display & Filters"))
	b.WriteString("\n")
	displayKeys := []struct{ key, desc string }{
		{"Tab →", "Next filter"},
		{"Shift+Tab ←", "Previous filter"},
		{"Shift+V", "Toggle view mode"},
	}
	for _, h := range displayKeys {
		b.WriteString(fmt.Sprintf("    %s  %s\n", KeyStyle.Width(16).Render(h.key), HelpTextStyle.Render(h.desc)))
	}
	b.WriteString(dividerStyle.Render("  " + strings.Repeat("─", 56)))
	b.WriteString("\n")

	// System section
	b.WriteString(HelpSectionStyle.Render("  ⚙️  System"))
	b.WriteString("\n")
	systemKeys := []struct{ key, desc string }{
		{"Shift+U", "Refresh marketplaces"},
		{"Esc", "Back / Clear / Cancel"},
		{"Ctrl+c / q", "Quit"},
	}
	for _, h := range systemKeys {
		b.WriteString(fmt.Sprintf("    %s  %s\n", KeyStyle.Width(16).Render(h.key), HelpTextStyle.Render(h.desc)))
	}

	b.WriteString(strings.Repeat("─", 56))
	b.WriteString("\n")
	b.WriteString(HelpTextStyle.Render("  Press any key to return  (↑↓ to scroll)"))

	return b.String()
}

// helpView renders the help view with viewport for scrolling
func (m Model) helpView() string {
	// Use viewport if initialized (content set on view enter)
	if m.helpViewport.Height > 0 {
		// Wrap viewport in box with max width
		helpBoxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(PlumBright).
			Padding(0, 2).
			MaxWidth(80)

		return AppStyle.Render(helpBoxStyle.Render(m.helpViewport.View()))
	}

	// Fallback: render without viewport
	helpContent := m.generateHelpContent()
	helpBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PlumBright).
		Padding(0, 2).
		MaxWidth(80)

	return AppStyle.Render(helpBoxStyle.Render(helpContent))
}
