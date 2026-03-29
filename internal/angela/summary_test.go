// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestScoreHeading_Priority(t *testing.T) {
	tests := []struct {
		heading string
		want    int
	}{
		{"Why", 30},
		{"Context", 30},
		{"Motivation", 30},
		{"Background", 30},
		{"Pourquoi", 30},
		{"Contexte", 30},
		{"Decision", 25},
		{"Decided", 25},
		{"Resolution", 25},
		{"Décision", 25},
		{"What", 20},
		{"Changes", 20},
		{"Implementation", 20},
		{"Quoi", 20},
		{"Changements", 20},
		{"Implémentation", 20},
		{"Alternatives", 15},
		{"Trade-offs", 15},
		{"Impact", 15},
		{"Consequences", 15},
		{"Compromis", 15},
		{"Conséquences", 15},
		{"Random heading", 5},
		{"Status", 5},
	}
	for _, tt := range tests {
		t.Run(tt.heading, func(t *testing.T) {
			got := scoreHeading(tt.heading)
			if got != tt.want {
				t.Errorf("scoreHeading(%q) = %d, want %d", tt.heading, got, tt.want)
			}
		})
	}
}

func TestScoreHeading_CaseInsensitive(t *testing.T) {
	if scoreHeading("WHY THIS MATTERS") != 30 {
		t.Error("should be case-insensitive")
	}
	if scoreHeading("pourquoi ce choix") != 30 {
		t.Error("should match French substring")
	}
}

func TestExtractAdaptiveSummary_ScoresOverLength(t *testing.T) {
	// Why section is short but should be selected over a long Status section
	body := "## Status\n" + strings.Repeat("status detail ", 50) +
		"\n\n## Why\nBecause reasons.\n\n## What\nThe change."

	result := ExtractAdaptiveSummary(body, 450)

	if !strings.Contains(result, "[Why]") {
		t.Errorf("should select Why (score 30), got: %s", result)
	}
	if !strings.Contains(result, "[What]") {
		t.Errorf("should select What (score 20), got: %s", result)
	}
}

func TestExtractAdaptiveSummary_BonusApplied(t *testing.T) {
	// Two sections with same base score (5), one > 100 chars gets +10
	short := "Short."
	long := strings.Repeat("a", 150)
	body := "## SectionA\n" + long + "\n\n## SectionB\n" + short

	result := ExtractAdaptiveSummary(body, 450)

	// SectionA should rank higher (5+10=15 vs 5)
	if !strings.Contains(result, "[SectionA]") {
		t.Errorf("should select SectionA (bonus applied), got: %s", result)
	}
}

func TestExtractAdaptiveSummary_Truncation(t *testing.T) {
	longBody := strings.Repeat("x", 500)
	body := "## Why\n" + longBody + "\n\n## What\n" + longBody + "\n\n## Impact\n" + longBody

	result := ExtractAdaptiveSummary(body, 450)

	// Each section should be max 150 runes (450/3)
	parts := strings.Split(result, " | ")
	for _, part := range parts {
		// Part includes "[Heading] " prefix
		if utf8.RuneCountInString(part) > 200 { // generous margin for heading prefix
			t.Errorf("section too long: %d runes", utf8.RuneCountInString(part))
		}
	}
}

func TestExtractAdaptiveSummary_LessThan3Sections(t *testing.T) {
	body := "## Why\nBecause.\n\n## What\nThe thing."

	result := ExtractAdaptiveSummary(body, 450)

	// Should return both sections, each gets 450/2 = 225 runes max
	if !strings.Contains(result, "[Why]") || !strings.Contains(result, "[What]") {
		t.Errorf("should return all available sections, got: %s", result)
	}
}

func TestExtractAdaptiveSummary_NoSections(t *testing.T) {
	body := "Just plain text without any headers."

	result := ExtractAdaptiveSummary(body, 450)

	if result != "Just plain text without any headers." {
		t.Errorf("no sections → return body truncated, got: %q", result)
	}
}

func TestExtractAdaptiveSummary_EmptyBody(t *testing.T) {
	result := ExtractAdaptiveSummary("", 450)

	if result != "" {
		t.Errorf("empty body → empty result, got: %q", result)
	}
}

func TestExtractAdaptiveSummary_FrenchKeywords(t *testing.T) {
	body := "## Pourquoi\nParce que c'est important.\n\n## Décision\nOn choisit X.\n\n## Statut\nEn cours."

	result := ExtractAdaptiveSummary(body, 450)

	if !strings.Contains(result, "[Pourquoi]") {
		t.Errorf("should select Pourquoi (score 30), got: %s", result)
	}
	if !strings.Contains(result, "[Décision]") {
		t.Errorf("should select Décision (score 25), got: %s", result)
	}
}
