// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package angela — evidence_validator.go
//
// Reject hallucinated findings via evidence validation.
//
// The AI sometimes invents plausible-sounding contradictions that do not
// actually exist in the corpus. This validator runs after the AI response
// is parsed: for every ReviewFinding it walks the Evidence array and
// checks, file-by-file, that each quoted snippet literally appears in the
// corresponding document (after whitespace normalization). Findings that
// fail any check are pulled out of the kept pile and reported separately
// so the user can see WHY they were dropped.
//
// The validator is pure Go and deterministic — no AI round-trip. It
// reads the corpus via the existing domain.CorpusReader so it works
// transparently in both lore-native and standalone modes.
package angela

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/greycoderk/lore/internal/domain"
)

// Validation strictness values for ReviewOpts.Evidence.Mode. Match the
// public config names exactly (cfg.Angela.Review.Evidence.Validation).
const (
	EvidenceModeStrict  = "strict"  // reject failing findings (default)
	EvidenceModeLenient = "lenient" // keep findings but record rejection reason
	EvidenceModeOff     = "off"     // skip validation entirely
)

// EvidenceValidation carries the subset of cfg.Angela.Review.Evidence
// that the angela package needs, so the package does not have to import
// internal/config (preserving the layering boundary).
type EvidenceValidation struct {
	// Required enables validation. When false, the validator is a no-op
	// and every finding passes through unchanged.
	Required bool

	// MinConfidence filters findings whose AI-reported confidence is
	// below this threshold (0.0 - 1.0). Only applied when Required.
	MinConfidence float64

	// Mode is one of EvidenceModeStrict / EvidenceModeLenient / EvidenceModeOff.
	// Empty defaults to strict for safety.
	Mode string
}

// RejectedFinding wraps a ReviewFinding that failed validation along with
// a human-readable reason. The CLI surfaces these in verbose mode.
type RejectedFinding struct {
	Finding ReviewFinding `json:"finding"`
	Reason  string        `json:"reason"`
}

// ValidationResult is the outcome of ValidateFindings: the kept findings
// and the pulled-aside rejects. Both slices are non-nil (possibly empty)
// so callers can freely JSON-marshal or iterate without nil checks.
type ValidationResult struct {
	Valid    []ReviewFinding   `json:"valid"`
	Rejected []RejectedFinding `json:"rejected"`
}

// wsRe collapses any run of whitespace (spaces, tabs, newlines) to a
// single space. Shared across the package so compiling the regex happens
// exactly once at import time.
var wsRe = regexp.MustCompile(`\s+`)

// normalizeWhitespace flattens all whitespace runs to single spaces and
// trims the edges. Quote matching is case-preserving and does NOT apply
// unicode normalization — explicitly scoped out of MVP (can be added
// if false rejects surface in practice).
func normalizeWhitespace(s string) string {
	return strings.TrimSpace(wsRe.ReplaceAllString(s, " "))
}

// ValidateFindings walks every finding and checks its Evidence against
// the corpus via `reader`. A finding is kept when all of the following
// hold:
//
//  1. Evidence is non-empty
//  2. Every Evidence.File exists in the corpus (reader.ReadDoc succeeds)
//  3. Every Evidence.Quote appears in its File content after
//     whitespace normalization
//  4. Confidence >= v.MinConfidence (only when v.Required is true)
//
// When v.Required is false OR v.Mode is "off", the validator is a total
// no-op and every finding is returned as Valid with an empty Rejected
// slice — this is the backward-compat escape hatch.
//
// When v.Mode is "lenient", a failing finding is KEPT in Valid but also
// recorded in Rejected with its reason, so the CLI can surface the
// drop-rationale without dropping the finding. This is the debug mode.
//
// Document content is cached per filename across the loop so a finding
// that cites the same document three times doesn't trigger three disk
// reads.
func ValidateFindings(findings []ReviewFinding, reader domain.CorpusReader, v EvidenceValidation) ValidationResult {
	result := ValidationResult{
		Valid:    make([]ReviewFinding, 0, len(findings)),
		Rejected: make([]RejectedFinding, 0),
	}

	// "off" and !Required are the same no-op path, written twice for
	// defensive clarity at the callsite.
	if !v.Required || strings.EqualFold(v.Mode, EvidenceModeOff) {
		result.Valid = append(result.Valid, findings...)
		return result
	}

	// Per-call cache: filename → content. nil reader means "no corpus
	// available" — every file-backed check becomes a missing-file
	// rejection, which is the right answer (without a reader we cannot
	// prove the quotes exist).
	contentCache := make(map[string]string)
	// Cache normalized content per file to avoid redundant normalizeWhitespace calls.
	normalizedCache := make(map[string]string)
	readContent := func(file string) (string, bool) {
		if reader == nil {
			return "", false
		}
		if cached, ok := contentCache[file]; ok {
			return cached, true
		}
		content, err := reader.ReadDoc(file)
		if err != nil {
			return "", false
		}
		contentCache[file] = content
		return content, true
	}
	readNormalized := func(file string) (string, bool) {
		if n, ok := normalizedCache[file]; ok {
			return n, true
		}
		content, ok := readContent(file)
		if !ok {
			return "", false
		}
		n := normalizeWhitespace(content)
		normalizedCache[file] = n
		return n, true
	}

	mode := strings.ToLower(strings.TrimSpace(v.Mode))
	if mode == "" {
		mode = EvidenceModeStrict
	}

	for _, f := range findings {
		reason := validateOne(f, v, readContent, readNormalized)
		if reason == "" {
			result.Valid = append(result.Valid, f)
			continue
		}
		rejection := RejectedFinding{Finding: f, Reason: reason}
		if mode == EvidenceModeLenient {
			// Keep it in the output but record why it would have been
			// dropped under strict mode.
			result.Valid = append(result.Valid, f)
		}
		result.Rejected = append(result.Rejected, rejection)
	}

	return result
}

// validateOne runs the four checks on a single finding and returns an
// empty string when the finding passes, or a human-readable failure
// reason otherwise. Separated from the main loop to keep the control
// flow in ValidateFindings flat.
//
// Whitespace-only quotes are normalized first and rejected if they
// collapse to empty (prevents bypass via `strings.Contains(x, "")`).
//
// NaN / +Inf / -Inf confidence values are explicitly rejected via
// `math.IsNaN` / `math.IsInf`. The threshold applies even when
// MinConfidence is zero (zero means "any finite confidence", not
// "no check").
func validateOne(f ReviewFinding, v EvidenceValidation, readContent func(string) (string, bool), readNormalized func(string) (string, bool)) string {
	// Check 1 — evidence presence
	if len(f.Evidence) == 0 {
		return "no evidence provided"
	}

	// Check 4 — confidence sanity + threshold.
	if math.IsNaN(f.Confidence) || math.IsInf(f.Confidence, 0) {
		return fmt.Sprintf("confidence is not a finite number (%v)", f.Confidence)
	}
	if f.Confidence < v.MinConfidence {
		return fmt.Sprintf("confidence %.2f below threshold %.2f", f.Confidence, v.MinConfidence)
	}

	// Checks 2 + 3 — file existence and quote presence
	for _, ev := range f.Evidence {
		if ev.File == "" {
			return "evidence missing file reference"
		}
		nq := normalizeWhitespace(ev.Quote)
		if nq == "" {
			return fmt.Sprintf("empty quote for %s", ev.File)
		}
		normalized, ok := readNormalized(ev.File)
		if !ok {
			return fmt.Sprintf("file %s does not exist", ev.File)
		}
		if !strings.Contains(normalized, nq) {
			// Unicode NFC/NFKD normalization is not applied
			// (golang.org/x/text dependency deferred past MVP). If the
			// quote contains non-ASCII, a mismatch may be a normalization
			// artifact rather than a true hallucination — downgrade to a
			// softer message so strict mode doesn't hard-reject valid
			// evidence that differs only in Unicode representation.
			if containsNonASCII(nq) {
				return fmt.Sprintf("possible Unicode normalization mismatch for quote in %s (non-ASCII content)", ev.File)
			}
			return fmt.Sprintf("quote not found in %s", ev.File)
		}
	}

	return ""
}

// containsNonASCII reports whether s contains any byte > 127.
func containsNonASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return true
		}
	}
	return false
}
