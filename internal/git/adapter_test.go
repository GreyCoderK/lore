// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initGitRepo creates a real git repo in a temp directory for testing.
func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run(t, dir, "git", "init")
	run(t, dir, "git", "config", "user.email", "test@test.com")
	run(t, dir, "git", "config", "user.name", "Test")
	return dir
}

// initGitRepoWithCommit creates a repo with one commit.
func initGitRepoWithCommit(t *testing.T) string {
	t.Helper()
	dir := initGitRepo(t)
	run(t, dir, "git", "commit", "--allow-empty", "-m", "initial commit")
	return dir
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
	}
}

func TestIsInsideWorkTree_True(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepo(t)
	a := NewAdapter(dir)

	if !a.IsInsideWorkTree() {
		t.Error("expected IsInsideWorkTree() = true for git repo")
	}
}

func TestIsInsideWorkTree_False(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := t.TempDir() // not a git repo
	a := NewAdapter(dir)

	if a.IsInsideWorkTree() {
		t.Error("expected IsInsideWorkTree() = false for non-git dir")
	}
}

func TestHeadRef_WithCommit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepoWithCommit(t)
	a := NewAdapter(dir)

	ref, err := a.HeadRef()
	if err != nil {
		t.Fatalf("HeadRef: %v", err)
	}
	if len(ref) < 7 {
		t.Errorf("expected a commit hash, got %q", ref)
	}
}

func TestHeadRef_NoCommits(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepo(t)
	a := NewAdapter(dir)

	_, err := a.HeadRef()
	if err == nil {
		t.Error("expected error for HeadRef with no commits")
	}
}

func TestHeadCommit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Use initGitRepoWithCommit so the test commit has a parent (non-root).
	dir := initGitRepoWithCommit(t)
	run(t, dir, "git", "commit", "--allow-empty", "-m", "feat(auth): add login endpoint")
	a := NewAdapter(dir)

	info, err := a.HeadCommit()
	if err != nil {
		t.Fatalf("HeadCommit: %v", err)
	}
	if info.Hash == "" || len(info.Hash) < 7 {
		t.Errorf("expected a valid commit hash, got %q", info.Hash)
	}
	if info.Author != "Test" {
		t.Errorf("expected author 'Test', got %q", info.Author)
	}
	if info.Type != "feat" {
		t.Errorf("expected type 'feat', got %q", info.Type)
	}
	if info.Scope != "auth" {
		t.Errorf("expected scope 'auth', got %q", info.Scope)
	}
	if info.Subject != "add login endpoint" {
		t.Errorf("expected subject 'add login endpoint', got %q", info.Subject)
	}
	if info.Date.IsZero() {
		t.Error("expected non-zero date")
	}
	if info.IsMerge {
		t.Error("expected IsMerge = false for non-merge commit")
	}
	if info.ParentCount != 1 {
		t.Errorf("expected ParentCount = 1, got %d", info.ParentCount)
	}

	// Verify HeadCommit matches HeadRef + Log
	ref, err := a.HeadRef()
	if err != nil {
		t.Fatalf("HeadRef: %v", err)
	}
	if info.Hash != ref {
		t.Errorf("HeadCommit hash %q != HeadRef %q", info.Hash, ref)
	}
}

func TestHeadCommit_NoCommits(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepo(t)
	a := NewAdapter(dir)

	_, err := a.HeadCommit()
	if err == nil {
		t.Error("expected error for HeadCommit with no commits")
	}
}

func TestHeadCommit_MergeCommit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepoWithCommit(t)
	run(t, dir, "git", "checkout", "-b", "feature")
	run(t, dir, "git", "commit", "--allow-empty", "-m", "feature commit")
	run(t, dir, "git", "checkout", "-")
	run(t, dir, "git", "merge", "--no-ff", "feature", "-m", "merge feature")

	a := NewAdapter(dir)
	info, err := a.HeadCommit()
	if err != nil {
		t.Fatalf("HeadCommit: %v", err)
	}
	if !info.IsMerge {
		t.Error("expected IsMerge = true for merge commit")
	}
	if info.ParentCount < 2 {
		t.Errorf("expected ParentCount >= 2, got %d", info.ParentCount)
	}
}

func TestCommitExists_True(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepoWithCommit(t)
	a := NewAdapter(dir)

	ref, _ := a.HeadRef()
	exists, err := a.CommitExists(ref)
	if err != nil {
		t.Fatalf("CommitExists: %v", err)
	}
	if !exists {
		t.Error("expected CommitExists = true for HEAD commit")
	}
}

func TestCommitExists_False(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepoWithCommit(t)
	a := NewAdapter(dir)

	exists, err := a.CommitExists("0000000000000000000000000000000000000000")
	if err != nil {
		t.Fatalf("CommitExists: %v", err)
	}
	if exists {
		t.Error("expected CommitExists = false for nonexistent commit")
	}
}

func TestAdapter_WorkDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepo(t)
	a := NewAdapter(dir)

	if !a.IsInsideWorkTree() {
		t.Error("adapter should work from workDir")
	}

	sub := filepath.Join(dir, "subdir")
	run(t, dir, "mkdir", sub)
	aSub := NewAdapter(sub)
	if !aSub.IsInsideWorkTree() {
		t.Error("adapter should work from subdirectory of git repo")
	}
}

// --- Integration tests (real git repos) ---

func TestLog_ConventionalCommit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepo(t)
	run(t, dir, "git", "commit", "--allow-empty", "-m", "feat(auth): add login endpoint")
	a := NewAdapter(dir)

	info, err := a.Log("HEAD")
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if info.Hash == "" {
		t.Error("expected non-empty hash")
	}
	if info.Author != "Test" {
		t.Errorf("expected author 'Test', got %q", info.Author)
	}
	if info.Type != "feat" {
		t.Errorf("expected type 'feat', got %q", info.Type)
	}
	if info.Scope != "auth" {
		t.Errorf("expected scope 'auth', got %q", info.Scope)
	}
	if info.Subject != "add login endpoint" {
		t.Errorf("expected subject 'add login endpoint', got %q", info.Subject)
	}
}

func TestLog_NonConventionalCommit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepo(t)
	run(t, dir, "git", "commit", "--allow-empty", "-m", "just a normal commit")
	a := NewAdapter(dir)

	info, err := a.Log("HEAD")
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if info.Type != "" {
		t.Errorf("expected empty type for non-CC, got %q", info.Type)
	}
	if info.Subject != "just a normal commit" {
		t.Errorf("expected subject = full message, got %q", info.Subject)
	}
}

func TestDiff_WithChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepoWithCommit(t)
	// Create a file and commit it
	if err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	run(t, dir, "git", "add", "test.txt")
	run(t, dir, "git", "commit", "-m", "add test.txt")

	a := NewAdapter(dir)
	diff, err := a.Diff("HEAD")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if !strings.Contains(diff, "test.txt") {
		t.Error("diff should contain the changed file")
	}
	if !strings.Contains(diff, "+hello") {
		t.Error("diff should contain the added content")
	}
}

func TestIsMergeCommit_False(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepoWithCommit(t)
	a := NewAdapter(dir)

	isMerge, err := a.IsMergeCommit("HEAD")
	if err != nil {
		t.Fatalf("IsMergeCommit: %v", err)
	}
	if isMerge {
		t.Error("expected IsMergeCommit = false for non-merge commit")
	}
}

func TestIsMergeCommit_True(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepoWithCommit(t)
	// Create a branch with a commit, then merge back
	run(t, dir, "git", "checkout", "-b", "feature")
	run(t, dir, "git", "commit", "--allow-empty", "-m", "feature commit")
	// Go back to the initial branch (could be main or master)
	run(t, dir, "git", "checkout", "-")
	run(t, dir, "git", "merge", "--no-ff", "feature", "-m", "merge feature")

	a := NewAdapter(dir)
	isMerge, err := a.IsMergeCommit("HEAD")
	if err != nil {
		t.Fatalf("IsMergeCommit: %v", err)
	}
	if !isMerge {
		t.Error("expected IsMergeCommit = true for merge commit")
	}
}

func TestIsRebaseInProgress_False(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepoWithCommit(t)
	a := NewAdapter(dir)

	inRebase, err := a.IsRebaseInProgress()
	if err != nil {
		t.Fatalf("IsRebaseInProgress: %v", err)
	}
	if inRebase {
		t.Error("expected IsRebaseInProgress = false")
	}
}

func TestIsRebaseInProgress_True(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepoWithCommit(t)
	// Simulate rebase in progress by creating the rebase-merge directory
	rebaseMerge := filepath.Join(dir, ".git", "rebase-merge")
	if err := os.MkdirAll(rebaseMerge, 0755); err != nil {
		t.Fatalf("create rebase-merge: %v", err)
	}

	a := NewAdapter(dir)
	inRebase, err := a.IsRebaseInProgress()
	if err != nil {
		t.Fatalf("IsRebaseInProgress: %v", err)
	}
	if !inRebase {
		t.Error("expected IsRebaseInProgress = true")
	}
}

func TestCommitMessageContains_True(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepo(t)
	run(t, dir, "git", "commit", "--allow-empty", "-m", "feat: add [skip-lore] marker")
	a := NewAdapter(dir)

	contains, err := a.CommitMessageContains("HEAD", "[skip-lore]")
	if err != nil {
		t.Fatalf("CommitMessageContains: %v", err)
	}
	if !contains {
		t.Error("expected CommitMessageContains = true")
	}
}

func TestCommitMessageContains_False(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepo(t)
	run(t, dir, "git", "commit", "--allow-empty", "-m", "feat: add login")
	a := NewAdapter(dir)

	contains, err := a.CommitMessageContains("HEAD", "[skip-lore]")
	if err != nil {
		t.Fatalf("CommitMessageContains: %v", err)
	}
	if contains {
		t.Error("expected CommitMessageContains = false")
	}
}

// --- ParseConventionalCommit unit tests ---

func TestParseConventionalCommit_Full(t *testing.T) {
	ccType, scope, subject := ParseConventionalCommit("feat(auth): add login endpoint")
	if ccType != "feat" {
		t.Errorf("type = %q, want feat", ccType)
	}
	if scope != "auth" {
		t.Errorf("scope = %q, want auth", scope)
	}
	if subject != "add login endpoint" {
		t.Errorf("subject = %q, want 'add login endpoint'", subject)
	}
}

func TestParseConventionalCommit_NoScope(t *testing.T) {
	ccType, scope, subject := ParseConventionalCommit("fix: correct typo")
	if ccType != "fix" {
		t.Errorf("type = %q, want fix", ccType)
	}
	if scope != "" {
		t.Errorf("scope = %q, want empty", scope)
	}
	if subject != "correct typo" {
		t.Errorf("subject = %q, want 'correct typo'", subject)
	}
}

func TestParseConventionalCommit_NonConventional(t *testing.T) {
	ccType, scope, subject := ParseConventionalCommit("just a commit message")
	if ccType != "" {
		t.Errorf("type = %q, want empty", ccType)
	}
	if scope != "" {
		t.Errorf("scope = %q, want empty", scope)
	}
	if subject != "just a commit message" {
		t.Errorf("subject = %q, want 'just a commit message'", subject)
	}
}

func TestParseConventionalCommit_Multiline(t *testing.T) {
	msg := "feat(api): add endpoint\n\nThis is the body of the commit."
	ccType, scope, subject := ParseConventionalCommit(msg)
	if ccType != "feat" {
		t.Errorf("type = %q, want feat", ccType)
	}
	if scope != "api" {
		t.Errorf("scope = %q, want api", scope)
	}
	if subject != "add endpoint" {
		t.Errorf("subject = %q, want 'add endpoint'", subject)
	}
}

// --- CommitRange / LatestTag integration tests ---

func TestLatestTag(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepoWithCommit(t)
	run(t, dir, "git", "tag", "v0.1.0")
	a := NewAdapter(dir)

	tag, err := a.LatestTag()
	if err != nil {
		t.Fatalf("LatestTag: %v", err)
	}
	if tag != "v0.1.0" {
		t.Errorf("expected v0.1.0, got %s", tag)
	}
}

func TestLatestTag_NoTags(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepoWithCommit(t)
	a := NewAdapter(dir)

	_, err := a.LatestTag()
	if err == nil {
		t.Fatal("expected error when no tags exist")
	}
}

func TestCommitRange(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepoWithCommit(t)
	run(t, dir, "git", "tag", "v0.1.0")
	run(t, dir, "git", "commit", "--allow-empty", "-m", "second commit")
	run(t, dir, "git", "commit", "--allow-empty", "-m", "third commit")
	a := NewAdapter(dir)

	commits, err := a.CommitRange("v0.1.0", "HEAD")
	if err != nil {
		t.Fatalf("CommitRange: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(commits))
	}
	for _, c := range commits {
		if len(c) != 40 {
			t.Errorf("expected 40-char SHA, got %q (len %d)", c, len(c))
		}
	}
}

func TestCommitRange_EmptyRange(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepoWithCommit(t)
	run(t, dir, "git", "tag", "v0.1.0")
	a := NewAdapter(dir)

	commits, err := a.CommitRange("v0.1.0", "HEAD")
	if err != nil {
		t.Fatalf("CommitRange: %v", err)
	}
	if len(commits) != 0 {
		t.Fatalf("expected 0 commits for empty range, got %d", len(commits))
	}
}

// --- validateRef unit tests ---

func TestValidateRef_Valid(t *testing.T) {
	valid := []string{"HEAD", "v1.0.0", "main", "feature/foo", "abc123", "v1.0.0-rc.1"}
	for _, ref := range valid {
		if err := validateRef(ref); err != nil {
			t.Errorf("validateRef(%q) unexpected error: %v", ref, err)
		}
	}
}

func TestValidateRef_FlagInjection(t *testing.T) {
	if err := validateRef("--all"); err == nil {
		t.Error("expected error for flag-like ref '--all'")
	}
	if err := validateRef("-n"); err == nil {
		t.Error("expected error for flag-like ref '-n'")
	}
}

func TestValidateRef_UnsafeChars(t *testing.T) {
	if err := validateRef("v1.0; rm -rf /"); err == nil {
		t.Error("expected error for ref with semicolons/spaces")
	}
}

func TestValidateRef_DoubleDot(t *testing.T) {
	if err := validateRef("v1..v2"); err == nil {
		t.Error("expected error for ref containing '..'")
	}
}

func TestValidateRef_Empty(t *testing.T) {
	if err := validateRef(""); err == nil {
		t.Error("empty ref should be invalid")
	}
}

// --- parseHeadCommitOutput unit tests ---

func TestParseHeadCommitOutput_Valid(t *testing.T) {
	out := "abc1234567890abcdef1234567890abcdef123456\nAlice\n2026-03-15T10:00:00+00:00\nparent123\nfeat(api): add endpoint\n"
	info, err := parseHeadCommitOutput(out)
	if err != nil {
		t.Fatalf("parseHeadCommitOutput: %v", err)
	}
	if info.Hash != "abc1234567890abcdef1234567890abcdef123456" {
		t.Errorf("Hash = %q", info.Hash)
	}
	if info.Author != "Alice" {
		t.Errorf("Author = %q", info.Author)
	}
	if info.Type != "feat" {
		t.Errorf("Type = %q", info.Type)
	}
	if info.Scope != "api" {
		t.Errorf("Scope = %q", info.Scope)
	}
	if info.Subject != "add endpoint" {
		t.Errorf("Subject = %q", info.Subject)
	}
	if info.ParentCount != 1 {
		t.Errorf("ParentCount = %d, want 1", info.ParentCount)
	}
}

func TestParseHeadCommitOutput_MergeCommit(t *testing.T) {
	out := "abc123\nAlice\n2026-03-15T10:00:00+00:00\nparent1 parent2\nMerge branch 'feature'\n"
	info, err := parseHeadCommitOutput(out)
	if err != nil {
		t.Fatalf("parseHeadCommitOutput: %v", err)
	}
	if !info.IsMerge {
		t.Error("expected IsMerge = true for merge commit")
	}
	if info.ParentCount != 2 {
		t.Errorf("ParentCount = %d, want 2", info.ParentCount)
	}
}

func TestParseHeadCommitOutput_BadFormat(t *testing.T) {
	_, err := parseHeadCommitOutput("too\nfew\nlines")
	if err == nil {
		t.Error("expected error for malformed output")
	}
}

// --- Embedded hook script test ---

func TestPostCommitScript_NonEmpty(t *testing.T) {
	if postCommitScript == "" {
		t.Error("postCommitScript should not be empty")
	}
	if !strings.Contains(postCommitScript, "lore _hook-post-commit") {
		t.Error("postCommitScript should contain 'lore _hook-post-commit'")
	}
}
