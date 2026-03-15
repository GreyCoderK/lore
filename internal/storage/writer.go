package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/museigen/lore/internal/domain"
)

// WriteResult contains the outcome of a WriteDoc operation.
type WriteResult struct {
	Filename string // e.g. "decision-auth-strategy-2026-03-07.md"
	Path     string // e.g. "/path/to/.lore/docs/decision-auth-strategy-2026-03-07.md"
	IndexErr error  // non-nil if index regeneration failed (non-fatal)
}

// AtomicWrite writes data to path via a temp file + rename for crash safety.
// Sets explicit 0644 permissions on the resulting file.
// Note: uses os.CreateTemp (randomized name) rather than path+".tmp" for
// collision safety. Orphaned temp files use the pattern ".lore-*.tmp".
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

// WriteDoc creates a document in the given directory via AtomicWrite.
// subject is used for the filename slug (e.g., "add JWT middleware" → "add-jwt-middleware").
// After writing, it regenerates the README index. Index errors are surfaced
// in WriteResult.IndexErr but do not cause WriteDoc itself to fail.
func WriteDoc(dir string, meta domain.DocMeta, subject string, body string) (WriteResult, error) {
	if err := ValidateMeta(meta); err != nil {
		return WriteResult{}, err
	}

	// Ensure the target directory exists before generating the filename.
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return WriteResult{}, fmt.Errorf("storage: write doc: mkdir: %w", err)
	}

	slug := slugify(subject)
	if slug == "" {
		slug = "untitled"
	}

	filename := fmt.Sprintf("%s-%s-%s.md", meta.Type, slug, meta.Date)
	path := filepath.Join(dir, filename)

	if _, err := os.Stat(path); err == nil {
		return WriteResult{}, fmt.Errorf("storage: write %s: file already exists", filename)
	}

	data, err := Marshal(meta, body)
	if err != nil {
		return WriteResult{}, fmt.Errorf("storage: marshal %s: %w", filename, err)
	}

	if err := AtomicWrite(path, data); err != nil {
		return WriteResult{}, fmt.Errorf("storage: write %s: %w", filename, err)
	}

	indexErr := RegenerateIndex(dir)

	return WriteResult{Filename: filename, Path: path, IndexErr: indexErr}, nil
}

// accentMap maps common accented characters to ASCII equivalents.
var accentMap = map[rune]string{
	'à': "a", 'á': "a", 'â': "a", 'ã': "a", 'ä': "a", 'å': "a",
	'è': "e", 'é': "e", 'ê': "e", 'ë': "e",
	'ì': "i", 'í': "i", 'î': "i", 'ï': "i",
	'ò': "o", 'ó': "o", 'ô': "o", 'õ': "o", 'ö': "o",
	'ù': "u", 'ú': "u", 'û': "u", 'ü': "u",
	'ñ': "n", 'ç': "c", 'ß': "ss",
	'À': "a", 'Á': "a", 'Â': "a", 'Ã': "a", 'Ä': "a", 'Å': "a",
	'È': "e", 'É': "e", 'Ê': "e", 'Ë': "e",
	'Ì': "i", 'Í': "i", 'Î': "i", 'Ï': "i",
	'Ò': "o", 'Ó': "o", 'Ô': "o", 'Õ': "o", 'Ö': "o",
	'Ù': "u", 'Ú': "u", 'Û': "u", 'Ü': "u",
	'Ñ': "n", 'Ç': "c",
}

var slugNonAlpha = regexp.MustCompile(`[^a-z0-9]+`)
var slugMultiDash = regexp.MustCompile(`-{2,}`)

// slugify converts a string to a URL-friendly slug.
// Lowercase, replaces accents with ASCII, removes non-alphanumeric,
// deduplicates dashes, trims dashes, and truncates to 50 chars max.
func slugify(s string) string {
	// Replace accents
	var b strings.Builder
	for _, r := range s {
		if repl, ok := accentMap[r]; ok {
			b.WriteString(repl)
		} else {
			b.WriteRune(r)
		}
	}
	s = b.String()

	s = strings.ToLower(s)
	s = slugNonAlpha.ReplaceAllString(s, "-")
	s = slugMultiDash.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")

	if len(s) > 50 {
		s = s[:50]
		s = strings.TrimRight(s, "-")
	}

	return s
}
