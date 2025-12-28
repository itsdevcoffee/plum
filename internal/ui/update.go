package ui

import (
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/itsdevcoffee/plum/internal/marketplace"
)

func init() {
	// Set functions to avoid circular import
	clearCacheAndReload = marketplace.RefreshAll // Use RefreshAll to fetch from registry
	checkForNewMarketplaces = func() ([]PopularMarketplace, int, error) {
		updated, newCount, err := marketplace.FetchRegistryWithComparison(marketplace.PopularMarketplaces)
		// Convert marketplace.PopularMarketplace to ui.PopularMarketplace
		result := make([]PopularMarketplace, len(updated))
		for i, m := range updated {
			result[i] = PopularMarketplace{
				Name:        m.Name,
				DisplayName: m.DisplayName,
				Repo:        m.Repo,
				Description: m.Description,
			}
		}
		return result, newCount, err
	}
}

// animationTickMsg is sent to update animations
type animationTickMsg time.Time

// clearCopiedFlashMsg clears the "Copied!" indicator
type clearCopiedFlashMsg struct{}

// clearClipboardErrorMsg clears the "Clipboard error!" indicator
type clearClipboardErrorMsg struct{}

// clearCopiedFlash returns a command that clears the flash after a delay
func clearCopiedFlash() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return clearCopiedFlashMsg{}
	})
}

// clearClipboardError returns a command that clears the error flash after a delay
func clearClipboardError() tea.Cmd {
	return tea.Tick(time.Second*3, func(t time.Time) tea.Msg {
		return clearClipboardErrorMsg{}
	})
}

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
			m.refreshing = false
			return m, nil
		}
		m.allPlugins = msg.plugins
		m.results = m.filteredSearch(m.textInput.Value())
		m.loading = false
		m.refreshing = false
		// Initialize cursor animation to current position
		m.cursorY = 0
		m.targetCursorY = 0
		return m, nil

	case refreshCacheMsg:
		// Start refresh process
		m.refreshing = true
		m.newMarketplacesCount = 0 // Clear notification during refresh
		return m, tea.Batch(
			m.spinner.Tick,
			doRefreshCache,
		)

	case registryCheckedMsg:
		// Registry check completed - store new marketplace count and force re-render
		m.newMarketplacesCount = msg.newCount
		// Return a no-op command to force Bubble Tea to re-render the view
		return m, func() tea.Msg { return nil }

	case refreshProgressMsg:
		// Update refresh progress
		m.refreshProgress = msg.completed
		m.refreshTotal = msg.total
		m.refreshCurrent = msg.current
		return m, nil

	case spinner.TickMsg:
		if m.loading || m.refreshing {
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

	case clearCopiedFlashMsg:
		m.copiedFlash = false
		return m, nil

	case clearClipboardErrorMsg:
		m.clipboardErrorFlash = false
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

	case "tab", "right":
		m.NextFilter()
		return m, nil

	case "shift+tab", "left":
		m.PrevFilter()
		return m, nil

	case "ctrl+v":
		m.ToggleDisplayMode()
		return m, nil

	case "ctrl+t":
		m.CycleTransitionStyle()
		return m, nil

	case "shift+u", "U":
		// Refresh cache - clear and re-fetch all marketplace data
		return m, func() tea.Msg {
			return refreshCacheMsg{}
		}

	// Clear search, cancel refresh, or quit
	case "esc", "ctrl+g":
		// If refreshing, cancel the refresh
		if m.refreshing {
			m.refreshing = false
			m.refreshProgress = 0
			m.refreshTotal = 0
			m.refreshCurrent = ""
			return m, nil
		}
		// Otherwise clear search or quit
		if m.textInput.Value() != "" {
			m.textInput.SetValue("")
			m.results = m.filteredSearch("")
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

	// Re-run search on input change (with filter)
	m.results = m.filteredSearch(newValue)

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
		// Copy marketplace install command (for discoverable) or plugin install (for normal)
		if p := m.SelectedPlugin(); p != nil && !p.Installed {
			var copyText string
			if p.IsDiscoverable {
				// Copy marketplace add command for discoverable plugins
				copyText = fmt.Sprintf("/plugin marketplace add %s", p.MarketplaceSource)
			} else {
				// Copy plugin install command for normal plugins
				copyText = p.InstallCommand()
			}

			if err := clipboard.WriteAll(copyText); err == nil {
				m.copiedFlash = true
				return m, clearCopiedFlash()
			} else {
				// Show error to user instead of silently failing
				m.clipboardErrorFlash = true
				return m, clearClipboardError()
			}
		}
		return m, nil

	case "y":
		// Copy plugin install command (only for discoverable plugins)
		if p := m.SelectedPlugin(); p != nil && !p.Installed && p.IsDiscoverable {
			if err := clipboard.WriteAll(p.InstallCommand()); err == nil {
				m.copiedFlash = true
				return m, clearCopiedFlash()
			} else {
				// Show error to user instead of silently failing
				m.clipboardErrorFlash = true
				return m, clearClipboardError()
			}
		}
		return m, nil

	case "g":
		// Open plugin GitHub URL in browser
		if p := m.SelectedPlugin(); p != nil {
			url := p.GitHubURL()
			if url != "" {
				// Try to open in browser (xdg-open on Linux, open on macOS)
				cmd := "xdg-open"
				if runtime.GOOS == "darwin" {
					cmd = "open"
				}
				exec.Command(cmd, url).Start()
			}
		}
		return m, nil

	case "l":
		// Copy plugin GitHub URL to clipboard
		if p := m.SelectedPlugin(); p != nil {
			url := p.GitHubURL()
			if url != "" {
				if err := clipboard.WriteAll(url); err == nil {
					m.copiedFlash = true
					return m, clearCopiedFlash()
				} else {
					m.clipboardErrorFlash = true
					return m, clearClipboardError()
				}
			}
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
