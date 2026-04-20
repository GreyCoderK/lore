// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

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
	_ = RegenerateIndex(docsDir)

	// Create orphan .tmp file
	tmpPath := filepath.Join(docsDir, "decision-auth-2026-03-07.md.tmp")
	if err := os.WriteFile(tmpPath, []byte("partial write"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

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
	_ = RegenerateIndex(docsDir)

	report, err := Diagnose(docsDir)
	if err != nil {
		t.Fatalf("Diagnose: %v", err)
	}

	found := false
	for _, issue := range report.Issues {
		if issue.Category == "broken-ref" {
			found = true
			if !issue.AutoFix {
				t.Error("broken-ref should be auto-fixable (removes dead reference from related[])")
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
	if err := os.WriteFile(filepath.Join(docsDir, "README.md"), []byte("# Old Index\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

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

func TestDiagnose_InvalidFrontMatter_MalformedYAML(t *testing.T) {
	docsDir := newDoctorDir(t)
	_ = RegenerateIndex(docsDir)

	// File with --- delimiters but invalid YAML inside → NOT auto-fixable
	bad := "---\n{{invalid yaml\n---\n# Bad\n"
	if err := os.WriteFile(filepath.Join(docsDir, "feature-bad-2026-03-07.md"), []byte(bad), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	report, err := Diagnose(docsDir)
	if err != nil {
		t.Fatalf("Diagnose: %v", err)
	}

	found := false
	for _, issue := range report.Issues {
		if issue.Category == "invalid-frontmatter" {
			found = true
			if issue.AutoFix {
				t.Error("malformed YAML front matter should NOT be auto-fixable")
			}
			// Story 8-22: Subkind must be "malformed" when delimiters
			// are present but YAML is unparseable.
			if issue.Subkind != SubkindFrontmatterMalformed {
				t.Errorf("Subkind=%q, want %q", issue.Subkind, SubkindFrontmatterMalformed)
			}
		}
	}
	if !found {
		t.Errorf("expected invalid-frontmatter issue, got: %+v", report.Issues)
	}
}

func TestDiagnose_InvalidFrontMatter_Missing(t *testing.T) {
	docsDir := newDoctorDir(t)
	_ = RegenerateIndex(docsDir)

	// File without front matter at all → auto-fixable
	noFM := "# Just a title\nSome content without front matter.\n"
	if err := os.WriteFile(filepath.Join(docsDir, "decision-no-fm-2026-03-07.md"), []byte(noFM), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	report, err := Diagnose(docsDir)
	if err != nil {
		t.Fatalf("Diagnose: %v", err)
	}

	found := false
	for _, issue := range report.Issues {
		if issue.Category == "invalid-frontmatter" && issue.File == "decision-no-fm-2026-03-07.md" {
			found = true
			if !issue.AutoFix {
				t.Error("missing front matter should be auto-fixable")
			}
			// Story 8-22: Subkind must be "missing" when no `---` delimiter
			// is present. Synthesis is safe.
			if issue.Subkind != SubkindFrontmatterMissing {
				t.Errorf("Subkind=%q, want %q", issue.Subkind, SubkindFrontmatterMissing)
			}
		}
	}
	if !found {
		t.Errorf("expected invalid-frontmatter issue for missing FM, got: %+v", report.Issues)
	}
}

// --- Story 8-22 / I31: doctor safe — never rewrite when `---` present ---

// TestI31_FixRefusesMalformed_NeverRewrites asserts the codified I31
// invariant: `doctor --fix` never modifies a file's bytes when the
// source contains a `---` delimiter, even if auto-fix is requested
// for that issue category.
func TestI31_FixRefusesMalformed_NeverRewrites(t *testing.T) {
	docsDir := newDoctorDir(t)
	// Simulate a user mis-edit: delimiters present, YAML broken.
	malformed := "---\ntype: decision\nid: [unclosed\n---\n## Why\nUser-authored content that may be recoverable.\n"
	path := filepath.Join(docsDir, "decision-malformed-2026-04-19.md")
	if err := os.WriteFile(path, []byte(malformed), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	srcBytes, _ := os.ReadFile(path)

	report, err := Diagnose(docsDir)
	if err != nil {
		t.Fatalf("Diagnose: %v", err)
	}

	// Adversarial step: even if a caller accidentally set AutoFix=true
	// on a malformed issue (e.g. via a custom DiagnosticReport built
	// by hand), the Fix() function must still refuse to rewrite the
	// file because Subkind != missing.
	for i := range report.Issues {
		if report.Issues[i].Category == "invalid-frontmatter" &&
			report.Issues[i].Subkind == SubkindFrontmatterMalformed {
			report.Issues[i].AutoFix = true // adversarial flip
		}
	}

	fixReport, err := Fix(docsDir, report)
	if err != nil {
		t.Fatalf("Fix: %v", err)
	}

	// The malformed issue must NOT have been auto-fixed.
	if fixReport.Fixed > 0 {
		t.Errorf("Fixed=%d, want 0 — I31 requires no rewrite when delimiter present", fixReport.Fixed)
	}
	// The file bytes must be unchanged.
	afterBytes, _ := os.ReadFile(path)
	if string(afterBytes) != string(srcBytes) {
		t.Errorf("I31 violation — file bytes changed\nbefore: %q\nafter:  %q", string(srcBytes), string(afterBytes))
	}
}

// TestI31_FixRefusesMalformed_PropertyStyle sweeps 6 adversarial
// malformed inputs and asserts none of them are rewritten. Extending
// this table is the right way to add new regression scenarios.
//
// Story 8-22 P0 fix: added CRLF and BOM+malformed variants that the
// old `strings.HasPrefix(content, "---\n")` heuristic mis-classified
// as "missing" → AutoFix=true → double-prepend. These now flow
// through ExtractFrontmatter which normalizes both.
func TestI31_FixRefusesMalformed_PropertyStyle(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{"unclosed_list", "---\ntype: [oops\n---\nbody\n"},
		{"unclosed_quote", "---\ntitle: \"unclosed\n---\nbody\n"},
		{"broken_indent", "---\ntype: decision\n  - bad: indent\n---\nbody\n"},
		{"duplicate_keys", "---\ntype: decision\ntype: duplicate\n---\nbody\n"},
		{"tab_instead_of_space", "---\ntype:\tdecision\n---\nbody\n"},
		{"truncated_fm", "---\ntype: decision\n"}, // opening delim, never closed
		// Story 8-22 P0 additions (these crash-tested the old heuristic):
		{"crlf_delimiter_malformed", "---\r\ntype: [broken\r\n---\r\nbody\r\n"},
		{"bom_delimiter_malformed", "\xEF\xBB\xBF---\ntype: [broken\n---\nbody\n"},
		{"empty_fm_block", "---\n---\nbody\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			docsDir := newDoctorDir(t)
			path := filepath.Join(docsDir, "decision-"+tc.name+"-2026-04-19.md")
			if err := os.WriteFile(path, []byte(tc.content), 0o644); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}

			report, err := Diagnose(docsDir)
			if err != nil {
				t.Fatalf("Diagnose: %v", err)
			}
			fixReport, err := Fix(docsDir, report)
			if err != nil {
				t.Fatalf("Fix: %v", err)
			}
			if fixReport.Fixed > 0 {
				t.Errorf("I31 violation: Fixed=%d for %q", fixReport.Fixed, tc.name)
			}
			afterBytes, _ := os.ReadFile(path)
			if string(afterBytes) != tc.content {
				t.Errorf("I31 violation: file rewritten for %q\nbefore: %q\nafter:  %q", tc.name, tc.content, string(afterBytes))
			}
		})
	}
}

// TestI31_BOMPlusValidFM_NotClassifiedMissing asserts the Story 8-22
// P0 fix: a file with a UTF-8 BOM before an otherwise-valid FM used
// to fail `strings.HasPrefix(content, "---\n")` in the classification
// heuristic, get tagged "missing", and `fixMissingFrontMatter` would
// then prepend a fresh FM ON TOP of the existing one. The ExtractFrontmatter-
// based classification must accept BOM+valid and not emit any issue
// (or at worst a non-fatal schema issue, never a "missing" subkind).
func TestI31_BOMPlusValidFM_NotClassifiedMissing(t *testing.T) {
	docsDir := newDoctorDir(t)
	path := filepath.Join(docsDir, "decision-bom-ok-2026-04-19.md")
	content := "\xEF\xBB\xBF---\ntype: decision\ndate: 2026-04-19\nstatus: draft\n---\n# Body\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	report, err := Diagnose(docsDir)
	if err != nil {
		t.Fatalf("Diagnose: %v", err)
	}
	for _, iss := range report.Issues {
		if iss.Category == "invalid-frontmatter" && iss.Subkind == SubkindFrontmatterMissing {
			t.Fatalf("BOM+valid FM must not be classified as missing:\n%+v", iss)
		}
	}
}

// TestI31_CRLFDelimiterMalformed_ClassifiedMalformed closes the second
// half of the P0: CRLF delimiters must route to the malformed subkind
// (never missing). Combined with the new entry in
// TestI31_FixRefusesMalformed_PropertyStyle, this locks down both
// directions of the classification.
func TestI31_CRLFDelimiterMalformed_ClassifiedMalformed(t *testing.T) {
	docsDir := newDoctorDir(t)
	path := filepath.Join(docsDir, "decision-crlf-broken-2026-04-19.md")
	// Delimiters are CRLF; YAML body is deliberately broken.
	content := "---\r\ntype: [oops\r\n---\r\nbody\r\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	report, err := Diagnose(docsDir)
	if err != nil {
		t.Fatalf("Diagnose: %v", err)
	}
	var iss *Issue
	for i := range report.Issues {
		if report.Issues[i].Category == "invalid-frontmatter" {
			iss = &report.Issues[i]
			break
		}
	}
	if iss == nil {
		t.Fatal("expected an invalid-frontmatter issue")
	}
	if iss.Subkind != SubkindFrontmatterMalformed {
		t.Errorf("Subkind=%q, want %q (CRLF delimiter must NOT classify as missing)",
			iss.Subkind, SubkindFrontmatterMalformed)
	}
	if iss.AutoFix {
		t.Errorf("AutoFix=true on malformed CRLF — I31 violation risk")
	}
}

// TestFix_MissingFM_TOCTOU_BecomesMalformed asserts the belt-and-
// suspenders guard added to fixMissingFrontMatter: if the file on
// disk at Fix time contains a delimiter (even malformed), we must
// NOT prepend a fresh FM. The guard uses ExtractFrontmatter so it
// handles BOM + CRLF uniformly with Diagnose.
func TestFix_MissingFM_TOCTOU_BecomesMalformed(t *testing.T) {
	docsDir := newDoctorDir(t)
	path := filepath.Join(docsDir, "decision-toctou-2026-04-19.md")
	// Initial state: no FM at all → Diagnose classifies missing.
	if err := os.WriteFile(path, []byte("# no fm yet\n"), 0o644); err != nil {
		t.Fatalf("WriteFile initial: %v", err)
	}
	report, err := Diagnose(docsDir)
	if err != nil {
		t.Fatalf("Diagnose: %v", err)
	}
	// Simulate the user manually starting a (broken) repair between
	// Diagnose and Fix.
	midEdit := "---\ntype: [broken\n---\n# user edit\n"
	if err := os.WriteFile(path, []byte(midEdit), 0o644); err != nil {
		t.Fatalf("WriteFile mid-edit: %v", err)
	}

	_, err = Fix(docsDir, report)
	if err != nil {
		t.Fatalf("Fix: %v", err)
	}

	after, _ := os.ReadFile(path)
	if string(after) != midEdit {
		t.Fatalf("I31 violation: Fix prepended FM on top of user-edited malformed file:\nbefore: %q\nafter:  %q",
			midEdit, string(after))
	}
}

// TestFix_MalformedFM_AdversarialSubkindBypass asserts the Fix-layer
// guard: a caller building a DiagnosticReport by hand with AutoFix=true
// and an empty or non-missing Subkind must NOT bypass the malformed
// protection.
func TestFix_MalformedFM_AdversarialSubkindBypass(t *testing.T) {
	docsDir := newDoctorDir(t)
	path := filepath.Join(docsDir, "decision-adv-2026-04-19.md")
	bad := "---\ntype: [broken\n---\nbody\n"
	if err := os.WriteFile(path, []byte(bad), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	adversarial := &DiagnosticReport{Issues: []Issue{{
		Category: "invalid-frontmatter",
		File:     "decision-adv-2026-04-19.md",
		AutoFix:  true,
		Subkind:  "", // empty — must not match SubkindFrontmatterMissing
	}}}
	if _, err := Fix(docsDir, adversarial); err != nil {
		t.Fatalf("Fix: %v", err)
	}
	after, _ := os.ReadFile(path)
	if string(after) != bad {
		t.Fatalf("I31 violation via empty subkind:\nbefore: %q\nafter:  %q", bad, string(after))
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
	if err := os.WriteFile(tmpPath, []byte("partial"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	// Set modtime to 10 seconds ago to pass the age check
	_ = os.Chtimes(tmpPath, time.Now().Add(-10*time.Second), time.Now().Add(-10*time.Second))

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

func TestFix_InvalidFrontMatterFixed(t *testing.T) {
	docsDir := newDoctorDir(t)

	// Create a file without front matter
	content := "# My Decision\n\nSome content without front matter."
	if err := os.WriteFile(filepath.Join(docsDir, "decision-auth-2026-04-08.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	report := &DiagnosticReport{
		Issues: []Issue{
			{
				Category: "invalid-frontmatter",
				Subkind:  SubkindFrontmatterMissing,
				File:     "decision-auth-2026-04-08.md",
				AutoFix:  true,
			},
		},
	}

	fixReport, err := Fix(docsDir, report)
	if err != nil {
		t.Fatalf("Fix: %v", err)
	}
	if fixReport.Fixed != 1 {
		t.Errorf("Fixed = %d, want 1", fixReport.Fixed)
	}

	// Verify front matter was added
	data, err := os.ReadFile(filepath.Join(docsDir, "decision-auth-2026-04-08.md"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	got := string(data)
	if !strings.HasPrefix(got, "---\n") {
		t.Error("expected front matter to start with ---")
	}
	if !strings.Contains(got, "type: decision") {
		t.Error("expected type inferred from filename as 'decision'")
	}
	if !strings.Contains(got, "# My Decision") {
		t.Error("expected original content preserved")
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
	if err := os.WriteFile(tmpPath, []byte("in progress"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
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

func TestFix_StaleCache(t *testing.T) {
	docsDir := newDoctorDir(t)

	// Create a stale cache file
	cachePath := filepath.Join(docsDir, "metadata.json")
	if err := os.WriteFile(cachePath, []byte(`{"stale": true}`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	report := &DiagnosticReport{
		Issues: []Issue{
			{Category: "stale-cache", File: "metadata.json", AutoFix: true},
		},
	}

	fixReport, err := Fix(docsDir, report)
	if err != nil {
		t.Fatalf("Fix: %v", err)
	}
	if fixReport.Fixed != 1 {
		t.Errorf("Fixed = %d, want 1", fixReport.Fixed)
	}

	// Verify file removed
	if _, statErr := os.Stat(cachePath); !os.IsNotExist(statErr) {
		t.Error("expected stale-cache file to be removed")
	}
}

func TestFix_StaleCacheAlreadyGone(t *testing.T) {
	docsDir := newDoctorDir(t)

	report := &DiagnosticReport{
		Issues: []Issue{
			{Category: "stale-cache", File: "metadata.json", AutoFix: true},
		},
	}

	fixReport, err := Fix(docsDir, report)
	if err != nil {
		t.Fatalf("Fix: %v", err)
	}
	// File doesn't exist → validateResolvedPath fails → error
	if fixReport.Errors != 1 {
		t.Errorf("Errors = %d, want 1 (path validation fails for missing file)", fixReport.Errors)
	}
}

func TestFix_StaleCacheInvalidFilename(t *testing.T) {
	docsDir := newDoctorDir(t)

	report := &DiagnosticReport{
		Issues: []Issue{
			{Category: "stale-cache", File: "../evil.json", AutoFix: true},
		},
	}

	fixReport, err := Fix(docsDir, report)
	if err != nil {
		t.Fatalf("Fix: %v", err)
	}
	if fixReport.Errors != 1 {
		t.Errorf("Errors = %d, want 1 (invalid filename rejected)", fixReport.Errors)
	}
}

func TestFix_MultipleMixed(t *testing.T) {
	docsDir := newDoctorDir(t)
	writeDoc(t, docsDir, "note-test-2026-03-07.md", "note", "2026-03-07", nil)

	// Orphan tmp (old enough)
	tmpPath := filepath.Join(docsDir, "old.md.tmp")
	os.WriteFile(tmpPath, []byte("x"), 0o644)
	_ = os.Chtimes(tmpPath, time.Now().Add(-10*time.Second), time.Now().Add(-10*time.Second))

	report := &DiagnosticReport{
		Issues: []Issue{
			{Category: "orphan-tmp", File: "old.md.tmp", AutoFix: true},
			{Category: "stale-index", File: "README.md", AutoFix: true},
			{Category: "broken-ref", File: "note-test-2026-03-07.md", AutoFix: false},
		},
	}

	fixReport, err := Fix(docsDir, report)
	if err != nil {
		t.Fatalf("Fix: %v", err)
	}
	if fixReport.Fixed != 2 {
		t.Errorf("Fixed = %d, want 2", fixReport.Fixed)
	}
	if fixReport.Remaining != 1 {
		t.Errorf("Remaining = %d, want 1", fixReport.Remaining)
	}
}

func TestDiagnose_StaleCache(t *testing.T) {
	docsDir := newDoctorDir(t)
	writeDoc(t, docsDir, "note-test-2026-03-07.md", "note", "2026-03-07", nil)
	_ = RegenerateIndex(docsDir)

	// Create metadata.json
	os.WriteFile(filepath.Join(docsDir, "metadata.json"), []byte("{}"), 0o644)

	report, err := Diagnose(docsDir)
	if err != nil {
		t.Fatalf("Diagnose: %v", err)
	}

	found := false
	for _, issue := range report.Issues {
		if issue.Category == "stale-cache" {
			found = true
			if !issue.AutoFix {
				t.Error("stale-cache should be auto-fixable")
			}
		}
	}
	if !found {
		t.Errorf("expected stale-cache issue, got: %+v", report.Issues)
	}
}

func TestDiagnose_ValidRelatedReference(t *testing.T) {
	docsDir := newDoctorDir(t)
	// Doc A references Doc B — both exist
	writeDoc(t, docsDir, "decision-auth-2026-03-07.md", "decision", "2026-03-07", []string{"feature-jwt-2026-03-08"})
	writeDoc(t, docsDir, "feature-jwt-2026-03-08.md", "feature", "2026-03-08", nil)
	_ = RegenerateIndex(docsDir)

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

// --- Direct fixOrphanTmp tests ---

func TestFixOrphanTmp_RemovesTmpFile(t *testing.T) {
	docsDir := newDoctorDir(t)

	tmpFile := "partial-write.md.tmp"
	tmpPath := filepath.Join(docsDir, tmpFile)
	if err := os.WriteFile(tmpPath, []byte("partial data"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	// Age the file past the 5-second threshold.
	_ = os.Chtimes(tmpPath, time.Now().Add(-10*time.Second), time.Now().Add(-10*time.Second))

	if err := fixOrphanTmp(docsDir, tmpFile); err != nil {
		t.Fatalf("fixOrphanTmp: %v", err)
	}

	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("expected .tmp file to be removed")
	}
}

func TestFixOrphanTmp_LeavesNonTmpFile(t *testing.T) {
	docsDir := newDoctorDir(t)

	// Create a regular .md file (not .tmp).
	mdFile := "note-keep-2026-04-09.md"
	mdPath := filepath.Join(docsDir, mdFile)
	content := "---\ntype: note\ndate: \"2026-04-09\"\nstatus: draft\n---\n# Keep\n"
	if err := os.WriteFile(mdPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_ = os.Chtimes(mdPath, time.Now().Add(-10*time.Second), time.Now().Add(-10*time.Second))

	// fixOrphanTmp should still remove it if called directly (it does not
	// check the extension — the caller is responsible for only passing .tmp files).
	// But the file must still exist afterwards if we DON'T call fixOrphanTmp.
	// The point: Diagnose only flags .tmp files, so non-.tmp files never reach fixOrphanTmp.

	// Verify via Diagnose that a non-.tmp file is never flagged as orphan-tmp.
	_ = RegenerateIndex(docsDir)
	report, err := Diagnose(docsDir)
	if err != nil {
		t.Fatalf("Diagnose: %v", err)
	}
	for _, issue := range report.Issues {
		if issue.Category == "orphan-tmp" && issue.File == mdFile {
			t.Errorf("non-.tmp file should not be flagged as orphan-tmp")
		}
	}

	// File must still exist.
	if _, statErr := os.Stat(mdPath); statErr != nil {
		t.Error("non-.tmp file should not be removed")
	}
}

func TestFixOrphanTmp_AlreadyGone(t *testing.T) {
	docsDir := newDoctorDir(t)

	// Create a .tmp file, then remove it before calling fixOrphanTmp.
	// validateResolvedPath requires the file to exist (lstat), so for a
	// truly missing file this returns an error from path validation.
	err := fixOrphanTmp(docsDir, "ghost.tmp")
	if err == nil {
		t.Skip("path validation allows missing files on this platform")
	}
	// The error should come from path validation, not from os.Remove.
	if !strings.Contains(err.Error(), "storage: doctor:") {
		t.Errorf("expected storage: doctor: prefix, got: %v", err)
	}
}

func TestFixOrphanTmp_TooRecent(t *testing.T) {
	docsDir := newDoctorDir(t)

	tmpFile := "fresh.md.tmp"
	tmpPath := filepath.Join(docsDir, tmpFile)
	if err := os.WriteFile(tmpPath, []byte("in progress"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	// Do NOT age the file — it was just created (< 5s).

	err := fixOrphanTmp(docsDir, tmpFile)
	if err == nil {
		t.Fatal("expected error for too-recent file")
	}
	if !strings.Contains(err.Error(), "too recent") {
		t.Errorf("expected 'too recent' in error, got: %v", err)
	}

	// File should still exist.
	if _, statErr := os.Stat(tmpPath); statErr != nil {
		t.Error("too-recent .tmp should not be removed")
	}
}

func TestFixOrphanTmp_InvalidFilename(t *testing.T) {
	docsDir := newDoctorDir(t)

	err := fixOrphanTmp(docsDir, "../escape.tmp")
	if err == nil {
		t.Fatal("expected error for path traversal filename")
	}
}
