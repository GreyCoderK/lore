// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package fileutil

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// restoreHooks saves current hook values and returns a cleanup function.
func restoreHooks(t *testing.T) {
	t.Helper()
	origChmod := osChmod
	origRename := osRename
	origLink := osLink
	t.Cleanup(func() {
		osChmod = origChmod
		osRename = origRename
		osLink = origLink
	})
}

// --- AtomicWrite error injection tests ---

func TestAtomicWrite_ChmodError(t *testing.T) {
	restoreHooks(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	osChmod = func(name string, mode os.FileMode) error {
		return errors.New("injected chmod error")
	}

	err := AtomicWrite(path, []byte("data"), 0644)
	if err == nil {
		t.Fatal("expected error from chmod injection")
	}
	if !strings.Contains(err.Error(), "fileutil: chmod temp") {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify temp file was cleaned up
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		t.Errorf("temp file left behind: %s", e.Name())
	}
}

func TestAtomicWrite_RenameError(t *testing.T) {
	restoreHooks(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	osRename = func(oldpath, newpath string) error {
		return errors.New("injected rename error")
	}

	err := AtomicWrite(path, []byte("data"), 0644)
	if err == nil {
		t.Fatal("expected error from rename injection")
	}
	if !strings.Contains(err.Error(), "fileutil: rename temp") {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify temp file was cleaned up
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		t.Errorf("temp file left behind: %s", e.Name())
	}
}

func TestAtomicWrite_ChmodError_CleanupFails(t *testing.T) {
	restoreHooks(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	osChmod = func(name string, mode os.FileMode) error {
		// Remove the temp file before returning error, so cleanup also fails
		os.Remove(name)
		return errors.New("injected chmod error")
	}

	err := AtomicWrite(path, []byte("data"), 0644)
	if err == nil {
		t.Fatal("expected error from chmod injection")
	}
	if !strings.Contains(err.Error(), "fileutil: chmod temp") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAtomicWrite_RenameError_CleanupFails(t *testing.T) {
	restoreHooks(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	osRename = func(oldpath, newpath string) error {
		// Remove the temp file so cleanup also fails
		os.Remove(oldpath)
		return errors.New("injected rename error")
	}

	err := AtomicWrite(path, []byte("data"), 0644)
	if err == nil {
		t.Fatal("expected error from rename injection")
	}
	if !strings.Contains(err.Error(), "fileutil: rename temp") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- AtomicWriteExclusive error injection tests ---

func TestAtomicWriteExclusive_ChmodError(t *testing.T) {
	restoreHooks(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	osChmod = func(name string, mode os.FileMode) error {
		return errors.New("injected chmod error")
	}

	err := AtomicWriteExclusive(path, []byte("data"), 0644)
	if err == nil {
		t.Fatal("expected error from chmod injection")
	}
	if !strings.Contains(err.Error(), "fileutil: chmod temp") {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify temp file was cleaned up
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		t.Errorf("temp file left behind: %s", e.Name())
	}
}

func TestAtomicWriteExclusive_LinkError(t *testing.T) {
	restoreHooks(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	osLink = func(oldname, newname string) error {
		return errors.New("injected link error")
	}

	err := AtomicWriteExclusive(path, []byte("data"), 0644)
	if err == nil {
		t.Fatal("expected error from link injection")
	}
	if !strings.Contains(err.Error(), "injected link error") {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify temp file was cleaned up
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		t.Errorf("temp file left behind: %s", e.Name())
	}
}

func TestAtomicWriteExclusive_ChmodError_CleanupFails(t *testing.T) {
	restoreHooks(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	osChmod = func(name string, mode os.FileMode) error {
		os.Remove(name)
		return errors.New("injected chmod error")
	}

	err := AtomicWriteExclusive(path, []byte("data"), 0644)
	if err == nil {
		t.Fatal("expected error from chmod injection")
	}
	if !strings.Contains(err.Error(), "fileutil: chmod temp") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAtomicWriteExclusive_LinkError_CleanupFails(t *testing.T) {
	restoreHooks(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	osLink = func(oldname, newname string) error {
		os.Remove(oldname)
		return errors.New("injected link error")
	}

	err := AtomicWriteExclusive(path, []byte("data"), 0644)
	if err == nil {
		t.Fatal("expected error from link injection")
	}
	if !strings.Contains(err.Error(), "injected link error") {
		t.Errorf("unexpected error: %v", err)
	}
}
