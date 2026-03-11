package cmd

import (
	"fmt"

	"github.com/museigen/lore/internal/config"
	"github.com/museigen/lore/internal/domain"
	"github.com/spf13/cobra"
)

func newNewCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:   "new",
		Short: "Document a decision right now",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("cmd: new not implemented")
		},
	}
}
