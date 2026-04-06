// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bytes"
	"testing"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/domain"
)

func TestCountSeverities(t *testing.T) {
	findings := []angela.ReviewFinding{
		{Severity: "contradiction"},
		{Severity: "gap"},
		{Severity: "gap"},
		{Severity: "style"},
	}
	result := countSeverities(findings)
	if result == "" {
		t.Fatal("countSeverities returned empty string")
	}
	// Should contain counts
	tests := []string{"1 contradiction", "2 gap", "1 style"}
	for _, want := range tests {
		found := false
		for i := 0; i <= len(result)-len(want); i++ {
			if result[i:i+len(want)] == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("countSeverities = %q, missing %q", result, want)
		}
	}
}

func TestCountSeverities_UnknownSeverity(t *testing.T) {
	findings := []angela.ReviewFinding{
		{Severity: "contradiction"},
		{Severity: "custom-sev"},
	}
	result := countSeverities(findings)
	if !contains(result, "1 contradiction") {
		t.Errorf("expected '1 contradiction', got %q", result)
	}
	if !contains(result, "1 custom-sev") {
		t.Errorf("expected '1 custom-sev' (unknown severity), got %q", result)
	}
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestCountSeverities_Empty(t *testing.T) {
	result := countSeverities(nil)
	if result != "" {
		t.Errorf("countSeverities(nil) = %q, want empty", result)
	}
}

func TestFormatReviewReport_NoFindings(t *testing.T) {
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}
	report := &angela.ReviewReport{
		Findings: []angela.ReviewFinding{},
		DocCount: 10,
	}
	formatReviewReport(streams, report, 10, false)
	// Should not panic and stderr should have header
	errOut := streams.Err.(*bytes.Buffer).String()
	if len(errOut) == 0 {
		t.Error("expected stderr output for review header")
	}
}

func TestFormatReviewReport_WithFindings(t *testing.T) {
	out := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	streams := domain.IOStreams{Out: out, Err: errBuf}
	report := &angela.ReviewReport{
		Findings: []angela.ReviewFinding{
			{Severity: "contradiction", Title: "JWT conflict", Description: "Two docs disagree", Documents: []string{"a.md", "b.md"}},
			{Severity: "style", Title: "Naming", Description: "Inconsistent terms"},
		},
		DocCount: 8,
	}
	formatReviewReport(streams, report, 10, false)
	stdout := out.String()
	if len(stdout) == 0 {
		t.Error("expected findings on stdout")
	}
}

func TestFormatReviewReport_Quiet(t *testing.T) {
	out := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	streams := domain.IOStreams{Out: out, Err: errBuf}
	report := &angela.ReviewReport{
		Findings: []angela.ReviewFinding{
			{Severity: "gap", Title: "Missing docs"},
		},
		DocCount: 5,
	}
	formatReviewReport(streams, report, 5, true)
	// Quiet mode: stderr should be empty (no header, no summary)
	if errBuf.Len() > 0 {
		t.Errorf("quiet mode should suppress stderr, got: %s", errBuf.String())
	}
}
