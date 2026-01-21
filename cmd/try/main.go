// Package main is the entry point for the try CLI
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

	"github.com/amulcse/try/internal/config"
	"github.com/amulcse/try/internal/tui"
)

func main() {
	args := append([]string{}, os.Args[1:]...)

	// Color flags (processed early)
	var disableColors bool
	args, disabled := removeFlag(args, "--no-colors")
	if disabled {
		disableColors = true
	}
	args, disabled = removeFlag(args, "--no-expand-tokens")
	if disabled {
		disableColors = true
	}
	if os.Getenv("NO_COLOR") != "" {
		disableColors = true
	}
	if disableColors {
		tui.DisableColors()
	}

	if containsFlag(args, "--help", "-h") {
		config.PrintHelp(config.DefaultTriesPath())
		os.Exit(0)
	}

	if containsFlag(args, "--version", "-v") {
		if config.BuildTime != "" {
			fmt.Printf("try %s (built %s)\n", config.Version, config.BuildTime)
		} else {
			fmt.Printf("try %s\n", config.Version)
		}
		os.Exit(0)
	}

	triesPath := extractOptionWithValue(&args, "--path")
	if triesPath == "" {
		triesPath = config.DefaultTriesPath()
	} else {
		triesPath = config.ExpandPath(triesPath)
	}

	andType := extractOptionWithValue(&args, "--and-type")
	var andExit bool
	args, andExit = removeFlag(args, "--and-exit")
	andKeysRaw := extractOptionWithValue(&args, "--and-keys")
	andConfirm := extractOptionWithValue(&args, "--and-confirm")
	andKeys := parseTestKeys(andKeysRaw)

	var command string
	if len(args) > 0 {
		command = args[0]
		args = args[1:]
	}

	switch command {
	case "":
		config.PrintHelp(triesPath)
		os.Exit(2)
	case "clone":
		cmds := cmdClone(args, triesPath)
		emitScript(cmds)
		os.Exit(0)
	case "init":
		cmdInit(args, triesPath)
		os.Exit(0)
	case "exec":
		sub := ""
		if len(args) > 0 {
			sub = args[0]
		}
		switch sub {
		case "clone":
			if len(args) > 0 {
				args = args[1:]
			}
			cmds := cmdClone(args, triesPath)
			emitScript(cmds)
			os.Exit(0)
		case "worktree":
			if len(args) > 0 {
				args = args[1:]
			}
			repo := ""
			if len(args) > 0 {
				repo = args[0]
				args = args[1:]
			}
			repoDir := repoDirFromArg(repo)
			fullPath := worktreePath(triesPath, repoDir, strings.Join(args, " "))
			cmds := scriptWorktree(fullPath, repoDir, true)
			emitScript(cmds)
			os.Exit(0)
		case "cd":
			if len(args) > 0 {
				args = args[1:]
			}
			cmds := cmdCd(args, triesPath, andType, andExit, andKeys, andConfirm)
			if cmds == nil {
				fmt.Println("Cancelled.")
				os.Exit(1)
			}
			emitScript(cmds)
			os.Exit(0)
		default:
			cmds := cmdCd(args, triesPath, andType, andExit, andKeys, andConfirm)
			if cmds == nil {
				fmt.Println("Cancelled.")
				os.Exit(1)
			}
			emitScript(cmds)
			os.Exit(0)
		}
	case "worktree":
		repo := ""
		if len(args) > 0 {
			repo = args[0]
			args = args[1:]
		}
		repoDir := repoDirFromArg(repo)
		fullPath := worktreePath(triesPath, repoDir, strings.Join(args, " "))
		cmds := scriptWorktree(fullPath, repoDir, true)
		emitScript(cmds)
		os.Exit(0)
	default:
		args = append([]string{command}, args...)
		cmds := cmdCd(args, triesPath, andType, andExit, andKeys, andConfirm)
		if cmds == nil {
			fmt.Println("Cancelled.")
			os.Exit(1)
		}
		emitScript(cmds)
		os.Exit(0)
	}
}

func cmdClone(args []string, triesPath string) []string {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: git URI required for clone command")
		fmt.Fprintln(os.Stderr, "Usage: try clone <git-uri> [name]")
		os.Exit(1)
	}
	gitURI := args[0]
	customName := ""
	if len(args) > 1 {
		customName = args[1]
	}
	name, err := generateCloneDirectoryName(gitURI, customName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Unable to parse git URI: %s\n", gitURI)
		os.Exit(1)
	}
	fullPath := filepath.Join(triesPath, name)
	return scriptClone(fullPath, gitURI)
}

func cmdInit(args []string, triesPath string) {
	scriptPath, err := os.Executable()
	if err != nil {
		scriptPath = os.Args[0]
	}
	scriptPath = config.ExpandPath(scriptPath)

	if len(args) > 0 && strings.HasPrefix(args[0], "/") {
		triesPath = config.ExpandPath(args[0])
		args = args[1:]
	}

	pathArg := ""
	if triesPath != "" {
		pathArg = fmt.Sprintf(" --path %s", q(triesPath))
	}

	bashOrZsh := fmt.Sprintf(`try() {
  local out
  out=$(%s exec%s "$@" 2>/dev/tty)
  if [ $? -eq 0 ]; then
    eval "$out"
  else
    echo "$out"
  fi
}
`, q(scriptPath), pathArg)

	fish := fmt.Sprintf(`function try
  set -l out (%s exec%s $argv 2>/dev/tty | string collect)
  if test $status -eq 0
    eval $out
  else
    echo $out
  end
end
`, q(scriptPath), pathArg)

	if fishShell() {
		fmt.Print(fish)
	} else {
		fmt.Print(bashOrZsh)
	}
	os.Exit(0)
}

func cmdCd(args []string, triesPath, andType string, andExit bool, andKeys []string, andConfirm string) []string {
	if len(args) > 0 && args[0] == "clone" {
		return cmdClone(args[1:], triesPath)
	}

	if len(args) > 0 && strings.HasPrefix(args[0], ".") {
		pathArg := args[0]
		args = args[1:]
		custom := strings.Join(args, " ")
		repoDir := config.ExpandPath(pathArg)
		if pathArg == "." && strings.TrimSpace(custom) == "" {
			fmt.Fprintln(os.Stderr, "Error: 'try .' requires a name argument")
			fmt.Fprintln(os.Stderr, "Usage: try . <name>")
			os.Exit(1)
		}
		base := ""
		if strings.TrimSpace(custom) != "" {
			base = strings.ReplaceAll(custom, " ", "-")
		} else {
			base = filepath.Base(repoDir)
		}
		datePrefix := time.Now().Format("2006-01-02")
		base = resolveUniqueNameWithVersioning(triesPath, datePrefix, base)
		fullPath := filepath.Join(triesPath, fmt.Sprintf("%s-%s", datePrefix, base))
		if _, err := os.Stat(filepath.Join(repoDir, ".git")); err == nil {
			return scriptWorktree(fullPath, repoDir, false)
		}
		return scriptMkdirCd(fullPath)
	}

	searchTerm := strings.Join(args, " ")
	fields := strings.Fields(searchTerm)
	if len(fields) > 0 && isGitURI(fields[0]) {
		gitURI := fields[0]
		custom := ""
		if len(fields) > 1 {
			custom = strings.Join(fields[1:], " ")
		}
		name, err := generateCloneDirectoryName(gitURI, custom)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Unable to parse git URI: %s\n", gitURI)
			os.Exit(1)
		}
		fullPath := filepath.Join(triesPath, name)
		return scriptClone(fullPath, gitURI)
	}

	selector := tui.NewSelector(searchTerm, triesPath, andType, andExit, andKeys, andConfirm)
	result := selector.Run()
	if result == nil {
		return nil
	}

	switch result.Type {
	case "delete":
		return scriptDelete(result.Paths, result.BasePath)
	case "mkdir":
		return scriptMkdirCd(result.Path)
	case "rename":
		return scriptRename(result.BasePath, result.OldName, result.NewName)
	default:
		return scriptCd(result.Path)
	}
}

func repoDirFromArg(repo string) string {
	if repo == "" || repo == "dir" {
		cwd, err := os.Getwd()
		if err != nil {
			return "."
		}
		return cwd
	}
	return config.ExpandPath(repo)
}

func worktreePath(triesPath, repoDir, customName string) string {
	base := ""
	if strings.TrimSpace(customName) != "" {
		base = strings.ReplaceAll(customName, " ", "-")
	} else {
		if real, err := filepath.EvalSymlinks(repoDir); err == nil {
			base = filepath.Base(real)
		} else {
			base = filepath.Base(repoDir)
		}
	}
	datePrefix := time.Now().Format("2006-01-02")
	base = resolveUniqueNameWithVersioning(triesPath, datePrefix, base)
	return filepath.Join(triesPath, fmt.Sprintf("%s-%s", datePrefix, base))
}

func scriptCd(path string) []string {
	return []string{
		"clear",
		fmt.Sprintf("touch %s", q(path)),
		fmt.Sprintf("cd %s", q(path)),
	}
}

func scriptMkdirCd(path string) []string {
	cmds := []string{fmt.Sprintf("mkdir -p %s", q(path))}
	cmds = append(cmds, scriptCd(path)...)
	return cmds
}

func scriptClone(path, uri string) []string {
	cmds := []string{
		fmt.Sprintf("mkdir -p %s", q(path)),
		fmt.Sprintf("echo %s", q(fmt.Sprintf("Using git clone to create this trial from %s.", uri))),
		fmt.Sprintf("git clone %s %s", q(uri), q(path)),
	}
	cmds = append(cmds, scriptCd(path)...)
	return cmds
}

func scriptWorktree(path, repo string, explicit bool) []string {
	src := repo
	if repo == "" || !explicit {
		cwd, err := os.Getwd()
		if err == nil {
			src = cwd
		}
	}

	var worktreeCmd string
	if repo != "" && explicit {
		r := q(repo)
		worktreeCmd = fmt.Sprintf("/usr/bin/env sh -c 'if git -C %s rev-parse --is-inside-work-tree >/dev/null 2>&1; then repo=$(git -C %s rev-parse --show-toplevel); git -C \"$repo\" worktree add --detach %s >/dev/null 2>&1 || true; fi; exit 0'", r, r, q(path))
	} else {
		worktreeCmd = fmt.Sprintf("/usr/bin/env sh -c 'if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then repo=$(git rev-parse --show-toplevel); git -C \"$repo\" worktree add --detach %s >/dev/null 2>&1 || true; fi; exit 0'", q(path))
	}

	cmds := []string{
		fmt.Sprintf("mkdir -p %s", q(path)),
		fmt.Sprintf("echo %s", q(fmt.Sprintf("Using git worktree to create this trial from %s.", src))),
		worktreeCmd,
	}
	cmds = append(cmds, scriptCd(path)...)
	return cmds
}

func scriptDelete(paths []tui.DeletePath, basePath string) []string {
	cmds := []string{fmt.Sprintf("cd %s", q(basePath))}
	for _, item := range paths {
		cmds = append(cmds, fmt.Sprintf("test -d %s && rm -rf %s", q(item.Basename), q(item.Basename)))
	}
	cwd, _ := os.Getwd()
	cmds = append(cmds, fmt.Sprintf("( cd %s 2>/dev/null || cd \"$HOME\" )", q(cwd)))
	return cmds
}

func scriptRename(basePath, oldName, newName string) []string {
	newPath := filepath.Join(basePath, newName)
	return []string{
		fmt.Sprintf("cd %s", q(basePath)),
		fmt.Sprintf("mv %s %s", q(oldName), q(newName)),
		fmt.Sprintf("echo %s", q(newPath)),
		fmt.Sprintf("cd %s", q(newPath)),
	}
}

func parseTestKeys(spec string) []string {
	if spec == "" {
		return nil
	}

	useTokenMode := strings.Contains(spec, ",") || regexpMustMatch(spec, `^[A-Z\-]+$`)
	if useTokenMode {
		tokens := splitTokens(spec)
		keys := []string{}
		for _, tok := range tokens {
			up := strings.ToUpper(tok)
			switch up {
			case "UP":
				keys = append(keys, "\x1b[A")
			case "DOWN":
				keys = append(keys, "\x1b[B")
			case "LEFT":
				keys = append(keys, "\x1b[D")
			case "RIGHT":
				keys = append(keys, "\x1b[C")
			case "ENTER", "RETURN":
				keys = append(keys, "\r")
			case "ESC", "ESCAPE":
				keys = append(keys, "\x1b")
			case "BACKSPACE", "BS":
				keys = append(keys, "\x7f")
			case "CTRL-A", "CTRLA":
				keys = append(keys, "\x01")
			case "CTRL-B", "CTRLB":
				keys = append(keys, "\x02")
			case "CTRL-D", "CTRLD":
				keys = append(keys, "\x04")
			case "CTRL-E", "CTRLE":
				keys = append(keys, "\x05")
			case "CTRL-F", "CTRLF":
				keys = append(keys, "\x06")
			case "CTRL-H", "CTRLH":
				keys = append(keys, "\x08")
			case "CTRL-K", "CTRLK":
				keys = append(keys, "\x0b")
			case "CTRL-N", "CTRLN":
				keys = append(keys, "\x0e")
			case "CTRL-P", "CTRLP":
				keys = append(keys, "\x10")
			case "CTRL-R", "CTRLR":
				keys = append(keys, "\x12")
			case "CTRL-T", "CTRLT":
				keys = append(keys, "\x14")
			case "CTRL-W", "CTRLW":
				keys = append(keys, "\x17")
			default:
				if strings.HasPrefix(up, "TYPE=") {
					for _, ch := range up[5:] {
						keys = append(keys, string(ch))
					}
				} else if len(tok) == 1 {
					keys = append(keys, tok)
				}
			}
		}
		return keys
	}

	keys := []string{}
	i := 0
	for i < len(spec) {
		if spec[i] == 0x1b && i+2 < len(spec) && spec[i+1] == '[' {
			keys = append(keys, spec[i:i+3])
			i += 3
		} else {
			keys = append(keys, spec[i:i+1])
			i++
		}
	}
	return keys
}

func splitTokens(spec string) []string {
	parts := strings.Split(spec, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func regexpMustMatch(text, pattern string) bool {
	matched, _ := regexp.MatchString(pattern, text)
	return matched
}

// Utility functions

func q(str string) string {
	return "'" + strings.ReplaceAll(str, "'", "'\"'\"'") + "'"
}

func emitScript(cmds []string) {
	fmt.Println(config.ScriptWarning)
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
