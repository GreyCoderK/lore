package cmd

import (
	"fmt"

	"github.com/museigen/lore/internal/config"
	"github.com/museigen/lore/internal/domain"
	"github.com/spf13/cobra"
)

func newDemoCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:   "demo",
		Short: "See Lore in action with a guided walkthrough",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("cmd: demo not implemented")
		},
	}
}
