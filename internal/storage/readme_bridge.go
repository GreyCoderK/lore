// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/greycoderk/lore/internal/fileutil"
	"github.com/greycoderk/lore/internal/i18n"
)

// GenerateReadmeBridge creates .lore/README.md as a discovery bridge.
// Skips silently if the file already exists (user may have edited it).
func GenerateReadmeBridge(loreDir string) error {
	path := filepath.Join(loreDir, "README.md")

	// Anti-overwrite: skip if file already exists.
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	t := i18n.T().UI
	content := fmt.Sprintf("# %s\n\n%s\n\n%s\n\n%s\n\n---\n\n*%s*\n",
		t.ReadmeBridgeTitle,
		t.ReadmeBridgeIntro,
		t.ReadmeBridgeDesc,
		t.ReadmeBridgeLink,
		t.ReadmeBridgeGenNote,
	)

	return fileutil.AtomicWrite(path, []byte(content), 0o644)
}
