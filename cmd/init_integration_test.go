// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/git"
)

func initRealGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
	return dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func TestIntegration_FullInit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initRealGitRepo(t)
	streams, _, errBuf := testStreams()

	deps := initDeps{
		git:     git.NewAdapter(dir),
		workDir: dir,
	}

	err := runInit(context.Background(), &config.Config{}, deps, streams, true)
	if err != nil {
		t.Fatalf("runInit: %v", err)
	}

	// Verify .lore/docs/ created
	if _, err := os.Stat(filepath.Join(dir, ".lore", "docs")); os.IsNotExist(err) {
		t.Error(".lore/docs/ should exist")
	}

	// Verify .lorerc content
	data, err := os.ReadFile(filepath.Join(dir, ".lorerc"))
	if err != nil {
		t.Fatalf("read .lorerc: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "ai:") {
		t.Error(".lorerc missing ai section")
	}
	if !strings.Contains(content, "angela:") {
		t.Error(".lorerc missing angela section")
	}
	if !strings.Contains(content, "hooks:") {
		t.Error(".lorerc missing hooks section")
	}

	// Verify .lorerc.local content
	data, err = os.ReadFile(filepath.Join(dir, ".lorerc.local"))
	if err != nil {
		t.Fatalf("read .lorerc.local: %v", err)
	}
	if !strings.Contains(string(data), "api_key") {
		t.Error(".lorerc.local missing api_key")
	}

	// Verify .gitignore
	data, err = os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if !strings.Contains(string(data), ".lorerc.local") {
		t.Error(".gitignore missing .lorerc.local")
	}

	// Verify hook installed
	hookPath := filepath.Join(dir, ".git", "hooks", "post-commit")
	data, err = os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read hook: %v", err)
	}
	hookContent := string(data)
	if !strings.Contains(hookContent, "# LORE-START") {
		t.Error("hook missing LORE-START marker")
	}

	// Verify hook is executable
	info, _ := os.Stat(hookPath)
	if info.Mode()&0111 == 0 {
		t.Error("hook should be executable")
	}

	// Verify output
	output := errBuf.String()
	if !strings.Contains(output, "Created") {
		t.Error("output missing 'Created' verbs")
	}
	if !strings.Contains(output, "Installed") {
		t.Error("output missing 'Installed' verb")
	}
}

func TestIntegration_NotGitDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := t.TempDir() // no git init
	streams, _, errBuf := testStreams()

	deps := initDeps{
		git:     git.NewAdapter(dir),
		workDir: dir,
	}

	err := runInit(context.Background(), &config.Config{}, deps, streams, true)
	if err != domain.ErrNotGitRepo {
		t.Errorf("expected ErrNotGitRepo, got %v", err)
	}

	output := errBuf.String()
	if !strings.Contains(output, "Not a git repository") {
		t.Error("should show error message")
	}

	// Verify nothing was created
	if _, err := os.Stat(filepath.Join(dir, ".lore")); !os.IsNotExist(err) {
		t.Error(".lore/ should NOT exist")
	}
}

func TestIntegration_DoubleInit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initRealGitRepo(t)
	streams1, _, _ := testStreams()

	deps := initDeps{
		git:     git.NewAdapter(dir),
		workDir: dir,
	}

	// First init
	if err := runInit(context.Background(), &config.Config{}, deps, streams1, true); err != nil {
		t.Fatalf("first init: %v", err)
	}

	// Record state after first init
	lorercData, _ := os.ReadFile(filepath.Join(dir, ".lorerc"))

	// Second init
	streams2, _, errBuf2 := testStreams()
	if err := runInit(context.Background(), &config.Config{}, deps, streams2, true); err != nil {
		t.Fatalf("second init: %v", err)
	}

	// Verify warning message
	if !strings.Contains(errBuf2.String(), "Lore already initialized") {
		t.Error("second init should show warning")
	}

	// Verify .lorerc not overwritten
	lorercDataAfter, _ := os.ReadFile(filepath.Join(dir, ".lorerc"))
	if string(lorercData) != string(lorercDataAfter) {
		t.Error(".lorerc should not be modified on second init")
	}
}
