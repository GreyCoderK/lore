// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/greycoderk/lore/internal/cli"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/ui"
	"github.com/greycoderk/lore/internal/upgrade"
	"github.com/greycoderk/lore/internal/version"
	"github.com/spf13/cobra"
)

const upgradeRepo = "GreyCoderK/lore"

func newUpgradeCmd(_ *config.Config, streams domain.IOStreams) *cobra.Command {
	var targetVersion string

	cmd := &cobra.Command{
		Use:           "upgrade",
		Short:         i18n.T().Cmd.UpgradeShort,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUpgrade(cmd, streams, targetVersion)
		},
	}

	cmd.Flags().StringVar(&targetVersion, "version", "", i18n.T().Cmd.UpgradeVersionFlag)

	return cmd
}

func runUpgrade(cmd *cobra.Command, streams domain.IOStreams, targetVersion string) error {
	t := i18n.T().Cmd

	// Guard: dev builds cannot self-update
	if version.Version == "dev" {
		fmt.Fprintf(streams.Err, "%s\n", t.UpgradeSkipDevBuild)
		return &cli.ExitCodeError{Code: cli.ExitError}
	}

	// Detect installation method
	execPath, err := os.Executable()
	if err != nil {
		return err
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return err
	}

	method, hint := upgrade.DetectInstallMethod(execPath)
	switch method {
	case upgrade.InstallHomebrew:
		fmt.Fprintf(streams.Err, t.UpgradeHomebrew+"\n", hint)
		return nil
	case upgrade.InstallGoInstall:
		fmt.Fprintf(streams.Err, t.UpgradeGoInstall+"\n", hint)
		return nil
	}

	// Check for release
	fmt.Fprintf(streams.Err, "%s\n", t.UpgradeChecking)
	ctx := cmd.Context()
	client := upgrade.NewHTTPClient()

	var release *upgrade.ReleaseInfo
	if targetVersion != "" {
		release, err = upgrade.FindRelease(ctx, client, upgradeRepo, targetVersion)
		if err != nil {
			fmt.Fprintf(streams.Err, "%s\n", t.UpgradeNetworkErr)
			return &cli.ExitCodeError{Code: cli.ExitError}
		}
		if release == nil {
			fmt.Fprintf(streams.Err, t.UpgradeVersionNotFnd+"\n", targetVersion)
			return &cli.ExitCodeError{Code: cli.ExitError}
		}
	} else {
		release, err = upgrade.CheckLatestRelease(ctx, client, upgradeRepo)
		if err != nil {
			fmt.Fprintf(streams.Err, "%s\n", t.UpgradeNetworkErr)
			return &cli.ExitCodeError{Code: cli.ExitError}
		}
		if release == nil {
			fmt.Fprintf(streams.Err, "%s\n", t.UpgradeNoRelease)
			return &cli.ExitCodeError{Code: cli.ExitError}
		}
	}

	// Compare versions
	currentVersion := "v" + version.Version
	if targetVersion == "" && upgrade.CompareVersions(currentVersion, release.TagName) >= 0 {
		fmt.Fprintf(streams.Err, t.UpgradeAlreadyLatest+"\n", currentVersion)
		return nil
	}

	fmt.Fprintf(streams.Err, t.UpgradeNewVersion+"\n", currentVersion, release.TagName)

	// Find the right asset for this platform
	archiveName := upgrade.AssetName()
	var assetURL, checksumsURL string
	for _, a := range release.Assets {
		if a.Name == archiveName {
			assetURL = a.BrowserDownloadURL
		}
		if a.Name == "checksums.txt" {
			checksumsURL = a.BrowserDownloadURL
		}
	}
	if assetURL == "" {
		fmt.Fprintf(streams.Err, t.UpgradeAPIErr+"\n", "no matching asset: "+archiveName)
		return &cli.ExitCodeError{Code: cli.ExitError}
	}

	// Check write permissions before downloading
	dir := filepath.Dir(execPath)
	tmpCheck, tmpErr := os.CreateTemp(dir, ".lore-upgrade-check-*")
	if tmpErr != nil {
		fmt.Fprintf(streams.Err, t.UpgradePermissionErr+"\n", execPath)
		return &cli.ExitCodeError{Code: cli.ExitError}
	}
	_ = tmpCheck.Close()
	_ = os.Remove(tmpCheck.Name())

	// Download to temp directory
	tmpDir, err := os.MkdirTemp("", "lore-upgrade-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	fmt.Fprintf(streams.Err, t.UpgradeDownloading+"\n", release.TagName)
	archivePath, err := upgrade.DownloadAsset(ctx, client, assetURL, tmpDir)
	if err != nil {
		fmt.Fprintf(streams.Err, "%s\n", t.UpgradeNetworkErr)
		return &cli.ExitCodeError{Code: cli.ExitError}
	}

	// Verify checksum if available
	if checksumsURL != "" {
		fmt.Fprintf(streams.Err, "%s\n", t.UpgradeVerifying)
		expected, err := upgrade.DownloadChecksum(ctx, client, checksumsURL, archiveName)
		if err != nil {
			fmt.Fprintf(streams.Err, "%s\n", t.UpgradeChecksumFail)
			return &cli.ExitCodeError{Code: cli.ExitError}
		}
		if err := upgrade.VerifySHA256(archivePath, expected); err != nil {
			fmt.Fprintf(streams.Err, "%s\n", t.UpgradeChecksumFail)
			return &cli.ExitCodeError{Code: cli.ExitError}
		}
	}

	// Extract binary
	newBinary, err := upgrade.ExtractBinary(archivePath, tmpDir)
	if err != nil {
		fmt.Fprintf(streams.Err, t.UpgradeAPIErr+"\n", err)
		return &cli.ExitCodeError{Code: cli.ExitError}
	}

	// Replace binary
	fmt.Fprintf(streams.Err, "%s\n", t.UpgradeInstalling)
	if err := upgrade.ReplaceBinary(execPath, newBinary); err != nil {
		fmt.Fprintf(streams.Err, t.UpgradePermissionErr+"\n", execPath)
		return &cli.ExitCodeError{Code: cli.ExitError}
	}

	ui.Verb(streams, "Upgraded", fmt.Sprintf(t.UpgradeSuccess, release.TagName))
	return nil
}
