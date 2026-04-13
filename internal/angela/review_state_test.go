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

// --- helpers -------------------------------------------------------------

// rf is a terse constructor for a ReviewFinding used by several tests.
func rf(severity, title string, docs ...string) ReviewFinding {
	return ReviewFinding{
		Severity:  severity,
		Title:     title,
		Documents: docs,
	}
}

func newReviewStatePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "review-state.json")
}

// --- ReviewFindingHash --------------------------------------------------

// TestReviewFindingHash_StableAcrossDescription — the description text
// must NOT influence the hash, otherwise the same finding rephrased
// next week would map to a new entry and the lifecycle would break.
func TestReviewFindingHash_StableAcrossDescription(t *testing.T) {
	a := rf("contradiction", "Auth strategy mismatch", "auth.md", "session.md")
	a.Description = "First version of the description"
	b := rf("contradiction", "Auth strategy mismatch", "auth.md", "session.md")
	b.Description = "Completely rewritten description that says the same thing differently"

	if ReviewFindingHash(a) != ReviewFindingHash(b) {
		t.Errorf("description should not affect hash: a=%s b=%s",
			ReviewFindingHash(a), ReviewFindingHash(b))
	}
}

// TestReviewFindingHash_DocumentOrderIrrelevant — pinning the canonical
// form: documents are sorted before hashing so the AI's emission order
// can't break stability.
func TestReviewFindingHash_DocumentOrderIrrelevant(t *testing.T) {
	a := rf("gap", "Same gap", "z.md", "a.md", "m.md")
	b := rf("gap", "Same gap", "a.md", "m.md", "z.md")
	if ReviewFindingHash(a) != ReviewFindingHash(b) {
		t.Errorf("doc order leaked into hash: a=%s b=%s",
			ReviewFindingHash(a), ReviewFindingHash(b))
	}
}

// TestReviewFindingHash_TitlePunctuationNormalized — small title
// punctuation differences must collapse to the same hash. Without
// this, "Auth: mismatch" and "Auth mismatch" would be tracked as two
// distinct findings.
func TestReviewFindingHash_TitlePunctuationNormalized(t *testing.T) {
	a := rf("style", "Auth: mismatch!", "x.md")
	b := rf("style", "Auth mismatch", "x.md")
	if ReviewFindingHash(a) != ReviewFindingHash(b) {
		t.Errorf("punctuation leaked into hash: a=%s b=%s",
			ReviewFindingHash(a), ReviewFindingHash(b))
	}
}

// TestReviewFindingHash_Length — fixture: 16 hex chars (64 bits).
func TestReviewFindingHash_Length(t *testing.T) {
	h := ReviewFindingHash(rf("contradiction", "X", "y.md"))
	if len(h) != 16 {
		t.Errorf("hash length = %d, want 16", len(h))
	}
}

// --- LoadReviewState / SaveReviewState ----------------------------------

// TestLoadReviewState_MissingFileReturnsEmpty mirrors the draft state
// load contract: missing file is the first-run case, not an error.
func TestLoadReviewState_MissingFileReturnsEmpty(t *testing.T) {
	s, err := LoadReviewState(newReviewStatePath(t))
	if err != nil {
		t.Fatalf("LoadReviewState: %v", err)
	}
	if s == nil || s.Findings == nil {
		t.Fatalf("expected non-nil empty state, got %+v", s)
	}
	if len(s.Findings) != 0 {
		t.Errorf("findings = %d, want 0", len(s.Findings))
	}
}

// TestLoadReviewState_VersionMismatch — AC-9: version drift triggers
// a fresh start plus a non-nil error so the runner can log a notice.
func TestLoadReviewState_VersionMismatch(t *testing.T) {
	path := newReviewStatePath(t)
	if err := os.WriteFile(path, []byte(`{"version":42,"findings":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := LoadReviewState(path)
	if err == nil {
		t.Error("expected error for version mismatch")
	}
	if s == nil || len(s.Findings) != 0 {
		t.Errorf("expected fresh empty state on mismatch")
	}
}

// TestSaveAndLoadReviewState_RoundTrip exercises the full save→load
// cycle so any bug in either function surfaces in a single test.
func TestSaveAndLoadReviewState_RoundTrip(t *testing.T) {
	path := newReviewStatePath(t)
	now := time.Now().UTC()
	in := &ReviewState{
		Version: ReviewStateVersion,
		LastRun: now,
		Findings: map[string]StatefulFinding{
			"deadbeefcafef00d": {
				Finding:   rf("style", "tone", "doc.md"),
				Status:    StatusActive,
				FirstSeen: now,
				LastSeen:  now,
			},
		},
	}
	if err := SaveReviewState(path, in); err != nil {
		t.Fatalf("SaveReviewState: %v", err)
	}
	out, err := LoadReviewState(path)
	if err != nil {
		t.Fatalf("LoadReviewState: %v", err)
	}
	if len(out.Findings) != 1 {
		t.Fatalf("findings = %d, want 1", len(out.Findings))
	}
	got, ok := out.Findings["deadbeefcafef00d"]
	if !ok {
		t.Fatalf("missing key after roundtrip")
	}
	if got.Status != StatusActive || got.Finding.Title != "tone" {
		t.Errorf("roundtrip mismatch: %+v", got)
	}
}

// --- ComputeReviewDiff (the AC-10 enumerated tests) --------------------

// TestReviewState_FirstRunAllNew — empty state → every current finding
// is NEW, no PERSISTING/RESOLVED.
func TestReviewState_FirstRunAllNew(t *testing.T) {
	prev := &ReviewState{Findings: map[string]StatefulFinding{}}
	current := []ReviewFinding{
		rf("contradiction", "alpha", "a.md"),
		rf("style", "beta", "b.md"),
	}
	diff := ComputeReviewDiff(prev, current)
	if len(diff.New) != 2 || len(diff.Persisting) != 0 || len(diff.Regressed) != 0 {
		t.Errorf("diff = %+v", diff)
	}
	for _, f := range diff.New {
		if f.DiffStatus != ReviewDiffNew {
			t.Errorf("tag = %q, want %q", f.DiffStatus, ReviewDiffNew)
		}
	}
}

// TestReviewState_SecondRunAllPersisting — same findings, all active
// in prev → all PERSISTING. The cache hit case.
func TestReviewState_SecondRunAllPersisting(t *testing.T) {
	first := rf("style", "alpha", "a.md")
	hash := ReviewFindingHash(first)
	prev := &ReviewState{
		Findings: map[string]StatefulFinding{
			hash: {Finding: first, Status: StatusActive},
		},
	}
	current := []ReviewFinding{first}
	diff := ComputeReviewDiff(prev, current)
	if len(diff.Persisting) != 1 || len(diff.New) != 0 || len(diff.Regressed) != 0 {
		t.Errorf("diff = %+v", diff)
	}
	if diff.Persisting[0].DiffStatus != ReviewDiffPersisting {
		t.Errorf("tag = %q, want persisting", diff.Persisting[0].DiffStatus)
	}
}

// TestReviewState_ResolvedStaysResolved — a finding the user marked
// resolved does not appear in any of the diff slices when the AI no
// longer returns it. The state entry is preserved (not pruned).
func TestReviewState_ResolvedStaysResolved(t *testing.T) {
	f := rf("style", "old finding", "x.md")
	hash := ReviewFindingHash(f)
	prev := &ReviewState{
		Findings: map[string]StatefulFinding{
			hash: {Finding: f, Status: StatusResolved},
		},
	}
	current := []ReviewFinding{} // AI didn't return it this run
	diff := ComputeReviewDiff(prev, current)
	if len(diff.New)+len(diff.Persisting)+len(diff.Regressed)+len(diff.Resolved) != 0 {
		t.Errorf("resolved entry should not appear in any diff slice, got %+v", diff)
	}
	// Entry must still exist in state.
	if _, ok := prev.Findings[hash]; !ok {
		t.Error("resolved entry should remain in state")
	}
}

// TestReviewState_IgnoredStaysIgnored — symmetric to the resolved
// case, for the ignore lifecycle.
func TestReviewState_IgnoredStaysIgnored(t *testing.T) {
	f := rf("style", "noisy finding", "x.md")
	hash := ReviewFindingHash(f)
	prev := &ReviewState{
		Findings: map[string]StatefulFinding{
			hash: {Finding: f, Status: StatusIgnored, IgnoreReason: "intentional"},
		},
	}
	diff := ComputeReviewDiff(prev, []ReviewFinding{})
	total := len(diff.New) + len(diff.Persisting) + len(diff.Regressed) + len(diff.Resolved)
	if total != 0 {
		t.Errorf("ignored entry should not appear in any diff slice, got %d total", total)
	}
}

// TestReviewState_RegressedDetection — the high-signal scenario: a
// finding marked resolved comes back. The new instance is REGRESSED,
// not NEW.
func TestReviewState_RegressedDetection(t *testing.T) {
	f := rf("contradiction", "auth conflict", "a.md", "b.md")
	hash := ReviewFindingHash(f)
	prev := &ReviewState{
		Findings: map[string]StatefulFinding{
			hash: {Finding: f, Status: StatusResolved},
		},
	}
	current := []ReviewFinding{f}
	diff := ComputeReviewDiff(prev, current)
	if len(diff.Regressed) != 1 {
		t.Fatalf("expected 1 REGRESSED, got %+v", diff)
	}
	if diff.Regressed[0].DiffStatus != ReviewDiffRegressed {
		t.Errorf("tag = %q, want regressed", diff.Regressed[0].DiffStatus)
	}
	if len(diff.New) != 0 {
		t.Errorf("regressed finding should not also be NEW")
	}
}

// TestReviewState_ResolvedFindingRemovedThenRestored — combined
// scenario: resolve in prev → finding gone from current run → next
// run finds it again → REGRESSED. We test the second hop directly.
func TestReviewState_ResolvedFindingRemovedThenRestored(t *testing.T) {
	f := rf("gap", "missing perms doc", "perms.md")
	hash := ReviewFindingHash(f)

	// State: previously seen + resolved + still resolved.
	resolvedAt := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	prev := &ReviewState{
		Findings: map[string]StatefulFinding{
			hash: {
				Finding:    f,
				Status:     StatusResolved,
				ResolvedAt: &resolvedAt,
			},
		},
	}
	// Two runs later, the AI surfaces it again.
	diff := ComputeReviewDiff(prev, []ReviewFinding{f})
	if len(diff.Regressed) != 1 {
		t.Fatalf("expected REGRESSED on re-discovery, got %+v", diff)
	}

	// UpdateReviewState should flip the lifecycle back to active and
	// clear ResolvedAt so the next run sees a clean active entry.
	UpdateReviewState(prev, diff, time.Now().UTC())
	got := prev.Findings[hash]
	if got.Status != StatusActive {
		t.Errorf("status after regression = %q, want active", got.Status)
	}
	if got.ResolvedAt != nil {
		t.Errorf("ResolvedAt should be cleared on regression, got %v", got.ResolvedAt)
	}
}

// TestReviewState_PrefixHashLookup — AC-4: the resolve subcommand
// accepts the first 6 chars of a hash if unambiguous. Test all three
// outcomes (hit, miss, ambiguous).
func TestReviewState_PrefixHashLookup(t *testing.T) {
	prev := &ReviewState{
		Findings: map[string]StatefulFinding{
			"abc123def4567890": {Finding: rf("style", "a")},
			"abc999def4567890": {Finding: rf("style", "b")},
			"def000aabbccddee": {Finding: rf("style", "c")},
		},
	}

	// Unambiguous prefix → hit.
	full, err := ResolveByPrefix(prev, "def000")
	if err != nil {
		t.Errorf("expected hit, got %v", err)
	}
	if full != "def000aabbccddee" {
		t.Errorf("full = %q", full)
	}

	// Ambiguous prefix → error listing the candidates.
	_, err = ResolveByPrefix(prev, "abc")
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expected ambiguous error, got %v", err)
	}

	// Missing prefix → not-found error.
	_, err = ResolveByPrefix(prev, "9999")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found, got %v", err)
	}
}

// TestReviewState_PrefixHashLookup_BoundaryLengths pins the
// ResolveByPrefix behavior at every length around the spelled-out
// 6-char target of AC-4. Paranoid-review fix (2026-04-11 MEDIUM
// test coverage): the previous test only exercised the three happy/
// error branches with a 6-char input; a regression that changed the
// prefix length handling (e.g. off-by-one on the substring bounds)
// would not have been caught.
func TestReviewState_PrefixHashLookup_BoundaryLengths(t *testing.T) {
	full := "abc123def4567890"
	other := "abcxxxyyyzzz0000"
	prev := &ReviewState{
		Findings: map[string]StatefulFinding{
			full:  {Finding: rf("style", "a")},
			other: {Finding: rf("style", "b")},
		},
	}

	cases := []struct {
		name    string
		prefix  string
		wantHit bool
		wantErr string // substring; empty = no error expected
	}{
		{"empty prefix", "", false, "empty"},
		{"1 char ambiguous", "a", false, "ambiguous"},
		{"3 chars ambiguous", "abc", false, "ambiguous"},
		{"4 chars unambiguous", "abc1", true, ""},
		{"5 chars unambiguous", "abc12", true, ""},
		{"6 chars AC target", "abc123", true, ""},
		{"7 chars unambiguous", "abc123d", true, ""},
		{"full 16 chars exact", full, true, ""},
		{"uppercase matches canonical", "ABC123", true, ""},
		{"padded prefix fails", "abc123def4567890x", false, "not found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveByPrefix(prev, tc.prefix)
			if tc.wantHit {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != full {
					t.Errorf("full = %q, want %q", got, full)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error, got hit %q", got)
			}
			if tc.wantErr != "" && !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want substring %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// TestReviewState_IgnoreRequiresReason — AC-5: ignore must be
// deliberate. An empty reason errors out before any state mutation.
func TestReviewState_IgnoreRequiresReason(t *testing.T) {
	f := rf("style", "needs ignore", "x.md")
	hash := ReviewFindingHash(f)
	state := &ReviewState{
		Findings: map[string]StatefulFinding{hash: {Finding: f, Status: StatusActive}},
	}
	if err := MarkIgnored(state, hash, "  ", time.Now()); err == nil {
		t.Error("expected error for whitespace-only reason")
	}
	// State must NOT have been mutated by the failed call.
	if state.Findings[hash].Status != StatusActive {
		t.Error("state mutated despite reason error")
	}
}

// TestReviewState_LogSortedByLastSeen — AC-6: the log subcommand
// prints entries by LastSeen DESC. Test the underlying LogEntries
// helper directly.
func TestReviewState_LogSortedByLastSeen(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	state := &ReviewState{
		Findings: map[string]StatefulFinding{
			"a": {LastSeen: t1, Finding: rf("style", "old")},
			"b": {LastSeen: t3, Finding: rf("style", "newest")},
			"c": {LastSeen: t2, Finding: rf("style", "middle")},
		},
	}
	out := LogEntries(state)
	if len(out) != 3 {
		t.Fatalf("len = %d", len(out))
	}
	if out[0].Finding.Title != "newest" || out[2].Finding.Title != "old" {
		t.Errorf("sort order wrong: %v", out)
	}
}

// TestUpdateReviewState_NewBumpsTimestamps — sanity check that
// inserting a NEW finding stamps both FirstSeen and LastSeen.
func TestUpdateReviewState_NewBumpsTimestamps(t *testing.T) {
	state := &ReviewState{Findings: map[string]StatefulFinding{}}
	now := time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)
	f := rf("style", "fresh", "x.md")
	f.Hash = ReviewFindingHash(f)
	UpdateReviewState(state, ReviewDiff{New: []ReviewFinding{f}}, now)
	got := state.Findings[f.Hash]
	if !got.FirstSeen.Equal(now) || !got.LastSeen.Equal(now) {
		t.Errorf("timestamps wrong: %+v", got)
	}
	if got.Status != StatusActive {
		t.Errorf("status = %q, want active", got.Status)
	}
}

// TestMarkResolved_NotFound — guarding the unhappy path of the
// resolve subcommand: an unknown hash errors out cleanly.
func TestMarkResolved_NotFound(t *testing.T) {
	state := &ReviewState{Findings: map[string]StatefulFinding{}}
	if err := MarkResolved(state, "deadbeef", "alice", time.Now()); err == nil {
		t.Error("expected error for unknown hash")
	}
}
