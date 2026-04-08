// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- DefaultBranch tests ---

func TestDefaultBranch_WithRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create a "remote" repo to clone from, so origin/HEAD is set.
	remote := initGitRepoWithCommit(t)

	// Clone it — git clone sets origin/HEAD automatically.
	cloneDir := t.TempDir()
	run(t, cloneDir, "git", "clone", remote, "repo")
	repoDir := filepath.Join(cloneDir, "repo")

	a := New(repoDir)
	branch, err := a.DefaultBranch()
	if err != nil {
		t.Fatalf("DefaultBranch: %v", err)
	}
	// Should be "main" or "master" depending on git version config.
	if branch == "" {
		t.Error("expected non-empty default branch name")
	}
}

func TestDefaultBranch_FallbackWhenNoRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepoWithCommit(t)
	a := New(dir)

	branch, err := a.DefaultBranch()
	if err == nil && branch != "" {
		t.Error("expected error or empty branch when no remote is configured")
	}
	// Error is the expected path here
	if err == nil {
		t.Error("expected error when no remote is configured")
	}
}

// --- GitDir tests ---

func TestGitDir_ReturnsGitPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepoWithCommit(t)
	a := New(dir)

	gitDir, err := a.GitDir()
	if err != nil {
		t.Fatalf("GitDir: %v", err)
	}
	expected := filepath.Join(dir, ".git")
	if gitDir != expected {
		t.Errorf("GitDir = %q, want %q", gitDir, expected)
	}
}

func TestGitDir_FromSubdirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepoWithCommit(t)
	sub := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	a := New(sub)
	gitDir, err := a.GitDir()
	if err != nil {
		t.Fatalf("GitDir: %v", err)
	}
	if !strings.HasSuffix(gitDir, ".git") {
		t.Errorf("GitDir = %q, expected .git suffix", gitDir)
	}
}

// --- Log with specific ref tests ---

func TestLog_WithSpecificRef(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepoWithCommit(t)
	run(t, dir, "git", "commit", "--allow-empty", "-m", "fix(db): resolve deadlock")
	a := New(dir)

	ref, _ := a.HeadRef()
	info, err := a.Log(ref)
	if err != nil {
		t.Fatalf("Log(%s): %v", ref, err)
	}
	if info.Type != "fix" {
		t.Errorf("Type = %q, want fix", info.Type)
	}
	if info.Scope != "db" {
		t.Errorf("Scope = %q, want db", info.Scope)
	}
	if info.Subject != "resolve deadlock" {
		t.Errorf("Subject = %q, want 'resolve deadlock'", info.Subject)
	}
}

// --- Diff basic output ---

func TestDiff_BasicOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepoWithCommit(t)

	// Create and commit a file with multiple lines
	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(filepath.Join(dir, "multi.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "add", "multi.txt")
	run(t, dir, "git", "commit", "-m", "add multi.txt")

	a := New(dir)
	diff, err := a.Diff("HEAD")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if !strings.Contains(diff, "+line1") {
		t.Error("diff should contain '+line1'")
	}
	if !strings.Contains(diff, "+line2") {
		t.Error("diff should contain '+line2'")
	}
	if !strings.Contains(diff, "multi.txt") {
		t.Error("diff should reference the filename")
	}
}

func TestDiff_EmptyCommit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepoWithCommit(t)
	run(t, dir, "git", "commit", "--allow-empty", "-m", "empty commit")

	a := New(dir)
	diff, err := a.Diff("HEAD")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if diff != "" {
		t.Errorf("expected empty diff for empty commit, got %q", diff)
	}
}

// --- IsMergeCommit with non-merge ---

func TestIsMergeCommit_NonMergeWithMultipleCommits(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepoWithCommit(t)
	run(t, dir, "git", "commit", "--allow-empty", "-m", "second commit")
	run(t, dir, "git", "commit", "--allow-empty", "-m", "third commit")

	a := New(dir)

	// Check each non-merge commit
	isMerge, err := a.IsMergeCommit("HEAD")
	if err != nil {
		t.Fatalf("IsMergeCommit: %v", err)
	}
	if isMerge {
		t.Error("expected IsMergeCommit = false for linear commit")
	}

	isMerge, err = a.IsMergeCommit("HEAD~1")
	if err != nil {
		t.Fatalf("IsMergeCommit HEAD~1: %v", err)
	}
	if isMerge {
		t.Error("expected IsMergeCommit = false for HEAD~1")
	}
}

// --- CommitMessageContains positive and negative ---

func TestCommitMessageContains_PositiveMatch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepo(t)
	run(t, dir, "git", "commit", "--allow-empty", "-m", "feat: add widget [reviewed]")

	a := New(dir)
	contains, err := a.CommitMessageContains("HEAD", "[reviewed]")
	if err != nil {
		t.Fatalf("CommitMessageContains: %v", err)
	}
	if !contains {
		t.Error("expected true for [reviewed] marker")
	}
}

func TestCommitMessageContains_NegativeMatch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepo(t)
	run(t, dir, "git", "commit", "--allow-empty", "-m", "feat: add widget")

	a := New(dir)
	contains, err := a.CommitMessageContains("HEAD", "[skip-lore]")
	if err != nil {
		t.Fatalf("CommitMessageContains: %v", err)
	}
	if contains {
		t.Error("expected false for absent marker")
	}
}

func TestCommitMessageContains_MultilineBody(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepo(t)
	// Commit with multiline message containing marker in body
	run(t, dir, "git", "commit", "--allow-empty", "-m", "feat: add auth\n\nBody with [skip-lore] marker")

	a := New(dir)
	contains, err := a.CommitMessageContains("HEAD", "[skip-lore]")
	if err != nil {
		t.Fatalf("CommitMessageContains: %v", err)
	}
	if !contains {
		t.Error("expected true for marker in commit body")
	}
}

// --- InstallHook / HookExists / UninstallHook lifecycle ---

func TestHookLifecycle_InstallVerifyUninstall(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepo(t)
	a := New(dir)

	// Before install: should not exist
	exists, err := a.HookExists("post-commit")
	if err != nil {
		t.Fatalf("HookExists: %v", err)
	}
	if exists {
		t.Error("expected HookExists = false before install")
	}

	// Install
	result, err := a.InstallHook("post-commit")
	if err != nil {
		t.Fatalf("InstallHook: %v", err)
	}
	if !result.Installed {
		t.Error("expected Installed = true")
	}

	// After install: should exist
	exists, err = a.HookExists("post-commit")
	if err != nil {
		t.Fatalf("HookExists after install: %v", err)
	}
	if !exists {
		t.Error("expected HookExists = true after install")
	}

	// Uninstall
	if err := a.UninstallHook("post-commit"); err != nil {
		t.Fatalf("UninstallHook: %v", err)
	}

	// After uninstall: should not exist
	exists, err = a.HookExists("post-commit")
	if err != nil {
		t.Fatalf("HookExists after uninstall: %v", err)
	}
	if exists {
		t.Error("expected HookExists = false after uninstall")
	}
}

// --- CommitRange with tag range ---

func TestCommitRange_WithTagRange(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepoWithCommit(t)
	run(t, dir, "git", "tag", "v1.0.0")
	run(t, dir, "git", "commit", "--allow-empty", "-m", "feat: after v1")
	run(t, dir, "git", "commit", "--allow-empty", "-m", "fix: patch")
	run(t, dir, "git", "tag", "v1.1.0")
	run(t, dir, "git", "commit", "--allow-empty", "-m", "feat: after v1.1")

	a := New(dir)

	// Range from v1.0.0 to v1.1.0 should have exactly 2 commits
	commits, err := a.CommitRange("v1.0.0", "v1.1.0")
	if err != nil {
		t.Fatalf("CommitRange: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("expected 2 commits between v1.0.0..v1.1.0, got %d", len(commits))
	}

	// Range from v1.1.0 to HEAD should have 1 commit
	commits, err = a.CommitRange("v1.1.0", "HEAD")
	if err != nil {
		t.Fatalf("CommitRange v1.1.0..HEAD: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit between v1.1.0..HEAD, got %d", len(commits))
	}
}

func TestCommitRange_EmptyTo_DefaultsToHEAD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepoWithCommit(t)
	run(t, dir, "git", "tag", "v0.1.0")
	run(t, dir, "git", "commit", "--allow-empty", "-m", "post-tag commit")

	a := New(dir)

	// Empty "to" should default to HEAD
	commits, err := a.CommitRange("v0.1.0", "")
	if err != nil {
		t.Fatalf("CommitRange: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(commits))
	}
}
