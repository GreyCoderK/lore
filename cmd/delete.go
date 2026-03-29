// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/greycoderk/lore/internal/ui"
	"github.com/spf13/cobra"
)

// deleteIsTTY is the TTY detection function used by the delete command.
// Tests override this to simulate interactive terminals.
var deleteIsTTY = ui.IsTerminal

func newDeleteCmd(_ *config.Config, streams domain.IOStreams) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   i18n.T().Cmd.DeleteUse,
		Short: i18n.T().Cmd.DeleteShort,
		Example: `  lore delete decision-auth-strategy-2026-03-07.md
  lore delete decision-auth-strategy-2026-03-07.md --force`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage:   true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireLoreDir(streams); err != nil {
				return err
			}

			filename := args[0]
			if err := storage.ValidateFilename(filename); err != nil {
				_, _ = fmt.Fprintf(streams.Err, "%s %s\n", ui.Error("Error:"), fmt.Sprintf(i18n.T().Cmd.DeleteInvalidName, filename))
				return fmt.Errorf("cmd: delete: %w", err)
			}
			docsDir := filepath.Join(".lore", "docs")
			docPath := filepath.Join(docsDir, filename)

			// AC-5: read document — produces friendly error if missing
			data, err := os.ReadFile(docPath)
			if os.IsNotExist(err) {
				_, _ = fmt.Fprintf(streams.Err, "%s %s\n", ui.Error("Error:"), fmt.Sprintf(i18n.T().Cmd.DeleteNotFound, filename))
				return fmt.Errorf("cmd: delete: %s: %w", filename, domain.ErrNotFound)
			} else if err != nil {
				return fmt.Errorf("cmd: delete: read %s: %w", filename, err)
			}
			meta, _, err := storage.Unmarshal(data)
			if err != nil {
				return fmt.Errorf("cmd: delete: parse %s: %w", filename, err)
			}

			// AC-4: warn about incoming references before confirmation
			refs, refErr := storage.FindReferencingDocs(docsDir, filename)
			if refErr != nil {
				_, _ = fmt.Fprintf(streams.Err, "Warning: %v\n", refErr)
			}
			if len(refs) > 0 {
				_, _ = fmt.Fprintf(streams.Err, "%s %s\n", ui.Warning("Warning:"), i18n.T().Cmd.DeleteRefWarning)
				for _, ref := range refs {
					_, _ = fmt.Fprintf(streams.Err, "  - %s\n", ref)
				}
				_, _ = fmt.Fprintf(streams.Err, "%s\n", i18n.T().Cmd.DeleteRefNotUpdated)
			}

			// AC-3: demo documents skip confirmation
			needConfirm := meta.Status != "demo" && !force

			if needConfirm {
				// AC-8: non-TTY without --force → refuse
				if !deleteIsTTY(streams) {
					_, _ = fmt.Fprintf(streams.Err, "%s %s\n", ui.Error("Error:"), i18n.T().Cmd.DeleteForceRequired)
					return fmt.Errorf("cmd: delete: %w", domain.ErrNotInteractive)
				}

				// AC-2: interactive confirmation
				if cmd.Context().Err() != nil {
					return cmd.Context().Err()
				}
				_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.DeleteConfirmPrompt, filename)
				// Read stdin byte-by-byte instead of bufio.NewReader to avoid
				// buffering ahead — same pattern as proactive.go AC-4 confirmation.
				var answerBuf []byte
				b := make([]byte, 1)
				for {
					n, readErr := streams.In.Read(b)
					if n > 0 {
						if b[0] == '\n' {
							break
						}
						answerBuf = append(answerBuf, b[0])
					}
					if readErr != nil {
						break
					}
				}
				answer := strings.TrimSpace(strings.ToLower(string(answerBuf)))
				if answer != "y" && answer != "yes" {
					_, _ = fmt.Fprintf(streams.Err, "%s\n", i18n.T().Cmd.DeleteCancelled)
					return nil
				}
			}

			// AC-1: delete the document
			if err := storage.DeleteDoc(docsDir, filename); err != nil {
				return fmt.Errorf("cmd: delete: %w", err)
			}

			// AC-1: success message with red verb
			ui.VerbDelete(streams, filename)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation (for scripts/CI)")
	return cmd
}
