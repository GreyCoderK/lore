package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/greycoderk/lore/internal/domain"
)

// compile-time check
var _ domain.CorpusReader = (*CorpusStore)(nil)

// CorpusStore implements domain.CorpusReader for the local filesystem.
type CorpusStore struct {
	Dir string // path to .lore/docs/
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
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("storage: list docs: %w", err)
	}

	var results []domain.DocMeta
	var parseErrs []error
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".md") || name == "README.md" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(s.Dir, name))
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
		if matchesFilter(meta, name, body, filter) {
			results = append(results, meta)
		}
	}

	return results, errors.Join(parseErrs...)
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
