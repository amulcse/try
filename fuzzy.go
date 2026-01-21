package main

import (
	"math"
	"strings"
)

type FuzzyEntry struct {
	Data      TryItem
	Text      string
	TextLower string
	BaseScore float64
	TextRunes []rune
}

type Fuzzy struct {
	Entries []FuzzyEntry
}

func NewFuzzy(entries []TryItem) *Fuzzy {
	fe := make([]FuzzyEntry, 0, len(entries))
	for _, e := range entries {
		text := e.Text
		fe = append(fe, FuzzyEntry{
			Data:      e,
			Text:      text,
			TextLower: strings.ToLower(text),
			BaseScore: e.BaseScore,
			TextRunes: []rune(strings.ToLower(text)),
		})
	}
	return &Fuzzy{Entries: fe}
}

type FuzzyMatch struct {
	Entry     TryItem
	Positions []int
	Score     float64
}

func (f *Fuzzy) Match(query string) []FuzzyMatch {
	query = query
	results := make([]FuzzyMatch, 0, len(f.Entries))
	for _, entry := range f.Entries {
		score, positions, ok := calculateMatch(entry, query)
		if !ok {
			continue
		}
		results = append(results, FuzzyMatch{
			Entry:     entry.Data,
			Positions: positions,
			Score:     score,
		})
	}

	sortByScore(results)
	return results
}

func sortByScore(matches []FuzzyMatch) {
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].Score > matches[i].Score {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}
}

func calculateMatch(entry FuzzyEntry, query string) (float64, []int, bool) {
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

