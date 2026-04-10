// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

func TestCheckCoherence_EmptyCorpus(t *testing.T) {
	doc := "---\ntype: decision\n---\nSome content."
	meta := domain.DocMeta{Type: "decision", Filename: "test.md"}

	suggestions := CheckCoherence(doc, meta, nil)
	if len(suggestions) != 0 {
		t.Errorf("expected 0 suggestions for empty corpus, got %d", len(suggestions))
	}
}

func TestCheckCoherence_SharedTags_CrossReference(t *testing.T) {
	doc := "---\ntype: decision\n---\nContent here."
	meta := domain.DocMeta{
		Type:     "decision",
		Tags:     []string{"api", "auth"},
		Filename: "decision-auth-2026.md",
	}
	corpus := []domain.DocMeta{
		{Type: "feature", Tags: []string{"api"}, Filename: "feature-api-2026.md"},
	}

	suggestions := CheckCoherence(doc, meta, corpus)
	var found bool
	for _, s := range suggestions {
		if s.Category == "coherence" && s.Severity == "info" {
			found = true
		}
	}
	if !found {
		t.Error("expected cross-reference suggestion for shared tags")
	}
}

func TestCheckCoherence_SameTypeSharedTag_Duplicate(t *testing.T) {
	doc := "---\ntype: decision\n---\nContent."
	meta := domain.DocMeta{
		Type:     "decision",
		Tags:     []string{"auth"},
		Filename: "decision-new-2026.md",
	}
	corpus := []domain.DocMeta{
		{Type: "decision", Tags: []string{"auth"}, Filename: "decision-old-2026.md"},
	}

	suggestions := CheckCoherence(doc, meta, corpus)
	var found bool
	for _, s := range suggestions {
		if s.Category == "coherence" && s.Severity == "warning" {
			found = true
		}
	}
	if !found {
		t.Error("expected duplicate warning for same type + shared tag")
	}
}

func TestCheckCoherence_ScopeOverlap(t *testing.T) {
	doc := "---\ntype: decision\n---\nContent."
	meta := domain.DocMeta{
		Type:     "decision",
		Scope:    "auth",
		Filename: "decision-new-2026.md",
	}
	corpus := []domain.DocMeta{
		{Type: "decision", Scope: "auth", Filename: "decision-old-2026.md"},
	}

	suggestions := CheckCoherence(doc, meta, corpus)
	var found bool
	for _, s := range suggestions {
		if s.Severity == "warning" && s.Category == "coherence" {
			found = true
		}
	}
	if !found {
		t.Error("expected scope overlap warning for same scope + type")
	}
}

func TestCheckCoherence_ScopeCrossRef(t *testing.T) {
	doc := "---\ntype: feature\n---\nContent."
	meta := domain.DocMeta{
		Type:     "feature",
		Scope:    "auth",
		Filename: "feature-login-2026.md",
	}
	corpus := []domain.DocMeta{
		{Type: "decision", Scope: "auth", Filename: "decision-auth-2026.md"},
	}

	suggestions := CheckCoherence(doc, meta, corpus)
	var found bool
	for _, s := range suggestions {
		if s.Severity == "info" && s.Category == "coherence" {
			found = true
		}
	}
	if !found {
		t.Error("expected scope cross-ref info for same scope, different type")
	}
}

func TestCheckCoherence_AlreadyRelated(t *testing.T) {
	doc := "---\ntype: feature\n---\nContent."
	meta := domain.DocMeta{
		Type:     "feature",
		Tags:     []string{"api"},
		Related:  []string{"decision-api-2026"},
		Filename: "feature-api-2026.md",
	}
	corpus := []domain.DocMeta{
		{Type: "decision", Tags: []string{"api"}, Filename: "decision-api-2026.md"},
	}

	suggestions := CheckCoherence(doc, meta, corpus)
	for _, s := range suggestions {
		if s.Category == "coherence" && s.Severity == "info" {
			t.Error("should not suggest cross-ref for already-related doc")
		}
	}
}

func TestCheckCoherence_BodyMention(t *testing.T) {
	doc := "---\ntype: decision\n---\nSee also decision-api-2026 for context."
	meta := domain.DocMeta{
		Type:     "decision",
		Filename: "decision-auth-2026.md",
	}
	corpus := []domain.DocMeta{
		{Type: "decision", Filename: "decision-api-2026.md"},
	}

	suggestions := CheckCoherence(doc, meta, corpus)
	var found bool
	for _, s := range suggestions {
		if s.Category == "coherence" && s.Severity == "info" {
			found = true
		}
	}
	if !found {
		t.Error("expected body mention suggestion")
	}
}

// --- Translation pair detection ---

func TestIsTranslationPair(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"installation.md", "installation.fr.md", true},
		{"installation.fr.md", "installation.md", true},
		{"installation.en.md", "installation.fr.md", true},
		{"index.md", "index.fr.md", true},
		{"guides/philosophy.md", "guides/philosophy.md", false},   // same file
		{"guides/philosophy.md", "guides/roadmap.md", false},      // different base
		{"api.v2.md", "api.v2.fr.md", true},                        // "v2" is not a lang code, so bases match
		{"api.v2.md", "api.v3.md", false},                          // neither v2/v3 is a lang code
		{"installation.md", "configuration.md", false},
		{"installation.xyz.md", "installation.fr.md", false},       // "xyz" is not in lang list
	}
	for _, tt := range tests {
		got := isTranslationPair(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("isTranslationPair(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

// EN/FR translation pairs with shared inferred tags (as produced by
// PlainCorpusStore in standalone mode) must NOT be flagged as duplicates.
func TestCheckCoherence_TranslationPair_NotFlaggedAsDuplicate(t *testing.T) {
	doc := "---\ntype: guide\n---\nSome content."
	// Decision type to enter the duplicate-check branch; even then, the
	// translation-pair guard must skip the warning.
	meta := domain.DocMeta{
		Type:     "decision",
		Tags:     []string{"install"},
		Filename: "installation.md",
	}
	corpus := []domain.DocMeta{
		{Type: "decision", Tags: []string{"install"}, Filename: "installation.fr.md"},
	}

	suggestions := CheckCoherence(doc, meta, corpus)
	for _, s := range suggestions {
		if s.Category == "coherence" && s.Severity == "warning" {
			t.Errorf("unexpected duplicate warning on translation pair: %s", s.Message)
		}
	}
}

// Free-form types (guide, note, tutorial, index) must not trigger any
// duplicate-by-shared-tag warnings, even if another free-form doc exists.
func TestCheckCoherence_FreeFormType_NoDuplicateWarning(t *testing.T) {
	doc := "---\ntype: guide\n---\nSome content."
	meta := domain.DocMeta{
		Type:     "guide",
		Tags:     []string{"install", "setup"},
		Filename: "getting-started.md",
	}
	corpus := []domain.DocMeta{
		{Type: "guide", Tags: []string{"install"}, Filename: "installation.md"},
		{Type: "guide", Tags: []string{"setup"}, Filename: "configuration.md"},
	}

	suggestions := CheckCoherence(doc, meta, corpus)
	for _, s := range suggestions {
		if s.Category == "coherence" && s.Severity == "warning" {
			t.Errorf("unexpected warning on free-form type: %s", s.Message)
		}
	}
}

func TestSplitLangFromFilename(t *testing.T) {
	tests := []struct {
		in       string
		wantBase string
		wantLang string
	}{
		{"installation.md", "installation", ""},
		{"installation.fr.md", "installation", "fr"},
		{"installation.en.md", "installation", "en"},
		{"api.v2.md", "api.v2", ""},     // v2 not in lang list → kept in base
		{"no-extension", "no-extension", ""},
		{"nested/path/file.de.md", "nested/path/file", "de"},
	}
	for _, tt := range tests {
		base, lang := splitLangFromFilename(tt.in)
		if base != tt.wantBase || lang != tt.wantLang {
			t.Errorf("splitLangFromFilename(%q) = (%q, %q), want (%q, %q)",
				tt.in, base, lang, tt.wantBase, tt.wantLang)
		}
	}
}
