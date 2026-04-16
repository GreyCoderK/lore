// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package synthesizer

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

// WarningCodeMissingSecuritySection is emitted by synthesizers running in
// degraded mode (I5-bis) - the source doc declares endpoints but lacks a
// Security section, so the framework relies on the well-known list to
// project server-injected fields out of the output.
const WarningCodeMissingSecuritySection = "missing-security-section"

// ContractFixture describes a single test case for the invariant contract
// suite. Concrete synthesizer test packages provide a slice of fixtures and
// invoke RunInvariantContracts to exercise the framework's I4-I7 contracts
// against their implementation.
type ContractFixture struct {
	// Name identifies the case in test output (e.g.
	// "complete-with-security", "no-security-section").
	Name string

	// Path is the synthetic file path attached to the parsed Doc.
	Path string

	// Content is the raw markdown (frontmatter + body) the framework will
	// parse with ParseDoc.
	Content string

	// ServerInjected lists the field names the doc explicitly declares as
	// server-injected (typically extracted from the Security section). The
	// I5 contract asserts NONE of these appear in the synthesizer's
	// Evidence.Field list. Empty when the fixture has no Security section.
	ServerInjected []string

	// WellKnown is the list passed to Config.WellKnownServerFields when
	// running this fixture. The I5-bis contract uses this list to assert
	// the degraded mode filter actually fires.
	WellKnown []string

	// ExpectMissingSecurity declares whether this fixture lacks a Security
	// section. When true, the I5-bis contract is exercised: every
	// Synthesize call MUST emit a Warning with Code ==
	// WarningCodeMissingSecuritySection.
	ExpectMissingSecurity bool

	// MinCandidates is the minimum number of Candidates Detect must return
	// for this fixture. Zero disables the check.
	MinCandidates int
}

// RunInvariantContracts exercises the I4-I7 invariants against a concrete
// synthesizer using the provided fixtures. Concrete test packages call this
// at the top of their test suite to inherit the framework's safety net.
//
// Calling pattern (from impls/<name>/<name>_test.go):
//
//	func TestAPIPostman_Contracts(t *testing.T) {
//	    fixtures := []synthesizer.ContractFixture{ ... }
//	    synthesizer.RunInvariantContracts(t, &APIPostman{}, fixtures)
//	}
//
// The function uses subtests so a single failed invariant on a single
// fixture surfaces as one targeted failure (e.g.
// "TestAPIPostman_Contracts/no-security/I5bis_warning_emitted").
func RunInvariantContracts(t *testing.T, s Synthesizer, fixtures []ContractFixture) {
	t.Helper()
	if s == nil {
		t.Fatal("RunInvariantContracts: nil synthesizer")
	}
	for _, fx := range fixtures {
		fx := fx
		t.Run(fx.Name, func(t *testing.T) {
			runContractFixture(t, s, fx)
		})
	}
}

func runContractFixture(t *testing.T, s Synthesizer, fx ContractFixture) {
	t.Helper()

	report, err := EvaluateFixture(s, fx)
	if err != nil {
		t.Fatalf("evaluate fixture: %v", err)
	}
	for name, err := range report.Results {
		name := name
		err := err
		t.Run(name, func(sub *testing.T) {
			if err != nil {
				sub.Fatal(err.Error())
			}
		})
	}
}

// FixtureReport captures the per-invariant outcome of EvaluateFixture. nil
// values mean "passed". The map keys are stable subtest names so callers
// (and CI consumers) can assert a specific invariant.
type FixtureReport struct {
	Results map[string]error
}

// EvaluateFixture runs a synthesizer against a single fixture and returns
// per-invariant results without touching testing.T. RunInvariantContracts
// uses this internally; negative-path tests can call EvaluateFixture
// directly to assert that a deliberately-broken synthesizer fails the
// expected invariant.
func EvaluateFixture(s Synthesizer, fx ContractFixture) (FixtureReport, error) {
	report := FixtureReport{Results: make(map[string]error)}

	doc, err := ParseDoc(fx.Path, []byte(fx.Content))
	if err != nil {
		return report, fmt.Errorf("ParseDoc: %w", err)
	}

	if !s.Applies(doc) {
		return report, fmt.Errorf("Applies returned false on fixture %q", fx.Name)
	}

	candidates, err := s.Detect(doc)
	if err != nil {
		return report, fmt.Errorf("Detect: %w", err)
	}
	if fx.MinCandidates > 0 && len(candidates) < fx.MinCandidates {
		return report, fmt.Errorf("Detect returned %d candidates, expected at least %d", len(candidates), fx.MinCandidates)
	}
	if len(candidates) == 0 {
		return report, fmt.Errorf("synthesizer returned no candidates - I4-I7 contracts cannot run")
	}

	cfg := Config{WellKnownServerFields: fx.WellKnown}

	blocks1, evs1, warns1, err := synthesizeAll(s, candidates, cfg)
	if err != nil {
		return report, fmt.Errorf("Synthesize run #1: %w", err)
	}

	report.Results["I4_evidence_literal_and_matches_source"] = checkI4(doc, evs1)
	report.Results["I5_no_server_field_in_output"] = checkI5(evs1, fx.ServerInjected)

	if fx.ExpectMissingSecurity {
		report.Results["I5bis_degraded_mode_filters_wellknown"] = checkI5bis(evs1, warns1, fx.WellKnown)
	}

	blocks2, _, _, err := synthesizeAll(s, candidates, cfg)
	if err != nil {
		return report, fmt.Errorf("Synthesize run #2: %w", err)
	}
	report.Results["I6_idempotent_blocks_byte_identical"] = checkI6(blocks1, blocks2)
	report.Results["I7_signature_flips_when_source_changes"] = checkI7(evs1)

	return report, nil
}

func synthesizeAll(s Synthesizer, candidates []Candidate, cfg Config) ([]Block, []Evidence, []Warning, error) {
	var blocks []Block
	var evs []Evidence
	var warns []Warning
	for _, c := range candidates {
		b, e, w, err := s.Synthesize(c, cfg)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("Synthesize(%q): %w", c.Key, err)
		}
		blocks = append(blocks, b)
		evs = append(evs, e...)
		warns = append(warns, w...)
	}
	return blocks, evs, warns, nil
}

func checkI4(doc *Doc, evs []Evidence) error {
	if len(evs) == 0 {
		return fmt.Errorf("synthesizer returned no evidence - I4 cannot hold (every output field needs >=1 literal evidence)")
	}
	for i, ev := range evs {
		if ev.Rule != "literal" {
			return fmt.Errorf("evidence #%d (Field=%q) has Rule=%q, MVP v1 requires \"literal\"", i, ev.Field, ev.Rule)
		}
		if ev.Field == "" {
			return fmt.Errorf("evidence #%d has empty Field - I4 requires every output key to be tagged", i)
		}
		if ev.Line < 1 || ev.Line >= len(doc.Lines) {
			return fmt.Errorf("evidence #%d (Field=%q): Line=%d out of range [1,%d)", i, ev.Field, ev.Line, len(doc.Lines))
		}
		line := doc.Lines[ev.Line]
		if ev.ColStart < 0 || ev.ColEnd > len(line) || ev.ColStart > ev.ColEnd {
			return fmt.Errorf("evidence #%d (Field=%q): span [%d:%d] out of range for line len %d",
				i, ev.Field, ev.ColStart, ev.ColEnd, len(line))
		}
		actual := line[ev.ColStart:ev.ColEnd]
		if actual != ev.Snippet {
			return fmt.Errorf("evidence #%d (Field=%q): Snippet drift — claimed %q, source has %q at %s:%d[%d:%d]",
				i, ev.Field, ev.Snippet, actual, doc.Path, ev.Line, ev.ColStart, ev.ColEnd)
		}
	}
	return nil
}

func checkI5(evs []Evidence, serverInjected []string) error {
	if len(serverInjected) == 0 {
		return nil
	}
	blocked := make(map[string]struct{}, len(serverInjected))
	for _, name := range serverInjected {
		blocked[name] = struct{}{}
	}
	for _, ev := range evs {
		if _, hit := blocked[ev.Field]; hit {
			return fmt.Errorf("I5 violation: server-injected field %q present in synthesizer output (evidence at %s:%d)",
				ev.Field, ev.File, ev.Line)
		}
	}
	return nil
}

func checkI5bis(evs []Evidence, warns []Warning, wellKnown []string) error {
	blocked := make(map[string]struct{}, len(wellKnown))
	for _, name := range wellKnown {
		blocked[name] = struct{}{}
	}
	for _, ev := range evs {
		if _, hit := blocked[ev.Field]; hit {
			return fmt.Errorf("I5-bis violation: well-known server field %q present in degraded-mode output", ev.Field)
		}
	}
	for _, w := range warns {
		if w.Code == WarningCodeMissingSecuritySection {
			return nil
		}
	}
	return fmt.Errorf("I5-bis violation: degraded mode must emit %q warning, got %v",
		WarningCodeMissingSecuritySection, warningCodes(warns))
}

func warningCodes(warns []Warning) []string {
	codes := make([]string, len(warns))
	for i, w := range warns {
		codes[i] = w.Code
	}
	sort.Strings(codes)
	return codes
}

func checkI6(first, second []Block) error {
	if len(first) != len(second) {
		return fmt.Errorf("I6 violation: candidate count drifted: run1=%d, run2=%d", len(first), len(second))
	}
	for i := range first {
		if !reflect.DeepEqual(first[i], second[i]) {
			return fmt.Errorf("I6 violation: block #%d drifted between runs.\nrun1: %+v\nrun2: %+v",
				i, first[i], second[i])
		}
	}
	return nil
}

func checkI7(evs []Evidence) error {
	if len(evs) == 0 {
		return nil
	}
	original := MakeSignature("contract-test", evs, nil, nil)
	mutated := make([]Evidence, len(evs))
	copy(mutated, evs)
	mutated[0].Snippet = mutated[0].Snippet + "_DRIFT"
	if IsFresh(original, mutated, "contract-test") {
		return fmt.Errorf("I7 violation: framework reports fresh after source edit - polish would silently overwrite")
	}
	return nil
}
