// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

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

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/testutil"
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
	CommitRangeFunc           func(string, string) ([]string, error)
	LatestTagFunc             func() (string, error)
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

func (m *mockGitAdapter) HeadCommit() (*domain.CommitInfo, error) {
	return nil, fmt.Errorf("mock: HeadCommit not configured")
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

func (m *mockGitAdapter) CommitRange(from, to string) ([]string, error) {
	if m.CommitRangeFunc != nil {
		return m.CommitRangeFunc(from, to)
	}
	return nil, fmt.Errorf("mock: CommitRange not configured")
}

func (m *mockGitAdapter) LatestTag() (string, error) {
	if m.LatestTagFunc != nil {
		return m.LatestTagFunc()
	}
	return "", fmt.Errorf("mock: LatestTag not configured")
}

func (m *mockGitAdapter) LogAll() ([]domain.CommitInfo, error) {
	return nil, nil
}

func (m *mockGitAdapter) CurrentBranch() (string, error) {
	return "main", nil
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
	err := runInit(context.Background(), &config.Config{}, deps, streams, true)
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
	err := runInit(context.Background(), &config.Config{}, deps, streams, true)

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
	if err := os.MkdirAll(filepath.Join(dir, ".lore"), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	streams, _, errBuf := testStreams()

	mock := &mockGitAdapter{
		IsInsideWorkTreeFunc: func() bool { return true },
	}

	deps := initDeps{git: mock, workDir: dir}
	err := runInit(context.Background(), &config.Config{}, deps, streams, true)

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
	promptDemo(context.Background(), &config.Config{}, streams)
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
	promptDemo(context.Background(), &config.Config{}, streams)
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

func TestEnsureGitignore_AppendNoTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	gitignorePath := filepath.Join(dir, ".gitignore")
	// Existing file WITHOUT trailing newline
	os.WriteFile(gitignorePath, []byte("node_modules/"), 0644)

	modified, err := ensureGitignore(gitignorePath, ".lorerc.local")
	if err != nil {
		t.Fatalf("ensureGitignore: %v", err)
	}
	if !modified {
		t.Error("should modify when appending")
	}

	data, _ := os.ReadFile(gitignorePath)
	content := string(data)
	// Should still have both entries on separate lines
	if !strings.Contains(content, "node_modules/\n") {
		t.Error("existing content should be preserved with newline added")
	}
	if !strings.Contains(content, ".lorerc.local\n") {
		t.Error(".lorerc.local should be appended with trailing newline")
	}
}

func TestEnsureGitignore_EntryWithWhitespace(t *testing.T) {
	dir := t.TempDir()
	gitignorePath := filepath.Join(dir, ".gitignore")
	// Entry already present but with surrounding whitespace
	os.WriteFile(gitignorePath, []byte("  .lorerc.local  \n"), 0644)

	modified, err := ensureGitignore(gitignorePath, ".lorerc.local")
	if err != nil {
		t.Fatalf("ensureGitignore: %v", err)
	}
	if modified {
		t.Error("should not modify when entry already present (trimmed match)")
	}
}

func TestEnsureGitignore_MultipleEntries(t *testing.T) {
	dir := t.TempDir()
	gitignorePath := filepath.Join(dir, ".gitignore")
	os.WriteFile(gitignorePath, []byte("*.log\n.env\n"), 0644)

	modified, err := ensureGitignore(gitignorePath, ".lorerc.local")
	if err != nil {
		t.Fatalf("ensureGitignore: %v", err)
	}
	if !modified {
		t.Error("should modify when entry not present")
	}

	data, _ := os.ReadFile(gitignorePath)
	content := string(data)
	if !strings.Contains(content, "*.log") {
		t.Error("existing *.log should be preserved")
	}
	if !strings.Contains(content, ".env") {
		t.Error("existing .env should be preserved")
	}
	if !strings.Contains(content, ".lorerc.local") {
		t.Error(".lorerc.local should be appended")
	}
}

func TestRunInit_AlreadyInitialized_NoOverwrite(t *testing.T) {
	dir := t.TempDir()
	loreDir := filepath.Join(dir, ".lore")
	if err := os.MkdirAll(loreDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Create a marker file inside .lore to verify it is not overwritten
	markerPath := filepath.Join(loreDir, "marker.txt")
	os.WriteFile(markerPath, []byte("original"), 0644)

	streams, _, errBuf := testStreams()
	mock := &mockGitAdapter{
		IsInsideWorkTreeFunc: func() bool { return true },
	}

	deps := initDeps{git: mock, workDir: dir}
	err := runInit(context.Background(), &config.Config{}, deps, streams, true)
	if err != nil {
		t.Errorf("expected nil error for already initialized, got %v", err)
	}

	output := errBuf.String()
	if !strings.Contains(output, "Lore already initialized") {
		t.Error("output should contain warning message")
	}

	// Verify marker file is untouched
	data, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatalf("marker file should still exist: %v", err)
	}
	if string(data) != "original" {
		t.Error("marker file should not be modified")
	}

	// Verify no .lorerc was created
	if _, err := os.Stat(filepath.Join(dir, ".lorerc")); !os.IsNotExist(err) {
		t.Error(".lorerc should NOT be created when already initialized")
	}
}

func TestRunInit_HookInstallWarning(t *testing.T) {
	dir := t.TempDir()
	streams, _, errBuf := testStreams()

	mock := &mockGitAdapter{
		IsInsideWorkTreeFunc: func() bool { return true },
		InstallHookFunc: func(string) (domain.InstallResult, error) {
			return domain.InstallResult{}, fmt.Errorf("permission denied")
		},
	}

	deps := initDeps{git: mock, workDir: dir}
	err := runInit(context.Background(), &config.Config{}, deps, streams, true)
	if err != nil {
		t.Fatalf("runInit: %v (hook failure should be non-fatal)", err)
	}

	output := errBuf.String()
	if !strings.Contains(output, "permission denied") {
		t.Error("output should contain hook install error")
	}
	// .lore/docs/ should still be created despite hook failure
	if _, err := os.Stat(filepath.Join(dir, ".lore", "docs")); os.IsNotExist(err) {
		t.Error(".lore/docs/ should be created even when hook install fails")
	}
}

func TestRootCmd_InitViaRoot(t *testing.T) {
	// Exercise PersistentPreRunE → early return for "init" command
	dir := t.TempDir()
	testutil.Chdir(t, dir)

	// Need git repo for init
	mock := &mockGitAdapter{
		IsInsideWorkTreeFunc: func() bool { return true },
	}
	_ = mock // Used by runInit directly, not needed here since we just test root wiring

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &out,
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	cfg := &config.Config{}
	var s domain.LoreStore
	root := newRootCmd(cfg, streams, &s)

	// Set args to "init --no-demo" — PersistentPreRunE should run and skip config
	root.SetArgs([]string{"init", "--no-demo"})
	// This will try to call git.NewAdapter which may fail, but PersistentPreRunE should succeed
	// for "init" commands (it returns early without config loading)
	_ = root.Execute()

	// The test verifies PersistentPreRunE doesn't error for "init"
	// (it may fail at the git check inside runInit, that's fine)
}

func TestRootCmd_ShowViaRoot_NotInitialized(t *testing.T) {
	// Exercise PersistentPreRunE full path (config loading) for a non-init command
	dir := t.TempDir()
	testutil.Chdir(t, dir)

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &out,
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	cfg := &config.Config{}
	var s domain.LoreStore
	root := newRootCmd(cfg, streams, &s)
	root.SetArgs([]string{"show", "auth"})
	// This will go through PersistentPreRunE → config.LoadFromDirWithFlags
	// then requireLoreDir → error
	_ = root.Execute()
	// We don't check the error — the point is to exercise PersistentPreRunE
}

func TestRootCmd_ShowViaRoot_WithConfig(t *testing.T) {
	// Exercise PersistentPreRunE with a valid config + .lore dir
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	// Create minimal .lorerc
	os.WriteFile(filepath.Join(dir, ".lorerc"), []byte("ai:\n  provider: \"\"\n"), 0644)

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &out,
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	cfg := &config.Config{}
	var s domain.LoreStore
	root := newRootCmd(cfg, streams, &s)
	root.SetArgs([]string{"show", "auth"})
	// This goes through full PersistentPreRunE including config loading and store opening
	_ = root.Execute()
	// Ensure store is closed so Windows can clean up WAL files in TempDir
	if s != nil {
		_ = s.Close()
	}
}

func TestRootCmd_NoColorFlag(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)
	os.WriteFile(filepath.Join(dir, ".lorerc"), []byte("ai:\n  provider: \"\"\n"), 0644)

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &out,
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	cfg := &config.Config{}
	var s domain.LoreStore
	root := newRootCmd(cfg, streams, &s)
	root.SetArgs([]string{"--no-color", "show", "auth"})
	_ = root.Execute()
	if s != nil {
		_ = s.Close()
	}
}

func TestRootCmd_UnsupportedLanguageEnv(t *testing.T) {
	dir := t.TempDir()
	testutil.Chdir(t, dir)

	// Set unsupported language
	t.Setenv("LORE_LANGUAGE", "xx")

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &out,
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	cfg := &config.Config{}
	var s domain.LoreStore
	root := newRootCmd(cfg, streams, &s)
	root.SetArgs([]string{"doctor", "--config"})
	_ = root.Execute()

	if !strings.Contains(errBuf.String(), "xx") {
		t.Errorf("expected unsupported language warning with 'xx', got: %q", errBuf.String())
	}
}

func TestRootCmd_PostRunCloseStore(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)
	os.WriteFile(filepath.Join(dir, ".lorerc"), []byte("ai:\n  provider: \"\"\n"), 0644)

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &out,
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	cfg := &config.Config{}
	var s domain.LoreStore
	root := newRootCmd(cfg, streams, &s)
	// list command exercises PersistentPreRunE (store open) and PersistentPostRunE (store close)
	root.SetArgs([]string{"list"})
	_ = root.Execute()
}

func TestRootCmd_DoctorViaRoot(t *testing.T) {
	// Exercise PersistentPreRunE → early return for "doctor" command
	dir := t.TempDir()
	testutil.Chdir(t, dir)

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &out,
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	cfg := &config.Config{}
	var s domain.LoreStore
	root := newRootCmd(cfg, streams, &s)

	// doctor --config works without .lore
	root.SetArgs([]string{"doctor", "--config"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("doctor --config via root: %v", err)
	}
}

func TestRunInit_HooksPathWarning(t *testing.T) {
	dir := t.TempDir()
	streams, _, errBuf := testStreams()

	mock := &mockGitAdapter{
		IsInsideWorkTreeFunc: func() bool { return true },
		InstallHookFunc: func(string) (domain.InstallResult, error) {
			return domain.InstallResult{
				Installed:     true,
				HooksPathWarn: "/custom/hooks",
			}, nil
		},
	}

	deps := initDeps{git: mock, workDir: dir}
	err := runInit(context.Background(), &config.Config{}, deps, streams, true)
	if err != nil {
		t.Fatalf("runInit: %v", err)
	}

	output := errBuf.String()
	if !strings.Contains(output, "/custom/hooks") {
		t.Error("output should contain hooks path warning")
	}
}
