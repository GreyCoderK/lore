// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"
	"time"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
	gitpkg "github.com/greycoderk/lore/internal/git"
	"github.com/greycoderk/lore/internal/status"
	"github.com/greycoderk/lore/internal/ui"
	"github.com/spf13/cobra"
)

func newStatusCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	var flagQuiet bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: i18n.T().Cmd.StatusShort,
		Long:  i18n.T().Cmd.StatusLong,
		Example: `  lore status
  lore status --quiet`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// AC-9: Check .lore/ exists
			if err := requireLoreDir(streams); err != nil {
				return err
			}

			git := gitpkg.NewAdapter(".")
			info, err := status.CollectStatus(cfg, git, ".lore")
			if err != nil {
				return fmt.Errorf("cmd: status: %w", err)
			}

			if flagQuiet {
				return renderQuiet(streams, info)
			}
			return renderDashboard(streams, info)
		},
	}

	cmd.Flags().BoolVar(&flagQuiet, "quiet", false, "Machine-readable output on stdout")

	return cmd
}

// renderDashboard writes the human-readable dashboard to stderr (interactive use).
// For pipeable output, use --quiet which writes to stdout.
func renderDashboard(streams domain.IOStreams, info *status.StatusInfo) error {
	w := streams.Err

	// Header
	_, _ = fmt.Fprintf(w, i18n.T().Cmd.StatusHeader+"\n\n", info.ProjectName)

	// Hook
	hookVal := ui.Error(i18n.T().Cmd.StatusHookNotInstalled) + i18n.T().Cmd.StatusHookNotInstHint
	if info.HookInstalled {
		hookVal = i18n.T().Cmd.StatusHookInstalled
	}
	_, _ = fmt.Fprintf(w, "%-10s%s\n", i18n.T().Cmd.StatusHookLabel, hookVal)

	// Docs
	docsVal := fmt.Sprintf(i18n.T().Cmd.StatusDocsDocumented, info.DocCount)
	if info.PendingCount > 0 {
		docsVal += fmt.Sprintf(i18n.T().Cmd.StatusDocsPending, info.PendingCount)
	}
	_, _ = fmt.Fprintf(w, "%-10s%s\n", i18n.T().Cmd.StatusDocsLabel, docsVal)

	// Express ratio
	total := info.ExpressCount + info.CompleteCount
	if total > 0 {
		pctComplete := info.CompleteCount * 100 / total
		pctExpress := 100 - pctComplete
		expressLine := fmt.Sprintf(i18n.T().Cmd.StatusExpressLine,
			pctExpress, info.ExpressCount, total, pctComplete)
		if info.ReadErrors > 0 {
			expressLine += " " + fmt.Sprintf(i18n.T().Cmd.StatusExpressUnreadable, info.ReadErrors)
		}
		_, _ = fmt.Fprintf(w, "%-10s%s\n", i18n.T().Cmd.StatusExpressLabel, expressLine)
	} else if info.ReadErrors > 0 {
		_, _ = fmt.Fprintf(w, "%-10s%s\n", i18n.T().Cmd.StatusExpressLabel, fmt.Sprintf(i18n.T().Cmd.StatusExpressUnreadable, info.ReadErrors))
	}

	// Angela
	angelaVal := fmt.Sprintf(i18n.T().Cmd.StatusAngelaMode, info.AngelaMode)
	if info.AngelaMode == "draft" {
		angelaVal += " " + i18n.T().Cmd.StatusAngelaNoApiKey
	} else if info.AIProvider != "" {
		angelaVal += " " + fmt.Sprintf(i18n.T().Cmd.StatusAngelaProvider, info.AIProvider)
	}
	if info.DocCount > 0 && info.AngelaDocsNeedReview > 0 {
		angelaVal += " " + fmt.Sprintf(i18n.T().Cmd.StatusAngelaDocsReview, info.AngelaDocsNeedReview)
	} else if info.DocCount > 0 {
		angelaVal += " — " + ui.Success(i18n.T().Cmd.StatusAngelaAllClean)
	}
	_, _ = fmt.Fprintf(w, "%-10s%s\n", i18n.T().Cmd.StatusAngelaLabel, angelaVal)

	// Last Angela Review (from cache)
	reviewCache, _ := angela.LoadReviewCache(".lore")
	if reviewCache != nil {
		reviewAge := formatReviewAge(reviewCache.LastReview)
		if len(reviewCache.Findings) == 0 {
			_, _ = fmt.Fprintf(w, "%-10s%s (%s)\n", i18n.T().Cmd.StatusReviewLabel, ui.Success(i18n.T().Cmd.StatusReviewNoIssues), reviewAge)
		} else {
			_, _ = fmt.Fprintf(w, "%-10s%s\n",
				i18n.T().Cmd.StatusReviewLabel, fmt.Sprintf(i18n.T().Cmd.StatusReviewFindings, len(reviewCache.Findings), reviewAge))
		}
	}

	// Health
	if info.HealthIssues == 0 {
		_, _ = fmt.Fprintf(w, "%-10s%s %s\n", i18n.T().Cmd.StatusHealthLabel, ui.Success("\u2713"), i18n.T().Cmd.StatusHealthAllGood)
	} else {
		_, _ = fmt.Fprintf(w, "%-10s%s %s\n",
			i18n.T().Cmd.StatusHealthLabel, ui.Error("\u2717"), fmt.Sprintf(i18n.T().Cmd.StatusHealthIssues, info.HealthIssues))
	}

	// Tagline
	_, _ = fmt.Fprintf(w, "\n%s\n", ui.Dim(i18n.T().Cmd.StatusTagline))

	return nil
}

// formatReviewAge returns a human-friendly age string for the last review.
func formatReviewAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Hour:
		return i18n.T().Cmd.StatusReviewAgeJustNow
	case d < 24*time.Hour:
		return fmt.Sprintf(i18n.T().Cmd.StatusReviewAgeHours, int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf(i18n.T().Cmd.StatusReviewAgeDays, int(d.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}

func renderQuiet(streams domain.IOStreams, info *status.StatusInfo) error {
	w := streams.Out

	hookStatus := "not-installed"
	if info.HookInstalled {
		hookStatus = "installed"
	}

	healthStatus := "ok"
	if info.HealthIssues > 0 {
		healthStatus = fmt.Sprintf("%d-issues", info.HealthIssues)
	}

	_, _ = fmt.Fprintf(w, "hook=%s\n", hookStatus)
	_, _ = fmt.Fprintf(w, "docs=%d\n", info.DocCount)
	_, _ = fmt.Fprintf(w, "pending=%d\n", info.PendingCount)
	_, _ = fmt.Fprintf(w, "health=%s\n", healthStatus)
	if info.ReadErrors > 0 {
		_, _ = fmt.Fprintf(w, "read_errors=%d\n", info.ReadErrors)
	}
	_, _ = fmt.Fprintf(w, "angela=%s\n", info.AngelaMode)
	_, _ = fmt.Fprintf(w, "angela_review=%d\n", info.AngelaDocsNeedReview)
	_, _ = fmt.Fprintf(w, "angela_suggestions=%d\n", info.AngelaSuggestions)

	reviewCache, _ := angela.LoadReviewCache(".lore")
	if reviewCache != nil {
		_, _ = fmt.Fprintf(w, "review_findings=%d\n", len(reviewCache.Findings))
		_, _ = fmt.Fprintf(w, "review_age=%s\n", formatReviewAge(reviewCache.LastReview))
	}

	return nil
}
