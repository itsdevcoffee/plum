package ui

import (
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/maskkiller/plum/internal/search"
)

// animationTickMsg is sent to update animations
type animationTickMsg time.Time

// animationTick returns a command that ticks the animation
func animationTick() tea.Cmd {
	return tea.Tick(time.Second/animationFPS, func(t time.Time) tea.Msg {
		return animationTickMsg(t)
	})
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		m.textInput.Width = msg.Width - 10
		return m, nil

	case pluginsLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.loading = false
			return m, nil
		}
		m.allPlugins = msg.plugins
		m.results = search.Search(m.textInput.Value(), m.allPlugins)
		m.loading = false
		// Initialize cursor animation to current position
		m.cursorY = 0
		m.targetCursorY = 0
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case animationTickMsg:
		// Update all animations
		m.UpdateCursorAnimation()
		m.UpdateViewTransition()

		// Continue ticking if any animation is active
		if m.IsAnimating() || m.IsViewTransitioning() {
			return m, animationTick()
		}
		return m, nil
	}

	return m, nil
}

// handleKeyMsg handles keyboard input
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	}

	// View-specific keys
	switch m.viewState {
	case ViewList:
		return m.handleListKeys(msg)
	case ViewDetail:
		return m.handleDetailKeys(msg)
	case ViewHelp:
		return m.handleHelpKeys(msg)
	}

	return m, nil
}

// handleListKeys handles keys in the list view
// Uses telescope/fzf pattern: Ctrl+key for navigation, typing goes to search
func (m Model) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	// Navigation: Ctrl + j/k/n/p or arrow keys
	case "up", "ctrl+k", "ctrl+p":
		if m.cursor > 0 {
			m.cursor--
		}
		m.UpdateScroll()
		m.SetCursorTarget()
		return m, animationTick()

	case "down", "ctrl+j", "ctrl+n":
		if m.cursor < len(m.results)-1 {
			m.cursor++
		}
		m.UpdateScroll()
		m.SetCursorTarget()
		return m, animationTick()

	// Page navigation
	case "pgup", "ctrl+u":
		m.cursor -= m.maxVisibleItems()
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.UpdateScroll()
		m.SetCursorTarget()
		return m, animationTick()

	case "pgdown", "ctrl+d":
		m.cursor += m.maxVisibleItems()
		if m.cursor >= len(m.results) {
			m.cursor = len(m.results) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.UpdateScroll()
		m.SetCursorTarget()
		return m, animationTick()

	// Jump to start/end
	case "home":
		m.cursor = 0
		m.scrollOffset = 0
		m.SetCursorTarget()
		return m, animationTick()

	case "end":
		if len(m.results) > 0 {
			m.cursor = len(m.results) - 1
		}
		m.UpdateScroll()
		m.SetCursorTarget()
		return m, animationTick()

	// Actions
	case "enter":
		if len(m.results) > 0 {
			m.StartViewTransition(ViewDetail, 1) // Forward transition
			return m, animationTick()
		}
		return m, nil

	case "?":
		m.StartViewTransition(ViewHelp, 1) // Forward transition
		return m, animationTick()

	// Clear search or quit
	case "esc", "ctrl+g":
		if m.textInput.Value() != "" {
			m.textInput.SetValue("")
			m.results = search.Search("", m.allPlugins)
			m.cursor = 0
			m.scrollOffset = 0
			m.SnapCursorToTarget()
		} else {
			return m, tea.Quit
		}
		return m, nil
	}

	// All other keys go to text input (typing)
	var cmd tea.Cmd
	oldValue := m.textInput.Value()
	m.textInput, cmd = m.textInput.Update(msg)
	newValue := m.textInput.Value()

	// Re-run search on input change
	m.results = search.Search(newValue, m.allPlugins)

	// Reset cursor to top on any search input change
	if newValue != oldValue {
		m.cursor = 0
		m.scrollOffset = 0
		m.SnapCursorToTarget()
	} else if m.cursor >= len(m.results) {
		// Clamp cursor if somehow out of bounds
		m.cursor = len(m.results) - 1
		if m.cursor < 0 {
			m.cursor = 0
		}
	}

	return m, cmd
}

// handleDetailKeys handles keys in the detail view
func (m Model) handleDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "esc", "backspace":
		m.StartViewTransition(ViewList, -1) // Back transition
		return m, animationTick()

	case "c":
		// Copy install command to clipboard
		if p := m.SelectedPlugin(); p != nil && !p.Installed {
			_ = clipboard.WriteAll(p.InstallCommand())
		}
		return m, nil

	case "?":
		m.StartViewTransition(ViewHelp, 1) // Forward transition
		return m, animationTick()
	}

	return m, nil
}

// handleHelpKeys handles keys in the help view
func (m Model) handleHelpKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "esc", "?", "backspace", "enter":
		m.StartViewTransition(ViewList, -1) // Back transition
		return m, animationTick()
	}

	return m, nil
}
