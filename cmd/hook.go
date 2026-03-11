package cmd

import (
	"fmt"

	"github.com/museigen/lore/internal/config"
	"github.com/museigen/lore/internal/domain"
	"github.com/spf13/cobra"
)

func newHookCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:   "hook",
		Short: "Manage the post-commit hook",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("cmd: hook not implemented")
		},
	}
}
