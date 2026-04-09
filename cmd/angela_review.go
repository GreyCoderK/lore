// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/greycoderk/lore/internal/ai"
	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/credential"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/ui"
	"github.com/spf13/cobra"
)

func newAngelaReviewCmd(cfg *config.Config, streams domain.IOStreams, flagPath *string) *cobra.Command {
	var flagQuiet bool
	var flagFor string
	var flagFilter string
	var flagAll bool

	cmd := &cobra.Command{
		Use:           "review",
		Short:         i18n.T().Cmd.AngelaReviewShort,
		Args:          cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Resolve docs directory: --path (standalone) or .lore/docs (normal)
			docsDir, standalone := resolveDocsDir(flagPath)
			if !standalone {
				// AC-9: Check .lore/ exists
				if err := requireLoreDir(streams); err != nil {
					return err
				}
			}

			// Build review filter from flags
			var reviewFilter angela.ReviewFilter
			if flagFilter != "" {
				re, reErr := regexp.Compile(flagFilter)
				if reErr != nil {
					return fmt.Errorf("angela: review: invalid --filter regex: %w", reErr)
				}
				reviewFilter.Pattern = re
			}
			reviewFilter.All = flagAll

			// AC-1, AC-2: Prepare doc summaries (before provider — no point requesting API if corpus too small)
			corpusStore := newCorpusReader(docsDir, standalone)
			summaries, totalCount, err := angela.PrepareDocSummaries(corpusStore, reviewFilter)
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

			ta := i18n.T().Angela

			// Cap audience length to prevent prompt bloat
			if len(flagFor) > 200 {
				flagFor = flagFor[:200]
			}

			// Show mode
			if flagFor != "" && !flagQuiet {
				_, _ = fmt.Fprintf(streams.Err, "\n%s\n", ui.Bold(fmt.Sprintf(ta.UIMode, flagFor)))
			}

			// --- Step 1/2: Preparing ---
			timeout := cfg.AI.Timeout
			if timeout <= 0 {
				timeout = 60 * time.Second
			}
			if !flagQuiet {
				_, _ = fmt.Fprintf(streams.Err, "\n%s\n", ui.Bold("[1/2] "+fmt.Sprintf(i18n.T().Cmd.AngelaReviewStep1, len(summaries))))
			}

			// Style guide
			var styleGuideStr string
			if cfg.Angela.StyleGuide != nil {
				guide := angela.ParseStyleGuide(cfg.Angela.StyleGuide)
				styleGuideStr = angela.FormatStyleGuideRules(guide)
			}

			// Preflight: estimate tokens, warn, abort if needed
			corpusSize := 0
			for _, s := range summaries {
				corpusSize += len(s.Summary) + len(s.Filename) + 50
			}
			maxTokens := angela.ResolveMaxTokens("review", 0, cfg.Angela.MaxTokens)
			pf := angela.Preflight(strings.Repeat("x", corpusSize), "", cfg.AI.Model, maxTokens, timeout)
			if !flagQuiet {
				_, _ = fmt.Fprintf(streams.Err, "      "+ta.UIReviewPreflight+"\n",
					len(summaries), pf.EstimatedInputTokens, pf.MaxOutputTokens, pf.Timeout)

				// Cost estimate
				if pf.EstimatedCost >= 0 {
					_, _ = fmt.Fprintf(streams.Err, "      "+ta.UIEstimatedCost+"\n", pf.EstimatedCost)
				}

				for _, w := range pf.Warnings {
					_, _ = fmt.Fprintf(streams.Err, "      %s %s\n", ui.Warning("⚠"), w)
				}

				// ABORT if input > max_output
				if pf.ShouldAbort {
					_, _ = fmt.Fprintf(streams.Err, "      %s %s\n", ui.Error("✗"), pf.AbortReason)
					return fmt.Errorf("angela: review: aborted — %s", pf.AbortReason)
				}
			}

			// --- Step 2/2: Calling AI ---
			if !flagQuiet {
				_, _ = fmt.Fprintf(streams.Err, "%s\n", ui.Bold("[2/2] "+fmt.Sprintf(i18n.T().Cmd.AngelaReviewStep2, len(summaries))))
			}
			var spin *ui.Spinner
			if !flagQuiet {
				spin = ui.StartSpinnerWithTimeout(streams, fmt.Sprintf(i18n.T().Cmd.AngelaReviewStep2, len(summaries)), timeout)
			}

			// AC-1: Exactly 1 API call
			var reviewOpts []angela.ReviewOpts
			if flagFor != "" {
				reviewOpts = append(reviewOpts, angela.ReviewOpts{Audience: flagFor})
			}
			report, err := angela.Review(cmd.Context(), provider, summaries, styleGuideStr, reviewOpts...)
			if err != nil {
				if spin != nil {
					elapsed := spin.Elapsed()
					spin.Stop()
					if isTimeoutError(err) {
						_, _ = fmt.Fprintf(streams.Err, "\n      %s\n", ui.Error(fmt.Sprintf(ta.UITimeoutErr, formatElapsed(timeout), formatElapsed(elapsed))))
						_, _ = fmt.Fprintf(streams.Err, "      %s\n", ui.Dim(ta.UITimeoutHint1))
						_, _ = fmt.Fprintf(streams.Err, "      %s\n", ui.Dim(ta.UITimeoutHint2))
						return fmt.Errorf("angela: review: timeout after %s", formatElapsed(elapsed))
					}
				}
				return err
			}

			if spin != nil {
				elapsed := spin.Elapsed()
				spin.Stop()

				// Show completion with stats
				var usage *domain.AIUsage
				if tracker, ok := provider.(domain.UsageTracker); ok {
					usage = tracker.LastUsage()
				}
				if usage != nil {
					_, _ = fmt.Fprintf(streams.Err, "      ✓ %s in %s\n", i18n.T().Cmd.AngelaReviewStep2Done, formatElapsed(elapsed))
					_, _ = fmt.Fprintf(streams.Err, "      "+ta.UITokenStats+"\n",
						usage.InputTokens, usage.OutputTokens, usage.Model)

					// Post-call analysis
					analysis := angela.AnalyzeUsage(usage, elapsed, maxTokens)
					for _, line := range analysis.Lines {
						_, _ = fmt.Fprintf(streams.Err, "      %s\n", ui.Dim(line))
					}
				} else {
					_, _ = fmt.Fprintf(streams.Err, "      ✓ %s in %s\n", i18n.T().Cmd.AngelaReviewStep2Done, formatElapsed(elapsed))
				}
			}

			_, _ = fmt.Fprintln(streams.Err)

			// Format report on stdout (AC-4, AC-11)
			formatReviewReport(streams, report, totalCount, flagQuiet)

			// Save review cache for lore status integration (skip in standalone mode)
			if !standalone {
				if err := angela.SaveReviewCache(domain.LoreDir, report, totalCount); err != nil {
					_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.AngelaReviewCacheWarn+"\n", err)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&flagQuiet, "quiet", false, "Suppress human messages on stderr")
	cmd.Flags().StringVar(&flagFor, "for", "", "Adapt findings for a target audience (e.g., \"CTO\", \"équipe commerciale\", \"nouveau développeur\")")
	cmd.Flags().StringVar(&flagFilter, "filter", "", "Regex to filter documents by filename (e.g., \"commands/.*\", \".*\\.fr\\.md$\")")
	cmd.Flags().BoolVar(&flagAll, "all", false, "Review all documents (no 25+25 sampling)")

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
		// severity + relevance + title
		label := f.Severity
		if f.Relevance != "" {
			label = fmt.Sprintf("%s [%s]", f.Severity, f.Relevance)
		}
		_, _ = fmt.Fprintf(streams.Out, "  %-22s %s\n", label, f.Title)
		// documents
		if len(f.Documents) > 0 {
			_, _ = fmt.Fprintf(streams.Out, "  %22s %s\n", "", strings.Join(f.Documents, " vs "))
		}
		// description
		if f.Description != "" {
			_, _ = fmt.Fprintf(streams.Out, "  %22s %s\n", "", f.Description)
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
