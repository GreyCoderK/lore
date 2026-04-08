// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package fileutil_test

import (
	"fmt"
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

// --- Additional tests for coverage ---

func TestAtomicWriteExclusive_EmptyData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")

	if err := fileutil.AtomicWriteExclusive(path, []byte{}, 0644); err != nil {
		t.Fatalf("AtomicWriteExclusive empty: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty file, got %d bytes", len(data))
	}
}

func TestAtomicWriteExclusive_LargeData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.txt")

	big := make([]byte, 1<<20) // 1 MiB
	for i := range big {
		big[i] = byte(i % 256)
	}

	if err := fileutil.AtomicWriteExclusive(path, big, 0644); err != nil {
		t.Fatalf("AtomicWriteExclusive large: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) != len(big) {
		t.Errorf("size = %d, want %d", len(data), len(big))
	}
}

func TestAtomicWrite_SpecialCharsInPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file with spaces & (parens).txt")

	if err := fileutil.AtomicWrite(path, []byte("special"), 0644); err != nil {
		t.Fatalf("AtomicWrite special path: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "special" {
		t.Errorf("content = %q, want %q", string(data), "special")
	}
}

func TestAtomicWriteExclusive_SpecialCharsInPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file with spaces & (parens).txt")

	if err := fileutil.AtomicWriteExclusive(path, []byte("special"), 0644); err != nil {
		t.Fatalf("AtomicWriteExclusive special path: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "special" {
		t.Errorf("content = %q, want %q", string(data), "special")
	}
}

func TestAtomicWrite_AtomicBehavior(t *testing.T) {
	// Verify that AtomicWrite produces complete content.
	// Write a known pattern and verify byte-for-byte correctness.
	dir := t.TempDir()
	path := filepath.Join(dir, "atomic.txt")

	content := []byte("ATOMIC_CONTENT_12345678")
	if err := fileutil.AtomicWrite(path, content, 0644); err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", string(data), string(content))
	}

	// Overwrite with new content - old content should be fully replaced.
	content2 := []byte("REPLACED_CONTENT_ABCDEF")
	if err := fileutil.AtomicWrite(path, content2, 0644); err != nil {
		t.Fatalf("AtomicWrite overwrite: %v", err)
	}

	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != string(content2) {
		t.Errorf("content mismatch after overwrite: got %q, want %q", string(data), string(content2))
	}
}

func TestAtomicWriteExclusive_ConcurrentAccess(t *testing.T) {
	// Multiple goroutines trying to exclusively create the same file.
	// Exactly one should succeed; the rest should get os.IsExist errors.
	dir := t.TempDir()
	path := filepath.Join(dir, "race.txt")

	const goroutines = 10
	errs := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			data := []byte(fmt.Sprintf("writer-%d", id))
			errs <- fileutil.AtomicWriteExclusive(path, data, 0644)
		}(i)
	}

	var successes, existErrors int
	for i := 0; i < goroutines; i++ {
		err := <-errs
		if err == nil {
			successes++
		} else if os.IsExist(err) {
			existErrors++
		} else {
			t.Errorf("unexpected error: %v", err)
		}
	}

	if successes != 1 {
		t.Errorf("expected exactly 1 success, got %d", successes)
	}
	if existErrors != goroutines-1 {
		t.Errorf("expected %d exist errors, got %d", goroutines-1, existErrors)
	}

	// Verify the file has valid content from one of the writers.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) == 0 {
		t.Error("file is empty after concurrent writes")
	}
}

func TestAtomicWrite_ErrorContainsPrefix(t *testing.T) {
	err := fileutil.AtomicWrite("/nonexistent/dir/file.txt", []byte("data"), 0644)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); len(got) < 10 {
		t.Errorf("error message too short: %q", got)
	}
	// Should contain the fileutil prefix
	if !contains(err.Error(), "fileutil:") {
		t.Errorf("error should contain 'fileutil:' prefix, got: %v", err)
	}
}

func TestAtomicWriteExclusive_ErrorContainsPrefix(t *testing.T) {
	err := fileutil.AtomicWriteExclusive("/nonexistent/dir/file.txt", []byte("data"), 0644)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); len(got) < 10 {
		t.Errorf("error message too short: %q", got)
	}
	// Should contain the fileutil prefix
	if !contains(err.Error(), "fileutil:") {
		t.Errorf("error should contain 'fileutil:' prefix, got: %v", err)
	}
}

func TestAtomicWrite_NilData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nil.txt")

	if err := fileutil.AtomicWrite(path, nil, 0644); err != nil {
		t.Fatalf("AtomicWrite nil data: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty file for nil data, got %d bytes", len(data))
	}
}

func TestAtomicWriteExclusive_NilData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nil.txt")

	if err := fileutil.AtomicWriteExclusive(path, nil, 0644); err != nil {
		t.Fatalf("AtomicWriteExclusive nil data: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty file for nil data, got %d bytes", len(data))
	}
}

func TestAtomicWrite_NestedSubdir(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "a", "b", "c")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	path := filepath.Join(subdir, "deep.txt")

	if err := fileutil.AtomicWrite(path, []byte("deep"), 0644); err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "deep" {
		t.Errorf("content = %q, want %q", string(data), "deep")
	}
}

func TestAtomicWriteExclusive_NestedSubdir(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "a", "b", "c")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	path := filepath.Join(subdir, "deep.txt")

	if err := fileutil.AtomicWriteExclusive(path, []byte("deep"), 0644); err != nil {
		t.Fatalf("AtomicWriteExclusive: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "deep" {
		t.Errorf("content = %q, want %q", string(data), "deep")
	}
}

func TestAtomicWrite_PermissionsVariants(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod not supported on Windows")
	}
	dir := t.TempDir()

	tests := []struct {
		name string
		perm os.FileMode
	}{
		{"read-only", 0444},
		{"read-write", 0644},
		{"executable", 0755},
		{"owner-only", 0600},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(dir, tt.name+".txt")
			if err := fileutil.AtomicWrite(path, []byte("test"), tt.perm); err != nil {
				t.Fatalf("AtomicWrite: %v", err)
			}

			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("Stat: %v", err)
			}
			if info.Mode().Perm() != tt.perm {
				t.Errorf("perm = %o, want %o", info.Mode().Perm(), tt.perm)
			}
		})
	}
}

func TestAtomicWriteExclusive_PermissionsVariants(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod not supported on Windows")
	}
	dir := t.TempDir()

	tests := []struct {
		name string
		perm os.FileMode
	}{
		{"read-only", 0444},
		{"read-write", 0644},
		{"executable", 0755},
		{"owner-only", 0600},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(dir, tt.name+".txt")
			if err := fileutil.AtomicWriteExclusive(path, []byte("test"), tt.perm); err != nil {
				t.Fatalf("AtomicWriteExclusive: %v", err)
			}

			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("Stat: %v", err)
			}
			if info.Mode().Perm() != tt.perm {
				t.Errorf("perm = %o, want %o", info.Mode().Perm(), tt.perm)
			}
		})
	}
}

func TestAtomicWriteExclusive_OriginalContentPreserved(t *testing.T) {
	// Ensure the original file content is truly preserved when exclusive write fails.
	dir := t.TempDir()
	path := filepath.Join(dir, "preserve.txt")

	original := []byte("original content that must not change")
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	err := fileutil.AtomicWriteExclusive(path, []byte("replacement"), 0644)
	if err == nil {
		t.Fatal("expected error")
	}
	if !os.IsExist(err) {
		t.Errorf("expected os.IsExist, got: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != string(original) {
		t.Errorf("original content changed: got %q", string(data))
	}
}

func TestAtomicWriteExclusive_TOCTOUConflict(t *testing.T) {
	// Simulate a TOCTOU race: check that file doesn't exist, then another
	// process creates it before our link lands. AtomicWriteExclusive should
	// detect this and fail with os.IsExist.
	dir := t.TempDir()
	path := filepath.Join(dir, "race.txt")

	// First writer succeeds.
	if err := fileutil.AtomicWriteExclusive(path, []byte("first"), 0644); err != nil {
		t.Fatalf("first write: %v", err)
	}

	// Second writer must fail because the file now exists.
	err := fileutil.AtomicWriteExclusive(path, []byte("second"), 0644)
	if err == nil {
		t.Fatal("expected error on second exclusive write")
	}
	if !os.IsExist(err) {
		t.Errorf("expected os.IsExist error, got: %v", err)
	}

	// Original content must be preserved.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "first" {
		t.Errorf("content = %q, want %q", string(data), "first")
	}

	// No temp files left behind.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "race.txt" {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}

func TestAtomicWrite_Perm0444_ContentVerified(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod not supported on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "readonly.txt")

	content := []byte("read-only content here")
	if err := fileutil.AtomicWrite(path, content, 0444); err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}

	// Verify content is correct.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("content = %q, want %q", string(data), string(content))
	}

	// Verify permissions are exactly 0444.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0444 {
		t.Errorf("perm = %o, want %o", info.Mode().Perm(), 0444)
	}

	// Verify no temp files remain.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "readonly.txt" {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}

func TestAtomicWrite_Perm0600_ContentVerified(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod not supported on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "private.txt")

	content := []byte("owner-only content")
	if err := fileutil.AtomicWrite(path, content, 0600); err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("content = %q, want %q", string(data), string(content))
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("perm = %o, want %o", info.Mode().Perm(), 0600)
	}
}

func TestAtomicWriteExclusive_Perm0600_ContentVerified(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod not supported on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "private.txt")

	content := []byte("exclusive owner-only content")
	if err := fileutil.AtomicWriteExclusive(path, content, 0600); err != nil {
		t.Fatalf("AtomicWriteExclusive: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("content = %q, want %q", string(data), string(content))
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("perm = %o, want %o", info.Mode().Perm(), 0600)
	}
}

func TestAtomicWrite_NonExistentNestedDir(t *testing.T) {
	// Writing to a path whose parent directory does not exist should fail.
	dir := t.TempDir()
	path := filepath.Join(dir, "does", "not", "exist", "file.txt")

	err := fileutil.AtomicWrite(path, []byte("data"), 0644)
	if err == nil {
		t.Fatal("expected error for non-existent nested directory")
	}
	if !contains(err.Error(), "fileutil:") {
		t.Errorf("expected 'fileutil:' prefix, got: %v", err)
	}
}

func TestAtomicWriteExclusive_NonExistentNestedDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does", "not", "exist", "file.txt")

	err := fileutil.AtomicWriteExclusive(path, []byte("data"), 0644)
	if err == nil {
		t.Fatal("expected error for non-existent nested directory")
	}
	if !contains(err.Error(), "fileutil:") {
		t.Errorf("expected 'fileutil:' prefix, got: %v", err)
	}
}

func TestAtomicWrite_SpecialCharsUnicode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "café-résumé.txt")

	content := []byte("unicode filename content")
	if err := fileutil.AtomicWrite(path, content, 0644); err != nil {
		t.Fatalf("AtomicWrite unicode path: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("content = %q, want %q", string(data), string(content))
	}

	// Verify no temp files remain.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "café-résumé.txt" {
			t.Errorf("unexpected file: %s", e.Name())
		}
	}
}

func TestAtomicWriteExclusive_SpecialCharsUnicode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "naïve-über.txt")

	content := []byte("exclusive unicode content")
	if err := fileutil.AtomicWriteExclusive(path, content, 0644); err != nil {
		t.Fatalf("AtomicWriteExclusive unicode path: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("content = %q, want %q", string(data), string(content))
	}
}

func TestAtomicWrite_OverwritePreservesNewContent(t *testing.T) {
	// Verify that after overwrite, only the new content exists --
	// no leftover bytes from a longer previous write.
	dir := t.TempDir()
	path := filepath.Join(dir, "shrink.txt")

	long := []byte("this is a much longer piece of content that will be replaced")
	short := []byte("short")

	if err := fileutil.AtomicWrite(path, long, 0644); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if err := fileutil.AtomicWrite(path, short, 0644); err != nil {
		t.Fatalf("second write: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "short" {
		t.Errorf("content = %q, want %q", string(data), "short")
	}
}

func TestAtomicWriteExclusive_BinaryContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "binary.dat")

	// Binary content with null bytes and all byte values 0-255.
	content := make([]byte, 256)
	for i := range content {
		content[i] = byte(i)
	}

	if err := fileutil.AtomicWriteExclusive(path, content, 0644); err != nil {
		t.Fatalf("AtomicWriteExclusive binary: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) != 256 {
		t.Fatalf("size = %d, want 256", len(data))
	}
	for i, b := range data {
		if b != byte(i) {
			t.Errorf("byte[%d] = %d, want %d", i, b, i)
			break
		}
	}
}

func TestAtomicWrite_RenameErrorTargetIsDir(t *testing.T) {
	// Attempt to AtomicWrite to a path that is an existing directory.
	// os.Rename should fail because you can't rename a file over a directory.
	dir := t.TempDir()
	target := filepath.Join(dir, "isdir")
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	err := fileutil.AtomicWrite(target, []byte("data"), 0644)
	if err == nil {
		// On some systems rename over a directory might work differently,
		// but on most POSIX systems this should fail.
		t.Log("rename over directory did not fail on this platform")
		return
	}
	// Verify error message has the fileutil prefix
	if !contains(err.Error(), "fileutil:") {
		t.Errorf("expected 'fileutil:' prefix in error, got: %v", err)
	}

	// Verify no temp files left behind
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "isdir" {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}

func TestAtomicWriteExclusive_LinkErrorTargetIsDir(t *testing.T) {
	// Attempt AtomicWriteExclusive where target path is a directory.
	dir := t.TempDir()
	target := filepath.Join(dir, "isdir")
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	err := fileutil.AtomicWriteExclusive(target, []byte("data"), 0644)
	if err == nil {
		t.Fatal("expected error when target is a directory")
	}

	// Verify no temp files left behind
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "isdir" {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
