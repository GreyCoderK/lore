// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/git"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/workflow"
	"github.com/spf13/cobra"
)

func newNewCmd(_ *config.Config, streams domain.IOStreams) *cobra.Command {
	var commitRef string

	cmd := &cobra.Command{
		Use:   i18n.T().Cmd.NewUse,
		Short: i18n.T().Cmd.NewShort,
		Example: `  lore new
  lore new feature "add auth middleware" "JWT for stateless auth"
  lore new --commit abc1234`,
		Args:              cobra.MaximumNArgs(3),
		SilenceUsage:      true, // N4 fix: prevent cobra from printing usage on RunE errors
		SilenceErrors:     true,
		ValidArgsFunction: newPositionalCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if Lore is initialized
			if err := requireLoreDir(streams); err != nil {
				return err
			}

			workDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("cmd: new: getwd: %w", err)
			}

			opts := workflow.ProactiveOpts{}

			if commitRef != "" {
				// Retroactive mode
				adapter := git.NewAdapter(workDir)

				exists, gitErr := adapter.CommitExists(commitRef)
				if gitErr != nil {
					return fmt.Errorf("cmd: new: %w", gitErr)
				}
				if !exists {
					// Actionable error for invalid/nonexistent commit
					_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.NewCommitNotFound+"\n", commitRef)
					return fmt.Errorf("cmd: new: commit '%s': %w", commitRef, domain.ErrNotFound)
				}

				// Log resolves short hash → full hash via CommitInfo.Hash
				commitInfo, gitErr := adapter.Log(commitRef)
				if gitErr != nil {
					return fmt.Errorf("cmd: new: %w", gitErr)
				}

				opts.Commit = commitInfo
			} else {
				// Manual mode — parse positional args
				if len(args) > 0 {
					opts.Type = args[0]
				}
				if len(args) > 1 {
					opts.What = args[1]
				}
				if len(args) > 2 {
					opts.Why = args[2]
				}
			}

			return workflow.HandleProactive(cmd.Context(), workDir, streams, opts)
		},
	}

	cmd.Flags().StringVar(&commitRef, "commit", "", i18n.T().Cmd.NewCommitFlagDesc)
	_ = cmd.RegisterFlagCompletionFunc("commit", gitRefFlagCompletion)
	return cmd
}

// newPositionalCompletion powers `lore new <TAB>`. The first positional
// arg is the document type (decision/feature/bugfix/...); the next two
// are free-form "what" and "why" strings that have no sensible
// completion, so we suppress file completion there.
func newPositionalCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return domain.DocTypeNames(), cobra.ShellCompDirectiveNoFileComp
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}
