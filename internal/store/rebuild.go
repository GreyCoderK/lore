// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package store

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/storage"
)

// RebuildFromSources reconstructs doc_index from .md files and commits from git log.
// This is the real implementation replacing the stub in stubs.go.
func (s *SQLiteStore) RebuildFromSources(ctx context.Context, docsDir string, git domain.GitAdapter) (int, int, int, error) {
	// Clear existing data for a clean rebuild (within a single transaction)
	tx, err := s.db.Begin()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("store: rebuild: begin tx: %w", err)
	}
	// SECURITY: table names are hardcoded constants below, never user input.
	// Parameterized DELETE is not possible for table names in SQL.
	for _, table := range []string{"doc_index", "commits", "commit_patterns"} {
		if _, err := tx.Exec("DELETE FROM " + table); err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				return 0, 0, 0, fmt.Errorf("store: rebuild: clear %s: %w (rollback failed: %v)", table, err, rbErr)
			}
			return 0, 0, 0, fmt.Errorf("store: rebuild: clear %s: %w", table, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, 0, fmt.Errorf("store: rebuild: commit tx: %w", err)
	}

	docCount, docSkipped, err := s.rebuildDocIndex(ctx, docsDir)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("store: rebuild doc_index: %w", err)
	}

	commitCount := 0
	commitFailed := 0
	if git != nil {
		commitCount, commitFailed, err = s.rebuildCommits(git)
		_ = commitFailed // tracked for observability, not fatal
		if err != nil {
			return docCount, docSkipped, 0, fmt.Errorf("store: rebuild commits: %w", err)
		}
		// Recalculate commit_patterns from reconstructed commits
		if err := s.rebuildPatterns(); err != nil {
			return docCount, docSkipped, commitCount, fmt.Errorf("store: rebuild patterns: %w", err)
		}
	}

	return docCount, docSkipped, commitCount, nil
}

func (s *SQLiteStore) rebuildDocIndex(ctx context.Context, docsDir string) (int, int, error) {
	// Validate docsDir does not contain path traversal sequences
	if strings.Contains(filepath.Clean(docsDir), "..") {
		return 0, 0, fmt.Errorf("rebuild: docsDir must not contain '..', got %q", docsDir)
	}
	entries, err := os.ReadDir(docsDir)
	if err != nil {
		return 0, 0, fmt.Errorf("read docs dir: %w", err)
	}

	indexed := 0
	skipped := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") || entry.Name() == "README.md" {
			continue
		}
		// Skip symlinks to prevent path traversal attacks
		if entry.Type()&os.ModeSymlink != 0 {
			skipped++
			continue
		}

		// Check for context cancellation each iteration
		if err := ctx.Err(); err != nil {
			return indexed, skipped, err
		}

		data, err := os.ReadFile(filepath.Join(docsDir, entry.Name()))
		if err != nil {
			skipped++
			fmt.Fprintf(os.Stderr, "Warning: rebuild: skip %s: read: %v\n", entry.Name(), err)
			continue
		}

		meta, body, err := storage.Unmarshal(data)
		if err != nil {
			skipped++
			fmt.Fprintf(os.Stderr, "Warning: rebuild: skip %s: parse: %v\n", entry.Name(), err)
			continue
		}

		hash := sha256.Sum256([]byte(body))
		title := storage.ExtractTitle(body, entry.Name())

		doc := domain.DocIndexEntry{
			Filename:    entry.Name(),
			Type:        meta.Type,
			Date:        meta.Date,
			CommitHash:  meta.Commit,
			Status:      meta.Status,
			Tags:        meta.Tags,
			Related:     meta.Related,
			GeneratedBy: meta.GeneratedBy,
			AngelaMode:  meta.AngelaMode,
			ContentHash: fmt.Sprintf("%x", hash),
			TitleExtracted: title,
			WordCount:   len(strings.Fields(body)),
			UpdatedAt:   time.Now(),
		}

		if err := s.IndexDoc(doc); err != nil {
			skipped++
			continue
		}
		indexed++
	}

	return indexed, skipped, nil
}

func (s *SQLiteStore) rebuildCommits(git domain.GitAdapter) (int, int, error) {
	commits, err := git.LogAll()
	if err != nil {
		return 0, 0, fmt.Errorf("git log all: %w", err)
	}

	count := 0
	failed := 0
	for _, ci := range commits {
		rec := domain.CommitRecord{
			Hash:         ci.Hash,
			Date:         ci.Date,
			Scope:        ci.Scope,
			ConvType:     ci.Type,
			Subject:      ci.Subject,
			Message:      ci.Message,
			Decision:     "unknown",
			QuestionMode: "none",
		}
		if err := s.RecordCommit(rec); err != nil {
			failed++
			continue
		}
		count++
	}
	return count, failed, nil
}

func (s *SQLiteStore) rebuildPatterns() error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO commit_patterns
		(conv_type, scope, total_count, documented_count, skipped_count, auto_skipped_count,
		 avg_diff_lines, avg_score, last_updated)
		SELECT
			conv_type,
			scope,
			COUNT(*) as total_count,
			SUM(CASE WHEN decision='documented' THEN 1 ELSE 0 END),
			SUM(CASE WHEN decision='skipped' THEN 1 ELSE 0 END),
			SUM(CASE WHEN decision='auto-skipped' THEN 1 ELSE 0 END),
			AVG(files_changed),
			AVG(COALESCE(decision_score, 0)),
			?
		FROM commits
		WHERE conv_type != ''
		GROUP BY conv_type, scope`, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("store: rebuild patterns: %w", err)
	}
	return nil
}
