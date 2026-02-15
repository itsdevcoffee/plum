---
name: skill-recommender
description: Analyze current project's tech stack and recommend relevant skills, plugins, agents, and MCP servers from the plum ecosystem data
triggers:
  - recommend skills
  - what plugins should I use
  - suggest tools
  - find relevant skills
  - what should I install
  - discover plugins
  - recommend plugins
  - what tools should I use
---

# Skill Recommender

Analyze the current project's tech stack and recommend relevant ecosystem tools.

## Instructions

1. Determine the target directory. Default to the current working directory unless the user specifies a different path.

2. Run the recommender script:
   ```bash
   python3 scripts/ecosystem/recommend.py [TARGET_DIR] --markdown
   ```

   **Flags:**
   - `--markdown` — Pre-formatted markdown tables (default for this skill)
   - `--top N` — Show more results per category (default: 5). Use `--top 10` if the user asks for more.
   - `--exclude-no-install` — Only show entries with install commands. Use when the user asks "what should I install" or wants actionable results.
   - `--min-score N` — Adjust score threshold (default: 35). Lower to 20 for broader results.
   - `--pretty` — Pretty-print JSON (use instead of `--markdown` if you need to parse the output programmatically)

3. Present the markdown output directly to the user. The script handles formatting, grouping, and deduplication.

4. Tags prefixed with `~` in the output are inferred from descriptions (not explicit in the ecosystem data). Mention this if the user asks about tag accuracy.

5. After the tables, add a brief "You might also like" section with 1-2 contextual suggestions based on the stack (e.g., "Since you're using BubbleTea, check out the Charm ecosystem tools" or "Consider adding a CI/CD plugin for your GitHub Actions workflow").

6. If the script fails, fall back to reading the ecosystem JSON files directly from `ecosystem/` and match entries that share at least one tag with the project's detected stack. Do not recommend entries with zero tag overlap.

## Notes

- Zero external dependencies (Python stdlib only)
- Scoring: 70% relevance (bidirectional tag match, geometric mean) + 30% popularity (log-scaled with recency boost)
- Hard gate: entries with zero matching tags (explicit or inferred) are excluded
- Deduplication: same repo across sources is shown once (highest score wins)
- Default: top 5 per category, minimum score 35
