# Plum

A fuzzy-search TUI plugin browser for Claude Code.

## Features

- Fuzzy search across all plugins from known marketplaces
- Filter by All / Available / Installed
- Card and simple list view modes
- View plugin details (version, author, description, keywords)
- Copy install commands to clipboard
- Visual distinction between installed (●) and available (○) plugins
- fzf-style keyboard navigation
- Responsive UI adapts to terminal width

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
| `↑` / `Ctrl+k` / `Ctrl+p` | Move up |
| `↓` / `Ctrl+j` / `Ctrl+n` | Move down |
| `Ctrl+u` / `PgUp` | Page up |
| `Ctrl+d` / `PgDn` | Page down |
| `Home` | Jump to top |
| `End` | Jump to bottom |

### Filtering & Display

| Key | Action |
|-----|--------|
| `Tab` | Next filter (All → Available → Installed) |
| `Shift+Tab` | Previous filter |
| `Ctrl+v` | Toggle view mode (card / simple) |
| `Ctrl+t` | Cycle transition style (instant / zoom / slide) |

### Actions

| Key | Action |
|-----|--------|
| `Enter` | View plugin details |
| `c` | Copy install command (in detail view) |
| `Esc` / `Ctrl+g` | Clear search or quit |
| `?` | Show help |
| `Ctrl+c` | Quit |

### Search

Just start typing — all keys go to search input. Use `Ctrl+key` for navigation.

## Views

### List View
Main view showing all plugins with filter tabs. Plugins displayed as cards (default) or simple one-line list.
- `●` = Installed
- `○` = Available

### Detail View
Full plugin info: version, author, marketplace, category, description, keywords, and install command. Press `c` to copy the install command.

### Help View
Quick reference for keyboard shortcuts.

## Tech Stack

- Go 1.24
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Styling
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [Harmonica](https://github.com/charmbracelet/harmonica) - Spring animations

## License

MIT
