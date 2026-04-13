// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"reflect"
	"testing"
)

// ═══════════════════════════════════════════════════════════════
// Story 8.5 tests: severity override + exit code resolver
// ═══════════════════════════════════════════════════════════════

// TestApplySeverityOverride_NoMatch verifies that a category without
// an override is passed through unchanged.
func TestApplySeverityOverride_NoMatch(t *testing.T) {
	in := []Suggestion{
		{Category: "structure", Severity: "warning", Message: "Section missing"},
	}
	out := ApplySeverityOverride(in, map[string]string{"style": "info"})
	if len(out) != 1 || out[0].Severity != "warning" {
		t.Errorf("expected pass-through, got %+v", out)
	}
}

// TestApplySeverityOverride_Downgrade verifies that overriding a
// category rewrites the severity in place.
func TestApplySeverityOverride_Downgrade(t *testing.T) {
	in := []Suggestion{
		{Category: "coherence", Severity: "warning", Message: "Duplicate detected"},
		{Category: "coherence", Severity: "warning", Message: "Another dup"},
		{Category: "structure", Severity: "warning", Message: "Untouched"},
	}
	out := ApplySeverityOverride(in, map[string]string{"coherence": "info"})

	if len(out) != 3 {
		t.Fatalf("expected 3 suggestions, got %d", len(out))
	}
	if out[0].Severity != "info" || out[1].Severity != "info" {
		t.Errorf("coherence not downgraded: %+v / %+v", out[0], out[1])
	}
	if out[2].Severity != "warning" {
		t.Errorf("structure should stay warning, got %q", out[2].Severity)
	}
}

// TestApplySeverityOverride_Off verifies that the "off" value drops
// matching suggestions entirely from the output slice.
func TestApplySeverityOverride_Off(t *testing.T) {
	in := []Suggestion{
		{Category: "coherence", Severity: "info", Message: "Drop me"},
		{Category: "structure", Severity: "warning", Message: "Keep me"},
		{Category: "coherence", Severity: "info", Message: "Drop me too"},
	}
	out := ApplySeverityOverride(in, map[string]string{"coherence": "off"})
	if len(out) != 1 {
		t.Fatalf("expected 1 suggestion, got %d: %+v", len(out), out)
	}
	if out[0].Category != "structure" {
		t.Errorf("wrong suggestion kept: %+v", out[0])
	}
}

// TestApplySeverityOverride_UnknownValueIgnored verifies that a typo in
// the override map doesn't crash — the suggestion is passed through
// with its original severity so a .lorerc typo fails gracefully.
func TestApplySeverityOverride_UnknownValueIgnored(t *testing.T) {
	in := []Suggestion{
		{Category: "coherence", Severity: "warning", Message: "Dup"},
	}
	out := ApplySeverityOverride(in, map[string]string{"coherence": "bloop"})
	if len(out) != 1 || out[0].Severity != "warning" {
		t.Errorf("expected pass-through on unknown override, got %+v", out)
	}
}

// TestApplySeverityOverride_EmptyInputs verifies that nil/empty inputs
// do not allocate or crash.
func TestApplySeverityOverride_EmptyInputs(t *testing.T) {
	if got := ApplySeverityOverride(nil, nil); got != nil {
		t.Errorf("nil input should return nil, got %v", got)
	}
	empty := []Suggestion{}
	if got := ApplySeverityOverride(empty, map[string]string{"x": "info"}); !reflect.DeepEqual(got, empty) {
		t.Errorf("empty suggestions should round-trip, got %v", got)
	}
	in := []Suggestion{{Category: "x", Severity: "info"}}
	if got := ApplySeverityOverride(in, nil); !reflect.DeepEqual(got, in) {
		t.Errorf("nil override should round-trip input, got %v", got)
	}
}

// TestApplySeverityOverride_CaseNormalization verifies that the override
// VALUE is case-normalized (so "OFF" == "off") but category key matches
// are exact (intentional — categories are fixed identifiers).
func TestApplySeverityOverride_CaseNormalization(t *testing.T) {
	in := []Suggestion{
		{Category: "coherence", Severity: "warning"},
	}
	out := ApplySeverityOverride(in, map[string]string{"coherence": "  OFF  "})
	if len(out) != 0 {
		t.Errorf("case-insensitive 'OFF' should drop, got %+v", out)
	}
}

// TestPromoteWarningsToErrors verifies that --strict upgrades all
// warnings to errors while leaving info findings untouched.
func TestPromoteWarningsToErrors(t *testing.T) {
	in := []Suggestion{
		{Category: "a", Severity: "info"},
		{Category: "b", Severity: "warning"},
		{Category: "c", Severity: "warning"},
		{Category: "d", Severity: "error"},
	}
	out := PromoteWarningsToErrors(in)
	wantSev := []string{"info", "error", "error", "error"}
	for i, s := range out {
		if s.Severity != wantSev[i] {
			t.Errorf("idx %d: got severity %q, want %q", i, s.Severity, wantSev[i])
		}
	}
}

// TestExitCodeFor_Never verifies that fail_on=never always returns 0
// regardless of finding severity.
func TestExitCodeFor_Never(t *testing.T) {
	in := []Suggestion{
		{Severity: "error"},
		{Severity: "warning"},
	}
	if got := ExitCodeFor(in, "never"); got != 0 {
		t.Errorf("never mode: got %d, want 0", got)
	}
}

// TestExitCodeFor_ErrorThreshold verifies the default mode:
// exit 2 only when an error-level finding is present.
func TestExitCodeFor_ErrorThreshold(t *testing.T) {
	tests := []struct {
		name  string
		in    []Suggestion
		want  int
	}{
		{"empty", nil, 0},
		{"info only", []Suggestion{{Severity: "info"}}, 0},
		{"warning only", []Suggestion{{Severity: "warning"}}, 0},
		{"error present", []Suggestion{{Severity: "warning"}, {Severity: "error"}}, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExitCodeFor(tc.in, "error"); got != tc.want {
				t.Errorf("fail_on=error %s: got %d, want %d", tc.name, got, tc.want)
			}
		})
	}
}

// TestExitCodeFor_WarningThreshold verifies that fail_on=warning
// exits 1 on warnings and 2 on errors.
func TestExitCodeFor_WarningThreshold(t *testing.T) {
	tests := []struct {
		name string
		in   []Suggestion
		want int
	}{
		{"info only", []Suggestion{{Severity: "info"}}, 0},
		{"warning only", []Suggestion{{Severity: "warning"}}, 1},
		{"warning + error", []Suggestion{{Severity: "warning"}, {Severity: "error"}}, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExitCodeFor(tc.in, "warning"); got != tc.want {
				t.Errorf("fail_on=warning %s: got %d, want %d", tc.name, got, tc.want)
			}
		})
	}
}

// TestExitCodeFor_InfoThreshold verifies that fail_on=info treats any
// finding as a failure.
func TestExitCodeFor_InfoThreshold(t *testing.T) {
	tests := []struct {
		name string
		in   []Suggestion
		want int
	}{
		{"empty", nil, 0},
		{"info only", []Suggestion{{Severity: "info"}}, 1},
		{"warning", []Suggestion{{Severity: "warning"}}, 1},
		{"error", []Suggestion{{Severity: "error"}}, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExitCodeFor(tc.in, "info"); got != tc.want {
				t.Errorf("fail_on=info %s: got %d, want %d", tc.name, got, tc.want)
			}
		})
	}
}

// TestExitCodeFor_UnknownThresholdDefaultsToError verifies that a
// typo in fail_on behaves like the documented default (error) rather
// than silently skipping the gate.
func TestExitCodeFor_UnknownThresholdDefaultsToError(t *testing.T) {
	in := []Suggestion{{Severity: "warning"}}
	if got := ExitCodeFor(in, "banana"); got != 0 {
		t.Errorf("unknown threshold should default to error (no fail on warnings), got %d", got)
	}
	errs := []Suggestion{{Severity: "error"}}
	if got := ExitCodeFor(errs, "banana"); got != 2 {
		t.Errorf("unknown threshold should default to error (fail on errors), got %d", got)
	}
}
