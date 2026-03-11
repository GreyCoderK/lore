package cmd

import (
	"fmt"

	"github.com/museigen/lore/internal/config"
	"github.com/museigen/lore/internal/domain"
	"github.com/spf13/cobra"
)

func newReleaseCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:   "release",
		Short: "Generate release notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("cmd: release not implemented")
		},
	}
}
