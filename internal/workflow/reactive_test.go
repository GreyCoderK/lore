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

	err := handleReactiveWithOpts(context.Background(), workDir, streams, adapter, DetectOpts{IsTTY: func(_ domain.IOStreams) bool { return true }})
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

	err := handleReactiveWithOpts(ctx, workDir, streams, adapter, DetectOpts{IsTTY: func(_ domain.IOStreams) bool { return true }})
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

	err := Dispatch(context.Background(), workDir, streams, adapter)
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
	origHash := "original1234567"
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

	// Simulate: Enter (type default), Enter (what default), why, Enter (alt), Enter (impact)
	input := "\n\nAmended because of review\n\n\n"
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
	})
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
			Commit: fmt.Sprintf("precreate%d", i),
		}, fmt.Sprintf("precreated decision %d", i), fmt.Sprintf("# Pre %d\n\nBody.\n", i))
		if err != nil {
			t.Fatalf("setup WriteDoc[%d]: %v", i, err)
		}
	}

	commit := &domain.CommitInfo{
		Hash:    "milestone3hash",
		Author:  "Dev",
		Date:    time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		Message: "feat: milestone trigger",
		Type:    "feat",
		Subject: "milestone trigger",
	}
	adapter := &mockGitAdapter{headRef: "milestone3hash", commit: commit}

	input := "\n\nBecause milestone\n\n\n"
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(input),
		Out: &bytes.Buffer{},
		Err: stderr,
	}

	err := handleReactiveWithOpts(context.Background(), workDir, streams, adapter, DetectOpts{
		IsTTY: func(_ domain.IOStreams) bool { return true },
	})
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
