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
		if m.IsAnimating() {
			m.UpdateCursorAnimation()
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
func (m Model) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		m.UpdateScroll()
		return m, animationTick()

	case "down", "j":
		if m.cursor < len(m.results)-1 {
			m.cursor++
		}
		m.UpdateScroll()
		return m, animationTick()

	case "enter":
		if len(m.results) > 0 {
			m.viewState = ViewDetail
		}
		return m, nil

	case "?":
		m.viewState = ViewHelp
		return m, nil

	case "esc":
		if m.textInput.Value() != "" {
			m.textInput.SetValue("")
			m.results = search.Search("", m.allPlugins)
			m.cursor = 0
			m.scrollOffset = 0
			m.cursorY = 0
			m.targetCursorY = 0
		}
		return m, nil

	case "home", "g":
		m.cursor = 0
		m.scrollOffset = 0
		return m, animationTick()

	case "end", "G":
		if len(m.results) > 0 {
			m.cursor = len(m.results) - 1
		}
		m.UpdateScroll()
		return m, animationTick()

	case "pgup", "ctrl+u":
		m.cursor -= m.maxVisibleItems()
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.UpdateScroll()
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
		return m, animationTick()
	}

	// Pass other keys to text input
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)

	// Re-run search
	m.results = search.Search(m.textInput.Value(), m.allPlugins)
	if m.cursor >= len(m.results) {
		m.cursor = len(m.results) - 1
		if m.cursor < 0 {
			m.cursor = 0
		}
	}
	m.scrollOffset = 0 // Reset scroll on search change
	m.cursorY = 0
	m.targetCursorY = 0

	return m, cmd
}

// handleDetailKeys handles keys in the detail view
func (m Model) handleDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "esc", "backspace":
		m.viewState = ViewList
		return m, nil

	case "c":
		// Copy install command to clipboard
		if p := m.SelectedPlugin(); p != nil && !p.Installed {
			_ = clipboard.WriteAll(p.InstallCommand())
		}
		return m, nil

	case "?":
		m.viewState = ViewHelp
		return m, nil
	}

	return m, nil
}

// handleHelpKeys handles keys in the help view
func (m Model) handleHelpKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "esc", "?", "backspace", "enter":
		m.viewState = ViewList
		return m, nil
	}

	return m, nil
}
