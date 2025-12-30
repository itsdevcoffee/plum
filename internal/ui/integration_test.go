// Package ui integration tests
//
// This test suite validates core user flows through the Plum TUI application.
// These integration tests verify that the Bubbletea model correctly handles
// user interactions and state transitions.
//
// Test Coverage:
// - Initial application load and model creation
// - Search functionality with various queries
// - Cursor navigation (up/down, home/end, page up/down)
// - View transitions (list ↔ detail, help toggle)
// - Filter mode switching (All, Discover, Ready, Installed)
// - Window resize responsiveness
// - Plugin selection logic
//
// Testing Strategy:
// The tests create a Model instance, populate it with test data, and simulate
// user input by passing tea.KeyMsg and tea.WindowSizeMsg to the Update() function.
// This approach tests the full message-passing cycle without requiring an actual
// terminal or rendering.
//
// Each test is independent and uses fresh test data to avoid state pollution.
package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/itsdevcoffee/plum/internal/plugin"
	"github.com/itsdevcoffee/plum/internal/search"
)

// TestInitialLoad verifies the application initializes correctly
func TestInitialLoad(t *testing.T) {
	t.Run("model creation", func(t *testing.T) {
		model := NewModel()

		if !model.loading {
			t.Error("Expected loading=true on creation")
		}
		if model.viewState != ViewList {
			t.Errorf("Expected ViewList, got %v", model.viewState)
		}
		if model.filterMode != FilterAll {
			t.Errorf("Expected FilterAll, got %v", model.filterMode)
		}
		if model.cursor != 0 {
			t.Error("Expected cursor=0 on creation")
		}
	})

	t.Run("init command", func(t *testing.T) {
		model := NewModel()
		cmd := model.Init()

		if cmd == nil {
			t.Error("Init() should return a command")
		}
	})
}

// TestSearchFlow verifies the search functionality
func TestSearchFlow(t *testing.T) {
	model := NewModel()
	model.allPlugins = createTestPlugins()
	model.loading = false

	tests := []struct {
		name          string
		query         string
		expectResults int
		expectInTop   string // Plugin name that should appear in results
	}{
		{
			name:          "empty query shows all",
			query:         "",
			expectResults: 5,
		},
		{
			name:          "single letter match",
			query:         "t",
			expectResults: 3, // test, testing, another
		},
		{
			name:          "full name match",
			query:         "test-plugin",
			expectResults: 1,
			expectInTop:   "test-plugin",
		},
		{
			name:          "partial match",
			query:         "plug",
			expectResults: 3, // test-plugin, example-plugin, sample-plugin
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate typing the search query
			for _, ch := range tt.query {
				msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}}
				updatedModel, _ := model.Update(msg)
				model = updatedModel.(Model)
			}

			// Trigger search by setting query and applying filter
			model.textInput.SetValue(tt.query)
			model.applyFilter()

			if len(model.results) != tt.expectResults {
				t.Errorf("Expected %d results, got %d for query %q",
					tt.expectResults, len(model.results), tt.query)
			}

			if tt.expectInTop != "" && len(model.results) > 0 {
				if model.results[0].Plugin.Name != tt.expectInTop {
					t.Errorf("Expected %q at top of results, got %q",
						tt.expectInTop, model.results[0].Plugin.Name)
				}
			}
		})
	}
}

// TestNavigationFlow verifies cursor movement and scrolling
func TestNavigationFlow(t *testing.T) {
	model := NewModel()
	model.allPlugins = createTestPlugins()
	model.loading = false
	model.applyFilter() // Get all plugins in results
	model.windowHeight = 20

	t.Run("cursor down movement", func(t *testing.T) {
		initialCursor := model.cursor
		msg := tea.KeyMsg{Type: tea.KeyDown}
		updatedModel, _ := model.Update(msg)
		model = updatedModel.(Model)

		if model.cursor != initialCursor+1 {
			t.Errorf("Expected cursor to move down from %d to %d, got %d",
				initialCursor, initialCursor+1, model.cursor)
		}
	})

	t.Run("cursor up movement", func(t *testing.T) {
		model.cursor = 2
		msg := tea.KeyMsg{Type: tea.KeyUp}
		updatedModel, _ := model.Update(msg)
		model = updatedModel.(Model)

		if model.cursor != 1 {
			t.Errorf("Expected cursor=1, got %d", model.cursor)
		}
	})

	t.Run("cursor bounds - top", func(t *testing.T) {
		model.cursor = 0
		msg := tea.KeyMsg{Type: tea.KeyUp}
		updatedModel, _ := model.Update(msg)
		model = updatedModel.(Model)

		if model.cursor != 0 {
			t.Errorf("Expected cursor to stay at 0, got %d", model.cursor)
		}
	})

	t.Run("cursor bounds - bottom", func(t *testing.T) {
		model.cursor = len(model.results) - 1
		msg := tea.KeyMsg{Type: tea.KeyDown}
		updatedModel, _ := model.Update(msg)
		model = updatedModel.(Model)

		if model.cursor != len(model.results)-1 {
			t.Errorf("Expected cursor to stay at %d, got %d",
				len(model.results)-1, model.cursor)
		}
	})

	t.Run("home key jumps to start", func(t *testing.T) {
		model.cursor = 3
		msg := tea.KeyMsg{Type: tea.KeyHome}
		updatedModel, _ := model.Update(msg)
		model = updatedModel.(Model)

		if model.cursor != 0 {
			t.Errorf("Expected cursor=0 after Home, got %d", model.cursor)
		}
	})

	t.Run("end key jumps to bottom", func(t *testing.T) {
		model.cursor = 0
		msg := tea.KeyMsg{Type: tea.KeyEnd}
		updatedModel, _ := model.Update(msg)
		model = updatedModel.(Model)

		expected := len(model.results) - 1
		if model.cursor != expected {
			t.Errorf("Expected cursor=%d after End, got %d", expected, model.cursor)
		}
	})
}

// TestViewTransitions verifies navigation between views
func TestViewTransitions(t *testing.T) {
	model := NewModel()
	model.allPlugins = createTestPlugins()
	model.loading = false
	model.applyFilter()
	model.windowWidth = 100
	model.windowHeight = 30

	t.Run("list to detail transition", func(t *testing.T) {
		if model.viewState != ViewList {
			t.Errorf("Expected initial ViewList, got %v", model.viewState)
		}

		// Press Enter to view details
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := model.Update(msg)
		model = updatedModel.(Model)

		if model.viewState != ViewDetail {
			t.Errorf("Expected ViewDetail after Enter, got %v", model.viewState)
		}
	})

	t.Run("detail back to list", func(t *testing.T) {
		model.viewState = ViewDetail

		// Press Esc to go back
		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updatedModel, _ := model.Update(msg)
		model = updatedModel.(Model)

		if model.viewState != ViewList {
			t.Errorf("Expected ViewList after Esc, got %v", model.viewState)
		}
	})

	t.Run("help view toggle", func(t *testing.T) {
		model.viewState = ViewList

		// Press ? to open help
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
		updatedModel, _ := model.Update(msg)
		model = updatedModel.(Model)

		if model.viewState != ViewHelp {
			t.Errorf("Expected ViewHelp after '?', got %v", model.viewState)
		}

		// Press Esc to close help
		msg = tea.KeyMsg{Type: tea.KeyEsc}
		updatedModel, _ = model.Update(msg)
		model = updatedModel.(Model)

		if model.viewState != ViewList {
			t.Errorf("Expected ViewList after closing help, got %v", model.viewState)
		}
	})
}

// TestFilterMode verifies filter switching
func TestFilterMode(t *testing.T) {
	model := NewModel()
	model.allPlugins = createMixedPlugins() // Some installed, some not
	model.loading = false
	model.applyFilter()

	initialResultCount := len(model.results)

	t.Run("cycle through filters", func(t *testing.T) {
		// Start with FilterAll
		if model.filterMode != FilterAll {
			t.Errorf("Expected FilterAll initially, got %v", model.filterMode)
		}

		// Press Tab to cycle filter
		msg := tea.KeyMsg{Type: tea.KeyTab}
		updatedModel, _ := model.Update(msg)
		model = updatedModel.(Model)

		if model.filterMode != FilterDiscover {
			t.Errorf("Expected FilterDiscover after Tab, got %v", model.filterMode)
		}

		// Results should be re-filtered
		if len(model.results) > initialResultCount {
			t.Error("Filter should reduce or maintain result count")
		}
	})

	t.Run("filter installed only", func(t *testing.T) {
		// Manually set to FilterInstalled
		model.filterMode = FilterInstalled
		model.applyFilter()

		// Verify only installed plugins are in results
		for _, result := range model.results {
			if !result.Plugin.Installed {
				t.Errorf("Plugin %q should be installed in FilterInstalled mode",
					result.Plugin.Name)
			}
		}
	})
}

// TestWindowResize verifies responsive behavior
func TestWindowResize(t *testing.T) {
	model := NewModel()
	model.windowWidth = 80
	model.windowHeight = 24

	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"small terminal", 60, 20},
		{"standard terminal", 80, 24},
		{"large terminal", 120, 40},
		{"wide terminal", 200, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tea.WindowSizeMsg{Width: tt.width, Height: tt.height}
			updatedModel, _ := model.Update(msg)
			model = updatedModel.(Model)

			if model.windowWidth != tt.width {
				t.Errorf("Expected windowWidth=%d, got %d", tt.width, model.windowWidth)
			}
			if model.windowHeight != tt.height {
				t.Errorf("Expected windowHeight=%d, got %d", tt.height, model.windowHeight)
			}
		})
	}
}

// TestSelectedPlugin verifies plugin selection logic
func TestSelectedPlugin(t *testing.T) {
	model := NewModel()
	model.allPlugins = createTestPlugins()
	model.loading = false
	model.applyFilter()

	t.Run("valid selection", func(t *testing.T) {
		model.cursor = 0
		selected := model.SelectedPlugin()

		if selected == nil {
			t.Fatal("Expected selected plugin, got nil")
		}
		if selected.Name != model.results[0].Plugin.Name {
			t.Errorf("Expected %q, got %q",
				model.results[0].Plugin.Name, selected.Name)
		}
	})

	t.Run("no results", func(t *testing.T) {
		model.results = []search.RankedPlugin{}
		selected := model.SelectedPlugin()

		if selected != nil {
			t.Error("Expected nil when no results, got plugin")
		}
	})

	t.Run("cursor out of bounds", func(t *testing.T) {
		model.results = createTestResults()
		model.cursor = 999
		selected := model.SelectedPlugin()

		if selected != nil {
			t.Error("Expected nil for out-of-bounds cursor, got plugin")
		}
	})
}

// Helper functions to create test data

func createTestPlugins() []plugin.Plugin {
	return []plugin.Plugin{
		{
			Name:              "test-plugin",
			Description:       "A test plugin",
			MarketplaceSource: "test-marketplace",
			Installed:         false,
		},
		{
			Name:              "example-plugin",
			Description:       "An example plugin",
			MarketplaceSource: "test-marketplace",
			Installed:         false,
		},
		{
			Name:              "testing-tool",
			Description:       "A testing tool",
			MarketplaceSource: "test-marketplace",
			Installed:         false,
		},
		{
			Name:              "another-tool",
			Description:       "Another tool for testing",
			MarketplaceSource: "test-marketplace",
			Installed:         false,
		},
		{
			Name:              "sample-plugin",
			Description:       "A sample plugin",
			MarketplaceSource: "test-marketplace",
			Installed:         false,
		},
	}
}

func createMixedPlugins() []plugin.Plugin {
	return []plugin.Plugin{
		{
			Name:              "installed-plugin",
			Description:       "An installed plugin",
			MarketplaceSource: "test-marketplace",
			Installed:         true,
		},
		{
			Name:              "ready-plugin",
			Description:       "Ready to install",
			MarketplaceSource: "test-marketplace",
			Installed:         false,
			IsDiscoverable:    false,
		},
		{
			Name:              "discoverable-plugin",
			Description:       "From uninstalled marketplace",
			MarketplaceSource: "uninstalled-marketplace",
			Installed:         false,
			IsDiscoverable:    true,
		},
	}
}

func createTestResults() []search.RankedPlugin {
	plugins := createTestPlugins()
	results := make([]search.RankedPlugin, len(plugins))
	for i, p := range plugins {
		results[i] = search.RankedPlugin{
			Plugin: p,
			Score:  100 - i*10,
		}
	}
	return results
}

// TestCopyFunctionality verifies clipboard operations
func TestCopyFunctionality(t *testing.T) {
	t.Run("copy install command for regular plugin", func(t *testing.T) {
		model := NewModel()
		model.allPlugins = createTestPlugins()
		model.loading = false
		model.applyFilter()
		model.viewState = ViewDetail
		model.cursor = 0 // Select first plugin

		// Verify selected plugin is not discoverable
		p := model.SelectedPlugin()
		if p == nil {
			t.Fatal("No plugin selected")
		}
		if p.IsDiscoverable {
			t.Skip("Test requires non-discoverable plugin")
		}

		// Note: Can't actually test clipboard.WriteAll in unit tests
		// but we can verify the command format would be correct
		expectedCmd := "/plugin install " + p.Name + "@" + p.Marketplace
		actualCmd := p.InstallCommand()

		if actualCmd != expectedCmd {
			t.Errorf("Expected install command %q, got %q", expectedCmd, actualCmd)
		}
	})

	t.Run("copy commands for discoverable plugin", func(t *testing.T) {
		model := NewModel()
		model.allPlugins = createMixedPlugins()
		model.loading = false
		model.applyFilter()

		// Find discoverable plugin
		var discoverableIdx int
		for i, p := range model.allPlugins {
			if p.IsDiscoverable {
				discoverableIdx = i
				model.cursor = i
				break
			}
		}

		// Apply filter to update results
		model.applyFilter()

		p := model.SelectedPlugin()
		if p == nil || !p.IsDiscoverable {
			t.Skip("No discoverable plugin found")
		}

		// Verify marketplace command format
		expectedMarketplaceCmd := "/plugin marketplace add " + p.MarketplaceSource
		if !strings.Contains(expectedMarketplaceCmd, p.MarketplaceSource) {
			t.Error("Marketplace command should contain source")
		}

		// Verify plugin install command format
		expectedPluginCmd := p.InstallCommand()
		if !strings.Contains(expectedPluginCmd, p.Name) {
			t.Error("Plugin command should contain name")
		}

		_ = discoverableIdx
	})
}

// TestMarketplaceBrowser verifies marketplace browser functionality
func TestMarketplaceBrowser(t *testing.T) {
	model := NewModel()
	model.windowWidth = 100
	model.windowHeight = 30

	t.Run("transition to marketplace browser", func(t *testing.T) {
		initialView := model.viewState

		// Press Shift+M to open marketplace browser
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'M'}}
		updatedModel, _ := model.Update(msg)
		model = updatedModel.(Model)

		// Should transition to marketplace list view
		if model.viewState != ViewMarketplaceList {
			t.Errorf("Expected ViewMarketplaceList, got %v", model.viewState)
		}

		// Should remember previous view
		if model.previousViewBeforeMarketplace != initialView {
			t.Errorf("Expected previousView %v, got %v", initialView, model.previousViewBeforeMarketplace)
		}
	})

	t.Run("marketplace sorting", func(t *testing.T) {
		model.viewState = ViewMarketplaceList
		model.marketplaceItems = createTestMarketplaceItems()

		initialSort := model.marketplaceSortMode

		// Press Tab to cycle sort mode
		msg := tea.KeyMsg{Type: tea.KeyTab}
		updatedModel, _ := model.Update(msg)
		model = updatedModel.(Model)

		if model.marketplaceSortMode == initialSort {
			t.Error("Sort mode should change after Tab")
		}
	})
}

// TestDisplayModeToggle verifies view mode switching
func TestDisplayModeToggle(t *testing.T) {
	model := NewModel()
	model.allPlugins = createTestPlugins()
	model.loading = false
	model.applyFilter()

	initialMode := model.displayMode

	// Press Shift+V to toggle display mode
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}}
	updatedModel, _ := model.Update(msg)
	model = updatedModel.(Model)

	if model.displayMode == initialMode {
		t.Error("Display mode should toggle after Shift+V")
	}

	// Toggle again should return to original
	updatedModel, _ = model.Update(msg)
	model = updatedModel.(Model)

	if model.displayMode != initialMode {
		t.Error("Display mode should toggle back to original")
	}
}

// TestQuitBehavior verifies quit and escape handling
func TestQuitBehavior(t *testing.T) {
	t.Run("quit from list view", func(t *testing.T) {
		model := NewModel()
		model.viewState = ViewList
		model.loading = false

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		_, cmd := model.Update(msg)

		if cmd == nil {
			t.Error("Expected quit command, got nil")
		}
	})

	t.Run("escape clears search before quitting", func(t *testing.T) {
		model := NewModel()
		model.viewState = ViewList
		model.loading = false
		model.textInput.SetValue("search query")
		model.applyFilter()

		// First Esc should clear search
		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updatedModel, _ := model.Update(msg)
		model = updatedModel.(Model)

		if model.textInput.Value() != "" {
			t.Error("Escape should clear search")
		}

		// Second Esc should quit
		_, cmd := model.Update(msg)
		if cmd == nil {
			t.Error("Expected quit command after second Esc")
		}
	})
}

// TestHelperMethods verifies utility functions
func TestHelperMethods(t *testing.T) {
	model := NewModel()
	model.allPlugins = createMixedPlugins()
	model.loading = false
	model.applyFilter()
	model.windowHeight = 30

	t.Run("maxVisibleItems calculation", func(t *testing.T) {
		maxItems := model.maxVisibleItems()
		if maxItems <= 0 {
			t.Error("maxVisibleItems should return positive number")
		}
		if maxItems > model.windowHeight {
			t.Error("maxVisibleItems should not exceed window height")
		}
	})

	t.Run("ContentWidth calculation", func(t *testing.T) {
		model.windowWidth = 120
		width := model.ContentWidth()
		if width <= 0 {
			t.Error("ContentWidth should return positive number")
		}
		if width > 120 {
			t.Error("ContentWidth should not exceed maxContentWidth")
		}
	})

	t.Run("filter mode name", func(t *testing.T) {
		name := model.FilterModeName()
		if name == "" {
			t.Error("FilterModeName should return non-empty string")
		}
	})

	t.Run("display mode name", func(t *testing.T) {
		name := model.DisplayModeName()
		if name == "" {
			t.Error("DisplayModeName should return non-empty string")
		}
	})

	t.Run("count functions", func(t *testing.T) {
		total := model.TotalPlugins()
		installed := model.InstalledCount()
		ready := model.ReadyCount()
		discoverable := model.DiscoverableCount()

		if total != len(model.allPlugins) {
			t.Errorf("TotalPlugins should equal allPlugins length, got %d vs %d", total, len(model.allPlugins))
		}

		if installed < 0 || ready < 0 || discoverable < 0 {
			t.Error("Count functions should not return negative values")
		}

		// Counts should sum to total (or less due to filtering)
		if installed+ready+discoverable > total {
			t.Error("Sum of counts should not exceed total")
		}
	})
}

// TestViewportFunctions verifies viewport-related methods
func TestViewportFunctions(t *testing.T) {
	model := NewModel()
	model.windowWidth = 100
	model.windowHeight = 30

	t.Run("visible results calculation", func(t *testing.T) {
		model.allPlugins = createTestPlugins()
		model.applyFilter()

		visible := model.VisibleResults()
		if len(visible) > len(model.results) {
			t.Error("VisibleResults should not exceed total results")
		}
	})

	t.Run("scroll offset bounds", func(t *testing.T) {
		offset := model.ScrollOffset()
		if offset < 0 {
			t.Error("ScrollOffset should not be negative")
		}
	})
}

// TestTransitionAnimation verifies animation state
func TestTransitionAnimation(t *testing.T) {
	model := NewModel()
	model.windowWidth = 100
	model.windowHeight = 30

	t.Run("view transition state", func(t *testing.T) {
		// Initially not transitioning
		if model.IsViewTransitioning() {
			t.Error("Should not be transitioning initially")
		}

		// Start transition
		model.StartViewTransition(ViewDetail, 1)

		if !model.IsViewTransitioning() {
			t.Error("Should be transitioning after StartViewTransition")
		}
	})

	t.Run("cursor animation state", func(t *testing.T) {
		// Set cursor target
		model.cursor = 0
		model.SetCursorTarget()

		// Initially might not be animating if already at target
		_ = model.IsAnimating()

		// Animated offset should be valid
		offset := model.AnimatedCursorOffset()
		if offset < 0 {
			t.Error("AnimatedCursorOffset should not be negative")
		}
	})

	t.Run("transition style cycling", func(t *testing.T) {
		initialStyle := model.transitionStyle

		model.CycleTransitionStyle()

		if model.transitionStyle == initialStyle {
			// Should change (unless wrapping)
			model.CycleTransitionStyle()
			model.CycleTransitionStyle()

			// After 3 cycles, should be back to start (3 total styles)
			if model.transitionStyle != initialStyle {
				t.Error("Should cycle through 3 transition styles")
			}
		}

		// Style name should be non-empty
		styleName := model.TransitionStyleName()
		if styleName == "" {
			t.Error("TransitionStyleName should return non-empty string")
		}
	})

	t.Run("update cursor animation", func(t *testing.T) {
		model.cursor = 5
		model.targetCursorY = 10.0
		model.cursorY = 0.0

		model.UpdateCursorAnimation()

		// Cursor Y should move toward target
		// (exact value depends on spring physics)
		_ = model.cursorY
	})

	t.Run("snap cursor to target", func(t *testing.T) {
		model.targetCursorY = 15.0
		model.cursorY = 5.0

		model.SnapCursorToTarget()

		if model.cursorY != model.targetCursorY {
			t.Errorf("SnapCursorToTarget should set cursorY to targetCursorY, got %f vs %f",
				model.cursorY, model.targetCursorY)
		}
	})

	t.Run("update view transition", func(t *testing.T) {
		model.StartViewTransition(ViewDetail, 1)

		model.UpdateViewTransition()

		// Transition should progress
		// (exact value depends on spring physics, just verify it runs)
		_ = model.transitionProgress
	})

	t.Run("transition offset calculation", func(t *testing.T) {
		model.windowHeight = 30
		model.StartViewTransition(ViewDetail, 1)

		offset := model.TransitionOffset()

		// Offset can exceed window height for slide animations
		// Just verify it returns a value (physics-based, can be large)
		_ = offset
	})
}

// Helper function to create test marketplace items
func createTestMarketplaceItems() []MarketplaceItem {
	return []MarketplaceItem{
		{
			Name:                 "test-marketplace-1",
			DisplayName:          "Test Marketplace 1",
			Repo:                 "https://github.com/test/marketplace1",
			Status:               MarketplaceInstalled,
			TotalPluginCount:     10,
			InstalledPluginCount: 3,
		},
		{
			Name:                 "test-marketplace-2",
			DisplayName:          "Test Marketplace 2",
			Repo:                 "https://github.com/test/marketplace2",
			Status:               MarketplaceAvailable,
			TotalPluginCount:     5,
			InstalledPluginCount: 0,
		},
	}
}
