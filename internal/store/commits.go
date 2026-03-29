// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/greycoderk/lore/internal/domain"
)

// commitColumns is the canonical column list for the commits table.
// SYNC: update when adding columns to the commits schema.
const commitColumns = `hash, date, branch, scope, conv_type, subject, message,
		files_changed, lines_added, lines_deleted, doc_id,
		decision, decision_score, decision_confidence, skip_reason, question_mode`

// RecordCommit inserts or replaces a commit record.
func (s *SQLiteStore) RecordCommit(rec domain.CommitRecord) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO commits
		(hash, date, branch, scope, conv_type, subject, message,
		 files_changed, lines_added, lines_deleted, doc_id,
		 decision, decision_score, decision_confidence, skip_reason, question_mode)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		rec.Hash, rec.Date.Unix(), rec.Branch, rec.Scope, rec.ConvType,
		rec.Subject, rec.Message, rec.FilesChanged, rec.LinesAdded, rec.LinesDeleted,
		nullStr(rec.DocID), rec.Decision,
		rec.DecisionScore, rec.DecisionConfidence,
		nullStr(rec.SkipReason), rec.QuestionMode,
	)
	if err != nil {
		return fmt.Errorf("store: record commit: %w", err)
	}
	return nil
}

// GetCommit returns a commit by hash, or nil if not found.
func (s *SQLiteStore) GetCommit(hash string) (*domain.CommitRecord, error) {
	row := s.db.QueryRow(`SELECT `+commitColumns+`
		FROM commits WHERE hash = ?`, hash)
	return scanCommit(row)
}

// CommitsByScope returns commits matching scope within the last N days.
func (s *SQLiteStore) CommitsByScope(scope string, days int) ([]domain.CommitRecord, error) {
	since := time.Now().AddDate(0, 0, -days).Unix()
	rows, err := s.db.Query(`SELECT `+commitColumns+`
		FROM commits WHERE scope = ? AND date >= ? ORDER BY date DESC LIMIT 10000`, scope, since)
	if err != nil {
		return nil, fmt.Errorf("store: commits by scope: %w", err)
	}
	defer rows.Close()
	return scanCommits(rows)
}

// CommitsByBranch returns commits on a given branch.
func (s *SQLiteStore) CommitsByBranch(branch string) ([]domain.CommitRecord, error) {
	rows, err := s.db.Query(`SELECT `+commitColumns+`
		FROM commits WHERE branch = ? ORDER BY date DESC`, branch)
	if err != nil {
		return nil, fmt.Errorf("store: commits by branch: %w", err)
	}
	defer rows.Close()
	return scanCommits(rows)
}

// CommitsSince returns commits after a given time.
func (s *SQLiteStore) CommitsSince(since time.Time) ([]domain.CommitRecord, error) {
	rows, err := s.db.Query(`SELECT `+commitColumns+`
		FROM commits WHERE date >= ? ORDER BY date DESC LIMIT 10000`, since.Unix())
	if err != nil {
		return nil, fmt.Errorf("store: commits since: %w", err)
	}
	defer rows.Close()
	return scanCommits(rows)
}

// UndocumentedCommits returns commits with decision pending or unknown.
func (s *SQLiteStore) UndocumentedCommits() ([]domain.CommitRecord, error) {
	rows, err := s.db.Query(`SELECT `+commitColumns+`
		FROM commits WHERE decision IN ('pending', 'unknown') ORDER BY date DESC LIMIT 10000`)
	if err != nil {
		return nil, fmt.Errorf("store: undocumented commits: %w", err)
	}
	defer rows.Close()
	return scanCommits(rows)
}

// CommitCountByDecision returns a map of decision → count.
func (s *SQLiteStore) CommitCountByDecision() (map[string]int, error) {
	rows, err := s.db.Query(`SELECT decision, COUNT(*) FROM commits GROUP BY decision`)
	if err != nil {
		return nil, fmt.Errorf("store: commit count by decision: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var decision string
		var count int
		if err := rows.Scan(&decision, &count); err != nil {
			return nil, err
		}
		counts[decision] = count
	}
	return counts, rows.Err()
}

// ScopeStats returns aggregated statistics for a scope, computing in SQL
// instead of loading all records into memory.
func (s *SQLiteStore) ScopeStats(scope string, days int) (domain.ScopeStatsResult, error) {
	since := time.Now().AddDate(0, 0, -days).Unix()
	var result domain.ScopeStatsResult
	err := s.db.QueryRow(`
		SELECT
			COUNT(*) AS total,
			COUNT(CASE WHEN decision = 'documented' THEN 1 END) AS documented,
			COUNT(CASE WHEN decision IN ('skipped', 'auto-skipped') THEN 1 END) AS skipped,
			COALESCE(MAX(CASE WHEN decision = 'documented' THEN date END), 0) AS last_doc_date,
			COALESCE(MAX(date), 0) AS last_commit_date
		FROM commits
		WHERE scope = ? AND date >= ?
	`, scope, since).Scan(
		&result.TotalCommits,
		&result.DocumentedCount,
		&result.SkippedCount,
		&result.LastDocDate,
		&result.LastCommitDate,
	)
	if err != nil {
		return domain.ScopeStatsResult{}, fmt.Errorf("store: scope stats: %w", err)
	}
	return result, nil
}

// --- scan helpers ---

// scanner is satisfied by both *sql.Row and the per-row interface of *sql.Rows.
type scanner interface {
	Scan(dest ...interface{}) error
}

// scanCommitRow scans a single commit row into a CommitRecord.
// Shared by both scanCommit (single row) and scanCommits (multi row).
func scanCommitRow(s scanner) (domain.CommitRecord, error) {
	var rec domain.CommitRecord
	var dateUnix int64
	var docID, skipReason sql.NullString
	var score sql.NullInt64
	var confidence sql.NullFloat64

	if err := s.Scan(&rec.Hash, &dateUnix, &rec.Branch, &rec.Scope, &rec.ConvType,
		&rec.Subject, &rec.Message, &rec.FilesChanged, &rec.LinesAdded, &rec.LinesDeleted,
		&docID, &rec.Decision, &score, &confidence, &skipReason, &rec.QuestionMode); err != nil {
		return rec, err
	}

	rec.Date = time.Unix(dateUnix, 0)
	rec.DocID = docID.String
	rec.SkipReason = skipReason.String
	if score.Valid {
		rec.DecisionScore = int(score.Int64)
	}
	if confidence.Valid {
		rec.DecisionConfidence = confidence.Float64
	}
	return rec, nil
}

func scanCommit(row *sql.Row) (*domain.CommitRecord, error) {
	rec, err := scanCommitRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: scan commit: %w", err)
	}
	return &rec, nil
}

func scanCommits(rows *sql.Rows) ([]domain.CommitRecord, error) {
	results := make([]domain.CommitRecord, 0, 64)
	for rows.Next() {
		rec, err := scanCommitRow(rows)
		if err != nil {
			return nil, fmt.Errorf("store: scan commits: %w", err)
		}
		results = append(results, rec)
	}
	return results, rows.Err()
}

// --- nullable helpers ---

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
