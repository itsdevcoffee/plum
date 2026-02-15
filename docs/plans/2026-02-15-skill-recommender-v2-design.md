# Skill Recommender v2 Design

**Date:** 2026-02-15
**Status:** Approved
**Branch:** feature/ecosystem-discovery

## Problem

The v1 skill recommender produces noisy results:
- Items match on single generic tags like "backend" (supabase in skills-leaderboard)
- High-popularity items score above threshold with zero tag overlap
- Duplicate entries appear across categories (e.g., everything-claude-code in plugins + skills)
- Ecosystem JSONs lack language/framework tags, limiting match quality for non-JS/React stacks
- No markdown output mode — Claude must parse JSON and reformat every time

## Decisions

- **Approach:** Comprehensive pass — extract scoring engine, enrich data, add features
- **Deduplication:** Keep highest-scoring entry only, annotate with `also_found_in`
- **Tag enrichment:** Curate JSONs manually + add lightweight keyword inference as fallback
- **Hard gate:** Items with zero matching tags (explicit or inferred) are excluded regardless of score

## Design

### 1. Scoring Engine (`scripts/ecosystem/scorer.py`)

New module with a multi-stage pipeline:

**Stage 1 — Tag matching (80% of relevance)**
- Bidirectional overlap: item tags ∩ stack tags
- Geometric mean of (overlap/item_tags) and (overlap/stack_tags)
- Prevents single-generic-tag matches from scoring high

**Stage 2 — Keyword inference (20% of relevance)**
- Curated `KEYWORD_TAG_MAP` maps description keywords → tags
- e.g., "terminal" → "terminal", "golang" → "go", "react native" → "react-native"
- Inferred tags count at 50% weight vs explicit tags
- Shown separately in output as `inferred_tags`

**Stage 3 — Popularity normalization**
- Log-scale within each source (unchanged)
- Recency boost: entries updated in past 3 months get 10% bump

**Stage 4 — Deduplication**
- Key on `owner/name` (lowercased)
- Keep entry with highest final score
- Annotate with `also_found_in: [source_keys]`

**Final formula:** `(relevance * 0.7) + (popularity * 0.3)`

**Hard gate:** Zero matching tags (explicit + inferred) → excluded.

### 2. Ecosystem Data Enrichment

Curate tags across all 5 JSON files:

| File | Focus |
|------|-------|
| `skills-leaderboard.json` | Add missing language tags, `development`/`general` for universal skills |
| `plugins.json` | Add `cli`, `github`, `development` where applicable |
| `agents.json` | Add `python`, `typescript`, `sdk`, `terminal` where applicable |
| `skills.json` | Add `go`, `python`, `php`, `tui` to domain-specific entries |
| `mcp-servers.json` | Add `slack`, `git`, `filesystem` to specialized servers |

Rules:
- Only factually accurate tags based on description/name
- Lowercase, hyphenated format
- Max 2-3 additions per entry
- Prefer tags matching `detect_stack.py` output

### 3. New Features for `recommend.py`

**`--markdown` flag**
- Pre-formatted markdown tables grouped by category
- Columns: Name, Score, Popularity, Matching Tags, Install Command

**`--exclude-no-install` flag**
- Filters entries where `install_cmd` is null

**`--json` flag (default)**
- Current behavior, explicit for clarity

**Updated defaults:**
- `--min-score` raised from 15 → 20
- Hard gate applied automatically

**Dedup in output:**
- Applied automatically
- `also_found_in` array on deduplicated entries

### 4. SKILL.md Updates

- Use `--markdown` by default
- Document `--exclude-no-install` for actionable-only queries
- Update fallback instructions (require tag match when reading JSONs directly)
- Add `--top 10` suggestion for expanded results

## Files Changed

| File | Action |
|------|--------|
| `scripts/ecosystem/scorer.py` | **New** — Multi-stage scoring engine |
| `scripts/ecosystem/recommend.py` | **Modified** — Import scorer, add flags, raise min-score |
| `scripts/ecosystem/detect_stack.py` | **No changes** |
| `ecosystem/skills-leaderboard.json` | **Modified** — Tag enrichment |
| `ecosystem/plugins.json` | **Modified** — Tag enrichment |
| `ecosystem/agents.json` | **Modified** — Tag enrichment |
| `ecosystem/skills.json` | **Modified** — Tag enrichment |
| `ecosystem/mcp-servers.json` | **Modified** — Tag enrichment |
| `.claude/skills/skill-recommender/SKILL.md` | **Modified** — Use new flags |

## Success Criteria

- Running against plum repo: skills_leaderboard returns 0 results (no relevant skills exist for Go/TUI in that dataset) rather than false positives
- No entries with empty `matching_tags` + empty `inferred_tags` appear in output
- No duplicate owner/name entries across categories
- `--markdown` produces clean, copy-pasteable tables
- `--exclude-no-install` filters curated lists correctly
