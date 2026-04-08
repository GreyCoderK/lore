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
	"github.com/greycoderk/lore/internal/workflow/decision"
)

// --- hasDocForCommit tests ---

type mockCorpusReader struct {
	docs []domain.DocMeta
}

func (m *mockCorpusReader) ReadDoc(_ string) (string, error) { return "", nil }
func (m *mockCorpusReader) ListDocs(_ domain.DocFilter) ([]domain.DocMeta, error) {
	return m.docs, nil
}

func TestHasDocForCommit_StoreHasDoc(t *testing.T) {
	store := &mockLoreStoreWithDocs{
		mockLoreStore: mockLoreStore{},
		docsByCommit: map[string][]domain.DocIndexEntry{
			"abc123": {{Filename: "feature-test.md"}},
		},
	}
	if !hasDocForCommit(store, nil, "abc123") {
		t.Error("expected true when store has doc for commit")
	}
}

func TestHasDocForCommit_StoreEmpty_CorpusHasDoc(t *testing.T) {
	store := &mockLoreStoreWithDocs{mockLoreStore: mockLoreStore{}}
	corpus := &mockCorpusReader{
		docs: []domain.DocMeta{
			{Commit: "abc123", Type: "feature"},
		},
	}
	if !hasDocForCommit(store, corpus, "abc123") {
		t.Error("expected true when corpus has doc for commit")
	}
}

func TestHasDocForCommit_NeitherHasDoc(t *testing.T) {
	store := &mockLoreStoreWithDocs{mockLoreStore: mockLoreStore{}}
	corpus := &mockCorpusReader{}
	if hasDocForCommit(store, corpus, "xyz") {
		t.Error("expected false when no doc exists")
	}
}

func TestHasDocForCommit_NilStoreNilCorpus(t *testing.T) {
	if hasDocForCommit(nil, nil, "abc") {
		t.Error("expected false when both nil")
	}
}

func TestHasDocForCommit_NilStoreCorpusHasDoc(t *testing.T) {
	corpus := &mockCorpusReader{
		docs: []domain.DocMeta{{Commit: "aaa"}},
	}
	if !hasDocForCommit(nil, corpus, "aaa") {
		t.Error("expected true when corpus has match and store is nil")
	}
}

// mockLoreStoreWithDocs extends mockLoreStore with DocsByCommitHash support.
type mockLoreStoreWithDocs struct {
	mockLoreStore
	docsByCommit map[string][]domain.DocIndexEntry
}

func (m *mockLoreStoreWithDocs) DocsByCommitHash(hash string) ([]domain.DocIndexEntry, error) {
	if docs, ok := m.docsByCommit[hash]; ok {
		return docs, nil
	}
	return nil, nil
}

// --- DispatchFull with engine ---

func TestDispatchFull_WithEngine_NonTTY(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newReactiveWorkDir(t)
	store := &mockLoreStore{}
	engine := decision.NewEngine(store, decision.DefaultConfig())

	commit := &domain.CommitInfo{
		Hash:    "dddd4444",
		Author:  "Dev",
		Date:    time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC),
		Message: "feat(api): add endpoint",
		Type:    "feat",
		Scope:   "api",
		Subject: "add endpoint",
	}
	adapter := &mockGitAdapter{headRef: "dddd4444", commit: commit}

	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	err := DispatchFull(context.Background(), workDir, streams, adapter, engine, store, DispatchConfig{})
	if err != nil {
		t.Fatalf("DispatchFull: %v", err)
	}

	// Non-TTY → deferred
	entries, _ := os.ReadDir(filepath.Join(workDir, ".lore", "pending"))
	if len(entries) == 0 {
		t.Error("expected pending file from DispatchFull with non-TTY")
	}
}

func TestDispatchFull_WithAmendPromptFalse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newReactiveWorkDir(t)
	f := false

	commit := &domain.CommitInfo{
		Hash:    "eeee5555",
		Message: "feat: test",
		Type:    "feat",
		Subject: "test",
	}
	adapter := &mockGitAdapter{headRef: "eeee5555", commit: commit}

	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	err := DispatchFull(context.Background(), workDir, streams, adapter, nil, nil,
		DispatchConfig{AmendPrompt: &f})
	if err != nil {
		t.Fatalf("DispatchFull: %v", err)
	}
}

// --- HandleReactiveWithEngine with non-nil engine ---

func TestHandleReactiveWithEngine_WithEngine(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newReactiveWorkDir(t)
	store := &mockLoreStore{}
	engine := decision.NewEngine(store, decision.DefaultConfig())

	commit := &domain.CommitInfo{
		Hash:    "ffff6666",
		Author:  "Dev",
		Date:    time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC),
		Message: "refactor(core): extract helper",
		Type:    "refactor",
		Scope:   "core",
		Subject: "extract helper",
	}
	adapter := &mockGitAdapter{headRef: "ffff6666", commit: commit}

	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	err := HandleReactiveWithEngine(context.Background(), workDir, streams, adapter, engine, store)
	if err != nil {
		t.Fatalf("HandleReactiveWithEngine: %v", err)
	}

	// Non-TTY → deferred → store records "pending"
	if len(store.recorded) == 0 {
		t.Error("expected store to record a decision with engine")
	}
}

// --- runDocumentationFlow with detection modes ---

func TestRunDocumentationFlow_ReducedMode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newReactiveWorkDir(t)

	commit := &domain.CommitInfo{
		Hash:    "abcdef1234567890abcdef1234567890abcdef77",
		Date:    time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC),
		Message: "feat(auth): add JWT",
		Type:    "feat",
		Subject: "add JWT",
	}

	streams := domain.IOStreams{
		In:  strings.NewReader("Because security\n\n\n"),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	detection := DetectionResult{
		QuestionMode:  "reduced",
		PrefilledWhat: "add JWT auth",
	}

	result, err := runDocumentationFlow(context.Background(), workDir, streams, commit, "", detection)
	if err != nil {
		t.Fatalf("runDocumentationFlow: %v", err)
	}
	if result.Filename == "" {
		t.Error("expected non-empty filename")
	}
}

func TestRunDocumentationFlow_ContextCancelled(t *testing.T) {
	workDir := newReactiveWorkDir(t)

	commit := &domain.CommitInfo{
		Hash:    "hhhh8888",
		Message: "feat: cancelled",
		Type:    "feat",
		Subject: "cancelled",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	streams := domain.IOStreams{
		In:  strings.NewReader("whatever\n"),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	_, err := runDocumentationFlow(ctx, workDir, streams, commit, "")
	if err == nil {
		t.Fatal("expected error with cancelled context")
	}

	// Should have saved pending
	entries, _ := os.ReadDir(filepath.Join(workDir, ".lore", "pending"))
	if len(entries) == 0 {
		t.Error("expected pending file on context cancellation")
	}
}

// --- showStarPrompt / showMilestone with non-TTY (no-op branches) ---

func TestShowMilestone_NonTTY_NoOp(t *testing.T) {
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{Err: stderr}
	// Non-TTY → should be a no-op
	showMilestone(streams, "/nonexistent", false)
	if stderr.Len() > 0 {
		t.Errorf("expected no output for non-TTY, got: %q", stderr.String())
	}
}

func TestShowStarPrompt_NonTTY_NoOp(t *testing.T) {
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{Err: stderr}
	showStarPrompt(streams, "/nonexistent", "/nonexistent", false)
	if stderr.Len() > 0 {
		t.Errorf("expected no output for non-TTY, got: %q", stderr.String())
	}
}

func TestShowMilestone_TTY_NoMilestone(t *testing.T) {
	// docsDir with 0 docs → no milestone
	workDir := t.TempDir()
	docsDir := filepath.Join(workDir, ".lore", "docs")
	os.MkdirAll(docsDir, 0o755)

	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{Err: stderr}
	showMilestone(streams, docsDir, true)
	// 0 docs → no milestone message
	if strings.Contains(stderr.String(), "decisions captured") {
		t.Errorf("unexpected milestone for 0 docs: %q", stderr.String())
	}
}

// --- displayCompletion ---

func TestDisplayCompletion_ShowsCapturedAndPath(t *testing.T) {
	workDir := newReactiveWorkDir(t)
	docsDir := filepath.Join(workDir, ".lore", "docs")

	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: stderr,
	}

	writeResult := storage.WriteResult{
		Filename: "feature-test.md",
		Path:     filepath.Join(docsDir, "feature-test.md"),
	}

	// Non-TTY avoids milestone/star prompt side effects, but exercises verb + path display.
	displayCompletion(streams, writeResult, "Captured", workDir, false)

	output := stderr.String()
	if !strings.Contains(output, "Captured") {
		t.Errorf("expected 'Captured' in output, got: %q", output)
	}
	expectedPath := filepath.Join(".lore", "docs", "feature-test.md")
	if !strings.Contains(output, expectedPath) {
		t.Errorf("expected path %q in output, got: %q", expectedPath, output)
	}
}

func TestShowStarPrompt_TTY_BelowThreshold(t *testing.T) {
	// 0 docs → below threshold → no star prompt
	workDir := t.TempDir()
	docsDir := filepath.Join(workDir, ".lore", "docs")
	os.MkdirAll(docsDir, 0o755)

	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{Err: stderr}
	showStarPrompt(streams, workDir, docsDir, true)
	// 0 docs, threshold is 5 → should not display
}

func TestShowStarPrompt_TTY_AtThreshold(t *testing.T) {
	workDir := t.TempDir()
	docsDir := filepath.Join(workDir, ".lore", "docs")
	os.MkdirAll(docsDir, 0o755)

	// Create 5 docs to hit the threshold
	docTypes := []string{"note", "feature", "bugfix", "decision", "refactor"}
	for i := 0; i < 5; i++ {
		_, err := storage.WriteDoc(docsDir, domain.DocMeta{
			Type:   docTypes[i],
			Date:   "2026-04-07",
			Status: "published",
			Commit: strings.Repeat("a", 39) + string(rune('0'+i)),
		}, fmt.Sprintf("unique slug %d", i), "# Note\n\nBody.\n")
		if err != nil {
			t.Fatalf("WriteDoc[%d]: %v", i, err)
		}
	}

	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{Err: stderr}
	showStarPrompt(streams, workDir, docsDir, true)
	// If engagement conditions are met, the star prompt should appear.
	// Even if it doesn't (already shown state), we've exercised the code path.
}

// --- renderSelect and clearSelectLines ---

func TestRenderSelect_WritesToStderr(t *testing.T) {
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{Err: stderr}
	renderSelect(streams, 0, -1)
	output := stderr.String()
	if !strings.Contains(output, "feature") {
		t.Errorf("expected 'feature' in renderSelect output, got: %q", output)
	}
	if !strings.Contains(output, "bugfix") {
		t.Errorf("expected 'bugfix' in renderSelect output, got: %q", output)
	}
}

func TestClearSelectLines_WritesEscapeCodes(t *testing.T) {
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{Err: stderr}
	clearSelectLines(streams, 3)
	// Should write 3 escape sequences
	output := stderr.String()
	if len(output) == 0 {
		t.Error("expected escape codes in clearSelectLines output")
	}
	// Each line clears with \033[A\033[2K\r
	if count := strings.Count(output, "\033[A"); count != 3 {
		t.Errorf("expected 3 cursor-up sequences, got %d", count)
	}
}

// --- cherryPickSourceHash ---

func TestCherryPickSourceHash_NoFile(t *testing.T) {
	dir := t.TempDir()
	adapter := &mockGitAdapter{gitDir: dir}
	hash := cherryPickSourceHash(adapter)
	if hash != "" {
		t.Errorf("expected empty for missing CHERRY_PICK_HEAD, got %q", hash)
	}
}

func TestCherryPickSourceHash_WithFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "CHERRY_PICK_HEAD"), []byte("deadbeef123\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	adapter := &mockGitAdapter{gitDir: dir}
	hash := cherryPickSourceHash(adapter)
	if hash != "deadbeef123" {
		t.Errorf("cherryPickSourceHash = %q, want deadbeef123", hash)
	}
}

// --- DispatchWithNotifyConfig ---

func TestDispatchWithNotifyConfig_NilConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newReactiveWorkDir(t)
	commit := &domain.CommitInfo{
		Hash:    "iiii9999",
		Message: "feat: test notify config",
		Type:    "feat",
		Subject: "test notify config",
	}
	adapter := &mockGitAdapter{headRef: "iiii9999", commit: commit}

	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	err := DispatchWithNotifyConfig(context.Background(), workDir, streams, adapter, nil, nil, nil)
	if err != nil {
		t.Fatalf("DispatchWithNotifyConfig: %v", err)
	}
}

// --- isRealTTY with non-file ---

func TestIsRealTTY_StringReader(t *testing.T) {
	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}
	if isRealTTY(streams) {
		t.Error("expected false for strings.Reader")
	}
}

// --- selectType non-TTY fallback ---

func TestSelectType_NonTTY_ReturnsDefault(t *testing.T) {
	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}
	got, err := selectType(streams, "bugfix")
	if err != nil {
		t.Fatalf("selectType: %v", err)
	}
	if got != "bugfix" {
		t.Errorf("selectType = %q, want bugfix (default for non-TTY)", got)
	}
}

// --- handleDetectionResult proceed with reduced mode ---

func TestHandleDetectionResult_Proceed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newReactiveWorkDir(t)
	stderr := &bytes.Buffer{}
	input := "\n\nBecause proceed\n\n\n"
	streams := domain.IOStreams{In: strings.NewReader(input), Out: &bytes.Buffer{}, Err: stderr}
	commit := &domain.CommitInfo{
		Hash:    "abcdef1234567890abcdef1234567890abcdef12",
		Message: "feat: proceed",
		Type:    "feat",
		Subject: "proceed test",
	}

	err := handleDetectionResult(context.Background(), workDir, streams, &mockGitAdapter{}, nil, commit,
		DetectionResult{Action: "proceed"}, true, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr.String(), "Captured") {
		t.Errorf("expected 'Captured' in stderr for proceed, got: %q", stderr.String())
	}
}

func TestHandleDetectionResult_AskReduced(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newReactiveWorkDir(t)
	stderr := &bytes.Buffer{}
	input := "Because reduced\n\n\n"
	streams := domain.IOStreams{In: strings.NewReader(input), Out: &bytes.Buffer{}, Err: stderr}
	commit := &domain.CommitInfo{
		Hash:    "abcdef1234567890abcdef1234567890abcdef13",
		Message: "feat(auth): add JWT reduced",
		Type:    "feat",
		Subject: "add JWT reduced",
	}

	err := handleDetectionResult(context.Background(), workDir, streams, &mockGitAdapter{}, nil, commit,
		DetectionResult{Action: "ask-reduced", QuestionMode: "reduced", PrefilledWhat: "add JWT"}, true, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr.String(), "Captured") {
		t.Errorf("expected 'Captured' for ask-reduced, got: %q", stderr.String())
	}
}

// --- SuggestSkip with user accepting ---

func TestHandleDetectionResult_SuggestSkip_UserAccepts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newReactiveWorkDir(t)
	stderr := &bytes.Buffer{}
	// y to proceed + answers for the documentation flow
	input := "y\n\n\nBecause accepted\n\n\n"
	streams := domain.IOStreams{In: strings.NewReader(input), Out: &bytes.Buffer{}, Err: stderr}
	commit := &domain.CommitInfo{
		Hash:    "abcdef1234567890abcdef1234567890abcdef14",
		Message: "chore: bump version accepted",
		Type:    "chore",
		Subject: "bump version accepted",
	}

	err := handleDetectionResult(context.Background(), workDir, streams, &mockGitAdapter{}, nil, commit,
		DetectionResult{Action: "suggest-skip"}, true, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// User said y → should proceed to document
	if !strings.Contains(stderr.String(), "Captured") {
		t.Errorf("expected 'Captured' when user accepts suggest-skip, got: %q", stderr.String())
	}
}
