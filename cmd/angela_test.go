// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/testutil"
	"github.com/greycoderk/lore/internal/ui"
)

func runAngelaDraft(t *testing.T, cfg *config.Config, args ...string) (stdout, stderr string, exitErr error) {
	t.Helper()
	restore := ui.SaveAndDisableColor()
	defer restore()

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &out,
		Err: &errBuf,
	}
	if cfg == nil {
		cfg = &config.Config{}
	}
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs(append([]string{"draft"}, args...))
	err := cmd.Execute()
	return out.String(), errBuf.String(), err
}

// AC-4, AC-1: Draft with suggestions produces formatted report
func TestAngelaDraft_WithSuggestions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	// Create a minimal document missing sections
	docsDir := filepath.Join(dir, ".lore", "docs")
	doc := "---\ntype: decision\nstatus: published\ndate: \"2026-03-19\"\n---\nShort."
	if err := os.WriteFile(filepath.Join(docsDir, "test-doc.md"), []byte(doc), 0644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := runAngelaDraft(t, nil, "test-doc.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr, "suggestions") {
		t.Errorf("expected suggestions in output, got: %s", stderr)
	}
	if !strings.Contains(stderr, "structure") {
		t.Errorf("expected 'structure' category in output, got: %s", stderr)
	}
}

// AC-5: No suggestions for a complete doc
func TestAngelaDraft_NoSuggestions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	docsDir := filepath.Join(dir, ".lore", "docs")
	doc := "---\ntype: decision\nstatus: published\ndate: \"2026-03-19\"\ntags: [api]\nrelated: [other]\n---\n" +
		"## What\nThis is a complete document about API decisions and strategy.\n\n" +
		"## Why\nBecause we need a clear API strategy for the long term success of the project.\n\n" +
		"## Alternatives\nWe could do nothing and let chaos reign.\n\n" +
		"## Impact\nThis affects the entire API surface.\n"
	if err := os.WriteFile(filepath.Join(docsDir, "complete-doc.md"), []byte(doc), 0644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := runAngelaDraft(t, nil, "complete-doc.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr, "No suggestions") {
		t.Errorf("expected 'No suggestions' message, got: %s", stderr)
	}
}

// AC-6: Document not found
func TestAngelaDraft_DocNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	_, _, err := runAngelaDraft(t, nil, "nonexistent.md")
	if err == nil {
		t.Fatal("expected error for missing document")
	}
	if !strings.Contains(err.Error(), "not found in .lore/docs/") {
		t.Errorf("error = %q, want 'not found in .lore/docs/'", err)
	}
}

// AC-7: Lore not initialized
func TestAngelaDraft_NotInitialized(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	dir := t.TempDir() // no .lore/
	testutil.Chdir(t, dir)

	_, _, err := runAngelaDraft(t, nil, "test.md")
	if err == nil {
		t.Fatal("expected error for uninitialized repo")
	}
	if !strings.Contains(err.Error(), "lore not initialized") {
		t.Errorf("error = %q, want 'lore not initialized'", err)
	}
}

// --- --all tests ---

func runAngelaDraftAll(t *testing.T, cfg *config.Config) (stdout, stderr string, exitErr error) {
	t.Helper()
	restore := ui.SaveAndDisableColor()
	defer restore()

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{In: strings.NewReader(""), Out: &out, Err: &errBuf}
	if cfg == nil {
		cfg = &config.Config{}
	}
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"draft", "--all"})
	err := cmd.Execute()
	return out.String(), errBuf.String(), err
}

func TestAngelaDraft_All_WithDocs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	docsDir := filepath.Join(dir, ".lore", "docs")
	doc := "---\ntype: decision\nstatus: published\ndate: \"2026-03-20\"\n---\nShort."
	_ = os.WriteFile(filepath.Join(docsDir, "test-doc.md"), []byte(doc), 0644)

	_, stderr, err := runAngelaDraftAll(t, nil)
	if err != nil {
		t.Fatalf("--all: %v", err)
	}
	if !strings.Contains(stderr, "1 documents") {
		t.Errorf("expected '1 documents' in output, got: %s", stderr)
	}
	if !strings.Contains(stderr, "need attention") {
		t.Errorf("expected 'need attention' summary, got: %s", stderr)
	}
}

func TestAngelaDraft_All_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	_, stderr, err := runAngelaDraftAll(t, nil)
	if err != nil {
		t.Fatalf("--all empty: %v", err)
	}
	if !strings.Contains(stderr, "No documents") {
		t.Errorf("expected 'No documents' message, got: %s", stderr)
	}
}

func TestAngelaDraft_All_NotInitialized(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := t.TempDir()
	testutil.Chdir(t, dir)

	_, _, err := runAngelaDraftAll(t, nil)
	if err == nil {
		t.Fatal("expected error for uninitialized repo")
	}
	if !strings.Contains(err.Error(), "lore not initialized") {
		t.Errorf("error = %q, want 'lore not initialized'", err)
	}
}

// --- polish tests ---

func runAngelaPolish(t *testing.T, cfg *config.Config, stdinInput string, args ...string) (stdout, stderr string, exitErr error) {
	t.Helper()
	restore := ui.SaveAndDisableColor()
	defer restore()

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{In: strings.NewReader(stdinInput), Out: &out, Err: &errBuf}
	if cfg == nil {
		cfg = &config.Config{}
	}
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs(append([]string{"polish"}, args...))
	err := cmd.Execute()
	return out.String(), errBuf.String(), err
}

// AC-4: No provider configured
func TestAngelaPolish_NoProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	docsDir := filepath.Join(dir, ".lore", "docs")
	doc := "---\ntype: decision\nstatus: published\ndate: \"2026-03-20\"\n---\n## Why\nReason."
	_ = os.WriteFile(filepath.Join(docsDir, "test.md"), []byte(doc), 0644)

	_, _, err := runAngelaPolish(t, nil, "", "test.md")
	if err == nil {
		t.Fatal("expected error for no provider")
	}
	if !strings.Contains(err.Error(), "no AI provider configured") {
		t.Errorf("error = %q, want 'no AI provider configured'", err)
	}
}

// AC-8: Document not found
func TestAngelaPolish_DocNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	_, _, err := runAngelaPolish(t, nil, "", "nonexistent.md")
	if err == nil {
		t.Fatal("expected error for missing doc")
	}
	if !strings.Contains(err.Error(), "not found in .lore/docs/") {
		t.Errorf("error = %q, want 'not found'", err)
	}
}

// AC-9: Not initialized
func TestAngelaPolish_NotInitialized(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := t.TempDir()
	testutil.Chdir(t, dir)

	_, _, err := runAngelaPolish(t, nil, "", "test.md")
	if err == nil {
		t.Fatal("expected error for uninitialized repo")
	}
	if !strings.Contains(err.Error(), "lore not initialized") {
		t.Errorf("error = %q, want 'lore not initialized'", err)
	}
}

// --- review tests ---

func runAngelaReview(t *testing.T, cfg *config.Config, args ...string) (stdout, stderr string, exitErr error) {
	t.Helper()
	restore := ui.SaveAndDisableColor()
	defer restore()

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{In: strings.NewReader(""), Out: &out, Err: &errBuf}
	if cfg == nil {
		cfg = &config.Config{}
	}
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs(append([]string{"review"}, args...))
	err := cmd.Execute()
	return out.String(), errBuf.String(), err
}

// createNDocs creates n valid documents in the .lore/docs directory.
func createNDocs(t *testing.T, dir string, n int) {
	t.Helper()
	docsDir := filepath.Join(dir, ".lore", "docs")
	for i := 0; i < n; i++ {
		doc := fmt.Sprintf("---\ntype: decision\nstatus: published\ndate: \"2026-03-%02d\"\ntags: [test]\n---\n## What\nDocument %d about topic %d.\n\n## Why\nBecause reason %d.\n", i+1, i, i, i)
		filename := fmt.Sprintf("decision-topic-%d-2026-03-%02d.md", i, i+1)
		if err := os.WriteFile(filepath.Join(docsDir, filename), []byte(doc), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

// AC-5: No provider configured
func TestAngelaReview_NoProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)
	createNDocs(t, dir, 5)

	_, _, err := runAngelaReview(t, nil)
	if err == nil {
		t.Fatal("expected error for no provider")
	}
	if !strings.Contains(err.Error(), "no AI provider configured") {
		t.Errorf("error = %q, want 'no AI provider configured'", err)
	}
}

// AC-2: Less than 5 docs
func TestAngelaReview_LessThan5Docs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)
	createNDocs(t, dir, 3)

	_, _, err := runAngelaReview(t, nil)
	if err == nil {
		t.Fatal("expected error for < 5 docs")
	}
	if !strings.Contains(err.Error(), "at least 5 documents required") {
		t.Errorf("error = %q, want 'at least 5 documents required'", err)
	}
}

// AC-9: Not initialized
func TestAngelaReview_NotInitialized(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := t.TempDir()
	testutil.Chdir(t, dir)

	_, _, err := runAngelaReview(t, nil)
	if err == nil {
		t.Fatal("expected error for uninitialized repo")
	}
	if !strings.Contains(err.Error(), "lore not initialized") {
		t.Errorf("error = %q, want 'lore not initialized'", err)
	}
}

// AC-8: --quiet suppresses stderr but stdout report remains
func TestAngelaReview_QuietFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)
	createNDocs(t, dir, 5)

	// Without provider, this will fail before output — but --quiet should still be parsed
	// Test that the flag is recognized by cobra (no "unknown flag" error)
	_, _, err := runAngelaReview(t, nil, "--quiet")
	if err == nil {
		t.Fatal("expected error (no provider)")
	}
	// Should be a provider error, not a flag parse error
	if strings.Contains(err.Error(), "unknown flag") {
		t.Errorf("--quiet flag should be recognized, got: %q", err)
	}
}

// AC-8: --quiet + extra args rejected (cobra.NoArgs)
func TestAngelaReview_RejectsArgs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	_, _, err := runAngelaReview(t, nil, "unexpected-arg")
	if err == nil {
		t.Fatal("expected error for unexpected arg")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("error = %q, want 'unknown command'", err)
	}
}
