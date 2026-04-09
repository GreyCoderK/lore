// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package brand

import (
	"os"
	"strings"
	"testing"
)

func TestLogoPNG_Embedded(t *testing.T) {
	if len(logoPNG) == 0 {
		t.Fatal("embedded logoPNG is empty")
	}
	// PNG magic bytes: 0x89 P N G
	if logoPNG[0] != 0x89 || logoPNG[1] != 'P' || logoPNG[2] != 'N' || logoPNG[3] != 'G' {
		t.Fatalf("logoPNG does not start with PNG magic bytes, got %x", logoPNG[:4])
	}
}

func TestLogoPNGPath_ReturnsValidFile(t *testing.T) {
	path := LogoPNGPath()
	if path == "" {
		t.Fatal("LogoPNGPath() returned empty string")
	}
	if !strings.HasSuffix(path, ".png") {
		t.Errorf("path %q does not end with .png", path)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %q: %v", path, err)
	}
	if info.Size() != int64(len(logoPNG)) {
		t.Errorf("file size %d != embedded size %d", info.Size(), len(logoPNG))
	}
}

func TestLogoPNGPath_Idempotent(t *testing.T) {
	p1 := LogoPNGPath()
	p2 := LogoPNGPath()
	if p1 != p2 {
		t.Errorf("LogoPNGPath() not idempotent: %q != %q", p1, p2)
	}
}

func TestLogoPNGPath_ContainsLorePrefix(t *testing.T) {
	path := LogoPNGPath()
	if !strings.Contains(path, "lore-logo-") {
		t.Errorf("path %q does not contain 'lore-logo-' prefix", path)
	}
}
