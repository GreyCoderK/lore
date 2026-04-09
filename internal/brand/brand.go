// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package brand

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"
)

//go:embed logo.png
var logoPNG []byte

var (
	cachedPath string
	once       sync.Once
)

// LogoPNGPath returns an absolute path to the Lore logo PNG on disk.
// The file is written once to os.TempDir() and reused across calls.
// Returns "" if the write fails.
func LogoPNGPath() string {
	once.Do(func() {
		hash := sha256.Sum256(logoPNG)
		name := "lore-logo-" + hex.EncodeToString(hash[:4]) + ".png"
		path := filepath.Join(os.TempDir(), name)

		// Reuse if already on disk from a previous run.
		if info, err := os.Stat(path); err == nil && info.Size() == int64(len(logoPNG)) {
			cachedPath = path
			return
		}

		if err := os.WriteFile(path, logoPNG, 0o644); err != nil {
			return
		}
		cachedPath = path
	})
	return cachedPath
}
