// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package store

import (
	"os"
	"path/filepath"
	"testing"

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
