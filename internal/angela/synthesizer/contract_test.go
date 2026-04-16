// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package synthesizer

import (
	"strings"
	"testing"
)

// docSynth is a self-test stub: it emits one Candidate per fixture and one
// Evidence per detected field. Its only purpose is to exercise the
// invariant contract suite itself - the real implementations live under
// impls/<name>/.
type docSynth struct {
	// fields is the ordered list of fields the stub will declare in its
	// Synthesize output. Each must appear literally in the fixture body so
	// the I4 assertion can locate a matching span.
	fields []string

	// emitMissingSecurity controls whether Synthesize emits the I5-bis
	// warning. The stub does NOT actually inspect the doc - it trusts the
	// test setup to align this with the fixture's ExpectMissingSecurity.
	emitMissingSecurity bool
}

func (d *docSynth) Name() string { return "doc-stub" }

func (d *docSynth) Applies(*Doc) bool { return true }

func (d *docSynth) Detect(doc *Doc) ([]Candidate, error) {
	return []Candidate{{Key: "stub-1"}}, nil
}

func (d *docSynth) Synthesize(c Candidate, cfg Config) (Block, []Evidence, []Warning, error) {
	// Locate each field in the doc and emit one literal Evidence per match.
	doc := stubLastDoc
	if doc == nil {
		return Block{}, nil, nil, nil
	}
	var evs []Evidence
	for _, field := range d.fields {
		ev, ok := findFirst(doc, field)
		if !ok {
			continue
		}
		evs = append(evs, ev)
	}
	var warns []Warning
	if d.emitMissingSecurity {
		warns = append(warns, Warning{Code: WarningCodeMissingSecuritySection, Message: "no security section"})
	}
	block := Block{
		Title:    "stub",
		Language: "json",
		Content:  "{}",
	}
	return block, evs, warns, nil
}

// stubLastDoc is set by the test harness via ParseDoc-then-call. The
// docSynth stub is intentionally minimal; real synthesizers carry doc
// references through Candidate.Extra.
var stubLastDoc *Doc

func findFirst(doc *Doc, field string) (Evidence, bool) {
	for ln, line := range doc.Lines {
		if ln == 0 {
			continue
		}
		idx := strings.Index(line, field)
		if idx < 0 {
			continue
		}
		return Evidence{
			Field:    field,
			File:     doc.Path,
			Line:     ln,
			ColStart: idx,
			ColEnd:   idx + len(field),
			Snippet:  field,
			Rule:     "literal",
		}, true
	}
	return Evidence{}, false
}

const fixtureWithSecurity = `---
type: feature
date: "2026-04-15"
status: draft
---

# Feature

## Endpoints

- POST /api/foo with month and amount

## Security

server-injected: corporateCode
`

const fixtureNoSecurity = `---
type: feature
date: "2026-04-15"
status: draft
---

# Feature

## Endpoints

- POST /api/foo with month and corporateCode
`

func TestRunInvariantContracts_PassesWithGoodStub(t *testing.T) {
	stub := &docSynth{fields: []string{"month", "amount"}}

	// Inject doc reference because the stub uses the package-level cache.
	doc, err := ParseDoc("with-sec.md", []byte(fixtureWithSecurity))
	if err != nil {
		t.Fatal(err)
	}
	stubLastDoc = doc
	t.Cleanup(func() { stubLastDoc = nil })

	fixtures := []ContractFixture{
		{
			Name:           "complete",
			Path:           "with-sec.md",
			Content:        fixtureWithSecurity,
			ServerInjected: []string{"corporateCode"},
			MinCandidates:  1,
		},
	}
	RunInvariantContracts(t, stub, fixtures)
}

func TestRunInvariantContracts_DegradedFixtureRequiresWarning(t *testing.T) {
	// First with the stub correctly emitting the warning - should pass.
	stub := &docSynth{fields: []string{"month"}, emitMissingSecurity: true}

	doc, err := ParseDoc("no-sec.md", []byte(fixtureNoSecurity))
	if err != nil {
		t.Fatal(err)
	}
	stubLastDoc = doc
	t.Cleanup(func() { stubLastDoc = nil })

	fixtures := []ContractFixture{
		{
			Name:                  "no-security",
			Path:                  "no-sec.md",
			Content:               fixtureNoSecurity,
			WellKnown:             []string{"corporateCode"},
			ExpectMissingSecurity: true,
			MinCandidates:         1,
		},
	}
	RunInvariantContracts(t, stub, fixtures)
}

func TestEvaluateFixture_FailsWhenStubSkipsWarning(t *testing.T) {
	stub := &docSynth{fields: []string{"month"}, emitMissingSecurity: false}

	doc, err := ParseDoc("no-sec.md", []byte(fixtureNoSecurity))
	if err != nil {
		t.Fatal(err)
	}
	stubLastDoc = doc
	t.Cleanup(func() { stubLastDoc = nil })

	fx := ContractFixture{
		Name:                  "no-security",
		Path:                  "no-sec.md",
		Content:               fixtureNoSecurity,
		WellKnown:             []string{"corporateCode"},
		ExpectMissingSecurity: true,
		MinCandidates:         1,
	}
	report, err := EvaluateFixture(stub, fx)
	if err != nil {
		t.Fatalf("EvaluateFixture: %v", err)
	}
	if got := report.Results["I5bis_degraded_mode_filters_wellknown"]; got == nil {
		t.Fatal("expected I5-bis to fail when stub omits the warning")
	}
}

func TestEvaluateFixture_FailsWhenEvidenceSnippetDrifts(t *testing.T) {
	stub := &corruptSynth{}

	doc, err := ParseDoc("d.md", []byte(fixtureWithSecurity))
	if err != nil {
		t.Fatal(err)
	}
	stubLastDoc = doc
	t.Cleanup(func() { stubLastDoc = nil })

	fx := ContractFixture{Name: "drift", Path: "d.md", Content: fixtureWithSecurity, MinCandidates: 1}
	report, err := EvaluateFixture(stub, fx)
	if err != nil {
		t.Fatalf("EvaluateFixture: %v", err)
	}
	if got := report.Results["I4_evidence_literal_and_matches_source"]; got == nil {
		t.Fatal("expected I4 to fail when stub returns drifted snippet")
	}
}

type corruptSynth struct{}

func (corruptSynth) Name() string                      { return "corrupt" }
func (corruptSynth) Applies(*Doc) bool                 { return true }
func (corruptSynth) Detect(*Doc) ([]Candidate, error)  { return []Candidate{{Key: "x"}}, nil }
func (corruptSynth) Synthesize(Candidate, Config) (Block, []Evidence, []Warning, error) {
	return Block{}, []Evidence{
		{
			Field: "month", File: "d.md", Line: 1, ColStart: 0, ColEnd: 5,
			Snippet: "WRONG", // does not match the source line at 0:5
			Rule:    "literal",
		},
	}, nil, nil
}
