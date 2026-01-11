# Plum Project Guide

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
- Run: `go test ./internal/marketplace && go build -o ./plum ./cmd/plum`

## Why This Matters

The README is users' first impression - accurate plugin counts and GitHub stats help them make informed decisions about installation.
