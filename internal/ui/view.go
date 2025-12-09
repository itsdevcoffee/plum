package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/maskkiller/plum/internal/plugin"
)

// View renders the current view
func (m Model) View() string {
	if m.err != nil {
		return AppStyle.Render(fmt.Sprintf("Error loading plugins: %v\n\nPress q to quit.", m.err))
	}

	switch m.viewState {
	case ViewDetail:
		return m.detailView()
	case ViewHelp:
		return m.helpView()
	default:
		return m.listView()
	}
}

// listView renders the main list view
func (m Model) listView() string {
	var b strings.Builder

	// Title
	b.WriteString(TitleStyle.Render("🍑 plum - Plugin Search"))
	b.WriteString("\n")

	// Search input
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")

	// Results
	if m.loading {
		b.WriteString(m.spinner.View())
		b.WriteString(" ")
		b.WriteString(DescriptionStyle.Render("Loading plugins..."))
	} else if len(m.allPlugins) == 0 {
		b.WriteString(DescriptionStyle.Render("No plugins found."))
	} else if len(m.results) == 0 {
		b.WriteString(DescriptionStyle.Render("No plugins found matching your search."))
	} else {
		visible := m.VisibleResults()
		offset := m.ScrollOffset()

		// Calculate animated cursor visual position
		animatedVisualPos := m.cursorY

		for i, rp := range visible {
			actualIdx := offset + i
			isSelected := actualIdx == m.cursor

			// Calculate how "selected" this item appears based on animation
			// This creates a smooth sliding highlight effect
			distFromAnimatedCursor := math.Abs(animatedVisualPos - float64(i))
			var highlightAmount float64
			if distFromAnimatedCursor < 1.0 {
				highlightAmount = 1.0 - distFromAnimatedCursor
			}

			b.WriteString(m.renderPluginItemWithHighlight(rp.Plugin, isSelected, highlightAmount))
			b.WriteString("\n")
		}
	}

	// Status bar
	b.WriteString("\n")
	b.WriteString(m.statusBar())

	return AppStyle.Render(b.String())
}

// renderPluginItem renders a single plugin item using the plugin package type
func (m Model) renderPluginItem(p plugin.Plugin, selected bool) string {
	return m.renderPluginItemWithHighlight(p, selected, 0)
}

// renderPluginItemWithHighlight renders a plugin with animated highlight
func (m Model) renderPluginItemWithHighlight(p plugin.Plugin, selected bool, highlightAmount float64) string {
	var b strings.Builder

	// Indicator
	var indicator string
	if p.Installed {
		indicator = InstalledIndicator.String()
	} else {
		indicator = AvailableIndicator.String()
	}

	// Create sliding highlight bar based on animation
	// highlightAmount: 0 = no highlight, 1 = full highlight
	var prefix string
	if highlightAmount > 0.8 {
		prefix = HighlightBarFull.String()
	} else if highlightAmount > 0.5 {
		prefix = HighlightBarMedium.String()
	} else if highlightAmount > 0.2 {
		prefix = HighlightBarLight.String()
	} else {
		prefix = "  " // Empty space for alignment
	}

	// First line: prefix + indicator + name@marketplace + version
	var nameStyle lipgloss.Style
	if selected {
		nameStyle = PluginNameSelectedStyle
	} else if highlightAmount > 0.3 {
		// Partial highlight during animation
		nameStyle = PluginNameStyle.Foreground(LightPurple)
	} else {
		nameStyle = PluginNameStyle
	}

	fullName := nameStyle.Render(p.Name) + MarketplaceStyle.Render("@"+p.Marketplace)
	versionTag := VersionStyle.Render("[" + p.Version + "]")

	line1 := fmt.Sprintf("%s%s %s %s", prefix, indicator, fullName, versionTag)

	// Second line: description (truncated)
	maxDescLen := m.windowWidth - 12
	if maxDescLen < 20 {
		maxDescLen = 20
	}
	truncDesc := p.Description
	if len(truncDesc) > maxDescLen {
		truncDesc = truncDesc[:maxDescLen-3] + "..."
	}
	line2 := "    " + DescriptionStyle.Render(truncDesc)

	// Apply background based on highlight amount
	if selected || highlightAmount > 0.5 {
		// Calculate background brightness based on highlight
		brightness := int(highlightAmount * 25)
		bgColor := fmt.Sprintf("#%02x%02x%02x", 35+brightness, 35+brightness, 45+brightness)
		itemStyle := lipgloss.NewStyle().Background(lipgloss.Color(bgColor)).Padding(0, 1)
		b.WriteString(itemStyle.Render(line1))
		b.WriteString("\n")
		b.WriteString(itemStyle.Render(line2))
	} else {
		b.WriteString(NormalItemStyle.Render(line1))
		b.WriteString("\n")
		b.WriteString(NormalItemStyle.Render(line2))
	}

	return b.String()
}

// statusBar renders the status bar
func (m Model) statusBar() string {
	var parts []string

	// Legend
	parts = append(parts, InstalledIndicator.String()+" = installed")

	// Result count
	if m.textInput.Value() != "" {
		parts = append(parts, fmt.Sprintf("%d results", len(m.results)))
	} else {
		parts = append(parts, fmt.Sprintf("%d plugins (%d installed)", m.TotalPlugins(), m.InstalledCount()))
	}

	// Navigation hint
	parts = append(parts, "↑↓ navigate  enter details  ? help  q quit")

	return StatusBarStyle.Render(strings.Join(parts, "  │  "))
}

// detailView renders the detail view for the selected plugin
func (m Model) detailView() string {
	p := m.SelectedPlugin()
	if p == nil {
		return AppStyle.Render("No plugin selected")
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
	b.WriteString(strings.Repeat("─", min(m.windowWidth-6, 50)))
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

	// Description
	b.WriteString("\n")
	b.WriteString(DetailDescStyle.Render(p.Description))
	b.WriteString("\n")

	// Keywords
	if len(p.Keywords) > 0 {
		b.WriteString("\n")
		b.WriteString(DetailLabelStyle.Render("Keywords:") + " " + DetailValueStyle.Render(strings.Join(p.Keywords, ", ")))
		b.WriteString("\n")
	}

	// Install command (only for non-installed plugins)
	if !p.Installed {
		b.WriteString("\n")
		b.WriteString(strings.Repeat("─", min(m.windowWidth-6, 50)))
		b.WriteString("\n")
		b.WriteString(DetailLabelStyle.Render("Install:") + " " + InstallCommandStyle.Render(p.InstallCommand()))
		b.WriteString("\n")
	}

	// Footer
	b.WriteString("\n")
	var footerParts []string
	footerParts = append(footerParts, KeyStyle.Render("esc")+" back")
	if !p.Installed {
		footerParts = append(footerParts, KeyStyle.Render("c")+" copy install command")
	}
	footerParts = append(footerParts, KeyStyle.Render("q")+" quit")
	b.WriteString(HelpStyle.Render(strings.Join(footerParts, "  │  ")))

	return AppStyle.Render(DetailBoxStyle.Render(b.String()))
}

// helpView renders the help view with grouped sections
func (m Model) helpView() string {
	var b strings.Builder

	b.WriteString(DetailTitleStyle.Render("🍑 plum Help"))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 44))
	b.WriteString("\n\n")

	// Navigation section
	b.WriteString(HelpSectionStyle.Render("  ◆ Navigation"))
	b.WriteString("\n")
	navKeys := []struct{ key, desc string }{
		{"↑ k", "Move up"},
		{"↓ j", "Move down"},
		{"g Home", "Jump to top"},
		{"G End", "Jump to bottom"},
		{"Ctrl+u PgUp", "Page up"},
		{"Ctrl+d PgDn", "Page down"},
	}
	for _, h := range navKeys {
		b.WriteString(fmt.Sprintf("    %s  %s\n", KeyStyle.Width(12).Render(h.key), HelpTextStyle.Render(h.desc)))
	}

	b.WriteString("\n")

	// Actions section
	b.WriteString(HelpSectionStyle.Render("  ◆ Actions"))
	b.WriteString("\n")
	actionKeys := []struct{ key, desc string }{
		{"Enter", "View plugin details"},
		{"c", "Copy install command"},
		{"Esc", "Clear search / Go back"},
	}
	for _, h := range actionKeys {
		b.WriteString(fmt.Sprintf("    %s  %s\n", KeyStyle.Width(12).Render(h.key), HelpTextStyle.Render(h.desc)))
	}

	b.WriteString("\n")

	// General section
	b.WriteString(HelpSectionStyle.Render("  ◆ General"))
	b.WriteString("\n")
	generalKeys := []struct{ key, desc string }{
		{"?", "Toggle this help"},
		{"q Ctrl+c", "Quit plum"},
	}
	for _, h := range generalKeys {
		b.WriteString(fmt.Sprintf("    %s  %s\n", KeyStyle.Width(12).Render(h.key), HelpTextStyle.Render(h.desc)))
	}

	b.WriteString("\n")

	// Tips section
	b.WriteString(HelpSectionStyle.Render("  ◆ Tips"))
	b.WriteString("\n")
	b.WriteString(HelpTextStyle.Render("    • Just start typing to search\n"))
	b.WriteString(HelpTextStyle.Render("    • Vim-style navigation (hjkl)\n"))
	b.WriteString(HelpTextStyle.Render("    • Green ● = installed plugin\n"))

	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 44))
	b.WriteString("\n")
	b.WriteString(HelpTextStyle.Render("  Press any key to return"))

	return AppStyle.Render(DetailBoxStyle.Render(b.String()))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
