package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// QuickAction represents a quick action menu item
type QuickAction struct {
	Key         string
	Label       string
	Description string
	Enabled     bool
}

// GetQuickActionsForView returns context-aware quick actions for the current view
func (m Model) GetQuickActionsForView() []QuickAction {
	switch m.viewState {
	case ViewList:
		return m.getPluginListQuickActions()
	case ViewDetail:
		return m.getPluginDetailQuickActions()
	case ViewMarketplaceList:
		return m.getMarketplaceListQuickActions()
	default:
		return []QuickAction{}
	}
}

// getPluginListQuickActions returns quick actions for plugin list view
func (m Model) getPluginListQuickActions() []QuickAction {
	return []QuickAction{
		{Key: "m", Label: "Browse Marketplaces", Description: "Open marketplace browser", Enabled: true},
		{Key: "f", Label: "Filter by Marketplace", Description: "Pick marketplace to filter plugins", Enabled: true},
		{Key: "s", Label: "Sort by", Description: "Change sort order", Enabled: true},
		{Key: "v", Label: "Toggle View", Description: "Switch between card and slim view", Enabled: true},
		{Key: "u", Label: "Refresh", Description: "Refresh marketplace data", Enabled: true},
	}
}

// getPluginDetailQuickActions returns quick actions for plugin detail view
func (m Model) getPluginDetailQuickActions() []QuickAction {
	p := m.SelectedPlugin()
	if p == nil {
		return []QuickAction{}
	}

	actions := []QuickAction{}

	if !p.Installed && p.Installable() {
		if p.IsDiscoverable {
			// Discoverable plugin - show 2-step install
			actions = append(actions, QuickAction{
				Key:         "i",
				Label:       "Copy 2-Step Install",
				Description: "Copy marketplace + plugin install commands",
				Enabled:     true,
			})
			actions = append(actions, QuickAction{
				Key:         "m",
				Label:       "Copy Marketplace",
				Description: "Copy marketplace install command",
				Enabled:     true,
			})
			actions = append(actions, QuickAction{
				Key:         "p",
				Label:       "Copy Plugin",
				Description: "Copy plugin install command",
				Enabled:     true,
			})
		} else {
			// Ready to install - show single install command
			actions = append(actions, QuickAction{
				Key:         "c",
				Label:       "Copy Install",
				Description: "Copy plugin install command",
				Enabled:     true,
			})
		}
	}

	// Always show GitHub and link copy
	actions = append(actions, QuickAction{
		Key:         "g",
		Label:       "GitHub",
		Description: "Open on GitHub",
		Enabled:     p.GitHubURL() != "",
	})
	actions = append(actions, QuickAction{
		Key:         "l",
		Label:       "Copy Link",
		Description: "Copy GitHub URL to clipboard",
		Enabled:     p.GitHubURL() != "",
	})

	// For installed plugins, show local actions
	if p.Installed && p.InstallPath != "" {
		actions = append(actions, QuickAction{
			Key:         "o",
			Label:       "Open Local",
			Description: "Open install directory",
			Enabled:     true,
		})
		actions = append(actions, QuickAction{
			Key:         "p",
			Label:       "Copy Path",
			Description: "Copy install path to clipboard",
			Enabled:     true,
		})
	}

	return actions
}

// getMarketplaceListQuickActions returns quick actions for marketplace list view
func (m Model) getMarketplaceListQuickActions() []QuickAction {
	hasSelection := len(m.marketplaceItems) > 0 && m.marketplaceCursor < len(m.marketplaceItems)

	return []QuickAction{
		{Key: "enter", Label: "View Details", Description: "Show marketplace details", Enabled: hasSelection},
		{Key: "f", Label: "Show Plugins", Description: "Filter plugin list by this marketplace", Enabled: hasSelection},
		{Key: "i", Label: "Copy Install", Description: "Copy marketplace install command", Enabled: hasSelection},
		{Key: "g", Label: "GitHub", Description: "Open on GitHub", Enabled: hasSelection},
	}
}

// renderQuickMenu renders the quick action menu overlay
func (m Model) renderQuickMenu() string {
	actions := m.GetQuickActionsForView()
	if len(actions) == 0 {
		return ""
	}

	// Menu styles
	menuBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PlumBright).
		Padding(1, 2).
		Width(40)

	titleStyle := lipgloss.NewStyle().
		Foreground(PlumBright).
		Bold(true)

	selectedStyle := lipgloss.NewStyle().
		Foreground(TextPrimary).
		Background(PlumMedium).
		Bold(true)

	normalStyle := lipgloss.NewStyle().
		Foreground(TextPrimary)

	disabledStyle := lipgloss.NewStyle().
		Foreground(TextMuted).
		Italic(true)

	keyStyle := lipgloss.NewStyle().
		Foreground(PeachSoft).
		Bold(true)

	// Build menu content
	var b strings.Builder
	b.WriteString(titleStyle.Render("Quick Actions"))
	b.WriteString("\n\n")

	for i, action := range actions {
		isSelected := i == m.quickMenuCursor

		// Build action line
		var line string
		if action.Enabled {
			keyPart := keyStyle.Render(fmt.Sprintf("[%s]", action.Key))
			labelPart := action.Label

			if isSelected {
				line = fmt.Sprintf("▸ %s %s", keyPart, selectedStyle.Render(labelPart))
			} else {
				line = fmt.Sprintf("  %s %s", keyPart, normalStyle.Render(labelPart))
			}
		} else {
			keyPart := fmt.Sprintf("[%s]", action.Key)
			line = fmt.Sprintf("  %s %s", keyPart, disabledStyle.Render(action.Label+" (unavailable)"))
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("↑↓ navigate  •  Enter or key to select  •  Esc to close"))

	return menuBorder.Render(b.String())
}

// renderQuickMenuOverlay renders the quick menu centered on screen
func (m Model) renderQuickMenuOverlay(baseView string) string {
	menu := m.renderQuickMenu()

	// Calculate centering
	menuWidth := lipgloss.Width(menu)
	menuHeight := lipgloss.Height(menu)

	// Position menu in center of screen
	horizontalPadding := (m.windowWidth - menuWidth) / 2
	if horizontalPadding < 0 {
		horizontalPadding = 0
	}

	verticalPadding := (m.windowHeight - menuHeight) / 2
	if verticalPadding < 0 {
		verticalPadding = 0
	}

	// Place menu over base view
	overlayStyle := lipgloss.NewStyle().
		PaddingLeft(horizontalPadding).
		PaddingTop(verticalPadding)

	// For simplicity, just render the menu centered
	// Note: baseView is passed for future overlay implementation
	_ = baseView
	return overlayStyle.Render(menu)
}

// OpenQuickMenu opens the quick action menu from current view
func (m *Model) OpenQuickMenu() {
	m.quickMenuActive = true
	m.quickMenuCursor = 0
	m.quickMenuPreviousView = m.viewState
	m.viewState = ViewQuickMenu
}

// CloseQuickMenu closes the quick action menu and returns to previous view
func (m *Model) CloseQuickMenu() {
	m.quickMenuActive = false
	m.quickMenuCursor = 0
	m.viewState = m.quickMenuPreviousView
}

// ExecuteQuickMenuAction executes the selected quick menu action
func (m *Model) ExecuteQuickMenuAction() tea.Cmd {
	actions := m.GetQuickActionsForView()
	if m.quickMenuCursor >= len(actions) || m.quickMenuCursor < 0 {
		return nil
	}

	action := actions[m.quickMenuCursor]
	if !action.Enabled {
		return nil
	}

	// Close menu first
	m.CloseQuickMenu()

	// Execute action based on key
	// We'll delegate to the appropriate key handler by synthesizing a key event
	return func() tea.Msg {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(action.Key)}
	}
}

// NextQuickMenuAction moves cursor to next action
func (m *Model) NextQuickMenuAction() {
	actions := m.GetQuickActionsForView()
	if len(actions) == 0 {
		return
	}
	m.quickMenuCursor = (m.quickMenuCursor + 1) % len(actions)
}

// PrevQuickMenuAction moves cursor to previous action
func (m *Model) PrevQuickMenuAction() {
	actions := m.GetQuickActionsForView()
	if len(actions) == 0 {
		return
	}
	m.quickMenuCursor = (m.quickMenuCursor - 1 + len(actions)) % len(actions)
}
