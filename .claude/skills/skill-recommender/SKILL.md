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
   python3 scripts/ecosystem/recommend.py [TARGET_DIR] --pretty
   ```
   If the user asks for more results, use `--top N` with a higher number.

3. Parse the JSON output and present results as organized markdown.

4. Format the output as follows:

   **Stack Profile** - Show detected languages, frameworks, and key tags in a brief summary.

   **Recommendations** - For each category that has matches, show a markdown table:

   | Name | Score | Popularity | Matching Tags | Install |
   |------|-------|-----------|---------------|---------|

   Only show categories that have results. Use these category labels:
   - `skills_leaderboard` → "Top Skills (skills.sh)"
   - `plugins` → "Plugins"
   - `agents` → "Agents"
   - `skills` → "Skill Repositories"
   - `mcp_servers` → "MCP Servers"

5. After the tables, add a brief "You might also like" section with 1-2 contextual suggestions based on the stack (e.g., "Since you're using BubbleTea, check out the Charm ecosystem tools" or "Consider adding a CI/CD plugin for your GitHub Actions workflow").

6. If the script fails, fall back to reading the ecosystem JSON files directly from `ecosystem/` and manually matching tags from the project's config files.

## Notes

- The script uses zero external dependencies (Python stdlib only)
- Scoring: 70% relevance (tag match) + 30% popularity (log-scaled)
- Default shows top 5 per category with minimum score of 15
