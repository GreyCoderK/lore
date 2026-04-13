// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"strings"
	"unicode/utf8"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
)

// Suggestion represents a single review finding from Angela draft analysis.
// JSON tags are snake-case to match the documented draft --format=json schema
// The field order (category, severity, message) is the same as
// the human-readable output so JSON consumers and readers see the same layout.
//
// DiffStatus is populated by the differential runner to tag each
// suggestion as "new", "persisting", or "resolved" relative to the previous
// draft state. Empty in single-run mode or when differential is disabled,
// hence the omitempty JSON tag.
type Suggestion struct {
	Category   string `json:"category"` // "structure", "completeness", "style", "coherence", "persona"
	Severity   string `json:"severity"` // "info", "warning", "error"
	Message    string `json:"message"`
	DiffStatus string `json:"diff_status,omitempty"` // "new" | "persisting" | "resolved"
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
	suggestions = append(suggestions, checkStructure(body, meta, guide)...)

	// Completeness checks
	suggestions = append(suggestions, checkCompletenessWithSections(body, meta, guide, corpus, sections)...)

	// Persona-specific draft checks. The hard free-form exclusion has been
	// replaced by smart selection in SelectPersonasForDoc — by the time we
	// reach here, the `personas`
	// slice already contains only the personas appropriate for this
	// document type. We just run them unconditionally.
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
// Free-form document types (note, guide, tutorial, reference, index) skip the
// What/Why/Alternatives/Impact requirements entirely — these checks only make
// sense for decision/feature/bugfix/refactor where the "why behind the code"
// framing applies. Without this gate, standalone mode on a mkdocs site
// produces hundreds of false-positive warnings.
func checkStructure(body string, meta domain.DocMeta, guide *StyleGuide) []Suggestion {
	var suggestions []Suggestion

	requireWhy := true
	requireAlternatives := false
	if guide != nil {
		requireWhy = guide.RequireWhy
		requireAlternatives = guide.RequireAlternatives
	}

	// Free-form document types don't follow the lore section conventions.
	if isFreeFormType(meta.Type) {
		requireWhy = false
		requireAlternatives = false
	}

	hasWhy := hasSection(body, "## Why")
	hasWhat := hasSection(body, "## What")
	hasAlternatives := hasSection(body, "## Alternatives")
	hasImpact := hasSection(body, "## Impact")

	t := i18n.T().Angela
	if !isFreeFormType(meta.Type) && !hasWhat {
		suggestions = append(suggestions, Suggestion{
			Category: "structure",
			Severity: "warning",
			Message:  t.DraftMissingWhat,
		})
	}

	if requireWhy && !hasWhy {
		suggestions = append(suggestions, Suggestion{
			Category: "structure",
			Severity: "warning",
			Message:  t.DraftMissingWhy,
		})
	}

	if requireAlternatives && !hasAlternatives {
		suggestions = append(suggestions, Suggestion{
			Category: "structure",
			Severity: "warning",
			Message:  t.DraftMissingAltWarn,
		})
	} else if !requireAlternatives && !hasAlternatives && !isFreeFormType(meta.Type) {
		suggestions = append(suggestions, Suggestion{
			Category: "structure",
			Severity: "info",
			Message:  t.DraftMissingAltInfo,
		})
	}

	if !hasImpact && !isFreeFormType(meta.Type) {
		suggestions = append(suggestions, Suggestion{
			Category: "structure",
			Severity: "info",
			Message:  t.DraftMissingImpact,
		})
	}

	// Body length check. For free-form types (notes, guides, index pages,
	// tutorials) a short body is often legitimate (landing page, stub,
	// section divider), so we downgrade to info to avoid breaking external
	// CI pipelines that run --fail-on warning on a mkdocs site.
	bodyRunes := utf8.RuneCountInString(body)
	if bodyRunes < 50 {
		severity := "warning"
		if isFreeFormType(meta.Type) {
			severity = "info"
		}
		suggestions = append(suggestions, Suggestion{
			Category: "structure",
			Severity: severity,
			Message:  t.DraftBodyTooShort,
		})
	}

	// Max body length (style guide)
	if guide != nil && guide.MaxBodyLength > 0 && bodyRunes > guide.MaxBodyLength {
		suggestions = append(suggestions, Suggestion{
			Category: "style",
			Severity: "info",
			Message:  t.DraftBodyExceedsMax,
		})
	}

	return suggestions
}

// checkCompletenessWithSections validates metadata and content depth using pre-parsed sections.
// Completeness checks tied to lore conventions (scope, related refs, ## Why
// substance) are skipped for free-form types — these concepts don't apply to
// external mkdocs/docusaurus sites.
func checkCompletenessWithSections(_ string, meta domain.DocMeta, guide *StyleGuide, corpus []domain.DocMeta, sections map[string]string) []Suggestion {
	var suggestions []Suggestion
	freeForm := isFreeFormType(meta.Type)
	t := i18n.T().Angela

	minTags := 0
	if guide != nil {
		minTags = guide.MinTags
	}

	// Scope check — if document has a commit but no scope, suggest adding one.
	// Only relevant for lore commit-capture docs.
	if !freeForm && meta.Commit != "" && meta.Scope == "" {
		suggestions = append(suggestions, Suggestion{
			Category: "completeness",
			Severity: "info",
			Message:  t.DraftAddScope,
		})
	}

	// Tags check — keep for all types, tags are a universal concept.
	if len(meta.Tags) < minTags || len(meta.Tags) == 0 {
		suggestions = append(suggestions, Suggestion{
			Category: "completeness",
			Severity: "info",
			Message:  t.DraftAddTags,
		})
	}

	// Related references (only if corpus is large enough). This is a lore
	// convention — external doc sites don't use the `related:` field, so
	// skip it for free-form types.
	if !freeForm && len(corpus) > 5 && len(meta.Related) == 0 {
		suggestions = append(suggestions, Suggestion{
			Category: "completeness",
			Severity: "info",
			Message:  t.DraftAddRelated,
		})
	}

	// Why section substance — only check if the doc actually has a ## Why.
	// Free-form types may legitimately have no Why section.
	whyContent := sections["## Why"]
	if whyContent != "" && utf8.RuneCountInString(strings.TrimSpace(whyContent)) < 20 {
		suggestions = append(suggestions, Suggestion{
			Category: "completeness",
			Severity: "warning",
			Message:  t.DraftWhyTooBrief,
		})
	}

	return suggestions
}

// isFreeFormType reports whether a document type is free-form narrative rather
// than one that follows the strict lore section conventions
// (What/Why/Alternatives/Impact).
//
// The predicate is inverted by design: we list the narrow set of *strict*
// types (the ones tied to the lore commit-capture workflow) and return true
// for everything else. This makes Angela safe to run on external mkdocs /
// docusaurus / astro sites where the front-matter `type` field may be
// anything (blog-post, howto, concept, explanation, release, landing…) —
// unknown types default to free-form instead of producing hundreds of
// false-positive warnings about missing "## What" sections.
//
// Strict types (whitelist): decision, feature, bugfix, refactor.
// Empty type (no front matter or missing field) is also strict only when
// the user has explicitly opted in to lore conventions via a real hook —
// standalone mode goes through BuildPlainMeta which assigns type="note",
// which is free-form via the default branch below.
func isFreeFormType(t string) bool {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "decision", "feature", "bugfix", "refactor":
		return false
	case "":
		// Empty type with front matter means a lore doc where the author
		// simply omitted the field — treat as strict so they still get
		// the structure guidance.
		return false
	}
	return true
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

