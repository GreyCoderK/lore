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
	Total     int               // 0-100
	Breakdown map[string]int    // category → points earned
	Missing   []string          // actionable items to improve score
	Grade     string            // A, B, C, D, F
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
func ScoreDocument(content string, meta domain.DocMeta) QualityScore {
	s := QualityScore{Breakdown: make(map[string]int)}
	lower := strings.ToLower(content)
	lines := strings.Split(content, "\n")
	words := len(strings.Fields(content))

	// 1. Why/Pourquoi section (15 pts)
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

	// Sum up
	for _, pts := range s.Breakdown {
		s.Total += pts
	}

	// Grade
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

	return s
}

// FormatScore returns a compact one-line summary: "72/100 (B)"
func FormatScore(s QualityScore) string {
	return fmt.Sprintf("%d/100 (%s)", s.Total, s.Grade)
}

// FormatScoreDetail returns a multi-line breakdown.
func FormatScoreDetail(s QualityScore) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Quality: %d/100 (%s)\n", s.Total, s.Grade)

	order := []struct{ key, label string; max int }{
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
		// Find content after heading until next ## or EOF
		rest := content[idx+len(h):]
		nextH := strings.Index(rest, "\n## ")
		if nextH > 0 {
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
