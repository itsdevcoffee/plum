package ui

import "github.com/charmbracelet/lipgloss"

// Colors - Orange/Peach themed semantic palette
var (
	// Brand / Primary (Orange Scale - Dark to Bright)
	PlumMedium  = lipgloss.Color("#A0522D") // Deep burnt orange for selected borders
	PlumBright  = lipgloss.Color("#E67E22") // Rich orange for active elements, highlights
	PlumGlow    = lipgloss.Color("#FF8C42") // Bright orange for hover, glow states

	// Accent (Warm Peach)
	PeachSoft = lipgloss.Color("#FFAB91") // Notifications, discovery, headers

	// Semantic
	Success = lipgloss.Color("#10B981") // Teal-green complements orange

	// Text Hierarchy (Warm-tinted)
	TextPrimary   = lipgloss.Color("#FFF5EE") // Warm white/seashell
	TextSecondary = lipgloss.Color("#D4C4B8") // Warm beige-gray for descriptions
	TextTertiary  = lipgloss.Color("#A89888") // Warm mid-gray for de-emphasized
	TextMuted     = lipgloss.Color("#6B5D54") // Warm dark gray for subtle text

	// UI Structure
	BorderSubtle = lipgloss.Color("#5C4033") // Warm brown for borders
)

// Styles
var (
	// App container
	AppStyle = lipgloss.NewStyle().
			Padding(1, 2)

	// Title
	TitleStyle = lipgloss.NewStyle().
			Foreground(PeachSoft).
			Bold(true).
			MarginBottom(1)

	// Update notification box with gradient border
	UpdateNotificationStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(PeachSoft).
				Foreground(PeachSoft).
				Bold(true).
				Padding(0, 1)

	// Search input
	SearchPromptStyle = lipgloss.NewStyle().
				Foreground(PlumBright).
				Bold(true)

	SearchInputStyle = lipgloss.NewStyle().
				Foreground(TextPrimary)

	// Plugin list item - installed
	InstalledIndicator = lipgloss.NewStyle().
				Foreground(Success).
				SetString("●")

	// Plugin list item - available
	AvailableIndicator = lipgloss.NewStyle().
				Foreground(TextTertiary).
				SetString("○")

	// Discover badge for plugins from uninstalled marketplaces
	DiscoverBadge = lipgloss.NewStyle().
			Foreground(PeachSoft).
			Bold(true).
			SetString("[Discover]")

	// Plugin name
	PluginNameStyle = lipgloss.NewStyle().
			Foreground(TextPrimary).
			Bold(true)

	// Plugin name when selected/highlighted
	PluginNameSelectedStyle = lipgloss.NewStyle().
				Foreground(PlumGlow).
				Bold(true)

	// Plugin marketplace tag
	MarketplaceStyle = lipgloss.NewStyle().
				Foreground(TextTertiary)

	// Plugin version
	VersionStyle = lipgloss.NewStyle().
			Foreground(TextMuted)

	// Plugin description
	DescriptionStyle = lipgloss.NewStyle().
				Foreground(TextSecondary)

	// Plugin card - normal state
	PluginCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderSubtle).
			Padding(0, 1)

	// Plugin card - selected state
	PluginCardSelectedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(PlumMedium). // Richer plum for selected cards
				Padding(0, 1)

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(TextTertiary).
			MarginTop(1)

	// Dim separator for tabs/status bar
	DimSeparator = lipgloss.NewStyle().
			Foreground(TextMuted)

	// Help text
	HelpStyle = lipgloss.NewStyle().
			Foreground(TextMuted)

	// Detail view styles
	DetailBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(PlumBright).
			Padding(1, 2)

	DetailTitleStyle = lipgloss.NewStyle().
				Foreground(TextPrimary).
				Bold(true).
				MarginBottom(1)

	DetailLabelStyle = lipgloss.NewStyle().
				Foreground(TextTertiary).
				Width(12)

	DetailValueStyle = lipgloss.NewStyle().
				Foreground(TextPrimary)

	DetailDescStyle = lipgloss.NewStyle().
			Foreground(TextSecondary).
			MarginTop(1).
			MarginBottom(1)

	InstallCommandStyle = lipgloss.NewStyle().
				Foreground(Success).
				Background(TextMuted).
				Padding(0, 1)

	// Discover message style for marketplace install instructions
	DiscoverMessageStyle = lipgloss.NewStyle().
				Foreground(PeachSoft).
				Italic(true)

	KeyStyle = lipgloss.NewStyle().
			Foreground(PlumBright).
			Bold(true)

	// Badge styles
	InstalledBadge = lipgloss.NewStyle().
			Foreground(Success).
			Bold(true).
			SetString("[Installed]")

	AvailableBadge = lipgloss.NewStyle().
			Foreground(TextTertiary).
			SetString("[Available]")

	// Help view styles
	HelpSectionStyle = lipgloss.NewStyle().
				Foreground(PeachSoft).
				Bold(true)

	HelpTextStyle = lipgloss.NewStyle().
			Foreground(TextSecondary)

	// Animation highlight bars - sliding selection indicator
	HighlightBarFull = lipgloss.NewStyle().
				Foreground(PlumBright).
				Bold(true).
				SetString("▌ ")

	HighlightBarMedium = lipgloss.NewStyle().
				Foreground(PlumGlow).
				SetString("▌ ")

	HighlightBarLight = lipgloss.NewStyle().
				Foreground(TextTertiary).
				SetString("│ ")
)
