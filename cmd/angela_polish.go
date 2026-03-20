// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/greycoderk/lore/internal/ai"
	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/credential"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/spf13/cobra"
)

func newAngelaPolishCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	var flagDryRun bool
	var flagYes bool

	cmd := &cobra.Command{
		Use:           "polish <filename>",
		Short:         "Improve a document with AI assistance",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			filename := args[0]

			// AC-9: Check .lore/ exists
			docsDir := filepath.Join(".", ".lore", "docs")
			if _, err := os.Stat(filepath.Join(".", ".lore")); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("lore not initialized, run: lore init")
				}
				return fmt.Errorf("angela: polish: %w", err)
			}

			// AC-8: Validate filename and check exists
			if err := storage.ValidateFilename(filename); err != nil {
				return fmt.Errorf("angela: polish: %w", err)
			}
			docPath := filepath.Join(docsDir, filename)
			if _, err := os.Stat(docPath); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("document '%s' not found in .lore/docs/", filename)
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
				return fmt.Errorf("no AI provider configured, set ai.provider in .lorerc.local (try: lore angela draft, no API needed)")
			}

			// Read document
			raw, err := os.ReadFile(docPath)
			if err != nil {
				return fmt.Errorf("angela: polish: read: %w", err)
			}
			originalContent := string(raw)

			meta, _, err := storage.Unmarshal(raw)
			if err != nil {
				return fmt.Errorf("angela: polish: parse: %w", err)
			}
			meta.Filename = filename

			// Style guide
			var styleGuideStr string
			if cfg.Angela.StyleGuide != nil {
				guide := angela.ParseStyleGuide(cfg.Angela.StyleGuide)
				if guide.RequireWhy {
					styleGuideStr += "- Section '## Why' is required\n"
				}
				if guide.RequireAlternatives {
					styleGuideStr += "- Section '## Alternatives' is required\n"
				}
				if guide.MaxBodyLength > 0 {
					styleGuideStr += fmt.Sprintf("- Maximum body length: %d characters\n", guide.MaxBodyLength)
				}
			}

			// Corpus summary
			corpusStore := &storage.CorpusStore{Dir: docsDir}
			corpus, _ := corpusStore.ListDocs(domain.DocFilter{})
			corpusSummary := angela.BuildCorpusSummary(corpus)

			// Resolve personas for this document (AC-4)
			scored := angela.ResolvePersonas(meta.Type, originalContent)
			personas := angela.Profiles(scored)

			// AC-1: Exactly 1 API call
			polished, err := angela.Polish(cmd.Context(), provider, originalContent, meta, styleGuideStr, corpusSummary, personas)
			if err != nil {
				return err
			}

			// Compute diff
			hunks := angela.ComputeDiff(originalContent, polished)

			// AC-10: No changes
			if len(hunks) == 0 {
				_, _ = fmt.Fprintf(streams.Err, "Angela: No changes suggested.\n")
				return nil
			}

			// AC-2: Interactive diff / AC-7: reject all
			accepted, err := angela.InteractiveDiff(hunks, streams, flagDryRun, flagYes)
			if err != nil {
				return fmt.Errorf("angela: polish: %w", err)
			}

			if flagDryRun {
				return nil
			}

			// Check if any accepted
			anyAccepted := false
			for _, a := range accepted {
				if a {
					anyAccepted = true
					break
				}
			}

			// AC-7: All rejected
			if !anyAccepted {
				_, _ = fmt.Fprintf(streams.Err, "No changes applied.\n")
				return nil
			}

			// TOCTOU guard: check file hasn't changed during interactive review
			currentRaw, err := os.ReadFile(docPath)
			if err != nil {
				return fmt.Errorf("angela: polish: re-read: %w", err)
			}
			if !bytes.Equal(currentRaw, raw) {
				return fmt.Errorf("angela: polish: document modified during review. Aborting to prevent data loss")
			}

			// AC-3: Apply changes
			result := angela.ApplyDiff(originalContent, hunks, accepted)

			// Update angela_mode in front matter
			resultMeta, resultBody, err := storage.Unmarshal([]byte(result))
			if err != nil {
				// If parsing fails, use the result as-is with original meta
				resultMeta = meta
				resultBody = result
			}
			resultMeta.AngelaMode = "polish"

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
				_, _ = fmt.Fprintf(streams.Err, "Warning: index regeneration: %s\n", err)
			}

			_, _ = fmt.Fprintf(streams.Err, "Polished   %s\n", filename)
			return nil
		},
	}

	cmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Show diff without applying changes")
	cmd.Flags().BoolVar(&flagYes, "yes", false, "Accept all changes without confirmation")

	return cmd
}
