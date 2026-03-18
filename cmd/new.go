package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/git"
	"github.com/greycoderk/lore/internal/ui"
	"github.com/greycoderk/lore/internal/workflow"
	"github.com/spf13/cobra"
)

func newNewCmd(_ *config.Config, streams domain.IOStreams) *cobra.Command {
	var commitRef string

	cmd := &cobra.Command{
		Use:   "new [type] [what] [why]",
		Short: "Document a decision right now",
		Example: `  lore new
  lore new feature "add auth middleware" "JWT for stateless auth"
  lore new --commit abc1234`,
		Args:         cobra.MaximumNArgs(3),
		SilenceUsage:  true, // N4 fix: prevent cobra from printing usage on RunE errors
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// AC-4: Check if Lore is initialized
			loreDir := filepath.Join(".", ".lore")
			if _, err := os.Stat(loreDir); os.IsNotExist(err) {
				ui.ActionableError(streams, "Lore not initialized.", "lore init")
				return fmt.Errorf("cmd: new: %w", domain.ErrNotInitialized)
			} else if err != nil {
				return fmt.Errorf("cmd: new: %w", err)
			}

			workDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("cmd: new: getwd: %w", err)
			}

			opts := workflow.ProactiveOpts{}

			if commitRef != "" {
				// Retroactive mode (AC-1, AC-3, AC-5)
				adapter := git.NewAdapter(workDir)

				exists, gitErr := adapter.CommitExists(commitRef)
				if gitErr != nil {
					return fmt.Errorf("cmd: new: %w", gitErr)
				}
				if !exists {
					// AC-3: actionable error for invalid/nonexistent commit
					fmt.Fprintf(streams.Err, "Error: Commit '%s' not found.\n", commitRef)
					return fmt.Errorf("cmd: new: commit '%s': %w", commitRef, domain.ErrNotFound)
				}

				// AC-5: Log resolves short hash → full hash via CommitInfo.Hash
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

	cmd.Flags().StringVar(&commitRef, "commit", "", "Document a past commit retroactively")
	return cmd
}
