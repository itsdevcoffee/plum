# Plum

A fuzzy-search TUI plugin browser for Claude Code.

## Features

- Fuzzy search across all plugins from known marketplaces
- View plugin details (version, author, description, keywords)
- Copy install commands to clipboard
- Visual distinction between installed/available plugins
- Vim-style keyboard navigation

## Installation

```bash
go install github.com/maskkiller/plum/cmd/plum@latest
```

Or build from source:

```bash
git clone https://github.com/maskkiller/plum.git
cd plum
go build -o plum ./cmd/plum
```

## Requirements

- Claude Code installed with `~/.claude/plugins/` directory
- At least one marketplace configured

## Usage

```bash
./plum
```

Start typing to fuzzy search plugins. Results update in real-time.

## Keyboard Shortcuts

### Navigation

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `g` / `Home` | Jump to top |
| `G` / `End` | Jump to bottom |
| `Ctrl+u` / `PgUp` | Page up |
| `Ctrl+d` / `PgDn` | Page down |

### Actions

| Key | Action |
|-----|--------|
| `Enter` | View plugin details |
| `c` | Copy install command |
| `Esc` | Clear search / Go back |
| `?` | Show help |
| `q` | Quit |

### Search

Just start typing — no need to focus the input. Press `Esc` to clear.

## Views

### List View
Main view showing all plugins. Installed plugins marked with `✓`.

### Detail View
Full plugin info: version, author, description, keywords, install command.

### Help View
Quick reference for keyboard shortcuts.

## Tech Stack

- Go 1.24
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Styling
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components

## License

MIT
