// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// --- helpers -------------------------------------------------------------

// sugg is a terse constructor for a Suggestion used by several tests.
// Keeping it here rather than inline cuts the test bodies in half.
func sugg(cat, sev, msg string) Suggestion {
	return Suggestion{Category: cat, Severity: sev, Message: msg}
}

// newStatePath returns a path inside a t.TempDir() that can be passed
// directly to LoadDraftState / SaveDraftState.
func newStatePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "draft-state.json")
}

// --- ContentHash ---------------------------------------------------------

// TestContentHash_DeterministicAndShaped — same input → same hash, and
// the "sha256:" prefix is honored so downstream consumers can future-
// proof against a later algorithm swap.
func TestContentHash_DeterministicAndShaped(t *testing.T) {
	h1 := ContentHash([]byte("hello world"))
	h2 := ContentHash([]byte("hello world"))
	if h1 != h2 {
		t.Errorf("ContentHash not deterministic: %q vs %q", h1, h2)
	}
	if h1[:7] != "sha256:" {
		t.Errorf("hash missing sha256 prefix: %q", h1)
	}
	// Known good fixture (lower-case hex SHA-256 of "hello world").
	want := "sha256:b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if h1 != want {
		t.Errorf("hash = %q, want %q", h1, want)
	}
}

// --- LoadDraftState / SaveDraftState ------------------------------------

// TestLoadDraftState_MissingFileReturnsEmpty — the runner loads state
// on every invocation; a missing file is the first-run case and must
// produce a usable empty state, not an error.
func TestLoadDraftState_MissingFileReturnsEmpty(t *testing.T) {
	path := newStatePath(t)
	s, err := LoadDraftState(path)
	if err != nil {
		t.Fatalf("LoadDraftState: %v", err)
	}
	if s == nil || s.Entries == nil {
		t.Fatalf("expected non-nil state with non-nil entries, got %+v", s)
	}
	if s.Version != DraftStateVersion {
		t.Errorf("version = %d, want %d", s.Version, DraftStateVersion)
	}
	if len(s.Entries) != 0 {
		t.Errorf("entries = %d, want 0", len(s.Entries))
	}
}

// TestDraftState_VersionMismatchReanalyzes — AC-11 enumerated test name
// from story 8.8. Loading a state file whose version doesn't match the
// current constant returns a fresh empty state + a non-nil error so
// the runner can log a notice and start over.
func TestDraftState_VersionMismatchReanalyzes(t *testing.T) {
	path := newStatePath(t)
	// Hand-crafted file with version 999 → definitely not current.
	if err := os.WriteFile(path, []byte(`{"version":999,"entries":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := LoadDraftState(path)
	if err == nil {
		t.Error("expected non-nil error for version mismatch")
	}
	if s == nil {
		t.Fatal("expected fallback empty state, got nil")
	}
	if len(s.Entries) != 0 {
		t.Errorf("expected fresh entries on version mismatch, got %d", len(s.Entries))
	}
}

// TestSaveAndLoadDraftState_RoundTrip exercises the full save→load
// roundtrip so any bug in either function surfaces in a single test.
func TestSaveAndLoadDraftState_RoundTrip(t *testing.T) {
	path := newStatePath(t)
	now := time.Now().UTC()
	in := &DraftState{
		Version: DraftStateVersion,
		LastRun: now,
		Entries: map[string]DraftEntry{
			"foo.md": {
				ContentHash:  "sha256:abc",
				LastAnalyzed: now,
				Suggestions:  []Suggestion{sugg("style", "info", "Watch tone")},
				Score:        90,
				Grade:        "A",
				Profile:      "strict",
			},
		},
	}
	if err := SaveDraftState(path, in); err != nil {
		t.Fatalf("SaveDraftState: %v", err)
	}
	out, err := LoadDraftState(path)
	if err != nil {
		t.Fatalf("LoadDraftState: %v", err)
	}
	if len(out.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(out.Entries))
	}
	e := out.Entries["foo.md"]
	if e.ContentHash != "sha256:abc" || e.Score != 90 || e.Grade != "A" {
		t.Errorf("roundtrip mismatch: %+v", e)
	}
	if len(e.Suggestions) != 1 || e.Suggestions[0].Message != "Watch tone" {
		t.Errorf("suggestions lost in roundtrip: %+v", e.Suggestions)
	}
}

// --- AnnotateAndDiff ----------------------------------------------------

// TestDraftState_FirstRunAllNew — empty previous state means every
// current finding is tagged NEW and the diff summary reflects that.
func TestDraftState_FirstRunAllNew(t *testing.T) {
	prev := &DraftState{Version: DraftStateVersion, Entries: map[string]DraftEntry{}}
	current := map[string][]Suggestion{
		"a.md": {sugg("style", "info", "one"), sugg("coherence", "warning", "two")},
	}
	diff, resolved := AnnotateAndDiff(prev, current)
	if diff.New != 2 || diff.Persisting != 0 || diff.Resolved != 0 {
		t.Errorf("diff = %+v", diff)
	}
	if len(resolved) != 0 {
		t.Errorf("resolved = %d, want 0", len(resolved))
	}
	for _, s := range current["a.md"] {
		if s.DiffStatus != DiffStatusNew {
			t.Errorf("suggestion %q tagged %q, want %q", s.Message, s.DiffStatus, DiffStatusNew)
		}
	}
}

// TestDraftState_SecondRunNoChanges — identical corpus → every finding
// must be tagged PERSISTING. This is the test that proves cache hits
// stay stable across runs.
func TestDraftState_SecondRunNoChanges(t *testing.T) {
	prev := &DraftState{
		Version: DraftStateVersion,
		Entries: map[string]DraftEntry{
			"a.md": {
				ContentHash: "sha256:x",
				Suggestions: []Suggestion{sugg("style", "info", "one")},
			},
		},
	}
	current := map[string][]Suggestion{
		"a.md": {sugg("style", "info", "one")},
	}
	diff, resolved := AnnotateAndDiff(prev, current)
	if diff.New != 0 || diff.Persisting != 1 || diff.Resolved != 0 {
		t.Errorf("diff = %+v, want new=0 persisting=1 resolved=0", diff)
	}
	if len(resolved) != 0 {
		t.Errorf("resolved = %d, want 0", len(resolved))
	}
	if current["a.md"][0].DiffStatus != DiffStatusPersisting {
		t.Errorf("tag = %q, want persisting", current["a.md"][0].DiffStatus)
	}
}

// TestDraftState_DocChangedReanalyzed — same filename but the current
// run produced a different finding → the old one is resolved, the new
// one is tagged NEW.
func TestDraftState_DocChangedReanalyzed(t *testing.T) {
	prev := &DraftState{
		Version: DraftStateVersion,
		Entries: map[string]DraftEntry{
			"a.md": {
				ContentHash: "sha256:x",
				Suggestions: []Suggestion{sugg("style", "info", "old finding")},
			},
		},
	}
	current := map[string][]Suggestion{
		"a.md": {sugg("style", "info", "fresh finding")},
	}
	diff, resolved := AnnotateAndDiff(prev, current)
	if diff.New != 1 || diff.Persisting != 0 || diff.Resolved != 1 {
		t.Errorf("diff = %+v", diff)
	}
	if len(resolved) != 1 || resolved[0].Suggestion.Message != "old finding" {
		t.Errorf("resolved = %+v", resolved)
	}
	if current["a.md"][0].DiffStatus != DiffStatusNew {
		t.Errorf("current tag = %q, want new", current["a.md"][0].DiffStatus)
	}
}

// TestDraftState_DocRemovedEntryPruned — a file that was in the
// previous state but is no longer in the corpus: its findings count
// as RESOLVED. Combined with PruneMissingEntries, the new state drops
// the entry so the file list stays in sync with the corpus.
func TestDraftState_DocRemovedEntryPruned(t *testing.T) {
	prev := &DraftState{
		Version: DraftStateVersion,
		Entries: map[string]DraftEntry{
			"deleted.md": {
				ContentHash: "sha256:x",
				Suggestions: []Suggestion{sugg("style", "info", "vanished")},
			},
		},
	}
	current := map[string][]Suggestion{} // no files this run
	diff, resolved := AnnotateAndDiff(prev, current)
	if diff.Resolved != 1 {
		t.Errorf("diff.Resolved = %d, want 1", diff.Resolved)
	}
	if len(resolved) != 1 || resolved[0].File != "deleted.md" {
		t.Errorf("resolved = %+v", resolved)
	}

	// Prune step: new state has NO entry for deleted.md after prune.
	newState := &DraftState{
		Version: DraftStateVersion,
		Entries: map[string]DraftEntry{
			"deleted.md": prev.Entries["deleted.md"], // simulate carry-over before prune
		},
	}
	removed := PruneMissingEntries(newState, map[string]bool{})
	if removed != 1 {
		t.Errorf("prune removed = %d, want 1", removed)
	}
	if _, ok := newState.Entries["deleted.md"]; ok {
		t.Errorf("expected deleted.md to be pruned")
	}
}

// TestDraftState_DiffOnlyHidesPersisting — the reporter is tested via
// the cmd layer, but at the validator layer we can at least confirm
// that PERSISTING findings are correctly tagged so the reporter has
// something to filter on.
func TestDraftState_DiffOnlyHidesPersisting(t *testing.T) {
	prev := &DraftState{
		Version: DraftStateVersion,
		Entries: map[string]DraftEntry{
			"a.md": {
				Suggestions: []Suggestion{
					sugg("style", "info", "kept"),
					sugg("style", "info", "going away"),
				},
			},
		},
	}
	current := map[string][]Suggestion{
		"a.md": {
			sugg("style", "info", "kept"),
			sugg("coherence", "warning", "brand new"),
		},
	}
	diff, resolved := AnnotateAndDiff(prev, current)
	if diff.New != 1 || diff.Persisting != 1 || diff.Resolved != 1 {
		t.Errorf("diff = %+v", diff)
	}
	if len(resolved) != 1 || resolved[0].Suggestion.Message != "going away" {
		t.Errorf("resolved = %+v", resolved)
	}
	tags := map[string]string{}
	for _, s := range current["a.md"] {
		tags[s.Message] = s.DiffStatus
	}
	if tags["kept"] != DiffStatusPersisting || tags["brand new"] != DiffStatusNew {
		t.Errorf("tags = %+v", tags)
	}
}

// TestDraftState_ResetFlagClearsState — simulated: remove the state
// file before LoadDraftState runs. This is what --reset-state does at
// the cmd layer, and the load function must still return a usable
// empty state.
func TestDraftState_ResetFlagClearsState(t *testing.T) {
	path := newStatePath(t)
	// Write a non-empty state first.
	if err := SaveDraftState(path, &DraftState{
		Version: DraftStateVersion,
		Entries: map[string]DraftEntry{"a.md": {ContentHash: "sha256:x"}},
	}); err != nil {
		t.Fatal(err)
	}
	// Simulate --reset-state.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	s, err := LoadDraftState(path)
	if err != nil {
		t.Fatalf("LoadDraftState after reset: %v", err)
	}
	if len(s.Entries) != 0 {
		t.Errorf("expected empty entries after reset, got %d", len(s.Entries))
	}
}

// TestDraftState_IdempotentFast — AC-7 performance target: a second
// run that cache-hits every file must complete well under 100ms. At
// the unit level we approximate this by running AnnotateAndDiff on a
// fixture with 10 synthetic entries and asserting the whole loop runs
// in a trivial amount of time. The benchmark below (BenchmarkDraftState)
// gives a per-op number for tracking regressions.
func TestDraftState_IdempotentFast(t *testing.T) {
	prev := &DraftState{
		Version: DraftStateVersion,
		Entries: make(map[string]DraftEntry, 10),
	}
	current := make(map[string][]Suggestion, 10)
	for i := 0; i < 10; i++ {
		name := filepath.Join("fixture", "doc-"+string(rune('a'+i))+".md")
		s := sugg("style", "info", "line "+string(rune('a'+i)))
		prev.Entries[name] = DraftEntry{Suggestions: []Suggestion{s}}
		current[name] = []Suggestion{s}
	}

	start := time.Now()
	diff, _ := AnnotateAndDiff(prev, current)
	elapsed := time.Since(start)
	if diff.Persisting != 10 {
		t.Errorf("expected 10 persisting, got %+v", diff)
	}
	// 100ms is the CLI target for a full run; AnnotateAndDiff alone
	// should run in microseconds, but we use 10ms as a very loose
	// upper bound so CI machines don't flake the test.
	if elapsed > 10*time.Millisecond {
		t.Errorf("AnnotateAndDiff over 10 entries took %v (want <10ms)", elapsed)
	}
}

// BenchmarkDraftState_AnnotateAndDiff tracks per-op cost of the core
// diff loop so regressions show up in `go test -bench`. Not gated in
// the AC but cheap to include and doubles as documentation.
func BenchmarkDraftState_AnnotateAndDiff(b *testing.B) {
	prev := &DraftState{Version: DraftStateVersion, Entries: map[string]DraftEntry{}}
	current := map[string][]Suggestion{}
	for i := 0; i < 100; i++ {
		name := "doc-" + string(rune('a'+i%26)) + string(rune('0'+i/26)) + ".md"
		s := sugg("style", "info", "line "+name)
		prev.Entries[name] = DraftEntry{Suggestions: []Suggestion{s}}
		current[name] = []Suggestion{s}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = AnnotateAndDiff(prev, current)
	}
}

// TestPruneMissingEntries_NilSafe — ensures the helper handles nil and
// empty inputs without panicking. The runner always passes a real
// state, but defensive-test the edge so future callers can trust it.
func TestPruneMissingEntries_NilSafe(t *testing.T) {
	if got := PruneMissingEntries(nil, nil); got != 0 {
		t.Errorf("nil state: got %d", got)
	}
	empty := &DraftState{Entries: map[string]DraftEntry{}}
	if got := PruneMissingEntries(empty, nil); got != 0 {
		t.Errorf("empty state: got %d", got)
	}
}

// TestFindingHash_AllSemanticFieldsParticipate pins the canonical
// form of findingHash so a future refactor cannot silently drop one
// of the contributing fields. Paranoid-review fix (2026-04-11 MEDIUM
// test coverage): findingHash is load-bearing for the NEW/PERSISTING
// lifecycle classification, and a regression that collapsed two
// fields together would ship as an invisible false-positive
// reduction.
//
// For each field (Category, Severity, Message), the test builds a
// baseline Suggestion, flips the field, and asserts the hash
// changes. Also asserts that whitespace/case variants of Message
// map to the SAME hash (the intended normalization behavior).
func TestFindingHash_AllSemanticFieldsParticipate(t *testing.T) {
	base := Suggestion{
		Category: "structure",
		Severity: "warning",
		Message:  "missing H1 heading",
	}
	baseHash := findingHash(base)

	t.Run("Category flip", func(t *testing.T) {
		alt := base
		alt.Category = "style"
		if findingHash(alt) == baseHash {
			t.Error("hash unchanged after Category flip")
		}
	})
	t.Run("Severity flip", func(t *testing.T) {
		alt := base
		alt.Severity = "error"
		if findingHash(alt) == baseHash {
			t.Error("hash unchanged after Severity flip")
		}
	})
	t.Run("Message flip", func(t *testing.T) {
		alt := base
		alt.Message = "missing H2 heading"
		if findingHash(alt) == baseHash {
			t.Error("hash unchanged after Message flip")
		}
	})
	t.Run("Message whitespace normalized", func(t *testing.T) {
		alt := base
		alt.Message = "  missing    H1   heading  "
		if findingHash(alt) != baseHash {
			t.Error("hash changed after whitespace-only variation")
		}
	})
	t.Run("Message case normalized", func(t *testing.T) {
		alt := base
		alt.Message = "MISSING H1 HEADING"
		if findingHash(alt) != baseHash {
			t.Error("hash changed after case-only variation")
		}
	})
	// TODO(S3-L3): add a test that verifies AnalyzerSchemaVersion is checked
	// on load and stale entries are treated as cache misses.
	// TODO(S3-M2): add a test for QuarantineCorruptState round-trip.
	t.Run("Pipe-in-field does not collide", func(t *testing.T) {
		// Paranoid-review fix rationale: NUL separator prevents a
		// pipe-in-category from colliding with a legitimately
		// distinct entry. We cannot easily construct a known
		// collision against the old format, but we can assert that
		// "a|b" in Category hashes differently than "a" + something
		// that would previously concat to "a|b".
		a := Suggestion{Category: "a|b", Severity: "x", Message: "y"}
		b := Suggestion{Category: "a", Severity: "b|x", Message: "y"}
		if findingHash(a) == findingHash(b) {
			t.Error("pipe-in-Category collides with pipe-in-Severity (separator is broken)")
		}
	})
}
