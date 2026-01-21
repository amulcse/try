package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
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
		DisableColors()
	}

	if containsFlag(args, "--help", "-h") {
		printGlobalHelp(defaultTriesPath())
		os.Exit(0)
	}

	if containsFlag(args, "--version", "-v") {
		if BuildTime != "" {
			fmt.Printf("try %s (built %s)\n", Version, BuildTime)
		} else {
			fmt.Printf("try %s\n", Version)
		}
		os.Exit(0)
	}

	triesPath := extractOptionWithValue(&args, "--path")
	if triesPath == "" {
		triesPath = defaultTriesPath()
	} else {
		triesPath = expandPath(triesPath)
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
		printGlobalHelp(triesPath)
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
	scriptPath = expandPath(scriptPath)

	if len(args) > 0 && strings.HasPrefix(args[0], "/") {
		triesPath = expandPath(args[0])
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
		repoDir := expandPath(pathArg)
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

	selector := NewTrySelector(searchTerm, triesPath, andType, andExit, andKeys, andConfirm)
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
	return expandPath(repo)
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

func scriptDelete(paths []DeletePath, basePath string) []string {
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
