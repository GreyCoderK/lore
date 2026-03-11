package cmd

import (
	"fmt"

	"github.com/museigen/lore/internal/config"
	"github.com/museigen/lore/internal/domain"
	"github.com/spf13/cobra"
)

func newAngelaCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:   "angela",
		Short: "Get AI writing assistance",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("cmd: angela not implemented")
		},
	}
}
