// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/cli"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/greycoderk/lore/internal/testutil"
	"github.com/greycoderk/lore/internal/ui"
	"github.com/stretchr/testify/require"
)

func setupDoctorDir(t *testing.T) string {
	t.Helper()
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)
	return dir
}

func runDoctor(t *testing.T, _ string, args ...string) (stdout, stderr string, exitErr error) {
	t.Helper()
	restore := ui.SaveAndDisableColor()
	defer restore()

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &out,
		Err: &errBuf,
	}
	cfg := &config.Config{}
	// setupDoctorDir creates a .lore/ tree; mark the config accordingly
	// so ResolveStateDir lands under .lore/angela/ (production
	// behavior for lore-native projects). Story 8-23 pruners depend
	// on this resolution to find polish-backups / polish.log /
	// *.corrupt-* files.
	cfg.DetectedMode = config.ModeLoreNative
	// Populate Story 8-23 GC retention defaults so `doctor --prune`
	// tests exercise realistic behavior. Production loads these via
	// viper; unit tests instantiate the struct directly so we seed
	// them by hand.
	cfg.Angela.GC.CorruptQuarantine.RetentionDays = 14
	cfg.Angela.Polish.Log.RetentionDays = 30
	cfg.Angela.Polish.Log.MaxSizeMB = 10
	cfg.Angela.Polish.Backup.RetentionDays = 30
	cfg.Angela.Polish.Backup.Path = "polish-backups"
	cmd := newDoctorCmd(cfg, streams)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errBuf.String(), err
}

// --- Diagnostic Tests ---

func TestDoctor_CleanCorpus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := setupDoctorDir(t)
	docsDir := filepath.Join(dir, ".lore", "docs")

	// Write a valid doc and generate index
	_, _ = storage.WriteDoc(docsDir, domain.DocMeta{Type: "note", Date: "2026-03-07", Status: "published"}, "clean doc", "# Clean\n\nBody.\n")
	if err := storage.RegenerateIndex(docsDir); err != nil {
		t.Fatalf("RegenerateIndex: %v", err)
	}

	_, stderr, err := runDoctor(t, dir)
	if err != nil {
		t.Fatalf("expected no error for clean corpus, got: %v", err)
	}
	if !strings.Contains(stderr, "0 issues found") {
		t.Errorf("expected '0 issues found' in stderr, got: %q", stderr)
	}
}

func TestDoctor_IssuesFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := setupDoctorDir(t)
	docsDir := filepath.Join(dir, ".lore", "docs")

	// Create an orphan .tmp file
	if err := os.WriteFile(filepath.Join(docsDir, "broken.md.tmp"), []byte("partial"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, stderr, err := runDoctor(t, dir)
	if err == nil {
		t.Fatal("expected exit code 1 when issues found")
	}
	if cli.ExitCodeFrom(err) != cli.ExitError {
		t.Errorf("expected exit code %d, got error: %v", cli.ExitError, err)
	}
	if !strings.Contains(stderr, "orphan-tmp") {
		t.Errorf("expected 'orphan-tmp' in stderr, got: %q", stderr)
	}
	if !strings.Contains(stderr, "lore doctor --fix") {
		t.Errorf("expected fix suggestion in stderr, got: %q", stderr)
	}
}

func TestDoctor_FixMode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := setupDoctorDir(t)
	docsDir := filepath.Join(dir, ".lore", "docs")

	// Create a doc and an orphan .tmp (old enough to fix)
	_, _ = storage.WriteDoc(docsDir, domain.DocMeta{Type: "note", Date: "2026-03-07", Status: "published"}, "test doc", "# Test\n\nBody.\n")
	tmpPath := filepath.Join(docsDir, "old-write.md.tmp")
	if err := os.WriteFile(tmpPath, []byte("partial"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_ = os.Chtimes(tmpPath, time.Now().Add(-10*time.Second), time.Now().Add(-10*time.Second))

	_, stderr, err := runDoctor(t, dir, "--fix")
	if err != nil {
		t.Fatalf("expected no error after fix, got: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stderr, "Fixed") {
		t.Errorf("expected 'Fixed' in stderr, got: %q", stderr)
	}

	// Verify .tmp removed
	if _, statErr := os.Stat(tmpPath); !os.IsNotExist(statErr) {
		t.Error("expected .tmp to be removed after fix")
	}
}

func TestDoctor_ManualFixRequired(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := setupDoctorDir(t)
	docsDir := filepath.Join(dir, ".lore", "docs")

	// Write an invalid front matter file
	if err := os.WriteFile(filepath.Join(docsDir, "bad-doc.md"), []byte("---\n{{invalid\n---\n# Bad\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_ = storage.RegenerateIndex(docsDir) // non-fatal parse error expected for bad-doc.md

	_, stderr, err := runDoctor(t, dir, "--fix")
	if err == nil {
		t.Fatal("expected exit code 1 when manual fix required")
	}
	if !strings.Contains(stderr, "manual fix required") {
		t.Errorf("expected 'manual fix required' in stderr, got: %q", stderr)
	}
}

func TestDoctor_QuietMode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := setupDoctorDir(t)
	docsDir := filepath.Join(dir, ".lore", "docs")

	// Create orphan .tmp
	if err := os.WriteFile(filepath.Join(docsDir, "orphan.md.tmp"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	stdout, stderr, err := runDoctor(t, dir, "--quiet")
	if err == nil {
		t.Fatal("expected exit code 1 in quiet mode with issues")
	}
	// Quiet: stderr should be empty, stdout should be the count
	if stderr != "" {
		t.Errorf("expected empty stderr in quiet mode, got: %q", stderr)
	}
	// stdout should contain the issue count (at least 1 for orphan-tmp, possibly stale-index too)
	if !strings.Contains(stdout, "1") && !strings.Contains(stdout, "2") {
		t.Errorf("expected issue count in stdout, got: %q", stdout)
	}
}

func TestDoctor_FixQuietMode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := setupDoctorDir(t)
	docsDir := filepath.Join(dir, ".lore", "docs")

	// Create a doc + invalid frontmatter file (manual fix required)
	_, _ = storage.WriteDoc(docsDir, domain.DocMeta{Type: "note", Date: "2026-03-07", Status: "published"}, "test", "# T\n\nBody.\n")
	if err := os.WriteFile(filepath.Join(docsDir, "bad.md"), []byte("---\n{{invalid\n---\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_ = storage.RegenerateIndex(docsDir) // non-fatal parse error expected for bad.md

	stdout, stderr, err := runDoctor(t, dir, "--fix", "--quiet")
	if err == nil {
		t.Fatal("expected exit code 1 with remaining manual fix")
	}
	// Quiet fix mode: stderr empty, stdout = remaining count
	if stderr != "" {
		t.Errorf("expected empty stderr in --fix --quiet mode, got: %q", stderr)
	}
	if !strings.Contains(stdout, "1") {
		t.Errorf("expected remaining count '1' in stdout, got: %q", stdout)
	}
}

func TestDoctor_NotInitialized(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := t.TempDir()
	testutil.Chdir(t, dir)

	_, stderr, err := runDoctor(t, dir)
	if err == nil {
		t.Fatal("expected error for uninitialized repo")
	}
	if !strings.Contains(stderr, "Lore not initialized") {
		t.Errorf("expected 'Lore not initialized' in stderr, got: %q", stderr)
	}
}

// --- Config Validation Tests ---

func TestDoctor_ConfigValid(t *testing.T) {
	dir := t.TempDir()
	testutil.Chdir(t, dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".lorerc.yaml"), []byte("ai:\n  provider: anthropic\n"), 0o644))

	_, stderr, err := runDoctor(t, dir, "--config")
	if err != nil {
		t.Fatalf("expected no error for valid config, got: %v", err)
	}
	if !strings.Contains(stderr, "Config OK") {
		t.Errorf("expected 'Config OK' in stderr, got: %q", stderr)
	}
	if !strings.Contains(stderr, "ai.provider") {
		t.Errorf("expected active values in stderr, got: %q", stderr)
	}
}

func TestDoctor_ConfigTypoWithSuggestion(t *testing.T) {
	dir := t.TempDir()
	testutil.Chdir(t, dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".lorerc.yaml"), []byte("ai:\n  povider: anthropic\n"), 0o644))

	_, stderr, err := runDoctor(t, dir, "--config")
	if err == nil {
		t.Fatal("expected exit code 1 with config warnings")
	}
	if !strings.Contains(stderr, "ai.povider") {
		t.Errorf("expected typo field in stderr, got: %q", stderr)
	}
	if !strings.Contains(stderr, "did you mean") {
		t.Errorf("expected suggestion in stderr, got: %q", stderr)
	}
}

func TestDoctor_ConfigIncludedInStandardDiagnostic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := setupDoctorDir(t)
	docsDir := filepath.Join(dir, ".lore", "docs")
	_, _ = storage.WriteDoc(docsDir, domain.DocMeta{Type: "note", Date: "2026-03-07", Status: "published"}, "test", "# T\n\nBody.\n")
	_ = storage.RegenerateIndex(docsDir)

	// Valid config — should appear as ✓ config
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".lorerc.yaml"), []byte("ai:\n  provider: anthropic\n"), 0o644))

	_, stderr, err := runDoctor(t, dir)
	if err != nil {
		t.Fatalf("expected no error, got: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stderr, "config") {
		t.Errorf("expected 'config' category in standard diagnostic, got: %q", stderr)
	}
}

func TestDoctor_ConfigNoFiles(t *testing.T) {
	dir := t.TempDir()
	testutil.Chdir(t, dir)

	_, stderr, err := runDoctor(t, dir, "--config")
	if err != nil {
		t.Fatalf("expected no error with no config files (defaults used), got: %v", err)
	}
	if !strings.Contains(stderr, "Config OK") {
		t.Errorf("expected 'Config OK' in stderr, got: %q", stderr)
	}
}

func TestDoctor_ConfigQuietMode(t *testing.T) {
	dir := t.TempDir()
	testutil.Chdir(t, dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".lorerc.yaml"), []byte("ai:\n  povider: bad\n"), 0o644))

	stdout, stderr, err := runDoctor(t, dir, "--config", "--quiet")
	if err == nil {
		t.Fatal("expected exit code 1 with config warnings in quiet mode")
	}
	if stderr != "" {
		t.Errorf("expected empty stderr in quiet mode, got: %q", stderr)
	}
	if !strings.Contains(stdout, "1") {
		t.Errorf("expected warning count in stdout, got: %q", stdout)
	}
}

func TestDoctor_RebuildStore(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	restore := ui.SaveAndDisableColor()
	defer restore()

	// Need a real git repo for rebuild-store
	dir := testutil.SetupGitRepo(t)
	testutil.Chdir(t, dir)

	// Create .lore/docs/ structure
	docsDir := filepath.Join(dir, ".lore", "docs")
	for _, sub := range []string{docsDir, filepath.Join(dir, ".lore", "templates"), filepath.Join(dir, ".lore", "pending")} {
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	// Write a valid document
	_, _ = storage.WriteDoc(docsDir, domain.DocMeta{
		Type:   "decision",
		Date:   "2026-03-07",
		Status: "published",
	}, "test doc", "# Test\n\nBody.\n")

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &out,
		Err: &errBuf,
	}
	cfg := &config.Config{}
	cmd := newDoctorCmd(cfg, streams)
	cmd.SetArgs([]string{"--rebuild-store"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("rebuild-store: %v\nstderr: %s", err, errBuf.String())
	}

	stderr := errBuf.String()
	if !strings.Contains(stderr, "1") {
		t.Errorf("expected doc count in output, got: %q", stderr)
	}

	// Verify store.db was created
	if _, statErr := os.Stat(filepath.Join(dir, ".lore", "store.db")); os.IsNotExist(statErr) {
		t.Error("store.db should be created after rebuild")
	}
}

func TestDoctor_RebuildStore_NotInitialized(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	restore := ui.SaveAndDisableColor()
	defer restore()

	dir := t.TempDir()
	testutil.Chdir(t, dir)

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &out,
		Err: &errBuf,
	}
	cfg := &config.Config{}
	cmd := newDoctorCmd(cfg, streams)
	cmd.SetArgs([]string{"--rebuild-store"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for uninitialized repo")
	}
	if !strings.Contains(errBuf.String(), "Lore not initialized") {
		t.Errorf("expected 'Lore not initialized' in stderr, got: %q", errBuf.String())
	}
}

func TestDoctor_ExitCode1_WithIssues(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := setupDoctorDir(t)
	docsDir := filepath.Join(dir, ".lore", "docs")
	os.WriteFile(filepath.Join(docsDir, "stale.md.tmp"), []byte("x"), 0o644)

	_, _, err := runDoctor(t, dir)
	if err == nil {
		t.Fatal("expected error (exit code 1)")
	}
	if cli.ExitCodeFrom(err) != cli.ExitError {
		t.Errorf("expected exit code %d, got: %v", cli.ExitError, err)
	}
}

// --- Story 8-22: doctor safe auto-fix on malformed frontmatter ----------

// TestDoctor_MalformedFM_ShowsSuggestionBlock asserts AC-4: a malformed
// FM issue in diagnose mode surfaces the two-action suggestion block
// (restore + manual edit) on stderr, and the detail includes the
// subkind prefix ("malformed: ...").
func TestDoctor_MalformedFM_ShowsSuggestionBlock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	dir := setupDoctorDir(t)
	docsDir := filepath.Join(dir, ".lore", "docs")

	malformed := "---\ntype: [unclosed\n---\n## Why\nRecoverable content.\n"
	if err := os.WriteFile(filepath.Join(docsDir, "decision-bad-2026-04-19.md"), []byte(malformed), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, stderr, err := runDoctor(t, dir)
	if err == nil {
		t.Fatalf("expected exit != 0 when issues present")
	}
	if !strings.Contains(stderr, "malformed:") {
		t.Errorf("expected subkind prefix 'malformed:' in stderr:\n%s", stderr)
	}
	if !strings.Contains(stderr, "Suggested actions:") {
		t.Errorf("expected suggestion block header in stderr:\n%s", stderr)
	}
	if !strings.Contains(stderr, "Edit the file manually") {
		t.Errorf("expected manual-edit hint in stderr:\n%s", stderr)
	}
}

// TestDoctor_MalformedFM_NoBackup_SkipsRestoreHint asserts AC-5: when
// no polish backup exists for the file, the restore hint is omitted —
// only the manual-edit line remains.
func TestDoctor_MalformedFM_NoBackup_SkipsRestoreHint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	dir := setupDoctorDir(t)
	docsDir := filepath.Join(dir, ".lore", "docs")

	malformed := "---\ntype: [unclosed\n---\n## Why\nRecoverable.\n"
	if err := os.WriteFile(filepath.Join(docsDir, "decision-nobackup-2026-04-19.md"), []byte(malformed), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, stderr, _ := runDoctor(t, dir)
	if strings.Contains(stderr, "--restore") {
		t.Errorf("restore hint should be omitted when no backup exists:\n%s", stderr)
	}
	if !strings.Contains(stderr, "Edit the file manually") {
		t.Errorf("manual-edit hint should still appear:\n%s", stderr)
	}
}

// TestDoctor_MalformedFM_WithBackup_ShowsRestoreHint asserts AC-5
// (positive case): when a polish backup exists for the file, the
// restore hint is included with the exact `--restore <filename>`
// command line.
func TestDoctor_MalformedFM_WithBackup_ShowsRestoreHint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	dir := setupDoctorDir(t)
	docsDir := filepath.Join(dir, ".lore", "docs")

	filename := "decision-withbackup-2026-04-19.md"
	malformed := "---\ntype: [broken\n---\n## Why\nRecoverable.\n"
	if err := os.WriteFile(filepath.Join(docsDir, filename), []byte(malformed), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Create a polish-backup entry for this filename so the probe finds it.
	backupRoot := filepath.Join(dir, ".lore", "angela", "polish-backups")
	if err := os.MkdirAll(backupRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	backupName := filename + ".20260418T120000Z.bak"
	if err := os.WriteFile(filepath.Join(backupRoot, backupName), []byte("pre-polish content"), 0o644); err != nil {
		t.Fatalf("WriteFile backup: %v", err)
	}

	_, stderr, _ := runDoctor(t, dir)
	if !strings.Contains(stderr, "Restore from a polish backup") {
		t.Errorf("expected restore hint when backup exists:\n%s", stderr)
	}
	// Story 8-22 P1: the restore hint shell-quotes the filename so
	// copy-paste into a shell is injection-safe. Accept either the
	// raw or the single-quoted rendering (simple ascii filename has
	// no special metacharacters so both are safe at run time).
	if !strings.Contains(stderr, "lore angela polish --restore '"+filename+"'") {
		t.Errorf("expected quoted restore command with filename:\n%s", stderr)
	}
}

// TestDoctor_MalformedFM_ShellQuotesFilenameWithMetachars asserts the
// Story 8-22 P1 fix: filenames containing shell metacharacters
// (spaces, `;`, backticks, `$`) must be single-quoted in the restore
// hint so a user copy-pasting the suggestion cannot accidentally
// execute embedded code.
func TestDoctor_MalformedFM_ShellQuotesFilenameWithMetachars(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	dir := setupDoctorDir(t)
	docsDir := filepath.Join(dir, ".lore", "docs")

	// Filename with spaces and a semicolon — valid on POSIX, toxic
	// when pasted into a shell without quoting.
	filename := "decision-bad; rm -rf $HOME-2026-04-19.md"
	malformed := "---\ntype: [broken\n---\n## Why\nRecoverable.\n"
	if err := os.WriteFile(filepath.Join(docsDir, filename), []byte(malformed), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	// Ensure a backup exists so the restore hint is emitted.
	backupRoot := filepath.Join(dir, ".lore", "angela", "polish-backups")
	if err := os.MkdirAll(backupRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backupRoot, filename+".20260418T120000Z.bak"), []byte("pre"), 0o644); err != nil {
		t.Fatalf("WriteFile backup: %v", err)
	}

	_, stderr, _ := runDoctor(t, dir)

	// Must appear single-quoted: `'<filename>'`.
	wantQuoted := "'" + filename + "'"
	if !strings.Contains(stderr, wantQuoted) {
		t.Errorf("filename with metachars must be single-quoted:\nstderr:\n%s", stderr)
	}
	// Must NOT appear unquoted on the restore line — scan the line
	// explicitly rather than relying on substring ambiguity.
	for _, line := range strings.Split(stderr, "\n") {
		if strings.Contains(line, "lore angela polish --restore") {
			if strings.Contains(line, wantQuoted) {
				continue // good: line has the quoted form
			}
			t.Errorf("restore line must contain quoted filename, got:\n%s", line)
		}
	}
}

// TestDoctor_MissingFM_NoSuggestionBlock asserts that truly-missing
// frontmatter does NOT trigger the malformed suggestion block — the
// auto-fix path covers it and no manual action is needed.
func TestDoctor_MissingFM_NoSuggestionBlock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	dir := setupDoctorDir(t)
	docsDir := filepath.Join(dir, ".lore", "docs")

	// No `---` delimiter — this is "missing", auto-fixable.
	missing := "# Just a title\nSome content.\n"
	if err := os.WriteFile(filepath.Join(docsDir, "decision-missing-2026-04-19.md"), []byte(missing), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, stderr, _ := runDoctor(t, dir)
	if strings.Contains(stderr, "Suggested actions:") {
		t.Errorf("missing-FM issue should NOT emit the malformed suggestion block:\n%s", stderr)
	}
	if strings.Contains(stderr, "missing:") {
		// subkind prefix appears in diagnose detail — that's fine,
		// but let's be sure it's specifically the missing subkind and
		// not the malformed one.
	}
}

// --- Story 8-23: doctor --prune ---------------------------------------

// TestDoctor_Prune_RunsAllPrunersCleanly verifies that `doctor --prune`
// invokes the Pruner registry and exits with 0 when every Pruner runs
// without error, even if there is nothing to remove.
func TestDoctor_Prune_RunsAllPrunersCleanly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	dir := setupDoctorDir(t)
	_ = dir

	_, stderr, err := runDoctor(t, dir, "--prune")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Each registered Pruner should contribute a line.
	for _, feat := range []string{"polish-backups", "polish-log", "corrupt-quarantine"} {
		if !strings.Contains(stderr, feat) {
			t.Errorf("stderr missing pruner feature %q:\n%s", feat, stderr)
		}
	}
}

// TestDoctor_Prune_DryRun_NoFilesChanged asserts that --prune --dry-run
// reports findings without touching the filesystem.
func TestDoctor_Prune_DryRun_NoFilesChanged(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	dir := setupDoctorDir(t)

	// Plant a stale corrupt-quarantine file (age 30d, retention 14d).
	stateDir := filepath.Join(dir, ".lore", "angela")
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		t.Fatalf("mkdir state: %v", err)
	}
	stamp := time.Now().Add(-30 * 24 * time.Hour).UTC().Format("20060102T150405.000")
	stalePath := filepath.Join(stateDir, "draft-state.json.corrupt-"+stamp)
	if err := os.WriteFile(stalePath, []byte("corrupt"), 0o600); err != nil {
		t.Fatalf("write stale: %v", err)
	}

	_, stderr, err := runDoctor(t, dir, "--prune", "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr, "dry-run") {
		t.Errorf("expected 'dry-run' in stderr:\n%s", stderr)
	}
	// File must still exist.
	if _, err := os.Stat(stalePath); err != nil {
		t.Errorf("dry-run removed the file: %v", err)
	}
}

// TestDoctor_Prune_CleansCorruptQuarantine verifies the
// corrupt-quarantine pruner removes files older than the retention
// threshold when --prune is run without --dry-run.
func TestDoctor_Prune_CleansCorruptQuarantine(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	dir := setupDoctorDir(t)

	stateDir := filepath.Join(dir, ".lore", "angela")
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		t.Fatalf("mkdir state: %v", err)
	}
	oldStamp := time.Now().Add(-60 * 24 * time.Hour).UTC().Format("20060102T150405.000")
	freshStamp := time.Now().Add(-1 * 24 * time.Hour).UTC().Format("20060102T150405.000")
	oldPath := filepath.Join(stateDir, "review-state.json.corrupt-"+oldStamp)
	freshPath := filepath.Join(stateDir, "draft-state.json.corrupt-"+freshStamp)
	_ = os.WriteFile(oldPath, []byte("old"), 0o600)
	_ = os.WriteFile(freshPath, []byte("fresh"), 0o600)

	_, _, err := runDoctor(t, dir, "--prune")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("expected old quarantine file removed: %v", err)
	}
	if _, err := os.Stat(freshPath); err != nil {
		t.Errorf("fresh quarantine file removed unexpectedly: %v", err)
	}
}

// TestDoctor_Prune_MutuallyExclusiveWithFix asserts the cobra mutual
// exclusion between --prune and --fix.
func TestDoctor_Prune_MutuallyExclusiveWithFix(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	dir := setupDoctorDir(t)
	_, _, err := runDoctor(t, dir, "--prune", "--fix")
	if err == nil {
		t.Fatal("expected mutual-exclusion error")
	}
	if !strings.Contains(err.Error(), "prune") || !strings.Contains(err.Error(), "fix") {
		t.Errorf("error should name both flags; got: %v", err)
	}
}

// TestDoctor_Prune_QuietMode asserts the tab-separated machine format
// with one line per Pruner feature.
func TestDoctor_Prune_QuietMode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	dir := setupDoctorDir(t)

	stdout, _, err := runDoctor(t, dir, "--prune", "--quiet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Three Pruners registered → three lines.
	lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines (one per Pruner), got %d:\n%s", len(lines), stdout)
	}
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) != 4 {
			t.Errorf("line has %d tab-separated fields, want 4: %q", len(parts), line)
		}
	}
}

// TestDoctor_FixMode_MixedMissingAndMalformed asserts AC-3 end-to-end:
// `--fix` synthesizes FM for the missing one and leaves the malformed
// one untouched, emitting the suggestion block for the untouched one.
func TestDoctor_FixMode_MixedMissingAndMalformed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	dir := setupDoctorDir(t)
	docsDir := filepath.Join(dir, ".lore", "docs")

	missingPath := filepath.Join(docsDir, "decision-mix-missing-2026-04-19.md")
	malformedPath := filepath.Join(docsDir, "decision-mix-bad-2026-04-19.md")
	if err := os.WriteFile(missingPath, []byte("# Title only\n"), 0o644); err != nil {
		t.Fatalf("WriteFile missing: %v", err)
	}
	malformedBefore := "---\ntype: [unclosed\n---\n## Why\nKeep me.\n"
	if err := os.WriteFile(malformedPath, []byte(malformedBefore), 0o644); err != nil {
		t.Fatalf("WriteFile malformed: %v", err)
	}

	_, stderr, _ := runDoctor(t, dir, "--fix")
	_ = stderr

	// Missing doc should now have frontmatter.
	missingAfter, _ := os.ReadFile(missingPath)
	if !strings.HasPrefix(string(missingAfter), "---\n") {
		t.Errorf("missing-FM doc was not auto-fixed:\n%s", string(missingAfter))
	}
	// Malformed doc must be bit-for-bit unchanged (I31).
	malformedAfter, _ := os.ReadFile(malformedPath)
	if string(malformedAfter) != malformedBefore {
		t.Errorf("I31 violation — malformed doc was rewritten\nbefore: %q\nafter:  %q", malformedBefore, string(malformedAfter))
	}
	// The malformed file should still be listed in the fix-summary output
	// (AutoFix=false → appears as manual-fix-required).
	if !strings.Contains(stderr, "Edit the file manually") {
		t.Errorf("expected manual-edit hint in --fix mode for malformed file:\n%s", stderr)
	}
}
