// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

func TestAnalyzeDraft_IncompleteDoc_Suggestions(t *testing.T) {
	doc := "---\ntype: decision\n---\nShort."
	meta := domain.DocMeta{Type: "decision"}

	suggestions := AnalyzeDraft(doc, meta, nil, nil, nil)
	if len(suggestions) == 0 {
		t.Fatal("expected suggestions for incomplete doc, got 0")
	}

	// Should flag: missing What, Why, Alternatives, Impact, short body, no tags
	var hasWhat, hasWhy bool
	for _, s := range suggestions {
		if s.Message == `Section "## What" is missing` {
			hasWhat = true
		}
		if s.Message == `Section "## Why" is missing` {
			hasWhy = true
		}
	}
	if !hasWhat {
		t.Error("expected suggestion for missing ## What")
	}
	if !hasWhy {
		t.Error("expected suggestion for missing ## Why")
	}
}

func TestAnalyzeDraft_CompleteDoc_NoSuggestions(t *testing.T) {
	doc := "---\ntype: decision\ntags: [api]\nrelated: [other-doc]\n---\n" +
		"## What\nThis is a complete document about something important.\n\n" +
		"## Why\nBecause we need this for good reasons that are explained here in detail.\n\n" +
		"## Alternatives\nWe could do nothing.\n\n" +
		"## Impact\nThis affects the API layer.\n"
	meta := domain.DocMeta{
		Type:    "decision",
		Tags:    []string{"api"},
		Related: []string{"other-doc"},
	}

	suggestions := AnalyzeDraft(doc, meta, nil, nil, nil)
	if len(suggestions) != 0 {
		for _, s := range suggestions {
			t.Errorf("unexpected suggestion: [%s] %s: %s", s.Severity, s.Category, s.Message)
		}
	}
}

func TestAnalyzeDraft_EmptyWhy_Warning(t *testing.T) {
	doc := "---\ntype: decision\n---\n" +
		"## What\nSomething important described here in enough detail to pass.\n\n" +
		"## Why\nok\n\n" +
		"## Alternatives\nNone.\n\n" +
		"## Impact\nMinimal.\n"
	meta := domain.DocMeta{Type: "decision"}

	suggestions := AnalyzeDraft(doc, meta, nil, nil, nil)
	var found bool
	for _, s := range suggestions {
		if s.Category == "completeness" && s.Severity == "warning" && s.Message == `Section "## Why" is too brief (< 20 characters)` {
			found = true
		}
	}
	if !found {
		t.Error("expected warning for brief Why section")
	}
}

func TestAnalyzeDraft_StyleGuide_RequireAlternatives_Warning(t *testing.T) {
	doc := "---\ntype: decision\n---\n" +
		"## What\nSomething important described here in enough detail to pass.\n\n" +
		"## Why\nThis is a detailed reason for the decision we are making.\n\n" +
		"## Impact\nMinimal impact.\n"
	meta := domain.DocMeta{Type: "decision", Tags: []string{"api"}}
	guide := &StyleGuide{RequireWhy: true, RequireAlternatives: true}

	suggestions := AnalyzeDraft(doc, meta, guide, nil, nil)
	var found bool
	for _, s := range suggestions {
		if s.Category == "structure" && s.Severity == "warning" && s.Message == `Section "## Alternatives" is missing` {
			found = true
		}
	}
	if !found {
		t.Error("expected warning for missing Alternatives when RequireAlternatives=true")
	}
}

func TestAnalyzeDraft_StyleGuide_MaxBodyLength(t *testing.T) {
	doc := "---\ntype: decision\n---\n" +
		"## What\nSomething important.\n\n" +
		"## Why\nThis is a detailed reason for the decision we are making.\n\n" +
		"## Alternatives\nNone.\n\n" +
		"## Impact\nMinimal.\n"
	meta := domain.DocMeta{Type: "decision", Tags: []string{"api"}}
	guide := &StyleGuide{RequireWhy: true, MaxBodyLength: 10}

	suggestions := AnalyzeDraft(doc, meta, guide, nil, nil)
	var found bool
	for _, s := range suggestions {
		if s.Category == "style" && s.Message == "Body exceeds recommended maximum length" {
			found = true
		}
	}
	if !found {
		t.Error("expected style suggestion for body exceeding MaxBodyLength=10")
	}
}

func TestAnalyzeDraft_StyleGuide_MinTags(t *testing.T) {
	doc := "---\ntype: decision\ntags: [api]\n---\n" +
		"## What\nSomething important described here in enough detail.\n\n" +
		"## Why\nThis is a detailed reason for the decision we are making.\n"
	meta := domain.DocMeta{Type: "decision", Tags: []string{"api"}}
	guide := &StyleGuide{RequireWhy: true, MinTags: 3}

	suggestions := AnalyzeDraft(doc, meta, guide, nil, nil)
	var found bool
	for _, s := range suggestions {
		if s.Category == "completeness" && s.Message == "Consider adding tags for discoverability" {
			found = true
		}
	}
	if !found {
		t.Error("expected completeness suggestion when tags < MinTags=3")
	}
}

func TestAnalyzeDraft_EmptyTags_Info(t *testing.T) {
	doc := "---\ntype: decision\ntags: []\n---\n" +
		"## What\nSomething important with enough content here to pass checks.\n\n" +
		"## Why\nThis is a detailed reason for the decision we are making.\n"
	meta := domain.DocMeta{Type: "decision", Tags: []string{}}

	suggestions := AnalyzeDraft(doc, meta, nil, nil, nil)
	var found bool
	for _, s := range suggestions {
		if s.Category == "completeness" && s.Message == "Consider adding tags for discoverability" {
			found = true
		}
	}
	if !found {
		t.Error("expected info suggestion for empty tags")
	}
}

func TestAnalyzeDraft_WithPersonas_IncludesPersonaSuggestions(t *testing.T) {
	// Story 6.5 AC-3: persona draft checks are merged into suggestions
	doc := "---\ntype: decision\n---\n" +
		"## What\nSomething important described here in enough detail to pass.\n\n" +
		"## Why\n- reason 1\n- reason 2\n- reason 3\n- reason 4\nShort.\n\n" +
		"## Alternatives\nNone.\n\n" +
		"## Impact\nMinimal.\n"
	meta := domain.DocMeta{Type: "decision", Tags: []string{"api"}}

	// Pass storyteller persona — should flag listy Why
	personas := []PersonaProfile{GetRegistry()[0]} // storyteller = Affoue
	suggestions := AnalyzeDraft(doc, meta, nil, nil, personas)

	var found bool
	for _, s := range suggestions {
		if s.Category == "persona" && strings.Contains(s.Message, "Affoue") {
			found = true
		}
	}
	if !found {
		t.Error("expected persona suggestion from Affoue (storyteller) when personas are active")
	}
}

func TestAnalyzeDraft_NilPersonas_NoPersonaSuggestions(t *testing.T) {
	doc := "---\ntype: decision\n---\n" +
		"## What\nSomething important described here in enough detail to pass.\n\n" +
		"## Why\n- reason 1\n- reason 2\n- reason 3\n- reason 4\nShort.\n\n" +
		"## Alternatives\nNone.\n\n" +
		"## Impact\nMinimal.\n"
	meta := domain.DocMeta{Type: "decision", Tags: []string{"api"}}

	suggestions := AnalyzeDraft(doc, meta, nil, nil, nil)
	for _, s := range suggestions {
		if s.Category == "persona" {
			t.Errorf("unexpected persona suggestion with nil personas: %s", s.Message)
		}
	}
}

// --- stripFrontMatter unit tests ---

func TestStripFrontMatter_NoFrontMatter(t *testing.T) {
	doc := "## What\nSome content here.\n"
	result := stripFrontMatter(doc)
	if result != doc {
		t.Errorf("stripFrontMatter should return doc unchanged when no front matter, got %q", result)
	}
}

func TestStripFrontMatter_MalformedOnlyOneDelimiter(t *testing.T) {
	// Only opening ---, no closing --- — should return doc unchanged.
	doc := "---\ntype: decision\nstatus: draft\nSome content after\n"
	result := stripFrontMatter(doc)
	if result != doc {
		t.Errorf("stripFrontMatter should return doc unchanged for malformed front matter (one ---), got %q", result)
	}
}

func TestStripFrontMatter_EmptyDoc(t *testing.T) {
	result := stripFrontMatter("")
	if result != "" {
		t.Errorf("stripFrontMatter should return empty string for empty doc, got %q", result)
	}
}

func TestStripFrontMatter_ValidFrontMatter(t *testing.T) {
	doc := "---\ntype: decision\n---\n## What\nContent.\n"
	result := stripFrontMatter(doc)
	expected := "## What\nContent.\n"
	if result != expected {
		t.Errorf("stripFrontMatter = %q, want %q", result, expected)
	}
}
