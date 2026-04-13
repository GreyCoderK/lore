// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/domain"
)

// ─────────────────────────────────────────────────────────────────
// newDraftReporter — format variants (covers 60% → higher)
// ─────────────────────────────────────────────────────────────────

func TestNewDraftReporter_JSONFormat(t *testing.T) {
	streams := domain.IOStreams{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}}
	r := newDraftReporter("json", streams, false)
	if r == nil {
		t.Fatal("expected non-nil reporter for json format")
	}
	// Call ReportFile — no-op for JSON, should not panic.
	r.ReportFile(DraftFileReport{Filename: "test.md"})
}

func TestNewDraftReporter_UnknownFormat(t *testing.T) {
	errBuf := &bytes.Buffer{}
	streams := domain.IOStreams{Out: &bytes.Buffer{}, Err: errBuf}
	r := newDraftReporter("bogus-format", streams, false)
	if r == nil {
		t.Fatal("expected non-nil reporter for unknown format (fallback)")
	}
	// Unknown format should warn and fall back to human.
	if !strings.Contains(errBuf.String(), "bogus-format") {
		t.Errorf("expected warning about unknown format, got: %s", errBuf.String())
	}
}

func TestNewDraftReporter_HumanDefault(t *testing.T) {
	streams := domain.IOStreams{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}}
	r := newDraftReporter("", streams, false)
	if r == nil {
		t.Fatal("expected non-nil reporter for default (human) format")
	}
}

func TestNewDraftReporter_HumanExplicit(t *testing.T) {
	streams := domain.IOStreams{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}}
	r := newDraftReporter("human", streams, true)
	if r == nil {
		t.Fatal("expected non-nil reporter for human format")
	}
}

// ─────────────────────────────────────────────────────────────────
// formatReviewReport — additional coverage paths
// ─────────────────────────────────────────────────────────────────

func TestFormatReviewReport_PartialCorpusSmall(t *testing.T) {
	errBuf := &bytes.Buffer{}
	streams := domain.IOStreams{Out: &bytes.Buffer{}, Err: errBuf}
	report := &angela.ReviewReport{
		Findings: []angela.ReviewFinding{},
		DocCount: 3,
	}
	// totalCorpus > report.DocCount → partial header
	formatReviewReport(streams, report, 50, false, false, false)
	if !strings.Contains(errBuf.String(), "3") {
		t.Errorf("expected doc count in partial header, got: %s", errBuf.String())
	}
}

func TestFormatReviewReport_DiffOnly_FiltersPresisting(t *testing.T) {
	out := &bytes.Buffer{}
	streams := domain.IOStreams{Out: out, Err: &bytes.Buffer{}}
	report := &angela.ReviewReport{
		DocCount: 3,
		Findings: []angela.ReviewFinding{
			{Severity: "gap", Title: "NEW finding", DiffStatus: angela.ReviewDiffNew},
			{Severity: "gap", Title: "PERSISTING finding", DiffStatus: angela.ReviewDiffPersisting},
		},
	}
	formatReviewReport(streams, report, 3, false, false, true)
	stdout := out.String()
	if !strings.Contains(stdout, "NEW finding") {
		t.Errorf("NEW finding should appear in diffOnly mode, got: %s", stdout)
	}
	if strings.Contains(stdout, "PERSISTING finding") {
		t.Errorf("PERSISTING finding should be filtered in diffOnly mode, got: %s", stdout)
	}
}

func TestFormatReviewReport_WithHash(t *testing.T) {
	out := &bytes.Buffer{}
	streams := domain.IOStreams{Out: out, Err: &bytes.Buffer{}}
	report := &angela.ReviewReport{
		DocCount: 2,
		Findings: []angela.ReviewFinding{
			{
				Severity:    "contradiction",
				Title:       "Hash test",
				Hash:        "abcdef1234567890",
				Description: "A detailed description.",
				Documents:   []string{"a.md", "b.md"},
			},
		},
	}
	formatReviewReport(streams, report, 2, false, false, false)
	stdout := out.String()
	if !strings.Contains(stdout, "abcdef") {
		t.Errorf("expected short hash in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "A detailed description.") {
		t.Errorf("expected description in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "a.md") {
		t.Errorf("expected document in output, got: %s", stdout)
	}
}

func TestFormatReviewReport_VerboseRejected(t *testing.T) {
	errBuf := &bytes.Buffer{}
	streams := domain.IOStreams{Out: &bytes.Buffer{}, Err: errBuf}
	report := &angela.ReviewReport{
		DocCount: 2,
		Findings: []angela.ReviewFinding{},
		Rejected: []angela.RejectedFinding{
			{Finding: angela.ReviewFinding{Title: "Hallucinated doc ref"}, Reason: "document not found"},
		},
	}
	formatReviewReport(streams, report, 2, false, true, false)
	if !strings.Contains(errBuf.String(), "Hallucinated doc ref") {
		t.Errorf("verbose mode should show rejected findings, got: %s", errBuf.String())
	}
}

func TestFormatReviewReport_WithDiff(t *testing.T) {
	errBuf := &bytes.Buffer{}
	streams := domain.IOStreams{Out: &bytes.Buffer{}, Err: errBuf}
	resolved := angela.ReviewFinding{Title: "Fixed gap", Severity: "gap"}
	diff := &angela.ReviewDiff{
		New:      []angela.ReviewFinding{{Title: "NewIssue"}},
		Resolved: []angela.ReviewFinding{resolved},
	}
	report := &angela.ReviewReport{
		DocCount: 3,
		Findings: []angela.ReviewFinding{},
		Diff:     diff,
	}
	// verbose=true to also exercise the resolved listing path.
	formatReviewReport(streams, report, 3, false, true, false)
	errStr := errBuf.String()
	// Differential summary should mention counts.
	if !strings.Contains(errStr, "0") && !strings.Contains(errStr, "1") {
		t.Errorf("expected diff summary in output, got: %s", errStr)
	}
}

func TestFormatReviewReport_ReggressedMarker(t *testing.T) {
	out := &bytes.Buffer{}
	streams := domain.IOStreams{Out: out, Err: &bytes.Buffer{}}
	report := &angela.ReviewReport{
		DocCount: 2,
		Findings: []angela.ReviewFinding{
			{Severity: "gap", Title: "Regressed", DiffStatus: angela.ReviewDiffRegressed},
		},
	}
	formatReviewReport(streams, report, 2, false, false, false)
	stdout := out.String()
	if !strings.Contains(stdout, "!") {
		t.Errorf("expected '!' marker for regressed finding, got: %s", stdout)
	}
}

func TestFormatReviewReport_WithRelevance(t *testing.T) {
	out := &bytes.Buffer{}
	streams := domain.IOStreams{Out: out, Err: &bytes.Buffer{}}
	report := &angela.ReviewReport{
		DocCount: 1,
		Findings: []angela.ReviewFinding{
			{Severity: "style", Title: "Style issue", Relevance: "high"},
		},
	}
	formatReviewReport(streams, report, 1, false, false, false)
	if !strings.Contains(out.String(), "high") {
		t.Errorf("expected relevance in output, got: %s", out.String())
	}
}

// ─────────────────────────────────────────────────────────────────
// DraftReport helpers
// ─────────────────────────────────────────────────────────────────

func TestDraftReport_AllSuggestions(t *testing.T) {
	r := DraftReport{
		Files: []DraftFileReport{
			{
				Filename:    "a.md",
				Suggestions: []angela.Suggestion{{Category: "style", Message: "fix"}},
			},
			{
				Filename:    "b.md",
				Suggestions: []angela.Suggestion{{Category: "coherence", Message: "add link"}},
			},
		},
	}
	all := r.allSuggestions()
	if len(all) != 2 {
		t.Errorf("expected 2 suggestions, got %d", len(all))
	}
}

func TestDraftReport_ComputeSummary(t *testing.T) {
	r := DraftReport{
		Files: []DraftFileReport{
			{
				Filename: "a.md",
				Score:    95,
				Suggestions: []angela.Suggestion{
					{Severity: angela.SeverityWarning, Category: "style", Message: "fix"},
				},
			},
			{
				Filename:    "b.md",
				Score:       100,
				Suggestions: []angela.Suggestion{},
			},
		},
	}
	r.computeSummary()
	if r.Summary.TotalSuggestions != 1 {
		t.Errorf("TotalSuggestions = %d, want 1", r.Summary.TotalSuggestions)
	}
	if r.Summary.BySeverity[angela.SeverityWarning] != 1 {
		t.Errorf("BySeverity[warning] = %d, want 1", r.Summary.BySeverity[angela.SeverityWarning])
	}
}

// ─────────────────────────────────────────────────────────────────
// jsonDraftReporter.ReportFile (line 166 — no-op, but uncovered)
// ─────────────────────────────────────────────────────────────────

func TestJsonReporterReportFile_NoOp(t *testing.T) {
	out := &bytes.Buffer{}
	streams := domain.IOStreams{Out: out, Err: &bytes.Buffer{}}
	r := newDraftReporter("json", streams, false)
	// ReportFile on JSON reporter should be a no-op (no writes to out).
	r.ReportFile(DraftFileReport{Filename: "test.md", Score: 90})
	if out.Len() != 0 {
		t.Errorf("JSON ReportFile should write nothing, got: %s", out.String())
	}
}

// ─────────────────────────────────────────────────────────────────
// runVHSCheck — no tape dir path (silent skip)
// ─────────────────────────────────────────────────────────────────

func TestRunVHSCheck_NoTapeDir(t *testing.T) {
	dir := t.TempDir()
	streams := domain.IOStreams{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}}
	// No assets/vhs directory — should return 0 silently.
	count := runVHSCheck(dir, streams)
	if count != 0 {
		t.Errorf("expected 0 findings for missing tape dir, got %d", count)
	}
}

func TestRunVHSCheck_EmptyTapeDir(t *testing.T) {
	dir := t.TempDir()
	// Create the tape dir but leave it empty.
	assetsDir := dir + "/assets/vhs"
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	docsDir := dir + "/docs"
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	streams := domain.IOStreams{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}}
	count := runVHSCheck(docsDir, streams)
	if count != 0 {
		t.Errorf("expected 0 findings for empty tape dir, got %d", count)
	}
}
