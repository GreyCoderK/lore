// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package fileutil_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/greycoderk/lore/internal/fileutil"
)

func TestNewFileLock_ReturnsNonNil(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	fl, err := fileutil.NewFileLock(path)
	if err != nil {
		t.Fatalf("NewFileLock: %v", err)
	}
	if fl == nil {
		t.Fatal("expected non-nil *FileLock")
	}
	fl.Unlock()
}

func TestNewFileLock_CreatesDotLockFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	fl, err := fileutil.NewFileLock(path)
	if err != nil {
		t.Fatalf("NewFileLock: %v", err)
	}
	defer fl.Unlock()

	lockPath := path + ".lock"
	if _, err := os.Stat(lockPath); err != nil {
		t.Errorf("lock file %q not found: %v", lockPath, err)
	}
}

func TestFileLock_UnlockIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	fl, err := fileutil.NewFileLock(path)
	if err != nil {
		t.Fatalf("NewFileLock: %v", err)
	}
	fl.Unlock()
	fl.Unlock()
}

func TestFileLock_UnlockNilNoPanic(t *testing.T) {
	var fl *fileutil.FileLock
	fl.Unlock()
}

func TestNewFileLock_CreatesNestedDir(t *testing.T) {
	base := t.TempDir()
	path := filepath.Join(base, "a", "b", "c", "state.json")

	fl, err := fileutil.NewFileLock(path)
	if err != nil {
		t.Fatalf("NewFileLock with nested path: %v", err)
	}
	defer fl.Unlock()

	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Errorf("nested dir not created: %v", err)
	}

	lockPath := path + ".lock"
	if _, err := os.Stat(lockPath); err != nil {
		t.Errorf("lock file not created in nested dir: %v", err)
	}
}

func TestNewFileLock_LockPathSuffix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "myfile.dat")

	fl, err := fileutil.NewFileLock(path)
	if err != nil {
		t.Fatalf("NewFileLock: %v", err)
	}
	defer fl.Unlock()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	found := false
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".lock") {
			found = true
		}
	}
	if !found {
		t.Error("no .lock file found in directory")
	}
}

func TestNewFileLock_GoroutinesSerialize(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("flock not supported on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "shared.json")

	const workers = 4
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		counter int
		errors  []error
	)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fl, err := fileutil.NewFileLock(path)
			if err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
				return
			}
			mu.Lock()
			counter++
			mu.Unlock()
			fl.Unlock()
		}()
	}
	wg.Wait()

	if len(errors) > 0 {
		t.Fatalf("goroutine errors: %v", errors)
	}
	if counter != workers {
		t.Errorf("counter = %d, want %d", counter, workers)
	}
}

func TestNewFileLock_SequentialAcquireRelease(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	for i := 0; i < 3; i++ {
		fl, err := fileutil.NewFileLock(path)
		if err != nil {
			t.Fatalf("iteration %d: NewFileLock: %v", i, err)
		}
		fl.Unlock()
	}
}

func TestNewFileLock_MkdirAllFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod not supported on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("root can create dirs anywhere")
	}
	dir := t.TempDir()
	blockingFile := filepath.Join(dir, "blocked")
	if err := os.WriteFile(blockingFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	path := filepath.Join(blockingFile, "subdir", "state.json")
	_, err := fileutil.NewFileLock(path)
	if err == nil {
		t.Fatal("expected error when MkdirAll fails")
	}
}

func TestNewFileLock_OpenFileFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod not supported on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("root can open any file")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	defer os.Chmod(dir, 0o755)

	path := filepath.Join(dir, "state.json")
	_, err := fileutil.NewFileLock(path)
	if err == nil {
		t.Fatal("expected error when lock file cannot be opened")
	}
}
