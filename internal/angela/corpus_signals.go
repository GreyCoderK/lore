// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"sort"
	"strings"
	"time"
)

// CorpusSignals holds locally computed analysis of the entire corpus.
// Zero API calls — all analysis is string matching and metadata comparison.
// Extends the CheckCoherence pattern (single-doc) to corpus-wide scope.
type CorpusSignals struct {
	// PotentialPairs are docs of the same type with shared tags but distant dates.
	// These are the most likely contradiction candidates.
	PotentialPairs []DocPair

	// TagClusters groups docs by tag for thematic analysis.
	TagClusters map[string][]string // tag → filenames

	// IsolatedDocs are docs with no shared tags with any other doc.
	IsolatedDocs []string

	// TypeDistribution counts docs per type.
	TypeDistribution map[string]int
}

// DocPair represents two documents that may contradict each other.
type DocPair struct {
	DocA     string // filename
	DocB     string // filename
	Type     string // shared type
	Tags     string // shared tags
	DaysDiff int    // approximate days between dates
}

// AnalyzeCorpusSignals performs local corpus-wide analysis.
// Uses only metadata (no doc content reads, no API calls).
func AnalyzeCorpusSignals(docs []DocSummary) *CorpusSignals {
	signals := &CorpusSignals{
		TagClusters:      make(map[string][]string),
		TypeDistribution: make(map[string]int),
	}

	if len(docs) == 0 {
		return signals
	}

	// Build tag clusters and type distribution
	tagSeen := make(map[string]map[string]bool) // tag → set of filenames
	for _, doc := range docs {
		signals.TypeDistribution[doc.Type]++
		for _, tag := range doc.Tags {
			if tagSeen[tag] == nil {
				tagSeen[tag] = make(map[string]bool)
			}
			tagSeen[tag][doc.Filename] = true
		}
	}
	for tag, files := range tagSeen {
		for f := range files {
			signals.TagClusters[tag] = append(signals.TagClusters[tag], f)
		}
		sort.Strings(signals.TagClusters[tag])
	}

	// Pre-compute: which docs have shared tags with ANY other doc? O(n*t)
	hasAnyConnection := make(map[string]bool)
	for _, files := range tagSeen {
		if len(files) >= 2 {
			for f := range files {
				hasAnyConnection[f] = true
			}
		}
	}

	// Find potential contradictory pairs: same type + shared tags + different dates.
	// Guard: this is O(n²) — cap at 100 docs to keep it bounded.
	// Callers already limit to 50 docs, but this is defense-in-depth.
	maxPairDocs := len(docs)
	if maxPairDocs > 100 {
		maxPairDocs = 100
	}
	for i := 0; i < maxPairDocs; i++ {
		for j := i + 1; j < maxPairDocs; j++ {
			if docs[i].Type != docs[j].Type {
				continue
			}
			shared := sharedTagList(docs[i].Tags, docs[j].Tags)
			if len(shared) == 0 {
				continue
			}

			daysDiff := approxDaysDiff(docs[i].Date, docs[j].Date)
			if daysDiff >= 14 { // 2+ weeks apart = worth flagging
				signals.PotentialPairs = append(signals.PotentialPairs, DocPair{
					DocA:     docs[i].Filename,
					DocB:     docs[j].Filename,
					Type:     docs[i].Type,
					Tags:     strings.Join(shared, ", "),
					DaysDiff: daysDiff,
				})
			}
		}

		// Isolated docs: have tags but no tag overlap with any other doc
		if len(docs[i].Tags) > 0 && !hasAnyConnection[docs[i].Filename] {
			signals.IsolatedDocs = append(signals.IsolatedDocs, docs[i].Filename)
		}
	}

	// Sort pairs by days diff descending (oldest divergences first)
	sort.SliceStable(signals.PotentialPairs, func(i, j int) bool {
		return signals.PotentialPairs[i].DaysDiff > signals.PotentialPairs[j].DaysDiff
	})

	// Limit to top 10 pairs to keep prompt manageable
	if len(signals.PotentialPairs) > 10 {
		signals.PotentialPairs = signals.PotentialPairs[:10]
	}

	return signals
}

// sharedTagList returns the deduplicated list of tags common to both slices.
func sharedTagList(a, b []string) []string {
	bSet := make(map[string]bool, len(b))
	for _, tag := range b {
		bSet[tag] = true
	}
	seen := make(map[string]bool)
	var shared []string
	for _, tag := range a {
		if bSet[tag] && !seen[tag] {
			shared = append(shared, tag)
			seen[tag] = true
		}
	}
	return shared
}

// approxDaysDiff computes an approximate day difference between two YYYY-MM-DD dates.
// Returns 0 if dates can't be parsed.
func approxDaysDiff(dateA, dateB string) int {
	if len(dateA) < 10 || len(dateB) < 10 {
		return 0
	}
	// Simple lexicographic date diff approximation
	// Parse year-month-day manually to avoid time.Parse overhead
	daysA := parseDateToDays(dateA)
	daysB := parseDateToDays(dateB)
	diff := daysA - daysB
	if diff < 0 {
		diff = -diff
	}
	return diff
}

// parseDateToDays converts YYYY-MM-DD to total days since epoch using time.Parse.
func parseDateToDays(date string) int {
	if len(date) < 10 {
		return 0
	}
	t, err := time.Parse("2006-01-02", date[:10])
	if err != nil {
		return 0
	}
	return int(t.Unix() / 86400)
}

// atoi is a simple string-to-int without error handling (dates are validated upstream).
func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}
