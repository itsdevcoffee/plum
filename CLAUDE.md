# Plum Project Guide

## Pre-Push Checklist

**ALWAYS run before pushing to main:**

```bash
# 1. Run linter (must pass)
golangci-lint run --timeout=5m

# 2. Run tests (must pass)
go test ./...

# 3. Verify build succeeds
go build -o ./plum ./cmd/plum

# 4. Run integration tests (optional but recommended for settings changes)
go test -tags=integration ./internal/integration/... -v
```

**If linter is not installed:**

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.8.0
```

## Routine Maintenance

**At the start of each session**, check marketplace data freshness:

```bash
# 1. Update GitHub stats (stars, forks, last updated)
bash scripts/update-marketplace-stats.sh

# 2. Check plugin counts from live manifests
go run scripts/check-plugin-counts.go
```

**If changes detected:**

- Update `internal/marketplace/discovery.go` with new stats and plugin counts
- Update `README.md` marketplace table with accurate counts
- Update total plugin count in README intro and features section
- Run pre-push checklist above

## Why This Matters

- **Linting** - CI will fail if linting doesn't pass locally. Always lint before pushing.
- **Accurate Data** - The README is users' first impression. Accurate plugin counts and GitHub stats help them make informed decisions about installation.

## Documentation

Files go in `docs/` except for obvious exceptions: README.md, CLAUDE.md, LICENSE.md, CONTRIBUTING.md, AGENT.md, ... (root only).

**Subdirectories:**

- `context/` - Architecture, domain knowledge, static reference
- `decisions/` - Architecture Decision Records (ADRs)
- `handoff/` - Session state for development continuity
- `project/` - Planning: todos, features, roadmap
- `research/` - Explorations, comparisons, technical analysis
- `tmp/` - Scratch files (safe to delete)

**Naming:** `YYYY-MM-DD-descriptive-name.md` (lowercase, hyphens)

**Rules:**

- Update existing docs before creating new ones
- Use `tmp/` when uncertain, flag for review
