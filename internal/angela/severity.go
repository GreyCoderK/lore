// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import "strings"

// Severity level constants used by the draft pipeline. Keep these
// aligned with the Suggestion.Severity values produced by the analyzers
// (draft.go, coherence.go, persona.go, style.go).
const (
	SeverityInfo    = "info"
	SeverityWarning = "warning"
	SeverityError   = "error"
	severityOff     = "off" // sentinel for "drop this category entirely"
)

// draftSeverityRank maps a severity string to a comparable integer. Higher
// rank = more severe. Used to decide whether a given finding crosses a
// user-configured fail_on threshold.
func draftSeverityRank(sev string) int {
	switch strings.ToLower(sev) {
	case SeverityInfo:
		return 1
	case SeverityWarning:
		return 2
	case SeverityError:
		return 3
	default:
		return 0
	}
}

// ApplySeverityOverride mutates a slice of Suggestions according to a
// per-category override map. Semantics:
//
//   - Missing key  → suggestion unchanged
//   - Value "off"  → suggestion dropped entirely from the result
//   - Value "info" / "warning" / "error" → severity replaced in place
//
// The override map is typically populated from
// cfg.Angela.Draft.SeverityOverride and merged with any
// --severity flag values.
//
// The returned slice is a new allocation when any drop occurred; when
// no drops happen it may be the same backing array (callers should not
// depend on either behavior).
func ApplySeverityOverride(suggestions []Suggestion, override map[string]string) []Suggestion {
	if len(override) == 0 || len(suggestions) == 0 {
		return suggestions
	}
	out := make([]Suggestion, 0, len(suggestions))
	for _, s := range suggestions {
		newSev, ok := override[s.Category]
		if !ok {
			out = append(out, s)
			continue
		}
		normalized := strings.ToLower(strings.TrimSpace(newSev))
		if normalized == severityOff {
			continue // drop silently
		}
		// Unknown override values are ignored (pass through as-is) to
		// avoid making a typo in .lorerc crash the pipeline.
		if normalized != SeverityInfo && normalized != SeverityWarning && normalized != SeverityError {
			out = append(out, s)
			continue
		}
		s.Severity = normalized
		out = append(out, s)
	}
	return out
}

// PromoteWarningsToErrors walks a suggestion slice and upgrades every
// warning-level entry to error. Used by `draft --strict` mode where the
// user wants a zero-tolerance CI gate. Info-level findings are left
// untouched — strict is about promoting known problems, not inventing
// new ones.
func PromoteWarningsToErrors(suggestions []Suggestion) []Suggestion {
	for i := range suggestions {
		if suggestions[i].Severity == SeverityWarning {
			suggestions[i].Severity = SeverityError
		}
	}
	return suggestions
}

// ExitCodeFor returns the process exit code for a set of suggestions
// given a fail_on threshold.
//
// fail_on sets the MINIMUM severity that triggers a non-zero exit.
// Findings below the threshold are ignored for exit-code purposes.
// Example: fail_on=warning ignores info-level findings.
//
// Semantics:
//
//	fail_on = "never"   → always 0 (unless a hard error happens elsewhere)
//	fail_on = "info"    → exit 1 if info-or-warning findings, exit 2 if
//	                       any error-level finding (error trumps)
//	fail_on = "warning" → exit 1 if only warnings exist, exit 2 if any
//	                       error-level findings exist (error trumps warning
//	                       in exit code)
//	fail_on = "error"   → 2 if any error, else 0  (default)
//
// The distinction between 1 and 2 is important for CI pipelines that
// want to mark warnings as soft failures (exit 1) but errors as hard
// failures (exit 2).
func ExitCodeFor(suggestions []Suggestion, failOn string) int {
	failLevel := strings.ToLower(strings.TrimSpace(failOn))
	if failLevel == "never" {
		return 0
	}
	// Determine the highest severity present in the slice.
	maxRank := 0
	for _, s := range suggestions {
		r := draftSeverityRank(s.Severity)
		if r > maxRank {
			maxRank = r
		}
	}

	// Default fallback: treat unknown fail_on as "error".
	threshold := draftSeverityRank(SeverityError)
	switch failLevel {
	case SeverityInfo:
		threshold = draftSeverityRank(SeverityInfo)
	case SeverityWarning:
		threshold = draftSeverityRank(SeverityWarning)
	case SeverityError:
		threshold = draftSeverityRank(SeverityError)
	}

	if maxRank < threshold {
		return 0
	}
	if maxRank >= draftSeverityRank(SeverityError) {
		return 2
	}
	return 1
}
