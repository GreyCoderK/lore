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
