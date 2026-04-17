// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/testutil"
	"github.com/greycoderk/lore/internal/ui"
	"github.com/greycoderk/lore/internal/workflow"
)

// ═══════════════════════════════════════════════════════════════════════════
// Invariant I23 — Non-TTY pending resolve never hangs.
//
// Contract: `lore pending resolve` with a non-TTY stdin (pipe / closed /
// bytes buffer) MUST:
//   (a) never block waiting for interactive input,
//   (b) never print the interactive "Select item to resolve" prompt,
//   (c) auto-select a sensible default (most recent pending, first in
//       descending-date order) when multiple items exist,
//   (d) exit cleanly when no items are pending.
//
// Rationale: Angela runs in CI, git hooks, and IDE plugins where stdin
// is typically not a terminal. A hang there silently wedges the user's
// commit workflow with no feedback.
//
// Layer 1 (pre-existing):
//   - cmd/pending_test.go::TestPendingResolve_SingleItemAutoSelect
//     (single item + non-TTY auto-selects without prompting)
// Layer 2 (below): explicit anchors with timeout-bounded execution and
// stderr inspection for the multi-item case that the Layer-1 test does
// not exercise.
// ═══════════════════════════════════════════════════════════════════════════

// TestI23_PendingResolveNonTTYMultipleItemsDoesNotHang is the core anchor:
// seeds 3 pending items, runs `resolve` with a closed stdin, and asserts
// the command returns within a hard timeout. A hang would fail the Angela
// contract for CI / hook contexts.
func TestI23_PendingResolveNonTTYMultipleItemsDoesNotHang(t *testing.T) {
	restore := ui.SaveAndDisableColor()
	defer restore()

	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	// 3 pending items with staggered dates — `items[0]` after sort(desc)
	// is "aaa1111" (most recent, offset -0h).
	for i, hash := range []string{"aaa1111deadbeef", "bbb2222deadbeef", "ccc3333deadbeef"} {
		writePending(t, dir, workflow.PendingRecord{
			Commit:  hash,
			Date:    time.Now().UTC().Add(-time.Duration(i) * time.Hour).Format(time.RFC3339),
			Message: fmt.Sprintf("feat: item %d", i+1),
			Status:  "partial",
			Reason:  "interrupted",
		})
	}

	var errBuf, outBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &outBuf,
		Err: &errBuf,
		In:  strings.NewReader(""), // empty, non-TTY
	}

	cmd := newPendingCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{"resolve"})

	done := make(chan struct{})
	go func() {
		// Execute will likely error (ResolvePending requires a real git repo),
		// but that's irrelevant to I23: we only assert it does not hang and
		// does not take the interactive-prompt branch.
		_ = cmd.Execute()
		close(done)
	}()

	select {
	case <-done:
		// OK — returned within the deadline.
	case <-time.After(5 * time.Second):
		t.Fatal("I23 violation: pending resolve hung on non-TTY stdin with multiple items (5s deadline exceeded)")
	}

	// Contract (b): the interactive "Select item to resolve" prompt is an
	// EN/FR branch that must NEVER fire when stdin is non-TTY.
	combined := errBuf.String() + outBuf.String()
	for _, promptMarker := range []string{
		"Select item to resolve",       // EN
		"Sélectionnez l'élément",       // FR
	} {
		if strings.Contains(combined, promptMarker) {
			t.Errorf("I23 violation: non-TTY branch emitted interactive prompt %q — output:\n%s",
				promptMarker, combined)
		}
	}
}

// TestI23_PendingResolveNoItemsReturnsCleanly covers the zero-items edge:
// empty pending dir + non-TTY stdin must exit nil (no error, clean message),
// not hang reading from stdin looking for a selection that does not apply.
func TestI23_PendingResolveNoItemsReturnsCleanly(t *testing.T) {
	restore := ui.SaveAndDisableColor()
	defer restore()

	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	cmd := newPendingCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{"resolve"})

	errCh := make(chan error, 1)
	go func() { errCh <- cmd.Execute() }()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("I23 violation: empty pending dir should exit cleanly, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("I23 violation: pending resolve hung on empty pending list")
	}

	// Contract: a "no pending documentation" message is emitted so the user
	// knows the command completed intentionally (not silent-no-op).
	if errBuf.Len() == 0 {
		t.Error("I23 violation: empty pending list produced no user-visible feedback (silent no-op)")
	}
}

// TestI23_PendingResolveNonTTYInvalidArgExitsCleanly guards the arg-parse
// error path: non-TTY + invalid item number must fail fast with a clean
// exit code, never falling through into an interactive read. This is the
// anti-regression anchor for a refactor that accidentally swaps the order
// of "validate arg" and "detect TTY".
func TestI23_PendingResolveNonTTYInvalidArgExitsCleanly(t *testing.T) {
	restore := ui.SaveAndDisableColor()
	defer restore()

	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	// Seed 2 items so the arg validator has a range [1..2] to reject against.
	for i, hash := range []string{"aaa1111deadbeef", "bbb2222deadbeef"} {
		writePending(t, dir, workflow.PendingRecord{
			Commit:  hash,
			Date:    time.Now().UTC().Add(-time.Duration(i) * time.Hour).Format(time.RFC3339),
			Message: fmt.Sprintf("feat: x%d", i),
			Status:  "partial",
			Reason:  "interrupted",
		})
	}

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	cmd := newPendingCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{"resolve", "99"}) // out of range

	errCh := make(chan error, 1)
	go func() { errCh <- cmd.Execute() }()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("I23 violation: invalid item number should return a non-nil error, got nil")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("I23 violation: invalid-arg path hung on non-TTY stdin")
	}

	// Must NOT have emitted the interactive prompt on its way to the error.
	if strings.Contains(errBuf.String(), "Select item to resolve") ||
		strings.Contains(errBuf.String(), "Sélectionnez l'élément") {
		t.Errorf("I23 violation: invalid-arg path leaked into interactive prompt: %q", errBuf.String())
	}
}
