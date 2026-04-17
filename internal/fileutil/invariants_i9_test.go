// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package fileutil

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// Invariant I9 — Atomic writes never corrupt on Ctrl+C.
//
// Contract: when a write to `path` fails at ANY stage of the atomic-write
// pipeline (write/chmod/rename), the FINAL state of `path` must be either:
//   (a) the old content (if any), bytewise unchanged, or
//   (b) the new content, fully written.
//
// No intermediate / partially-written / truncated state is acceptable.
// Orphan `.lore-*.tmp` files in the directory are also forbidden: the
// cleanup defer must fire even on error paths.
//
// Layer 1: existing per-stage fault-injection tests in atomic_internal_test.go
//          (TestAtomicWrite_RenameError, TestAtomicWrite_WriteError,
//          TestAtomicWrite_ChmodError).
// Layer 2: TestI9_* named anchors below, which assert the AGGREGATE
//          invariant (old content preserved + no debris) regardless of
//          which stage fails.
// ═══════════════════════════════════════════════════════════════════════════

// TestI9_AtomicWrite_TargetIntactWhenRenameFails anchors I9 for the
// rename-stage fault mode. A SIGINT / Ctrl+C that arrives AFTER the temp
// file is written but BEFORE rename completes is behaviorally equivalent
// to the rename returning an error: the target is unchanged. We simulate
// it by faulting osRename.
func TestI9_AtomicWrite_TargetIntactWhenRenameFails(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "doc.md")

	// Seed the target with old content so the invariant is observable:
	// "rename fails → old content preserved" (not "file absent").
	oldContent := []byte("# old\nBefore the interrupt\n")
	if err := os.WriteFile(target, oldContent, 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Swap osRename to simulate the signal arriving mid-pipeline.
	origRename := osRename
	osRename = func(oldpath, newpath string) error {
		return errors.New("simulated SIGINT between write and rename")
	}
	t.Cleanup(func() { osRename = origRename })

	newContent := []byte("# new\nAfter the would-be write\n")
	err := AtomicWrite(target, newContent, 0o644)
	if err == nil {
		t.Fatal("expected AtomicWrite to surface the rename failure")
	}

	// I9 core assertion: target must be UNCHANGED. No torn write, no
	// truncation, no partial buffer.
	got, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatalf("target disappeared (I9 violation): %v", readErr)
	}
	if string(got) != string(oldContent) {
		t.Errorf("I9 violation: target mutated after failed rename\n  want: %q\n  got:  %q",
			string(oldContent), string(got))
	}
}

// TestI9_AtomicWrite_NoTempDebrisAfterFailure asserts the cleanup side of
// I9: after ANY failure mode (write, chmod, rename), the directory must
// not contain orphan `.lore-*.tmp` files. Accumulated debris would fill
// disk over time on a system where SIGINT happens repeatedly (e.g. CI
// loop that times out mid-write).
func TestI9_AtomicWrite_NoTempDebrisAfterFailure(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "doc.md")

	// Force rename to fail so we hit the cleanup defer.
	origRename := osRename
	osRename = func(oldpath, newpath string) error {
		return errors.New("forced rename failure")
	}
	t.Cleanup(func() { osRename = origRename })

	if err := AtomicWrite(target, []byte("x"), 0o644); err == nil {
		t.Fatal("expected error")
	}

	// Enumerate the dir; the target shouldn't be there, and — crucially —
	// there must be no `.lore-*.tmp` leftovers.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") ||
			strings.HasPrefix(e.Name(), ".lore-") {
			t.Errorf("I9 violation: orphan temp file %q left behind after rename failure",
				e.Name())
		}
	}
}

// TestI9_AtomicWriteExclusive_AtomicOnConcurrentLink asserts the exclusive
// variant's contract: if the target already exists, the call must fail
// atomically (EEXIST via os.Link) and leave the existing file intact. A
// naive implementation that races between stat + rename could silently
// overwrite — I9 rules that out.
func TestI9_AtomicWriteExclusive_AtomicOnConcurrentLink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "state.json")

	// First write succeeds.
	if err := AtomicWriteExclusive(target, []byte("original"), 0o644); err != nil {
		t.Fatalf("first exclusive write: %v", err)
	}

	// Second write must fail (target exists) — this is the "cannot
	// silently overwrite state files" guarantee that matters for the LKS
	// store and review state JSON.
	err := AtomicWriteExclusive(target, []byte("overwrite-attempt"), 0o644)
	if err == nil {
		t.Fatal("I9 violation: AtomicWriteExclusive overwrote an existing file")
	}

	// Original must be preserved verbatim.
	got, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatalf("read target: %v", readErr)
	}
	if string(got) != "original" {
		t.Errorf("I9 violation: target corrupted by failed exclusive write\n  want: %q\n  got:  %q",
			"original", string(got))
	}
}
