// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ─────────────────────────────────────────────────────────────────
// ReviewDiff.Counts
// ─────────────────────────────────────────────────────────────────

func TestReviewDiffCounts_AllZero(t *testing.T) {
	d := ReviewDiff{}
	n, p, r, res := d.Counts()
	if n != 0 || p != 0 || r != 0 || res != 0 {
		t.Errorf("expected all zeros, got %d %d %d %d", n, p, r, res)
	}
}

func TestReviewDiffCounts_AllPopulated(t *testing.T) {
	d := ReviewDiff{
		New:        []ReviewFinding{{Title: "A"}, {Title: "B"}},
		Persisting: []ReviewFinding{{Title: "C"}},
		Regressed:  []ReviewFinding{{Title: "D"}, {Title: "E"}, {Title: "F"}},
		Resolved:   []ReviewFinding{{Title: "G"}},
	}
	n, p, reg, res := d.Counts()
	if n != 2 || p != 1 || reg != 3 || res != 1 {
		t.Errorf("counts = %d %d %d %d, want 2 1 3 1", n, p, reg, res)
	}
}

// ─────────────────────────────────────────────────────────────────
// WriteUnifiedDiff
// ─────────────────────────────────────────────────────────────────

func TestWriteUnifiedDiff_IdenticalContent(t *testing.T) {
	var b strings.Builder
	err := WriteUnifiedDiff(&b, "hello\n", "hello\n", UnifiedDiffOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No hunks — output should be empty.
	if b.Len() != 0 {
		t.Errorf("expected empty diff for identical content, got: %q", b.String())
	}
}

func TestWriteUnifiedDiff_Diff(t *testing.T) {
	var b strings.Builder
	err := WriteUnifiedDiff(&b, "line1\nline2\n", "line1\nchanged\n", UnifiedDiffOptions{
		FromFile: "a.md",
		ToFile:   "b.md",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, "line2") && !strings.Contains(out, "changed") {
		t.Errorf("diff missing expected lines, got: %s", out)
	}
}

// ─────────────────────────────────────────────────────────────────
// AutofixWriteWithBackup
// ─────────────────────────────────────────────────────────────────

func TestAutofixWriteWithBackup_NoBackup(t *testing.T) {
	dir := t.TempDir()
	docPath := filepath.Join(dir, "test.md")
	content := "# Test\n\nContent.\n"

	got, err := AutofixWriteWithBackup(docPath, content, false, dir, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != docPath {
		t.Errorf("returned path = %q, want %q", got, docPath)
	}
	data, _ := os.ReadFile(docPath)
	if string(data) != content {
		t.Errorf("file content mismatch: got %q, want %q", string(data), content)
	}
}

func TestAutofixWriteWithBackup_WithBackup(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, ".lore", "angela")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// WriteBackup requires a relative path from workDir.
	relDoc := "test.md"
	absDoc := filepath.Join(dir, relDoc)
	content := "# Test\n\nContent.\n"
	// Write original first so backup has something to copy.
	if err := os.WriteFile(absDoc, []byte("# Original\n"), 0o644); err != nil {
		t.Fatalf("write original: %v", err)
	}

	got, err := AutofixWriteWithBackup(relDoc, content, true, dir, stateDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// When relPath is relative, the returned path is also relative.
	if got != relDoc {
		t.Errorf("returned path = %q, want %q", got, relDoc)
	}
}

// ─────────────────────────────────────────────────────────────────
// ReviewState: LoadReviewState error paths
// ─────────────────────────────────────────────────────────────────

func TestLoadReviewState_CorruptJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "review-state.json")
	if err := os.WriteFile(path, []byte("not-json{{{"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	state, err := LoadReviewState(path)
	if err == nil {
		t.Fatal("expected error for corrupt JSON")
	}
	// Should return usable empty state.
	if state == nil {
		t.Error("expected non-nil state even on error")
	}
}

func TestLoadReviewState_WrongVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "review-state.json")
	data := `{"version": 9999, "findings": {}}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := LoadReviewState(path)
	if err == nil {
		t.Fatal("expected error for wrong version")
	}
}

// ─────────────────────────────────────────────────────────────────
// UpdateReviewState
// ─────────────────────────────────────────────────────────────────

func TestUpdateReviewState_NewFinding(t *testing.T) {
	state := &ReviewState{
		Version:  ReviewStateVersion,
		Findings: make(map[string]StatefulFinding),
	}
	finding := ReviewFinding{Title: "New gap", Severity: "gap"}
	diff := ReviewDiff{New: []ReviewFinding{finding}}

	UpdateReviewState(state, diff, time.Now())

	if len(state.Findings) != 1 {
		t.Errorf("expected 1 finding, got %d", len(state.Findings))
	}
}

func TestUpdateReviewState_RegressedFinding(t *testing.T) {
	// A previously-resolved finding that reappears should flip back to active.
	finding := ReviewFinding{Title: "Old gap", Severity: "gap"}
	hash := ReviewFindingHash(finding)
	finding.Hash = hash
	now := time.Now()
	resolvedAt := now.Add(-time.Hour)

	state := &ReviewState{
		Version: ReviewStateVersion,
		Findings: map[string]StatefulFinding{
			hash: {Finding: finding, Status: StatusResolved, FirstSeen: now, LastSeen: now, ResolvedAt: &resolvedAt},
		},
	}
	diff := ReviewDiff{Regressed: []ReviewFinding{finding}}
	UpdateReviewState(state, diff, now.Add(time.Minute))

	entry := state.Findings[hash]
	if entry.Status != StatusActive {
		t.Errorf("status = %q, want active after regression", entry.Status)
	}
	if entry.ResolvedAt != nil {
		t.Error("ResolvedAt should be cleared on regression")
	}
}

// ─────────────────────────────────────────────────────────────────
// parseDateToDays
// ─────────────────────────────────────────────────────────────────

func TestParseDateToDays_ValidDate(t *testing.T) {
	days := parseDateToDays("2026-01-15")
	if days <= 0 {
		t.Errorf("expected positive days for valid date, got %d", days)
	}
}

func TestParseDateToDays_InvalidDate(t *testing.T) {
	days := parseDateToDays("not-a-date")
	if days != 0 {
		t.Errorf("expected 0 for invalid date, got %d", days)
	}
}

func TestParseDateToDays_EmptyString(t *testing.T) {
	days := parseDateToDays("")
	if days != 0 {
		t.Errorf("expected 0 for empty string, got %d", days)
	}
}
