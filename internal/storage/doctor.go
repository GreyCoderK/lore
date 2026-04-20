// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Issue represents a single diagnostic finding.
type Issue struct {
	Category string // "orphan-tmp", "broken-ref", "stale-index", "stale-cache", "invalid-frontmatter"
	// Subkind narrows an issue to a precise sub-category. Currently
	// only populated for Category == "invalid-frontmatter":
	//   - "missing"   — no "---\n" delimiter; a synthesized FM is safe
	//   - "malformed" — "---" present but the YAML between delimiters
	//                   is unparseable; auto-fix would destroy
	//                   potentially recoverable authentic content
	// Story 8-22 / invariant I31: `doctor --fix` never rewrites an
	// FM when the source contains a "---" delimiter — the malformed
	// path surfaces a restore hint rather than synthesizing.
	Subkind string
	File    string // file concerned
	Detail  string // human-readable description
	AutoFix bool   // repairable automatically?
}

// Invalid-frontmatter subkinds. Kept as constants so callers can
// pattern-match on stable strings.
const (
	SubkindFrontmatterMissing   = "missing"
	SubkindFrontmatterMalformed = "malformed"
)

// DiagnosticReport holds the results of a full corpus health check.
type DiagnosticReport struct {
	Issues   []Issue
	DocCount int // number of valid .md documents scanned
	Checked  int // total number of checks performed
}

// FixReport holds the results of automatic repairs.
type FixReport struct {
	Fixed     int      // number of issues successfully fixed
	Remaining int      // number of issues that still need manual attention
	Errors    int      // number of errors during fix (permission, etc.)
	Details   []string // descriptions of actions taken
}

// Diagnose performs a comprehensive health check on the docs directory.
// It checks for orphan .tmp files, broken references, stale index,
// stale cache (metadata.json), and invalid front matter.
func Diagnose(docsDir string) (*DiagnosticReport, error) {
	report := &DiagnosticReport{}

	entries, err := os.ReadDir(docsDir)
	if err != nil {
		return nil, fmt.Errorf("storage: doctor: read dir: %w", err)
	}

	// Single pass over entries: collect .tmp files and parse .md files directly.
	// This avoids calling scanDocs separately (no double I/O) and tracks parse
	// failures with their filenames instead of parsing error strings.
	var docs []parsedDoc

	report.Checked++ // Check 1: orphan .tmp
	report.Checked++ // Check 2: invalid front matter
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			continue
		}

		// Check 1: orphan .tmp files
		if strings.HasSuffix(name, ".tmp") {
			report.Issues = append(report.Issues, Issue{
				Category: "orphan-tmp",
				File:     name,
				Detail:   "orphan temporary file (interrupted write)",
				AutoFix:  true,
			})
			continue
		}

		// Parse .md files (skip README.md)
		if !strings.HasSuffix(name, ".md") || name == "README.md" {
			continue
		}

		data, readErr := os.ReadFile(filepath.Join(docsDir, name))
		if readErr != nil {
			report.Issues = append(report.Issues, Issue{
				Category: "invalid-frontmatter",
				Subkind:  SubkindFrontmatterMalformed,
				File:     name,
				Detail:   fmt.Sprintf("cannot read file: %v", readErr),
				AutoFix:  false,
			})
			continue
		}

		// Story 8-22 / I31: classify via the typed sentinels produced
		// by ExtractFrontmatter — they are the single source of truth
		// for "is the delimiter present?". Using a string heuristic
		// here instead would diverge on CRLF-terminated delimiters and
		// BOM-prefixed files (both were misclassified as missing by
		// the previous `strings.HasPrefix(content, "---\n")` probe).
		_, _, extractErr := ExtractFrontmatter(data)
		if extractErr != nil {
			issue := Issue{
				Category: "invalid-frontmatter",
				File:     name,
				Detail:   fmt.Sprintf("YAML parse error: %v", extractErr),
			}
			switch {
			case errors.Is(extractErr, ErrFrontmatterMissing):
				issue.Subkind = SubkindFrontmatterMissing
				issue.AutoFix = true
			case errors.Is(extractErr, ErrFrontmatterMalformed):
				issue.Subkind = SubkindFrontmatterMalformed
				issue.AutoFix = false
			default:
				// Oversized / other I/O-like errors — treat conservatively:
				// a non-sentinel error means we cannot be sure whether a
				// delimiter exists, so refuse auto-fix.
				issue.Subkind = SubkindFrontmatterMalformed
				issue.AutoFix = false
			}
			report.Issues = append(report.Issues, issue)
			continue
		}

		// FM bytes parse cleanly — run full validation (ValidateMeta)
		// to surface semantic errors (unknown type, bad date). These
		// are classified malformed (auto-fix would destroy valid YAML).
		meta, body, parseErr := Unmarshal(data)
		if parseErr != nil {
			report.Issues = append(report.Issues, Issue{
				Category: "invalid-frontmatter",
				Subkind:  SubkindFrontmatterMalformed,
				File:     name,
				Detail:   fmt.Sprintf("YAML parse error: %v", parseErr),
				AutoFix:  false,
			})
			continue
		}

		meta.Filename = name
		docs = append(docs, parsedDoc{Name: name, Meta: meta, Body: body})
	}
	report.DocCount = len(docs)

	// --- Check 3: broken references ---
	report.Checked++
	existingDocs := make(map[string]bool)
	for _, d := range docs {
		// Store without .md extension for reference matching
		existingDocs[strings.TrimSuffix(d.Name, ".md")] = true
	}
	for _, d := range docs {
		for _, ref := range d.Meta.Related {
			ref = strings.TrimSuffix(ref, ".md") // normalize
			if !existingDocs[ref] {
				report.Issues = append(report.Issues, Issue{
					Category: "broken-ref",
					File:     d.Name,
					Detail:   fmt.Sprintf("related reference %q not found — file deleted?", ref),
					AutoFix:  true, // fix = remove the dead reference from related[]
				})
			}
		}
	}

	// --- Check 4: stale README.md index ---
	report.Checked++
	if len(docs) > 0 && isIndexStale(docsDir, docs) {
		report.Issues = append(report.Issues, Issue{
			Category: "stale-index",
			File:     "README.md",
			Detail:   "index out of sync with documents",
			AutoFix:  true,
		})
	}

	// --- Check 5: stale metadata.json cache (OPTIONAL — C7) ---
	report.Checked++
	metadataPath := filepath.Join(docsDir, "metadata.json")
	if _, statErr := os.Stat(metadataPath); statErr == nil {
		report.Issues = append(report.Issues, Issue{
			Category: "stale-cache",
			File:     "metadata.json",
			Detail:   "metadata.json exists but cache system not yet active — consider removing",
			AutoFix:  true,
		})
	}

	return report, nil
}

// Fix applies automatic repairs based on the diagnostic report.
// Only issues with AutoFix=true are attempted.
// Individual fix failures are logged but do not stop other repairs.
func Fix(docsDir string, report *DiagnosticReport) (*FixReport, error) {
	fix := &FixReport{}

	for _, issue := range report.Issues {
		if !issue.AutoFix {
			fix.Remaining++
			continue
		}

		switch issue.Category {
		case "orphan-tmp":
			if err := fixOrphanTmp(docsDir, issue.File); err != nil {
				fix.Errors++
				fix.Details = append(fix.Details, fmt.Sprintf("error removing %s: %v", issue.File, err))
			} else {
				fix.Fixed++
				fix.Details = append(fix.Details, fmt.Sprintf("removed orphan %s", issue.File))
			}

		case "broken-ref":
			if err := fixBrokenRef(docsDir, issue.File, issue.Detail); err != nil {
				fix.Errors++
				fix.Details = append(fix.Details, fmt.Sprintf("error fixing ref in %s: %v", issue.File, err))
			} else {
				fix.Fixed++
				fix.Details = append(fix.Details, fmt.Sprintf("removed dead reference from %s", issue.File))
			}

		case "stale-index":
			if err := RegenerateIndex(docsDir); err != nil {
				fix.Errors++
				fix.Details = append(fix.Details, fmt.Sprintf("error regenerating index: %v", err))
			} else {
				fix.Fixed++
				fix.Details = append(fix.Details, "regenerated index README.md")
			}

		case "invalid-frontmatter":
			// Story 8-22 / I31: belt-and-suspenders guard. Only a
			// truly-missing frontmatter is safe to synthesize. A
			// malformed FM (delimiters present, YAML broken) must
			// never be overwritten — the caller can still have
			// partially-readable metadata we would otherwise destroy.
			// Issues with AutoFix=true and a non-missing Subkind are
			// treated as if AutoFix were false.
			if issue.Subkind != SubkindFrontmatterMissing {
				fix.Remaining++
				continue
			}
			if err := fixMissingFrontMatter(docsDir, issue.File); err != nil {
				fix.Errors++
				fix.Details = append(fix.Details, fmt.Sprintf("error fixing frontmatter %s: %v", issue.File, err))
			} else {
				fix.Fixed++
				fix.Details = append(fix.Details, fmt.Sprintf("added front matter to %s", issue.File))
			}

		case "stale-cache":
			if err := validateFilename(issue.File); err != nil {
				fix.Errors++
				fix.Details = append(fix.Details, fmt.Sprintf("invalid filename %s: %v", issue.File, err))
				continue
			}
			path := filepath.Join(docsDir, issue.File)
			if err := validateResolvedPath(docsDir, path); err != nil {
				fix.Errors++
				fix.Details = append(fix.Details, fmt.Sprintf("path validation failed %s: %v", issue.File, err))
				continue
			}
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				fix.Errors++
				fix.Details = append(fix.Details, fmt.Sprintf("error removing %s: %v", issue.File, err))
			} else {
				fix.Fixed++
				fix.Details = append(fix.Details, fmt.Sprintf("removed stale %s", issue.File))
			}
		}
	}

	return fix, nil
}

// fixOrphanTmp removes a .tmp file from docsDir after validating the path
// and checking the file is old enough (>5s) to avoid racing with concurrent writes.
func fixOrphanTmp(docsDir, filename string) error {
	if err := validateFilename(filename); err != nil {
		return fmt.Errorf("storage: doctor: %w", err)
	}

	fullPath := filepath.Join(docsDir, filename)

	if err := validateResolvedPath(docsDir, fullPath); err != nil {
		return fmt.Errorf("storage: doctor: %w", err)
	}

	// Check age — skip files younger than 5 seconds to avoid racing
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // already gone
		}
		return fmt.Errorf("storage: doctor: stat %s: %w", filename, err)
	}
	if time.Since(info.ModTime()) < 5*time.Second {
		return fmt.Errorf("storage: doctor: %s too recent (may be in use)", filename)
	}

	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("storage: doctor: remove %s: %w", filename, err)
	}
	return nil
}

// fixMissingFrontMatter prepends a default YAML front matter block to a document
// that is missing one. Infers type from the filename slug (e.g. "decision-..." → decision).
// The original content is preserved after the front matter.
func fixMissingFrontMatter(docsDir, filename string) error {
	if err := validateFilename(filename); err != nil {
		return fmt.Errorf("storage: doctor: %w", err)
	}
	fullPath := filepath.Join(docsDir, filename)
	if err := validateResolvedPath(docsDir, fullPath); err != nil {
		return fmt.Errorf("storage: doctor: %w", err)
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("storage: doctor: read %s: %w", filename, err)
	}

	// Belt-and-suspenders TOCTOU guard: if the file gained a delimiter
	// (BOM-prefixed, CRLF-terminated, or plain LF) between Diagnose and
	// Fix, refuse to prepend. Uses the same sentinel classification as
	// Diagnose so the two paths agree on "is there a delimiter?".
	if _, _, err := ExtractFrontmatter(data); err == nil || errors.Is(err, ErrFrontmatterMalformed) {
		return nil
	}

	// Infer type from filename: "decision-auth-2026-04-07.md" → "decision"
	docType := "note"
	base := strings.TrimSuffix(filename, ".md")
	for _, t := range []string{"decision", "feature", "bugfix", "refactor", "release", "summary"} {
		if strings.HasPrefix(base, t+"-") || base == t {
			docType = t
			break
		}
	}

	today := time.Now().Format("2006-01-02")
	frontMatter := fmt.Sprintf("---\ntype: %s\ndate: \"%s\"\nstatus: draft\ngenerated_by: doctor-fix\n---\n", docType, today)

	merged := append([]byte(frontMatter), data...)
	if err := os.WriteFile(fullPath, merged, 0644); err != nil {
		return fmt.Errorf("storage: doctor: write %s: %w", filename, err)
	}
	return nil
}

// fixBrokenRef removes a dead reference from the related[] field of a document.
// The detail string from the issue contains the quoted reference name. This
// function re-reads the file, filters the broken ref, and rewrites via Marshal.
func fixBrokenRef(docsDir, filename, detail string) error {
	if err := validateFilename(filename); err != nil {
		return fmt.Errorf("storage: doctor: %w", err)
	}
	fullPath := filepath.Join(docsDir, filename)
	if err := validateResolvedPath(docsDir, fullPath); err != nil {
		return fmt.Errorf("storage: doctor: %w", err)
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("storage: doctor: read %s: %w", filename, err)
	}

	meta, body, err := Unmarshal(data)
	if err != nil {
		// Try permissive if strict fails
		meta, body, err = UnmarshalPermissive(data)
		if err != nil {
			return fmt.Errorf("storage: doctor: parse %s: %w", filename, err)
		}
	}

	// Extract the broken ref name from the detail message.
	// Detail format: `related reference "some-ref" not found — file deleted?`
	brokenRef := ""
	if start := strings.Index(detail, `"`); start >= 0 {
		rest := detail[start+1:]
		if end := strings.Index(rest, `"`); end >= 0 {
			brokenRef = rest[:end]
		}
	}
	if brokenRef == "" {
		return fmt.Errorf("storage: doctor: cannot extract ref name from %q", detail)
	}

	// Filter out the broken ref (match with and without .md)
	var cleaned []string
	for _, ref := range meta.Related {
		norm := strings.TrimSuffix(ref, ".md")
		if norm == brokenRef || ref == brokenRef {
			continue
		}
		cleaned = append(cleaned, ref)
	}
	meta.Related = cleaned

	out, err := Marshal(meta, body)
	if err != nil {
		return fmt.Errorf("storage: doctor: marshal %s: %w", filename, err)
	}
	return os.WriteFile(fullPath, out, 0644)
}

// isIndexStale checks whether the README.md content matches the current docs.
// Uses exact link matching: each doc must appear as a markdown link [name](name)
// in the index, and no stale links to removed docs should remain.
func isIndexStale(docsDir string, docs []parsedDoc) bool {
	readmePath := filepath.Join(docsDir, "README.md")
	current, err := os.ReadFile(readmePath)
	if err != nil {
		// Missing README.md = stale (needs regeneration)
		return true
	}

	content := string(current)

	// Each doc must appear as an exact markdown link reference: (filename.md)
	for _, d := range docs {
		// Look for exact link target pattern: (filename.md)
		if !strings.Contains(content, "("+d.Name+")") {
			return true
		}
	}

	// Check for stale links: extract all .md filenames from link targets
	existingDocs := make(map[string]bool)
	for _, d := range docs {
		existingDocs[d.Name] = true
	}
	for _, line := range strings.Split(content, "\n") {
		if !strings.HasPrefix(line, "|") {
			continue
		}
		// Extract filenames from markdown link targets: ](filename.md)
		for {
			idx := strings.Index(line, "](")
			if idx < 0 {
				break
			}
			line = line[idx+2:]
			closeIdx := strings.Index(line, ")")
			if closeIdx < 0 {
				break
			}
			filename := line[:closeIdx]
			line = line[closeIdx+1:]
			if strings.HasSuffix(filename, ".md") && filename != "README.md" && !existingDocs[filename] {
				return true
			}
		}
	}

	return false
}

