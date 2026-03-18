// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"

	"github.com/greycoderk/lore/internal/cli"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	gitpkg "github.com/greycoderk/lore/internal/git"
	"github.com/greycoderk/lore/internal/status"
	"github.com/greycoderk/lore/internal/ui"
	"github.com/spf13/cobra"
)

func newStatusCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	var flagQuiet bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check your documentation health",
		Long:  "Display a dashboard showing hooks, documents, pending items, and health.",
		Example: `  lore status
  lore status --quiet`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// AC-9: Check .lore/ exists
			if _, err := os.Stat(".lore"); err != nil {
				if os.IsNotExist(err) {
					fmt.Fprintln(streams.Err, "Error: Lore not initialized.")
					fmt.Fprintln(streams.Err, "  Run: lore init")
				} else {
					fmt.Fprintf(streams.Err, "Error: cannot access .lore/: %v\n", err)
				}
				return &cli.ExitCodeError{Code: cli.ExitError}
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

func renderDashboard(streams domain.IOStreams, info *status.StatusInfo) error {
	w := streams.Err

	// Header
	fmt.Fprintf(w, "lore status — %s\n\n", info.ProjectName)

	// Hook
	hookVal := ui.Error("not installed") + ". Run: lore hook install"
	if info.HookInstalled {
		hookVal = "installed (post-commit)"
	}
	fmt.Fprintf(w, "%-10s%s\n", "Hook:", hookVal)

	// Docs
	docsVal := fmt.Sprintf("%d documented", info.DocCount)
	if info.PendingCount > 0 {
		docsVal += fmt.Sprintf(", %d pending", info.PendingCount)
	}
	fmt.Fprintf(w, "%-10s%s\n", "Docs:", docsVal)

	// Express ratio
	total := info.ExpressCount + info.CompleteCount
	if total > 0 {
		pctComplete := info.CompleteCount * 100 / total
		pctExpress := 100 - pctComplete
		expressLine := fmt.Sprintf("%d%% (%d/%d) — %d%% with alternatives/impact",
			pctExpress, info.ExpressCount, total, pctComplete)
		if info.ReadErrors > 0 {
			expressLine += fmt.Sprintf(" (%d unreadable)", info.ReadErrors)
		}
		fmt.Fprintf(w, "%-10s%s\n", "Express:", expressLine)
	} else if info.ReadErrors > 0 {
		fmt.Fprintf(w, "%-10s%s\n", "Express:", fmt.Sprintf("(%d unreadable)", info.ReadErrors))
	}

	// Angela
	angelaVal := fmt.Sprintf("%s mode", info.AngelaMode)
	if info.AngelaMode == "draft" {
		angelaVal += " (no API key)"
	} else if info.AIProvider != "" {
		angelaVal += fmt.Sprintf(" (%s)", info.AIProvider)
	}
	fmt.Fprintf(w, "%-10s%s\n", "Angela:", angelaVal)

	// Health
	if info.HealthIssues == 0 {
		fmt.Fprintf(w, "%-10s%s all good\n", "Health:", ui.Success("\u2713"))
	} else {
		fmt.Fprintf(w, "%-10s%s %d issues. Run: lore doctor\n",
			"Health:", ui.Error("\u2717"), info.HealthIssues)
	}

	// Tagline
	fmt.Fprintf(w, "\n%s\n", ui.Dim("Your code knows what. Lore knows why."))

	return nil
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

	fmt.Fprintf(w, "hook=%s\n", hookStatus)
	fmt.Fprintf(w, "docs=%d\n", info.DocCount)
	fmt.Fprintf(w, "pending=%d\n", info.PendingCount)
	fmt.Fprintf(w, "health=%s\n", healthStatus)
	if info.ReadErrors > 0 {
		fmt.Fprintf(w, "read_errors=%d\n", info.ReadErrors)
	}
	fmt.Fprintf(w, "angela=%s\n", info.AngelaMode)

	return nil
}
