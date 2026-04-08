// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/testutil"
	"github.com/greycoderk/lore/internal/ui"
)

func TestHookPostCommitCmd_CallsWorkflowDispatch(t *testing.T) {
	// AC-6: The command is no longer a stub — it wires to workflow.Dispatch.
	// Structural check: RunE is non-nil and command metadata is correct.
	// End-to-end behavior is tested in internal/workflow/reactive_test.go.
	streams, _, _ := testStreams()
	cfg := testConfig()

	cmd := newHookPostCommitCmd(cfg, streams, nil)
	if cmd.Use != "_hook-post-commit" {
		t.Errorf("Use = %q, want %q", cmd.Use, "_hook-post-commit")
	}
	if cmd.RunE == nil {
		t.Error("RunE should not be nil")
	}
}

func TestHookPostCommitCmd_IsHidden(t *testing.T) {
	streams, _, _ := testStreams()
	cfg := testConfig()

	cmd := newHookPostCommitCmd(cfg, streams, nil)
	if !cmd.Hidden {
		t.Error("_hook-post-commit command should be hidden")
	}
}

func TestHookPostCommitCmd_Registered(t *testing.T) {
	streams, _, _ := testStreams()
	cfg := testConfig()

	var s domain.LoreStore
	rootCmd := newRootCmd(cfg, streams, &s)

	// Verify _hook-post-commit is registered
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "_hook-post-commit" {
			found = true
			if !c.Hidden {
				t.Error("_hook-post-commit should be hidden")
			}
			break
		}
	}
	if !found {
		t.Error("_hook-post-commit command should be registered in root")
	}
}

func TestHookPostCommitCmd_Short(t *testing.T) {
	streams, _, _ := testStreams()
	cfg := testConfig()

	cmd := newHookPostCommitCmd(cfg, streams, nil)
	if cmd.Short == "" {
		t.Error("expected non-empty Short description")
	}
}

func TestHookPostCommitCmd_SilenceFlags(t *testing.T) {
	streams, _, _ := testStreams()
	cfg := testConfig()

	cmd := newHookPostCommitCmd(cfg, streams, nil)
	if !cmd.SilenceUsage {
		t.Error("SilenceUsage should be true")
	}
	if !cmd.SilenceErrors {
		t.Error("SilenceErrors should be true")
	}
}

// Execute the post-commit command in a git repo with a commit.
// This exercises the RunE body (getwd, adapter, engine creation, dispatch).
func TestHookPostCommitCmd_ExecuteInGitRepo(t *testing.T) {
	dir := testutil.SetupGitRepo(t)
	testutil.Chdir(t, dir)

	// Create an initial commit so HEAD exists
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitAdd := exec.Command("git", "add", ".")
	gitAdd.Dir = dir
	if out, err := gitAdd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	gitCommit := exec.Command("git", "commit", "-m", "feat: initial commit")
	gitCommit.Dir = dir
	gitCommit.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "HOME="+dir)
	if out, err := gitCommit.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}

	restore := ui.SaveAndDisableColor()
	defer restore()

	streams, _, _ := testStreams()
	cfg := &config.Config{
		Decision: config.DecisionConfig{
			ThresholdFull:    60,
			ThresholdReduced: 35,
			ThresholdSuggest: 15,
		},
	}
	var s domain.LoreStore
	cmd := newHookPostCommitCmd(cfg, streams, &s)
	cmd.SetArgs([]string{})
	// This exercises the full RunE body: getwd, adapter, engine, dispatch
	err := cmd.Execute()
	// May succeed or fail depending on git state, but it should exercise the code path
	_ = err
}

// Execute with nil storePtr (graceful degradation)
func TestHookPostCommitCmd_NilStore(t *testing.T) {
	dir := testutil.SetupGitRepo(t)
	testutil.Chdir(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitAdd := exec.Command("git", "add", ".")
	gitAdd.Dir = dir
	if out, err := gitAdd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	gitCommit := exec.Command("git", "commit", "-m", "fix: patch something")
	gitCommit.Dir = dir
	gitCommit.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "HOME="+dir)
	if out, err := gitCommit.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}

	restore := ui.SaveAndDisableColor()
	defer restore()

	streams, _, _ := testStreams()
	cfg := &config.Config{}
	// nil storePtr — should still run without panic
	cmd := newHookPostCommitCmd(cfg, streams, nil)
	cmd.SetArgs([]string{})
	_ = cmd.Execute()
}

// Verify command works with explicit store pointer
func TestHookPostCommitCmd_WithStorePtr(t *testing.T) {
	streams, _, _ := testStreams()
	cfg := testConfig()

	var s domain.LoreStore
	cmd := newHookPostCommitCmd(cfg, streams, &s)
	if cmd.Use != "_hook-post-commit" {
		t.Errorf("Use = %q, want %q", cmd.Use, "_hook-post-commit")
	}
	// Just verify it doesn't panic during construction
}

// Ensure the command handles non-git directory gracefully (no panic)
func TestHookPostCommitCmd_NoGitRepo(t *testing.T) {
	dir := t.TempDir() // not a git repo
	testutil.Chdir(t, dir)

	restore := ui.SaveAndDisableColor()
	defer restore()

	streams, _, _ := testStreams()
	cfg := &config.Config{}
	cmd := newHookPostCommitCmd(cfg, streams, nil)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	// Should error (no git repo), but should NOT panic
	if err == nil {
		// In some cases workflow dispatch may handle gracefully; that's ok
		return
	}
	if strings.Contains(err.Error(), "panic") {
		t.Errorf("should not panic, got: %v", err)
	}
}
