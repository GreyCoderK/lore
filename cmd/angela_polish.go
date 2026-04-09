// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/greycoderk/lore/internal/ai"
	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/credential"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/service"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/greycoderk/lore/internal/ui"
	"github.com/spf13/cobra"
)

func newAngelaPolishCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	var flagDryRun bool
	var flagYes bool
	var flagFor string
	var flagAuto bool

	cmd := &cobra.Command{
		Use:          "polish <filename>",
		Short:        i18n.T().Cmd.AngelaPolishShort,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			filename := args[0]

			// AC-9: Check .lore/ exists
			if err := requireLoreDir(streams); err != nil {
				return err
			}
			docsDir := filepath.Join(domain.LoreDir, domain.DocsDir)

			// AC-8: Validate filename and check exists
			if err := storage.ValidateFilename(filename); err != nil {
				return fmt.Errorf("angela: polish: %w", err)
			}
			docPath := filepath.Join(docsDir, filename)
			if _, err := os.Stat(docPath); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf(i18n.T().Cmd.AngelaPolishNotFound, filename)
				}
				return fmt.Errorf("angela: polish: %w", err)
			}

			// AC-4: Instantiate provider — nil = no provider
			store := credential.NewStore()
			provider, err := ai.NewProvider(cfg, store, streams.Err)
			if err != nil {
				return err
			}
			if provider == nil {
				return fmt.Errorf("%s", i18n.T().Cmd.AngelaPolishNoProvider)
			}

			ta := i18n.T().Angela

			// Cap audience length to prevent prompt bloat
			if len(flagFor) > 200 {
				flagFor = flagFor[:200]
			}

			// Show mode
			if flagFor != "" {
				_, _ = fmt.Fprintf(streams.Err, "\n%s\n", ui.Bold(fmt.Sprintf(ta.UIMode, flagFor)))
			}

			// --- Step 1/3: Preparing ---
			_, _ = fmt.Fprintf(streams.Err, "\n%s\n", ui.Bold("[1/3] "+fmt.Sprintf(i18n.T().Cmd.AngelaPolishStep1, filename)))

			// Pre-flight check
			timeout := cfg.AI.Timeout
			if timeout <= 0 {
				timeout = 60 * time.Second
			}
			raw, readErr := os.ReadFile(docPath)
			if readErr == nil {
				docWords := len(strings.Fields(string(raw)))
				maxTokens := angela.ResolveMaxTokens("polish", docWords, cfg.Angela.MaxTokens)
				pf := angela.Preflight(string(raw), "", cfg.AI.Model, maxTokens, timeout)
				_, _ = fmt.Fprintf(streams.Err, "      "+ta.UITokenEstimate+"\n",
					pf.EstimatedInputTokens, pf.MaxOutputTokens, pf.Timeout)

				// Personas
				docMeta, _, _ := storage.Unmarshal(raw)
				scored := angela.ResolvePersonasForAudience(docMeta.Type, string(raw), flagFor)
				_, _ = fmt.Fprintf(streams.Err, "      "+ta.UIPersonas+"\n", angela.DescribePersonas(scored))

				// Quality
				scoreBefore := angela.ScoreDocument(string(raw), docMeta)
				_, _ = fmt.Fprintf(streams.Err, "      "+ta.UIQuality+"\n", angela.FormatScore(scoreBefore))

				// Cost estimate
				if pf.EstimatedCost >= 0 {
					_, _ = fmt.Fprintf(streams.Err, "      "+ta.UIEstimatedCost+"\n", pf.EstimatedCost)
				}

				if angela.ShouldMultiPass(docWords) {
					sections := angela.SplitSections(string(raw))
					_, _ = fmt.Fprintf(streams.Err, "      "+ta.UIMultiPass+"\n", len(sections)-1)
				}

				for _, w := range pf.Warnings {
					_, _ = fmt.Fprintf(streams.Err, "      %s %s\n", ui.Warning("⚠"), w)
				}

				// ABORT if input > max_output (will certainly truncate)
				if pf.ShouldAbort {
					_, _ = fmt.Fprintf(streams.Err, "      %s %s\n", ui.Error("✗"), pf.AbortReason)
					for _, w := range pf.Warnings {
						_, _ = fmt.Fprintf(streams.Err, "      %s\n", ui.Dim(w))
					}
					return fmt.Errorf("angela: polish: aborted — %s", pf.AbortReason)
				}
			}

			// --- Step 2/3: Calling AI ---
			_, _ = fmt.Fprintf(streams.Err, "%s\n", ui.Bold("[2/3] "+fmt.Sprintf(i18n.T().Cmd.AngelaPolishStep2, filename)))
			spin := ui.StartSpinnerWithTimeout(streams, fmt.Sprintf(i18n.T().Cmd.AngelaPolishStep2, filename), timeout)
			var polishOpts []service.PolishOptions
			if flagFor != "" {
				polishOpts = append(polishOpts, service.PolishOptions{Audience: flagFor})
			}
			result, err := service.PolishDocument(cmd.Context(), provider, cfg, docsDir, filename, polishOpts...)
			if err != nil {
				elapsed := spin.Elapsed()
				spin.Stop()
				if isTimeoutError(err) {
					_, _ = fmt.Fprintf(streams.Err, "\n      %s\n", ui.Error(fmt.Sprintf(ta.UITimeoutErr, formatElapsed(timeout), formatElapsed(elapsed))))
					_, _ = fmt.Fprintf(streams.Err, "      %s\n", ui.Dim(ta.UITimeoutHint1))
					_, _ = fmt.Fprintf(streams.Err, "      %s\n", ui.Dim(ta.UITimeoutHint2))
					return fmt.Errorf("angela: polish: timeout after %s", formatElapsed(elapsed))
				}
				return err
			}
			elapsed := spin.Elapsed()
			spin.Stop()

			if result == nil {
				return fmt.Errorf("angela: polish: no result from AI provider")
			}

			// Show completion with stats
			var usage *domain.AIUsage
			if tracker, ok := provider.(domain.UsageTracker); ok {
				usage = tracker.LastUsage()
			}
			if usage != nil {
				_, _ = fmt.Fprintf(streams.Err, "      ✓ %s in %s\n", i18n.T().Cmd.AngelaPolishStep2Done, formatElapsed(elapsed))
				_, _ = fmt.Fprintf(streams.Err, "      "+ta.UITokenStats+"\n",
					usage.InputTokens, usage.OutputTokens, usage.Model)

				// Post-call analysis
				docWords := len(strings.Fields(result.Original))
				maxTokens := angela.ResolveMaxTokens("polish", docWords, cfg.Angela.MaxTokens)
				analysis := angela.AnalyzeUsage(usage, elapsed, maxTokens)
				for _, line := range analysis.Lines {
					_, _ = fmt.Fprintf(streams.Err, "      %s\n", ui.Dim(line))
				}

				// Truncation guard
				if usage.OutputTokens >= maxTokens-10 {
					_, _ = fmt.Fprintf(streams.Err, "\n      %s\n", ui.Error(fmt.Sprintf(ta.UITruncated, usage.OutputTokens, maxTokens)))
					_, _ = fmt.Fprintf(streams.Err, "      %s\n", ui.Dim(ta.UITruncatedHint))
					return nil
				}
			} else {
				_, _ = fmt.Fprintf(streams.Err, "      ✓ %s in %s\n", i18n.T().Cmd.AngelaPolishStep2Done, formatElapsed(elapsed))
			}

			// --for mode: ask user whether to create new file or overwrite
			if flagFor != "" {
				_, _ = fmt.Fprintf(streams.Err, "\n      %s\n", fmt.Sprintf(ta.UIForPrompt, flagFor))
				_, _ = fmt.Fprint(streams.Err, "      "+ta.UIForNewFile)
				scanner := bufio.NewScanner(streams.In)
				overwrite := false
				if scanner.Scan() {
					input := strings.TrimSpace(strings.ToLower(scanner.Text()))
					overwrite = input == "o" || input == "overwrite" || input == "é" || input == "écraser"
				}
				if overwrite {
					_, _ = fmt.Fprintf(streams.Err, "      %s\n", ta.UIForOverwrite)
					// Fall through to interactive diff on original
				} else {
					outName := strings.TrimSuffix(filename, ".md") + "." + sanitizeAudience(flagFor) + ".md"
					outPath := filepath.Join(docsDir, outName)
					if err := os.WriteFile(outPath, []byte(result.Polished), 0644); err != nil {
						return fmt.Errorf("angela: polish: write %s: %w", outName, err)
					}
					_, _ = fmt.Fprintf(streams.Err, "\n      "+ta.UIRewrittenFor+"\n", flagFor, outName)
					_, _ = fmt.Fprintf(streams.Err, "      %s\n", ui.Dim(ta.UIOriginalUnchanged))
					return nil
				}
			}

			// --- Step 3/3: Computing diff ---
			_, _ = fmt.Fprintf(streams.Err, "%s\n", ui.Bold("[3/3] "+i18n.T().Cmd.AngelaPolishStep3))

			originalContent := result.Original
			meta := result.Meta
			hunks := result.Diff

			// Quality score after (if all changes accepted)
			scoreAfter := angela.ScoreDocument(result.Polished, meta)
			scoreBefore := angela.ScoreDocument(originalContent, meta)
			delta := scoreAfter.Total - scoreBefore.Total
			deltaStr := fmt.Sprintf("%+d", delta)
			if delta > 0 {
				deltaStr = ui.Success(deltaStr)
			} else if delta < 0 {
				deltaStr = ui.Error(deltaStr)
			}
			_, _ = fmt.Fprintf(streams.Err, "      "+ta.UIChangesQuality+"\n\n",
				len(hunks), angela.FormatScore(scoreBefore), angela.FormatScore(scoreAfter), deltaStr)

			// AC-10: No changes
			if len(hunks) == 0 {
				_, _ = fmt.Fprintf(streams.Err, "%s\n", i18n.T().Cmd.AngelaPolishNoChanges)
				return nil
			}

			// AC-2: Interactive diff / AC-7: reject all
			choices, err := angela.InteractiveDiff(hunks, streams, angela.DiffOptions{DryRun: flagDryRun, YesAll: flagYes, Auto: flagAuto})
			if err != nil {
				return fmt.Errorf("angela: polish: %w", err)
			}

			if flagDryRun {
				return nil
			}

			// Check if any accepted or both
			anyChosen := false
			for _, c := range choices {
				if c != angela.DiffReject {
					anyChosen = true
					break
				}
			}

			// AC-7: All rejected
			if !anyChosen {
				_, _ = fmt.Fprintf(streams.Err, "%s\n", i18n.T().Cmd.AngelaPolishNoneApplied)
				return nil
			}

			// TOCTOU guard: check file hasn't changed during interactive review
			currentRaw, err := os.ReadFile(docPath)
			if err != nil {
				return fmt.Errorf("angela: polish: re-read: %w", err)
			}
			if !bytes.Equal(currentRaw, []byte(originalContent)) {
				return fmt.Errorf("angela: polish: %s", i18n.T().Cmd.AngelaPolishModified)
			}

			// AC-3: Apply changes
			applied := angela.ApplyDiff(originalContent, hunks, choices)

			// Validate and update angela_mode in front matter
			resultMeta, resultBody, err := storage.Unmarshal([]byte(applied))
			if err != nil {
				// If parsing fails, use the result as-is with original meta
				resultMeta = meta
				resultBody = applied
			}
			resultMeta.AngelaMode = "polish"

			// Validate the resulting frontmatter before writing to disk
			if valErr := storage.ValidateMeta(resultMeta); valErr != nil {
				return fmt.Errorf("angela: polish: AI response produced invalid frontmatter: %w", valErr)
			}

			// Write atomically
			marshaled, err := storage.Marshal(resultMeta, resultBody)
			if err != nil {
				return fmt.Errorf("angela: polish: marshal: %w", err)
			}
			if err := storage.AtomicWrite(docPath, marshaled); err != nil {
				return fmt.Errorf("angela: polish: write: %w", err)
			}

			// Regenerate index
			if err := storage.RegenerateIndex(docsDir); err != nil {
				_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.AngelaPolishIndexWarn+"\n", err)
			}

			_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.AngelaPolishVerb+"\n", filename)
			return nil
		},
	}

	cmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Show diff without applying changes")
	cmd.Flags().BoolVar(&flagYes, "yes", false, "Accept all changes without confirmation")
	cmd.Flags().StringVar(&flagFor, "for", "", "Rewrite document for a target audience (e.g., \"équipe commerciale\", \"CTO\", \"nouveau développeur\")")
	cmd.Flags().BoolVarP(&flagAuto, "auto", "a", false, "Auto-accept additions, auto-reject deletions, ask only for modifications")

	return cmd
}

// sanitizeAudience converts an audience string to a safe filename slug.
// Preserves Unicode letters (accented characters) for French and other languages.
func sanitizeAudience(audience string) string {
	s := strings.ToLower(audience)
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return '-'
	}, s)
	// Collapse multiple dashes
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")
	if s == "" {
		s = "audience"
	}
	// Truncate to avoid filesystem limits
	if len(s) > 50 {
		s = s[:50]
	}
	return s
}

// formatElapsed returns a compact elapsed time string.
func formatElapsed(d interface{ Seconds() float64 }) string {
	totalSec := d.Seconds()
	if totalSec < 60 {
		return fmt.Sprintf("%.1fs", totalSec)
	}
	m := int(totalSec) / 60
	s := int(totalSec) % 60
	return fmt.Sprintf("%dm%ds", m, s)
}

// isTimeoutError checks if an error is a context deadline exceeded (timeout).
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "context deadline exceeded") ||
		strings.Contains(msg, "Client.Timeout")
}
