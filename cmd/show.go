package cmd

import (
	"fmt"

	"github.com/museigen/lore/internal/config"
	"github.com/museigen/lore/internal/domain"
	"github.com/spf13/cobra"
)

func newShowCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Find a past decision",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("cmd: show not implemented")
		},
	}
}
