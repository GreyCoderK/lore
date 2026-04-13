// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/fileutil"
	"github.com/spf13/cobra"
)

// newAngelaReviewResolveCmd wires `lore angela review resolve <hash>`.
//
// A user explicitly marks a review finding as fixed.
// Subsequent runs hide it from the active list; if the AI returns it
// again on a later run it surfaces as REGRESSED so the user notices
// the regression.
//
// Hash matching accepts the full 16-char hex value or any prefix that
// is unambiguous in the current state. Six characters is the spelled-
// out target in the AC but the resolver works for any prefix length
// the user types.
func newAngelaReviewResolveCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:          "resolve <hash>",
		Short:        "Mark a review finding as resolved",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			statePath, err := reviewStatePath(cfg)
			if err != nil {
				return err
			}
			// Serialize with other review invocations.
			lock, lockErr := fileutil.NewFileLock(statePath)
			if lockErr != nil {
				return fmt.Errorf("angela: review resolve: state lock: %w", lockErr)
			}
			defer lock.Unlock()
			// Honor the (usable state, error) contract of LoadReviewState.
			// On a corrupt file, quarantine the broken file aside and
			// proceed with a fresh state; the user sees an explicit notice.
			state, loadErr := angela.LoadReviewState(statePath)
			if loadErr != nil {
				if errors.Is(loadErr, angela.ErrStateCorrupt) {
					if quarPath, qerr := angela.QuarantineCorruptState(statePath); qerr == nil {
						fmt.Fprintf(streams.Err, "review: state file was corrupt; quarantined at %s\n", quarPath)
					} else {
						return fmt.Errorf("angela: review resolve: corrupt state and cannot quarantine: %w", qerr)
					}
				} else {
					fmt.Fprintf(streams.Err, "review resolve: %v (using fresh state)\n", loadErr)
				}
			}
			full, err := angela.ResolveByPrefix(state, args[0])
			if err != nil {
				return fmt.Errorf("angela: review resolve: %w", err)
			}
			// Sanitize $USER before embedding it in the state file to prevent
			// ANSI escape codes or newline-injection payloads.
			by := sanitizeResolvedBy(os.Getenv("USER"))
			if err := angela.MarkResolved(state, full, by, time.Now().UTC()); err != nil {
				return fmt.Errorf("angela: review resolve: %w", err)
			}
			if err := angela.SaveReviewState(statePath, state); err != nil {
				return fmt.Errorf("angela: review resolve: save: %w", err)
			}
			fmt.Fprintf(streams.Err, "✓ resolved %s — %s\n", full[:6], state.Findings[full].Finding.Title)
			return nil
		},
	}
}

// sanitizeResolvedBy returns a safe form of the $USER environment
// value for embedding in the state file. Strips ASCII control chars
// (ANSI escapes, newlines, NUL) and caps the length at 64 runes so a
// pathological env value cannot balloon the JSON. Empty input (or
// input that collapses to empty after sanitization) maps to "unknown"
// so log rendering always has a non-empty author column.
//
func sanitizeResolvedBy(s string) string {
	const maxLen = 64
	var b []rune
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			continue
		}
		b = append(b, r)
		if len(b) >= maxLen {
			break
		}
	}
	out := strings.TrimSpace(string(b))
	if out == "" {
		return "unknown"
	}
	return out
}

// reviewStatePath resolves the review state file path using the same
// rules as the runner: ResolveStateDir + cfg.Angela.Review.Differential.StateFile,
// with a "review-state.json" fallback. Shared by the three lifecycle
// subcommands so a misconfiguration only needs fixing in one place.
func reviewStatePath(cfg *config.Config) (string, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("angela: review state: cwd: %w", err)
	}
	stateDir := config.ResolveStateDir(workDir, cfg, cfg.DetectedMode)
	stateFile := cfg.Angela.Review.Differential.StateFile
	if stateFile == "" {
		stateFile = "review-state.json"
	}
	// Reject state_file that would escape stateDir.
	if err := angela.AssertContainedRelPath(stateFile); err != nil {
		return "", fmt.Errorf("angela: review state: state_file: %w", err)
	}
	return filepath.Join(stateDir, stateFile), nil
}
