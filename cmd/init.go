package cmd

import (
	"fmt"

	"github.com/museigen/lore/internal/config"
	"github.com/museigen/lore/internal/domain"
	"github.com/spf13/cobra"
)

func newInitCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Set up Lore in this repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("cmd: init not implemented")
		},
	}
}
