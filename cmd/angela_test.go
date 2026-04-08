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

	"github.com/greycoderk/lore/internal/angela"
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

// --- angela draft additional tests ---

// draft with no args and no docs → error about no file
func TestAngelaDraft_NoArgsNoDocs(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	_, _, err := runAngelaDraft(t, nil)
	if err == nil {
		t.Fatal("expected error for no filename and no docs")
	}
}

// draft with no args but docs exist → auto-picks most recent
func TestAngelaDraft_NoArgsAutoPickRecent(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	docsDir := filepath.Join(dir, ".lore", "docs")
	// Create two docs with different dates
	olderDoc := "---\ntype: decision\nstatus: published\ndate: \"2026-01-01\"\n---\nOlder doc."
	newerDoc := "---\ntype: decision\nstatus: published\ndate: \"2026-03-15\"\n---\nNewer doc about something important."
	if err := os.WriteFile(filepath.Join(docsDir, "decision-old-2026-01-01.md"), []byte(olderDoc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "decision-new-2026-03-15.md"), []byte(newerDoc), 0644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := runAngelaDraft(t, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should auto-pick the newer doc and show its filename in output
	if !strings.Contains(stderr, "decision-new-2026-03-15.md") {
		t.Errorf("expected auto-picked newer doc in output, got: %s", stderr)
	}
}

// draft --all with doc that has warnings (covers runDraftAll warning count path)
func TestAngelaDraft_All_WithWarnings(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	docsDir := filepath.Join(dir, ".lore", "docs")
	// Short doc triggers warnings
	doc := "---\ntype: decision\nstatus: published\ndate: \"2026-03-01\"\n---\nShort."
	if err := os.WriteFile(filepath.Join(docsDir, "decision-short-2026-03-01.md"), []byte(doc), 0644); err != nil {
		t.Fatal(err)
	}
	// Complete doc with no issues
	goodDoc := "---\ntype: decision\nstatus: published\ndate: \"2026-03-02\"\ntags: [api]\nrelated: [other]\n---\n" +
		"## What\nThis is complete.\n\n## Why\nBecause we need it.\n\n## Alternatives\nDo nothing.\n\n## Impact\nBig.\n"
	if err := os.WriteFile(filepath.Join(docsDir, "decision-complete-2026-03-02.md"), []byte(goodDoc), 0644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := runAngelaDraftAll(t, nil)
	if err != nil {
		t.Fatalf("--all: %v", err)
	}
	// Should show mixed results (one ok, one review)
	if !strings.Contains(stderr, "2 documents") {
		t.Errorf("expected '2 documents' in output, got: %s", stderr)
	}
	if !strings.Contains(stderr, "need attention") {
		t.Errorf("expected 'need attention' summary, got: %s", stderr)
	}
}

// draft --all with multiple docs
func TestAngelaDraft_All_MultipleDocs(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	docsDir := filepath.Join(dir, ".lore", "docs")
	for i := 0; i < 3; i++ {
		doc := fmt.Sprintf("---\ntype: decision\nstatus: published\ndate: \"2026-03-%02d\"\n---\nShort.", i+1)
		filename := fmt.Sprintf("decision-topic-%d-2026-03-%02d.md", i, i+1)
		if err := os.WriteFile(filepath.Join(docsDir, filename), []byte(doc), 0644); err != nil {
			t.Fatal(err)
		}
	}

	_, stderr, err := runAngelaDraftAll(t, nil)
	if err != nil {
		t.Fatalf("--all: %v", err)
	}
	if !strings.Contains(stderr, "3 documents") {
		t.Errorf("expected '3 documents' in output, got: %s", stderr)
	}
}

// draft nonexistent.md → error
func TestAngelaDraft_NonexistentFile(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	_, _, err := runAngelaDraft(t, nil, "totally-nonexistent.md")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want 'not found'", err)
	}
}

// No filename and no --all (original test renamed)
func TestAngelaDraft_NoFilename(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	_, _, err := runAngelaDraft(t, nil)
	if err == nil {
		t.Fatal("expected error for no filename and no --all")
	}
}

// Invalid filename validation
func TestAngelaDraft_InvalidFilename(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	_, _, err := runAngelaDraft(t, nil, "../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal filename")
	}
	if !strings.Contains(err.Error(), "angela: draft:") {
		t.Errorf("error = %q, want 'angela: draft:' prefix", err)
	}
}

// --- angela polish validation tests (no AI provider needed) ---

// AC-8: Filename validation — path traversal rejected
func TestAngelaPolish_InvalidFilename_PathTraversal(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	_, _, err := runAngelaPolish(t, nil, "", "../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal filename")
	}
	if !strings.Contains(err.Error(), "angela: polish:") {
		t.Errorf("error = %q, want 'angela: polish:' prefix", err)
	}
}

// AC-8: Filename validation — absolute path rejected
func TestAngelaPolish_InvalidFilename_AbsolutePath(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	_, _, err := runAngelaPolish(t, nil, "", "/tmp/test.md")
	if err == nil {
		t.Fatal("expected error for absolute filename")
	}
	if !strings.Contains(err.Error(), "angela: polish:") {
		t.Errorf("error = %q, want 'angela: polish:' prefix", err)
	}
}

// AC-8: Filename validation — path separator rejected
func TestAngelaPolish_InvalidFilename_PathSeparator(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	_, _, err := runAngelaPolish(t, nil, "", "subdir/test.md")
	if err == nil {
		t.Fatal("expected error for filename with path separator")
	}
	if !strings.Contains(err.Error(), "angela: polish:") {
		t.Errorf("error = %q, want 'angela: polish:' prefix", err)
	}
}

// AC-8: Reserved filename rejected
func TestAngelaPolish_ReservedFilename(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	_, _, err := runAngelaPolish(t, nil, "", "README.md")
	if err == nil {
		t.Fatal("expected error for reserved filename")
	}
	if !strings.Contains(err.Error(), "angela: polish:") {
		t.Errorf("error = %q, want 'angela: polish:' prefix", err)
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

// --- additional angela polish tests ---

// Provider configured but API call fails (covers provider creation + service call error path)
func TestAngelaPolish_ProviderCallFails(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	docsDir := filepath.Join(dir, ".lore", "docs")
	doc := "---\ntype: decision\nstatus: published\ndate: \"2026-03-20\"\n---\n## Why\nSome reason here."
	if err := os.WriteFile(filepath.Join(docsDir, "test-doc.md"), []byte(doc), 0644); err != nil {
		t.Fatal(err)
	}

	// Use ollama provider with a bad endpoint — provider gets created but API call fails
	cfg := &config.Config{}
	cfg.AI.Provider = "ollama"
	cfg.AI.Endpoint = "http://127.0.0.1:1" // port 1 — connection refused
	cfg.AI.Model = "test"

	_, _, err := runAngelaPolish(t, cfg, "", "test-doc.md")
	if err == nil {
		t.Fatal("expected error from failed API call")
	}
	// Error should come from the polish service call, not from "no provider" check
	if strings.Contains(err.Error(), "no AI provider configured") {
		t.Errorf("provider should have been created, got: %q", err)
	}
}

// Provider configured but API call fails for review (covers provider creation path)
func TestAngelaReview_ProviderCallFails(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)
	createNDocs(t, dir, 5)

	cfg := &config.Config{}
	cfg.AI.Provider = "ollama"
	cfg.AI.Endpoint = "http://127.0.0.1:1"
	cfg.AI.Model = "test"

	_, _, err := runAngelaReview(t, cfg)
	if err == nil {
		t.Fatal("expected error from failed API call")
	}
	if strings.Contains(err.Error(), "no AI provider configured") {
		t.Errorf("provider should have been created, got: %q", err)
	}
}

// cobra.ExactArgs(1): no arguments provided
func TestAngelaPolish_NoArgs(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	_, _, err := runAngelaPolish(t, nil, "")
	if err == nil {
		t.Fatal("expected error for no arguments")
	}
}

// cobra.ExactArgs(1): too many arguments
func TestAngelaPolish_TooManyArgs(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	_, _, err := runAngelaPolish(t, nil, "", "file1.md", "file2.md")
	if err == nil {
		t.Fatal("expected error for too many arguments")
	}
}

// Empty filename rejected
func TestAngelaPolish_EmptyFilename(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	_, _, err := runAngelaPolish(t, nil, "", "")
	if err == nil {
		t.Fatal("expected error for empty filename")
	}
}

// Dot-dot filename rejected (path traversal variant)
func TestAngelaPolish_DotDotFilename(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	_, _, err := runAngelaPolish(t, nil, "", "..%2f..%2fetc%2fpasswd")
	if err == nil {
		t.Fatal("expected error for encoded path traversal")
	}
}

// Hidden file — not found (passes validation but file doesn't exist)
func TestAngelaPolish_HiddenFile(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	_, _, err := runAngelaPolish(t, nil, "", ".hidden.md")
	if err == nil {
		t.Fatal("expected error for hidden filename")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want 'not found'", err)
	}
}

// --- additional angela review tests ---

// Review with exactly 0 docs
func TestAngelaReview_ZeroDocs(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	_, _, err := runAngelaReview(t, nil)
	if err == nil {
		t.Fatal("expected error for 0 docs")
	}
	if !strings.Contains(err.Error(), "at least 5 documents required") {
		t.Errorf("error = %q, want 'at least 5 documents required'", err)
	}
}

// Review with exactly 4 docs (boundary)
func TestAngelaReview_FourDocs(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)
	createNDocs(t, dir, 4)

	_, _, err := runAngelaReview(t, nil)
	if err == nil {
		t.Fatal("expected error for 4 docs (< 5 required)")
	}
	if !strings.Contains(err.Error(), "at least 5 documents required") {
		t.Errorf("error = %q, want 'at least 5 documents required'", err)
	}
}

// Review with exactly 1 doc
func TestAngelaReview_OneDoc(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)
	createNDocs(t, dir, 1)

	_, _, err := runAngelaReview(t, nil)
	if err == nil {
		t.Fatal("expected error for 1 doc")
	}
	if !strings.Contains(err.Error(), "at least 5 documents required") {
		t.Errorf("error = %q, want 'at least 5 documents required'", err)
	}
}

// formatReviewReport: partial corpus path (totalCorpus > DocCount)
func TestFormatReviewReport_PartialCorpus(t *testing.T) {
	restore := ui.SaveAndDisableColor()
	defer restore()

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{In: strings.NewReader(""), Out: &out, Err: &errBuf}

	report := &angela.ReviewReport{
		Findings: nil,
		DocCount: 50,
	}
	// totalCorpus > DocCount triggers partial header
	formatReviewReport(streams, report, 60, false)

	if errBuf.Len() == 0 {
		t.Error("expected stderr output for partial corpus header")
	}
}

// countSeverities with all four known severities
func TestCountSeverities_AllKnown(t *testing.T) {
	findings := []angela.ReviewFinding{
		{Severity: "contradiction"},
		{Severity: "gap"},
		{Severity: "gap"},
		{Severity: "style"},
		{Severity: "obsolete"},
	}
	result := countSeverities(findings)
	if !strings.Contains(result, "1 contradiction") {
		t.Errorf("result = %q, want '1 contradiction'", result)
	}
	if !strings.Contains(result, "2 gap") {
		t.Errorf("result = %q, want '2 gap'", result)
	}
	if !strings.Contains(result, "1 style") {
		t.Errorf("result = %q, want '1 style'", result)
	}
	if !strings.Contains(result, "1 obsolete") {
		t.Errorf("result = %q, want '1 obsolete'", result)
	}
}
