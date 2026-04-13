// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package angela — polish_state.go
//
// State file tracking per-document section hashes so re-polish only
// touches sections that changed.
//
// The state schema is intentionally minimal: one entry per polished
// document, each containing a flat map of `## heading` → SHA-256 of
// the section body. Section identity is the heading text (normalized
// whitespace, original casing) so a renamed heading is treated as a
// new section — which is correct: if the heading changed, the AI
// should evaluate the content under its new name.
//
// Opt-in by default: the state file is only written when
// `cfg.Angela.Polish.Incremental.Enabled` is true or the user
// passes `--incremental` on the CLI.
package angela

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/greycoderk/lore/internal/fileutil"
)

// PolishStateVersion is the schema version of the polish state file.
// Bump on any incompatible schema change.
const PolishStateVersion = 1

// PolishState is the on-disk schema for `polish-state.json`. Entries
// is keyed by document filename relative to docsDir.
type PolishState struct {
	Version int                          `json:"version"`
	Entries map[string]PolishStateEntry  `json:"entries"`
}

// PolishStateEntry records the section-level hashes of a document at
// the time it was last polished. SectionHashes keys are the raw `##`
// heading text ("## Why", "## How It Works"); values are "sha256:<hex>"
// produced by ContentHash.
type PolishStateEntry struct {
	LastPolished  time.Time         `json:"last_polished"`
	SectionHashes map[string]string `json:"sections"`
}

// LoadPolishState reads the state file from path. Missing file →
// empty state (no error). Corrupt or version mismatch → empty state
// plus a non-nil error. Same contract as LoadDraftState.
func LoadPolishState(path string) (*PolishState, error) {
	empty := &PolishState{
		Version: PolishStateVersion,
		Entries: make(map[string]PolishStateEntry),
	}
	const maxSize = 64 << 20
	info, statErr := os.Stat(path)
	if statErr != nil {
		if errors.Is(statErr, fs.ErrNotExist) {
			return empty, nil
		}
		return empty, fmt.Errorf("angela: polish state: stat: %w", statErr)
	}
	if info.Size() > maxSize {
		return empty, fmt.Errorf("angela: polish state: file too large (%d bytes): %w", info.Size(), ErrStateCorrupt)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return empty, nil
		}
		return empty, fmt.Errorf("angela: polish state: read: %w", err)
	}
	var state PolishState
	if err := json.Unmarshal(data, &state); err != nil {
		return empty, fmt.Errorf("angela: polish state: parse: %w: %w", ErrStateCorrupt, err)
	}
	if state.Version != PolishStateVersion {
		return empty, fmt.Errorf("angela: polish state: version %d != expected %d: %w", state.Version, PolishStateVersion, ErrStateCorrupt)
	}
	if state.Entries == nil {
		state.Entries = make(map[string]PolishStateEntry)
	}
	return &state, nil
}

// SavePolishState writes state to path atomically.
func SavePolishState(path string, state *PolishState) error {
	if state == nil {
		return fmt.Errorf("angela: polish state: nil state")
	}
	state.Version = PolishStateVersion
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("angela: polish state: mkdir: %w", err)
	}
	if err := fileutil.AtomicWriteJSON(path, state, 0o600); err != nil {
		return fmt.Errorf("angela: polish state: %w", err)
	}
	return nil
}

// HashSections computes a `"sha256:<hex>"` hash for each section in a
// document parsed via SplitSections. The preamble (index 0, empty
// heading) is stored under the key "" so it participates in change
// detection — if front matter changes the whole doc should re-polish.
// HashSections computes hashes for each section. Note: duplicate headings
// are not supported — if two sections share the same heading text, the
// last one wins (last-wins behavior). This is acceptable for MVP because
// well-formed lore documents should not have duplicate ## headings.
func HashSections(sections []Section) map[string]string {
	hashes := make(map[string]string, len(sections))
	keyCounts := make(map[string]int, len(sections))
	for _, s := range sections {
		// Body only (heading is the key). Trim trailing whitespace so
		// adding/removing a blank line at the end of a section doesn't
		// cause a false change.
		key := s.Heading
		keyCounts[key]++
		if keyCounts[key] > 1 {
			key = fmt.Sprintf("%s#%d", key, keyCounts[key])
		}
		body := trimTrailingWhitespace(s.Body)
		hashes[key] = ContentHash([]byte(body))
	}
	return hashes
}

// trimTrailingWhitespace removes trailing blank lines and spaces.
func trimTrailingWhitespace(s string) string {
	i := len(s)
	for i > 0 && (s[i-1] == ' ' || s[i-1] == '\n' || s[i-1] == '\r' || s[i-1] == '\t') {
		i--
	}
	return s[:i]
}

// DetectChangedSections compares current section hashes with stored
// ones and returns the indices of sections (in the `sections` slice)
// that are new or changed. The `minChangeLines` threshold skips
// sections with fewer than N non-blank lines in total (a size threshold,
// not a diff-size threshold — we only have stored hashes, not the
// previous body text, so an exact diff line count is not available).
//
// Returns (changedIndices, allUnchanged). If allUnchanged is true the
// orchestrator can skip the AI call entirely.
func DetectChangedSections(sections []Section, stored map[string]string, minChangeLines int) ([]int, bool) {
	current := HashSections(sections)
	var changed []int
	for i, s := range sections {
		key := s.Heading
		storedHash, found := stored[key]
		if !found || current[key] != storedHash {
			// New or hash differs. Check min-lines threshold.
			if minChangeLines > 0 && found {
				// Only apply threshold to sections that existed before;
				// new sections always need polish.
				lines := nonBlankLineCount(s.Body)
				if lines < minChangeLines {
					continue
				}
			}
			changed = append(changed, i)
		}
	}
	return changed, len(changed) == 0
}

// nonBlankLineCount counts non-empty, non-whitespace-only lines.
func nonBlankLineCount(s string) int {
	n := 0
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == '\n' {
			line := s[start:i]
			blank := true
			for j := 0; j < len(line); j++ {
				if line[j] != ' ' && line[j] != '\t' && line[j] != '\r' {
					blank = false
					break
				}
			}
			if !blank {
				n++
			}
			start = i + 1
		}
	}
	return n
}
