// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DeleteDoc removes a document from docsDir and regenerates the README index.
// Returns an error if the file does not exist or cannot be removed.
// Index regeneration errors are returned separately (wrapped with context).
func DeleteDoc(docsDir, filename string) error {
	if err := validateFilename(filename); err != nil {
		return fmt.Errorf("storage: delete %s: %w", filename, err)
	}
	// Protect the auto-generated index from accidental deletion.
	if filename == "README.md" {
		return fmt.Errorf("storage: delete README.md: protected file (auto-generated index)")
	}

	path := filepath.Join(docsDir, filename)

	if err := validateResolvedPath(docsDir, path); err != nil {
		return fmt.Errorf("storage: delete %s: %w", filename, err)
	}

	// Guard against directories — only regular files should be deleted.
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("storage: delete %s: %w", filename, err)
	}
	if info.IsDir() {
		return fmt.Errorf("storage: delete %s: is a directory, not a document", filename)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("storage: delete %s: %w", filename, err)
	}

	if err := RegenerateIndex(docsDir); err != nil {
		return fmt.Errorf("storage: delete %s: index: %w", filename, err)
	}

	return nil
}

// FindReferencingDocs returns filenames of documents whose `related:` field
// references the given filename (compared without .md extension).
func FindReferencingDocs(docsDir, filename string) ([]string, error) {
	target := strings.TrimSuffix(filename, ".md")

	docs, _, fatalErr := scanDocs(docsDir)
	if fatalErr != nil {
		return nil, fmt.Errorf("storage: find referencing: %w", fatalErr)
	}

	var refs []string
	for _, d := range docs {
		if d.Name == filename {
			continue // skip self
		}
		for _, rel := range d.Meta.Related {
			if rel == target {
				refs = append(refs, d.Name)
				break
			}
		}
	}
	return refs, nil
}
