// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package synthesizer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/greycoderk/lore/internal/domain"
)

// ComputeHash derives a stable, deterministic sha256 over the canonical form
// of evs. The hash is EVIDENCE-BASED, not output-based: it is a function of
// (File, Line, ColStart, ColEnd, Snippet) tuples sorted in a canonical order.
//
// Why evidence-based:
//
//   - I6 (idempotency): the synthesizer's output may include cosmetic version
//     bumps or template tweaks that should NOT trigger a regeneration. What
//     matters for "do we need to re-emit" is whether the source spans changed.
//   - I7 (no silent merge): the framework compares evidence hashes to decide
//     whether to propose a diff. Output-based hashing would erroneously flag
//     user-edited values as drift.
//
// The Field component of Evidence is included in the hash so that adding a
// new field (with new evidence) shifts the hash even when other evidences
// stayed put. Rule and Snippet are also included; if the underlying source
// text at the span changes (rename, doc edit), the snippet changes and the
// hash flips.
func ComputeHash(evs []Evidence) string {
	if len(evs) == 0 {
		return ""
	}
	canonical := canonicalEvidenceLines(evs)
	sum := sha256.Sum256([]byte(strings.Join(canonical, "\n")))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// canonicalEvidenceLines returns one line per evidence, sorted lexically, in
// a stable shape that downstream tools can also compute and compare. Format:
//
//	<File>|<Line>|<ColStart>|<ColEnd>|<Field>|<Rule>|<Snippet-escaped>
//
// Snippet is escaped so embedded pipes and newlines don't break the format.
func canonicalEvidenceLines(evs []Evidence) []string {
	out := make([]string, len(evs))
	for i, ev := range evs {
		out[i] = fmt.Sprintf(
			"%s|%d|%d|%d|%s|%s|%s",
			ev.File,
			ev.Line,
			ev.ColStart,
			ev.ColEnd,
			ev.Field,
			ev.Rule,
			escapeSnippet(ev.Snippet),
		)
	}
	sort.Strings(out)
	return out
}

func escapeSnippet(s string) string {
	r := strings.NewReplacer(
		`\`, `\\`,
		"|", `\|`,
		"\n", `\n`,
		"\r", `\r`,
	)
	return r.Replace(s)
}

// MakeSignature builds a Signature for a synthesizer run from its evidence
// list, the sections it consulted, and the warnings it emitted. Used by both
// the polish hook (when proposing a new block) and tests.
//
// version is the synthesizer's own output-format version (passed by the
// concrete synthesizer). At is the current UTC time formatted as RFC3339
// with seconds precision - sub-second jitter would defeat I6 on machines
// with high-resolution clocks.
func MakeSignature(version string, evs []Evidence, sections []string, warnings []Warning) Signature {
	codes := make([]string, 0, len(warnings))
	for _, w := range warnings {
		codes = append(codes, w.Code)
	}
	codes = SortedStrings(codes)
	sortedSections := SortedStrings(sections)
	return Signature{
		Hash:          ComputeHash(evs),
		At:            time.Now().UTC().Format(time.RFC3339),
		Version:       version,
		Sections:      sortedSections,
		EvidenceCount: len(evs),
		Warnings:      codes,
	}
}

// SignaturesFromMeta converts the generic frontmatter map (as stored on
// domain.DocMeta) into a typed map keyed by synthesizer name. Returns an
// empty map when the frontmatter has no synthesized key, never nil.
//
// Unknown extra fields inside a synthesizer's signature subtree are ignored
// silently - this lets newer binaries write fields that older binaries
// silently strip on round-trip without hard errors.
func SignaturesFromMeta(meta domain.DocMeta) map[string]Signature {
	out := make(map[string]Signature, len(meta.Synthesized))
	for name, raw := range meta.Synthesized {
		out[name] = decodeSignature(raw)
	}
	return out
}

func decodeSignature(raw map[string]any) Signature {
	sig := Signature{}
	if v, ok := raw["hash"].(string); ok {
		sig.Hash = v
	}
	if v, ok := raw["at"].(string); ok {
		sig.At = v
	}
	if v, ok := raw["version"].(string); ok {
		sig.Version = v
	}
	if v, ok := raw["evidence_count"].(int); ok {
		sig.EvidenceCount = v
	}
	if v, ok := raw["sections"].([]any); ok {
		for _, item := range v {
			if s, ok := item.(string); ok {
				sig.Sections = append(sig.Sections, s)
			}
		}
	}
	if v, ok := raw["warnings"].([]any); ok {
		for _, item := range v {
			if s, ok := item.(string); ok {
				sig.Warnings = append(sig.Warnings, s)
			}
		}
	}
	return sig
}

// SignaturesToMeta writes a typed signature map back into the generic
// frontmatter shape on domain.DocMeta. Empty input deletes the
// "synthesized" key (omitempty in the YAML tag handles serialization).
//
// The function MUTATES meta. Callers that need an unmodified copy should
// clone meta beforehand.
func SignaturesToMeta(meta *domain.DocMeta, signatures map[string]Signature) {
	if len(signatures) == 0 {
		meta.Synthesized = nil
		return
	}
	out := make(map[string]map[string]any, len(signatures))
	for name, sig := range signatures {
		entry := map[string]any{
			"hash":           sig.Hash,
			"at":             sig.At,
			"version":        sig.Version,
			"evidence_count": sig.EvidenceCount,
		}
		if len(sig.Sections) > 0 {
			entry["sections"] = stringSliceToAny(sig.Sections)
		}
		if len(sig.Warnings) > 0 {
			entry["warnings"] = stringSliceToAny(sig.Warnings)
		}
		out[name] = entry
	}
	meta.Synthesized = out
}

func stringSliceToAny(in []string) []any {
	out := make([]any, len(in))
	for i, s := range in {
		out[i] = s
	}
	return out
}

// IsFresh reports whether sig still matches the evidence list currentEvs.
// Used by the polish hook to decide skip vs regenerate.
//
// Returns false when:
//   - sig.Hash is empty (never synthesized)
//   - currentEvs hash differs from sig.Hash
//   - sig.Version differs from version (synthesizer output format bumped)
//
// Returns true only when both the source spans AND the format version are
// unchanged - guarantees identical regeneration on cache hit (I6).
func IsFresh(sig Signature, currentEvs []Evidence, version string) bool {
	if sig.Hash == "" {
		return false
	}
	if sig.Version != version {
		return false
	}
	return sig.Hash == ComputeHash(currentEvs)
}
