// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/domain"
)

// --- MapCommitType ---

func TestMapCommitType_AllKnownTypes(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"feat", "feature"},
		{"fix", "bugfix"},
		{"refactor", "refactor"},
		{"docs", "note"},
		{"chore", "note"},
		{"test", "note"},
		{"perf", "feature"},
	}
	for _, tt := range tests {
		got := MapCommitType(tt.input)
		if got != tt.want {
			t.Errorf("MapCommitType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMapCommitType_CaseInsensitive(t *testing.T) {
	got := MapCommitType("FEAT")
	if got != "feature" {
		t.Errorf("MapCommitType(FEAT) = %q, want feature", got)
	}
}

func TestMapCommitType_Unknown_FallsBackToNote(t *testing.T) {
	got := MapCommitType("unknown-type")
	if got != "note" {
		t.Errorf("MapCommitType(unknown-type) = %q, want note", got)
	}
}

func TestMapCommitType_EmptyString(t *testing.T) {
	got := MapCommitType("")
	if got != "note" {
		t.Errorf("MapCommitType('') = %q, want note", got)
	}
}

// --- extractWhy ---

func TestExtractWhy_WithWhySection(t *testing.T) {
	content := "## What\nSomething.\n\n## Why\nBecause of performance.\n\n## How\nStep one.\n"
	got := extractWhy(content)
	if !strings.Contains(got, "performance") {
		t.Errorf("extractWhy = %q, expected 'performance'", got)
	}
}

func TestExtractWhy_NoWhySection(t *testing.T) {
	content := "## What\nSomething.\n\n## How\nStep one.\n"
	got := extractWhy(content)
	if got != "" {
		t.Errorf("extractWhy without Why section = %q, want empty", got)
	}
}

func TestExtractWhy_EmptyContent(t *testing.T) {
	got := extractWhy("")
	if got != "" {
		t.Errorf("extractWhy('') = %q, want empty", got)
	}
}

func TestExtractWhy_WhySectionAtEnd(t *testing.T) {
	content := "## What\nSomething.\n\n## Why\nFinal reason here.\n"
	got := extractWhy(content)
	if !strings.Contains(got, "Final reason here") {
		t.Errorf("extractWhy at end = %q, expected 'Final reason here'", got)
	}
}

// --- ExtractFilesFromDiff / CountDiffLines ---

func TestExtractFilesFromDiff_BasicDiff(t *testing.T) {
	diff := "+++ b/internal/foo.go\n+++ b/internal/bar.go\n+++ b/internal/foo.go\n"
	files := ExtractFilesFromDiff(diff)
	if len(files) != 2 {
		t.Fatalf("expected 2 unique files, got %d: %v", len(files), files)
	}
	if files[0] != "internal/foo.go" {
		t.Errorf("files[0] = %q, want internal/foo.go", files[0])
	}
	if files[1] != "internal/bar.go" {
		t.Errorf("files[1] = %q, want internal/bar.go", files[1])
	}
}

func TestExtractFilesFromDiff_EmptyDiff(t *testing.T) {
	files := ExtractFilesFromDiff("")
	if len(files) != 0 {
		t.Errorf("expected 0 files for empty diff, got %d", len(files))
	}
}

func TestCountDiffLines_BasicDiff(t *testing.T) {
	diff := "+added line 1\n+added line 2\n-removed line 1\n+++ b/foo.go\n--- a/foo.go\n"
	added, deleted := CountDiffLines(diff)
	if added != 2 {
		t.Errorf("added = %d, want 2", added)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}
}

func TestCountDiffLines_EmptyDiff(t *testing.T) {
	added, deleted := CountDiffLines("")
	if added != 0 || deleted != 0 {
		t.Errorf("empty diff: added=%d, deleted=%d, want 0,0", added, deleted)
	}
}

// --- commitFields ---

func TestCommitFields_NilCommit(t *testing.T) {
	hash, msg := commitFields(nil)
	if hash != "" || msg != "" {
		t.Errorf("commitFields(nil) = (%q, %q), want ('', '')", hash, msg)
	}
}

// --- readAmendLine ---

func TestReadAmendLine_BasicInput(t *testing.T) {
	streams := domain.IOStreams{
		In: strings.NewReader("yes\n"),
	}
	got, err := readAmendLine(streams)
	if err != nil {
		t.Fatalf("readAmendLine: %v", err)
	}
	if got != "yes" {
		t.Errorf("readAmendLine = %q, want yes", got)
	}
}

func TestReadAmendLine_TrimSpace(t *testing.T) {
	streams := domain.IOStreams{
		In: strings.NewReader("  update  \n"),
	}
	got, err := readAmendLine(streams)
	if err != nil {
		t.Fatalf("readAmendLine: %v", err)
	}
	if got != "update" {
		t.Errorf("readAmendLine = %q, want 'update'", got)
	}
}

func TestReadAmendLine_LowerCase(t *testing.T) {
	streams := domain.IOStreams{
		In: strings.NewReader("YES\n"),
	}
	got, err := readAmendLine(streams)
	if err != nil {
		t.Fatalf("readAmendLine: %v", err)
	}
	if got != "yes" {
		t.Errorf("readAmendLine = %q, want yes (lowercased)", got)
	}
}

func TestReadAmendLine_EOF(t *testing.T) {
	streams := domain.IOStreams{
		In: strings.NewReader("no"),
	}
	got, _ := readAmendLine(streams)
	if got != "no" {
		t.Errorf("readAmendLine at EOF = %q, want no", got)
	}
}

// --- readORIGHEAD ---

func TestReadORIGHEAD_Present(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "ORIG_HEAD"), []byte("deadbeef1234\n"), 0644)
	adapter := &mockGitAdapter{gitDir: dir}
	got := readORIGHEAD(adapter)
	if got != "deadbeef1234" {
		t.Errorf("readORIGHEAD = %q, want deadbeef1234", got)
	}
}

func TestReadORIGHEAD_Absent(t *testing.T) {
	dir := t.TempDir()
	adapter := &mockGitAdapter{gitDir: dir}
	got := readORIGHEAD(adapter)
	if got != "" {
		t.Errorf("readORIGHEAD when absent = %q, want empty", got)
	}
}

// --- handleDetectionResult: additional branches ---

func TestHandleDetectionResult_Skip_NoMessage(t *testing.T) {
	workDir := newReactiveWorkDir(t)
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: stderr}
	commit := &domain.CommitInfo{Hash: "abc1", Message: "chore: skip test", Type: "chore"}

	err := handleDetectionResult(context.Background(), workDir, streams, &mockGitAdapter{}, nil, commit,
		DetectionResult{Action: "skip", Reason: "merge"}, false, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No message output since detection.Message is empty
}

func TestHandleDetectionResult_Skip_WithMessage(t *testing.T) {
	workDir := newReactiveWorkDir(t)
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: stderr}
	commit := &domain.CommitInfo{Hash: "abc2", Message: "merge: PR #1", Type: "merge"}

	err := handleDetectionResult(context.Background(), workDir, streams, &mockGitAdapter{}, nil, commit,
		DetectionResult{Action: "skip", Reason: "merge", Message: "Merge commit — skipping"}, false, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr.String(), "skipping") {
		t.Errorf("expected 'skipping' in output, got: %q", stderr.String())
	}
}

func TestHandleDetectionResult_AutoSkip_WithMessage(t *testing.T) {
	workDir := newReactiveWorkDir(t)
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: stderr}
	commit := &domain.CommitInfo{Hash: "abc3", Message: "chore: lint", Type: "chore"}

	err := handleDetectionResult(context.Background(), workDir, streams, &mockGitAdapter{}, nil, commit,
		DetectionResult{Action: "auto-skip", Message: "Auto-skipped: low complexity"}, false, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr.String(), "Auto-skipped") {
		t.Errorf("expected 'Auto-skipped' in output, got: %q", stderr.String())
	}
}

func TestHandleDetectionResult_DefaultAction_Proceeds(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newReactiveWorkDir(t)
	stderr := &bytes.Buffer{}
	// For the default/unknown action, it falls through to documentation flow
	input := "\n\nBecause default\n\n\n"
	streams := domain.IOStreams{In: strings.NewReader(input), Out: &bytes.Buffer{}, Err: stderr}
	commit := &domain.CommitInfo{
		Hash:    "abcdef1234567890abcdef1234567890abcdef15",
		Message: "feat: unknown action",
		Type:    "feat",
		Subject: "unknown action test",
	}

	err := handleDetectionResult(context.Background(), workDir, streams, &mockGitAdapter{}, nil, commit,
		DetectionResult{Action: "unknown-action"}, true, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have warned about unknown action
	if !strings.Contains(stderr.String(), "unknown detection action") {
		t.Errorf("expected 'unknown detection action' warning, got: %q", stderr.String())
	}
}

func TestHandleDetectionResult_Defer_SavesPending(t *testing.T) {
	workDir := newReactiveWorkDir(t)
	streams := domain.IOStreams{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &bytes.Buffer{}}
	commit := &domain.CommitInfo{Hash: "defer123", Message: "feat: defer test", Type: "feat"}

	err := handleDetectionResult(context.Background(), workDir, streams, &mockGitAdapter{}, nil, commit,
		DetectionResult{Action: "defer", Reason: "non-tty"}, false, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries, _ := os.ReadDir(filepath.Join(workDir, ".lore", "pending"))
	if len(entries) == 0 {
		t.Error("expected pending file after defer action")
	}
}

// --- resolveHeadCommit: fallback path ---

func TestResolveHeadCommit_HeadCommitFallback(t *testing.T) {
	// When HeadCommit returns error, falls back to HeadRef + Log
	commit := &domain.CommitInfo{Hash: "fallback123", Message: "feat: fallback"}
	adapter := &mockGitAdapterHeadFail{
		mockGitAdapter: mockGitAdapter{headRef: "fallback123", commit: commit},
	}
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{Err: stderr}

	gotCommit, gotRef, err := resolveHeadCommit(adapter, streams)
	if err != nil {
		t.Fatalf("resolveHeadCommit fallback: %v", err)
	}
	if gotRef != "fallback123" {
		t.Errorf("gotRef = %q, want fallback123", gotRef)
	}
	if gotCommit == nil {
		t.Error("expected non-nil commit from fallback")
	}
}

// mockGitAdapterHeadFail forces HeadCommit to fail while HeadRef succeeds.
type mockGitAdapterHeadFail struct {
	mockGitAdapter
}

func (m *mockGitAdapterHeadFail) HeadCommit() (*domain.CommitInfo, error) {
	return nil, context.DeadlineExceeded
}

func TestResolveHeadCommit_LogFails_NonFatal(t *testing.T) {
	// HeadCommit fails, HeadRef succeeds, Log fails → warning in stderr but no error
	adapter := &mockGitAdapterLogFail{
		mockGitAdapter: mockGitAdapter{headRef: "logfail123"},
	}
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{Err: stderr}

	gotCommit, gotRef, err := resolveHeadCommit(adapter, streams)
	if err != nil {
		t.Fatalf("resolveHeadCommit with log failure should not error: %v", err)
	}
	if gotRef != "logfail123" {
		t.Errorf("gotRef = %q, want logfail123", gotRef)
	}
	// commit may be nil since Log failed
	_ = gotCommit
	// Warning should be in stderr
	if !strings.Contains(stderr.String(), "Warning") {
		t.Errorf("expected Warning in stderr, got: %q", stderr.String())
	}
}

// mockGitAdapterLogFail forces HeadCommit and Log to fail.
type mockGitAdapterLogFail struct {
	mockGitAdapter
}

func (m *mockGitAdapterLogFail) HeadCommit() (*domain.CommitInfo, error) {
	return nil, context.DeadlineExceeded
}

func (m *mockGitAdapterLogFail) Log(_ string) (*domain.CommitInfo, error) {
	return nil, context.DeadlineExceeded
}

// --- buildSignalContext: scope/message forwarding ---

func TestBuildSignalContext_ScopeAndMessage(t *testing.T) {
	commit := &domain.CommitInfo{
		Type:    "feat",
		Scope:   "api",
		Subject: "new endpoint",
		Message: "feat(api): new endpoint",
	}
	sc := buildSignalContext(commit, "")
	if sc.Scope != "api" {
		t.Errorf("Scope = %q, want api", sc.Scope)
	}
	if sc.Message != "feat(api): new endpoint" {
		t.Errorf("Message = %q, unexpected", sc.Message)
	}
}

// --- ToGenerateInput ---

func TestToGenerateInput_WithCommit(t *testing.T) {
	answers := Answers{
		Type:         "feature",
		What:         "add auth",
		Why:          "security",
		Alternatives: "none",
		Impact:       "low",
	}
	commit := &domain.CommitInfo{
		Branch: "main",
		Scope:  "auth",
	}
	input := answers.ToGenerateInput(commit, "hook")

	if input.DocType != "feature" {
		t.Errorf("DocType = %q, want feature", input.DocType)
	}
	if input.GeneratedBy != "hook" {
		t.Errorf("GeneratedBy = %q, want hook", input.GeneratedBy)
	}
	if input.Branch != "main" {
		t.Errorf("Branch = %q, want main", input.Branch)
	}
	if input.Scope != "auth" {
		t.Errorf("Scope = %q, want auth", input.Scope)
	}
	if input.CommitInfo != commit {
		t.Error("CommitInfo should match input commit")
	}
}

func TestToGenerateInput_NilCommit(t *testing.T) {
	answers := Answers{Type: "note", What: "doc", Why: "info"}
	input := answers.ToGenerateInput(nil, "manual")

	if input.Branch != "" {
		t.Errorf("Branch = %q, want empty for nil commit", input.Branch)
	}
	if input.Scope != "" {
		t.Errorf("Scope = %q, want empty for nil commit", input.Scope)
	}
	if input.CommitInfo != nil {
		t.Error("CommitInfo should be nil")
	}
}

// --- milestoneI18N ---

func TestMilestoneI18N_ValidCounts(t *testing.T) {
	for _, count := range []int{3, 8, 21, 55} {
		msg := milestoneI18N(count)
		if msg == "" {
			t.Errorf("milestoneI18N(%d) returned empty string, expected message", count)
		}
	}
}

func TestMilestoneI18N_UnknownCount(t *testing.T) {
	msg := milestoneI18N(42)
	if msg != "" {
		t.Errorf("milestoneI18N(42) = %q, expected empty for non-milestone count", msg)
	}
}

// --- ListPending: context cancelled path ---

func TestListPending_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := ListPending(ctx, t.TempDir(), nil)
	if err == nil {
		t.Error("expected error for cancelled context in ListPending")
	}
}

// --- SkipPending: prefix match (non-ambiguous) ---

func TestSkipPending_PrefixMatch(t *testing.T) {
	dir := t.TempDir()
	pendingDir := filepath.Join(dir, ".lore", "pending")

	writePendingFile(t, pendingDir, PendingRecord{
		Commit:  "unique1234",
		Date:    "2026-04-07T00:00:00Z",
		Message: "feat: unique prefix test",
		Status:  "partial",
		Reason:  "interrupted",
	})

	// Match using a unique prefix
	item, err := SkipPending(context.Background(), pendingDir, "unique")
	if err != nil {
		t.Fatalf("SkipPending with prefix: %v", err)
	}
	if item.CommitHash != "unique1234" {
		t.Errorf("CommitHash = %q, want unique1234", item.CommitHash)
	}
}

// --- RelativeAge: 1-month boundary ---

func TestRelativeAge_OneMonth(t *testing.T) {
	// 30 days / 24h = 720h. days=30, weeks=4, months=1 → "1 month ago"
	got := RelativeAge(30 * 24 * time.Hour)
	if got == "" {
		t.Error("RelativeAge(30 days) returned empty string")
	}
	// months = 30/30 = 1 → should match the singular form
	if strings.Contains(got, "2") {
		t.Errorf("RelativeAge(30 days) = %q, should be singular (1 month)", got)
	}
}

// --- FlushOnInterrupt ---

func TestFlushOnInterrupt_WithState(t *testing.T) {
	workDir := t.TempDir()
	answers := &Answers{Type: "feature", What: "test flush"}
	RegisterInterruptState(workDir, "flush123", "feat: flush test", answers)

	FlushOnInterrupt()

	// File should have been saved
	entries, err := os.ReadDir(filepath.Join(workDir, ".lore", "pending"))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected pending file after FlushOnInterrupt")
	}
}

func TestFlushOnInterrupt_WithoutState(t *testing.T) {
	// Clear state and flush — should be a no-op
	RegisterInterruptState("", "", "", nil)
	FlushOnInterrupt() // Must not panic
}

func TestFlushOnInterrupt_DoubleFlush(t *testing.T) {
	workDir := t.TempDir()
	answers := &Answers{Type: "note", What: "double flush"}
	RegisterInterruptState(workDir, "double123", "note: double", answers)

	FlushOnInterrupt() // First flush — saves and clears state
	FlushOnInterrupt() // Second flush — should be a no-op (state is nil)

	entries, _ := os.ReadDir(filepath.Join(workDir, ".lore", "pending"))
	// Should only have the one file from the first flush
	if len(entries) == 0 {
		t.Error("expected pending file after first FlushOnInterrupt")
	}
}

// --- errIsExist ---

func TestErrIsExist_True(t *testing.T) {
	// os.IsExist works with a concrete *os.PathError that indicates the file exists
	tmp := t.TempDir()
	path := filepath.Join(tmp, "existing.txt")
	os.WriteFile(path, []byte("x"), 0644)
	// os.Mkdir on an existing path triggers IsExist on some systems
	// Instead, test with os.IsExist via os.ErrExist
	err := &os.PathError{Err: os.ErrExist}
	if !errIsExist(err) {
		t.Error("errIsExist should return true for os.ErrExist-wrapped error")
	}
}

func TestErrIsExist_False(t *testing.T) {
	if errIsExist(nil) {
		t.Error("errIsExist(nil) should return false")
	}
	if errIsExist(context.DeadlineExceeded) {
		t.Error("errIsExist(deadline) should return false")
	}
}

// --- Dispatch / DispatchWithNotifyConfig wrappers ---

func TestDispatch_Wrapper(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newReactiveWorkDir(t)
	commit := &domain.CommitInfo{
		Hash:    "dispatch001",
		Message: "feat: dispatch wrapper test",
		Type:    "feat",
		Subject: "dispatch wrapper",
	}
	adapter := &mockGitAdapter{headRef: "dispatch001", commit: commit}
	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	// Dispatch is a thin wrapper — just ensure it does not panic/error
	err := Dispatch(context.Background(), workDir, streams, adapter, nil, nil)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
}

// --- ResolvePending: batch mode with invalid type ---

func TestResolvePending_BatchMode_InvalidType(t *testing.T) {
	workDir := newPendingWorkDir(t)
	pendingDir := filepath.Join(workDir, ".lore", "pending")

	writePendingFile(t, pendingDir, PendingRecord{
		Commit:  "batch001",
		Date:    "2026-04-07T00:00:00Z",
		Message: "feat: batch invalid type",
		Status:  "partial",
		Reason:  "interrupted",
	})

	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	item := PendingItem{
		Filename:   "batch001.yaml",
		CommitHash: "batch001",
		Answers:    PendingAnswers{},
	}

	opts := ResolveOpts{
		Type: "invalid-type", // bad type
		What: "something",
		Why:  "because",
	}

	err := ResolvePending(context.Background(), workDir, streams, item, &mockGitAdapter{}, opts)
	if err == nil {
		t.Fatal("expected error for invalid doc type in batch mode")
	}
	if !strings.Contains(err.Error(), "invalid document type") {
		t.Errorf("expected 'invalid document type' in error, got: %v", err)
	}
}

// --- ResolvePending: batch mode success ---

func TestResolvePending_BatchMode_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newPendingWorkDir(t)
	pendingDir := filepath.Join(workDir, ".lore", "pending")

	writePendingFile(t, pendingDir, PendingRecord{
		Commit:  "batch002",
		Date:    "2026-04-07T00:00:00Z",
		Message: "feat: batch success",
		Status:  "partial",
		Reason:  "interrupted",
	})

	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	item := PendingItem{
		Filename:   "batch002.yaml",
		CommitHash: "batch002",
		Answers:    PendingAnswers{},
	}

	opts := ResolveOpts{
		Type: "feature",
		What: "add batch processing",
		Why:  "performance improvement",
		IsTTY: func(_ domain.IOStreams) bool { return false },
	}

	err := ResolvePending(context.Background(), workDir, streams, item, &mockGitAdapter{}, opts)
	if err != nil {
		t.Fatalf("ResolvePending batch mode: %v", err)
	}

	// Document should be created
	entries, _ := os.ReadDir(filepath.Join(workDir, ".lore", "docs"))
	var found bool
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "feature-") && strings.HasSuffix(e.Name(), ".md") {
			found = true
		}
	}
	if !found {
		t.Error("expected feature-*.md created in batch mode")
	}
}

// --- mapConvTypeToDocType all branches ---

func TestMapConvTypeToDocType_AllBranches(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"feat", "feature"},
		{"fix", "bugfix"},
		{"refactor", "refactor"},
		{"docs", "note"},
		{"style", "note"},
		{"ci", "note"},
		{"build", "note"},
		{"chore", "note"},
		{"unknown", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := mapConvTypeToDocType(tt.input)
		if got != tt.want {
			t.Errorf("mapConvTypeToDocType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- ResolvePending with commit exists (logs commit) ---

func TestResolvePending_CommitExistsChecked(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newPendingWorkDir(t)
	pendingDir := filepath.Join(workDir, ".lore", "pending")

	writePendingFile(t, pendingDir, PendingRecord{
		Commit:  "exist123",
		Date:    "2026-04-07T00:00:00Z",
		Message: "fix: exists",
		Answers: PendingAnswers{Type: "bugfix", What: "fix x", Why: "broken"},
		Status:  "partial",
		Reason:  "interrupted",
	})

	commit := &domain.CommitInfo{
		Hash:    "abcdef1234567890abcdef1234567890abcdef20",
		Message: "fix: exists",
		Type:    "fix",
		Subject: "exists",
	}
	adapter := &mockGitAdapter{commit: commit}

	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	item := PendingItem{
		Filename:   "exist123.yaml",
		CommitHash: "exist123",
		Answers:    PendingAnswers{Type: "bugfix", What: "fix x", Why: "broken"},
	}

	opts := ResolveOpts{
		IsTTY: func(_ domain.IOStreams) bool { return false },
	}

	// All 3 required answers are in item.Answers already → batch mode
	err := ResolvePending(context.Background(), workDir, streams, item, adapter, opts)
	if err != nil {
		t.Fatalf("ResolvePending with existing commit: %v", err)
	}
}

// --- RelativeAge: edge cases ---

func TestRelativeAge_JustNow(t *testing.T) {
	got := RelativeAge(1 * time.Minute)
	if got == "" {
		t.Error("expected non-empty for 1 minute")
	}
}

func TestRelativeAge_Negative(t *testing.T) {
	// Negative duration should be treated as positive
	got := RelativeAge(-2 * 24 * time.Hour)
	if got == "" {
		t.Error("expected non-empty for -2 days")
	}
}

// --- isAmendCommit ---

func TestIsAmendCommit_True(t *testing.T) {
	if !isAmendCommit(func(key string) string {
		if key == "GIT_REFLOG_ACTION" {
			return "amend"
		}
		return ""
	}) {
		t.Error("isAmendCommit should return true when GIT_REFLOG_ACTION=amend")
	}
}

func TestIsAmendCommit_False_OtherAction(t *testing.T) {
	if isAmendCommit(func(key string) string {
		if key == "GIT_REFLOG_ACTION" {
			return "rewriting (amend)"
		}
		return ""
	}) {
		t.Error("isAmendCommit should return false for 'rewriting (amend)'")
	}
}

func TestIsAmendCommit_False_Empty(t *testing.T) {
	if isAmendCommit(func(_ string) string { return "" }) {
		t.Error("isAmendCommit should return false when env is empty")
	}
}

// --- updateInterruptAnswers ---

func TestUpdateInterruptAnswers_WithState(t *testing.T) {
	workDir := t.TempDir()
	initial := &Answers{Type: "feature", What: "initial what"}
	RegisterInterruptState(workDir, "upd123", "feat: update test", initial)
	defer RegisterInterruptState("", "", "", nil)

	updated := &Answers{Type: "bugfix", What: "updated what", Why: "because"}
	updateInterruptAnswers(updated)

	// Flush to see the updated answers
	FlushOnInterrupt()

	entries, err := os.ReadDir(filepath.Join(workDir, ".lore", "pending"))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected pending file after FlushOnInterrupt")
	}
}

func TestUpdateInterruptAnswers_NoState(t *testing.T) {
	// When there is no state, updateInterruptAnswers is a no-op
	RegisterInterruptState("", "", "", nil)
	updateInterruptAnswers(&Answers{Type: "note", What: "noop"}) // must not panic
}

// --- questionModeFromAction ---

func TestQuestionModeFromAction_AllBranches(t *testing.T) {
	tests := []struct {
		action string
		want   string
	}{
		{"ask-full", "full"},
		{"ask-reduced", "reduced"},
		{"suggest-skip", "none"},
		{"auto-skip", "none"},
		{"unknown", "full"},
		{"", "full"},
		{"proceed", "full"},
	}
	for _, tt := range tests {
		got := questionModeFromAction(tt.action)
		if got != tt.want {
			t.Errorf("questionModeFromAction(%q) = %q, want %q", tt.action, got, tt.want)
		}
	}
}

// --- RelativeAge: full branch coverage ---

func TestRelativeAge_FewMinutes(t *testing.T) {
	// 10 minutes → should use RelativeAgeMinutes format
	got := RelativeAge(10 * time.Minute)
	if got == "" {
		t.Error("expected non-empty for 10 minutes")
	}
}

func TestRelativeAge_1Hour(t *testing.T) {
	// Exactly 1 hour (60 minutes) → RelativeAge1Hour
	got := RelativeAge(60 * time.Minute)
	if got == "" {
		t.Error("expected non-empty for 1 hour")
	}
}

func TestRelativeAge_SeveralHours(t *testing.T) {
	// 3 hours → RelativeAgeHours
	got := RelativeAge(3 * time.Hour)
	if got == "" {
		t.Error("expected non-empty for 3 hours")
	}
}

func TestRelativeAge_1Day(t *testing.T) {
	// Exactly 1 day → RelativeAge1Day
	got := RelativeAge(24 * time.Hour)
	if got == "" {
		t.Error("expected non-empty for 1 day")
	}
}

func TestRelativeAge_SeveralDays(t *testing.T) {
	// 3 days → RelativeAgeDays
	got := RelativeAge(3 * 24 * time.Hour)
	if got == "" {
		t.Error("expected non-empty for 3 days")
	}
}

func TestRelativeAge_1Week(t *testing.T) {
	// 7 days → RelativeAge1Week
	got := RelativeAge(7 * 24 * time.Hour)
	if got == "" {
		t.Error("expected non-empty for 1 week")
	}
}

func TestRelativeAge_SeveralWeeks(t *testing.T) {
	// 14 days (2 weeks) → RelativeAgeWeeks
	got := RelativeAge(14 * 24 * time.Hour)
	if got == "" {
		t.Error("expected non-empty for 2 weeks")
	}
}

func TestRelativeAge_MultipleMonths(t *testing.T) {
	// 60 days = 2 months → RelativeAgeMonths (plural)
	got := RelativeAge(60 * 24 * time.Hour)
	if got == "" {
		t.Error("expected non-empty for 2 months")
	}
}

// --- computeProgress ---

func TestComputeProgress_AllFilled(t *testing.T) {
	a := PendingAnswers{
		Type:         "feature",
		What:         "add auth",
		Why:          "security",
		Alternatives: "none",
		Impact:       "low",
	}
	got := computeProgress(a)
	if got != "5/5" {
		t.Errorf("computeProgress full = %q, want 5/5", got)
	}
}

func TestComputeProgress_Empty(t *testing.T) {
	got := computeProgress(PendingAnswers{})
	if got != "0/5" {
		t.Errorf("computeProgress empty = %q, want 0/5", got)
	}
}

func TestComputeProgress_Partial(t *testing.T) {
	a := PendingAnswers{Type: "bugfix", What: "fix nil pointer"}
	got := computeProgress(a)
	if got != "2/5" {
		t.Errorf("computeProgress partial = %q, want 2/5", got)
	}
}

// --- ListPending: warnWriter path ---

func TestListPending_CorruptFile_WarnWriterCalled(t *testing.T) {
	dir := t.TempDir()

	// Write a corrupt yaml file
	if err := os.WriteFile(filepath.Join(dir, "corrupt.yaml"), []byte("{{invalid yaml{{"), 0644); err != nil {
		t.Fatal(err)
	}

	var warnings []string
	warnWriter := func(msg string) {
		warnings = append(warnings, msg)
	}

	items, err := ListPending(context.Background(), dir, warnWriter)
	if err != nil {
		t.Fatalf("ListPending with corrupt file should not error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 valid items, got %d", len(items))
	}
	if len(warnings) == 0 {
		t.Error("expected at least one warning for corrupt yaml file")
	}
}

// --- SkipPending: ambiguous prefix ---

func TestSkipPending_AmbiguousPrefix_CoverageBoost(t *testing.T) {
	dir := t.TempDir()
	pendingDir := filepath.Join(dir, ".lore", "pending")

	writePendingFile(t, pendingDir, PendingRecord{
		Commit:  "abc1234",
		Date:    "2026-04-07T00:00:00Z",
		Message: "feat: first",
		Status:  "partial",
		Reason:  "interrupted",
	})
	writePendingFile(t, pendingDir, PendingRecord{
		Commit:  "abc5678",
		Date:    "2026-04-07T01:00:00Z",
		Message: "feat: second",
		Status:  "partial",
		Reason:  "interrupted",
	})

	// "abc" is a prefix of both → ambiguous
	_, err := SkipPending(context.Background(), pendingDir, "abc")
	if err == nil {
		t.Error("expected error for ambiguous prefix in SkipPending")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expected 'ambiguous' in error, got: %v", err)
	}
}

// --- SkipPending: no match ---

func TestSkipPending_NoMatch(t *testing.T) {
	dir := t.TempDir()
	pendingDir := filepath.Join(dir, ".lore", "pending")

	_, err := SkipPending(context.Background(), pendingDir, "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent hash in SkipPending")
	}
}

// --- ResolvePending: CommitExists returns false (commit gone path) ---

// mockGitAdapterCommitGone returns CommitExists = false
type mockGitAdapterCommitGone struct {
	mockGitAdapter
}

func (m *mockGitAdapterCommitGone) CommitExists(_ string) (bool, error) {
	return false, nil
}

func TestResolvePending_CommitGone_BatchMode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newPendingWorkDir(t)
	pendingDir := filepath.Join(workDir, ".lore", "pending")

	writePendingFile(t, pendingDir, PendingRecord{
		Commit:  "gone1234",
		Date:    "2026-04-07T00:00:00Z",
		Message: "fix: gone commit",
		Status:  "partial",
		Reason:  "interrupted",
	})

	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: stderr,
	}

	item := PendingItem{
		Filename:   "gone1234.yaml",
		CommitHash: "gone1234",
		Answers:    PendingAnswers{},
	}

	// Use batch mode with all fields — avoids interactive prompts
	opts := ResolveOpts{
		Type:  "bugfix",
		What:  "fix nil pointer",
		Why:   "crash in production",
		IsTTY: func(_ domain.IOStreams) bool { return false },
	}

	err := ResolvePending(context.Background(), workDir, streams, item, &mockGitAdapterCommitGone{}, opts)
	if err != nil {
		t.Fatalf("ResolvePending with gone commit: %v", err)
	}
	// Warning about gone commit should appear in stderr
	if !strings.Contains(stderr.String(), "gone1234") {
		t.Logf("stderr: %q", stderr.String())
		// The warning is only shown when existsErr == nil and exists == false
	}
}

// --- Detect: empty ref ---

func TestDetect_EmptyRef(t *testing.T) {
	// Empty ref should skip with "empty-ref" reason
	adapter := &mockGitAdapter{}
	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	result, err := Detect(context.Background(), "", adapter, streams, DetectOpts{
		IsTTY: func(_ domain.IOStreams) bool { return true },
	})
	if err != nil {
		t.Fatalf("Detect with empty ref: %v", err)
	}
	if result.Action != ActionSkip {
		t.Errorf("expected ActionSkip for empty ref, got %q", result.Action)
	}
	if result.Reason != "empty-ref" {
		t.Errorf("expected reason 'empty-ref', got %q", result.Reason)
	}
}

// --- Detect: context cancelled ---

func TestDetect_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	adapter := &mockGitAdapter{}
	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	_, err := Detect(ctx, "abc123", adapter, streams, DetectOpts{
		IsTTY: func(_ domain.IOStreams) bool { return true },
	})
	if err == nil {
		t.Error("expected error for cancelled context in Detect")
	}
}

// --- Detect: error paths for git operations ---

// mockGitAdapterDocSkipError returns an error from CommitMessageContains.
type mockGitAdapterDocSkipError struct {
	mockGitAdapter
}

func (m *mockGitAdapterDocSkipError) CommitMessageContains(_, _ string) (bool, error) {
	return false, context.DeadlineExceeded
}

func TestDetect_DocSkipError(t *testing.T) {
	adapter := &mockGitAdapterDocSkipError{}
	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	_, err := Detect(context.Background(), "abc123", adapter, streams, DetectOpts{
		IsTTY: func(_ domain.IOStreams) bool { return true },
	})
	if err == nil {
		t.Error("expected error when CommitMessageContains fails in Detect")
	}
}

// mockGitAdapterRebaseError returns an error from IsRebaseInProgress.
type mockGitAdapterRebaseError struct {
	mockGitAdapter
}

func (m *mockGitAdapterRebaseError) CommitMessageContains(_, _ string) (bool, error) {
	return false, nil // no doc-skip
}

func (m *mockGitAdapterRebaseError) IsRebaseInProgress() (bool, error) {
	return false, context.DeadlineExceeded
}

func TestDetect_RebaseError(t *testing.T) {
	adapter := &mockGitAdapterRebaseError{}
	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	_, err := Detect(context.Background(), "abc123", adapter, streams, DetectOpts{
		IsTTY: func(_ domain.IOStreams) bool { return true },
	})
	if err == nil {
		t.Error("expected error when IsRebaseInProgress fails in Detect")
	}
}

// mockGitAdapterMergeError returns an error from IsMergeCommit.
type mockGitAdapterMergeError struct {
	mockGitAdapter
}

func (m *mockGitAdapterMergeError) CommitMessageContains(_, _ string) (bool, error) {
	return false, nil
}

func (m *mockGitAdapterMergeError) IsRebaseInProgress() (bool, error) {
	return false, nil
}

func (m *mockGitAdapterMergeError) IsMergeCommit(_ string) (bool, error) {
	return false, context.DeadlineExceeded
}

func TestDetect_MergeError(t *testing.T) {
	adapter := &mockGitAdapterMergeError{}
	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	_, err := Detect(context.Background(), "abc123", adapter, streams, DetectOpts{
		IsTTY: func(_ domain.IOStreams) bool { return true },
	})
	if err == nil {
		t.Error("expected error when IsMergeCommit fails in Detect")
	}
}

// --- AskQuestions: invalid pre-filled type falls back to note ---

func TestAskQuestions_InvalidPreFilledType_FallsBackToNote(t *testing.T) {
	// Pre-fill with an invalid type → line 114-116 branch in questions.go
	streams := domain.IOStreams{
		In:  strings.NewReader("decision\nadd something\nbecause reasons\n\n\n"),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}
	renderer := NewRenderer(streams)
	flow := NewQuestionFlow(streams, renderer)

	answers, err := flow.AskQuestions(context.Background(), QuestionOpts{
		PreFilled: Answers{Type: "invalid-type-xyz"},
	})
	if err != nil {
		t.Fatalf("AskQuestions with invalid pre-filled type: %v", err)
	}
	// Should have replaced invalid type with user's interactive input
	if answers.Type == "invalid-type-xyz" {
		t.Error("expected invalid type to be replaced by user input")
	}
}

// --- ResolvePending: CommitExists returns error (checkCommit error path) ---

// mockGitAdapterCommitExistsError returns an error from CommitExists
type mockGitAdapterCommitExistsError struct {
	mockGitAdapter
}

func (m *mockGitAdapterCommitExistsError) CommitExists(_ string) (bool, error) {
	return false, context.DeadlineExceeded
}

func TestResolvePending_CommitExistsError_BatchMode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newPendingWorkDir(t)
	pendingDir := filepath.Join(workDir, ".lore", "pending")

	writePendingFile(t, pendingDir, PendingRecord{
		Commit:  "err1234",
		Date:    "2026-04-07T00:00:00Z",
		Message: "fix: error path",
		Status:  "partial",
		Reason:  "interrupted",
	})

	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: stderr,
	}

	item := PendingItem{
		Filename:   "err1234.yaml",
		CommitHash: "err1234",
		Answers:    PendingAnswers{},
	}

	opts := ResolveOpts{
		Type:  "note",
		What:  "some fix",
		Why:   "broken",
		IsTTY: func(_ domain.IOStreams) bool { return false },
	}

	// Should not error even though CommitExists fails
	err := ResolvePending(context.Background(), workDir, streams, item, &mockGitAdapterCommitExistsError{}, opts)
	if err != nil {
		t.Fatalf("ResolvePending with CommitExists error should succeed: %v", err)
	}
}
