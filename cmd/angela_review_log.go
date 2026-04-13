// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/fileutil"
	"github.com/spf13/cobra"
)

// newAngelaReviewLogCmd wires `lore angela review log`.
//
// Print every entry in the review state file in
// human-readable form, sorted by LastSeen descending so the most
// recent activity is at the top. Useful for spotting stale findings
// and for grepping a specific hash before calling resolve / ignore.
//
// Honors --format=json so the same data is available to scripts.
// JSON output is the raw []StatefulFinding slice (already sorted)
// rather than the full state file, because that's the most useful
// shape for downstream consumers.
func newAngelaReviewLogCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	var flagFormat string
	cmd := &cobra.Command{
		Use:          "log",
		Short:        "List every tracked review finding (newest first)",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			statePath, err := reviewStatePath(cfg)
			if err != nil {
				return err
			}
			// Serialize with writers so we never read a half-rotated state file.
			lock, lockErr := fileutil.NewFileLock(statePath)
			if lockErr != nil {
				return fmt.Errorf("angela: review log: state lock: %w", lockErr)
			}
			defer lock.Unlock()
			// Honor the (usable state, error) contract. For `log`, a corrupt
			// file means an empty listing instead of a hard failure.
			state, loadErr := angela.LoadReviewState(statePath)
			if loadErr != nil {
				if errors.Is(loadErr, angela.ErrStateCorrupt) {
					if quarPath, qerr := angela.QuarantineCorruptState(statePath); qerr == nil {
						fmt.Fprintf(streams.Err, "review: state file was corrupt; quarantined at %s\n", quarPath)
					} else {
						return fmt.Errorf("angela: review log: corrupt state and cannot quarantine: %w", qerr)
					}
				} else {
					fmt.Fprintf(streams.Err, "review log: %v (using fresh state)\n", loadErr)
				}
			}
			entries := angela.LogEntries(state)

			switch flagFormat {
			case "json":
				enc := json.NewEncoder(streams.Out)
				enc.SetIndent("", "  ")
				if entries == nil {
					entries = []angela.StatefulFinding{}
				}
				return enc.Encode(entries)
			default:
				if len(entries) == 0 {
					fmt.Fprintln(streams.Err, "(no findings tracked yet — run `lore angela review` first)")
					return nil
				}
				fmt.Fprintf(streams.Out, "%-8s %-10s %-19s %-19s %s\n",
					"hash", "status", "first_seen", "last_seen", "title")
				for _, e := range entries {
					hash := e.Finding.Hash
					if hash == "" {
						hash = "??????"
					}
					if len(hash) > 6 {
						hash = hash[:6]
					}
					fmt.Fprintf(streams.Out, "%-8s %-10s %-19s %-19s %s\n",
						hash,
						e.Status,
						e.FirstSeen.Local().Format("2006-01-02 15:04:05"),
						e.LastSeen.Local().Format("2006-01-02 15:04:05"),
						e.Finding.Title,
					)
				}
				return nil
			}
		},
	}
	cmd.Flags().StringVar(&flagFormat, "format", "", "Output format: human (default) | json")
	return cmd
}
