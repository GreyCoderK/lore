// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/domain"
)

func benchStore(b *testing.B) *SQLiteStore {
	b.Helper()
	dir := b.TempDir()
	s, err := Open(filepath.Join(dir, "bench.db"))
	if err != nil {
		b.Fatalf("Open: %v", err)
	}
	b.Cleanup(func() { _ = s.Close() })
	return s
}

func BenchmarkRecordCommit(b *testing.B) {
	s := benchStore(b)
	rec := domain.CommitRecord{
		Date: time.Now(), Branch: "main", Scope: "auth", ConvType: "feat",
		Subject: "add login", Message: "feat(auth): add login",
		FilesChanged: 3, LinesAdded: 50, LinesDeleted: 10,
		Decision: "documented", QuestionMode: "full",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec.Hash = fmt.Sprintf("bench-commit-%d", i)
		if err := s.RecordCommit(rec); err != nil {
			b.Fatalf("RecordCommit: %v", err)
		}
	}
}

func BenchmarkIndexDoc(b *testing.B) {
	s := benchStore(b)
	entry := domain.DocIndexEntry{
		Type: "decision", Date: "2026-03-15", Status: "draft",
		Tags: []string{"auth", "security"}, ContentHash: "abc123",
		WordCount: 150, UpdatedAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry.Filename = fmt.Sprintf("bench-doc-%d.md", i)
		if err := s.IndexDoc(entry); err != nil {
			b.Fatalf("IndexDoc: %v", err)
		}
	}
}

func BenchmarkGetCommit(b *testing.B) {
	s := benchStore(b)
	rec := domain.CommitRecord{
		Hash: "bench-get-commit", Date: time.Now(), Branch: "main", Scope: "auth",
		ConvType: "feat", Subject: "add login", Message: "feat(auth): add login",
		FilesChanged: 3, LinesAdded: 50, LinesDeleted: 10,
		Decision: "documented", QuestionMode: "full",
	}
	if err := s.RecordCommit(rec); err != nil {
		b.Fatalf("RecordCommit: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.GetCommit("bench-get-commit")
		if err != nil {
			b.Fatalf("GetCommit: %v", err)
		}
	}
}

func BenchmarkCommitsByScope_1000Rows(b *testing.B) {
	s := benchStore(b)

	// Seed 1000 commits
	for i := 0; i < 1000; i++ {
		scope := "auth"
		if i%3 == 0 {
			scope = "api"
		}
		rec := domain.CommitRecord{
			Hash: fmt.Sprintf("seed-%d", i), Date: time.Now().Add(-time.Duration(i) * time.Hour),
			Branch: "main", Scope: scope, ConvType: "feat", Subject: "s",
			Message: "m", Decision: "documented", QuestionMode: "full",
		}
		s.RecordCommit(rec)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.CommitsByScope("auth", 365)
		if err != nil {
			b.Fatalf("CommitsByScope: %v", err)
		}
	}
}

func BenchmarkRebuild_50Docs_200Commits(b *testing.B) {
	docsDir := filepath.Join(b.TempDir(), "docs")
	os.MkdirAll(docsDir, 0o755)

	// Create 50 fixture docs
	for i := 0; i < 50; i++ {
		content := fmt.Sprintf("---\ntype: decision\ndate: \"2026-03-%02d\"\nstatus: draft\n---\n# Doc %d\n\nBody content for benchmark document number %d.", (i%28)+1, i, i)
		os.WriteFile(filepath.Join(docsDir, fmt.Sprintf("decision-bench-%d-2026-03-%02d.md", i, (i%28)+1)), []byte(content), 0o644)
	}

	// Mock git with 200 commits
	commits := make([]domain.CommitInfo, 200)
	for i := range commits {
		commits[i] = domain.CommitInfo{
			Hash: fmt.Sprintf("bench-rebuild-%d", i), Date: time.Now(),
			Message: fmt.Sprintf("feat(bench): commit %d", i), Type: "feat", Scope: "bench", Subject: fmt.Sprintf("commit %d", i),
		}
	}
	git := &mockGitForRebuild{commits: commits}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := benchStore(b)
		_, _, _, err := s.RebuildFromSources(context.Background(), docsDir, git)
		if err != nil {
			b.Fatalf("Rebuild: %v", err)
		}
	}
}

func BenchmarkConcurrentReads_WAL(b *testing.B) {
	s := benchStore(b)

	// Seed data
	for i := 0; i < 100; i++ {
		s.RecordCommit(domain.CommitRecord{
			Hash: fmt.Sprintf("wal-%d", i), Date: time.Now(), Branch: "main",
			Scope: "auth", ConvType: "feat", Subject: "s", Message: "m",
			Decision: "documented", QuestionMode: "full",
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		errCh := make(chan error, 2)

		wg.Add(2)
		go func() {
			defer wg.Done()
			_, err := s.CommitsByScope("auth", 365)
			if err != nil {
				errCh <- err
			}
		}()
		go func() {
			defer wg.Done()
			_, err := s.CommitCountByDecision()
			if err != nil {
				errCh <- err
			}
		}()
		wg.Wait()
		close(errCh)

		for err := range errCh {
			b.Fatalf("concurrent read error (SQLITE_BUSY?): %v", err)
		}
	}
}
