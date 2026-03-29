// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package decision

import (
	"context"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/domain"
)

// mockLoreStore implements domain.LoreStore for testing Signal 5.
type mockLoreStore struct {
	commits       []domain.CommitRecord
	commitCounts  map[string]int
	scopeCommits  map[string][]domain.CommitRecord
}

func (m *mockLoreStore) RecordCommit(domain.CommitRecord) error                          { return nil }
func (m *mockLoreStore) GetCommit(string) (*domain.CommitRecord, error)                  { return nil, nil }
func (m *mockLoreStore) CommitsByBranch(string) ([]domain.CommitRecord, error)            { return nil, nil }
func (m *mockLoreStore) CommitsSince(time.Time) ([]domain.CommitRecord, error)            { return nil, nil }
func (m *mockLoreStore) UndocumentedCommits() ([]domain.CommitRecord, error)              { return nil, nil }
func (m *mockLoreStore) StoreSignatures(string, []domain.CodeSignature) error             { return nil }
func (m *mockLoreStore) FindBySignatureHash(string) ([]domain.CodeSignature, error)       { return nil, nil }
func (m *mockLoreStore) SignaturesForCommit(string) ([]domain.CodeSignature, error)       { return nil, nil }
func (m *mockLoreStore) EntityHistory(string, string) ([]domain.CodeSignature, error)     { return nil, nil }
func (m *mockLoreStore) IndexDoc(domain.DocIndexEntry) error                              { return nil }
func (m *mockLoreStore) RemoveDoc(string) error                                           { return nil }
func (m *mockLoreStore) GetDoc(string) (*domain.DocIndexEntry, error)                     { return nil, nil }
func (m *mockLoreStore) DocsByScope(string) ([]domain.DocIndexEntry, error)               { return nil, nil }
func (m *mockLoreStore) DocsByBranch(string) ([]domain.DocIndexEntry, error)              { return nil, nil }
func (m *mockLoreStore) DocsByType(string) ([]domain.DocIndexEntry, error)                { return nil, nil }
func (m *mockLoreStore) UnconsolidatedDocs(string) ([]domain.DocIndexEntry, error)        { return nil, nil }
func (m *mockLoreStore) AllDocSummaries(int) ([]domain.DocIndexEntry, error)              { return nil, nil }
func (m *mockLoreStore) DocsByCommitHash(string) ([]domain.DocIndexEntry, error)          { return nil, nil }
func (m *mockLoreStore) SearchDocs(context.Context, string) ([]domain.DocIndexEntry, error) { return nil, nil }
func (m *mockLoreStore) DocCount() (int, error)                                           { return 0, nil }
func (m *mockLoreStore) RecordAIUsage(domain.AIUsageRecord) error                         { return nil }
func (m *mockLoreStore) AIStatsSince(time.Time) (*domain.AIStatsAggregate, error)         { return nil, nil }
func (m *mockLoreStore) AIStatsByDay(int) ([]domain.DailyAIStats, error)                  { return nil, nil }
func (m *mockLoreStore) CacheReview(domain.ReviewCacheEntry) error                        { return nil }
func (m *mockLoreStore) GetCachedReview(string) (*domain.ReviewCacheEntry, error)         { return nil, nil }
func (m *mockLoreStore) ReviewHistory(int) ([]domain.ReviewCacheEntry, error)             { return nil, nil }
func (m *mockLoreStore) UpdatePattern(string, string, string, int, int) error             { return nil }
func (m *mockLoreStore) GetPattern(string, string) (*domain.CommitPattern, error)         { return nil, nil }
func (m *mockLoreStore) AllPatterns() ([]domain.CommitPattern, error)                     { return nil, nil }
func (m *mockLoreStore) Rebuild(context.Context, string, domain.GitAdapter) error         { return nil }
func (m *mockLoreStore) Vacuum() error                                                    { return nil }
func (m *mockLoreStore) Close() error                                                     { return nil }

func (m *mockLoreStore) CommitsByScope(scope string, days int) ([]domain.CommitRecord, error) {
	if m.scopeCommits != nil {
		return m.scopeCommits[scope], nil
	}
	return nil, nil
}

func (m *mockLoreStore) ScopeStats(scope string, days int) (domain.ScopeStatsResult, error) {
	commits := m.scopeCommits[scope]
	var result domain.ScopeStatsResult
	result.TotalCommits = len(commits)
	for _, c := range commits {
		switch c.Decision {
		case "documented":
			result.DocumentedCount++
			if c.Date.Unix() > result.LastDocDate {
				result.LastDocDate = c.Date.Unix()
			}
		case "skipped", "auto-skipped":
			result.SkippedCount++
		}
		if c.Date.Unix() > result.LastCommitDate {
			result.LastCommitDate = c.Date.Unix()
		}
	}
	return result, nil
}

func (m *mockLoreStore) CommitCountByDecision() (map[string]int, error) {
	if m.commitCounts != nil {
		return m.commitCounts, nil
	}
	return map[string]int{}, nil
}

// --- Tests ---

func TestScoreLKSHistory_NilStore(t *testing.T) {
	s := scoreLKSHistory(nil, "auth", DefaultConfig(), time.Now)
	if s.Score != 0 {
		t.Errorf("nil store: score = %d, want 0", s.Score)
	}
	if s.Reason != "store unavailable" {
		t.Errorf("reason = %q, want 'store unavailable'", s.Reason)
	}
}

func TestScoreLKSHistory_ScopeNeverDocumented(t *testing.T) {
	store := &mockLoreStore{
		commitCounts: map[string]int{"documented": 30}, // past warm-up
	}
	s := scoreLKSHistory(store, "auth", DefaultConfig(), time.Now)
	if s.Score != 15 {
		t.Errorf("never documented: score = %d, want 15", s.Score)
	}
}

func TestScoreLKSHistory_RecentDoc_LessThan7Days(t *testing.T) {
	store := &mockLoreStore{
		scopeCommits: map[string][]domain.CommitRecord{
			"auth": {
				{Decision: "documented", Date: time.Now().Add(-3 * 24 * time.Hour)}, // 3 days ago
			},
		},
		commitCounts: map[string]int{"documented": 30},
	}
	s := scoreLKSHistory(store, "auth", DefaultConfig(), time.Now)
	if s.Score != -10 {
		t.Errorf("recent doc <7d: score = %d, want -10", s.Score)
	}
}

func TestScoreLKSHistory_OldDoc_MoreThan30Days(t *testing.T) {
	store := &mockLoreStore{
		scopeCommits: map[string][]domain.CommitRecord{
			"auth": {
				{Decision: "documented", Date: time.Now().Add(-60 * 24 * time.Hour)}, // 60 days ago
			},
		},
		commitCounts: map[string]int{"documented": 30},
	}
	s := scoreLKSHistory(store, "auth", DefaultConfig(), time.Now)
	if s.Score != 10 {
		t.Errorf("old doc >30d: score = %d, want 10", s.Score)
	}
}

func TestScoreLKSHistory_CriticalScope(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CriticalScopes = []string{"auth", "security"}
	store := &mockLoreStore{
		commitCounts: map[string]int{"documented": 30},
	}
	s := scoreLKSHistory(store, "auth", cfg, time.Now)
	// critical(+20) + never documented(+15) = 35
	if s.Score < 20 {
		t.Errorf("critical scope: score = %d, want >= 20", s.Score)
	}
}

func TestScoreLKSHistory_HighSkipRate(t *testing.T) {
	store := &mockLoreStore{
		scopeCommits: map[string][]domain.CommitRecord{
			"auth": {
				{Decision: "documented", Date: time.Now().Add(-15 * 24 * time.Hour)},
				{Decision: "skipped"},
				{Decision: "skipped"},
				{Decision: "skipped"},
				{Decision: "skipped"},
				{Decision: "auto-skipped"},
				{Decision: "auto-skipped"},
				{Decision: "auto-skipped"},
				{Decision: "auto-skipped"},
				{Decision: "auto-skipped"},
			},
		},
		commitCounts: map[string]int{"documented": 30, "skipped": 20},
	}
	s := scoreLKSHistory(store, "auth", DefaultConfig(), time.Now)
	// skip rate = 9/10 = 90% > 80% → -5. Doc 15 days ago (between 7 and 30) → 0
	if s.Score != -5 {
		t.Errorf("high skip rate: score = %d, want -5", s.Score)
	}
}

func TestScoreLKSHistory_WarmupHalving(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CriticalScopes = []string{"auth"}
	store := &mockLoreStore{
		commitCounts: map[string]int{"documented": 5}, // < 20 warm-up
	}
	s := scoreLKSHistory(store, "auth", cfg, time.Now)
	// critical(+20) + never documented(+15) = 35 → halved = 17
	if s.Score != 17 {
		t.Errorf("warm-up halved: score = %d, want 17", s.Score)
	}
}
