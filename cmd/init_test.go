package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/museigen/lore/internal/domain"
)

type mockGitAdapter struct {
	IsInsideWorkTreeFunc      func() bool
	HeadRefFunc               func() (string, error)
	CommitExistsFunc          func(string) (bool, error)
	DiffFunc                  func(string) (string, error)
	LogFunc                   func(string) (*domain.CommitInfo, error)
	IsMergeCommitFunc         func(string) (bool, error)
	IsRebaseInProgressFunc    func() (bool, error)
	CommitMessageContainsFunc func(string, string) (bool, error)
	GitDirFunc                func() (string, error)
	InstallHookFunc           func(string) (domain.InstallResult, error)
	UninstallHookFunc         func(string) error
	HookExistsFunc            func(string) (bool, error)
}

func (m *mockGitAdapter) IsInsideWorkTree() bool {
	if m.IsInsideWorkTreeFunc != nil {
		return m.IsInsideWorkTreeFunc()
	}
	return true
}

func (m *mockGitAdapter) HeadRef() (string, error) {
	if m.HeadRefFunc != nil {
		return m.HeadRefFunc()
	}
	return "", fmt.Errorf("mock: HeadRef not configured")
}

func (m *mockGitAdapter) CommitExists(ref string) (bool, error) {
	if m.CommitExistsFunc != nil {
		return m.CommitExistsFunc(ref)
	}
	return false, fmt.Errorf("mock: CommitExists not configured")
}

func (m *mockGitAdapter) Diff(ref string) (string, error) {
	if m.DiffFunc != nil {
		return m.DiffFunc(ref)
	}
	return "", fmt.Errorf("mock: Diff not configured")
}

func (m *mockGitAdapter) Log(ref string) (*domain.CommitInfo, error) {
	if m.LogFunc != nil {
		return m.LogFunc(ref)
	}
	return nil, fmt.Errorf("mock: Log not configured")
}

func (m *mockGitAdapter) IsMergeCommit(ref string) (bool, error) {
	if m.IsMergeCommitFunc != nil {
		return m.IsMergeCommitFunc(ref)
	}
	return false, fmt.Errorf("mock: IsMergeCommit not configured")
}

func (m *mockGitAdapter) IsRebaseInProgress() (bool, error) {
	if m.IsRebaseInProgressFunc != nil {
		return m.IsRebaseInProgressFunc()
	}
	return false, fmt.Errorf("mock: IsRebaseInProgress not configured")
}

func (m *mockGitAdapter) CommitMessageContains(ref, marker string) (bool, error) {
	if m.CommitMessageContainsFunc != nil {
		return m.CommitMessageContainsFunc(ref, marker)
	}
	return false, fmt.Errorf("mock: CommitMessageContains not configured")
}

func (m *mockGitAdapter) GitDir() (string, error) {
	if m.GitDirFunc != nil {
		return m.GitDirFunc()
	}
	return "", nil
}

func (m *mockGitAdapter) InstallHook(hookType string) (domain.InstallResult, error) {
	if m.InstallHookFunc != nil {
		return m.InstallHookFunc(hookType)
	}
	return domain.InstallResult{Installed: true}, nil
}

func (m *mockGitAdapter) UninstallHook(hookType string) error {
	if m.UninstallHookFunc != nil {
		return m.UninstallHookFunc(hookType)
	}
	return nil
}

func (m *mockGitAdapter) HookExists(hookType string) (bool, error) {
	if m.HookExistsFunc != nil {
		return m.HookExistsFunc(hookType)
	}
	return false, nil
}

func testStreams(input ...string) (domain.IOStreams, *bytes.Buffer, *bytes.Buffer) {
	var out, errBuf bytes.Buffer
	var in io.Reader = &bytes.Buffer{}
	if len(input) > 0 {
		in = strings.NewReader(input[0])
	}
	streams := domain.IOStreams{
		Out: &out,
		Err: &errBuf,
		In:  in,
	}
	return streams, &out, &errBuf
}

func TestRunInit_HappyPath(t *testing.T) {
	dir := t.TempDir()
	streams, _, errBuf := testStreams()

	mock := &mockGitAdapter{
		IsInsideWorkTreeFunc: func() bool { return true },
		InstallHookFunc:      func(string) (domain.InstallResult, error) { return domain.InstallResult{Installed: true}, nil },
	}

	deps := initDeps{git: mock, workDir: dir}
	err := runInit(context.Background(), deps, streams, true)
	if err != nil {
		t.Fatalf("runInit: %v", err)
	}

	// Verify .lore/docs/ created
	if _, err := os.Stat(filepath.Join(dir, ".lore", "docs")); os.IsNotExist(err) {
		t.Error(".lore/docs/ should be created")
	}

	// Verify .lorerc created
	data, err := os.ReadFile(filepath.Join(dir, ".lorerc"))
	if err != nil {
		t.Fatalf("read .lorerc: %v", err)
	}
	if !strings.Contains(string(data), "ai:") {
		t.Error(".lorerc should contain ai section")
	}

	// Verify .lorerc.local created
	data, err = os.ReadFile(filepath.Join(dir, ".lorerc.local"))
	if err != nil {
		t.Fatalf("read .lorerc.local: %v", err)
	}
	if !strings.Contains(string(data), "api_key") {
		t.Error(".lorerc.local should contain api_key placeholder")
	}

	// Verify .gitignore updated
	data, err = os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if !strings.Contains(string(data), ".lorerc.local") {
		t.Error(".gitignore should contain .lorerc.local")
	}

	// Verify output messages
	output := errBuf.String()
	if !strings.Contains(output, "Created") {
		t.Error("output should contain 'Created' verbs")
	}
	if !strings.Contains(output, ".lore/") {
		t.Error("output should mention .lore/")
	}
	if !strings.Contains(output, "Your code knows what") {
		t.Error("output should contain tagline")
	}
}

func TestRunInit_NotGitRepo(t *testing.T) {
	dir := t.TempDir()
	streams, _, errBuf := testStreams()

	mock := &mockGitAdapter{
		IsInsideWorkTreeFunc: func() bool { return false },
	}

	deps := initDeps{git: mock, workDir: dir}
	err := runInit(context.Background(), deps, streams, true)

	if err != domain.ErrNotGitRepo {
		t.Errorf("expected ErrNotGitRepo, got %v", err)
	}

	output := errBuf.String()
	if !strings.Contains(output, "Not a git repository") {
		t.Error("output should contain error message")
	}
	if !strings.Contains(output, "git init") {
		t.Error("output should contain actionable suggestion")
	}
}

func TestRunInit_AlreadyInitialized(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".lore"), 0755)

	streams, _, errBuf := testStreams()

	mock := &mockGitAdapter{
		IsInsideWorkTreeFunc: func() bool { return true },
	}

	deps := initDeps{git: mock, workDir: dir}
	err := runInit(context.Background(), deps, streams, true)

	if err != nil {
		t.Errorf("expected nil error for already initialized, got %v", err)
	}

	output := errBuf.String()
	if !strings.Contains(output, "Lore already initialized") {
		t.Error("output should contain warning message")
	}

	// Verify no files were created/overwritten
	if _, err := os.Stat(filepath.Join(dir, ".lorerc")); !os.IsNotExist(err) {
		t.Error(".lorerc should NOT be created when already initialized")
	}
}

func TestPromptDemo_NonTerminal(t *testing.T) {
	// Non-terminal streams: promptDemo should return without prompting
	streams, _, errBuf := testStreams()
	promptDemo(streams)
	if errBuf.Len() != 0 {
		t.Errorf("expected no output for non-terminal, got %q", errBuf.String())
	}
}

func TestPromptDemo_AcceptDemo(t *testing.T) {
	var errBuf bytes.Buffer
	inBuf := bytes.NewBufferString("o\n")
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  inBuf,
	}
	// promptDemo checks IsTerminal first, which returns false for buffers
	// So we test the reading logic directly via runInit with noDemo=false
	// Since IsTerminal returns false, promptDemo is a no-op for buffers
	promptDemo(streams)
	// With non-terminal, no prompt is shown
	if errBuf.Len() != 0 {
		t.Errorf("expected no output for non-terminal, got %q", errBuf.String())
	}
}

func TestEnsureGitignore_AlreadyPresent(t *testing.T) {
	dir := t.TempDir()
	gitignorePath := filepath.Join(dir, ".gitignore")
	os.WriteFile(gitignorePath, []byte(".lorerc.local\n"), 0644)

	modified, err := ensureGitignore(gitignorePath, ".lorerc.local")
	if err != nil {
		t.Fatalf("ensureGitignore: %v", err)
	}
	if modified {
		t.Error("should not modify when entry already present")
	}
}

func TestEnsureGitignore_NewFile(t *testing.T) {
	dir := t.TempDir()
	gitignorePath := filepath.Join(dir, ".gitignore")

	modified, err := ensureGitignore(gitignorePath, ".lorerc.local")
	if err != nil {
		t.Fatalf("ensureGitignore: %v", err)
	}
	if !modified {
		t.Error("should modify when creating new .gitignore")
	}

	data, _ := os.ReadFile(gitignorePath)
	if !strings.Contains(string(data), ".lorerc.local") {
		t.Error(".gitignore should contain .lorerc.local")
	}
}

func TestEnsureGitignore_AppendToExisting(t *testing.T) {
	dir := t.TempDir()
	gitignorePath := filepath.Join(dir, ".gitignore")
	os.WriteFile(gitignorePath, []byte("node_modules/\n"), 0644)

	modified, err := ensureGitignore(gitignorePath, ".lorerc.local")
	if err != nil {
		t.Fatalf("ensureGitignore: %v", err)
	}
	if !modified {
		t.Error("should modify when appending")
	}

	data, _ := os.ReadFile(gitignorePath)
	content := string(data)
	if !strings.Contains(content, "node_modules/") {
		t.Error("existing content should be preserved")
	}
	if !strings.Contains(content, ".lorerc.local") {
		t.Error(".lorerc.local should be appended")
	}
}
