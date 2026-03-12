package git

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/museigen/lore/internal/domain"
)

// Adapter implements domain.GitAdapter via exec.Command("git", ...).
type Adapter struct {
	workDir string
}

// compile-time check
var _ domain.GitAdapter = (*Adapter)(nil)

func NewAdapter(workDir string) *Adapter {
	return &Adapter{workDir: workDir}
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

// Stubs for methods implemented in Story 2.1
func (a *Adapter) Diff(_ string) (string, error) {
	return "", fmt.Errorf("git: Diff: not implemented")
}

func (a *Adapter) Log(_ string) (*domain.CommitInfo, error) {
	return nil, fmt.Errorf("git: Log: not implemented")
}

func (a *Adapter) IsMergeCommit(_ string) (bool, error) {
	return false, fmt.Errorf("git: IsMergeCommit: not implemented")
}

func (a *Adapter) IsRebaseInProgress() (bool, error) {
	return false, fmt.Errorf("git: IsRebaseInProgress: not implemented")
}

func (a *Adapter) CommitMessageContains(_, _ string) (bool, error) {
	return false, fmt.Errorf("git: CommitMessageContains: not implemented")
}

func (a *Adapter) InstallHook(hookType string) error {
	return installHook(a.workDir, hookType)
}

func (a *Adapter) UninstallHook(hookType string) error {
	return uninstallHook(a.workDir, hookType)
}

func (a *Adapter) HookExists(hookType string) (bool, error) {
	return hookExists(a.workDir, hookType)
}
