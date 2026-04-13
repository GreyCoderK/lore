// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package brand

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func resetOnce() {
	once = sync.Once{}
	cachedPath = ""
}

func expectedLogoPath() string {
	hash := sha256.Sum256(logoPNG)
	name := "lore-logo-" + hex.EncodeToString(hash[:4]) + ".png"
	return filepath.Join(os.TempDir(), name)
}

func TestLogoPNGPath_WriteThenReuse(t *testing.T) {
	logoPath := expectedLogoPath()
	_ = os.Remove(logoPath)

	resetOnce()
	defer resetOnce()

	p1 := LogoPNGPath()
	if p1 == "" {
		t.Fatal("first call (write branch) returned empty path")
	}

	if _, err := os.Stat(p1); err != nil {
		t.Fatalf("written file not found: %v", err)
	}

	resetOnce()

	p2 := LogoPNGPath()
	if p2 == "" {
		t.Fatal("second call (reuse branch) returned empty path")
	}
	if p1 != p2 {
		t.Errorf("reuse branch returned different path: %q vs %q", p1, p2)
	}
}
