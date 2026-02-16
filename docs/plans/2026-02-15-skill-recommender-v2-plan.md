# Skill Recommender v2 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Improve the skill recommender with a proper scoring engine, ecosystem data enrichment, deduplication, and markdown output.

**Architecture:** Extract scoring logic into `scripts/ecosystem/scorer.py` with a 4-stage pipeline (tag matching, keyword inference, popularity normalization, deduplication). Refactor `recommend.py` to use the scorer and add `--markdown`/`--exclude-no-install` flags. Enrich all 5 ecosystem JSONs with missing language/framework tags.

**Tech Stack:** Python 3 (stdlib only), JSON

---

### Task 1: Create the scoring engine module

**Files:**
- Create: `scripts/ecosystem/scorer.py`

**Step 1: Create `scorer.py` with constants and keyword map**

Create `scripts/ecosystem/scorer.py` with:

```python
#!/usr/bin/env python3
"""Multi-stage scoring engine for ecosystem recommendations.

4-stage pipeline:
1. Tag matching (bidirectional, geometric mean)
2. Keyword inference (description → inferred tags)
3. Popularity normalization (log-scale + recency)
4. Deduplication (owner/name key, keep highest)
"""

import math
from datetime import datetime, timedelta

# Weights
RELEVANCE_WEIGHT = 0.7
POPULARITY_WEIGHT = 0.3
TAG_MATCH_WEIGHT = 0.8
KEYWORD_INFER_WEIGHT = 0.2
INFERRED_TAG_DISCOUNT = 0.5  # Inferred tags count at 50% vs explicit
RECENCY_MONTHS = 3
RECENCY_BOOST = 0.10  # 10% boost for recent entries

# Maps description/name keywords to inferred tags.
# Keys are lowercase substrings to search for in text.
# Values are lists of tags to infer.
KEYWORD_TAG_MAP = {
    "golang": ["go"],
    "go module": ["go"],
    "go.mod": ["go"],
    " go ": ["go"],
    "python": ["python"],
    "typescript": ["typescript"],
    "javascript": ["javascript"],
    "node.js": ["nodejs", "javascript"],
    "nodejs": ["nodejs", "javascript"],
    "rust": ["rust"],
    "ruby": ["ruby"],
    "java ": ["java"],
    "kotlin": ["kotlin"],
    "swift": ["swift"],
    "php": ["php"],
    "c++": ["cpp"],
    "react": ["react", "frontend"],
    "react native": ["react-native", "mobile"],
    "next.js": ["nextjs", "react", "frontend"],
    "nextjs": ["nextjs", "react", "frontend"],
    "vue": ["vue", "frontend"],
    "nuxt": ["nuxt", "vue", "frontend"],
    "svelte": ["svelte", "frontend"],
    "angular": ["angular", "frontend"],
    "remotion": ["remotion", "video"],
    "terminal": ["terminal"],
    "tui": ["tui", "terminal"],
    "command line": ["cli"],
    "command-line": ["cli"],
    " cli ": ["cli"],
    "cli tool": ["cli"],
    "docker": ["docker", "containerization"],
    "kubernetes": ["kubernetes", "infrastructure"],
    "terraform": ["terraform", "infrastructure"],
    "github action": ["github-actions", "ci-cd"],
    "ci/cd": ["ci-cd"],
    "ci-cd": ["ci-cd"],
    "database": ["database"],
    "postgresql": ["postgresql", "database"],
    "postgres": ["postgresql", "database"],
    "mongodb": ["mongodb", "database"],
    "redis": ["redis", "database"],
    "api": ["api"],
    "rest api": ["api", "backend"],
    "graphql": ["graphql", "api"],
    "grpc": ["grpc", "api"],
    "authentication": ["authentication", "security"],
    "oauth": ["oauth", "authentication", "security"],
    "machine learning": ["ml", "ai"],
    "deep learning": ["ml", "ai"],
    "llm": ["llm", "ai"],
    "testing": ["testing"],
    "unit test": ["testing"],
    "e2e": ["testing", "e2e"],
    "browser automation": ["browser", "automation"],
    "web scraping": ["scraping", "automation"],
    "slack": ["slack"],
    "git ": ["git"],
    "markdown": ["markdown"],
    "pdf": ["pdf", "document"],
    "document": ["document"],
    "design": ["design"],
    "ui/ux": ["design", "ui"],
    "accessibility": ["accessibility"],
    "seo": ["seo"],
    "marketing": ["marketing"],
    "deployment": ["deployment"],
    "serverless": ["serverless", "cloud"],
    "aws": ["aws", "cloud"],
    "azure": ["azure", "cloud"],
    "gcp": ["gcp", "cloud"],
    "cloudflare": ["cloudflare"],
    "vercel": ["vercel"],
    "supabase": ["supabase", "database"],
    "firebase": ["firebase", "backend"],
}
```

**Step 2: Add the 4 scoring functions**

Append to `scorer.py`:

```python
def infer_tags(item):
    """Stage 2: Infer tags from item description and name.

    Returns set of inferred tag strings.
    """
    text = " " + (item.get("description", "") + " " + item.get("name", "")).lower() + " "
    inferred = set()
    for keyword, tags in KEYWORD_TAG_MAP.items():
        if keyword in text:
            inferred.update(tags)
    # Remove tags that are already explicit on the item
    explicit = set(t.lower() for t in item.get("tags", []))
    return inferred - explicit


def compute_tag_score(item_tags, stack_tags, inferred_tags=None):
    """Stage 1: Bidirectional tag matching with geometric mean.

    Args:
        item_tags: set of lowercase tags from the ecosystem entry
        stack_tags: set of lowercase tags from the detected stack
        inferred_tags: set of inferred tags (counted at 50% weight)

    Returns:
        (score 0-100, list of matching explicit tags, list of matching inferred tags)
    """
    if not item_tags and not inferred_tags:
        return 0.0, [], []

    # Explicit tag overlap
    explicit_overlap = item_tags & stack_tags
    # Inferred tag overlap (discounted)
    inferred_overlap = (inferred_tags & stack_tags) if inferred_tags else set()

    # Weighted overlap count: explicit=1.0, inferred=0.5
    weighted_overlap = len(explicit_overlap) + len(inferred_overlap) * INFERRED_TAG_DISCOUNT

    if weighted_overlap == 0:
        return 0.0, [], []

    # Total tags considered (explicit + inferred at discount)
    total_item_tags = len(item_tags) + len(inferred_tags or set()) * INFERRED_TAG_DISCOUNT
    total_stack_tags = len(stack_tags)

    if total_item_tags == 0 or total_stack_tags == 0:
        return 0.0, [], []

    # Bidirectional: how much of item matches stack AND how much of stack matches item
    item_ratio = weighted_overlap / total_item_tags
    stack_ratio = weighted_overlap / total_stack_tags

    # Geometric mean prevents single-tag matches from scoring high
    geo_mean = math.sqrt(item_ratio * stack_ratio)
    score = geo_mean * 100

    return score, sorted(explicit_overlap), sorted(inferred_overlap)


def normalize_popularity(items, pop_field, last_updated_str=None):
    """Stage 3: Log-scale popularity with optional recency boost.

    Mutates items in-place, adding '_pop_norm' field.

    Args:
        items: list of ecosystem entry dicts
        pop_field: key name for the popularity metric (e.g., 'stars', 'installs')
        last_updated_str: ISO date string from the source file's last_updated field
    """
    values = [item.get(pop_field, 0) for item in items]
    max_val = max(values) if values else 0

    # Determine if source data is recent (within RECENCY_MONTHS)
    is_recent = False
    if last_updated_str:
        try:
            last_updated = datetime.strptime(last_updated_str, "%Y-%m-%d")
            cutoff = datetime.now() - timedelta(days=RECENCY_MONTHS * 30)
            is_recent = last_updated >= cutoff
        except ValueError:
            pass

    for item in items:
        raw = item.get(pop_field, 0)
        if max_val > 0:
            norm = math.log1p(raw) / math.log1p(max_val) * 100
        else:
            norm = 0.0

        if is_recent:
            norm = min(norm * (1 + RECENCY_BOOST), 100.0)

        item["_pop_norm"] = norm

    return items


def deduplicate(all_scored):
    """Stage 4: Deduplicate across sources, keep highest score.

    Args:
        all_scored: list of dicts, each with 'owner', 'name', 'score', 'source', plus all other fields

    Returns:
        list of deduplicated dicts, highest-score entry kept, with 'also_found_in' added where applicable
    """
    seen = {}  # key: "owner/name" -> best entry dict
    source_tracker = {}  # key: "owner/name" -> list of source labels

    for entry in all_scored:
        key = f"{entry['owner']}/{entry['name']}".lower()

        if key not in source_tracker:
            source_tracker[key] = []
        source_tracker[key].append(entry["source"])

        if key not in seen or entry["score"] > seen[key]["score"]:
            seen[key] = entry

    # Annotate with also_found_in
    result = []
    for key, entry in seen.items():
        sources = source_tracker[key]
        if len(sources) > 1:
            entry["also_found_in"] = sorted(set(s for s in sources if s != entry["source"]))
        result.append(entry)

    return result


def score_item(item, stack_tags, pop_field):
    """Run the full scoring pipeline on a single item.

    Args:
        item: ecosystem entry dict
        stack_tags: set of lowercase tags from detected stack

    Returns:
        (final_score, matching_tags, inferred_tags) or None if excluded by hard gate
    """
    item_tags = set(t.lower() for t in item.get("tags", []))
    inferred = infer_tags(item)

    # Stage 1+2: Tag score (explicit + inferred)
    tag_score, matching_explicit, matching_inferred = compute_tag_score(
        item_tags, stack_tags, inferred
    )

    # Hard gate: must have at least one matching tag (explicit or inferred)
    if not matching_explicit and not matching_inferred:
        return None

    # Relevance = tag matching only (keyword inference is baked into tag inference)
    relevance = tag_score

    # Stage 3: Popularity (already normalized on item as _pop_norm)
    pop_score = item.get("_pop_norm", 0)

    # Final score
    final_score = (relevance * RELEVANCE_WEIGHT) + (pop_score * POPULARITY_WEIGHT)

    return final_score, matching_explicit, matching_inferred
```

**Step 3: Run a quick syntax check**

Run: `python3 -c "import scripts.ecosystem.scorer; print('OK')" 2>&1 || python3 scripts/ecosystem/scorer.py`

Expected: No syntax errors (module has no `__main__` block, so just verifying it imports)

**Step 4: Commit**

```bash
git add scripts/ecosystem/scorer.py
git commit -m "feat: add multi-stage scoring engine for ecosystem recommendations"
```

---

### Task 2: Refactor recommend.py to use scorer

**Files:**
- Modify: `scripts/ecosystem/recommend.py`

**Step 1: Replace scoring logic with scorer imports**

Rewrite `recommend.py` to:
- Remove inline scoring functions (`_compute_relevance`, `_build_keywords`, `_normalize_popularity`)
- Import from `scorer`: `score_item`, `normalize_popularity`, `deduplicate`, `infer_tags`
- Keep `_load_json` and `_format_entry` (but update `_format_entry` to include `inferred_tags` and `also_found_in`)
- Update `recommend()` to:
  1. Build `stack_tags` set from profile (languages + tags + frameworks, all lowercased)
  2. For each source: normalize popularity, score each item via `score_item()`, collect results
  3. Flatten all scored items, run `deduplicate()`
  4. Re-group by source, sort by score, apply top_n
- Raise `DEFAULT_MIN_SCORE` from 15 to 20

**Step 2: Add new CLI flags**

Add to argparse:
- `--markdown` flag: calls a new `_format_markdown(result)` function
- `--exclude-no-install` flag: filters entries where `install_cmd` is missing
- `--json` flag: explicit JSON mode (default)
- Output format group: `--json` | `--markdown` (mutually exclusive)

**Step 3: Implement `_format_markdown(result)`**

New function that outputs:

```
## Stack Profile

**Languages:** Go, Python
**Frameworks:** bubbletea, cobra, lipgloss
**Tags:** cli, tui, terminal, ci-cd, ...

## Top Skills (skills.sh)

| Name | Score | Installs | Tags | Install |
|------|-------|----------|------|---------|
| ... | ... | ... | ... | `cmd` |

## Plugins

| Name | Score | Stars | Tags | Install |
|------|-------|-------|------|---------|
...
```

Category label mapping:
- `skills_leaderboard` → "Top Skills (skills.sh)"
- `plugins` → "Plugins"
- `agents` → "Agents"
- `skills` → "Skill Repositories"
- `mcp_servers` → "MCP Servers"

Only print categories with results.

**Step 4: Update `_format_entry` to include new fields**

Add `inferred_tags` list and conditional `also_found_in` list to the entry dict.

**Step 5: Test JSON output**

Run: `python3 scripts/ecosystem/recommend.py . --pretty`

Expected:
- `total_matches` likely lower than v1 (hard gate filters noise)
- No entries with both `matching_tags: []` and `inferred_tags: []`
- No duplicate `owner/name` across categories

**Step 6: Test markdown output**

Run: `python3 scripts/ecosystem/recommend.py . --markdown`

Expected: Clean markdown tables with headers, no empty categories.

**Step 7: Test exclude-no-install flag**

Run: `python3 scripts/ecosystem/recommend.py . --markdown --exclude-no-install`

Expected: Only entries with install commands appear.

**Step 8: Test error case**

Run: `python3 scripts/ecosystem/recommend.py /nonexistent --pretty`

Expected: `{"error": "Not a directory: /nonexistent"}`

**Step 9: Commit**

```bash
git add scripts/ecosystem/recommend.py
git commit -m "feat: refactor recommend.py to use scorer engine, add markdown and filter flags"
```

---

### Task 3: Enrich ecosystem JSON tags — skills-leaderboard.json

**Files:**
- Modify: `ecosystem/skills-leaderboard.json`

**Step 1: Add language/framework tags to each entry**

Apply these specific tag additions (only adding, never removing existing tags):

| Entry | Add Tags |
|-------|----------|
| `find-skills` | `"meta"` (already has it via category) — no change needed |
| `vercel-react-best-practices` | `"javascript"`, `"typescript"` |
| `web-design-guidelines` | `"css"`, `"frontend"` |
| `remotion-best-practices` | `"javascript"`, `"typescript"`, `"react"` |
| `frontend-design` | `"css"`, `"frontend"` |
| `vercel-composition-patterns` | `"javascript"`, `"typescript"` |
| `agent-browser` | `"web"`, `"javascript"` |
| `skill-creator` | `"development"`, `"meta"` |
| `browser-use` | `"web"`, `"javascript"` |
| `vercel-react-native-skills` | `"javascript"`, `"typescript"` |
| `ui-ux-pro-max` | `"css"`, `"frontend"` |
| `audit-website` | `"web"` |
| `brainstorming` | `"development"`, `"general"` |
| `seo-audit` | `"web"` |
| `supabase-postgres-best-practices` | `"sql"` |
| `pdf` | `"python"` |
| `copywriting` | `"general"`, `"content"` |
| `pptx` | `"python"` |
| `next-best-practices` | `"javascript"`, `"typescript"`, `"react"` |
| `better-auth-best-practices` | `"web"`, `"backend"` |
| `docx` | `"python"` |
| `agent-tools` | `"development"` |
| `xlsx` | `"python"` |
| `systematic-debugging` | `"development"`, `"general"` |
| `marketing-psychology` | `"general"` |

**Step 2: Verify JSON is valid**

Run: `python3 -c "import json; json.load(open('ecosystem/skills-leaderboard.json')); print('valid')"`

**Step 3: Commit**

```bash
git add ecosystem/skills-leaderboard.json
git commit -m "chore: enrich skills-leaderboard.json with language and domain tags"
```

---

### Task 4: Enrich ecosystem JSON tags — plugins.json

**Files:**
- Modify: `ecosystem/plugins.json`

**Step 1: Add tags**

| Entry | Add Tags |
|-------|----------|
| `claude-code` (anthropics) | `"development"` |
| `everything-claude-code` | `"development"`, `"testing"` |
| `awesome-claude-code` (hesreallyhim) | `"development"` |
| `claude-plugins-official` | `"development"` |
| `claude-code-action` | `"github-actions"` |
| `gmickel-claude-marketplace` | `"development"` |
| `awesome-claude-code-plugins` | `"development"` |
| `claude-code-plugin-template` | `"template"`, `"development"` |
| `cc-marketplace-boilerplate` | `"development"` |
| `claude-starter-kit` | `"development"` |
| `ai-coding-project-boilerplate` | `"development"` |
| `ralph-loop-setup` | `"development"`, `"testing"` |
| `flow-next-opencode` | `"development"` |

**Step 2: Verify JSON**

Run: `python3 -c "import json; json.load(open('ecosystem/plugins.json')); print('valid')"`

**Step 3: Commit**

```bash
git add ecosystem/plugins.json
git commit -m "chore: enrich plugins.json with development and domain tags"
```

---

### Task 5: Enrich ecosystem JSON tags — agents.json

**Files:**
- Modify: `ecosystem/agents.json`

**Step 1: Add tags**

| Entry | Add Tags |
|-------|----------|
| `anthropic-cookbook` | `"python"`, `"development"` |
| `agents` (wshobson) | `"development"` |
| `SuperClaude_Framework` | `"development"` |
| `claude-quickstarts` | `"development"`, `"python"` |
| `claude-flow` | `"development"` |
| `awesome-claude-code-subagents` | `"development"` |
| `claude-squad` | `"go"`, `"cli"` |
| `claude-agent-sdk-python` | — already has `"python"` |
| `awesome-claude-agents` (vijaythecoder) | `"development"` |
| `claude-agent-sdk-typescript` | — already has `"typescript"` |

**Step 2: Verify JSON**

Run: `python3 -c "import json; json.load(open('ecosystem/agents.json')); print('valid')"`

**Step 3: Commit**

```bash
git add ecosystem/agents.json
git commit -m "chore: enrich agents.json with language and development tags"
```

---

### Task 6: Enrich ecosystem JSON tags — skills.json

**Files:**
- Modify: `ecosystem/skills.json`

**Step 1: Add tags**

| Entry | Add Tags |
|-------|----------|
| `skills` (anthropics) | `"python"`, `"development"` |
| `everything-claude-code` (affaan-m) | `"development"`, `"testing"` |
| `awesome-claude-skills` (ComposioHQ) | `"development"` |
| `agents` (wshobson) | `"development"` |
| `awesome-claude-skills` (travisvn) | `"development"` |
| `awesome-agent-skills` (VoltAgent) | `"development"`, `"multi-platform"` |
| `awesome-claude-skills` (BehiSecc) | `"development"` |
| `claude-skills` (alirezarezvani) | `"development"` |
| `claude-code-skills` (daymade) | `"development"` |
| `claude-code-skill-factory` | `"development"` |
| `claude-code-marketplace` (netresearch) | — already has `"typo3"`, `"php"` |
| `claude-marketplace` (dashed) | `"cli"`, `"development"` |

**Step 2: Verify JSON**

Run: `python3 -c "import json; json.load(open('ecosystem/skills.json')); print('valid')"`

**Step 3: Commit**

```bash
git add ecosystem/skills.json
git commit -m "chore: enrich skills.json with language and development tags"
```

---

### Task 7: Enrich ecosystem JSON tags — mcp-servers.json

**Files:**
- Modify: `ecosystem/mcp-servers.json`

**Step 1: Add tags**

| Entry | Add Tags |
|-------|----------|
| `servers` (modelcontextprotocol) | `"git"`, `"filesystem"`, `"development"` |
| `registry` (modelcontextprotocol) | `"development"` |
| `slack-mcp-server` (korotovsky) | — already has `"slack"` |
| `slack-mcp-server` (jtalk22) | — already has `"slack"` |
| `claude-code-slack-bot` | — already has `"slack"` |
| `MCPJungle` | `"development"`, `"docker"` |
| `metamcp` | `"development"` |

**Step 2: Verify JSON**

Run: `python3 -c "import json; json.load(open('ecosystem/mcp-servers.json')); print('valid')"`

**Step 3: Commit**

```bash
git add ecosystem/mcp-servers.json
git commit -m "chore: enrich mcp-servers.json with development and domain tags"
```

---

### Task 8: Update SKILL.md

**Files:**
- Modify: `.claude/skills/skill-recommender/SKILL.md`

**Step 1: Update the skill instructions**

Replace the content with updated instructions that:
- Use `--markdown` by default instead of `--pretty`
- Document `--exclude-no-install` for "what should I install" queries
- Mention `--top 10` for more results
- Update fallback instructions to require tag match
- Update the Notes section with new min-score default of 20

**Step 2: Commit**

```bash
git add .claude/skills/skill-recommender/SKILL.md
git commit -m "feat: update SKILL.md to use markdown output and document new flags"
```

---

### Task 9: End-to-end verification

**Step 1: Test against plum repo (Go/TUI/CLI stack)**

Run: `python3 scripts/ecosystem/recommend.py . --markdown`

Verify:
- No false positives in skills_leaderboard (supabase should NOT appear — only matched on generic "backend")
- `claude-squad` should appear (has "terminal" tag + now "go", "cli" tags)
- No duplicate entries across categories
- Markdown tables are clean and readable

**Step 2: Test JSON output still works**

Run: `python3 scripts/ecosystem/recommend.py . --pretty`

Verify: Valid JSON, entries have `matching_tags` and `inferred_tags` fields.

**Step 3: Test exclude-no-install**

Run: `python3 scripts/ecosystem/recommend.py . --markdown --exclude-no-install`

Verify: Only entries with install commands shown.

**Step 4: Test error handling**

Run: `python3 scripts/ecosystem/recommend.py /nonexistent`

Verify: Returns JSON error without traceback.

**Step 5: Compare v1 vs v2 match counts**

Run: `python3 scripts/ecosystem/recommend.py . --pretty | python3 -c "import sys,json; d=json.load(sys.stdin); print('Total:', d['total_matches']); [print(f'  {k}: {len(v)}') for k,v in d['recommendations'].items()]"`

Verify: Lower total than v1's 16 matches, with higher precision (every match has tag overlap).

**Step 6: Push all changes**

```bash
git push
```
