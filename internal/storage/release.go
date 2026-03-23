// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/greycoderk/lore/internal/domain"
)

// ReleaseDoc enriches DocMeta with the document title for release notes display.
type ReleaseDoc struct {
	domain.DocMeta
	Title string // first # heading from body, fallback: slug from filename
}

// CollectReleaseDocuments scans docsDir for documents whose front matter commit
// field matches one of the given commit hashes. Documents are grouped by type
// and sorted by date within each group. Returns matched docs alongside any
// non-fatal parse errors (callers should surface these as warnings).
func CollectReleaseDocuments(docsDir string, commits []string) ([]ReleaseDoc, error, error) {
	if len(commits) == 0 {
		return nil, nil, nil
	}

	commitSet := make(map[string]bool, len(commits))
	for _, c := range commits {
		commitSet[c] = true
	}

	docs, parseErr, fatalErr := scanDocs(docsDir)
	if fatalErr != nil {
		return nil, nil, fmt.Errorf("storage: release: %w", fatalErr)
	}

	var matched []ReleaseDoc
	for _, d := range docs {
		if d.Meta.Commit == "" {
			continue
		}
		rd := ReleaseDoc{DocMeta: d.Meta, Title: extractTitle(d.Body, d.Name)}
		if commitSet[d.Meta.Commit] {
			matched = append(matched, rd)
			continue
		}
		// Handle short SHA prefix matching — require minimum 7 chars to avoid
		// false positives (e.g. commit: "a" matching 6% of all commits).
		if len(d.Meta.Commit) >= 7 {
			for c := range commitSet {
				if len(c) >= 7 && (strings.HasPrefix(c, d.Meta.Commit) || strings.HasPrefix(d.Meta.Commit, c)) {
					matched = append(matched, rd)
					break
				}
			}
		}
	}

	sort.Slice(matched, func(i, j int) bool {
		if matched[i].Type != matched[j].Type {
			return matched[i].Type < matched[j].Type
		}
		return matched[i].Date < matched[j].Date
	})

	return matched, parseErr, nil
}

// typeToSection maps Lore document types to release notes section headers.
var typeToSection = map[string]string{
	domain.DocTypeFeature:  "Features",
	domain.DocTypeBugfix:   "Bug Fixes",
	domain.DocTypeRefactor: "Refactors",
	domain.DocTypeDecision: "Decisions",
	domain.DocTypeNote:     "Notes",
}

// typeToChangelog maps Lore document types to Keep a Changelog categories.
var typeToChangelog = map[string]string{
	domain.DocTypeFeature:  "Added",
	domain.DocTypeBugfix:   "Fixed",
	domain.DocTypeRefactor: "Changed",
	domain.DocTypeDecision: "Changed",
	domain.DocTypeNote:     "Changed",
}

// GenerateReleaseNotes produces a release notes Markdown document with front matter.
func GenerateReleaseNotes(version string, date string, docs []ReleaseDoc, docsDir string) (string, error) {
	meta := domain.DocMeta{
		Type:   domain.DocTypeRelease,
		Date:   date,
		Status: "published",
	}

	// Group docs by type
	groups := make(map[string][]ReleaseDoc)
	for _, d := range docs {
		groups[d.Type] = append(groups[d.Type], d)
	}

	var body strings.Builder
	fmt.Fprintf(&body, "# Release %s\n\n", version)

	// Render groups in defined order
	orderedTypes := []string{
		domain.DocTypeFeature,
		domain.DocTypeBugfix,
		domain.DocTypeRefactor,
		domain.DocTypeDecision,
		domain.DocTypeNote,
	}

	for _, docType := range orderedTypes {
		group, ok := groups[docType]
		if !ok {
			continue
		}
		section := typeToSection[docType]
		fmt.Fprintf(&body, "## %s\n\n", section)
		for _, d := range group {
			slug := ExtractSlug(d.Filename)
			desc := d.Title
			if desc == "" {
				desc = d.Filename
			}
			fmt.Fprintf(&body, "- **%s** — %s\n", slug, desc)
		}
		body.WriteString("\n")
	}

	data, err := Marshal(meta, body.String())
	if err != nil {
		return "", fmt.Errorf("storage: release: %w", err)
	}

	// Sanitize version to prevent path traversal (e.g. "../../../evil")
	safeVersion := sanitizeVersion(version)
	filename := fmt.Sprintf("release-%s-%s.md", safeVersion, date)
	if err := validateFilename(filename); err != nil {
		return "", fmt.Errorf("storage: release: %w", err)
	}
	path := filepath.Join(docsDir, filename)

	if err := AtomicWriteExclusive(path, data); err != nil {
		if os.IsExist(err) {
			return "", fmt.Errorf("storage: release: file %s already exists", filename)
		}
		return "", fmt.Errorf("storage: release: write %s: %w", filename, err)
	}

	return filename, nil
}

// ReleaseEntry represents one release in releases.json.
type ReleaseEntry struct {
	Version   string   `json:"version"`
	Date      string   `json:"date"`
	Documents []string `json:"documents"`
}

// UpdateReleasesJSON adds a new release entry to .lore/releases.json.
func UpdateReleasesJSON(loreDir string, version string, date string, docFilenames []string) error {
	path := filepath.Join(loreDir, "releases.json")

	var entries []ReleaseEntry
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("storage: release: read releases.json: %w", err)
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &entries); err != nil {
			return fmt.Errorf("storage: release: parse releases.json: %w", err)
		}
	}

	// Check for duplicate version
	for _, e := range entries {
		if e.Version == version {
			return fmt.Errorf("storage: release: release '%s' already exists in releases.json", version)
		}
	}

	entries = append(entries, ReleaseEntry{
		Version:   version,
		Date:      date,
		Documents: docFilenames,
	})

	out, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("storage: release: marshal releases.json: %w", err)
	}
	out = append(out, '\n')

	if err := AtomicWrite(path, out); err != nil {
		return fmt.Errorf("storage: release: write releases.json: %w", err)
	}

	return nil
}

// changelogHeader is the standard Keep a Changelog header.
const changelogHeader = `# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
`

// UpdateChangelog updates or creates CHANGELOG.md with a new release section.
// Returns (headerMissing, error). headerMissing is true when an existing CHANGELOG
// lacks the "# Changelog" header — callers should warn on stderr.
func UpdateChangelog(projectDir string, version string, date string, docs []ReleaseDoc) (bool, error) {
	path := filepath.Join(projectDir, "CHANGELOG.md")

	var existing string
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("storage: release: read CHANGELOG.md: %w", err)
	}
	if len(data) > 0 {
		existing = string(data)
	}

	// Build release section in Keep a Changelog format
	groups := make(map[string][]ReleaseDoc)
	for _, d := range docs {
		cat := typeToChangelog[d.Type]
		if cat == "" {
			cat = "Changed"
		}
		groups[cat] = append(groups[cat], d)
	}

	var section strings.Builder
	fmt.Fprintf(&section, "## [%s] - %s\n", version, date)

	orderedCats := []string{"Added", "Fixed", "Changed"}
	for _, cat := range orderedCats {
		group, ok := groups[cat]
		if !ok {
			continue
		}
		fmt.Fprintf(&section, "\n### %s\n", cat)
		for _, d := range group {
			slug := ExtractSlug(d.Filename)
			fmt.Fprintf(&section, "- %s: %s (%s)\n", capitalize(d.Type), slug, d.Filename)
		}
	}

	var result string
	var headerMissing bool
	if existing == "" {
		// Create new CHANGELOG
		result = changelogHeader + "\n" + section.String()
	} else {
		// Insert after header
		headerEnd, found := findChangelogInsertPoint(existing)
		headerMissing = !found
		result = existing[:headerEnd] + "\n" + section.String() + "\n" + existing[headerEnd:]
	}

	if err := AtomicWrite(path, []byte(result)); err != nil {
		return false, fmt.Errorf("storage: release: write CHANGELOG.md: %w", err)
	}

	return headerMissing, nil
}

// sanitizeVersion strips unsafe characters from version strings to prevent
// path traversal. Allows alphanumeric, dots, dashes, underscores, and plus.
func sanitizeVersion(v string) string {
	var b strings.Builder
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == '.' || r == '-' || r == '_' || r == '+' {
			b.WriteRune(r)
		}
	}
	s := b.String()
	if s == "" {
		return "unversioned"
	}
	return s
}

// capitalize returns s with its first rune uppercased. Safe for multi-byte UTF-8.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// findChangelogInsertPoint finds where to insert new release content.
// Looks for the header line "# Changelog" or "# CHANGELOG", then skips
// description lines until the first "## " section or end of file.
// findChangelogInsertPoint returns (position, headerFound). When headerFound
// is false the caller should warn the user that insertion happens at the top.
func findChangelogInsertPoint(content string) (int, bool) {
	lines := strings.Split(content, "\n")
	headerFound := false
	pos := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !headerFound {
			lower := strings.ToLower(trimmed)
			if lower == "# changelog" {
				headerFound = true
			}
			pos += len(line) + 1
			continue
		}
		// After header, skip description lines until we hit a ## section
		if strings.HasPrefix(trimmed, "## ") {
			// Insert before this section
			return pos, true
		}
		pos += len(line) + 1
	}

	if !headerFound {
		// No header found — insert at top
		return 0, false
	}
	// End of file — ensure we don't exceed content length
	if pos > len(content) {
		pos = len(content)
	}
	return pos, true
}
