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
	"github.com/greycoderk/lore/internal/storage"
	"github.com/greycoderk/lore/internal/ui"
	"github.com/spf13/cobra"
)

func newReleaseCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	var (
		fromFlag    string
		toFlag      string
		versionFlag string
		quietFlag   bool
	)

	cmd := &cobra.Command{
		Use:           "release",
		Short:         "Generate release notes from your documentation",
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
	// AC-8: Check if Lore is initialized
	if _, err := os.Stat(".lore"); os.IsNotExist(err) {
		if !quiet {
			ui.ActionableError(streams, "Lore not initialized.", "lore init")
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
				fmt.Fprintf(streams.Err, "%s No Git tags found. Create a tag first: git tag v0.1.0\n", ui.Error("Error:"))
			fmt.Fprintf(streams.Err, "  Or use: lore release --from <commit-sha>\n")
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
	docs, parseErr, err := storage.CollectReleaseDocuments(docsDir, commits)
	if err != nil {
		return fmt.Errorf("cmd: release: %w", err)
	}
	if parseErr != nil && !quiet {
		fmt.Fprintf(streams.Err, "%s\n", ui.Warning("Warning: some documents could not be parsed: "+parseErr.Error()))
	}

	// AC-6: No documents
	if len(docs) == 0 {
		if !quiet {
			fmt.Fprintf(streams.Err, "No documented changes in this range.\n")
			fmt.Fprintf(streams.Err, "  Try: lore release --from <earlier-tag>\n")
		}
		return nil
	}

	date := time.Now().Format("2006-01-02")

	// Generate release notes file
	filename, err := storage.GenerateReleaseNotes(version, date, docs, docsDir)
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
		fmt.Fprintf(streams.Err, "%s\n", ui.Warning("Warning: CHANGELOG.md header not found, inserting at top."))
	}

	// Regenerate index
	if indexErr := storage.RegenerateIndex(docsDir); indexErr != nil {
		if !quiet {
			fmt.Fprintf(streams.Err, "%s\n", ui.Warning("Warning: index regeneration failed: "+indexErr.Error()))
		}
	}

	// Output
	if quiet {
		// AC-7: stdout = path only
		fmt.Fprintln(streams.Out, filepath.Join(docsDir, filename))
	} else {
		ui.Verb(streams, "Released", filename)
	}

	return nil
}
