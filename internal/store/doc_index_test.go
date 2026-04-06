// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package store

import (
	"context"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/domain"
)

func testDoc(filename, docType, scope string) domain.DocIndexEntry {
	return domain.DocIndexEntry{
		Filename:    filename,
		Type:        docType,
		Date:        "2026-03-15",
		CommitHash:  "abc123",
		Branch:      "main",
		Scope:       scope,
		Status:      "draft",
		Tags:        []string{"auth", "security"},
		Related:     []string{"related.md"},
		GeneratedBy: "hook",
		ContentHash: "sha256-abc",
		SummaryWhy:  "Because reasons",
		SummaryWhat: "JWT authentication",
		WordCount:   150,
		UpdatedAt:   time.Now(),
	}
}

func TestIndexDoc_GetDoc_Roundtrip(t *testing.T) {
	s, _ := tempDB(t)

	entry := testDoc("decision-auth.md", "decision", "auth")

	if err := s.IndexDoc(entry); err != nil {
		t.Fatalf("IndexDoc: %v", err)
	}

	got, err := s.GetDoc("decision-auth.md")
	if err != nil {
		t.Fatalf("GetDoc: %v", err)
	}
	if got == nil {
		t.Fatal("GetDoc returned nil")
	}

	if got.Filename != "decision-auth.md" {
		t.Errorf("Filename = %q, want decision-auth.md", got.Filename)
	}
	if got.Type != "decision" {
		t.Errorf("Type = %q, want decision", got.Type)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "auth" || got.Tags[1] != "security" {
		t.Errorf("Tags = %v, want [auth security]", got.Tags)
	}
	if got.SummaryWhy != "Because reasons" {
		t.Errorf("SummaryWhy = %q, want 'Because reasons'", got.SummaryWhy)
	}
	if got.WordCount != 150 {
		t.Errorf("WordCount = %d, want 150", got.WordCount)
	}
}

func TestGetDoc_NotFound(t *testing.T) {
	s, _ := tempDB(t)
	got, err := s.GetDoc("nonexistent.md")
	if err != nil {
		t.Fatalf("GetDoc: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent doc")
	}
}

func TestRemoveDoc(t *testing.T) {
	s, _ := tempDB(t)

	if err := s.IndexDoc(testDoc("to-delete.md", "decision", "auth")); err != nil {
		t.Fatalf("IndexDoc: %v", err)
	}

	if err := s.RemoveDoc("to-delete.md"); err != nil {
		t.Fatalf("RemoveDoc: %v", err)
	}

	got, err := s.GetDoc("to-delete.md")
	if err != nil {
		t.Fatalf("GetDoc: %v", err)
	}
	if got != nil {
		t.Error("doc should be deleted")
	}
}

func TestDocsByScope(t *testing.T) {
	s, _ := tempDB(t)

	s.IndexDoc(testDoc("a.md", "decision", "auth"))
	s.IndexDoc(testDoc("b.md", "decision", "auth"))
	s.IndexDoc(testDoc("c.md", "feature", "api"))

	results, err := s.DocsByScope("auth")
	if err != nil {
		t.Fatalf("DocsByScope: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("auth docs = %d, want 2", len(results))
	}
}

func TestDocsByType(t *testing.T) {
	s, _ := tempDB(t)

	s.IndexDoc(testDoc("a.md", "decision", "auth"))
	s.IndexDoc(testDoc("b.md", "feature", "api"))
	s.IndexDoc(testDoc("c.md", "decision", "db"))

	results, err := s.DocsByType("decision")
	if err != nil {
		t.Fatalf("DocsByType: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("decision docs = %d, want 2", len(results))
	}
}

func TestSearchDocs(t *testing.T) {
	s, _ := tempDB(t)

	entry := testDoc("decision-auth-jwt.md", "decision", "auth")
	entry.TitleExtracted = "JWT Authentication Strategy"
	s.IndexDoc(entry)

	entry2 := testDoc("feature-api.md", "feature", "api")
	entry2.TitleExtracted = "REST API Endpoints"
	entry2.SummaryWhat = "REST endpoints for users"
	entry2.SummaryWhy = "Speed improvement"
	s.IndexDoc(entry2)

	results, err := s.SearchDocs(context.Background(), "JWT")
	if err != nil {
		t.Fatalf("SearchDocs: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("search 'JWT' = %d results, want 1", len(results))
	}
	if len(results) > 0 && results[0].Filename != "decision-auth-jwt.md" {
		t.Errorf("result = %q, want decision-auth-jwt.md", results[0].Filename)
	}
}

func TestDocCount(t *testing.T) {
	s, _ := tempDB(t)

	s.IndexDoc(testDoc("a.md", "decision", "auth"))
	s.IndexDoc(testDoc("b.md", "feature", "api"))

	count, err := s.DocCount()
	if err != nil {
		t.Fatalf("DocCount: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestDocsByBranch(t *testing.T) {
	s, _ := tempDB(t)

	d1 := testDoc("a.md", "decision", "auth")
	d1.Branch = "feat/auth"
	s.IndexDoc(d1)

	d2 := testDoc("b.md", "feature", "auth")
	d2.Branch = "feat/auth"
	s.IndexDoc(d2)

	d3 := testDoc("c.md", "note", "")
	d3.Branch = "main"
	s.IndexDoc(d3)

	results, err := s.DocsByBranch("feat/auth")
	if err != nil {
		t.Fatalf("DocsByBranch: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("feat/auth docs = %d, want 2", len(results))
	}

	empty, _ := s.DocsByBranch("nonexistent")
	if len(empty) != 0 {
		t.Errorf("nonexistent docs = %d, want 0", len(empty))
	}
}

func TestUnconsolidatedDocs(t *testing.T) {
	s, _ := tempDB(t)

	d1 := testDoc("a.md", "decision", "auth")
	d1.ConsolidatedInto = ""
	s.IndexDoc(d1)

	d2 := testDoc("b.md", "feature", "auth")
	d2.ConsolidatedInto = "summary-auth.md"
	s.IndexDoc(d2)

	d3 := testDoc("c.md", "bugfix", "auth")
	d3.ConsolidatedInto = ""
	s.IndexDoc(d3)

	results, err := s.UnconsolidatedDocs("auth")
	if err != nil {
		t.Fatalf("UnconsolidatedDocs: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("unconsolidated auth = %d, want 2", len(results))
	}
}

func TestAllDocSummaries(t *testing.T) {
	s, _ := tempDB(t)

	for i := 0; i < 5; i++ {
		d := testDoc("doc-"+string(rune('a'+i))+".md", "decision", "auth")
		d.Date = "2026-03-" + string(rune('0'+1+i)) + "0"
		s.IndexDoc(d)
	}

	results, err := s.AllDocSummaries(3)
	if err != nil {
		t.Fatalf("AllDocSummaries: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("summaries = %d, want 3 (limited)", len(results))
	}
}

func TestDocsByCommitHash(t *testing.T) {
	s, _ := tempDB(t)

	d1 := testDoc("a.md", "decision", "auth")
	d1.CommitHash = "abc123"
	s.IndexDoc(d1)

	d2 := testDoc("b.md", "feature", "api")
	d2.CommitHash = "def456"
	s.IndexDoc(d2)

	results, err := s.DocsByCommitHash("abc123")
	if err != nil {
		t.Fatalf("DocsByCommitHash: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("docs for abc123 = %d, want 1", len(results))
	}
	if len(results) > 0 && results[0].Filename != "a.md" {
		t.Errorf("filename = %q, want a.md", results[0].Filename)
	}

	empty, _ := s.DocsByCommitHash("nonexistent")
	if len(empty) != 0 {
		t.Errorf("nonexistent docs = %d, want 0", len(empty))
	}
}

func TestDocEmptyTags(t *testing.T) {
	s, _ := tempDB(t)

	entry := testDoc("no-tags.md", "decision", "auth")
	entry.Tags = nil
	entry.Related = nil
	s.IndexDoc(entry)

	got, _ := s.GetDoc("no-tags.md")
	if got == nil {
		t.Fatal("GetDoc returned nil")
	}
	if len(got.Tags) != 0 {
		t.Errorf("Tags = %v, want empty", got.Tags)
	}
}
