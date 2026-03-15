package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/museigen/lore/internal/domain"
)

// Adapter implements domain.GitAdapter via exec.Command("git", ...).
type Adapter struct {
	workDir string
}

// compile-time check
var _ domain.GitAdapter = (*Adapter)(nil)

// New creates an Adapter for the given working directory (M4: canonical constructor name).
func New(workDir string) *Adapter {
	return &Adapter{workDir: workDir}
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

func (a *Adapter) CommitExists(ref string) (bool, error) {
	_, err := a.run("cat-file", "-t", ref)
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (a *Adapter) Log(ref string) (*domain.CommitInfo, error) {
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
	out, err := a.run("diff", ref+"^!", "--")
	if err != nil {
		return "", fmt.Errorf("git: diff %s: %w", ref, err)
	}
	return out, nil
}

func (a *Adapter) IsMergeCommit(ref string) (bool, error) {
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
	if os.Getenv("GIT_SEQUENCE_EDITOR") != "" {
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
