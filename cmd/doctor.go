package cmd

import (
	"fmt"

	"github.com/museigen/lore/internal/config"
	"github.com/museigen/lore/internal/domain"
	"github.com/spf13/cobra"
)

func newDoctorCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Fix documentation inconsistencies",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("cmd: doctor not implemented")
		},
	}
}
