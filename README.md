# try-go

**A Go port of [tobi/try](https://github.com/tobi/try)** - fresh directories for every vibe 🏠

> This is a pure Go implementation of [Tobi Lütke's](https://github.com/tobi) brilliant `try` tool.
> All credit for the original idea, design, and functionality goes to [the original Ruby version](https://github.com/tobi/try).

[![Go Report Card](https://goreportcard.com/badge/github.com/YOUR_USERNAME/try-go)](https://goreportcard.com/report/github.com/YOUR_USERNAME/try-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## Why a Go Port?

- **Single binary** - No Ruby runtime required
- **Cross-platform** - Pre-built binaries for macOS, Linux, Windows
- **Same features** - 100% compatible with the original
- **Fast** - Native compiled performance

## What It Does

![Demo](https://raw.githubusercontent.com/tobi/try/main/docs/try-fuzzy-search-demo.gif)

*Demo from the [original try](https://github.com/tobi/try)*

Instantly navigate through all your experiment directories with:

- **Fuzzy search** that just works
- **Smart sorting** - recently used stuff bubbles to the top
- **Auto-dating** - creates directories like `2025-08-17-redis-experiment`
- **Zero config** - just one binary, no dependencies

## Installation

### Quick Install (Recommended)

```bash
# macOS/Linux
curl -fsSL https://raw.githubusercontent.com/YOUR_USERNAME/try-go/main/install.sh | bash
```

### Go Install

```bash
go install github.com/YOUR_USERNAME/try-go@latest
```

### From Source

```bash
git clone https://github.com/YOUR_USERNAME/try-go.git
cd try-go
make build
sudo mv try /usr/local/bin/
```

### Pre-built Binaries

Download from [Releases](https://github.com/YOUR_USERNAME/try-go/releases):

| Platform | Architecture | Download |
|----------|--------------|----------|
| macOS    | Intel (x64)  | [try-darwin-amd64](https://github.com/YOUR_USERNAME/try-go/releases/latest) |
| macOS    | Apple Silicon (arm64) | [try-darwin-arm64](https://github.com/YOUR_USERNAME/try-go/releases/latest) |
| Linux    | x64          | [try-linux-amd64](https://github.com/YOUR_USERNAME/try-go/releases/latest) |
| Linux    | arm64        | [try-linux-arm64](https://github.com/YOUR_USERNAME/try-go/releases/latest) |
| Windows  | x64          | [try-windows-amd64.exe](https://github.com/YOUR_USERNAME/try-go/releases/latest) |

### Homebrew (Coming Soon)

```bash
brew install YOUR_USERNAME/tap/try-go
```

## Shell Setup

After installation, add to your shell config:

### Bash / Zsh

```bash
# Add to ~/.bashrc or ~/.zshrc
eval "$(try init)"

# Or specify a custom path:
eval "$(try init ~/experiments)"
```

### Fish

```fish
# Add to ~/.config/fish/config.fish
eval (try init | string collect)

# Or specify a custom path:
eval (try init ~/experiments | string collect)
```

**Default directory:** `~/src/tries`

## Usage

```bash
try                        # Browse all experiments
try redis                  # Jump to redis experiment or create new
try new api                # Create "2025-01-21-api"
try .                      # Create dated worktree for current repo
try ./path/to/repo         # Create worktree from another repo
try clone https://...      # Clone repo into dated directory
try https://github.com/... # Shorthand for clone
try --help                 # See all options
```

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate |
| `Ctrl-P` / `Ctrl-N` | Navigate (vim-style) |
| `Ctrl-J` / `Ctrl-K` | Navigate (vim-style) |
| `Enter` | Select or create |
| `Backspace` | Delete character |
| `Ctrl-U` | Clear input |
| `Ctrl-D` | Delete directory (with confirmation) |
| `Ctrl-R` | Rename directory |
| `ESC` | Cancel |

### Git Repository Cloning

```bash
# Clone with auto-generated directory name
try clone https://github.com/tobi/try.git
# Creates: 2025-01-21-tobi-try

# Clone with custom name
try clone https://github.com/tobi/try.git my-fork
# Creates: my-fork

# Shorthand (detects git URLs automatically)
try https://github.com/tobi/try.git
```

Supported URL formats:
- `https://github.com/user/repo.git`
- `git@github.com:user/repo.git`
- `https://gitlab.com/user/repo.git`

## Configuration

Set `TRY_PATH` to change where experiments are stored:

```bash
export TRY_PATH=~/code/sketches
```

Default: `~/src/tries`

## Features

### 🎯 Smart Fuzzy Search

Not just substring matching - it's smart:
- `rds` matches `redis-server`
- `connpool` matches `connection-pool`
- Recent stuff scores higher
- Shorter names win on equal matches

### ⏰ Time-Aware

- Shows how long ago you touched each project
- Recently accessed directories float to the top
- Perfect for "what was I working on yesterday?"

### 🎨 Beautiful TUI

- Clean, minimal interface
- Highlights matches as you type
- Shows relative time (2h, 3d, 2w)
- Respects `NO_COLOR` environment variable

### 📁 Organized Chaos

- Everything in one place (`~/src/tries` by default)
- Auto-prefixes with dates: `2025-01-21-your-idea`
- Git worktree support for quick branching

## The Philosophy

> Your brain doesn't work in neat folders. You have ideas, you try things, you context-switch like a caffeinated squirrel. This tool embraces that.

Every experiment gets a home. Every home is instantly findable. Your 2am coding sessions are no longer lost to the void.

## Credits

**All credit goes to [Tobi Lütke](https://github.com/tobi) and the original [try](https://github.com/tobi/try) project.**

This is simply a Go port to provide:
- A single binary distribution
- No Ruby dependency
- Cross-platform pre-built binaries

The original Ruby version is excellent and you should check it out: **[github.com/tobi/try](https://github.com/tobi/try)**

## License

MIT License - Same as the [original project](https://github.com/tobi/try/blob/main/LICENSE).

See [LICENSE](LICENSE) for details.

---

*Built for developers with ADHD by developers with ADHD.*

*Your experiments deserve a home.* 🏠
