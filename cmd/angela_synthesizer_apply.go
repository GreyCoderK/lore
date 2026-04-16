// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/angela/synthesizer"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/fileutil"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/storage"
)

// runSynthesizerApply is invoked by `lore angela polish --synthesize`. It
// applies every active synthesizer proposal to the target doc and writes
// the result back via storage.Marshal + atomic file replacement.
//
// Offline path (I1 for the draft-adjacent variant is respected because
// polish's AI pipeline is bypassed entirely): no AI provider is created,
// no credentials are consulted. Only synthesizers listed in
// cfg.Angela.Synthesizers.Enabled run.
//
// Writes are atomic (fileutil.AtomicWrite). If the write fails after the
// first successful application, earlier proposals ARE retained in the
// intermediate doc state held in memory — the atomic write guarantees the
// file on disk is either fully updated or untouched.
//
// setStatus, when non-empty, updates the doc's frontmatter Status field
// AFTER proposals are applied. Re-running with a different setStatus is
// always allowed (the field is overwritten, never rejected). When empty,
// Status is preserved verbatim.
func runSynthesizerApply(cfg *config.Config, streams domain.IOStreams, docPath, setStatus string) error {
	raw, err := os.ReadFile(docPath)
	if err != nil {
		return fmt.Errorf("angela: polish --synthesize: read %s: %w", docPath, err)
	}

	doc, err := synthesizer.ParseDoc(docPath, raw)
	if err != nil {
		return fmt.Errorf("angela: polish --synthesize: parse %s: %w", docPath, err)
	}

	proposals, err := angela.SynthesizerProposalsForDoc(doc, synthesizer.DefaultRegistry, cfg.Angela.Synthesizers)
	if err != nil {
		return fmt.Errorf("angela: polish --synthesize: %w", err)
	}
	if len(proposals) == 0 {
		_, _ = fmt.Fprintf(streams.Err, "→ %s\n  aucune proposition — signature fraîche ou aucun synthesizer applicable\n", docPath)
		return nil
	}

	// Apply proposals serially. Each ApplySynthesizerProposal returns a
	// fresh (body, meta) from the current doc state. To chain proposals,
	// re-parse the updated body before the next iteration - the section
	// line numbers shift when a block is inserted.
	currentBody := doc.Body
	currentMeta := doc.Meta
	applied := 0
	for i, p := range proposals {
		// Re-parse the current state so p.Doc references up-to-date sections.
		fresh, parseErr := synthesizer.ParseDoc(docPath, []byte(marshalDoc(currentMeta, currentBody)))
		if parseErr != nil {
			return fmt.Errorf("angela: polish --synthesize: re-parse before proposal %d: %w", i+1, parseErr)
		}
		p.Doc = fresh

		newBody, newMeta, applyErr := angela.ApplySynthesizerProposal(p)
		if applyErr != nil {
			_, _ = fmt.Fprintf(streams.Err, "  ⚠ proposition %d/%d (%s): %v — skipped\n", i+1, len(proposals), p.CandidateKey, applyErr)
			continue
		}
		currentBody = newBody
		currentMeta = newMeta
		applied++
		_, _ = fmt.Fprintf(streams.Err, "  ✓ proposition %d/%d appliquée: %s (%s)\n", i+1, len(proposals), p.CandidateKey, p.SynthesizerName)
	}
	if applied == 0 && setStatus == "" {
		_, _ = fmt.Fprintf(streams.Err, "→ %s\n  %s\n", docPath, i18n.T().Cmd.SynthApplyNone)
		return nil
	}

	// Status promotion is ALWAYS allowed, even on repeated runs. This
	// makes re-polishing safe: users can bump draft → reviewed → published
	// incrementally. No lifecycle enforcement — the lore CLI trusts the
	// operator to pick a meaningful value.
	oldStatus := currentMeta.Status
	if setStatus != "" && currentMeta.Status != setStatus {
		currentMeta.Status = setStatus
	}

	final := marshalDoc(currentMeta, currentBody)
	if err := fileutil.AtomicWrite(docPath, []byte(final), 0o644); err != nil {
		return fmt.Errorf("angela: polish --synthesize: atomic write: %w", err)
	}
	msg := fmt.Sprintf(i18n.T().Cmd.SynthApplyDone, applied)
	if setStatus != "" && oldStatus != setStatus {
		msg += fmt.Sprintf(i18n.T().Cmd.SynthApplyStatusChange, oldStatus, setStatus)
	}
	_, _ = fmt.Fprintf(streams.Err, "\n→ %s\n  %s.\n", docPath, msg)
	return nil
}

// marshalDoc serializes meta + body via the storage layer. A thin wrapper
// so the apply path uses the same canonical frontmatter format the rest
// of lore_cli writes.
func marshalDoc(meta domain.DocMeta, body string) string {
	out, err := storage.Marshal(meta, body)
	if err != nil {
		// Marshal errors on DocMeta are effectively impossible (pure YAML
		// round-trip); fall back to a best-effort concatenation rather
		// than aborting the apply flow.
		return "---\n---\n" + body
	}
	return string(out)
}
