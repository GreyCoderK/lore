// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
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

// backupDisabledAckFilename is the name of the acknowledgement marker
// that polish drops under stateDir the first time the user runs with
// backup disabled. Its presence suppresses the warning on subsequent
// runs. Exported as a constant so tests can assert on the exact path.
//
// Previously a package-level sync.Once gated the warning, which (a)
// reset on every CLI invocation since lore runs as a fresh process,
// and (b) would fire at most once per process in a daemon. The marker
// file persists across CLI invocations, matches the "first time"
// intent, and is trivially resettable by tests via t.TempDir() as
// the state root.
const backupDisabledAckFilename = ".backup-disabled-acked"

// emitBackupDisabledWarning prints the disabled-backups warning to
// stderr unless an ack marker already exists in stateDir. On first
// call for a given stateDir it creates the marker. Marker creation
// errors are non-fatal: a failure just means the user will see the
// warning again on the next run, which is the safer default.
func emitBackupDisabledWarning(stateDir string, streams domain.IOStreams) {
	markerPath := filepath.Join(stateDir, backupDisabledAckFilename)
	if _, err := os.Stat(markerPath); err == nil {
		return // already acknowledged
	}
	_, _ = fmt.Fprintf(streams.Err, "      %s\n", ui.Warning(i18n.T().Cmd.AngelaPolishBackupDisabled))
	if err := os.MkdirAll(stateDir, 0o700); err == nil {
		// Best-effort touch. Ignore errors — the next run will warn
		// again, which is the fail-safe direction.
		if f, ferr := os.OpenFile(markerPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600); ferr == nil {
			_ = f.Close()
		}
	}
}

// polishProviderFactory is the seam that lets tests inject a fake
// domain.AIProvider without going through the real ai.NewProvider path
// (which requires keychain credentials, network endpoints, etc.). Tests
// set this to return their mock, then restore it with the returned
// closure. Production code leaves it at its nil default and falls back
// to ai.NewProvider.
//
// The swap is protected by polishProviderFactoryMu so a future
// t.Parallel() in the cmd package does not race.
var (
	polishProviderFactoryMu sync.RWMutex
	polishProviderFactory   func(cfg *config.Config, streams domain.IOStreams) (domain.AIProvider, error)
)

// setPolishProviderFactory installs a test-only factory and returns a
// restore function. Keep this in the same file as the hook so the
// package-private name stays visible to the test file without exporting
// it to the broader API.
func setPolishProviderFactory(f func(cfg *config.Config, streams domain.IOStreams) (domain.AIProvider, error)) func() {
	polishProviderFactoryMu.Lock()
	prev := polishProviderFactory
	polishProviderFactory = f
	polishProviderFactoryMu.Unlock()
	return func() {
		polishProviderFactoryMu.Lock()
		polishProviderFactory = prev
		polishProviderFactoryMu.Unlock()
	}
}

// loadPolishProviderFactory reads the current factory under the read
// lock so production and tests both go through a race-safe path.
func loadPolishProviderFactory() func(cfg *config.Config, streams domain.IOStreams) (domain.AIProvider, error) {
	polishProviderFactoryMu.RLock()
	defer polishProviderFactoryMu.RUnlock()
	return polishProviderFactory
}

// appendPolishLog writes one JSONL entry to <state_dir>/polish.log.
// Story 8-21 / invariant I30: every terminal state of a polish command
// that modifies (or could have modified) a file writes exactly one log
// line. Dry-run is the one exception (AC-14: zero side-effect).
//
// Errors during append are surfaced to stderr as a non-fatal note —
// polish success is never blocked on a failed log write.
func appendPolishLog(
	streams domain.IOStreams,
	cfg *config.Config,
	filename, mode, result string,
	exit int,
	findings angela.LogFindings,
	ai *angela.LogAIInfo,
) {
	workDir, err := os.Getwd()
	if err != nil {
		_, _ = fmt.Fprintf(streams.Err, "      %s polish.log: cwd failed — %v (non-fatal)\n", ui.Dim("•"), err)
		return
	}
	stateDir := config.ResolveStateDir(workDir, cfg, cfg.DetectedMode)
	entry := angela.LogEntry{
		File:     filename,
		Mode:     mode,
		AI:       ai,
		Findings: findings,
		Result:   result,
		Exit:     exit,
	}
	if err := angela.AppendLogEntry(stateDir, entry); err != nil {
		_, _ = fmt.Fprintf(streams.Err, "      %s polish.log append failed — %v (non-fatal)\n", ui.Dim("•"), err)
	}
}

// findingsFromReport maps the angela.SanitizeReport (produced by the
// sanitize layer) into the LogFindings slice used by polish.log. The
// mapping is intentionally lossy: log readers get heading + count +
// resolution, not the full byte content of each occurrence.
func findingsFromReport(report angela.SanitizeReport) angela.LogFindings {
	var out angela.LogFindings
	if report.LeakedFM.Stripped {
		out.LeakedFM = &angela.LogLeakedFM{
			Stripped: true,
			Bytes:    report.LeakedFM.Bytes,
		}
	}
	if len(report.DupGroups) == 0 {
		return out
	}
	// Build Duplicates slice. Align with resolutions if present;
	// otherwise record Resolution as "<source>:pending" so the log
	// reflects that arbitration never completed (abort/refused paths).
	dups := make([]angela.LogDuplicate, 0, len(report.DupGroups))
	for i, g := range report.DupGroups {
		d := angela.LogDuplicate{
			Heading: g.Heading,
			Count:   len(g.Occurrences),
		}
		if i < len(report.Resolutions) {
			d.Resolution = angela.FormatResolution(report.Source, report.Resolutions[i].Choice)
		} else {
			// Abort/refused — record the source with an abort tag.
			src := report.Source
			if src == "" {
				src = "user"
			}
			d.Resolution = src + ":abort"
		}
		dups = append(dups, d)
	}
	out.Duplicates = dups
	return out
}

// polishModeForLog maps the command flags to a LogMode constant.
func polishModeForLog(incremental, interactive, dryRun bool) string {
	switch {
	case dryRun:
		return angela.LogModeDryRun
	case interactive:
		return angela.LogModeInteractive
	case incremental:
		return angela.LogModeIncremental
	default:
		return angela.LogModeFull
	}
}

func newAngelaPolishCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	var flagDryRun bool
	var flagYes bool
	var flagFor string
	var flagAuto bool
	var flagIncremental bool
	var flagFull bool
	var flagHallucinationStrictness string
	var flagForce bool
	var flagInteractive bool
	var flagSynthesizers []string
	var flagNoSynthesizers bool
	var flagSynthesizerDryRun bool
	var flagSynthesize bool
	var flagSetStatus string
	var flagPersona string
	var flagArbitrateRule string
	var flagVerbose bool

	cmd := &cobra.Command{
		Use:               "polish <filename>",
		Short:             i18n.T().Cmd.AngelaPolishShort,
		Args:              cobra.ExactArgs(1),
		SilenceUsage:      true,
		ValidArgsFunction: docsFileCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Mutual exclusion: --interactive and --dry-run.
			if flagInteractive && flagDryRun {
				return fmt.Errorf("angela: polish: --interactive and --dry-run are mutually exclusive")
			}

			// Validate --arbitrate-rule value.
			if flagArbitrateRule != "" && !angela.ValidArbitrationRule(flagArbitrateRule) {
				return fmt.Errorf("angela: polish: invalid --arbitrate-rule %q (must be first|second|both|abort)", flagArbitrateRule)
			}

			applySynthesizerFlags(cfg, flagSynthesizers, flagNoSynthesizers, cmd.Flags().Changed("synthesizers"))

			filename := args[0]

			// Check .lore/ exists
			if err := requireLoreDir(streams); err != nil {
				return err
			}
			docsDir := filepath.Join(domain.LoreDir, domain.DocsDir)

			// Validate filename and check exists
			if err := storage.ValidateFilename(filename); err != nil {
				return fmt.Errorf("angela: polish: %w", err)
			}
			docPath := filepath.Join(docsDir, filename)
			if _, err := os.Stat(docPath); err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return fmt.Errorf(i18n.T().Cmd.AngelaPolishNotFound, filename)
				}
				return fmt.Errorf("angela: polish: %w", err)
			}

			// Synthesizer dry-run — offline preview of the blocks that would
			// be proposed by synthesizers. Short-circuits
			// before the AI polish path: no provider, no credentials, no
			// mutation. This is exactly what pending_enrichment points to.
			if flagSynthesizerDryRun {
				return runSynthesizerDryRun(cfg, streams, docPath)
			}
			// Synthesizer apply — write the proposed blocks into the doc.
			// Still offline: no AI provider, no credentials. Writes via
			// storage.Marshal + atomic file replacement.
			if flagSynthesize {
				return runSynthesizerApply(cfg, streams, docPath, flagSetStatus)
			}

			// --set-status without --synthesize: just promote the status
			// field in-place. Useful when the user wants to flip
			// draft → reviewed → published without running anything else.
			// Re-running polish with a different status is always allowed:
			// the field is overwritten every time, never rejected.
			if flagSetStatus != "" {
				return runStatusSet(streams, docPath, flagSetStatus)
			}

			// Instantiate provider — nil = no provider.
			// Test hook polishProviderFactory bypasses the real
			// keychain/network path when set. Production leaves it nil.
			// Read through the lock so concurrent swaps are safe.
			var provider domain.AIProvider
			var err error
			if factory := loadPolishProviderFactory(); factory != nil {
				provider, err = factory(cfg, streams)
			} else {
				store := credential.NewStore()
				provider, err = ai.NewProvider(cfg, store, streams.Err)
			}
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

			// Pre-flight check.
			// Propagate the read error instead of silently ignoring it.
			// Without this, the preflight block (including the ABORT
			// gate) would be skipped and the full polish call would
			// proceed on a document that would certainly truncate.
			timeout := cfg.AI.Timeout
			if timeout <= 0 {
				timeout = 60 * time.Second
			}
			raw, readErr := os.ReadFile(docPath)
			if readErr != nil {
				return fmt.Errorf("angela: polish: preflight read: %w", readErr)
			}

			// Story 8-21 / I28: refuse malformed-frontmatter sources BEFORE
			// contacting the provider. No AI call, no credits consumed.
			// The neutral message points at `lore doctor` so the user can
			// diagnose or roll back via `angela polish --restore`.
			//
			// fmBytes is captured here for the post-AI reassembly stage
			// (I24 byte-identity): the bytes that go back to disk are the
			// exact same bytes that came off disk for the frontmatter
			// region. No re-serialization.
			// Resolve the incremental vs full decision once — used
			// both by the log entry's "mode" field (I30) and by the
			// service.PolishDocument call later. --full overrides
			// --incremental overrides config.
			useIncremental := cfg.Angela.Polish.Incremental.Enabled
			if flagIncremental {
				useIncremental = true
			}
			if flagFull {
				useIncremental = false
			}
			plannedMode := polishModeForLog(useIncremental, flagInteractive, flagDryRun)

			// Story 8-21 / I30: emit one polish.log entry at every terminal
			// state. Dry-run is the documented exception (AC-14: zero
			// side-effect). Defaults reflect a successful write; error
			// paths update these values before returning.
			var (
				logResult   = angela.LogResultWritten
				logExit     = 0
				logFindings angela.LogFindings
				logAI       *angela.LogAIInfo
			)
			defer func() {
				if flagDryRun {
					return
				}
				appendPolishLog(streams, cfg, filename, plannedMode, logResult, logExit, logFindings, logAI)
			}()

			srcFMBytes, _, fmErr := storage.ExtractFrontmatter(raw)
			if fmErr != nil {
				_, _ = fmt.Fprintf(streams.Err,
					"%s  %s\n",
					ui.Error("✗"), i18n.T().Cmd.AngelaPolishCorruptSource)
				_, _ = fmt.Fprintf(streams.Err,
					"      "+i18n.T().Cmd.AngelaPolishCorruptSourceHint+"\n",
					filename)
				logResult = angela.LogResultAbortedCorruptSrc
				logExit = 1
				return fmt.Errorf("angela: polish: %w", fmErr)
			}
			{
				docWords := len(strings.Fields(string(raw)))
				maxTokens := angela.ResolveMaxTokens("polish", docWords, cfg.Angela.MaxTokens)
				pf := angela.Preflight(string(raw), "", cfg.AI.Model, maxTokens, timeout)
				_, _ = fmt.Fprintf(streams.Err, "      "+ta.UITokenEstimate+"\n",
					pf.EstimatedInputTokens, pf.MaxOutputTokens, pf.Timeout)

				// Personas
				docMeta, _, _ := storage.Unmarshal(raw)
				scored := angela.ResolvePersonasForAudience(docMeta.Type, string(raw), flagFor)
				if flagPersona != "" {
					// --persona shortcut: narrow to the single chosen persona
					// so the polish prompt is fed exactly one lens. Unknown
					// names produce an empty set — caller gets no persona
					// lens rather than a silent fallback.
					scored = filterScoredToPersona(scored, flagPersona)
				}
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
			// useIncremental + plannedMode are resolved earlier (near the
			// corrupt-source gate) so they can be used for polish.log
			// entries on early-exit paths. Re-using here for service call.
			var polishOpts []service.PolishOptions
			po := service.PolishOptions{Audience: flagFor}
			if useIncremental {
				workDir, wderr := os.Getwd()
				if wderr != nil {
					return fmt.Errorf("angela: polish: cwd: %w", wderr)
				}
				stateDir := config.ResolveStateDir(workDir, cfg, cfg.DetectedMode)
				po.Incremental = true
				po.PolishStatePath = filepath.Join(stateDir, "polish-state.json")
			}
			polishOpts = append(polishOpts, po)
			result, err := service.PolishDocument(cmd.Context(), provider, cfg, docsDir, filename, polishOpts...)
			if err != nil {
				elapsed := spin.Elapsed()
				spin.Stop()
				logResult = angela.LogResultAIError
				logExit = 1
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
			// Populate logAI for polish.log from the usage tracker, if
			// the provider exposes one. Zero-value safe.
			if usage != nil {
				logAI = &angela.LogAIInfo{
					Provider:         cfg.AI.Provider,
					Model:            usage.Model,
					PromptTokens:     usage.InputTokens,
					CompletionTokens: usage.OutputTokens,
				}
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
					if err := os.WriteFile(outPath, []byte(result.Polished), 0o600); err != nil {
						return fmt.Errorf("angela: polish: write %s: %w", outName, err)
					}
					_, _ = fmt.Fprintf(streams.Err, "\n      "+ta.UIRewrittenFor+"\n", flagFor, outName)
					_, _ = fmt.Fprintf(streams.Err, "      %s\n", ui.Dim(ta.UIOriginalUnchanged))
					return nil
				}
			}

			// Hallucination check.
			halluStrictness := cfg.Angela.Polish.HallucinationCheck.Strictness
			if flagHallucinationStrictness != "" {
				halluStrictness = flagHallucinationStrictness
			}
			if halluStrictness == "" {
				halluStrictness = "warn"
			}
			if cfg.Angela.Polish.HallucinationCheck.Enabled && halluStrictness != "off" {
				corpusStore := &storage.CorpusStore{Dir: docsDir}
				allDocs, _ := corpusStore.ListDocs(domain.DocFilter{})
				corpusSummary := angela.BuildCorpusSummary(allDocs)
				hcheck := angela.CheckHallucinations(result.Original, result.Polished, corpusSummary)
				if len(hcheck.Unsupported) > 0 {
					_, _ = fmt.Fprintf(streams.Err, "\n      %s\n", ui.Warning(i18n.T().Angela.PolishHallucinationWarn))
					for _, c := range hcheck.Unsupported {
						_, _ = fmt.Fprintf(streams.Err, "        - %q (no source in original) [%s]\n", c.Text, c.Type)
					}
					if halluStrictness == "reject" && !flagForce {
						_, _ = fmt.Fprintf(streams.Err, "      %s\n", ui.Error(i18n.T().Angela.PolishHallucinationReject))
						return fmt.Errorf("angela: polish: hallucination check rejected %d unsupported claims", len(hcheck.Unsupported))
					}
					_, _ = fmt.Fprintf(streams.Err, "      %s\n", ui.Dim(i18n.T().Angela.PolishHallucinationHint))
				}
			}

			// Story 8-21 / Task 6.b: sanitize + arbitrate the AI response
			// before any display or write. In dry-run we detect findings
			// only and report on stderr (AC-14, no prompt, no side-effect).
			// Outside dry-run, SanitizeAIOutput may prompt in TTY or
			// apply the --arbitrate-rule flag in non-TTY. On
			// abort/refuse it returns a typed error we surface neutrally
			// without mentioning "credits" or "corruption".
			isTTY := ui.IsTerminal(streams)
			arbRule := angela.ArbitrationRule(flagArbitrateRule)
			arbOpts := angela.ArbitrateOptions{Verbose: flagVerbose}
			if flagDryRun {
				// Dry-run: detect only. Report findings on stderr; do
				// not alter result.Polished — the raw polished payload
				// flows to stdout untouched so piped tools see exactly
				// what the AI produced.
				_, report := angela.DetectStructuralIssues([]byte(result.Polished))
				if report.LeakedFM.Stripped && flagVerbose {
					_, _ = fmt.Fprintf(streams.Err,
						"      %s "+i18n.T().Cmd.AngelaPolishLeakedFMStripped+"\n",
						ui.Dim("•"), report.LeakedFM.Bytes, report.LeakedFM.Line)
				}
				if len(report.DupGroups) > 0 {
					_, _ = fmt.Fprintf(streams.Err,
						"      %s "+i18n.T().Cmd.AngelaPolishDryRunDuplicates+"\n",
						ui.Warning("⚠"), len(report.DupGroups))
					for _, g := range report.DupGroups {
						_, _ = fmt.Fprintf(streams.Err,
							"        "+i18n.T().Cmd.AngelaPolishDuplicateHeadingRow+"\n",
							g.Heading, len(g.Occurrences))
					}
				}
			} else {
				cleaned, report, sanErr := angela.SanitizeAIOutput([]byte(result.Polished), arbRule, isTTY, streams, arbOpts)
				// Record findings on the deferred log entry whether or
				// not arbitration succeeded.
				logFindings = findingsFromReport(report)
				if sanErr != nil {
					logResult = angela.LogResultAbortedArbitrate
					logExit = 1
					dupCount := len(report.DupGroups)
					if errors.Is(sanErr, angela.ErrArbitrateAbort) {
						// Story 8-21 P0-3: CI operators need the root cause
						// (how many duplicate groups, which rule fired) to
						// diagnose without opening polish.log. Message stays
						// neutral — no "credits" / "corruption" framing.
						_, _ = fmt.Fprintf(streams.Err,
							"%s  "+i18n.T().Cmd.AngelaPolishArbitrateAbortMsg+"\n",
							ui.Warning("✗"), dupCount)
						for _, g := range report.DupGroups {
							_, _ = fmt.Fprintf(streams.Err,
								"        "+i18n.T().Cmd.AngelaPolishDuplicateHeadingRow+"\n",
								g.Heading, len(g.Occurrences))
						}
						return sanErr
					}
					if errors.Is(sanErr, angela.ErrArbitrateRefused) {
						_, _ = fmt.Fprintf(streams.Err,
							"%s  "+i18n.T().Cmd.AngelaPolishArbitrateRefusedMsg+"\n",
							ui.Warning("⚠"), dupCount)
						for _, g := range report.DupGroups {
							_, _ = fmt.Fprintf(streams.Err,
								"        "+i18n.T().Cmd.AngelaPolishDuplicateHeadingRow+"\n",
								g.Heading, len(g.Occurrences))
						}
						_, _ = fmt.Fprintln(streams.Err,
							"      "+i18n.T().Cmd.AngelaPolishArbitrateRefusedHint)
						return sanErr
					}
					return fmt.Errorf("angela: polish: sanitize: %w", sanErr)
				}
				if report.LeakedFM.Stripped && flagVerbose {
					_, _ = fmt.Fprintf(streams.Err,
						"      %s "+i18n.T().Cmd.AngelaPolishLeakedFMStripped+"\n",
						ui.Dim("•"), report.LeakedFM.Bytes, report.LeakedFM.Line)
				}
				// Reassemble the full polished doc using the source's
				// fm bytes verbatim (I24 byte-identity) + the cleaned
				// body. Recompute hunks against the reassembly so the
				// downstream diff/display/write stages see consistent
				// content — full doc on both sides.
				full := string(srcFMBytes) + string(cleaned)
				result.Polished = full
				result.Diff = angela.ComputeDiff(result.Original, full)
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

			// Interactive section-level approval.
			if flagInteractive {
				if !angela.IsTTYAvailable() {
					_, _ = fmt.Fprintf(streams.Err, "%s", i18n.T().UI.InteractiveFallback)
				} else if len(hunks) > 0 {
					model := angela.NewPolishInteractiveModel(originalContent, result.Polished, filename)
					p := tea.NewProgram(model, tea.WithAltScreen())
					finalModel, tuiErr := p.Run()
					if tuiErr != nil {
						return fmt.Errorf("angela: polish: interactive: %w", tuiErr)
					}
					pm, ok := finalModel.(angela.PolishInteractiveModel)
					if !ok {
						return nil
					}
					if pm.QuitSummary != "" {
						_, _ = fmt.Fprintf(streams.Err, "%s\n", pm.QuitSummary)
					}
					if pm.Written && pm.FinalDoc != "" {
						// Story 8-21 / I24 guard: the interactive TUI assembles
						// its final doc from sections, which should preserve
						// the original frontmatter byte-for-byte. Verify
						// before writing — if this ever regresses we surface
						// it as a hard error rather than silently corrupting.
						finalFM, _, finalErr := storage.ExtractFrontmatter([]byte(pm.FinalDoc))
						if finalErr != nil {
							return fmt.Errorf("angela: polish: interactive final doc has invalid frontmatter: %w", finalErr)
						}
						if !bytes.Equal(finalFM, srcFMBytes) {
							return fmt.Errorf("angela: polish: I24 violation — interactive final doc frontmatter differs from source")
						}
						// Backup before write
						backupCfg := cfg.Angela.Polish.Backup
						workDir, wderr := os.Getwd()
						if wderr != nil {
							return fmt.Errorf("angela: polish: cwd: %w", wderr)
						}
						stDir := config.ResolveStateDir(workDir, cfg, cfg.DetectedMode)
						if backupCfg.Enabled {
							backupSubdir := backupCfg.Path
							if backupSubdir == "" {
								backupSubdir = "polish-backups"
							}
							backupPath, berr := angela.WriteBackup(workDir, stDir, backupSubdir, filepath.Join(docsDir, filename))
							if berr != nil {
								return fmt.Errorf("angela: polish: backup: %w", berr)
							}
							_, _ = fmt.Fprintf(streams.Err, "      %s\n", ui.Dim(fmt.Sprintf(i18n.T().Cmd.AngelaPolishBackupCreated, backupPath)))
						} else {
							emitBackupDisabledWarning(stDir, streams)
						}
						if err := storage.AtomicWrite(docPath, []byte(pm.FinalDoc)); err != nil {
							return fmt.Errorf("angela: polish: write: %w", err)
						}
						if err := storage.RegenerateIndex(docsDir); err != nil {
							_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.AngelaPolishIndexWarn+"\n", err)
						}
					}
					return nil
				}
			}

			// --dry-run is a non-interactive,
			// pipeable preview. We stream the polished content to stdout so
			// it can feed `diff`, `bat`, or a redirect, and we emit a plain
			// unified diff to stderr. Backups are explicitly NOT written in
			// dry-run mode because nothing touches the file — there
			// is nothing to recover.
			if flagDryRun {
				if _, werr := io.WriteString(streams.Out, result.Polished); werr != nil {
					return fmt.Errorf("angela: polish: write polished to stdout: %w", werr)
				}
				if len(hunks) > 0 {
					colored := ui.ColorEnabled(streams)
					diffOut, derr := angela.UnifiedDiffString(originalContent, result.Polished, angela.UnifiedDiffOptions{
						FromFile: filename + " (original)",
						ToFile:   filename + " (polished)",
						Context:  3,
						Colored:  colored,
					})
					if derr != nil {
						return fmt.Errorf("angela: polish: %w", derr)
					}
					if _, werr := io.WriteString(streams.Err, diffOut); werr != nil {
						return fmt.Errorf("angela: polish: write diff to stderr: %w", werr)
					}
				} else {
					_, _ = fmt.Fprintf(streams.Err, "%s\n", i18n.T().Cmd.AngelaPolishNoChanges)
				}
				return nil
			}

			// No changes
			if len(hunks) == 0 {
				_, _ = fmt.Fprintf(streams.Err, "%s\n", i18n.T().Cmd.AngelaPolishNoChanges)
				return nil
			}

			// Interactive diff
			choices, err := angela.InteractiveDiff(hunks, streams, angela.DiffOptions{YesAll: flagYes, Auto: flagAuto})
			if err != nil {
				return fmt.Errorf("angela: polish: %w", err)
			}

			// Check if any accepted or both
			anyChosen := false
			for _, c := range choices {
				if c != angela.DiffReject {
					anyChosen = true
					break
				}
			}

			// All rejected
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

			// Apply changes
			applied := angela.ApplyDiff(originalContent, hunks, choices)

			// Story 8-21 / I24 guard: the final bytes must preserve the
			// source's frontmatter exactly. No Unmarshal+Marshal dance
			// (that would re-serialize YAML and lose key ordering,
			// comments, and quote styles).
			//
			// The angela_mode="polish" field update the old path used to
			// perform is intentionally dropped: it would violate I24.
			// The polish.log JSONL audit trail (Task 6.d) provides a
			// richer "what was polished when" record anyway.
			appliedFM, _, fmErr := storage.ExtractFrontmatter([]byte(applied))
			if fmErr != nil {
				return fmt.Errorf("angela: polish: post-apply frontmatter invalid: %w", fmErr)
			}
			if !bytes.Equal(appliedFM, srcFMBytes) {
				return fmt.Errorf("angela: polish: I24 violation — frontmatter bytes drifted during diff apply")
			}
			marshaled := []byte(applied)
			_ = meta // still referenced earlier; compiler happy without re-use here

			// Automatic backup of the
			// *current* on-disk content before we overwrite it. The backup
			// captures the file as the user last saw it, not the polished
			// version. Skip entirely when Backup.Enabled is false, in which
			// case we emit a one-time warning so the user is aware they
			// have no safety net. If the backup itself fails we abort the
			// write — losing the user's work silently would be worse than
			// a failed polish.
			backupCfg := cfg.Angela.Polish.Backup
			workDir, wderr := os.Getwd()
			if wderr != nil {
				return fmt.Errorf("angela: polish: cwd: %w", wderr)
			}
			stateDir := config.ResolveStateDir(workDir, cfg, cfg.DetectedMode)
			if backupCfg.Enabled {
				backupSubdir := backupCfg.Path
				if backupSubdir == "" {
					backupSubdir = "polish-backups"
				}
				backupPath, berr := angela.WriteBackup(workDir, stateDir, backupSubdir, filepath.Join(docsDir, filename))
				if berr != nil {
					return fmt.Errorf("angela: polish: backup: %w", berr)
				}
				_, _ = fmt.Fprintf(streams.Err, "      %s\n", ui.Dim(fmt.Sprintf(i18n.T().Cmd.AngelaPolishBackupCreated, backupPath)))
				// Pruning runs AFTER the fresh backup is on disk so
				// a failure here cannot leave the user without any copy of
				// the document. Errors are logged, not fatal.
				if backupCfg.RetentionDays > 0 {
					backupRoot := filepath.Join(stateDir, backupSubdir)
					if perr := angela.PruneOldBackups(backupRoot, backupCfg.RetentionDays); perr != nil {
						_, _ = fmt.Fprintf(streams.Err, "      %s\n", ui.Dim(fmt.Sprintf(i18n.T().Cmd.AngelaPolishBackupPruneWarn, perr)))
					}
				}
			} else {
				// Persistent ack marker instead of a process-global
				// sync.Once. See emitBackupDisabledWarning.
				emitBackupDisabledWarning(stateDir, streams)
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

	tc := i18n.T().Cmd
	cmd.Flags().BoolVar(&flagDryRun, "dry-run", false, tc.AngelaPolishFlagDryRun)
	cmd.Flags().BoolVar(&flagYes, "yes", false, tc.AngelaPolishFlagYes)
	cmd.Flags().StringVar(&flagFor, "for", "", tc.AngelaPolishFlagFor)
	cmd.Flags().BoolVarP(&flagAuto, "auto", "a", false, tc.AngelaPolishFlagAuto)
	cmd.Flags().BoolVar(&flagIncremental, "incremental", false, tc.AngelaPolishFlagIncremental)
	cmd.Flags().BoolVar(&flagFull, "full", false, tc.AngelaPolishFlagFull)
	cmd.Flags().StringVar(&flagHallucinationStrictness, "hallucination-strictness", "", tc.AngelaPolishFlagHalluStrict)
	cmd.Flags().BoolVar(&flagForce, "force", false, tc.AngelaPolishFlagForce)
	// Interactive section-level polish.
	cmd.Flags().BoolVarP(&flagInteractive, "interactive", "i", false, tc.AngelaPolishFlagInteractive)

	// Synthesizer flags. The flags are declared on every angela
	// sub-command so users can override per-run; the framework only acts on
	// docs whose synthesizers are registered.
	cmd.Flags().StringSliceVar(&flagSynthesizers, "synthesizers", nil, tc.AngelaPolishFlagSynthesizers)
	cmd.Flags().BoolVar(&flagNoSynthesizers, "no-synthesizers", false, tc.AngelaPolishFlagNoSynthesizers)
	cmd.Flags().BoolVar(&flagSynthesizerDryRun, "synthesizer-dry-run", false, tc.AngelaPolishFlagSynthDryRun)
	cmd.Flags().BoolVar(&flagSynthesize, "synthesize", false, tc.AngelaPolishFlagSynthesize)
	cmd.Flags().StringVar(&flagSetStatus, "set-status", "", tc.AngelaPolishFlagSetStatus)

	cmd.Flags().StringVar(&flagPersona, "persona", "", tc.AngelaPolishFlagPersona)

	// Story 8-21: structural integrity controls.
	// --arbitrate-rule resolves AI-produced duplicate sections in non-TTY
	// runs (CI pipelines); in TTY the user is prompted per group.
	// -v / --verbose surfaces the otherwise-silent "leaked frontmatter
	// stripped" event on stderr (always recorded in polish.log).
	cmd.Flags().StringVar(&flagArbitrateRule, "arbitrate-rule", "", tc.AngelaPolishFlagArbitrateRule)
	cmd.Flags().BoolVarP(&flagVerbose, "verbose", "v", false, tc.AngelaPolishFlagVerbose)
	cmd.MarkFlagsMutuallyExclusive("arbitrate-rule", "interactive")

	_ = cmd.RegisterFlagCompletionFunc("synthesizers", synthesizerFlagCompletion)
	_ = cmd.RegisterFlagCompletionFunc("set-status", statusFlagCompletion)
	_ = cmd.RegisterFlagCompletionFunc("persona", personaFlagCompletion)
	_ = cmd.RegisterFlagCompletionFunc("arbitrate-rule", arbitrateRuleFlagCompletion)
	_ = cmd.RegisterFlagCompletionFunc("hallucination-strictness", hallucinationStrictnessFlagCompletion)

	// Restore subcommand for polish backups.
	cmd.AddCommand(newAngelaPolishRestoreCmd(cfg, streams))

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

// isTimeoutError reports whether err is or wraps a timeout.
//
// isTimeoutError uses errors.Is against context.DeadlineExceeded as
// the canonical check; net.Error.Timeout() is kept as a fallback for
// legacy http/net errors that do not implement the Is method. A final
// substring fallback covers wrapped errors whose stringification is
// all we have.
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	// Final fallback for wrapped errors whose stringification is all
	// we have — kept so we never regress a case the sentinel checks
	// above happen to miss.
	msg := err.Error()
	return strings.Contains(msg, "context deadline exceeded") ||
		strings.Contains(msg, "Client.Timeout")
}
