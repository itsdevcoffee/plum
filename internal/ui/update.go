package ui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
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

func clearCopiedFlash() tea.Cmd {
	return clearFlashAfter(2*time.Second, clearCopiedFlashMsg{})
}

func clearLinkCopiedFlash() tea.Cmd {
	return clearFlashAfter(2*time.Second, clearLinkCopiedFlashMsg{})
}

func clearPathCopiedFlash() tea.Cmd {
	return clearFlashAfter(2*time.Second, clearPathCopiedFlashMsg{})
}

func clearGithubOpenedFlash() tea.Cmd {
	return clearFlashAfter(2*time.Second, clearGithubOpenedFlashMsg{})
}

func clearLocalOpenedFlash() tea.Cmd {
	return clearFlashAfter(2*time.Second, clearLocalOpenedFlashMsg{})
}

func clearClipboardError() tea.Cmd {
	return clearFlashAfter(3*time.Second, clearClipboardErrorMsg{})
}

func clearFlashAfter(duration time.Duration, msg tea.Msg) tea.Cmd {
	return tea.Tick(duration, func(t time.Time) tea.Msg {
		return msg
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

	case tea.MouseMsg:
		// Pass mouse events to viewports for scroll wheel support
		var cmd tea.Cmd
		if m.viewState == ViewHelp && m.helpViewport.Height > 0 {
			m.helpViewport, cmd = m.helpViewport.Update(msg)
			return m, cmd
		}
		if m.viewState == ViewDetail && m.detailViewport.Height > 0 {
			m.detailViewport, cmd = m.detailViewport.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		m.textInput.Width = msg.Width - 10

		(&m).initOrUpdateHelpViewport(msg.Height)
		(&m).initOrUpdateDetailViewport(msg.Height)

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

func (m *Model) initOrUpdateHelpViewport(terminalHeight int) {
	const viewportWidth = 58
	const overhead = 9

	if m.helpViewport.Width == 0 {
		viewportHeight := terminalHeight - overhead
		if viewportHeight < 3 {
			viewportHeight = 3
		}
		if viewportHeight > terminalHeight-4 {
			viewportHeight = terminalHeight - 4
		}
		m.helpViewport = viewport.New(viewportWidth, viewportHeight)
		return
	}

	m.helpViewport.Width = viewportWidth

	if m.viewState == ViewHelp {
		sectionsContent := m.generateHelpSections()
		contentHeight := lipgloss.Height(sectionsContent)
		maxHeight := terminalHeight - overhead
		if maxHeight < 3 {
			maxHeight = 3
		}

		if contentHeight < maxHeight {
			m.helpViewport.Height = contentHeight
		} else {
			m.helpViewport.Height = maxHeight
		}

		m.helpViewport.SetContent(sectionsContent)
	}
}

func (m *Model) initOrUpdateDetailViewport(terminalHeight int) {
	const overhead = 9
	const minWidth = 40

	detailViewportWidth := m.ContentWidth() - 10
	if detailViewportWidth < minWidth {
		detailViewportWidth = minWidth
	}

	if m.detailViewport.Width == 0 {
		viewportHeight := terminalHeight - overhead
		if viewportHeight < 5 {
			viewportHeight = 5
		}
		m.detailViewport = viewport.New(detailViewportWidth, viewportHeight)
		return
	}

	m.detailViewport.Width = detailViewportWidth

	if m.viewState == ViewDetail {
		if p := m.SelectedPlugin(); p != nil {
			detailContent := m.generateDetailContent(p, detailViewportWidth)
			contentHeight := lipgloss.Height(detailContent)
			maxHeight := terminalHeight - overhead
			if maxHeight < 3 {
				maxHeight = 3
			}

			if contentHeight < maxHeight {
				m.detailViewport.Height = contentHeight
			} else {
				m.detailViewport.Height = maxHeight
			}

			m.detailViewport.SetContent(detailContent)
		}
	}
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
		// Handle marketplace autocomplete navigation
		if m.marketplaceAutocompleteActive {
			if m.marketplaceAutocompleteCursor > 0 {
				m.marketplaceAutocompleteCursor--
			}
			return m, nil
		}

		if m.cursor > 0 {
			m.cursor--
		}
		m.UpdateScroll()
		m.SetCursorTarget()
		return m, animationTick()

	case "down", "ctrl+j", "ctrl+n":
		// Handle marketplace autocomplete navigation
		if m.marketplaceAutocompleteActive {
			if m.marketplaceAutocompleteCursor < len(m.marketplaceAutocompleteList)-1 {
				m.marketplaceAutocompleteCursor++
			}
			return m, nil
		}

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
		// Handle marketplace autocomplete selection
		if m.marketplaceAutocompleteActive {
			m.SelectMarketplaceAutocomplete()
			m.results = m.filteredSearch(m.textInput.Value())
			return m, nil
		}

		if len(m.results) > 0 {
			// Set detail viewport content before transition (like help menu)
			if m.detailViewport.Width > 0 {
				if p := m.SelectedPlugin(); p != nil {
					contentWidth := m.ContentWidth() - 10
					if contentWidth < 40 {
						contentWidth = 40
					}
					detailContent := m.generateDetailContent(p, contentWidth)

					// Calculate viewport height (match WindowSizeMsg overhead)
					contentHeight := lipgloss.Height(detailContent)
					maxHeight := m.windowHeight - 9
					if maxHeight < 3 {
						maxHeight = 3
					}

					if contentHeight < maxHeight {
						m.detailViewport.Height = contentHeight
					} else {
						m.detailViewport.Height = maxHeight
					}

					m.detailViewport.SetContent(detailContent)
					m.detailViewport.GotoTop() // Reset scroll position
				}
			}
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

	// Update marketplace autocomplete state
	m.UpdateMarketplaceAutocomplete(newValue)

	// Re-run search on input change (with filter)
	if !m.marketplaceAutocompleteActive {
		m.results = m.filteredSearch(newValue)
	}

	// Reset cursor to top on any search input change
	if newValue != oldValue {
		m.cursor = 0
		m.scrollOffset = 0
		m.marketplaceAutocompleteCursor = 0
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
// TODO(Phase 4.2): Split into sub-handlers to reduce complexity (currently 35)
//   - handleDetailCopyActions() for c, y, l, p keys
//   - handleDetailNavigationActions() for open, back, transitions
//   - See keybindings.go for centralized key definitions
func (m Model) handleDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "esc", "backspace":
		m.StartViewTransition(ViewList, -1) // Back transition
		return m, animationTick()

	case "c":
		if p := m.SelectedPlugin(); p != nil && !p.Installed {
			var copyText string
			if p.IsDiscoverable {
				copyText = fmt.Sprintf("/plugin marketplace add %s", p.MarketplaceSource)
			} else {
				copyText = p.InstallCommand()
			}

			if err := clipboard.WriteAll(copyText); err == nil {
				m.copiedFlash = true
				return m, clearCopiedFlash()
			}
			m.clipboardErrorFlash = true
			return m, clearClipboardError()
		}
		return m, nil

	case "y":
		if p := m.SelectedPlugin(); p != nil && !p.Installed && p.IsDiscoverable {
			if err := clipboard.WriteAll(p.InstallCommand()); err == nil {
				m.copiedFlash = true
				return m, clearCopiedFlash()
			}
			m.clipboardErrorFlash = true
			return m, clearClipboardError()
		}
		return m, nil

	case "g":
		if p := m.SelectedPlugin(); p != nil {
			url := p.GitHubURL()
			if url != "" && strings.HasPrefix(url, "https://github.com/") {
				openURL(url)
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
		if p := m.SelectedPlugin(); p != nil && p.Installed && p.InstallPath != "" {
			openPath(p.InstallPath)
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
		return m, animationTick()

	case "?":
		m.StartViewTransition(ViewHelp, 1) // Forward transition
		return m, animationTick()

	default:
		// Pass other keys to viewport for scrolling (up/down/pgup/pgdown)
		var cmd tea.Cmd
		m.detailViewport, cmd = m.detailViewport.Update(msg)
		return m, cmd
	}
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
			// Create a copy to avoid holding a pointer to a slice element
			item := m.marketplaceItems[m.marketplaceCursor]
			m.selectedMarketplace = &item
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
		if m.selectedMarketplace != nil && m.selectedMarketplace.Status != MarketplaceInstalled {
			installCmd := fmt.Sprintf("/plugin marketplace add %s",
				extractMarketplaceSource(m.selectedMarketplace.Repo))
			if err := clipboard.WriteAll(installCmd); err == nil {
				m.copiedFlash = true
				return m, clearCopiedFlash()
			}
			m.clipboardErrorFlash = true
			return m, clearClipboardError()
		}
		return m, nil

	case "f":
		m.previousViewBeforeMarketplace = ViewList
		m.StartViewTransition(ViewList, -1)
		m.textInput.SetValue("@" + m.selectedMarketplace.Name)
		m.results = m.filteredSearch(m.textInput.Value())
		m.cursor = 0
		m.scrollOffset = 0
		return m, animationTick()

	case "g":
		if m.selectedMarketplace != nil {
			url := m.selectedMarketplace.Repo
			if strings.HasPrefix(url, "https://github.com/") {
				openURL(url)
				m.githubOpenedFlash = true
				return m, clearGithubOpenedFlash()
			}
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

func openURL(url string) {
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

	// #nosec G204 -- cmd is determined by runtime.GOOS (trusted), args is validated URL
	_ = exec.Command(cmd, args...).Start()
}

func openPath(path string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{path}
	case "windows":
		cmd = "explorer"
		args = []string{path}
	default:
		cmd = "xdg-open"
		args = []string{path}
	}

	// #nosec G204 -- cmd is determined by runtime.GOOS (trusted), args is install path from config
	_ = exec.Command(cmd, args...).Start()
}
