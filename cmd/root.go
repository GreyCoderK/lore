package cmd

import (
	"os"

	"github.com/museigen/lore/internal/config"
	"github.com/museigen/lore/internal/domain"
	"github.com/spf13/cobra"
)

func newRootCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lore",
		Short: "Your code knows what. Lore knows why.",
		Long:  "Your code knows what. Lore knows why.",
	}

	cmd.AddCommand(
		newInitCmd(cfg, streams),
		newHookCmd(cfg, streams),
		newNewCmd(cfg, streams),
		newShowCmd(cfg, streams),
		newListCmd(cfg, streams),
		newStatusCmd(cfg, streams),
		newPendingCmd(cfg, streams),
		newAngelaCmd(cfg, streams),
		newDoctorCmd(cfg, streams),
		newReleaseCmd(cfg, streams),
		newDemoCmd(cfg, streams),
		newNoteCmd(cfg, streams),
	)

	return cmd
}

func Execute() {
	cfg, err := config.Load()
	if err != nil {
		os.Exit(1)
	}

	streams := domain.IOStreams{
		Out: os.Stdout,
		Err: os.Stderr,
		In:  os.Stdin,
	}

	cmd := newRootCmd(cfg, streams)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
