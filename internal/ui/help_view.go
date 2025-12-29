package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// helpView renders the help view with sticky header/footer
func (m Model) helpView() string {
	// Wrapper with margin (no bottom)
	helpWrapperStyle := lipgloss.NewStyle().
		Padding(1, 2, 0, 2)

	// Generate sticky header
	header := m.generateHelpHeader()

	// Generate sticky footer
	footer := m.generateHelpFooter()

	// Use viewport for scrollable content
	if m.helpViewport.Height > 0 {
		viewportContent := m.helpViewport.View()

		// Add scrollbar (aligned with viewport only)
		scrollbar := m.renderHelpScrollbar()
		contentWithScrollbar := lipgloss.JoinHorizontal(lipgloss.Top, viewportContent, scrollbar)

		// Stack: header (sticky) + viewport (scrolls) + footer (sticky)
		fullContent := lipgloss.JoinVertical(lipgloss.Left,
			header,
			contentWithScrollbar,
			footer,
		)

		// Wrap in box
		helpBoxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(PlumBright).
			Padding(1, 2)

		return helpWrapperStyle.Render(helpBoxStyle.Render(fullContent))
	}

	// Fallback: render everything together (no viewport)
	var fullContent strings.Builder
	fullContent.WriteString(header)
	fullContent.WriteString("\n")
	fullContent.WriteString(m.generateHelpSections())
	fullContent.WriteString("\n")
	fullContent.WriteString(footer)

	helpBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PlumBright).
		Padding(1, 2)

	return helpWrapperStyle.Render(helpBoxStyle.Render(fullContent.String()))
}

// generateHelpHeader generates the sticky header
func (m Model) generateHelpHeader() string {
	var b strings.Builder

	installedOnlyStyle := lipgloss.NewStyle().Foreground(Success)
	contentWidth := 58

	title := DetailTitleStyle.Render("🍑 plum Help")

	legendText := installedOnlyStyle.Render("🟢") + " = installed only"
	legendStyle := lipgloss.NewStyle().
		Foreground(TextMuted).
		Align(lipgloss.Right).
		Width(contentWidth - lipgloss.Width(title))
	legend := legendStyle.Render(legendText)

	headerLine := lipgloss.JoinHorizontal(lipgloss.Top, title, legend)

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

	// Plugin Actions section
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

	return b.String()
}

// renderHelpScrollbar renders a plum-themed scrollbar for the help viewport
func (m Model) renderHelpScrollbar() string {
	if m.helpViewport.Height <= 0 {
		return ""
	}

	// Check if content is scrollable
	if m.helpViewport.AtTop() && m.helpViewport.AtBottom() {
		return "" // Content fits, no scrollbar needed
	}

	// Get dimensions
	visibleHeight := m.helpViewport.Height
	scrollPercent := m.helpViewport.ScrollPercent()

	// Estimate total content height (heuristic)
	totalHeight := visibleHeight * 2

	// Calculate thumb size (proportional)
	thumbHeight := (visibleHeight * visibleHeight) / totalHeight
	if thumbHeight < 1 {
		thumbHeight = 1
	}
	if thumbHeight > visibleHeight {
		thumbHeight = visibleHeight
	}

	// Calculate thumb position
	trackHeight := visibleHeight - thumbHeight
	thumbPos := int(float64(trackHeight) * scrollPercent)

	// Render scrollbar with plum theme
	var scrollbar strings.Builder

	thumbStyle := lipgloss.NewStyle().Foreground(PlumBright)    // Orange thumb
	trackStyle := lipgloss.NewStyle().Foreground(BorderSubtle) // Brown track

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
