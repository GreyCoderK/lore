// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// SetupGitRepo creates a temporary Git repo (git init) and returns its path.
// Uses t.TempDir() for auto-cleanup. Does NOT chdir — the caller uses the
// returned path explicitly.
func SetupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_NOSYSTEM=1",
		"HOME="+dir, // avoid reading user's global gitconfig
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}

	// Configure minimal git identity for commits
	for _, kv := range [][2]string{
		{"user.name", "Test User"},
		{"user.email", "test@example.com"},
	} {
		cmd := exec.Command("git", "config", kv[0], kv[1])
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git config %s failed: %v", kv[0], err)
		}
	}

	return dir
}

// SetupGitRepoWithHook creates a Git repo with a lore post-commit hook marker installed.
// Returns the repo path.
//
// NOTE: hook content is hardcoded here and may drift from the actual hook installed
// by git.installHook (which reads from an embedded script). testutil cannot import
// git/ (import boundary). If the hook format changes, update this fixture accordingly.
func SetupGitRepoWithHook(t *testing.T) string {
	t.Helper()
	dir := SetupGitRepo(t)

	hooksDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("create hooks dir: %v", err)
	}

	hookContent := "#!/bin/sh\n# LORE-START\nlore hook-post-commit\n# LORE-END\n"
	hookPath := filepath.Join(hooksDir, "post-commit")
	if err := os.WriteFile(hookPath, []byte(hookContent), 0o755); err != nil {
		t.Fatalf("write hook: %v", err)
	}

	return dir
}
