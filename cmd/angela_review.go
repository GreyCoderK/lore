// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/greycoderk/lore/internal/ai"
	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/credential"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/fileutil"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/ui"
	"github.com/spf13/cobra"
)

func newAngelaReviewCmd(cfg *config.Config, streams domain.IOStreams, flagPath *string) *cobra.Command {
	var flagQuiet bool
	var flagVerbose bool
	var flagFor string
	var flagFilter string
	var flagAll bool
	var flagDiffOnly bool
	var flagInteractive bool

	cmd := &cobra.Command{
		Use:           "review",
		Short:         i18n.T().Cmd.AngelaReviewShort,
		Args:          cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Resolve docs directory: --path (standalone) or .lore/docs (normal)
			docsDir, standalone := resolveDocsDir(flagPath)
			if !standalone {
				// Check .lore/ exists
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

			// Prepare doc summaries (before provider — no point requesting API if corpus too small)
			corpusStore := newCorpusReader(docsDir, standalone)
			summaries, totalCount, err := angela.PrepareDocSummaries(corpusStore, reviewFilter)
			if err != nil {
				return err
			}

			// Instantiate provider
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

			// Exactly 1 API call.
			// Thread the evidence validation config and the corpus reader
			// through ReviewOpts so Review() can validate findings
			// against the real document content before sorting.
			opts := angela.ReviewOpts{
				Audience: flagFor,
				Reader:   corpusStore,
				Evidence: angela.EvidenceValidation{
					Required:      cfg.Angela.Review.Evidence.Required,
					MinConfidence: cfg.Angela.Review.Evidence.MinConfidence,
					Mode:          cfg.Angela.Review.Evidence.Validation,
				},
				ConfigMaxTokens: cfg.Angela.MaxTokens,
			}
			report, err := angela.Review(cmd.Context(), provider, summaries, styleGuideStr, opts)
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

			// Differential review. Load the persisted state, classify the
			// current findings as NEW / PERSISTING / REGRESSED / RESOLVED,
			// update the state, and save it. The state file lives under
			// ResolveStateDir(...) so it works in both lore-native and
			// standalone modes. Failures are logged as warnings — a
			// broken state file must never block a review. A corrupt
			// state file is quarantined before being overwritten so a
			// single bad write does not destroy lifecycle marks.
			// Hoisted out of the conditional so the interactive TUI can
			// access prev and statePath for resolve/ignore actions.
			var reviewStatePath string
			var reviewState *angela.ReviewState
			if cfg.Angela.Review.Differential.Enabled {
				workDir, wderr := os.Getwd()
				if wderr != nil {
					return fmt.Errorf("angela: review: cwd: %w", wderr)
				}
				stateDir := config.ResolveStateDir(workDir, cfg, cfg.DetectedMode)
				stateFile := cfg.Angela.Review.Differential.StateFile
				if stateFile == "" {
					stateFile = "review-state.json"
				}
				// Reject escapes in state_file path.
				if err := angela.AssertContainedRelPath(stateFile); err != nil {
					return fmt.Errorf("angela: review: state_file: %w", err)
				}
				reviewStatePath = filepath.Join(stateDir, stateFile)

				// Serialize concurrent runs over the review state file.
				lock, lockErr := fileutil.NewFileLock(reviewStatePath)
				if lockErr != nil {
					return fmt.Errorf("angela: review: state lock: %w", lockErr)
				}
				defer lock.Unlock()

				prev, loadErr := angela.LoadReviewState(reviewStatePath)
				if loadErr != nil {
					if errors.Is(loadErr, angela.ErrStateCorrupt) {
						quarPath, qerr := angela.QuarantineCorruptState(reviewStatePath)
						if qerr != nil {
							return fmt.Errorf("angela: review: corrupt state at %s and cannot quarantine: %w", reviewStatePath, qerr)
						}
						if !flagQuiet {
							fmt.Fprintf(streams.Err, "review: state file was corrupt; quarantined at %s\n", quarPath)
						}
					} else if !flagQuiet {
						fmt.Fprintf(streams.Err, "review: %v (continuing with fresh state)\n", loadErr)
					}
				}
				reviewState = prev

				// Pass the validator's rejected findings into the diff
				// so they are not classified as natural RESOLVED.
				diff := angela.ComputeReviewDiffWithRejected(prev, report.Findings, report.Rejected, flagFor)
				report.Diff = &diff

				angela.UpdateReviewState(prev, diff, time.Now().UTC())
				if saveErr := angela.SaveReviewState(reviewStatePath, prev); saveErr != nil {
					fmt.Fprintf(streams.Err, "review: state save warning: %v\n", saveErr)
				}
			}

			// Interactive TUI mode.
			if flagInteractive {
				if !angela.IsInteractiveAvailable() {
					_, _ = fmt.Fprintf(streams.Err, "%s", i18n.T().UI.InteractiveFallback)
				} else if len(report.Findings) > 0 {
					model := angela.NewReviewInteractiveModel(
						report.Findings,
						reviewState,
						reviewStatePath,
						flagFor,
						provider,
						corpusStore,
					)
					p := tea.NewProgram(model, tea.WithAltScreen())
					finalModel, tuiErr := p.Run()
					if tuiErr != nil {
						return fmt.Errorf("angela: review: interactive: %w", tuiErr)
					}
					if m, ok := finalModel.(angela.ReviewInteractiveModel); ok && m.QuitSummary != "" {
						_, _ = fmt.Fprintf(streams.Err, "%s\n", m.QuitSummary)
					}
					// Save review cache after interactive session
					if !standalone {
						if err := angela.SaveReviewCache(domain.LoreDir, report, totalCount); err != nil {
							_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.AngelaReviewCacheWarn+"\n", err)
						}
					}
					return nil
				}
			}

			// Format report on stdout
			formatReviewReport(streams, report, totalCount, flagQuiet, flagVerbose, flagDiffOnly)

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
	cmd.Flags().BoolVarP(&flagVerbose, "verbose", "v", false, "Print detailed rejection reasons for findings dropped by the evidence validator")
	cmd.Flags().StringVar(&flagFor, "for", "", "Adapt findings for a target audience (e.g., \"CTO\", \"équipe commerciale\", \"nouveau développeur\")")
	cmd.Flags().StringVar(&flagFilter, "filter", "", "Regex to filter documents by filename (e.g., \"commands/.*\", \".*\\.fr\\.md$\")")
	cmd.Flags().BoolVar(&flagAll, "all", false, "Review all documents (no 25+25 sampling)")
	// Differential review.
	cmd.Flags().BoolVar(&flagDiffOnly, "diff-only", false, "Show only NEW + REGRESSED findings (and counts of PERSISTING/RESOLVED). Ideal for CI.")
	// Interactive TUI mode.
	cmd.Flags().BoolVarP(&flagInteractive, "interactive", "i", false, "Launch interactive TUI to navigate and triage findings")

	// Lifecycle subcommands.
	cmd.AddCommand(newAngelaReviewResolveCmd(cfg, streams))
	cmd.AddCommand(newAngelaReviewIgnoreCmd(cfg, streams))
	cmd.AddCommand(newAngelaReviewLogCmd(cfg, streams))

	return cmd
}

// formatReviewReport writes the review report to stdout and human messages
// to stderr. When verbose is true, findings pulled by the evidence
// validator are listed with their rejection reason below the severity
// summary. Without verbose, only the count is shown.
//
// When diffOnly is true, only NEW and REGRESSED findings are shown in
// the per-finding listing — PERSISTING entries are hidden from the row
// list (their count still appears in the diff summary line). Each
// finding's title is also prefixed with a status marker ('+', '=', '!')
// so the user can read the lifecycle at a glance.
func formatReviewReport(streams domain.IOStreams, report *angela.ReviewReport, totalCorpus int, quiet, verbose, diffOnly bool) {
	// Header on stderr (unless quiet)
	if !quiet {
		if totalCorpus > report.DocCount {
			_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.AngelaReviewHdrPartial+"\n\n", report.DocCount, totalCorpus)
		} else {
			_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.AngelaReviewHdrFull+"\n\n", report.DocCount)
		}
	}

	// Findings on stdout (always). Filter PERSISTING when
	// diffOnly is set, and tag each kept finding with its diff marker.
	for _, f := range report.Findings {
		if diffOnly && f.DiffStatus == angela.ReviewDiffPersisting {
			continue
		}
		marker := " "
		switch f.DiffStatus {
		case angela.ReviewDiffNew:
			marker = "+"
		case angela.ReviewDiffPersisting:
			marker = "="
		case angela.ReviewDiffRegressed:
			marker = "!"
		}
		// severity + relevance + title
		label := f.Severity
		if f.Relevance != "" {
			label = fmt.Sprintf("%s [%s]", f.Severity, f.Relevance)
		}
		_, _ = fmt.Fprintf(streams.Out, " %s %-22s %s\n", marker, label, f.Title)
		// hash short form so the user can copy-paste it into resolve/ignore
		if f.Hash != "" {
			_, _ = fmt.Fprintf(streams.Out, " %s %22s [%s]\n", " ", "", f.Hash[:6])
		}
		// documents
		if len(f.Documents) > 0 {
			_, _ = fmt.Fprintf(streams.Out, " %s %22s %s\n", " ", "", strings.Join(f.Documents, " vs "))
		}
		// description
		if f.Description != "" {
			_, _ = fmt.Fprintf(streams.Out, " %s %22s %s\n", " ", "", f.Description)
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

	// Rejected findings visibility.
	// Always print the count on stderr (unless quiet); in verbose mode,
	// itemise each rejection with its reason so the user can debug the AI
	// prompt or confirm that the validator caught a real hallucination.
	if !quiet && len(report.Rejected) > 0 {
		_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.AngelaReviewRejectedCount+"\n", len(report.Rejected))
		if verbose {
			for _, r := range report.Rejected {
				_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.AngelaReviewRejectedLine+"\n", r.Finding.Title, r.Reason)
			}
		}
	}

	// Differential summary line. Only printed when the
	// runner attached a Diff to the report (differential mode active).
	// Resolved findings (the natural ones — corpus changed and finding
	// went away) get listed in verbose mode so the user can see what
	// cleared up.
	if !quiet && report.Diff != nil {
		newC, persC, regC, resC := report.Diff.Counts()
		_, _ = fmt.Fprintf(streams.Err, i18n.T().Angela.ReviewDiffSummary,
			newC, persC, regC, resC)
		if verbose && len(report.Diff.Resolved) > 0 {
			_, _ = fmt.Fprintf(streams.Err, "%s", i18n.T().Angela.ReviewDiffResolved)
			for _, r := range report.Diff.Resolved {
				_, _ = fmt.Fprintf(streams.Err, "  - %-22s %s\n", r.Severity, r.Title)
			}
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
