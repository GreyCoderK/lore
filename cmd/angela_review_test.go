// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bytes"
	"strings"
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
	for _, want := range []string{"1 contradiction", "2 gap", "1 style"} {
		if !strings.Contains(result, want) {
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
	if !strings.Contains(result, "1 contradiction") {
		t.Errorf("expected '1 contradiction', got %q", result)
	}
	if !strings.Contains(result, "1 custom-sev") {
		t.Errorf("expected '1 custom-sev' (unknown severity), got %q", result)
	}
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
	formatReviewReport(streams, report, 10, false, false, false)
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
	formatReviewReport(streams, report, 10, false, false, false)
	stdout := out.String()
	// Paranoid-review fix: replace the previous vacuous-truth
	// assertion (`len(stdout) == 0`) with explicit content checks.
	// A formatter that printed only whitespace used to pass.
	expectContains := []string{
		"JWT conflict",
		"contradiction",
		"a.md",
		"b.md",
		"Two docs disagree",
		"Naming",
		"style",
		"Inconsistent terms",
	}
	for _, want := range expectContains {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout missing %q, got:\n%s", want, stdout)
		}
	}
	// The severity summary line should have surfaced the aggregated counts.
	if !strings.Contains(errBuf.String(), "contradiction") {
		t.Errorf("stderr missing severity summary, got: %s", errBuf.String())
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
	formatReviewReport(streams, report, 5, true, false, false)
	// Quiet mode: stderr should be empty (no header, no summary)
	if errBuf.Len() > 0 {
		t.Errorf("quiet mode should suppress stderr, got: %s", errBuf.String())
	}
}
