# Plum Ecosystem Knowledge Base

**Purpose**: Machine-readable reference maps of the Claude Code ecosystem for intelligent skill/plugin recommendations and automated discovery.

## 📁 Directory Structure

```
ecosystem/
├── README.md                    # This file
├── metadata.json                # Stats, last updated, totals
├── plugins.json                 # Plugin repository index
├── agents.json                  # Agent repository index
├── skills.json                  # Skills repository index
├── skills-leaderboard.json      # Top skills from skills.sh
├── mcp-servers.json            # MCP server index
├── marketplaces.json           # Marketplace directory
└── research/                    # Detailed research reports
    ├── 2026-02-15-claude-code-ecosystem-discovery.md
    └── 2026-02-15-claude-skills-ecosystem.md
```

## 🎯 Purpose

This directory serves as a **living knowledge base** that:

1. **Enables Fast Queries**: Agents can query compact JSON files instead of reading 500+ line markdown docs
2. **Powers Recommendations**: Context-aware skill/plugin suggestions based on current repo
3. **Stays Current**: Periodic updates via `ecosystem-update` skill
4. **Reduces Context**: Only load what you need (filter by tags, category, popularity)

## 📊 JSON Schema

All JSON files follow consistent patterns:

### Common Fields
- `name` - Repository/skill name
- `owner` - GitHub owner or organization
- `stars` / `installs` - Popularity metric
- `category` - Type/classification
- `tags` - Filterable keywords
- `description` - Brief purpose
- `url` - Link to resource
- `install_cmd` - Quick install command

### Example Entry
```json
{
  "name": "find-skills",
  "owner": "vercel-labs",
  "repo": "vercel-labs/skills",
  "installs": 231100,
  "category": "meta",
  "tags": ["discovery", "essential"],
  "description": "Skill discovery and management",
  "install_cmd": "npx skillsadd vercel-labs/skills/find-skills"
}
```

## 🔍 Usage Examples

### For Agents/Skills

**Recommend React skills:**
```python
import json

with open('ecosystem/skills-leaderboard.json') as f:
    data = json.load(f)
    react_skills = [s for s in data['skills'] if 'react' in s['tags']]
    top_5 = sorted(react_skills, key=lambda x: x['installs'], reverse=True)[:5]
```

**Find popular plugins:**
```python
with open('ecosystem/plugins.json') as f:
    data = json.load(f)
    popular = [p for p in data['repositories'] if p['stars'] > 10000]
```

**Get MCP servers by category:**
```python
with open('ecosystem/mcp-servers.json') as f:
    data = json.load(f)
    slack_servers = [s for s in data['servers'] if 'slack' in s['tags']]
```

### For Humans

Read detailed analysis in `research/` directory:
- Full context and insights
- Ecosystem trends and patterns
- Strategic recommendations

## 🔄 Updating Data

### Manual Update
Run the ecosystem update skill:
```bash
# Future: once skill is built
/ecosystem-update
```

### What Gets Updated
- New repositories discovered via GitHub search
- Star counts refreshed for existing entries
- skills.sh leaderboard rankings
- Metadata timestamps and totals

### Update Frequency
- **Recommended**: Weekly
- **Minimum**: Monthly
- **After Major Events**: New Anthropic releases, hackathons, etc.

## 📋 Data Sources

All data sourced from:
- **GitHub Search API** - Repository discovery
- **skills.sh** - Skill installation rankings
- **Official Docs** - Anthropic documentation
- **Community Resources** - Awesome lists, marketplaces

## 🎓 File Descriptions

| File | Purpose | Entries |
|------|---------|---------|
| `metadata.json` | Stats, timestamps, totals | 1 object |
| `plugins.json` | Plugin repositories | 50+ repos |
| `agents.json` | Agent/subagent repos | 30+ repos |
| `skills.json` | Skill collections | 24+ repos |
| `skills-leaderboard.json` | Top skills by installs | 25+ skills |
| `mcp-servers.json` | MCP server collections | 15+ collections |
| `marketplaces.json` | Marketplace directories | 10+ marketplaces |

## 🏗️ Future Enhancements

- [ ] Tag taxonomy (standardize tags across files)
- [ ] Compatibility matrix (which skills work with which frameworks)
- [ ] Historical trends (track star growth over time)
- [ ] Quality scores (combine stars, installs, recency)
- [ ] Dependency graphs (skill prerequisites)
- [ ] Auto-detection scripts (detect tech stack from repo)

## 📝 Contributing

When adding new entries:
1. Follow the schema consistently
2. Include all required fields
3. Sort by popularity (stars/installs descending)
4. Update metadata.json totals
5. Add research notes to `research/` if needed

---

**Last Updated**: 2026-02-15
**Maintained By**: Plum ecosystem update workflows
