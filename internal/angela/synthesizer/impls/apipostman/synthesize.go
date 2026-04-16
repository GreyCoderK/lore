// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package apipostman

import (
	"fmt"
	"strings"

	"github.com/greycoderk/lore/internal/angela/synthesizer"
)

// buildBlocksForCandidate turns a Detect Candidate into exactly one
// Block (AC-10 format) plus the literal-evidence backing for every field
// it emits (I4). AC-11 asks for TWO variants per endpoint (minimal + full);
// because synthesizer.Synthesize returns a single Block, we emit the
// "full" variant as the primary Block and append the "minimal" variant
// as a second fenced snippet inside the same Block content (users can
// then copy whichever they need without a second round-trip).
//
// The composite is still byte-deterministic: field order, formatting, and
// note composition are canonical.
func buildBlocksForCandidate(c synthesizer.Candidate, cfg synthesizer.Config) (synthesizer.Block, []synthesizer.Evidence, []synthesizer.Warning, error) {
	ep, ok := c.Extra["endpoint"].(endpointHit)
	if !ok {
		return synthesizer.Block{}, nil, nil, fmt.Errorf("api-postman: candidate missing endpoint data")
	}
	fields, _ := c.Extra["fields"].([]fieldHit)
	serverInjected, _ := c.Extra["serverInjected"].(map[string]struct{})
	hasSecurity, _ := c.Extra["hasSecurity"].(bool)
	endpointsHeading, _ := c.Extra["endpointsHeading"].(string)

	projected, excluded, warnings := projectFields(fields, serverInjected, hasSecurity, cfg.WellKnownServerFields)

	// Assemble Block content. Output is STRICTLY deterministic - we rely on
	// projectFields preserving source order (required first, then optional)
	// and never recomputing with unstable iteration.
	var buf strings.Builder
	buf.WriteString("# Full — required + optional fields\n")
	writeRequestHead(&buf, ep.Method, ep.Path)
	writeJSONBody(&buf, projected, false /*fullVariant*/)

	// Minimal variant: only required fields.
	hasRequired := false
	for _, p := range projected {
		if p.Required {
			hasRequired = true
			break
		}
	}
	if hasRequired {
		buf.WriteString("\n\n# Minimal — required fields only\n")
		writeRequestHead(&buf, ep.Method, ep.Path)
		writeJSONBody(&buf, projected, true /*minimalVariant*/)
	}

	evidence := collectEvidence(c.Anchor, projected)
	notes := buildNotes(excluded, hasSecurity)

	block := synthesizer.Block{
		Title:              fmt.Sprintf("%s %s — exemple d'appel", ep.Method, ep.Path),
		Language:           "http",
		Content:            buf.String(),
		Notes:              notes,
		InsertAfterHeading: endpointsHeading,
	}
	return block, evidence, warnings, nil
}

// projectedField is an internal field after projection (security filter +
// ordering). It carries back-references to the source fieldHit so we can
// emit evidence at the literal source span.
type projectedField struct {
	Source   fieldHit
	Required bool
	// Position 0..N-1 used for deterministic ordering. Required fields get
	// low positions (0..R-1), optional fields get R..R+O-1 in source order.
	Order int
}

// projectFields applies I5 / I5-bis filtering and computes the stable
// ordering (required first, then optional, both in source order).
//
// The returned `excluded` slice lists the names removed by the security
// filter - used for the "injected server-side" note under the block.
func projectFields(
	fields []fieldHit,
	serverInjected map[string]struct{},
	hasSecurity bool,
	wellKnown []string,
) (projected []projectedField, excluded []string, warnings []synthesizer.Warning) {
	wellKnownSet := make(map[string]struct{}, len(wellKnown))
	for _, n := range wellKnown {
		wellKnownSet[n] = struct{}{}
	}

	kept := make([]fieldHit, 0, len(fields))
	excludedSet := make(map[string]struct{})
	for _, f := range fields {
		// I5 — explicit security exclusion.
		if _, hit := serverInjected[f.Name]; hit {
			excludedSet[f.Name] = struct{}{}
			continue
		}
		// I5-bis — degraded mode filters the well-known list AND flags.
		if !hasSecurity {
			if _, hit := wellKnownSet[f.Name]; hit {
				excludedSet[f.Name] = struct{}{}
				continue
			}
		}
		kept = append(kept, f)
	}
	if len(excludedSet) > 0 {
		for n := range excludedSet {
			excluded = append(excluded, n)
		}
	}
	excluded = synthesizer.SortedStrings(excluded)

	if !hasSecurity {
		warnings = append(warnings, synthesizer.Warning{
			Code:    synthesizer.WarningCodeMissingSecuritySection,
			Message: "Doc has endpoints but no Security section — degraded mode applied well-known well_known_server_fields filter. Add a Security section listing server-injected fields for strict projection.",
		})
	}

	// Deterministic order: required-in-source-order, then optional-in-source-order.
	var required, optional []fieldHit
	for _, f := range kept {
		if f.Required {
			required = append(required, f)
		} else {
			optional = append(optional, f)
		}
	}
	projected = make([]projectedField, 0, len(kept))
	for i, f := range required {
		projected = append(projected, projectedField{Source: f, Required: true, Order: i})
	}
	for i, f := range optional {
		projected = append(projected, projectedField{Source: f, Required: false, Order: len(required) + i})
	}
	return projected, excluded, warnings
}

// writeRequestHead renders the HTTP verb, URL, and standard headers.
func writeRequestHead(buf *strings.Builder, method, path string) {
	buf.WriteString(method)
	buf.WriteByte(' ')
	buf.WriteString("{{baseUrl}}")
	buf.WriteString(path)
	buf.WriteByte('\n')
	buf.WriteString("Authorization: Bearer {{jwt}}\n")
	buf.WriteString("Content-Type: application/json\n")
	buf.WriteByte('\n')
}

// writeJSONBody renders the JSON body. minimalVariant == true emits only
// required fields as variables. The output uses 2-space indent, no
// trailing whitespace, stable key order.
func writeJSONBody(buf *strings.Builder, projected []projectedField, minimalVariant bool) {
	buf.WriteString("{\n")
	first := true
	for _, p := range projected {
		if minimalVariant && !p.Required {
			continue
		}
		if !first {
			buf.WriteString(",\n")
		}
		first = false
		buf.WriteString("  ")
		buf.WriteByte('"')
		buf.WriteString(p.Source.Name)
		buf.WriteString(`": `)
		if p.Required {
			buf.WriteByte('"')
			buf.WriteString("{{")
			buf.WriteString(p.Source.Name)
			buf.WriteString("}}")
			buf.WriteByte('"')
		} else {
			buf.WriteString("null")
		}
	}
	if !first {
		buf.WriteByte('\n')
	}
	buf.WriteString("}")
}

// collectEvidence produces one Evidence per projected field (output key).
// For Min/Max-expanded fields, the evidence points at the (+Min/Max)
// trigger - its literal text contains "Min" or "Max" so I4 holds
// ("the token Min appears literally in the source").
//
// The anchor evidence (endpoint-scoped, Field == "__endpoint__") is
// attached at position 0 so callers who want a canonical span for the
// endpoint heading find it quickly.
func collectEvidence(anchor synthesizer.Evidence, projected []projectedField) []synthesizer.Evidence {
	out := make([]synthesizer.Evidence, 0, len(projected)+1)
	// Anchor for the endpoint itself. The Field is set to the endpoint
	// pseudo-key so I4 still holds (every emitted key has evidence) even
	// though the endpoint isn't in the JSON body.
	out = append(out, anchor)

	for _, p := range projected {
		if p.Source.IsMinMaxBase {
			// Evidence for expanded fields points at the trigger span -
			// the literal tokens "Min" or "Max" inside "(+Min/Max)".
			triggerLine := p.Source.TriggerSpan[0]
			triggerStart := p.Source.TriggerSpan[1]
			triggerEnd := p.Source.TriggerSpan[2]
			out = append(out, synthesizer.Evidence{
				Field:    p.Source.Name,
				File:     anchor.File,
				Line:     triggerLine,
				ColStart: triggerStart,
				ColEnd:   triggerEnd,
				Snippet:  p.Source.TriggerToken,
				Rule:     "literal",
			})
			continue
		}
		out = append(out, synthesizer.Evidence{
			Field:    p.Source.Name,
			File:     anchor.File,
			Line:     p.Source.Line,
			ColStart: p.Source.ColStart,
			ColEnd:   p.Source.ColEnd,
			Snippet:  fieldSnippet(anchor.File, p.Source),
			Rule:     "literal",
		})

		// The fieldHit.markRequired helper is the only sanctioned way to
		// flip Required=true, and it refuses to set the flag without a
		// valid span. If we ever see Required=true with a zero span here,
		// it's a programming error we want to surface in tests rather
		// than skip silently (code review finding #6).
		if p.Required {
			if p.Source.RequiredSpan[0] == 0 {
				continue
			}
			out = append(out, synthesizer.Evidence{
				Field:    synthesizer.RequiredFieldKey(p.Source.Name),
				File:     anchor.File,
				Line:     p.Source.RequiredSpan[0],
				ColStart: p.Source.RequiredSpan[1],
				ColEnd:   p.Source.RequiredSpan[2],
				Snippet:  p.Source.RequiredToken,
				Rule:     "literal",
			})
		}
	}
	return out
}

// fieldSnippet returns the literal text at the field's ColStart:ColEnd on
// its declared line. We store it in Evidence.Snippet so I4's validator
// can compare without re-reading the file.
func fieldSnippet(_ string, f fieldHit) string {
	// The framework validates Snippet against doc.Lines[line][start:end] -
	// we just store the raw name characters, which is what the source has
	// between the backticks. The backticks themselves are NOT in the span
	// (match regex captures group 1 inside the ticks).
	return f.Name
}

func buildNotes(excluded []string, hasSecurity bool) []string {
	var notes []string
	if len(excluded) > 0 {
		kind := "Security section"
		if !hasSecurity {
			kind = "well_known_server_fields list"
		}
		notes = append(notes, fmt.Sprintf(
			"Champs injectés côté serveur (filtrés depuis la %s) : %s",
			kind, strings.Join(excluded, ", "),
		))
	}
	notes = append(notes,
		"Les champs requis sont des variables Postman `{{nom}}` — déclarez-les dans votre environnement.",
		"Les champs optionnels valent `null` par défaut — remplacez par la valeur désirée.",
	)
	if !hasSecurity {
		notes = append(notes,
			"⚠️ Aucune section \"Sécurité\" détectée dans le doc. Le filtrage des champs serveur repose sur la liste well_known_server_fields — ajoute une section \"Sécurité\" explicite pour garantir l'exclusion des autres champs injectés.",
		)
	}
	return notes
}
