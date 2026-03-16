package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/ui"
	"github.com/greycoderk/lore/internal/workflow"
	"github.com/spf13/cobra"
)

func newNewCmd(_ *config.Config, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:   "new [type] [what] [why]",
		Short: "Document a decision right now",
		Example: `  lore new
  lore new feature "add auth middleware" "JWT for stateless auth"`,
		Args:         cobra.MaximumNArgs(3),
		SilenceUsage: true, // N4 fix: prevent cobra from printing usage on RunE errors
		RunE: func(cmd *cobra.Command, args []string) error {
			// AC-4: Check if Lore is initialized
			loreDir := filepath.Join(".", ".lore")
			if _, err := os.Stat(loreDir); os.IsNotExist(err) {
				ui.ActionableError(streams, "Lore not initialized.", "lore init")
				return fmt.Errorf("cmd: new: %w", domain.ErrNotInitialized)
			}

			workDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("cmd: new: getwd: %w", err)
			}

			opts := workflow.ProactiveOpts{}
			if len(args) > 0 {
				opts.Type = args[0]
			}
			if len(args) > 1 {
				opts.What = args[1]
			}
			if len(args) > 2 {
				opts.Why = args[2]
			}

			return workflow.HandleProactive(cmd.Context(), workDir, streams, opts)
		},
	}
}
