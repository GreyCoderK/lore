package cmd

import (
	"fmt"

	"github.com/museigen/lore/internal/config"
	"github.com/museigen/lore/internal/domain"
	"github.com/spf13/cobra"
)

func newStatusCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check your documentation health",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("cmd: status not implemented")
		},
	}
}
