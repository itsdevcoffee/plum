package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// helpView renders the help view with sticky header/footer
func (m Model) helpView() string {
	helpWrapperStyle := lipgloss.NewStyle().Padding(0, 2, 0, 2)
	helpBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PlumBright).
		Padding(1, 2)

	header := m.generateHelpHeader()
	footer := m.generateHelpFooter()

	if m.helpViewport.Height > 0 {
		viewportContent := m.helpViewport.View()
		scrollbar := m.renderHelpScrollbar()
		contentWithScrollbar := lipgloss.JoinHorizontal(lipgloss.Top, viewportContent, scrollbar)

		fullContent := lipgloss.JoinVertical(lipgloss.Left, header, contentWithScrollbar, footer)
		return helpWrapperStyle.Render(helpBoxStyle.Render(fullContent))
	}

	// Fallback when viewport not initialized
	var fullContent strings.Builder
	fullContent.WriteString(header)
	fullContent.WriteString("\n")
	fullContent.WriteString(m.generateHelpSections())
	fullContent.WriteString("\n")
	fullContent.WriteString(footer)

	return helpWrapperStyle.Render(helpBoxStyle.Render(fullContent.String()))
}

// generateHelpHeader generates the sticky header
func (m Model) generateHelpHeader() string {
	const contentWidth = 58

	title := DetailTitleStyle.Render("🍑 plum Help")

	installedOnlyStyle := lipgloss.NewStyle().Foreground(Success)
	legendText := installedOnlyStyle.Render("🟢") + " = installed only"
	legendStyle := lipgloss.NewStyle().
		Foreground(TextMuted).
		Align(lipgloss.Right).
		Width(contentWidth - lipgloss.Width(title))
	legend := legendStyle.Render(legendText)

	headerLine := lipgloss.JoinHorizontal(lipgloss.Top, title, legend)

	var b strings.Builder
	b.WriteString(headerLine)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", contentWidth))
	return b.String()
}

// generateHelpFooter generates the sticky footer
func (m Model) generateHelpFooter() string {
	var b strings.Builder
	b.WriteString(strings.Repeat("─", 58))
	b.WriteString("\n")
	b.WriteString(HelpTextStyle.Render("  Press any key to return  (↑↓ to scroll)"))
	return b.String()
}

// generateHelpSections generates only the scrollable sections (no header/footer)
func (m Model) generateHelpSections() string {
	var b strings.Builder

	contextStyle := lipgloss.NewStyle().Foreground(TextMuted).Italic(true)
	installedOnlyStyle := lipgloss.NewStyle().Foreground(Success)
	dividerStyle := lipgloss.NewStyle().Foreground(BorderSubtle)

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
	viewKeys := []struct{ key, desc, context string }{
		{"Enter", "View details", "(plugin/marketplace list)"},
		{"Shift+M", "Marketplace browser", "(any view)"},
		{"Space", "Quick action menu", "(most views)"},
		{"?", "Toggle help", "(any view)"},
	}
	for _, h := range viewKeys {
		desc := HelpTextStyle.Render(h.desc)
		if h.context != "" {
			desc += " " + contextStyle.Render(h.context)
		}
		b.WriteString(fmt.Sprintf("    %s  %s\n", KeyStyle.Width(16).Render(h.key), desc))
	}
	b.WriteString(dividerStyle.Render("  " + strings.Repeat("─", 56)))
	b.WriteString("\n")

	// Plugin Actions section
	b.WriteString(HelpSectionStyle.Render("  📦 Plugin Actions ") + contextStyle.Render("(plugin detail view)"))
	b.WriteString("\n")
	pluginKeys := []struct{ key, desc, suffix string }{
		{"c", "Copy install command", ""},
		{"i", "Copy 2-step install", " (discover only)"},
		{"y", "Copy plugin install", " (discover only)"},
		{"g", "Open on GitHub", ""},
		{"o", "Open local directory", " 🟢"},
		{"p", "Copy local path", " 🟢"},
		{"l", "Copy GitHub link", ""},
	}
	for _, h := range pluginKeys {
		desc := HelpTextStyle.Render(h.desc)
		if h.suffix != "" {
			if strings.Contains(h.suffix, "🟢") {
				desc += installedOnlyStyle.Render(h.suffix)
			} else {
				desc += contextStyle.Render(h.suffix)
			}
		}
		b.WriteString(fmt.Sprintf("    %s  %s\n", KeyStyle.Width(16).Render(h.key), desc))
	}
	b.WriteString(dividerStyle.Render("  " + strings.Repeat("─", 56)))
	b.WriteString("\n")

	// Marketplace Actions section
	b.WriteString(HelpSectionStyle.Render("  🏪 Marketplace Actions ") + contextStyle.Render("(marketplace detail)"))
	b.WriteString("\n")
	marketplaceKeys := []struct{ key, desc string }{
		{"c", "Copy marketplace install command"},
		{"f", "Filter plugins by this marketplace"},
		{"g", "Open on GitHub"},
		{"l", "Copy GitHub link"},
	}
	for _, h := range marketplaceKeys {
		b.WriteString(fmt.Sprintf("    %s  %s\n", KeyStyle.Width(16).Render(h.key), HelpTextStyle.Render(h.desc)))
	}
	b.WriteString(dividerStyle.Render("  " + strings.Repeat("─", 56)))
	b.WriteString("\n")

	// Display & Filters section
	b.WriteString(HelpSectionStyle.Render("  🎨 Display & Facets ") + contextStyle.Render("(plugin list)"))
	b.WriteString("\n")
	displayKeys := []struct{ key, desc string }{
		{"Tab →", "Next facet (filters + sorts)"},
		{"Shift+Tab ←", "Previous facet"},
		{"Shift+V", "Toggle display mode (card/slim)"},
		{"Shift+F", "Marketplace picker"},
		{"@marketplace", "Filter by marketplace (in search)"},
	}
	for _, h := range displayKeys {
		b.WriteString(fmt.Sprintf("    %s  %s\n", KeyStyle.Width(16).Render(h.key), HelpTextStyle.Render(h.desc)))
	}
	b.WriteString(dividerStyle.Render("  " + strings.Repeat("─", 56)))
	b.WriteString("\n")

	// Marketplace Facets section
	b.WriteString(HelpSectionStyle.Render("  🔄 Marketplace Facets ") + contextStyle.Render("(marketplace list)"))
	b.WriteString("\n")
	sortKeys := []struct{ key, desc string }{
		{"Tab →", "Next facet (sort orders)"},
		{"Shift+Tab ←", "Previous facet"},
	}
	for _, h := range sortKeys {
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

	return b.String()
}

// renderHelpScrollbar renders a plum-themed scrollbar for the help viewport
func (m Model) renderHelpScrollbar() string {
	if m.helpViewport.Height <= 0 || (m.helpViewport.AtTop() && m.helpViewport.AtBottom()) {
		return ""
	}

	visibleHeight := m.helpViewport.Height
	scrollPercent := m.helpViewport.ScrollPercent()
	totalHeight := visibleHeight * 2

	thumbHeight := (visibleHeight * visibleHeight) / totalHeight
	if thumbHeight < 1 {
		thumbHeight = 1
	}
	if thumbHeight > visibleHeight {
		thumbHeight = visibleHeight
	}

	trackHeight := visibleHeight - thumbHeight
	thumbPos := int(float64(trackHeight) * scrollPercent)

	thumbStyle := lipgloss.NewStyle().Foreground(PlumBright)
	trackStyle := lipgloss.NewStyle().Foreground(BorderSubtle)

	var scrollbar strings.Builder
	for i := 0; i < visibleHeight; i++ {
		if i >= thumbPos && i < thumbPos+thumbHeight {
			scrollbar.WriteString(thumbStyle.Render("█"))
		} else {
			scrollbar.WriteString(trackStyle.Render("░"))
		}
		if i < visibleHeight-1 {
			scrollbar.WriteString("\n")
		}
	}

	return " " + scrollbar.String()
}
