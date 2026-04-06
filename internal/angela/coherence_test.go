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
