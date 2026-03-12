package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/museigen/lore/internal/domain"
)

// AtomicWrite writes data to path via a temp file + rename for crash safety.
// Sets explicit 0644 permissions on the resulting file.
func AtomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".lore-*.tmp")
	if err != nil {
		return fmt.Errorf("storage: create temp: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("storage: write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("storage: close temp: %w", err)
	}
	if err := os.Chmod(tmpName, 0644); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("storage: chmod temp: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("storage: rename temp: %w", err)
	}
	return nil
}

// WriteDoc creates a document in the given directory via atomicWrite.
// subject is used for the filename slug (e.g., "add JWT middleware" → "add-jwt-middleware").
// Returns the generated filename.
func WriteDoc(dir string, meta domain.DocMeta, subject string, body string) (string, error) {
	slug := domain.Slugify(subject)
	if slug == "" {
		slug = "untitled"
	}

	filename := fmt.Sprintf("%s-%s-%s.md", meta.Type, slug, meta.Date.Format("2006-01-02"))

	data, err := Marshal(meta, body)
	if err != nil {
		return "", fmt.Errorf("storage: marshal %s: %w", filename, err)
	}

	path := filepath.Join(dir, filename)
	if err := AtomicWrite(path, data); err != nil {
		return "", fmt.Errorf("storage: write %s: %w", filename, err)
	}

	return filename, nil
}
