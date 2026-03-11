package cmd

import (
	"fmt"

	"github.com/museigen/lore/internal/config"
	"github.com/museigen/lore/internal/domain"
	"github.com/spf13/cobra"
)

func newListCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "See all documented decisions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("cmd: list not implemented")
		},
	}
}
