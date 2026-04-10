// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/greycoderk/lore/internal/ui"
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

func newAngelaDraftCmd(cfg *config.Config, streams domain.IOStreams, flagPath *string) *cobra.Command {
	var flagAll bool
	var flagVerbose bool

	cmd := &cobra.Command{
		Use:           "draft [filename]",
		Short:         i18n.T().Cmd.AngelaDraftShort,
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// --all mode: analyze entire corpus
			if flagAll {
				return runDraftAll(cfg, streams, flagPath, flagVerbose)
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

			// AC-6: Validate filename and check exists
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
				if os.IsNotExist(err) {
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

			// Resolve personas for this document
			scored := angela.ResolvePersonas(meta.Type, content)
			personas := angela.Profiles(scored)

			// Run analysis (with persona draft checks — AC-3)
			suggestions := angela.AnalyzeDraft(content, meta, guide, corpus, personas)
			coherence := angela.CheckCoherence(content, meta, corpus)
			suggestions = append(suggestions, coherence...)

			// Include style guide parse warnings
			suggestions = append(suggestions, guide.Warnings...)

			// AC-5: No suggestions
			if len(suggestions) == 0 {
				_, _ = fmt.Fprintf(streams.Err, "%s\n", i18n.T().Cmd.AngelaDraftNoSuggestions)
				return nil
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

	cmd.Flags().BoolVar(&flagAll, "all", false, "Analyze all documents in the corpus")
	cmd.Flags().BoolVarP(&flagVerbose, "verbose", "v", false, "Show every suggestion inline (default: warnings only)")
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

// runDraftAll analyzes every document in the corpus and produces a summary.
// When verbose is true, every suggestion is printed inline; otherwise only
// warnings are shown inline and a hint invites the user to re-run with -v.
func runDraftAll(cfg *config.Config, streams domain.IOStreams, flagPath *string, verbose bool) error {
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

	if len(corpus) == 0 {
		_, _ = fmt.Fprintf(streams.Err, "%s\n", i18n.T().Cmd.AngelaDraftAllNoDocs)
		return nil
	}

	var styleRules map[string]interface{}
	if cfg.Angela.StyleGuide != nil {
		styleRules = cfg.Angela.StyleGuide
	}
	guide := angela.ParseStyleGuide(styleRules)

	var totalSuggestions int
	var totalWarnings int
	var docsWithIssues int

	_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.AngelaDraftAllHeader+"\n\n", len(corpus))

	for idx, meta := range corpus {
		ui.Progress(streams, idx+1, len(corpus), meta.Filename)
		raw, err := os.ReadFile(filepath.Join(docsDir, meta.Filename))
		if err != nil {
			_, _ = fmt.Fprintf(streams.Err, "  %-8s %-40s %s\n", "error", meta.Filename, err)
			continue
		}
		content := string(raw)

		scored := angela.ResolvePersonas(meta.Type, content)
		suggestions := angela.AnalyzeDraft(content, meta, guide, corpus, angela.Profiles(scored))
		suggestions = append(suggestions, angela.CheckCoherence(content, meta, corpus)...)

		score := angela.ScoreDocument(content, meta)
		grade := angela.FormatScore(score)

		if len(suggestions) > 0 {
			docsWithIssues++
			totalSuggestions += len(suggestions)
			warnings := 0
			for _, s := range suggestions {
				if s.Severity == "warning" {
					warnings++
				}
			}
			totalWarnings += warnings
			label := fmt.Sprintf(i18n.T().Cmd.AngelaDraftAllSugg, len(suggestions))
			if warnings > 0 {
				label = fmt.Sprintf(i18n.T().Cmd.AngelaDraftAllSuggWarn, len(suggestions), warnings)
			}
			_, _ = fmt.Fprintf(streams.Err, "  %-4s %-8s %-40s %s\n", grade, "review", meta.Filename, label)

			// Inline details:
			//   - verbose: show every suggestion
			//   - default: show only warnings (they are blockers the user must see)
			for _, s := range suggestions {
				if !verbose && s.Severity != "warning" {
					continue
				}
				_, _ = fmt.Fprintf(streams.Err, "         %-8s %-14s %s\n",
					s.Severity, s.Category, s.Message)
			}
		} else {
			_, _ = fmt.Fprintf(streams.Err, "  %-4s %-8s %s\n", grade, "ok", meta.Filename)
		}
	}

	// VHS tape ↔ doc cross-check (if tape directory exists) — before summary
	vhsFindings := runVHSCheck(docsDir, streams)
	totalSuggestions += vhsFindings

	_, _ = fmt.Fprintf(streams.Err, "\n"+i18n.T().Cmd.AngelaDraftAllSummary+"\n",
		docsWithIssues, len(corpus), totalSuggestions)

	// If non-verbose and there are info-level suggestions that were hidden,
	// invite the user to re-run with -v for the full detail.
	if !verbose && totalSuggestions > totalWarnings {
		_, _ = fmt.Fprintf(streams.Err, "%s\n", i18n.T().Cmd.AngelaDraftAllVerboseHint)
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
