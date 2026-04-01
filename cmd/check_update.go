// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/greycoderk/lore/internal/cli"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/upgrade"
	"github.com/greycoderk/lore/internal/version"
	"github.com/spf13/cobra"
)

func newCheckUpdateCmd(_ *config.Config, streams domain.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "check-update",
		Short:         i18n.T().Cmd.CheckUpdateShort,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCheckUpdate(cmd, streams)
		},
	}
	return cmd
}

func runCheckUpdate(cmd *cobra.Command, streams domain.IOStreams) error {
	t := i18n.T().Cmd

	currentVersion := version.Version
	if currentVersion != "dev" {
		currentVersion = "v" + currentVersion
	}

	fmt.Fprintf(streams.Err, "%s\n", t.UpgradeChecking)

	ctx := cmd.Context()
	client := upgrade.NewHTTPClient()

	newer, err := upgrade.ListNewerReleases(ctx, client, upgradeRepo, currentVersion)
	if err != nil {
		fmt.Fprintf(streams.Err, "%s\n", t.UpgradeNetworkErr)
		return &cli.ExitCodeError{Code: cli.ExitError}
	}

	if len(newer) == 0 {
		fmt.Fprintf(streams.Err, t.CheckUpdateUpToDate+"\n", currentVersion)
		return nil
	}

	// Header: current → latest
	fmt.Fprintf(streams.Err, t.CheckUpdateAvail+"\n", currentVersion, newer[0].TagName)
	fmt.Fprintln(streams.Err)

	// List all newer versions
	for _, r := range newer {
		if r.Prerelease {
			fmt.Fprintf(streams.Err, "  %-20s (%s)\n", r.TagName, t.CheckUpdatePreRelease)
		} else {
			fmt.Fprintf(streams.Err, "  %s\n", r.TagName)
		}
	}

	fmt.Fprintln(streams.Err)
	fmt.Fprintf(streams.Err, "%s\n", t.CheckUpdateHint)
	return nil
}
