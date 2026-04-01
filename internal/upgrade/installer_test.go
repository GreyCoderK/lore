// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package upgrade

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAssetName(t *testing.T) {
	name := AssetName()

	// Must start with "lore_"
	if !strings.HasPrefix(name, "lore_") {
		t.Errorf("AssetName() = %q, expected prefix 'lore_'", name)
	}

	// OS must be title-cased
	switch runtime.GOOS {
	case "darwin":
		if !strings.Contains(name, "Darwin") {
			t.Errorf("AssetName() = %q, expected 'Darwin' on macOS", name)
		}
	case "linux":
		if !strings.Contains(name, "Linux") {
			t.Errorf("AssetName() = %q, expected 'Linux' on Linux", name)
		}
	case "windows":
		if !strings.Contains(name, "Windows") {
			t.Errorf("AssetName() = %q, expected 'Windows' on Windows", name)
		}
	}

	// GOARCH mapping
	switch runtime.GOARCH {
	case "amd64":
		if !strings.Contains(name, "x86_64") {
			t.Errorf("AssetName() = %q, expected 'x86_64' for amd64", name)
		}
	case "arm64":
		if !strings.Contains(name, "arm64") {
			t.Errorf("AssetName() = %q, expected 'arm64'", name)
		}
	}

	// Extension
	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(name, ".zip") {
			t.Errorf("AssetName() = %q, expected .zip on Windows", name)
		}
	} else {
		if !strings.HasSuffix(name, ".tar.gz") {
			t.Errorf("AssetName() = %q, expected .tar.gz", name)
		}
	}
}

func TestVerifySHA256(t *testing.T) {
	tmp := t.TempDir()
	content := []byte("hello lore upgrade")
	path := filepath.Join(tmp, "test.bin")
	os.WriteFile(path, content, 0644)

	h := sha256.Sum256(content)
	expected := hex.EncodeToString(h[:])

	t.Run("matching checksum", func(t *testing.T) {
		if err := VerifySHA256(path, expected); err != nil {
			t.Errorf("VerifySHA256() unexpected error: %v", err)
		}
	})

	t.Run("mismatching checksum", func(t *testing.T) {
		if err := VerifySHA256(path, "0000000000000000000000000000000000000000000000000000000000000000"); err == nil {
			t.Error("VerifySHA256() expected error for mismatching checksum")
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		if err := VerifySHA256(path, strings.ToUpper(expected)); err != nil {
			t.Errorf("VerifySHA256() should be case insensitive: %v", err)
		}
	})
}

func TestExtractBinaryFromTarGz(t *testing.T) {
	tmp := t.TempDir()

	// Create a tar.gz with a fake "lore" binary
	archivePath := filepath.Join(tmp, "lore_test.tar.gz")
	binaryContent := []byte("#!/bin/sh\necho fake lore")

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	tw.WriteHeader(&tar.Header{
		Name: "lore",
		Mode: 0755,
		Size: int64(len(binaryContent)),
	})
	tw.Write(binaryContent)
	tw.Close()
	gw.Close()
	f.Close()

	destDir := filepath.Join(tmp, "extracted")
	os.MkdirAll(destDir, 0755)

	got, err := ExtractBinary(archivePath, destDir)
	if err != nil {
		t.Fatalf("ExtractBinary() error: %v", err)
	}

	data, err := os.ReadFile(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(binaryContent) {
		t.Errorf("extracted content mismatch")
	}
}

func TestReplaceBinary(t *testing.T) {
	tmp := t.TempDir()

	// Create "old" binary
	oldPath := filepath.Join(tmp, "lore")
	os.WriteFile(oldPath, []byte("old"), 0755)

	// Create "new" binary
	newPath := filepath.Join(tmp, "lore.new")
	os.WriteFile(newPath, []byte("new"), 0755)

	if err := ReplaceBinary(oldPath, newPath); err != nil {
		t.Fatalf("ReplaceBinary() error: %v", err)
	}

	data, err := os.ReadFile(oldPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Errorf("ReplaceBinary() binary content = %q, want %q", data, "new")
	}

	// Backup should be cleaned up
	if _, err := os.Stat(oldPath + ".bak"); !os.IsNotExist(err) {
		t.Error("ReplaceBinary() backup file should have been removed")
	}
}
