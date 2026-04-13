// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"context"
	"testing"
)

// TestReview_E2E_RejectsHallucinatedQuote walks the full Review() flow
// with a mock provider that returns a plausible-looking finding whose
// evidence quote does NOT exist in the actual corpus. The validator
// must catch the hallucination and move the finding into
// ReviewReport.Rejected instead of Findings.
//
// This is the story's end-to-end guarantee: even if the AI produces a
// well-formed JSON response, an unverifiable quote is enough to pull
// the finding before the user sees it.
func TestReview_E2E_RejectsHallucinatedQuote(t *testing.T) {
	// Corpus: a single document whose actual content talks about Redis
	// sessions. The AI's (fake) response will invent a JWT quote.
	reader := &fakeCorpusReader{docs: map[string]string{
		"auth.md": "The auth layer uses server-side sessions stored in Redis.",
	}}

	aiResponse := `{"findings": [{
		"severity": "contradiction",
		"title": "Fake auth contradiction",
		"description": "The AI claims a JWT/session conflict that does not exist.",
		"documents": ["auth.md"],
		"evidence": [
			{"file": "auth.md", "quote": "we will authenticate all API calls with stateless JWT tokens"}
		],
		"confidence": 0.95
	}]}`

	provider := newMockProviderWith(aiResponse, nil)
	docs := []DocSummary{{Filename: "auth.md", Summary: "sessions"}}

	report, err := Review(context.Background(), provider, docs, "", ReviewOpts{
		Reader: reader,
		Evidence: EvidenceValidation{
			Required:      true,
			MinConfidence: 0.4,
			Mode:          EvidenceModeStrict,
		},
	})
	if err != nil {
		t.Fatalf("Review: %v", err)
	}
	if len(report.Findings) != 0 {
		t.Errorf("expected hallucinated finding to be pulled, Findings=%d", len(report.Findings))
	}
	if len(report.Rejected) != 1 {
		t.Fatalf("expected 1 rejected finding, got %d", len(report.Rejected))
	}
	if report.Rejected[0].Finding.Title != "Fake auth contradiction" {
		t.Errorf("rejected title = %q", report.Rejected[0].Finding.Title)
	}
}

// TestReview_E2E_KeepsGenuineQuote is the symmetric sanity check: when
// the AI's response cites a quote that IS in the document, the finding
// passes through Review() unmodified. Without this test a validator
// that rejected everything would also pass the hallucination test.
func TestReview_E2E_KeepsGenuineQuote(t *testing.T) {
	reader := &fakeCorpusReader{docs: map[string]string{
		"auth.md": "We will authenticate all API calls with stateless JWT tokens.",
	}}
	aiResponse := `{"findings": [{
		"severity": "gap",
		"title": "Real finding",
		"description": "Genuine gap grounded in a real quote.",
		"documents": ["auth.md"],
		"evidence": [
			{"file": "auth.md", "quote": "stateless JWT tokens"}
		],
		"confidence": 0.85
	}]}`
	provider := newMockProviderWith(aiResponse, nil)
	docs := []DocSummary{{Filename: "auth.md"}}

	report, err := Review(context.Background(), provider, docs, "", ReviewOpts{
		Reader:   reader,
		Evidence: EvidenceValidation{Required: true, MinConfidence: 0.4, Mode: EvidenceModeStrict},
	})
	if err != nil {
		t.Fatalf("Review: %v", err)
	}
	if len(report.Findings) != 1 {
		t.Fatalf("expected 1 kept finding, got %d", len(report.Findings))
	}
	if len(report.Rejected) != 0 {
		t.Errorf("expected 0 rejected, got %d", len(report.Rejected))
	}
	// Evidence should survive unmodified through the parse→validate→sort pipeline.
	if len(report.Findings[0].Evidence) != 1 {
		t.Errorf("evidence lost through Review flow: %v", report.Findings[0])
	}
}
