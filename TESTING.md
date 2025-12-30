# Testing Guide

## Quick Start

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test -v ./internal/search

# Run specific test
go test -v ./internal/ui -run TestSearchFlow

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Coverage Standards

| Package | Current | Target | Status |
|---------|---------|--------|--------|
| search | 98.1% | 60%+ | ✅ Excellent |
| plugin | 100% | 40%+ | ✅ Perfect |
| ui | 20.2% | 20%+ | ✅ Good |
| marketplace | 41.0% | 40%+ | ✅ Good |
| config | 43.0% | 40%+ | ✅ Good |

**PR Requirements:**
- New code should maintain or improve package coverage
- Critical paths (search, plugin logic) require tests
- UI changes should include integration tests

## Testing Patterns

### 1. Table-Driven Tests

Used extensively for testing multiple scenarios efficiently.

**Example:** `internal/search/search_test.go`

```go
tests := []struct {
    name        string
    query       string
    expectCount int
}{
    {"empty query", "", 5},
    {"exact match", "test-plugin", 1},
    {"partial match", "test", 2},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        results := Search(tt.query, plugins)
        if len(results) != tt.expectCount {
            t.Errorf("Expected %d, got %d", tt.expectCount, len(results))
        }
    })
}
```

### 2. Integration Tests (Bubbletea)

Test TUI behavior by simulating user input through Update() cycle.

**Example:** `internal/ui/integration_test.go`

```go
model := NewModel()
model.allPlugins = createTestPlugins()

// Simulate key press
msg := tea.KeyMsg{Type: tea.KeyDown}
updatedModel, _ := model.Update(msg)
model = updatedModel.(Model)

// Verify state change
if model.cursor != 1 {
    t.Error("Cursor should move down")
}
```

### 3. Test Fixtures

Use helper functions to create consistent test data.

**Example:**

```go
func createTestPlugins() []plugin.Plugin {
    return []plugin.Plugin{
        {Name: "test-plugin", Installed: false},
        {Name: "example-plugin", Installed: true},
    }
}
```

### 4. Temporary Files/Directories

Use `t.TempDir()` and `t.Setenv()` for isolated tests.

**Example:** `internal/marketplace/cache_test.go`

```go
tmpDir := t.TempDir()
t.Setenv("CLAUDE_CONFIG_DIR", tmpDir)

// Test code that uses config directory
// Cleanup happens automatically
```

## Package-Specific Notes

### internal/search
- **Pattern:** Table-driven tests
- **Focus:** Scoring algorithm, edge cases, sorting
- **Coverage:** 98.1% (near-perfect)

### internal/plugin
- **Pattern:** Table-driven tests for all methods
- **Focus:** String formatting, URL construction, author resolution
- **Coverage:** 100% (perfect)

### internal/ui
- **Pattern:** Integration tests via Bubbletea message passing
- **Focus:** User flows, state transitions, viewport behavior
- **Coverage:** 20.2% (foundation)
- **Note:** Can't test clipboard operations directly (external dependency)

### internal/marketplace
- **Pattern:** Cache + HTTP mocking patterns
- **Focus:** Cache integrity, atomic writes, GitHub API parsing
- **Coverage:** 41.0% (solid)
- **Note:** Network tests use temp directories with function overrides

### internal/config
- **Pattern:** Filesystem tests with temp directories
- **Focus:** JSON parsing, file loading, validation
- **Coverage:** 43.0% (existing)

## Adding New Tests

### For New Search Logic
1. Add test case to `TestSearch` table
2. Verify scoring in `TestScorePlugin`
3. Check edge cases in `TestSearchEdgeCases`

### For New UI Features
1. Add integration test to `internal/ui/integration_test.go`
2. Create Model, set up state, simulate key presses
3. Verify state transitions and side effects

### For New Plugin Methods
1. Add table-driven test in `internal/plugin/plugin_test.go`
2. Cover all input variations and edge cases
3. Aim for 100% coverage (it's achievable)

## Running Tests in CI

CI automatically runs on:
- Pull requests
- Pushes to main
- Tag pushes

**CI Commands:**
```bash
go test -v -race -coverprofile=coverage.out ./...
```

**Linting:**
```bash
golangci-lint run --timeout=5m
```

## Debugging Failed Tests

```bash
# Verbose output
go test -v ./internal/search

# Run specific test
go test -run TestSearchFlow ./internal/search

# Show test coverage details
go test -coverprofile=coverage.out ./internal/search
go tool cover -func=coverage.out
```

## Best Practices

✅ **DO:**
- Use `t.TempDir()` for file operations
- Use table-driven tests for multiple scenarios
- Test edge cases (empty input, nil values, bounds)
- Use `t.Fatal()` for setup failures
- Use `t.Error()` for assertion failures
- Keep tests independent (no shared state)

❌ **DON'T:**
- Use `panic()` in tests
- Rely on specific timing (use channels/sync for concurrency)
- Test implementation details (test behavior, not internals)
- Create global test fixtures (use functions)
- Skip cleanup (trust `t.TempDir()` and `defer`)

## Coverage Goals by Phase

**Phase 1-2 (Complete):**
- search: 60%+ → **98.1%** ✅
- plugin: 40%+ → **100%** ✅
- ui: 10%+ → **20.2%** ✅
- marketplace: 40%+ → **41.0%** ✅

**Phase 3-4 (Complete):**
- Maintain existing coverage
- Add tests for new features
- Document complex refactoring needs

**Phase 5 (Future):**
- Target 60%+ overall weighted coverage
- Add performance benchmarks
- Integration test expansion

## Questions?

See README.md Development section for setup instructions.
Run `go test -h` for more testing options.
