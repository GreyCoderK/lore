// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"time"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/greycoderk/lore/internal/workflow/decision"
)

// --- EngineConfigFromApp ---

func TestEngineConfigFromApp_PopulatesFromConfig(t *testing.T) {
	cfg := &config.Config{
		Decision: config.DecisionConfig{
			ThresholdFull:    70,
			ThresholdReduced: 40,
			ThresholdSuggest: 20,
			AlwaysAsk:        []string{"feat"},
			AlwaysSkip:       []string{"ci"},
			CriticalScopes:   []string{"auth"},
			Learning:         true,
			LearningMinCommits: 30,
		},
	}
	ec := EngineConfigFromApp(cfg)
	if ec.ThresholdFull != 70 {
		t.Errorf("ThresholdFull = %d, want 70", ec.ThresholdFull)
	}
	if ec.ThresholdReduced != 40 {
		t.Errorf("ThresholdReduced = %d, want 40", ec.ThresholdReduced)
	}
	if len(ec.AlwaysAsk) != 1 || ec.AlwaysAsk[0] != "feat" {
		t.Errorf("AlwaysAsk = %v, want [feat]", ec.AlwaysAsk)
	}
	if !ec.Learning {
		t.Error("Learning should be true")
	}
}

func TestEngineConfigFromApp_FallsBackToDefault(t *testing.T) {
	cfg := &config.Config{} // all zeros
	ec := EngineConfigFromApp(cfg)
	def := decision.DefaultConfig()
	if ec.ThresholdFull != def.ThresholdFull {
		t.Errorf("ThresholdFull = %d, want default %d", ec.ThresholdFull, def.ThresholdFull)
	}
	if ec.ThresholdReduced != def.ThresholdReduced {
		t.Errorf("ThresholdReduced = %d, want default %d", ec.ThresholdReduced, def.ThresholdReduced)
	}
}

// --- PolishDocument ---

type mockProvider struct {
	response string
	err      error
}

func (m *mockProvider) Complete(_ context.Context, _ string, _ ...domain.Option) (string, error) {
	return m.response, m.err
}

func TestPolishDocument_HappyPath(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, ".lore", "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := "---\ntype: decision\ndate: 2026-04-05\nstatus: draft\n---\n# Test\n\n## Why\nBecause reasons.\n"
	if err := os.WriteFile(filepath.Join(docsDir, "decision-test-2026-04-05.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	provider := &mockProvider{response: content} // returns unchanged
	cfg := &config.Config{}

	result, err := PolishDocument(context.Background(), provider, cfg, docsDir, "decision-test-2026-04-05.md")
	if err != nil {
		t.Fatalf("PolishDocument error: %v", err)
	}
	if result.Filename != "decision-test-2026-04-05.md" {
		t.Errorf("Filename = %q", result.Filename)
	}
	if result.Meta.Type != "decision" {
		t.Errorf("Meta.Type = %q, want decision", result.Meta.Type)
	}
}

func TestPolishDocument_ProviderError(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, ".lore", "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := "---\ntype: decision\ndate: 2026-04-05\nstatus: draft\n---\n# Test\n\n## Why\nBecause.\n"
	if err := os.WriteFile(filepath.Join(docsDir, "decision-test-2026-04-05.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	provider := &mockProvider{err: fmt.Errorf("API down")}
	cfg := &config.Config{}

	_, err := PolishDocument(context.Background(), provider, cfg, docsDir, "decision-test-2026-04-05.md")
	if err == nil {
		t.Fatal("expected error from provider failure")
	}
}

func TestPolishDocument_FileNotFound(t *testing.T) {
	docsDir := t.TempDir()
	provider := &mockProvider{response: "ok"}
	cfg := &config.Config{}

	_, err := PolishDocument(context.Background(), provider, cfg, docsDir, "nonexistent.md")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

// --- ReviewCorpus ---

type mockCorpusReader struct {
	docs []domain.DocMeta
	body string
}

func (m *mockCorpusReader) ListDocs(_ domain.DocFilter) ([]domain.DocMeta, error) {
	return m.docs, nil
}

func (m *mockCorpusReader) ReadDoc(_ string) (string, error) {
	return m.body, nil
}

func TestReviewCorpus_HappyPath(t *testing.T) {
	docs := make([]domain.DocMeta, 6)
	for i := range docs {
		docs[i] = domain.DocMeta{
			Type:     "decision",
			Date:     fmt.Sprintf("2026-03-%02d", i+1),
			Status:   "draft",
			Filename: fmt.Sprintf("decision-%d-2026-03-%02d.md", i, i+1),
		}
	}

	reader := &mockCorpusReader{
		docs: docs,
		body: "---\ntype: decision\ndate: 2026-03-01\nstatus: draft\n---\n# Test\n\n## Why\nBecause.\n",
	}

	jsonResponse := `{"findings": [{"severity": "style", "title": "test", "description": "desc", "documents": ["a.md"]}]}`
	provider := &mockProvider{response: jsonResponse}

	report, total, err := ReviewCorpus(context.Background(), provider, reader, &config.Config{}, nil)
	if err != nil {
		t.Fatalf("ReviewCorpus error: %v", err)
	}
	if total != 6 {
		t.Errorf("total = %d, want 6", total)
	}
	if len(report.Findings) != 1 {
		t.Errorf("findings = %d, want 1", len(report.Findings))
	}
}

func TestReviewCorpus_TooFewDocs(t *testing.T) {
	reader := &mockCorpusReader{
		docs: []domain.DocMeta{
			{Type: "decision", Date: "2026-03-01", Status: "draft", Filename: "a.md"},
		},
		body: "# test",
	}
	provider := &mockProvider{response: `{"findings": []}`}

	_, _, err := ReviewCorpus(context.Background(), provider, reader, &config.Config{}, nil)
	if err == nil {
		t.Fatal("expected error for too few docs (< 5)")
	}
}

// --- EvaluateCommit ---

type mockGitAdapter struct {
	logResult  *domain.CommitInfo
	logErr     error
	diffResult string
	diffErr    error
}

func (m *mockGitAdapter) Log(_ string) (*domain.CommitInfo, error)          { return m.logResult, m.logErr }
func (m *mockGitAdapter) Diff(_ string) (string, error)                     { return m.diffResult, m.diffErr }
func (m *mockGitAdapter) HeadRef() (string, error)                          { return "", nil }
func (m *mockGitAdapter) HeadCommit() (*domain.CommitInfo, error)           { return m.logResult, m.logErr }
func (m *mockGitAdapter) CommitExists(_ string) (bool, error)               { return true, nil }
func (m *mockGitAdapter) IsMergeCommit(_ string) (bool, error)              { return false, nil }
func (m *mockGitAdapter) IsInsideWorkTree() bool                            { return true }
func (m *mockGitAdapter) IsRebaseInProgress() (bool, error)                 { return false, nil }
func (m *mockGitAdapter) CommitMessageContains(_, _ string) (bool, error)   { return false, nil }
func (m *mockGitAdapter) GitDir() (string, error)                           { return "", nil }
func (m *mockGitAdapter) InstallHook(_ string) (domain.InstallResult, error) { return domain.InstallResult{}, nil }
func (m *mockGitAdapter) UninstallHook(_ string) error                      { return nil }
func (m *mockGitAdapter) HookExists(_ string) (bool, error)                 { return false, nil }
func (m *mockGitAdapter) CommitRange(_, _ string) ([]string, error)         { return nil, nil }
func (m *mockGitAdapter) LatestTag() (string, error)                        { return "", nil }
func (m *mockGitAdapter) LogAll() ([]domain.CommitInfo, error)              { return nil, nil }
func (m *mockGitAdapter) CurrentBranch() (string, error)                    { return "main", nil }

type mockStore struct{}

func (m *mockStore) RecordCommit(_ domain.CommitRecord) error { return nil }
func (m *mockStore) GetCommit(_ string) (*domain.CommitRecord, error) { return nil, nil }
func (m *mockStore) CommitsByScope(_ string, _ int) ([]domain.CommitRecord, error) { return nil, nil }
func (m *mockStore) CommitsByBranch(_ string) ([]domain.CommitRecord, error) { return nil, nil }
func (m *mockStore) CommitsSince(_ time.Time) ([]domain.CommitRecord, error) { return nil, nil }
func (m *mockStore) UndocumentedCommits() ([]domain.CommitRecord, error) { return nil, nil }
func (m *mockStore) CommitCountByDecision() (map[string]int, error) { return nil, nil }
func (m *mockStore) ScopeStats(_ string, _ int) (domain.ScopeStatsResult, error) { return domain.ScopeStatsResult{}, nil }
func (m *mockStore) IndexDoc(_ domain.DocIndexEntry) error { return nil }
func (m *mockStore) RemoveDoc(_ string) error { return nil }
func (m *mockStore) GetDoc(_ string) (*domain.DocIndexEntry, error) { return nil, nil }
func (m *mockStore) DocsByScope(_ string) ([]domain.DocIndexEntry, error) { return nil, nil }
func (m *mockStore) DocsByBranch(_ string) ([]domain.DocIndexEntry, error) { return nil, nil }
func (m *mockStore) DocsByType(_ string) ([]domain.DocIndexEntry, error) { return nil, nil }
func (m *mockStore) UnconsolidatedDocs(_ string) ([]domain.DocIndexEntry, error) { return nil, nil }
func (m *mockStore) AllDocSummaries(_ int) ([]domain.DocIndexEntry, error) { return nil, nil }
func (m *mockStore) DocsByCommitHash(_ string) ([]domain.DocIndexEntry, error) { return nil, nil }
func (m *mockStore) SearchDocs(_ context.Context, _ string) ([]domain.DocIndexEntry, error) { return nil, nil }
func (m *mockStore) DocCount() (int, error) { return 0, nil }
func (m *mockStore) StoreSignatures(_ string, _ []domain.CodeSignature) error { return nil }
func (m *mockStore) FindBySignatureHash(_ string) ([]domain.CodeSignature, error) { return nil, nil }
func (m *mockStore) SignaturesForCommit(_ string) ([]domain.CodeSignature, error) { return nil, nil }
func (m *mockStore) EntityHistory(_, _ string) ([]domain.CodeSignature, error) { return nil, nil }
func (m *mockStore) RecordAIUsage(_ domain.AIUsageRecord) error { return nil }
func (m *mockStore) AIStatsSince(_ time.Time) (*domain.AIStatsAggregate, error) { return nil, nil }
func (m *mockStore) AIStatsByDay(_ int) ([]domain.DailyAIStats, error) { return nil, nil }
func (m *mockStore) CacheReview(_ domain.ReviewCacheEntry) error { return nil }
func (m *mockStore) GetCachedReview(_ string) (*domain.ReviewCacheEntry, error) { return nil, nil }
func (m *mockStore) ReviewHistory(_ int) ([]domain.ReviewCacheEntry, error) { return nil, nil }
func (m *mockStore) UpdatePattern(_, _, _ string, _, _ int) error { return nil }
func (m *mockStore) GetPattern(_, _ string) (*domain.CommitPattern, error) { return nil, nil }
func (m *mockStore) AllPatterns() ([]domain.CommitPattern, error) { return nil, nil }
func (m *mockStore) Rebuild(_ context.Context, _ string, _ domain.GitAdapter) error { return nil }
func (m *mockStore) Vacuum() error { return nil }
func (m *mockStore) Close() error { return nil }

func TestEvaluateCommit_HappyPath(t *testing.T) {
	adapter := &mockGitAdapter{
		logResult: &domain.CommitInfo{
			Hash:    "abc123",
			Type:    "feat",
			Scope:   "auth",
			Subject: "add JWT middleware",
			Message: "feat(auth): add JWT middleware",
		},
		diffResult: "+func authHandler() {\n+}\n",
	}
	store := &mockStore{}
	cfg := &config.Config{}

	result, err := EvaluateCommit(store, cfg, "abc123", adapter)
	if err != nil {
		t.Fatalf("EvaluateCommit error: %v", err)
	}
	if result.Decision == nil {
		t.Fatal("Decision should not be nil")
	}
	if result.CommitInfo.Hash != "abc123" {
		t.Errorf("CommitInfo.Hash = %q, want abc123", result.CommitInfo.Hash)
	}
}

func TestEvaluateCommit_LogError(t *testing.T) {
	adapter := &mockGitAdapter{
		logErr: fmt.Errorf("git log failed"),
	}
	store := &mockStore{}
	cfg := &config.Config{}

	_, err := EvaluateCommit(store, cfg, "bad", adapter)
	if err == nil {
		t.Fatal("expected error from Log failure")
	}
}

func TestEvaluateCommit_DiffError_NonFatal(t *testing.T) {
	adapter := &mockGitAdapter{
		logResult: &domain.CommitInfo{
			Hash:    "abc123",
			Type:    "fix",
			Subject: "fix bug",
			Message: "fix: fix bug",
		},
		diffErr: fmt.Errorf("diff failed"),
	}
	store := &mockStore{}
	cfg := &config.Config{}

	result, err := EvaluateCommit(store, cfg, "abc123", adapter)
	if err != nil {
		t.Fatalf("diff error should be non-fatal: %v", err)
	}
	if result.Decision == nil {
		t.Fatal("Decision should not be nil even without diff")
	}
}

func TestPolishDocument_BrokenFrontmatter(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, ".lore", "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Broken YAML frontmatter: unclosed bracket
	content := "---\ntype: [broken\n---\n# Test\n"
	if err := os.WriteFile(filepath.Join(docsDir, "decision-bad-2026-04-05.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	provider := &mockProvider{response: "polished"}
	cfg := &config.Config{}

	_, err := PolishDocument(context.Background(), provider, cfg, docsDir, "decision-bad-2026-04-05.md")
	if err == nil {
		t.Fatal("expected error from broken frontmatter")
	}
}

func TestPolishDocument_WithStyleGuide(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, ".lore", "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := "---\ntype: decision\ndate: 2026-04-05\nstatus: draft\n---\n# Test\n\n## Why\nBecause reasons.\n"
	if err := os.WriteFile(filepath.Join(docsDir, "decision-style-2026-04-05.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	provider := &mockProvider{response: content}
	cfg := &config.Config{
		Angela: config.AngelaConfig{
			StyleGuide: map[string]interface{}{"tone": "formal"},
		},
	}

	result, err := PolishDocument(context.Background(), provider, cfg, docsDir, "decision-style-2026-04-05.md")
	if err != nil {
		t.Fatalf("PolishDocument with style guide error: %v", err)
	}
	if result.Meta.Type != "decision" {
		t.Errorf("Meta.Type = %q, want decision", result.Meta.Type)
	}
}

func TestReviewCorpus_WithStyleGuide(t *testing.T) {
	docs := make([]domain.DocMeta, 6)
	for i := range docs {
		docs[i] = domain.DocMeta{
			Type:     "decision",
			Date:     fmt.Sprintf("2026-03-%02d", i+1),
			Status:   "draft",
			Filename: fmt.Sprintf("decision-%d-2026-03-%02d.md", i, i+1),
		}
	}

	reader := &mockCorpusReader{
		docs: docs,
		body: "---\ntype: decision\ndate: 2026-03-01\nstatus: draft\n---\n# Test\n\n## Why\nBecause.\n",
	}

	jsonResponse := `{"findings": [{"severity": "style", "title": "tone", "description": "not formal", "documents": ["a.md"]}]}`
	provider := &mockProvider{response: jsonResponse}

	styleGuide := map[string]interface{}{"tone": "formal"}
	report, total, err := ReviewCorpus(context.Background(), provider, reader, &config.Config{}, styleGuide)
	if err != nil {
		t.Fatalf("ReviewCorpus with style guide error: %v", err)
	}
	if total != 6 {
		t.Errorf("total = %d, want 6", total)
	}
	if len(report.Findings) != 1 {
		t.Errorf("findings = %d, want 1", len(report.Findings))
	}
}

// Ensure angela import is used (for type checking).
var _ = angela.DocSummary{}
// Ensure storage import is used.
var _ = storage.CorpusStore{}
