package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/term"
)

// ANSI escape sequences
const (
	ansiClearEOL      = "\x1b[K"
	ansiClearScreen   = "\x1b[2J"
	ansiHome          = "\x1b[H"
	ansiHide          = "\x1b[?25l"
	ansiShow          = "\x1b[?25h"
	ansiCursorBlink   = "\x1b[1 q"
	ansiCursorDefault = "\x1b[0 q"
	ansiAltScreenOn   = "\x1b[?1049h"
	ansiAltScreenOff  = "\x1b[?1049l"
	ansiReset         = "\x1b[0m"
	ansiResetFG       = "\x1b[39m"
	ansiResetBG       = "\x1b[49m"
	ansiBold          = "\x1b[1m"
	ansiDim           = "\x1b[2m"
	ansiReverse       = "\x1b[7m"
	ansiReverseOff    = "\x1b[27m"

	// Colors
	colorMuted      = "\x1b[38;5;245m"
	colorHighlight  = "\x1b[1m\x1b[33m"       // Bold + yellow (separate sequences)
	colorAccent     = "\x1b[1m\x1b[38;5;214m" // Bold + 256-color orange (separate)
	colorSelectedBG = "\x1b[48;5;238m"
	colorDangerBG   = "\x1b[48;5;52m"
)

var colorsEnabled = true

func DisableColors() {
	colorsEnabled = false
}

func EnableColors() {
	colorsEnabled = true
}

func ansiWrap(text, prefix, suffix string) string {
	if text == "" || !colorsEnabled {
		return text
	}
	return prefix + text + suffix
}

func dim(text string) string {
	return ansiWrap(text, colorMuted, ansiResetFG)
}

func bold(text string) string {
	return ansiWrap(text, ansiBold, ansiReset)
}

func highlight(text string) string {
	return ansiWrap(text, colorHighlight, ansiResetFG+"\x1b[22m")
}

func accent(text string) string {
	return ansiWrap(text, colorAccent, ansiResetFG+"\x1b[22m")
}

// TryItem represents a directory entry
type TryItem struct {
	Text      string
	Basename  string
	Path      string
	IsNew     bool
	Ctime     time.Time
	Mtime     time.Time
	BaseScore float64
}

// TryEntry wraps a TryItem with match data
type TryEntry struct {
	Item               TryItem
	Score              float64
	HighlightPositions []int
}

// SelectionResult is the result of the TUI selection
type SelectionResult struct {
	Type     string       // "cd", "mkdir", "delete", "rename"
	Path     string       // for cd/mkdir
	Paths    []DeletePath // for delete
	BasePath string       // for delete/rename
	OldName  string       // for rename
	NewName  string       // for rename
}

// DeletePath represents a path marked for deletion
type DeletePath struct {
	Path     string
	Basename string
}

// TrySelector is the interactive TUI selector
type TrySelector struct {
	searchTerm      string
	cursorPos       int
	inputCursorPos  int
	scrollOffset    int
	inputBuffer     string
	selected        *SelectionResult
	allTries        []TryItem
	basePath        string
	deleteStatus    string
	deleteMode      bool
	markedForDelete []string
	testRenderOnce  bool
	testNoCls       bool
	testKeys        []string
	testHadKeys     bool
	testConfirm     string
	needsRedraw     bool
	fuzzy           *Fuzzy
	io              *os.File
	oldState        *term.State
	width           int
	height          int
}

// NewTrySelector creates a new TrySelector
func NewTrySelector(searchTerm, basePath, andType string, andExit bool, andKeys []string, andConfirm string) *TrySelector {
	initialInput := searchTerm
	if andType != "" {
		initialInput = andType
	}
	initialInput = strings.ReplaceAll(initialInput, " ", "-")

	s := &TrySelector{
		searchTerm:      strings.ReplaceAll(searchTerm, " ", "-"),
		inputBuffer:     initialInput,
		inputCursorPos:  len(initialInput),
		basePath:        basePath,
		markedForDelete: []string{},
		testRenderOnce:  andExit,
		testNoCls:       andExit || (andKeys != nil && len(andKeys) > 0),
		testKeys:        andKeys,
		testHadKeys:     andKeys != nil && len(andKeys) > 0,
		testConfirm:     andConfirm,
		io:              os.Stderr,
		width:           80,
		height:          24,
	}

	// Ensure base path exists
	if err := os.MkdirAll(basePath, 0755); err == nil {
		// path created or exists
	}

	return s
}

// Run starts the TUI and returns the selection result
func (s *TrySelector) Run() *SelectionResult {
	s.setupTerminal()
	defer s.restoreTerminal()

	// Test mode: render once and exit
	if s.testRenderOnce && (s.testKeys == nil || len(s.testKeys) == 0) {
		tries := s.getTries()
		s.render(tries)
		return nil
	}

	// Check for TTY
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stderr.Fd())) {
		if s.testKeys == nil || len(s.testKeys) == 0 {
			fmt.Fprintln(os.Stderr, "Error: try requires an interactive terminal")
			return nil
		}
	}

	// Set raw mode for interactive use
	if term.IsTerminal(int(os.Stdin.Fd())) {
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err == nil {
			s.oldState = oldState
		}
	}

	s.mainLoop()
	return s.selected
}

func (s *TrySelector) setupTerminal() {
	s.refreshSize()

	if !s.testNoCls {
		fmt.Fprint(s.io, ansiAltScreenOn)
		fmt.Fprint(s.io, ansiClearScreen)
		fmt.Fprint(s.io, ansiHome)
		fmt.Fprint(s.io, ansiCursorBlink)
		s.io.Sync() // Flush to ensure alternate screen is active
	}

	// Handle window resize (Unix only, no-op on Windows)
	setupResizeHandler(func() {
		s.needsRedraw = true
	})
}

func (s *TrySelector) restoreTerminal() {
	if s.oldState != nil {
		term.Restore(int(os.Stdin.Fd()), s.oldState)
	}

	if !s.testNoCls {
		fmt.Fprint(s.io, ansiReset)
		fmt.Fprint(s.io, ansiCursorDefault)
		fmt.Fprint(s.io, ansiAltScreenOff)
		s.io.Sync() // Flush to ensure screen is restored before any output
	}
}

func (s *TrySelector) refreshSize() {
	// Check environment overrides first
	if w := os.Getenv("TRY_WIDTH"); w != "" {
		fmt.Sscanf(w, "%d", &s.width)
	}
	if h := os.Getenv("TRY_HEIGHT"); h != "" {
		fmt.Sscanf(h, "%d", &s.height)
	}

	// Try to get actual terminal size
	if s.width == 0 || s.height == 0 || os.Getenv("TRY_WIDTH") == "" {
		if w, h, err := term.GetSize(int(os.Stderr.Fd())); err == nil {
			if os.Getenv("TRY_WIDTH") == "" {
				s.width = w
			}
			if os.Getenv("TRY_HEIGHT") == "" {
				s.height = h
			}
		}
	}

	// Fallback defaults
	if s.width <= 0 {
		s.width = 80
	}
	if s.height <= 0 {
		s.height = 24
	}
}

func (s *TrySelector) loadAllTries() {
	if s.allTries != nil {
		return
	}

	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		s.allTries = []TryItem{}
		return
	}

	now := time.Now()
	s.allTries = make([]TryItem, 0, len(entries))

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if !entry.IsDir() {
			continue
		}

		path := filepath.Join(s.basePath, name)
		info, err := entry.Info()
		if err != nil {
			continue
		}

		mtime := info.ModTime()
		hoursSinceAccess := now.Sub(mtime).Hours()
		baseScore := 3.0 / math.Sqrt(hoursSinceAccess+1)

		// Date prefix bonus
		if matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}-`, name); matched {
			baseScore += 2.0
		}

		s.allTries = append(s.allTries, TryItem{
			Text:      name,
			Basename:  name,
			Path:      path,
			IsNew:     false,
			Mtime:     mtime,
			BaseScore: baseScore,
		})
	}
}

func (s *TrySelector) getTries() []TryEntry {
	s.loadAllTries()
	if s.fuzzy == nil {
		s.fuzzy = NewFuzzy(s.allTries)
	}

	matches := s.fuzzy.Match(s.inputBuffer)
	results := make([]TryEntry, 0, len(matches))
	for _, m := range matches {
		results = append(results, TryEntry{
			Item:               m.Entry,
			Score:              m.Score,
			HighlightPositions: m.Positions,
		})
	}
	return results
}

func (s *TrySelector) mainLoop() {
	for {
		tries := s.getTries()
		showCreateNew := s.inputBuffer != ""
		totalItems := len(tries)
		if showCreateNew {
			totalItems++
		}

		// Ensure cursor within bounds
		if s.cursorPos < 0 {
			s.cursorPos = 0
		}
		maxPos := totalItems - 1
		if maxPos < 0 {
			maxPos = 0
		}
		if s.cursorPos > maxPos {
			s.cursorPos = maxPos
		}

		s.render(tries)

		key := s.readKey()
		if key == "" {
			continue // resize or timeout
		}

		switch key {
		case "\r", "\n": // Enter
			if s.deleteMode && len(s.markedForDelete) > 0 {
				s.confirmBatchDelete(tries)
				if s.selected != nil {
					return
				}
			} else if s.cursorPos < len(tries) {
				s.handleSelection(tries[s.cursorPos])
				if s.selected != nil {
					return
				}
			} else if showCreateNew {
				s.handleCreateNew()
				if s.selected != nil {
					return
				}
			}

		case "\x1b[A", "\x10": // Up arrow or Ctrl-P
			if s.cursorPos > 0 {
				s.cursorPos--
			}

		case "\x1b[B", "\x0e": // Down arrow or Ctrl-N
			if s.cursorPos < totalItems-1 {
				s.cursorPos++
			}

		case "\x1b[C", "\x1b[D": // Left/Right arrows - ignore

		case "\x7f", "\x08": // Backspace (DEL) or Ctrl-H
			if s.inputCursorPos > 0 {
				s.inputBuffer = s.inputBuffer[:s.inputCursorPos-1] + s.inputBuffer[s.inputCursorPos:]
				s.inputCursorPos--
			}
			s.cursorPos = 0

		case "\x01": // Ctrl-A
			s.inputCursorPos = 0

		case "\x05": // Ctrl-E
			s.inputCursorPos = len(s.inputBuffer)

		case "\x02": // Ctrl-B
			if s.inputCursorPos > 0 {
				s.inputCursorPos--
			}

		case "\x06": // Ctrl-F
			if s.inputCursorPos < len(s.inputBuffer) {
				s.inputCursorPos++
			}

		case "\x0b": // Ctrl-K
			s.inputBuffer = s.inputBuffer[:s.inputCursorPos]

		case "\x17": // Ctrl-W
			if s.inputCursorPos > 0 {
				pos := s.inputCursorPos - 1
				// Skip non-alphanumeric
				for pos >= 0 && !isWordChar(rune(s.inputBuffer[pos])) {
					pos--
				}
				// Skip alphanumeric
				for pos >= 0 && isWordChar(rune(s.inputBuffer[pos])) {
					pos--
				}
				newPos := pos + 1
				s.inputBuffer = s.inputBuffer[:newPos] + s.inputBuffer[s.inputCursorPos:]
				s.inputCursorPos = newPos
			}

		case "\x04": // Ctrl-D - toggle delete mark
			if s.cursorPos < len(tries) {
				path := tries[s.cursorPos].Item.Path
				idx := indexOf(s.markedForDelete, path)
				if idx >= 0 {
					s.markedForDelete = append(s.markedForDelete[:idx], s.markedForDelete[idx+1:]...)
				} else {
					s.markedForDelete = append(s.markedForDelete, path)
					s.deleteMode = true
				}
				if len(s.markedForDelete) == 0 {
					s.deleteMode = false
				}
			}

		case "\x14": // Ctrl-T - create new
			s.handleCreateNew()
			if s.selected != nil {
				return
			}

		case "\x12": // Ctrl-R - rename
			if s.cursorPos < len(tries) {
				s.runRenameDialog(tries[s.cursorPos])
				if s.selected != nil {
					return
				}
			}

		case "\x03", "\x1b": // Ctrl-C or ESC
			if s.deleteMode {
				s.markedForDelete = nil
				s.deleteMode = false
			} else {
				s.selected = nil
				return
			}

		default:
			// Only accept printable characters
			if len(key) == 1 && isPrintable(key[0]) {
				s.inputBuffer = s.inputBuffer[:s.inputCursorPos] + key + s.inputBuffer[s.inputCursorPos:]
				s.inputCursorPos++
				s.cursorPos = 0
			}
		}
	}
}

func (s *TrySelector) readKey() string {
	if s.testKeys != nil && len(s.testKeys) > 0 {
		key := s.testKeys[0]
		s.testKeys = s.testKeys[1:]
		return key
	}
	// Auto-exit in test mode with no more keys
	if s.testHadKeys && (s.testKeys == nil || len(s.testKeys) == 0) {
		return "\x1b"
	}

	// Set read deadline for resize checking
	buf := make([]byte, 6)
	n, err := os.Stdin.Read(buf)
	if err != nil {
		if s.needsRedraw {
			s.needsRedraw = false
			s.refreshSize()
			return ""
		}
		return ""
	}

	if s.needsRedraw {
		s.needsRedraw = false
		s.refreshSize()
		if !s.testNoCls {
			fmt.Fprint(s.io, ansiClearScreen+ansiHome)
		}
		return ""
	}

	return string(buf[:n])
}

func (s *TrySelector) render(tries []TryEntry) {
	s.refreshSize()
	var out strings.Builder

	out.WriteString(ansiHome)

	// Header
	headerLines := []string{}
	headerLines = append(headerLines, s.renderHeaderLine("🏠", accent(" Try Directory Selection")))
	headerLines = append(headerLines, dim(strings.Repeat("─", s.width-1)))
	headerLines = append(headerLines, s.renderSearchLine())
	headerLines = append(headerLines, dim(strings.Repeat("─", s.width-1)))

	// Footer
	footerLines := []string{}
	footerLines = append(footerLines, dim(strings.Repeat("─", s.width-1)))
	if s.deleteStatus != "" {
		footerLines = append(footerLines, bold(s.deleteStatus))
		s.deleteStatus = ""
	} else if s.deleteMode {
		footerLines = append(footerLines, s.renderDeleteModeFooter())
	} else {
		footerLines = append(footerLines, s.centerText(dim("↑/↓: Navigate  Enter: Select  ^R: Rename  ^D: Delete  Esc: Cancel")))
	}

	// Calculate body space
	maxVisible := s.height - len(headerLines) - len(footerLines)
	if maxVisible < 3 {
		maxVisible = 3
	}

	showCreateNew := s.inputBuffer != ""
	totalItems := len(tries)
	if showCreateNew {
		totalItems++
	}

	// Adjust scroll offset
	if s.cursorPos < s.scrollOffset {
		s.scrollOffset = s.cursorPos
	} else if s.cursorPos >= s.scrollOffset+maxVisible {
		s.scrollOffset = s.cursorPos - maxVisible + 1
	}

	visibleEnd := s.scrollOffset + maxVisible
	if visibleEnd > totalItems {
		visibleEnd = totalItems
	}

	// Write header
	for _, line := range headerLines {
		out.WriteString("\r" + ansiClearEOL)
		out.WriteString(s.truncateLine(line))
		out.WriteString("\n")
	}

	// Write body
	bodyLinesRendered := 0
	for idx := s.scrollOffset; idx < visibleEnd; idx++ {
		// Empty line before "Create new" if there are existing entries
		if idx == len(tries) && len(tries) > 0 && idx >= s.scrollOffset {
			out.WriteString("\r" + ansiClearEOL + "\n")
			bodyLinesRendered++
			if bodyLinesRendered >= maxVisible {
				break
			}
		}

		if idx < len(tries) {
			out.WriteString("\r" + ansiClearEOL)
			out.WriteString(s.renderEntryLine(tries[idx], idx == s.cursorPos))
			out.WriteString(ansiReset + "\n")
		} else {
			out.WriteString("\r" + ansiClearEOL)
			out.WriteString(s.renderCreateLine(idx == s.cursorPos))
			out.WriteString(ansiReset + "\n")
		}
		bodyLinesRendered++
	}

	// Fill remaining body space with blank lines
	for bodyLinesRendered < maxVisible {
		out.WriteString("\r" + ansiClearEOL + "\n")
		bodyLinesRendered++
	}

	// Write footer (last line without newline to avoid scrolling)
	for i, line := range footerLines {
		out.WriteString("\r" + ansiClearEOL)
		out.WriteString(s.truncateLine(line))
		if i < len(footerLines)-1 {
			out.WriteString("\n")
		}
	}

	// Position cursor at search input
	searchLineRow := 3 // Header line 3 (1-indexed)
	searchPrefix := "Search: "
	cursorCol := len(searchPrefix) + s.inputCursorPos + 1
	out.WriteString(fmt.Sprintf("\x1b[%d;%dH", searchLineRow, cursorCol))
	out.WriteString(ansiShow)
	out.WriteString(ansiReset)

	s.io.WriteString(out.String())
}

func (s *TrySelector) renderHeaderLine(emoji, text string) string {
	return emoji + text
}

func (s *TrySelector) renderSearchLine() string {
	prefix := dim("Search: ")
	input := s.renderInput(s.inputBuffer, s.inputCursorPos)
	return prefix + input
}

func (s *TrySelector) renderInput(text string, cursor int) string {
	if text == "" {
		return dim("")
	}

	before := text[:cursor]
	cursorChar := " "
	if cursor < len(text) {
		cursorChar = string(text[cursor])
	}
	after := ""
	if cursor < len(text) {
		after = text[cursor+1:]
	}

	var out strings.Builder
	out.WriteString(before)
	if colorsEnabled {
		out.WriteString(ansiReverse)
	}
	out.WriteString(cursorChar)
	if colorsEnabled {
		out.WriteString(ansiReverseOff)
	}
	out.WriteString(after)
	return out.String()
}

func (s *TrySelector) renderEntryLine(entry TryEntry, isSelected bool) string {
	isMarked := indexOf(s.markedForDelete, entry.Item.Path) >= 0
	var out strings.Builder

	// Background
	if isMarked && colorsEnabled {
		out.WriteString(colorDangerBG)
	} else if isSelected && colorsEnabled {
		out.WriteString(colorSelectedBG)
	}

	// Arrow
	if isSelected {
		out.WriteString(highlight("→ "))
	} else {
		out.WriteString("  ")
	}

	// Emoji
	if isMarked {
		out.WriteString("🗑️ ")
	} else {
		out.WriteString("📁 ")
	}

	// Name with highlights
	plainName, renderedName := s.formattedEntryName(entry)

	// Metadata (right-aligned)
	meta := fmt.Sprintf("%s, %.1f", formatRelativeTime(entry.Item.Mtime), entry.Score)

	// Calculate available width (max content = width - 1 to avoid wrapping)
	maxContent := s.width - 1
	prefixWidth := 5 // "→ " or "  " + emoji + space = ~5 visible chars
	metaWidth := visibleLen(meta)

	// Truncate name only if it exceeds line width (not just to make room for metadata)
	maxNameWidth := maxContent - prefixWidth - 1
	if visibleLen(plainName) > maxNameWidth && maxNameWidth > 2 {
		renderedName = truncateWithAnsi(renderedName, maxNameWidth-1) + "…"
	}

	out.WriteString(renderedName)

	// Calculate positions for right-aligned metadata
	nameWidth := visibleLen(renderedName)
	leftContentWidth := prefixWidth + nameWidth
	rightCol := maxContent - metaWidth

	// Fill gap with spaces to position metadata at right edge
	gap := rightCol - leftContentWidth
	if gap > 0 {
		out.WriteString(strings.Repeat(" ", gap))
	}
	out.WriteString(dim(meta))

	return out.String()
}

func (s *TrySelector) formattedEntryName(entry TryEntry) (plain, rendered string) {
	basename := entry.Item.Basename
	positions := entry.HighlightPositions

	// Check for date prefix
	dateRe := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})-(.+)$`)
	if m := dateRe.FindStringSubmatch(basename); m != nil {
		datePart := m[1]
		namePart := m[2]
		dateLen := len(datePart) + 1 // +1 for hyphen

		var rendered strings.Builder
		rendered.WriteString(dim(datePart))
		// Hyphen highlight check
		if contains(positions, 10) {
			rendered.WriteString(highlight("-"))
		} else {
			rendered.WriteString(dim("-"))
		}
		rendered.WriteString(highlightWithPositions(namePart, positions, dateLen))

		return basename, rendered.String()
	}

	return basename, highlightWithPositions(basename, positions, 0)
}

func highlightWithPositions(text string, positions []int, offset int) string {
	var result strings.Builder
	for i, ch := range text {
		if contains(positions, i+offset) {
			result.WriteString(highlight(string(ch)))
		} else {
			result.WriteString(string(ch))
		}
	}
	return result.String()
}

func (s *TrySelector) renderCreateLine(isSelected bool) string {
	var out strings.Builder

	if isSelected && colorsEnabled {
		out.WriteString(colorSelectedBG)
	}

	if isSelected {
		out.WriteString(highlight("→ "))
	} else {
		out.WriteString("  ")
	}

	datePrefix := time.Now().Format("2006-01-02")
	if s.inputBuffer == "" {
		out.WriteString(fmt.Sprintf("📂 Create new: %s-", datePrefix))
	} else {
		out.WriteString(fmt.Sprintf("📂 Create new: %s-%s", datePrefix, s.inputBuffer))
	}

	return out.String()
}

func (s *TrySelector) renderDeleteModeFooter() string {
	var out strings.Builder
	if colorsEnabled {
		out.WriteString(colorDangerBG)
	}
	out.WriteString(bold(" DELETE MODE "))
	out.WriteString(fmt.Sprintf(" %d marked  |  Ctrl-D: Toggle  Enter: Confirm  Esc: Cancel", len(s.markedForDelete)))
	return out.String()
}

func (s *TrySelector) centerText(text string) string {
	textWidth := visibleLen(text)
	padding := (s.width - textWidth) / 2
	if padding < 0 {
		padding = 0
	}
	return strings.Repeat(" ", padding) + text
}

func (s *TrySelector) truncateLine(line string) string {
	if visibleLen(line) <= s.width-1 {
		return line
	}
	return truncateWithAnsi(line, s.width-2) + "…"
}

func (s *TrySelector) handleSelection(entry TryEntry) {
	s.selected = &SelectionResult{
		Type: "cd",
		Path: entry.Item.Path,
	}
}

func (s *TrySelector) handleCreateNew() {
	datePrefix := time.Now().Format("2006-01-02")

	if s.inputBuffer != "" {
		finalName := fmt.Sprintf("%s-%s", datePrefix, strings.ReplaceAll(s.inputBuffer, " ", "-"))
		fullPath := filepath.Join(s.basePath, finalName)
		s.selected = &SelectionResult{
			Type: "mkdir",
			Path: fullPath,
		}
		return
	}

	// If no input, would need interactive prompt - for simplicity return nil
	s.selected = nil
}

func (s *TrySelector) confirmBatchDelete(tries []TryEntry) {
	// Find marked items
	markedItems := []TryEntry{}
	for _, t := range tries {
		if indexOf(s.markedForDelete, t.Item.Path) >= 0 {
			markedItems = append(markedItems, t)
		}
	}
	if len(markedItems) == 0 {
		return
	}

	// In test mode, use provided confirmation
	var confirmationBuffer string
	if s.testKeys != nil && len(s.testKeys) > 0 {
		// Read from test keys
		for len(s.testKeys) > 0 {
			ch := s.testKeys[0]
			s.testKeys = s.testKeys[1:]
			if ch == "\r" || ch == "\n" {
				break
			}
			confirmationBuffer += ch
		}
		s.processDeleteConfirmation(markedItems, confirmationBuffer)
		return
	} else if s.testConfirm != "" {
		confirmationBuffer = s.testConfirm
		s.processDeleteConfirmation(markedItems, confirmationBuffer)
		return
	}

	// Interactive delete confirmation
	confirmationCursor := 0
	for {
		s.renderDeleteDialog(markedItems, confirmationBuffer, confirmationCursor)

		key := s.readKey()
		switch key {
		case "\r", "\n":
			s.processDeleteConfirmation(markedItems, confirmationBuffer)
			return
		case "\x1b":
			s.deleteStatus = "Delete cancelled"
			s.markedForDelete = nil
			s.deleteMode = false
			return
		case "\x7f", "\b":
			if confirmationCursor > 0 {
				confirmationBuffer = confirmationBuffer[:confirmationCursor-1] + confirmationBuffer[confirmationCursor:]
				confirmationCursor--
			}
		case "\x03":
			s.deleteStatus = "Delete cancelled"
			s.markedForDelete = nil
			s.deleteMode = false
			return
		default:
			if len(key) == 1 && key[0] >= 32 {
				confirmationBuffer = confirmationBuffer[:confirmationCursor] + key + confirmationBuffer[confirmationCursor:]
				confirmationCursor++
			}
		}
	}
}

func (s *TrySelector) renderDeleteDialog(markedItems []TryEntry, confirmation string, cursor int) {
	var out strings.Builder
	out.WriteString(ansiHome)

	count := len(markedItems)
	plural := "directories"
	if count == 1 {
		plural = "directory"
	}

	// Header
	header := s.centerText(fmt.Sprintf("🗑️%s  Delete %d %s?", accent(""), count, plural))
	out.WriteString("\r" + ansiClearEOL + header + "\n")
	out.WriteString("\r" + ansiClearEOL + dim(strings.Repeat("─", s.width-1)) + "\n")

	// List items
	for _, item := range markedItems {
		line := "🗑️ " + item.Item.Basename
		if colorsEnabled {
			out.WriteString("\r" + ansiClearEOL + colorDangerBG + line + ansiReset + "\n")
		} else {
			out.WriteString("\r" + ansiClearEOL + line + "\n")
		}
	}

	// Blank lines
	out.WriteString("\r" + ansiClearEOL + "\n")
	out.WriteString("\r" + ansiClearEOL + "\n")

	// Confirmation prompt
	prefix := "Type YES to confirm: "
	prompt := s.centerText(dim(prefix) + s.renderInput(confirmation, cursor))
	out.WriteString("\r" + ansiClearEOL + prompt + "\n")

	// Fill remaining space
	usedLines := 2 + len(markedItems) + 3 + 2 // header + items + blanks + prompt + footer
	for i := usedLines; i < s.height-2; i++ {
		out.WriteString("\r" + ansiClearEOL + "\n")
	}

	// Footer
	out.WriteString("\r" + ansiClearEOL + dim(strings.Repeat("─", s.width-1)) + "\n")
	out.WriteString("\r" + ansiClearEOL + s.centerText(dim("Enter: Confirm  Esc: Cancel")))

	out.WriteString(ansiShow)
	out.WriteString(ansiReset)

	s.io.WriteString(out.String())
}

func (s *TrySelector) processDeleteConfirmation(markedItems []TryEntry, confirmation string) {
	if confirmation == "YES" {
		baseReal, err := filepath.EvalSymlinks(s.basePath)
		if err != nil {
			baseReal = s.basePath
		}

		paths := []DeletePath{}
		for _, item := range markedItems {
			targetReal, err := filepath.EvalSymlinks(item.Item.Path)
			if err != nil {
				targetReal = item.Item.Path
			}
			if !strings.HasPrefix(targetReal, baseReal+"/") {
				s.deleteStatus = fmt.Sprintf("Safety check failed: %s not in %s", targetReal, baseReal)
				return
			}
			paths = append(paths, DeletePath{
				Path:     targetReal,
				Basename: item.Item.Basename,
			})
		}

		s.selected = &SelectionResult{
			Type:     "delete",
			Paths:    paths,
			BasePath: baseReal,
		}

		names := make([]string, len(paths))
		for i, p := range paths {
			names[i] = p.Basename
		}
		s.deleteStatus = "Deleted: " + strings.Join(names, ", ")
		s.allTries = nil
		s.fuzzy = nil
		s.markedForDelete = nil
		s.deleteMode = false
	} else {
		s.deleteStatus = "Delete cancelled"
		s.markedForDelete = nil
		s.deleteMode = false
	}
}

func (s *TrySelector) runRenameDialog(entry TryEntry) {
	s.deleteMode = false
	s.markedForDelete = nil

	currentName := entry.Item.Basename
	renameBuffer := currentName
	renameCursor := len(renameBuffer)
	var renameError string

	for {
		s.renderRenameDialog(currentName, renameBuffer, renameCursor, renameError)

		key := s.readKey()
		switch key {
		case "\r", "\n":
			result := s.finalizeRename(entry, renameBuffer)
			if result == "" {
				return
			}
			renameError = result

		case "\x1b", "\x03":
			s.needsRedraw = true
			return

		case "\x7f", "\x08": // Backspace (DEL) or Ctrl-H
			if renameCursor > 0 {
				renameBuffer = renameBuffer[:renameCursor-1] + renameBuffer[renameCursor:]
				renameCursor--
			}
			renameError = ""

		case "\x01": // Ctrl-A
			renameCursor = 0

		case "\x05": // Ctrl-E
			renameCursor = len(renameBuffer)

		case "\x02": // Ctrl-B
			if renameCursor > 0 {
				renameCursor--
			}

		case "\x06": // Ctrl-F
			if renameCursor < len(renameBuffer) {
				renameCursor++
			}

		case "\x0b": // Ctrl-K
			renameBuffer = renameBuffer[:renameCursor]
			renameError = ""

		case "\x17": // Ctrl-W
			if renameCursor > 0 {
				pos := renameCursor - 1
				for pos > 0 && !isWordChar(rune(renameBuffer[pos])) {
					pos--
				}
				for pos > 0 && isWordChar(rune(renameBuffer[pos-1])) {
					pos--
				}
				renameBuffer = renameBuffer[:pos] + renameBuffer[renameCursor:]
				renameCursor = pos
			}
			renameError = ""

		default:
			if len(key) == 1 && isRenamePrintable(key[0]) {
				renameBuffer = renameBuffer[:renameCursor] + key + renameBuffer[renameCursor:]
				renameCursor++
				renameError = ""
			}
		}
	}
}

func (s *TrySelector) renderRenameDialog(currentName, renameBuffer string, renameCursor int, renameError string) {
	var out strings.Builder
	out.WriteString(ansiHome)

	// Header
	header := s.centerText("✏️" + accent("  Rename directory"))
	out.WriteString("\r" + ansiClearEOL + header + "\n")
	out.WriteString("\r" + ansiClearEOL + dim(strings.Repeat("─", s.width-1)) + "\n")

	// Current name
	out.WriteString("\r" + ansiClearEOL + "📁 " + currentName + "\n")

	// Blank lines
	out.WriteString("\r" + ansiClearEOL + "\n")
	out.WriteString("\r" + ansiClearEOL + "\n")

	// New name prompt
	prefix := "New name: "
	prompt := s.centerText(dim(prefix) + s.renderInput(renameBuffer, renameCursor))
	out.WriteString("\r" + ansiClearEOL + prompt + "\n")

	// Error message
	if renameError != "" {
		out.WriteString("\r" + ansiClearEOL + "\n")
		out.WriteString("\r" + ansiClearEOL + s.centerText(bold(renameError)) + "\n")
	}

	// Fill remaining space
	usedLines := 6
	if renameError != "" {
		usedLines += 2
	}
	for i := usedLines; i < s.height-2; i++ {
		out.WriteString("\r" + ansiClearEOL + "\n")
	}

	// Footer
	out.WriteString("\r" + ansiClearEOL + dim(strings.Repeat("─", s.width-1)) + "\n")
	out.WriteString("\r" + ansiClearEOL + s.centerText(dim("Enter: Confirm  Esc: Cancel")))

	out.WriteString(ansiShow)
	out.WriteString(ansiReset)

	s.io.WriteString(out.String())
}

func (s *TrySelector) finalizeRename(entry TryEntry, renameBuffer string) string {
	newName := strings.TrimSpace(strings.ReplaceAll(renameBuffer, " ", "-"))
	oldName := entry.Item.Basename

	if newName == "" {
		return "Name cannot be empty"
	}
	if strings.Contains(newName, "/") {
		return "Name cannot contain /"
	}
	if newName == oldName {
		s.needsRedraw = true
		return "" // No change, just exit
	}
	if _, err := os.Stat(filepath.Join(s.basePath, newName)); err == nil {
		return fmt.Sprintf("Directory exists: %s", newName)
	}

	s.selected = &SelectionResult{
		Type:     "rename",
		OldName:  oldName,
		NewName:  newName,
		BasePath: s.basePath,
	}
	return ""
}

// Utility functions

func isWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func isPrintable(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') || b == '-' || b == '_' || b == '.' || b == ' '
}

func isRenamePrintable(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') || b == '-' || b == '_' || b == '.' || b == ' ' || b == '/'
}

func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}

func contains(slice []int, item int) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func visibleLen(s string) int {
	// Strip ANSI codes
	ansiRe := regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
	stripped := ansiRe.ReplaceAllString(s, "")

	// Count visible width (emoji = 2, others = 1)
	width := 0
	for _, r := range stripped {
		if r >= 0xFE00 && r <= 0xFE0F {
			// Zero-width variation selectors
			continue
		} else if r >= 0x1F300 && r <= 0x1FAFF {
			// Emoji
			width += 2
		} else {
			width++
		}
	}
	return width
}

func truncateWithAnsi(text string, maxLen int) string {
	if visibleLen(text) <= maxLen {
		return text
	}

	visibleCount := 0
	var result strings.Builder
	inAnsi := false

	for _, ch := range text {
		if ch == '\x1b' {
			inAnsi = true
			result.WriteRune(ch)
		} else if inAnsi {
			result.WriteRune(ch)
			if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
				inAnsi = false
			}
		} else {
			cw := 1
			if ch >= 0x1F300 && ch <= 0x1FAFF {
				cw = 2
			}
			if visibleCount+cw > maxLen {
				break
			}
			result.WriteRune(ch)
			visibleCount += cw
		}
	}

	return strings.TrimRight(result.String(), " ")
}

// Write raw bytes to writer
func writeBytes(w io.Writer, data []byte) {
	w.Write(data)
}
