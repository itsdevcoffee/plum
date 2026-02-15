#!/usr/bin/env python3
"""Recommend ecosystem skills, plugins, agents, and MCP servers for a project.

Analyzes the project's tech stack and scores entries from ecosystem JSON files
using the multi-stage scorer engine (scorer.py).

Zero external dependencies - stdlib only.

Usage:
    python3 scripts/ecosystem/recommend.py [TARGET_DIR] [--top N] [--min-score N] [--pretty]
    python3 scripts/ecosystem/recommend.py [TARGET_DIR] --markdown [--exclude-no-install]
"""

import json
from pathlib import Path

from scorer import score_item, normalize_popularity, deduplicate

# Resolve ecosystem dir relative to this script: ../../ecosystem/
SCRIPT_DIR = Path(__file__).resolve().parent
ECOSYSTEM_DIR = SCRIPT_DIR.parent.parent / "ecosystem"

# Source file config: (filename, items_key, popularity_field, output_key)
SOURCES = [
    ("skills-leaderboard.json", "skills",       "installs",  "skills_leaderboard"),
    ("plugins.json",            "repositories",  "stars",     "plugins"),
    ("agents.json",             "repositories",  "stars",     "agents"),
    ("skills.json",             "repositories",  "stars",     "skills"),
    ("mcp-servers.json",        "collections",   "stars",     "mcp_servers"),
]

DEFAULT_TOP = 5
DEFAULT_MIN_SCORE = 20

# Category label mapping for markdown output
CATEGORY_LABELS = {
    "skills_leaderboard": "Top Skills (skills.sh)",
    "plugins":            "Plugins",
    "agents":             "Agents",
    "skills":             "Skill Repositories",
    "mcp_servers":        "MCP Servers",
}


def _load_json(filepath):
    """Load a JSON file, return empty dict on failure."""
    try:
        return json.loads(Path(filepath).read_text(errors="ignore"))
    except (OSError, json.JSONDecodeError):
        return {}


def _format_entry(item, score, matching_tags, inferred_tags, pop_field, source_label):
    """Format a scored item into the output structure."""
    entry = {
        "name": item.get("name", ""),
        "owner": item.get("owner", ""),
        "description": item.get("description", ""),
        "score": round(score, 1),
        "matching_tags": matching_tags,
        "inferred_tags": inferred_tags,
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


def _format_markdown(result):
    """Render the recommendation result as markdown tables."""
    lines = []

    # Stack profile header
    profile = result.get("stack_profile", {})
    lines.append("## Stack Profile")
    lines.append("")
    lines.append(f"**Languages:** {', '.join(profile.get('languages', [])) or 'none'}")
    lines.append(f"**Frameworks:** {', '.join(profile.get('frameworks', [])) or 'none'}")
    lines.append(f"**Tags:** {', '.join(profile.get('tags', [])) or 'none'}")
    lines.append("")

    recommendations = result.get("recommendations", {})

    # Iterate sources in order to maintain consistent output
    for _filename, _items_key, pop_field, output_key in SOURCES:
        entries = recommendations.get(output_key)
        if not entries:
            continue

        label = CATEGORY_LABELS.get(output_key, output_key)
        pop_col = pop_field.capitalize()

        lines.append(f"## {label}")
        lines.append("")
        lines.append(f"| Name | Score | {pop_col} | Tags | Install |")
        lines.append("|------|-------|" + "-" * (len(pop_col) + 2) + "|------|---------|")

        for entry in entries:
            name = entry.get("name", "")
            score = entry.get("score", 0)

            # Get popularity value
            pop_metric = entry.get("popularity_metric", {})
            pop_val = pop_metric.get(pop_field, 0)
            pop_str = f"{pop_val:,}"

            # Build tags column: matching + inferred (prefixed with ~)
            tag_parts = list(entry.get("matching_tags", []))
            for t in entry.get("inferred_tags", []):
                tag_parts.append(f"~{t}")
            tags_str = ", ".join(tag_parts) if tag_parts else "-"

            # Install column
            install_cmd = entry.get("install_cmd")
            install_str = f"`{install_cmd}`" if install_cmd else "-"

            lines.append(f"| {name} | {score} | {pop_str} | {tags_str} | {install_str} |")

        lines.append("")

    if not recommendations:
        lines.append("*No recommendations found above the score threshold.*")
        lines.append("")

    return "\n".join(lines)


def recommend(target_dir, top_n=DEFAULT_TOP, min_score=DEFAULT_MIN_SCORE,
              exclude_no_install=False):
    """Run recommendation engine on the target directory.

    Returns structured JSON-serializable dict.
    """
    # Import detect_stack from same package
    from detect_stack import detect_stack

    stack_profile = detect_stack(target_dir)
    if "error" in stack_profile:
        return {"error": stack_profile["error"]}

    # Build stack_tags set: languages + tags + frameworks, all lowercased
    stack_tags = set()
    for val in stack_profile.get("languages", []):
        stack_tags.add(val.lower())
    for val in stack_profile.get("tags", []):
        stack_tags.add(val.lower())
    for val in stack_profile.get("frameworks", []):
        stack_tags.add(val.lower())

    # Collect all scored entries across all sources
    all_scored = []

    for filename, items_key, pop_field, output_key in SOURCES:
        filepath = ECOSYSTEM_DIR / filename
        data = _load_json(filepath)
        items = data.get(items_key, [])

        if not items:
            continue

        # Get last_updated from the source data for recency boost
        last_updated = data.get("last_updated")

        # Stage 3: normalize popularity with recency
        normalize_popularity(items, pop_field, last_updated)

        # Stage 1+2+combined: score each item
        for item in items:
            result = score_item(item, stack_tags, pop_field)
            if result is None:
                continue

            final_score, matching_tags, inferred_tags = result

            if final_score >= min_score:
                entry = _format_entry(
                    item, final_score, matching_tags, inferred_tags,
                    pop_field, output_key,
                )
                all_scored.append(entry)

    # Stage 4: deduplicate across all sources
    deduplicated = deduplicate(all_scored)

    # Apply --exclude-no-install filter AFTER dedup but BEFORE grouping
    if exclude_no_install:
        deduplicated = [e for e in deduplicated if e.get("install_cmd")]

    # Re-group by source, sort by score, apply top_n per category
    recommendations = {}
    total_matches = 0

    for entry in deduplicated:
        source = entry.get("source", "")
        recommendations.setdefault(source, []).append(entry)

    for source in recommendations:
        recommendations[source].sort(key=lambda e: e.get("score", 0), reverse=True)
        recommendations[source] = recommendations[source][:top_n]
        total_matches += len(recommendations[source])

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
    parser.add_argument("--top", type=int, default=DEFAULT_TOP,
                        help=f"Top N per category (default: {DEFAULT_TOP})")
    parser.add_argument("--min-score", type=int, default=DEFAULT_MIN_SCORE,
                        help=f"Minimum score threshold (default: {DEFAULT_MIN_SCORE})")
    parser.add_argument("--exclude-no-install", action="store_true",
                        help="Filter out entries without an install command")

    # Output format: JSON by default, --markdown for tables
    parser.add_argument("--markdown", action="store_true",
                        help="Output as formatted markdown tables (default: JSON)")

    parser.add_argument("--pretty", action="store_true", help="Pretty-print JSON output")

    args = parser.parse_args()

    result = recommend(
        args.target_dir,
        top_n=args.top,
        min_score=args.min_score,
        exclude_no_install=args.exclude_no_install,
    )

    if args.markdown:
        print(_format_markdown(result))
    else:
        indent = 2 if args.pretty else None
        print(json.dumps(result, indent=indent))


if __name__ == "__main__":
    main()
