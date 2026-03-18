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
)

func setupDoctorDir(t *testing.T) string {
	t.Helper()
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)
	return dir
}

func runDoctor(t *testing.T, dir string, args ...string) (stdout, stderr string, exitErr error) {
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
