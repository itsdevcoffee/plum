# 🍑 Plum

**A better way to discover Claude Code marketplace plugins.**

Plum is a fast, fuzzy-search TUI that helps you browse, search, and install plugins from all your configured Claude Code marketplaces in one place.

![List View](assets/list-view-card.png)

## Popular Marketplaces

Plum works with any Claude Code marketplace. Here are some popular ones to get started:

<table>
<tr>
<th width="280">Marketplace</th>
<th>Description</th>
</tr>
<tr>
<td><a href="https://github.com/jeremylongshore/claude-code-plugins">claude-code-plugins-plus</a></td>
<td>The largest collection with <strong>254 plugins</strong> and 185 Agent Skills, focusing on production-ready automation tools across DevOps, security, testing, and AI/ML workflows.</td>
</tr>
<tr>
<td><a href="https://github.com/ananddtyagi/claude-code-marketplace">claude-code-marketplace</a></td>
<td>Community-driven marketplace featuring curated commands and agents with granular installation and auto-sync from a live database. Browse by category and install only what you need.</td>
</tr>
<tr>
<td><a href="https://github.com/anthropics/claude-code">claude-code-plugins</a></td>
<td>Official Anthropic plugins that extend Claude Code's core functionality. These plugins are maintained by the Claude Code team and ship with the tool.</td>
</tr>
<tr>
<td><a href="https://github.com/MadAppGang/claude-code">mag-claude-plugins</a></td>
<td>Battle-tested workflows from top developers with <strong>4 specialized plugins</strong> for frontend development, code analysis, Bun backend, and orchestration patterns.</td>
</tr>
<tr>
<td><a href="https://github.com/Dev-GOM/claude-code-marketplace">dev-gom-plugins</a></td>
<td>Automation-focused collection with <strong>15 plugins</strong> specializing in Unity game development, Blender 3D workflows, browser automation, and code quality monitoring.</td>
</tr>
<tr>
<td><a href="https://github.com/feed-mob/claude-code-marketplace">feedmob-plugins</a></td>
<td>Productivity and workflow tools with <strong>6 specialized plugins</strong> for data processing (CSV parsing), testing, commit automation, presentation generation, and AI news aggregation.</td>
</tr>
<tr>
<td><a href="https://github.com/anthropics/skills">anthropic-agent-skills</a></td>
<td>Official Anthropic Agent Skills reference repository with document manipulation capabilities (PDF, DOCX, PPTX, XLSX) and production-quality skill implementation examples.</td>
</tr>
</table>

**Have a marketplace?** Submit a PR to add it to this list! We welcome all Claude Code plugin marketplaces.

## Installation

```bash
go install github.com/itsdevcoffee/plum/cmd/plum@latest
```

Then run:

```bash
plum
```

**Requirements:** [Claude Code](https://claude.ai/claude-code) must be installed and configured with at least one marketplace.

## Key Features

- **Instant fuzzy search** across all your marketplaces
- **Filter by status**: All, Available, or Installed plugins
- **Multiple view modes**: Card (detailed) or Slim (compact)
- **One-click install commands** - press `c` to copy
- **Responsive design** that adapts to your terminal size

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| Type anything | Search plugins |
| `↑↓` or `Ctrl+j/k` | Navigate |
| `Enter` | View details |
| `Tab` | Cycle filters (All/Available/Installed) |
| `Ctrl+v` | Toggle card/slim view |
| `c` | Copy install command (in detail view) |
| `?` | Show help |
| `Esc` or `q` | Quit |

## Screenshots

### Fuzzy Search
Type to instantly filter plugins across all marketplaces:

![Search](assets/search-fuzzy.png)

### Plugin Details
View comprehensive information with one-click install commands:

![Detail View](assets/detail-view.png)

### Multiple View Modes
Switch between card and slim views with `Ctrl+v`:

<table>
<tr>
<td width="50%">

**Card View** (Default)
<img src="assets/list-view-card.png" alt="Card View">

</td>
<td width="50%">

**Slim View** (Compact)
<img src="assets/list-view-slim.png" alt="Slim View">

</td>
</tr>
</table>

### Built-in Help
Press `?` to see all keyboard shortcuts:

![Help Menu](assets/help-menu.png)

## Building from Source

```bash
git clone https://github.com/itsdevcoffee/plum.git
cd plum
go build -o plum ./cmd/plum
./plum
```

## Troubleshooting

**"Claude Code settings not found"**
- Run `claude-code` at least once to initialize your configuration

**"No plugins found"**
- Make sure you have marketplaces configured
- Run `/plugin` in Claude Code to browse and add marketplaces
- Run `/plugin marketplace update` to sync

**Custom config directory**
- Set `CLAUDE_CONFIG_DIR` environment variable if you use a non-standard location

## Contributing

Contributions are welcome! Whether it's:
- Adding your marketplace to the Popular Marketplaces list
- Reporting bugs or suggesting features
- Improving documentation
- Submitting code improvements

Feel free to open an issue or pull request.

## License

MIT - see [LICENSE](LICENSE) for details.

---

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) • Styled with [Lip Gloss](https://github.com/charmbracelet/lipgloss)
