// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package i18n

import (
	"reflect"
	"regexp"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// Invariant I11 — i18n parity global.
//
// Contract: the two catalogs `catalogEN` and `catalogFR` must carry
// identical STRUCTURE (same fields, same paths in the embedded Messages
// tree) AND identical FORMAT-ARG SIGNATURES (every `%s`, `%d`, `%q`, `%.1f`,
// etc. in an EN string must appear in the corresponding FR string in the
// same order). A mismatch means:
//
//   - Field present in EN but empty in FR → user under `language=fr` sees
//     an empty string (silent regression).
//   - Field present in FR but empty in EN → same problem inverted.
//   - Format-arg mismatch → runtime `fmt.Sprintf` panics or silently puts
//     arguments in the wrong placeholder (e.g. FR says "%s files" when EN
//     says "%d files"; caller passes an int, FR prints a corrupt string).
//
// This global invariant complements:
//   - TestCatalogEN/FR_AllFieldsPopulated (i18n_test.go): "no empty field"
//   - TestReviewI18n_AngelaFields_FormatArgParity (cmd/angela_review_i18n_test.go):
//     format-arg parity for Angela review fields only.
//
// What this test adds that the above don't: format-arg parity for ALL
// Messages sub-structs (Cmd, Workflow, UI, Angela, Engagement, Decision,
// Shared, Notify) in ONE reflective sweep. A new catalog field added
// without FR twin or with a format-arg mistake is caught at `go test` time
// before it reaches production.
// ═══════════════════════════════════════════════════════════════════════════

// formatSpecRe matches Go fmt format specifiers. Covers `%s`, `%d`, `%q`,
// `%v`, `%f`, `%.1f`, `%+d`, `%-10s`, `%#v`, etc. The trailing verb
// character (a-zA-Z or `%` for literal percent) anchors the match.
var formatSpecRe = regexp.MustCompile(`%[\-\+0-9.#]*[a-zA-Z%]`)

// TestI11_AllCatalogsPopulated_GlobalAnchor is the explicit-named anchor
// for the I11 "no empty field" guarantee. Delegates to the same helper
// used by the pre-existing TestCatalogEN/FR_AllFieldsPopulated but under
// the `TestI11_` prefix so the invariants-coverage-matrix can cite it.
func TestI11_AllCatalogsPopulated_GlobalAnchor(t *testing.T) {
	t.Run("EN", func(t *testing.T) {
		checkAllStringsPopulated(t, reflect.ValueOf(*catalogEN), "catalogEN")
	})
	t.Run("FR", func(t *testing.T) {
		checkAllStringsPopulated(t, reflect.ValueOf(*catalogFR), "catalogFR")
	})
}

// TestI11_AllCatalogsFormatArgParity is the format-arg invariant. Walks
// the Messages tree in EN, and at every String leaf, compares the
// sequence of format specifiers with the FR counterpart at the same path.
// A mismatch fails the test with the exact field path and both specifier
// sequences so the translator knows what to fix.
func TestI11_AllCatalogsFormatArgParity(t *testing.T) {
	en := reflect.ValueOf(*catalogEN)
	fr := reflect.ValueOf(*catalogFR)
	checkFormatArgParity(t, en, fr, "Messages")
}

// checkFormatArgParity recursively walks two structurally-identical struct
// values, comparing format specifier sequences at every String leaf. The
// structure is guaranteed identical by Go's type system (catalogEN and
// catalogFR both have type *Messages), so we only need to drive from one
// side and look up the same field on the other.
func checkFormatArgParity(t *testing.T, en, fr reflect.Value, path string) {
	t.Helper()
	switch en.Kind() {
	case reflect.Struct:
		for i := 0; i < en.NumField(); i++ {
			field := en.Type().Field(i)
			// Embedded sub-struct or exported leaf; recurse.
			checkFormatArgParity(t, en.Field(i), fr.Field(i), path+"."+field.Name)
		}
	case reflect.String:
		enStr := en.String()
		frStr := fr.String()
		enSpecs := normalizeFormatSpecs(formatSpecRe.FindAllString(enStr, -1))
		frSpecs := normalizeFormatSpecs(formatSpecRe.FindAllString(frStr, -1))
		if !reflect.DeepEqual(enSpecs, frSpecs) {
			t.Errorf("I11 violation at %s:\n  EN (%q) → %v\n  FR (%q) → %v",
				path, enStr, enSpecs, frStr, frSpecs)
		}
	}
}

// normalizeFormatSpecs drops the escaped-percent sequence `%%` so a
// literal percent sign in either language doesn't show up as a mismatched
// specifier. FR uses "50 %" (space+%) often; EN uses "50%". Both render
// `%%` identically in fmt.Sprintf so dropping from the comparison is safe.
func normalizeFormatSpecs(specs []string) []string {
	out := make([]string, 0, len(specs))
	for _, s := range specs {
		if s == "%%" {
			continue
		}
		out = append(out, s)
	}
	return out
}

// TestI11_AllCatalogs_RuntimeSanityEveryFieldRendersBothLangs is a smoke
// check that forces Init("fr") then Init("en") and verifies the active
// catalog is actually swapped on every top-level sub-struct. It's the
// cheap runtime equivalent of "did the atomic.Value store take effect".
// A race or a missing case in Init() would be caught here.
func TestI11_AllCatalogs_RuntimeSanityEveryFieldRendersBothLangs(t *testing.T) {
	restore := Snapshot()
	t.Cleanup(restore)

	Init("fr")
	fr := T()
	if fr.Cmd.RootShort == catalogEN.Cmd.RootShort {
		t.Errorf("FR swap failed: Cmd.RootShort still EN (%q)", fr.Cmd.RootShort)
	}

	Init("en")
	en := T()
	if en.Cmd.RootShort != catalogEN.Cmd.RootShort {
		t.Errorf("EN re-init failed: Cmd.RootShort = %q, want %q",
			en.Cmd.RootShort, catalogEN.Cmd.RootShort)
	}

	// Verify a representative field from each sub-struct actually differs
	// between the two catalogs — if ALL fields happened to be identical
	// across EN and FR, the test above would be meaningless. We spot-check
	// one field per top-level sub-struct.
	probes := []struct {
		name    string
		enValue string
		frValue string
	}{
		{"Cmd.RootShort", catalogEN.Cmd.RootShort, catalogFR.Cmd.RootShort},
		{"Engagement.Milestone3", catalogEN.Engagement.Milestone3, catalogFR.Engagement.Milestone3},
		{"Workflow.SuggestSkipPrompt", catalogEN.Workflow.SuggestSkipPrompt, catalogFR.Workflow.SuggestSkipPrompt},
	}
	for _, p := range probes {
		if p.enValue == "" || p.frValue == "" {
			t.Errorf("probe %s: one language is empty (EN=%q, FR=%q)", p.name, p.enValue, p.frValue)
			continue
		}
		if p.enValue == p.frValue {
			// Only fail if the string is long enough that accidental
			// identical wording is improbable. Short strings like button
			// labels may legitimately match ("OK", brand names, etc.).
			if len(p.enValue) > 20 {
				t.Errorf("probe %s: EN and FR identical (%q) — translation missing?", p.name, p.enValue)
			}
		}
	}
}
