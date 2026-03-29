// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package store

import (
	"fmt"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/domain"
)

func testCommit(hash, scope, decision string) domain.CommitRecord {
	return domain.CommitRecord{
		Hash:         hash,
		Date:         time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC),
		Branch:       "main",
		Scope:        scope,
		ConvType:     "feat",
		Subject:      "add feature",
		Message:      "feat(auth): add feature",
		FilesChanged: 3,
		LinesAdded:   50,
		LinesDeleted: 10,
		DocID:        "decision-auth.md",
		Decision:     decision,
		QuestionMode: "full",
	}
}

func TestRecordCommit_GetCommit_Roundtrip(t *testing.T) {
	s, _ := tempDB(t)

	rec := testCommit("abc123", "auth", "documented")
	rec.DecisionScore = 85
	rec.DecisionConfidence = 0.92
	rec.SkipReason = ""

	if err := s.RecordCommit(rec); err != nil {
		t.Fatalf("RecordCommit: %v", err)
	}

	got, err := s.GetCommit("abc123")
	if err != nil {
		t.Fatalf("GetCommit: %v", err)
	}
	if got == nil {
		t.Fatal("GetCommit returned nil")
	}

	if got.Hash != "abc123" {
		t.Errorf("Hash = %q, want abc123", got.Hash)
	}
	if got.Scope != "auth" {
		t.Errorf("Scope = %q, want auth", got.Scope)
	}
	if got.Decision != "documented" {
		t.Errorf("Decision = %q, want documented", got.Decision)
	}
	if got.FilesChanged != 3 {
		t.Errorf("FilesChanged = %d, want 3", got.FilesChanged)
	}
	if got.DecisionScore != 85 {
		t.Errorf("DecisionScore = %d, want 85", got.DecisionScore)
	}
	if got.DocID != "decision-auth.md" {
		t.Errorf("DocID = %q, want decision-auth.md", got.DocID)
	}
}

func TestGetCommit_NotFound(t *testing.T) {
	s, _ := tempDB(t)
	got, err := s.GetCommit("nonexistent")
	if err != nil {
		t.Fatalf("GetCommit: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent commit")
	}
}

func TestCommitsByScope_FiltersCorrectly(t *testing.T) {
	s, _ := tempDB(t)

	// 2 auth commits, 3 api commits
	for i, scope := range []string{"auth", "auth", "api", "api", "api"} {
		rec := testCommit("hash"+string(rune('a'+i)), scope, "documented")
		rec.Date = time.Now().Add(-time.Duration(i) * time.Hour) // recent
		if err := s.RecordCommit(rec); err != nil {
			t.Fatalf("RecordCommit: %v", err)
		}
	}

	results, err := s.CommitsByScope("auth", 30)
	if err != nil {
		t.Fatalf("CommitsByScope: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("auth commits = %d, want 2", len(results))
	}
}

func TestUndocumentedCommits(t *testing.T) {
	s, _ := tempDB(t)

	decisions := []string{"documented", "pending", "unknown", "skipped", "pending"}
	for i, d := range decisions {
		rec := testCommit(fmt.Sprintf("hash-undoc-%d", i), "auth", d)
		if err := s.RecordCommit(rec); err != nil {
			t.Fatalf("RecordCommit: %v", err)
		}
	}

	results, err := s.UndocumentedCommits()
	if err != nil {
		t.Fatalf("UndocumentedCommits: %v", err)
	}
	// pending + unknown + pending = 3
	if len(results) != 3 {
		t.Errorf("undocumented = %d, want 3", len(results))
	}
}

func TestScopeStats(t *testing.T) {
	s, _ := tempDB(t)

	// Insert commits with various decisions for scope "auth"
	decisions := []string{"documented", "documented", "skipped", "auto-skipped", "pending"}
	for i, d := range decisions {
		rec := testCommit(fmt.Sprintf("hash-stats-%d", i), "auth", d)
		rec.Date = time.Now().Add(-time.Duration(i) * 24 * time.Hour) // spread across days
		if err := s.RecordCommit(rec); err != nil {
			t.Fatalf("RecordCommit: %v", err)
		}
	}

	// Also insert a commit in a different scope to ensure filtering
	other := testCommit("hash-stats-other", "api", "documented")
	other.Date = time.Now()
	if err := s.RecordCommit(other); err != nil {
		t.Fatalf("RecordCommit: %v", err)
	}

	stats, err := s.ScopeStats("auth", 30)
	if err != nil {
		t.Fatalf("ScopeStats: %v", err)
	}
	if stats.TotalCommits != 5 {
		t.Errorf("TotalCommits = %d, want 5", stats.TotalCommits)
	}
	if stats.DocumentedCount != 2 {
		t.Errorf("DocumentedCount = %d, want 2", stats.DocumentedCount)
	}
	if stats.SkippedCount != 2 {
		t.Errorf("SkippedCount = %d, want 2", stats.SkippedCount)
	}
	if stats.LastDocDate == 0 {
		t.Error("LastDocDate should be non-zero")
	}
	if stats.LastCommitDate == 0 {
		t.Error("LastCommitDate should be non-zero")
	}

	// Empty scope should return zero counts
	empty, err := s.ScopeStats("nonexistent", 30)
	if err != nil {
		t.Fatalf("ScopeStats empty: %v", err)
	}
	if empty.TotalCommits != 0 {
		t.Errorf("empty TotalCommits = %d, want 0", empty.TotalCommits)
	}
}

func TestCommitCountByDecision(t *testing.T) {
	s, _ := tempDB(t)

	for i, d := range []string{"documented", "documented", "skipped", "pending"} {
		rec := testCommit("hash"+string(rune('a'+i)), "auth", d)
		if err := s.RecordCommit(rec); err != nil {
			t.Fatalf("RecordCommit: %v", err)
		}
	}

	counts, err := s.CommitCountByDecision()
	if err != nil {
		t.Fatalf("CommitCountByDecision: %v", err)
	}
	if counts["documented"] != 2 {
		t.Errorf("documented = %d, want 2", counts["documented"])
	}
	if counts["skipped"] != 1 {
		t.Errorf("skipped = %d, want 1", counts["skipped"])
	}
	if counts["pending"] != 1 {
		t.Errorf("pending = %d, want 1", counts["pending"])
	}
}
