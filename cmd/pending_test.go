// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/testutil"
	"github.com/greycoderk/lore/internal/ui"
	"github.com/greycoderk/lore/internal/workflow"
	"gopkg.in/yaml.v3"
)

func writePending(t *testing.T, dir string, record workflow.PendingRecord) {
	t.Helper()
	pendingDir := filepath.Join(dir, ".lore", "pending")
	if err := os.MkdirAll(pendingDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	name := record.Commit
	if name == "" {
		name = "unknown"
	}
	data, err := yaml.Marshal(record)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pendingDir, name+".yaml"), data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestPendingList_NoPending(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	restore := ui.SaveAndDisableColor()
	defer restore()

	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  &bytes.Buffer{},
	}

	cmd := newPendingCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("pending list: %v", err)
	}

	if !strings.Contains(errBuf.String(), "No pending documentation") {
		t.Errorf("expected 'No pending documentation' message, got: %q", errBuf.String())
	}
}

func TestPendingList_WithItems(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	restore := ui.SaveAndDisableColor()
	defer restore()

	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	writePending(t, dir, workflow.PendingRecord{
		Commit:  "abc1234deadbeef",
		Date:    time.Now().UTC().Add(-2 * 24 * time.Hour).Format(time.RFC3339),
		Message: "feat(auth): add JWT middleware",
		Answers: workflow.PendingAnswers{Type: "feature", What: "add JWT"},
		Status:  "partial",
		Reason:  "interrupted",
	})

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  &bytes.Buffer{},
	}

	cmd := newPendingCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("pending list: %v", err)
	}

	output := errBuf.String()
	if !strings.Contains(output, "Pending documentation:") {
		t.Errorf("expected header, got: %q", output)
	}
	if !strings.Contains(output, "abc1234") {
		t.Errorf("expected short hash in output, got: %q", output)
	}
	if !strings.Contains(output, "2/5") {
		t.Errorf("expected progress 2/5, got: %q", output)
	}
}

func TestPendingList_Quiet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	writePending(t, dir, workflow.PendingRecord{
		Commit:  "abc1234",
		Date:    time.Now().UTC().Format(time.RFC3339),
		Message: "feat: something",
		Status:  "partial",
		Reason:  "interrupted",
	})

	var outBuf, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &outBuf,
		Err: &errBuf,
		In:  &bytes.Buffer{},
	}

	cmd := newPendingCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{"list", "--quiet"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("pending list --quiet: %v", err)
	}

	// Machine output on stdout
	stdout := outBuf.String()
	if !strings.Contains(stdout, "abc1234") {
		t.Errorf("expected hash on stdout, got: %q", stdout)
	}
	if !strings.Contains(stdout, "0/5") {
		t.Errorf("expected progress on stdout, got: %q", stdout)
	}

	// No human messages on stderr
	if errBuf.Len() > 0 {
		t.Errorf("expected no stderr in quiet mode, got: %q", errBuf.String())
	}
}

func TestPendingSkip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	restore := ui.SaveAndDisableColor()
	defer restore()

	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	writePending(t, dir, workflow.PendingRecord{
		Commit:  "skip1234",
		Date:    time.Now().UTC().Format(time.RFC3339),
		Message: "feat: skip this",
		Status:  "partial",
		Reason:  "interrupted",
	})

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  &bytes.Buffer{},
	}

	cmd := newPendingCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{"skip", "skip1234"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("pending skip: %v", err)
	}

	output := errBuf.String()
	if !strings.Contains(output, "Skipped") {
		t.Errorf("expected 'Skipped' in output, got: %q", output)
	}

	// File should be gone
	if _, statErr := os.Stat(filepath.Join(dir, ".lore", "pending", "skip1234.yaml")); !os.IsNotExist(statErr) {
		t.Error("pending file should have been deleted")
	}
}

func TestPendingNotInitialized(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	restore := ui.SaveAndDisableColor()
	defer restore()

	dir := t.TempDir()
	testutil.Chdir(t, dir)

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  &bytes.Buffer{},
	}

	cmd := newPendingCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for uninitialised repo")
	}

	if !strings.Contains(errBuf.String(), "Lore not initialized") {
		t.Errorf("expected init error message, got: %q", errBuf.String())
	}
}

// --- pending resolve tests ---

func TestPendingResolve_NoPending(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	restore := ui.SaveAndDisableColor()
	defer restore()

	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  &bytes.Buffer{},
	}

	cmd := newPendingCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{"resolve"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected nil error for no pending, got: %v", err)
	}

	if !strings.Contains(errBuf.String(), "No pending") {
		t.Errorf("expected 'No pending' message, got: %q", errBuf.String())
	}
}

func TestPendingResolve_InvalidSelectionNumber(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	restore := ui.SaveAndDisableColor()
	defer restore()

	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	writePending(t, dir, workflow.PendingRecord{
		Commit:  "abc1234deadbeef",
		Date:    time.Now().UTC().Format(time.RFC3339),
		Message: "feat(auth): add JWT",
		Status:  "partial",
		Reason:  "interrupted",
	})

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  &bytes.Buffer{},
	}

	cmd := newPendingCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{"resolve", "99"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid selection number")
	}

	if !strings.Contains(errBuf.String(), "99") {
		t.Errorf("expected invalid selection '99' echoed in error, got: %q", errBuf.String())
	}
}

func TestPendingResolve_InvalidSelectionZero(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	restore := ui.SaveAndDisableColor()
	defer restore()

	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	writePending(t, dir, workflow.PendingRecord{
		Commit:  "def5678deadbeef",
		Date:    time.Now().UTC().Format(time.RFC3339),
		Message: "fix: something",
		Status:  "partial",
		Reason:  "interrupted",
	})

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  &bytes.Buffer{},
	}

	cmd := newPendingCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{"resolve", "0"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for selection 0")
	}

	if !strings.Contains(errBuf.String(), "0") {
		t.Errorf("expected '0' in error message, got: %q", errBuf.String())
	}
}

func TestPendingResolve_InvalidSelectionNonNumeric(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	restore := ui.SaveAndDisableColor()
	defer restore()

	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	writePending(t, dir, workflow.PendingRecord{
		Commit:  "ghi9012deadbeef",
		Date:    time.Now().UTC().Format(time.RFC3339),
		Message: "chore: cleanup",
		Status:  "partial",
		Reason:  "interrupted",
	})

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  &bytes.Buffer{},
	}

	cmd := newPendingCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{"resolve", "abc"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-numeric selection")
	}
}

func TestPendingResolve_NotInitialized(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	restore := ui.SaveAndDisableColor()
	defer restore()

	dir := t.TempDir()
	testutil.Chdir(t, dir)

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  &bytes.Buffer{},
	}

	cmd := newPendingCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{"resolve"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for uninitialized repo")
	}

	if !strings.Contains(errBuf.String(), "Lore not initialized") {
		t.Errorf("expected init error, got: %q", errBuf.String())
	}
}

func TestPendingResolve_CommitFilterNoMatch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	restore := ui.SaveAndDisableColor()
	defer restore()

	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	writePending(t, dir, workflow.PendingRecord{
		Commit:  "abc1234deadbeef",
		Date:    time.Now().UTC().Format(time.RFC3339),
		Message: "feat: something",
		Status:  "partial",
		Reason:  "interrupted",
	})

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  &bytes.Buffer{},
	}

	cmd := newPendingCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{"resolve", "--commit", "zzz9999"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for no matching commit")
	}

	if !strings.Contains(errBuf.String(), "zzz9999") {
		t.Errorf("expected commit filter in error message, got: %q", errBuf.String())
	}
}

func TestPendingSkip_NotInitialized(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	restore := ui.SaveAndDisableColor()
	defer restore()

	dir := t.TempDir()
	testutil.Chdir(t, dir)

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  &bytes.Buffer{},
	}

	cmd := newPendingCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{"skip", "abc123"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for uninitialized repo")
	}
	if !strings.Contains(errBuf.String(), "Lore not initialized") {
		t.Errorf("expected init error, got: %q", errBuf.String())
	}
}

func TestPendingSkip_NoMatchingHash(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	restore := ui.SaveAndDisableColor()
	defer restore()

	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  &bytes.Buffer{},
	}

	cmd := newPendingCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{"skip", "nonexistent"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-matching hash")
	}
}

func TestPendingListQuiet_NotInitialized(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := t.TempDir()
	testutil.Chdir(t, dir)

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  &bytes.Buffer{},
	}

	cmd := newPendingCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{"list", "--quiet"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for uninitialized repo")
	}
	if !strings.Contains(errBuf.String(), "Lore not initialized") {
		t.Errorf("expected init error, got: %q", errBuf.String())
	}
}

func TestPendingListQuiet_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	var outBuf, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &outBuf,
		Err: &errBuf,
		In:  &bytes.Buffer{},
	}

	cmd := newPendingCmd(&config.Config{}, streams)
	cmd.SetArgs([]string{"list", "--quiet"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("pending list --quiet empty: %v", err)
	}

	// No items → no output
	if outBuf.Len() != 0 {
		t.Errorf("expected no stdout for empty list, got: %q", outBuf.String())
	}
	if errBuf.Len() != 0 {
		t.Errorf("expected no stderr in quiet mode, got: %q", errBuf.String())
	}
}

func TestShortHash(t *testing.T) {
	if got := shortHash("abc1234deadbeef"); got != "abc1234" {
		t.Errorf("shortHash = %q, want abc1234", got)
	}
	if got := shortHash("abc"); got != "abc" {
		t.Errorf("shortHash = %q, want abc", got)
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("short", 40); got != "short" {
		t.Errorf("truncate = %q", got)
	}
	long := strings.Repeat("a", 50)
	if got := truncate(long, 40); len(got) != 40 || !strings.HasSuffix(got, "...") {
		t.Errorf("truncate = %q (len %d)", got, len(got))
	}
}

func TestTruncate_VeryShortMax(t *testing.T) {
	// maxLen <= 3: no room for "...", just truncate
	if got := truncate("abcdef", 2); got != "ab" {
		t.Errorf("truncate(2) = %q, want %q", got, "ab")
	}
	if got := truncate("abcdef", 3); got != "abc" {
		t.Errorf("truncate(3) = %q, want %q", got, "abc")
	}
}

func TestTruncate_ExactLength(t *testing.T) {
	if got := truncate("abcd", 4); got != "abcd" {
		t.Errorf("truncate exact = %q, want %q", got, "abcd")
	}
}
