package cmd

import (
	"fmt"
	"os"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/git"
	"github.com/greycoderk/lore/internal/workflow"
	"github.com/spf13/cobra"
)

func newHookPostCommitCmd(_ *config.Config, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:           "_hook-post-commit",
		Short:         "Internal: invoked by the post-commit hook",
		Hidden:        true,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			workDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("cmd: hook-post-commit getwd: %w", err)
			}

			adapter := git.NewAdapter(workDir)
			if err := workflow.Dispatch(cmd.Context(), workDir, streams, adapter); err != nil {
				return fmt.Errorf("cmd: hook-post-commit: %w", err)
			}
			return nil
		},
	}
}
