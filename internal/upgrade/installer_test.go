// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package upgrade

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
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

	binaryName := "lore"
	if runtime.GOOS == "windows" {
		binaryName = "lore.exe"
	}

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	tw.WriteHeader(&tar.Header{
		Name: binaryName,
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

func TestDownloadAsset(t *testing.T) {
	content := []byte("fake binary content")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer srv.Close()

	destDir := t.TempDir()
	ctx := context.Background()

	path, err := DownloadAsset(ctx, srv.Client(), srv.URL+"/lore_Darwin_arm64.tar.gz", destDir)
	if err != nil {
		t.Fatalf("DownloadAsset: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("content mismatch: got %q", data)
	}
	if filepath.Base(path) != "lore_Darwin_arm64.tar.gz" {
		t.Errorf("filename = %q, want lore_Darwin_arm64.tar.gz", filepath.Base(path))
	}
}

func TestDownloadAsset_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	destDir := t.TempDir()
	ctx := context.Background()

	_, err := DownloadAsset(ctx, srv.Client(), srv.URL+"/missing.tar.gz", destDir)
	if err == nil {
		t.Error("expected error for 404 status")
	}
}

func TestDownloadChecksum(t *testing.T) {
	checksums := "abc123def456  lore_Darwin_arm64.tar.gz\nfedcba987654  lore_Linux_x86_64.tar.gz\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(checksums))
	}))
	defer srv.Close()

	ctx := context.Background()

	hash, err := DownloadChecksum(ctx, srv.Client(), srv.URL+"/checksums.txt", "lore_Darwin_arm64.tar.gz")
	if err != nil {
		t.Fatalf("DownloadChecksum: %v", err)
	}
	if hash != "abc123def456" {
		t.Errorf("hash = %q, want abc123def456", hash)
	}
}

func TestDownloadChecksum_NotFound(t *testing.T) {
	checksums := "abc123  other_file.tar.gz\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(checksums))
	}))
	defer srv.Close()

	ctx := context.Background()

	_, err := DownloadChecksum(ctx, srv.Client(), srv.URL+"/checksums.txt", "lore_missing.tar.gz")
	if err == nil {
		t.Error("expected error when checksum not found")
	}
}

func TestDownloadChecksum_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx := context.Background()

	_, err := DownloadChecksum(ctx, srv.Client(), srv.URL+"/checksums.txt", "any.tar.gz")
	if err == nil {
		t.Error("expected error for 500 status")
	}
}

func TestExtractBinary_TarGz_NotFound(t *testing.T) {
	tmp := t.TempDir()

	// Create tar.gz with no "lore" binary
	archivePath := filepath.Join(tmp, "empty.tar.gz")
	f, _ := os.Create(archivePath)
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	tw.WriteHeader(&tar.Header{Name: "README.md", Mode: 0644, Size: 5})
	tw.Write([]byte("hello"))
	tw.Close()
	gw.Close()
	f.Close()

	destDir := filepath.Join(tmp, "out")
	os.MkdirAll(destDir, 0755)

	_, err := ExtractBinary(archivePath, destDir)
	if err == nil {
		t.Error("expected error when binary not found in archive")
	}
}

func TestExtractBinary_Zip(t *testing.T) {
	tmp := t.TempDir()

	binaryName := "lore"
	if runtime.GOOS == "windows" {
		binaryName = "lore.exe"
	}

	// Create zip with a lore binary
	archivePath := filepath.Join(tmp, "lore.zip")
	zf, _ := os.Create(archivePath)
	zw := zip.NewWriter(zf)
	w, _ := zw.Create(binaryName)
	w.Write([]byte("zip binary content"))
	zw.Close()
	zf.Close()

	destDir := filepath.Join(tmp, "out")
	os.MkdirAll(destDir, 0755)

	got, err := ExtractBinary(archivePath, destDir)
	if err != nil {
		t.Fatalf("ExtractBinary zip: %v", err)
	}

	data, _ := os.ReadFile(got)
	if string(data) != "zip binary content" {
		t.Errorf("content = %q, want 'zip binary content'", data)
	}
}

func TestExtractBinary_Zip_NotFound(t *testing.T) {
	tmp := t.TempDir()

	archivePath := filepath.Join(tmp, "empty.zip")
	zf, _ := os.Create(archivePath)
	zw := zip.NewWriter(zf)
	w, _ := zw.Create("README.md")
	w.Write([]byte("no binary here"))
	zw.Close()
	zf.Close()

	destDir := filepath.Join(tmp, "out")
	os.MkdirAll(destDir, 0755)

	_, err := ExtractBinary(archivePath, destDir)
	if err == nil {
		t.Error("expected error when binary not found in zip")
	}
}

func TestReplaceBinary_NonExistent(t *testing.T) {
	tmp := t.TempDir()
	err := ReplaceBinary(filepath.Join(tmp, "nonexistent"), filepath.Join(tmp, "new"))
	if err == nil {
		t.Error("expected error for nonexistent target")
	}
}

func TestCheckWritable(t *testing.T) {
	dir := t.TempDir()
	if err := checkWritable(dir); err != nil {
		t.Errorf("checkWritable on writable dir: %v", err)
	}
}

func TestCopyFile(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.bin")
	dst := filepath.Join(tmp, "dst.bin")
	os.WriteFile(src, []byte("copy me"), 0644)

	if err := copyFile(src, dst, 0755); err != nil {
		t.Fatalf("copyFile: %v", err)
	}

	data, _ := os.ReadFile(dst)
	if string(data) != "copy me" {
		t.Errorf("content = %q, want 'copy me'", data)
	}
}
