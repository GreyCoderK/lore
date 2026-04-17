// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/greycoderk/lore/internal/angela/synthesizer"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
)

// RunSynthesizerReview runs every enabled Example Synthesizer
// against each doc in docs and converts the resulting opportunities into
// ReviewFindings. Findings are evidence-grounded by construction
// because each synthesizer Detect/Synthesize call returns a literal
// Evidence list pointing at the source spans that justify the proposed
// enrichment.
//
// Severity defaults to "info" per the 2026-04-15 design decision (Q9).
// Operators escalate selectively via cfg.Synthesizers.PerSynthesizer
// "<name>.severity" or globally via cfg.Synthesizers.PerSynthesizer
// "review.severity".
//
// The function never returns the AI provider's findings - the cmd layer
// merges these synthesizer findings with Review()'s output.
func RunSynthesizerReview(
	ctx context.Context,
	reader domain.CorpusReader,
	docs []domain.DocMeta,
	registry *synthesizer.Registry,
	cfg config.SynthesizersConfig,
) ([]ReviewFinding, error) {
	if registry == nil || reader == nil {
		return nil, nil
	}
	enabled := registry.Enabled(synthesizer.EnabledConfig{Enabled: cfg.Enabled})
	if len(enabled) == 0 {
		return nil, nil
	}

	synthCfg := synthesizer.Config{
		WellKnownServerFields: cfg.WellKnownServerFields,
		PerSynthesizer:        cfg.PerSynthesizer,
	}
	severity := resolveSynthesizerSeverity(cfg)

	var findings []ReviewFinding
	for _, meta := range docs {
		if ctx != nil {
			if err := ctx.Err(); err != nil {
				return findings, err
			}
		}
		raw, err := reader.ReadDoc(meta.Filename)
		if err != nil {
			// Single-doc read failure must not abort the whole review.
			continue
		}
		doc, err := synthesizer.ParseDoc(meta.Filename, []byte(raw))
		if err != nil {
			continue
		}
		applicable := registry.ForDoc(doc, synthesizer.EnabledConfig{Enabled: cfg.Enabled})
		for _, s := range applicable {
			docFindings, err := evaluateSynthesizerForReview(s, doc, synthCfg, severity)
			if err != nil {
				return findings, fmt.Errorf("synthesizer %s on %s: %w", s.Name(), meta.Filename, err)
			}
			findings = append(findings, docFindings...)
		}
	}

	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Documents[0] != findings[j].Documents[0] {
			return findings[i].Documents[0] < findings[j].Documents[0]
		}
		return findings[i].Title < findings[j].Title
	})

	return findings, nil
}

// evaluateSynthesizerForReview runs Detect + Synthesize and emits one
// ReviewFinding per stale or missing synthesizer block. Fresh blocks (the
// signature in frontmatter matches current evidence) emit nothing - I6
// guarantees re-run silence.
func evaluateSynthesizerForReview(
	s synthesizer.Synthesizer,
	doc *synthesizer.Doc,
	cfg synthesizer.Config,
	severity string,
) ([]ReviewFinding, error) {
	candidates, err := s.Detect(doc)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	existing := doc.Signatures[s.Name()]

	var allEvidence []synthesizer.Evidence
	var allWarnings []synthesizer.Warning
	candidateKeys := make([]string, 0, len(candidates))
	for _, c := range candidates {
		_, evs, warns, err := s.Synthesize(c, cfg)
		if err != nil {
			return nil, err
		}
		allEvidence = append(allEvidence, evs...)
		allWarnings = append(allWarnings, warns...)
		candidateKeys = append(candidateKeys, c.Key)
	}

	// I6 short-circuit: identical evidence + matching version => no finding.
	// The per-synthesizer version lives in cfg.PerSynthesizer[<name>]["version"],
	// the same place polish reads. Keeping review and polish on the same
	// freshness check prevents divergent caches (code review finding #10).
	version := synthesizerVersionFromBag(cfg.PerSynthesizer, s.Name())
	if synthesizer.IsFresh(existing, allEvidence, version) {
		return findingsForWarnings(s, doc, allWarnings, severity), nil
	}

	finding := ReviewFinding{
		Severity: severity,
		Title:    fmt.Sprintf("[%s] enrichment available (%d candidate(s))", s.Name(), len(candidates)),
		Description: fmt.Sprintf(
			"Le synthesizer %q peut générer %d bloc(s) ready-to-use depuis ce doc (%s). "+
				"Lancez `lore angela polish --synthesizer-dry-run` pour prévisualiser.",
			s.Name(), len(candidates), strings.Join(candidateKeys, ", "),
		),
		Documents:  []string{doc.Path},
		Evidence:   convertEvidence(allEvidence),
		Confidence: 1.0, // literal evidence - no inference, no doubt
	}

	out := []ReviewFinding{finding}
	out = append(out, findingsForWarnings(s, doc, allWarnings, severity)...)
	return out, nil
}

// synthesizerVersionFromBag reads the configured output-format version for
// a synthesizer out of the PerSynthesizer options bag. Shared by the review
// and polish hooks so both converge on the same IsFresh decision (code
// review finding #10).
func synthesizerVersionFromBag(bag map[string]map[string]any, name string) string {
	if bag == nil {
		return ""
	}
	perSynth, ok := bag[name]
	if !ok {
		return ""
	}
	if v, ok := perSynth["version"].(string); ok {
		return v
	}
	return ""
}

func findingsForWarnings(
	s synthesizer.Synthesizer,
	doc *synthesizer.Doc,
	warnings []synthesizer.Warning,
	severity string,
) []ReviewFinding {
	if len(warnings) == 0 {
		return nil
	}
	// Deduplicate warnings by code so we emit one finding per unique code
	// even if several candidates raised the same warning.
	seen := make(map[string]struct{})
	var out []ReviewFinding
	for _, w := range warnings {
		if _, dup := seen[w.Code]; dup {
			continue
		}
		seen[w.Code] = struct{}{}
		out = append(out, ReviewFinding{
			Severity:    severity,
			Title:       fmt.Sprintf("[%s] %s", s.Name(), w.Code),
			Description: w.Message,
			Documents:   []string{doc.Path},
			Confidence:  1.0,
		})
	}
	return out
}

// convertEvidence bridges synthesizer.Evidence (rich span info) to the
// review's Evidence (file + quote + line) used by the evidence validator.
// The Snippet field maps to Quote; ColStart/ColEnd are dropped because the
// review evidence model is line-granular.
func convertEvidence(evs []synthesizer.Evidence) []Evidence {
	out := make([]Evidence, len(evs))
	for i, ev := range evs {
		out[i] = Evidence{
			File:  ev.File,
			Quote: ev.Snippet,
			Line:  ev.Line,
		}
	}
	return out
}

// resolveSynthesizerSeverity returns the severity to use for synthesizer
// findings. The hierarchy is:
//
//  1. cfg.PerSynthesizer["review"]["severity"] (global override)
//  2. "info" (default per 2026-04-15 decision)
//
// Per-synthesizer overrides (each synthesizer declaring its own severity)
// are a future extension; the current framework uses the global override.
func resolveSynthesizerSeverity(cfg config.SynthesizersConfig) string {
	if review, ok := cfg.PerSynthesizer["review"]; ok {
		if sev, ok := review["severity"].(string); ok && sev != "" {
			return sev
		}
	}
	return "info"
}
