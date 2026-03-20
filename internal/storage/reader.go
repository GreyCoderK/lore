// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/greycoderk/lore/internal/domain"
)

// SearchResult represents a single search hit from SearchDocs.
type SearchResult struct {
	Filename string
	Path     string
	Meta     domain.DocMeta
	Title    string // first # heading from body, fallback: slug from filename
}

// compile-time check
var _ domain.CorpusReader = (*CorpusStore)(nil)

// CorpusStore implements domain.CorpusReader for the local filesystem.
type CorpusStore struct {
	Dir string // path to .lore/docs/
}

// validateResolvedPath checks that fullPath, after symlink resolution, is still
// inside dir. Both dir and fullPath are resolved via filepath.EvalSymlinks so
// that a symlink in either location is accounted for.
func validateResolvedPath(dir, fullPath string) error {
	resolvedDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return fmt.Errorf("storage: resolve dir: %w", err)
	}
	resolvedPath, err := filepath.EvalSymlinks(fullPath)
	if err != nil {
		return fmt.Errorf("storage: resolve path: %w", err)
	}
	// Ensure the resolved path lives under the resolved dir.
	if !strings.HasPrefix(resolvedPath, resolvedDir+string(filepath.Separator)) {
		return fmt.Errorf("storage: path %s escapes docs directory", fullPath)
	}
	return nil
}

// ValidateFilename rejects path traversal attempts in user-supplied filenames.
func ValidateFilename(filename string) error {
	return validateFilename(filename)
}

// validateFilename rejects path traversal attempts in user-supplied filenames.
func validateFilename(filename string) error {
	if filename == "" {
		return fmt.Errorf("storage: filename is empty")
	}
	if filepath.IsAbs(filename) {
		return fmt.Errorf("storage: filename must be relative: %s", filename)
	}
	if strings.Contains(filename, "..") {
		return fmt.Errorf("storage: filename must not contain '..': %s", filename)
	}
	if strings.ContainsAny(filename, `/\`) {
		return fmt.Errorf("storage: filename must not contain path separators: %s", filename)
	}
	return nil
}

// ReadDoc reads a single document by ID (filename without extension).
func (s *CorpusStore) ReadDoc(id string) (string, error) {
	// Try with .md extension if not provided
	filename := id
	if !strings.HasSuffix(filename, ".md") {
		filename += ".md"
	}

	if err := validateFilename(filename); err != nil {
		return "", fmt.Errorf("storage: read doc %s: %w", id, err)
	}

	path := filepath.Join(s.Dir, filename)

	if err := validateResolvedPath(s.Dir, path); err != nil {
		return "", fmt.Errorf("storage: read doc %s: %w", id, err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("storage: read doc %s: %w", id, domain.ErrNotFound)
		}
		return "", fmt.Errorf("storage: read doc %s: %w", id, err)
	}

	return string(data), nil
}

// ListDocs scans the directory for .md files and returns their metadata.
// Skips README.md (auto-generated index). Returns partial results alongside
// a combined error if any files could not be read or parsed.
func (s *CorpusStore) ListDocs(filter domain.DocFilter) ([]domain.DocMeta, error) {
	docs, parseErr, fatalErr := scanDocs(s.Dir)
	if fatalErr != nil {
		return nil, fmt.Errorf("storage: list docs: %w", fatalErr)
	}

	var results []domain.DocMeta
	for _, d := range docs {
		if matchesFilter(d.Meta, d.Name, d.Body, filter) {
			results = append(results, d.Meta)
		}
	}

	return results, parseErr
}

func matchesFilter(meta domain.DocMeta, filename string, body string, filter domain.DocFilter) bool {
	if filter.Type != "" && meta.Type != filter.Type {
		return false
	}
	if filter.Status != "" && meta.Status != filter.Status {
		return false
	}
	if filter.After != "" && meta.Date < filter.After {
		return false
	}
	if filter.Before != "" && meta.Date > filter.Before {
		return false
	}
	if len(filter.Tags) > 0 {
		for _, tag := range filter.Tags {
			if !containsString(meta.Tags, tag) {
				return false
			}
		}
	}
	if filter.Text != "" {
		search := strings.ToLower(filter.Text)
		if !strings.Contains(strings.ToLower(body), search) &&
			!strings.Contains(strings.ToLower(filename), search) {
			return false
		}
	}
	return true
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// parsedDoc holds the result of scanning a single .md file from the docs directory.
type parsedDoc struct {
	Name string
	Meta domain.DocMeta
	Body string
}

// scanDocs reads all .md files (excluding README.md) from dir, parses front matter,
// and returns parsed documents alongside any non-fatal parse errors.
func scanDocs(dir string) ([]parsedDoc, error, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("storage: scan: %w", err)
	}

	var docs []parsedDoc
	var parseErrs []error

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".md") || name == "README.md" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			parseErrs = append(parseErrs, fmt.Errorf("storage: read %s: %w", name, err))
			continue
		}

		meta, body, err := Unmarshal(data)
		if err != nil {
			parseErrs = append(parseErrs, fmt.Errorf("storage: parse %s: %w", name, err))
			continue
		}

		meta.Filename = name
		docs = append(docs, parsedDoc{Name: name, Meta: meta, Body: body})
	}

	return docs, errors.Join(parseErrs...), nil
}

// NormalizeAfter converts a YYYY-MM date to YYYY-MM-01 for lexicographic comparison.
// YYYY-MM-DD values pass through unchanged.
func NormalizeAfter(after string) string {
	if len(after) == 7 && after[4] == '-' {
		return after + "-01"
	}
	return after
}

// SearchDocs searches documents matching keyword (case-insensitive) in filename, tags, and body,
// then applies type and date filters from DocFilter. Returns results sorted by date descending.
func SearchDocs(dir string, keyword string, filter domain.DocFilter) ([]SearchResult, error) {
	docs, parseErr, fatalErr := scanDocs(dir)
	if fatalErr != nil {
		return nil, fmt.Errorf("storage: search: %w", fatalErr)
	}

	// Normalize After filter for YYYY-MM support
	filter.After = NormalizeAfter(filter.After)

	search := strings.ToLower(keyword)
	var results []SearchResult

	for _, d := range docs {
		// Apply type and date filters via matchesFilter (keyword handled separately)
		filterNoText := domain.DocFilter{
			Type:  filter.Type,
			After: filter.After,
		}
		if !matchesFilter(d.Meta, d.Name, d.Body, filterNoText) {
			continue
		}

		// Keyword match: filename, tags, or body (case-insensitive)
		if search != "" {
			nameLower := strings.ToLower(d.Name)
			bodyLower := strings.ToLower(d.Body)
			tagsLower := strings.ToLower(strings.Join(d.Meta.Tags, " "))

			if !strings.Contains(nameLower, search) &&
				!strings.Contains(bodyLower, search) &&
				!strings.Contains(tagsLower, search) {
				continue
			}
		}

		results = append(results, SearchResult{
			Filename: d.Name,
			Path:     filepath.Join(dir, d.Name),
			Meta:     d.Meta,
			Title:    extractTitle(d.Body, d.Name),
		})
	}

	// Sort by date descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Meta.Date > results[j].Meta.Date
	})

	return results, parseErr
}

// FindDocByCommit searches for a document whose front matter commit field matches
// commitHash (strict equality — caller resolves short hashes before calling).
// Returns nil if no match is found.
func FindDocByCommit(dir string, commitHash string) (*SearchResult, error) {
	docs, parseErr, fatalErr := scanDocs(dir)
	if fatalErr != nil {
		return nil, fmt.Errorf("storage: find by commit: %w", fatalErr)
	}
	for _, d := range docs {
		if d.Meta.Commit == commitHash {
			return &SearchResult{
				Filename: d.Name,
				Path:     filepath.Join(dir, d.Name),
				Meta:     d.Meta,
				Title:    extractTitle(d.Body, d.Name),
			}, parseErr
		}
	}
	return nil, parseErr
}

// ReadDocContent returns the full file content (front matter + body) for display.
// The path argument is expected to come from SearchResult.Path (produced by SearchDocs),
// which constructs paths via filepath.Join(dir, entry.Name()) — never from user input.
func ReadDocContent(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("storage: read content: %w", err)
	}
	return string(data), nil
}

// extractTitle returns the first top-level markdown heading (# ...) from body.
// Falls back to the slug portion of filename (e.g. "auth-strategy" from "decision-auth-strategy-2026-03-07.md")
// when the body has no heading. Only matches # (h1), not ## or deeper.
func extractTitle(body string, filename string) string {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(trimmed[2:])
		}
	}
	return ExtractSlug(filename)
}

// ExtractSlug extracts the slug from a filename with pattern {type}-{slug}-{date}.md.
// Example: "decision-auth-strategy-2026-03-07.md" → "auth-strategy"
func ExtractSlug(filename string) string {
	// Remove .md extension
	name := strings.TrimSuffix(filename, ".md")

	// Remove date suffix (last 10 chars: YYYY-MM-DD)
	if len(name) > 11 && name[len(name)-11] == '-' {
		datePart := name[len(name)-10:]
		// Validate it looks like a date
		if len(datePart) == 10 && datePart[4] == '-' && datePart[7] == '-' {
			name = name[:len(name)-11]
		}
	}

	// Remove type prefix (first segment before first '-')
	idx := strings.Index(name, "-")
	if idx >= 0 && idx < len(name)-1 {
		name = name[idx+1:]
	} else {
		// No slug found (e.g. type-only or type-date filename)
		return "untitled"
	}

	if name == "" {
		return "untitled"
	}

	return name
}
