// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/git"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/greycoderk/lore/internal/ui"
	"github.com/spf13/cobra"
)

func newReleaseCmd(_ *config.Config, streams domain.IOStreams) *cobra.Command {
	var (
		fromFlag    string
		toFlag      string
		versionFlag string
		quietFlag   bool
	)

	cmd := &cobra.Command{
		Use:           "release",
		Short:         i18n.T().Cmd.ReleaseShort,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			workDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("cmd: release: getwd: %w", err)
			}
			adapter := git.NewAdapter(workDir)
			return runRelease(streams, adapter, fromFlag, toFlag, versionFlag, quietFlag)
		},
	}

	cmd.Flags().StringVar(&fromFlag, "from", "", "Start of commit range (tag or SHA)")
	cmd.Flags().StringVar(&toFlag, "to", "", "End of commit range (tag or SHA, default: HEAD)")
	cmd.Flags().StringVar(&versionFlag, "version", "", "Version label for release notes")
	cmd.Flags().BoolVar(&quietFlag, "quiet", false, "Output only the release notes file path")

	return cmd
}

func runRelease(streams domain.IOStreams, adapter domain.GitAdapter, fromFlag, toFlag, versionFlag string, quiet bool) error {
	// Check if Lore is initialized
	if _, err := os.Stat(".lore"); os.IsNotExist(err) {
		if !quiet {
			ui.ActionableError(streams, i18n.T().Cmd.ReleaseNotInitMsg, i18n.T().Cmd.ReleaseNotInitHint)
		}
		return fmt.Errorf("cmd: release: %w", domain.ErrNotInitialized)
	}

	loreDir := ".lore"
	docsDir := filepath.Join(loreDir, "docs")

	// Resolve latest tag once to avoid inconsistent double-calls (M6)
	var cachedTag string
	latestTag := func() (string, error) {
		if cachedTag != "" {
			return cachedTag, nil
		}
		tag, err := adapter.LatestTag()
		if err != nil {
			return "", err
		}
		cachedTag = tag
		return tag, nil
	}

	// Determine range
	from := fromFlag
	to := toFlag

	if from == "" {
		tag, tagErr := latestTag()
		if tagErr != nil {
			if !quiet {
				_, _ = fmt.Fprintf(streams.Err, "%s %s\n", ui.Error("Error:"), i18n.T().Cmd.ReleaseNoTagsError)
				_, _ = fmt.Fprintf(streams.Err, "  %s\n", i18n.T().Cmd.ReleaseNoTagsHint)
			}
			return fmt.Errorf("cmd: release: %w", tagErr)
		}
		from = tag
	}
	if to == "" {
		to = "HEAD"
	}

	// Determine version
	version := versionFlag
	if version == "" {
		if toFlag != "" && toFlag != "HEAD" {
			version = toFlag
		} else {
			tag, tagErr := latestTag()
			if tagErr != nil {
				return fmt.Errorf("cmd: release: %w", tagErr)
			}
			version = tag
		}
	}

	// Get commits in range
	commits, err := adapter.CommitRange(from, to)
	if err != nil {
		return fmt.Errorf("cmd: release: %w", err)
	}

	// Collect documents
	var spin *ui.Spinner
	if !quiet {
		spin = ui.StartSpinner(streams, i18n.T().Cmd.ReleaseCollecting)
	}
	docs, parseErr, err := storage.CollectReleaseDocuments(docsDir, commits)
	if spin != nil {
		spin.Stop()
	}
	if err != nil {
		return fmt.Errorf("cmd: release: %w", err)
	}
	if parseErr != nil && !quiet {
		_, _ = fmt.Fprintf(streams.Err, "%s\n", ui.Warning(fmt.Sprintf(i18n.T().Cmd.ReleaseParseWarning, parseErr.Error())))
	}

	// No documents
	if len(docs) == 0 {
		if !quiet {
			_, _ = fmt.Fprintf(streams.Err, "%s\n", i18n.T().Cmd.ReleaseNoChanges)
			_, _ = fmt.Fprintf(streams.Err, "  %s\n", i18n.T().Cmd.ReleaseNoChangesHint)
		}
		return nil
	}

	date := time.Now().Format("2006-01-02")

	// Generate release notes file
	var spinGen *ui.Spinner
	if !quiet {
		spinGen = ui.StartSpinner(streams, i18n.T().Cmd.ReleaseGenerating)
	}
	filename, err := storage.GenerateReleaseNotes(version, date, docs, docsDir)
	if spinGen != nil {
		spinGen.Stop()
	}
	if err != nil {
		return fmt.Errorf("cmd: release: %w", err)
	}

	// Collect document filenames for releases.json
	docFilenames := make([]string, len(docs))
	for i, d := range docs {
		docFilenames[i] = d.Filename
	}

	// Update releases.json
	if err := storage.UpdateReleasesJSON(loreDir, version, date, docFilenames); err != nil {
		return fmt.Errorf("cmd: release: %w", err)
	}

	// Update CHANGELOG.md
	headerMissing, err := storage.UpdateChangelog(".", version, date, docs)
	if err != nil {
		return fmt.Errorf("cmd: release: %w", err)
	}
	if headerMissing && !quiet {
		_, _ = fmt.Fprintf(streams.Err, "%s\n", ui.Warning(i18n.T().Cmd.ReleaseChangelogHdrWarn))
	}

	// Regenerate index
	if indexErr := storage.RegenerateIndex(docsDir); indexErr != nil {
		if !quiet {
			_, _ = fmt.Fprintf(streams.Err, "%s\n", ui.Warning(fmt.Sprintf(i18n.T().Cmd.ReleaseIndexRegenWarn, indexErr.Error())))
		}
	}

	// Output
	if quiet {
		// stdout = path only
		_, _ = fmt.Fprintln(streams.Out, filepath.Join(docsDir, filename))
	} else {
		ui.Verb(streams, "Released", filename)
	}

	return nil
}
