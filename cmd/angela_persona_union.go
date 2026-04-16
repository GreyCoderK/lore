// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"github.com/greycoderk/lore/internal/angela"
)

// filterScoredToPersona narrows a ResolvePersonas result down to the
// single persona with the matching identifier. Used by the --persona
// shortcut flag to force a single-lens polish/review run without touching
// the broader persona-selection machinery.
//
// Unknown names return an empty slice — the caller sees "no persona
// lens" rather than a silent fallback to auto selection, which would
// defeat the operator's explicit choice.
func filterScoredToPersona(scored []angela.ScoredPersona, name string) []angela.ScoredPersona {
	for _, sp := range scored {
		if sp.Profile.Name == name {
			return []angela.ScoredPersona{sp}
		}
	}
	// Not in scored — but the persona may still exist in the registry.
	// Build a synthetic ScoredPersona so the operator's explicit choice
	// still activates. Score 0 signals "forced, not signal-driven".
	for _, p := range angela.GetRegistry() {
		if p.Name == name {
			return []angela.ScoredPersona{{Profile: p, Score: 0}}
		}
	}
	return nil
}

// unionSignalDrivenPersonas returns base augmented with any persona from
// scored that isn't already present. Used by the draft pipeline to let
// content-signal-driven personas (like api-designer/Ouattara on a feature
// doc mentioning endpoints) run their DraftChecks alongside the
// type-driven selection from SelectPersonasForDoc.
//
// Each persona's DraftChecks gate internally on relevant content, so the
// union adds no false-positive suggestions on docs that don't match a
// given lens — only real coverage gaps are surfaced.
func unionSignalDrivenPersonas(base []angela.PersonaProfile, scored []angela.ScoredPersona) []angela.PersonaProfile {
	if len(scored) == 0 {
		return base
	}
	present := make(map[string]struct{}, len(base))
	for _, p := range base {
		present[p.Name] = struct{}{}
	}
	out := base
	for _, sp := range scored {
		if _, ok := present[sp.Profile.Name]; ok {
			continue
		}
		present[sp.Profile.Name] = struct{}{}
		out = append(out, sp.Profile)
	}
	return out
}
