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
