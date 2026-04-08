// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/storage"
)

// mockGitAdapter implements domain.GitAdapter for testing.
type mockGitAdapter struct {
	headRef string
	commit  *domain.CommitInfo
	headErr error
	logErr  error
	gitDir  string
}

func (m *mockGitAdapter) HeadRef() (string, error)                              { return m.headRef, m.headErr }
func (m *mockGitAdapter) HeadCommit() (*domain.CommitInfo, error)               { return m.commit, m.headErr }
func (m *mockGitAdapter) Log(_ string) (*domain.CommitInfo, error)              { return m.commit, m.logErr }
func (m *mockGitAdapter) Diff(_ string) (string, error)                         { return "", nil }
func (m *mockGitAdapter) CommitExists(_ string) (bool, error)                   { return true, nil }
func (m *mockGitAdapter) IsMergeCommit(_ string) (bool, error)                  { return false, nil }
func (m *mockGitAdapter) IsInsideWorkTree() bool                                { return true }
func (m *mockGitAdapter) IsRebaseInProgress() (bool, error)                     { return false, nil }
func (m *mockGitAdapter) CommitMessageContains(_, _ string) (bool, error)       { return false, nil }
func (m *mockGitAdapter) GitDir() (string, error)                               { return m.gitDir, nil }
func (m *mockGitAdapter) InstallHook(_ string) (domain.InstallResult, error)    { return domain.InstallResult{}, nil }
func (m *mockGitAdapter) UninstallHook(_ string) error                          { return nil }
func (m *mockGitAdapter) HookExists(_ string) (bool, error)                     { return false, nil }
func (m *mockGitAdapter) CommitRange(_, _ string) ([]string, error)              { return nil, nil }
func (m *mockGitAdapter) LatestTag() (string, error)                             { return "", nil }
func (m *mockGitAdapter) LogAll() ([]domain.CommitInfo, error)                   { return nil, nil }
func (m *mockGitAdapter) CurrentBranch() (string, error)                         { return "main", nil }

// newReactiveWorkDir creates a minimal .lore directory structure under a temp dir.
func newReactiveWorkDir(t *testing.T) string {
	t.Helper()
	workDir := t.TempDir()
	for _, sub := range []string{".lore/docs", ".lore/templates"} {
		if err := os.MkdirAll(filepath.Join(workDir, sub), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", sub, err)
		}
	}
	return workDir
}

func TestHandleReactive_FullFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newReactiveWorkDir(t)

	commit := &domain.CommitInfo{
		Hash:    "abc1234",
		Author:  "Dev",
		Date:    time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC),
		Message: "feat(auth): add JWT",
		Type:    "feat",
		Subject: "add JWT",
	}
	adapter := &mockGitAdapter{headRef: "abc1234", commit: commit}

	// Simulate: Enter (type default=feature), Enter (what default=add JWT), why, Enter (no alt), Enter (no impact)
	input := "\n\nBecause JWT is stateless\n\n\n"
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(input),
		Out: &bytes.Buffer{},
		Err: stderr,
	}

	err := handleReactiveWithOpts(context.Background(), workDir, streams, adapter, DetectOpts{IsTTY: func(_ domain.IOStreams) bool { return true }}, nil)
	if err != nil {
		t.Fatalf("HandleReactive: %v", err)
	}

	// Verify a document was written under .lore/docs/
	entries, err := os.ReadDir(filepath.Join(workDir, ".lore", "docs"))
	if err != nil {
		t.Fatalf("ReadDir docs: %v", err)
	}
	var docFound bool
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") && strings.HasPrefix(e.Name(), "feature-") {
			docFound = true
		}
	}
	if !docFound {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("expected feature-*.md in docs, got: %v", names)
	}

	// Verify "Captured" appears in stderr output
	if !strings.Contains(stderr.String(), "Captured") {
		t.Errorf("expected 'Captured' in stderr output, got: %q", stderr.String())
	}
}

func TestHandleReactive_ContextCancelled_SavesPending(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newReactiveWorkDir(t)

	commit := &domain.CommitInfo{
		Hash:    "deadbeef",
		Message: "feat: interrupted",
		Type:    "feat",
		Subject: "interrupted",
	}
	adapter := &mockGitAdapter{headRef: "deadbeef", commit: commit}

	// Cancel context immediately — flow will fail with context.Canceled on first readLine
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	streams := domain.IOStreams{
		In:  strings.NewReader("any\n"),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	err := handleReactiveWithOpts(ctx, workDir, streams, adapter, DetectOpts{IsTTY: func(_ domain.IOStreams) bool { return true }}, nil)
	if err == nil {
		t.Fatal("expected error with cancelled context, got nil")
	}

	// A pending file should have been written
	pendingDir := filepath.Join(workDir, ".lore", "pending")
	entries, readErr := os.ReadDir(pendingDir)
	if readErr != nil {
		t.Fatalf("ReadDir pending: %v", readErr)
	}
	if len(entries) == 0 {
		t.Error("expected a pending file to be created on context cancellation")
	}
}

func TestDispatch_NonTTY_SavesPending(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newReactiveWorkDir(t)

	commit := &domain.CommitInfo{
		Hash:    "cafe1234",
		Type:    "docs",
		Subject: "update readme",
		Message: "docs: update readme",
		Date:    time.Now(),
	}
	adapter := &mockGitAdapter{headRef: "cafe1234", commit: commit}

	// Non-os.File streams → IsInteractiveTTY returns false → non-TTY path
	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	err := Dispatch(context.Background(), workDir, streams, adapter, nil, nil)
	if err != nil {
		t.Fatalf("Dispatch non-TTY: %v", err)
	}

	// Verify a pending file was created (not a doc)
	pendingDir := filepath.Join(workDir, ".lore", "pending")
	entries, readErr := os.ReadDir(pendingDir)
	if readErr != nil {
		t.Fatalf("ReadDir pending: %v", readErr)
	}
	if len(entries) == 0 {
		t.Error("expected a deferred pending file in non-TTY mode")
	}

	// Verify no docs were created (the interactive flow was NOT run)
	docsEntries, _ := os.ReadDir(filepath.Join(workDir, ".lore", "docs"))
	var mdCount int
	for _, e := range docsEntries {
		if strings.HasSuffix(e.Name(), ".md") {
			mdCount++
		}
	}
	if mdCount > 0 {
		t.Errorf("expected no docs in non-TTY mode, got %d", mdCount)
	}
}

// L3 fix: end-to-end test for amend with existing doc — verifies "Updated" verb.
func TestHandleReactive_AmendWithExistingDoc(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newReactiveWorkDir(t)
	docsDir := filepath.Join(workDir, ".lore", "docs")

	// Pre-create a doc for the original commit.
	origHash := "0123456789abcdef0123456789abcdef01234567"
	_, writeErr := storage.WriteDoc(docsDir, domain.DocMeta{
		Type:   "feature",
		Date:   "2026-03-10",
		Status: "published",
		Commit: origHash,
	}, "original feature", "# Original\n\nBody.\n")
	if writeErr != nil {
		t.Fatalf("setup WriteDoc: %v", writeErr)
	}

	// Set up git dir with ORIG_HEAD pointing to the original hash.
	gitDir := filepath.Join(workDir, ".git-test")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("mkdir gitDir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "ORIG_HEAD"), []byte(origHash+"\n"), 0o644); err != nil {
		t.Fatalf("write ORIG_HEAD: %v", err)
	}

	commit := &domain.CommitInfo{
		Hash:    "amended1234567",
		Author:  "Dev",
		Date:    time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
		Message: "feat: amended feature",
		Type:    "feat",
		Subject: "amended feature",
	}
	adapter := &mockGitAdapter{headRef: "amended1234567", commit: commit, gitDir: gitDir}

	// Simulate: Enter (Q0: yes), u (update), Enter (type default), Enter (what default), why, Enter (alt), Enter (impact)
	input := "\nu\n\n\nAmended because of review\n\n\n"
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(input),
		Out: &bytes.Buffer{},
		Err: stderr,
	}

	err := handleReactiveWithOpts(context.Background(), workDir, streams, adapter, DetectOpts{
		IsTTY: func(_ domain.IOStreams) bool { return true },
		GetEnv: func(key string) string {
			if key == "GIT_REFLOG_ACTION" {
				return "amend"
			}
			return ""
		},
	}, nil)
	if err != nil {
		t.Fatalf("HandleReactive amend: %v", err)
	}

	// Verify "Updated" verb appears in stderr (not "Captured")
	if !strings.Contains(stderr.String(), "Updated") {
		t.Errorf("expected 'Updated' in stderr for amend with existing doc, got: %q", stderr.String())
	}
}

// End-to-end test — pre-create 2 docs, run full reactive flow (creates 3rd),
// verify milestone-3 message appears in stderr AFTER the "Captured" line.
// TTY is threaded via DetectOpts.IsTTY.
func TestHandleReactive_MilestoneAtThreshold3(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newReactiveWorkDir(t)
	docsDir := filepath.Join(workDir, ".lore", "docs")

	// Pre-create 2 documents so the flow's 3rd document triggers the milestone.
	for i := 0; i < 2; i++ {
		_, err := storage.WriteDoc(docsDir, domain.DocMeta{
			Type:   "decision",
			Date:   "2026-03-15",
			Status: "published",
			Commit: fmt.Sprintf("bbb%037d", i),
		}, fmt.Sprintf("precreated decision %d", i), fmt.Sprintf("# Pre %d\n\nBody.\n", i))
		if err != nil {
			t.Fatalf("setup WriteDoc[%d]: %v", i, err)
		}
	}

	commit := &domain.CommitInfo{
		Hash:    "eee0000000000000000000000000000000000003",
		Author:  "Dev",
		Date:    time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		Message: "feat: milestone trigger",
		Type:    "feat",
		Subject: "milestone trigger",
	}
	adapter := &mockGitAdapter{headRef: "eee0000000000000000000000000000000000003", commit: commit}

	input := "\n\nBecause milestone\n\n\n"
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(input),
		Out: &bytes.Buffer{},
		Err: stderr,
	}

	err := handleReactiveWithOpts(context.Background(), workDir, streams, adapter, DetectOpts{
		IsTTY: func(_ domain.IOStreams) bool { return true },
	}, nil)
	if err != nil {
		t.Fatalf("HandleReactive milestone: %v", err)
	}

	output := stderr.String()
	if !strings.Contains(output, "3 decisions captured") {
		t.Errorf("expected milestone-3 message in stderr, got: %q", output)
	}

	// N8 fix: verify milestone appears AFTER "Captured" in output ordering.
	capturedIdx := strings.Index(output, "Captured")
	milestoneIdx := strings.Index(output, "3 decisions captured")
	if capturedIdx < 0 || milestoneIdx < 0 {
		t.Fatalf("missing expected strings in output: %q", output)
	}
	if milestoneIdx <= capturedIdx {
		t.Errorf("milestone message should appear after Captured line, got Captured at %d, milestone at %d", capturedIdx, milestoneIdx)
	}
}

// --- Pure function unit tests ---

func TestExtractFilesFromDiff(t *testing.T) {
	diff := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
+import "fmt"
diff --git a/util.go b/util.go
--- a/util.go
+++ b/util.go
@@ -1 +1 @@
-old
+new`

	files := ExtractFilesFromDiff(diff)
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(files), files)
	}
	if files[0] != "main.go" || files[1] != "util.go" {
		t.Errorf("files = %v, want [main.go util.go]", files)
	}
}

func TestExtractFilesFromDiff_Empty(t *testing.T) {
	files := ExtractFilesFromDiff("")
	if len(files) != 0 {
		t.Errorf("expected 0 files for empty diff, got %d", len(files))
	}
}

func TestExtractFilesFromDiff_Dedup(t *testing.T) {
	diff := "+++ b/main.go\n+++ b/main.go\n"
	files := ExtractFilesFromDiff(diff)
	if len(files) != 1 {
		t.Errorf("expected 1 file (deduplicated), got %d", len(files))
	}
}

func TestCountDiffLines(t *testing.T) {
	diff := `--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@
 context
-deleted line
+added line 1
+added line 2
 context`

	added, deleted := CountDiffLines(diff)
	if added != 2 {
		t.Errorf("added = %d, want 2", added)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}
}

func TestCountDiffLines_Empty(t *testing.T) {
	added, deleted := CountDiffLines("")
	if added != 0 || deleted != 0 {
		t.Errorf("expected 0/0 for empty diff, got %d/%d", added, deleted)
	}
}

func TestMapConvTypeToDocType(t *testing.T) {
	tests := []struct {
		conv string
		want string
	}{
		{"feat", "feature"},
		{"fix", "bugfix"},
		{"refactor", "refactor"},
		{"docs", "note"},
		{"style", "note"},
		{"ci", "note"},
		{"build", "note"},
		{"chore", "note"},
		{"", ""},
		{"unknown", ""},
	}
	for _, tt := range tests {
		got := mapConvTypeToDocType(tt.conv)
		if got != tt.want {
			t.Errorf("mapConvTypeToDocType(%q) = %q, want %q", tt.conv, got, tt.want)
		}
	}
}

func TestExtractWhy(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			"with why section",
			"# Title\n\n## Why\n\nBecause stateless auth.\n\n## Impact\n\nRoutes secured.\n",
			"Because stateless auth.",
		},
		{
			"no why section",
			"# Title\n\n## Impact\n\nSomething.\n",
			"",
		},
		{
			"why at end",
			"# Title\n\n## Why\n\nFinal section content.\n",
			"Final section content.",
		},
		{
			"multiline why",
			"# Title\n\n## Why\n\nLine one.\nLine two.\n\n## Impact\n",
			"Line one.\nLine two.",
		},
		{
			"empty",
			"",
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractWhy(tt.content)
			if got != tt.want {
				t.Errorf("extractWhy() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReadAmendAnswer(t *testing.T) {
	streams := domain.IOStreams{
		In:  strings.NewReader("u\n"),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}
	answer, err := readAmendAnswer(streams)
	if err != nil {
		t.Fatalf("readAmendAnswer: %v", err)
	}
	if answer != "u" {
		t.Errorf("answer = %q, want %q", answer, "u")
	}
}

func TestReadAmendAnswer_Empty(t *testing.T) {
	streams := domain.IOStreams{
		In:  strings.NewReader("\n"),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}
	answer, err := readAmendAnswer(streams)
	if err != nil {
		t.Fatalf("readAmendAnswer: %v", err)
	}
	if answer != "" {
		t.Errorf("answer = %q, want empty", answer)
	}
}

func TestHandleReactive_AmendQuestion0_Skip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newReactiveWorkDir(t)
	docsDir := filepath.Join(workDir, ".lore", "docs")

	origHash := "0123456789abcdef0123456789abcdef01234567"
	_, writeErr := storage.WriteDoc(docsDir, domain.DocMeta{
		Type:   "feature",
		Date:   "2026-03-10",
		Status: "published",
		Commit: origHash,
	}, "original feature", "# Original\n\nBody.\n")
	if writeErr != nil {
		t.Fatalf("setup WriteDoc: %v", writeErr)
	}

	gitDir := filepath.Join(workDir, ".git-test")
	os.MkdirAll(gitDir, 0o755)
	os.WriteFile(filepath.Join(gitDir, "ORIG_HEAD"), []byte(origHash+"\n"), 0o644)

	commit := &domain.CommitInfo{
		Hash:    "amended1234567",
		Author:  "Dev",
		Date:    time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
		Message: "feat: amended feature",
		Type:    "feat",
		Subject: "amended feature",
	}
	adapter := &mockGitAdapter{headRef: "amended1234567", commit: commit, gitDir: gitDir}

	// Answer "n" to Question 0 → skip amend entirely
	input := "n\n"
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(input),
		Out: &bytes.Buffer{},
		Err: stderr,
	}

	err := handleReactiveWithOpts(context.Background(), workDir, streams, adapter, DetectOpts{
		IsTTY: func(_ domain.IOStreams) bool { return true },
		GetEnv: func(key string) string {
			if key == "GIT_REFLOG_ACTION" {
				return "amend"
			}
			return ""
		},
	}, nil)
	if err != nil {
		t.Fatalf("HandleReactive amend skip: %v", err)
	}

	// Should NOT have "Updated" or "Captured" — skipped entirely
	output := stderr.String()
	if strings.Contains(output, "Updated") || strings.Contains(output, "Captured") {
		t.Errorf("expected no doc creation when Q0 answered 'n', got: %q", output)
	}
}

func TestQuestionModeFromAction(t *testing.T) {
	tests := []struct {
		action string
		want   string
	}{
		{"ask-full", "full"},
		{"ask-reduced", "reduced"},
		{"suggest-skip", "none"},
		{"auto-skip", "none"},
		{"proceed", "full"},
		{"", "full"},
	}
	for _, tt := range tests {
		got := questionModeFromAction(tt.action)
		if got != tt.want {
			t.Errorf("questionModeFromAction(%q) = %q, want %q", tt.action, got, tt.want)
		}
	}
}

// --- buildSignalContext unit tests ---

func TestBuildSignalContext_WithDiff(t *testing.T) {
	commit := &domain.CommitInfo{
		Type:    "feat",
		Scope:   "auth",
		Subject: "add JWT",
		Message: "feat(auth): add JWT",
	}
	diff := "+++ b/main.go\n+added line\n-removed line\n"
	ctx := buildSignalContext(commit, diff)

	if ctx.ConvType != "feat" {
		t.Errorf("ConvType = %q, want feat", ctx.ConvType)
	}
	if ctx.Scope != "auth" {
		t.Errorf("Scope = %q, want auth", ctx.Scope)
	}
	if len(ctx.FilesChanged) != 1 {
		t.Errorf("FilesChanged = %v, want 1 file", ctx.FilesChanged)
	}
	if ctx.LinesAdded != 1 {
		t.Errorf("LinesAdded = %d, want 1", ctx.LinesAdded)
	}
	if ctx.LinesDeleted != 1 {
		t.Errorf("LinesDeleted = %d, want 1", ctx.LinesDeleted)
	}
}

func TestBuildSignalContext_EmptyDiff(t *testing.T) {
	commit := &domain.CommitInfo{
		Type:    "fix",
		Subject: "typo",
		Message: "fix: typo",
	}
	ctx := buildSignalContext(commit, "")
	if len(ctx.FilesChanged) != 0 {
		t.Errorf("expected no files for empty diff, got %v", ctx.FilesChanged)
	}
	if ctx.LinesAdded != 0 || ctx.LinesDeleted != 0 {
		t.Errorf("expected 0/0 lines for empty diff")
	}
}

// --- resolveHeadCommit unit tests ---

func TestResolveHeadCommit_HeadCommitSuccess(t *testing.T) {
	commit := &domain.CommitInfo{Hash: "abc123", Message: "feat: test"}
	adapter := &mockGitAdapter{commit: commit}

	got, ref, err := resolveHeadCommit(adapter, domain.IOStreams{Err: &bytes.Buffer{}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Hash != "abc123" {
		t.Errorf("Hash = %q, want abc123", got.Hash)
	}
	if ref != "abc123" {
		t.Errorf("ref = %q, want abc123", ref)
	}
}

func TestResolveHeadCommit_FallbackWhenLogFails(t *testing.T) {
	// HeadCommit returns nil commit (triggers fallback to HeadRef + Log)
	// Log also fails → warning emitted but no fatal error
	adapter := &mockGitAdapter{
		headRef: "def456",
		headErr: nil,
		commit:  nil,   // nil commit triggers fallback
		logErr:  fmt.Errorf("log failed"),
	}

	stderr := &bytes.Buffer{}
	got, ref, err := resolveHeadCommit(adapter, domain.IOStreams{Err: stderr})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref != "def456" {
		t.Errorf("ref = %q, want def456", ref)
	}
	if got != nil {
		t.Errorf("expected nil commit when log fails, got %+v", got)
	}
	if !strings.Contains(stderr.String(), "Warning") {
		t.Errorf("expected warning in stderr, got: %q", stderr.String())
	}
}

func TestResolveHeadCommit_BothFail(t *testing.T) {
	adapter := &mockGitAdapter{
		headErr: fmt.Errorf("git failed"),
		headRef: "",
		commit:  nil,
	}

	stderr := &bytes.Buffer{}
	_, _, err := resolveHeadCommit(adapter, domain.IOStreams{Err: stderr})
	if err == nil {
		t.Fatal("expected error when both HeadCommit and HeadRef fail")
	}
}

// --- recordDecision unit tests ---

func TestRecordDecision_NilStore(t *testing.T) {
	commit := &domain.CommitInfo{Hash: "abc"}
	// Should not panic with nil store
	recordDecision(nil, commit, DetectionResult{}, "documented")
}

func TestRecordDecision_NilCommit(t *testing.T) {
	// Should not panic with nil commit
	recordDecision(nil, nil, DetectionResult{}, "documented")
}

// --- commitFields unit tests ---

func TestCommitFields_Nil(t *testing.T) {
	hash, msg := commitFields(nil)
	if hash != "" || msg != "" {
		t.Errorf("expected empty for nil, got %q %q", hash, msg)
	}
}

func TestCommitFields_WithCommit(t *testing.T) {
	commit := &domain.CommitInfo{Hash: "abc123", Message: "feat: test"}
	hash, msg := commitFields(commit)
	if hash != "abc123" {
		t.Errorf("hash = %q, want abc123", hash)
	}
	if msg != "feat: test" {
		t.Errorf("msg = %q, want 'feat: test'", msg)
	}
}

// --- readORIGHEAD unit tests ---

func TestReadORIGHEAD_GitDirError(t *testing.T) {
	adapter := &mockGitAdapter{gitDir: ""}
	// Mock returns empty string for gitDir; readORIGHEAD will fail on os.ReadFile
	result := readORIGHEAD(adapter)
	// Should return empty on error (non-existent path)
	if result != "" {
		t.Errorf("expected empty for missing ORIG_HEAD, got %q", result)
	}
}

// --- handleDetectionResult unit tests ---

func TestHandleDetectionResult_Skip(t *testing.T) {
	workDir := newReactiveWorkDir(t)
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: stderr}
	commit := &domain.CommitInfo{Hash: "abc123", Message: "merge: test"}

	err := handleDetectionResult(context.Background(), workDir, streams, &mockGitAdapter{}, nil, commit,
		DetectionResult{Action: "skip", Reason: "merge", Message: "Skipping merge commit"}, false, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr.String(), "Skipping merge commit") {
		t.Errorf("expected skip message in stderr, got: %q", stderr.String())
	}
}

func TestHandleDetectionResult_AutoSkip(t *testing.T) {
	workDir := newReactiveWorkDir(t)
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: stderr}
	commit := &domain.CommitInfo{Hash: "def456", Message: "chore: bump version"}

	err := handleDetectionResult(context.Background(), workDir, streams, &mockGitAdapter{}, nil, commit,
		DetectionResult{Action: "auto-skip", Message: "Auto-skipping chore"}, false, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr.String(), "Auto-skipping chore") {
		t.Errorf("expected auto-skip message, got: %q", stderr.String())
	}
}

func TestHandleDetectionResult_Defer(t *testing.T) {
	workDir := newReactiveWorkDir(t)
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: stderr}
	commit := &domain.CommitInfo{Hash: "ghi789", Message: "feat: deferred"}

	err := handleDetectionResult(context.Background(), workDir, streams, &mockGitAdapter{}, nil, commit,
		DetectionResult{Action: "defer", Reason: "rebase"}, false, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify pending file created
	entries, _ := os.ReadDir(filepath.Join(workDir, ".lore", "pending"))
	if len(entries) == 0 {
		t.Error("expected pending file to be created on defer")
	}
}

func TestHandleDetectionResult_SuggestSkip_UserDeclines(t *testing.T) {
	workDir := newReactiveWorkDir(t)
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{In: strings.NewReader("n\n"), Out: &bytes.Buffer{}, Err: stderr}
	commit := &domain.CommitInfo{Hash: "jkl012", Message: "chore: minor"}

	err := handleDetectionResult(context.Background(), workDir, streams, &mockGitAdapter{}, nil, commit,
		DetectionResult{Action: "suggest-skip"}, true, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr.String(), "Skipped") {
		t.Errorf("expected 'Skipped' message when user declines, got: %q", stderr.String())
	}
}

func TestHandleDetectionResult_SuggestSkip_ContextCancelled(t *testing.T) {
	workDir := newReactiveWorkDir(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	streams := domain.IOStreams{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &bytes.Buffer{}}
	commit := &domain.CommitInfo{Hash: "mno345", Message: "feat: cancelled"}

	err := handleDetectionResult(ctx, workDir, streams, &mockGitAdapter{}, nil, commit,
		DetectionResult{Action: "suggest-skip"}, true, nil, nil)
	if err == nil {
		t.Fatal("expected error with cancelled context")
	}
}

func TestHandleDetectionResult_UnknownAction(t *testing.T) {
	workDir := newReactiveWorkDir(t)
	stderr := &bytes.Buffer{}
	// Provide enough input for the full question flow
	input := "\n\nBecause reasons\n\n\n"
	streams := domain.IOStreams{In: strings.NewReader(input), Out: &bytes.Buffer{}, Err: stderr}
	commit := &domain.CommitInfo{Hash: "abcdef1234567890abcdef1234567890abcdef12", Message: "feat: unknown", Type: "feat", Subject: "unknown"}

	err := handleDetectionResult(context.Background(), workDir, streams, &mockGitAdapter{}, nil, commit,
		DetectionResult{Action: "weird-action"}, true, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr.String(), "Warning: unknown detection action") {
		t.Errorf("expected warning for unknown action, got: %q", stderr.String())
	}
}

// mockCommitStore records RecordCommit calls for verification.
type mockCommitStore struct {
	domain.LoreStore
	recorded []domain.CommitRecord
}

func (m *mockCommitStore) RecordCommit(rec domain.CommitRecord) error {
	m.recorded = append(m.recorded, rec)
	return nil
}

func TestRecordDecision_ValidRecord(t *testing.T) {
	store := &mockCommitStore{}
	commit := &domain.CommitInfo{
		Hash:    "abc123def456",
		Date:    time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC),
		Branch:  "main",
		Scope:   "api",
		Type:    "feat",
		Subject: "add endpoint",
		Message: "feat(api): add endpoint",
	}
	detection := DetectionResult{
		Action:       "ask-full",
		Score:        75,
		QuestionMode: "full",
		Reason:       "",
	}
	recordDecision(store, commit, detection, "documented")

	if len(store.recorded) != 1 {
		t.Fatalf("expected 1 recorded commit, got %d", len(store.recorded))
	}
	rec := store.recorded[0]
	if rec.Hash != "abc123def456" {
		t.Errorf("Hash = %q, want %q", rec.Hash, "abc123def456")
	}
	if rec.Decision != "documented" {
		t.Errorf("Decision = %q, want %q", rec.Decision, "documented")
	}
	if rec.DecisionScore != 75 {
		t.Errorf("DecisionScore = %d, want 75", rec.DecisionScore)
	}
	if rec.QuestionMode != "full" {
		t.Errorf("QuestionMode = %q, want %q", rec.QuestionMode, "full")
	}
	if rec.ConvType != "feat" {
		t.Errorf("ConvType = %q, want %q", rec.ConvType, "feat")
	}
}

func TestRecordDecision_SkipWithReason(t *testing.T) {
	store := &mockCommitStore{}
	commit := &domain.CommitInfo{Hash: "aaa111", Subject: "merge branch"}
	detection := DetectionResult{
		Action: "skip",
		Reason: "merge",
		Score:  10,
	}
	recordDecision(store, commit, detection, "merge-skipped")

	if len(store.recorded) != 1 {
		t.Fatalf("expected 1 recorded commit, got %d", len(store.recorded))
	}
	rec := store.recorded[0]
	if rec.Decision != "merge-skipped" {
		t.Errorf("Decision = %q, want %q", rec.Decision, "merge-skipped")
	}
	if rec.SkipReason != "merge" {
		t.Errorf("SkipReason = %q, want %q", rec.SkipReason, "merge")
	}
}

func TestRecordDecision_BothNil(t *testing.T) {
	// Should not panic when both store and commit are nil
	recordDecision(nil, nil, DetectionResult{}, "skipped")
}

func TestReadORIGHEAD_WithFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ORIG_HEAD"), []byte("abc123def\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	adapter := &mockGitAdapter{gitDir: dir}
	result := readORIGHEAD(adapter)
	if result != "abc123def" {
		t.Errorf("readORIGHEAD = %q, want abc123def", result)
	}
}
