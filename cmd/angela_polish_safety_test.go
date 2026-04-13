// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/testutil"
)

// --- test helpers -------------------------------------------------------

// safetyMockProvider implements domain.AIProvider with a caller-controlled
// Complete function. Story 8.6 tests use it via setPolishProviderFactory
// to drive the polish command deterministically without any network.
type safetyMockProvider struct {
	fn func(ctx context.Context, prompt string, opts ...domain.Option) (string, error)
}

func (m *safetyMockProvider) Complete(ctx context.Context, prompt string, opts ...domain.Option) (string, error) {
	return m.fn(ctx, prompt, opts...)
}

// setupPolishSafety prepares a lore-native working directory with one
// document under .lore/docs/, installs a fake provider that returns
// `polishedBody` as the AI's polished response, and returns a *config.Config
// with the defaults the polish command expects (provider non-empty so the
// nil check passes, DetectedMode = lore-native so ResolveStateDir lands
// inside .lore/angela/).
func setupPolishSafety(t *testing.T, filename, originalBody, polishedBody string) (*config.Config, string) {
	t.Helper()
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	docsDir := filepath.Join(dir, ".lore", "docs")
	if err := os.WriteFile(filepath.Join(docsDir, filename), []byte(originalBody), 0o644); err != nil {
		t.Fatalf("write source doc: %v", err)
	}

	cfg := &config.Config{}
	cfg.AI.Provider = "mock"
	cfg.DetectedMode = config.ModeLoreNative
	cfg.Angela.Polish.Backup.Enabled = true
	cfg.Angela.Polish.Backup.Path = "polish-backups"
	cfg.Angela.Polish.Backup.RetentionDays = 30

	restore := setPolishProviderFactory(func(_ *config.Config, _ domain.IOStreams) (domain.AIProvider, error) {
		return &safetyMockProvider{
			fn: func(ctx context.Context, prompt string, opts ...domain.Option) (string, error) {
				return polishedBody, nil
			},
		}, nil
	})
	t.Cleanup(restore)

	return cfg, dir
}

// backupRootFor returns the absolute path of the polish-backups directory
// for the current test's working dir. Matches what the command computes.
func backupRootFor(t *testing.T, cfg *config.Config, workDir string) string {
	t.Helper()
	stateDir := config.ResolveStateDir(workDir, cfg, cfg.DetectedMode)
	return filepath.Join(stateDir, cfg.Angela.Polish.Backup.Path)
}

// --- AC-10 Story 8.6 integration tests ----------------------------------

// TestPolishCmd_DryRunDoesNotWrite covers AC-1 + AC-8: --dry-run never
// modifies the source document on disk. We verify by comparing the mtime
// before and after the command, which is the stricter check than a
// content comparison (the latter wouldn't catch a "same-content rewrite").
func TestPolishCmd_DryRunDoesNotWrite(t *testing.T) {
	original := "---\ntype: decision\nstatus: published\ndate: \"2026-04-10\"\n---\n## Why\nOriginal reason.\n"
	polished := "---\ntype: decision\nstatus: published\ndate: \"2026-04-10\"\n---\n## Why\nPolished reason.\n"
	cfg, dir := setupPolishSafety(t, "dry-run-doc.md", original, polished)

	srcPath := filepath.Join(dir, ".lore", "docs", "dry-run-doc.md")
	stBefore, err := os.Stat(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	// Some filesystems round mtime to the nearest second — sleep a beat
	// so a no-op rewrite would still be detectable.
	time.Sleep(10 * time.Millisecond)

	streams, stdout, _ := testStreams()
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"polish", "--dry-run", "dry-run-doc.md"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("polish --dry-run: %v", err)
	}

	stAfter, err := os.Stat(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if !stAfter.ModTime().Equal(stBefore.ModTime()) {
		t.Errorf("source file mtime changed: before=%v after=%v", stBefore.ModTime(), stAfter.ModTime())
	}
	// And the content must still match the original byte-for-byte.
	got, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != original {
		t.Errorf("source content mutated on disk; got %q", string(got))
	}
	// Stdout should have something — the polished content.
	if stdout.Len() == 0 {
		t.Errorf("expected polished content on stdout, got empty")
	}
}

// TestPolishCmd_DryRunOutputIsPolished checks AC-1's stdout contract: the
// polished content goes to stdout so it can be piped into `diff`, `bat`,
// or a file redirect without any interactive prompts mixed in.
func TestPolishCmd_DryRunOutputIsPolished(t *testing.T) {
	original := "---\ntype: decision\nstatus: published\ndate: \"2026-04-10\"\n---\n## Why\nFirst draft.\n"
	polished := "---\ntype: decision\nstatus: published\ndate: \"2026-04-10\"\n---\n## Why\nSecond draft, much better.\n"
	cfg, _ := setupPolishSafety(t, "dry-run-out.md", original, polished)

	streams, stdout, stderr := testStreams()
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"polish", "--dry-run", "dry-run-out.md"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("polish --dry-run: %v", err)
	}

	// The stdout stream must contain the polished body so downstream
	// pipes can consume it as the document.
	if !strings.Contains(stdout.String(), "Second draft, much better.") {
		t.Errorf("stdout missing polished content; got %q", stdout.String())
	}
	// And stderr should contain a unified diff with a removed original
	// line and an added polished line. "## Why" is context, not a diff
	// target, so only the changed rows carry +/- markers.
	errOut := stderr.String()
	if !strings.Contains(errOut, "-First draft.") {
		t.Errorf("stderr missing original line removal; got %q", errOut)
	}
	if !strings.Contains(errOut, "+Second draft, much better.") {
		t.Errorf("stderr missing polished line addition; got %q", errOut)
	}
	// The standard unified diff headers must be present so downstream
	// tooling (bat, diff, etc.) can parse the output.
	if !strings.Contains(errOut, "--- dry-run-out.md (original)") {
		t.Errorf("stderr missing --- header; got %q", errOut)
	}
	if !strings.Contains(errOut, "+++ dry-run-out.md (polished)") {
		t.Errorf("stderr missing +++ header; got %q", errOut)
	}
}

// TestPolishCmd_CreatesBackupByDefault covers AC-2: an accepted polish run
// produces a timestamped backup file under the default state dir.
func TestPolishCmd_CreatesBackupByDefault(t *testing.T) {
	original := "---\ntype: decision\nstatus: published\ndate: \"2026-04-10\"\n---\n## Why\nOne reason.\n"
	polished := "---\ntype: decision\nstatus: published\ndate: \"2026-04-10\"\n---\n## Why\nAnother clearer reason.\n"
	cfg, dir := setupPolishSafety(t, "backup-default.md", original, polished)

	restore := angela.SetBackupClock(func() time.Time {
		return time.Date(2026, 4, 10, 14, 30, 22, 0, time.UTC)
	})
	t.Cleanup(restore)

	streams, _, _ := testStreams()
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"polish", "--yes", "backup-default.md"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("polish --yes: %v", err)
	}

	root := backupRootFor(t, cfg, dir)
	// Walk the backup root and look for any file whose name contains the
	// source's basename and our fixed stamp — the exact layout is an
	// implementation detail of WriteBackup, so we don't hard-code it here.
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if strings.HasPrefix(base, "backup-default.md.") && strings.Contains(base, "20260410T143022Z") && strings.HasSuffix(base, ".bak") {
			found = true
		}
		return nil
	})
	if !found {
		t.Errorf("expected backup file under %s, walk found none", root)
	}
}

// TestPolishCmd_BackupPreservesSubdirs exercises AC-4 at the unit level
// because the polish CLI itself validates filenames and rejects path
// separators. The goal of the story's test name is to prove the backup
// writer preserves nested paths — which is exactly what this check does.
func TestPolishCmd_BackupPreservesSubdirs(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "work")
	stateDir := filepath.Join(root, "state")
	if err := os.MkdirAll(filepath.Join(workDir, "docs", "guides"), 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(workDir, "docs", "guides", "intro.md")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Paranoid-review fix: UTC clock so the produced filename matches
	// the UTC BackupTimeFormat regardless of the runner's local zone.
	restore := angela.SetBackupClock(func() time.Time {
		return time.Date(2026, 4, 10, 14, 30, 22, 0, time.UTC)
	})
	defer restore()

	backupPath, err := angela.WriteBackup(workDir, stateDir, "polish-backups",
		filepath.Join("docs", "guides", "intro.md"))
	if err != nil {
		t.Fatalf("WriteBackup: %v", err)
	}

	wantSuffix := filepath.Join("polish-backups", "docs", "guides", "intro.md.20260410T143022Z.bak")
	if !strings.HasSuffix(backupPath, wantSuffix) {
		t.Errorf("backup path %q does not preserve subdirs (want suffix %q)", backupPath, wantSuffix)
	}
}

// TestPolishCmd_RestoreLatest walks the full polish → (simulated) unwanted
// edit → restore loop through the CLI. This is the scenario users hit when
// they want to undo a bad polish output.
func TestPolishCmd_RestoreLatest(t *testing.T) {
	original := "---\ntype: decision\nstatus: published\ndate: \"2026-04-10\"\n---\n## Why\nOriginal wisdom.\n"
	polished := "---\ntype: decision\nstatus: published\ndate: \"2026-04-10\"\n---\n## Why\nUpdated polished wisdom.\n"
	cfg, dir := setupPolishSafety(t, "restore-me.md", original, polished)

	fixed := time.Date(2026, 4, 10, 14, 30, 22, 0, time.UTC)
	restoreClock := angela.SetBackupClock(func() time.Time { return fixed })
	t.Cleanup(restoreClock)

	streams, _, _ := testStreams()
	// Run polish which creates a backup of the original.
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"polish", "--yes", "restore-me.md"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("polish --yes: %v", err)
	}

	// Simulate the user realising the polish was bad by overwriting the
	// document with garbage — then ask the CLI to restore.
	srcPath := filepath.Join(dir, ".lore", "docs", "restore-me.md")
	if err := os.WriteFile(srcPath, []byte("GARBAGE\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Fresh command tree so flag state from the previous invocation is
	// not carried over.
	streams2, _, _ := testStreams()
	cmd2 := newAngelaCmd(cfg, streams2)
	cmd2.SetArgs([]string{"polish", "restore", "restore-me.md"})
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("polish restore: %v", err)
	}

	got, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != original {
		t.Errorf("after restore, content = %q, want %q", string(got), original)
	}
}

// TestPolishCmd_RetentionPrunesOldBackups seeds a very old backup alongside
// a fresh polish run and verifies the old one is removed after the new
// backup lands. Matches AC-6's "pruning runs AFTER the new backup is
// written" safety ordering.
func TestPolishCmd_RetentionPrunesOldBackups(t *testing.T) {
	original := "---\ntype: decision\nstatus: published\ndate: \"2026-04-10\"\n---\n## Why\nA.\n"
	polished := "---\ntype: decision\nstatus: published\ndate: \"2026-04-10\"\n---\n## Why\nB.\n"
	cfg, dir := setupPolishSafety(t, "retention.md", original, polished)
	cfg.Angela.Polish.Backup.RetentionDays = 7

	// Seed an ancient backup directly into the backup area (60 days ago).
	root := backupRootFor(t, cfg, dir)
	if err := os.MkdirAll(filepath.Join(root, ".lore", "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	oldStamp := time.Date(2026, 2, 10, 9, 0, 0, 0, time.UTC).Format(angela.BackupTimeFormat)
	oldBackup := filepath.Join(root, ".lore", "docs", "retention.md."+oldStamp+".bak")
	if err := os.WriteFile(oldBackup, []byte("ancient"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Freeze "now" so the pruning cutoff is deterministic.
	fixed := time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)
	restoreClock := angela.SetBackupClock(func() time.Time { return fixed })
	t.Cleanup(restoreClock)

	streams, _, _ := testStreams()
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"polish", "--yes", "retention.md"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("polish --yes: %v", err)
	}

	if _, err := os.Stat(oldBackup); !os.IsNotExist(err) {
		t.Errorf("ancient backup should have been pruned, stat err = %v", err)
	}
}

// TestPolishCmd_BackupDisabled: when the user opts out via config,
// the polish command must not create any backup file — nothing under
// the backup root at all — and it must emit a warning on stderr the
// first time it happens (AC-7).
//
// Paranoid-review fix (HIGH test): the previous test admitted it could
// not assert on the warning because a package-level sync.Once made
// ordering matter across tests. The warning is now gated by a marker
// file under stateDir, so `t.TempDir()` gives each test a clean slate
// and we can observe the behavior directly.
func TestPolishCmd_BackupDisabled(t *testing.T) {
	original := "---\ntype: decision\nstatus: published\ndate: \"2026-04-10\"\n---\n## Why\nA.\n"
	polished := "---\ntype: decision\nstatus: published\ndate: \"2026-04-10\"\n---\n## Why\nB.\n"
	cfg, dir := setupPolishSafety(t, "no-backup.md", original, polished)
	cfg.Angela.Polish.Backup.Enabled = false

	// --- First run: warning must fire and ack marker must be written.
	streams1, _, stderr1 := testStreams()
	cmd1 := newAngelaCmd(cfg, streams1)
	cmd1.SetArgs([]string{"polish", "--yes", "no-backup.md"})
	if err := cmd1.Execute(); err != nil {
		t.Fatalf("first polish --yes: %v", err)
	}
	if !strings.Contains(stderr1.String(), "disabled") && !strings.Contains(stderr1.String(), "désactiv") {
		t.Errorf("first run: expected disabled-backup warning on stderr, got:\n%s", stderr1.String())
	}

	// --- Backup area must be free of .bak files on disk.
	root := backupRootFor(t, cfg, dir)
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".bak") {
			t.Errorf("backup disabled but found backup file at %s", path)
		}
		return nil
	})

	// Reset the source file so the second polish has something to diff.
	srcPath := filepath.Join(dir, ".lore", "docs", "no-backup.md")
	if err := os.WriteFile(srcPath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	// --- Second run: marker present → warning must NOT fire again.
	streams2, _, stderr2 := testStreams()
	cmd2 := newAngelaCmd(cfg, streams2)
	cmd2.SetArgs([]string{"polish", "--yes", "no-backup.md"})
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("second polish --yes: %v", err)
	}
	if strings.Contains(stderr2.String(), "disabled") || strings.Contains(stderr2.String(), "désactiv") {
		t.Errorf("second run: warning should be suppressed after ack, got:\n%s", stderr2.String())
	}
}
