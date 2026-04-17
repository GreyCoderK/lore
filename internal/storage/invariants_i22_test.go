// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

// ═══════════════════════════════════════════════════════════════════════════
// Invariant I22 — Corpus partial corruption recoverable.
//
// Contract: 1+ malformed docs in the corpus MUST NOT
//   (a) crash the scanner (SearchDocs, Diagnose, FindDocByCommit),
//   (b) prevent valid docs from being returned,
//   (c) surface as a fatal error that aborts the calling command.
//
// Parse failures are reported as a non-fatal joined error from SearchDocs
// and as "invalid-frontmatter" issues from Diagnose. The user always has
// a path to fix the corruption without losing access to the healthy corpus.
//
// Layer 1 (pre-existing):
//   - frontmatter_test.go::TestUnmarshal_MalformedYAML         (unit)
//   - doctor_test.go::TestDiagnose_InvalidFrontMatter_*        (doctor path)
//   - store/invariants_i8_test.go::TestI8_PartialCorruption... (store rebuild)
// Layer 2 (below): explicit anchor on the scanner public surface used by
// `lore show`, `lore doctor`, `lore angela review`, etc.
// ═══════════════════════════════════════════════════════════════════════════

// TestI22_SearchDocsReturnsValidDocsWithMixedCorruption verifies the core
// recovery contract: a scrambled corpus with multiple corruption flavors
// still lets valid docs flow through, and parse errors are surfaced
// non-fatally (returned as the second value, not as a blocking failure).
func TestI22_SearchDocsReturnsValidDocsWithMixedCorruption(t *testing.T) {
	dir := t.TempDir()

	// 3 valid docs.
	writeDocI22(t, dir, "decision-ok-2026-03-01.md",
		"---\ntype: decision\ndate: \"2026-03-01\"\nstatus: draft\n---\n# OK A\n")
	writeDocI22(t, dir, "feature-ok-2026-03-02.md",
		"---\ntype: feature\ndate: \"2026-03-02\"\nstatus: draft\n---\n# OK B\n")
	writeDocI22(t, dir, "bugfix-ok-2026-03-03.md",
		"---\ntype: bugfix\ndate: \"2026-03-03\"\nstatus: draft\n---\n# OK C\n")
	// 4 corruption variants.
	writeDocI22(t, dir, "bad-unclosed-quote.md",
		"---\ntype: \"feature\ndate: 2026-03-04\n---\n# Body\n")
	writeDocI22(t, dir, "bad-no-frontmatter.md",
		"# just markdown, no frontmatter\n")
	writeDocI22(t, dir, "bad-binary.md",
		"\x00\xff\xferaw binary garbage\x00")
	writeDocI22(t, dir, "bad-truncated.md",
		"---\ntype: decision\ndate:")

	results, parseErr := SearchDocs(dir, "", domain.DocFilter{})

	if len(results) != 3 {
		t.Errorf("I22 violation: expected 3 valid docs in results, got %d — corruption starved healthy docs", len(results))
	}
	if parseErr == nil {
		t.Error("I22 violation: parse errors must be surfaced non-fatally (parseErr != nil), got nil")
	}
	got := map[string]bool{}
	for _, r := range results {
		got[r.Filename] = true
	}
	for _, want := range []string{
		"decision-ok-2026-03-01.md",
		"feature-ok-2026-03-02.md",
		"bugfix-ok-2026-03-03.md",
	} {
		if !got[want] {
			t.Errorf("I22 violation: valid doc %q missing from results (corrupt sibling masked it)", want)
		}
	}
	// Corrupt filenames must appear somewhere in the joined error message so
	// the user can tell what broke — silent skip would violate the contract.
	if parseErr != nil {
		msg := parseErr.Error()
		for _, badName := range []string{"bad-unclosed-quote.md", "bad-no-frontmatter.md", "bad-truncated.md"} {
			if !strings.Contains(msg, badName) {
				t.Errorf("I22 violation: corrupt doc %q not named in parseErr message — user cannot diagnose which file broke", badName)
			}
		}
	}
}

// TestI22_DiagnoseReportsCorruptionWithoutFatalError verifies `lore doctor`
// returns a populated report rather than a fatal error when the corpus has
// multiple broken docs — the whole point of the doctor command is to be
// robust against exactly the situations it is asked to diagnose.
func TestI22_DiagnoseReportsCorruptionWithoutFatalError(t *testing.T) {
	dir := t.TempDir()

	writeDocI22(t, dir, "feature-ok-2026-03-01.md",
		"---\ntype: feature\ndate: \"2026-03-01\"\nstatus: draft\n---\n# OK\n")
	// 3 distinct corruption flavors.
	writeDocI22(t, dir, "bad-1-2026-03-02.md", "---\n{{invalid yaml\n---\n# bad1\n")
	writeDocI22(t, dir, "bad-2-2026-03-03.md", "---\ntype: [unterminated\n---\n")
	writeDocI22(t, dir, "bad-3-2026-03-04.md", "---\n: : :\n---\n")
	// NOTE: we intentionally do NOT call RegenerateIndex here — it currently
	// aborts on malformed docs (separate post-MVP finding, tracked in the
	// phase-12 journal). Diagnose itself must remain resilient regardless.

	report, err := Diagnose(dir)
	if err != nil {
		t.Fatalf("I22 violation: Diagnose returned fatal error on partial corruption: %v", err)
	}
	if report == nil {
		t.Fatal("I22 violation: Diagnose returned nil report")
	}
	invalidCount := 0
	for _, iss := range report.Issues {
		if iss.Category == "invalid-frontmatter" {
			invalidCount++
		}
	}
	if invalidCount != 3 {
		t.Errorf("I22 violation: expected 3 invalid-frontmatter issues (one per corrupt doc), got %d: %+v",
			invalidCount, report.Issues)
	}
	if report.DocCount != 1 {
		t.Errorf("I22 violation: valid DocCount = %d, want 1 — corrupt docs must not hide healthy ones from the count", report.DocCount)
	}
}

// TestI22_CorruptionVariantsNeverPanic is the anti-regression property:
// whatever garbage lands in `.lore/docs/`, SearchDocs must return (possibly
// empty) results + possibly-non-nil error, but never panic or abort.
// Go's yaml.v3 parser and our delimiter scanner must both handle these.
func TestI22_CorruptionVariantsNeverPanic(t *testing.T) {
	variants := map[string]string{
		"empty":                "",
		"single_dash":          "-",
		"frontmatter_empty":    "---\n---\n",
		"triple_dash_no_close": "---\ntype: feature\n# no closing fence\n",
		"binary_bytes":         "\x00\x01\x02\x03\xff\xfe\xfd",
		"very_long_key":        "---\n" + strings.Repeat("k", 10000) + ": v\n---\n",
		"utf8_bom_prefix":      "\ufeff---\ntype: decision\ndate: \"2026-03-01\"\nstatus: draft\n---\n# ok\n",
		"unterminated_string":  "---\nmessage: \"no close\n---\n# body",
		"tab_indent_mix":       "---\ntype:\tfeature\n  date: 2026-03-01\n---\n",
		"deeply_nested":        "---\na:\n b:\n  c:\n   d:\n    e: f\n---\n",
	}

	for name, content := range variants {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeDocI22(t, dir, "probe-"+name+".md", content)
			// One neighbor valid doc — the scan must still yield it untouched.
			writeDocI22(t, dir, "valid-2026-03-01.md",
				"---\ntype: feature\ndate: \"2026-03-01\"\nstatus: draft\n---\n# ok\n")

			// The contract: no panic, no fatal error. parseErr may be non-nil
			// for variants that fail parsing — that is acceptable.
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("I22 violation: SearchDocs panicked on variant %q: %v", name, r)
				}
			}()
			results, _ := SearchDocs(dir, "", domain.DocFilter{})
			// The valid neighbor must always survive.
			foundValid := false
			for _, r := range results {
				if r.Filename == "valid-2026-03-01.md" {
					foundValid = true
				}
			}
			if !foundValid {
				t.Errorf("I22 violation: valid neighbor doc lost when scanning alongside variant %q", name)
			}
		})
	}
}

func writeDocI22(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
