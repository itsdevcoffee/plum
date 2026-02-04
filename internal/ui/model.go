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
	ViewQuickMenu         // Quick action menu overlay
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

// PluginSortMode represents how to sort plugin results
type PluginSortMode int

const (
	PluginSortRelevance PluginSortMode = iota // Default: search relevance
	PluginSortName                            // Alphabetical by plugin name
	PluginSortUpdated                         // Most recently updated first
	PluginSortStars                           // Most stars first (if available)
)

// PluginSortModeNames for display
var PluginSortModeNames = []string{"Relevance", "↑Name", "↑Updated", "↑Stars"}

// FacetType distinguishes between filter and sort facets
type FacetType int

const (
	FacetFilter FacetType = iota
	FacetSort
)

// Facet represents a unified filter or sort option
type Facet struct {
	Type            FacetType
	DisplayName     string
	FilterMode      FilterMode
	SortMode        PluginSortMode
	MarketplaceSort MarketplaceSortMode
	IsActive        bool
}

// Plugin list facets: All | Discover | Ready | Installed | ↑Name | ↑Updated | ↑Stars
func (m Model) GetPluginFacets() []Facet {
	facets := []Facet{
		// Filters (mutually exclusive)
		{Type: FacetFilter, DisplayName: "All", FilterMode: FilterAll, IsActive: m.filterMode == FilterAll},
		{Type: FacetFilter, DisplayName: "Discover", FilterMode: FilterDiscover, IsActive: m.filterMode == FilterDiscover},
		{Type: FacetFilter, DisplayName: "Ready", FilterMode: FilterReady, IsActive: m.filterMode == FilterReady},
		{Type: FacetFilter, DisplayName: "Installed", FilterMode: FilterInstalled, IsActive: m.filterMode == FilterInstalled},
		// Sorts (independently active)
		{Type: FacetSort, DisplayName: "↑Name", SortMode: PluginSortName, IsActive: m.pluginSortMode == PluginSortName},
		{Type: FacetSort, DisplayName: "↑Updated", SortMode: PluginSortUpdated, IsActive: m.pluginSortMode == PluginSortUpdated},
		{Type: FacetSort, DisplayName: "↑Stars", SortMode: PluginSortStars, IsActive: m.pluginSortMode == PluginSortStars},
	}
	return facets
}

// Marketplace list facets: All | Installed | Cached | ↑Plugins | ↑Stars | ↑Name | ↑Updated
func (m Model) GetMarketplaceFacets() []Facet {
	// For marketplace view, we currently don't have filter modes, so we'll add basic status filters
	facets := []Facet{
		// Sorts (marketplace currently only has sorts, but we'll keep structure consistent)
		{Type: FacetSort, DisplayName: "↑Plugins", MarketplaceSort: SortByPluginCount, IsActive: m.marketplaceSortMode == SortByPluginCount},
		{Type: FacetSort, DisplayName: "↑Stars", MarketplaceSort: SortByStars, IsActive: m.marketplaceSortMode == SortByStars},
		{Type: FacetSort, DisplayName: "↑Name", MarketplaceSort: SortByName, IsActive: m.marketplaceSortMode == SortByName},
		{Type: FacetSort, DisplayName: "↑Updated", MarketplaceSort: SortByLastUpdated, IsActive: m.marketplaceSortMode == SortByLastUpdated},
	}
	return facets
}

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

// Model is the main Bubble Tea application model for Plum TUI.
// It manages all UI state including plugins, search results, viewports,
// and marketplace data. Thread-safe for use in Bubble Tea's Update() loop.
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
	pluginSortMode      PluginSortMode
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

	// Marketplace autocomplete state (for @marketplace-name filtering)
	marketplaceAutocompleteActive bool              // True when showing marketplace picker
	marketplaceAutocompleteList   []MarketplaceItem // Filtered marketplaces for autocomplete
	marketplaceAutocompleteCursor int               // Selected index in autocomplete list

	// Quick menu state
	quickMenuActive       bool      // True when quick menu overlay is shown
	quickMenuCursor       int       // Selected action index in quick menu
	quickMenuPreviousView ViewState // View to return to when closing menu

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
	ti.Placeholder = "Search plugins (or @marketplace-name to filter)..."
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

	if m.cursor < m.scrollOffset+scrollBuffer {
		m.scrollOffset = maxInt(m.cursor-scrollBuffer, 0)
		return
	}

	if m.cursor >= m.scrollOffset+maxVisible-scrollBuffer {
		m.scrollOffset = minInt(m.cursor-maxVisible+scrollBuffer+1, len(m.results)-maxVisible)
		m.scrollOffset = maxInt(m.scrollOffset, 0)
	}
}

// minInt returns the smaller of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// maxInt returns the larger of two integers
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// clearSearch clears the search input and resets results
func (m *Model) clearSearch() {
	m.textInput.SetValue("")
	m.results = m.filteredSearch("")
	m.cursor = 0
	m.scrollOffset = 0
	m.SnapCursorToTarget()
}

// cancelRefresh cancels an ongoing refresh operation
func (m *Model) cancelRefresh() {
	m.refreshing = false
	m.refreshProgress = 0
	m.refreshTotal = 0
	m.refreshCurrent = ""
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

// NextFacet cycles to the next facet (unified filter/sort)
func (m *Model) NextFacet() {
	m.cycleFacet(1)
}

// PrevFacet cycles to the previous facet (unified filter/sort)
func (m *Model) PrevFacet() {
	m.cycleFacet(-1)
}

// cycleFacet handles facet navigation in either direction
func (m *Model) cycleFacet(direction int) {
	facets := m.GetPluginFacets()
	if len(facets) == 0 {
		return
	}

	currentIdx := m.findActiveFacetIndex(facets)
	if currentIdx == -1 {
		currentIdx = 0
		if direction < 0 {
			currentIdx = len(facets) - 1
		}
	}

	nextIdx := (currentIdx + direction + len(facets)) % len(facets)
	m.applyFacet(facets[nextIdx])
}

// findActiveFacetIndex returns the index of the currently active facet
func (m Model) findActiveFacetIndex(facets []Facet) int {
	for i, f := range facets {
		if !f.IsActive {
			continue
		}
		if f.Type == FacetSort && f.SortMode == m.pluginSortMode {
			return i
		}
		if f.Type == FacetFilter && f.FilterMode == m.filterMode {
			return i
		}
	}
	return -1
}

// applyFacet applies the given facet to the model
func (m *Model) applyFacet(facet Facet) {
	if facet.Type == FacetFilter {
		m.filterMode = facet.FilterMode
		m.applyFilter()
	} else {
		m.pluginSortMode = facet.SortMode
		m.applySortAndFilter()
	}
}

// applySortAndFilter applies both filter and sort to current results
func (m *Model) applySortAndFilter() {
	// Re-run search with current filter
	m.results = m.filteredSearch(m.textInput.Value())

	// Apply sort mode
	m.applyPluginSort()

	// Reset cursor
	m.cursor = 0
	m.scrollOffset = 0
	m.SnapCursorToTarget()
}

// applyPluginSort sorts the current results based on pluginSortMode
func (m *Model) applyPluginSort() {
	if m.pluginSortMode == PluginSortRelevance {
		return
	}

	sort.Slice(m.results, func(i, j int) bool {
		return m.results[i].Plugin.Name < m.results[j].Plugin.Name
	})
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
	if strings.HasPrefix(query, "@") {
		return m.marketplaceFilteredSearch(query)
	}

	allResults := search.Search(query, m.allPlugins)
	return m.applyFilterMode(allResults)
}

// marketplaceFilteredSearch handles @marketplace-name filtering
func (m Model) marketplaceFilteredSearch(query string) []search.RankedPlugin {
	parts := strings.SplitN(query[1:], " ", 2)
	marketplaceName := parts[0]
	searchTerms := ""
	if len(parts) > 1 {
		searchTerms = parts[1]
	}

	var marketplacePlugins []plugin.Plugin
	for _, p := range m.allPlugins {
		if p.Marketplace == marketplaceName {
			marketplacePlugins = append(marketplacePlugins, p)
		}
	}

	if searchTerms != "" {
		return search.Search(searchTerms, marketplacePlugins)
	}

	filtered := make([]search.RankedPlugin, 0, len(marketplacePlugins))
	for _, p := range marketplacePlugins {
		filtered = append(filtered, search.RankedPlugin{Plugin: p, Score: 1.0})
	}
	return filtered
}

// applyFilterMode filters results based on the current filter mode
func (m Model) applyFilterMode(results []search.RankedPlugin) []search.RankedPlugin {
	if m.filterMode == FilterAll {
		return results
	}

	filtered := make([]search.RankedPlugin, 0)
	for _, r := range results {
		if m.matchesFilterMode(r.Plugin) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// matchesFilterMode checks if a plugin matches the current filter mode
func (m Model) matchesFilterMode(p plugin.Plugin) bool {
	switch m.filterMode {
	case FilterDiscover:
		return p.IsDiscoverable
	case FilterReady:
		return !p.Installed && !p.IsDiscoverable
	case FilterInstalled:
		return p.Installed
	default:
		return true
	}
}

// FilterModeName returns the current filter mode name
func (m Model) FilterModeName() string {
	return FilterModeNames[m.filterMode]
}

// getDynamicFilterCounts calculates counts for each filter mode based on current search query
func (m Model) getDynamicFilterCounts(query string) map[FilterMode]int {
	counts := make(map[FilterMode]int)

	// For each filter mode, calculate how many results we'd get
	for _, mode := range []FilterMode{FilterAll, FilterDiscover, FilterReady, FilterInstalled} {
		// Temporarily set filter mode and get results
		tempModel := m
		tempModel.filterMode = mode
		results := tempModel.filteredSearch(query)
		counts[mode] = len(results)
	}

	return counts
}

// ReadyCount returns count of ready-to-install plugins
func (m Model) ReadyCount() int {
	return m.countPlugins(func(p plugin.Plugin) bool {
		return !p.Installed && !p.IsDiscoverable
	})
}

// DiscoverableCount returns count of discoverable plugins
func (m Model) DiscoverableCount() int {
	return m.countPlugins(func(p plugin.Plugin) bool {
		return p.IsDiscoverable
	})
}

// InstalledCount returns count of installed plugins
func (m Model) InstalledCount() int {
	return m.countPlugins(func(p plugin.Plugin) bool {
		return p.Installed
	})
}

// TotalPlugins returns total plugin count
func (m Model) TotalPlugins() int {
	return len(m.allPlugins)
}

func (m Model) countPlugins(predicate func(plugin.Plugin) bool) int {
	count := 0
	for _, p := range m.allPlugins {
		if predicate(p) {
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
			parts := strings.SplitN(fullName, "@", 2)
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

		// Try to get total plugin count from cached manifest OR local installation
		if cached, _ := marketplace.LoadFromCache(pm.Name); cached != nil {
			item.TotalPluginCount = len(cached.Plugins)
			if item.Status == MarketplaceAvailable {
				item.Status = MarketplaceCached
			}
		} else if entry, isInstalled := knownMarketplaces[pm.Name]; isInstalled {
			// Marketplace is installed locally - try to load from installation
			if localManifest, err := config.LoadMarketplaceManifest(entry.InstallLocation); err == nil {
				item.TotalPluginCount = len(localManifest.Plugins)
			}
		}

		// Load GitHub stats: prefer cache, fallback to static stats
		if stats, err := marketplace.LoadStatsFromCache(pm.Name); err == nil && stats != nil {
			item.GitHubStats = stats
		} else {
			// Fallback to static stats from PopularMarketplaces (by name lookup)
			item.GitHubStats = getStaticStatsByName(pm.Name)
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
		sort.Slice(items, func(i, j int) bool {
			return items[i].TotalPluginCount > items[j].TotalPluginCount
		})
	case SortByStars:
		sort.Slice(items, func(i, j int) bool {
			return getStars(items[i]) > getStars(items[j])
		})
	case SortByName:
		sort.Slice(items, func(i, j int) bool {
			return items[i].DisplayName < items[j].DisplayName
		})
	case SortByLastUpdated:
		sort.Slice(items, func(i, j int) bool {
			return getLastPushed(items[i]).After(getLastPushed(items[j]))
		})
	}

	m.marketplaceItems = items
}

// getStars safely extracts star count from marketplace item
func getStars(item MarketplaceItem) int {
	if item.GitHubStats != nil {
		return item.GitHubStats.Stars
	}
	return 0
}

// getLastPushed safely extracts last pushed time from marketplace item
func getLastPushed(item MarketplaceItem) time.Time {
	if item.GitHubStats != nil {
		return item.GitHubStats.LastPushedAt
	}
	return time.Time{}
}

// getStaticStatsByName looks up static stats from PopularMarketplaces by name
func getStaticStatsByName(name string) *marketplace.GitHubStats {
	for _, pm := range marketplace.PopularMarketplaces {
		if pm.Name == name && pm.StaticStats != nil {
			return pm.StaticStats
		}
	}
	return nil
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

	if m.marketplaceCursor < m.marketplaceScrollOffset+scrollBuffer {
		m.marketplaceScrollOffset = m.marketplaceCursor - scrollBuffer
		if m.marketplaceScrollOffset < 0 {
			m.marketplaceScrollOffset = 0
		}
		return
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

// NextMarketplaceFacet cycles to next facet in marketplace view
func (m *Model) NextMarketplaceFacet() {
	m.cycleMarketplaceFacet(1)
}

// PrevMarketplaceFacet cycles to previous facet in marketplace view
func (m *Model) PrevMarketplaceFacet() {
	m.cycleMarketplaceFacet(-1)
}

// cycleMarketplaceFacet handles marketplace facet navigation
func (m *Model) cycleMarketplaceFacet(direction int) {
	facets := m.GetMarketplaceFacets()
	if len(facets) == 0 {
		return
	}

	currentIdx := m.findActiveMarketplaceFacetIndex(facets)
	if currentIdx == -1 {
		currentIdx = 0
		if direction < 0 {
			currentIdx = len(facets) - 1
		}
	}

	nextIdx := (currentIdx + direction + len(facets)) % len(facets)
	m.applyMarketplaceFacet(facets[nextIdx])
}

// findActiveMarketplaceFacetIndex returns the index of the active marketplace facet
func (m Model) findActiveMarketplaceFacetIndex(facets []Facet) int {
	for i, f := range facets {
		if f.IsActive && f.MarketplaceSort == m.marketplaceSortMode {
			return i
		}
	}
	return -1
}

// applyMarketplaceFacet applies a marketplace facet and resets cursor
func (m *Model) applyMarketplaceFacet(facet Facet) {
	m.marketplaceSortMode = facet.MarketplaceSort
	m.ApplyMarketplaceSort()
	m.marketplaceCursor = 0
	m.marketplaceScrollOffset = 0
}

// UpdateMarketplaceAutocomplete updates the marketplace autocomplete list based on query
func (m *Model) UpdateMarketplaceAutocomplete(query string) {
	// Extract marketplace filter part (everything after @ until first space)
	if !strings.HasPrefix(query, "@") {
		m.marketplaceAutocompleteActive = false
		return
	}

	// Find first space to separate marketplace name from search terms
	parts := strings.SplitN(query[1:], " ", 2)
	marketplaceFilter := parts[0]

	// If there's a space (even if empty search after), exit autocomplete mode
	// This handles both "@marketplace search" and "@marketplace " (trailing space)
	if len(parts) > 1 {
		m.marketplaceAutocompleteActive = false
		return
	}

	// Lazy-load marketplace items if not already loaded
	if len(m.marketplaceItems) == 0 {
		_ = m.LoadMarketplaceItems()
	}

	// We're in autocomplete mode - filter marketplaces
	m.marketplaceAutocompleteActive = true
	m.marketplaceAutocompleteList = []MarketplaceItem{}

	for _, item := range m.marketplaceItems {
		// Match on marketplace name (case-insensitive)
		if marketplaceFilter == "" || strings.Contains(strings.ToLower(item.Name), strings.ToLower(marketplaceFilter)) {
			m.marketplaceAutocompleteList = append(m.marketplaceAutocompleteList, item)
		}
	}

	// Reset cursor if out of bounds
	if m.marketplaceAutocompleteCursor >= len(m.marketplaceAutocompleteList) {
		m.marketplaceAutocompleteCursor = 0
	}
}

// SelectMarketplaceAutocomplete completes the marketplace name in the search box
func (m *Model) SelectMarketplaceAutocomplete() {
	if !m.marketplaceAutocompleteActive || len(m.marketplaceAutocompleteList) == 0 {
		return
	}

	selected := m.marketplaceAutocompleteList[m.marketplaceAutocompleteCursor]
	m.textInput.SetValue("@" + selected.Name + " ")
	m.marketplaceAutocompleteActive = false
	m.marketplaceAutocompleteCursor = 0

	// Move cursor to end
	m.textInput.SetCursor(len(m.textInput.Value()))
}
