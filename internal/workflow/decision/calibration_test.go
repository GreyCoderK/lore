// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package decision

import (
	"context"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/domain"
)

// mockStoreForCalibration implements the subset of LoreStore needed for calibration.
type mockStoreForCalibration struct {
	counts  map[string]int
	commits []domain.CommitRecord
}

func (m *mockStoreForCalibration) CommitCountByDecision() (map[string]int, error) {
	return m.counts, nil
}
func (m *mockStoreForCalibration) CommitsSince(time.Time) ([]domain.CommitRecord, error) {
	return m.commits, nil
}

// Stubs for remaining LoreStore methods.
func (m *mockStoreForCalibration) RecordCommit(domain.CommitRecord) error                { return nil }
func (m *mockStoreForCalibration) GetCommit(string) (*domain.CommitRecord, error)        { return nil, nil }
func (m *mockStoreForCalibration) CommitsByScope(string, int) ([]domain.CommitRecord, error) {
	return nil, nil
}
func (m *mockStoreForCalibration) CommitsByBranch(string) ([]domain.CommitRecord, error) {
	return nil, nil
}
func (m *mockStoreForCalibration) UndocumentedCommits() ([]domain.CommitRecord, error) {
	return nil, nil
}
func (m *mockStoreForCalibration) StoreSignatures(string, []domain.CodeSignature) error { return nil }
func (m *mockStoreForCalibration) FindBySignatureHash(string) ([]domain.CodeSignature, error) {
	return nil, nil
}
func (m *mockStoreForCalibration) SignaturesForCommit(string) ([]domain.CodeSignature, error) {
	return nil, nil
}
func (m *mockStoreForCalibration) EntityHistory(string, string) ([]domain.CodeSignature, error) {
	return nil, nil
}
func (m *mockStoreForCalibration) IndexDoc(domain.DocIndexEntry) error             { return nil }
func (m *mockStoreForCalibration) RemoveDoc(string) error                          { return nil }
func (m *mockStoreForCalibration) GetDoc(string) (*domain.DocIndexEntry, error)    { return nil, nil }
func (m *mockStoreForCalibration) DocsByScope(string) ([]domain.DocIndexEntry, error) {
	return nil, nil
}
func (m *mockStoreForCalibration) DocsByBranch(string) ([]domain.DocIndexEntry, error) {
	return nil, nil
}
func (m *mockStoreForCalibration) DocsByType(string) ([]domain.DocIndexEntry, error) {
	return nil, nil
}
func (m *mockStoreForCalibration) UnconsolidatedDocs(string) ([]domain.DocIndexEntry, error) {
	return nil, nil
}
func (m *mockStoreForCalibration) AllDocSummaries(int) ([]domain.DocIndexEntry, error) {
	return nil, nil
}
func (m *mockStoreForCalibration) DocsByCommitHash(string) ([]domain.DocIndexEntry, error) {
	return nil, nil
}
func (m *mockStoreForCalibration) SearchDocs(context.Context, string) ([]domain.DocIndexEntry, error) {
	return nil, nil
}
func (m *mockStoreForCalibration) DocCount() (int, error) { return 0, nil }
func (m *mockStoreForCalibration) RecordAIUsage(domain.AIUsageRecord) error {
	return nil
}
func (m *mockStoreForCalibration) AIStatsSince(time.Time) (*domain.AIStatsAggregate, error) {
	return nil, nil
}
func (m *mockStoreForCalibration) AIStatsByDay(int) ([]domain.DailyAIStats, error) {
	return nil, nil
}
func (m *mockStoreForCalibration) CacheReview(domain.ReviewCacheEntry) error          { return nil }
func (m *mockStoreForCalibration) GetCachedReview(string) (*domain.ReviewCacheEntry, error) {
	return nil, nil
}
func (m *mockStoreForCalibration) ReviewHistory(int) ([]domain.ReviewCacheEntry, error) {
	return nil, nil
}
func (m *mockStoreForCalibration) UpdatePattern(string, string, string, int, int) error { return nil }
func (m *mockStoreForCalibration) GetPattern(string, string) (*domain.CommitPattern, error) {
	return nil, nil
}
func (m *mockStoreForCalibration) AllPatterns() ([]domain.CommitPattern, error) { return nil, nil }
func (m *mockStoreForCalibration) Rebuild(context.Context, string, domain.GitAdapter) error {
	return nil
}
func (m *mockStoreForCalibration) Vacuum() error { return nil }
func (m *mockStoreForCalibration) Close() error  { return nil }
func (m *mockStoreForCalibration) ScopeStats(string, int) (domain.ScopeStatsResult, error) {
	return domain.ScopeStatsResult{}, nil
}

func TestComputeCalibration_NilStore(t *testing.T) {
	_, err := ComputeCalibration(nil)
	if err == nil {
		t.Fatal("expected error for nil store")
	}
}

func TestComputeCalibration_KnownDecisions(t *testing.T) {
	// Setup: 20 commits total
	// 10 auto-skipped, 1 false negative (auto-skipped then manually documented)
	// 5 ask-full, 1 of which was skipped (false positive)
	// 3 ask-reduced
	// 1 suggest-skipped
	commits := []domain.CommitRecord{
		// 10 auto-skipped
		{Hash: "a1", Decision: "auto-skipped", QuestionMode: "none"},
		{Hash: "a2", Decision: "auto-skipped", QuestionMode: "none"},
		{Hash: "a3", Decision: "auto-skipped", QuestionMode: "none"},
		{Hash: "a4", Decision: "auto-skipped", QuestionMode: "none"},
		{Hash: "a5", Decision: "auto-skipped", QuestionMode: "none"},
		{Hash: "a6", Decision: "auto-skipped", QuestionMode: "none"},
		{Hash: "a7", Decision: "auto-skipped", QuestionMode: "none"},
		{Hash: "a8", Decision: "auto-skipped", QuestionMode: "none"},
		{Hash: "a9", Decision: "auto-skipped", QuestionMode: "none"},
		{Hash: "a10", Decision: "auto-skipped", QuestionMode: "none"},
		// 1 false negative: was auto-skipped, later resolved → documented with mode=none
		{Hash: "fn1", Decision: "documented", QuestionMode: "none"},
		// 5 ask-full: 4 documented + 1 skipped
		{Hash: "f1", Decision: "documented", QuestionMode: "full"},
		{Hash: "f2", Decision: "documented", QuestionMode: "full"},
		{Hash: "f3", Decision: "documented", QuestionMode: "full"},
		{Hash: "f4", Decision: "documented", QuestionMode: "full"},
		{Hash: "f5", Decision: "skipped", QuestionMode: "full"},
		// 3 ask-reduced: all documented
		{Hash: "r1", Decision: "documented", QuestionMode: "reduced"},
		{Hash: "r2", Decision: "documented", QuestionMode: "reduced"},
		{Hash: "r3", Decision: "documented", QuestionMode: "reduced"},
		// 1 suggest-skipped
		{Hash: "ss1", Decision: "skipped", QuestionMode: ""},
	}

	store := &mockStoreForCalibration{
		counts: map[string]int{
			"auto-skipped": 10,
			"documented":   8,
			"skipped":      2,
		},
		commits: commits,
	}

	r, err := ComputeCalibration(store)
	if err != nil {
		t.Fatalf("ComputeCalibration: %v", err)
	}

	if r.TotalCommits != 20 {
		t.Errorf("TotalCommits = %d, want 20", r.TotalCommits)
	}
	if r.AutoSkipped != 10 {
		t.Errorf("AutoSkipped = %d, want 10", r.AutoSkipped)
	}
	if r.AskFull != 5 {
		t.Errorf("AskFull = %d, want 5", r.AskFull)
	}
	if r.AskReduced != 3 {
		t.Errorf("AskReduced = %d, want 3", r.AskReduced)
	}

	// False negative: 1 / (10 + 1) ≈ 9.09%
	wantFN := 1.0 / 11.0
	if diff := r.FalseNegativeRate - wantFN; diff > 0.01 || diff < -0.01 {
		t.Errorf("FalseNegativeRate = %.4f, want %.4f", r.FalseNegativeRate, wantFN)
	}

	// False positive: 1 / 5 = 20%
	if r.FalsePositiveRate != 0.2 {
		t.Errorf("FalsePositiveRate = %.4f, want 0.2", r.FalsePositiveRate)
	}

	// Ask-full doc rate: 4 / 5 = 80%
	if r.AskFullDocRate != 0.8 {
		t.Errorf("AskFullDocRate = %.4f, want 0.8", r.AskFullDocRate)
	}

	// Auto-skip rate: 10 / 20 = 50%
	if r.AutoSkipRate != 0.5 {
		t.Errorf("AutoSkipRate = %.4f, want 0.5", r.AutoSkipRate)
	}
}

func TestComputeCalibration_EmptyStore(t *testing.T) {
	store := &mockStoreForCalibration{
		counts:  map[string]int{},
		commits: nil,
	}

	r, err := ComputeCalibration(store)
	if err != nil {
		t.Fatalf("ComputeCalibration: %v", err)
	}

	if r.TotalCommits != 0 {
		t.Errorf("TotalCommits = %d, want 0", r.TotalCommits)
	}
	if r.FalseNegativeRate != 0 {
		t.Errorf("FalseNegativeRate = %f, want 0", r.FalseNegativeRate)
	}
}

func TestFormatCalibration(t *testing.T) {
	r := &CalibrationReport{
		TotalCommits:      100,
		AutoSkipped:       40,
		FalseNegativeRate: 0.03,
		FalsePositiveRate: 0.15,
		AskFullDocRate:    0.85,
		AutoSkipRate:      0.40,
	}
	out := FormatCalibration(r)
	if out == "" {
		t.Error("FormatCalibration returned empty string")
	}
}
