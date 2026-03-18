package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/greycoderk/lore/internal/cli"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/greycoderk/lore/internal/ui"
	"github.com/spf13/cobra"
)

func newDoctorCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	var fix bool
	var quiet bool

	cmd := &cobra.Command{
		Use:           "doctor",
		Short:         "Fix documentation inconsistencies",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkLoreDir(streams); err != nil {
				return err
			}

			docsDir := filepath.Join(".lore", "docs")
			report, err := storage.Diagnose(docsDir)
			if err != nil {
				return fmt.Errorf("cmd: doctor: %w", err)
			}

			if quiet && !fix {
				fmt.Fprintf(streams.Out, "%d\n", len(report.Issues))
				if len(report.Issues) > 0 {
					return &cli.ExitCodeError{Code: cli.ExitError}
				}
				return nil
			}

			if !fix {
				// Diagnostic mode
				return runDoctorDiagnose(streams, report)
			}

			// Fix mode
			fixReport, fixErr := storage.Fix(docsDir, report)
			if fixErr != nil {
				return fmt.Errorf("cmd: doctor: %w", fixErr)
			}

			if quiet {
				fmt.Fprintf(streams.Out, "%d\n", fixReport.Remaining)
				if fixReport.Remaining > 0 || fixReport.Errors > 0 {
					return &cli.ExitCodeError{Code: cli.ExitError}
				}
				return nil
			}

			return runDoctorFix(streams, report, fixReport)
		},
	}

	cmd.Flags().BoolVar(&fix, "fix", false, "Automatically repair fixable issues")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "Machine output: issue count on stdout")

	return cmd
}

// runDoctorDiagnose displays the diagnostic report on stderr.
func runDoctorDiagnose(streams domain.IOStreams, report *storage.DiagnosticReport) error {
	fmt.Fprintf(streams.Err, "\nChecking .lore/docs/ ...\n\n")

	categories := []string{"orphan-tmp", "stale-index", "stale-cache", "broken-ref", "invalid-frontmatter"}
	issuesByCategory := make(map[string][]storage.Issue)
	for _, issue := range report.Issues {
		issuesByCategory[issue.Category] = append(issuesByCategory[issue.Category], issue)
	}

	for _, cat := range categories {
		issues := issuesByCategory[cat]
		if len(issues) == 0 {
			fmt.Fprintf(streams.Err, "  %s  %-22s %s\n", ui.Success("✓"), cat, ui.Dim("(none found)"))
		} else {
			for _, issue := range issues {
				detail := issue.File
				if issue.Detail != "" && issue.Detail != issue.File {
					detail = issue.File + " (" + issue.Detail + ")"
				}
				fmt.Fprintf(streams.Err, "  %s  %-22s %s\n", ui.Error("✗"), cat, detail)
			}
		}
	}

	fmt.Fprintln(streams.Err)
	if len(report.Issues) == 0 {
		fmt.Fprintf(streams.Err, "Health: all good. 0 issues found.\n")
		return nil
	}

	fmt.Fprintf(streams.Err, "%d issues found. Run: lore doctor --fix\n", len(report.Issues))
	return &cli.ExitCodeError{Code: cli.ExitError}
}

// runDoctorFix displays the fix report on stderr.
func runDoctorFix(streams domain.IOStreams, report *storage.DiagnosticReport, fixReport *storage.FixReport) error {
	fmt.Fprintln(streams.Err)
	for _, detail := range fixReport.Details {
		ui.Verb(streams, "Fixed", detail)
	}

	// Show manual-fix-required items
	for _, issue := range report.Issues {
		if !issue.AutoFix {
			suggestion := fmt.Sprintf("%s — manual fix required: %s", issue.File, issue.Detail)
			fmt.Fprintf(streams.Err, "  %s  %s\n", ui.Warning("⚠"), suggestion)
		}
	}

	fmt.Fprintln(streams.Err)
	fmt.Fprintf(streams.Err, "Fixed: %d issues. %d remaining.\n", fixReport.Fixed, fixReport.Remaining)

	if fixReport.Remaining > 0 || fixReport.Errors > 0 {
		return &cli.ExitCodeError{Code: cli.ExitError}
	}
	return nil
}
