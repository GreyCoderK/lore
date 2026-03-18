// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package status

import (
	"os"
	"path/filepath"
	"testing"

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
func (m *mockGit) IsRebaseInProgress() (bool, error)                      { return false, nil }
func (m *mockGit) CommitMessageContains(ref, marker string) (bool, error) { return false, nil }
func (m *mockGit) GitDir() (string, error)                                { return ".git", nil }
func (m *mockGit) InstallHook(hookType string) (domain.InstallResult, error) {
	return domain.InstallResult{}, nil
}
func (m *mockGit) UninstallHook(hookType string) error { return nil }

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
