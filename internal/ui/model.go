package ui

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"
	"github.com/charmbracelet/lipgloss"
	"github.com/maskkiller/plum/internal/config"
	"github.com/maskkiller/plum/internal/plugin"
	"github.com/maskkiller/plum/internal/search"
)

// ViewState represents the current view
type ViewState int

const (
	ViewList ViewState = iota
	ViewDetail
	ViewHelp
)

// Scroll buffer - cursor stays this many items from edge before scrolling
const scrollBuffer = 2

// Animation constants
const (
	animationFPS      = 60
	springFrequency   = 7.0  // Higher = faster
	springDamping     = 0.8  // < 1 = bouncy, 1 = smooth, > 1 = slow
)

// Model is the main application model
type Model struct {
	// Data
	allPlugins []plugin.Plugin
	results    []search.RankedPlugin
	loading    bool

	// UI state
	textInput    textinput.Model
	spinner      spinner.Model
	cursor       int
	scrollOffset int
	viewState    ViewState
	windowWidth  int
	windowHeight int

	// Animation state
	cursorY         float64 // Animated cursor position
	cursorYVelocity float64
	targetCursorY   float64
	spring          harmonica.Spring

	// View transition state
	transitionProgress    float64   // 0.0 = old view, 1.0 = new view
	transitionVelocity    float64
	targetTransition      float64
	previousView          ViewState // View we're transitioning FROM
	transitionDirection   int       // 1 = forward (right to left), -1 = back (left to right)

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
	s.Style = lipgloss.NewStyle().Foreground(Peach)

	// Initialize spring for animations
	spring := harmonica.NewSpring(harmonica.FPS(animationFPS), springFrequency, springDamping)

	return Model{
		textInput:          ti,
		spinner:            s,
		spring:             spring,
		loading:            true,
		viewState:          ViewList,
		previousView:       ViewList,
		transitionProgress: 1.0, // Start fully transitioned (no animation on init)
		targetTransition:   1.0,
		windowWidth:        80,
		windowHeight:       24,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.spinner.Tick,
		loadPlugins,
	)
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
	// Account for title, search input, status bar, padding
	available := m.windowHeight - 8
	// Each item takes 2 lines (name + description)
	return available / 2
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
