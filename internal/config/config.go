// Package config provides configuration and version information
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Version info - set via ldflags during build
var (
	Version   = "1.7.5"
	BuildTime = ""
)

const ScriptWarning = "# if you can read this, you didn't launch try from an alias. run try --help."

// ExpandPath expands ~ and returns absolute path
func ExpandPath(path string) string {
	if path == "" {
		return path
	}
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				path = home
			} else if strings.HasPrefix(path, "~/") {
				path = filepath.Join(home, path[2:])
			}
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

// DefaultTriesPath returns the default path for tries
func DefaultTriesPath() string {
	if env := os.Getenv("TRY_PATH"); env != "" {
		return ExpandPath(env)
	}
	return ExpandPath(filepath.Join("~", "src", "tries"))
}

// PrintHelp prints the help text
func PrintHelp(currentPath string) {
	text := fmt.Sprintf(`try v%[1]s - ephemeral workspace manager

To use try, add to your shell config:

  # bash/zsh (~/.bashrc or ~/.zshrc)
  eval "$(try init ~/src/tries)"

  # fish (~/.config/fish/config.fish)
  eval (try init ~/src/tries | string collect)

Usage:
  try [query]           Interactive directory selector
  try clone <url>       Clone repo into dated directory
  try worktree <name>   Create worktree from current git repo
  try --help            Show this help

Commands:
  init [path]           Output shell function definition
  clone <url> [name]    Clone git repo into date-prefixed directory
  worktree <name>       Create worktree in dated directory

Examples:
  try                   Open interactive selector
  try project           Selector with initial filter
  try clone https://github.com/user/repo
  try worktree feature-branch

Manual mode (without alias):
  try exec [query]      Output shell script to eval

Defaults:
  Default path: ~/src/tries
  Current: %[2]s
`, Version, currentPath)
	fmt.Print(text)
}
