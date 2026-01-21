// Package fuzzy provides fuzzy string matching
package fuzzy

import (
	"math"
	"strings"
)

// Item represents an item to be matched
type Item struct {
	Text      string
	Path      string
	BaseScore float64
}

// Entry is an internal representation for matching
type Entry struct {
	Data      Item
	Text      string
	TextLower string
	BaseScore float64
	TextRunes []rune
}

// Matcher performs fuzzy matching on a set of entries
type Matcher struct {
	Entries []Entry
}

// Match represents a successful fuzzy match
type Match struct {
	Entry     Item
	Positions []int
	Score     float64
}

// New creates a new fuzzy matcher
func New(items []Item) *Matcher {
	entries := make([]Entry, 0, len(items))
	for _, item := range items {
		entries = append(entries, Entry{
			Data:      item,
			Text:      item.Text,
			TextLower: strings.ToLower(item.Text),
			BaseScore: item.BaseScore,
			TextRunes: []rune(strings.ToLower(item.Text)),
		})
	}
	return &Matcher{Entries: entries}
}

// Match finds all entries matching the query
func (m *Matcher) Match(query string) []Match {
	results := make([]Match, 0, len(m.Entries))
	for _, entry := range m.Entries {
		score, positions, ok := calculateMatch(entry, query)
		if !ok {
			continue
		}
		results = append(results, Match{
			Entry:     entry.Data,
			Positions: positions,
			Score:     score,
		})
	}

	sortByScore(results)
	return results
}

func sortByScore(matches []Match) {
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].Score > matches[i].Score {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}
}

func calculateMatch(entry Entry, query string) (float64, []int, bool) {
	positions := []int{}
	score := entry.BaseScore
	if query == "" {
		return score, positions, true
	}

	queryLower := strings.ToLower(query)
	queryRunes := []rune(queryLower)
	textRunes := entry.TextRunes

	lastPos := -1
	pos := 0
	for _, qc := range queryRunes {
		found := -1
		for i := pos; i < len(textRunes); i++ {
			if textRunes[i] == qc {
				found = i
				break
			}
		}
		if found == -1 {
			return 0, nil, false
		}
		positions = append(positions, found)
		score += 1.0

		if found == 0 || !isAlphaNum(textRunes[found-1]) {
			score += 1.0
		}

		if lastPos >= 0 {
			gap := found - lastPos - 1
			if gap < 16 {
				score += sqrtTable[gap]
			} else {
				score += 2.0 / math.Sqrt(float64(gap+1))
			}
		}

		lastPos = found
		pos = found + 1
	}

	if lastPos >= 0 {
		score *= float64(len(queryRunes)) / float64(lastPos+1)
	}
	score *= 10.0 / (float64(len([]rune(entry.Text))) + 10.0)

	return score, positions, true
}

var sqrtTable = func() []float64 {
	table := make([]float64, 17)
	for i := 0; i <= 16; i++ {
		table[i] = 2.0 / math.Sqrt(float64(i+1))
	}
	return table
}()

func isAlphaNum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}
