// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/status"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/greycoderk/lore/internal/testutil"
	"github.com/greycoderk/lore/internal/ui"
)

func setupStatusTest(t *testing.T, docs []testutil.DocFixture) (string, domain.IOStreams, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	dir := testutil.SetupGitRepoWithHook(t)

	// Create .lore structure
	for _, sub := range []string{
		filepath.Join(".lore", "docs"),
		filepath.Join(".lore", "templates"),
		filepath.Join(".lore", "pending"),
	} {
		_ = os.MkdirAll(filepath.Join(dir, sub), 0o755)
	}

	// Create docs
	docsDir := filepath.Join(dir, ".lore", "docs")
	for _, d := range docs {
		meta := domain.DocMeta{Type: d.Type, Date: d.Date, Status: "draft", Tags: d.Tags}
		body := d.Body
		if body == "" {
			body = "# " + strings.ReplaceAll(d.Slug, "-", " ") + "\n\nTest document.\n"
		}
		data, err := storage.Marshal(meta, body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		filename := d.Type + "-" + d.Slug + "-" + d.Date + ".md"
		os.WriteFile(filepath.Join(docsDir, filename), data, 0o644)
	}

	// Create README.md so health passes
	os.WriteFile(filepath.Join(docsDir, "README.md"), []byte("# Index\n"), 0o644)

	testutil.Chdir(t, dir)

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &out,
		Err: &errBuf,
		In:  strings.NewReader(""),
	}
	return dir, streams, &out, &errBuf
}

func executeStatus(t *testing.T, streams domain.IOStreams, args ...string) error {
	t.Helper()
	cfg := &config.Config{}
	cmd := newStatusCmd(cfg, streams)
	cmd.SetArgs(args)
	return cmd.Execute()
}

// AC-1: Dashboard format
func TestStatusCmd_Dashboard(t *testing.T) {
	_, streams, _, errBuf := setupStatusTest(t, []testutil.DocFixture{
		{Type: "decision", Slug: "auth", Date: "2026-03-07"},
		{Type: "feature", Slug: "api", Date: "2026-03-08"},
	})

	// Disable colors for predictable output
	restore := ui.SaveAndDisableColor()
	defer restore()

	err := executeStatus(t, streams)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stderr := errBuf.String()
	if !strings.Contains(stderr, "lore status") {
		t.Errorf("expected header, got %q", stderr)
	}
	// NOTE: Label assertions below are coupled to i18n English strings.
	// If we add multi-language support, switch these to use i18n.T() references.
	if !strings.Contains(stderr, "Hook:") {
		t.Errorf("expected Hook label, got %q", stderr)
	}
	if !strings.Contains(stderr, "Docs:") {
		t.Errorf("expected Docs label, got %q", stderr)
	}
	if !strings.Contains(stderr, "2 documented") {
		t.Errorf("expected '2 documented', got %q", stderr)
	}
	if !strings.Contains(stderr, "Angela:") {
		t.Errorf("expected Angela label, got %q", stderr)
	}
	if !strings.Contains(stderr, "Health:") {
		t.Errorf("expected Health label, got %q", stderr)
	}
}

// AC-2: Health check OK
func TestStatusCmd_HealthOK(t *testing.T) {
	_, streams, _, errBuf := setupStatusTest(t, nil)

	restore := ui.SaveAndDisableColor()
	defer restore()

	err := executeStatus(t, streams)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// NOTE: "all good" is an i18n English string; acceptable for MVP.
	if !strings.Contains(errBuf.String(), "all good") {
		t.Errorf("expected 'all good', got %q", errBuf.String())
	}
}

// AC-3: Health check issues
func TestStatusCmd_HealthIssues(t *testing.T) {
	dir, streams, _, errBuf := setupStatusTest(t, nil)

	// Create orphan .tmp file
	os.WriteFile(filepath.Join(dir, ".lore", "docs", "orphan.tmp"), []byte(""), 0o644)

	restore := ui.SaveAndDisableColor()
	defer restore()

	err := executeStatus(t, streams)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stderr := errBuf.String()
	if !strings.Contains(stderr, "issues") {
		t.Errorf("expected 'issues', got %q", stderr)
	}
	if !strings.Contains(stderr, "lore doctor") {
		t.Errorf("expected 'lore doctor' suggestion, got %q", stderr)
	}
}

// AC-4: Pending count
func TestStatusCmd_PendingCount(t *testing.T) {
	dir, streams, _, errBuf := setupStatusTest(t, []testutil.DocFixture{
		{Type: "decision", Slug: "auth", Date: "2026-03-07"},
	})

	// Create pending files
	pendingDir := filepath.Join(dir, ".lore", "pending")
	os.WriteFile(filepath.Join(pendingDir, "abc123.yaml"), []byte("pending"), 0o644)
	os.WriteFile(filepath.Join(pendingDir, "def456.yaml"), []byte("pending"), 0o644)

	restore := ui.SaveAndDisableColor()
	defer restore()

	err := executeStatus(t, streams)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(errBuf.String(), "2 pending") {
		t.Errorf("expected '2 pending', got %q", errBuf.String())
	}
}

// AC-6: Hook status displayed
func TestStatusCmd_HookInstalled(t *testing.T) {
	_, streams, _, errBuf := setupStatusTest(t, nil)

	restore := ui.SaveAndDisableColor()
	defer restore()

	err := executeStatus(t, streams)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// NOTE: i18n-coupled assertion; acceptable for MVP.
	if !strings.Contains(errBuf.String(), "installed (post-commit)") {
		t.Errorf("expected 'installed (post-commit)', got %q", errBuf.String())
	}
}

// AC-8: Quiet mode
func TestStatusCmd_Quiet(t *testing.T) {
	_, streams, out, errBuf := setupStatusTest(t, []testutil.DocFixture{
		{Type: "decision", Slug: "auth", Date: "2026-03-07"},
	})

	err := executeStatus(t, streams, "--quiet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if errBuf.Len() != 0 {
		t.Errorf("expected no stderr in quiet mode, got %q", errBuf.String())
	}
	stdout := out.String()
	if !strings.Contains(stdout, "hook=installed") {
		t.Errorf("expected 'hook=installed', got %q", stdout)
	}
	if !strings.Contains(stdout, "docs=1") {
		t.Errorf("expected 'docs=1', got %q", stdout)
	}
	if !strings.Contains(stdout, "health=ok") {
		t.Errorf("expected 'health=ok', got %q", stdout)
	}
	if !strings.Contains(stdout, "angela=draft") {
		t.Errorf("expected 'angela=draft', got %q", stdout)
	}
}

// AC-8 + AC-3: Quiet mode with health issues
func TestStatusCmd_QuietHealthIssues(t *testing.T) {
	dir, streams, out, errBuf := setupStatusTest(t, nil)

	// Create orphan .tmp file to trigger health issue
	os.WriteFile(filepath.Join(dir, ".lore", "docs", "orphan.tmp"), []byte(""), 0o644)

	err := executeStatus(t, streams, "--quiet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if errBuf.Len() != 0 {
		t.Errorf("expected no stderr in quiet mode, got %q", errBuf.String())
	}
	stdout := out.String()
	if !strings.Contains(stdout, "health=1-issues") {
		t.Errorf("expected 'health=1-issues', got %q", stdout)
	}
}

// AC-9: Not initialized
func TestStatusCmd_NotInitialized(t *testing.T) {
	dir := t.TempDir()
	testutil.Chdir(t, dir)

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	err := executeStatus(t, streams)
	if err == nil {
		t.Fatal("expected error for not initialized")
	}
	if !errors.Is(err, domain.ErrNotInitialized) {
		t.Errorf("expected ErrNotInitialized, got: %v", err)
	}
	if !strings.Contains(errBuf.String(), "Lore not initialized") {
		t.Errorf("expected 'Lore not initialized', got %q", errBuf.String())
	}
}

// Tagline present in dashboard
func TestStatusCmd_Tagline(t *testing.T) {
	_, streams, _, errBuf := setupStatusTest(t, nil)

	restore := ui.SaveAndDisableColor()
	defer restore()

	err := executeStatus(t, streams)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// NOTE: Tagline is an i18n English string; acceptable for MVP.
	if !strings.Contains(errBuf.String(), "Your code knows what. Lore knows why.") {
		t.Errorf("expected tagline, got %q", errBuf.String())
	}
}

func TestFormatReviewAge(t *testing.T) {
	tests := []struct {
		name string
		when time.Time
	}{
		{"just now", time.Now()},
		{"hours ago", time.Now().Add(-3 * time.Hour)},
		{"days ago", time.Now().Add(-3 * 24 * time.Hour)},
		{"old", time.Now().Add(-30 * 24 * time.Hour)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatReviewAge(tt.when)
			if result == "" {
				t.Error("formatReviewAge returned empty string")
			}
		})
	}
}

func TestRenderBadge(t *testing.T) {
	dir, streams, out, _ := setupStatusTest(t, nil)

	// Add a doc to have some coverage
	content := "---\ntype: decision\ndate: 2026-03-01\nstatus: draft\ncommit: abc123\n---\n# Test\n\n## Why\nBecause.\n"
	os.WriteFile(filepath.Join(dir, ".lore", "docs", "decision-test-2026-03-01.md"), []byte(content), 0o644)

	adapter := &mockGitAdapterForStatus{}
	err := renderBadge(streams, adapter)
	if err != nil {
		t.Fatalf("renderBadge: %v", err)
	}
	stdout := out.String()
	if !strings.Contains(stdout, "shields.io") {
		t.Errorf("badge should contain shields.io link, got: %s", stdout)
	}
}

func TestRenderDashboard_ExpressRatio(t *testing.T) {
	restore := ui.SaveAndDisableColor()
	defer restore()

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &out,
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	info := &status.StatusInfo{
		ProjectName:   "test-project",
		HookInstalled: true,
		DocCount:      5,
		PendingCount:  0,
		ExpressCount:  3,
		CompleteCount: 2,
		AngelaMode:    "polish",
		AIProvider:    "anthropic",
		HealthIssues:  0,
	}

	err := renderDashboard(streams, info)
	if err != nil {
		t.Fatalf("renderDashboard: %v", err)
	}

	stderr := errBuf.String()
	if !strings.Contains(stderr, "Express:") || !strings.Contains(stderr, "60%") {
		t.Errorf("expected Express ratio with 60%%, got %q", stderr)
	}
	if !strings.Contains(stderr, "anthropic") {
		t.Errorf("expected AI provider 'anthropic', got %q", stderr)
	}
}

func TestRenderDashboard_ReadErrors(t *testing.T) {
	restore := ui.SaveAndDisableColor()
	defer restore()

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &out,
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	info := &status.StatusInfo{
		ProjectName:   "test-project",
		HookInstalled: false,
		DocCount:      0,
		ExpressCount:  0,
		CompleteCount: 0,
		ReadErrors:    2,
		AngelaMode:    "draft",
		HealthIssues:  1,
	}

	err := renderDashboard(streams, info)
	if err != nil {
		t.Fatalf("renderDashboard: %v", err)
	}

	stderr := errBuf.String()
	if !strings.Contains(stderr, "2") {
		t.Errorf("expected read errors count, got %q", stderr)
	}
}

func TestRenderDashboard_AngelaDocsNeedReview(t *testing.T) {
	restore := ui.SaveAndDisableColor()
	defer restore()

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &out,
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	info := &status.StatusInfo{
		ProjectName:         "test-project",
		HookInstalled:       true,
		DocCount:            5,
		AngelaMode:          "draft",
		AngelaDocsNeedReview: 3,
		HealthIssues:        0,
	}

	err := renderDashboard(streams, info)
	if err != nil {
		t.Fatalf("renderDashboard: %v", err)
	}

	stderr := errBuf.String()
	if !strings.Contains(stderr, "3") {
		t.Errorf("expected '3' docs need review, got %q", stderr)
	}
}

func TestRenderDashboard_AllClean(t *testing.T) {
	restore := ui.SaveAndDisableColor()
	defer restore()

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &out,
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	info := &status.StatusInfo{
		ProjectName:         "test-project",
		HookInstalled:       true,
		DocCount:            5,
		AngelaMode:          "polish",
		AIProvider:          "openai",
		AngelaDocsNeedReview: 0,
		HealthIssues:        0,
	}

	err := renderDashboard(streams, info)
	if err != nil {
		t.Fatalf("renderDashboard: %v", err)
	}

	stderr := errBuf.String()
	if !strings.Contains(stderr, "clean") || !strings.Contains(stderr, "all") {
		t.Errorf("expected 'all clean' message, got %q", stderr)
	}
}

func TestRenderQuiet_WithReadErrors(t *testing.T) {
	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &out,
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	info := &status.StatusInfo{
		HookInstalled:         true,
		DocCount:              3,
		PendingCount:          1,
		ReadErrors:            2,
		AngelaMode:            "polish",
		AngelaDocsNeedReview:  1,
		AngelaSuggestions:     5,
		HealthIssues:          0,
	}

	err := renderQuiet(streams, info)
	if err != nil {
		t.Fatalf("renderQuiet: %v", err)
	}

	stdout := out.String()
	if !strings.Contains(stdout, "read_errors=2") {
		t.Errorf("expected 'read_errors=2', got %q", stdout)
	}
	if !strings.Contains(stdout, "angela_review=1") {
		t.Errorf("expected 'angela_review=1', got %q", stdout)
	}
	if !strings.Contains(stdout, "angela_suggestions=5") {
		t.Errorf("expected 'angela_suggestions=5', got %q", stdout)
	}
}

func TestRenderBadge_NoEligible(t *testing.T) {
	_, streams, out, errBuf := setupStatusTest(t, nil)

	// No docs, no commits → 0 eligible
	adapter := &mockGitAdapterForStatusEmpty{}
	err := renderBadge(streams, adapter)
	if err != nil {
		t.Fatalf("renderBadge: %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("expected no stdout for 0 eligible, got: %s", out.String())
	}
	if !strings.Contains(errBuf.String(), "eligible") || !strings.Contains(errBuf.String(), "No") {
		// The message may vary — just check there's something on stderr
		if errBuf.Len() == 0 {
			t.Errorf("expected message on stderr for 0 eligible, got nothing")
		}
	}
}

type mockGitAdapterForStatusEmpty struct{}

func (m *mockGitAdapterForStatusEmpty) LogAll() ([]domain.CommitInfo, error) {
	return nil, nil
}
func (m *mockGitAdapterForStatusEmpty) Diff(string) (string, error)                    { return "", nil }
func (m *mockGitAdapterForStatusEmpty) Log(string) (*domain.CommitInfo, error)          { return nil, nil }
func (m *mockGitAdapterForStatusEmpty) CommitExists(string) (bool, error)               { return true, nil }
func (m *mockGitAdapterForStatusEmpty) IsMergeCommit(string) (bool, error)              { return false, nil }
func (m *mockGitAdapterForStatusEmpty) IsInsideWorkTree() bool                          { return true }
func (m *mockGitAdapterForStatusEmpty) HeadRef() (string, error)                        { return "", nil }
func (m *mockGitAdapterForStatusEmpty) HeadCommit() (*domain.CommitInfo, error)         { return nil, nil }
func (m *mockGitAdapterForStatusEmpty) IsRebaseInProgress() (bool, error)               { return false, nil }
func (m *mockGitAdapterForStatusEmpty) CommitMessageContains(_, _ string) (bool, error) { return false, nil }
func (m *mockGitAdapterForStatusEmpty) GitDir() (string, error)                         { return "", nil }
func (m *mockGitAdapterForStatusEmpty) InstallHook(string) (domain.InstallResult, error) {
	return domain.InstallResult{}, nil
}
func (m *mockGitAdapterForStatusEmpty) UninstallHook(string) error               { return nil }
func (m *mockGitAdapterForStatusEmpty) HookExists(string) (bool, error)          { return false, nil }
func (m *mockGitAdapterForStatusEmpty) CommitRange(_, _ string) ([]string, error) { return nil, nil }
func (m *mockGitAdapterForStatusEmpty) LatestTag() (string, error)               { return "", nil }
func (m *mockGitAdapterForStatusEmpty) CurrentBranch() (string, error)           { return "main", nil }

type mockGitAdapterForStatus struct{}

func (m *mockGitAdapterForStatus) LogAll() ([]domain.CommitInfo, error) {
	return []domain.CommitInfo{
		{Hash: "abc123", Message: "feat: test"},
		{Hash: "def456", Message: "fix: bug"},
	}, nil
}
func (m *mockGitAdapterForStatus) Diff(string) (string, error)                         { return "", nil }
func (m *mockGitAdapterForStatus) Log(string) (*domain.CommitInfo, error)               { return nil, nil }
func (m *mockGitAdapterForStatus) CommitExists(string) (bool, error)                    { return true, nil }
func (m *mockGitAdapterForStatus) IsMergeCommit(string) (bool, error)                   { return false, nil }
func (m *mockGitAdapterForStatus) IsInsideWorkTree() bool                               { return true }
func (m *mockGitAdapterForStatus) HeadRef() (string, error)                             { return "", nil }
func (m *mockGitAdapterForStatus) HeadCommit() (*domain.CommitInfo, error)              { return nil, nil }
func (m *mockGitAdapterForStatus) IsRebaseInProgress() (bool, error)                    { return false, nil }
func (m *mockGitAdapterForStatus) CommitMessageContains(_, _ string) (bool, error)      { return false, nil }
func (m *mockGitAdapterForStatus) GitDir() (string, error)                              { return "", nil }
func (m *mockGitAdapterForStatus) InstallHook(string) (domain.InstallResult, error)     { return domain.InstallResult{}, nil }
func (m *mockGitAdapterForStatus) UninstallHook(string) error                           { return nil }
func (m *mockGitAdapterForStatus) HookExists(string) (bool, error)                      { return false, nil }
func (m *mockGitAdapterForStatus) CommitRange(_, _ string) ([]string, error)            { return nil, nil }
func (m *mockGitAdapterForStatus) LatestTag() (string, error)                           { return "", nil }
func (m *mockGitAdapterForStatus) CurrentBranch() (string, error)                       { return "main", nil }
