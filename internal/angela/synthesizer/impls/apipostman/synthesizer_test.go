// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package apipostman

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/angela/synthesizer"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func parseFixture(t *testing.T, name string) *synthesizer.Doc {
	t.Helper()
	doc, err := synthesizer.ParseDoc(name, readFixture(t, name))
	if err != nil {
		t.Fatalf("parse fixture %s: %v", name, err)
	}
	return doc
}

// --- AC-2: Applies --------------------------------------------------------

func TestAPIPostman_Applies_TypeFeatureWithEndpoints(t *testing.T) {
	doc := parseFixture(t, "account-statement-complete.md")
	if !(&Synthesizer{}).Applies(doc) {
		t.Fatal("Applies should be true for feature doc with endpoints section")
	}
}

func TestAPIPostman_Applies_NoEndpointsSection(t *testing.T) {
	content := []byte(`---
type: feature
date: "2026-04-15"
status: draft
---

# Plain feature

No endpoints here.
`)
	doc, err := synthesizer.ParseDoc("plain.md", content)
	if err != nil {
		t.Fatal(err)
	}
	if (&Synthesizer{}).Applies(doc) {
		t.Fatal("Applies should be false when no endpoints section")
	}
}

func TestAPIPostman_Applies_WrongType(t *testing.T) {
	content := []byte(`---
type: decision
date: "2026-04-15"
status: draft
---

### Endpoints

- POST /api/foo
`)
	doc, err := synthesizer.ParseDoc("decision.md", content)
	if err != nil {
		t.Fatal(err)
	}
	if (&Synthesizer{}).Applies(doc) {
		t.Fatal("Applies should be false for type=decision")
	}
}

// --- AC-3: Detect endpoints -----------------------------------------------

func TestAPIPostman_Detect_TwoEndpointsFromFixture(t *testing.T) {
	doc := parseFixture(t, "account-statement-complete.md")
	candidates, err := (&Synthesizer{}).Detect(doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 2 {
		t.Fatalf("want 2 candidates, got %d", len(candidates))
	}
	wantKeys := map[string]bool{
		"POST /api/invoices/search":        true,
		"POST /api/invoices/export-excel": true,
	}
	for _, c := range candidates {
		if !wantKeys[c.Key] {
			t.Fatalf("unexpected candidate Key: %q", c.Key)
		}
	}
}

// --- AC-5: Min/Max expansion ----------------------------------------------

func TestAPIPostman_Detect_MinMaxExpansionExpandsTo3(t *testing.T) {
	doc := parseFixture(t, "account-statement-complete.md")
	candidates, err := (&Synthesizer{}).Detect(doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) == 0 {
		t.Fatal("no candidates detected")
	}
	fields, _ := candidates[0].Extra["fields"].([]fieldHit)

	has := func(name string) bool {
		for _, f := range fields {
			if f.Name == name {
				return true
			}
		}
		return false
	}
	for _, want := range []string{"creditAmount", "creditAmountMin", "creditAmountMax", "debitAmount", "debitAmountMin", "debitAmountMax"} {
		if !has(want) {
			t.Fatalf("expanded field %q missing: %v", want, sortedFieldNames(fields))
		}
	}
}

// --- AC-6: Required marker detection --------------------------------------

func TestAPIPostman_Detect_RequiredMarkerFromBoldRequis(t *testing.T) {
	doc := parseFixture(t, "account-statement-complete.md")
	candidates, _ := (&Synthesizer{}).Detect(doc)
	fields, _ := candidates[0].Extra["fields"].([]fieldHit)

	var month *fieldHit
	for i := range fields {
		if fields[i].Name == "month" {
			month = &fields[i]
			break
		}
	}
	if month == nil {
		t.Fatal("month field not detected")
	}
	if !month.Required {
		t.Fatal("month should be marked required (source has **requis**)")
	}
	if !strings.Contains(strings.ToLower(month.RequiredToken), "requis") {
		t.Fatalf("required token should be literal 'requis', got %q", month.RequiredToken)
	}
}

// --- AC-7: Security projection (strict mode) ------------------------------

func TestAPIPostman_Synthesize_SecurityFieldsExcluded(t *testing.T) {
	doc := parseFixture(t, "account-statement-complete.md")
	candidates, _ := (&Synthesizer{}).Detect(doc)
	if len(candidates) == 0 {
		t.Fatal("no candidates")
	}

	cfg := synthesizer.Config{}
	block, evs, warns, err := (&Synthesizer{}).Synthesize(candidates[0], cfg)
	if err != nil {
		t.Fatal(err)
	}

	for _, forbidden := range []string{"organizationId", "branchCode"} {
		if strings.Contains(block.Content, `"`+forbidden+`"`) {
			t.Fatalf("I5 violation: %q appears in block content:\n%s", forbidden, block.Content)
		}
		for _, ev := range evs {
			if ev.Field == forbidden {
				t.Fatalf("I5 violation: evidence for %q present", forbidden)
			}
		}
	}
	// Strict mode => no missing-security warning.
	for _, w := range warns {
		if w.Code == synthesizer.WarningCodeMissingSecuritySection {
			t.Fatalf("strict mode must not emit %q", synthesizer.WarningCodeMissingSecuritySection)
		}
	}
}

// --- AC-8: Degraded mode (no Security section) ----------------------------

func TestAPIPostman_Synthesize_DegradedModeFiltersWellKnown(t *testing.T) {
	doc := parseFixture(t, "account-statement-no-security.md")
	candidates, _ := (&Synthesizer{}).Detect(doc)
	if len(candidates) == 0 {
		t.Fatal("no candidates")
	}

	cfg := synthesizer.Config{WellKnownServerFields: []string{"tenantId"}}
	block, evs, warns, err := (&Synthesizer{}).Synthesize(candidates[0], cfg)
	if err != nil {
		t.Fatal(err)
	}

	// tenantId MUST be filtered.
	if strings.Contains(block.Content, `"tenantId"`) {
		t.Fatalf("I5-bis: tenantId should be filtered in degraded mode:\n%s", block.Content)
	}
	for _, ev := range evs {
		if ev.Field == "tenantId" {
			t.Fatal("I5-bis: tenantId evidence should be removed")
		}
	}
	// Warning MUST be emitted.
	foundWarn := false
	for _, w := range warns {
		if w.Code == synthesizer.WarningCodeMissingSecuritySection {
			foundWarn = true
			break
		}
	}
	if !foundWarn {
		t.Fatalf("degraded mode must emit %q, got %v", synthesizer.WarningCodeMissingSecuritySection, warns)
	}
}

// --- AC-9 / AC-10: Body shape β + HTTP+JSON format ------------------------

func TestAPIPostman_Synthesize_BodyShapeStrategyBeta(t *testing.T) {
	doc := parseFixture(t, "account-statement-complete.md")
	candidates, _ := (&Synthesizer{}).Detect(doc)

	block, _, _, err := (&Synthesizer{}).Synthesize(candidates[0], synthesizer.Config{})
	if err != nil {
		t.Fatal(err)
	}

	// month is required → variable.
	if !strings.Contains(block.Content, `"month": "{{month}}"`) {
		t.Fatalf("required field must render as Postman variable:\n%s", block.Content)
	}
	// accountNumber is optional → null.
	if !strings.Contains(block.Content, `"accountNumber": null`) {
		t.Fatalf("optional field must render as null:\n%s", block.Content)
	}
	// HTTP head present.
	if !strings.Contains(block.Content, "POST {{baseUrl}}/api/invoices/search") {
		t.Fatal("request line missing or malformed")
	}
	if !strings.Contains(block.Content, "Authorization: Bearer {{jwt}}") {
		t.Fatal("Authorization header missing")
	}
	if !strings.Contains(block.Content, "Content-Type: application/json") {
		t.Fatal("Content-Type header missing")
	}
}

// --- AC-11: Two variants (minimal + full) ---------------------------------

func TestAPIPostman_Synthesize_TwoVariantsMinimalAndFull(t *testing.T) {
	doc := parseFixture(t, "account-statement-complete.md")
	candidates, _ := (&Synthesizer{}).Detect(doc)
	block, _, _, _ := (&Synthesizer{}).Synthesize(candidates[0], synthesizer.Config{})

	if !strings.Contains(block.Content, "Full — required + optional fields") {
		t.Fatal("full variant missing")
	}
	if !strings.Contains(block.Content, "Minimal — required fields only") {
		t.Fatal("minimal variant missing")
	}
}

func TestAPIPostman_Synthesize_NoRequiredFieldEmitsOnlyFull(t *testing.T) {
	content := []byte(`---
type: feature
date: "2026-04-15"
status: draft
---

### Endpoints

| Méthode | URL | Description |
|---------|-----|-------------|
| `+"`"+`POST`+"`"+` | `+"`"+`/api/x`+"`"+` | desc |

### Filtres

`+"`"+`foo`+"`"+`, `+"`"+`bar`+"`"+`

## Sécurité

- Rien injecté.
`)
	doc, err := synthesizer.ParseDoc("no-req.md", content)
	if err != nil {
		t.Fatal(err)
	}
	candidates, _ := (&Synthesizer{}).Detect(doc)
	if len(candidates) == 0 {
		t.Fatal("no candidates")
	}
	block, _, _, _ := (&Synthesizer{}).Synthesize(candidates[0], synthesizer.Config{})
	if strings.Contains(block.Content, "Minimal — required fields only") {
		t.Fatal("minimal variant should be absent when no required field exists")
	}
	if !strings.Contains(block.Content, "Full — required + optional fields") {
		t.Fatal("full variant should always be present")
	}
}

// --- AC-12: Notes ---------------------------------------------------------

func TestAPIPostman_Synthesize_NotesMentionExcludedFields(t *testing.T) {
	doc := parseFixture(t, "account-statement-complete.md")
	candidates, _ := (&Synthesizer{}).Detect(doc)
	block, _, _, _ := (&Synthesizer{}).Synthesize(candidates[0], synthesizer.Config{})

	joined := strings.Join(block.Notes, "\n")
	if !strings.Contains(joined, "organizationId") || !strings.Contains(joined, "branchCode") {
		t.Fatalf("notes must mention server-injected fields: %v", block.Notes)
	}
}

// --- AC-13: Invariants (contract suite) -----------------------------------

func TestAPIPostman_Contracts(t *testing.T) {
	fixtures := []synthesizer.ContractFixture{
		{
			Name:           "complete-with-security",
			Path:           "account-statement-complete.md",
			Content:        string(readFixture(t, "account-statement-complete.md")),
			ServerInjected: []string{"organizationId", "branchCode"},
			MinCandidates:  2,
		},
		{
			Name:                  "no-security-degraded",
			Path:                  "account-statement-no-security.md",
			Content:               string(readFixture(t, "account-statement-no-security.md")),
			WellKnown:             []string{"tenantId"},
			ExpectMissingSecurity: true,
			MinCandidates:         1,
		},
	}
	synthesizer.RunInvariantContracts(t, &Synthesizer{}, fixtures)
}

// --- I6: canonical output stable ------------------------------------------

func TestAPIPostman_Synthesize_CanonicalOutputStable(t *testing.T) {
	doc := parseFixture(t, "account-statement-complete.md")
	candidates, _ := (&Synthesizer{}).Detect(doc)
	b1, _, _, _ := (&Synthesizer{}).Synthesize(candidates[0], synthesizer.Config{})
	b2, _, _, _ := (&Synthesizer{}).Synthesize(candidates[0], synthesizer.Config{})
	if b1.Content != b2.Content {
		t.Fatalf("I6 violation: non-deterministic output:\n%s\nvs\n%s", b1.Content, b2.Content)
	}
}

// --- AC-15 Dogfood: run against the live feature doc ---------------------
//
// This test uses the production dogfood target committed to the repo. If the
// doc moves or is renamed, skip gracefully - we don't want framework
// evolution to break CI just because a doc path changes.

func TestAPIPostman_DogfoodFixture(t *testing.T) {
	dogfoodPath := filepath.Join("..", "..", "..", "..", "..", ".lore", "docs",
		"feature-account-statement-and-detailed-account-statement.md")
	data, err := os.ReadFile(dogfoodPath)
	if err != nil {
		t.Skipf("dogfood doc not readable at %s: %v", dogfoodPath, err)
	}

	doc, err := synthesizer.ParseDoc(dogfoodPath, data)
	if err != nil {
		t.Fatalf("parse dogfood: %v", err)
	}
	s := &Synthesizer{}
	if !s.Applies(doc) {
		t.Fatal("Applies returned false on the dogfood doc - regression")
	}
	candidates, err := s.Detect(doc)
	if err != nil {
		t.Fatalf("detect dogfood: %v", err)
	}
	if len(candidates) == 0 {
		t.Fatal("expected at least one candidate on the dogfood doc")
	}

	// I5 check on the real doc.
	for _, c := range candidates {
		block, evs, _, err := s.Synthesize(c, synthesizer.Config{
			WellKnownServerFields: []string{"tenantId", "authenticatedUsername", "principalId"},
		})
		if err != nil {
			t.Fatalf("synthesize dogfood: %v", err)
		}
		for _, forbidden := range []string{"organizationId", "branchCode"} {
			if strings.Contains(block.Content, `"`+forbidden+`"`) {
				t.Fatalf("I5 violation on dogfood for %s: %s", forbidden, c.Key)
			}
			for _, ev := range evs {
				if ev.Field == forbidden {
					t.Fatalf("I5 violation (evidence): %s present in evidence for %s", forbidden, c.Key)
				}
			}
		}
	}
}
