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

// TestHandleReactive_DelegatesCorrectly verifies HandleReactive is a thin
// wrapper that produces the same result as calling HandleReactiveWithEngine
// with nil engine and nil store.
func TestHandleReactive_FullFlowEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newReactiveWorkDir(t)

	commit := &domain.CommitInfo{
		Hash:    "aaaa1111",
		Author:  "Dev",
		Date:    time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC),
		Message: "feat(api): add endpoint",
		Type:    "feat",
		Subject: "add endpoint",
	}
	adapter := &mockGitAdapter{headRef: "aaaa1111", commit: commit}

	// Provide answers: default type (Enter), default what (Enter), why, skip alt, skip impact
	input := "\n\nBecause REST\n\n\n"
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(input),
		Out: &bytes.Buffer{},
		Err: stderr,
	}

	err := HandleReactive(context.Background(), workDir, streams, adapter)
	if err != nil {
		t.Fatalf("HandleReactive: %v", err)
	}

	// Non-TTY streams → deferred to pending (non-tty detection)
	pendingDir := filepath.Join(workDir, ".lore", "pending")
	entries, readErr := os.ReadDir(pendingDir)
	if readErr != nil {
		t.Fatalf("ReadDir pending: %v", readErr)
	}
	if len(entries) == 0 {
		t.Error("expected pending file from HandleReactive with non-TTY streams")
	}
}

// TestHandleReactiveWithEngine_NilEngine verifies that passing nil engine
// does not cause a panic and produces the same result as HandleReactive.
func TestHandleReactiveWithEngine_NilEngine(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newReactiveWorkDir(t)

	commit := &domain.CommitInfo{
		Hash:    "bbbb2222",
		Author:  "Dev",
		Date:    time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC),
		Message: "fix: null pointer",
		Type:    "fix",
		Subject: "null pointer",
	}
	adapter := &mockGitAdapter{headRef: "bbbb2222", commit: commit}

	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	err := HandleReactiveWithEngine(context.Background(), workDir, streams, adapter, nil, nil)
	if err != nil {
		t.Fatalf("HandleReactiveWithEngine: %v", err)
	}

	// Non-TTY → deferred
	entries, _ := os.ReadDir(filepath.Join(workDir, ".lore", "pending"))
	if len(entries) == 0 {
		t.Error("expected pending file from HandleReactiveWithEngine with non-TTY")
	}
}

// TestHandleReactiveWithEngine_WithStore verifies that passing a store
// records the decision via recordDecision.
func TestHandleReactiveWithEngine_WithStore(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newReactiveWorkDir(t)
	store := &mockLoreStore{}

	commit := &domain.CommitInfo{
		Hash:    "cccc3333",
		Author:  "Dev",
		Date:    time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC),
		Message: "docs: update readme",
		Type:    "docs",
		Subject: "update readme",
	}
	adapter := &mockGitAdapter{headRef: "cccc3333", commit: commit}

	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	err := HandleReactiveWithEngine(context.Background(), workDir, streams, adapter, nil, store)
	if err != nil {
		t.Fatalf("HandleReactiveWithEngine: %v", err)
	}

	// Non-TTY → defer → recordDecision should be called with "pending"
	if len(store.recorded) == 0 {
		t.Error("expected store to record a decision")
	} else if store.recorded[0].Decision != "pending" {
		t.Errorf("Decision = %q, want pending", store.recorded[0].Decision)
	}
}

// TestHandleReactive_HeadRefError tests error propagation when git fails.
func TestHandleReactive_HeadRefError(t *testing.T) {
	workDir := newReactiveWorkDir(t)

	adapter := &mockGitAdapter{
		headErr: context.DeadlineExceeded,
		commit:  nil,
	}

	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	err := HandleReactive(context.Background(), workDir, streams, adapter)
	if err == nil {
		t.Fatal("expected error when git adapter fails")
	}
	if !strings.Contains(err.Error(), "head ref") {
		t.Errorf("error = %q, want to contain 'head ref'", err.Error())
	}
}
