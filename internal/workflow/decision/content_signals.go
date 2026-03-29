// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package decision

import (
	"fmt"
	"regexp"
	"strings"
)

type diffPattern struct {
	name  string
	re    *regexp.Regexp
	score int
}

var diffPatterns = []diffPattern{
	{"security", regexp.MustCompile(`(?i)(api_key|secret|token|password|auth)`), 15},
	{"public-api", regexp.MustCompile(`(?m)^[+-].*\b(func [A-Z]|export function|export class|export const)\b`), 10},
	{"infra", regexp.MustCompile(`(?i)(database|redis|kafka|\.port|\.host|endpoint)`), 10},
	{"entity-deleted", regexp.MustCompile(`(?m)^-.*\b(func|class|def|interface)\b`), 8},
	{"tech-debt", regexp.MustCompile(`(?m)^\+.*\b(TODO|FIXME|HACK|WORKAROUND)\b`), 5},
}

// ScanDiffContent analyses diff lines for security, API, infra, deletion, and tech-debt patterns.
// Each pattern scores at most once per commit (deduplication).
// Analysis is limited to the first 1000 lines of the diff for performance.
func ScanDiffContent(diffContent string) SignalScore {
	if diffContent == "" {
		return SignalScore{Name: "diff-content", Input: "(empty)", Score: 0, Reason: "no diff"}
	}

	// Limit to first 1000 lines
	limited := diffContent
	if idx := nthIndex(diffContent, '\n', 1000); idx >= 0 {
		limited = diffContent[:idx]
	}

	total := 0
	matched := make([]string, 0, len(diffPatterns))

	for _, p := range diffPatterns {
		if p.re.MatchString(limited) {
			total += p.score
			matched = append(matched, fmt.Sprintf("%s +%d", p.name, p.score))
		}
	}

	reason := "no patterns"
	if len(matched) > 0 {
		reason = strings.Join(matched, ", ")
	}

	// Count lines for the input label
	lineCount := strings.Count(limited, "\n") + 1
	return SignalScore{Name: "diff-content", Input: fmt.Sprintf("%d lines", lineCount), Score: total, Reason: reason}
}

// nthIndex returns the index of the nth occurrence of sep in s, or -1 if not found.
func nthIndex(s string, sep byte, n int) int {
	count := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			count++
			if count >= n {
				return i
			}
		}
	}
	return -1
}
