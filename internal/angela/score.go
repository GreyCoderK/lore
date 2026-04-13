// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/greycoderk/lore/internal/domain"
)

// QualityScore holds the result of a document quality assessment.
type QualityScore struct {
	Total     int            // 0-100
	Breakdown map[string]int // category → points earned
	Missing   []string       // actionable items to improve score
	Grade     string         // A, B, C, D, F
	Profile   string         // "strict" | "free-form" — which scoring path produced this
}

// bannedPhrases are filler phrases that reduce quality.
var bannedPhrases = []string{
	"it is worth noting", "moreover", "furthermore", "in conclusion",
	"it should be noted", "as mentioned above", "this is important because",
	"improves the overall", "ensures reliability", "il convient de noter",
	"en conclusion", "comme mentionné", "il est important de",
}

// ScoreDocument evaluates a markdown document's quality on a 0-100 scale.
// Works on raw content (with or without front matter).
// Entirely local — no API calls.
//
// Two scoring profiles:
//   - Strict (decision/feature/bugfix/refactor): the original lore scoring
//     with heavy weight on ## Why, related refs, front-matter status — the
//     hallmarks of a commit-capture document.
//   - Free-form (notes, tutorials, guides, blog posts, concept pages, any
//     unknown type): the same signals minus the lore-specific ones, with
//     points redistributed so a well-written tutorial can legitimately
//     reach an A. Before this split, a perfect tutorial plateaued at F.
func ScoreDocument(content string, meta domain.DocMeta) QualityScore {
	if isFreeFormType(meta.Type) {
		return scoreFreeForm(content, meta)
	}
	return scoreStrict(content, meta)
}

// scoreStrict is the original lore-tuned scoring for decision/feature/bugfix/
// refactor documents. Max 100, distribution unchanged from the initial design.
func scoreStrict(content string, meta domain.DocMeta) QualityScore {
	s := QualityScore{Breakdown: make(map[string]int), Profile: "strict"}
	lower := strings.ToLower(content)
	lines := strings.Split(content, "\n")
	words := len(strings.Fields(content))

	// 1. Why/Pourquoi section (15 pts)
	// TODO(i18n): Missing[] strings are user-facing but not yet routed
	// through i18n.T(). Deferred per 8-3 scope.
	if hasSubstantialSection(content, "## Why", "## Pourquoi") {
		s.Breakdown["why"] = 15
	} else {
		s.Missing = append(s.Missing, "add ## Pourquoi / ## Why section (>100 chars)")
	}

	// 2. Mermaid diagram (15 pts)
	if strings.Contains(lower, "```mermaid") {
		s.Breakdown["diagram"] = 15
	} else {
		s.Missing = append(s.Missing, "add a mermaid diagram (architecture, sequence, or flowchart)")
	}

	// 3. Table (10 pts)
	if strings.Contains(content, "|---") || strings.Contains(content, "| ---") {
		s.Breakdown["table"] = 10
	} else {
		s.Missing = append(s.Missing, "add a comparison or reference table")
	}

	// 4. Code blocks with language tags (10 pts)
	fenced, naked := countCodeFences(lines)
	if fenced > 0 {
		s.Breakdown["code"] = 10
	} else if words > 300 {
		s.Missing = append(s.Missing, "add code/config examples with language tags")
	}

	// 5. No naked code fences (5 pts)
	if naked == 0 {
		s.Breakdown["code-tags"] = 5
	} else {
		s.Missing = append(s.Missing, fmt.Sprintf("%d code fence(s) missing language tag", naked))
	}

	// 6. Structured sections (10 pts) — at least 3 ## headings
	headingCount := countHeadings(lines)
	if headingCount >= 3 {
		s.Breakdown["structure"] = 10
	} else {
		s.Missing = append(s.Missing, fmt.Sprintf("add more sections (have %d, need ≥3)", headingCount))
	}

	// 7. Front matter complete (10 pts) — type + date + status
	fmPoints := 0
	if meta.Type != "" {
		fmPoints += 3
	}
	if meta.Date != "" {
		fmPoints += 3
	}
	if meta.Status != "" {
		fmPoints += 4
	}
	s.Breakdown["frontmatter"] = fmPoints
	if fmPoints < 10 {
		var missing []string
		if meta.Type == "" {
			missing = append(missing, "type")
		}
		if meta.Date == "" {
			missing = append(missing, "date")
		}
		if meta.Status == "" {
			missing = append(missing, "status")
		}
		s.Missing = append(s.Missing, fmt.Sprintf("front matter missing: %s", strings.Join(missing, ", ")))
	}

	// 8. Related references (5 pts)
	if len(meta.Related) > 0 {
		s.Breakdown["references"] = 5
	} else {
		s.Missing = append(s.Missing, "add related document references in front matter")
	}

	// 9. Density — 200-3000 words (10 pts)
	if words >= 200 && words <= 3000 {
		s.Breakdown["density"] = 10
	} else if words < 200 {
		s.Missing = append(s.Missing, fmt.Sprintf("document too short (%d words, aim for ≥200)", words))
	} else {
		s.Breakdown["density"] = 5 // still some points for long docs
		s.Missing = append(s.Missing, fmt.Sprintf("document very long (%d words) — consider splitting", words))
	}

	// 10. No TODO/FIXME (5 pts)
	if !strings.Contains(lower, "todo") && !strings.Contains(lower, "fixme") &&
		!strings.Contains(lower, "xxx") && !strings.Contains(lower, "hack") {
		s.Breakdown["clean"] = 5
	} else {
		s.Missing = append(s.Missing, "remove TODO/FIXME/HACK markers")
	}

	// 11. Style — no banned phrases (5 pts)
	hasBanned := false
	for _, phrase := range bannedPhrases {
		if strings.Contains(lower, phrase) {
			hasBanned = true
			break
		}
	}
	if !hasBanned {
		s.Breakdown["style"] = 5
	} else {
		s.Missing = append(s.Missing, "remove generic filler phrases")
	}

	finalizeScore(&s)
	return s
}

// scoreFreeForm rescales the scoring for narrative / external docs:
//   - drops Why section (15 pts) and Related refs (5 pts) — not applicable
//   - drops "status" sub-criterion in front matter (4 pts → type+date only, 6 pts)
//   - freed budget: 15 + 5 + 4 = 24 pts
//   - redistribution: diagram −5, code +5, structure +10, density +10,
//     paragraphs +4 = net +24 pts → total stays at 100.
//
// Historical note: the original redistribution
// spent +29 pts (paragraphs was +9 instead of +4), causing free-form docs
// to score up to 105/100. The paragraphs weight was rebalanced to 4 to
// close the overflow while keeping the signal alive. Structure and density
// remain the headline free-form signals at 20 pts each.
func scoreFreeForm(content string, meta domain.DocMeta) QualityScore {
	s := QualityScore{Breakdown: make(map[string]int), Profile: "free-form"}
	lower := strings.ToLower(content)
	lines := strings.Split(content, "\n")
	words := len(strings.Fields(content))

	// 1. Mermaid diagram (10 pts — reduced from 15)
	if strings.Contains(lower, "```mermaid") {
		s.Breakdown["diagram"] = 10
	}

	// 2. Table (10 pts)
	if strings.Contains(content, "|---") || strings.Contains(content, "| ---") {
		s.Breakdown["table"] = 10
	}

	// 3. Code blocks with language tags (15 pts — up from 10)
	fenced, naked := countCodeFences(lines)
	if fenced > 0 {
		s.Breakdown["code"] = 15
	} else if words > 300 {
		s.Missing = append(s.Missing, "add code/config examples with language tags")
	}

	// 4. No naked code fences (5 pts)
	if naked == 0 {
		s.Breakdown["code-tags"] = 5
	} else {
		s.Missing = append(s.Missing, fmt.Sprintf("%d code fence(s) missing language tag", naked))
	}

	// 5. Structured sections (20 pts — up from 10)
	//    Tutorials and guides live or die on heading structure.
	headingCount := countHeadings(lines)
	switch {
	case headingCount >= 5:
		s.Breakdown["structure"] = 20
	case headingCount >= 3:
		s.Breakdown["structure"] = 15
	case headingCount >= 1:
		s.Breakdown["structure"] = 8
	default:
		s.Missing = append(s.Missing, fmt.Sprintf("add section headings (have %d)", headingCount))
	}

	// 6. Front matter (6 pts — only type + date, status is lore-specific)
	fmPoints := 0
	if meta.Type != "" {
		fmPoints += 3
	}
	if meta.Date != "" {
		fmPoints += 3
	}
	s.Breakdown["frontmatter"] = fmPoints

	// 7. Density — 150-5000 words (20 pts — up from 10, wider range)
	//    External docs can legitimately be shorter (landing pages) or longer
	//    (comprehensive guides) than lore's 200-3000 sweet spot.
	switch {
	case words >= 150 && words <= 5000:
		s.Breakdown["density"] = 20
	case words >= 80 && words < 150:
		s.Breakdown["density"] = 12 // short but acceptable
	case words > 5000:
		s.Breakdown["density"] = 15 // long docs still get most points
		s.Missing = append(s.Missing, fmt.Sprintf("document very long (%d words) — consider splitting", words))
	default:
		s.Missing = append(s.Missing, fmt.Sprintf("document too short (%d words, aim for ≥80)", words))
	}

	// 8. Good paragraph/line ratio (4 pts)
	//    Well-structured prose has short paragraphs. A single giant blob is
	//    a smell. Originally 9 pts, reduced to 4 to keep the total at 100.
	if hasReasonableParagraphs(content) {
		s.Breakdown["paragraphs"] = 4
	}

	// 9. No TODO/FIXME (5 pts)
	if !strings.Contains(lower, "todo") && !strings.Contains(lower, "fixme") &&
		!strings.Contains(lower, "xxx") && !strings.Contains(lower, "hack") {
		s.Breakdown["clean"] = 5
	} else {
		s.Missing = append(s.Missing, "remove TODO/FIXME/HACK markers")
	}

	// 10. Style — no banned phrases (5 pts)
	hasBanned := false
	for _, phrase := range bannedPhrases {
		if strings.Contains(lower, phrase) {
			hasBanned = true
			break
		}
	}
	if !hasBanned {
		s.Breakdown["style"] = 5
	} else {
		s.Missing = append(s.Missing, "remove generic filler phrases")
	}

	finalizeScore(&s)
	return s
}

// finalizeScore sums the breakdown and assigns a letter grade. Shared between
// scoreStrict and scoreFreeForm so the grade thresholds stay consistent.
func finalizeScore(s *QualityScore) {
	s.Total = 0
	for _, pts := range s.Breakdown {
		s.Total += pts
	}
	switch {
	case s.Total >= 85:
		s.Grade = "A"
	case s.Total >= 70:
		s.Grade = "B"
	case s.Total >= 50:
		s.Grade = "C"
	case s.Total >= 30:
		s.Grade = "D"
	default:
		s.Grade = "F"
	}
}

// hasReasonableParagraphs reports whether the doc has multiple paragraphs
// separated by blank lines (vs. one giant blob). Used by free-form scoring.
func hasReasonableParagraphs(content string) bool {
	// Split on blank lines to count paragraph-like blocks outside code fences.
	inFence := false
	blocks := 0
	current := 0
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if trimmed == "" {
			if current > 0 {
				blocks++
				current = 0
			}
			continue
		}
		// Ignore heading lines as "prose paragraphs"
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		current++
	}
	if current > 0 {
		blocks++
	}
	return blocks >= 2
}

// FormatScore returns a compact one-line summary: "72/100 (B)"
func FormatScore(s QualityScore) string {
	return fmt.Sprintf("%d/100 (%s)", s.Total, s.Grade)
}

// scoreCategoryOrder describes how a single category is rendered in the
// breakdown: its map key, human label, and maximum for the profile.
type scoreCategoryOrder struct {
	key   string
	label string
	max   int
}

// strictCategoryOrder is the breakdown layout for scoreStrict documents.
// The sum of max values is 100 (enforced by TestScoringInvariant_StrictMaxesSum100).
var strictCategoryOrder = []scoreCategoryOrder{
	{"why", "Pourquoi/Why", 15},
	{"diagram", "Diagram", 15},
	{"table", "Table", 10},
	{"code", "Code blocks", 10},
	{"code-tags", "Code tags", 5},
	{"structure", "Sections", 10},
	{"frontmatter", "Front matter", 10},
	{"references", "References", 5},
	{"density", "Density", 10},
	{"clean", "Clean", 5},
	{"style", "Style", 5},
}

// freeFormCategoryOrder is the breakdown layout for scoreFreeForm documents.
// The sum of max values is 100 (enforced by TestScoringInvariant_FreeFormMaxesSum100).
// Note: no "why" or "references" rows — these concepts don't apply to
// free-form content. "paragraphs" is free-form specific and replaces them.
var freeFormCategoryOrder = []scoreCategoryOrder{
	{"diagram", "Diagram", 10},
	{"table", "Table", 10},
	{"code", "Code blocks", 15},
	{"code-tags", "Code tags", 5},
	{"structure", "Sections", 20},
	{"frontmatter", "Front matter", 6},
	{"density", "Density", 20},
	{"paragraphs", "Paragraphs", 4},
	{"clean", "Clean", 5},
	{"style", "Style", 5},
}

// FormatScoreDetail is retained for test compatibility and future CLI use.
//
// FormatScoreDetail returns a multi-line breakdown whose per-category max
// values reflect the scoring profile that produced the score. A "strict"
// profile shows the lore-native categories (Why, References, ...); a
// "free-form" profile shows the narrative layout (Paragraphs, higher
// Structure and Density caps, no Why row).
func FormatScoreDetail(s QualityScore) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Quality: %d/100 (%s)\n", s.Total, s.Grade)

	order := strictCategoryOrder
	if s.Profile == "free-form" {
		order = freeFormCategoryOrder
	}

	for _, item := range order {
		pts := s.Breakdown[item.key]
		if pts >= item.max {
			fmt.Fprintf(&sb, "  ✓ %s (%d/%d)\n", item.label, pts, item.max)
		} else if pts > 0 {
			fmt.Fprintf(&sb, "  ~ %s (%d/%d)\n", item.label, pts, item.max)
		} else {
			fmt.Fprintf(&sb, "  ✗ %s (0/%d)\n", item.label, item.max)
		}
	}

	return sb.String()
}

func hasSubstantialSection(content string, headings ...string) bool {
	lower := strings.ToLower(content)
	for _, h := range headings {
		idx := strings.Index(lower, strings.ToLower(h))
		if idx < 0 {
			continue
		}
		// Find content after heading until next ## or EOF.
		// Use idx on `lower` (not `content`) since we only need to
		// check substance (word count), not preserve case.
		rest := lower[idx+len(h):]
		nextH := strings.Index(rest, "\n## ")
		// nextH == 0 means the next heading is immediately
		// after, which is still a valid slice bound.
		if nextH >= 0 {
			rest = rest[:nextH]
		}
		if utf8.RuneCountInString(strings.TrimSpace(rest)) > 100 {
			return true
		}
	}
	return false
}

func countCodeFences(lines []string) (fenced, naked int) {
	inFence := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "```") {
			continue
		}
		if !inFence {
			// Opening fence
			if trimmed == "```" {
				naked++ // bare open — missing language tag
			} else {
				fenced++ // tagged open (```java, ```mermaid, etc.)
			}
			inFence = true
		} else {
			// Closing fence — don't count, just toggle state
			inFence = false
		}
	}
	return fenced, naked
}

func countHeadings(lines []string) int {
	count := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "### ") {
			count++
		}
	}
	return count
}
