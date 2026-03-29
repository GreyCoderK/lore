// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package domain

import "path/filepath"

// Directory components within the .lore structure.
// These are path *segments*, not full paths. Use filepath.Join
// with a workDir or loreDir to construct full paths.
const (
	// LoreDir is the root directory name for Lore data.
	LoreDir = ".lore"

	// DocsDir is the subdirectory name for documentation files.
	DocsDir = "docs"

	// TemplatesDir is the subdirectory name for project-local templates.
	TemplatesDir = "templates"
)

// DocsPath returns the path to the docs directory relative to workDir.
func DocsPath(workDir string) string {
	return filepath.Join(workDir, LoreDir, DocsDir)
}

// NOTE: PendingPath and StorePath helpers are deferred until PendingDir and
// StoreFile constants are defined in this package.
