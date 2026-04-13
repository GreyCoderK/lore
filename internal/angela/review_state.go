// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package angela — review_state.go
//
// JSON state file tracking the lifecycle of every review finding
// across runs (differential review).
//
// Where the draft state file is a content-hash cache that short-circuits
// expensive analysis, the review state file is a
// finding-lifecycle tracker. Review calls cost an AI round-trip every
// time, so we don't try to skip the call — instead we annotate each
// returned finding as NEW / PERSISTING / REGRESSED / RESOLVED relative
// to the previous run, and let the user mark findings as resolved or
// ignored to keep the noise floor low.
//
// REGRESSED is the highest-signal status: a finding that the user
// previously marked `resolved` (it was supposedly fixed) or `ignored`
// (it was a known false positive) has resurfaced. The validator surfaces
// these prominently so the user notices that something has come back.
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
	"unicode"

	"github.com/greycoderk/lore/internal/fileutil"
)

// ReviewStateVersion is the schema version of the review state file.
// Bump it on any incompatible change to ReviewState or StatefulFinding —
// LoadReviewState detects the mismatch and returns a fresh empty state
// alongside a non-nil error so the caller can log a notice.
const ReviewStateVersion = 1

// Status* constants are the lifecycle values stored in
// StatefulFinding.Status. They are NOT the same set as DiffStatus*: a
// finding's persistent status is one of {active, resolved, ignored},
// while its per-run DiffStatus is one of {new, persisting, regressed,
// resolved}. The transition table is documented in UpdateReviewState.
const (
	StatusActive   = "active"
	StatusResolved = "resolved"
	StatusIgnored  = "ignored"
)

// ReviewDiff* aliases of the unified DiffStatus* constants. Kept so
// existing callers compile unchanged; new code should use
// DiffStatus* directly.
//
// The per-run diff labels alias the unified DiffStatus* family in
// draft_state.go so the two packages cannot drift.
const (
	ReviewDiffNew        = DiffStatusNew
	ReviewDiffPersisting = DiffStatusPersisting
	ReviewDiffRegressed  = DiffStatusRegressed
	ReviewDiffResolved   = DiffStatusResolved
)

// ReviewState is the on-disk schema for the review state file. One
// entry per stable finding hash (NOT per run). Findings is keyed by
// the 16-char hex hash returned by ReviewFindingHash.
type ReviewState struct {
	Version  int                          `json:"version"`
	LastRun  time.Time                    `json:"last_run"`
	Findings map[string]StatefulFinding   `json:"findings"`
}

// StatefulFinding wraps a ReviewFinding with the lifecycle metadata
// the differential runner needs: when did we first see this issue,
// when did we last see it, and what is its current status. Stored
// once per stable hash; the same finding across many runs collapses
// into a single entry whose LastSeen drifts forward.
type StatefulFinding struct {
	Finding      ReviewFinding `json:"finding"`
	Status       string        `json:"status"` // active|resolved|ignored
	FirstSeen    time.Time     `json:"first_seen"`
	LastSeen     time.Time     `json:"last_seen"`
	ResolvedAt   *time.Time    `json:"resolved_at,omitempty"`
	ResolvedBy   string        `json:"resolved_by,omitempty"`
	IgnoreReason string        `json:"ignore_reason,omitempty"`
}

// ReviewDiff aggregates the per-run lifecycle counts and the slices
// the reporter walks. NEW + REGRESSED are the high-signal lists; the
// PERSISTING and RESOLVED lists exist so --diff-only mode can still
// report counts even when it hides the rows.
type ReviewDiff struct {
	New        []ReviewFinding `json:"new,omitempty"`
	Persisting []ReviewFinding `json:"persisting,omitempty"`
	Regressed  []ReviewFinding `json:"regressed,omitempty"`
	Resolved   []ReviewFinding `json:"resolved,omitempty"`
}

// Counts returns the four counts in a single struct for the summary
// line in the human reporter.
func (d ReviewDiff) Counts() (new_, persisting, regressed, resolved int) {
	return len(d.New), len(d.Persisting), len(d.Regressed), len(d.Resolved)
}

// ReviewFindingHash returns a stable identity for a review finding
// scoped to a specific audience. Inputs are severity, sorted document
// filenames, normalized title, and a normalized audience key. The
// description is intentionally NOT part of the hash so an AI that
// rephrases the same finding next week still maps to the same entry.
//
// The hash is the first 16 hex chars of SHA-256 over a canonical
// NUL-separated form — 64 bits of identity. NUL is chosen as separator
// because it cannot legally appear inside any of the inputs (severity
// is a fixed vocabulary, filenames reject NUL on every supported OS,
// titles are text). This avoids a `|`-delimiter collision where a doc
// named `a,b.md` in Documents would merge with two docs `[a.md, b.md]`.
//
// Previously the hash was pipe-delimited and audience-agnostic, so
// running `review` followed by `review --for CTO` surfaced every
// finding as NEW. Audience is now part of the canonical input so the
// two runs maintain independent lifecycles.
// Callers that need audience-scoped hashes must call HashReviewFindingWithAudience directly.
// ReviewFindingHash is retained for test compatibility; production uses HashReviewFindingWithAudience.
// This wrapper is for audience-agnostic contexts only.
func ReviewFindingHash(f ReviewFinding) string {
	return HashReviewFindingWithAudience(f, "")
}

// HashReviewFindingWithAudience computes the stable identity for a
// finding under a specific audience. Empty audience is the plain
// "default" lifecycle.
func HashReviewFindingWithAudience(f ReviewFinding, audience string) string {
	docs := append([]string(nil), f.Documents...)
	sort.Strings(docs)
	title := normalizeTitle(f.Title)
	aud := normalizeAudience(audience)
	sev := strings.ToLower(strings.TrimSpace(f.Severity))
	// NUL-delimited canonical form. sha256 absorbs raw bytes so the
	// NUL is perfectly safe to use here.
	var b strings.Builder
	b.WriteString(sev)
	b.WriteByte(0)
	for _, d := range docs {
		b.WriteString(d)
		b.WriteByte(0)
	}
	b.WriteByte(0) // second NUL terminates the docs list
	b.WriteString(title)
	b.WriteByte(0)
	b.WriteString(aud)
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:8])
}

// normalizeAudience trims, lowercases, and collapses whitespace so
// `--for "CTO"` and `--for cto ` map to the same lifecycle bucket.
func normalizeAudience(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = wsRe.ReplaceAllString(s, " ")
	return s
}

// normalizeTitle lowercases, collapses whitespace, and strips simple
// punctuation so cosmetic title changes don't break hash stability.
// We deliberately keep this minimal — Unicode normalization is out of
// scope for MVP, matching the same decision in evidence_validator.go.
func normalizeTitle(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = wsRe.ReplaceAllString(s, " ")
	// Strip a small set of common punctuation that AIs vary on without
	// changing meaning. Keep alphanumerics, spaces, and hyphens.
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z',
			r >= '0' && r <= '9',
			r == ' ', r == '-',
			unicode.IsLetter(r):
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

// LoadReviewState reads a state file from path. Missing file → fresh
// empty state, no error. Corrupt file or version mismatch → fresh
// empty state plus a non-nil error so the runner can log a notice.
// Same fallback-first contract as LoadDraftState.
func LoadReviewState(path string) (*ReviewState, error) {
	empty := &ReviewState{
		Version:  ReviewStateVersion,
		Findings: make(map[string]StatefulFinding),
	}

	const maxStateFileSize = 64 << 20 // 64 MiB — see LoadDraftState rationale
	info, statErr := os.Stat(path)
	if statErr != nil {
		if errors.Is(statErr, fs.ErrNotExist) {
			return empty, nil
		}
		return empty, fmt.Errorf("angela: review state: stat: %w", statErr)
	}
	if info.Size() > maxStateFileSize {
		return empty, fmt.Errorf("angela: review state: file too large (%d bytes): %w", info.Size(), ErrStateCorrupt)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return empty, nil
		}
		return empty, fmt.Errorf("angela: review state: read: %w", err)
	}
	var state ReviewState
	if err := json.Unmarshal(data, &state); err != nil {
		return empty, fmt.Errorf("angela: review state: parse: %w: %w", ErrStateCorrupt, err)
	}
	if state.Version != ReviewStateVersion {
		return empty, fmt.Errorf("angela: review state: version %d != expected %d: %w", state.Version, ReviewStateVersion, ErrStateCorrupt)
	}
	if state.Findings == nil {
		state.Findings = make(map[string]StatefulFinding)
	}
	for h, sf := range state.Findings {
		switch sf.Status {
		case StatusActive, StatusResolved, StatusIgnored:
			// valid
		default:
			sf.Status = StatusActive
			state.Findings[h] = sf
		}
	}
	return &state, nil
}

// SaveReviewState writes state to path atomically (tempfile in the
// same dir + os.Rename). Creates the parent directory if missing so
// callers don't have to mkdir first. Stamps LastRun at save time.
//
// Delegates the tempfile+sync+rename+fsync dance to
// fileutil.AtomicWriteJSON so durability tweaks touch one place.
func SaveReviewState(path string, state *ReviewState) error {
	if state == nil {
		return fmt.Errorf("angela: review state: nil state")
	}
	state.Version = ReviewStateVersion
	state.LastRun = time.Now().UTC()

	dir := filepath.Dir(path)
	// 0700 — see SaveDraftState.
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("angela: review state: mkdir: %w", err)
	}
	if err := fileutil.AtomicWriteJSON(path, state, 0o600); err != nil {
		return fmt.Errorf("angela: review state: %w", err)
	}
	return nil
}

// ComputeReviewDiff classifies each current finding relative to the
// previous state and produces a ReviewDiff with NEW / PERSISTING /
// REGRESSED / RESOLVED slices. Each input finding's Hash is populated
// in place so callers don't need to recompute it.
//
// REGRESSED is the high-signal class: a finding the user previously
// marked resolved or ignored has come back, meaning either the fix
// regressed or the false-positive assumption was wrong.
//
// RESOLVED here means "stored as active in prev, missing from current"
// — i.e. the corpus changed and the finding naturally went away. This
// is distinct from `StatusResolved` which is the user's explicit mark.
//
// ComputeReviewDiff is retained for test compatibility; production uses ComputeReviewDiffWithRejected.
//
// Deprecated: kept as a thin wrapper for callers that don't care
// about audience scoping. New code should call
// ComputeReviewDiffWithAudience.
func ComputeReviewDiff(prev *ReviewState, current []ReviewFinding) ReviewDiff {
	return ComputeReviewDiffWithAudience(prev, current, "")
}

// ComputeReviewDiffWithAudience is the audience-scoped variant. All
// finding hashes are computed under `audience`, so running review
// without --for and then with --for keeps two independent lifecycles
// keeps two independent lifecycles.
//
// Rejected findings (e.g. validator drops) should be
// passed via ComputeReviewDiffWithRejected so they do NOT appear as
// natural RESOLVED — the AI kept returning them, they were just
// suppressed client-side. This version treats rejected as absent
// which is only correct when there are none.
func ComputeReviewDiffWithAudience(prev *ReviewState, current []ReviewFinding, audience string) ReviewDiff {
	return ComputeReviewDiffWithRejected(prev, current, nil, audience)
}

// ComputeReviewDiffWithRejected is the full-context variant used by
// the review runner when the evidence validator is active.
// `rejected` is the set of findings the AI produced this run but
// that the validator dropped for bad/missing evidence; they are
// considered "still present" for lifecycle purposes so previously
// stored entries matching them are not mistakenly classified as
// RESOLVED. The hashes for rejected findings are computed but they
// are NOT added to the NEW/PERSISTING/REGRESSED slices — the user
// already saw them via the report.Rejected surface.
//
func ComputeReviewDiffWithRejected(prev *ReviewState, current []ReviewFinding, rejected []RejectedFinding, audience string) ReviewDiff {
	var diff ReviewDiff
	if prev == nil {
		prev = &ReviewState{Findings: map[string]StatefulFinding{}}
	}

	// Hash every current finding once and index by hash so we can
	// detect "active in prev but missing from current" in pass 2.
	currentByHash := make(map[string]int, len(current))
	for i := range current {
		current[i].Hash = HashReviewFindingWithAudience(current[i], audience)
		currentByHash[current[i].Hash] = i
	}
	// Hash rejected findings too so pass 2 treats them as "seen"
	// even though they are not in `current`.
	rejectedHashes := make(map[string]bool, len(rejected))
	for _, r := range rejected {
		h := HashReviewFindingWithAudience(r.Finding, audience)
		rejectedHashes[h] = true
	}

	// Pass 1: classify every current finding.
	for i := range current {
		f := &current[i]
		entry, seen := prev.Findings[f.Hash]
		switch {
		case !seen:
			f.DiffStatus = ReviewDiffNew
			diff.New = append(diff.New, *f)
		case entry.Status == StatusActive:
			f.DiffStatus = ReviewDiffPersisting
			diff.Persisting = append(diff.Persisting, *f)
		default:
			// Stored as resolved or ignored → user thought it was
			// dealt with → flag REGRESSED.
			f.DiffStatus = ReviewDiffRegressed
			diff.Regressed = append(diff.Regressed, *f)
		}
	}

	// Pass 2: any prev entry with status=active that didn't appear in
	// the current run is a NATURAL resolve (the corpus changed). User-
	// marked resolved/ignored entries do NOT show up in this list —
	// they're already accounted for and not part of the run summary.
	// Entries that matched a validator-rejected finding are ALSO
	// skipped: the AI kept returning them, they were just hidden
	// client-side, so naming them RESOLVED would lie to the user.
	for hash, entry := range prev.Findings {
		if _, ok := currentByHash[hash]; ok {
			continue
		}
		if rejectedHashes[hash] {
			continue
		}
		if entry.Status != StatusActive {
			continue
		}
		f := entry.Finding
		f.Hash = hash
		f.DiffStatus = ReviewDiffResolved
		diff.Resolved = append(diff.Resolved, f)
	}

	// Determinism: sort each slice by hash so run-to-run output is
	// stable even if the AI returns findings in a different order.
	sortByHash(diff.New)
	sortByHash(diff.Persisting)
	sortByHash(diff.Regressed)
	sortByHash(diff.Resolved)
	return diff
}

// sortByHash is a tiny helper extracted because all four diff slices
// need the same sort order.
func sortByHash(s []ReviewFinding) {
	sort.SliceStable(s, func(i, j int) bool { return s[i].Hash < s[j].Hash })
}

// UpdateReviewState merges the diff back into the persisted state so
// the next run has an accurate snapshot. Rules:
//
//   - NEW finding → insert with status=active, FirstSeen=LastSeen=now
//   - PERSISTING finding → bump LastSeen, keep status (active)
//   - REGRESSED finding → flip status back to active, bump LastSeen,
//     clear ResolvedAt/ResolvedBy/IgnoreReason so the user knows it's
//     been re-opened, and remember the regression in LastSeen
//   - RESOLVED finding (natural) → leave the prev entry alone — it
//     simply stops bumping LastSeen. The user can prune it later via
//     the `log` subcommand if they care.
//
// User-marked statuses (set via the resolve/ignore subcommands) are
// preserved across runs. If the user resolved a finding and the AI
// stops returning it, the entry stays in state with its resolved
// timestamp intact.
// Document rename → shadow duplicates: When corpus documents
// are renamed, findings citing the old name naturally RESOLVE and
// re-appear as NEW under the new name. Users who had ignored the old
// finding will not see the connection. A future dedup pass could match
// by title+severity across audience-scoped hashes to detect renames.
func UpdateReviewState(state *ReviewState, diff ReviewDiff, now time.Time) {
	if state == nil {
		return
	}
	if state.Findings == nil {
		state.Findings = make(map[string]StatefulFinding)
	}
	for _, f := range diff.New {
		state.Findings[f.Hash] = StatefulFinding{
			Finding:   f,
			Status:    StatusActive,
			FirstSeen: now,
			LastSeen:  now,
		}
	}
	for _, f := range diff.Persisting {
		entry := state.Findings[f.Hash]
		entry.Finding = f
		entry.LastSeen = now
		state.Findings[f.Hash] = entry
	}
	for _, f := range diff.Regressed {
		entry := state.Findings[f.Hash]
		entry.Finding = f
		entry.Status = StatusActive
		entry.LastSeen = now
		entry.ResolvedAt = nil
		entry.ResolvedBy = ""
		entry.IgnoreReason = ""
		state.Findings[f.Hash] = entry
	}
}

// MarkResolved flips a stored finding's status to "resolved" and
// stamps ResolvedAt + ResolvedBy. Used by the `lore angela review
// resolve` subcommand. Returns an error if the hash is not in state.
func MarkResolved(state *ReviewState, hash, by string, now time.Time) error {
	if state == nil {
		return fmt.Errorf("angela: review state: nil state")
	}
	entry, ok := state.Findings[hash]
	if !ok {
		return fmt.Errorf("angela: review state: hash %s not found", hash)
	}
	entry.Status = StatusResolved
	stamp := now
	entry.ResolvedAt = &stamp
	entry.ResolvedBy = by
	entry.IgnoreReason = ""
	state.Findings[hash] = entry
	return nil
}

// MarkIgnored flips a stored finding's status to "ignored" and
// records the user's reason. Used by `lore angela review ignore`.
// Returns an error if the hash is not in state OR the reason is empty
// (ignore must be deliberate).
func MarkIgnored(state *ReviewState, hash, reason string, now time.Time) error {
	if state == nil {
		return fmt.Errorf("angela: review state: nil state")
	}
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("angela: review state: ignore requires a reason")
	}
	entry, ok := state.Findings[hash]
	if !ok {
		return fmt.Errorf("angela: review state: hash %s not found", hash)
	}
	entry.Status = StatusIgnored
	stamp := now
	entry.ResolvedAt = &stamp
	entry.IgnoreReason = strings.TrimSpace(reason)
	entry.ResolvedBy = ""
	state.Findings[hash] = entry
	return nil
}

// ResolveByPrefix returns the full hash matching a user-supplied
// prefix. Required so the resolve / ignore subcommands can accept
// abbreviated hashes (first 6 chars if unambiguous).
// Note: the ambiguous-prefix error message leaks full hashes. This is
// acceptable for a CLI tool where the user already sees hashes in the
// review output.
//
// Three outcomes:
//
//   - exactly one match → returns the full hash, nil error
//   - zero matches → returns "", error("hash %s not found")
//   - more than one match → returns "", error listing the candidates
func ResolveByPrefix(state *ReviewState, prefix string) (string, error) {
	if state == nil {
		return "", fmt.Errorf("angela: review state: nil state")
	}
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	if prefix == "" {
		return "", fmt.Errorf("angela: review state: empty hash prefix")
	}
	var matches []string
	for h := range state.Findings {
		if strings.HasPrefix(h, prefix) {
			matches = append(matches, h)
		}
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("angela: review state: hash %s not found", prefix)
	case 1:
		return matches[0], nil
	default:
		sort.Strings(matches)
		return "", fmt.Errorf("angela: review state: ambiguous hash %s matches %d entries: %s",
			prefix, len(matches), strings.Join(matches, ", "))
	}
}

// LogEntries returns the state's findings sorted by LastSeen
// descending — the order the `log` subcommand prints them in.
func LogEntries(state *ReviewState) []StatefulFinding {
	if state == nil || len(state.Findings) == 0 {
		return nil
	}
	out := make([]StatefulFinding, 0, len(state.Findings))
	for _, e := range state.Findings {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].LastSeen.After(out[j].LastSeen)
	})
	return out
}
