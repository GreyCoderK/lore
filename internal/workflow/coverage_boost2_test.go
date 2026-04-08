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
	"github.com/greycoderk/lore/internal/engagement"
	"github.com/greycoderk/lore/internal/storage"
	"gopkg.in/yaml.v3"
)

// --- HandleReactive end-to-end in a temp git repo with a commit ---

func TestHandleReactive_EndToEnd_WithRealCommit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newReactiveWorkDir(t)

	commit := &domain.CommitInfo{
		Hash:    "abcdef1234567890abcdef1234567890abcdef99",
		Author:  "TestDev",
		Date:    time.Now().UTC(),
		Message: "feat(api): add user endpoint",
		Type:    "feat",
		Scope:   "api",
		Subject: "add user endpoint",
	}
	adapter := &mockGitAdapter{headRef: "abcdef1234567890abcdef1234567890abcdef99", commit: commit}

	// TTY flow: default type (Enter), default what (Enter), why, skip alt, skip impact
	input := "\n\nBecause users need an API\n\n\n"
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(input),
		Out: &bytes.Buffer{},
		Err: stderr,
	}

	err := handleReactiveWithOpts(context.Background(), workDir, streams, adapter,
		DetectOpts{IsTTY: func(_ domain.IOStreams) bool { return true }}, nil)
	if err != nil {
		t.Fatalf("HandleReactive: %v", err)
	}

	// Verify a document was created
	entries, _ := os.ReadDir(filepath.Join(workDir, ".lore", "docs"))
	var docFound bool
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") && strings.HasPrefix(e.Name(), "feature-") {
			docFound = true
		}
	}
	if !docFound {
		t.Error("expected feature-*.md document to be created")
	}

	// Verify Captured in stderr
	if !strings.Contains(stderr.String(), "Captured") {
		t.Errorf("expected 'Captured' in stderr, got: %q", stderr.String())
	}
}

// --- runDocumentationFlow with a mock question flow (via reduced mode and prefill) ---

func TestRunDocumentationFlow_WithPrefill(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newReactiveWorkDir(t)

	commit := &domain.CommitInfo{
		Hash:    "abcdef1234567890abcdef1234567890abcdef88",
		Date:    time.Now().UTC(),
		Message: "fix(auth): patch token validation",
		Type:    "fix",
		Subject: "patch token validation",
	}

	// Reduced mode: only asks why (type and what are prefilled)
	input := "Because tokens were expiring too soon\n\n\n"
	streams := domain.IOStreams{
		In:  strings.NewReader(input),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	detection := DetectionResult{
		QuestionMode:  "reduced",
		PrefilledWhat: "fix token validation",
	}

	result, err := runDocumentationFlow(context.Background(), workDir, streams, commit, "", detection)
	if err != nil {
		t.Fatalf("runDocumentationFlow: %v", err)
	}
	if result.Filename == "" {
		t.Error("expected non-empty filename")
	}
	if !strings.HasPrefix(result.Filename, "bugfix-") {
		t.Errorf("expected bugfix- prefix, got %q", result.Filename)
	}
}

// --- SavePending: test saving and verifying file content ---

func TestSavePending_ContentVerification(t *testing.T) {
	workDir := t.TempDir()

	record := BuildPendingRecord(
		Answers{Type: "feature", What: "add auth", Why: "security"},
		"abc1234567890",
		"feat(auth): add authentication",
		"interrupted",
		"partial",
	)

	err := SavePending(workDir, record)
	if err != nil {
		t.Fatalf("SavePending: %v", err)
	}

	// Read the file and verify contents
	pendingDir := filepath.Join(workDir, ".lore", "pending")
	entries, err := os.ReadDir(pendingDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 pending file, got %d", len(entries))
	}

	data, err := os.ReadFile(filepath.Join(pendingDir, entries[0].Name()))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var parsed PendingRecord
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if parsed.Commit != "abc1234567890" {
		t.Errorf("Commit = %q, want abc1234567890", parsed.Commit)
	}
	if parsed.Message != "feat(auth): add authentication" {
		t.Errorf("Message = %q", parsed.Message)
	}
	if parsed.Status != "partial" {
		t.Errorf("Status = %q, want partial", parsed.Status)
	}
	if parsed.Reason != "interrupted" {
		t.Errorf("Reason = %q, want interrupted", parsed.Reason)
	}
	if parsed.Answers.Type != "feature" {
		t.Errorf("Answers.Type = %q, want feature", parsed.Answers.Type)
	}
	if parsed.Answers.What != "add auth" {
		t.Errorf("Answers.What = %q, want 'add auth'", parsed.Answers.What)
	}
	if parsed.Answers.Why != "security" {
		t.Errorf("Answers.Why = %q, want security", parsed.Answers.Why)
	}
}

func TestSavePending_EmptyCommitHash(t *testing.T) {
	workDir := t.TempDir()

	record := BuildPendingRecord(Answers{}, "", "", "interrupted", "partial")
	err := SavePending(workDir, record)
	if err != nil {
		t.Fatalf("SavePending with empty hash: %v", err)
	}

	pendingDir := filepath.Join(workDir, ".lore", "pending")
	entries, _ := os.ReadDir(pendingDir)
	if len(entries) != 1 {
		t.Fatalf("expected 1 pending file, got %d", len(entries))
	}
	// Filename should start with "unknown-"
	if !strings.HasPrefix(entries[0].Name(), "unknown-") {
		t.Errorf("expected unknown- prefix for empty hash, got %q", entries[0].Name())
	}
}

// --- ListPending with pending files present ---

func TestListPending_WithPendingFilesPresent(t *testing.T) {
	dir := t.TempDir()
	pendingDir := filepath.Join(dir, ".lore", "pending")

	now := time.Now().UTC()

	// Write 3 pending files
	for i, rec := range []PendingRecord{
		{
			Commit:  "aaa1111",
			Date:    now.Add(-1 * 24 * time.Hour).Format(time.RFC3339),
			Message: "feat: first",
			Answers: PendingAnswers{Type: "feature", What: "first"},
			Status:  "partial",
			Reason:  "interrupted",
		},
		{
			Commit:  "bbb2222",
			Date:    now.Add(-3 * 24 * time.Hour).Format(time.RFC3339),
			Message: "fix: second",
			Answers: PendingAnswers{Type: "bugfix", What: "second", Why: "broken"},
			Status:  "deferred",
			Reason:  "non-tty",
		},
		{
			Commit:  "ccc3333",
			Date:    now.Format(time.RFC3339),
			Message: "docs: third",
			Answers: PendingAnswers{},
			Status:  "deferred",
			Reason:  "non-tty",
		},
	} {
		_ = i
		writePendingFile(t, pendingDir, rec)
	}

	items, err := ListPending(context.Background(), pendingDir, nil)
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	// Sorted by date descending: ccc3333 (now), aaa1111 (1 day ago), bbb2222 (3 days ago)
	if items[0].CommitHash != "ccc3333" {
		t.Errorf("first item = %q, want ccc3333", items[0].CommitHash)
	}
	if items[1].CommitHash != "aaa1111" {
		t.Errorf("second item = %q, want aaa1111", items[1].CommitHash)
	}
	if items[2].CommitHash != "bbb2222" {
		t.Errorf("third item = %q, want bbb2222", items[2].CommitHash)
	}

	// Progress checks
	if items[0].Progress != "0/5" {
		t.Errorf("ccc3333 progress = %q, want 0/5", items[0].Progress)
	}
	if items[1].Progress != "2/5" {
		t.Errorf("aaa1111 progress = %q, want 2/5", items[1].Progress)
	}
	if items[2].Progress != "3/5" {
		t.Errorf("bbb2222 progress = %q, want 3/5", items[2].Progress)
	}
}

// --- showStarPrompt TTY and non-TTY paths ---

func TestShowStarPrompt_NonTTY_SilentNoOp(t *testing.T) {
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{Err: stderr}
	showStarPrompt(streams, t.TempDir(), t.TempDir(), false)
	if stderr.Len() > 0 {
		t.Errorf("non-TTY showStarPrompt should produce no output, got: %q", stderr.String())
	}
}

func TestShowStarPrompt_TTY_WithDocsAtThreshold(t *testing.T) {
	workDir := t.TempDir()
	docsDir := filepath.Join(workDir, ".lore", "docs")
	os.MkdirAll(docsDir, 0o755)
	// Also need .lore dir for state file
	os.MkdirAll(filepath.Join(workDir, ".lore"), 0o755)

	// Create 5 docs to reach the threshold
	for i := 0; i < 5; i++ {
		docType := []string{"note", "feature", "bugfix", "decision", "refactor"}[i]
		_, err := storage.WriteDoc(docsDir, domain.DocMeta{
			Type:   docType,
			Date:   "2026-04-07",
			Status: "published",
			Commit: strings.Repeat("b", 39) + string(rune('0'+i)),
		}, "unique test slug "+string(rune('a'+i)), "# Test\n\nBody.\n")
		if err != nil {
			t.Fatalf("WriteDoc[%d]: %v", i, err)
		}
	}

	// Ensure star prompt has not been shown yet
	statePath := engagement.StatePath(workDir)
	state := engagement.LoadState(statePath)
	if state.StarPromptShown {
		t.Skip("star prompt already shown in state")
	}

	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{Err: stderr}
	showStarPrompt(streams, workDir, docsDir, true)
	// We have exercised the TTY code path with docs at threshold.
	// The prompt may or may not appear depending on engagement config,
	// but the function should not panic.
}

func TestShowStarPrompt_TTY_ZeroDocs(t *testing.T) {
	workDir := t.TempDir()
	docsDir := filepath.Join(workDir, ".lore", "docs")
	os.MkdirAll(docsDir, 0o755)

	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{Err: stderr}
	showStarPrompt(streams, workDir, docsDir, true)
	// 0 docs, below threshold -> no prompt
	if strings.Contains(stderr.String(), "star") || strings.Contains(stderr.String(), "Star") {
		t.Errorf("should not show star prompt with 0 docs, got: %q", stderr.String())
	}
}

func TestShowStarPrompt_TTY_AlreadyShown(t *testing.T) {
	workDir := t.TempDir()
	docsDir := filepath.Join(workDir, ".lore", "docs")
	os.MkdirAll(docsDir, 0o755)
	os.MkdirAll(filepath.Join(workDir, ".lore"), 0o755)

	// Create 5 docs
	for i := 0; i < 5; i++ {
		docType := []string{"note", "feature", "bugfix", "decision", "refactor"}[i]
		_, _ = storage.WriteDoc(docsDir, domain.DocMeta{
			Type:   docType,
			Date:   "2026-04-07",
			Status: "published",
			Commit: strings.Repeat("c", 39) + string(rune('0'+i)),
		}, "shown test slug "+string(rune('a'+i)), "# Test\n\nBody.\n")
	}

	// Mark star prompt as already shown
	statePath := engagement.StatePath(workDir)
	state := engagement.EngagementState{StarPromptShown: true}
	_ = engagement.SaveState(statePath, state)

	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{Err: stderr}
	showStarPrompt(streams, workDir, docsDir, true)
	// Already shown -> no output
	if stderr.Len() > 0 {
		t.Errorf("should not show star prompt again, got: %q", stderr.String())
	}
}
