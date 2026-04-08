// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package fileutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// Test hooks – default to real os operations. Tests can override these
// to inject faults into specific stages of the atomic-write pipeline.
var (
	osChmod  = os.Chmod
	osRename = os.Rename
	osLink   = os.Link
)

// AtomicWrite writes data to path via a temporary file + rename for crash safety.
// The perm argument sets the final file permissions (e.g. 0644 for docs, 0755 for hooks).
func AtomicWrite(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".lore-*.tmp")
	if err != nil {
		return fmt.Errorf("fileutil: create temp: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		if removeErr := os.Remove(tmpName); removeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to clean temp file %s: %v\n", tmpName, removeErr)
		}
		return fmt.Errorf("fileutil: write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		if removeErr := os.Remove(tmpName); removeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to clean temp file %s: %v\n", tmpName, removeErr)
		}
		return fmt.Errorf("fileutil: close temp: %w", err)
	}
	if err := osChmod(tmpName, perm); err != nil {
		if removeErr := os.Remove(tmpName); removeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to clean temp file %s: %v\n", tmpName, removeErr)
		}
		return fmt.Errorf("fileutil: chmod temp: %w", err)
	}
	if err := osRename(tmpName, path); err != nil {
		if removeErr := os.Remove(tmpName); removeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to clean temp file %s: %v\n", tmpName, removeErr)
		}
		return fmt.Errorf("fileutil: rename temp: %w", err)
	}
	return nil
}

// AtomicWriteExclusive writes data to path via a temporary file + hard link.
// Unlike AtomicWrite, it fails if path already exists (returns an error
// where os.IsExist reports true). This avoids the TOCTOU race inherent
// in Stat-then-Rename patterns.
func AtomicWriteExclusive(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".lore-*.tmp")
	if err != nil {
		return fmt.Errorf("fileutil: create temp: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		if removeErr := os.Remove(tmpName); removeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to clean temp file %s: %v\n", tmpName, removeErr)
		}
		return fmt.Errorf("fileutil: write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		if removeErr := os.Remove(tmpName); removeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to clean temp file %s: %v\n", tmpName, removeErr)
		}
		return fmt.Errorf("fileutil: close temp: %w", err)
	}
	if err := osChmod(tmpName, perm); err != nil {
		if removeErr := os.Remove(tmpName); removeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to clean temp file %s: %v\n", tmpName, removeErr)
		}
		return fmt.Errorf("fileutil: chmod temp: %w", err)
	}
	// os.Link fails atomically with EEXIST if path already exists (POSIX).
	// NOTE: the os.Link error is returned unwrapped intentionally so that
	// callers can use os.IsExist(err) to detect the "already exists" case.
	if err := osLink(tmpName, path); err != nil {
		if removeErr := os.Remove(tmpName); removeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to clean temp file %s: %v\n", tmpName, removeErr)
		}
		return err
	}
	_ = os.Remove(tmpName) // hard link created; remove temp name
	return nil
}
