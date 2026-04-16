// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"fmt"

	"github.com/greycoderk/lore/internal/angela/synthesizer"
	"github.com/greycoderk/lore/internal/config"
)

// SynthesizerDraftSuggestions surfaces enrichment opportunities during
// `lore angela draft` WITHOUT modifying anything. Draft is offline forever
// (invariant I1) - this code path imports no AI provider, performs no
// network I/O, and writes nothing.
//
// What it does: parse the doc, ask each enabled synthesizer if it has
// candidates whose signature is stale or missing, and emit one Suggestion
// per stale synthesizer with category "synthesizer" and code
// "pending_enrichment". The user sees these alongside structural and
// persona suggestions and can decide whether to run polish.
//
// Dual-mode (I2): the function uses synthesizer.ParseDoc which uses the
// permissive frontmatter parser, so docs in standalone mkdocs/hugo corpora
// are accepted even when their frontmatter lacks lore-required fields.
//
// docContent is the raw doc bytes (frontmatter + body); the function does
// its own parse rather than reusing AnalyzeDraft's stripFrontMatter path
// because the synthesizer needs structured access to sections and lines.
func SynthesizerDraftSuggestions(
	docPath string,
	docContent []byte,
	registry *synthesizer.Registry,
	cfg config.SynthesizersConfig,
) []Suggestion {
	if registry == nil {
		return nil
	}
	enabledCfg := synthesizer.EnabledConfig{Enabled: cfg.Enabled}
	enabled := registry.Enabled(enabledCfg)
	if len(enabled) == 0 {
		return nil
	}

	doc, err := synthesizer.ParseDoc(docPath, docContent)
	if err != nil {
		// Parse failures are not draft errors: the structural checks
		// upstream already report them.
		return nil
	}

	synthCfg := synthesizer.Config{
		WellKnownServerFields: cfg.WellKnownServerFields,
		PerSynthesizer:        cfg.PerSynthesizer,
	}

	var suggestions []Suggestion
	for _, s := range enabled {
		if !s.Applies(doc) {
			continue
		}
		candidates, err := s.Detect(doc)
		if err != nil || len(candidates) == 0 {
			continue
		}

		// Compute current evidence to compare against the stored signature.
		// Synthesize is pure - no side effects - so calling it here is
		// safe and cheap (concrete synthesizers cap their work to the
		// candidate's parsed extras).
		var allEvs []synthesizer.Evidence
		for _, c := range candidates {
			_, evs, _, err := s.Synthesize(c, synthCfg)
			if err != nil {
				continue
			}
			allEvs = append(allEvs, evs...)
		}
		if len(allEvs) == 0 {
			continue
		}

		existing := doc.Signatures[s.Name()]
		version := lookupSynthesizerVersion(cfg, s.Name())
		if synthesizer.IsFresh(existing, allEvs, version) {
			continue
		}

		suggestions = append(suggestions, Suggestion{
			Category: "synthesizer",
			Severity: "info",
			Message: fmt.Sprintf(
				"pending_enrichment: %s peut générer %d bloc(s) ready-to-use depuis ce doc — lance `lore angela polish --synthesizer-dry-run` pour prévisualiser",
				s.Name(), len(candidates),
			),
		})
	}
	return suggestions
}
