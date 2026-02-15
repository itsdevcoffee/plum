#!/usr/bin/env python3
"""Recommend ecosystem skills, plugins, agents, and MCP servers for a project.

Analyzes the project's tech stack and scores entries from ecosystem JSON files.
Zero external dependencies - stdlib only.

Usage:
    python3 scripts/ecosystem/recommend.py [TARGET_DIR] [--top N] [--min-score N] [--pretty]
"""

import json
import math
import os
import sys
from pathlib import Path

# Resolve ecosystem dir relative to this script: ../../ecosystem/
SCRIPT_DIR = Path(__file__).resolve().parent
ECOSYSTEM_DIR = SCRIPT_DIR.parent.parent / "ecosystem"

# Source file config: (filename, items_key, popularity_field)
SOURCES = [
    ("skills-leaderboard.json", "skills",       "installs",  "skills_leaderboard"),
    ("plugins.json",            "repositories",  "stars",     "plugins"),
    ("agents.json",             "repositories",  "stars",     "agents"),
    ("skills.json",             "repositories",  "stars",     "skills"),
    ("mcp-servers.json",        "collections",   "stars",     "mcp_servers"),
]

# Weights
RELEVANCE_WEIGHT = 0.7
POPULARITY_WEIGHT = 0.3
TAG_OVERLAP_WEIGHT = 0.8
KEYWORD_HIT_WEIGHT = 0.2
DEFAULT_TOP = 5
DEFAULT_MIN_SCORE = 15


def _load_json(filepath):
    """Load a JSON file, return empty dict on failure."""
    try:
        return json.loads(Path(filepath).read_text(errors="ignore"))
    except (OSError, json.JSONDecodeError):
        return {}


def _normalize_popularity(items, pop_field):
    """Add normalized popularity (0-100, log-scale) to each item."""
    values = [item.get(pop_field, 0) for item in items]
    max_val = max(values) if values else 0

    for item in items:
        raw = item.get(pop_field, 0)
        if max_val > 0:
            item["_pop_norm"] = math.log1p(raw) / math.log1p(max_val) * 100
        else:
            item["_pop_norm"] = 0
    return items


def _compute_relevance(item, stack_tags, stack_keywords):
    """Compute relevance score (0-100) for an item against the stack profile.

    - Tag overlap ratio (weighted 80%)
    - Description keyword hits (weighted 20%)
    """
    item_tags = set(t.lower() for t in item.get("tags", []))
    stack_tag_set = set(t.lower() for t in stack_tags)

    # Tag overlap: what fraction of the item's tags match the stack
    if item_tags:
        overlap = item_tags & stack_tag_set
        tag_score = len(overlap) / len(item_tags) * 100
        matching_tags = sorted(overlap)
    else:
        tag_score = 0
        matching_tags = []

    # Description keyword hits
    desc = (item.get("description", "") + " " + item.get("name", "")).lower()
    if stack_keywords:
        hits = sum(1 for kw in stack_keywords if kw in desc)
        keyword_score = min(hits / max(len(stack_keywords) * 0.3, 1) * 100, 100)
    else:
        keyword_score = 0

    relevance = (tag_score * TAG_OVERLAP_WEIGHT) + (keyword_score * KEYWORD_HIT_WEIGHT)
    return relevance, matching_tags


def _build_keywords(stack_profile):
    """Build keyword set from the stack profile for description matching."""
    keywords = set()
    for lang in stack_profile.get("languages", []):
        keywords.add(lang.lower())
    for tag in stack_profile.get("tags", []):
        keywords.add(tag.lower())
    for fw in stack_profile.get("frameworks", []):
        keywords.add(fw.lower())
    # Remove very generic terms
    keywords.discard("backend")
    keywords.discard("make")
    return keywords


def _format_entry(item, score, matching_tags, pop_field, source_label):
    """Format a scored item into the output structure."""
    entry = {
        "name": item.get("name", ""),
        "owner": item.get("owner", ""),
        "description": item.get("description", ""),
        "score": round(score, 1),
        "matching_tags": matching_tags,
        "popularity_metric": {pop_field: item.get(pop_field, 0)},
        "source": source_label,
    }
    install = item.get("install_cmd")
    if install:
        entry["install_cmd"] = install
    url = item.get("url") or item.get("repo")
    if url:
        entry["url"] = url
    return entry


def recommend(target_dir, top_n=DEFAULT_TOP, min_score=DEFAULT_MIN_SCORE):
    """Run recommendation engine on the target directory.

    Returns structured JSON-serializable dict.
    """
    # Import detect_stack from same package
    from detect_stack import detect_stack

    stack_profile = detect_stack(target_dir)
    if "error" in stack_profile:
        return {"error": stack_profile["error"]}

    stack_tags = stack_profile["tags"] + stack_profile["languages"] + stack_profile["frameworks"]
    stack_keywords = _build_keywords(stack_profile)

    recommendations = {}
    total_matches = 0

    for filename, items_key, pop_field, output_key in SOURCES:
        filepath = ECOSYSTEM_DIR / filename
        data = _load_json(filepath)
        items = data.get(items_key, [])

        if not items:
            continue

        items = _normalize_popularity(items, pop_field)

        scored = []
        for item in items:
            relevance, matching_tags = _compute_relevance(item, stack_tags, stack_keywords)

            if relevance < 1:
                continue

            pop_score = item.get("_pop_norm", 0)
            final_score = (relevance * RELEVANCE_WEIGHT) + (pop_score * POPULARITY_WEIGHT)

            if final_score >= min_score:
                scored.append((final_score, matching_tags, item))

        scored.sort(key=lambda x: x[0], reverse=True)
        top_items = scored[:top_n]

        if top_items:
            recommendations[output_key] = [
                _format_entry(item, score, mtags, pop_field, output_key)
                for score, mtags, item in top_items
            ]
            total_matches += len(top_items)

    return {
        "target_dir": str(Path(target_dir).resolve()),
        "stack_profile": {
            "languages": stack_profile["languages"],
            "tags": stack_profile["tags"],
            "frameworks": stack_profile["frameworks"],
        },
        "recommendations": recommendations,
        "total_matches": total_matches,
    }


def main():
    import argparse

    parser = argparse.ArgumentParser(description="Recommend ecosystem tools for a project")
    parser.add_argument("target_dir", nargs="?", default=".", help="Project directory to analyze")
    parser.add_argument("--top", type=int, default=DEFAULT_TOP, help=f"Top N per category (default: {DEFAULT_TOP})")
    parser.add_argument("--min-score", type=int, default=DEFAULT_MIN_SCORE, help=f"Minimum score threshold (default: {DEFAULT_MIN_SCORE})")
    parser.add_argument("--pretty", action="store_true", help="Pretty-print JSON output")

    args = parser.parse_args()
    result = recommend(args.target_dir, top_n=args.top, min_score=args.min_score)

    indent = 2 if args.pretty else None
    print(json.dumps(result, indent=indent))


if __name__ == "__main__":
    main()
