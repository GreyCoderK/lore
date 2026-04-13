// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"context"
	"fmt"
	"strings"

	"github.com/greycoderk/lore/internal/domain"
)

// Section represents a document section split by ## headings.
type Section struct {
	Heading string // "## 4. Logique Métier" (empty for preamble before first ##)
	Body    string // content until next ## or EOF
	Index   int
}

// SplitSections divides a document into sections by ## headings.
// Section 0 is the preamble (front matter + content before first ##).
func SplitSections(doc string) []Section {
	lines := strings.Split(doc, "\n")
	var sections []Section
	current := Section{Index: 0}
	var bodyLines []string
	inCodeFence := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track code fence state to avoid splitting on ## inside code blocks
		if strings.HasPrefix(trimmed, "```") {
			inCodeFence = !inCodeFence
		}

		isHeading := !inCodeFence && strings.HasPrefix(trimmed, "## ")
		if isHeading && (current.Heading != "" || len(bodyLines) > 0) {
			// Save previous section
			current.Body = strings.Join(bodyLines, "\n")
			sections = append(sections, current)
			// Start new section
			current = Section{Heading: trimmed, Index: len(sections)}
			bodyLines = nil
		} else {
			bodyLines = append(bodyLines, line)
		}
	}
	// Save last section
	current.Body = strings.Join(bodyLines, "\n")
	sections = append(sections, current)

	return sections
}

// MergeSections reassembles sections into a full document.
func MergeSections(sections []Section) string {
	var parts []string
	for _, s := range sections {
		if s.Heading != "" {
			parts = append(parts, s.Heading)
		}
		parts = append(parts, s.Body)
	}
	return strings.Join(parts, "\n")
}

// buildSectionSummaries creates a compact summary of each section (heading + first sentence).
func buildSectionSummaries(sections []Section) []string {
	summaries := make([]string, len(sections))
	for i, s := range sections {
		if s.Heading == "" {
			summaries[i] = "(preamble/front matter)"
			continue
		}
		// Extract first non-empty line as summary
		for _, line := range strings.Split(s.Body, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "```") && !strings.HasPrefix(trimmed, "|") && !strings.HasPrefix(trimmed, ">") {
				if len(trimmed) > 100 {
					trimmed = trimmed[:100] + "…"
				}
				summaries[i] = s.Heading + " — " + trimmed
				break
			}
		}
		if summaries[i] == "" {
			summaries[i] = s.Heading
		}
	}
	return summaries
}

// ShouldMultiPass returns true if the document is large enough to benefit from multi-pass polishing.
func ShouldMultiPass(docWordCount int) bool {
	maxTokens := ResolveMaxTokens("polish", docWordCount)
	estimatedInput := float64(docWordCount) * 1.3 // rough token estimate
	return estimatedInput > float64(maxTokens)*0.60
}

// MultiPassProgress is called after each section is polished.
type MultiPassProgress func(sectionIndex, totalSections int, heading string, changed bool)

// PolishMultiPass polishes a document section by section.
// Each section sees the context of other sections and the style of previously polished ones.
// Returns the full polished document. Progress callback is called after each section.
func PolishMultiPass(ctx context.Context, provider domain.AIProvider, doc string, meta domain.DocMeta,
	styleGuide string, personas []PersonaProfile, progress MultiPassProgress, audience string) (string, error) {

	sections := SplitSections(doc)

	// Section 0 is preamble (front matter) — never polish it
	if len(sections) <= 1 {
		// No ## headings found — fall back to single-pass
		return Polish(ctx, provider, doc, meta, styleGuide, "", personas, PolishOpts{Audience: audience})
	}

	summaries := buildSectionSummaries(sections)
	polished := make([]Section, len(sections))
	polished[0] = sections[0] // keep preamble as-is

	var prevPolishedStyle string // style reference from last polished section

	for i := 1; i < len(sections); i++ {
		// Early exit if context is cancelled (avoid N timeout errors)
		if ctx.Err() != nil {
			// Keep remaining sections as-is
			for j := i; j < len(sections); j++ {
				polished[j] = sections[j]
			}
			break
		}
		s := sections[i]

		// Build context: summaries of other sections
		var contextLines []string
		for j, sum := range summaries {
			if j == 0 || j == i {
				continue
			}
			contextLines = append(contextLines, fmt.Sprintf("  %s", sum))
		}

		// Build prompt for this section
		sectionContent := s.Heading + "\n" + s.Body
		sectionPrompt := buildSectionPrompt(sectionContent, contextLines, prevPolishedStyle, audience)

		// Use smaller max_tokens per section
		sectionWords := len(strings.Fields(sectionContent))
		maxTokens := sectionWords*3 + 512 // generous but bounded
		if maxTokens > 4096 {
			maxTokens = 4096
		}

		sysPrompt, _ := BuildPolishPrompt(doc, meta, styleGuide, "", personas, audience)

		result, err := provider.Complete(ctx, sectionPrompt,
			domain.WithSystem(sysPrompt),
			domain.WithMaxTokens(maxTokens))
		if err != nil {
			// On error, keep original section
			polished[i] = s
			if progress != nil {
				progress(i, len(sections)-1, s.Heading, false)
			}
			continue
		}

		// Only strip
		// code fencing from the AI result if the original section did
		// NOT already start with one. Otherwise a faithfully-echoed
		// ```python block inside a tutorial section would lose its
		// fence on the first multipass pass and render as prose on
		// the second pass.
		if !strings.HasPrefix(strings.TrimSpace(sectionContent), "```") {
			result = stripCodeFence(result)
		}
		result = PostProcess(sectionContent, result)

		// Parse result back into heading + body
		resultLines := strings.Split(result, "\n")
		if len(resultLines) > 0 && strings.HasPrefix(strings.TrimSpace(resultLines[0]), "## ") {
			polished[i] = Section{
				Heading: strings.TrimSpace(resultLines[0]),
				Body:    strings.Join(resultLines[1:], "\n"),
				Index:   i,
			}
		} else {
			polished[i] = Section{
				Heading: s.Heading, // keep original heading
				Body:    result,
				Index:   i,
			}
		}

		changed := polished[i].Body != s.Body
		if changed {
			// Update style reference for next section
			trimmed := polished[i].Body
			if len(trimmed) > 200 {
				trimmed = trimmed[:200]
			}
			prevPolishedStyle = trimmed
		}

		if progress != nil {
			progress(i, len(sections)-1, s.Heading, changed)
		}
	}

	return MergeSections(polished), nil
}

// buildSectionPrompt creates the user prompt for a single section polish.
//
// Every value interpolated between delimiters is run through
// sanitizePromptContent. Previously `section` and `prevStyle` were
// written raw, which let a doc (or the AI's own previous-section output
// via prevPolishedStyle) close the <<<SECTION>>>/<<<STYLE>>> blocks and
// inject arbitrary instructions into the next iteration of a multi-pass
// polish.
func buildSectionPrompt(section string, contextSummaries []string, prevStyle string, audience string) string {
	var sb strings.Builder

	if audience != "" {
		sb.WriteString("TARGET AUDIENCE: " + sanitizeShortField(audience) + "\n")
		sb.WriteString("Rewrite the section below for this audience.\n\n")
	}

	if len(contextSummaries) > 0 {
		sb.WriteString("OTHER SECTIONS IN THIS DOCUMENT (for context only, DO NOT polish these):\n")
		for _, c := range contextSummaries {
			sb.WriteString(sanitizePromptContent(c) + "\n")
		}
		sb.WriteString("\n")
	}

	if prevStyle != "" {
		sb.WriteString("STYLE REFERENCE (match this tone/style from a previously polished section):\n")
		sb.WriteString("<<<STYLE>>>\n")
		sb.WriteString(sanitizePromptContent(prevStyle))
		sb.WriteString("\n<<<END_STYLE>>>\n\n")
	}

	sb.WriteString("SECTION TO IMPROVE (return ONLY this section, complete with ## heading):\n")
	sb.WriteString("<<<SECTION>>>\n")
	sb.WriteString(sanitizePromptContent(section))
	sb.WriteString("\n<<<END_SECTION>>>\n\n")
	sb.WriteString("Return ONLY the improved section. No explanations, no wrapping.")

	return sb.String()
}

