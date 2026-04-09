// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/greycoderk/lore/internal/domain"
)

// compile-time check
var _ domain.CorpusReader = (*PlainCorpusStore)(nil)

// PlainCorpusStore implements domain.CorpusReader for any directory of Markdown files.
// Unlike CorpusStore, it gracefully handles files without YAML front matter,
// making it suitable for standalone Angela usage on non-lore projects.
type PlainCorpusStore struct {
	Dir string // path to the markdown directory
}

// ReadDoc reads a single document by relative path (e.g., "commands/angela-polish.md").
// Unlike CorpusStore.ReadDoc which only accepts flat filenames, PlainCorpusStore
// supports subdirectory paths for recursive standalone mode.
func (s *PlainCorpusStore) ReadDoc(id string) (string, error) {
	filename := id
	if !strings.HasSuffix(filename, ".md") {
		filename += ".md"
	}

	if filename == "" {
		return "", fmt.Errorf("storage: read doc: filename is empty")
	}
	if filepath.IsAbs(filename) {
		return "", fmt.Errorf("storage: read doc: filename must be relative: %s", filename)
	}
	if strings.Contains(filename, "..") {
		return "", fmt.Errorf("storage: read doc: filename must not contain '..': %s", filename)
	}
	// Block reserved filenames (base name only — subdirs like commands/index.md are fine)
	baseName := strings.ToLower(filepath.Base(filename))
	if baseName == "readme.md" || baseName == ".index.lock" {
		return "", fmt.Errorf("storage: %q is a reserved filename", filename)
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

// ListDocs scans the directory recursively for .md files and returns their metadata.
// Files with valid YAML front matter use the parsed metadata.
// Files without front matter get synthetic metadata derived from the filename and file modification time.
func (s *PlainCorpusStore) ListDocs(filter domain.DocFilter) ([]domain.DocMeta, error) {
	if _, err := os.Stat(s.Dir); os.IsNotExist(err) {
		return nil, nil
	}

	var results []domain.DocMeta

	err := filepath.WalkDir(s.Dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if d.IsDir() {
			return nil // continue into subdirectories
		}

		name := d.Name()
		if !strings.HasSuffix(name, ".md") || name == "README.md" {
			return nil
		}

		// Reject symlinks that escape the docs directory
		if err := validateResolvedPath(s.Dir, path); err != nil {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil // skip unreadable files
		}

		// Use relative path from Dir as the display filename
		relPath, _ := filepath.Rel(s.Dir, path)
		if relPath == "" {
			relPath = name
		}

		meta, _, parseErr := Unmarshal(data)
		if parseErr != nil {
			// No valid front matter — build synthetic metadata from file info
			meta = buildSyntheticMeta(name, d)
		}
		meta.Filename = relPath

		if matchesFilter(meta, relPath, string(data), filter) {
			results = append(results, meta)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("storage: plain list: %w", err)
	}

	return results, nil
}

// BuildPlainMeta creates synthetic metadata from a filename for standalone mode.
// Exported for use in cmd/ when a single file fails front matter parsing.
func BuildPlainMeta(filename string) domain.DocMeta {
	return domain.DocMeta{
		Type:   "note",
		Date:   time.Now().Format("2006-01-02"),
		Status: "published",
		Tags:   inferTagsFromFilename(filename),
	}
}

// buildSyntheticMeta creates metadata for a plain Markdown file without front matter.
func buildSyntheticMeta(name string, entry os.DirEntry) domain.DocMeta {
	date := time.Now().Format("2006-01-02")
	if info, err := entry.Info(); err == nil {
		date = info.ModTime().Format("2006-01-02")
	}

	return domain.DocMeta{
		Type:   "note",
		Date:   date,
		Status: "published",
		Tags:   inferTagsFromFilename(name),
	}
}

// inferTagsFromFilename extracts tags from a filename by splitting on hyphens.
// Example: "api-authentication-guide.md" → ["api", "authentication", "guide"]
func inferTagsFromFilename(name string) []string {
	slug := strings.TrimSuffix(name, ".md")
	parts := strings.Split(slug, "-")

	// Filter out very short parts and date-like segments
	var tags []string
	for _, p := range parts {
		if len(p) < 3 {
			continue
		}
		// Skip date-like segments (YYYY, MM, DD patterns)
		if len(p) == 4 && p[0] >= '0' && p[0] <= '9' {
			continue
		}
		tags = append(tags, strings.ToLower(p))
	}

	return tags
}
