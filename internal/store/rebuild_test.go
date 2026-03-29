// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/domain"
)

// mockGitForRebuild implements domain.GitAdapter for rebuild tests.
type mockGitForRebuild struct {
	commits []domain.CommitInfo
}

func (m *mockGitForRebuild) LogAll() ([]domain.CommitInfo, error) { return m.commits, nil }
func (m *mockGitForRebuild) CurrentBranch() (string, error)      { return "main", nil }

// Stubs for remaining GitAdapter methods
func (m *mockGitForRebuild) Diff(string) (string, error)                  { return "", nil }
func (m *mockGitForRebuild) Log(string) (*domain.CommitInfo, error)        { return nil, nil }
func (m *mockGitForRebuild) CommitExists(string) (bool, error)            { return false, nil }
func (m *mockGitForRebuild) IsMergeCommit(string) (bool, error)           { return false, nil }
func (m *mockGitForRebuild) IsInsideWorkTree() bool                       { return true }
func (m *mockGitForRebuild) HeadRef() (string, error)                     { return "abc", nil }
func (m *mockGitForRebuild) HeadCommit() (*domain.CommitInfo, error)      { return nil, nil }
func (m *mockGitForRebuild) IsRebaseInProgress() (bool, error)            { return false, nil }
func (m *mockGitForRebuild) CommitMessageContains(string, string) (bool, error) { return false, nil }
func (m *mockGitForRebuild) GitDir() (string, error)                      { return ".git", nil }
func (m *mockGitForRebuild) InstallHook(string) (domain.InstallResult, error) {
	return domain.InstallResult{}, nil
}
func (m *mockGitForRebuild) UninstallHook(string) error          { return nil }
func (m *mockGitForRebuild) HookExists(string) (bool, error)     { return false, nil }
func (m *mockGitForRebuild) CommitRange(string, string) ([]string, error) { return nil, nil }
func (m *mockGitForRebuild) LatestTag() (string, error)          { return "", nil }

func TestRebuild_DocIndex_FromFixtures(t *testing.T) {
	s, _ := tempDB(t)

	// Create fixture docs directory
	docsDir := filepath.Join(t.TempDir(), "docs")
	os.MkdirAll(docsDir, 0o755)

	// Write 3 valid docs
	for _, name := range []string{"decision-auth-2026-03-01.md", "feature-api-2026-03-02.md", "bugfix-login-2026-03-03.md"} {
		docType := "decision"
		if name[:7] == "feature" {
			docType = "feature"
		} else if name[:6] == "bugfix" {
			docType = "bugfix"
		}
		content := "---\ntype: " + docType + "\ndate: \"2026-03-01\"\nstatus: draft\n---\n# Title\n\nBody content here."
		os.WriteFile(filepath.Join(docsDir, name), []byte(content), 0o644)
	}

	// Write 1 malformed doc (missing type)
	os.WriteFile(filepath.Join(docsDir, "bad-doc-2026-03-04.md"), []byte("---\ndate: \"2026-03-04\"\nstatus: draft\n---\nNo type."), 0o644)

	docCount, docSkipped, _, err := s.RebuildFromSources(context.Background(), docsDir, nil)
	if err != nil {
		t.Fatalf("RebuildFromSources: %v", err)
	}

	if docCount != 3 {
		t.Errorf("docCount = %d, want 3", docCount)
	}
	if docSkipped != 1 {
		t.Errorf("docSkipped = %d, want 1", docSkipped)
	}

	// Verify indexed docs
	count, _ := s.DocCount()
	if count != 3 {
		t.Errorf("DocCount = %d, want 3", count)
	}

	got, _ := s.GetDoc("decision-auth-2026-03-01.md")
	if got == nil {
		t.Fatal("decision-auth doc not found in index")
	}
	if got.Type != "decision" {
		t.Errorf("Type = %q, want decision", got.Type)
	}
	if got.TitleExtracted != "Title" {
		t.Errorf("TitleExtracted = %q, want Title", got.TitleExtracted)
	}
	if got.WordCount < 2 {
		t.Errorf("WordCount = %d, want >= 2", got.WordCount)
	}
}

func TestRebuild_Commits_FromGitMock(t *testing.T) {
	s, _ := tempDB(t)

	docsDir := filepath.Join(t.TempDir(), "docs")
	os.MkdirAll(docsDir, 0o755)

	git := &mockGitForRebuild{
		commits: []domain.CommitInfo{
			{Hash: "aaa111", Date: time.Now(), Message: "feat(auth): add login", Type: "feat", Scope: "auth", Subject: "add login"},
			{Hash: "bbb222", Date: time.Now(), Message: "fix(api): timeout", Type: "fix", Scope: "api", Subject: "timeout"},
			{Hash: "ccc333", Date: time.Now(), Message: "chore: deps", Type: "chore", Subject: "deps"},
			{Hash: "ddd444", Date: time.Now(), Message: "docs: readme", Type: "docs", Subject: "readme"},
			{Hash: "eee555", Date: time.Now(), Message: "feat(db): schema", Type: "feat", Scope: "db", Subject: "schema"},
		},
	}

	_, _, commitCount, err := s.RebuildFromSources(context.Background(), docsDir, git)
	if err != nil {
		t.Fatalf("RebuildFromSources: %v", err)
	}

	if commitCount != 5 {
		t.Errorf("commitCount = %d, want 5", commitCount)
	}

	// Verify commits have decision=unknown
	got, _ := s.GetCommit("aaa111")
	if got == nil {
		t.Fatal("commit aaa111 not found")
	}
	if got.Decision != "unknown" {
		t.Errorf("Decision = %q, want unknown", got.Decision)
	}
	if got.FilesChanged != 0 {
		t.Errorf("FilesChanged = %d, want 0 (rebuild has no diff stats)", got.FilesChanged)
	}
}

func TestVacuum_RunsWithoutError(t *testing.T) {
	s, _ := tempDB(t)
	if err := s.Vacuum(); err != nil {
		t.Errorf("Vacuum: %v", err)
	}
}
