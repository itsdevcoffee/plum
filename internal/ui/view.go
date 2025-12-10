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

	// Get the current view content
	var content string
	switch m.viewState {
	case ViewDetail:
		content = m.detailView()
	case ViewHelp:
		content = m.helpView()
	default:
		content = m.listView()
	}

	// Apply transition effect if animating
	if m.IsViewTransitioning() {
		switch m.transitionStyle {
		case TransitionZoom:
			content = m.applyZoomTransition(content)
		case TransitionSlideH:
			content = m.applySlideHTransition(content)
		case TransitionSlideV:
			content = m.applySlideVTransition(content)
		case TransitionFade:
			content = m.applyFadeTransition(content)
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

// applySlideHTransition creates a horizontal slide effect
func (m Model) applySlideHTransition(content string) string {
	progress := m.transitionProgress
	if progress >= 1.0 {
		return content
	}

	remaining := 1.0 - progress
	offset := int(remaining * float64(m.windowWidth) * float64(m.transitionDirection))

	lines := strings.Split(content, "\n")
	var result strings.Builder

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}

		if offset > 0 {
			// Sliding in from right: pad left
			padding := strings.Repeat(" ", offset)
			result.WriteString(padding)
			result.WriteString(truncateLine(line, m.windowWidth-offset))
		} else if offset < 0 {
			// Sliding in from left: skip chars
			result.WriteString(skipChars(line, -offset))
		} else {
			result.WriteString(line)
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

// applyFadeTransition creates a crossfade effect via dimming
func (m Model) applyFadeTransition(content string) string {
	progress := m.transitionProgress
	if progress >= 1.0 {
		return content
	}
	if progress < 0.1 {
		progress = 0.1
	}

	// Create a dim style based on progress
	// At progress=0: very dim, at progress=1: full brightness
	dimLevel := int((1.0 - progress) * 180) // 0-180 range for gray
	dimColor := lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", 80+dimLevel/2, 80+dimLevel/2, 80+dimLevel/2))

	// Apply dim overlay to content by reducing contrast
	// This is a simple approach - we dim the entire content
	dimStyle := lipgloss.NewStyle().Foreground(dimColor)

	lines := strings.Split(content, "\n")
	var result strings.Builder

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}
		// Only dim non-empty lines, preserve structure
		if len(strings.TrimSpace(line)) > 0 && progress < 0.7 {
			// Partial dim during early transition
			result.WriteString(dimStyle.Render(stripAnsi(line)))
		} else {
			result.WriteString(line)
		}
	}

	return result.String()
}

// truncateLine truncates a line to maxLen visible characters
func truncateLine(line string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(line)
	if len(runes) <= maxLen {
		return line
	}
	return string(runes[:maxLen])
}

// skipChars skips n characters from the start of a line
func skipChars(line string, n int) string {
	runes := []rune(line)
	if n >= len(runes) {
		return ""
	}
	return string(runes[n:])
}

// stripAnsi removes ANSI escape codes from a string (for re-styling)
func stripAnsi(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
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

// renderPluginItem renders a single plugin item based on display mode
func (m Model) renderPluginItem(p plugin.Plugin, selected bool) string {
	if m.displayMode == DisplaySimple {
		return m.renderPluginItemSimple(p, selected)
	}
	return m.renderPluginItemCard(p, selected)
}

// renderPluginItemSimple renders a compact one-line plugin item
func (m Model) renderPluginItemSimple(p plugin.Plugin, selected bool) string {
	// Indicator
	var indicator string
	if p.Installed {
		indicator = InstalledIndicator.String()
	} else {
		indicator = AvailableIndicator.String()
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
	cardWidth := m.windowWidth - 6
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
		parts = append(parts, "esc clear  ↑↓ navigate  enter select")
	} else {
		// Not searching: show total and installed
		parts = append(parts, fmt.Sprintf("%d/%d (%d installed)", m.cursor+1, m.TotalPlugins(), m.InstalledCount()))
		parts = append(parts, "↑↓ navigate  enter select  ? help")
	}

	// Show display mode (shift+tab to toggle)
	parts = append(parts, fmt.Sprintf("⇧tab: %s", m.DisplayModeName()))

	return StatusBarStyle.Render(strings.Join(parts, "  │  "))
}

// detailView renders the detail view for the selected plugin
func (m Model) detailView() string {
	p := m.SelectedPlugin()
	if p == nil {
		return AppStyle.Render("No plugin selected")
	}

	// Calculate content width (account for borders and padding)
	contentWidth := m.windowWidth - 10
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
		b.WriteString(DetailLabelStyle.Render("Install:") + " " + InstallCommandStyle.Render(p.InstallCommand()))
		b.WriteString("\n")
	}

	// Footer
	b.WriteString("\n")
	var footerParts []string
	footerParts = append(footerParts, KeyStyle.Render("esc")+" back")
	if !p.Installed {
		if m.copiedFlash {
			// Show "Copied!" feedback
			copiedStyle := lipgloss.NewStyle().Foreground(Green).Bold(true)
			footerParts = append(footerParts, copiedStyle.Render("✓ Copied!"))
		} else {
			footerParts = append(footerParts, KeyStyle.Render("c")+" copy install command")
		}
	}
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
