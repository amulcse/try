package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Version info - set via ldflags during build
var (
	Version   = "1.7.1"
	BuildTime = ""
)

// For backwards compatibility
var version = Version

const scriptWarning = "# if you can read this, you didn't launch try from an alias. run try --help."

func expandPath(path string) string {
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

func q(str string) string {
	return "'" + strings.ReplaceAll(str, "'", "'\"'\"'") + "'"
}

func emitScript(cmds []string) {
	fmt.Println(scriptWarning)
	for i, cmd := range cmds {
		if i == 0 {
			fmt.Print(cmd)
		} else {
			fmt.Print("  " + cmd)
		}
		if i < len(cmds)-1 {
			fmt.Println(" && \\")
		} else {
			fmt.Println()
		}
	}
}

func printGlobalHelp(currentPath string) {
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
`, version, currentPath)
	fmt.Print(text)
}

func defaultTriesPath() string {
	if env := os.Getenv("TRY_PATH"); env != "" {
		return expandPath(env)
	}
	return expandPath(filepath.Join("~", "src", "tries"))
}

func removeFlag(args []string, flag string) ([]string, bool) {
	removed := false
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == flag {
			removed = true
			continue
		}
		filtered = append(filtered, arg)
	}
	return filtered, removed
}

func containsFlag(args []string, flags ...string) bool {
	for _, arg := range args {
		for _, flag := range flags {
			if arg == flag {
				return true
			}
		}
	}
	return false
}

func extractOptionWithValue(args *[]string, optName string) string {
	idx := -1
	for i := len(*args) - 1; i >= 0; i-- {
		arg := (*args)[i]
		if arg == optName || strings.HasPrefix(arg, optName+"=") {
			idx = i
			break
		}
	}
	if idx == -1 {
		return ""
	}
	arg := (*args)[idx]
	*args = append((*args)[:idx], (*args)[idx+1:]...)
	if strings.Contains(arg, "=") {
		parts := strings.SplitN(arg, "=", 2)
		return parts[1]
	}
	if idx < len(*args) {
		val := (*args)[idx]
		*args = append((*args)[:idx], (*args)[idx+1:]...)
		return val
	}
	return ""
}

func parseGitURI(uri string) (user string, repo string, host string, ok bool) {
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return "", "", "", false
	}
	uri = strings.TrimSuffix(uri, ".git")

	if m := regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/]+)`).FindStringSubmatch(uri); m != nil {
		return m[1], m[2], "github.com", true
	}
	if m := regexp.MustCompile(`^git@github\.com:([^/]+)/([^/]+)`).FindStringSubmatch(uri); m != nil {
		return m[1], m[2], "github.com", true
	}
	if m := regexp.MustCompile(`^https?://([^/]+)/([^/]+)/([^/]+)`).FindStringSubmatch(uri); m != nil {
		return m[2], m[3], m[1], true
	}
	if m := regexp.MustCompile(`^git@([^:]+):([^/]+)/([^/]+)`).FindStringSubmatch(uri); m != nil {
		return m[2], m[3], m[1], true
	}
	return "", "", "", false
}

func generateCloneDirectoryName(gitURI, customName string) (string, error) {
	if strings.TrimSpace(customName) != "" {
		return customName, nil
	}
	user, repo, _, ok := parseGitURI(gitURI)
	if !ok {
		return "", errors.New("unable to parse git URI")
	}
	datePrefix := time.Now().Format("2006-01-02")
	return fmt.Sprintf("%s-%s-%s", datePrefix, user, repo), nil
}

func isGitURI(arg string) bool {
	if arg == "" {
		return false
	}
	return strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") ||
		strings.HasPrefix(arg, "git@") || strings.Contains(arg, "github.com") ||
		strings.Contains(arg, "gitlab.com") || strings.HasSuffix(arg, ".git")
}

func uniqueDirName(triesPath, dirName string) string {
	candidate := dirName
	counter := 2
	for {
		if _, err := os.Stat(filepath.Join(triesPath, candidate)); os.IsNotExist(err) {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", dirName, counter)
		counter++
	}
}

func resolveUniqueNameWithVersioning(triesPath, datePrefix, base string) string {
	initial := fmt.Sprintf("%s-%s", datePrefix, base)
	if _, err := os.Stat(filepath.Join(triesPath, initial)); os.IsNotExist(err) {
		return base
	}

	re := regexp.MustCompile(`^(.*?)(\d+)$`)
	if m := re.FindStringSubmatch(base); m != nil {
		stem := m[1]
		numStr := m[2]
		var num int
		_, _ = fmt.Sscanf(numStr, "%d", &num)
		candidateNum := num + 1
		for {
			candidateBase := fmt.Sprintf("%s%d", stem, candidateNum)
			candidateFull := filepath.Join(triesPath, fmt.Sprintf("%s-%s", datePrefix, candidateBase))
			if _, err := os.Stat(candidateFull); os.IsNotExist(err) {
				return candidateBase
			}
			candidateNum++
		}
	}

	unique := uniqueDirName(triesPath, fmt.Sprintf("%s-%s", datePrefix, base))
	unique = strings.TrimPrefix(unique, datePrefix+"-")
	return unique
}

func fishShell() bool {
	shell := os.Getenv("SHELL")
	if shell == "" {
		out, err := exec.Command("ps", "c", "-p", fmt.Sprintf("%d", os.Getppid()), "-o", "ucomm=").Output()
		if err == nil {
			shell = strings.TrimSpace(string(out))
		}
	}
	return strings.Contains(shell, "fish")
}

func formatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return "?"
	}

	now := time.Now()
	seconds := now.Sub(t).Seconds()
	minutes := seconds / 60
	hours := minutes / 60
	days := hours / 24

	switch {
	case seconds < 60:
		return "just now"
	case minutes < 60:
		return fmt.Sprintf("%dm ago", int(minutes))
	case hours < 24:
		return fmt.Sprintf("%dh ago", int(hours))
	case days < 7:
		return fmt.Sprintf("%dd ago", int(days))
	default:
		return fmt.Sprintf("%dw ago", int(days/7))
	}
}

