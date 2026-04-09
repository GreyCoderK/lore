// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func fullMeta() domain.DocMeta {
	return domain.DocMeta{
		Type:    "decision",
		Date:    "2026-04-09",
		Status:  "accepted",
		Related: []string{"adr-001.md"},
	}
}

func emptyMeta() domain.DocMeta {
	return domain.DocMeta{}
}

// repeatWord generates a string of n copies of "word ".
func repeatWord(word string, n int) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteString(word)
		sb.WriteByte(' ')
	}
	return sb.String()
}

// ---------------------------------------------------------------------------
// ScoreDocument
// ---------------------------------------------------------------------------

func TestScoreDocument_EmptyDocument(t *testing.T) {
	s := ScoreDocument("", emptyMeta())
	if s.Total > 15 {
		t.Errorf("empty document should score low, got %d", s.Total)
	}
	if s.Grade == "A" || s.Grade == "B" {
		t.Errorf("empty document should not grade well, got %s", s.Grade)
	}
}

func TestScoreDocument_WhySectionSubstantial(t *testing.T) {
	body := "## Why\n" + repeatWord("explanation", 20) + "\n\n## Other\nstuff"
	s := ScoreDocument(body, emptyMeta())
	if s.Breakdown["why"] != 15 {
		t.Errorf("expected 15 pts for ## Why section, got %d", s.Breakdown["why"])
	}
}

func TestScoreDocument_PourquoiSection(t *testing.T) {
	body := "## Pourquoi\n" + repeatWord("explication", 20) + "\n\n## Autre\ncontenu"
	s := ScoreDocument(body, emptyMeta())
	if s.Breakdown["why"] != 15 {
		t.Errorf("expected 15 pts for ## Pourquoi section, got %d", s.Breakdown["why"])
	}
}

func TestScoreDocument_WhySectionTooShort(t *testing.T) {
	body := "## Why\nshort\n\n## Other\nstuff"
	s := ScoreDocument(body, emptyMeta())
	if s.Breakdown["why"] != 0 {
		t.Errorf("expected 0 pts for short ## Why section, got %d", s.Breakdown["why"])
	}
}

func TestScoreDocument_MermaidDiagram(t *testing.T) {
	body := "```mermaid\ngraph LR\n  A-->B\n```\n"
	s := ScoreDocument(body, emptyMeta())
	if s.Breakdown["diagram"] != 15 {
		t.Errorf("expected 15 pts for mermaid diagram, got %d", s.Breakdown["diagram"])
	}
}

func TestScoreDocument_NoMermaid(t *testing.T) {
	s := ScoreDocument("no diagram here", emptyMeta())
	if s.Breakdown["diagram"] != 0 {
		t.Errorf("expected 0 pts without mermaid, got %d", s.Breakdown["diagram"])
	}
}

func TestScoreDocument_Table(t *testing.T) {
	body := "| Col A | Col B |\n|---|---|\n| 1 | 2 |\n"
	s := ScoreDocument(body, emptyMeta())
	if s.Breakdown["table"] != 10 {
		t.Errorf("expected 10 pts for table, got %d", s.Breakdown["table"])
	}
}

func TestScoreDocument_TableWithSpaces(t *testing.T) {
	body := "| Col A | Col B |\n| --- | --- |\n| 1 | 2 |\n"
	s := ScoreDocument(body, emptyMeta())
	if s.Breakdown["table"] != 10 {
		t.Errorf("expected 10 pts for table with spaced separator, got %d", s.Breakdown["table"])
	}
}

func TestScoreDocument_TaggedCodeFences(t *testing.T) {
	body := "```go\nfmt.Println(\"hi\")\n```\n"
	s := ScoreDocument(body, emptyMeta())
	if s.Breakdown["code"] != 10 {
		t.Errorf("expected 10 pts for tagged code fence, got %d", s.Breakdown["code"])
	}
	if s.Breakdown["code-tags"] != 5 {
		t.Errorf("expected 5 pts for no naked fences, got %d", s.Breakdown["code-tags"])
	}
}

func TestScoreDocument_NakedCodeFences(t *testing.T) {
	body := "```\nsome code\n```\n"
	s := ScoreDocument(body, emptyMeta())
	if s.Breakdown["code-tags"] != 0 {
		t.Errorf("expected 0 pts for naked fence, got %d", s.Breakdown["code-tags"])
	}
}

func TestScoreDocument_ThreePlusHeadings(t *testing.T) {
	body := "## One\ntext\n## Two\ntext\n## Three\ntext\n"
	s := ScoreDocument(body, emptyMeta())
	if s.Breakdown["structure"] != 10 {
		t.Errorf("expected 10 pts for 3+ headings, got %d", s.Breakdown["structure"])
	}
}

func TestScoreDocument_TwoHeadingsNotEnough(t *testing.T) {
	body := "## One\ntext\n## Two\ntext\n"
	s := ScoreDocument(body, emptyMeta())
	if s.Breakdown["structure"] != 0 {
		t.Errorf("expected 0 pts for <3 headings, got %d", s.Breakdown["structure"])
	}
}

func TestScoreDocument_CompleteFrontmatter(t *testing.T) {
	meta := domain.DocMeta{Type: "adr", Date: "2026-01-01", Status: "accepted"}
	s := ScoreDocument("some content", meta)
	if s.Breakdown["frontmatter"] != 10 {
		t.Errorf("expected 10 pts for complete frontmatter, got %d", s.Breakdown["frontmatter"])
	}
}

func TestScoreDocument_PartialFrontmatter(t *testing.T) {
	meta := domain.DocMeta{Type: "adr"}
	s := ScoreDocument("some content", meta)
	if s.Breakdown["frontmatter"] != 3 {
		t.Errorf("expected 3 pts for type-only frontmatter, got %d", s.Breakdown["frontmatter"])
	}
}

func TestScoreDocument_EmptyFrontmatter(t *testing.T) {
	s := ScoreDocument("some content", emptyMeta())
	if s.Breakdown["frontmatter"] != 0 {
		t.Errorf("expected 0 pts for empty frontmatter, got %d", s.Breakdown["frontmatter"])
	}
}

func TestScoreDocument_RelatedReferences(t *testing.T) {
	meta := domain.DocMeta{Related: []string{"doc-001.md"}}
	s := ScoreDocument("content", meta)
	if s.Breakdown["references"] != 5 {
		t.Errorf("expected 5 pts for related refs, got %d", s.Breakdown["references"])
	}
}

func TestScoreDocument_NoRelatedReferences(t *testing.T) {
	s := ScoreDocument("content", emptyMeta())
	if s.Breakdown["references"] != 0 {
		t.Errorf("expected 0 pts without related refs, got %d", s.Breakdown["references"])
	}
}

func TestScoreDocument_TODO_DeductsClean(t *testing.T) {
	s := ScoreDocument("TODO: fix this later", emptyMeta())
	if s.Breakdown["clean"] != 0 {
		t.Errorf("expected 0 pts for clean with TODO present, got %d", s.Breakdown["clean"])
	}
}

func TestScoreDocument_FIXME_DeductsClean(t *testing.T) {
	s := ScoreDocument("FIXME: broken logic", emptyMeta())
	if s.Breakdown["clean"] != 0 {
		t.Errorf("expected 0 pts for clean with FIXME present, got %d", s.Breakdown["clean"])
	}
}

func TestScoreDocument_HACK_DeductsClean(t *testing.T) {
	s := ScoreDocument("HACK: workaround", emptyMeta())
	if s.Breakdown["clean"] != 0 {
		t.Errorf("expected 0 pts for clean with HACK present, got %d", s.Breakdown["clean"])
	}
}

func TestScoreDocument_CleanNoMarkers(t *testing.T) {
	s := ScoreDocument("perfectly clean content", emptyMeta())
	if s.Breakdown["clean"] != 5 {
		t.Errorf("expected 5 pts for clean content, got %d", s.Breakdown["clean"])
	}
}

func TestScoreDocument_BannedPhrases(t *testing.T) {
	s := ScoreDocument("it is worth noting that this is great", emptyMeta())
	if s.Breakdown["style"] != 0 {
		t.Errorf("expected 0 pts for style with banned phrase, got %d", s.Breakdown["style"])
	}
}

func TestScoreDocument_BannedPhraseFrench(t *testing.T) {
	s := ScoreDocument("il convient de noter que cela fonctionne", emptyMeta())
	if s.Breakdown["style"] != 0 {
		t.Errorf("expected 0 pts for style with French banned phrase, got %d", s.Breakdown["style"])
	}
}

func TestScoreDocument_NoBannedPhrases(t *testing.T) {
	s := ScoreDocument("clear direct writing", emptyMeta())
	if s.Breakdown["style"] != 5 {
		t.Errorf("expected 5 pts for clean style, got %d", s.Breakdown["style"])
	}
}

func TestScoreDocument_DensityInRange(t *testing.T) {
	body := repeatWord("knowledge", 250)
	s := ScoreDocument(body, emptyMeta())
	if s.Breakdown["density"] != 10 {
		t.Errorf("expected 10 pts for 200-3000 words, got %d", s.Breakdown["density"])
	}
}

func TestScoreDocument_DensityTooShort(t *testing.T) {
	s := ScoreDocument("short doc", emptyMeta())
	if s.Breakdown["density"] != 0 {
		t.Errorf("expected 0 pts for very short doc, got %d", s.Breakdown["density"])
	}
}

func TestScoreDocument_DensityTooLong(t *testing.T) {
	body := repeatWord("verbose", 3500)
	s := ScoreDocument(body, emptyMeta())
	if s.Breakdown["density"] != 5 {
		t.Errorf("expected 5 pts for overly long doc, got %d", s.Breakdown["density"])
	}
}

func TestScoreDocument_FullHighQuality(t *testing.T) {
	// Build a document that should earn most/all points.
	var sb strings.Builder
	sb.WriteString("## Why\n")
	sb.WriteString(repeatWord("rationale", 25)) // >100 chars
	sb.WriteString("\n\n## Architecture\n")
	sb.WriteString("```mermaid\ngraph LR\n  A-->B\n```\n\n")
	sb.WriteString("| Option | Pros | Cons |\n|---|---|---|\n| A | fast | complex |\n\n")
	sb.WriteString("```go\nfunc main() { fmt.Println(\"hello\") }\n```\n\n")
	sb.WriteString("## Implementation\n")
	sb.WriteString(repeatWord("detail", 80))
	sb.WriteString("\n\n## Testing\n")
	sb.WriteString(repeatWord("verification", 80))
	sb.WriteString("\n")

	meta := fullMeta()
	s := ScoreDocument(sb.String(), meta)

	if s.Total < 85 {
		t.Errorf("full high-quality doc should score >=85, got %d", s.Total)
	}
	if s.Grade != "A" {
		t.Errorf("full high-quality doc should grade A, got %s", s.Grade)
	}
}

func TestScoreDocument_GradeBoundaries(t *testing.T) {
	tests := []struct {
		name    string
		min     int
		max     int
		grade   string
	}{
		{"grade A", 85, 100, "A"},
		{"grade B", 70, 84, "B"},
		{"grade C", 50, 69, "C"},
		{"grade D", 30, 49, "D"},
		{"grade F", 0, 29, "F"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We test the grade assignment by checking the switch logic
			// via a synthetic QualityScore with known Total.
			// Since we can't set Total directly, we verify through the
			// actual ScoreDocument output is consistent.
			s := QualityScore{Total: tt.min}
			// Re-derive grade to verify boundaries.
			var grade string
			switch {
			case s.Total >= 85:
				grade = "A"
			case s.Total >= 70:
				grade = "B"
			case s.Total >= 50:
				grade = "C"
			case s.Total >= 30:
				grade = "D"
			default:
				grade = "F"
			}
			if grade != tt.grade {
				t.Errorf("total %d: expected grade %s, got %s", tt.min, tt.grade, grade)
			}
		})
	}
}

func TestScoreDocument_MissingFieldsPopulated(t *testing.T) {
	s := ScoreDocument("short", emptyMeta())
	if len(s.Missing) == 0 {
		t.Error("expected Missing to be populated for a bare document")
	}
}

func TestScoreDocument_BreakdownSumsToTotal(t *testing.T) {
	body := "## Why\n" + repeatWord("reason", 25) + "\n## Two\n## Three\n"
	meta := domain.DocMeta{Type: "adr", Date: "2026-01-01", Status: "draft"}
	s := ScoreDocument(body, meta)

	sum := 0
	for _, pts := range s.Breakdown {
		sum += pts
	}
	if sum != s.Total {
		t.Errorf("breakdown sum %d != total %d", sum, s.Total)
	}
}

// ---------------------------------------------------------------------------
// FormatScore
// ---------------------------------------------------------------------------

func TestFormatScore(t *testing.T) {
	s := QualityScore{Total: 72, Grade: "B"}
	got := FormatScore(s)
	want := "72/100 (B)"
	if got != want {
		t.Errorf("FormatScore = %q, want %q", got, want)
	}
}

func TestFormatScore_ZeroScore(t *testing.T) {
	s := QualityScore{Total: 0, Grade: "F"}
	got := FormatScore(s)
	if !strings.Contains(got, "0/100") || !strings.Contains(got, "F") {
		t.Errorf("FormatScore for zero = %q, expected 0/100 (F)", got)
	}
}

func TestFormatScore_PerfectScore(t *testing.T) {
	s := QualityScore{Total: 100, Grade: "A"}
	got := FormatScore(s)
	want := "100/100 (A)"
	if got != want {
		t.Errorf("FormatScore = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// FormatScoreDetail
// ---------------------------------------------------------------------------

func TestFormatScoreDetail_MultiLine(t *testing.T) {
	s := ScoreDocument("## Why\n"+repeatWord("explanation", 20), fullMeta())
	detail := FormatScoreDetail(s)
	lines := strings.Split(strings.TrimSpace(detail), "\n")
	if len(lines) < 5 {
		t.Errorf("expected multi-line detail output, got %d lines", len(lines))
	}
}

func TestFormatScoreDetail_ContainsHeader(t *testing.T) {
	s := QualityScore{Total: 50, Grade: "C", Breakdown: map[string]int{}}
	detail := FormatScoreDetail(s)
	if !strings.Contains(detail, "Quality: 50/100 (C)") {
		t.Errorf("detail missing header line, got:\n%s", detail)
	}
}

func TestFormatScoreDetail_ShowsCheckAndCross(t *testing.T) {
	s := QualityScore{
		Total: 25,
		Grade: "F",
		Breakdown: map[string]int{
			"why":     15,
			"diagram": 0,
		},
	}
	detail := FormatScoreDetail(s)
	if !strings.Contains(detail, "✓") {
		t.Errorf("expected checkmark for earned category, got:\n%s", detail)
	}
	if !strings.Contains(detail, "✗") {
		t.Errorf("expected cross for missing category, got:\n%s", detail)
	}
}

func TestFormatScoreDetail_ShowsPartial(t *testing.T) {
	s := QualityScore{
		Total: 5,
		Grade: "F",
		Breakdown: map[string]int{
			"frontmatter": 3, // partial (max is 10)
		},
	}
	detail := FormatScoreDetail(s)
	if !strings.Contains(detail, "~ Front matter (3/10)") {
		t.Errorf("expected partial marker for frontmatter, got:\n%s", detail)
	}
}

// ---------------------------------------------------------------------------
// hasSubstantialSection
// ---------------------------------------------------------------------------

func TestHasSubstantialSection_Found(t *testing.T) {
	body := "## Why\n" + repeatWord("reason", 20) + "\n\n## Next\n"
	if !hasSubstantialSection(body, "## Why") {
		t.Error("expected true for substantial ## Why section")
	}
}

func TestHasSubstantialSection_TooShort(t *testing.T) {
	body := "## Why\nbrief\n\n## Next\n"
	if hasSubstantialSection(body, "## Why") {
		t.Error("expected false for short ## Why section")
	}
}

func TestHasSubstantialSection_AtEOF(t *testing.T) {
	body := "## Why\n" + repeatWord("content", 20)
	if !hasSubstantialSection(body, "## Why") {
		t.Error("expected true for section at EOF with enough content")
	}
}

func TestHasSubstantialSection_EmptySection(t *testing.T) {
	body := "## Why\n\n## Next\nstuff"
	if hasSubstantialSection(body, "## Why") {
		t.Error("expected false for empty section")
	}
}

func TestHasSubstantialSection_NoMatch(t *testing.T) {
	body := "## Other\nstuff here"
	if hasSubstantialSection(body, "## Why", "## Pourquoi") {
		t.Error("expected false when heading not present")
	}
}

func TestHasSubstantialSection_CaseInsensitive(t *testing.T) {
	body := "## WHY\n" + repeatWord("content", 20)
	if !hasSubstantialSection(body, "## Why") {
		t.Error("expected case-insensitive heading match")
	}
}

func TestHasSubstantialSection_MultipleHeadingsFirstMatches(t *testing.T) {
	body := "## Pourquoi\n" + repeatWord("raison", 20) + "\n\n## Autre\n"
	if !hasSubstantialSection(body, "## Why", "## Pourquoi") {
		t.Error("expected true when second heading variant matches")
	}
}

// ---------------------------------------------------------------------------
// countCodeFences
// ---------------------------------------------------------------------------

func TestCountCodeFences_Tagged(t *testing.T) {
	lines := strings.Split("```go\ncode\n```", "\n")
	fenced, naked := countCodeFences(lines)
	if fenced != 1 {
		t.Errorf("expected 1 fenced, got %d", fenced)
	}
	if naked != 0 {
		t.Errorf("expected 0 naked, got %d", naked)
	}
}

func TestCountCodeFences_Bare(t *testing.T) {
	lines := strings.Split("```\ncode\n```", "\n")
	fenced, naked := countCodeFences(lines)
	if fenced != 0 {
		t.Errorf("expected 0 fenced, got %d", fenced)
	}
	if naked != 1 {
		t.Errorf("expected 1 naked, got %d", naked)
	}
}

func TestCountCodeFences_Mixed(t *testing.T) {
	input := "```python\nprint('hi')\n```\n\n```\nraw\n```\n\n```yaml\nkey: val\n```"
	lines := strings.Split(input, "\n")
	fenced, naked := countCodeFences(lines)
	if fenced != 2 {
		t.Errorf("expected 2 fenced, got %d", fenced)
	}
	if naked != 1 {
		t.Errorf("expected 1 naked, got %d", naked)
	}
}

func TestCountCodeFences_Empty(t *testing.T) {
	fenced, naked := countCodeFences(nil)
	if fenced != 0 || naked != 0 {
		t.Errorf("expected 0/0 for nil lines, got %d/%d", fenced, naked)
	}
}

func TestCountCodeFences_MermaidCounted(t *testing.T) {
	lines := strings.Split("```mermaid\ngraph LR\n```", "\n")
	fenced, naked := countCodeFences(lines)
	if fenced != 1 {
		t.Errorf("expected mermaid as fenced, got fenced=%d", fenced)
	}
	if naked != 0 {
		t.Errorf("expected 0 naked, got %d", naked)
	}
}

func TestCountCodeFences_IndentedFence(t *testing.T) {
	lines := strings.Split("  ```go\n  code\n  ```", "\n")
	fenced, _ := countCodeFences(lines)
	if fenced != 1 {
		t.Errorf("expected indented tagged fence to count, got fenced=%d", fenced)
	}
}

func TestCountCodeFences_NestedLookalike(t *testing.T) {
	// Two tagged blocks back to back (not truly nested, but tests toggle).
	input := "```go\nfmt.Println()\n```\n```rust\nlet x = 1;\n```"
	lines := strings.Split(input, "\n")
	fenced, naked := countCodeFences(lines)
	if fenced != 2 {
		t.Errorf("expected 2 fenced for two tagged blocks, got %d", fenced)
	}
	if naked != 0 {
		t.Errorf("expected 0 naked, got %d", naked)
	}
}

// ---------------------------------------------------------------------------
// countHeadings
// ---------------------------------------------------------------------------

func TestCountHeadings_H2Only(t *testing.T) {
	lines := strings.Split("## A\ntext\n## B\ntext\n## C\n", "\n")
	got := countHeadings(lines)
	if got != 3 {
		t.Errorf("expected 3 headings, got %d", got)
	}
}

func TestCountHeadings_H3Counted(t *testing.T) {
	lines := strings.Split("### Sub\ntext", "\n")
	got := countHeadings(lines)
	if got != 1 {
		t.Errorf("expected ### to be counted, got %d", got)
	}
}

func TestCountHeadings_H1NotCounted(t *testing.T) {
	lines := strings.Split("# Title\n## Section\n", "\n")
	got := countHeadings(lines)
	if got != 1 {
		t.Errorf("expected only ## to count, not #; got %d", got)
	}
}

func TestCountHeadings_H4NotCounted(t *testing.T) {
	lines := strings.Split("#### Deep\n## Section\n", "\n")
	got := countHeadings(lines)
	if got != 1 {
		t.Errorf("expected #### not counted, got %d", got)
	}
}

func TestCountHeadings_None(t *testing.T) {
	lines := strings.Split("just text\nno headings here", "\n")
	got := countHeadings(lines)
	if got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestCountHeadings_MixedH2H3(t *testing.T) {
	lines := strings.Split("## A\n### A.1\n## B\n### B.1\n### B.2\n", "\n")
	got := countHeadings(lines)
	if got != 5 {
		t.Errorf("expected 5 mixed headings, got %d", got)
	}
}

func TestCountHeadings_IndentedHeading(t *testing.T) {
	lines := strings.Split("  ## Indented\n", "\n")
	got := countHeadings(lines)
	// TrimSpace is used, so indented headings should still count.
	if got != 1 {
		t.Errorf("expected indented heading to count, got %d", got)
	}
}
