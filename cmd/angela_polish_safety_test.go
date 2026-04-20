// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/storage"
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

// --- Story 8-21 / I28: corrupt-source refused before AI call ----------

// TestPolishCmd_RefusesCorruptSource_NoProviderCall asserts that when
// the source document has malformed YAML in its frontmatter, polish
// exits with a non-zero status BEFORE issuing any provider call. This
// is invariant I28: zero credits consumed on a broken source.
func TestPolishCmd_RefusesCorruptSource_NoProviderCall(t *testing.T) {
	// A malformed YAML payload: unclosed list inside the frontmatter.
	original := "---\ntype: [unclosed\n---\n## Why\nOriginal.\n"
	cfg, _ := setupPolishSafety(t, "corrupt-src.md", original, "polished body")

	// Replace the mock provider with one that fails if called. If the
	// gate works, this function is never invoked.
	var callCount int
	restore := setPolishProviderFactory(func(_ *config.Config, _ domain.IOStreams) (domain.AIProvider, error) {
		return &safetyMockProvider{
			fn: func(ctx context.Context, prompt string, opts ...domain.Option) (string, error) {
				callCount++
				return "should never reach here", nil
			},
		}, nil
	})
	t.Cleanup(restore)

	streams, _, stderr := testStreams()
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"polish", "corrupt-src.md"})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected polish to return an error on corrupt source")
	}
	if callCount != 0 {
		t.Errorf("provider was called %d time(s); I28 requires zero calls on corrupt source", callCount)
	}
	// Message should be neutral and point at `lore doctor`.
	errMsg := stderr.String()
	if !strings.Contains(errMsg, "cannot polish: source frontmatter is not valid YAML") {
		t.Errorf("stderr missing neutral corrupt-source message:\n%s", errMsg)
	}
	if !strings.Contains(errMsg, "lore doctor") {
		t.Errorf("stderr should point at `lore doctor`:\n%s", errMsg)
	}
}

// TestPolishCmd_InvalidArbitrateRule_Rejected asserts that a bogus
// --arbitrate-rule value is rejected cleanly with a clear error,
// before any file I/O or provider setup.
func TestPolishCmd_InvalidArbitrateRule_Rejected(t *testing.T) {
	cfg, _ := setupPolishSafety(t, "any.md", "---\ntype: note\ndate: \"2026-04-10\"\nstatus: draft\n---\nbody\n", "polished")

	streams, _, _ := testStreams()
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"polish", "--arbitrate-rule=bogus", "any.md"})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error for invalid --arbitrate-rule")
	}
	if !strings.Contains(err.Error(), "invalid --arbitrate-rule") {
		t.Errorf("error should mention invalid rule; got: %v", err)
	}
}

// --- Story 8-21 / Task 6.b: sanitize + arbitrate in cmd integration ---

// TestPolishCmd_DuplicateSection_NonTTYRefusesWithoutRule asserts
// invariant I27: when the AI response contains duplicate sections and
// stdin is not a TTY and no --arbitrate-rule is set, polish refuses
// with a neutral message rather than silently de-duplicating.
func TestPolishCmd_DuplicateSection_NonTTYRefusesWithoutRule(t *testing.T) {
	original := "---\ntype: decision\nstatus: published\ndate: \"2026-04-10\"\n---\n## Why\nOriginal.\n"
	// Mock AI returns body with a duplicate `## Why`.
	polished := "## Why\nFirst version.\n## Why\nSecond version.\n"
	cfg, dir := setupPolishSafety(t, "dup-refuse.md", original, polished)

	srcPath := filepath.Join(dir, ".lore", "docs", "dup-refuse.md")
	bytesBefore, _ := os.ReadFile(srcPath)

	streams, _, stderr := testStreams()
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"polish", "--yes", "dup-refuse.md"})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected polish to refuse duplicate sections in non-TTY mode")
	}
	// Source must be bit-for-bit unchanged (I29).
	bytesAfter, _ := os.ReadFile(srcPath)
	if string(bytesAfter) != string(bytesBefore) {
		t.Errorf("source mutated despite refusal")
	}
	// Stderr should mention the duplicate issue and point at the flag.
	// Story 8-21 P0-3: message now enriched with the group count and
	// a per-heading breakdown. Accept either the pre- or post-enrichment
	// wording to keep the assertion robust against later polishing.
	errMsg := stderr.String()
	if !strings.Contains(errMsg, "duplicate section") {
		t.Errorf("stderr should mention duplicate section(s):\n%s", errMsg)
	}
	if !strings.Contains(errMsg, "--arbitrate-rule") {
		t.Errorf("stderr should mention --arbitrate-rule option:\n%s", errMsg)
	}
}

// TestPolishCmd_DuplicateSection_ArbitrateRuleFirst_KeepsFirst asserts
// that --arbitrate-rule=first resolves duplicates deterministically
// and the polish proceeds to write.
func TestPolishCmd_DuplicateSection_ArbitrateRuleFirst_KeepsFirst(t *testing.T) {
	original := "---\ntype: decision\nstatus: published\ndate: \"2026-04-10\"\n---\n## Why\nOriginal reason.\n"
	polished := "## Why\nFirst version.\n## Why\nSecond version.\n"
	cfg, dir := setupPolishSafety(t, "dup-first.md", original, polished)

	streams, _, _ := testStreams()
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"polish", "--yes", "--arbitrate-rule=first", "dup-first.md"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("polish: %v", err)
	}

	srcPath := filepath.Join(dir, ".lore", "docs", "dup-first.md")
	written, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	// Must contain the first version only; the second must be gone.
	if !strings.Contains(string(written), "First version.") {
		t.Errorf("written file missing first version:\n%s", string(written))
	}
	if strings.Contains(string(written), "Second version.") {
		t.Errorf("written file still contains second version (arbitration failed):\n%s", string(written))
	}
}

// TestPolishCmd_LeakedFrontmatter_SilentByDefault asserts that when
// the AI cheats and emits a full document (frontmatter + body), the
// extra `---` block is stripped silently by default — no "stripped"
// message on stderr (invariant I26).
func TestPolishCmd_LeakedFrontmatter_SilentByDefault(t *testing.T) {
	original := "---\ntype: decision\nstatus: published\ndate: \"2026-04-10\"\n---\n## Why\nOriginal.\n"
	// AI cheated: emitted `---\n...` block on top of its body.
	polished := "---\nid: 999\ntype: decision\nstatus: published\ndate: \"2026-04-10\"\n---\n## Why\nPolished body.\n"
	cfg, _ := setupPolishSafety(t, "leaked.md", original, polished)

	streams, _, stderr := testStreams()
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"polish", "--yes", "leaked.md"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("polish: %v", err)
	}
	// Without -v the "stripped leaked frontmatter" message must not appear.
	if strings.Contains(stderr.String(), "stripped leaked frontmatter") {
		t.Errorf("silent-by-default violated; stderr contains strip message:\n%s", stderr.String())
	}
}

// TestPolishCmd_LeakedFrontmatter_VerboseShowsStderr asserts that with
// -v / --verbose, the otherwise-silent strip event surfaces as a
// single stderr line.
func TestPolishCmd_LeakedFrontmatter_VerboseShowsStderr(t *testing.T) {
	original := "---\ntype: decision\nstatus: published\ndate: \"2026-04-10\"\n---\n## Why\nOriginal.\n"
	polished := "---\nid: 999\ntype: decision\nstatus: published\ndate: \"2026-04-10\"\n---\n## Why\nPolished body.\n"
	cfg, _ := setupPolishSafety(t, "leaked-verbose.md", original, polished)

	streams, _, stderr := testStreams()
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"polish", "--yes", "-v", "leaked-verbose.md"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("polish: %v", err)
	}
	if !strings.Contains(stderr.String(), "stripped leaked frontmatter") {
		t.Errorf("verbose should surface the strip message; stderr:\n%s", stderr.String())
	}
}

// TestPolishCmd_LogWrittenOnSuccessfulPolish asserts invariant I30:
// a successful polish run appends one LogResultWritten line to the
// polish.log audit trail.
func TestPolishCmd_LogWrittenOnSuccessfulPolish(t *testing.T) {
	original := "---\ntype: decision\nstatus: published\ndate: \"2026-04-10\"\n---\n## Why\nOriginal.\n"
	polished := "## Why\nPolished body.\n"
	cfg, dir := setupPolishSafety(t, "log-ok.md", original, polished)

	streams, _, _ := testStreams()
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"polish", "--yes", "log-ok.md"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("polish: %v", err)
	}

	stateDir := config.ResolveStateDir(dir, cfg, cfg.DetectedMode)
	entries, err := angela.ReadLogEntries(stateDir)
	if err != nil {
		t.Fatalf("ReadLogEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Result != angela.LogResultWritten {
		t.Errorf("Result=%q, want %q", e.Result, angela.LogResultWritten)
	}
	if e.File != "log-ok.md" {
		t.Errorf("File=%q, want 'log-ok.md'", e.File)
	}
	if e.Exit != 0 {
		t.Errorf("Exit=%d, want 0", e.Exit)
	}
	if e.Op != "polish" {
		t.Errorf("Op=%q, want 'polish'", e.Op)
	}
}

// TestPolishCmd_LogOnCorruptSource asserts that a refused-on-corrupt
// source run still writes a LogResultAbortedCorruptSrc entry (I30 +
// I28 combined: observability of refused runs).
func TestPolishCmd_LogOnCorruptSource(t *testing.T) {
	original := "---\ntype: [broken\n---\n## Why\nOriginal.\n"
	cfg, dir := setupPolishSafety(t, "log-corrupt.md", original, "any")

	streams, _, _ := testStreams()
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"polish", "log-corrupt.md"})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected error on corrupt source")
	}

	stateDir := config.ResolveStateDir(dir, cfg, cfg.DetectedMode)
	entries, err := angela.ReadLogEntries(stateDir)
	if err != nil {
		t.Fatalf("ReadLogEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	if entries[0].Result != angela.LogResultAbortedCorruptSrc {
		t.Errorf("Result=%q, want %q", entries[0].Result, angela.LogResultAbortedCorruptSrc)
	}
	if entries[0].Exit != 1 {
		t.Errorf("Exit=%d, want 1", entries[0].Exit)
	}
}

// TestPolishCmd_LogOnArbitrationAbort asserts LogResultAbortedArbitrate
// is recorded when --arbitrate-rule=abort triggers, AND that the
// findings include the detected duplicate group (I30).
func TestPolishCmd_LogOnArbitrationAbort(t *testing.T) {
	original := "---\ntype: decision\nstatus: published\ndate: \"2026-04-10\"\n---\n## Why\nOriginal.\n"
	polished := "## Why\nFirst.\n## Why\nSecond.\n"
	cfg, dir := setupPolishSafety(t, "log-abort.md", original, polished)

	streams, _, _ := testStreams()
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"polish", "--yes", "--arbitrate-rule=abort", "log-abort.md"})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected abort to return error")
	}

	stateDir := config.ResolveStateDir(dir, cfg, cfg.DetectedMode)
	entries, err := angela.ReadLogEntries(stateDir)
	if err != nil {
		t.Fatalf("ReadLogEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Result != angela.LogResultAbortedArbitrate {
		t.Errorf("Result=%q, want %q", e.Result, angela.LogResultAbortedArbitrate)
	}
	if len(e.Findings.Duplicates) != 1 {
		t.Fatalf("expected 1 duplicate finding, got %d", len(e.Findings.Duplicates))
	}
	if e.Findings.Duplicates[0].Heading != "## Why" {
		t.Errorf("duplicate Heading=%q, want '## Why'", e.Findings.Duplicates[0].Heading)
	}
	if e.Findings.Duplicates[0].Resolution != "rule:abort" {
		t.Errorf("Resolution=%q, want 'rule:abort'", e.Findings.Duplicates[0].Resolution)
	}
}

// TestPolishCmd_DryRun_NoLogWrite asserts AC-14: dry-run never writes
// to polish.log (zero side-effect).
func TestPolishCmd_DryRun_NoLogWrite(t *testing.T) {
	original := "---\ntype: decision\nstatus: published\ndate: \"2026-04-10\"\n---\n## Why\nOriginal.\n"
	polished := "## Why\nPolished.\n"
	cfg, dir := setupPolishSafety(t, "log-dryrun.md", original, polished)

	streams, _, _ := testStreams()
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"polish", "--dry-run", "log-dryrun.md"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("polish --dry-run: %v", err)
	}

	stateDir := config.ResolveStateDir(dir, cfg, cfg.DetectedMode)
	entries, err := angela.ReadLogEntries(stateDir)
	if err != nil {
		t.Fatalf("ReadLogEntries: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("dry-run wrote %d log entries; AC-14 requires zero", len(entries))
	}
}

// TestPolishCmd_ArbitrateRuleSecond_KeepsSecond verifies the `second`
// value of --arbitrate-rule at the cmd layer.
func TestPolishCmd_ArbitrateRuleSecond_KeepsSecond(t *testing.T) {
	original := "---\ntype: decision\nstatus: published\ndate: \"2026-04-10\"\n---\n## Why\nOriginal.\n"
	polished := "## Why\nFirst version.\n## Why\nSecond version.\n"
	cfg, dir := setupPolishSafety(t, "arb-second.md", original, polished)

	streams, _, _ := testStreams()
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"polish", "--yes", "--arbitrate-rule=second", "arb-second.md"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("polish: %v", err)
	}
	written, _ := os.ReadFile(filepath.Join(dir, ".lore", "docs", "arb-second.md"))
	if !strings.Contains(string(written), "Second version.") || strings.Contains(string(written), "First version.") {
		t.Errorf("expected only second version written; got:\n%s", string(written))
	}
}

// TestPolishCmd_ArbitrateRuleBoth_ConcatsInSourceOrder verifies the
// `both` value keeps every duplicate in source order.
func TestPolishCmd_ArbitrateRuleBoth_ConcatsInSourceOrder(t *testing.T) {
	original := "---\ntype: decision\nstatus: published\ndate: \"2026-04-10\"\n---\n## Why\nOriginal.\n"
	polished := "## Why\nFirst.\n## Why\nSecond.\n## Why\nThird.\n"
	cfg, dir := setupPolishSafety(t, "arb-both.md", original, polished)

	streams, _, _ := testStreams()
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"polish", "--yes", "--arbitrate-rule=both", "arb-both.md"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("polish: %v", err)
	}
	written, _ := os.ReadFile(filepath.Join(dir, ".lore", "docs", "arb-both.md"))
	body := string(written)
	// All three occurrences present in source order.
	idxFirst := strings.Index(body, "First.")
	idxSecond := strings.Index(body, "Second.")
	idxThird := strings.Index(body, "Third.")
	if idxFirst < 0 || idxSecond < 0 || idxThird < 0 {
		t.Fatalf("all three occurrences expected; body:\n%s", body)
	}
	if !(idxFirst < idxSecond && idxSecond < idxThird) {
		t.Errorf("source order not preserved: first@%d second@%d third@%d", idxFirst, idxSecond, idxThird)
	}
}

// TestPolishCmd_ArbitrateRuleMutuallyExclusiveWithInteractive asserts
// that cobra's mutual-exclusion declaration fires — the user cannot
// combine --arbitrate-rule and --interactive on the same invocation.
func TestPolishCmd_ArbitrateRuleMutuallyExclusiveWithInteractive(t *testing.T) {
	cfg, _ := setupPolishSafety(t, "mx.md", "---\ntype: note\ndate: \"2026-04-10\"\nstatus: draft\n---\nbody\n", "polished")

	streams, _, _ := testStreams()
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"polish", "--arbitrate-rule=first", "--interactive", "mx.md"})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected mutual-exclusion error")
	}
	// Cobra's phrasing: "if any flags in the group [...] are set none
	// of the others can be". We check for both flag names to be
	// robust to future cobra wording changes.
	msg := err.Error()
	if !strings.Contains(msg, "arbitrate-rule") || !strings.Contains(msg, "interactive") {
		t.Errorf("error should reference both mutually-exclusive flags; got: %v", err)
	}
}

// TestPolishCmd_LogOnAIError asserts that an AI provider error path
// writes a LogResultAIError entry, completing the I30 coverage of
// terminal states.
func TestPolishCmd_LogOnAIError(t *testing.T) {
	original := "---\ntype: decision\nstatus: published\ndate: \"2026-04-10\"\n---\n## Why\nOriginal.\n"
	cfg, dir := setupPolishSafety(t, "log-ai-err.md", original, "never used")
	// Override the factory to return a provider that always fails.
	restore := setPolishProviderFactory(func(_ *config.Config, _ domain.IOStreams) (domain.AIProvider, error) {
		return &safetyMockProvider{
			fn: func(ctx context.Context, prompt string, opts ...domain.Option) (string, error) {
				return "", errors.New("provider boom")
			},
		}, nil
	})
	t.Cleanup(restore)

	streams, _, _ := testStreams()
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"polish", "--yes", "log-ai-err.md"})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected error from provider failure")
	}

	stateDir := config.ResolveStateDir(dir, cfg, cfg.DetectedMode)
	entries, err := angela.ReadLogEntries(stateDir)
	if err != nil {
		t.Fatalf("ReadLogEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	if entries[0].Result != angela.LogResultAIError {
		t.Errorf("Result=%q, want %q", entries[0].Result, angela.LogResultAIError)
	}
	if entries[0].Exit != 1 {
		t.Errorf("Exit=%d, want 1", entries[0].Exit)
	}
}

// TestPolishCmd_FrontmatterBytesIdentical is the cornerstone I24 test:
// after a polish that writes the file, the frontmatter region of the
// written doc must be byte-for-byte identical to the source's. No
// YAML re-serialization, no key reordering, no quote-style change.
func TestPolishCmd_FrontmatterBytesIdentical(t *testing.T) {
	// Hand-crafted source with key-ordering quirks that yaml.Marshal
	// would mutate: a comment, single-quoted and double-quoted values
	// mixed, and a non-standard custom field.
	original := "---\n" +
		"type: decision\n" +
		"date: \"2026-04-10\"\n" +
		"# explicit status, do not touch\n" +
		"status: published\n" +
		"custom: 'single-quoted value'\n" +
		"---\n" +
		"## Why\nOriginal reason.\n"
	// Mock AI returns a cleaner body — no leak, no duplicates.
	polished := "## Why\nPolished reason, crisper and shorter.\n"
	cfg, dir := setupPolishSafety(t, "i24.md", original, polished)

	// Capture the source's fm bytes before polish.
	srcPath := filepath.Join(dir, ".lore", "docs", "i24.md")
	srcBytes, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("read src: %v", err)
	}
	srcFM, _, err := storage.ExtractFrontmatter(srcBytes)
	if err != nil {
		t.Fatalf("extract src FM: %v", err)
	}

	streams, _, _ := testStreams()
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"polish", "--yes", "i24.md"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("polish: %v", err)
	}

	// Read back and extract FM.
	writtenBytes, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("read written: %v", err)
	}
	writtenFM, _, err := storage.ExtractFrontmatter(writtenBytes)
	if err != nil {
		t.Fatalf("extract written FM: %v", err)
	}
	if !bytes.Equal(srcFM, writtenFM) {
		t.Errorf("I24 violation — FM bytes drifted\nsrc: %q\nwritten: %q", string(srcFM), string(writtenFM))
	}
	// Body should have been replaced with the polished version.
	if !strings.Contains(string(writtenBytes), "Polished reason, crisper") {
		t.Errorf("written body missing polished content:\n%s", string(writtenBytes))
	}
}

// TestPolishCmd_ArbitrateRuleAbort_NoWriteNoBackup asserts invariant
// I29: --arbitrate-rule=abort exits cleanly without writing and
// without consuming a backup slot.
func TestPolishCmd_ArbitrateRuleAbort_NoWriteNoBackup(t *testing.T) {
	original := "---\ntype: decision\nstatus: published\ndate: \"2026-04-10\"\n---\n## Why\nOriginal.\n"
	polished := "## Why\nFirst.\n## Why\nSecond.\n" // duplicates trigger arbitration
	cfg, dir := setupPolishSafety(t, "arb-abort.md", original, polished)

	srcPath := filepath.Join(dir, ".lore", "docs", "arb-abort.md")
	bytesBefore, _ := os.ReadFile(srcPath)

	streams, _, _ := testStreams()
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"polish", "--yes", "--arbitrate-rule=abort", "arb-abort.md"})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected abort to return an error")
	}
	// Source unchanged.
	bytesAfter, _ := os.ReadFile(srcPath)
	if string(bytesAfter) != string(bytesBefore) {
		t.Errorf("source mutated on abort")
	}
	// No backup directory should exist (or if it does, be empty).
	root := backupRootFor(t, cfg, dir)
	_, statErr := os.Stat(root)
	if statErr == nil {
		// Directory exists — assert it has no entries.
		entries, _ := os.ReadDir(root)
		if len(entries) > 0 {
			t.Errorf("backup dir has %d entries on abort; expected 0", len(entries))
		}
	}
}

// TestPolishCmd_ArbitrateRuleAbort_StderrMentionsDuplicates asserts the
// Story 8-21 P0-3 fix: the abort message must name the root cause
// (how many duplicate section groups were detected) so a CI operator
// can diagnose without opening polish.log. The message must stay
// neutral — never framing the situation as "wasted credits" or the
// AI being "corrupted" (user feedback: ai_corruption_handling memory).
func TestPolishCmd_ArbitrateRuleAbort_StderrMentionsDuplicates(t *testing.T) {
	original := "---\ntype: decision\nstatus: published\ndate: \"2026-04-10\"\n---\n## Why\nOriginal.\n"
	polished := "## Why\nFirst.\n## How\nH1.\n## How\nH2.\n## Why\nSecond.\n"
	cfg, _ := setupPolishSafety(t, "arb-abort-msg.md", original, polished)

	streams, _, errBuf := testStreams()
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"polish", "--yes", "--arbitrate-rule=abort", "arb-abort-msg.md"})
	_ = cmd.Execute()

	stderr := errBuf.String()

	// Must mention duplicate section group count.
	if !strings.Contains(stderr, "duplicate section group") {
		t.Errorf("stderr must mention duplicate section group(s), got:\n%s", stderr)
	}
	// Must cite the "abort" rule explicitly so the operator knows the
	// rule took effect rather than an unrelated failure.
	if !strings.Contains(stderr, "abort") {
		t.Errorf("stderr must cite the abort rule, got:\n%s", stderr)
	}
	// Must stay neutral — reject any framing that blames the AI or
	// wastes credits.
	forbidden := []string{"wasted", "credit", "corrupt"}
	for _, w := range forbidden {
		if strings.Contains(strings.ToLower(stderr), w) {
			t.Errorf("stderr contains forbidden framing %q:\n%s", w, stderr)
		}
	}
	// Must list each duplicate heading with its count so the operator
	// sees WHAT is duplicated, not just how many groups.
	if !strings.Contains(stderr, `"## Why" × 2`) {
		t.Errorf("stderr must list the duplicate headings with counts, got:\n%s", stderr)
	}
}
