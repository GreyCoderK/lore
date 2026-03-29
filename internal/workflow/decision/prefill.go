// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package decision

import (
	"regexp"
	"strings"
)

// conventionalPrefix matches "type(scope): " or "type: " patterns.
var conventionalPrefix = regexp.MustCompile(`^[a-z]+(\([^)]*\))?!?:\s*`)

// Separator keywords for splitting What/Why. Order matters — first match wins.
var separators = []string{
	// Multi-word first (avoid partial matches)
	"in order to", "so that", "due to", "afin de", "parce que", "suite à", "suite a",
	// Single-word
	"because", "for", "pour", "car",
}

// ExtractImplicitWhy splits a commit subject into What and Why parts.
// Returns what, why, confidence. If no separator found, confidence is 0.
func ExtractImplicitWhy(subject string) (string, string, float64) {
	// Strip conventional commit prefix
	raw := conventionalPrefix.ReplaceAllString(subject, "")
	raw = strings.TrimSpace(raw)

	if raw == "" {
		return "", "", 0
	}

	// Try each separator (case-insensitive, word boundary via space context)
	lower := strings.ToLower(raw)
	for _, sep := range separators {
		idx := strings.Index(lower, " "+sep+" ")
		if idx < 0 {
			continue
		}

		what := strings.TrimSpace(raw[:idx])
		why := strings.TrimSpace(raw[idx+len(sep)+2:])

		if why == "" {
			continue
		}

		confidence := 0.7
		if len(why) < 10 {
			confidence = 0.4
		}

		return what, why, confidence
	}

	// No separator found
	return raw, "", 0
}
