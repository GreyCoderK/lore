// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package angela — polish_incremental.go
//
// Orchestrator that re-polishes only changed sections of a document,
// falling back to full polish when conditions aren't met.
//
// The incremental path works as follows:
//
//  1. Parse the document into sections via SplitSections.
//  2. Hash each section and compare with the stored hashes from the
//     polish state file.
//  3. If no sections changed → skip AI entirely, return original doc.
//  4. Build a prompt containing the FULL outline (all headings) plus
//     the body of changed sections ONLY. This gives the AI enough
//     context to maintain coherence while limiting the token cost.
//  5. Parse the AI response, match polished sections back to their
//     originals by heading, and reassemble with unchanged sections.
//  6. Update the polish state with new section hashes.
//
// Fallback triggers:
//   - Fewer than 2 sections → fall back (section splitting unreliable)
//   - First-run (no stored entry) → full polish then store hashes
//   - AI returns unparseable structure → fall back to full polish
//   - Any error in the incremental path → fall back + stderr warning
package angela

import (
	"context"
	"fmt"
	"strings"

	"github.com/greycoderk/lore/internal/domain"
)

// IncrementalOpts configures the incremental polish pass.
type IncrementalOpts struct {
	// Provider is the AI backend (must not be nil).
	Provider domain.AIProvider

	// Meta is the document metadata.
	Meta domain.DocMeta

	// StyleGuide is the loaded style guide content (may be empty).
	StyleGuide string

	// CorpusSummary is the corpus-wide context summary (may be empty).
	CorpusSummary string

	// Personas is the resolved persona profiles for polish.
	Personas []PersonaProfile

	// Audience is the --for flag value (may be empty).
	Audience string

	// ConfigMaxToks is angela.max_tokens from .lorerc (0 = auto).
	ConfigMaxToks int

	// MinChangeLines is cfg.Angela.Polish.Incremental.MinChangeLines.
	// Sections with fewer non-blank lines of change are skipped.
	MinChangeLines int
}

// IncrementalResult describes what happened during an incremental
// polish attempt.
type IncrementalResult struct {
	// Polished is the resulting document content.
	Polished string

	// WasIncremental is true when incremental mode was used (not all
	// sections re-polished). False when a fallback to full polish
	// happened.
	WasIncremental bool

	// ChangedCount is the number of sections that were sent to the AI.
	ChangedCount int

	// Skipped is true when 0 sections changed → no AI call at all.
	Skipped bool

	// NewHashes is the section hash map to persist into the state
	// file after a successful polish.
	NewHashes map[string]string
}

// PolishIncremental attempts a section-level incremental polish.
//
// `storedHashes` is the PolishStateEntry.SectionHashes from the
// previous run (nil or empty on first run → falls back to full).
//
// Returns an IncrementalResult; any error means a hard failure that
// should abort the polish entirely (e.g. provider error). The caller
// is responsible for fallback-on-warning via the WasIncremental flag.
func PolishIncremental(ctx context.Context, doc string, storedHashes map[string]string, o IncrementalOpts) (*IncrementalResult, error) {
	if o.Provider == nil {
		return nil, fmt.Errorf("angela: polish incremental: no AI provider")
	}
	if len(doc) > maxAIInputSize {
		return nil, fmt.Errorf("angela: polish incremental: document too large (%d bytes, max %d)", len(doc), maxAIInputSize)
	}

	sections := SplitSections(doc)

	// Fewer than 2 sections → fall back to full.
	if len(sections) < 2 {
		return fullPolishFallback(ctx, doc, sections, o)
	}

	// First-run (no stored hashes) → full polish + capture hashes.
	if len(storedHashes) == 0 {
		return fullPolishFallback(ctx, doc, sections, o)
	}

	// Detect changed sections.
	changedIdx, allUnchanged := DetectChangedSections(sections, storedHashes, o.MinChangeLines)

	// No changes → skip AI entirely.
	if allUnchanged {
		return &IncrementalResult{
			Polished:       doc,
			WasIncremental: true,
			Skipped:        true,
			NewHashes:      HashSections(sections),
		}, nil
	}

	// Build incremental prompt.
	sysPrompt, userContent := buildIncrementalPrompt(sections, changedIdx, o)

	wordCount := 0
	for _, i := range changedIdx {
		wordCount += len(strings.Fields(sections[i].Body))
	}
	maxTokens := ResolveMaxTokens("polish", wordCount, o.ConfigMaxToks)

	result, err := o.Provider.Complete(ctx, userContent,
		domain.WithSystem(sysPrompt),
		domain.WithMaxTokens(maxTokens))
	if err != nil {
		return nil, fmt.Errorf("angela: polish incremental: %w", err)
	}

	result = stripCodeFence(result)

	// Reassemble polished sections into the full document.
	merged, ok := reassemble(sections, changedIdx, result)
	if !ok {
		// AI returned unparseable structure → fall back to full.
		return fullPolishFallback(ctx, doc, sections, o)
	}

	return &IncrementalResult{
		Polished:       merged,
		WasIncremental: true,
		ChangedCount:   len(changedIdx),
		NewHashes:      HashSections(SplitSections(merged)),
	}, nil
}

// buildIncrementalPrompt constructs the system+user prompt for
// incremental polish. The system prompt is the standard polish
// prompt; the user content includes the full outline + only the
// changed section bodies.
func buildIncrementalPrompt(sections []Section, changedIdx []int, o IncrementalOpts) (string, string) {
	// Build full outline.
	var outline strings.Builder
	outline.WriteString("## Document outline\n\n")
	for _, s := range sections {
		if s.Heading != "" {
			outline.WriteString(s.Heading)
			outline.WriteByte('\n')
		}
	}

	// Build changed sections block.
	changedSet := make(map[int]bool, len(changedIdx))
	for _, i := range changedIdx {
		changedSet[i] = true
	}
	var changed strings.Builder
	changed.WriteString("\n## Sections to polish\n\n")
	changed.WriteString("Return ONLY the polished versions of these sections, each starting with its original heading.\n\n")
	for _, i := range changedIdx {
		s := sections[i]
		if s.Heading != "" {
			changed.WriteString(s.Heading)
			changed.WriteByte('\n')
		}
		changed.WriteString(s.Body)
		changed.WriteString("\n\n---\n\n")
	}

	// Reuse the standard polish system prompt.
	audience := ""
	if o.Audience != "" {
		audience = sanitizeShortField(o.Audience)
	}
	sysPrompt, _ := BuildPolishPrompt("", o.Meta, o.StyleGuide, o.CorpusSummary, o.Personas, audience)

	userContent := outline.String() + changed.String()
	return sysPrompt, userContent
}

// reassemble merges the AI-polished sections back into the original
// document. The AI is instructed to return sections with their
// original headings, so we match by heading. Returns (merged, true)
// on success or ("", false) if the parse fails.
func reassemble(original []Section, changedIdx []int, aiResult string) (string, bool) {
	// Parse the AI response into sections. SplitSections may fold
	// the first heading into the preamble when there is no content
	// before it — parseAISections handles this edge case by treating
	// a line starting with "## " at the very beginning as a heading.
	polished := parseAISections(aiResult)

	// Index polished sections by heading for O(1) lookup.
	polishedByHeading := make(map[string]Section, len(polished))
	for _, s := range polished {
		if s.Heading != "" {
			polishedByHeading[s.Heading] = s
		}
	}

	// Verify we got back at least some of the sections we sent.
	changedSet := make(map[int]bool, len(changedIdx))
	for _, i := range changedIdx {
		changedSet[i] = true
	}
	matched := 0
	for _, i := range changedIdx {
		if _, ok := polishedByHeading[original[i].Heading]; ok {
			matched++
		}
	}
	// If we matched fewer than half the changed sections, consider it
	// a parse failure and fall back.
	if matched == 0 || (len(changedIdx) > 1 && matched*2 < len(changedIdx)) {
		return "", false
	}

	// Merge: for each original section, use the polished version if
	// it was changed and the AI returned it, otherwise keep original.
	merged := make([]Section, len(original))
	for i, s := range original {
		if changedSet[i] {
			if p, ok := polishedByHeading[s.Heading]; ok {
				merged[i] = Section{
					Heading: s.Heading,
					Body:    PostProcess(s.Body, p.Body),
					Index:   s.Index,
				}
				continue
			}
		}
		merged[i] = s
	}

	return MergeSections(merged), true
}

// parseAISections splits an AI response into sections by `## `
// headings. Unlike SplitSections (designed for full documents with
// front-matter preambles), this parser treats a `## ` on the very
// first non-blank line as a heading rather than folding it into a
// preamble. This is needed because the AI typically starts its
// response with the first heading and no preamble.
func parseAISections(s string) []Section {
	lines := strings.Split(s, "\n")
	var sections []Section
	var curHeading string
	var curBody []string

	flush := func() {
		if curHeading != "" {
			sections = append(sections, Section{
				Heading: curHeading,
				Body:    strings.Join(curBody, "\n"),
				Index:   len(sections),
			})
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Accept ### (and deeper) headings too — normalize to
		// ## form so they match the original section headings used as
		// map keys. The AI sometimes emits ### instead of ##.
		if strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "### ") {
			heading := trimmed
			// Strip extra leading '#' chars to normalize to "## Title".
			for strings.HasPrefix(heading, "### ") {
				heading = "## " + heading[4:]
			}
			flush()
			curHeading = heading
			curBody = nil
			continue
		}
		if curHeading != "" {
			curBody = append(curBody, line)
		}
		// Lines before the first heading are discarded (AI preamble text).
	}
	flush()
	return sections
}

// fullPolishFallback runs a standard full Polish and wraps the
// result in an IncrementalResult with WasIncremental=false.
func fullPolishFallback(ctx context.Context, doc string, sections []Section, o IncrementalOpts) (*IncrementalResult, error) {
	polished, err := Polish(ctx, o.Provider, doc, o.Meta, o.StyleGuide, o.CorpusSummary, o.Personas, PolishOpts{
		Audience:      o.Audience,
		ConfigMaxToks: o.ConfigMaxToks,
	})
	if err != nil {
		return nil, err
	}
	return &IncrementalResult{
		Polished:       polished,
		WasIncremental: false,
		ChangedCount:   len(sections),
		NewHashes:      HashSections(SplitSections(polished)),
	}, nil
}
