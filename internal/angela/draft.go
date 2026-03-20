// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"strings"
	"unicode/utf8"

	"github.com/greycoderk/lore/internal/domain"
)

// Suggestion represents a single review finding from Angela draft analysis.
type Suggestion struct {
	Category string // "structure", "completeness", "style", "coherence", "persona"
	Message  string
	Severity string // "info", "warning"
}

// AnalyzeDraft performs local structural analysis of a document.
// Zero API calls — fully deterministic. Returns nil if no suggestions.
// When personas is non-nil, persona-specific draft checks are included.
func AnalyzeDraft(doc string, meta domain.DocMeta, guide *StyleGuide, corpus []domain.DocMeta, personas []PersonaProfile) []Suggestion {
	var suggestions []Suggestion

	// Strip front matter to get body only
	body := stripFrontMatter(doc)

	// Parse sections once — shared by completeness checks and persona draft checks
	sections := extractAllSections(body)

	// Structural checks
	suggestions = append(suggestions, checkStructure(body, guide)...)

	// Completeness checks
	suggestions = append(suggestions, checkCompletenessWithSections(body, meta, guide, corpus, sections)...)

	// Persona-specific draft checks (AC-3)
	if len(personas) > 0 {
		suggestions = append(suggestions, runPersonaDraftChecksWithSections(body, personas, sections)...)
	}

	return suggestions
}

// stripFrontMatter removes YAML front matter (---\n...\n---\n) from doc content.
func stripFrontMatter(doc string) string {
	if !strings.HasPrefix(doc, "---\n") {
		return doc
	}
	end := strings.Index(doc[4:], "\n---\n")
	if end < 0 {
		return doc
	}
	return doc[4+end+5:]
}

// checkStructure validates document sections exist and are non-empty.
func checkStructure(body string, guide *StyleGuide) []Suggestion {
	var suggestions []Suggestion

	requireWhy := true
	requireAlternatives := false
	if guide != nil {
		requireWhy = guide.RequireWhy
		requireAlternatives = guide.RequireAlternatives
	}

	hasWhy := hasSection(body, "## Why")
	hasWhat := hasSection(body, "## What")
	hasAlternatives := hasSection(body, "## Alternatives")
	hasImpact := hasSection(body, "## Impact")

	if !hasWhat {
		suggestions = append(suggestions, Suggestion{
			Category: "structure",
			Severity: "warning",
			Message:  `Section "## What" is missing`,
		})
	}

	if requireWhy && !hasWhy {
		suggestions = append(suggestions, Suggestion{
			Category: "structure",
			Severity: "warning",
			Message:  `Section "## Why" is missing`,
		})
	}

	if requireAlternatives && !hasAlternatives {
		suggestions = append(suggestions, Suggestion{
			Category: "structure",
			Severity: "warning",
			Message:  `Section "## Alternatives" is missing`,
		})
	} else if !requireAlternatives && !hasAlternatives {
		suggestions = append(suggestions, Suggestion{
			Category: "structure",
			Severity: "info",
			Message:  `Section "## Alternatives" is missing`,
		})
	}

	if !hasImpact {
		suggestions = append(suggestions, Suggestion{
			Category: "structure",
			Severity: "info",
			Message:  `Section "## Impact" is missing`,
		})
	}

	// Body length check
	bodyRunes := utf8.RuneCountInString(body)
	if bodyRunes < 50 {
		suggestions = append(suggestions, Suggestion{
			Category: "structure",
			Severity: "warning",
			Message:  "Document body is too short (< 50 characters)",
		})
	}

	// Max body length (style guide)
	if guide != nil && guide.MaxBodyLength > 0 && bodyRunes > guide.MaxBodyLength {
		suggestions = append(suggestions, Suggestion{
			Category: "style",
			Severity: "info",
			Message:  "Body exceeds recommended maximum length",
		})
	}

	return suggestions
}

// checkCompletenessWithSections validates metadata and content depth using pre-parsed sections.
func checkCompletenessWithSections(body string, meta domain.DocMeta, guide *StyleGuide, corpus []domain.DocMeta, sections map[string]string) []Suggestion {
	var suggestions []Suggestion

	minTags := 0
	if guide != nil {
		minTags = guide.MinTags
	}

	// Tags check
	if len(meta.Tags) < minTags || len(meta.Tags) == 0 {
		suggestions = append(suggestions, Suggestion{
			Category: "completeness",
			Severity: "info",
			Message:  "Consider adding tags for discoverability",
		})
	}

	// Related references (only if corpus is large enough)
	if len(corpus) > 5 && len(meta.Related) == 0 {
		suggestions = append(suggestions, Suggestion{
			Category: "completeness",
			Severity: "info",
			Message:  "Consider adding related document references",
		})
	}

	// Why section substance
	whyContent := sections["## Why"]
	if whyContent != "" && utf8.RuneCountInString(strings.TrimSpace(whyContent)) < 20 {
		suggestions = append(suggestions, Suggestion{
			Category: "completeness",
			Severity: "warning",
			Message:  `Section "## Why" is too brief (< 20 characters)`,
		})
	}

	return suggestions
}

// hasSection checks if a markdown section header exists in the body.
func hasSection(body, header string) bool {
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == header {
			return true
		}
	}
	return false
}

