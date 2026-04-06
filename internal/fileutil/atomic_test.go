// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package fileutil_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/greycoderk/lore/internal/fileutil"
)

func TestAtomicWrite_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	if err := fileutil.AtomicWrite(path, []byte("hello"), 0644); err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("content = %q, want %q", string(data), "hello")
	}
}

func TestAtomicWrite_SetsPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod not supported on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "hook.sh")

	if err := fileutil.AtomicWrite(path, []byte("#!/bin/sh"), 0755); err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm()&0100 == 0 {
		t.Errorf("expected executable permission, got %o", info.Mode().Perm())
	}
}

func TestAtomicWrite_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	if err := fileutil.AtomicWrite(path, []byte("first"), 0644); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if err := fileutil.AtomicWrite(path, []byte("second"), 0644); err != nil {
		t.Fatalf("second write: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "second" {
		t.Errorf("content = %q, want %q", string(data), "second")
	}
}

func TestAtomicWrite_NoTempLeftOnSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	if err := fileutil.AtomicWrite(path, []byte("data"), 0644); err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "test.txt" {
			t.Errorf("unexpected temp file left behind: %s", e.Name())
		}
	}
}

func TestAtomicWriteExclusive_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.txt")

	if err := fileutil.AtomicWriteExclusive(path, []byte("exclusive"), 0644); err != nil {
		t.Fatalf("AtomicWriteExclusive: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "exclusive" {
		t.Errorf("content = %q, want %q", string(data), "exclusive")
	}
}

func TestAtomicWriteExclusive_FailsIfExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exists.txt")

	if err := os.WriteFile(path, []byte("first"), 0644); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	err := fileutil.AtomicWriteExclusive(path, []byte("second"), 0644)
	if err == nil {
		t.Fatal("expected error for existing file")
	}
	if !os.IsExist(err) {
		t.Errorf("expected os.IsExist(err), got: %v", err)
	}

	// Original content must be preserved
	data, _ := os.ReadFile(path)
	if string(data) != "first" {
		t.Errorf("original content overwritten: got %q", string(data))
	}
}

func TestAtomicWriteExclusive_SetsPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod not supported on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "exec.sh")

	if err := fileutil.AtomicWriteExclusive(path, []byte("#!/bin/sh"), 0755); err != nil {
		t.Fatalf("AtomicWriteExclusive: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm()&0100 == 0 {
		t.Errorf("expected executable permission, got %o", info.Mode().Perm())
	}
}

func TestAtomicWrite_ReadOnlyDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("read-only directories not enforced on Windows")
	}
	dir := t.TempDir()
	roDir := filepath.Join(dir, "readonly")
	os.MkdirAll(roDir, 0o755)
	os.Chmod(roDir, 0o555)
	t.Cleanup(func() { os.Chmod(roDir, 0o755) })

	path := filepath.Join(roDir, "test.txt")
	err := fileutil.AtomicWrite(path, []byte("data"), 0644)
	if err == nil {
		t.Error("expected error writing to read-only directory")
	}
}

func TestAtomicWrite_BadDir(t *testing.T) {
	err := fileutil.AtomicWrite("/nonexistent/dir/file.txt", []byte("data"), 0644)
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}

func TestAtomicWriteExclusive_BadDir(t *testing.T) {
	err := fileutil.AtomicWriteExclusive("/nonexistent/dir/file.txt", []byte("data"), 0644)
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}

func TestAtomicWriteExclusive_ReadOnlyDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("read-only directories not enforced on Windows")
	}
	dir := t.TempDir()
	roDir := filepath.Join(dir, "readonly")
	os.MkdirAll(roDir, 0o755)
	os.Chmod(roDir, 0o555)
	t.Cleanup(func() { os.Chmod(roDir, 0o755) })

	path := filepath.Join(roDir, "new.txt")
	err := fileutil.AtomicWriteExclusive(path, []byte("data"), 0644)
	if err == nil {
		t.Error("expected error writing to read-only directory")
	}
}

func TestAtomicWriteExclusive_NoTempLeftOnSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clean.txt")

	if err := fileutil.AtomicWriteExclusive(path, []byte("data"), 0644); err != nil {
		t.Fatalf("AtomicWriteExclusive: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "clean.txt" {
			t.Errorf("unexpected temp file left behind: %s", e.Name())
		}
	}
}

func TestAtomicWriteExclusive_NoTempLeftOnFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exists.txt")
	os.WriteFile(path, []byte("keep"), 0644)

	_ = fileutil.AtomicWriteExclusive(path, []byte("second"), 0644)

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "exists.txt" {
			t.Errorf("temp file left behind after failure: %s", e.Name())
		}
	}
}

func TestAtomicWrite_EmptyData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")

	if err := fileutil.AtomicWrite(path, []byte{}, 0644); err != nil {
		t.Fatalf("AtomicWrite empty: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty file, got %d bytes", len(data))
	}
}

func TestAtomicWrite_LargeData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.txt")

	big := make([]byte, 1<<20) // 1 MiB
	for i := range big {
		big[i] = byte(i % 256)
	}

	if err := fileutil.AtomicWrite(path, big, 0644); err != nil {
		t.Fatalf("AtomicWrite large: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) != len(big) {
		t.Errorf("size = %d, want %d", len(data), len(big))
	}
}
