// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package status

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/greycoderk/lore/internal/testutil"
)

// mockGit implements domain.GitAdapter for testing.
type mockGit struct {
	hookExists bool
	hookErr    error
}

func (m *mockGit) HookExists(hookType string) (bool, error) { return m.hookExists, m.hookErr }
func (m *mockGit) Diff(ref string) (string, error)          { return "", nil }
func (m *mockGit) Log(ref string) (*domain.CommitInfo, error) {
	return &domain.CommitInfo{}, nil
}
func (m *mockGit) CommitExists(ref string) (bool, error)                  { return true, nil }
func (m *mockGit) IsMergeCommit(ref string) (bool, error)                 { return false, nil }
func (m *mockGit) IsInsideWorkTree() bool                                 { return true }
func (m *mockGit) HeadRef() (string, error)                               { return "HEAD", nil }
func (m *mockGit) HeadCommit() (*domain.CommitInfo, error)                { return nil, nil }
func (m *mockGit) IsRebaseInProgress() (bool, error)                      { return false, nil }
func (m *mockGit) CommitMessageContains(ref, marker string) (bool, error) { return false, nil }
func (m *mockGit) GitDir() (string, error)                                { return ".git", nil }
func (m *mockGit) InstallHook(hookType string) (domain.InstallResult, error) {
	return domain.InstallResult{}, nil
}
func (m *mockGit) UninstallHook(hookType string) error              { return nil }
func (m *mockGit) CommitRange(from, to string) ([]string, error)    { return nil, nil }
func (m *mockGit) LatestTag() (string, error)                       { return "", nil }
func (m *mockGit) LogAll() ([]domain.CommitInfo, error)             { return nil, nil }
func (m *mockGit) CurrentBranch() (string, error)                   { return "main", nil }

func setupCollectorTest(t *testing.T, docs []testutil.DocFixture) string {
	t.Helper()
	dir := testutil.SetupLoreDirWithDocs(t, docs)
	testutil.Chdir(t, dir)
	return dir
}

func TestCollectStatus_HookInstalled(t *testing.T) {
	dir := setupCollectorTest(t, nil)
	// Create README.md so health check passes
	os.WriteFile(filepath.Join(dir, ".lore", "docs", "README.md"), []byte("# Index\n"), 0o644)

	cfg := &config.Config{}
	git := &mockGit{hookExists: true}

	info, err := CollectStatus(cfg, git, filepath.Join(dir, ".lore"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !info.HookInstalled {
		t.Error("expected HookInstalled true")
	}
}

func TestCollectStatus_DocCount(t *testing.T) {
	dir := setupCollectorTest(t, []testutil.DocFixture{
		{Type: "decision", Slug: "auth", Date: "2026-03-07"},
		{Type: "feature", Slug: "api", Date: "2026-03-08"},
	})
	os.WriteFile(filepath.Join(dir, ".lore", "docs", "README.md"), []byte("# Index\n"), 0o644)

	cfg := &config.Config{}
	git := &mockGit{}

	info, err := CollectStatus(cfg, git, filepath.Join(dir, ".lore"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.DocCount != 2 {
		t.Errorf("expected DocCount 2, got %d", info.DocCount)
	}
}

func TestCollectStatus_PendingCount(t *testing.T) {
	dir := setupCollectorTest(t, nil)
	os.WriteFile(filepath.Join(dir, ".lore", "docs", "README.md"), []byte("# Index\n"), 0o644)

	// Create pending files
	pendingDir := filepath.Join(dir, ".lore", "pending")
	for _, name := range []string{"abc123.yaml", "def456.yaml"} {
		os.WriteFile(filepath.Join(pendingDir, name), []byte("pending: true"), 0o644)
	}

	cfg := &config.Config{}
	git := &mockGit{}

	info, err := CollectStatus(cfg, git, filepath.Join(dir, ".lore"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.PendingCount != 2 {
		t.Errorf("expected PendingCount 2, got %d", info.PendingCount)
	}
}

func TestCollectStatus_AngelaModeDraft(t *testing.T) {
	dir := setupCollectorTest(t, nil)
	os.WriteFile(filepath.Join(dir, ".lore", "docs", "README.md"), []byte("# Index\n"), 0o644)

	cfg := &config.Config{}
	git := &mockGit{}

	info, err := CollectStatus(cfg, git, filepath.Join(dir, ".lore"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.AngelaMode != "draft" {
		t.Errorf("expected AngelaMode 'draft', got %q", info.AngelaMode)
	}
}

func TestCollectStatus_AngelaModePolish(t *testing.T) {
	dir := setupCollectorTest(t, nil)
	os.WriteFile(filepath.Join(dir, ".lore", "docs", "README.md"), []byte("# Index\n"), 0o644)

	cfg := &config.Config{
		AI: config.AIConfig{Provider: "anthropic", APIKey: "sk-test"},
	}
	git := &mockGit{}

	info, err := CollectStatus(cfg, git, filepath.Join(dir, ".lore"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.AngelaMode != "polish" {
		t.Errorf("expected AngelaMode 'polish', got %q", info.AngelaMode)
	}
	if info.AIProvider != "anthropic" {
		t.Errorf("expected AIProvider 'anthropic', got %q", info.AIProvider)
	}
}

func TestCollectStatus_ExpressRatio(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	docsDir := filepath.Join(dir, ".lore", "docs")
	// Create a "complete" doc with ## Alternatives section
	completeMeta := domain.DocMeta{Type: "decision", Date: "2026-03-07", Status: "draft"}
	completeBody := "# Auth\n\n## Alternatives\nOption A vs B\n\n## Impact\nHigh\n"
	data, _ := storage.Marshal(completeMeta, completeBody)
	os.WriteFile(filepath.Join(docsDir, "decision-auth-2026-03-07.md"), data, 0o644)

	// Create an "express" doc without those sections
	expressMeta := domain.DocMeta{Type: "feature", Date: "2026-03-08", Status: "draft"}
	expressBody := "# API\n\nSimple feature.\n"
	data2, _ := storage.Marshal(expressMeta, expressBody)
	os.WriteFile(filepath.Join(docsDir, "feature-api-2026-03-08.md"), data2, 0o644)

	os.WriteFile(filepath.Join(docsDir, "README.md"), []byte("# Index\n"), 0o644)

	cfg := &config.Config{}
	git := &mockGit{}

	info, err := CollectStatus(cfg, git, filepath.Join(dir, ".lore"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.CompleteCount != 1 {
		t.Errorf("expected CompleteCount 1, got %d", info.CompleteCount)
	}
	if info.ExpressCount != 1 {
		t.Errorf("expected ExpressCount 1, got %d", info.ExpressCount)
	}
}

func TestCountUniqueDocsInFindings(t *testing.T) {
	findings := []angela.ReviewFinding{
		{Documents: []string{"a.md", "b.md"}},
		{Documents: []string{"b.md", "c.md"}},
		{Documents: []string{}},
	}
	count := countUniqueDocsInFindings(findings)
	if count != 3 {
		t.Errorf("unique docs = %d, want 3", count)
	}
}

func TestCountUniqueDocsInFindings_Empty(t *testing.T) {
	count := countUniqueDocsInFindings(nil)
	if count != 0 {
		t.Errorf("unique docs = %d, want 0", count)
	}
}

func TestDetectAIProvider_EnvVar(t *testing.T) {
	t.Setenv("LORE_AI_API_KEY", "sk-test-env-key")

	// With provider set in config, should return the provider name
	cfg := &config.Config{AI: config.AIConfig{Provider: "anthropic"}}
	got := detectAIProvider(cfg)
	if got != "anthropic" {
		t.Errorf("detectAIProvider with env + provider = %q, want 'anthropic'", got)
	}

	// Without provider in config, should return "configured"
	cfg2 := &config.Config{}
	got2 := detectAIProvider(cfg2)
	if got2 != "configured" {
		t.Errorf("detectAIProvider with env only = %q, want 'configured'", got2)
	}
}

func TestDetectAIProvider_PlaintextKey(t *testing.T) {
	// Ensure env var is not set
	t.Setenv("LORE_AI_API_KEY", "")

	// Plaintext key with provider
	cfg := &config.Config{AI: config.AIConfig{Provider: "openai", APIKey: "sk-plaintext"}}
	got := detectAIProvider(cfg)
	if got != "openai" {
		t.Errorf("detectAIProvider plaintext+provider = %q, want 'openai'", got)
	}

	// Plaintext key without provider
	cfg2 := &config.Config{AI: config.AIConfig{APIKey: "sk-plaintext"}}
	got2 := detectAIProvider(cfg2)
	if got2 != "configured" {
		t.Errorf("detectAIProvider plaintext only = %q, want 'configured'", got2)
	}
}

func TestDetectAIProvider_KeychainMarker(t *testing.T) {
	// @keychain marker should NOT count as plaintext key
	t.Setenv("LORE_AI_API_KEY", "")
	cfg := &config.Config{AI: config.AIConfig{Provider: "", APIKey: "@keychain"}}
	got := detectAIProvider(cfg)
	// Without provider, keychain path won't be checked; should return ""
	if got != "" {
		t.Errorf("detectAIProvider @keychain no provider = %q, want empty", got)
	}
}

func TestDetectAIProvider_NoConfig(t *testing.T) {
	t.Setenv("LORE_AI_API_KEY", "")
	cfg := &config.Config{}
	got := detectAIProvider(cfg)
	if got != "" {
		t.Errorf("detectAIProvider empty config = %q, want empty", got)
	}
}

func TestCollectStatus_AngelaReviewCache(t *testing.T) {
	dir := setupCollectorTest(t, nil)
	os.WriteFile(filepath.Join(dir, ".lore", "docs", "README.md"), []byte("# Index\n"), 0o644)

	// Create a review cache with findings
	cacheDir := filepath.Join(dir, ".lore", "cache")
	os.MkdirAll(cacheDir, 0o755)
	cacheData := `{
		"version": 1,
		"last_review": "2026-03-15T10:00:00Z",
		"doc_count": 3,
		"total_docs": 5,
		"findings": [
			{"severity": "gap", "title": "Missing auth doc", "description": "No auth doc", "documents": ["a.md", "b.md"]},
			{"severity": "style", "title": "Style issue", "description": "Bad style", "documents": ["b.md", "c.md"]}
		]
	}`
	os.WriteFile(filepath.Join(cacheDir, "review.json"), []byte(cacheData), 0o644)

	cfg := &config.Config{}
	git := &mockGit{}

	info, err := CollectStatus(cfg, git, filepath.Join(dir, ".lore"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.AngelaSuggestions != 2 {
		t.Errorf("AngelaSuggestions = %d, want 2", info.AngelaSuggestions)
	}
	if info.AngelaDocsNeedReview != 3 {
		t.Errorf("AngelaDocsNeedReview = %d, want 3", info.AngelaDocsNeedReview)
	}
}
