// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/angela/gc"
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

// shellSingleQuote wraps s in POSIX single quotes so that copy-pasting
// the suggestion into a shell cannot execute embedded metacharacters
// (spaces, `;`, `$`, backticks, newlines, etc). The escape for an
// embedded single quote is the standard POSIX `'\''` trick.
//
// Story 8-22 P1: a filename coming off `os.ReadDir` is not filtered
// through ValidateFilename by the doctor caller, so a file literally
// named `foo.md; rm -rf ~` could land in the hint. Shell-quoting is
// the minimal, correct guard.
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// hasPolishBackup reports whether at least one polish backup exists
// for the given docs-relative filename. Used by doctor to decide
// whether to include the "restore from polish backup" hint on a
// malformed-frontmatter issue (story 8-22, AC-5).
//
// Errors during backup listing are treated as "no backup available":
// a missing state dir, a filesystem permission error, or a
// non-existent backup subdirectory all mean the hint is irrelevant.
// This function never returns an error to the caller — it is a
// best-effort probe.
func hasPolishBackup(cfg *config.Config, filename string) bool {
	if cfg == nil {
		return false
	}
	workDir, err := os.Getwd()
	if err != nil {
		return false
	}
	stateDir := config.ResolveStateDir(workDir, cfg, cfg.DetectedMode)
	backupSubdir := cfg.Angela.Polish.Backup.Path
	if backupSubdir == "" {
		backupSubdir = "polish-backups"
	}
	entries, err := angela.ListBackups(stateDir, backupSubdir, filename)
	if err != nil {
		return false
	}
	return len(entries) > 0
}

// emitMalformedFrontmatterHint prints the two-action suggestion block
// for a malformed-frontmatter issue (story 8-22 AC-4). The restore
// action is conditional on an existing polish backup (AC-5).
//
// Filenames are always shell-quoted so a user copy-pasting the
// suggested command cannot accidentally execute embedded shell
// metacharacters (spaces, `;`, `$`, backticks, newlines).
func emitMalformedFrontmatterHint(streams domain.IOStreams, cfg *config.Config, filename string) {
	quoted := shellSingleQuote(filename)
	_, _ = fmt.Fprintf(streams.Err, "      %s\n", ui.Dim(i18n.T().Cmd.DoctorSuggestedActions))
	if hasPolishBackup(cfg, filename) {
		_, _ = fmt.Fprintf(streams.Err, "        %s %s\n", ui.Dim("-"), i18n.T().Cmd.DoctorMalformedRestore)
		_, _ = fmt.Fprintf(streams.Err, "            lore angela polish --restore %s\n", quoted)
	}
	_, _ = fmt.Fprintf(streams.Err, "        %s %s\n", ui.Dim("-"), i18n.T().Cmd.DoctorMalformedEditManual)
}

func newDoctorCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	var fix bool
	var quiet bool
	var configOnly bool
	var rebuildStore bool
	var prune bool
	var dryRun bool

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

			// Story 8-23: --prune mode runs the GC Pruner registry.
			// Mutually exclusive with --fix, --rebuild-store, --config
			// (enforced via cobra.MarkFlagsMutuallyExclusive).
			if prune {
				return runDoctorPrune(cmd.Context(), streams, cfg, dryRun, quiet)
			}

			docsDir := filepath.Join(".lore", "docs")

			var spin *ui.Spinner
			if !quiet {
				spin = ui.StartSpinner(streams, i18n.T().Cmd.DoctorScanning)
			}
			report, err := storage.Diagnose(docsDir)
			if err != nil {
				if spin != nil {
					spin.Stop()
				}
				return fmt.Errorf("cmd: doctor: %w", err)
			}

			// Run config validation as part of standard diagnostic.
			cfgReport := config.ValidateConfig(".")
			if spin != nil {
				spin.StopWith(fmt.Sprintf("✓ %s", fmt.Sprintf(i18n.T().Cmd.DoctorScanned, report.DocCount, report.Checked)))
			}

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
				return runDoctorDiagnoseWithConfig(streams, cfg, report, cfgReport)
			}

			// Fix mode
			var spinFix *ui.Spinner
			if !quiet {
				spinFix = ui.StartSpinner(streams, i18n.T().Cmd.DoctorFixing)
			}
			fixReport, fixErr := storage.Fix(docsDir, report)
			if spinFix != nil {
				spinFix.Stop()
			}
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

			return runDoctorFixWithConfig(streams, cfg, report, fixReport, cfgReport)
		},
	}

	tc := i18n.T().Cmd
	cmd.Flags().BoolVar(&fix, "fix", false, tc.DoctorFlagFix)
	cmd.Flags().BoolVar(&quiet, "quiet", false, tc.DoctorFlagQuiet)
	cmd.Flags().BoolVar(&configOnly, "config", false, tc.DoctorFlagConfig)
	cmd.Flags().BoolVar(&rebuildStore, "rebuild-store", false, tc.DoctorFlagRebuildStore)
	// Story 8-23: unified retention.
	cmd.Flags().BoolVar(&prune, "prune", false, tc.DoctorFlagPrune)
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, tc.DoctorFlagDryRun)
	cmd.MarkFlagsMutuallyExclusive("prune", "fix")
	cmd.MarkFlagsMutuallyExclusive("prune", "rebuild-store")
	cmd.MarkFlagsMutuallyExclusive("prune", "config")

	return cmd
}

// runDoctorPrune runs every registered Pruner and renders a summary
// of the outcome on stderr (or a tab-separated machine output on
// stdout with --quiet). Returns an error if any Pruner reports a
// non-nil Err, so CI pipelines can gate on prune success.
func runDoctorPrune(ctx context.Context, streams domain.IOStreams, cfg *config.Config, dryRun, quiet bool) error {
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cmd: doctor: prune: cwd: %w", err)
	}
	reports := gc.RunAll(ctx, workDir, cfg, dryRun)

	if quiet {
		// Tab-separated: feature\tremoved\tkept\tbytes
		hasErr := false
		for _, r := range reports {
			_, _ = fmt.Fprintf(streams.Out, "%s\t%d\t%d\t%d\n", r.Feature, r.Removed, r.Kept, r.Bytes)
			if r.Err != nil {
				hasErr = true
			}
		}
		if hasErr {
			return &cli.ExitCodeError{Code: cli.ExitError}
		}
		return nil
	}

	_, _ = fmt.Fprintln(streams.Err)
	_, _ = fmt.Fprintf(streams.Err, "%s\n", ui.Bold(i18n.T().Cmd.DoctorPruneHeader))
	var totalBytes int64
	var totalRemoved int
	var anyErr error
	for _, r := range reports {
		if r.Err != nil {
			_, _ = fmt.Fprintf(streams.Err, "  %s  "+i18n.T().Cmd.DoctorPruneRowErr+"\n", ui.Error("✗"), r.Feature, r.Err)
			if anyErr == nil {
				anyErr = r.Err
			}
			continue
		}
		status := ui.Success("✓")
		_, _ = fmt.Fprintf(streams.Err, "  %s  "+i18n.T().Cmd.DoctorPruneRowOK+"\n",
			status, r.Feature, r.Removed, r.Kept, humanBytes(r.Bytes))
		totalBytes += r.Bytes
		totalRemoved += r.Removed
	}
	_, _ = fmt.Fprintf(streams.Err, "  %-26s "+i18n.T().Cmd.DoctorPruneTotal+"\n", "", humanBytes(totalBytes))
	if dryRun {
		_, _ = fmt.Fprintf(streams.Err, "  %s\n", ui.Dim(i18n.T().Cmd.DoctorPruneDryRunFoot))
	}
	if anyErr != nil {
		return &cli.ExitCodeError{Code: cli.ExitError}
	}
	return nil
}

// humanBytes renders a byte count as "12 B", "3.4 KB", "2.1 MB".
func humanBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
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
func runDoctorDiagnoseWithConfig(streams domain.IOStreams, cfg *config.Config, report *storage.DiagnosticReport, cfgReport *config.ConfigReport) error {
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
					// Story 8-22: prefix the detail with the subkind
					// so malformed and missing can be told apart at a
					// glance in the diagnose output.
					prefix := ""
					if cat == "invalid-frontmatter" && issue.Subkind != "" {
						prefix = issue.Subkind + ": "
					}
					detail = issue.File + " (" + prefix + issue.Detail + ")"
				}
				_, _ = fmt.Fprintf(streams.Err, "  %s  %-22s %s\n", ui.Error("✗"), cat, detail)
				// Story 8-22 / AC-4+AC-5: surface the structured
				// suggestion block on malformed FM so users know the
				// restore/manual-edit options immediately — not only
				// under --fix.
				if cat == "invalid-frontmatter" && issue.Subkind == storage.SubkindFrontmatterMalformed {
					emitMalformedFrontmatterHint(streams, cfg, issue.File)
				}
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
func runDoctorFixWithConfig(streams domain.IOStreams, cfg *config.Config, report *storage.DiagnosticReport, fixReport *storage.FixReport, cfgReport *config.ConfigReport) error {
	_, _ = fmt.Fprintln(streams.Err)
	for _, detail := range fixReport.Details {
		ui.Verb(streams, "Fixed", detail)
	}

	// Show manual-fix-required items
	for _, issue := range report.Issues {
		if !issue.AutoFix {
			suggestion := fmt.Sprintf(i18n.T().Cmd.DoctorManualFix, issue.File, issue.Detail)
			_, _ = fmt.Fprintf(streams.Err, "  %s  %s\n", ui.Warning("⚠"), suggestion)
			// Story 8-22 / AC-4+AC-5: malformed frontmatter gets the
			// restore+edit suggestion block right after the ⚠ line.
			if issue.Category == "invalid-frontmatter" && issue.Subkind == storage.SubkindFrontmatterMalformed {
				emitMalformedFrontmatterHint(streams, cfg, issue.File)
			}
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

	spin := ui.StartSpinner(streams, i18n.T().Cmd.DoctorRebuilding)
	docCount, docSkipped, commitCount, err := s.RebuildFromSources(context.Background(), docsDir, git)
	spin.Stop()
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
