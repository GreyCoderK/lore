package cmd

import (
	"fmt"

	"github.com/museigen/lore/internal/config"
	"github.com/museigen/lore/internal/domain"
	"github.com/spf13/cobra"
)

func newNoteCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:   "note",
		Short: "Manage reusable notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("cmd: note not implemented")
		},
	}
}
