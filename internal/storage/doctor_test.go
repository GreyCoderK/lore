package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newDoctorDir creates a minimal .lore/docs/ structure for doctor tests.
func newDoctorDir(t *testing.T) string {
	t.Helper()
	docsDir := filepath.Join(t.TempDir(), ".lore", "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	return docsDir
}

// writeDoc writes a valid doc with front matter to the docs dir.
func writeDoc(t *testing.T, docsDir, filename, docType, date string, related []string) {
	t.Helper()
	var relatedYAML string
	if len(related) > 0 {
		relatedYAML = "related:\n"
		for _, r := range related {
			relatedYAML += "  - " + r + "\n"
		}
	}
	content := "---\ntype: " + docType + "\ndate: " + date + "\nstatus: published\n" + relatedYAML + "---\n# Test\n\nBody.\n"
	if err := os.WriteFile(filepath.Join(docsDir, filename), []byte(content), 0o644); err != nil {
		t.Fatalf("write doc %s: %v", filename, err)
	}
}

// --- Diagnose Tests ---

func TestDiagnose_CleanCorpus(t *testing.T) {
	docsDir := newDoctorDir(t)
	writeDoc(t, docsDir, "decision-auth-2026-03-07.md", "decision", "2026-03-07", nil)

	// Generate a valid index
	if err := RegenerateIndex(docsDir); err != nil {
		t.Fatalf("RegenerateIndex: %v", err)
	}

	report, err := Diagnose(docsDir)
	if err != nil {
		t.Fatalf("Diagnose: %v", err)
	}
	if len(report.Issues) != 0 {
		t.Errorf("expected 0 issues, got %d: %+v", len(report.Issues), report.Issues)
	}
	if report.DocCount != 1 {
		t.Errorf("DocCount = %d, want 1", report.DocCount)
	}
}

func TestDiagnose_OrphanTmpFiles(t *testing.T) {
	docsDir := newDoctorDir(t)
	writeDoc(t, docsDir, "note-test-2026-03-07.md", "note", "2026-03-07", nil)
	RegenerateIndex(docsDir)

	// Create orphan .tmp file
	tmpPath := filepath.Join(docsDir, "decision-auth-2026-03-07.md.tmp")
	os.WriteFile(tmpPath, []byte("partial write"), 0o644)

	report, err := Diagnose(docsDir)
	if err != nil {
		t.Fatalf("Diagnose: %v", err)
	}

	found := false
	for _, issue := range report.Issues {
		if issue.Category == "orphan-tmp" && issue.File == "decision-auth-2026-03-07.md.tmp" {
			found = true
			if !issue.AutoFix {
				t.Error("orphan-tmp should be auto-fixable")
			}
		}
	}
	if !found {
		t.Errorf("expected orphan-tmp issue, got: %+v", report.Issues)
	}
}

func TestDiagnose_BrokenReferences(t *testing.T) {
	docsDir := newDoctorDir(t)
	// Doc A references "nonexistent-doc" which doesn't exist
	writeDoc(t, docsDir, "decision-auth-2026-03-07.md", "decision", "2026-03-07", []string{"nonexistent-doc"})
	RegenerateIndex(docsDir)

	report, err := Diagnose(docsDir)
	if err != nil {
		t.Fatalf("Diagnose: %v", err)
	}

	found := false
	for _, issue := range report.Issues {
		if issue.Category == "broken-ref" {
			found = true
			if issue.AutoFix {
				t.Error("broken-ref should NOT be auto-fixable")
			}
			if !strings.Contains(issue.Detail, "nonexistent-doc") {
				t.Errorf("expected detail to mention nonexistent-doc, got: %q", issue.Detail)
			}
		}
	}
	if !found {
		t.Errorf("expected broken-ref issue, got: %+v", report.Issues)
	}
}

func TestDiagnose_StaleIndex(t *testing.T) {
	docsDir := newDoctorDir(t)
	writeDoc(t, docsDir, "note-test-2026-03-07.md", "note", "2026-03-07", nil)

	// Write a stale README that doesn't match
	os.WriteFile(filepath.Join(docsDir, "README.md"), []byte("# Old Index\n"), 0o644)

	report, err := Diagnose(docsDir)
	if err != nil {
		t.Fatalf("Diagnose: %v", err)
	}

	found := false
	for _, issue := range report.Issues {
		if issue.Category == "stale-index" {
			found = true
			if !issue.AutoFix {
				t.Error("stale-index should be auto-fixable")
			}
		}
	}
	if !found {
		t.Errorf("expected stale-index issue, got: %+v", report.Issues)
	}
}

func TestDiagnose_InvalidFrontMatter(t *testing.T) {
	docsDir := newDoctorDir(t)
	RegenerateIndex(docsDir)

	// Write a file with invalid YAML front matter
	bad := "---\n{{invalid yaml\n---\n# Bad\n"
	os.WriteFile(filepath.Join(docsDir, "feature-bad-2026-03-07.md"), []byte(bad), 0o644)

	report, err := Diagnose(docsDir)
	if err != nil {
		t.Fatalf("Diagnose: %v", err)
	}

	found := false
	for _, issue := range report.Issues {
		if issue.Category == "invalid-frontmatter" {
			found = true
			if issue.AutoFix {
				t.Error("invalid-frontmatter should NOT be auto-fixable")
			}
		}
	}
	if !found {
		t.Errorf("expected invalid-frontmatter issue, got: %+v", report.Issues)
	}
}

func TestDiagnose_MissingReadme(t *testing.T) {
	docsDir := newDoctorDir(t)
	writeDoc(t, docsDir, "note-test-2026-03-07.md", "note", "2026-03-07", nil)
	// No README.md at all

	report, err := Diagnose(docsDir)
	if err != nil {
		t.Fatalf("Diagnose: %v", err)
	}

	found := false
	for _, issue := range report.Issues {
		if issue.Category == "stale-index" {
			found = true
		}
	}
	if !found {
		t.Error("expected stale-index issue when README.md is missing")
	}
}

// --- Fix Tests ---

func TestFix_OrphanTmpRemoved(t *testing.T) {
	docsDir := newDoctorDir(t)

	// Create orphan .tmp file with old modtime
	tmpPath := filepath.Join(docsDir, "old-write.md.tmp")
	os.WriteFile(tmpPath, []byte("partial"), 0o644)
	// Set modtime to 10 seconds ago to pass the age check
	os.Chtimes(tmpPath, time.Now().Add(-10*time.Second), time.Now().Add(-10*time.Second))

	report := &DiagnosticReport{
		Issues: []Issue{
			{Category: "orphan-tmp", File: "old-write.md.tmp", AutoFix: true},
		},
	}

	fixReport, err := Fix(docsDir, report)
	if err != nil {
		t.Fatalf("Fix: %v", err)
	}
	if fixReport.Fixed != 1 {
		t.Errorf("Fixed = %d, want 1", fixReport.Fixed)
	}
	if fixReport.Errors != 0 {
		t.Errorf("Errors = %d, want 0; details: %v", fixReport.Errors, fixReport.Details)
	}

	// Verify file is gone
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("expected .tmp file to be removed")
	}
}

func TestFix_StaleIndexRegenerated(t *testing.T) {
	docsDir := newDoctorDir(t)
	writeDoc(t, docsDir, "note-test-2026-03-07.md", "note", "2026-03-07", nil)

	report := &DiagnosticReport{
		Issues: []Issue{
			{Category: "stale-index", File: "README.md", AutoFix: true},
		},
	}

	fixReport, err := Fix(docsDir, report)
	if err != nil {
		t.Fatalf("Fix: %v", err)
	}
	if fixReport.Fixed != 1 {
		t.Errorf("Fixed = %d, want 1", fixReport.Fixed)
	}

	// Verify README.md now exists and references the doc
	data, readErr := os.ReadFile(filepath.Join(docsDir, "README.md"))
	if readErr != nil {
		t.Fatalf("ReadFile README.md: %v", readErr)
	}
	if !strings.Contains(string(data), "note-test-2026-03-07.md") {
		t.Error("regenerated index should reference note-test-2026-03-07.md")
	}
}

func TestFix_InvalidFrontMatterNotFixed(t *testing.T) {
	docsDir := newDoctorDir(t)

	report := &DiagnosticReport{
		Issues: []Issue{
			{Category: "invalid-frontmatter", File: "bad.md", AutoFix: false},
		},
	}

	fixReport, err := Fix(docsDir, report)
	if err != nil {
		t.Fatalf("Fix: %v", err)
	}
	if fixReport.Fixed != 0 {
		t.Errorf("Fixed = %d, want 0 (invalid-frontmatter is manual)", fixReport.Fixed)
	}
	if fixReport.Remaining != 1 {
		t.Errorf("Remaining = %d, want 1", fixReport.Remaining)
	}
}

func TestFix_NeverDeletesMdFiles(t *testing.T) {
	docsDir := newDoctorDir(t)
	writeDoc(t, docsDir, "decision-auth-2026-03-07.md", "decision", "2026-03-07", nil)

	// Even if we somehow pass a .md file as orphan-tmp, validateFilename won't help
	// but the Fix function only processes AutoFix=true issues with specific categories
	report := &DiagnosticReport{
		Issues: []Issue{
			{Category: "broken-ref", File: "decision-auth-2026-03-07.md", AutoFix: false},
		},
	}

	fixReport, err := Fix(docsDir, report)
	if err != nil {
		t.Fatalf("Fix: %v", err)
	}
	if fixReport.Fixed != 0 {
		t.Errorf("Fixed = %d, want 0 (broken-ref is not auto-fixable)", fixReport.Fixed)
	}

	// Verify .md file still exists
	if _, statErr := os.Stat(filepath.Join(docsDir, "decision-auth-2026-03-07.md")); statErr != nil {
		t.Error("Fix should NEVER delete .md user files")
	}
}

func TestFix_TmpTooRecent_Skipped(t *testing.T) {
	docsDir := newDoctorDir(t)

	// Create a very recent .tmp file (simulates concurrent write)
	tmpPath := filepath.Join(docsDir, "concurrent.md.tmp")
	os.WriteFile(tmpPath, []byte("in progress"), 0o644)
	// Don't change modtime — it's NOW, so <5s check applies

	report := &DiagnosticReport{
		Issues: []Issue{
			{Category: "orphan-tmp", File: "concurrent.md.tmp", AutoFix: true},
		},
	}

	fixReport, err := Fix(docsDir, report)
	if err != nil {
		t.Fatalf("Fix: %v", err)
	}
	// Should fail (too recent) and count as error
	if fixReport.Errors != 1 {
		t.Errorf("Errors = %d, want 1 (too recent)", fixReport.Errors)
	}

	// File should still exist
	if _, statErr := os.Stat(tmpPath); statErr != nil {
		t.Error("too-recent .tmp should NOT be removed")
	}
}

func TestDiagnose_ValidRelatedReference(t *testing.T) {
	docsDir := newDoctorDir(t)
	// Doc A references Doc B — both exist
	writeDoc(t, docsDir, "decision-auth-2026-03-07.md", "decision", "2026-03-07", []string{"feature-jwt-2026-03-08"})
	writeDoc(t, docsDir, "feature-jwt-2026-03-08.md", "feature", "2026-03-08", nil)
	RegenerateIndex(docsDir)

	report, err := Diagnose(docsDir)
	if err != nil {
		t.Fatalf("Diagnose: %v", err)
	}

	for _, issue := range report.Issues {
		if issue.Category == "broken-ref" {
			t.Errorf("unexpected broken-ref issue: %+v", issue)
		}
	}
}
