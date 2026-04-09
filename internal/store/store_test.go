// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/domain"
)

func tempDB(t *testing.T) (*SQLiteStore, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "store.db")
	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s, dbPath
}

func TestOpen_CreatesFile(t *testing.T) {
	_, dbPath := tempDB(t)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("store.db should be created")
	}
}

func TestOpen_WALMode(t *testing.T) {
	s, _ := tempDB(t)
	var mode string
	if err := s.db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want wal", mode)
	}
}

func TestOpen_ForeignKeys(t *testing.T) {
	s, _ := tempDB(t)
	var fk int
	if err := s.db.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatalf("PRAGMA foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d, want 1", fk)
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	s, _ := tempDB(t)
	// Migrate is called in Open already — calling again should be no-op
	if err := s.Migrate(); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	// Verify schema_version
	var version int
	if err := s.db.QueryRow("SELECT MAX(version) FROM schema_version").Scan(&version); err != nil {
		t.Fatalf("schema_version query: %v", err)
	}
	if version != 2 {
		t.Errorf("schema_version = %d, want 2", version)
	}
}

func TestMigrate_AllTablesExist(t *testing.T) {
	s, _ := tempDB(t)

	tables := []string{
		"schema_version", "commits", "code_signatures", "doc_index",
		"ai_usage", "review_cache", "commit_patterns",
		"commit_relations", "doc_edges",
	}
	for _, table := range tables {
		var name string
		err := s.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}

func TestClose_NilSafe(t *testing.T) {
	var s *SQLiteStore
	if err := s.Close(); err != nil {
		t.Errorf("Close on nil should not error: %v", err)
	}
}

func TestReservedTables_ExistButEmpty(t *testing.T) {
	s, _ := tempDB(t)

	reserved := []string{"code_signatures", "ai_usage", "review_cache", "commit_patterns", "commit_relations", "doc_edges"}
	for _, table := range reserved {
		var count int
		err := s.db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count)
		if err != nil {
			t.Errorf("table %q query error: %v", table, err)
			continue
		}
		if count != 0 {
			t.Errorf("table %q should be empty, has %d rows", table, count)
		}
	}
}

func TestRecordCommit_Upsert(t *testing.T) {
	s, _ := tempDB(t)

	rec := domain.CommitRecord{
		Hash:         "upsert-test",
		Date:         time.Now(),
		Branch:       "main",
		Scope:        "auth",
		ConvType:     "feat",
		Subject:      "original subject",
		Message:      "feat(auth): original",
		FilesChanged: 1,
		Decision:     "pending",
		QuestionMode: "full",
	}
	if err := s.RecordCommit(rec); err != nil {
		t.Fatalf("RecordCommit (insert): %v", err)
	}

	// Update same hash with new decision
	rec.Decision = "documented"
	rec.Subject = "updated subject"
	rec.DocID = "decision-auth.md"
	if err := s.RecordCommit(rec); err != nil {
		t.Fatalf("RecordCommit (upsert): %v", err)
	}

	got, err := s.GetCommit("upsert-test")
	if err != nil {
		t.Fatalf("GetCommit: %v", err)
	}
	if got == nil {
		t.Fatal("GetCommit returned nil after upsert")
	}
	if got.Decision != "documented" {
		t.Errorf("Decision = %q, want 'documented' after upsert", got.Decision)
	}
	if got.Subject != "updated subject" {
		t.Errorf("Subject = %q, want 'updated subject' after upsert", got.Subject)
	}
	if got.DocID != "decision-auth.md" {
		t.Errorf("DocID = %q, want 'decision-auth.md' after upsert", got.DocID)
	}
}

func TestCommitsByScope_OldCommitsExcluded(t *testing.T) {
	s, _ := tempDB(t)

	// Insert a commit from 60 days ago
	old := domain.CommitRecord{
		Hash: "old-commit", Date: time.Now().AddDate(0, 0, -60),
		Branch: "main", Scope: "auth", ConvType: "feat", Subject: "old",
		Message: "old commit", Decision: "documented", QuestionMode: "full",
	}
	if err := s.RecordCommit(old); err != nil {
		t.Fatalf("RecordCommit (old): %v", err)
	}

	// Insert a recent commit
	recent := domain.CommitRecord{
		Hash: "recent-commit", Date: time.Now().Add(-1 * time.Hour),
		Branch: "main", Scope: "auth", ConvType: "feat", Subject: "recent",
		Message: "recent commit", Decision: "documented", QuestionMode: "full",
	}
	if err := s.RecordCommit(recent); err != nil {
		t.Fatalf("RecordCommit (recent): %v", err)
	}

	// Query with 30-day window should exclude old commit
	results, err := s.CommitsByScope("auth", 30)
	if err != nil {
		t.Fatalf("CommitsByScope: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("commits within 30 days = %d, want 1", len(results))
	}
	if len(results) > 0 && results[0].Hash != "recent-commit" {
		t.Errorf("got hash %q, want 'recent-commit'", results[0].Hash)
	}
}

func TestCommitCountByDecision_Empty(t *testing.T) {
	s, _ := tempDB(t)

	counts, err := s.CommitCountByDecision()
	if err != nil {
		t.Fatalf("CommitCountByDecision: %v", err)
	}
	if len(counts) != 0 {
		t.Errorf("expected empty map, got %v", counts)
	}
}

func TestScopeStats_NarrowWindow(t *testing.T) {
	s, _ := tempDB(t)

	// Insert commits spread across time
	for i, d := range []string{"documented", "skipped", "pending"} {
		rec := domain.CommitRecord{
			Hash: fmt.Sprintf("narrow-%d", i), Date: time.Now().Add(-time.Duration(i*10) * 24 * time.Hour),
			Branch: "main", Scope: "db", ConvType: "fix", Subject: "fix",
			Message: "fix(db): something", Decision: d, QuestionMode: "full",
		}
		if err := s.RecordCommit(rec); err != nil {
			t.Fatalf("RecordCommit: %v", err)
		}
	}

	// 5-day window should only include the most recent commit
	stats, err := s.ScopeStats("db", 5)
	if err != nil {
		t.Fatalf("ScopeStats: %v", err)
	}
	if stats.TotalCommits != 1 {
		t.Errorf("TotalCommits (5d window) = %d, want 1", stats.TotalCommits)
	}
	if stats.DocumentedCount != 1 {
		t.Errorf("DocumentedCount (5d window) = %d, want 1", stats.DocumentedCount)
	}
}

func TestOpen_InvalidPath(t *testing.T) {
	// Opening a DB at an invalid path should return an error during migration/pragma
	_, err := Open("/nonexistent/dir/db.sqlite")
	if err == nil {
		t.Error("expected error opening DB at invalid path")
	}
}

func TestRecordCommit_ClosedDB(t *testing.T) {
	s, _ := tempDB(t)
	_ = s.Close()

	rec := domain.CommitRecord{
		Hash: "closed-db", Date: time.Now(), Branch: "main",
		Decision: "documented", QuestionMode: "full", Message: "msg",
	}
	err := s.RecordCommit(rec)
	if err == nil {
		t.Error("expected error recording commit on closed DB")
	}
}

func TestCommitsByScope_ClosedDB(t *testing.T) {
	s, _ := tempDB(t)
	_ = s.Close()

	_, err := s.CommitsByScope("auth", 30)
	if err == nil {
		t.Error("expected error querying closed DB")
	}
}

func TestCommitCountByDecision_ClosedDB(t *testing.T) {
	s, _ := tempDB(t)
	_ = s.Close()

	_, err := s.CommitCountByDecision()
	if err == nil {
		t.Error("expected error querying closed DB")
	}
}

func TestScopeStats_ClosedDB(t *testing.T) {
	s, _ := tempDB(t)
	_ = s.Close()

	_, err := s.ScopeStats("auth", 30)
	if err == nil {
		t.Error("expected error querying closed DB")
	}
}

func TestIndexDoc_ClosedDB(t *testing.T) {
	s, _ := tempDB(t)
	_ = s.Close()

	err := s.IndexDoc(domain.DocIndexEntry{
		Filename: "test.md", Type: "decision", Date: "2026-03-15",
		Status: "draft", ContentHash: "abc", UpdatedAt: time.Now(),
	})
	if err == nil {
		t.Error("expected error indexing doc on closed DB")
	}
}

func TestRemoveDoc_ClosedDB(t *testing.T) {
	s, _ := tempDB(t)
	_ = s.Close()

	err := s.RemoveDoc("test.md")
	if err == nil {
		t.Error("expected error removing doc on closed DB")
	}
}

func TestDocCount_ClosedDB(t *testing.T) {
	s, _ := tempDB(t)
	_ = s.Close()

	_, err := s.DocCount()
	if err == nil {
		t.Error("expected error from DocCount on closed DB")
	}
}

func TestCommitsByBranch_ClosedDB(t *testing.T) {
	s, _ := tempDB(t)
	_ = s.Close()

	_, err := s.CommitsByBranch("main")
	if err == nil {
		t.Error("expected error querying closed DB")
	}
}

func TestCommitsSince_ClosedDB(t *testing.T) {
	s, _ := tempDB(t)
	_ = s.Close()

	_, err := s.CommitsSince(time.Now().Add(-24 * time.Hour))
	if err == nil {
		t.Error("expected error querying closed DB")
	}
}

func TestUndocumentedCommits_ClosedDB(t *testing.T) {
	s, _ := tempDB(t)
	_ = s.Close()

	_, err := s.UndocumentedCommits()
	if err == nil {
		t.Error("expected error querying closed DB")
	}
}

func TestSearchDocs_ClosedDB(t *testing.T) {
	s, _ := tempDB(t)
	_ = s.Close()

	_, err := s.SearchDocs(context.Background(), "test")
	if err == nil {
		t.Error("expected error from SearchDocs on closed DB")
	}
}

func TestReservedTables_ConstraintsEnforced(t *testing.T) {
	s, _ := tempDB(t)

	// code_signatures requires valid entity_type
	_, err := s.db.Exec(`INSERT INTO code_signatures (commit_hash, file_path, entity_name, entity_type, sig_hash, lang)
		VALUES ('nonexistent', 'file.go', 'Foo', 'invalid_type', 'hash', 'go')`)
	if err == nil {
		t.Error("code_signatures should reject invalid entity_type")
	}

	// commit_relations requires valid relation
	_, err = s.db.Exec(`INSERT INTO commit_relations (source_hash, target_hash, relation, detected_at)
		VALUES ('a', 'b', 'invalid_relation', 1234)`)
	if err == nil {
		t.Error("commit_relations should reject invalid relation")
	}

	// doc_edges requires valid edge_type
	_, err = s.db.Exec(`INSERT INTO doc_edges (source_doc, target_doc, edge_type, detected_at)
		VALUES ('a.md', 'b.md', 'invalid_type', 1234)`)
	if err == nil {
		t.Error("doc_edges should reject invalid edge_type")
	}
}

func TestStubs_DoNotPanic(t *testing.T) {
	s, _ := tempDB(t)

	// All stubs should return domain.ErrNotImplemented
	if err := s.StoreSignatures("abc", nil); err != domain.ErrNotImplemented {
		t.Errorf("StoreSignatures: got %v, want ErrNotImplemented", err)
	}
	if _, err := s.FindBySignatureHash("abc"); err != domain.ErrNotImplemented {
		t.Errorf("FindBySignatureHash: got %v, want ErrNotImplemented", err)
	}
	if err := s.RecordAIUsage(domain.AIUsageRecord{}); err != domain.ErrNotImplemented {
		t.Errorf("RecordAIUsage: got %v, want ErrNotImplemented", err)
	}
	if err := s.CacheReview(domain.ReviewCacheEntry{}); err != domain.ErrNotImplemented {
		t.Errorf("CacheReview: got %v, want ErrNotImplemented", err)
	}
	if err := s.UpdatePattern("feat", "auth", "documented", 10, 80); err != domain.ErrNotImplemented {
		t.Errorf("UpdatePattern: got %v, want ErrNotImplemented", err)
	}
}

func TestMigrate_FreshDB_LatestVersion(t *testing.T) {
	s, _ := tempDB(t)

	var version int
	err := s.db.QueryRow("SELECT MAX(version) FROM schema_version").Scan(&version)
	if err != nil {
		t.Fatalf("query schema_version: %v", err)
	}
	// migrations slice has 2 entries (v1, v2), so latest should be 2.
	if version != len(migrations) {
		t.Errorf("schema version = %d, want %d (latest)", version, len(migrations))
	}

	// Verify each migration has a non-empty description and plausible applied_at.
	rows, err := s.db.Query("SELECT version, applied_at, description FROM schema_version ORDER BY version")
	if err != nil {
		t.Fatalf("query rows: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var v int
		var at int64
		var desc string
		if err := rows.Scan(&v, &at, &desc); err != nil {
			t.Fatalf("scan: %v", err)
		}
		count++
		if desc == "" {
			t.Errorf("migration v%d has empty description", v)
		}
		if at <= 0 {
			t.Errorf("migration v%d has invalid applied_at: %d", v, at)
		}
	}
	if count != len(migrations) {
		t.Errorf("schema_version row count = %d, want %d", count, len(migrations))
	}
}

func TestMigrate_Idempotent_ThreeCalls(t *testing.T) {
	s, _ := tempDB(t)
	// Open already called Migrate once. Call two more times.
	for i := 0; i < 2; i++ {
		if err := s.Migrate(); err != nil {
			t.Fatalf("Migrate call %d: %v", i+2, err)
		}
	}

	// Version should still be at latest, no duplicates.
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM schema_version").Scan(&count)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != len(migrations) {
		t.Errorf("schema_version rows = %d after 3 Migrate calls, want %d", count, len(migrations))
	}
}

func TestMigrate_V2IndexExists(t *testing.T) {
	s, _ := tempDB(t)

	// Verify the V2 composite index was created.
	var indexName string
	err := s.db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type='index' AND name='idx_commits_scope_date'",
	).Scan(&indexName)
	if err != nil {
		t.Fatalf("idx_commits_scope_date not found: %v", err)
	}
	if indexName != "idx_commits_scope_date" {
		t.Errorf("index name = %q, want idx_commits_scope_date", indexName)
	}
}
