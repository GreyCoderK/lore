// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"sort"
	"strings"
	"unicode/utf8"
)

// sectionScore holds a parsed section with its relevance score.
type sectionScore struct {
	heading string
	body    string
	score   int
}

// scoreHeading returns a relevance score for a section heading.
// Supports both English and French keywords. Case-insensitive substring match.
func scoreHeading(heading string) int {
	h := strings.ToLower(heading)

	// Score 30: Why / Context / Motivation
	for _, kw := range []string{"why", "context", "motivation", "background", "pourquoi", "contexte"} {
		if strings.Contains(h, kw) {
			return 30
		}
	}

	// Score 25: Decision
	for _, kw := range []string{"decision", "decided", "resolution", "décision", "résolution"} {
		if strings.Contains(h, kw) {
			return 25
		}
	}

	// Score 20: What / Changes / Implementation
	for _, kw := range []string{"what", "changes", "implementation", "quoi", "changements", "implémentation"} {
		if strings.Contains(h, kw) {
			return 20
		}
	}

	// Score 15: Alternatives / Impact
	for _, kw := range []string{"alternatives", "trade-offs", "impact", "consequences", "compromis", "conséquences"} {
		if strings.Contains(h, kw) {
			return 15
		}
	}

	return 5
}

// ExtractAdaptiveSummary extracts top 3 sections from a document body,
// scored by semantic relevance rather than length.
// Each section is truncated to maxRunes/N runes (N = number of selected sections).
// Budget is not redistributed from short sections to longer ones.
func ExtractAdaptiveSummary(body string, maxRunes int) string {
	sections := parseAllSections(body)

	if len(sections) == 0 {
		// No ## sections — return first maxRunes of body
		body = strings.TrimSpace(body)
		return truncateRunes(body, maxRunes)
	}

	// Score each section
	scored := make([]sectionScore, len(sections))
	for i, sec := range sections {
		score := scoreHeading(sec.heading)
		if utf8.RuneCountInString(sec.body) > 100 {
			score += 10
		}
		scored[i] = sectionScore{heading: sec.heading, body: sec.body, score: score}
	}

	// Sort by score descending (stable to preserve doc order for equal scores)
	sort.SliceStable(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Select top 3 (or fewer)
	n := 3
	if len(scored) < n {
		n = len(scored)
	}
	selected := scored[:n]

	// Truncate each to maxRunes/n runes
	perSection := maxRunes / n
	var parts []string
	for _, sec := range selected {
		truncated := truncateRunes(sec.body, perSection)
		parts = append(parts, "["+sec.heading+"] "+truncated)
	}

	return strings.Join(parts, " | ")
}
