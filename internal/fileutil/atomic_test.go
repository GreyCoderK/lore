// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package fileutil_test

import (
	"os"
	"path/filepath"
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

func TestAtomicWrite_BadDir(t *testing.T) {
	err := fileutil.AtomicWrite("/nonexistent/dir/file.txt", []byte("data"), 0644)
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}
