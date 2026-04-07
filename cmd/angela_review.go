// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/greycoderk/lore/internal/ai"
	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/credential"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/spf13/cobra"
)

func newAngelaReviewCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	var flagQuiet bool

	cmd := &cobra.Command{
		Use:           "review",
		Short:         i18n.T().Cmd.AngelaReviewShort,
		Args:          cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// AC-9: Check .lore/ exists
			if err := requireLoreDir(streams); err != nil {
				return err
			}
			docsDir := filepath.Join(domain.LoreDir, domain.DocsDir)

			// TODO(post-mvp): extract review orchestration to internal/service/angela.go

			// AC-1, AC-2: Prepare doc summaries (before provider — no point requesting API if corpus too small)
			corpusStore := &storage.CorpusStore{Dir: docsDir}
			summaries, totalCount, err := angela.PrepareDocSummaries(corpusStore)
			if err != nil {
				return err
			}

			// AC-5: Instantiate provider
			store := credential.NewStore()
			provider, err := ai.NewProvider(cfg, store, streams.Err)
			if err != nil {
				return err
			}
			if provider == nil {
				return fmt.Errorf("%s", i18n.T().Cmd.AngelaReviewNoProvider)
			}

			// Warn if corpus > 50 docs (on stderr)
			if totalCount > 50 && !flagQuiet {
				_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.AngelaReviewCorpusNote+"\n", totalCount)
			}

			// Style guide
			var styleGuideStr string
			if cfg.Angela.StyleGuide != nil {
				guide := angela.ParseStyleGuide(cfg.Angela.StyleGuide)
				styleGuideStr = angela.FormatStyleGuideRules(guide)
			}

			// AC-1: Exactly 1 API call
			report, err := angela.Review(cmd.Context(), provider, summaries, styleGuideStr)
			if err != nil {
				return err
			}

			// Format report on stdout (AC-4, AC-11)
			formatReviewReport(streams, report, totalCount, flagQuiet)

			// Save review cache for lore status integration
			if err := angela.SaveReviewCache(domain.LoreDir, report, totalCount); err != nil {
				_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.AngelaReviewCacheWarn+"\n", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&flagQuiet, "quiet", false, "Suppress human messages on stderr")

	return cmd
}

// formatReviewReport writes the review report to stdout and human messages to stderr.
func formatReviewReport(streams domain.IOStreams, report *angela.ReviewReport, totalCorpus int, quiet bool) {
	// Header on stderr (unless quiet)
	if !quiet {
		if totalCorpus > report.DocCount {
			_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.AngelaReviewHdrPartial+"\n\n", report.DocCount, totalCorpus)
		} else {
			_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.AngelaReviewHdrFull+"\n\n", report.DocCount)
		}
	}

	// Findings on stdout (always)
	for _, f := range report.Findings {
		// severity + title
		_, _ = fmt.Fprintf(streams.Out, "  %-14s %s\n", f.Severity, f.Title)
		// documents with date hint
		if len(f.Documents) > 0 {
			_, _ = fmt.Fprintf(streams.Out, "  %14s %s\n", "", strings.Join(f.Documents, " vs "))
		}
		// description
		if f.Description != "" {
			_, _ = fmt.Fprintf(streams.Out, "  %14s %s\n", "", f.Description)
		}
		_, _ = fmt.Fprintln(streams.Out)
	}

	// Summary on stderr (unless quiet)
	if !quiet {
		if len(report.Findings) == 0 {
			_, _ = fmt.Fprintf(streams.Err, "%s\n", i18n.T().Cmd.AngelaReviewCoherent)
		} else {
			counts := countSeverities(report.Findings)
			_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.AngelaReviewFindingSum+"\n", len(report.Findings), counts)
		}
	}
}

// countSeverities builds a summary string like "1 contradiction, 1 gap, 1 style".
func countSeverities(findings []angela.ReviewFinding) string {
	counts := map[string]int{}
	for _, f := range findings {
		counts[f.Severity]++
	}

	order := []string{"contradiction", "gap", "obsolete", "style"}
	seen := map[string]bool{}
	var parts []string
	for _, sev := range order {
		if n, ok := counts[sev]; ok {
			parts = append(parts, fmt.Sprintf("%d %s", n, sev))
			seen[sev] = true
		}
	}
	// Catch unknown severities
	for sev, n := range counts {
		if !seen[sev] {
			parts = append(parts, fmt.Sprintf("%d %s", n, sev))
		}
	}
	return strings.Join(parts, ", ")
}
