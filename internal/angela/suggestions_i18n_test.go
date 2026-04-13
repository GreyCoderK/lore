// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/i18n"
)

// ═══════════════════════════════════════════════════════════════
// Story 8.3 tests: i18n Angela suggestions
// ═══════════════════════════════════════════════════════════════

// scannedAngelaFiles lists the source files that participate in Angela's
// local draft analysis and whose `Suggestion{Message: ...}` literals were
// migrated to i18n in Story 8.3. Adding a new analyzer file to this package
// should include an entry here so the anti-regression guard covers it.
// score.go is intentionally excluded — its Missing[] strings are not
// Suggestion.Message fields. A separate i18n migration is tracked as
// a TODO in score.go.
var scannedAngelaFiles = []string{
	"draft.go",
	"coherence.go",
	"persona.go",
	"style.go",
}

// hardcodedMessageRe matches a Suggestion struct-literal that assigns a
// plain string literal (double-quoted or backticked) to the Message field.
// Pattern: `Message: "..."` or `Message: ` + backticked form. Struct-literal
// context is the dominant spot for new analyzer code, so guarding it stops
// new Suggestion{Message: "literal"} regressions at compile time.
//
// Intentionally does NOT match `x.Message = ...` reassignment. Those are
// used by runPersonaDraftChecksWithSections to decorate an already-i18n'd
// message with a persona badge — legitimate wrapping, not a bypass.
var hardcodedMessageRe = regexp.MustCompile(`Message:\s*(?:"[^"]*"|` + "`" + `[^` + "`" + `]*` + "`" + `)`)

// hardcodedSprintfRe matches the same struct-literal context but with a
// fmt.Sprintf("literal", ...) call. Pattern:
// `Message: fmt.Sprintf("...", ...)`. A contributor writing this bypasses
// i18n just as surely as a plain literal, so we flag it.
//
// Anchored to `Message:` to avoid false positives on unrelated Sprintf
// calls elsewhere in the file (display strings, logging, etc.).
var hardcodedSprintfRe = regexp.MustCompile(
	`Message:\s*fmt\.Sprintf\(\s*(?:"[^"]*"|` + "`" + `[^` + "`" + `]*` + "`" + `)`)

// locateAngelaDir finds the directory that contains the analyzer files.
// It walks up from the current test file location.
func locateAngelaDir(t *testing.T) string {
	t.Helper()
	// The test runs with CWD = package directory.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return wd
}

// TestAngelaSuggestions_NoHardcodedLiterals is the anti-regression guard
// for Story 8.3. It scans the Angela analyzer source files and fails if
// any `Suggestion{Message: "literal"}` or `Message: fmt.Sprintf("literal"`
// pattern is found. Prevents reintroducing hardcoded English strings.
func TestAngelaSuggestions_NoHardcodedLiterals(t *testing.T) {
	dir := locateAngelaDir(t)
	var violations []string

	for _, name := range scannedAngelaFiles {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}

		// Scan line by line so the error reports the line number.
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if hardcodedMessageRe.MatchString(line) {
				violations = append(violations,
					name+":"+itoa(i+1)+": hardcoded Message literal — use i18n.T().Angela.*")
			}
			if hardcodedSprintfRe.MatchString(line) {
				violations = append(violations,
					name+":"+itoa(i+1)+": fmt.Sprintf with string literal — use i18n.T().Angela.* as the format")
			}
		}
	}

	if len(violations) > 0 {
		t.Errorf("found %d hardcoded Suggestion literal(s) — Story 8.3 invariant broken:\n  %s",
			len(violations), strings.Join(violations, "\n  "))
	}
}

// itoa is a tiny helper to avoid importing strconv just for this.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}

// suggestionFields enumerates the SuggestionsCatalog-like fields within
// AngelaMessages that must be populated in BOTH the EN and FR catalogs.
// Covers Story 8.3 (draft, coherence, persona, style) plus the already-
// existing UI fields used by preflight / AnalyzeUsage.
//
// Listed explicitly (not via reflection over every Angela field) so that
// fields we intentionally leave English-only (PromptDirective, Principles)
// are not flagged.
var suggestionFields = []string{
	// persona.go (6)
	"PersonaWhyTooListy",
	"PersonaLongParagraphs",
	"PersonaMissingVerify",
	"PersonaNoTradeoffs",
	"PersonaUxNoImpact",
	"PersonaBusinessNoValue",
	// draft.go (11)
	"DraftMissingWhat",
	"DraftMissingWhy",
	"DraftMissingAltWarn",
	"DraftMissingAltInfo",
	"DraftMissingImpact",
	"DraftBodyTooShort",
	"DraftBodyExceedsMax",
	"DraftAddScope",
	"DraftAddTags",
	"DraftAddRelated",
	"DraftWhyTooBrief",
	// coherence.go (5)
	"CoherencePossibleDup",
	"CoherenceRelatedFound",
	"CoherenceSameScopeOverlap",
	"CoherenceSameScopeRelated",
	"CoherenceMentionedBody",
	// style.go (1)
	"StyleUnknownRule",
}

// TestAngelaSuggestions_BothLanguagesPopulated asserts that every migrated
// Suggestion field is present and non-empty in the EN and FR catalogs.
// Uses reflection to look up each field by name — a missing assignment in
// either catalog leaves the default zero-value (empty string) and fails.
func TestAngelaSuggestions_BothLanguagesPopulated(t *testing.T) {
	en := getAngelaCatalog(t, "en")
	fr := getAngelaCatalog(t, "fr")

	for _, field := range suggestionFields {
		enVal := fieldStringByName(t, en, field)
		frVal := fieldStringByName(t, fr, field)

		if enVal == "" {
			t.Errorf("EN catalog missing %s", field)
		}
		if frVal == "" {
			t.Errorf("FR catalog missing %s", field)
		}
	}
}

// TestAngelaSuggestions_FormatSpecifierParity asserts that for each
// Suggestion field, EN and FR have the same count AND order of format
// specifiers (%s, %d, %q, %f). A mismatch would crash fmt.Sprintf or
// produce silent wrong-arg-order bugs when the catalog is switched.
func TestAngelaSuggestions_FormatSpecifierParity(t *testing.T) {
	en := getAngelaCatalog(t, "en")
	fr := getAngelaCatalog(t, "fr")

	specRe := regexp.MustCompile(`%[a-zA-Z]`)

	for _, field := range suggestionFields {
		enVal := fieldStringByName(t, en, field)
		frVal := fieldStringByName(t, fr, field)

		enSpecs := specRe.FindAllString(enVal, -1)
		frSpecs := specRe.FindAllString(frVal, -1)

		if !reflect.DeepEqual(enSpecs, frSpecs) {
			t.Errorf("%s: format specifier mismatch\n  EN: %q → %v\n  FR: %q → %v",
				field, enVal, enSpecs, frVal, frSpecs)
		}
	}
}

// getAngelaCatalog forces the i18n loader to return the catalog for a
// specific language and extracts the embedded AngelaMessages struct.
// Uses i18n.Snapshot to capture the current catalog BEFORE switching
// and restores it via t.Cleanup, so a sibling test that legitimately
// ran under FR is not clobbered back to EN just because this helper
// ran.
//
// Paranoid-review fix (2026-04-11 LOW test hygiene).
func getAngelaCatalog(t *testing.T, lang string) i18n.AngelaMessages {
	t.Helper()
	restore := i18n.Snapshot()
	t.Cleanup(restore)
	i18n.Init(lang)
	return i18n.T().Angela
}

// fieldStringByName returns the string value of a field on AngelaMessages
// by reflection. Fails the test if the field does not exist.
func fieldStringByName(t *testing.T, msgs i18n.AngelaMessages, name string) string {
	t.Helper()
	v := reflect.ValueOf(msgs)
	f := v.FieldByName(name)
	if !f.IsValid() {
		t.Fatalf("AngelaMessages has no field %q — fix suggestionFields list", name)
	}
	if f.Kind() != reflect.String {
		t.Fatalf("AngelaMessages.%s is not a string (kind=%s)", name, f.Kind())
	}
	return f.String()
}
