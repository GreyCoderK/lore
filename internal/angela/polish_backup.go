// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package angela — polish_backup.go
//
// Backup writer and restore helpers (polish safety nets).
//
// Before `lore angela polish` overwrites a document, WriteBackup copies the
// current on-disk version into a timestamped file under the state directory.
// This gives users a recoverable "undo" even when the document has not been
// committed to git yet — which is the exact scenario polish is designed for
// (polishing drafts before the first commit).
//
// The timestamp format is ISO 8601 basic (YYYYMMDDTHHmmss) local time: it is
// lexicographically sortable, contains no filesystem-hostile characters, and
// is unambiguous when combined with the `.bak` suffix. The relative path from
// workDir is preserved inside the backup directory so two files with the same
// basename (e.g. `docs/guides/intro.md` and `docs/admin/intro.md`) never
// collide.
//
// Retention pruning runs AFTER the new backup is written so a crash during
// pruning cannot leave the user without any backup.
package angela

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/greycoderk/lore/internal/fileutil"
)

// BackupTimeFormat is the filename-safe, lexicographically-sortable timestamp
// format used to stamp backup files. UTC is used so pruning stays
// deterministic across DST transitions and container TZ changes.
const BackupTimeFormat = "20060102T150405Z"

// AssertContainedRelPath rejects a relative path that would escape the
// containment root when joined. It implements the canonical Go guard
// (filepath.Rel check + explicit ".." / absolute rejection) and is used
// by every filesystem operation that joins a user-supplied
// subdirectory with a state or backup root.
//
// .lorerc fields such as `angela.polish.backup.path` and
// `angela.draft.differential.state_file` used to be joined onto the state
// directory without validation, giving a malicious config-file author a path
// to arbitrary file creation/deletion under the user's uid.
func AssertContainedRelPath(relPath string) error {
	if relPath == "" {
		return fmt.Errorf("angela: path: empty")
	}
	if filepath.IsAbs(relPath) {
		return fmt.Errorf("angela: path: %q is absolute", relPath)
	}
	clean := filepath.Clean(relPath)
	if clean == ".." || clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("angela: path: %q escapes containment root", relPath)
	}
	return nil
}

// BackupSuffix is appended to the stamped filename.
const BackupSuffix = ".bak"

// backupTimestampRe captures the timestamp component (and optional
// same-second counter) from a backup filename.
// Shape: <original-filename>.<stamp>[-<counter>].bak
// Stamp is `YYYYMMDDTHHmmssZ` (UTC); counter is 1..999 used when
// multiple backups land in the same second. Leading anchor matches
// the final stamp segment even if the original filename has dots.
var backupTimestampRe = regexp.MustCompile(`\.(\d{8}T\d{6}Z)(?:-\d+)?\.bak$`)

// BackupEntry describes a single backup file for a given source document.
// Instances are returned by ListBackups sorted newest-first.
type BackupEntry struct {
	Path      string    // absolute path to the backup file on disk
	Timestamp time.Time // parsed from the filename (local time)
	Stamp     string    // raw stamp string as it appears in the filename
}

// BackupClock is the time source used by the package. Tests override it via
// SetBackupClock to get deterministic timestamps. Production uses time.Now.
//
// Protected by backupClockMu so concurrent reads and SetBackupClock swaps
// are race-free.
var (
	backupClockMu sync.RWMutex
	backupClock   = time.Now
)

// currentBackupTime reads the package clock with the read lock held.
// Always reads `.UTC()` so every caller (write, prune, list) observes
// the same reference frame — matching the UTC BackupTimeFormat.
func currentBackupTime() time.Time {
	backupClockMu.RLock()
	fn := backupClock
	backupClockMu.RUnlock()
	return fn().UTC()
}

// SetBackupClock swaps the package-level clock. The returned function
// restores the previous value — callers typically defer it.
//
//	restore := angela.SetBackupClock(func() time.Time { return fixed })
//	defer restore()
func SetBackupClock(now func() time.Time) func() {
	backupClockMu.Lock()
	prev := backupClock
	backupClock = now
	backupClockMu.Unlock()
	return func() {
		backupClockMu.Lock()
		backupClock = prev
		backupClockMu.Unlock()
	}
}

// WriteBackup copies the file at <workDir>/<relPath> into the backup area
// under stateDir. The returned path is absolute.
//
// Parameters:
//   - workDir: the user's working directory (typically ".").
//   - stateDir: absolute state directory (as returned by config.ResolveStateDir).
//   - backupSubdir: directory under stateDir that holds backups. Empty string
//     falls back to "polish-backups" so callers that haven't loaded the full
//     config still behave sensibly.
//   - relPath: the document path RELATIVE to workDir (e.g. ".lore/docs/foo.md"
//     or simply "foo.md"). Preserved inside the backup area so nested docs
//     with identical basenames don't collide.
//
// The copy is atomic with respect to the backup filename: the content is
// streamed into a tempfile in the same directory, flushed, and then renamed
// into place. A crash before the rename leaves either no backup or a
// correctly-finished one — never a half-written file.
func WriteBackup(workDir, stateDir, backupSubdir, relPath string) (string, error) {
	if relPath == "" {
		return "", fmt.Errorf("angela: backup: relPath is empty")
	}
	if stateDir == "" {
		return "", fmt.Errorf("angela: backup: stateDir is empty")
	}
	if backupSubdir == "" {
		backupSubdir = "polish-backups"
	}

	// Validate BOTH the backup subdir coming from config AND the
	// relative document path BEFORE opening anything.
	if err := AssertContainedRelPath(backupSubdir); err != nil {
		return "", fmt.Errorf("angela: backup: backupSubdir: %w", err)
	}
	if err := AssertContainedRelPath(relPath); err != nil {
		return "", fmt.Errorf("angela: backup: relPath: %w", err)
	}
	cleanRel := filepath.Clean(relPath)
	srcPath := filepath.Join(workDir, cleanRel)

	// Reject symlink source files so a malicious doc name cannot point
	// at a secret file outside the docs tree. Lstat does not follow the symlink.
	if info, lerr := os.Lstat(srcPath); lerr == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("angela: backup: refusing to back up symlink %s", srcPath)
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("angela: backup: open source: %w", err)
	}
	defer func() { _ = src.Close() }()

	// UTC end-to-end. BackupTimeFormat includes the trailing Z so filenames
	// are unambiguous and sort identically across DST transitions and machines.
	stamp := currentBackupTime().Format(BackupTimeFormat)
	backupRoot := filepath.Join(stateDir, backupSubdir)
	backupPath := filepath.Join(backupRoot, cleanRel+"."+stamp+BackupSuffix)

	// Same-second collision guard. If a polish runs twice within the same
	// second, the naive rename would silently overwrite the first backup.
	// We detect the collision and append a counter to keep both.
	if _, statErr := os.Stat(backupPath); statErr == nil {
		found := false
		for counter := 1; counter < 1000; counter++ {
			candidate := filepath.Join(backupRoot, fmt.Sprintf("%s.%s-%d%s", cleanRel, stamp, counter, BackupSuffix))
			if _, err := os.Stat(candidate); errors.Is(err, fs.ErrNotExist) {
				backupPath = candidate
				found = true
				break
			}
		}
		if !found {
			return "", fmt.Errorf("angela: backup: all 999 same-second slots exhausted for %s at stamp %s", cleanRel, stamp)
		}
	} else if !errors.Is(statErr, fs.ErrNotExist) {
		return "", fmt.Errorf("angela: backup: stat: %w", statErr)
	}

	backupDir := filepath.Dir(backupPath)
	// 0700 on the backup dir so unrelated local users cannot list/read
	// backups of sensitive draft content.
	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		return "", fmt.Errorf("angela: backup: mkdir: %w", err)
	}

	// The tempfile+sync+rename+fsync dance lives in fileutil.AtomicWriteStream.
	if err := fileutil.AtomicWriteStream(backupPath, src, 0o600); err != nil {
		return "", fmt.Errorf("angela: backup: %w", err)
	}
	return backupPath, nil
}

// PruneOldBackups deletes backup files under backupDir whose timestamp is
// older than the cutoff (now - retentionDays). A retentionDays of 0 (or less)
// disables pruning and is a no-op.
//
// The walk is non-recursive-tolerant: subdirectories are traversed so backups
// of nested documents are also pruned. Entries with unparseable filenames are
// left alone so unrelated files in the directory stay untouched.
//
// Errors encountered while removing a single file do not abort the walk —
// they are collected and returned as a joined error so the caller can log
// them without losing the other successful deletions.
func PruneOldBackups(backupDir string, retentionDays int) error {
	if retentionDays <= 0 {
		return nil
	}
	info, err := os.Stat(backupDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil // nothing to prune
		}
		return fmt.Errorf("angela: prune: stat: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("angela: prune: %s is not a directory", backupDir)
	}

	cutoff := currentBackupTime().Add(-time.Duration(retentionDays) * 24 * time.Hour)
	var walkErrs []string

	err = filepath.WalkDir(backupDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			walkErrs = append(walkErrs, fmt.Sprintf("%s: %v", path, walkErr))
			return nil
		}
		if d.IsDir() {
			return nil
		}
		m := backupTimestampRe.FindStringSubmatch(d.Name())
		if m == nil {
			return nil // not a backup file, leave alone
		}
		t, parseErr := time.Parse(BackupTimeFormat, m[1])
		if parseErr != nil {
			return nil
		}
		if t.Before(cutoff) {
			if rmErr := os.Remove(path); rmErr != nil {
				walkErrs = append(walkErrs, fmt.Sprintf("%s: %v", path, rmErr))
			}
		}
		return nil
	})
	// Combine the outer-walk error with any per-entry errors so diagnostics
	// from the partial prune are not lost when WalkDir itself bails.
	if err != nil {
		if len(walkErrs) > 0 {
			return fmt.Errorf("angela: prune: walk: %w (also: %s)", err, strings.Join(walkErrs, "; "))
		}
		return fmt.Errorf("angela: prune: walk: %w", err)
	}
	if len(walkErrs) > 0 {
		// Cap the list so one bad mount cannot blow up the error.
		if len(walkErrs) > 10 {
			walkErrs = append(walkErrs[:10], fmt.Sprintf("(and %d more)", len(walkErrs)-10))
		}
		return fmt.Errorf("angela: prune: %s", strings.Join(walkErrs, "; "))
	}
	return nil
}

// ListBackups returns every backup for the document at relPath, newest first.
// The relPath is interpreted the same way as in WriteBackup: relative to
// workDir, with the directory tree preserved inside the backup area.
//
// Returns an empty slice (not an error) when the backup directory does not
// exist or contains no matching files.
func ListBackups(stateDir, backupSubdir, relPath string) ([]BackupEntry, error) {
	if backupSubdir == "" {
		backupSubdir = "polish-backups"
	}
	if err := AssertContainedRelPath(backupSubdir); err != nil {
		return nil, fmt.Errorf("angela: list backups: backupSubdir: %w", err)
	}
	if err := AssertContainedRelPath(relPath); err != nil {
		return nil, fmt.Errorf("angela: list backups: relPath: %w", err)
	}
	cleanRel := filepath.Clean(relPath)

	base := filepath.Join(stateDir, backupSubdir, cleanRel)
	parent := filepath.Dir(base)
	prefix := filepath.Base(base) + "."

	entries, err := os.ReadDir(parent)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("angela: list backups: readdir: %w", err)
	}

	out := make([]BackupEntry, 0, len(entries))
	// Use exact regex matching instead of prefix-match to avoid listing
	// backups for sibling files (e.g. `intro.md.fr.md` when listing
	// backups for `intro.md`).
	baseName := filepath.Base(base)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		m := backupTimestampRe.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		// Exact-match guard. The regex captures the stamp; anything
		// between `baseName.` and the matched stamp suffix that is
		// non-empty means a different file's backup.
		suffix := name[len(baseName):] // starts with "."
		// Must begin with ".", then the captured stamp, then (optional)
		// "-<int>", then ".bak".
		if !strings.HasPrefix(suffix, ".") {
			continue
		}
		remainder := suffix[1:]
		if !strings.HasPrefix(remainder, m[1]) {
			continue
		}
		tail := remainder[len(m[1]):]
		// Equivalent to !(tail==suffix || (hasPrefix- && hasSuffix.bak)),
		// expanded for staticcheck QF1001 readability.
		if tail != BackupSuffix &&
			(!strings.HasPrefix(tail, "-") || !strings.HasSuffix(tail, BackupSuffix)) {
			continue
		}
		t, parseErr := time.Parse(BackupTimeFormat, m[1])
		if parseErr != nil {
			continue
		}
		out = append(out, BackupEntry{
			Path:      filepath.Join(parent, name),
			Timestamp: t,
			Stamp:     m[1],
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Timestamp.After(out[j].Timestamp)
	})
	return out, nil
}

// RestoreBackup copies the backup at backupPath back to the source document
// at <workDir>/<relPath>, atomically (tempfile + rename). Returns an error if
// the backup is missing or the rename fails. The caller is responsible for
// picking the right backup via ListBackups first.
func RestoreBackup(workDir, relPath, backupPath string) error {
	// Containment check on the destination relpath BEFORE opening the
	// backup. The caller (cmd layer) already validated the path, but
	// defense in depth is cheap.
	if err := AssertContainedRelPath(relPath); err != nil {
		return fmt.Errorf("angela: restore: %w", err)
	}
	// Reject symlink backup files (MEDIUM): same rationale as
	// WriteBackup — if someone planted a symlink in the backup area,
	// we would otherwise copy its target into the destination.
	if info, lerr := os.Lstat(backupPath); lerr == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("angela: restore: refusing to restore from symlink %s", backupPath)
	}
	src, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("angela: restore: open backup: %w", err)
	}
	defer func() { _ = src.Close() }()

	destPath := filepath.Join(workDir, filepath.Clean(relPath))
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("angela: restore: mkdir dest: %w", err)
	}
	// Delegate the tempfile+sync+rename+fsync dance to fileutil.AtomicWriteStream.
	if err := fileutil.AtomicWriteStream(destPath, src, 0o600); err != nil {
		return fmt.Errorf("angela: restore: %w", err)
	}
	return nil
}

// FindBackupByStamp scans ListBackups output for the first entry whose
// Stamp matches the exact string. Returns ("", false) when no match exists.
// Convenience helper for the `--timestamp` flag of the restore command.
func FindBackupByStamp(entries []BackupEntry, stamp string) (BackupEntry, bool) {
	for _, e := range entries {
		if e.Stamp == stamp {
			return e, true
		}
	}
	return BackupEntry{}, false
}
