package ui

// KeyAction represents an action that can be triggered by a key press
type KeyAction int

// Key action constants define all possible user actions in the TUI
const (
	ActionNone KeyAction = iota // No action
	ActionQuit
	ActionBack
	ActionSelectItem
	ActionToggleHelp
	ActionToggleDisplayMode
	ActionCycleFilterNext
	ActionCycleFilterPrev
	ActionCopyInstallCommand
	ActionCopyMarketplaceCommand
	ActionCopyPluginCommand
	ActionCopyLink
	ActionCopyPath
	ActionOpenGitHub
	ActionOpenLocal
	ActionOpenMarketplaceBrowser
	ActionRefreshCache
	ActionCancelRefresh
	ActionClearSearch
)

// KeyBindings maps key strings to actions for each view
type KeyBindings map[string]KeyAction

// ListViewKeys defines key bindings for the list view
var ListViewKeys = KeyBindings{
	"q":         ActionQuit,
	"ctrl+c":    ActionQuit,
	"?":         ActionToggleHelp,
	"enter":     ActionSelectItem,
	"shift+v":   ActionToggleDisplayMode,
	"V":         ActionToggleDisplayMode,
	"tab":       ActionCycleFilterNext,
	"right":     ActionCycleFilterNext,
	"shift+tab": ActionCycleFilterPrev,
	"left":      ActionCycleFilterPrev,
	"shift+m":   ActionOpenMarketplaceBrowser,
	"M":         ActionOpenMarketplaceBrowser,
	"shift+u":   ActionRefreshCache,
	"U":         ActionRefreshCache,
	"esc":       ActionClearSearch, // Clears search, or quits if empty
	"ctrl+g":    ActionClearSearch,
}

// DetailViewKeys defines key bindings for the detail view
var DetailViewKeys = KeyBindings{
	"q":         ActionQuit,
	"esc":       ActionBack,
	"backspace": ActionBack,
	"c":         ActionCopyInstallCommand, // Or marketplace command if discoverable
	"y":         ActionCopyPluginCommand,  // For discoverable only
	"g":         ActionOpenGitHub,
	"l":         ActionCopyLink,
	"o":         ActionOpenLocal, // For installed only
	"p":         ActionCopyPath,  // For installed only
	"shift+m":   ActionOpenMarketplaceBrowser,
	"M":         ActionOpenMarketplaceBrowser,
	"?":         ActionToggleHelp,
}

// HelpViewKeys defines key bindings for the help view
var HelpViewKeys = KeyBindings{
	"q":         ActionQuit,
	"esc":       ActionBack,
	"?":         ActionBack,
	"backspace": ActionBack,
	"enter":     ActionBack,
	"shift+m":   ActionOpenMarketplaceBrowser,
	"M":         ActionOpenMarketplaceBrowser,
}

// MarketplaceListViewKeys defines key bindings for marketplace list view
var MarketplaceListViewKeys = KeyBindings{
	"q":         ActionQuit,
	"esc":       ActionBack,
	"backspace": ActionBack,
	"enter":     ActionSelectItem,
	"?":         ActionToggleHelp,
}

// MarketplaceDetailViewKeys defines key bindings for marketplace detail view
var MarketplaceDetailViewKeys = KeyBindings{
	"q":         ActionQuit,
	"esc":       ActionBack,
	"backspace": ActionBack,
	"?":         ActionToggleHelp,
	"f":         ActionNone, // Special: filter by marketplace (handled separately)
	"g":         ActionOpenGitHub,
	"l":         ActionCopyLink,
}

// GetKeyAction returns the action for a given key in the current view
func (m Model) GetKeyAction(key string) KeyAction {
	var bindings KeyBindings

	switch m.viewState {
	case ViewList:
		bindings = ListViewKeys
	case ViewDetail:
		bindings = DetailViewKeys
	case ViewHelp:
		bindings = HelpViewKeys
	case ViewMarketplaceList:
		bindings = MarketplaceListViewKeys
	case ViewMarketplaceDetail:
		bindings = MarketplaceDetailViewKeys
	default:
		return ActionNone
	}

	action, exists := bindings[key]
	if !exists {
		return ActionNone
	}

	return action
}
