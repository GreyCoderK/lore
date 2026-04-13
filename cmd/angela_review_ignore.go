// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/fileutil"
	"github.com/spf13/cobra"
)

// newAngelaReviewIgnoreCmd wires `lore angela review ignore <hash> --reason "..."`.
//
// Ignore is semantically distinct from resolve — it
// means "the AI keeps reporting this and we know it's a false positive
// or an intentional exception". The required --reason flag prevents a
// silent ignore that would later confuse the team.
//
// Like resolve, an ignored finding that surfaces again in a future run
// is flagged REGRESSED — the AI is now finding something the user
// thought wasn't worth tracking, which is itself a signal worth
// surfacing.
func newAngelaReviewIgnoreCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	var flagReason string
	cmd := &cobra.Command{
		Use:          "ignore <hash>",
		Short:        "Mark a review finding as a known false positive with a required reason",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(flagReason) == "" {
				return fmt.Errorf("angela: review ignore: --reason is required")
			}
			statePath, err := reviewStatePath(cfg)
			if err != nil {
				return err
			}
			// Serialize with other review invocations.
			lock, lockErr := fileutil.NewFileLock(statePath)
			if lockErr != nil {
				return fmt.Errorf("angela: review ignore: state lock: %w", lockErr)
			}
			defer lock.Unlock()
			// Honor the (usable state, error) contract — quarantine corrupt
			// files instead of hard-failing.
			state, loadErr := angela.LoadReviewState(statePath)
			if loadErr != nil {
				if errors.Is(loadErr, angela.ErrStateCorrupt) {
					if quarPath, qerr := angela.QuarantineCorruptState(statePath); qerr == nil {
						fmt.Fprintf(streams.Err, "review: state file was corrupt; quarantined at %s\n", quarPath)
					} else {
						return fmt.Errorf("angela: review ignore: corrupt state and cannot quarantine: %w", qerr)
					}
				} else {
					fmt.Fprintf(streams.Err, "review ignore: %v (using fresh state)\n", loadErr)
				}
			}
			full, err := angela.ResolveByPrefix(state, args[0])
			if err != nil {
				return fmt.Errorf("angela: review ignore: %w", err)
			}
			if err := angela.MarkIgnored(state, full, flagReason, time.Now().UTC()); err != nil {
				return fmt.Errorf("angela: review ignore: %w", err)
			}
			if err := angela.SaveReviewState(statePath, state); err != nil {
				return fmt.Errorf("angela: review ignore: save: %w", err)
			}
			fmt.Fprintf(streams.Err, "✓ ignored %s — %s\n  reason: %s\n",
				full[:6], state.Findings[full].Finding.Title, flagReason)
			return nil
		},
	}
	cmd.Flags().StringVar(&flagReason, "reason", "", "Required: explain why this finding is being ignored")
	return cmd
}
