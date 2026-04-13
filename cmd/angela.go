// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/cli"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/fileutil"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/spf13/cobra"
)

func newAngelaCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	var flagPath string

	cmd := &cobra.Command{
		Use:          "angela",
		Short:        i18n.T().Cmd.AngelaShort,
		SilenceUsage: true,
	}

	cmd.PersistentFlags().StringVar(&flagPath, "path", "", "Path to a markdown directory (enables standalone mode without lore init)")

	cmd.AddCommand(newAngelaDraftCmd(cfg, streams, &flagPath))
	cmd.AddCommand(newAngelaPolishCmd(cfg, streams))
	cmd.AddCommand(newAngelaReviewCmd(cfg, streams, &flagPath))

	return cmd
}

// draftFlags collects the flags that affect output, gating, and
// severity rewriting, plus the differential flags. Held in a struct
// so that runDraft and runDraftAll share the same resolution logic.
type draftFlags struct {
	all            bool
	verbose        bool
	format         string
	failOn         string
	strict         bool
	severity       []string // repeated --severity category=level
	diffOnly       bool     // hide PERSISTING findings, show only NEW + RESOLVED
	resetState     bool     // delete state file and treat all findings as NEW
	personas       string   // "auto" | "manual" | "all" | "none"
	manualPersonas []string // persona names for --personas manual
	interactive    bool     // interactive fix-it TUI
	autofix        string   // "safe" or "aggressive"
	dryRun         bool     // show diff without writing
}

// resolveDraftFlags merges CLI flags with config defaults and returns
// the effective values plus the merged severity override map. Called
// once at the top of both runDraft and runDraftAll so behavior stays
// consistent across single-file and --all modes.
//
// Fallback order for each field:
//
//	CLI flag  →  .lorerc value  →  hardcoded default
//
// The hardcoded defaults match setAngelaDefaults so tests that bypass
// the full config load path (and therefore see zero-value configs) still
// get usable values instead of a flag-validation error.
func resolveDraftFlags(cfg *config.Config, f draftFlags) (effFormat, effFailOn string, effStrict bool, overrides map[string]string, err error) {
	effFormat = f.format
	if effFormat == "" {
		effFormat = cfg.Angela.Draft.Output.Format
	}
	if effFormat == "" {
		effFormat = "human"
	}

	effFailOn = f.failOn
	if effFailOn == "" {
		effFailOn = cfg.Angela.Draft.ExitCode.FailOn
	}
	if effFailOn == "" {
		effFailOn = "error"
	}
	if err = validateFailOn(effFailOn); err != nil {
		return "", "", false, nil, err
	}
	effStrict = f.strict || cfg.Angela.Draft.ExitCode.Strict

	flagOverrides, perr := parseSeverityFlag(f.severity)
	if perr != nil {
		return "", "", false, nil, perr
	}
	overrides = mergeSeverityOverride(cfg.Angela.Draft.SeverityOverride, flagOverrides)
	return effFormat, effFailOn, effStrict, overrides, nil
}

func newAngelaDraftCmd(cfg *config.Config, streams domain.IOStreams, flagPath *string) *cobra.Command {
	var flags draftFlags

	cmd := &cobra.Command{
		Use:           "draft [filename]",
		Short:         i18n.T().Cmd.AngelaDraftShort,
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		// SilenceErrors prevents cobra from printing "Error: exit code 1"
		// when runDraftAll returns a *cli.ExitCodeError for gating purposes.
		// The exit code is still propagated to the process via root.go.
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// CLI flags override persona config.
			if flags.personas != "" {
				cfg.Angela.Personas.Selection = flags.personas
			}
			if len(flags.manualPersonas) > 0 {
				cfg.Angela.Personas.ManualList = flags.manualPersonas
			}

			// --interactive and --autofix are mutually exclusive.
			if flags.interactive && flags.autofix != "" {
				return fmt.Errorf("angela: draft: --interactive and --autofix are mutually exclusive")
			}

			// --all mode: analyze entire corpus
			if flags.all {
				return runDraftAll(cfg, streams, flagPath, flags)
			}

			// Resolve docs directory: --path (standalone) or .lore/docs (normal)
			docsDir, standalone := resolveDocsDir(flagPath)
			if !standalone {
				if err := requireLoreDir(streams); err != nil {
					return err
				}
			}

			var filename string
			if len(args) == 0 {
				// No argument: analyze the most recent document.
				store := newCorpusReader(docsDir, standalone)
				docs, err := store.ListDocs(domain.DocFilter{})
				if err != nil || len(docs) == 0 {
					return fmt.Errorf("%s", i18n.T().Cmd.AngelaDraftNoFile)
				}
				// ListDocs returns alphabetical order; pick latest by date.
				latest := docs[0]
				for _, d := range docs[1:] {
					if d.Date > latest.Date {
						latest = d
					}
				}
				filename = latest.Filename
				_, _ = fmt.Fprintf(streams.Err, "→ %s\n\n", filename)
			} else {
				filename = args[0]
			}

			// Validate filename and check exists
			if !standalone {
				if err := storage.ValidateFilename(filename); err != nil {
					return fmt.Errorf("angela: draft: %w", err)
				}
			} else {
				// Standalone: reject path traversal even without strict lore filename format
				if strings.Contains(filename, "..") || filepath.IsAbs(filename) {
					return fmt.Errorf("angela: draft: filename must not contain '..' or be absolute: %s", filename)
				}
			}
			docPath := filepath.Join(docsDir, filename)
			if _, err := os.Stat(docPath); err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return fmt.Errorf(i18n.T().Cmd.AngelaDraftNotFound, filename)
				}
				return fmt.Errorf("angela: draft: %w", err)
			}

			// Read document once
			raw, err := os.ReadFile(docPath)
			if err != nil {
				return fmt.Errorf("angela: draft: read: %w", err)
			}
			content := string(raw)

			// Parse front matter. In standalone mode use the permissive
			// parser so partial/external front matter (e.g. just `type`
			// without `status`) is preserved instead of silently discarded.
			var meta domain.DocMeta
			var parseErr error
			if standalone {
				meta, _, parseErr = storage.UnmarshalPermissive(raw)
			} else {
				meta, _, parseErr = storage.Unmarshal(raw)
			}
			if parseErr != nil {
				if !standalone {
					return fmt.Errorf("angela: draft: parse: %w", parseErr)
				}
				// Standalone: synthetic metadata from filename
				meta = storage.BuildPlainMeta(filename)
			}
			meta.Filename = filename

			// Load corpus (warn on error, don't fail)
			store := newCorpusReader(docsDir, standalone)
			corpus, corpusErr := store.ListDocs(domain.DocFilter{})
			if corpusErr != nil {
				_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.AngelaDraftCorpusWarn+"\n", corpusErr)
			}

			// Parse style guide
			var styleRules map[string]interface{}
			if cfg.Angela.StyleGuide != nil {
				styleRules = cfg.Angela.StyleGuide
			}
			guide := angela.ParseStyleGuide(styleRules)

			// Smart persona selection per doc type. ResolvePersonas is kept
			// for scoring/display; the actual personas passed to AnalyzeDraft
			// come from SelectPersonasForDoc which honors config selection
			// mode + free-form mode.
			scored := angela.ResolvePersonas(meta.Type, content)
			personas := angela.SelectPersonasForDoc(meta.Type, cfg.Angela.Personas)

			// Run analysis (with persona draft checks)
			suggestions := angela.AnalyzeDraft(content, meta, guide, corpus, personas)
			coherence := angela.CheckCoherence(content, meta, corpus)
			suggestions = append(suggestions, coherence...)

			// Include style guide parse warnings
			suggestions = append(suggestions, guide.Warnings...)

			// No suggestions
			if len(suggestions) == 0 {
				_, _ = fmt.Fprintf(streams.Err, "%s\n", i18n.T().Cmd.AngelaDraftNoSuggestions)
				return nil
			}

			// Autofix mode.
			if flags.autofix != "" {
				mode, modeErr := angela.ParseAutofixMode(flags.autofix)
				if modeErr != nil {
					return modeErr
				}
				fixed, result := angela.RunAutofix(content, meta, mode, corpus)
				if result.Error != nil {
					return result.Error
				}
				if len(result.Fixed) == 0 {
					_, _ = fmt.Fprintf(streams.Err, "autofix: no fixable findings in %s\n", filename)
					return nil
				}
				if flags.dryRun {
					diff := angela.AutofixDryRun(content, fixed, filename)
					if diff != "" {
						_, _ = fmt.Fprint(streams.Out, diff)
					}
					return nil
				}
				docPath := filepath.Join(docsDir, filename)
				// Backup before autofix write, mirroring polish backup logic.
				if cfg.Angela.Draft.Autofix.Backup {
					workDir, wderr := os.Getwd()
					if wderr != nil {
						return fmt.Errorf("angela: autofix: cwd: %w", wderr)
					}
					stDir := config.ResolveStateDir(workDir, cfg, cfg.DetectedMode)
					relPath, _ := filepath.Rel(workDir, docPath)
					if _, berr := angela.WriteBackup(workDir, stDir, "draft-backups", relPath); berr != nil {
						return fmt.Errorf("angela: autofix: backup: %w", berr)
					}
				}
				if err := fileutil.AtomicWrite(docPath, []byte(fixed), 0o644); err != nil {
					return fmt.Errorf("angela: autofix: write: %w", err)
				}
				// Note: the draft state cache (if any) will be naturally invalidated
				// on the next --all run because the file's ContentHash will have changed.
				remaining := angela.ReanalyzeAfterFix(fixed, meta, guide, corpus)
				_, _ = fmt.Fprintf(streams.Err, "autofix: %s — %d fixes applied", filename, len(result.Fixed))
				if remaining > 0 {
					_, _ = fmt.Fprintf(streams.Err, " (%d findings remain)", remaining)
				}
				_, _ = fmt.Fprintf(streams.Err, "\n")
				for _, f := range result.Fixed {
					_, _ = fmt.Fprintf(streams.Err, "  %s\n", f)
				}
				return nil
			}

			// Interactive fix-it TUI.
			if flags.interactive {
				if !angela.IsTTYAvailable() {
					_, _ = fmt.Fprintf(streams.Err, "%s", i18n.T().UI.InteractiveFallback)
				} else {
					findings := make([]angela.DraftFinding, 0, len(suggestions))
					for _, s := range suggestions {
						findings = append(findings, angela.DraftFinding{
							Filename:   filename,
							Suggestion: s,
							Hash:       angela.DraftFindingHash(filename, s),
						})
					}
					metaMap := map[string]domain.DocMeta{filename: meta}
					model := angela.NewDraftInteractiveModel(
						findings, docsDir, metaMap, guide, corpus, personas, standalone,
					)
					p := tea.NewProgram(model, tea.WithAltScreen())
					finalModel, tuiErr := p.Run()
					if tuiErr != nil {
						return fmt.Errorf("angela: draft: interactive: %w", tuiErr)
					}
					if m, ok := finalModel.(angela.DraftInteractiveModel); ok && m.QuitSummary != "" {
						_, _ = fmt.Fprintf(streams.Err, "%s\n", m.QuitSummary)
					}
					return nil
				}
			}

			// Angela score (average of top personas)
			avg := angela.AverageScore(scored)
			var activeNames []string
			for _, sp := range scored {
				activeNames = append(activeNames, sp.Profile.DisplayName)
			}

			// Format output
			_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.AngelaDraftHeader+"\n", filename)
			_, _ = fmt.Fprintf(streams.Err, "  "+i18n.T().Cmd.AngelaDraftScoreLine+"\n\n", strings.Join(activeNames, " + "), avg)
			for _, s := range suggestions {
				_, _ = fmt.Fprintf(streams.Err, "  %-8s %-14s %s\n",
					s.Severity, s.Category, s.Message)
			}
			_, _ = fmt.Fprintf(streams.Err, "\n"+i18n.T().Cmd.AngelaDraftSuggCount+"\n", len(suggestions))

			return nil
		},
	}

	cmd.Flags().BoolVar(&flags.all, "all", false, "Analyze all documents in the corpus")
	cmd.Flags().BoolVarP(&flags.verbose, "verbose", "v", false, "Show every suggestion inline (default: warnings only)")
	// CI output & gating flags
	cmd.Flags().StringVar(&flags.format, "format", "", "Output format: human | json (default: from config)")
	cmd.Flags().StringVar(&flags.failOn, "fail-on", "", "Exit non-zero when findings at this level or higher exist: error | warning | info | never")
	cmd.Flags().BoolVar(&flags.strict, "strict", false, "Promote all warnings to errors for exit-code purposes")
	cmd.Flags().StringSliceVar(&flags.severity, "severity", nil, "Override category severity (repeatable: --severity coherence=off --severity style=warning)")
	// Differential flags
	cmd.Flags().BoolVar(&flags.diffOnly, "diff-only", false, "Hide PERSISTING findings and show only NEW/RESOLVED since the previous --all run")
	cmd.Flags().BoolVar(&flags.resetState, "reset-state", false, "Delete the draft state file and treat all current findings as NEW")
	cmd.Flags().StringVar(&flags.personas, "personas", "", "Persona selection mode: auto | manual | all | none")
	cmd.Flags().StringSliceVar(&flags.manualPersonas, "manual-personas", nil, "Persona names for --personas manual (e.g. storyteller,architect)")
	// Interactive fix-it TUI
	cmd.Flags().BoolVarP(&flags.interactive, "interactive", "i", false, "Launch interactive fix-it TUI to walk through findings")
	// Autofix mode
	cmd.Flags().StringVar(&flags.autofix, "autofix", "", "Apply mechanical fixes: safe | aggressive")
	cmd.Flags().BoolVar(&flags.dryRun, "dry-run", false, "Preview autofix changes without writing")
	return cmd
}

// resolveDocsDir returns the docs directory and whether standalone mode is active.
func resolveDocsDir(flagPath *string) (string, bool) {
	if flagPath != nil && *flagPath != "" {
		return *flagPath, true
	}
	return filepath.Join(".lore", "docs"), false
}

// newCorpusReader returns the appropriate CorpusReader based on mode.
// Standalone mode uses PlainCorpusStore (graceful without front matter).
// Normal mode uses CorpusStore (strict lore format).
func newCorpusReader(dir string, standalone bool) domain.CorpusReader {
	if standalone {
		return &storage.PlainCorpusStore{Dir: dir}
	}
	return &storage.CorpusStore{Dir: dir}
}

// runDraftAll analyzes every document in the corpus and produces a
// structured DraftReport, then delegates rendering to the configured
// reporter (human or JSON). Instead of printing inline while walking
// the corpus, we build a complete report first, apply severity
// overrides, then render.
//
// Differential analysis: when cfg.Angela.Draft.Differential.Enabled is
// true (default), each doc's SHA-256 content hash is looked up in a
// persistent state file. Cache hits skip AnalyzeDraft + CheckCoherence +
// ScoreDocument entirely and reuse the stored result. A second run with
// no file changes completes in well under 100ms on a 68-doc corpus.
// Every finding is tagged with DiffStatus relative to the previous run
// so the reporter can render NEW / PERSISTING / RESOLVED annotations
// (or hide PERSISTING with --diff-only for CI-friendly output).
//
// Note: suggestions are cached PRE-override so that changing the
// severity-override config between runs does not require invalidating
// the whole state file — overrides are re-applied fresh on each run.
func runDraftAll(cfg *config.Config, streams domain.IOStreams, flagPath *string, f draftFlags) error {
	if f.autofix != "" {
		return fmt.Errorf("angela: draft --all: --autofix is not yet supported with --all")
	}

	effFormat, effFailOn, effStrict, overrides, err := resolveDraftFlags(cfg, f)
	if err != nil {
		return err
	}

	docsDir, standalone := resolveDocsDir(flagPath)
	if !standalone {
		if err := requireLoreDir(streams); err != nil {
			return err
		}
	}

	store := newCorpusReader(docsDir, standalone)
	corpus, err := store.ListDocs(domain.DocFilter{})
	if err != nil {
		return fmt.Errorf("angela: draft --all: %w", err)
	}

	// Resolve state file location, honor --reset-state, and load the
	// previous state. Any load error is logged as a verbose notice
	// (fresh state is returned either way) — a broken state file must
	// never prevent the user from running draft.
	diffEnabled := cfg.Angela.Draft.Differential.Enabled
	var statePath string
	var prevState *angela.DraftState
	if diffEnabled {
		workDir, wderr := os.Getwd()
		if wderr != nil {
			return fmt.Errorf("angela: draft --all: cwd: %w", wderr)
		}
		stateDir := config.ResolveStateDir(workDir, cfg, cfg.DetectedMode)
		stateFile := cfg.Angela.Draft.Differential.StateFile
		if stateFile == "" {
			stateFile = "draft-state.json"
		}
		// Reject any state_file that would escape stateDir. A malicious
		// .lorerc using `state_file: "../../../etc/passwd"` could
		// otherwise delete or overwrite arbitrary files via --reset-state
		// or SaveDraftState.
		if err := angela.AssertContainedRelPath(stateFile); err != nil {
			return fmt.Errorf("angela: draft --all: state_file: %w", err)
		}
		statePath = filepath.Join(stateDir, stateFile)

		// Take an exclusive flock on a side-car `.lock` file for the
		// entire load → mutate → save critical section so two
		// concurrent runs on the same workspace (CI fan-out,
		// pre-commit + manual run) cannot ping-pong the state file.
		lock, lockErr := fileutil.NewFileLock(statePath)
		if lockErr != nil {
			return fmt.Errorf("angela: draft --all: state lock: %w", lockErr)
		}
		defer lock.Unlock()

		if f.resetState {
			// --reset-state: fail loudly on os.Remove errors unless the
			// file was already missing. A perms/Windows-handle failure
			// must not leave the user thinking the state was cleared
			// while the next load would pick up the old data.
			if err := os.Remove(statePath); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("angela: draft --all: --reset-state: %w", err)
			}
			if !strings.EqualFold(effFormat, "json") {
				fmt.Fprintf(streams.Err, "draft: state file reset (--reset-state)\n")
			}
		}
		var loadErr error
		prevState, loadErr = angela.LoadDraftState(statePath)
		if loadErr != nil {
			// A corrupt or incompatible state file is quarantined aside
			// with a .corrupt-<ts> suffix so the user can recover it
			// by hand. Always announce on stderr, not just under
			// --verbose: the user must know their cache was reset.
			// Stderr is safe to write in both human and JSON modes
			// (stdout carries the JSON payload).
			if errors.Is(loadErr, angela.ErrStateCorrupt) {
				if quarPath, qerr := angela.QuarantineCorruptState(statePath); qerr == nil {
					fmt.Fprintf(streams.Err, "draft: state file was corrupt; quarantined at %s\n", quarPath)
				} else {
					// If we cannot even rename the file aside, refuse
					// to blow away the user's data. Surface the error.
					return fmt.Errorf("angela: draft --all: corrupt state at %s and cannot quarantine: %w", statePath, qerr)
				}
			} else {
				fmt.Fprintf(streams.Err, "draft: %v (continuing with fresh state)\n", loadErr)
			}
		}
	} else {
		prevState = &angela.DraftState{Version: angela.DraftStateVersion, Entries: map[string]angela.DraftEntry{}}
	}

	if len(corpus) == 0 {
		// Even on an empty corpus we still compute the diff against
		// prevState so deleted files show up as RESOLVED and the
		// state file is kept in sync. Without this, a user who
		// deletes their last doc sees the old entries forever.
		empty := DraftReport{
			Version: draftJSONSchemaVersion,
			Mode:    cfg.DetectedMode.String(),
			Scanned: 0,
			Files:   []DraftFileReport{},
		}
		if diffEnabled {
			diff, resolved := angela.AnnotateAndDiff(prevState, map[string][]angela.Suggestion{})
			empty.Diff = &diff
			empty.Resolved = resolved
			emptyState := &angela.DraftState{
				Version: angela.DraftStateVersion,
				Entries: map[string]angela.DraftEntry{},
			}
			if err := angela.SaveDraftState(statePath, emptyState); err != nil {
				fmt.Fprintf(streams.Err, "draft: state save warning: %v\n", err)
			}
		}
		empty.computeSummary()
		if !strings.EqualFold(effFormat, "json") {
			fmt.Fprintf(streams.Err, "%s\n", i18n.T().Cmd.AngelaDraftAllNoDocs)
		}
		return newDraftReporter(effFormat, streams, f.verbose).Report(empty)
	}

	var styleRules map[string]interface{}
	if cfg.Angela.StyleGuide != nil {
		styleRules = cfg.Angela.StyleGuide
	}
	guide := angela.ParseStyleGuide(styleRules)

	// Header goes to stderr in human mode. In JSON mode it's suppressed
	// so the single JSON document on stdout remains pure.
	if !strings.EqualFold(effFormat, "json") {
		fmt.Fprintf(streams.Err, i18n.T().Cmd.AngelaDraftAllHeader+"\n\n", len(corpus))
	}

	report := DraftReport{
		Version: draftJSONSchemaVersion,
		Mode:    cfg.DetectedMode.String(),
		Scanned: len(corpus),
		Files:   make([]DraftFileReport, 0, len(corpus)),
	}

	// Single reporter instance — the human reporter uses its
	// ReportFile hook to stream per-file rows inline with progress.
	// JSON reporter no-ops ReportFile and emits the complete payload
	// only from Report() at the end. When --diff-only is set, the
	// reporter will filter PERSISTING findings at render time.
	reporter := newDraftReporter(effFormat, streams, f.verbose)
	if hr, ok := reporter.(*humanDraftReporter); ok {
		hr.diffOnly = f.diffOnly
	}

	// Build the fresh state in place as we walk the corpus. Kept
	// separate from prevState so a mid-run crash cannot corrupt the
	// existing on-disk file.
	newState := &angela.DraftState{
		Version: angela.DraftStateVersion,
		Entries: make(map[string]angela.DraftEntry, len(corpus)),
	}
	currentFiles := make(map[string]bool, len(corpus))
	currentSuggestions := make(map[string][]angela.Suggestion, len(corpus))

	for idx, meta := range corpus {
		drawProgress(effFormat, streams, idx+1, len(corpus), meta.Filename)
		currentFiles[meta.Filename] = true
		raw, err := os.ReadFile(filepath.Join(docsDir, meta.Filename))
		if err != nil {
			// Render as a single error-level finding so CI pipelines
			// that fail on errors catch unreadable files. Do NOT cache
			// these — retrying is free and the error might be transient.
			row := DraftFileReport{
				Filename: meta.Filename,
				Suggestions: []angela.Suggestion{{
					Category: "io",
					Severity: angela.SeverityError,
					Message:  err.Error(),
				}},
			}
			report.Files = append(report.Files, row)
			reporter.ReportFile(row)
			continue
		}
		content := string(raw)

		// Cache lookup. A hit replays the stored pre-override
		// suggestions + score; a miss runs the full analyzer.
		var preOverride []angela.Suggestion
		var score angela.QualityScore
		var profile string

		hash := angela.ContentHash(raw)
		if diffEnabled {
			// Cache hit requires BOTH the content hash AND the analyzer
			// schema version to match. Otherwise a persona-registry or
			// coherence-rule edit would silently serve stale suggestions
			// until every file changed.
			if cached, ok := prevState.Entries[meta.Filename]; ok &&
				cached.ContentHash == hash &&
				cached.AnalyzerSchemaVersion == angela.AnalyzerSchemaVersion {
				preOverride = append([]angela.Suggestion(nil), cached.Suggestions...) // defensive copy
				score.Total = cached.Score
				score.Grade = cached.Grade
				profile = cached.Profile
			}
		}
		if preOverride == nil {
			// Smart persona selection per doc type.
			personas := angela.SelectPersonasForDoc(meta.Type, cfg.Angela.Personas)
			preOverride = angela.AnalyzeDraft(content, meta, guide, corpus, personas)
			preOverride = append(preOverride, angela.CheckCoherence(content, meta, corpus)...)
			score = angela.ScoreDocument(content, meta)
			profile = score.Profile
		}

		// Persist the pre-override snapshot so future runs can re-apply
		// overrides against the raw analysis result.
		if diffEnabled {
			newState.Entries[meta.Filename] = angela.DraftEntry{
				ContentHash:           hash,
				LastAnalyzed:          time.Now().UTC(),
				Suggestions:           preOverride,
				Score:                 score.Total,
				Grade:                 score.Grade,
				Profile:               profile,
				AnalyzerSchemaVersion: angela.AnalyzerSchemaVersion,
			}
		}

		// Apply severity overrides BEFORE scoring the report row so the
		// per-file warning count reflects the user's gating choices.
		suggestions := angela.ApplySeverityOverride(preOverride, overrides)
		if effStrict {
			suggestions = angela.PromoteWarningsToErrors(suggestions)
		}
		currentSuggestions[meta.Filename] = suggestions

		row := DraftFileReport{
			Filename:    meta.Filename,
			Score:       score.Total,
			Grade:       score.Grade,
			Profile:     profile,
			Suggestions: suggestions,
		}
		report.Files = append(report.Files, row)
	}

	// VHS tape ↔ doc cross-check (if tape directory exists).
	// Only in human mode — VHS findings don't have a per-file home in
	// the structured report yet. A follow-up story can fold them in.
	if !strings.EqualFold(effFormat, "json") {
		_ = runVHSCheck(docsDir, streams)
	}

	// Annotate each suggestion with DiffStatus relative to the previous
	// run, compute the diff summary, and record any RESOLVED findings
	// (whose source file may or may not still exist). AnnotateAndDiff
	// is called BEFORE the reporting loop so that diff tags are set
	// before rendering.
	if diffEnabled {
		diff, resolved := angela.AnnotateAndDiff(prevState, currentSuggestions)
		report.Diff = &diff
		report.Resolved = resolved

		// Prune state entries for files that disappeared from the
		// corpus so the state file stays small.
		pruned := angela.PruneMissingEntries(newState, currentFiles)
		_ = pruned // reserved for future verbose logging

		// Save the new state unconditionally so the next run has an
		// up-to-date snapshot. Failures are non-fatal and logged on
		// stderr in human mode only.
		if err := angela.SaveDraftState(statePath, newState); err != nil && !strings.EqualFold(effFormat, "json") {
			fmt.Fprintf(streams.Err, "draft: state save warning: %v\n", err)
		}
	}

	// Report files after AnnotateAndDiff so diff tags are set before rendering.
	for _, row := range report.Files {
		reporter.ReportFile(row)
	}

	report.computeSummary()

	// Interactive TUI for --all mode.
	if f.interactive {
		if !angela.IsTTYAvailable() {
			fmt.Fprintf(streams.Err, "%s", i18n.T().UI.InteractiveFallback)
		} else {
			var findings []angela.DraftFinding
			metaMap := make(map[string]domain.DocMeta, len(corpus))
			for _, c := range corpus {
				metaMap[c.Filename] = c
			}
			for _, fr := range report.Files {
				for _, s := range fr.Suggestions {
					findings = append(findings, angela.DraftFinding{
						Filename:   fr.Filename,
						Suggestion: s,
						Hash:       angela.DraftFindingHash(fr.Filename, s),
					})
				}
			}
			if len(findings) > 0 {
				model := angela.NewDraftInteractiveModel(
					findings, docsDir, metaMap, guide, corpus,
					angela.SelectPersonasForDoc("", cfg.Angela.Personas),
					standalone,
				)
				p := tea.NewProgram(model, tea.WithAltScreen())
				finalModel, tuiErr := p.Run()
				if tuiErr != nil {
					return fmt.Errorf("angela: draft --all: interactive: %w", tuiErr)
				}
				if m, ok := finalModel.(angela.DraftInteractiveModel); ok && m.QuitSummary != "" {
					fmt.Fprintf(streams.Err, "%s\n", m.QuitSummary)
				}
				return nil
			}
		}
	}

	if err := reporter.Report(report); err != nil {
		return fmt.Errorf("angela: draft --all: render: %w", err)
	}

	// Verbose hint only in human mode when there are hidden info-level
	// findings. JSON consumers already see everything in the payload.
	if !strings.EqualFold(effFormat, "json") && !f.verbose {
		infoCount := 0
		for _, s := range report.allSuggestions() {
			if s.Severity == angela.SeverityInfo {
				infoCount++
			}
		}
		if infoCount > 0 {
			fmt.Fprintf(streams.Err, "%s\n", i18n.T().Cmd.AngelaDraftAllVerboseHint)
		}
	}

	// Exit code resolution. The existing *cli.ExitCodeError type
	// propagates the code through cobra and into root.go's
	// Execute() → os.Exit handler.
	code := angela.ExitCodeFor(report.allSuggestions(), effFailOn)
	if code != 0 {
		return &cli.ExitCodeError{Code: code}
	}
	return nil
}

// runVHSCheck looks for VHS tape files near the docs directory and reports mismatches.
// Searches common locations: assets/vhs/, ../assets/vhs/ (relative to docsDir).
//
// Severity = info, not warning: VHS tape/doc mismatches are a convention
// lore uses internally, but external users running Angela as a CI linter
// on their docs site shouldn't have their pipelines fail because of an
// unrelated GIF reference. Warning-level would also conflict with the
// scripts/angela-ci.sh severity counter (which treats "warning" as a
// blocker for --fail-on warning).
func runVHSCheck(docsDir string, streams domain.IOStreams) int {
	// Try common tape directory locations relative to the docs dir
	candidates := []string{
		filepath.Join(docsDir, "..", "assets", "vhs"),
		filepath.Join(docsDir, "..", "..", "assets", "vhs"),
		filepath.Join(docsDir, "assets", "vhs"),
	}

	var tapeDir string
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			tapeDir = c
			break
		}
	}

	if tapeDir == "" {
		return 0 // no tape directory found — skip silently
	}

	signals := angela.AnalyzeVHSSignals(tapeDir, docsDir, nil)
	findings := 0

	for _, tape := range signals.OrphanTapes {
		_, _ = fmt.Fprintf(streams.Err, "  %-8s %-14s %s → output GIF not referenced in any doc\n",
			"info", "vhs", tape)
		findings++
	}

	for _, ref := range signals.OrphanGIFs {
		_, _ = fmt.Fprintf(streams.Err, "  %-8s %-14s %s references %s → no .tape source found\n",
			"info", "vhs", ref.DocFilename, ref.GIFPath)
		findings++
	}

	for _, mm := range signals.CommandMismatches {
		_, _ = fmt.Fprintf(streams.Err, "  %-8s %-14s %s: %q → %s\n",
			"info", "vhs", mm.TapeFile, mm.Command, mm.Reason)
		findings++
	}

	return findings
}
