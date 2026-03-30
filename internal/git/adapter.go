// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/greycoderk/lore/internal/domain"
)

// Adapter implements domain.GitAdapter via exec.Command("git", ...).
type Adapter struct {
	workDir string
	getenv  func(string) string // injectable for testing; defaults to os.Getenv
}

// compile-time check
var _ domain.GitAdapter = (*Adapter)(nil)

// New creates an Adapter for the given working directory (M4: canonical constructor name).
func New(workDir string) *Adapter {
	return &Adapter{workDir: workDir, getenv: os.Getenv}
}

// NewAdapter is an alias for New, kept for compatibility with existing callers.
func NewAdapter(workDir string) *Adapter {
	return New(workDir)
}

func (a *Adapter) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = a.workDir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git: %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (a *Adapter) IsInsideWorkTree() bool {
	out, err := a.run("rev-parse", "--is-inside-work-tree")
	return err == nil && out == "true"
}

func (a *Adapter) HeadRef() (string, error) {
	ref, err := a.run("rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("git: head ref: %w", err)
	}
	return ref, nil
}

// HeadCommit returns the full commit info for HEAD in a single git invocation,
// avoiding multiple exec.Command calls. It replaces separate HeadRef() + Log()
// calls with one `git log` invocation.
func (a *Adapter) HeadCommit() (*domain.CommitInfo, error) {
	out, err := a.run("log", "-1", "HEAD", "--format=%H%n%an%n%aI%n%P%n%B")
	if err != nil {
		return nil, fmt.Errorf("git: head commit: %w", err)
	}
	return parseHeadCommitOutput(out)
}

// parseHeadCommitOutput parses the output of git log -1 HEAD --format=%H%n%an%n%aI%n%P%n%B.
// The %P line contains space-separated parent hashes; >1 parent means a merge commit.
func parseHeadCommitOutput(out string) (*domain.CommitInfo, error) {
	lines := strings.SplitN(out, "\n", 5)
	if len(lines) < 5 {
		return nil, fmt.Errorf("git: head commit: unexpected format")
	}

	date, err := time.Parse(time.RFC3339, lines[2])
	if err != nil {
		return nil, fmt.Errorf("git: head commit: parse date: %w", err)
	}

	message := strings.TrimSpace(lines[4])
	ccType, scope, subject := ParseConventionalCommit(message)

	parentCount := len(strings.Fields(lines[3]))

	return &domain.CommitInfo{
		Hash:         lines[0],
		Author:       lines[1],
		Date:         date,
		Message:      message,
		Type:         ccType,
		Scope:        scope,
		Subject:      subject,
		IsMerge:      parentCount > 1,
		ParentCount:  parentCount,
	}, nil
}

func (a *Adapter) CommitExists(ref string) (bool, error) {
	if err := validateRef(ref); err != nil {
		return false, err
	}
	_, err := a.run("cat-file", "-t", ref)
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (a *Adapter) Log(ref string) (*domain.CommitInfo, error) {
	if err := validateRef(ref); err != nil {
		return nil, err
	}
	out, err := a.run("log", "-1", "--format=%H%n%an%n%aI%n%B", ref)
	if err != nil {
		return nil, fmt.Errorf("git: log %s: %w", ref, err)
	}
	return parseLogOutput(out)
}

func parseLogOutput(out string) (*domain.CommitInfo, error) {
	lines := strings.SplitN(out, "\n", 4)
	if len(lines) < 4 {
		return nil, fmt.Errorf("git: log: unexpected format")
	}

	date, err := time.Parse(time.RFC3339, lines[2])
	if err != nil {
		return nil, fmt.Errorf("git: log: parse date: %w", err)
	}

	message := strings.TrimSpace(lines[3])
	ccType, scope, subject := ParseConventionalCommit(message)

	return &domain.CommitInfo{
		Hash:    lines[0],
		Author:  lines[1],
		Date:    date,
		Message: message,
		Type:    ccType,
		Scope:   scope,
		Subject: subject,
	}, nil
}

func (a *Adapter) Diff(ref string) (string, error) {
	if err := validateRef(ref); err != nil {
		return "", err
	}
	out, err := a.run("diff", ref+"^!", "--")
	if err != nil {
		return "", fmt.Errorf("git: diff %s: %w", ref, err)
	}
	return out, nil
}

func (a *Adapter) IsMergeCommit(ref string) (bool, error) {
	if err := validateRef(ref); err != nil {
		return false, err
	}
	out, err := a.run("rev-list", "--parents", "-1", ref)
	if err != nil {
		return false, fmt.Errorf("git: is merge commit %s: %w", ref, err)
	}
	// A merge commit has more than 2 space-separated SHAs (the commit + 2+ parents)
	parts := strings.Fields(out)
	return len(parts) > 2, nil
}

func (a *Adapter) IsRebaseInProgress() (bool, error) {
	// H1 fix: GIT_SEQUENCE_EDITOR is set by git when an interactive rebase is
	// being initiated — before the rebase-merge/ directory is created. Checking
	// it first ensures we catch `git rebase -i` launched from IDEs or scripts.
	if a.getenv("GIT_SEQUENCE_EDITOR") != "" {
		return true, nil
	}
	// H3 partial: use GitDir() so rebase dirs are resolved correctly in worktrees.
	gitDir, err := a.GitDir()
	if err != nil {
		// Fallback to workDir/.git if GitDir() fails (e.g. bare repo edge case).
		gitDir = filepath.Join(a.workDir, ".git")
	}
	for _, dir := range []string{"rebase-merge", "rebase-apply"} {
		if _, statErr := os.Stat(filepath.Join(gitDir, dir)); statErr == nil {
			return true, nil
		}
	}
	return false, nil
}

func (a *Adapter) GitDir() (string, error) {
	out, err := a.run("rev-parse", "--git-dir")
	if err != nil {
		return "", fmt.Errorf("git: git-dir: %w", err)
	}
	if filepath.IsAbs(out) {
		return out, nil
	}
	return filepath.Join(a.workDir, out), nil
}

func (a *Adapter) CommitMessageContains(ref, marker string) (bool, error) {
	if err := validateRef(ref); err != nil {
		return false, err
	}
	out, err := a.run("log", "-1", "--format=%B", ref)
	if err != nil {
		return false, fmt.Errorf("git: commit message %s: %w", ref, err)
	}
	return strings.Contains(out, marker), nil
}

func (a *Adapter) InstallHook(hookType string) (domain.InstallResult, error) {
	// H3 fix: resolve git dir via GitDir() rather than hardcoding workDir/.git/hooks.
	gitDir, err := a.GitDir()
	if err != nil {
		return domain.InstallResult{}, fmt.Errorf("git: install hook %s: git dir: %w", hookType, err)
	}
	return installHook(a.workDir, filepath.Join(gitDir, "hooks"), hookType)
}

func (a *Adapter) UninstallHook(hookType string) error {
	gitDir, err := a.GitDir()
	if err != nil {
		return fmt.Errorf("git: uninstall hook %s: git dir: %w", hookType, err)
	}
	return uninstallHook(filepath.Join(gitDir, "hooks"), hookType)
}

func (a *Adapter) HookExists(hookType string) (bool, error) {
	gitDir, err := a.GitDir()
	if err != nil {
		return false, fmt.Errorf("git: hook exists %s: git dir: %w", hookType, err)
	}
	return hookExists(filepath.Join(gitDir, "hooks"), hookType)
}

// validRef matches safe git ref characters: alphanumeric, dots, dashes, slashes, underscores.
var validRef = regexp.MustCompile(`^[a-zA-Z0-9._\-/^~]+$`)

// validateRef rejects refs that look like flags or contain unsafe characters.
func validateRef(ref string) error {
	if ref == "" {
		return fmt.Errorf("git: ref cannot be empty")
	}
	if strings.HasPrefix(ref, "-") {
		return fmt.Errorf("git: invalid ref %q: must not start with '-'", ref)
	}
	if strings.Contains(ref, "..") {
		return fmt.Errorf("git: invalid ref %q: must not contain '..'", ref)
	}
	if !validRef.MatchString(ref) {
		return fmt.Errorf("git: invalid ref %q: contains unsafe characters", ref)
	}
	return nil
}

func (a *Adapter) CommitRange(from, to string) ([]string, error) {
	if to == "" {
		to = "HEAD"
	}
	if from == "" {
		tag, err := a.LatestTag()
		if err != nil {
			return nil, err
		}
		from = tag
	}
	if err := validateRef(from); err != nil {
		return nil, err
	}
	if err := validateRef(to); err != nil {
		return nil, err
	}
	out, err := a.run("log", "--format=%H", from+".."+to)
	if err != nil {
		return nil, fmt.Errorf("git: commit range %s..%s: %w", from, to, err)
	}
	if out == "" {
		return []string{}, nil
	}
	return strings.Split(out, "\n"), nil
}

func (a *Adapter) LatestTag() (string, error) {
	tag, err := a.run("describe", "--tags", "--abbrev=0")
	if err != nil {
		return "", fmt.Errorf("git: no tags found: create a tag first: git tag v0.1.0")
	}
	return tag, nil
}

// LogAll returns all commits in the repository.
func (a *Adapter) LogAll() ([]domain.CommitInfo, error) {
	out, err := a.run("log", "--all", "--format=%H%n%an%n%aI%n%B%x00")
	if err != nil {
		return nil, fmt.Errorf("git: log all: %w", err)
	}
	if out == "" {
		return nil, nil
	}

	entries := strings.Split(out, "\x00")
	var commits []domain.CommitInfo
	var skipped int
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		ci, err := parseLogOutput(entry)
		if err != nil {
			skipped++
			continue
		}
		commits = append(commits, *ci)
	}
	if skipped > 0 {
		fmt.Fprintf(os.Stderr, "warning: skipped %d unparseable git log entries\n", skipped)
	}
	return commits, nil
}

// LogAllWithLimit returns up to maxCommits from git log --all.
// If maxCommits <= 0, all commits are returned (no limit).
// This method is not part of the GitAdapter interface; use it directly
// on *Adapter when you need bounded output (e.g. doctor rebuild).
func (a *Adapter) LogAllWithLimit(maxCommits int) ([]domain.CommitInfo, error) {
	args := []string{"log", "--all", "--format=%H%n%an%n%aI%n%B%x00"}
	if maxCommits > 0 {
		args = append(args, fmt.Sprintf("--max-count=%d", maxCommits))
	}
	out, err := a.run(args...)
	if err != nil {
		return nil, fmt.Errorf("git: log all: %w", err)
	}
	if out == "" {
		return nil, nil
	}

	entries := strings.Split(out, "\x00")
	var commits []domain.CommitInfo
	var skipped int
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		ci, err := parseLogOutput(entry)
		if err != nil {
			skipped++
			continue
		}
		commits = append(commits, *ci)
	}
	if skipped > 0 {
		fmt.Fprintf(os.Stderr, "warning: skipped %d unparseable git log entries\n", skipped)
	}
	return commits, nil
}

// CurrentBranch returns the current branch name.
func (a *Adapter) CurrentBranch() (string, error) {
	branch, err := a.run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("git: current branch: %w", err)
	}
	return branch, nil
}
