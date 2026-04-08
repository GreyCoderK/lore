// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"context"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/domain"
)

// mockLoreStore captures RecordCommit calls for verification.
type mockLoreStore struct {
	recorded []domain.CommitRecord
	err      error
}

func (m *mockLoreStore) RecordCommit(rec domain.CommitRecord) error {
	m.recorded = append(m.recorded, rec)
	return m.err
}

// Stubs for remaining LoreStore interface methods.
func (m *mockLoreStore) GetCommit(_ string) (*domain.CommitRecord, error) { return nil, nil }
func (m *mockLoreStore) CommitsByScope(_ string, _ int) ([]domain.CommitRecord, error) {
	return nil, nil
}
func (m *mockLoreStore) CommitsByBranch(_ string) ([]domain.CommitRecord, error) { return nil, nil }
func (m *mockLoreStore) CommitsSince(_ time.Time) ([]domain.CommitRecord, error) { return nil, nil }
func (m *mockLoreStore) UndocumentedCommits() ([]domain.CommitRecord, error)     { return nil, nil }
func (m *mockLoreStore) CommitCountByDecision() (map[string]int, error)          { return nil, nil }
func (m *mockLoreStore) ScopeStats(_ string, _ int) (domain.ScopeStatsResult, error) {
	return domain.ScopeStatsResult{}, nil
}
func (m *mockLoreStore) IndexDoc(_ domain.DocIndexEntry) error                { return nil }
func (m *mockLoreStore) RemoveDoc(_ string) error                             { return nil }
func (m *mockLoreStore) GetDoc(_ string) (*domain.DocIndexEntry, error)       { return nil, nil }
func (m *mockLoreStore) DocsByScope(_ string) ([]domain.DocIndexEntry, error) { return nil, nil }
func (m *mockLoreStore) DocsByBranch(_ string) ([]domain.DocIndexEntry, error) {
	return nil, nil
}
func (m *mockLoreStore) DocsByType(_ string) ([]domain.DocIndexEntry, error) { return nil, nil }
func (m *mockLoreStore) UnconsolidatedDocs(_ string) ([]domain.DocIndexEntry, error) {
	return nil, nil
}
func (m *mockLoreStore) AllDocSummaries(_ int) ([]domain.DocIndexEntry, error) { return nil, nil }
func (m *mockLoreStore) DocsByCommitHash(_ string) ([]domain.DocIndexEntry, error) {
	return nil, nil
}
func (m *mockLoreStore) SearchDocs(_ context.Context, _ string) ([]domain.DocIndexEntry, error) {
	return nil, nil
}
func (m *mockLoreStore) DocCount() (int, error) { return 0, nil }
func (m *mockLoreStore) StoreSignatures(_ string, _ []domain.CodeSignature) error {
	return nil
}
func (m *mockLoreStore) FindBySignatureHash(_ string) ([]domain.CodeSignature, error) {
	return nil, nil
}
func (m *mockLoreStore) SignaturesForCommit(_ string) ([]domain.CodeSignature, error) {
	return nil, nil
}
func (m *mockLoreStore) EntityHistory(_, _ string) ([]domain.CodeSignature, error) {
	return nil, nil
}
func (m *mockLoreStore) RecordAIUsage(_ domain.AIUsageRecord) error { return nil }
func (m *mockLoreStore) AIStatsSince(_ time.Time) (*domain.AIStatsAggregate, error) {
	return nil, nil
}
func (m *mockLoreStore) AIStatsByDay(_ int) ([]domain.DailyAIStats, error) { return nil, nil }
func (m *mockLoreStore) CacheReview(_ domain.ReviewCacheEntry) error       { return nil }
func (m *mockLoreStore) GetCachedReview(_ string) (*domain.ReviewCacheEntry, error) {
	return nil, nil
}
func (m *mockLoreStore) ReviewHistory(_ int) ([]domain.ReviewCacheEntry, error) { return nil, nil }
func (m *mockLoreStore) UpdatePattern(_, _, _ string, _, _ int) error          { return nil }
func (m *mockLoreStore) GetPattern(_, _ string) (*domain.CommitPattern, error) { return nil, nil }
func (m *mockLoreStore) AllPatterns() ([]domain.CommitPattern, error)          { return nil, nil }
func (m *mockLoreStore) Rebuild(_ context.Context, _ string, _ domain.GitAdapter) error {
	return nil
}
func (m *mockLoreStore) Vacuum() error { return nil }
func (m *mockLoreStore) Close() error  { return nil }

func TestRecordDecision_WithStore(t *testing.T) {
	store := &mockLoreStore{}
	commit := &domain.CommitInfo{
		Hash:    "abc123",
		Date:    time.Date(2026, 4, 7, 10, 0, 0, 0, time.UTC),
		Branch:  "main",
		Scope:   "auth",
		Type:    "feat",
		Subject: "add JWT",
		Message: "feat(auth): add JWT",
	}
	detection := DetectionResult{
		Score:        85,
		QuestionMode: "reduced",
		Reason:       "low-complexity",
	}

	recordDecision(store, commit, detection, "documented")

	if len(store.recorded) != 1 {
		t.Fatalf("expected 1 recorded commit, got %d", len(store.recorded))
	}
	rec := store.recorded[0]
	if rec.Hash != "abc123" {
		t.Errorf("Hash = %q, want abc123", rec.Hash)
	}
	if rec.Decision != "documented" {
		t.Errorf("Decision = %q, want documented", rec.Decision)
	}
	if rec.DecisionScore != 85 {
		t.Errorf("DecisionScore = %d, want 85", rec.DecisionScore)
	}
	if rec.QuestionMode != "reduced" {
		t.Errorf("QuestionMode = %q, want reduced", rec.QuestionMode)
	}
	if rec.SkipReason != "low-complexity" {
		t.Errorf("SkipReason = %q, want low-complexity", rec.SkipReason)
	}
	if rec.Branch != "main" {
		t.Errorf("Branch = %q, want main", rec.Branch)
	}
	if rec.Scope != "auth" {
		t.Errorf("Scope = %q, want auth", rec.Scope)
	}
	if rec.ConvType != "feat" {
		t.Errorf("ConvType = %q, want feat", rec.ConvType)
	}
	if rec.Subject != "add JWT" {
		t.Errorf("Subject = %q, want 'add JWT'", rec.Subject)
	}
}

func TestRecordDecision_SkipTypes(t *testing.T) {
	tests := []struct {
		decisionType string
		reason       string
	}{
		{"skipped", "user-declined"},
		{"merge-skipped", "merge"},
		{"auto-skipped", "chore-commit"},
		{"pending", "non-tty"},
	}
	for _, tt := range tests {
		t.Run(tt.decisionType, func(t *testing.T) {
			store := &mockLoreStore{}
			commit := &domain.CommitInfo{
				Hash:    "def456",
				Message: "chore: bump",
				Type:    "chore",
			}
			detection := DetectionResult{Reason: tt.reason}

			recordDecision(store, commit, detection, tt.decisionType)

			if len(store.recorded) != 1 {
				t.Fatalf("expected 1 recorded commit, got %d", len(store.recorded))
			}
			if store.recorded[0].Decision != tt.decisionType {
				t.Errorf("Decision = %q, want %q", store.recorded[0].Decision, tt.decisionType)
			}
			if store.recorded[0].SkipReason != tt.reason {
				t.Errorf("SkipReason = %q, want %q", store.recorded[0].SkipReason, tt.reason)
			}
		})
	}
}

func TestRecordDecision_StoreError_DoesNotPanic(t *testing.T) {
	store := &mockLoreStore{err: context.DeadlineExceeded}
	commit := &domain.CommitInfo{Hash: "xyz"}
	// Should not panic even if store returns error
	recordDecision(store, commit, DetectionResult{}, "documented")
}
