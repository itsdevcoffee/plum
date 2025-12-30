package ui

import (
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"
	"github.com/charmbracelet/lipgloss"
	"github.com/itsdevcoffee/plum/internal/config"
	"github.com/itsdevcoffee/plum/internal/marketplace"
	"github.com/itsdevcoffee/plum/internal/plugin"
	"github.com/itsdevcoffee/plum/internal/search"
)

// ViewState represents the current view
type ViewState int

const (
	ViewList ViewState = iota
	ViewDetail
	ViewHelp
	ViewMarketplaceList   // Marketplace browser view
	ViewMarketplaceDetail // Marketplace detail view
)

// TransitionStyle represents the animation style for view transitions
type TransitionStyle int

const (
	TransitionInstant TransitionStyle = iota // No animation
	TransitionZoom                           // Center expand/contract
	TransitionSlideV                         // Vertical slide (push up/down)
)

// ListDisplayMode represents how plugin items are displayed
type ListDisplayMode int

const (
	DisplayCard ListDisplayMode = iota // Card view with borders and description
	DisplaySlim                        // Slim one-line view
)

// FilterMode represents which plugins to show
type FilterMode int

const (
	FilterAll       FilterMode = iota // Show all plugins (installed + ready + discoverable)
	FilterDiscover                    // Show only discoverable (from uninstalled marketplaces)
	FilterReady                       // Show only ready to install (marketplace installed, plugin not)
	FilterInstalled                   // Show only installed
)

// FilterModeNames for display
var FilterModeNames = []string{"All", "Discover", "Ready", "Installed"}

// TransitionStyleNames for display
var TransitionStyleNames = []string{"Instant", "Zoom", "Slide V"}

// Scroll buffer - cursor stays this many items from edge before scrolling
const scrollBuffer = 2

// Layout constraints
const maxContentWidth = 120

// Animation constants
const (
	animationFPS    = 60
	springFrequency = 20.0 // Higher = faster (snappy)
	springDamping   = 0.9  // < 1 = bouncy, 1 = smooth, > 1 = slow
)

// Model is the main application model
type Model struct {
	// Data
	allPlugins           []plugin.Plugin
	results              []search.RankedPlugin
	loading              bool
	refreshing           bool   // True when manually refreshing cache
	refreshProgress      int    // Number of marketplaces refreshed
	refreshTotal         int    // Total marketplaces to refresh
	refreshCurrent       string // Current marketplace being fetched
	newMarketplacesCount int    // Number of new marketplaces available in registry

	// UI state
	textInput           textinput.Model
	spinner             spinner.Model
	helpViewport        viewport.Model
	detailViewport      viewport.Model
	cursor              int
	scrollOffset        int
	viewState           ViewState
	displayMode         ListDisplayMode
	filterMode          FilterMode
	windowWidth         int
	windowHeight        int
	copiedFlash         bool // Brief "Copied!" indicator (for 'c')
	linkCopiedFlash     bool // Brief "Link Copied!" indicator (for 'l')
	pathCopiedFlash     bool // Brief "Path Copied!" indicator (for 'p')
	githubOpenedFlash   bool // Brief "Opened!" indicator (for 'g')
	localOpenedFlash    bool // Brief "Opened!" indicator (for 'o')
	clipboardErrorFlash bool // Brief "Clipboard error!" indicator

	// Marketplace view state
	marketplaceItems              []MarketplaceItem
	marketplaceCursor             int
	marketplaceScrollOffset       int
	marketplaceSortMode           MarketplaceSortMode
	selectedMarketplace           *MarketplaceItem
	previousViewBeforeMarketplace ViewState

	// Animation state
	cursorY         float64 // Animated cursor position
	cursorYVelocity float64
	targetCursorY   float64
	spring          harmonica.Spring

	// View transition state
	transitionProgress  float64 // 0.0 = old view, 1.0 = new view
	transitionVelocity  float64
	targetTransition    float64
	previousView        ViewState       // View we're transitioning FROM
	transitionDirection int             // 1 = forward (right to left), -1 = back (left to right)
	transitionStyle     TransitionStyle // Current animation style

	// Error state
	err error
}

// NewModel creates a new Model with initial state
func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Search plugins..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 40
	ti.PromptStyle = SearchPromptStyle
	ti.TextStyle = SearchInputStyle
	ti.Prompt = "> "

	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(PeachSoft)

	// Initialize spring for animations
	spring := harmonica.NewSpring(harmonica.FPS(animationFPS), springFrequency, springDamping)

	return Model{
		textInput:                     ti,
		spinner:                       s,
		spring:                        spring,
		loading:                       true,
		viewState:                     ViewList,
		previousView:                  ViewList,
		displayMode:                   DisplaySlim,       // Default to slim mode
		marketplaceSortMode:           SortByPluginCount, // Default marketplace sort
		transitionProgress:            1.0,               // Start fully transitioned (no animation on init)
		targetTransition:              1.0,
		transitionStyle:               TransitionInstant, // Default to instant (no animation)
		windowWidth:                   80,
		windowHeight:                  24,
		previousViewBeforeMarketplace: ViewList,
	}
}

// CycleTransitionStyle cycles to the next transition style
func (m *Model) CycleTransitionStyle() {
	m.transitionStyle = (m.transitionStyle + 1) % 3
}

// TransitionStyleName returns the current transition style name
func (m Model) TransitionStyleName() string {
	return TransitionStyleNames[m.transitionStyle]
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.spinner.Tick,
		loadPlugins,
		checkRegistryForUpdates, // Check for new marketplaces
	)
}

// checkRegistryForUpdates checks if there are new marketplaces in the registry
func checkRegistryForUpdates() tea.Msg {
	// Will be set by update.go to call marketplace.FetchRegistryWithComparison
	_, newCount, err := checkForNewMarketplaces()
	if err != nil || newCount == 0 {
		return registryCheckedMsg{newCount: 0}
	}
	return registryCheckedMsg{newCount: newCount}
}

// PopularMarketplace is re-exported to avoid import in function signature
type PopularMarketplace struct {
	Name        string
	DisplayName string
	Repo        string
	Description string
}

// checkForNewMarketplaces wrapper to avoid circular import
var checkForNewMarketplaces = func() ([]PopularMarketplace, int, error) {
	return nil, 0, nil // Will be set by update.go
}

// pluginsLoadedMsg is sent when plugins are loaded
type pluginsLoadedMsg struct {
	plugins []plugin.Plugin
	err     error
}

// loadPlugins loads all plugins from config
func loadPlugins() tea.Msg {
	plugins, err := config.LoadAllPlugins()
	return pluginsLoadedMsg{plugins: plugins, err: err}
}

// refreshCacheMsg is sent to initiate cache refresh
type refreshCacheMsg struct{}

// registryCheckedMsg is sent when registry check completes
type registryCheckedMsg struct {
	newCount int
}

// refreshProgressMsg is sent during refresh to update progress
type refreshProgressMsg struct {
	current   string // Current marketplace being fetched
	completed int    // Number completed so far
	total     int    // Total to fetch
}

// doRefreshCache performs the actual cache refresh
// This runs in a goroutine automatically by Bubble Tea
func doRefreshCache() tea.Msg {
	// TODO: Add progress updates here once we refactor clearCacheAndReload
	// to accept a progress callback

	// Clear cache and reload
	if err := clearCacheAndReload(); err != nil {
		return pluginsLoadedMsg{plugins: nil, err: err}
	}

	// Reload plugins after cache clear
	plugins, err := config.LoadAllPlugins()
	if err != nil {
		return pluginsLoadedMsg{plugins: nil, err: err}
	}

	return pluginsLoadedMsg{plugins: plugins, err: nil}
}

// clearCacheAndReload is set by update.go to avoid circular import
var clearCacheAndReload = func() error {
	return nil // Will be set by update.go
}

// SelectedPlugin returns the currently selected plugin, if any
func (m Model) SelectedPlugin() *plugin.Plugin {
	if len(m.results) == 0 || m.cursor >= len(m.results) {
		return nil
	}
	return &m.results[m.cursor].Plugin
}

// VisibleResults returns the results that should be visible given the window size
func (m Model) VisibleResults() []search.RankedPlugin {
	maxVisible := m.maxVisibleItems()
	if len(m.results) <= maxVisible {
		return m.results
	}

	start := m.scrollOffset
	end := start + maxVisible
	if end > len(m.results) {
		end = len(m.results)
	}

	return m.results[start:end]
}

// ScrollOffset returns the current scroll offset
func (m Model) ScrollOffset() int {
	return m.scrollOffset
}

// UpdateScroll adjusts scroll offset to keep cursor visible with buffer
func (m *Model) UpdateScroll() {
	maxVisible := m.maxVisibleItems()
	if len(m.results) <= maxVisible {
		m.scrollOffset = 0
		return
	}

	// Cursor too close to top - scroll up
	if m.cursor < m.scrollOffset+scrollBuffer {
		m.scrollOffset = m.cursor - scrollBuffer
		if m.scrollOffset < 0 {
			m.scrollOffset = 0
		}
	}

	// Cursor too close to bottom - scroll down
	if m.cursor >= m.scrollOffset+maxVisible-scrollBuffer {
		m.scrollOffset = m.cursor - maxVisible + scrollBuffer + 1
		if m.scrollOffset > len(m.results)-maxVisible {
			m.scrollOffset = len(m.results) - maxVisible
		}
	}
}

// maxVisibleItems returns the maximum number of items that can be displayed
func (m Model) maxVisibleItems() int {
	// Account for title (1) + blanks (2) + search (1) + blank (1) + filters (1) + blanks (2)
	// + blank before status (1) + status (1) + AppStyle padding top/bottom (2) = 12 lines
	available := m.windowHeight - 12
	if m.displayMode == DisplaySlim {
		// Slim view: 1 line per item
		return available
	}
	// Card view: 4 lines per item (2 content rows + 2 border rows)
	return available / 4
}

// ToggleDisplayMode switches between card and slim view
func (m *Model) ToggleDisplayMode() {
	if m.displayMode == DisplayCard {
		m.displayMode = DisplaySlim
	} else {
		m.displayMode = DisplayCard
	}
	// Reset scroll to keep cursor visible with new item heights
	m.UpdateScroll()
}

// DisplayModeName returns the current display mode name
func (m Model) DisplayModeName() string {
	if m.displayMode == DisplaySlim {
		return "slim"
	}
	return "verbose"
}

// ContentWidth returns the effective content width (capped at max)
func (m Model) ContentWidth() int {
	if m.windowWidth > maxContentWidth {
		return maxContentWidth
	}
	return m.windowWidth
}

// NextFilter cycles to the next filter mode
func (m *Model) NextFilter() {
	m.filterMode = (m.filterMode + 1) % 4
	m.applyFilter()
}

// PrevFilter cycles to the previous filter mode
func (m *Model) PrevFilter() {
	m.filterMode = (m.filterMode + 3) % 4 // +3 is same as -1 mod 4
	m.applyFilter()
}

// applyFilter re-runs search with current filter and resets cursor
func (m *Model) applyFilter() {
	m.results = m.filteredSearch(m.textInput.Value())
	m.cursor = 0
	m.scrollOffset = 0
	m.SnapCursorToTarget()
}

// filteredSearch runs search and applies the current filter
func (m Model) filteredSearch(query string) []search.RankedPlugin {
	// Check for marketplace filter (starts with @)
	if strings.HasPrefix(query, "@") {
		marketplaceName := strings.TrimPrefix(query, "@")
		var filtered []search.RankedPlugin
		for _, p := range m.allPlugins {
			if p.Marketplace == marketplaceName {
				filtered = append(filtered, search.RankedPlugin{
					Plugin: p,
					Score:  1.0,
				})
			}
		}
		return filtered
	}

	// First get all search results
	allResults := search.Search(query, m.allPlugins)

	// Apply filter
	switch m.filterMode {
	case FilterDiscover:
		// Show only discoverable (from uninstalled marketplaces)
		filtered := make([]search.RankedPlugin, 0)
		for _, r := range allResults {
			if r.Plugin.IsDiscoverable {
				filtered = append(filtered, r)
			}
		}
		return filtered

	case FilterReady:
		// Show only ready to install (not installed, marketplace IS installed)
		filtered := make([]search.RankedPlugin, 0)
		for _, rp := range allResults {
			if !rp.Plugin.Installed && !rp.Plugin.IsDiscoverable {
				filtered = append(filtered, rp)
			}
		}
		return filtered
	case FilterInstalled:
		filtered := make([]search.RankedPlugin, 0)
		for _, rp := range allResults {
			if rp.Plugin.Installed {
				filtered = append(filtered, rp)
			}
		}
		return filtered
	default:
		return allResults
	}
}

// FilterModeName returns the current filter mode name
func (m Model) FilterModeName() string {
	return FilterModeNames[m.filterMode]
}

// ReadyCount returns count of ready-to-install plugins (marketplace installed, plugin not)
func (m Model) ReadyCount() int {
	count := 0
	for _, p := range m.allPlugins {
		if !p.Installed && !p.IsDiscoverable {
			count++
		}
	}
	return count
}

// DiscoverableCount returns count of discoverable plugins (from uninstalled marketplaces)
func (m Model) DiscoverableCount() int {
	count := 0
	for _, p := range m.allPlugins {
		if p.IsDiscoverable {
			count++
		}
	}
	return count
}

// TotalPlugins returns total plugin count
func (m Model) TotalPlugins() int {
	return len(m.allPlugins)
}

// InstalledCount returns count of installed plugins
func (m Model) InstalledCount() int {
	count := 0
	for _, p := range m.allPlugins {
		if p.Installed {
			count++
		}
	}
	return count
}

// SetCursorTarget updates the animation target immediately (call on cursor change)
func (m *Model) SetCursorTarget() {
	m.targetCursorY = float64(m.cursor - m.scrollOffset)
}

// UpdateCursorAnimation advances the spring animation one frame
func (m *Model) UpdateCursorAnimation() {
	m.cursorY, m.cursorYVelocity = m.spring.Update(m.cursorY, m.cursorYVelocity, m.targetCursorY)
}

// SnapCursorToTarget instantly moves cursor to target (no animation)
func (m *Model) SnapCursorToTarget() {
	m.targetCursorY = float64(m.cursor - m.scrollOffset)
	m.cursorY = m.targetCursorY
	m.cursorYVelocity = 0
}

// AnimatedCursorOffset returns how far the animated cursor is from target (for glow effect)
func (m Model) AnimatedCursorOffset() float64 {
	diff := m.cursorY - m.targetCursorY
	if diff < 0 {
		diff = -diff
	}
	return diff
}

// IsAnimating returns true if cursor animation is in progress
func (m Model) IsAnimating() bool {
	diff := m.AnimatedCursorOffset()
	velocityMagnitude := m.cursorYVelocity
	if velocityMagnitude < 0 {
		velocityMagnitude = -velocityMagnitude
	}
	return diff > 0.01 || velocityMagnitude > 0.01
}

// StartViewTransition begins a transition to a new view
func (m *Model) StartViewTransition(newView ViewState, direction int) {
	if m.viewState == newView {
		return
	}
	m.previousView = m.viewState
	m.viewState = newView
	m.transitionProgress = 0.0
	m.transitionVelocity = 0.0
	m.targetTransition = 1.0
	m.transitionDirection = direction
}

// UpdateViewTransition advances the view transition animation
func (m *Model) UpdateViewTransition() {
	m.transitionProgress, m.transitionVelocity = m.spring.Update(
		m.transitionProgress, m.transitionVelocity, m.targetTransition,
	)
}

// IsViewTransitioning returns true if a view transition is in progress
func (m Model) IsViewTransitioning() bool {
	diff := m.targetTransition - m.transitionProgress
	if diff < 0 {
		diff = -diff
	}
	velMag := m.transitionVelocity
	if velMag < 0 {
		velMag = -velMag
	}
	return diff > 0.01 || velMag > 0.01
}

// TransitionOffset returns the horizontal offset for rendering during transition
// Returns a value from 0 to windowWidth based on progress and direction
func (m Model) TransitionOffset() int {
	remaining := 1.0 - m.transitionProgress
	return int(remaining * float64(m.windowWidth) * float64(m.transitionDirection))
}

// Marketplace View Functions

// LoadMarketplaceItems loads and processes all marketplaces with status and stats
func (m *Model) LoadMarketplaceItems() error {
	// 1. Load known marketplaces (installed)
	knownMarketplaces, err := config.LoadKnownMarketplaces()
	if err != nil {
		knownMarketplaces = make(config.KnownMarketplaces)
	}

	// 2. Get marketplace list from registry (or hardcoded fallback)
	marketplaceList, err := marketplace.FetchRegistry()
	if err != nil {
		marketplaceList = marketplace.PopularMarketplaces
	}

	// 3. Count installed plugins per marketplace
	installed, _ := config.LoadInstalledPlugins()
	installedByMarketplace := make(map[string]int)
	if installed != nil {
		for fullName := range installed.Plugins {
			// fullName format: "plugin@marketplace"
			parts := []string{fullName}
			if idx := len(fullName) - 1; idx >= 0 {
				for i := len(fullName) - 1; i >= 0; i-- {
					if fullName[i] == '@' {
						parts = []string{fullName[:i], fullName[i+1:]}
						break
					}
				}
			}
			if len(parts) == 2 {
				installedByMarketplace[parts[1]]++
			}
		}
	}

	// 4. Build MarketplaceItem array
	var items []MarketplaceItem
	for _, pm := range marketplaceList {
		item := MarketplaceItem{
			Name:                 pm.Name,
			DisplayName:          pm.DisplayName,
			Repo:                 pm.Repo,
			Description:          pm.Description,
			InstalledPluginCount: installedByMarketplace[pm.Name],
		}

		// Determine status
		if _, isInstalled := knownMarketplaces[pm.Name]; isInstalled {
			item.Status = MarketplaceInstalled
		} else {
			item.Status = MarketplaceAvailable
		}

		// Try to get total plugin count from cached manifest
		if cached, _ := marketplace.LoadFromCache(pm.Name); cached != nil {
			item.TotalPluginCount = len(cached.Plugins)
			if item.Status == MarketplaceAvailable {
				item.Status = MarketplaceCached
			}
		}

		// Load GitHub stats: prefer cache, fallback to static stats
		if stats, err := marketplace.LoadStatsFromCache(pm.Name); err == nil && stats != nil {
			item.GitHubStats = stats
		} else if pm.StaticStats != nil {
			// Use static stats as fallback (snapshot from codebase)
			item.GitHubStats = pm.StaticStats
		}

		items = append(items, item)
	}

	m.marketplaceItems = items
	m.ApplyMarketplaceSort()

	return nil
}

// ApplyMarketplaceSort sorts marketplace items based on current sort mode
func (m *Model) ApplyMarketplaceSort() {
	items := m.marketplaceItems

	switch m.marketplaceSortMode {
	case SortByPluginCount:
		sortMarketplacesByPluginCount(items)
	case SortByStars:
		sortMarketplacesByStars(items)
	case SortByName:
		sortMarketplacesByName(items)
	case SortByLastUpdated:
		sortMarketplacesByLastUpdated(items)
	}

	m.marketplaceItems = items
}

// sortMarketplacesByPluginCount sorts by total plugin count (descending)
func sortMarketplacesByPluginCount(items []MarketplaceItem) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].TotalPluginCount > items[j].TotalPluginCount
	})
}

// sortMarketplacesByStars sorts by GitHub stars (descending)
func sortMarketplacesByStars(items []MarketplaceItem) {
	sort.Slice(items, func(i, j int) bool {
		si := 0
		sj := 0
		if items[i].GitHubStats != nil {
			si = items[i].GitHubStats.Stars
		}
		if items[j].GitHubStats != nil {
			sj = items[j].GitHubStats.Stars
		}
		return si > sj
	})
}

// sortMarketplacesByName sorts alphabetically by display name
func sortMarketplacesByName(items []MarketplaceItem) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].DisplayName < items[j].DisplayName
	})
}

// sortMarketplacesByLastUpdated sorts by last push date (most recent first)
func sortMarketplacesByLastUpdated(items []MarketplaceItem) {
	sort.Slice(items, func(i, j int) bool {
		var ti, tj time.Time
		if items[i].GitHubStats != nil {
			ti = items[i].GitHubStats.LastPushedAt
		}
		if items[j].GitHubStats != nil {
			tj = items[j].GitHubStats.LastPushedAt
		}
		return ti.After(tj)
	})
}

// VisibleMarketplaceItems returns visible marketplace items based on scroll
func (m Model) VisibleMarketplaceItems() []MarketplaceItem {
	maxVisible := m.maxVisibleItems()
	if len(m.marketplaceItems) <= maxVisible {
		return m.marketplaceItems
	}

	start := m.marketplaceScrollOffset
	end := start + maxVisible
	if end > len(m.marketplaceItems) {
		end = len(m.marketplaceItems)
	}

	return m.marketplaceItems[start:end]
}

// UpdateMarketplaceScroll adjusts scroll offset for marketplace view
func (m *Model) UpdateMarketplaceScroll() {
	maxVisible := m.maxVisibleItems()
	if len(m.marketplaceItems) <= maxVisible {
		m.marketplaceScrollOffset = 0
		return
	}

	// Keep cursor visible with buffer
	if m.marketplaceCursor < m.marketplaceScrollOffset+scrollBuffer {
		m.marketplaceScrollOffset = m.marketplaceCursor - scrollBuffer
		if m.marketplaceScrollOffset < 0 {
			m.marketplaceScrollOffset = 0
		}
	}

	if m.marketplaceCursor >= m.marketplaceScrollOffset+maxVisible-scrollBuffer {
		m.marketplaceScrollOffset = m.marketplaceCursor - maxVisible + scrollBuffer + 1
		if m.marketplaceScrollOffset > len(m.marketplaceItems)-maxVisible {
			m.marketplaceScrollOffset = len(m.marketplaceItems) - maxVisible
		}
	}
}

// NextMarketplaceSort cycles to next sort mode
func (m *Model) NextMarketplaceSort() {
	m.marketplaceSortMode = (m.marketplaceSortMode + 1) % 4
	m.ApplyMarketplaceSort()
	m.marketplaceCursor = 0
	m.marketplaceScrollOffset = 0
}

// PrevMarketplaceSort cycles to previous sort mode
func (m *Model) PrevMarketplaceSort() {
	m.marketplaceSortMode = (m.marketplaceSortMode + 3) % 4
	m.ApplyMarketplaceSort()
	m.marketplaceCursor = 0
	m.marketplaceScrollOffset = 0
}
