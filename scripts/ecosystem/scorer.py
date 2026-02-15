#!/usr/bin/env python3
"""Multi-stage scoring engine for ecosystem recommendations.

Pipeline stages:
  1. Tag matching (80% of relevance) — bidirectional geometric-mean overlap
  2. Keyword inference (20% of relevance) — curated map, 50% weight discount
  3. Popularity normalization — log-scale per source, recency boost
  4. Deduplication — key on owner/name, keep highest score

Zero external dependencies — stdlib only.
"""

import math
import re
from datetime import datetime, timedelta

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

RELEVANCE_WEIGHT = 0.7
POPULARITY_WEIGHT = 0.3
TAG_MATCH_WEIGHT = 0.8          # explicit tag share of relevance
INFERRED_TAG_DISCOUNT = 0.5     # inferred tags count at half weight
RECENCY_MONTHS = 3
RECENCY_BOOST = 0.10

# ---------------------------------------------------------------------------
# Keyword → tag map  (description keywords → canonical tags)
# ---------------------------------------------------------------------------

# Each entry is (regex_pattern_string, canonical_tag).
# Short/ambiguous words use \b word boundaries to prevent substring matches.
KEYWORD_TAG_MAP: list[tuple[str, str]] = [
    # Languages — short words need \b boundaries
    (r"golang",              "go"),
    (r"\bgo\b",              "go"),
    (r"python",              "python"),
    (r"typescript",          "typescript"),
    (r"javascript",          "javascript"),
    (r"nodejs",              "nodejs"),
    (r"node\.js",            "nodejs"),
    (r"\trust\b",            "rust"),
    (r"\bruby\b",            "ruby"),
    (r"\bjava\b",            "java"),
    (r"kotlin",              "kotlin"),
    (r"\bswift\b",           "swift"),
    (r"\bphp\b",             "php"),
    (r"c\+\+",              "cpp"),
    (r"\bcpp\b",             "cpp"),

    # Frameworks — check multi-word patterns before shorter ones
    (r"react[- ]native",     "react-native"),
    (r"\breact\b",           "react"),
    (r"next\.js",            "nextjs"),
    (r"nextjs",              "nextjs"),
    (r"\bvue\b",             "vue"),
    (r"\bnuxt\b",            "nuxt"),
    (r"svelte",              "svelte"),
    (r"angular",             "angular"),
    (r"remotion",            "remotion"),

    # Domains — terminal / CLI
    (r"terminal",            "terminal"),
    (r"\btui\b",             "tui"),
    (r"\bcli\b",             "cli"),

    # Containers / infra
    (r"docker",              "docker"),
    (r"kubernetes",          "kubernetes"),
    (r"\bk8s\b",             "kubernetes"),
    (r"terraform",           "terraform"),
    (r"github[- ]actions?",  "github-actions"),
    (r"ci/?cd",              "ci-cd"),
    (r"\bci\b.*\bcd\b",      "ci-cd"),

    # Databases
    (r"database",            "database"),
    (r"postgresql",          "postgresql"),
    (r"postgres",            "postgresql"),
    (r"mongodb",             "mongodb"),
    (r"\bmongo\b",           "mongodb"),
    (r"\bredis\b",           "redis"),

    # API
    (r"\bapi\b",             "api"),
    (r"graphql",             "graphql"),
    (r"\bgrpc\b",            "grpc"),

    # Auth
    (r"authentication",      "authentication"),
    (r"\boauth\b",           "oauth"),
    (r"\bauth\b",            "authentication"),

    # ML / AI
    (r"machine[- ]learning", "machine-learning"),
    (r"\bllm\b",             "llm"),

    # Testing
    (r"testing",             "testing"),
    (r"\be2e\b",             "e2e"),
    (r"end[- ]to[- ]end",   "e2e"),
    (r"browser[- ]automation", "browser-automation"),

    # Misc tooling / services
    (r"\bslack\b",           "slack"),
    (r"\bgit\b",             "git"),
    (r"markdown",            "markdown"),
    (r"\bpdf\b",             "pdf"),
    (r"document",            "document"),
    (r"\bdesign\b",          "design"),
    (r"ui/?ux",              "ui-ux"),
    (r"accessibility",       "accessibility"),
    (r"\bseo\b",             "seo"),
    (r"marketing",           "marketing"),
    (r"deployment",          "deployment"),
    (r"serverless",          "serverless"),
    (r"\baws\b",             "aws"),
    (r"\bazure\b",           "azure"),
    (r"\bgcp\b",             "gcp"),
    (r"cloudflare",          "cloudflare"),
    (r"\bvercel\b",          "vercel"),
    (r"supabase",            "supabase"),
    (r"firebase",            "firebase"),
]

# Pre-compile for performance
_COMPILED_KEYWORDS: list[tuple[re.Pattern, str]] = [
    (re.compile(pattern, re.IGNORECASE), tag)
    for pattern, tag in KEYWORD_TAG_MAP
]


# ---------------------------------------------------------------------------
# Stage 1+2 — tag matching & keyword inference
# ---------------------------------------------------------------------------

def infer_tags(item: dict) -> set[str]:
    """Return inferred tags from item description/name that are NOT already
    in the item's explicit tags."""
    explicit = {t.lower() for t in item.get("tags", [])}
    text = (
        (item.get("description", "") or "")
        + " "
        + (item.get("name", "") or "")
    )

    inferred: set[str] = set()
    for pattern, tag in _COMPILED_KEYWORDS:
        if tag not in explicit and pattern.search(text):
            inferred.add(tag)

    # Avoid overly broad inference: if "react" is inferred but text contains
    # "react native" / "react-native", prefer the more specific tag.
    if "react" in inferred and "react-native" in inferred:
        inferred.discard("react")

    return inferred


def compute_tag_score(
    item_tags: set[str],
    stack_tags: set[str],
    inferred_tags: set[str] | None = None,
) -> tuple[float, list[str], list[str]]:
    """Compute a tag-match score (0-100).

    Uses the geometric mean of two ratios:
        coverage  = overlap / len(item_all_tags)
        precision = overlap / len(stack_tags)

    Inferred tags contribute at ``INFERRED_TAG_DISCOUNT`` (50%) weight.

    Returns (score, matching_explicit, matching_inferred).
    """
    if inferred_tags is None:
        inferred_tags = set()

    stack_lower = {t.lower() for t in stack_tags}
    explicit_lower = {t.lower() for t in item_tags}
    inferred_lower = {t.lower() for t in inferred_tags}

    matching_explicit = sorted(explicit_lower & stack_lower)
    matching_inferred = sorted(inferred_lower & stack_lower)

    if not stack_lower:
        return (0.0, matching_explicit, matching_inferred)

    # Effective overlap: explicit matches count 1, inferred matches count 0.5
    effective_overlap = len(matching_explicit) + len(matching_inferred) * INFERRED_TAG_DISCOUNT

    if effective_overlap == 0:
        return (0.0, matching_explicit, matching_inferred)

    # Effective item-tag count (explicit full, inferred discounted)
    effective_item_count = len(explicit_lower) + len(inferred_lower) * INFERRED_TAG_DISCOUNT

    if effective_item_count == 0:
        return (0.0, matching_explicit, matching_inferred)

    coverage = effective_overlap / effective_item_count
    precision = effective_overlap / len(stack_lower)

    # Geometric mean — sqrt(coverage * precision) clamped to [0, 1]
    geo = math.sqrt(coverage * precision)
    score = min(geo * 100, 100.0)

    return (score, matching_explicit, matching_inferred)


# ---------------------------------------------------------------------------
# Stage 3 — popularity normalization
# ---------------------------------------------------------------------------

def normalize_popularity(
    items: list[dict],
    pop_field: str,
    last_updated_str: str | None = None,
) -> list[dict]:
    """Mutate *items* to add ``_pop_norm`` (0-100, log-scale).

    If *last_updated_str* is an ISO date within ``RECENCY_MONTHS`` months of
    today, each item receives a ``RECENCY_BOOST`` bump (capped at 100).
    """
    values = [item.get(pop_field, 0) for item in items]
    max_val = max(values) if values else 0

    # Determine recency boost applicability
    recency_applicable = False
    if last_updated_str:
        try:
            last_updated = datetime.strptime(last_updated_str, "%Y-%m-%d")
            cutoff = datetime.now() - timedelta(days=RECENCY_MONTHS * 30)
            recency_applicable = last_updated >= cutoff
        except (ValueError, TypeError):
            pass

    for item in items:
        raw = item.get(pop_field, 0)
        if max_val > 0:
            norm = math.log1p(raw) / math.log1p(max_val) * 100
        else:
            norm = 0.0

        if recency_applicable:
            norm = min(norm * (1 + RECENCY_BOOST), 100.0)

        item["_pop_norm"] = norm

    return items


# ---------------------------------------------------------------------------
# Stage 4 — deduplication
# ---------------------------------------------------------------------------

def deduplicate(all_scored: list[dict]) -> list[dict]:
    """Deduplicate entries keyed on ``owner/name`` (lowercased).

    Keeps the entry with the highest ``score``. Adds an ``also_found_in``
    list noting other source keys where the entry appeared.
    """
    best: dict[str, dict] = {}      # key → best entry
    sources: dict[str, list[str]] = {}  # key → [all source keys]

    for entry in all_scored:
        owner = (entry.get("owner") or "").lower().strip()
        name = (entry.get("name") or "").lower().strip()
        key = f"{owner}/{name}"

        source_key = entry.get("source", "")
        sources.setdefault(key, []).append(source_key)

        existing = best.get(key)
        if existing is None or entry.get("score", 0) > existing.get("score", 0):
            best[key] = entry

    result: list[dict] = []
    for key, entry in best.items():
        other_sources = [s for s in sources[key] if s != entry.get("source", "")]
        if other_sources:
            entry["also_found_in"] = sorted(set(other_sources))
        result.append(entry)

    # Preserve descending-score order
    result.sort(key=lambda e: e.get("score", 0), reverse=True)
    return result


# ---------------------------------------------------------------------------
# Combined scorer
# ---------------------------------------------------------------------------

def score_item(
    item: dict,
    stack_tags: set[str],
    pop_field: str,
) -> tuple[float, list[str], list[str]] | None:
    """Score a single ecosystem item against the user's stack.

    Returns ``(final_score, matching_explicit, matching_inferred)`` or
    ``None`` if the item has zero matching tags (hard gate).

    Assumes ``normalize_popularity`` has already been called on the item's
    batch so that ``_pop_norm`` is present.
    """
    explicit_tags = {t.lower() for t in item.get("tags", [])}
    inferred = infer_tags(item)

    tag_score, matching_explicit, matching_inferred = compute_tag_score(
        explicit_tags, stack_tags, inferred,
    )

    # Hard gate: no matches at all → exclude
    if not matching_explicit and not matching_inferred:
        return None

    pop_score = item.get("_pop_norm", 0)

    final_score = (tag_score * RELEVANCE_WEIGHT) + (pop_score * POPULARITY_WEIGHT)

    return (round(final_score, 2), matching_explicit, matching_inferred)
