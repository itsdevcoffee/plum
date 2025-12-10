package ui

import "github.com/charmbracelet/lipgloss"

// Colors
var (
	Purple     = lipgloss.Color("#7D56F4")
	LightPurple = lipgloss.Color("#9D76FF")
	Green      = lipgloss.Color("#04B575")
	Gray       = lipgloss.Color("#626262")
	LightGray  = lipgloss.Color("#ABABAB")
	DarkGray   = lipgloss.Color("#3C3C3C")
	White      = lipgloss.Color("#FAFAFA")
	Peach      = lipgloss.Color("#FFAB91")
)

// Styles
var (
	// App container
	AppStyle = lipgloss.NewStyle().
			Padding(1, 2)

	// Title
	TitleStyle = lipgloss.NewStyle().
			Foreground(Peach).
			Bold(true).
			MarginBottom(1)

	// Search input
	SearchPromptStyle = lipgloss.NewStyle().
				Foreground(Purple).
				Bold(true)

	SearchInputStyle = lipgloss.NewStyle().
				Foreground(White)

	// Plugin list item - installed
	InstalledIndicator = lipgloss.NewStyle().
				Foreground(Green).
				SetString("●")

	// Plugin list item - available
	AvailableIndicator = lipgloss.NewStyle().
				Foreground(Gray).
				SetString("○")

	// Plugin name
	PluginNameStyle = lipgloss.NewStyle().
			Foreground(White).
			Bold(true)

	// Plugin name when selected/highlighted
	PluginNameSelectedStyle = lipgloss.NewStyle().
				Foreground(LightPurple).
				Bold(true)

	// Plugin marketplace tag
	MarketplaceStyle = lipgloss.NewStyle().
				Foreground(Gray)

	// Plugin version
	VersionStyle = lipgloss.NewStyle().
			Foreground(DarkGray)

	// Plugin description
	DescriptionStyle = lipgloss.NewStyle().
				Foreground(LightGray)

	// Plugin card - normal state
	PluginCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(DarkGray).
			Padding(0, 1)

	// Plugin card - selected state
	PluginCardSelectedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Purple).
				Padding(0, 1)

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(Gray).
			MarginTop(1)

	// Help text
	HelpStyle = lipgloss.NewStyle().
			Foreground(DarkGray)

	// Detail view styles
	DetailBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Purple).
			Padding(1, 2)

	DetailTitleStyle = lipgloss.NewStyle().
				Foreground(White).
				Bold(true).
				MarginBottom(1)

	DetailLabelStyle = lipgloss.NewStyle().
				Foreground(Gray).
				Width(12)

	DetailValueStyle = lipgloss.NewStyle().
				Foreground(White)

	DetailDescStyle = lipgloss.NewStyle().
			Foreground(LightGray).
			MarginTop(1).
			MarginBottom(1)

	InstallCommandStyle = lipgloss.NewStyle().
				Foreground(Green).
				Background(DarkGray).
				Padding(0, 1)

	KeyStyle = lipgloss.NewStyle().
			Foreground(Purple).
			Bold(true)

	// Badge styles
	InstalledBadge = lipgloss.NewStyle().
			Foreground(Green).
			Bold(true).
			SetString("[Installed]")

	AvailableBadge = lipgloss.NewStyle().
			Foreground(Gray).
			SetString("[Available]")

	// Help view styles
	HelpSectionStyle = lipgloss.NewStyle().
				Foreground(Peach).
				Bold(true)

	HelpTextStyle = lipgloss.NewStyle().
			Foreground(LightGray)

	// Animation highlight bars - sliding selection indicator
	HighlightBarFull = lipgloss.NewStyle().
				Foreground(Purple).
				Bold(true).
				SetString("▌ ")

	HighlightBarMedium = lipgloss.NewStyle().
				Foreground(LightPurple).
				SetString("▌ ")

	HighlightBarLight = lipgloss.NewStyle().
				Foreground(Gray).
				SetString("│ ")
)
