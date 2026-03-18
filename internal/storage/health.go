package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// QuickHealthCheck performs a fast health check on the docs directory.
// Returns the number of issues found:
//   - orphan .tmp files (interrupted writes)
//   - missing or empty README.md index
//
// This is NOT the full diagnostic — see lore doctor (Story 5.1).
func QuickHealthCheck(docsDir string) (int, error) {
	entries, err := os.ReadDir(docsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 1, nil // docs dir missing is an issue
		}
		return 0, fmt.Errorf("storage: health check: %w", err)
	}

	issues := 0

	// Count orphan .tmp files
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tmp") {
			issues++
		}
	}

	// Check README.md index exists and is non-empty
	readmePath := filepath.Join(docsDir, "README.md")
	info, err := os.Stat(readmePath)
	if err != nil {
		if os.IsNotExist(err) {
			issues++
		}
		// other stat errors: don't count as issue, don't fail
	} else if info.Size() == 0 {
		issues++
	}

	return issues, nil
}
