// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/greycoderk/lore/internal/cli"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
	gitpkg "github.com/greycoderk/lore/internal/git"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/greycoderk/lore/internal/store"
	"github.com/greycoderk/lore/internal/ui"
	"github.com/spf13/cobra"
)

func newDoctorCmd(_ *config.Config, streams domain.IOStreams) *cobra.Command {
	var fix bool
	var quiet bool
	var configOnly bool
	var rebuildStore bool

	cmd := &cobra.Command{
		Use:           "doctor",
		Short:         i18n.T().Cmd.DoctorShort,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// --config mode: validate config only, no .lore dir required.
			if configOnly {
				return runDoctorConfig(streams, ".", quiet)
			}

			if err := requireLoreDir(streams); err != nil {
				return err
			}

			// --rebuild-store mode
			if rebuildStore {
				return runRebuildStore(streams)
			}

			docsDir := filepath.Join(".lore", "docs")
			report, err := storage.Diagnose(docsDir)
			if err != nil {
				return fmt.Errorf("cmd: doctor: %w", err)
			}

			// Run config validation as part of standard diagnostic.
			cfgReport := config.ValidateConfig(".")

			if quiet && !fix {
				total := len(report.Issues) + len(cfgReport.Warnings) + len(cfgReport.Errors)
				_, _ = fmt.Fprintf(streams.Out, "%d\n", total)
				if total > 0 {
					return &cli.ExitCodeError{Code: cli.ExitError}
				}
				return nil
			}

			if !fix {
				// Diagnostic mode: corpus + config.
				return runDoctorDiagnoseWithConfig(streams, report, cfgReport)
			}

			// Fix mode
			fixReport, fixErr := storage.Fix(docsDir, report)
			if fixErr != nil {
				return fmt.Errorf("cmd: doctor: %w", fixErr)
			}

			if quiet {
				remaining := fixReport.Remaining + len(cfgReport.Warnings) + len(cfgReport.Errors)
				_, _ = fmt.Fprintf(streams.Out, "%d\n", remaining)
				if remaining > 0 || fixReport.Errors > 0 {
					return &cli.ExitCodeError{Code: cli.ExitError}
				}
				return nil
			}

			return runDoctorFixWithConfig(streams, report, fixReport, cfgReport)
		},
	}

	cmd.Flags().BoolVar(&fix, "fix", false, "Automatically repair fixable issues")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "Machine output: issue count on stdout")
	cmd.Flags().BoolVar(&configOnly, "config", false, "Validate configuration files only")
	cmd.Flags().BoolVar(&rebuildStore, "rebuild-store", false, "Reconstruct store.db from .lore/docs/ and git log")

	return cmd
}

// runDoctorConfig validates config files only (--config mode).
func runDoctorConfig(streams domain.IOStreams, dir string, quiet bool) error {
	cfgReport := config.ValidateConfig(dir)

	if quiet {
		total := len(cfgReport.Warnings) + len(cfgReport.Errors)
		_, _ = fmt.Fprintf(streams.Out, "%d\n", total)
		if total > 0 {
			return &cli.ExitCodeError{Code: cli.ExitError}
		}
		return nil
	}

	return displayConfigReport(streams, cfgReport)
}

// displayConfigReport renders the config validation report on stderr.
func displayConfigReport(streams domain.IOStreams, cfgReport *config.ConfigReport) error {
	_, _ = fmt.Fprintf(streams.Err, "\n%s\n\n", i18n.T().Cmd.DoctorConfigCheck)

	for _, e := range cfgReport.Errors {
		_, _ = fmt.Fprintf(streams.Err, "  %s  config  %s\n", ui.Error("✗"), e)
	}
	for _, w := range cfgReport.Warnings {
		_, _ = fmt.Fprintf(streams.Err, "  %s  config  %s\n", ui.Warning("⚠"), w)
	}

	if cfgReport.OK() {
		_, _ = fmt.Fprintf(streams.Err, "  %s  config  %s\n", ui.Success("✓"), i18n.T().Cmd.DoctorConfigOK)
		_, _ = fmt.Fprintf(streams.Err, "\n%s\n", i18n.T().Cmd.DoctorActiveValues)
		keys := make([]string, 0, len(cfgReport.Active))
		for k := range cfgReport.Active {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := cfgReport.Active[k]
			if v == "" {
				v = i18n.T().Cmd.DoctorValueNotSet
			}
			_, _ = fmt.Fprintf(streams.Err, "  %-22s %s\n", k, v)
		}
	}

	_, _ = fmt.Fprintln(streams.Err)

	total := len(cfgReport.Warnings) + len(cfgReport.Errors)
	if total > 0 {
		return &cli.ExitCodeError{Code: cli.ExitError}
	}
	return nil
}

// runDoctorDiagnoseWithConfig displays corpus + config diagnostic on stderr.
func runDoctorDiagnoseWithConfig(streams domain.IOStreams, report *storage.DiagnosticReport, cfgReport *config.ConfigReport) error {
	_, _ = fmt.Fprintf(streams.Err, "\n%s\n\n", i18n.T().Cmd.DoctorDocsCheck)

	categories := []string{"orphan-tmp", "stale-index", "stale-cache", "broken-ref", "invalid-frontmatter"}
	issuesByCategory := make(map[string][]storage.Issue)
	for _, issue := range report.Issues {
		issuesByCategory[issue.Category] = append(issuesByCategory[issue.Category], issue)
	}

	for _, cat := range categories {
		issues := issuesByCategory[cat]
		if len(issues) == 0 {
			_, _ = fmt.Fprintf(streams.Err, "  %s  %-22s %s\n", ui.Success("✓"), cat, ui.Dim(i18n.T().Cmd.DoctorNoneFound))
		} else {
			for _, issue := range issues {
				detail := issue.File
				if issue.Detail != "" && issue.Detail != issue.File {
					detail = issue.File + " (" + issue.Detail + ")"
				}
				_, _ = fmt.Fprintf(streams.Err, "  %s  %-22s %s\n", ui.Error("✗"), cat, detail)
			}
		}
	}

	// Config section
	if cfgReport.OK() {
		_, _ = fmt.Fprintf(streams.Err, "  %s  %-22s %s\n", ui.Success("✓"), "config", ui.Dim(i18n.T().Cmd.DoctorConfigOKInline))
	} else {
		for _, e := range cfgReport.Errors {
			_, _ = fmt.Fprintf(streams.Err, "  %s  %-22s %s\n", ui.Error("✗"), "config", e)
		}
		for _, w := range cfgReport.Warnings {
			_, _ = fmt.Fprintf(streams.Err, "  %s  %-22s %s\n", ui.Warning("⚠"), "config", w)
		}
	}

	_, _ = fmt.Fprintln(streams.Err)
	total := len(report.Issues) + len(cfgReport.Warnings) + len(cfgReport.Errors)
	if total == 0 {
		_, _ = fmt.Fprintf(streams.Err, "%s\n", i18n.T().Cmd.DoctorHealthAllGood)
		return nil
	}

	_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.DoctorIssuesFound+"\n", total)
	return &cli.ExitCodeError{Code: cli.ExitError}
}

// runDoctorFixWithConfig displays the fix report + config warnings on stderr.
func runDoctorFixWithConfig(streams domain.IOStreams, report *storage.DiagnosticReport, fixReport *storage.FixReport, cfgReport *config.ConfigReport) error {
	_, _ = fmt.Fprintln(streams.Err)
	for _, detail := range fixReport.Details {
		ui.Verb(streams, "Fixed", detail)
	}

	// Show manual-fix-required items
	for _, issue := range report.Issues {
		if !issue.AutoFix {
			suggestion := fmt.Sprintf(i18n.T().Cmd.DoctorManualFix, issue.File, issue.Detail)
			_, _ = fmt.Fprintf(streams.Err, "  %s  %s\n", ui.Warning("⚠"), suggestion)
		}
	}

	// Show config warnings/errors
	for _, e := range cfgReport.Errors {
		_, _ = fmt.Fprintf(streams.Err, "  %s  config  %s\n", ui.Error("✗"), e)
	}
	for _, w := range cfgReport.Warnings {
		_, _ = fmt.Fprintf(streams.Err, "  %s  config  %s\n", ui.Warning("⚠"), w)
	}

	_, _ = fmt.Fprintln(streams.Err)
	remaining := fixReport.Remaining + len(cfgReport.Warnings) + len(cfgReport.Errors)
	_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.DoctorFixSummary+"\n", fixReport.Fixed, remaining)

	if remaining > 0 || fixReport.Errors > 0 {
		return &cli.ExitCodeError{Code: cli.ExitError}
	}
	return nil
}

// runRebuildStore reconstructs store.db from .lore/docs/ and git log.
func runRebuildStore(streams domain.IOStreams) error {
	storePath := filepath.Join(".lore", "store.db")
	docsDir := filepath.Join(".lore", "docs")

	s, err := store.Open(storePath)
	if err != nil {
		return fmt.Errorf("cmd: doctor: open store: %w", err)
	}
	defer func() { _ = s.Close() }()

	git := gitpkg.NewAdapter(".")

	docCount, docSkipped, commitCount, err := s.RebuildFromSources(context.Background(), docsDir, git)
	if err != nil {
		return fmt.Errorf("cmd: doctor: rebuild: %w", err)
	}

	_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.DoctorStoreRebuilt, docCount)
	if docSkipped > 0 {
		_, _ = fmt.Fprintf(streams.Err, " "+i18n.T().Cmd.DoctorStoreSkipped, docSkipped)
	}
	_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.DoctorStoreCommits+"\n", commitCount)
	return nil
}
