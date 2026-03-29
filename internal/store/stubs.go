// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package store

import (
	"context"
	"time"

	"github.com/greycoderk/lore/internal/domain"
)

// --- Code Signatures stubs (Phase 1) ---
// These return ErrNotImplemented to prevent silent data loss.

func (s *SQLiteStore) StoreSignatures(commitHash string, sigs []domain.CodeSignature) error {
	return domain.ErrNotImplemented
}

func (s *SQLiteStore) FindBySignatureHash(sigHash string) ([]domain.CodeSignature, error) {
	return nil, domain.ErrNotImplemented
}

func (s *SQLiteStore) SignaturesForCommit(commitHash string) ([]domain.CodeSignature, error) {
	return nil, domain.ErrNotImplemented
}

func (s *SQLiteStore) EntityHistory(entityName, lang string) ([]domain.CodeSignature, error) {
	return nil, domain.ErrNotImplemented
}

// --- AI Usage stubs (Phase 1) ---

func (s *SQLiteStore) RecordAIUsage(usage domain.AIUsageRecord) error {
	return domain.ErrNotImplemented
}

func (s *SQLiteStore) AIStatsSince(since time.Time) (*domain.AIStatsAggregate, error) {
	return nil, domain.ErrNotImplemented
}

func (s *SQLiteStore) AIStatsByDay(days int) ([]domain.DailyAIStats, error) {
	return nil, domain.ErrNotImplemented
}

// --- Review Cache stubs (Phase 1) ---

func (s *SQLiteStore) CacheReview(report domain.ReviewCacheEntry) error {
	return domain.ErrNotImplemented
}

func (s *SQLiteStore) GetCachedReview(corpusHash string) (*domain.ReviewCacheEntry, error) {
	return nil, domain.ErrNotImplemented
}

func (s *SQLiteStore) ReviewHistory(limit int) ([]domain.ReviewCacheEntry, error) {
	return nil, domain.ErrNotImplemented
}

// --- Commit Patterns stubs (Phase 2) ---

func (s *SQLiteStore) UpdatePattern(convType, scope string, decision string, diffLines, score int) error {
	return domain.ErrNotImplemented
}

func (s *SQLiteStore) GetPattern(convType, scope string) (*domain.CommitPattern, error) {
	return nil, domain.ErrNotImplemented
}

func (s *SQLiteStore) AllPatterns() ([]domain.CommitPattern, error) {
	return nil, domain.ErrNotImplemented
}

// --- Maintenance ---

func (s *SQLiteStore) Rebuild(ctx context.Context, docsDir string, git domain.GitAdapter) error {
	_, _, _, err := s.RebuildFromSources(ctx, docsDir, git)
	return err
}

func (s *SQLiteStore) Vacuum() error {
	_, err := s.db.Exec("VACUUM")
	return err
}
