// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package testutil_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/testutil"
)

func TestSetupGitRepo(t *testing.T) {
	dir := testutil.SetupGitRepo(t)

	// .git/ must exist
	info, err := os.Stat(filepath.Join(dir, ".git"))
	if err != nil {
		t.Fatalf("expected .git/ to exist: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected .git to be a directory")
	}

	// git status must work
	cmd := exec.Command("git", "status")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git status failed in test repo: %v", err)
	}
}

func TestSetupGitRepoWithHook(t *testing.T) {
	dir := testutil.SetupGitRepoWithHook(t)

	hookPath := filepath.Join(dir, ".git", "hooks", "post-commit")
	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("expected post-commit hook to exist: %v", err)
	}
	if !strings.Contains(string(data), "LORE-START") {
		t.Fatal("hook file does not contain LORE-START marker")
	}

	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0100 == 0 {
		t.Fatal("hook file is not executable")
	}
}

func TestSetupLoreDir(t *testing.T) {
	dir := testutil.SetupLoreDir(t)

	for _, sub := range []string{
		filepath.Join(".lore", "docs"),
		filepath.Join(".lore", "templates"),
		filepath.Join(".lore", "pending"),
	} {
		path := filepath.Join(dir, sub)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected %s to exist: %v", sub, err)
		}
		if !info.IsDir() {
			t.Fatalf("expected %s to be a directory", sub)
		}
	}
}

func TestSetupLoreDirWithDocs(t *testing.T) {
	docs := []testutil.DocFixture{
		{Type: "decision", Slug: "auth-strategy", Date: "2026-03-07", Tags: []string{"auth", "security"}},
		{Type: "feature", Slug: "add-jwt", Date: "2026-03-05"},
	}

	dir := testutil.SetupLoreDirWithDocs(t, docs)
	docsDir := filepath.Join(dir, ".lore", "docs")

	// Check files exist
	entries, err := os.ReadDir(docsDir)
	if err != nil {
		t.Fatalf("read docs dir: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 docs, got %d", len(entries))
	}

	// Check first doc has valid front matter (starts with ---)
	data, err := os.ReadFile(filepath.Join(docsDir, "decision-auth-strategy-2026-03-07.md"))
	if err != nil {
		t.Fatalf("read doc: %v", err)
	}
	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		t.Fatal("doc does not start with front matter delimiter")
	}
	if !strings.Contains(content, "type: decision") {
		t.Fatal("doc front matter missing type field")
	}
	if !strings.Contains(content, "auth strategy") {
		t.Fatalf("doc body missing slug-based heading, got:\n%s", content)
	}
}

func TestChdir(t *testing.T) {
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()

	testutil.Chdir(t, tmpDir)

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// Resolve symlinks for macOS /private/var vs /var
	resolvedCwd, _ := filepath.EvalSymlinks(cwd)
	resolvedTmp, _ := filepath.EvalSymlinks(tmpDir)
	if resolvedCwd != resolvedTmp {
		t.Errorf("expected cwd=%s, got %s", resolvedTmp, resolvedCwd)
	}

	// Cleanup should restore the original dir, but we can't test that here
	// without running the cleanup manually. Just verify the original is different.
	resolvedOrig, _ := filepath.EvalSymlinks(origDir)
	if resolvedOrig == resolvedTmp {
		t.Error("expected original dir to differ from tmpDir")
	}
}

func TestTestConfig(t *testing.T) {
	cfg := testutil.TestConfig()
	if cfg == nil {
		t.Fatal("TestConfig returned nil")
	}
	if cfg.AI.Provider != "anthropic" {
		t.Errorf("expected provider 'anthropic', got %q", cfg.AI.Provider)
	}
	if cfg.Angela.Mode != "draft" {
		t.Errorf("expected angela mode 'draft', got %q", cfg.Angela.Mode)
	}
}

func TestTestStreams(t *testing.T) {
	streams, out, errBuf := testutil.TestStreams()
	if streams.Out == nil || streams.Err == nil || streams.In == nil {
		t.Fatal("TestStreams returned nil streams")
	}
	if out == nil || errBuf == nil {
		t.Fatal("TestStreams returned nil buffers")
	}
}
