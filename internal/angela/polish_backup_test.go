// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newBackupTestDir returns a clean pair (workDir, stateDir) under a t.TempDir
// so every test operates in isolation and is cleaned up automatically.
func newBackupTestDir(t *testing.T) (workDir, stateDir string) {
	t.Helper()
	root := t.TempDir()
	workDir = filepath.Join(root, "work")
	stateDir = filepath.Join(root, "state")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("mkdir workDir: %v", err)
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir stateDir: %v", err)
	}
	return workDir, stateDir
}

// writeFile is a tiny convenience that also returns the path so the tests
// read top-to-bottom instead of juggling intermediate variables.
func writeFile(t *testing.T, path, content string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// TestWriteBackup_CreatesFileWithExpectedShape verifies that a flat filename
// produces a backup under <stateDir>/<subdir>/<filename>.<stamp>.bak and
// that the content matches the source.
func TestWriteBackup_CreatesFileWithExpectedShape(t *testing.T) {
	workDir, stateDir := newBackupTestDir(t)
	writeFile(t, filepath.Join(workDir, "foo.md"), "hello")

	fixed := time.Date(2026, 4, 10, 14, 30, 22, 0, time.UTC)
	restore := SetBackupClock(func() time.Time { return fixed })
	defer restore()

	backupPath, err := WriteBackup(workDir, stateDir, "polish-backups", "foo.md")
	if err != nil {
		t.Fatalf("WriteBackup: %v", err)
	}

	wantSuffix := filepath.Join("polish-backups", "foo.md.20260410T143022Z.bak")
	if !strings.HasSuffix(backupPath, wantSuffix) {
		t.Errorf("backupPath = %q, want suffix %q", backupPath, wantSuffix)
	}
	got, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("backup content = %q, want %q", got, "hello")
	}
}

// TestWriteBackup_EmptySubdirFallsBack mirrors the behavior callers get when
// they forgot to set cfg.Angela.Polish.Backup.Path (zero value).
func TestWriteBackup_EmptySubdirFallsBack(t *testing.T) {
	workDir, stateDir := newBackupTestDir(t)
	writeFile(t, filepath.Join(workDir, "x.md"), "x")

	p, err := WriteBackup(workDir, stateDir, "", "x.md")
	if err != nil {
		t.Fatalf("WriteBackup: %v", err)
	}
	if !strings.Contains(p, "polish-backups") {
		t.Errorf("empty subdir should fall back to polish-backups, got %q", p)
	}
}

// TestWriteBackup_PreservesSubdirs covers AC-4: nested source paths keep
// their relative tree inside the backup area so two files with the same
// basename never collide.
func TestWriteBackup_PreservesSubdirs(t *testing.T) {
	workDir, stateDir := newBackupTestDir(t)
	rel := filepath.Join("docs", "guides", "foo.md")
	writeFile(t, filepath.Join(workDir, rel), "guide content")

	restore := SetBackupClock(func() time.Time {
		return time.Date(2026, 4, 10, 14, 30, 22, 0, time.UTC)
	})
	defer restore()

	backupPath, err := WriteBackup(workDir, stateDir, "polish-backups", rel)
	if err != nil {
		t.Fatalf("WriteBackup: %v", err)
	}
	wantSuffix := filepath.Join("polish-backups", "docs", "guides", "foo.md.20260410T143022Z.bak")
	if !strings.HasSuffix(backupPath, wantSuffix) {
		t.Errorf("backupPath = %q, want suffix %q", backupPath, wantSuffix)
	}
	if _, err := os.Stat(backupPath); err != nil {
		t.Errorf("backup file missing: %v", err)
	}
}

// TestWriteBackup_RejectsPathEscape ensures the relPath guard blocks
// attempts to write outside the backup area via `..`.
func TestWriteBackup_RejectsPathEscape(t *testing.T) {
	workDir, stateDir := newBackupTestDir(t)
	if _, err := WriteBackup(workDir, stateDir, "polish-backups", "../evil.md"); err == nil {
		t.Fatal("expected error for escaping relPath")
	}
}

// TestWriteBackup_MissingSource surfaces a clear error (rather than writing
// a zero-byte backup) when the source document is gone.
func TestWriteBackup_MissingSource(t *testing.T) {
	workDir, stateDir := newBackupTestDir(t)
	if _, err := WriteBackup(workDir, stateDir, "polish-backups", "ghost.md"); err == nil {
		t.Fatal("expected error for missing source")
	}
}

// TestPruneOldBackups_RemovesExpired exercises the retention cutoff logic.
// Files older than retentionDays must be deleted; anything newer or
// unrelated (stray README.md) must be kept.
func TestPruneOldBackups_RemovesExpired(t *testing.T) {
	_, stateDir := newBackupTestDir(t)
	root := filepath.Join(stateDir, "polish-backups")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	// 10 days old — should be removed (retention = 7 days).
	oldStamp := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC).Format(BackupTimeFormat)
	oldPath := filepath.Join(root, "foo.md."+oldStamp+".bak")
	writeFile(t, oldPath, "old")

	// 1 day old — should survive.
	newStamp := time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC).Format(BackupTimeFormat)
	newPath := filepath.Join(root, "foo.md."+newStamp+".bak")
	writeFile(t, newPath, "new")

	// Unrelated file that mimics the .bak suffix without a valid stamp.
	strayPath := filepath.Join(root, "README.md")
	writeFile(t, strayPath, "stray")

	restore := SetBackupClock(func() time.Time {
		return time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC)
	})
	defer restore()

	if err := PruneOldBackups(root, 7); err != nil {
		t.Fatalf("PruneOldBackups: %v", err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("old backup should be removed, stat err = %v", err)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("recent backup should survive: %v", err)
	}
	if _, err := os.Stat(strayPath); err != nil {
		t.Errorf("unrelated file should be untouched: %v", err)
	}
}

// TestPruneOldBackups_ZeroMeansForever matches the config semantics: a
// RetentionDays of 0 disables pruning entirely.
func TestPruneOldBackups_ZeroMeansForever(t *testing.T) {
	_, stateDir := newBackupTestDir(t)
	root := filepath.Join(stateDir, "polish-backups")
	stamp := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Format(BackupTimeFormat)
	p := filepath.Join(root, "foo.md."+stamp+".bak")
	writeFile(t, p, "ancient")

	if err := PruneOldBackups(root, 0); err != nil {
		t.Fatalf("PruneOldBackups: %v", err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Errorf("retention=0 should keep everything, stat err = %v", err)
	}
}

// TestPruneOldBackups_MissingDirIsNoop confirms pruning a non-existent
// directory is silent rather than an error — matches "nothing to prune".
func TestPruneOldBackups_MissingDirIsNoop(t *testing.T) {
	_, stateDir := newBackupTestDir(t)
	if err := PruneOldBackups(filepath.Join(stateDir, "does-not-exist"), 7); err != nil {
		t.Fatalf("PruneOldBackups on missing dir: %v", err)
	}
}

// TestListBackups_SortedNewestFirst confirms ListBackups returns entries in
// descending timestamp order, matching what the restore command expects.
func TestListBackups_SortedNewestFirst(t *testing.T) {
	workDir, stateDir := newBackupTestDir(t)
	writeFile(t, filepath.Join(workDir, "foo.md"), "v0")

	// Three backups at different times for the same source file.
	times := []time.Time{
		time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC),
	}
	for _, ts := range times {
		restore := SetBackupClock(func() time.Time { return ts })
		if _, err := WriteBackup(workDir, stateDir, "polish-backups", "foo.md"); err != nil {
			restore()
			t.Fatalf("WriteBackup: %v", err)
		}
		restore()
	}

	entries, err := ListBackups(stateDir, "polish-backups", "foo.md")
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("len(entries) = %d, want 3", len(entries))
	}
	// Sorted newest first?
	for i := 0; i < len(entries)-1; i++ {
		if entries[i].Timestamp.Before(entries[i+1].Timestamp) {
			t.Errorf("entries not sorted desc: %v before %v", entries[i].Timestamp, entries[i+1].Timestamp)
		}
	}
}

// TestListBackups_EmptyDirReturnsEmpty: no backup dir = no entries, no error.
func TestListBackups_EmptyDirReturnsEmpty(t *testing.T) {
	_, stateDir := newBackupTestDir(t)
	entries, err := ListBackups(stateDir, "polish-backups", "foo.md")
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("len(entries) = %d, want 0", len(entries))
	}
}

// TestListBackups_FiltersByBasename verifies that a backup for "bar.md"
// does not show up when we ask for "foo.md" backups — the prefix match
// must be strict enough to avoid cross-contamination.
func TestListBackups_FiltersByBasename(t *testing.T) {
	workDir, stateDir := newBackupTestDir(t)
	writeFile(t, filepath.Join(workDir, "foo.md"), "a")
	writeFile(t, filepath.Join(workDir, "bar.md"), "b")

	restore := SetBackupClock(func() time.Time {
		return time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)
	})
	defer restore()

	if _, err := WriteBackup(workDir, stateDir, "polish-backups", "foo.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteBackup(workDir, stateDir, "polish-backups", "bar.md"); err != nil {
		t.Fatal(err)
	}

	fooEntries, err := ListBackups(stateDir, "polish-backups", "foo.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(fooEntries) != 1 {
		t.Errorf("foo entries = %d, want 1", len(fooEntries))
	}
	for _, e := range fooEntries {
		if !strings.Contains(filepath.Base(e.Path), "foo.md.") {
			t.Errorf("unexpected entry for foo: %s", e.Path)
		}
	}
}

// TestRestoreBackup_RoundTrip is the most important test: modify the file,
// restore, verify the content matches the original that was backed up.
// Matches the story's `TestPolishCmd_RestoreLatest` scenario at the unit
// level (the CLI test covers the cobra wiring).
func TestRestoreBackup_RoundTrip(t *testing.T) {
	workDir, stateDir := newBackupTestDir(t)
	srcPath := filepath.Join(workDir, "foo.md")
	writeFile(t, srcPath, "original content")

	restore := SetBackupClock(func() time.Time {
		return time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)
	})
	defer restore()

	backupPath, err := WriteBackup(workDir, stateDir, "polish-backups", "foo.md")
	if err != nil {
		t.Fatal(err)
	}
	// Simulate a polish that overwrote the file with something unwanted.
	if err := os.WriteFile(srcPath, []byte("garbage from AI"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := RestoreBackup(workDir, "foo.md", backupPath); err != nil {
		t.Fatalf("RestoreBackup: %v", err)
	}
	got, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "original content" {
		t.Errorf("after restore, content = %q, want %q", got, "original content")
	}
}

// TestFindBackupByStamp_HitAndMiss covers both the happy path and the
// `--timestamp unknown` error path used by the restore subcommand.
// S2-L3: the second fixture uses the old pre-UTC format (no trailing Z).
// Kept intentionally to verify FindBackupByStamp matches exact stamps
// regardless of format; production stamps always have the Z suffix per
// BackupTimeFormat.
func TestFindBackupByStamp_HitAndMiss(t *testing.T) {
	entries := []BackupEntry{
		{Stamp: "20260410T143022Z"},
		{Stamp: "20260401T100000"},
	}
	if _, ok := FindBackupByStamp(entries, "20260410T143022Z"); !ok {
		t.Error("expected hit for existing stamp")
	}
	if _, ok := FindBackupByStamp(entries, "19990101T000000"); ok {
		t.Error("expected miss for unknown stamp")
	}
}
