// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

// fakeCorpusReader is a minimal in-memory domain.CorpusReader for unit
// tests. The validator only calls ReadDoc, so ListDocs is a stub.
type fakeCorpusReader struct {
	docs map[string]string
}

// ReadDoc returns the content for a known id, or an fs.ErrNotExist-
// wrapped error for an unknown id. Paranoid-review fix (2026-04-11
// MEDIUM test hygiene): the previous version returned a plain
// fmt.Errorf which does NOT satisfy errors.Is(err, fs.ErrNotExist),
// so any validator code that started relying on the canonical "not
// found" check would see a false negative against this fake even
// though production readers wrap fs.ErrNotExist correctly.
func (f *fakeCorpusReader) ReadDoc(id string) (string, error) {
	content, ok := f.docs[id]
	if !ok {
		return "", fmt.Errorf("fake: %s: %w", id, fs.ErrNotExist)
	}
	return content, nil
}

func (f *fakeCorpusReader) ListDocs(_ domain.DocFilter) ([]domain.DocMeta, error) {
	out := make([]domain.DocMeta, 0, len(f.docs))
	for name := range f.docs {
		out = append(out, domain.DocMeta{Filename: name})
	}
	return out, nil
}

// strict is the default strictness used by most tests. Kept as a helper
// so that when the story later adds knobs, only one place changes.
func strict() EvidenceValidation {
	return EvidenceValidation{Required: true, MinConfidence: 0.4, Mode: EvidenceModeStrict}
}

// TestValidateFindings_ExactQuoteFound — the happy path: the quote is a
// literal substring of the source file and confidence is above the
// threshold, so the finding survives validation intact.
func TestValidateFindings_ExactQuoteFound(t *testing.T) {
	reader := &fakeCorpusReader{docs: map[string]string{
		"auth.md": "We will authenticate all API calls with stateless JWT tokens.",
	}}
	findings := []ReviewFinding{{
		Severity: "contradiction",
		Title:    "Auth strategy",
		Evidence: []Evidence{{
			File:  "auth.md",
			Quote: "stateless JWT tokens",
		}},
		Confidence: 0.9,
	}}
	res := ValidateFindings(findings, reader, strict())
	if len(res.Valid) != 1 {
		t.Fatalf("Valid = %d, want 1", len(res.Valid))
	}
	if len(res.Rejected) != 0 {
		t.Errorf("Rejected = %d, want 0", len(res.Rejected))
	}
}

// TestValidateFindings_QuoteNotFound — the AI invented a quote the
// document does not contain. Strict mode must drop it with a
// "quote not found" reason so the CLI can surface why.
func TestValidateFindings_QuoteNotFound(t *testing.T) {
	reader := &fakeCorpusReader{docs: map[string]string{
		"auth.md": "We will use sessions stored in Redis.",
	}}
	findings := []ReviewFinding{{
		Title: "Hallucinated contradiction",
		Evidence: []Evidence{{
			File:  "auth.md",
			Quote: "we will use stateless JWT tokens",
		}},
		Confidence: 0.9,
	}}
	res := ValidateFindings(findings, reader, strict())
	if len(res.Valid) != 0 {
		t.Errorf("Valid = %d, want 0", len(res.Valid))
	}
	if len(res.Rejected) != 1 {
		t.Fatalf("Rejected = %d, want 1", len(res.Rejected))
	}
	if !strings.Contains(res.Rejected[0].Reason, "quote not found") {
		t.Errorf("Reason = %q, want contains 'quote not found'", res.Rejected[0].Reason)
	}
}

// TestValidateFindings_FileNotFound — the evidence points at a file
// that does not exist in the corpus (typo, deleted doc, etc). Reject
// with the file-missing reason.
func TestValidateFindings_FileNotFound(t *testing.T) {
	reader := &fakeCorpusReader{docs: map[string]string{
		"auth.md": "anything",
	}}
	findings := []ReviewFinding{{
		Title: "Cites missing file",
		Evidence: []Evidence{{
			File:  "ghost.md",
			Quote: "whatever",
		}},
		Confidence: 0.9,
	}}
	res := ValidateFindings(findings, reader, strict())
	if len(res.Valid) != 0 {
		t.Errorf("Valid = %d, want 0", len(res.Valid))
	}
	if len(res.Rejected) != 1 || !strings.Contains(res.Rejected[0].Reason, "ghost.md") {
		t.Errorf("rejection reason = %v", res.Rejected)
	}
}

// TestValidateFindings_NoEvidence — a finding with an empty Evidence
// slice violates AC-4 rule 1. Strict mode drops it with the "no
// evidence provided" reason.
func TestValidateFindings_NoEvidence(t *testing.T) {
	reader := &fakeCorpusReader{docs: map[string]string{}}
	findings := []ReviewFinding{{
		Title:      "No citations",
		Evidence:   nil,
		Confidence: 0.9,
	}}
	res := ValidateFindings(findings, reader, strict())
	if len(res.Valid) != 0 {
		t.Errorf("Valid = %d, want 0", len(res.Valid))
	}
	if len(res.Rejected) != 1 || !strings.Contains(res.Rejected[0].Reason, "no evidence") {
		t.Errorf("rejection reason = %v", res.Rejected)
	}
}

// TestValidateFindings_WhitespaceNormalization — a quote copied with
// extra spaces or newlines must still match the source text. This is
// AC-5: the validator collapses whitespace runs before comparing.
func TestValidateFindings_WhitespaceNormalization(t *testing.T) {
	reader := &fakeCorpusReader{docs: map[string]string{
		"doc.md": "We decided\nto\tuse JWT   tokens.",
	}}
	findings := []ReviewFinding{{
		Title: "Whitespace tolerant",
		Evidence: []Evidence{{
			File:  "doc.md",
			Quote: "We decided to use JWT tokens.",
		}},
		Confidence: 0.9,
	}}
	res := ValidateFindings(findings, reader, strict())
	if len(res.Valid) != 1 {
		t.Errorf("expected quote to match after whitespace normalization, got Valid=%d Rejected=%v", len(res.Valid), res.Rejected)
	}
}

// TestValidateFindings_LenientKeepsRejected — lenient mode keeps the
// finding in the kept set but still records the rejection reason, so
// debug workflows can see what the strict validator would have dropped
// without losing the findings themselves.
func TestValidateFindings_LenientKeepsRejected(t *testing.T) {
	reader := &fakeCorpusReader{docs: map[string]string{
		"auth.md": "Sessions in Redis.",
	}}
	findings := []ReviewFinding{{
		Title: "Bad quote",
		Evidence: []Evidence{{
			File:  "auth.md",
			Quote: "JWT tokens",
		}},
		Confidence: 0.9,
	}}
	v := EvidenceValidation{Required: true, MinConfidence: 0.4, Mode: EvidenceModeLenient}
	res := ValidateFindings(findings, reader, v)
	if len(res.Valid) != 1 {
		t.Errorf("lenient mode should keep the finding, Valid=%d", len(res.Valid))
	}
	if len(res.Rejected) != 1 {
		t.Errorf("lenient mode should still record the rejection, Rejected=%d", len(res.Rejected))
	}
}

// TestValidateFindings_ConfidenceThreshold — a finding with confidence
// below MinConfidence is dropped with a clear reason, even if its
// evidence would otherwise validate.
func TestValidateFindings_ConfidenceThreshold(t *testing.T) {
	reader := &fakeCorpusReader{docs: map[string]string{
		"auth.md": "We will use JWT.",
	}}
	findings := []ReviewFinding{{
		Title: "Low confidence",
		Evidence: []Evidence{{
			File:  "auth.md",
			Quote: "JWT",
		}},
		Confidence: 0.2,
	}}
	res := ValidateFindings(findings, reader, strict())
	if len(res.Valid) != 0 {
		t.Errorf("Valid = %d, want 0", len(res.Valid))
	}
	if len(res.Rejected) != 1 || !strings.Contains(res.Rejected[0].Reason, "confidence") {
		t.Errorf("rejection reason = %v", res.Rejected)
	}
}

// TestValidateFindings_OffIsNoop — Required=false or Mode=off bypass the
// validator entirely so the old behavior is preserved for escape-hatch
// users. Covers AC-9.
func TestValidateFindings_OffIsNoop(t *testing.T) {
	reader := &fakeCorpusReader{docs: map[string]string{}}
	findings := []ReviewFinding{{Title: "No evidence at all"}}

	// Required=false
	res := ValidateFindings(findings, reader, EvidenceValidation{Required: false})
	if len(res.Valid) != 1 || len(res.Rejected) != 0 {
		t.Errorf("Required=false: Valid=%d Rejected=%d", len(res.Valid), len(res.Rejected))
	}
	// Mode=off
	res = ValidateFindings(findings, reader, EvidenceValidation{Required: true, Mode: EvidenceModeOff})
	if len(res.Valid) != 1 || len(res.Rejected) != 0 {
		t.Errorf("Mode=off: Valid=%d Rejected=%d", len(res.Valid), len(res.Rejected))
	}
}

// TestValidateFindings_MultipleEvidenceAllMustMatch — when a finding
// cites two files, BOTH quotes must validate. A single bad quote drops
// the whole finding, otherwise the AI could smuggle hallucinations in
// beside legitimate ones.
func TestValidateFindings_MultipleEvidenceAllMustMatch(t *testing.T) {
	reader := &fakeCorpusReader{docs: map[string]string{
		"a.md": "This is document A.",
		"b.md": "This is document B.",
	}}
	findings := []ReviewFinding{{
		Title: "Mixed evidence",
		Evidence: []Evidence{
			{File: "a.md", Quote: "document A"},
			{File: "b.md", Quote: "NEVER WRITTEN"}, // hallucinated
		},
		Confidence: 0.9,
	}}
	res := ValidateFindings(findings, reader, strict())
	if len(res.Valid) != 0 {
		t.Errorf("expected rejection when any quote fails, Valid=%d", len(res.Valid))
	}
}

// TestNormalizeWhitespace is a thin safety net around the normalization
// helper that the validator uses. If this regresses the whole story
// breaks silently, so the small explicit test is worth its weight.
func TestNormalizeWhitespace(t *testing.T) {
	cases := []struct {
		in, out string
	}{
		{"hello world", "hello world"},
		{"  hello\tworld  ", "hello world"},
		{"a\n\nb\n\tc", "a b c"},
		{"", ""},
	}
	for _, tc := range cases {
		got := normalizeWhitespace(tc.in)
		if got != tc.out {
			t.Errorf("normalizeWhitespace(%q) = %q, want %q", tc.in, got, tc.out)
		}
	}
}
