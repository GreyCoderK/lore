// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/i18n"
)

// ═══════════════════════════════════════════════════════════════════════════
// i18n coverage tests for the review preview (8-20) + persona opt-in (8-19).
//
// Three layers (mirrors suggestions_i18n_test.go design for Story 8.3):
//   1. Field-presence: every migrated field is non-empty in EN and FR.
//   2. Format-arg parity: every field has matching %-specifier sequence EN/FR.
//   3. Anti-regression guard: scan source files for known migrated strings
//      reappearing as hardcoded literals (regression would mean an operator
//      running `language=fr` sees English).
//
// These tests do NOT assert the French wording is correct — a translator's
// review is a separate, human concern. They assert STRUCTURAL parity + catalog
// completeness, which is what mechanically breaks the bilingual contract.
// ═══════════════════════════════════════════════════════════════════════════

// reviewAngelaFields enumerates the AngelaMessages fields introduced by the
// 8-19 / 8-20 i18n migration. Keeping this list explicit (rather than
// inferring via reflection over all UI* fields) avoids accidentally
// strengthening the gate to cover fields that belong to an older story with
// its own policy.
var reviewAngelaFields = []string{
	// Preview report (8-20)
	"UIReviewPreviewHeader",
	"UIReviewPreviewCorpus",
	"UIReviewPreviewModel",
	"UIReviewPreviewPersonasBaseline",
	"UIReviewPreviewPersonasList",
	"UIReviewPreviewAudienceNone",
	"UIReviewPreviewAudience",
	"UIReviewPreviewTokens",
	"UIReviewPreviewContextWindow",
	"UIReviewPreviewCost",
	"UIReviewPreviewCostUnknown",
	"UIReviewPreviewExpectedTime",
	"UIReviewPreviewWarningsHeader",
	"UIReviewPreviewAbort",
	// Persona opt-in UX (8-19)
	"UIPersonaConfiguredHeader",
	"UIPersonaCostDeltaBaselineLabel",
	"UIPersonaCostDeltaAugmentedLabel",
	"UIPersonaAddContext",
	"UIPersonaPromptQuestion",
	"UIPersonaNonTTYInfo",
	"UIPersonaCostBaselineZero",
	"UIPersonaCostDeltaUndefined",
	"UIPersonaCostDeltaPct",
	"UIPersonaCostSameRoughly",
	"UIPersonaCostUnknown",
	"UIPersonaInputTokens",
	"UIPersonaCostInline",
	// Review header + per-finding + TUI (8-19 follow-up)
	"UIReviewAngleHeader",
	"UIReviewAnglePersonaRow",
	"UIReviewFlaggedBy",
	"UIReviewAgreementConcur",
	"UIReviewAgreementLineFormat",
}

// reviewCmdFields enumerates the CmdMessages fields carrying review-specific
// error strings introduced by the migration.
var reviewCmdFields = []string{
	"AngelaReviewErrFormatRequiresPreview",
	"AngelaReviewErrMutuallyExclusive",
	"AngelaReviewErrUnknownPersonas",
	"AngelaReviewErrUnknownConfiguredPersona",
	"AngelaReviewErrUseConfiguredNoManual",
	"AngelaReviewErrUnknownFormat",
}

// TestReviewI18n_AngelaFields_BothLanguagesPopulated verifies every new
// AngelaMessages field is non-empty in EN and FR. A missing field is an
// immediate bilingual contract break.
func TestReviewI18n_AngelaFields_BothLanguagesPopulated(t *testing.T) {
	en := getCatalogAngela(t, "en")
	fr := getCatalogAngela(t, "fr")

	for _, name := range reviewAngelaFields {
		if v := reflectString(t, en, name); v == "" {
			t.Errorf("EN catalog missing or empty: Angela.%s", name)
		}
		if v := reflectString(t, fr, name); v == "" {
			t.Errorf("FR catalog missing or empty: Angela.%s", name)
		}
	}
}

// TestReviewI18n_CmdFields_BothLanguagesPopulated — same guarantee for the 6
// error fields migrated to i18n.T().Cmd.
func TestReviewI18n_CmdFields_BothLanguagesPopulated(t *testing.T) {
	en := getCatalogCmd(t, "en")
	fr := getCatalogCmd(t, "fr")

	for _, name := range reviewCmdFields {
		if v := reflectString(t, en, name); v == "" {
			t.Errorf("EN catalog missing or empty: Cmd.%s", name)
		}
		if v := reflectString(t, fr, name); v == "" {
			t.Errorf("FR catalog missing or empty: Cmd.%s", name)
		}
	}
}

// TestReviewI18n_AngelaFields_FormatArgParity asserts each migrated field has
// the SAME sequence of %-specifiers in EN and FR. A mismatch silently produces
// wrong-arg-order output or a runtime crash when the catalog is switched.
func TestReviewI18n_AngelaFields_FormatArgParity(t *testing.T) {
	en := getCatalogAngela(t, "en")
	fr := getCatalogAngela(t, "fr")
	specRe := regexp.MustCompile(`%[\-\+0-9.#]*[a-zA-Z%]`)

	for _, name := range reviewAngelaFields {
		enVal := reflectString(t, en, name)
		frVal := reflectString(t, fr, name)
		enSpecs := normalizeSpecs(specRe.FindAllString(enVal, -1))
		frSpecs := normalizeSpecs(specRe.FindAllString(frVal, -1))
		if !reflect.DeepEqual(enSpecs, frSpecs) {
			t.Errorf("Angela.%s: format-arg parity broken\n  EN: %q → %v\n  FR: %q → %v",
				name, enVal, enSpecs, frVal, frSpecs)
		}
	}
}

// TestReviewI18n_CmdFields_FormatArgParity — same assertion for the Cmd error
// fields.
func TestReviewI18n_CmdFields_FormatArgParity(t *testing.T) {
	en := getCatalogCmd(t, "en")
	fr := getCatalogCmd(t, "fr")
	specRe := regexp.MustCompile(`%[\-\+0-9.#]*[a-zA-Z%]`)

	for _, name := range reviewCmdFields {
		enVal := reflectString(t, en, name)
		frVal := reflectString(t, fr, name)
		enSpecs := normalizeSpecs(specRe.FindAllString(enVal, -1))
		frSpecs := normalizeSpecs(specRe.FindAllString(frVal, -1))
		if !reflect.DeepEqual(enSpecs, frSpecs) {
			t.Errorf("Cmd.%s: format-arg parity broken\n  EN: %q → %v\n  FR: %q → %v",
				name, enVal, enSpecs, frVal, frSpecs)
		}
	}
}

// scannedReviewSources lists the files that MUST NOT contain a hardcoded
// form of any migrated English string. If a contributor inlines "Corpus:"
// or "Flagged by:" in one of these files, we fail so an FR user does not
// silently get English output.
var scannedReviewSources = []string{
	"angela_review_preview.go",
	"angela_review_personas.go",
	"angela_review.go",
}

var scannedReviewSourcesInterpretable = []string{
	"../internal/angela/review_interactive.go",
}

// migratedEnglishPhrases is the list of substrings we banished from the code
// path. If any appears as a literal in the scanned files (outside comments),
// it means a refactor accidentally reintroduced a hardcoded English UI
// string. Kept narrow on purpose — matches exact phrases the migration
// replaced, not any English text.
var migratedEnglishPhrases = []string{
	"Review preview",
	"Corpus:          ",
	"Model:           ",
	"baseline (no personas)",
	"Audience:        ",
	"Estimated tokens:",
	"Context window:  ",
	"Estimated cost:  ",
	"Expected time:   ",
	"unknown (model pricing not registered)",
	"ABORT: ",
	"Personas add context",
	"Use them for this review? [y/N]",
	"Baseline review:",
	"Review with ",
	"(~same cost)",
	"(cost unknown — model pricing not registered)",
	"(zero-token baseline)",
	"(delta undefined)",
	"personas configured but not activated",
	"Review angle:",
	"Flagged by:",
	"Agreement:",
	"--format requires --preview",
	"mutually exclusive flags:",
	"unknown persona(s):",
	"unknown --format",
	"--use-configured-personas requires angela.review.personas.selection",
}

// TestReviewI18n_NoHardcodedMigratedStrings is the anti-regression guard.
// Scans the post-migration source files and fails if any banned English
// phrase reappears as a literal. Comments are NOT stripped — the cost of a
// false positive on a comment is tiny (add a hyphen or reword it), the cost
// of a missed regression is a user-visible bilingual break.
func TestReviewI18n_NoHardcodedMigratedStrings(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	var violations []string

	scan := func(path string) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		content := string(data)
		for _, phrase := range migratedEnglishPhrases {
			if strings.Contains(content, `"`+phrase+`"`) ||
				strings.Contains(content, "`"+phrase+"`") {
				violations = append(violations,
					filepath.Base(path)+": hardcoded literal %q reintroduced — must pass through i18n.T()")
			}
		}
	}

	for _, name := range scannedReviewSources {
		scan(filepath.Join(wd, name))
	}
	for _, rel := range scannedReviewSourcesInterpretable {
		scan(filepath.Join(wd, rel))
	}

	if len(violations) > 0 {
		t.Errorf("found %d hardcoded migrated string(s) — bilingual contract broken:\n  %s",
			len(violations), strings.Join(violations, "\n  "))
	}
}

// ─────────────────────────────────────────────────────────────────────────
// Runtime FR rendering tests — prove the migration works end-to-end
// ─────────────────────────────────────────────────────────────────────────

// TestReviewI18n_PreviewReport_RendersFR forces the catalog to FR, runs
// runReviewPreview, and asserts the output contains FR strings (not the EN
// literals). Protects against a regression where the renderer is wired to the
// wrong catalog field or constructs the text outside i18n.T().
func TestReviewI18n_PreviewReport_RendersFR(t *testing.T) {
	restore := i18n.Snapshot()
	t.Cleanup(restore)
	i18n.Init("fr")

	streams, out, _ := streamsForPreview()
	in := baseReviewPreviewInputs()
	if err := runReviewPreview(streams, &config.Config{}, in); err != nil {
		t.Fatalf("runReviewPreview error: %v", err)
	}
	got := out.String()

	// The FR catalog for the preview uses "Aperçu", "Corpus :", "Modèle :",
	// "Tokens estimés", "Coût estimé" (substrings — we don't assert on full
	// translations so a translator can tune wording without breaking the
	// test).
	wantSubstrings := []string{
		"Aperçu", // header
		"Corpus",
		"Modèle",
		"Tokens estim",
		"Coût estim",
	}
	for _, s := range wantSubstrings {
		if !strings.Contains(got, s) {
			t.Errorf("FR preview missing expected substring %q\nfull output:\n%s", s, got)
		}
	}
	// Sanity: EN header must NOT leak into FR output.
	if strings.Contains(got, "Review preview\n──────────────") {
		t.Errorf("EN preview header leaked into FR output:\n%s", got)
	}
}

// TestReviewI18n_PersonaPrompt_RendersFR forces FR and exercises the persona
// confirmation prompt renderer via a direct call to renderPersonaCostDelta +
// promptPersonaConfirmation. The latter requires a stdin; we feed "n\n" so
// the helper returns false without triggering further rendering.
func TestReviewI18n_PersonaPrompt_RendersFR(t *testing.T) {
	restore := i18n.Snapshot()
	t.Cleanup(restore)
	i18n.Init("fr")

	streams, _, errBuf := streamsForPreview()
	// Inject the "n\n" so the prompt returns false and we don't block.
	streams.In = bytes.NewBufferString("n\n")

	name := firstRegistryPersonaName(t)
	_, perr := promptPersonaConfirmation(streams, personaPromptInputs{
		CorpusBytes: 2000,
		Model:       "claude-sonnet-4-6",
		MaxTokens:   4000,
		Timeout:     60 * time.Second,
		Candidates:  []string{name},
	})
	if perr != nil {
		t.Fatalf("prompt error: %v", perr)
	}

	got := errBuf.String()
	wantSubstrings := []string{
		"configurée", // FR for "configured"
		"Les activer pour cette review", // FR prompt question
		"[o/N]",                          // FR y/N
	}
	for _, s := range wantSubstrings {
		if !strings.Contains(got, s) {
			t.Errorf("FR persona prompt missing expected substring %q\nfull stderr:\n%s", s, got)
		}
	}
	if strings.Contains(got, "Use them for this review?") {
		t.Errorf("EN prompt question leaked into FR output:\n%s", got)
	}
}

// TestReviewI18n_ErrFormatRequiresPreview_RendersFR fires the cobra command
// under FR and asserts the returned error reads as the FR catalog entry, not
// the EN one.
func TestReviewI18n_ErrFormatRequiresPreview_RendersFR(t *testing.T) {
	restore := i18n.Snapshot()
	t.Cleanup(restore)
	i18n.Init("fr")

	streams, _, _ := streamsForPreview()
	cfg := &config.Config{}
	path := ""
	cmd := newAngelaReviewCmd(cfg, streams, &path)
	cmd.SetArgs([]string{"--format", "json"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --format is used without --preview")
	}
	if !strings.Contains(err.Error(), "requiert --preview") {
		t.Errorf("FR error expected to contain 'requiert --preview', got: %v", err)
	}
	if strings.Contains(err.Error(), "requires --preview") {
		t.Errorf("EN error phrase leaked into FR output: %v", err)
	}
}

// TestReviewI18n_NonTTYInfo_RendersFR forces FR and calls the non-TTY info
// renderer directly, asserting the FR message appears.
func TestReviewI18n_NonTTYInfo_RendersFR(t *testing.T) {
	restore := i18n.Snapshot()
	t.Cleanup(restore)
	i18n.Init("fr")

	streams, _, errBuf := streamsForPreview()
	renderNonTTYPersonaInfo(streams, []string{"architect", "qa-reviewer"})
	got := errBuf.String()
	if !strings.Contains(got, "configurées mais non activées") {
		t.Errorf("FR non-TTY info missing expected FR phrase.\nGot:\n%s", got)
	}
}

// ─────────────────────────────────────────────────────────────────────────
// Reflection helpers
// ─────────────────────────────────────────────────────────────────────────

func getCatalogAngela(t *testing.T, lang string) i18n.AngelaMessages {
	t.Helper()
	restore := i18n.Snapshot()
	t.Cleanup(restore)
	i18n.Init(lang)
	return i18n.T().Angela
}

func getCatalogCmd(t *testing.T, lang string) i18n.CmdMessages {
	t.Helper()
	restore := i18n.Snapshot()
	t.Cleanup(restore)
	i18n.Init(lang)
	return i18n.T().Cmd
}

func reflectString(t *testing.T, msgs any, name string) string {
	t.Helper()
	v := reflect.ValueOf(msgs)
	f := v.FieldByName(name)
	if !f.IsValid() {
		t.Fatalf("struct has no field %q — update test list", name)
	}
	if f.Kind() != reflect.String {
		t.Fatalf("field %q is not a string (kind=%s)", name, f.Kind())
	}
	return f.String()
}

// normalizeSpecs drops "%" (which matches the regex but is a literal percent
// escape, not a format specifier) so `75%%` in a format string doesn't show
// up as a mismatched spec.
func normalizeSpecs(specs []string) []string {
	out := make([]string, 0, len(specs))
	for _, s := range specs {
		if s == "%%" {
			continue
		}
		out = append(out, s)
	}
	return out
}

