// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"testing"
	"time"
)

// ─────────────────────────────────────────────────────────────────
// SaveReviewState — nil state
// ─────────────────────────────────────────────────────────────────

func TestSaveReviewState_NilState(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/review-state.json"
	err := SaveReviewState(path, nil)
	if err == nil {
		t.Error("expected error for nil state")
	}
}

// ─────────────────────────────────────────────────────────────────
// LoadReviewState — unknown status is normalized to active
// ─────────────────────────────────────────────────────────────────

func TestLoadReviewState_UnknownStatusNormalized(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/review-state.json"
	// Manually craft a state with an unknown status value.
	state := &ReviewState{
		Version: ReviewStateVersion,
		Findings: map[string]StatefulFinding{
			"aabb1122aabb1122": {
				Finding: ReviewFinding{Title: "test"},
				Status:  "bogus-unknown-status",
			},
		},
	}
	if err := SaveReviewState(path, state); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := LoadReviewState(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	entry := loaded.Findings["aabb1122aabb1122"]
	if entry.Status != StatusActive {
		t.Errorf("unknown status should normalize to active, got %q", entry.Status)
	}
}

// ─────────────────────────────────────────────────────────────────
// UpdateReviewState — persisting path
// ─────────────────────────────────────────────────────────────────

func TestUpdateReviewState_PersistingUpdatesLastSeen(t *testing.T) {
	finding := ReviewFinding{Title: "Persisting gap", Severity: "gap"}
	hash := ReviewFindingHash(finding)
	finding.Hash = hash
	t1 := time.Now().Add(-time.Hour)
	t2 := time.Now()

	state := &ReviewState{
		Version: ReviewStateVersion,
		Findings: map[string]StatefulFinding{
			hash: {Finding: finding, Status: StatusActive, FirstSeen: t1, LastSeen: t1},
		},
	}
	diff := ReviewDiff{Persisting: []ReviewFinding{finding}}
	UpdateReviewState(state, diff, t2)

	entry := state.Findings[hash]
	if !entry.LastSeen.Equal(t2) {
		t.Errorf("LastSeen not updated: got %v, want %v", entry.LastSeen, t2)
	}
	if entry.Status != StatusActive {
		t.Errorf("persisting should stay active, got %q", entry.Status)
	}
}

// ─────────────────────────────────────────────────────────────────
// ComputeReviewDiffWithRejected
// ─────────────────────────────────────────────────────────────────

func TestComputeReviewDiffWithRejected_RejectedDoesNotShowResolved(t *testing.T) {
	// A finding that is "active" in prev state and appears in rejected
	// should NOT be counted as RESOLVED.
	finding := ReviewFinding{Title: "Rejected finding", Severity: "style"}
	hash := HashReviewFindingWithAudience(finding, "")
	finding.Hash = hash

	prev := &ReviewState{
		Version: ReviewStateVersion,
		Findings: map[string]StatefulFinding{
			hash: {Finding: finding, Status: StatusActive, FirstSeen: time.Now(), LastSeen: time.Now()},
		},
	}
	rejected := []RejectedFinding{{Finding: finding, Reason: "not real"}}
	diff := ComputeReviewDiffWithRejected(prev, nil, rejected, "")

	for _, r := range diff.Resolved {
		if r.Hash == hash {
			t.Error("rejected finding should not appear as RESOLVED")
		}
	}
}

func TestComputeReviewDiffWithRejected_RegressedPath(t *testing.T) {
	// A finding stored as StatusResolved that reappears should be REGRESSED.
	finding := ReviewFinding{Title: "Regressed issue", Severity: "gap"}
	finding.Hash = HashReviewFindingWithAudience(finding, "")

	prev := &ReviewState{
		Version: ReviewStateVersion,
		Findings: map[string]StatefulFinding{
			finding.Hash: {Finding: finding, Status: StatusResolved, FirstSeen: time.Now(), LastSeen: time.Now()},
		},
	}
	diff := ComputeReviewDiffWithRejected(prev, []ReviewFinding{finding}, nil, "")

	if len(diff.Regressed) != 1 {
		t.Errorf("expected 1 REGRESSED finding, got %d", len(diff.Regressed))
	}
}

func TestComputeReviewDiffWithRejected_NilPrev(t *testing.T) {
	finding := ReviewFinding{Title: "New", Severity: "gap"}
	diff := ComputeReviewDiffWithRejected(nil, []ReviewFinding{finding}, nil, "")
	if len(diff.New) != 1 {
		t.Errorf("expected 1 NEW finding with nil prev, got %d", len(diff.New))
	}
}

// ─────────────────────────────────────────────────────────────────
// MarkResolved / MarkIgnored — error paths
// ─────────────────────────────────────────────────────────────────

func TestMarkResolved_HashMissing(t *testing.T) {
	state := &ReviewState{
		Version:  ReviewStateVersion,
		Findings: make(map[string]StatefulFinding),
	}
	err := MarkResolved(state, "nonexistent", "user", time.Now())
	if err == nil {
		t.Error("expected error for hash not in state")
	}
}

func TestMarkIgnored_HashMissing(t *testing.T) {
	state := &ReviewState{
		Version:  ReviewStateVersion,
		Findings: make(map[string]StatefulFinding),
	}
	err := MarkIgnored(state, "nonexistent", "reason", time.Now())
	if err == nil {
		t.Error("expected error for hash not in state")
	}
}

// ─────────────────────────────────────────────────────────────────
// ResolveByPrefix — edge cases
// ─────────────────────────────────────────────────────────────────

func TestResolveByPrefix_NilState(t *testing.T) {
	_, err := ResolveByPrefix(nil, "abc")
	if err == nil {
		t.Error("expected error for nil state")
	}
}

func TestResolveByPrefix_EmptyPrefix(t *testing.T) {
	state := &ReviewState{
		Version:  ReviewStateVersion,
		Findings: make(map[string]StatefulFinding),
	}
	_, err := ResolveByPrefix(state, "")
	if err == nil {
		t.Error("expected error for empty prefix")
	}
}

func TestResolveByPrefix_AmbiguousPrefix(t *testing.T) {
	// Two hashes with the same prefix → ambiguous error.
	state := &ReviewState{
		Version: ReviewStateVersion,
		Findings: map[string]StatefulFinding{
			"aabb1122aabb1122": {Finding: ReviewFinding{Title: "A"}},
			"aabb334455667788": {Finding: ReviewFinding{Title: "B"}},
		},
	}
	_, err := ResolveByPrefix(state, "aabb")
	if err == nil {
		t.Error("expected error for ambiguous prefix")
	}
}

// ─────────────────────────────────────────────────────────────────
// LogEntries — nil state
// ─────────────────────────────────────────────────────────────────

func TestLogEntries_NilState(t *testing.T) {
	entries := LogEntries(nil)
	if entries != nil {
		t.Errorf("expected nil for nil state, got %v", entries)
	}
}

func TestLogEntries_SortedByLastSeen(t *testing.T) {
	now := time.Now()
	state := &ReviewState{
		Version: ReviewStateVersion,
		Findings: map[string]StatefulFinding{
			"hash1": {Finding: ReviewFinding{Title: "Older"}, LastSeen: now.Add(-time.Hour)},
			"hash2": {Finding: ReviewFinding{Title: "Newer"}, LastSeen: now},
		},
	}
	entries := LogEntries(state)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Finding.Title != "Newer" {
		t.Errorf("first entry should be newest, got %q", entries[0].Finding.Title)
	}
}
