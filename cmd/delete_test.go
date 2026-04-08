// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/greycoderk/lore/internal/testutil"
	"github.com/greycoderk/lore/internal/ui"
)

// setupDeleteDir creates a .lore dir with a document to delete.
// Returns (dir, filename).
func setupDeleteDir(t *testing.T) (string, string) {
	t.Helper()
	dir := testutil.SetupLoreDir(t)
	docsDir := filepath.Join(dir, ".lore", "docs")

	result, err := storage.WriteDoc(docsDir, domain.DocMeta{
		Type:   "decision",
		Date:   "2026-03-07",
		Status: "published",
	}, "auth strategy", "# Auth Strategy\n")
	if err != nil {
		t.Fatalf("setup WriteDoc: %v", err)
	}
	return dir, result.Filename
}

// overrideTTY sets deleteIsTTY to return the given value and restores it on cleanup.
func overrideTTY(t *testing.T, val bool) {
	t.Helper()
	orig := deleteIsTTY
	deleteIsTTY = func(_ domain.IOStreams) bool { return val }
	t.Cleanup(func() { deleteIsTTY = orig })
}

func TestDeleteCmd_ConfirmYes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	restore := ui.SaveAndDisableColor()
	defer restore()
	overrideTTY(t, true)

	dir, filename := setupDeleteDir(t)
	testutil.Chdir(t, dir)

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader("y\n"),
	}

	cmd := newDeleteCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{filename})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("delete with confirm: %v", err)
	}

	// File should be gone
	docPath := filepath.Join(dir, ".lore", "docs", filename)
	if _, statErr := os.Stat(docPath); !os.IsNotExist(statErr) {
		t.Error("expected file to be deleted")
	}

	// Success message
	if !strings.Contains(errBuf.String(), "Deleted") {
		t.Errorf("expected 'Deleted' in stderr, got: %q", errBuf.String())
	}
	if !strings.Contains(errBuf.String(), filename) {
		t.Errorf("expected filename in stderr, got: %q", errBuf.String())
	}

	// AC-2: confirmation prompt shown
	if !strings.Contains(errBuf.String(), "Delete "+filename+"? [y/N]") {
		t.Errorf("expected confirmation prompt, got: %q", errBuf.String())
	}
}

func TestDeleteCmd_ConfirmNo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	overrideTTY(t, true)

	dir, filename := setupDeleteDir(t)
	testutil.Chdir(t, dir)

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader("n\n"),
	}

	cmd := newDeleteCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{filename})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("delete decline: %v", err)
	}

	// File should still exist
	docPath := filepath.Join(dir, ".lore", "docs", filename)
	if _, statErr := os.Stat(docPath); os.IsNotExist(statErr) {
		t.Error("file should still exist after declining")
	}

	// Should show "Not deleted." feedback
	if !strings.Contains(errBuf.String(), "Not deleted.") {
		t.Errorf("expected 'Not deleted.' in stderr, got: %q", errBuf.String())
	}
}

func TestDeleteCmd_DemoNoConfirm(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	restore := ui.SaveAndDisableColor()
	defer restore()

	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)
	docsDir := filepath.Join(dir, ".lore", "docs")

	result, err := storage.WriteDoc(docsDir, domain.DocMeta{
		Type:   "decision",
		Date:   "2026-03-07",
		Status: "demo",
	}, "demo doc", "# Demo\n")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	// No stdin input needed — demo skips confirmation
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	cmd := newDeleteCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{result.Filename})
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("delete demo: %v", err)
	}

	if _, statErr := os.Stat(result.Path); !os.IsNotExist(statErr) {
		t.Error("demo doc should be deleted without confirmation")
	}

	if !strings.Contains(errBuf.String(), "Deleted") {
		t.Errorf("expected 'Deleted' in stderr, got: %q", errBuf.String())
	}
}

func TestDeleteCmd_ForceNoConfirm(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	restore := ui.SaveAndDisableColor()
	defer restore()

	dir, filename := setupDeleteDir(t)
	testutil.Chdir(t, dir)

	// No stdin input — --force skips confirmation
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	cmd := newDeleteCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{filename, "--force"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("delete --force: %v", err)
	}

	docPath := filepath.Join(dir, ".lore", "docs", filename)
	if _, statErr := os.Stat(docPath); !os.IsNotExist(statErr) {
		t.Error("file should be deleted with --force")
	}

	if !strings.Contains(errBuf.String(), "Deleted") {
		t.Errorf("expected 'Deleted' in stderr, got: %q", errBuf.String())
	}
}

func TestDeleteCmd_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	restore := ui.SaveAndDisableColor()
	defer restore()

	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	cmd := newDeleteCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{"nonexistent.md"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent doc")
	}

	// AC-5: actionable error
	if !strings.Contains(errBuf.String(), "not found") {
		t.Errorf("expected 'not found' in stderr, got: %q", errBuf.String())
	}
}

func TestDeleteCmd_NotInitialized(t *testing.T) {
	restore := ui.SaveAndDisableColor()
	defer restore()

	dir := t.TempDir()
	testutil.Chdir(t, dir)

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	cmd := newDeleteCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{"anything.md"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when not initialized")
	}

	if !strings.Contains(errBuf.String(), "Lore not initialized") {
		t.Errorf("expected 'Lore not initialized' in stderr, got: %q", errBuf.String())
	}
}

func TestDeleteCmd_NonTTYNoForce(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	restore := ui.SaveAndDisableColor()
	defer restore()

	// Ensure deleteIsTTY returns false (default behavior with strings.NewReader)
	overrideTTY(t, false)

	dir, filename := setupDeleteDir(t)
	testutil.Chdir(t, dir)

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	cmd := newDeleteCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{filename})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error in non-TTY without --force")
	}

	// AC-8: actionable message
	if !strings.Contains(errBuf.String(), "Confirmation required") {
		t.Errorf("expected 'Confirmation required' in stderr, got: %q", errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "--force") {
		t.Errorf("expected '--force' suggestion in stderr, got: %q", errBuf.String())
	}

	// File should still exist
	docPath := filepath.Join(dir, ".lore", "docs", filename)
	if _, statErr := os.Stat(docPath); os.IsNotExist(statErr) {
		t.Error("file should not be deleted in non-TTY without --force")
	}
}

func TestDeleteCmd_ReferencedDocWarning(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	restore := ui.SaveAndDisableColor()
	defer restore()

	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)
	docsDir := filepath.Join(dir, ".lore", "docs")

	// Create target doc
	result, err := storage.WriteDoc(docsDir, domain.DocMeta{
		Type: "decision", Date: "2026-03-07", Status: "published",
	}, "auth strategy", "# Auth Strategy\n")
	if err != nil {
		t.Fatalf("setup target: %v", err)
	}

	// Create referencing doc
	refName := strings.TrimSuffix(result.Filename, ".md")
	_, _ = storage.WriteDoc(docsDir, domain.DocMeta{
		Type: "feature", Date: "2026-03-08", Status: "published",
		Related: []string{refName},
	}, "login flow", "# Login\n")

	// --force to skip TTY/confirmation issues, focus on warning
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	cmd := newDeleteCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{result.Filename, "--force"})
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("delete referenced doc: %v", err)
	}

	// AC-4: warning about references
	output := errBuf.String()
	if !strings.Contains(output, "referenced by") {
		t.Errorf("expected 'referenced by' warning, got: %q", output)
	}
	if !strings.Contains(output, "feature-") {
		t.Errorf("expected referencing filename in warning, got: %q", output)
	}

	// Doc should still be deleted
	if !strings.Contains(output, "Deleted") {
		t.Errorf("expected 'Deleted' after warning, got: %q", output)
	}
}

// Delete a document with invalid front matter → parse error
func TestDeleteCmd_InvalidFrontMatter(t *testing.T) {
	restore := ui.SaveAndDisableColor()
	defer restore()

	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)
	docsDir := filepath.Join(dir, ".lore", "docs")

	// Write a file with broken YAML front matter
	badDoc := "---\n{{invalid yaml\n---\n# Bad\n"
	if err := os.WriteFile(filepath.Join(docsDir, "bad-doc.md"), []byte(badDoc), 0644); err != nil {
		t.Fatal(err)
	}

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	cmd := newDeleteCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{"bad-doc.md", "--force"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid front matter")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("error = %q, want 'parse'", err)
	}
}

// Delete a document with filename that fails validation
func TestDeleteCmd_InvalidFilename(t *testing.T) {
	restore := ui.SaveAndDisableColor()
	defer restore()

	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	cmd := newDeleteCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{"../etc/passwd"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid filename")
	}
}

func TestDeleteCmd_Registered(t *testing.T) {
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
		In:  strings.NewReader(""),
	}
	var s domain.LoreStore
	root := newRootCmd(&config.Config{}, streams, &s)

	for _, sub := range root.Commands() {
		if sub.Name() == "delete" {
			return
		}
	}
	t.Error("expected 'delete' subcommand to be registered")
}
