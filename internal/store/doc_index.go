// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/greycoderk/lore/internal/domain"
)

// docIndexColumns is the canonical column list for the doc_index table.
// SYNC: update when adding columns to the doc_index schema.
const docIndexColumns = `filename, type, date, commit_hash, branch, scope, status,
		tags, related, generated_by, angela_mode, consolidated_into,
		content_hash, summary_why, summary_what, title_extracted, word_count, updated_at`

// IndexDoc inserts or replaces a document index entry.
func (s *SQLiteStore) IndexDoc(entry domain.DocIndexEntry) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO doc_index
		(filename, type, date, commit_hash, branch, scope, status,
		 tags, related, generated_by, angela_mode, consolidated_into,
		 content_hash, summary_why, summary_what, title_extracted, word_count, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		entry.Filename, entry.Type, entry.Date, nullStr(entry.CommitHash),
		entry.Branch, entry.Scope, entry.Status,
		joinTags(entry.Tags), joinTags(entry.Related),
		entry.GeneratedBy, entry.AngelaMode, entry.ConsolidatedInto,
		entry.ContentHash, entry.SummaryWhy, entry.SummaryWhat,
		entry.TitleExtracted, entry.WordCount, entry.UpdatedAt.Unix(),
	)
	if err != nil {
		return fmt.Errorf("store: index doc: %w", err)
	}
	return nil
}

// RemoveDoc deletes a document from the index.
func (s *SQLiteStore) RemoveDoc(filename string) error {
	_, err := s.db.Exec(`DELETE FROM doc_index WHERE filename = ?`, filename)
	if err != nil {
		return fmt.Errorf("store: remove doc: %w", err)
	}
	return nil
}

// GetDoc returns a document by filename, or nil if not found.
func (s *SQLiteStore) GetDoc(filename string) (*domain.DocIndexEntry, error) {
	row := s.db.QueryRow(`SELECT `+docIndexColumns+`
		FROM doc_index WHERE filename = ?`, filename)
	return scanDoc(row)
}

// DocsByScope returns documents matching a scope.
func (s *SQLiteStore) DocsByScope(scope string) ([]domain.DocIndexEntry, error) {
	return s.queryDocs(`SELECT `+docIndexColumns+`
		FROM doc_index WHERE scope = ? ORDER BY date DESC`, scope)
}

// DocsByBranch returns documents from a given branch.
func (s *SQLiteStore) DocsByBranch(branch string) ([]domain.DocIndexEntry, error) {
	return s.queryDocs(`SELECT `+docIndexColumns+`
		FROM doc_index WHERE branch = ? ORDER BY date DESC`, branch)
}

// DocsByType returns documents of a given type.
func (s *SQLiteStore) DocsByType(docType string) ([]domain.DocIndexEntry, error) {
	return s.queryDocs(`SELECT `+docIndexColumns+`
		FROM doc_index WHERE type = ? ORDER BY date DESC`, docType)
}

// UnconsolidatedDocs returns docs not yet consolidated for a scope.
func (s *SQLiteStore) UnconsolidatedDocs(scope string) ([]domain.DocIndexEntry, error) {
	return s.queryDocs(`SELECT `+docIndexColumns+`
		FROM doc_index WHERE scope = ? AND consolidated_into = '' ORDER BY date DESC`, scope)
}

// AllDocSummaries returns up to limit documents ordered by date descending.
func (s *SQLiteStore) AllDocSummaries(limit int) ([]domain.DocIndexEntry, error) {
	return s.queryDocs(`SELECT `+docIndexColumns+`
		FROM doc_index ORDER BY date DESC LIMIT ?`, limit)
}

// DocsByCommitHash returns documents linked to a commit.
func (s *SQLiteStore) DocsByCommitHash(hash string) ([]domain.DocIndexEntry, error) {
	return s.queryDocs(`SELECT `+docIndexColumns+`
		FROM doc_index WHERE commit_hash = ?`, hash)
}

// SearchDocs performs a simple LIKE search across key text fields.
func (s *SQLiteStore) SearchDocs(ctx context.Context, query string) ([]domain.DocIndexEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Escape LIKE wildcards in user query (L1 fix)
	escaped := strings.NewReplacer("%", "\\%", "_", "\\_").Replace(query)
	pattern := "%" + escaped + "%"
	return s.queryDocsCtx(ctx, `SELECT `+docIndexColumns+`
		FROM doc_index WHERE filename LIKE ? ESCAPE '\' OR tags LIKE ? ESCAPE '\'
		OR summary_why LIKE ? ESCAPE '\' OR summary_what LIKE ? ESCAPE '\'
		OR title_extracted LIKE ? ESCAPE '\' ORDER BY date DESC`,
		pattern, pattern, pattern, pattern, pattern)
}

// DocCount returns total number of indexed documents.
func (s *SQLiteStore) DocCount() (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM doc_index`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("store: doc count: %w", err)
	}
	return count, nil
}

// --- helpers ---

func (s *SQLiteStore) queryDocs(query string, args ...interface{}) ([]domain.DocIndexEntry, error) {
	return s.queryDocsCtx(context.Background(), query, args...)
}

func (s *SQLiteStore) queryDocsCtx(ctx context.Context, query string, args ...interface{}) ([]domain.DocIndexEntry, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query docs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanDocs(rows)
}

// scanDocRow scans a single doc_index row. Shared by scanDoc and scanDocs.
func scanDocRow(s scanner) (domain.DocIndexEntry, error) {
	var e domain.DocIndexEntry
	var commitHash sql.NullString
	var tagsRaw, relatedRaw string
	var updatedAtUnix int64

	if err := s.Scan(&e.Filename, &e.Type, &e.Date, &commitHash, &e.Branch, &e.Scope, &e.Status,
		&tagsRaw, &relatedRaw, &e.GeneratedBy, &e.AngelaMode, &e.ConsolidatedInto,
		&e.ContentHash, &e.SummaryWhy, &e.SummaryWhat, &e.TitleExtracted, &e.WordCount, &updatedAtUnix); err != nil {
		return e, err
	}

	e.CommitHash = commitHash.String
	e.Tags = splitTags(tagsRaw)
	e.Related = splitTags(relatedRaw)
	e.UpdatedAt = time.Unix(updatedAtUnix, 0)
	return e, nil
}

func scanDoc(row *sql.Row) (*domain.DocIndexEntry, error) {
	e, err := scanDocRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: scan doc: %w", err)
	}
	return &e, nil
}

func scanDocs(rows *sql.Rows) ([]domain.DocIndexEntry, error) {
	results := make([]domain.DocIndexEntry, 0, 64)
	for rows.Next() {
		e, err := scanDocRow(rows)
		if err != nil {
			return nil, fmt.Errorf("store: scan docs: %w", err)
		}
		results = append(results, e)
	}
	return results, rows.Err()
}

// tagSeparator is the delimiter used for serializing tag/related lists in SQLite.
// Using Unit Separator (U+001F) instead of comma to avoid corruption when
// tags contain commas (e.g. "breaking change, auth").
const tagSeparator = "\x1F"

// joinTags serializes a string slice for storage in SQLite.
func joinTags(tags []string) string {
	return strings.Join(tags, tagSeparator)
}

// splitTags deserializes a tag/related string from SQLite.
// Supports both the new separator (\x1F) and legacy comma-separated values
// for backward compatibility with existing databases.
func splitTags(s string) []string {
	if s == "" {
		return nil
	}
	// Use new separator if present, otherwise fall back to comma (legacy)
	sep := tagSeparator
	if !strings.Contains(s, tagSeparator) {
		sep = ","
	}
	parts := strings.Split(s, sep)
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
