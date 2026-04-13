// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/testutil"
)

// ─────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────

// seedReviewState writes a review state file containing one active finding
// and returns the full hash.
func seedReviewState(t *testing.T, statePath string) (fullHash string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		t.Fatalf("seedReviewState mkdir: %v", err)
	}
	finding := angela.ReviewFinding{
		Title:       "Test finding",
		Severity:    "gap",
		Description: "Something is missing.",
	}
	hash := angela.ReviewFindingHash(finding)

	state := &angela.ReviewState{
		Version: angela.ReviewStateVersion,
		Findings: map[string]angela.StatefulFinding{
			hash: {
				Finding:   finding,
				Status:    angela.StatusActive,
				FirstSeen: time.Now().Add(-time.Hour),
				LastSeen:  time.Now(),
			},
		},
	}
	if err := angela.SaveReviewState(statePath, state); err != nil {
		t.Fatalf("seedReviewState save: %v", err)
	}
	return hash
}

// streams returns a fresh IOStreams backed by bytes.Buffer.
func makeStreams() (domain.IOStreams, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	return domain.IOStreams{Out: out, Err: errBuf, In: &bytes.Buffer{}}, out, errBuf
}

// ─────────────────────────────────────────────────────────────────
// sanitizeResolvedBy
// ─────────────────────────────────────────────────────────────────

func TestSanitizeResolvedBy_Normal(t *testing.T) {
	if got := sanitizeResolvedBy("alice"); got != "alice" {
		t.Errorf("got %q, want alice", got)
	}
}

func TestSanitizeResolvedBy_Empty(t *testing.T) {
	if got := sanitizeResolvedBy(""); got != "unknown" {
		t.Errorf("got %q, want unknown", got)
	}
}

func TestSanitizeResolvedBy_OnlyWhitespace(t *testing.T) {
	if got := sanitizeResolvedBy("   "); got != "unknown" {
		t.Errorf("got %q, want unknown", got)
	}
}

func TestSanitizeResolvedBy_ControlCharsStripped(t *testing.T) {
	// \x1b (ESC) and \n are control chars (< 0x20); other chars remain.
	input := "ali\x1b[0mce\nmalicious"
	got := sanitizeResolvedBy(input)
	if strings.ContainsAny(got, "\x1b\n") {
		t.Errorf("control chars not stripped: %q", got)
	}
	// "ali", "[0mce", "malicious" should all survive (only ESC and LF removed)
	if !strings.Contains(got, "ali") {
		t.Errorf("clean prefix removed: %q", got)
	}
}

func TestSanitizeResolvedBy_LongInputTruncated(t *testing.T) {
	long := strings.Repeat("a", 200)
	got := sanitizeResolvedBy(long)
	if len([]rune(got)) > 64 {
		t.Errorf("expected max 64 runes, got %d", len([]rune(got)))
	}
}

// ─────────────────────────────────────────────────────────────────
// reviewStatePath
// ─────────────────────────────────────────────────────────────────

func TestReviewStatePath_DefaultFilename(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	cfg := &config.Config{}
	path, err := reviewStatePath(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(path, "review-state.json") {
		t.Errorf("expected review-state.json suffix, got %q", path)
	}
}

func TestReviewStatePath_CustomFilename(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	cfg := &config.Config{}
	cfg.Angela.Review.Differential.StateFile = "custom-state.json"
	path, err := reviewStatePath(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(path, "custom-state.json") {
		t.Errorf("expected custom-state.json suffix, got %q", path)
	}
}

func TestReviewStatePath_PathTraversalRejected(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	cfg := &config.Config{}
	cfg.Angela.Review.Differential.StateFile = "../escape.json"
	_, err := reviewStatePath(cfg)
	if err == nil {
		t.Error("expected error for path traversal, got nil")
	}
}

// ─────────────────────────────────────────────────────────────────
// lore angela review resolve
// ─────────────────────────────────────────────────────────────────

func TestAngelaReviewResolve_HappyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	cfg := &config.Config{}
	statePath, _ := reviewStatePath(cfg)
	fullHash := seedReviewState(t, statePath)

	streams, _, errBuf := makeStreams()
	cmd := newAngelaReviewResolveCmd(cfg, streams)
	cmd.SetArgs([]string{fullHash[:6]})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !strings.Contains(errBuf.String(), "resolved") {
		t.Errorf("expected 'resolved' in output, got: %s", errBuf.String())
	}

	// Verify state on disk.
	state, err := angela.LoadReviewState(statePath)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	entry, ok := state.Findings[fullHash]
	if !ok {
		t.Fatal("finding not in state after resolve")
	}
	if entry.Status != angela.StatusResolved {
		t.Errorf("status = %q, want %q", entry.Status, angela.StatusResolved)
	}
}

func TestAngelaReviewResolve_UnknownHash(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	cfg := &config.Config{}
	statePath, _ := reviewStatePath(cfg)
	seedReviewState(t, statePath)

	streams, _, _ := makeStreams()
	cmd := newAngelaReviewResolveCmd(cfg, streams)
	cmd.SetArgs([]string{"000000"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for unknown hash")
	}
}

func TestAngelaReviewResolve_EmptyState(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	cfg := &config.Config{}
	streams, _, _ := makeStreams()
	cmd := newAngelaReviewResolveCmd(cfg, streams)
	cmd.SetArgs([]string{"abcdef"})
	// No state file exists — should error cleanly (not panic).
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error resolving from empty state")
	}
}

// ─────────────────────────────────────────────────────────────────
// lore angela review ignore
// ─────────────────────────────────────────────────────────────────

func TestAngelaReviewIgnore_HappyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	cfg := &config.Config{}
	statePath, _ := reviewStatePath(cfg)
	fullHash := seedReviewState(t, statePath)

	streams, _, errBuf := makeStreams()
	cmd := newAngelaReviewIgnoreCmd(cfg, streams)
	cmd.SetArgs([]string{fullHash[:6], "--reason", "intentional design"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("ignore: %v", err)
	}
	if !strings.Contains(errBuf.String(), "ignored") {
		t.Errorf("expected 'ignored' in output, got: %s", errBuf.String())
	}

	state, err := angela.LoadReviewState(statePath)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	entry := state.Findings[fullHash]
	if entry.Status != angela.StatusIgnored {
		t.Errorf("status = %q, want %q", entry.Status, angela.StatusIgnored)
	}
	if entry.IgnoreReason != "intentional design" {
		t.Errorf("ignore reason = %q, want 'intentional design'", entry.IgnoreReason)
	}
}

func TestAngelaReviewIgnore_MissingReason(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	cfg := &config.Config{}
	statePath, _ := reviewStatePath(cfg)
	fullHash := seedReviewState(t, statePath)

	streams, _, _ := makeStreams()
	cmd := newAngelaReviewIgnoreCmd(cfg, streams)
	cmd.SetArgs([]string{fullHash[:6]}) // no --reason
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when --reason is missing")
	}
}

func TestAngelaReviewIgnore_EmptyReason(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	cfg := &config.Config{}
	statePath, _ := reviewStatePath(cfg)
	fullHash := seedReviewState(t, statePath)

	streams, _, _ := makeStreams()
	cmd := newAngelaReviewIgnoreCmd(cfg, streams)
	cmd.SetArgs([]string{fullHash[:6], "--reason", "   "}) // blank reason
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for blank --reason")
	}
}

// ─────────────────────────────────────────────────────────────────
// lore angela review log
// ─────────────────────────────────────────────────────────────────

func TestAngelaReviewLog_EmptyState(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	cfg := &config.Config{}
	streams, _, errBuf := makeStreams()
	cmd := newAngelaReviewLogCmd(cfg, streams)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("log empty: %v", err)
	}
	// Should print the "no findings" hint.
	if !strings.Contains(errBuf.String(), "no findings") {
		t.Errorf("expected 'no findings' message, got: %s", errBuf.String())
	}
}

func TestAngelaReviewLog_HumanFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	cfg := &config.Config{}
	statePath, _ := reviewStatePath(cfg)
	seedReviewState(t, statePath)

	streams, out, _ := makeStreams()
	cmd := newAngelaReviewLogCmd(cfg, streams)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("log: %v", err)
	}
	// Human table should have hash column header.
	if !strings.Contains(out.String(), "hash") {
		t.Errorf("expected 'hash' header in human output, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "Test finding") {
		t.Errorf("expected finding title in output, got: %s", out.String())
	}
}

func TestAngelaReviewLog_JSONFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	cfg := &config.Config{}
	statePath, _ := reviewStatePath(cfg)
	seedReviewState(t, statePath)

	streams, out, _ := makeStreams()
	cmd := newAngelaReviewLogCmd(cfg, streams)
	cmd.SetArgs([]string{"--format", "json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("log json: %v", err)
	}

	var entries []angela.StatefulFinding
	if err := json.Unmarshal(out.Bytes(), &entries); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, out.String())
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Finding.Title != "Test finding" {
		t.Errorf("unexpected title: %q", entries[0].Finding.Title)
	}
}

func TestAngelaReviewLog_JSONFormat_EmptyState(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	cfg := &config.Config{}
	streams, out, _ := makeStreams()
	cmd := newAngelaReviewLogCmd(cfg, streams)
	cmd.SetArgs([]string{"--format", "json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("log json empty: %v", err)
	}
	// Should produce empty JSON array, not null.
	trimmed := strings.TrimSpace(out.String())
	if trimmed != "[]" {
		t.Errorf("expected [] for empty state, got: %s", trimmed)
	}
}
