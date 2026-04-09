// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// SplitSections
// ---------------------------------------------------------------------------

func TestSplitSections_EmptyDoc(t *testing.T) {
	sections := SplitSections("")
	if len(sections) != 1 {
		t.Fatalf("expected 1 section (preamble), got %d", len(sections))
	}
	if sections[0].Heading != "" {
		t.Errorf("preamble heading should be empty, got %q", sections[0].Heading)
	}
}

func TestSplitSections_ThreeHeadings(t *testing.T) {
	doc := `# Title

Some preamble text.

## Section One

Body one.

## Section Two

Body two.

## Section Three

Body three.
`
	sections := SplitSections(doc)
	if len(sections) != 4 {
		t.Fatalf("expected 4 sections (preamble + 3), got %d", len(sections))
	}
	if sections[0].Heading != "" {
		t.Errorf("section 0 should be preamble, got heading %q", sections[0].Heading)
	}
	if sections[1].Heading != "## Section One" {
		t.Errorf("section 1 heading = %q, want %q", sections[1].Heading, "## Section One")
	}
	if sections[2].Heading != "## Section Two" {
		t.Errorf("section 2 heading = %q, want %q", sections[2].Heading, "## Section Two")
	}
	if sections[3].Heading != "## Section Three" {
		t.Errorf("section 3 heading = %q, want %q", sections[3].Heading, "## Section Three")
	}
}

func TestSplitSections_HeadingInsideCodeFence(t *testing.T) {
	doc := "Preamble\n\n## Real Heading\n\nSome text.\n\n```markdown\n## Not A Heading\nstuff\n```\n\nMore text."
	sections := SplitSections(doc)
	// Should have 2 sections: preamble + "Real Heading".
	// The ## inside the code fence must NOT create a third section.
	if len(sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sections))
	}
	if sections[1].Heading != "## Real Heading" {
		t.Errorf("section 1 heading = %q, want %q", sections[1].Heading, "## Real Heading")
	}
	if !strings.Contains(sections[1].Body, "## Not A Heading") {
		t.Error("code-fenced ## should appear in body, not split into a new section")
	}
}

func TestSplitSections_BodyContentPreserved(t *testing.T) {
	doc := "Preamble\n\n## Intro\n\nFirst paragraph.\n\nSecond paragraph with **bold**."
	sections := SplitSections(doc)
	if len(sections) < 2 {
		t.Fatalf("expected at least 2 sections, got %d", len(sections))
	}
	// Section 0 is preamble, section 1 has heading "## Intro".
	sec := sections[1]
	if sec.Heading != "## Intro" {
		t.Errorf("section 1 heading = %q, want %q", sec.Heading, "## Intro")
	}
	if !strings.Contains(sec.Body, "First paragraph.") {
		t.Error("body missing 'First paragraph.'")
	}
	if !strings.Contains(sec.Body, "Second paragraph with **bold**.") {
		t.Error("body missing 'Second paragraph with **bold**.'")
	}
}

// ---------------------------------------------------------------------------
// MergeSections — round-trip
// ---------------------------------------------------------------------------

func TestMergeSections_RoundTrip(t *testing.T) {
	docs := []string{
		"",
		"Just preamble, no headings.",
		"## Only heading\n\nBody.",
		"Preamble\n\n## A\n\nBody A\n\n## B\n\nBody B\n\n## C\n\nBody C",
	}
	for _, doc := range docs {
		sections := SplitSections(doc)
		merged := MergeSections(sections)
		if merged != doc {
			t.Errorf("round-trip failed.\nOriginal: %q\nMerged:   %q", doc, merged)
		}
	}
}

// ---------------------------------------------------------------------------
// ShouldMultiPass
// ---------------------------------------------------------------------------

func TestShouldMultiPass_SmallDoc(t *testing.T) {
	if ShouldMultiPass(100) {
		t.Error("100-word doc should NOT trigger multi-pass")
	}
}

func TestShouldMultiPass_LargeDoc(t *testing.T) {
	if !ShouldMultiPass(5000) {
		t.Error("5000-word doc should trigger multi-pass")
	}
}

// ---------------------------------------------------------------------------
// buildSectionSummaries
// ---------------------------------------------------------------------------

func TestBuildSectionSummaries(t *testing.T) {
	sections := []Section{
		{Heading: "", Body: "---\ntitle: Test\n---", Index: 0},
		{Heading: "## Introduction", Body: "\nThis is the first sentence. More text follows.", Index: 1},
		{Heading: "## Empty Body", Body: "", Index: 2},
	}
	summaries := buildSectionSummaries(sections)
	if len(summaries) != 3 {
		t.Fatalf("expected 3 summaries, got %d", len(summaries))
	}
	if summaries[0] != "(preamble/front matter)" {
		t.Errorf("summary[0] = %q, want preamble marker", summaries[0])
	}
	if !strings.Contains(summaries[1], "## Introduction") {
		t.Errorf("summary[1] should contain heading, got %q", summaries[1])
	}
	if !strings.Contains(summaries[1], "This is the first sentence.") {
		t.Errorf("summary[1] should contain first sentence, got %q", summaries[1])
	}
	// Empty body section should fall back to heading only.
	if summaries[2] != "## Empty Body" {
		t.Errorf("summary[2] = %q, want %q", summaries[2], "## Empty Body")
	}
}

func TestBuildSectionSummaries_LongFirstLine(t *testing.T) {
	longLine := strings.Repeat("word ", 30) // 150 chars
	sections := []Section{
		{Heading: "## Long", Body: "\n" + longLine, Index: 0},
	}
	summaries := buildSectionSummaries(sections)
	// Should be truncated to 100 chars + ellipsis.
	if !strings.HasSuffix(summaries[0], "…") {
		t.Errorf("expected truncated summary ending with ellipsis, got %q", summaries[0])
	}
}

// ---------------------------------------------------------------------------
// buildSectionPrompt
// ---------------------------------------------------------------------------

func TestBuildSectionPrompt_Basic(t *testing.T) {
	prompt := buildSectionPrompt("## Heading\nBody text.", nil, "", "")
	if !strings.Contains(prompt, "## Heading") {
		t.Error("prompt should contain section content")
	}
	if !strings.Contains(prompt, "<<<SECTION>>>") {
		t.Error("prompt should contain SECTION delimiter")
	}
}

func TestBuildSectionPrompt_WithContext(t *testing.T) {
	ctx := []string{"## Other — summary of other"}
	prompt := buildSectionPrompt("## A\nBody", ctx, "", "")
	if !strings.Contains(prompt, "OTHER SECTIONS") {
		t.Error("prompt should include context header when summaries provided")
	}
	if !strings.Contains(prompt, "summary of other") {
		t.Error("prompt should include context summaries")
	}
}

func TestBuildSectionPrompt_WithStyleReference(t *testing.T) {
	prompt := buildSectionPrompt("## A\nBody", nil, "Previously polished text here.", "")
	if !strings.Contains(prompt, "STYLE REFERENCE") {
		t.Error("prompt should include style reference header")
	}
	if !strings.Contains(prompt, "<<<STYLE>>>") {
		t.Error("prompt should include STYLE delimiters")
	}
	if !strings.Contains(prompt, "Previously polished text here.") {
		t.Error("prompt should include the style text")
	}
}

func TestBuildSectionPrompt_WithAudience(t *testing.T) {
	prompt := buildSectionPrompt("## A\nBody", nil, "", "junior developers")
	if !strings.Contains(prompt, "TARGET AUDIENCE: junior developers") {
		t.Error("prompt should include audience when provided")
	}
}

func TestBuildSectionPrompt_AllOptions(t *testing.T) {
	ctx := []string{"## B — summary B"}
	prompt := buildSectionPrompt("## A\nBody", ctx, "style ref", "senior engineers")
	if !strings.Contains(prompt, "TARGET AUDIENCE") {
		t.Error("missing audience")
	}
	if !strings.Contains(prompt, "OTHER SECTIONS") {
		t.Error("missing context")
	}
	if !strings.Contains(prompt, "STYLE REFERENCE") {
		t.Error("missing style reference")
	}
	if !strings.Contains(prompt, "<<<SECTION>>>") {
		t.Error("missing section delimiters")
	}
}
