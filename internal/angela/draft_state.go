// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package angela — draft_state.go
//
// Differential draft — hash-based incremental analysis.
//
// Running `lore angela draft --all` on a 60+ doc corpus produces the same
// ~130 findings week after week. The user drowns in repeat noise and
// misses what actually changed. This file adds a JSON state file that
// records each document's content hash and last-computed suggestions so
// that a second run can:
//
//  1. Skip analysis entirely for docs whose hash matches the stored value
//  2. Classify every finding as NEW, PERSISTING, or RESOLVED relative to
//     the previous run
//  3. Let the user hide the persisting noise with `--diff-only`
//
// Draft stays strictly offline (invariant I1 from the Angela MVP v1 spec):
// hashing is SHA-256 of the raw file bytes, diffing is pure Go, the state
// file is local JSON written under the state directory.
package angela

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/greycoderk/lore/internal/fileutil"
)

// ErrStateCorrupt is returned by LoadDraftState / LoadReviewState when
// the on-disk file exists but cannot be parsed or has an unexpected
// schema version. Callers can use `errors.Is(err, ErrStateCorrupt)` to
// decide whether to quarantine the file via QuarantineCorruptState
// before overwriting it with a fresh snapshot. The previous contract
// silently overwrote corrupt files on the next run, losing every
// resolve/ignore mark a user had accumulated.
var ErrStateCorrupt = errors.New("state file corrupt or incompatible")

// QuarantineCorruptState renames a broken state file aside with a
// timestamped `.corrupt-<ts>` suffix so the user can recover it by
// hand if needed. Returns the quarantine path on success, or an empty
// string and an error if the rename fails (e.g. perms). A best-effort
// recovery: callers should treat a QuarantineCorruptState failure as
// a reason to NOT save over the original.
func QuarantineCorruptState(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("angela: quarantine: empty path")
	}
	stamp := time.Now().UTC().Format("20060102T150405.000")
	quarPath := path + ".corrupt-" + stamp
	if err := os.Rename(path, quarPath); err != nil {
		return "", fmt.Errorf("angela: quarantine: rename: %w", err)
	}
	return quarPath, nil
}

// DraftStateVersion is the schema version of the persisted state file.
// Bump it whenever an incompatible change is made to DraftState or
// DraftEntry — LoadDraftState will notice the mismatch and return a
// fresh empty state instead of garbage.
//
// Version 2: added AnalyzerSchemaVersion to DraftEntry so stale
// cached suggestions are invalidated when the analyzer's internal
// schema evolves (e.g. a persona registry edit or a new coherence
// rule). Version 1 entries cannot be re-used safely because we
// cannot know which analyzer version produced them.
const DraftStateVersion = 2

// AnalyzerSchemaVersion is a monotonic integer bumped whenever the
// output of AnalyzeDraft / CheckCoherence / ScoreDocument / the
// persona registry changes in a way that makes cached suggestions
// from the old version invalid. Cached entries whose
// AnalyzerSchemaVersion differs from this value are treated as a
// cache miss on read. Bump this on any behavior-affecting change in
// internal/angela/draft.go, coherence.go, score.go, or personas.
const AnalyzerSchemaVersion = 1

// DiffStatus* constants are the string values written into
// Suggestion.DiffStatus (draft) and ReviewFinding.DiffStatus (review)
// by the differential runners. Exported so callers in cmd/ can
// reference them without magic strings.
//
// A single unified namespace shared by both draft-side and
// review-side code. The only extra status is DiffStatusRegressed,
// which only review uses (a draft finding cannot regress because it
// has no lifecycle marks).
const (
	DiffStatusNew        = "new"
	DiffStatusPersisting = "persisting"
	DiffStatusRegressed  = "regressed" // review-only
	DiffStatusResolved   = "resolved"
)

// DraftState is the on-disk schema for the draft state file. One
// instance per corpus. Entries is keyed by filename (relative to
// docsDir) so lookup is O(1) per document in the run loop.
type DraftState struct {
	Version int                    `json:"version"`
	LastRun time.Time              `json:"last_run"`
	Entries map[string]DraftEntry  `json:"entries"`
}

// DraftEntry is one document's cached analysis result. Suggestions are
// stored BEFORE any severity-override / strict-mode processing so that
// reading old state under new config gives the same answer as a fresh
// run would.
// LastAnalyzed is stamped with time.Now() by the runner; it is not
// mockable at the DraftEntry level. Tests that need deterministic
// timestamps should override at the runner/caller layer.
type DraftEntry struct {
	ContentHash          string       `json:"content_hash"` // "sha256:<hex>"
	LastAnalyzed         time.Time    `json:"last_analyzed"`
	Suggestions          []Suggestion `json:"suggestions"`
	Score                int          `json:"score"`
	Grade                string       `json:"grade"`
	Profile              string       `json:"profile"`
	AnalyzerSchemaVersion int         `json:"analyzer_schema_version"`
}

// DraftDiff summarises the per-run difference between the previous
// DraftState and the current analysis results. Counts are aggregated
// across the whole corpus; per-finding labels live on Suggestion.DiffStatus.
type DraftDiff struct {
	New        int `json:"new"`
	Persisting int `json:"persisting"`
	Resolved   int `json:"resolved"`
}

// ContentHash returns "sha256:<hex>" for the given bytes. Exported so
// the runner and tests can compute the same value. The sha256 prefix is
// intentional: it future-proofs the schema against a later switch to
// BLAKE3 or similar — consumers can check the prefix before comparing.
func ContentHash(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// LoadDraftState reads a state file from path. Missing file → empty
// state with the current version (NOT an error). Corrupt file or
// wrong version → empty state plus a non-nil error that the caller can
// log as a notice before proceeding. Returning a usable state in every
// case keeps the runner loop simple: there is no "state is missing"
// branch to handle.
func LoadDraftState(path string) (*DraftState, error) {
	empty := &DraftState{
		Version: DraftStateVersion,
		Entries: make(map[string]DraftEntry),
	}

	// Cap the file size to protect against an accidentally-huge state
	// file (corrupt runaway append, DoS from a shared CI volume).
	const maxStateFileSize = 64 << 20 // 64 MiB
	info, statErr := os.Stat(path)
	if statErr != nil {
		if errors.Is(statErr, fs.ErrNotExist) {
			return empty, nil
		}
		return empty, fmt.Errorf("angela: draft state: stat: %w", statErr)
	}
	if info.Size() > maxStateFileSize {
		return empty, fmt.Errorf("angela: draft state: file too large (%d bytes): %w", info.Size(), ErrStateCorrupt)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return empty, nil
		}
		return empty, fmt.Errorf("angela: draft state: read: %w", err)
	}

	var state DraftState
	if err := json.Unmarshal(data, &state); err != nil {
		return empty, fmt.Errorf("angela: draft state: parse: %w: %w", ErrStateCorrupt, err)
	}

	// Version check — any mismatch means the schema has evolved and the
	// cached suggestions might not match today's analyzer output. Fresh
	// state forces a full re-analysis next loop iteration.
	if state.Version != DraftStateVersion {
		return empty, fmt.Errorf("angela: draft state: version %d != expected %d: %w", state.Version, DraftStateVersion, ErrStateCorrupt)
	}

	// Defensive: nil map in the JSON would crash the runner on first
	// insert. Normalize it here so callers can trust the shape.
	if state.Entries == nil {
		state.Entries = make(map[string]DraftEntry)
	}
	// Drop cached entries whose Suggestion slice contains unknown
	// severity or category values. Such entries are evidence of a
	// state file mutated by an older analyzer (or by a human editing
	// the JSON); using them verbatim would leak the foreign vocabulary
	// into the next run's output. Treat the entry as a cache miss —
	// the runner re-analyzes the file and rewrites a clean entry.
	var invalidKeys []string
	for name, entry := range state.Entries {
		if !suggestionsAllValid(entry.Suggestions) {
			invalidKeys = append(invalidKeys, name)
		}
	}
	for _, name := range invalidKeys {
		delete(state.Entries, name)
	}
	return &state, nil
}

// validDraftSeverities / validDraftCategories are the closed sets of
// values that AnalyzeDraft and CheckCoherence emit. Cached entries
// containing anything else are considered untrustworthy.
var (
	validDraftSeverities = map[string]bool{
		"info":    true,
		"warning": true,
		"error":   true,
	}
	validDraftCategories = map[string]bool{
		"structure":    true,
		"completeness": true,
		"style":        true,
		"coherence":    true,
		"persona":      true,
		"io":           true,
	}
	validDraftDiffStatus = map[string]bool{
		"":                 true, // entries from a non-differential run
		DiffStatusNew:      true,
		DiffStatusPersisting: true,
		DiffStatusResolved: true,
	}
)

// suggestionsAllValid returns true when every suggestion in s has a
// recognized category, severity, and diff status. Used by LoadDraftState
// to reject tampered cached entries.
func suggestionsAllValid(s []Suggestion) bool {
	for _, sg := range s {
		if !validDraftCategories[sg.Category] {
			return false
		}
		if !validDraftSeverities[sg.Severity] {
			return false
		}
		if !validDraftDiffStatus[sg.DiffStatus] {
			return false
		}
	}
	return true
}

// SaveDraftState writes state to path atomically (tempfile + rename in
// the same directory). Creates the parent directory if missing so
// callers don't need to pre-mkdir the state root. The file is indented
// JSON: state files are small (one entry per doc, ~200 bytes each) and
// human diff-ability is worth more than the handful of bytes saved.
//
// The tempfile+sync+rename+fsync dance lives in fileutil.AtomicWriteJSON,
// shared with SaveReviewState and the polish backup writers.
func SaveDraftState(path string, state *DraftState) error {
	if state == nil {
		return fmt.Errorf("angela: draft state: nil state")
	}
	state.Version = DraftStateVersion
	state.LastRun = time.Now().UTC()

	dir := filepath.Dir(path)
	// State directories are 0700 so unrelated local users cannot list
	// or read the cached suggestions (they contain document titles,
	// coherence findings, and other metadata users would not expect
	// to be world-readable).
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("angela: draft state: mkdir: %w", err)
	}
	if err := fileutil.AtomicWriteJSON(path, state, 0o600); err != nil {
		return fmt.Errorf("angela: draft state: %w", err)
	}
	return nil
}

// findingHash returns a stable identity key for a Suggestion. Two
// findings with the same category, severity, and normalized message
// (lowercased, whitespace-collapsed) hash to the same value and are
// therefore considered "the same finding" across runs. The hash is
// deliberately short (16 hex chars = 64 bits) — collisions inside a
// single document's findings list are astronomically unlikely.
//
// The canonical form is NUL-separated instead of `|`-separated so a
// future finding whose Category, Severity, or Message contains a
// literal pipe cannot collide with a legitimately distinct entry.
func findingHash(s Suggestion) string {
	norm := strings.ToLower(strings.TrimSpace(s.Message))
	norm = wsRe.ReplaceAllString(norm, " ")
	var b strings.Builder
	b.WriteString(s.Category)
	b.WriteByte(0)
	b.WriteString(s.Severity)
	b.WriteByte(0)
	b.WriteString(norm)
	h := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(h[:8])
}

// AnnotateAndDiff walks the per-file suggestions, tags each one with a
// DiffStatus relative to the previous state, and produces the aggregate
// DraftDiff counts. It also returns a slice of "resolved" suggestions
// (previous findings that disappeared) so the reporter can show them
// even though they don't belong to any current file row.
//
// prevHashes is consumed destructively; callers must not reuse prev
// after calling this function.
//
// The resolved slice is built by iterating the previous state's
// entries: any stored suggestion whose findingHash does not appear in
// the current run is a RESOLVED finding. This includes both:
//
//  1. Files still present in the corpus where a specific finding went
//     away (the user fixed it)
//  2. Files deleted entirely from the corpus (their old findings all
//     become resolved in one go)
//
// `currentFiles` is indexed by the same filename keys as `prev.Entries`
// so we only diff pairs that were loaded from the same place.
func AnnotateAndDiff(prev *DraftState, currentFiles map[string][]Suggestion) (DraftDiff, []ResolvedSuggestion) {
	var diff DraftDiff
	var resolved []ResolvedSuggestion

	// Build per-file previous hash sets for O(1) lookup while we walk
	// the current suggestions. Done once upfront so each file's diff is
	// linear in len(currentFiles[file]) + len(prev.Entries[file].Suggestions).
	prevHashes := make(map[string]map[string]Suggestion, len(prev.Entries))
	for filename, entry := range prev.Entries {
		set := make(map[string]Suggestion, len(entry.Suggestions))
		for _, s := range entry.Suggestions {
			set[findingHash(s)] = s
		}
		prevHashes[filename] = set
	}

	// Pass 1: label every current suggestion NEW or PERSISTING, and
	// remove it from the prev set so what remains is the RESOLVED set.
	for filename, suggestions := range currentFiles {
		prevSet := prevHashes[filename]
		for i := range suggestions {
			h := findingHash(suggestions[i])
			if _, ok := prevSet[h]; ok {
				suggestions[i].DiffStatus = DiffStatusPersisting
				delete(prevSet, h)
				diff.Persisting++
			} else {
				suggestions[i].DiffStatus = DiffStatusNew
				diff.New++
			}
		}
	}

	// Pass 2: anything left in prevHashes is a finding that no longer
	// exists in the current run — either the file was edited (finding
	// went away) or the file was deleted (everything left is a resolve).
	for filename, remaining := range prevHashes {
		for _, s := range remaining {
			s.DiffStatus = DiffStatusResolved
			resolved = append(resolved, ResolvedSuggestion{
				File:       filename,
				Suggestion: s,
			})
			diff.Resolved++
		}
	}

	// Sort resolved by filename then by message so the report is
	// deterministic run-to-run. The runner calls this function once so
	// the sort cost is negligible (typically ≤10 items).
	sort.Slice(resolved, func(i, j int) bool {
		if resolved[i].File != resolved[j].File {
			return resolved[i].File < resolved[j].File
		}
		return resolved[i].Suggestion.Message < resolved[j].Suggestion.Message
	})

	return diff, resolved
}

// ResolvedSuggestion pairs a RESOLVED finding with the file it used to
// live in. Needed in the reporter because a resolved finding doesn't
// belong to any current file row (the file may have been deleted).
type ResolvedSuggestion struct {
	File       string     `json:"file"`
	Suggestion Suggestion `json:"suggestion"`
}

// PruneMissingEntries removes state entries for files that no longer
// exist in the current corpus. Called by the runner after the per-file
// loop so the state file stays small and accurate.
// Returns the number of entries removed so the caller can log a
// verbose notice if desired.
//
// Snapshot the keys before deleting so the semantics are obvious
// and future-proof against accidental concurrent writes.
func PruneMissingEntries(state *DraftState, currentFiles map[string]bool) int {
	if state == nil || len(state.Entries) == 0 {
		return 0
	}
	toDelete := make([]string, 0, len(state.Entries))
	for name := range state.Entries {
		if !currentFiles[name] {
			toDelete = append(toDelete, name)
		}
	}
	for _, name := range toDelete {
		delete(state.Entries, name)
	}
	return len(toDelete)
}
