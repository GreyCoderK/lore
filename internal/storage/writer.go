// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/fileutil"
)

// WriteResult contains the outcome of a WriteDoc operation.
type WriteResult struct {
	Filename string // e.g. "decision-auth-strategy-2026-03-07.md"
	Path     string // e.g. "/path/to/.lore/docs/decision-auth-strategy-2026-03-07.md"
	IndexErr error  // non-nil if index regeneration failed (non-fatal)
}

// AtomicWrite writes data to path via a temp file + rename for crash safety.
// Sets explicit 0644 permissions on the resulting file. Overwrites if exists.
func AtomicWrite(path string, data []byte) error {
	return fileutil.AtomicWrite(path, data, 0644)
}

// AtomicWriteExclusive writes data to path via a temp file + hard link.
// Unlike AtomicWrite, it fails if path already exists (returns an error
// where os.IsExist reports true). This avoids the TOCTOU race inherent
// in Stat-then-Rename patterns.
func AtomicWriteExclusive(path string, data []byte) error {
	return fileutil.AtomicWriteExclusive(path, data, 0644)
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
	if err := validateFilename(filename); err != nil {
		return WriteResult{}, fmt.Errorf("storage: write doc: %w", err)
	}
	path := filepath.Join(dir, filename)

	data, err := Marshal(meta, body)
	if err != nil {
		return WriteResult{}, fmt.Errorf("storage: marshal %s: %w", filename, err)
	}

	// M3 fix: AtomicWriteExclusive uses os.Link which fails atomically if
	// path exists — eliminates the TOCTOU race of the old os.Stat + AtomicWrite.
	if err := AtomicWriteExclusive(path, data); err != nil {
		if os.IsExist(err) {
			return WriteResult{}, fmt.Errorf("storage: write %s: file already exists", filename)
		}
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
