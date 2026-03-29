// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/greycoderk/lore/internal/ai"
	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/credential"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/service"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/spf13/cobra"
)

func newAngelaPolishCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	var flagDryRun bool
	var flagYes bool

	cmd := &cobra.Command{
		Use:           "polish <filename>",
		Short:         i18n.T().Cmd.AngelaPolishShort,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			filename := args[0]

			// AC-9: Check .lore/ exists
			if err := requireLoreDir(streams); err != nil {
				return err
			}
			docsDir := filepath.Join(domain.LoreDir, domain.DocsDir)

			// AC-8: Validate filename and check exists
			if err := storage.ValidateFilename(filename); err != nil {
				return fmt.Errorf("angela: polish: %w", err)
			}
			docPath := filepath.Join(docsDir, filename)
			if _, err := os.Stat(docPath); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf(i18n.T().Cmd.AngelaPolishNotFound, filename)
				}
				return fmt.Errorf("angela: polish: %w", err)
			}

			// AC-4: Instantiate provider — nil = no provider
			store := credential.NewStore()
			provider, err := ai.NewProvider(cfg, store, streams.Err)
			if err != nil {
				return err
			}
			if provider == nil {
				return fmt.Errorf("%s", i18n.T().Cmd.AngelaPolishNoProvider)
			}

			// Orchestrate polish via service layer
			result, err := service.PolishDocument(cmd.Context(), provider, cfg, docsDir, filename)
			if err != nil {
				return err
			}

			originalContent := result.Original
			meta := result.Meta
			hunks := result.Diff

			// AC-10: No changes
			if len(hunks) == 0 {
				_, _ = fmt.Fprintf(streams.Err, "%s\n", i18n.T().Cmd.AngelaPolishNoChanges)
				return nil
			}

			// AC-2: Interactive diff / AC-7: reject all
			accepted, err := angela.InteractiveDiff(hunks, streams, angela.DiffOptions{DryRun: flagDryRun, YesAll: flagYes})
			if err != nil {
				return fmt.Errorf("angela: polish: %w", err)
			}

			if flagDryRun {
				return nil
			}

			// Check if any accepted
			anyAccepted := false
			for _, a := range accepted {
				if a {
					anyAccepted = true
					break
				}
			}

			// AC-7: All rejected
			if !anyAccepted {
				_, _ = fmt.Fprintf(streams.Err, "%s\n", i18n.T().Cmd.AngelaPolishNoneApplied)
				return nil
			}

			// TOCTOU guard: check file hasn't changed during interactive review
			currentRaw, err := os.ReadFile(docPath)
			if err != nil {
				return fmt.Errorf("angela: polish: re-read: %w", err)
			}
			if !bytes.Equal(currentRaw, []byte(originalContent)) {
				return fmt.Errorf("angela: polish: %s", i18n.T().Cmd.AngelaPolishModified)
			}

			// AC-3: Apply changes
			applied := angela.ApplyDiff(originalContent, hunks, accepted)

			// Validate and update angela_mode in front matter
			resultMeta, resultBody, err := storage.Unmarshal([]byte(applied))
			if err != nil {
				// If parsing fails, use the result as-is with original meta
				resultMeta = meta
				resultBody = applied
			}
			resultMeta.AngelaMode = "polish"

			// Validate the resulting frontmatter before writing to disk
			if valErr := storage.ValidateMeta(resultMeta); valErr != nil {
				return fmt.Errorf("angela: polish: AI response produced invalid frontmatter: %w", valErr)
			}

			// Write atomically
			marshaled, err := storage.Marshal(resultMeta, resultBody)
			if err != nil {
				return fmt.Errorf("angela: polish: marshal: %w", err)
			}
			if err := storage.AtomicWrite(docPath, marshaled); err != nil {
				return fmt.Errorf("angela: polish: write: %w", err)
			}

			// Regenerate index
			if err := storage.RegenerateIndex(docsDir); err != nil {
				_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.AngelaPolishIndexWarn+"\n", err)
			}

			_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.AngelaPolishVerb+"\n", filename)
			return nil
		},
	}

	cmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Show diff without applying changes")
	cmd.Flags().BoolVar(&flagYes, "yes", false, "Accept all changes without confirmation")

	return cmd
}
