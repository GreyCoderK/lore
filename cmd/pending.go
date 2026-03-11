package cmd

import (
	"fmt"

	"github.com/museigen/lore/internal/config"
	"github.com/museigen/lore/internal/domain"
	"github.com/spf13/cobra"
)

func newPendingCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:   "pending",
		Short: "Resume skipped documentation",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("cmd: pending not implemented")
		},
	}
}
