// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package fileutil

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAtomicWriteJSON_ChmodError(t *testing.T) {
	restoreHooks(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	osChmod = func(name string, mode os.FileMode) error {
		return errors.New("injected chmod error")
	}

	err := AtomicWriteJSON(path, map[string]int{"a": 1}, 0o644)
	if err == nil {
		t.Fatal("expected error from chmod injection")
	}
	if !strings.Contains(err.Error(), "fileutil: chmod temp") {
		t.Errorf("unexpected error: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		t.Errorf("temp file left behind: %s", e.Name())
	}
}

func TestAtomicWriteJSON_RenameError(t *testing.T) {
	restoreHooks(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	osRename = func(oldpath, newpath string) error {
		return errors.New("injected rename error")
	}

	err := AtomicWriteJSON(path, map[string]int{"a": 1}, 0o644)
	if err == nil {
		t.Fatal("expected error from rename injection")
	}
	if !strings.Contains(err.Error(), "fileutil: rename temp") {
		t.Errorf("unexpected error: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		t.Errorf("temp file left behind: %s", e.Name())
	}
}

func TestAtomicWriteStream_ChmodError(t *testing.T) {
	restoreHooks(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "backup.dat")

	osChmod = func(name string, mode os.FileMode) error {
		return errors.New("injected chmod error")
	}

	err := AtomicWriteStream(path, bytes.NewReader([]byte("data")), 0o644)
	if err == nil {
		t.Fatal("expected error from chmod injection")
	}
	if !strings.Contains(err.Error(), "fileutil: chmod temp") {
		t.Errorf("unexpected error: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		t.Errorf("temp file left behind: %s", e.Name())
	}
}

func TestAtomicWriteStream_RenameError(t *testing.T) {
	restoreHooks(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "backup.dat")

	osRename = func(oldpath, newpath string) error {
		return errors.New("injected rename error")
	}

	err := AtomicWriteStream(path, bytes.NewReader([]byte("data")), 0o644)
	if err == nil {
		t.Fatal("expected error from rename injection")
	}
	if !strings.Contains(err.Error(), "fileutil: rename temp") {
		t.Errorf("unexpected error: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		t.Errorf("temp file left behind: %s", e.Name())
	}
}

func TestAtomicWriteJSON_ChmodError_CleanupFails(t *testing.T) {
	restoreHooks(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	osChmod = func(name string, mode os.FileMode) error {
		os.Remove(name)
		return errors.New("injected chmod error")
	}

	err := AtomicWriteJSON(path, map[string]int{"a": 1}, 0o644)
	if err == nil {
		t.Fatal("expected error from chmod injection")
	}
	if !strings.Contains(err.Error(), "fileutil: chmod temp") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAtomicWriteJSON_RenameError_CleanupFails(t *testing.T) {
	restoreHooks(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	osRename = func(oldpath, newpath string) error {
		os.Remove(oldpath)
		return errors.New("injected rename error")
	}

	err := AtomicWriteJSON(path, map[string]int{"a": 1}, 0o644)
	if err == nil {
		t.Fatal("expected error from rename injection")
	}
	if !strings.Contains(err.Error(), "fileutil: rename temp") {
		t.Errorf("unexpected error: %v", err)
	}
}

type errReader struct{ err error }

func (e *errReader) Read(p []byte) (int, error) { return 0, e.err }

func TestAtomicWriteStream_WriteCallbackError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backup.dat")

	r := &errReader{err: errors.New("injected read error")}
	err := AtomicWriteStream(path, r, 0o644)
	if err == nil {
		t.Fatal("expected error from failing reader")
	}
	if !strings.Contains(err.Error(), "fileutil: copy") {
		t.Errorf("unexpected error: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		t.Errorf("temp file left behind: %s", e.Name())
	}
}
