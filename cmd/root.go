package cmd

import (
	"fmt"
	"os"

	"github.com/museigen/lore/internal/config"
	"github.com/museigen/lore/internal/domain"
	"github.com/museigen/lore/internal/ui"
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
	streams := domain.IOStreams{
		Out: os.Stdout,
		Err: os.Stderr,
		In:  os.Stdin,
	}

	ui.SetColorEnabled(ui.ColorEnabled(streams))

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(streams.Err, "Error: %v\n  Run: lore doctor\n", err)
		os.Exit(1)
	}

	cmd := newRootCmd(cfg, streams)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
