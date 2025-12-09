package ui

import (
	"fmt"
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

// renderPluginItem renders a single plugin item
func (m Model) renderPluginItem(p plugin.Plugin, selected bool) string {
	var b strings.Builder

	// Indicator
	var indicator string
	if p.Installed {
		indicator = InstalledIndicator.String()
	} else {
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

	// Apply style
	if selected {
		b.WriteString(SelectedItemStyle.Render(line1))
		b.WriteString("\n")
		b.WriteString(SelectedItemStyle.Render(line2))
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

	// Position and context
	if m.textInput.Value() != "" {
		// Searching: show query and position
		query := m.textInput.Value()
		if len(query) > 20 {
			query = query[:17] + "..."
		}
		parts = append(parts, fmt.Sprintf("\"%s\" %d/%d", query, m.cursor+1, len(m.results)))
		parts = append(parts, "esc clear  ↑↓ Ctrl+j/k navigate  enter select")
	} else {
		// Not searching: show total and installed
		parts = append(parts, fmt.Sprintf("%d/%d (%d installed)", m.cursor+1, m.TotalPlugins(), m.InstalledCount()))
		parts = append(parts, "↑↓ Ctrl+j/k navigate  enter select  ? help")
	}

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
		{"↑ Ctrl+k/p", "Move up"},
		{"↓ Ctrl+j/n", "Move down"},
		{"Ctrl+u PgUp", "Page up"},
		{"Ctrl+d PgDn", "Page down"},
		{"Home", "Jump to top"},
		{"End", "Jump to bottom"},
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
		{"Esc Ctrl+g", "Clear search / Quit"},
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
		{"Ctrl+c", "Quit plum"},
	}
	for _, h := range generalKeys {
		b.WriteString(fmt.Sprintf("    %s  %s\n", KeyStyle.Width(12).Render(h.key), HelpTextStyle.Render(h.desc)))
	}

	b.WriteString("\n")

	// Tips section
	b.WriteString(HelpSectionStyle.Render("  ◆ Tips"))
	b.WriteString("\n")
	b.WriteString(HelpTextStyle.Render("    • Just start typing to search\n"))
	b.WriteString(HelpTextStyle.Render("    • Ctrl+key for navigation (fzf-style)\n"))
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
