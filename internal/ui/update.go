package ui

import (
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// clearLinkCopiedFlashMsg clears the "Link Copied!" indicator
type clearLinkCopiedFlashMsg struct{}

// clearPathCopiedFlashMsg clears the "Path Copied!" indicator
type clearPathCopiedFlashMsg struct{}

// clearGithubOpenedFlashMsg clears the "Opened!" indicator for GitHub
type clearGithubOpenedFlashMsg struct{}

// clearLocalOpenedFlashMsg clears the "Opened!" indicator for local
type clearLocalOpenedFlashMsg struct{}

// clearClipboardErrorMsg clears the "Clipboard error!" indicator
type clearClipboardErrorMsg struct{}

// clearCopiedFlash returns a command that clears the flash after a delay
func clearCopiedFlash() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return clearCopiedFlashMsg{}
	})
}

// clearLinkCopiedFlash returns a command that clears the flash after a delay
func clearLinkCopiedFlash() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return clearLinkCopiedFlashMsg{}
	})
}

// clearPathCopiedFlash returns a command that clears the flash after a delay
func clearPathCopiedFlash() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return clearPathCopiedFlashMsg{}
	})
}

// clearGithubOpenedFlash returns a command that clears the flash after a delay
func clearGithubOpenedFlash() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return clearGithubOpenedFlashMsg{}
	})
}

// clearLocalOpenedFlash returns a command that clears the flash after a delay
func clearLocalOpenedFlash() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return clearLocalOpenedFlashMsg{}
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

		// Initialize/update help viewport
		viewportWidth := 58

		if m.helpViewport.Width == 0 {
			// Initial creation
			viewportHeight := msg.Height - 9
			if viewportHeight < 5 {
				viewportHeight = 5
			}
			m.helpViewport = viewport.New(viewportWidth, viewportHeight)
		} else {
			// Always update width
			m.helpViewport.Width = viewportWidth

			// If in help view, recalculate height based on content + terminal size
			if m.viewState == ViewHelp {
				sectionsContent := m.generateHelpSections()
				contentHeight := lipgloss.Height(sectionsContent)
				maxHeight := msg.Height - 9

				if maxHeight < 3 {
					maxHeight = 3
				}

				// Resize viewport to fit
				if contentHeight < maxHeight {
					m.helpViewport.Height = contentHeight
				} else {
					m.helpViewport.Height = maxHeight
				}

				// Re-set content to update wrapping
				m.helpViewport.SetContent(sectionsContent)
			}
		}

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

	default:
		// Update viewport if in help view (handles smooth scrolling)
		if m.viewState == ViewHelp && m.helpViewport.Height > 0 {
			var cmd tea.Cmd
			m.helpViewport, cmd = m.helpViewport.Update(msg)
			return m, cmd
		}

	case clearCopiedFlashMsg:
		m.copiedFlash = false
		return m, nil

	case clearLinkCopiedFlashMsg:
		m.linkCopiedFlash = false
		return m, nil

	case clearPathCopiedFlashMsg:
		m.pathCopiedFlash = false
		return m, nil

	case clearGithubOpenedFlashMsg:
		m.githubOpenedFlash = false
		return m, nil

	case clearLocalOpenedFlashMsg:
		m.localOpenedFlash = false
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
	case ViewMarketplaceList:
		return m.handleMarketplaceListKeys(msg)
	case ViewMarketplaceDetail:
		return m.handleMarketplaceDetailKeys(msg)
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
		// Set help SECTIONS content in viewport (not header/footer)
		if m.helpViewport.Width > 0 {
			sectionsContent := m.generateHelpSections()

			// Calculate fixed overhead heights
			headerHeight := 3 // Title + divider
			footerHeight := 2 // Divider + text
			boxPadding := 4   // Box padding top/bottom (2) + borders (2)

			// Available height for viewport = terminal - all overhead
			maxHeight := m.windowHeight - headerHeight - footerHeight - boxPadding

			if maxHeight < 3 {
				maxHeight = 3 // Absolute minimum
			}

			// Calculate actual content height
			contentHeight := lipgloss.Height(sectionsContent)

			// Use smaller of content or available space
			if contentHeight < maxHeight {
				m.helpViewport.Height = contentHeight
			} else {
				m.helpViewport.Height = maxHeight
			}

			m.helpViewport.SetContent(sectionsContent)
			m.helpViewport.GotoTop()
		}
		m.StartViewTransition(ViewHelp, 1)
		return m, animationTick()

	case "tab", "right":
		m.NextFilter()
		return m, nil

	case "shift+tab", "left":
		m.PrevFilter()
		return m, nil

	case "shift+v", "V":
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

	case "shift+m", "M":
		// Open marketplace browser
		_ = m.LoadMarketplaceItems()
		m.previousViewBeforeMarketplace = ViewList
		m.StartViewTransition(ViewMarketplaceList, 1)
		// TODO: Start background GitHub stats loading
		return m, animationTick()

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
				// Open in browser (cross-platform)
				var cmd string
				var args []string
				switch runtime.GOOS {
				case "darwin":
					cmd = "open"
					args = []string{url}
				case "windows":
					cmd = "cmd"
					args = []string{"/c", "start", url}
				default: // linux, bsd, etc.
					cmd = "xdg-open"
					args = []string{url}
				}
				_ = exec.Command(cmd, args...).Start()
				m.githubOpenedFlash = true
				return m, clearGithubOpenedFlash()
			}
		}
		return m, nil

	case "l":
		// Copy plugin GitHub URL to clipboard
		if p := m.SelectedPlugin(); p != nil {
			url := p.GitHubURL()
			if url != "" {
				if err := clipboard.WriteAll(url); err == nil {
					m.linkCopiedFlash = true
					return m, clearLinkCopiedFlash()
				} else {
					m.clipboardErrorFlash = true
					return m, clearClipboardError()
				}
			}
		}
		return m, nil

	case "o":
		// Open local install directory (only for installed plugins)
		if p := m.SelectedPlugin(); p != nil && p.Installed && p.InstallPath != "" {
			// Open in file manager (cross-platform)
			var cmd string
			var args []string
			switch runtime.GOOS {
			case "darwin":
				cmd = "open"
				args = []string{p.InstallPath}
			case "windows":
				cmd = "explorer"
				args = []string{p.InstallPath}
			default: // linux, bsd, etc.
				cmd = "xdg-open"
				args = []string{p.InstallPath}
			}
			// #nosec G204 -- cmd is determined by runtime.GOOS (trusted), args is install path from config (validated)
			_ = exec.Command(cmd, args...).Start()
			m.localOpenedFlash = true
			return m, clearLocalOpenedFlash()
		}
		return m, nil

	case "p":
		// Copy local install path to clipboard (only for installed plugins)
		if p := m.SelectedPlugin(); p != nil && p.Installed && p.InstallPath != "" {
			if err := clipboard.WriteAll(p.InstallPath); err == nil {
				m.pathCopiedFlash = true
				return m, clearPathCopiedFlash()
			} else {
				m.clipboardErrorFlash = true
				return m, clearClipboardError()
			}
		}
		return m, nil

	case "shift+m", "M":
		// Open marketplace browser
		_ = m.LoadMarketplaceItems()
		m.previousViewBeforeMarketplace = ViewDetail
		m.StartViewTransition(ViewMarketplaceList, 1)
		// TODO: Start background GitHub stats loading
		return m, animationTick()

	case "?":
		m.StartViewTransition(ViewHelp, 1) // Forward transition
		return m, animationTick()
	}

	return m, nil
}

// handleHelpKeys handles keys in the help view
func (m Model) handleHelpKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "shift+m", "M":
		// Open marketplace browser
		_ = m.LoadMarketplaceItems()
		m.previousViewBeforeMarketplace = ViewHelp
		m.StartViewTransition(ViewMarketplaceList, 1)
		// TODO: Start background GitHub stats loading
		return m, animationTick()

	case "esc", "?", "backspace", "enter":
		m.StartViewTransition(ViewList, -1) // Back transition
		return m, animationTick()

	default:
		// Pass other keys to viewport for scrolling
		m.helpViewport, cmd = m.helpViewport.Update(msg)
		return m, cmd
	}
}

// handleMarketplaceListKeys handles keys in the marketplace list view
func (m Model) handleMarketplaceListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "ctrl+k", "ctrl+p":
		if m.marketplaceCursor > 0 {
			m.marketplaceCursor--
		}
		m.UpdateMarketplaceScroll()
		return m, nil

	case "down", "ctrl+j", "ctrl+n":
		if m.marketplaceCursor < len(m.marketplaceItems)-1 {
			m.marketplaceCursor++
		}
		m.UpdateMarketplaceScroll()
		return m, nil

	case "enter":
		if len(m.marketplaceItems) > 0 && m.marketplaceCursor < len(m.marketplaceItems) {
			m.selectedMarketplace = &m.marketplaceItems[m.marketplaceCursor]
			m.StartViewTransition(ViewMarketplaceDetail, 1)
			return m, animationTick()
		}
		return m, nil

	case "tab", "right":
		m.NextMarketplaceSort()
		return m, nil

	case "shift+tab", "left":
		m.PrevMarketplaceSort()
		return m, nil

	case "esc", "ctrl+g":
		// Return to plugin list view
		m.StartViewTransition(ViewList, -1)
		return m, animationTick()

	case "?":
		m.StartViewTransition(ViewHelp, 1)
		return m, animationTick()

	case "q":
		return m, tea.Quit
	}

	return m, nil
}

// handleMarketplaceDetailKeys handles keys in the marketplace detail view
func (m Model) handleMarketplaceDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
		m.StartViewTransition(ViewMarketplaceList, -1)
		return m, animationTick()

	case "c":
		// Copy marketplace install command
		if m.selectedMarketplace != nil &&
			m.selectedMarketplace.Status != MarketplaceInstalled {
			installCmd := fmt.Sprintf("/plugin marketplace add %s",
				extractMarketplaceSource(m.selectedMarketplace.Repo))
			if err := clipboard.WriteAll(installCmd); err == nil {
				m.copiedFlash = true
				return m, clearCopiedFlash()
			} else {
				m.clipboardErrorFlash = true
				return m, clearClipboardError()
			}
		}
		return m, nil

	case "f":
		// Filter plugins by this marketplace
		m.previousViewBeforeMarketplace = ViewList
		m.StartViewTransition(ViewList, -1)
		// Set search to filter by marketplace
		m.textInput.SetValue("@" + m.selectedMarketplace.Name)
		m.results = m.filteredSearch(m.textInput.Value())
		m.cursor = 0
		m.scrollOffset = 0
		return m, animationTick()

	case "g":
		// Open GitHub repo
		if m.selectedMarketplace != nil {
			url := m.selectedMarketplace.Repo
			var cmd string
			var args []string
			switch runtime.GOOS {
			case "darwin":
				cmd = "open"
				args = []string{url}
			case "windows":
				cmd = "cmd"
				args = []string{"/c", "start", url}
			default:
				cmd = "xdg-open"
				args = []string{url}
			}
			// #nosec G204 -- cmd is determined by runtime.GOOS (trusted), args is marketplace repo URL (from registry)
			_ = exec.Command(cmd, args...).Start()
			m.githubOpenedFlash = true
			return m, clearGithubOpenedFlash()
		}
		return m, nil

	case "?":
		m.StartViewTransition(ViewHelp, 1)
		return m, animationTick()

	case "q":
		return m, tea.Quit
	}

	return m, nil
}
