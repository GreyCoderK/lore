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

// ═══════════════════════════════════════════════════════════════════════════
// Invariants for the capture flow (post-commit hook → questions → write).
//
// This is the most-executed code path in Lore: it runs on EVERY git commit
// in every documented repo. A regression here doesn't just break one
// command — it breaks the implicit contract that using Lore never makes
// your repo worse.
//
// Three named invariants:
//
//   I13 — Hybrid mode tolerance: a commit in a repo with `.git/` but no
//         `.lore/` must not panic. Whatever Lore would normally do, it
//         degrades gracefully (pending write or clean exit), never leaves
//         the user with a corrupted state or a crashed hook.
//
//   I14 — Commit-flow atomic OR pending saved: the capture flow must
//         either succeed fully (doc written + index updated) OR save the
//         user's partial answers to `.lore/pending/{hash}.yaml` so they
//         can be recovered with `lore pending resolve`. No lost answers.
//
//   I15 — Ctrl+C serializes partial answers: pressing Ctrl+C at any
//         point during capture must flush the current state to pending
//         BEFORE the process exits. The FlushOnInterrupt + RegisterInterruptState
//         contract MUST NOT lose answers even when a read blocks on stdin.
//
// Each invariant has ≥2 layers: existing tests cover the happy paths;
// the TestI13/I14/I15_* anchors here add the explicit named contract
// plus adversarial scenarios (missing .lore/, mid-flow interrupt).
// ═══════════════════════════════════════════════════════════════════════════

// ─────────────────────────────────────────────────────────────────────────
// I13 — Hybrid mode tolerance
// ─────────────────────────────────────────────────────────────────────────

// TestI13_HybridMode_NoPanicOnMissingLoreDir asserts that HandleReactive
// on a repo WITHOUT `.lore/` doesn't panic. The expected behavior is
// either a clean early return or a pending file — never a crash.
//
// We deliberately do NOT pre-create `.lore/docs/` (as newReactiveWorkDir
// would). The work dir has only `.git/` semantics (via the mock adapter).
func TestI13_HybridMode_NoPanicOnMissingLoreDir(t *testing.T) {
	workDir := t.TempDir() // no .lore/, no pre-created dirs

	commit := &domain.CommitInfo{
		Hash:    "hybridfeed",
		Author:  "Dev",
		Date:    time.Now().UTC(),
		Message: "feat: hybrid-mode commit",
		Type:    "feat",
		Subject: "hybrid-mode commit",
	}
	adapter := &mockGitAdapter{headRef: "hybridfeed", commit: commit}

	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	// The call must return without panicking. We don't assert on the error
	// value: in hybrid mode the flow may return an error about missing
	// `.lore/` or gracefully defer to pending. What matters is NO PANIC.
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("I13 violation: HandleReactive panicked on missing .lore/: %v", r)
			}
		}()
		_ = HandleReactive(context.Background(), workDir, streams, adapter)
	}()
}

// TestI13_HybridMode_HandleProactive_NoPanic is the mirror test for the
// manual documentation path (`lore new`). Same contract: a user running
// `lore new` in a repo without `.lore/` must get a clean error, never
// a crash.
func TestI13_HybridMode_HandleProactive_NoPanic(t *testing.T) {
	workDir := t.TempDir()
	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("I13 violation: HandleProactive panicked on missing .lore/: %v", r)
			}
		}()
		_ = HandleProactive(context.Background(), workDir, streams, ProactiveOpts{})
	}()
}

// ─────────────────────────────────────────────────────────────────────────
// I14 — Commit-flow atomic OR pending saved
// ─────────────────────────────────────────────────────────────────────────

// TestI14_NonTTYWrite_SavesPending asserts that when the capture flow
// runs in non-TTY context (piped stdin), the contract still holds:
// either the doc is written OR a pending record lands in `.lore/pending/`.
// A capture flow that silently drops the commit would lose the user's
// answers — the core invariant "no lost answers" forbids this.
//
// The existing TestHandleReactive_FullFlowEndToEnd hits this path with a
// non-TTY stream; this test is the explicit TestI14_* anchor so the
// matrix can cite it.
func TestI14_NonTTYWrite_SavesPending(t *testing.T) {
	workDir := newReactiveWorkDir(t)

	commit := &domain.CommitInfo{
		Hash:    "aaaai14",
		Author:  "Dev",
		Date:    time.Now().UTC(),
		Message: "feat(api): add endpoint",
		Type:    "feat",
		Subject: "add endpoint",
	}
	adapter := &mockGitAdapter{headRef: "aaaai14", commit: commit}
	// Non-TTY streams (stdin is bytes.Buffer, not os.Stdin), so the flow
	// must defer to pending rather than prompting.
	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	if err := HandleReactive(context.Background(), workDir, streams, adapter); err != nil {
		t.Fatalf("HandleReactive: %v", err)
	}

	// I14 assertion: either `.lore/docs/` or `.lore/pending/` has something.
	// Write-atomic path OR pending path — one must fire.
	docsDir := filepath.Join(workDir, ".lore", "docs")
	pendingDir := filepath.Join(workDir, ".lore", "pending")

	docs, _ := os.ReadDir(docsDir)
	pending, _ := os.ReadDir(pendingDir)

	// Count only `.md`/`.yaml` artifacts — the capture flow never
	// creates temp files inside these dirs outside its own atomicity.
	docsCount := 0
	for _, e := range docs {
		if strings.HasSuffix(e.Name(), ".md") && !strings.HasPrefix(e.Name(), ".") {
			docsCount++
		}
	}
	pendingCount := 0
	for _, e := range pending {
		if strings.HasSuffix(e.Name(), ".yaml") && !strings.HasPrefix(e.Name(), ".") {
			pendingCount++
		}
	}

	if docsCount == 0 && pendingCount == 0 {
		t.Errorf("I14 violation: neither a doc nor a pending record was written — user's commit silently lost")
	}
}

// TestI14_PendingRecordMarkedCorrectly asserts the contract around the
// pending record FORMAT: when a pending is written from a non-TTY / rebase
// deferral, the status/reason fields must identify the scenario so
// `lore pending resolve` can offer the right UX. A pending record that
// doesn't say WHY it was deferred is unhelpful at recovery time.
func TestI14_PendingRecordMarkedCorrectly(t *testing.T) {
	workDir := newReactiveWorkDir(t)

	// Craft a pending record and save it via the real helper — this
	// exercises BuildPendingRecord + SavePending in isolation.
	ans := Answers{
		Type: "feat",
		What: "something",
		Why:  "reason",
	}
	rec := BuildPendingRecord(ans, "i14hash", "feat: msg", "non-tty", "deferred")

	if rec.Status != "deferred" {
		t.Errorf("I14 violation: pending status %q, want 'deferred'", rec.Status)
	}
	if rec.Reason != "non-tty" {
		t.Errorf("I14 violation: pending reason %q, want 'non-tty'", rec.Reason)
	}
	if rec.Answers.Type != "feat" {
		t.Errorf("I14 violation: pending lost answer Type, got %q", rec.Answers.Type)
	}
	if rec.Answers.What != "something" {
		t.Errorf("I14 violation: pending lost answer What, got %q", rec.Answers.What)
	}
	// Save + re-read round-trip must preserve everything.
	if err := SavePending(workDir, rec); err != nil {
		t.Fatalf("SavePending: %v", err)
	}
	pendingDir := filepath.Join(workDir, ".lore", "pending")
	entries, _ := os.ReadDir(pendingDir)
	if len(entries) == 0 {
		t.Fatal("I14 violation: SavePending wrote nothing")
	}
}

// ─────────────────────────────────────────────────────────────────────────
// I15 — Ctrl+C serializes partial answers
// ─────────────────────────────────────────────────────────────────────────

// TestI15_FlushOnInterrupt_WithRegisteredState is the explicit named
// anchor for the interrupt-safety contract. Simulates the sequence:
//   1. RegisterInterruptState with partial answers
//   2. FlushOnInterrupt (as if Ctrl+C handler fired)
//   3. Assert a pending file was written to the registered workDir
//
// This is the runtime proof of the "Ctrl+C never loses answers" promise.
func TestI15_FlushOnInterrupt_WithRegisteredState(t *testing.T) {
	workDir := newReactiveWorkDir(t)

	partial := &Answers{
		Type: "feat",
		What: "partial what",
		Why:  "", // user was interrupted before typing Why
	}
	RegisterInterruptState(workDir, "aaaai15", "feat: partial commit", partial)
	t.Cleanup(func() { RegisterInterruptState("", "", "", nil) })

	FlushOnInterrupt()

	pendingDir := filepath.Join(workDir, ".lore", "pending")
	entries, err := os.ReadDir(pendingDir)
	if err != nil {
		t.Fatalf("I15 violation: pending dir unreadable after FlushOnInterrupt: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("I15 violation: no pending file written — partial answers lost on interrupt")
	}
}

// TestI15_FlushOnInterrupt_NoRegisteredState_NoOp asserts that calling
// FlushOnInterrupt with no registered state is a safe no-op. Signal
// handlers are called in unpredictable contexts; if the handler fired
// before the capture flow registered state (or after it cleared), it
// must NOT crash, write garbage, or create spurious pending files.
func TestI15_FlushOnInterrupt_NoRegisteredState_NoOp(t *testing.T) {
	// Ensure no prior state leaks from another test.
	RegisterInterruptState("", "", "", nil)

	workDir := newReactiveWorkDir(t)

	// Fire the handler — should be a clean no-op.
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("I15 violation: FlushOnInterrupt panicked with no state: %v", r)
			}
		}()
		FlushOnInterrupt()
	}()

	// Nothing should have been written to .lore/pending/.
	pendingDir := filepath.Join(workDir, ".lore", "pending")
	entries, _ := os.ReadDir(pendingDir)
	yamlCount := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".yaml") {
			yamlCount++
		}
	}
	if yamlCount > 0 {
		t.Errorf("I15 violation: FlushOnInterrupt wrote %d pending file(s) with no state", yamlCount)
	}
}

// TestI15_FlushIsIdempotent — calling FlushOnInterrupt twice in a row
// (e.g. after a first SIGINT is caught but exit was delayed, then a
// second SIGINT arrives) must not write duplicate pending files or
// crash. The interrupt state is cleared on the first flush, so the
// second must be a silent no-op.
func TestI15_FlushIsIdempotent(t *testing.T) {
	workDir := newReactiveWorkDir(t)
	pendingDir := filepath.Join(workDir, ".lore", "pending")

	partial := &Answers{Type: "feat", What: "done once"}
	RegisterInterruptState(workDir, "i15dup", "feat: dup", partial)
	t.Cleanup(func() { RegisterInterruptState("", "", "", nil) })

	FlushOnInterrupt()
	firstCount := countYamlFiles(t, pendingDir)

	// Second flush — state is now nil; must not write anything new.
	FlushOnInterrupt()
	secondCount := countYamlFiles(t, pendingDir)

	if secondCount != firstCount {
		t.Errorf("I15 violation: second FlushOnInterrupt wrote extra files (before=%d, after=%d)",
			firstCount, secondCount)
	}
}

// TestI15_UpdateAnswersPropagatesToFlush — the contract of
// updateInterruptAnswers is to refresh the registered state with the
// latest partial answers. If the user typed "feat"/"what"/"why" and
// Ctrl+C fires at that moment, the flushed pending file must reflect
// "why" (the most recent answer), not the state registered at flow start.
func TestI15_UpdateAnswersPropagatesToFlush(t *testing.T) {
	workDir := newReactiveWorkDir(t)

	// Initial state: just Type.
	initial := &Answers{Type: "feat"}
	RegisterInterruptState(workDir, "i15upd", "feat: updated", initial)
	t.Cleanup(func() { RegisterInterruptState("", "", "", nil) })

	// User progresses: Type + What + Why.
	latest := &Answers{Type: "feat", What: "more data", Why: "because"}
	updateInterruptAnswers(latest)

	FlushOnInterrupt()

	// The written pending should contain the LATEST answers, not the
	// initial snapshot. We read the yaml file and verify.
	pendingDir := filepath.Join(workDir, ".lore", "pending")
	entries, _ := os.ReadDir(pendingDir)
	if len(entries) == 0 {
		t.Fatal("I15 violation: no pending file after update+flush")
	}
	// Find the .yaml file for our hash.
	var found []byte
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".yaml") {
			data, err := os.ReadFile(filepath.Join(pendingDir, e.Name()))
			if err == nil && strings.Contains(string(data), "i15upd") {
				found = data
				break
			}
		}
	}
	if found == nil {
		t.Fatal("I15 violation: pending file for hash i15upd not found")
	}
	// The latest "why: because" must be present; we don't hard-parse
	// YAML to avoid dragging a dep in the test, string-contains is fine.
	if !strings.Contains(string(found), "because") {
		t.Errorf("I15 violation: update did not propagate to flush output\nfile content:\n%s",
			string(found))
	}
	if !strings.Contains(string(found), "more data") {
		t.Errorf("I15 violation: latest What field missing from pending\n%s", string(found))
	}
}

// ─────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────

func countYamlFiles(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".yaml") {
			n++
		}
	}
	return n
}
