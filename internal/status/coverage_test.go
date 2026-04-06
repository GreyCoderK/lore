// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package status

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/stretchr/testify/assert"
)

type mockGitForCoverage struct {
	commits []domain.CommitInfo
	logErr  error
}

func (m *mockGitForCoverage) LogAll() ([]domain.CommitInfo, error) {
	return m.commits, m.logErr
}
func (m *mockGitForCoverage) Diff(string) (string, error)                         { return "", nil }
func (m *mockGitForCoverage) Log(string) (*domain.CommitInfo, error)               { return nil, nil }
func (m *mockGitForCoverage) CommitExists(string) (bool, error)                    { return true, nil }
func (m *mockGitForCoverage) IsMergeCommit(string) (bool, error)                   { return false, nil }
func (m *mockGitForCoverage) IsInsideWorkTree() bool                               { return true }
func (m *mockGitForCoverage) HeadRef() (string, error)                             { return "abc", nil }
func (m *mockGitForCoverage) HeadCommit() (*domain.CommitInfo, error)              { return nil, nil }
func (m *mockGitForCoverage) IsRebaseInProgress() (bool, error)                    { return false, nil }
func (m *mockGitForCoverage) CommitMessageContains(_, _ string) (bool, error)      { return false, nil }
func (m *mockGitForCoverage) GitDir() (string, error)                              { return "", nil }
func (m *mockGitForCoverage) InstallHook(string) (domain.InstallResult, error)     { return domain.InstallResult{}, nil }
func (m *mockGitForCoverage) UninstallHook(string) error                           { return nil }
func (m *mockGitForCoverage) HookExists(string) (bool, error)                      { return false, nil }
func (m *mockGitForCoverage) CommitRange(_, _ string) ([]string, error)            { return nil, nil }
func (m *mockGitForCoverage) LatestTag() (string, error)                           { return "", nil }
func (m *mockGitForCoverage) CurrentBranch() (string, error)                       { return "main", nil }

func TestCalculateCoverage_HappyPath(t *testing.T) {
	// Setup docs dir with 2 documented commits
	docsDir := filepath.Join(t.TempDir(), ".lore", "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, doc := range []struct{ name, commit string }{
		{"decision-a-2026-03-01.md", "aaa111"},
		{"feature-b-2026-03-02.md", "bbb222"},
	} {
		content := "---\ntype: decision\ndate: 2026-03-01\nstatus: draft\ncommit: " + doc.commit + "\n---\n# Test\n\n## Why\nBecause.\n"
		if err := os.WriteFile(filepath.Join(docsDir, doc.name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	adapter := &mockGitForCoverage{
		commits: []domain.CommitInfo{
			{Hash: "aaa111", Message: "feat: add A", Date: time.Now()},
			{Hash: "bbb222", Message: "feat: add B", Date: time.Now()},
			{Hash: "ccc333", Message: "fix: bug C", Date: time.Now()},
			{Hash: "ddd444", Message: "merge branch", IsMerge: true, Date: time.Now()},
		},
	}

	result := CalculateCoverage(docsDir, adapter)
	assert.Equal(t, 4, result.TotalCommits)
	assert.Equal(t, 1, result.MergeCommits)
	assert.Equal(t, 3, result.Eligible) // 4 - 1 merge
	assert.Equal(t, 2, result.Documented)
	assert.True(t, result.Coverage > 0)
}

func TestCalculateCoverage_NoCommits(t *testing.T) {
	docsDir := t.TempDir()
	adapter := &mockGitForCoverage{commits: nil}
	result := CalculateCoverage(docsDir, adapter)
	assert.Equal(t, 0, result.TotalCommits)
	assert.Equal(t, 0, result.Coverage)
}

func TestCalculateCoverage_AllMerges(t *testing.T) {
	docsDir := t.TempDir()
	adapter := &mockGitForCoverage{
		commits: []domain.CommitInfo{
			{Hash: "aaa", Message: "merge", IsMerge: true, Date: time.Now()},
			{Hash: "bbb", Message: "merge", IsMerge: true, Date: time.Now()},
		},
	}
	result := CalculateCoverage(docsDir, adapter)
	assert.Equal(t, 2, result.MergeCommits)
	assert.Equal(t, 0, result.Eligible)
	assert.Equal(t, 0, result.Coverage)
}

func TestCalculateCoverage_DocSkip(t *testing.T) {
	docsDir := t.TempDir()
	adapter := &mockGitForCoverage{
		commits: []domain.CommitInfo{
			{Hash: "aaa", Message: "feat: add [doc-skip]", Date: time.Now()},
			{Hash: "bbb", Message: "feat: normal", Date: time.Now()},
		},
	}
	result := CalculateCoverage(docsDir, adapter)
	assert.Equal(t, 1, result.DocSkipped)
	assert.Equal(t, 1, result.Covered) // doc-skip counts as covered
}

func TestCalculateCoverage_LogError(t *testing.T) {
	docsDir := t.TempDir()
	adapter := &mockGitForCoverage{logErr: fmt.Errorf("git broken")}
	result := CalculateCoverage(docsDir, adapter)
	assert.Equal(t, 0, result.TotalCommits)
}

// Verify storage import is used
var _ = storage.CorpusStore{}

func TestBadgeColor(t *testing.T) {
	assert.Equal(t, "555", BadgeColor(0))
	assert.Equal(t, "555", BadgeColor(49))
	assert.Equal(t, "5c2", BadgeColor(50))
	assert.Equal(t, "5c2", BadgeColor(79))
	assert.Equal(t, "d4a017", BadgeColor(80))
	assert.Equal(t, "d4a017", BadgeColor(100))
}

func TestBadgeColor_GoldRange(t *testing.T) {
	// Verify gold for all values >= 80 but < 100
	for _, pct := range []int{80, 85, 90, 95, 99} {
		assert.Equal(t, "d4a017", BadgeColor(pct), "BadgeColor(%d) should be gold", pct)
	}
}

func TestFormatBadgeMarkdown(t *testing.T) {
	badge := FormatBadgeMarkdown(78, "documented")
	assert.Contains(t, badge, "78%")
	assert.Contains(t, badge, "documented")
	assert.Contains(t, badge, "shields.io")
	assert.Contains(t, badge, "5c2") // green for 78%
}

func TestFormatBadgeMarkdown_100(t *testing.T) {
	badge := FormatBadgeMarkdown(100, "documented")
	assert.Contains(t, badge, "100%")
	assert.Contains(t, badge, "d4a017") // gold
	// 100% uses emoji display in the badge URL
	assert.Contains(t, badge, "\U0001f4af")
}

func TestFormatBadgeMarkdown_LessThan100(t *testing.T) {
	badge := FormatBadgeMarkdown(90, "documented")
	assert.Contains(t, badge, "90%")
	assert.Contains(t, badge, "documented")
	assert.Contains(t, badge, "d4a017") // gold for 90%
	assert.Contains(t, badge, "shields.io")
	// < 100% uses percentage display, not emoji
	assert.Contains(t, badge, "90%25")
}

func TestFormatBadgeMarkdown_0(t *testing.T) {
	badge := FormatBadgeMarkdown(0, "documented")
	assert.Contains(t, badge, "0%")
	assert.Contains(t, badge, "555") // grey
}
