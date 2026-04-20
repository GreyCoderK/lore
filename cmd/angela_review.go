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
	var flagSynthesizers []string
	var flagNoSynthesizers bool

	// Persona opt-in flags. Mutually exclusive.
	// - flagPersonaNames: --persona repeatable; activates listed personas.
	// - flagNoPersonas: --no-personas; hard-disable even if .lorerc configures them.
	// - flagUseConfiguredPersonas: --use-configured-personas; skip interactive
	//   prompt and use the configured list directly (useful for TTY scripts).
	var flagPersonaNames []string
	var flagNoPersonas bool
	var flagUseConfiguredPersonas bool

	// --preview prints a cost estimate + planned personas then exits without
	// any API call. --format selects text (default) or json for the preview
	// report. The flags are scoped to --preview for now; the full --format
	// flow on review itself is a later follow-up.
	var flagPreview bool
	var flagFormat string

	cmd := &cobra.Command{
		Use:           "review",
		Short:         i18n.T().Cmd.AngelaReviewShort,
		Args:          cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			applySynthesizerFlags(cfg, flagSynthesizers, flagNoSynthesizers, cmd.Flags().Changed("synthesizers"))

			// --preview / --interactive mutual exclusion is declared via
			// cmd.MarkFlagsMutuallyExclusive below; cobra raises the error
			// at flag-parse time before RunE runs.

			// --format is preview-only. Silently ignoring it hid CI mistakes
			// where an operator expected JSON for machine parsing but got the
			// regular text report instead. Fail loud so the mistake is caught
			// at review time, not by a downstream parser that blows up on the
			// first line.
			if cmd.Flags().Changed("format") && !flagPreview {
				return fmt.Errorf("%s", i18n.T().Cmd.AngelaReviewErrFormatRequiresPreview)
			}

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

			// Preview short-circuit. Runs Preflight locally and exits with status
			// 0 WITHOUT instantiating the AI provider (zero-HTTP contract) and
			// without writing any state (no-side-effect contract). Personas are
			// resolved here too so the preview cost reflects the real payload
			// a non-preview run would send.
			if flagPreview {
				personasForPreview, perr := resolveReviewPersonasForPreview(
					cfg, flagPersonaNames, flagNoPersonas, flagUseConfiguredPersonas,
				)
				if perr != nil {
					return perr
				}
				corpusBytes := 0
				for _, s := range summaries {
					corpusBytes += len(s.Summary) + len(s.Filename) + 50
				}
				previewTimeout := cfg.AI.Timeout
				if previewTimeout <= 0 {
					previewTimeout = 60 * time.Second
				}
				// Resolve style guide upstream so the preview prompt matches
				// what the real review would produce.
				var previewStyleGuide string
				if cfg.Angela.StyleGuide != nil {
					guide := angela.ParseStyleGuide(cfg.Angela.StyleGuide)
					previewStyleGuide = angela.FormatStyleGuideRules(guide)
				}
				previewInputs := reviewPreviewInputs{
					Summaries:      summaries,
					StyleGuide:     previewStyleGuide,
					VHSSignals:     nil, // VHS probe is cheap but off the hot preview path
					CorpusBytes:    corpusBytes,
					CorpusDocCount: totalCount,
					Model:          cfg.AI.Model,
					MaxTokens:      angela.ResolveMaxTokens("review", 0, cfg.Angela.MaxTokens),
					Timeout:        previewTimeout,
					Personas:       personasForPreview,
					Audience:       flagFor,
					Format:         flagFormat,
				}
				return runReviewPreview(streams, cfg, previewInputs)
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

			// ─────────────────────────────────────────────────────────────
			// Resolve persona opt-in BEFORE preflight + step 1/2 so the
			// activated persona list feeds into the cost estimate shown on
			// the preflight line (and into --preview output).
			// ─────────────────────────────────────────────────────────────
			personaDecision, err := decideReviewPersonas(
				cfg, flagPersonaNames, flagNoPersonas, flagUseConfiguredPersonas,
				ui.IsTerminal(streams),
			)
			if err != nil {
				return err
			}
			var activePersonas []angela.PersonaProfile
			switch personaDecision.Resolution {
			case personaFromFlag, personaFromConfig:
				activePersonas = personaDecision.Personas
			case personaPromptRequired:
				// Compute corpus bytes once for the preflight calls inside the prompt.
				corpusBytesForPrompt := 0
				for _, s := range summaries {
					corpusBytesForPrompt += len(s.Summary) + len(s.Filename) + 50
				}
				timeoutForPrompt := cfg.AI.Timeout
				if timeoutForPrompt <= 0 {
					timeoutForPrompt = 60 * time.Second
				}
				ok, perr := promptPersonaConfirmation(streams, personaPromptInputs{
					CorpusBytes: corpusBytesForPrompt,
					Model:       cfg.AI.Model,
					MaxTokens:   angela.ResolveMaxTokens("review", 0, cfg.Angela.MaxTokens),
					Timeout:     timeoutForPrompt,
					Candidates:  personaDecision.Candidates,
				})
				if perr != nil {
					return perr
				}
				if ok {
					profiles, unknown := resolvePersonaNames(personaDecision.Candidates)
					if len(unknown) > 0 {
						return fmt.Errorf(i18n.T().Cmd.AngelaReviewErrUnknownConfiguredPersona, strings.Join(unknown, ", "))
					}
					activePersonas = profiles
				}
			case personaNonTTYInfo:
				if !flagQuiet {
					renderNonTTYPersonaInfo(streams, personaDecision.Candidates)
				}
			case personaBaseline:
				// nothing to do
			}

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
			evidence := angela.EvidenceValidation{
				Required:      cfg.Angela.Review.Evidence.Required,
				MinConfidence: cfg.Angela.Review.Evidence.MinConfidence,
				Mode:          cfg.Angela.Review.Evidence.Validation,
			}
			// CRITICAL invariant I4 enforcement: persona-attributed findings
			// must pass through the evidence validator. If the user opts into
			// personas but has not enabled evidence validation in .lorerc, a
			// default .lorerc would let hallucinated persona-attributed
			// findings through. Force the validator ON with strict mode
			// whenever personas are active, regardless of config defaults.
			// The user can still loosen MinConfidence via config.
			if len(activePersonas) > 0 {
				evidence.Required = true
				if strings.EqualFold(evidence.Mode, angela.EvidenceModeOff) || evidence.Mode == "" {
					evidence.Mode = angela.EvidenceModeStrict
				}
			}
			opts := angela.ReviewOpts{
				Audience:        flagFor,
				Reader:           corpusStore,
				Evidence:         evidence,
				ConfigMaxTokens:  cfg.Angela.MaxTokens,
				Personas:         activePersonas, // nil unless user opted in
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

	tc := i18n.T().Cmd
	cmd.Flags().BoolVar(&flagQuiet, "quiet", false, tc.AngelaReviewFlagQuiet)
	cmd.Flags().BoolVarP(&flagVerbose, "verbose", "v", false, tc.AngelaReviewFlagVerbose)
	cmd.Flags().StringVar(&flagFor, "for", "", tc.AngelaReviewFlagFor)
	cmd.Flags().StringVar(&flagFilter, "filter", "", tc.AngelaReviewFlagFilter)
	cmd.Flags().BoolVar(&flagAll, "all", false, tc.AngelaReviewFlagAll)
	// Differential review.
	cmd.Flags().BoolVar(&flagDiffOnly, "diff-only", false, tc.AngelaReviewFlagDiffOnly)
	// Interactive TUI mode.
	cmd.Flags().BoolVarP(&flagInteractive, "interactive", "i", false, tc.AngelaReviewFlagInteractive)
	cmd.Flags().StringSliceVar(&flagSynthesizers, "synthesizers", nil, tc.AngelaReviewFlagSynthesizers)
	cmd.Flags().BoolVar(&flagNoSynthesizers, "no-synthesizers", false, tc.AngelaReviewFlagNoSynthesizers)
	// Persona opt-in flags (mutually exclusive).
	cmd.Flags().StringSliceVar(&flagPersonaNames, "persona", nil, tc.AngelaReviewFlagPersona)
	cmd.Flags().BoolVar(&flagNoPersonas, "no-personas", false, tc.AngelaReviewFlagNoPersonas)
	cmd.Flags().BoolVar(&flagUseConfiguredPersonas, "use-configured-personas", false, tc.AngelaReviewFlagUseConfiguredPersonas)
	// Preview flags.
	cmd.Flags().BoolVar(&flagPreview, "preview", false, tc.AngelaReviewFlagPreview)
	cmd.Flags().StringVar(&flagFormat, "format", "", tc.AngelaReviewFlagFormat)

	_ = cmd.RegisterFlagCompletionFunc("synthesizers", synthesizerFlagCompletion)
	// Persona name completion on --persona (matches draft/polish).
	_ = cmd.RegisterFlagCompletionFunc("persona", personaFlagCompletion)
	// Fixed completion set for --format (text | json).
	_ = cmd.RegisterFlagCompletionFunc("format", textJSONFormatFlagCompletion)
	// Mutual exclusions declared declaratively so a refactor cannot silently
	// drop them. Cobra enforces these at flag-parse time, before RunE runs.
	cmd.MarkFlagsMutuallyExclusive("persona", "no-personas")
	cmd.MarkFlagsMutuallyExclusive("persona", "use-configured-personas")
	cmd.MarkFlagsMutuallyExclusive("no-personas", "use-configured-personas")
	cmd.MarkFlagsMutuallyExclusive("preview", "interactive")

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
		// When any finding is persona-attributed, surface the active persona
		// lenses up-front so the reader understands the review angle before
		// scanning findings. The set is built from the union of Personas
		// across findings (so the order is stable even if configured personas
		// returned zero findings for this run).
		if active := activePersonasInReport(report); len(active) > 0 {
			ta := i18n.T().Angela
			_, _ = fmt.Fprintf(streams.Err, ta.UIReviewAngleHeader,
				len(active), pluralS(len(active)))
			for _, p := range active {
				_, _ = fmt.Fprintf(streams.Err, ta.UIReviewAnglePersonaRow, p.Icon, p.DisplayName, p.Expertise)
			}
			_, _ = fmt.Fprintln(streams.Err)
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
		// Persona attribution line per finding so the reader sees which lens
		// flagged the issue without leaving the text report. Resolved names
		// carry Icon + DisplayName; unknown names fall back to the raw
		// identifier.
		if len(f.Personas) > 0 {
			_, _ = fmt.Fprintf(streams.Out, " %s %22s %s\n",
				" ", "", fmt.Sprintf(i18n.T().Angela.UIReviewFlaggedBy, formatFlaggedByLine(f.Personas)))
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
