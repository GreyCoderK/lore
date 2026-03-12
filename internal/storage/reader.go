package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/museigen/lore/internal/domain"
)

// compile-time check
var _ domain.CorpusReader = (*CorpusStore)(nil)

// CorpusStore implements domain.CorpusReader for the local filesystem.
type CorpusStore struct {
	Dir string // path to .lore/docs/
}

// ReadDoc reads a single document by ID (filename without extension).
func (s *CorpusStore) ReadDoc(id string) (string, error) {
	// Try with .md extension if not provided
	filename := id
	if !strings.HasSuffix(filename, ".md") {
		filename += ".md"
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
func (s *CorpusStore) ListDocs(filter domain.DocFilter) ([]domain.DocMeta, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("storage: list docs: %w", err)
	}

	var results []domain.DocMeta
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(s.Dir, entry.Name()))
		if err != nil {
			continue
		}

		meta, _, err := Unmarshal(data)
		if err != nil {
			continue
		}

		if matchesFilter(meta, filter) {
			results = append(results, meta)
		}
	}

	return results, nil
}

func matchesFilter(meta domain.DocMeta, filter domain.DocFilter) bool {
	if filter.Type != "" && meta.Type != filter.Type {
		return false
	}
	if filter.Keyword != "" {
		// Simple keyword match against type
		if !strings.Contains(strings.ToLower(meta.Type), strings.ToLower(filter.Keyword)) {
			return false
		}
	}
	if !filter.After.IsZero() && meta.Date.Time.Before(filter.After) {
		return false
	}
	if !filter.Before.IsZero() && meta.Date.Time.After(filter.Before) {
		return false
	}
	return true
}
