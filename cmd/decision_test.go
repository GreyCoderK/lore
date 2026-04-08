// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bytes"
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

func TestRunCalibration_NotInitialized(t *testing.T) {
	dir := t.TempDir()
	testutil.Chdir(t, dir)

	restore := ui.SaveAndDisableColor()
	defer restore()

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	err := runCalibration(streams, nil)
	if err == nil {
		t.Fatal("expected error for uninitialized repo")
	}
	if !strings.Contains(errBuf.String(), "Lore not initialized") {
		t.Errorf("expected 'Lore not initialized' in stderr, got: %q", errBuf.String())
	}
}

func TestRunCalibration_NilStore(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	restore := ui.SaveAndDisableColor()
	defer restore()

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	// nil storePtr
	err := runCalibration(streams, nil)
	if err == nil {
		t.Fatal("expected error for nil store")
	}
	if !strings.Contains(err.Error(), "unavail") && !strings.Contains(err.Error(), "store") {
		t.Errorf("error = %q, want store unavailable message", err)
	}
}

func TestRunCalibration_NilStoreValue(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	restore := ui.SaveAndDisableColor()
	defer restore()

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	// storePtr points to nil
	var s domain.LoreStore
	err := runCalibration(streams, &s)
	if err == nil {
		t.Fatal("expected error for nil store value")
	}
}

func TestDecisionCmd_CalibrationNotInitialized(t *testing.T) {
	dir := t.TempDir()
	testutil.Chdir(t, dir)

	restore := ui.SaveAndDisableColor()
	defer restore()

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &out,
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	cfg := &config.Config{}
	var s domain.LoreStore
	cmd := newDecisionCmd(cfg, streams, &s)
	cmd.SetArgs([]string{"--calibration"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for uninitialized repo")
	}
	if !strings.Contains(errBuf.String(), "Lore not initialized") {
		t.Errorf("expected 'Lore not initialized' in stderr, got: %q", errBuf.String())
	}
}

func TestDecisionCmd_CalibrationNilStore(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	restore := ui.SaveAndDisableColor()
	defer restore()

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &out,
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	cfg := &config.Config{}
	var s domain.LoreStore
	cmd := newDecisionCmd(cfg, streams, &s)
	cmd.SetArgs([]string{"--calibration"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for nil store")
	}
}

func TestDecisionCmd_HasFlags(t *testing.T) {
	streams, _, _ := testStreams()
	cfg := &config.Config{}
	var s domain.LoreStore
	cmd := newDecisionCmd(cfg, streams, &s)

	if cmd.Use != "decision" {
		t.Errorf("Use = %q, want %q", cmd.Use, "decision")
	}

	explainFlag := cmd.Flag("explain")
	if explainFlag == nil {
		t.Error("expected --explain flag")
	}

	calibrationFlag := cmd.Flag("calibration")
	if calibrationFlag == nil {
		t.Error("expected --calibration flag")
	}
}

// Default (HEAD) in a non-git directory should fail on git log
func TestDecisionCmd_DefaultHEAD_NoGitRepo(t *testing.T) {
	dir := t.TempDir() // not a git repo
	testutil.Chdir(t, dir)

	restore := ui.SaveAndDisableColor()
	defer restore()

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{Out: &out, Err: &errBuf, In: strings.NewReader("")}
	cfg := &config.Config{}
	var s domain.LoreStore
	cmd := newDecisionCmd(cfg, streams, &s)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}
	if !strings.Contains(err.Error(), "log") {
		t.Errorf("error = %q, want git log error", err)
	}
}

// --explain with invalid ref in a git repo
func TestDecisionCmd_ExplainInvalidRef(t *testing.T) {
	dir := testutil.SetupGitRepo(t)
	testutil.Chdir(t, dir)

	restore := ui.SaveAndDisableColor()
	defer restore()

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{Out: &out, Err: &errBuf, In: strings.NewReader("")}
	cfg := &config.Config{}
	var s domain.LoreStore
	cmd := newDecisionCmd(cfg, streams, &s)
	cmd.SetArgs([]string{"--explain", "nonexistent-ref-abc123"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid git ref")
	}
	if !strings.Contains(err.Error(), "log") {
		t.Errorf("error = %q, want git log error", err)
	}
}

// --explain HEAD in a git repo with a commit should succeed
func TestDecisionCmd_ExplainHEAD_WithCommit(t *testing.T) {
	dir := testutil.SetupGitRepo(t)
	testutil.Chdir(t, dir)

	// Create an initial commit
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitAdd := exec.Command("git", "add", ".")
	gitAdd.Dir = dir
	if out, err := gitAdd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	gitCommit := exec.Command("git", "commit", "-m", "feat(api): initial commit")
	gitCommit.Dir = dir
	gitCommit.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "HOME="+dir)
	if out, err := gitCommit.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}

	restore := ui.SaveAndDisableColor()
	defer restore()

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{Out: &out, Err: &errBuf, In: strings.NewReader("")}
	cfg := &config.Config{
		Decision: config.DecisionConfig{
			ThresholdFull:    60,
			ThresholdReduced: 35,
			ThresholdSuggest: 15,
		},
	}
	var s domain.LoreStore
	cmd := newDecisionCmd(cfg, streams, &s)
	cmd.SetArgs([]string{"--explain", "HEAD"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Output should contain commit info and score
	stdout := out.String()
	if !strings.Contains(stdout, "initial commit") {
		t.Errorf("expected subject in output, got: %q", stdout)
	}
}

// Default (no flags) in a git repo with a commit
func TestDecisionCmd_Default_WithCommit(t *testing.T) {
	dir := testutil.SetupGitRepo(t)
	testutil.Chdir(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitAdd := exec.Command("git", "add", ".")
	gitAdd.Dir = dir
	if out, err := gitAdd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	gitCommit := exec.Command("git", "commit", "-m", "fix(auth): patch token validation")
	gitCommit.Dir = dir
	gitCommit.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "HOME="+dir)
	if out, err := gitCommit.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}

	restore := ui.SaveAndDisableColor()
	defer restore()

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{Out: &out, Err: &errBuf, In: strings.NewReader("")}
	cfg := &config.Config{
		Decision: config.DecisionConfig{
			ThresholdFull:    60,
			ThresholdReduced: 35,
			ThresholdSuggest: 15,
		},
	}
	var s domain.LoreStore
	cmd := newDecisionCmd(cfg, streams, &s)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stdout := out.String()
	if !strings.Contains(stdout, "patch token validation") {
		t.Errorf("expected subject in output, got: %q", stdout)
	}
}
