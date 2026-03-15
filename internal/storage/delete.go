package storage

import (
	"fmt"
	"os"
	"path/filepath"
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

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("storage: delete %s: %w", filename, err)
	}

	if err := RegenerateIndex(docsDir); err != nil {
		return fmt.Errorf("storage: delete %s: index: %w", filename, err)
	}

	return nil
}
