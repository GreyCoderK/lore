// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/angela/synthesizer"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
)

// runSynthesizerDryRun is invoked by `lore angela polish --synthesizer-dry-run`.
// It reads the doc, runs every enabled synthesizer, and prints the rendered
// block markdown to stdout without mutating anything. Diagnostics go to
// stderr. Exits 0 even when no proposals are available - the absence of
// proposals is information, not an error.
func runSynthesizerDryRun(cfg *config.Config, streams domain.IOStreams, docPath string) error {
	raw, err := os.ReadFile(docPath)
	if err != nil {
		return fmt.Errorf("angela: polish --synthesizer-dry-run: read %s: %w", docPath, err)
	}

	doc, err := synthesizer.ParseDoc(docPath, raw)
	if err != nil {
		return fmt.Errorf("angela: polish --synthesizer-dry-run: parse %s: %w", docPath, err)
	}

	proposals, err := angela.SynthesizerProposalsForDoc(doc, synthesizer.DefaultRegistry, cfg.Angela.Synthesizers)
	if err != nil {
		return fmt.Errorf("angela: polish --synthesizer-dry-run: %w", err)
	}

	if len(proposals) == 0 {
		_, _ = fmt.Fprintf(streams.Err, "→ %s\n  %s\n", docPath, i18n.T().Cmd.SynthApplyNone)
		return nil
	}

	_, _ = fmt.Fprintf(streams.Err, "→ %s\n  "+i18n.T().Cmd.SynthDryRunHeader+"\n\n", docPath, len(proposals))

	for i, p := range proposals {
		_, _ = fmt.Fprintf(streams.Out, "━━━ "+i18n.T().Cmd.SynthDryRunProposal+" ━━━\n",
			i+1, len(proposals), p.SynthesizerName, p.CandidateKey)
		_, _ = fmt.Fprintf(streams.Out, "Insertion: après « %s »\n", p.Block.InsertAfterHeading)
		_, _ = fmt.Fprintf(streams.Out, "Hash évidence: %s\n", p.Signature.Hash)
		_, _ = fmt.Fprintf(streams.Out, "Evidence count: %d\n\n", p.Signature.EvidenceCount)
		_, _ = fmt.Fprintln(streams.Out, p.RenderedMarkdown)

		if len(p.Warnings) > 0 {
			_, _ = fmt.Fprintln(streams.Out, "Warnings:")
			for _, w := range p.Warnings {
				_, _ = fmt.Fprintf(streams.Out, "  - [%s] %s\n", w.Code, w.Message)
			}
			_, _ = fmt.Fprintln(streams.Out)
		}
	}

	_, _ = fmt.Fprintln(streams.Err, strings.Repeat("─", 60))
	_, _ = fmt.Fprintln(streams.Err, i18n.T().Cmd.SynthDryRunFooter)
	return nil
}
