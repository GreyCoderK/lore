package git

import (
	"os/exec"
	"path/filepath"
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
	dir := initGitRepo(t)
	a := NewAdapter(dir)

	if !a.IsInsideWorkTree() {
		t.Error("expected IsInsideWorkTree() = true for git repo")
	}
}

func TestIsInsideWorkTree_False(t *testing.T) {
	dir := t.TempDir() // not a git repo
	a := NewAdapter(dir)

	if a.IsInsideWorkTree() {
		t.Error("expected IsInsideWorkTree() = false for non-git dir")
	}
}

func TestHeadRef_WithCommit(t *testing.T) {
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
	dir := initGitRepo(t)
	a := NewAdapter(dir)

	_, err := a.HeadRef()
	if err == nil {
		t.Error("expected error for HeadRef with no commits")
	}
}

func TestCommitExists_True(t *testing.T) {
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
	dir := initGitRepo(t)
	a := NewAdapter(dir)

	// Verify the adapter works from the specified directory
	if !a.IsInsideWorkTree() {
		t.Error("adapter should work from workDir")
	}

	// Verify with a subdirectory
	sub := filepath.Join(dir, "subdir")
	run(t, dir, "mkdir", sub)
	aSub := NewAdapter(sub)
	if !aSub.IsInsideWorkTree() {
		t.Error("adapter should work from subdirectory of git repo")
	}
}
