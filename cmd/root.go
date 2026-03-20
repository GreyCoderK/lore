// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"

	"github.com/greycoderk/lore/internal/cli"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/ui"
	"github.com/spf13/cobra"
)

func newRootCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lore",
		Short: "Your code knows what. Lore knows why.",
		Long:  "Your code knows what. Lore knows why.",
		PersistentPreRunE: func(c *cobra.Command, args []string) error {
			// Skip config loading for commands that must work without a valid config
			name := c.Name()
			if name == "init" || name == "doctor" {
				return nil
			}

			loaded, err := config.LoadFromDirWithFlags(".", c)
			if err != nil {
				return fmt.Errorf("%v\n  Run: lore doctor", err)
			}
			*cfg = *loaded

			// --no-color flag overrides terminal detection
			noColor, _ := c.Flags().GetBool("no-color")
			if noColor {
				ui.SetColorEnabled(false)
			}

			return nil
		},
	}

	config.RegisterFlags(cmd)

	cmd.AddCommand(
		newInitCmd(cfg, streams),
		newHookCmd(cfg, streams),
		newHookPostCommitCmd(cfg, streams),
		newNewCmd(cfg, streams),
		newShowCmd(cfg, streams),
		newListCmd(cfg, streams),
		newStatusCmd(cfg, streams),
		newPendingCmd(cfg, streams),
		newAngelaCmd(cfg, streams),
		newConfigCmd(cfg, streams),
		newDoctorCmd(cfg, streams),
		newReleaseCmd(cfg, streams),
		newDeleteCmd(cfg, streams),
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

	cfg := &config.Config{}
	cmd := newRootCmd(cfg, streams)

	if err := cmd.Execute(); err != nil {
		if code := cli.ExitCodeFrom(err); code >= 0 {
			os.Exit(code)
		}
		os.Exit(cli.ExitError)
	}
}
