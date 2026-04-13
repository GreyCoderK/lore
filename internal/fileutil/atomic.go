// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package fileutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Test hooks – default to real os operations. Tests can override these
// to inject faults into specific stages of the atomic-write pipeline.
var (
	osChmod    = os.Chmod
	osRename   = os.Rename
	osLink     = os.Link
	fileWrite  = func(f *os.File, data []byte) (int, error) { return f.Write(data) }
	fileClose  = func(f *os.File) error { return f.Close() }
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

	if _, err := fileWrite(tmp, data); err != nil {
		_ = tmp.Close()
		if removeErr := os.Remove(tmpName); removeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to clean temp file %s: %v\n", tmpName, removeErr)
		}
		return fmt.Errorf("fileutil: write temp: %w", err)
	}
	if err := fileClose(tmp); err != nil {
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

// writeAtomicDurable is the shared tempfile+rename core used by the
// durable variants (AtomicWriteJSON, AtomicWriteStream). It differs
// from AtomicWrite in two ways: the tempfile is fsynced before close
// so its contents survive a crash after the rename, and the parent
// directory is fsynced after the rename so POSIX-compliant systems
// guarantee the new name is on disk. Reserved for callers that care
// about durability (state files, backups) — the cheaper AtomicWrite
// remains fine for outputs the user can regenerate.
//
// Consolidates the atomic write bodies from SaveDraftState,
// SaveReviewState, WriteBackup, RestoreBackup into one place so
// future durability tweaks touch one file instead of four.
func writeAtomicDurable(path string, perm os.FileMode, write func(*os.File) error) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".lore-durable-*.tmp")
	if err != nil {
		return fmt.Errorf("fileutil: create temp: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if err := write(tmp); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("fileutil: sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("fileutil: close temp: %w", err)
	}
	if err := osChmod(tmpName, perm); err != nil {
		return fmt.Errorf("fileutil: chmod temp: %w", err)
	}
	if err := osRename(tmpName, path); err != nil {
		return fmt.Errorf("fileutil: rename temp: %w", err)
	}
	cleanup = false
	// fsync parent dir so the rename survives a crash (POSIX).
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

// AtomicWriteJSON marshals v as indented JSON and writes it to path
// via writeAtomicDurable. Used for state files where the indented
// form is worth the handful of extra bytes for human diff-ability.
// Returns a non-nil error if marshaling fails; the target file is
// left unchanged in that case.
func AtomicWriteJSON(path string, v any, perm os.FileMode) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("fileutil: json encode: %w", err)
	}
	return writeAtomicDurable(path, perm, func(f *os.File) error {
		if _, err := f.Write(buf.Bytes()); err != nil {
			return fmt.Errorf("fileutil: write temp: %w", err)
		}
		return nil
	})
}

// AtomicWriteStream streams src into path via writeAtomicDurable.
// Used by backup/restore where the source is an already-open file
// and buffering the full contents into memory would be wasteful.
func AtomicWriteStream(path string, src io.Reader, perm os.FileMode) error {
	return writeAtomicDurable(path, perm, func(f *os.File) error {
		if _, err := io.Copy(f, src); err != nil {
			return fmt.Errorf("fileutil: copy: %w", err)
		}
		return nil
	})
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

	if _, err := fileWrite(tmp, data); err != nil {
		_ = tmp.Close()
		if removeErr := os.Remove(tmpName); removeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to clean temp file %s: %v\n", tmpName, removeErr)
		}
		return fmt.Errorf("fileutil: write temp: %w", err)
	}
	if err := fileClose(tmp); err != nil {
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
