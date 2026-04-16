// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package synthesizer

import (
	"strings"
	"testing"
)

const sampleDoc = `---
type: feature
date: "2026-04-15"
status: draft
---

# Title

## Why

A reason.

## Endpoints

A list:
- POST /api/foo
- GET /api/bar

### Sub-detail

Details here.

## Sécurité

Server-injected: corporate, saleCode.
`

func TestParseDoc_FrontmatterBodyAndLines(t *testing.T) {
	doc, err := ParseDoc("test.md", []byte(sampleDoc))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Meta.Type != "feature" {
		t.Fatalf("Meta.Type: %q", doc.Meta.Type)
	}
	if !strings.Contains(doc.Body, "## Endpoints") {
		t.Fatal("Body should contain ## Endpoints")
	}
	if doc.Lines[0] != "" {
		t.Fatal("Lines[0] must be the placeholder empty string")
	}
	// The first body line is empty (between --- and # Title).
	if len(doc.Lines) < 3 {
		t.Fatalf("Lines too short: %d", len(doc.Lines))
	}
}

func TestParseSections_FlatList(t *testing.T) {
	body := strings.Split(sampleDoc, "---\n")[2] // skip frontmatter
	secs := ParseSections(body)

	wantTitles := []string{"Title", "Why", "Endpoints", "Sub-detail", "Sécurité"}
	if len(secs) != len(wantTitles) {
		t.Fatalf("got %d sections, want %d: %+v", len(secs), len(wantTitles), titlesOf(secs))
	}
	for i, want := range wantTitles {
		if secs[i].Title != want {
			t.Fatalf("section %d: got %q, want %q", i, secs[i].Title, want)
		}
	}
	if secs[0].Level != 1 || secs[1].Level != 2 || secs[3].Level != 3 {
		t.Fatalf("levels wrong: %+v", levelsOf(secs))
	}
}

func TestParseSections_ContentSpansSubheadings(t *testing.T) {
	body := strings.Split(sampleDoc, "---\n")[2]
	secs := ParseSections(body)

	var endpoints *Section
	for i := range secs {
		if secs[i].Title == "Endpoints" {
			endpoints = &secs[i]
		}
	}
	if endpoints == nil {
		t.Fatal("Endpoints section not found")
	}
	if !strings.Contains(endpoints.Content, "POST /api/foo") {
		t.Fatalf("Endpoints content should contain POST /api/foo: %q", endpoints.Content)
	}
	if !strings.Contains(endpoints.Content, "Sub-detail") {
		t.Fatal("Endpoints content should include the sub-heading subtree")
	}
}

func TestFuzzyFindSection_ExactMatch(t *testing.T) {
	doc := mustParse(t, sampleDoc)
	sec, conf, ok := FuzzyFindSection(doc, []string{"Endpoints"})
	if !ok {
		t.Fatal("expected match")
	}
	if conf != 1.0 {
		t.Fatalf("exact match should score 1.0, got %.2f", conf)
	}
	if sec.Title != "Endpoints" {
		t.Fatalf("matched wrong section: %q", sec.Title)
	}
}

func TestFuzzyFindSection_RegexMatch(t *testing.T) {
	doc := mustParse(t, sampleDoc)
	sec, conf, ok := FuzzyFindSection(doc, []string{"(?i)endpoints?|routes?|apis?"})
	if !ok {
		t.Fatal("expected regex match")
	}
	if conf < 0.7 {
		t.Fatalf("regex match should be >=0.7, got %.2f", conf)
	}
	if sec.Title != "Endpoints" {
		t.Fatalf("matched wrong section: %q", sec.Title)
	}
}

func TestFuzzyFindSection_FrenchAccentsPreserved(t *testing.T) {
	doc := mustParse(t, sampleDoc)
	sec, conf, ok := FuzzyFindSection(doc, []string{"sécurité", "security"})
	if !ok {
		t.Fatal("French accent should match")
	}
	if conf != 1.0 {
		t.Fatalf("expected exact match, got %.2f", conf)
	}
	if sec.Title != "Sécurité" {
		t.Fatalf("got %q", sec.Title)
	}
}

func TestFuzzyFindSection_NoMatchReturnsFalse(t *testing.T) {
	doc := mustParse(t, sampleDoc)
	_, _, ok := FuzzyFindSection(doc, []string{"completely-unrelated"})
	if ok {
		t.Fatal("expected no match")
	}
}

func TestFuzzyFindSection_NilDocReturnsFalse(t *testing.T) {
	if _, _, ok := FuzzyFindSection(nil, []string{"x"}); ok {
		t.Fatal("nil doc must return false")
	}
}

func TestNormalize_StripsLeadingNumberingAndEmphasis(t *testing.T) {
	cases := map[string]string{
		"1. Endpoints":       "endpoints",
		"**Endpoints**":      "endpoints**", // leading * stripped, trailing kept
		"  Endpoints  ":      "endpoints",
		"Endpoints  details": "endpoints details",
	}
	for in, want := range cases {
		if got := normalize(in); got != want {
			t.Fatalf("normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseSections_IgnoresHeadingsInsideCodeFences(t *testing.T) {
	body := `## Real

Intro

### Inside

` + "```http\n# Full - required + optional fields\nPOST /api/x\n```" + `

Post-fence content for Inside.

## Other
`
	secs := ParseSections(body)
	// Expect 3 sections: ## Real, ### Inside, ## Other.
	// WITHOUT the fix, "# Full" would be parsed as a level-1 heading and
	// close Real+Inside prematurely, leaving Inside.EndLine pointing at
	// the blank line right after the opening fence.
	titles := titlesOf(secs)
	if len(secs) != 3 {
		t.Fatalf("want 3 sections, got %d: %v", len(secs), titles)
	}
	if titles[0] != "Real" || titles[1] != "Inside" || titles[2] != "Other" {
		t.Fatalf("unexpected sections: %v", titles)
	}

	var inside *Section
	for i := range secs {
		if secs[i].Title == "Inside" {
			inside = &secs[i]
		}
	}
	if inside == nil {
		t.Fatal("Inside section missing")
	}
	if !strings.Contains(inside.Content, "Post-fence content for Inside.") {
		t.Fatalf("Inside content should include text after the fence; got:\n%s", inside.Content)
	}
}

func TestParseSections_NestedFencesStillClose(t *testing.T) {
	body := "## A\n\n```" + "\ncontent\n```" + "\n\n## B\n"
	secs := ParseSections(body)
	if len(secs) != 2 {
		t.Fatalf("want 2 sections, got %d", len(secs))
	}
}

func TestContainsWholeWord(t *testing.T) {
	if !containsWholeWord("api endpoints", "endpoints") {
		t.Fatal("should match whole word")
	}
	if containsWholeWord("endpointsx", "endpoints") {
		t.Fatal("should not match partial")
	}
	if !containsWholeWord("endpoints", "endpoints") {
		t.Fatal("identical should match")
	}
}

// helpers
func titlesOf(secs []Section) []string {
	out := make([]string, len(secs))
	for i, s := range secs {
		out[i] = s.Title
	}
	return out
}

func levelsOf(secs []Section) []int {
	out := make([]int, len(secs))
	for i, s := range secs {
		out[i] = s.Level
	}
	return out
}

func mustParse(t *testing.T, content string) *Doc {
	t.Helper()
	d, err := ParseDoc("t.md", []byte(content))
	if err != nil {
		t.Fatal(err)
	}
	return d
}
