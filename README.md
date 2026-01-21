# try

**A Go port of [tobi/try](https://github.com/tobi/try)** - fresh directories for every vibe 🏠

> All credit for the original idea goes to [Tobi Lütke](https://github.com/tobi) and the [original Ruby version](https://github.com/tobi/try).

[![Go Report Card](https://goreportcard.com/badge/github.com/amulcse/try)](https://goreportcard.com/report/github.com/amulcse/try)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## Quick Start (No Cloning Required!)

### One-Line Install

**macOS / Linux:**
```bash
curl -fsSL https://raw.githubusercontent.com/amulcse/try/main/install.sh | bash
```

### Manual Binary Download

Download the binary for your platform, make it executable, and move to your PATH:

**macOS (Apple Silicon):**
```bash
curl -L -o try.tar.gz $(curl -s https://api.github.com/repos/amulcse/try/releases/latest | grep "browser_download_url.*darwin_arm64.tar.gz" | cut -d '"' -f 4)
tar -xzf try.tar.gz && chmod +x try && sudo mv try /usr/local/bin/
```

**macOS (Intel):**
```bash
curl -L -o try.tar.gz $(curl -s https://api.github.com/repos/amulcse/try/releases/latest | grep "browser_download_url.*darwin_amd64.tar.gz" | cut -d '"' -f 4)
tar -xzf try.tar.gz && chmod +x try && sudo mv try /usr/local/bin/
```

**Linux (x64):**
```bash
curl -L -o try.tar.gz $(curl -s https://api.github.com/repos/amulcse/try/releases/latest | grep "browser_download_url.*linux_amd64.tar.gz" | cut -d '"' -f 4)
tar -xzf try.tar.gz && chmod +x try && sudo mv try /usr/local/bin/
```

**Linux (arm64):**
```bash
curl -L -o try.tar.gz $(curl -s https://api.github.com/repos/amulcse/try/releases/latest | grep "browser_download_url.*linux_arm64.tar.gz" | cut -d '"' -f 4)
tar -xzf try.tar.gz && chmod +x try && sudo mv try /usr/local/bin/
```

**Windows (PowerShell):**
```powershell
$url = (Invoke-RestMethod https://api.github.com/repos/amulcse/try/releases/latest).assets | Where-Object { $_.name -like "*windows_amd64.zip" } | Select-Object -ExpandProperty browser_download_url
Invoke-WebRequest -Uri $url -OutFile "try.zip"
Expand-Archive -Path "try.zip" -DestinationPath "." -Force
Move-Item -Path "try.exe" -Destination "$env:USERPROFILE\bin\try.exe" -Force
```

### Go Install

If you have Go installed:
```bash
go install github.com/amulcse/try@latest
```

---

## Shell Setup (Required)

After installing the binary, add this to your shell config:

**Bash / Zsh** (`~/.bashrc` or `~/.zshrc`):
```bash
eval "$(try init)"
```

**Fish** (`~/.config/fish/config.fish`):
```fish
eval (try init | string collect)
```

Then restart your shell or run `source ~/.zshrc`.

---

## What It Does

![Demo](https://raw.githubusercontent.com/tobi/try/main/docs/try-fuzzy-search-demo.gif)

Instantly navigate through all your experiment directories with:

- **Fuzzy search** that just works
- **Smart sorting** - recently used stuff bubbles to the top
- **Auto-dating** - creates directories like `2025-01-21-redis-experiment`
- **Zero config** - just one binary, no dependencies

---

## Usage

```bash
try                        # Browse all experiments
try redis                  # Jump to redis experiment or create new
try new api                # Create "2025-01-21-api"
try .                      # Create dated worktree for current repo
try clone https://...      # Clone repo into dated directory
try https://github.com/... # Shorthand for clone
try delete                 # Delete a directory
try rename                 # Rename a directory
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
| `Ctrl-D` | Delete directory |
| `Ctrl-R` | Rename directory |
| `ESC` | Cancel |

---

## Configuration

Set `TRY_PATH` to change where experiments are stored:

```bash
export TRY_PATH=~/code/sketches
```

Default: `~/src/tries`

---

## Why a Go Port?

- **Single binary** - No Ruby runtime required
- **Cross-platform** - Pre-built binaries for macOS, Linux, Windows
- **Same features** - 100% compatible with the original
- **Fast** - Native compiled performance

---

## Features

### 🎯 Smart Fuzzy Search
- `rds` matches `redis-server`
- `connpool` matches `connection-pool`
- Recent stuff scores higher

### ⏰ Time-Aware
- Shows how long ago you touched each project
- Recently accessed directories float to the top

### 🎨 Beautiful TUI
- Clean, minimal interface
- Highlights matches as you type
- Respects `NO_COLOR` environment variable

### 📁 Organized Chaos
- Everything in one place
- Auto-prefixes with dates
- Git worktree support

---

## The Philosophy

> Your brain doesn't work in neat folders. You have ideas, you try things, you context-switch like a caffeinated squirrel. This tool embraces that.

Every experiment gets a home. Every home is instantly findable.

---

## Credits

**All credit goes to [Tobi Lütke](https://github.com/tobi) and the original [try](https://github.com/tobi/try) project.**

This is a Go port providing:
- Single binary distribution
- No Ruby dependency
- Cross-platform binaries

Check out the original: **[github.com/tobi/try](https://github.com/tobi/try)**

## License

MIT License - Same as the [original project](https://github.com/tobi/try/blob/main/LICENSE).

---

*Your experiments deserve a home.* 🏠
