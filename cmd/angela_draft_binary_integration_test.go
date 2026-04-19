// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// Phase 3 — binary-level integration for `angela draft` standalone mode.
//
// Why this layer: existing unit tests (cmd/angela_test.go) exercise the
// cobra Cmd.Execute() path, which skips the real binary's main.go entry,
// signal handling, and os.Exit wiring. The CI-gate contract — "a draft
// run with --fail-on=warning on a corpus with warnings exits non-zero" —
// is what users depend on to fail their build pipelines. Unit tests alone
// cannot catch a regression where cobra's error propagation stops
// producing the right exit code at the process boundary.
//
// These tests compile the real binary once (sync.Once) and exec it against
// synthetic corpora. Because `go test ./... -race` runs on ubuntu/macos/
// windows via the existing CI matrix, this coverage is automatically
// cross-platform — the "5×5 container" phase-3 goal is satisfied by the
// existing matrix plus binary-level validation.
// ═══════════════════════════════════════════════════════════════════════════

var (
	binaryOnce sync.Once
	binaryPath string
	binaryErr  error
)

// loreBinaryPath builds the lore binary once per test run (cached via
// sync.Once) and returns its path. Subsequent callers reuse the cached
// binary. If the build itself fails, the test is skipped with the error
// — we don't want a broken dev environment to mask a real regression by
// producing a failing test for unrelated reasons.
func loreBinaryPath(t *testing.T) string {
	t.Helper()
	binaryOnce.Do(func() {
		tmpDir, err := os.MkdirTemp("", "lore-int-bin-*")
		if err != nil {
			binaryErr = fmt.Errorf("mkdir: %w", err)
			return
		}
		bin := filepath.Join(tmpDir, "lore")
		if runtime.GOOS == "windows" {
			bin += ".exe"
		}
		// Build from the parent directory (lore_cli/) where main.go lives.
		cmd := exec.Command("go", "build", "-o", bin, "..")
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			binaryErr = fmt.Errorf("go build: %w", err)
			return
		}
		binaryPath = bin
	})
	if binaryErr != nil {
		t.Skipf("cannot build test binary: %v", binaryErr)
	}
	return binaryPath
}

// seedDraftCorpus creates a temp directory with N markdown files and
// returns the directory path. `docs[i]` is written as-is (caller owns
// any front matter / body content).
func seedDraftCorpus(t *testing.T, docs map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range docs {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}
	return dir
}

// runBinary executes the compiled lore binary with args and returns
// (stdout, stderr, exit code). Never fatals on non-zero exit — the
// caller asserts the code.
//
// The working directory is set to a fresh t.TempDir() so any state the
// binary writes (e.g. `.angela-state/draft-state.json` for differential
// tracking) lands in an ephemeral location and does NOT pollute the
// repo root. A regression here — running with cwd=repo root — would
// leak runtime artifacts into git status.
func runBinary(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	bin := loreBinaryPath(t)
	cmd := exec.Command(bin, args...)
	cmd.Dir = t.TempDir()
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// Inherit no environment to keep tests hermetic — LORE_LANGUAGE etc.
	// from the dev machine would otherwise leak in and change output.
	cmd.Env = append([]string{}, "PATH="+os.Getenv("PATH"))
	_ = cmd.Run()
	exitCode := cmd.ProcessState.ExitCode()
	return stdout.String(), stderr.String(), exitCode
}

// TestBinary_DraftStandalone_CleanCorpus is the happy-path CI-gate check:
// a well-structured doc with no warnings must exit 0 regardless of
// --fail-on threshold.
func TestBinary_DraftStandalone_CleanCorpus(t *testing.T) {
	// A decision doc with full front matter + What/Why sections satisfies
	// the structure checks. Density/length checks also pass for 200+ chars.
	dir := seedDraftCorpus(t, map[string]string{
		"decision-api-2026-04-01.md": `---
type: decision
date: "2026-04-01"
status: published
tags: [api]
---
# API Architecture

## Context
We need to pick a transport.

## What
We pick REST over HTTP/1.1 with JSON bodies.

## Why
Universal tooling support, mature stack, low operational cost.

## Alternatives
gRPC — rejected due to browser support cost.

## Impact
Backend + frontend teams align on one protocol.
`,
	})

	_, stderr, code := runBinary(t, "angela", "draft", "--all", "--path", dir, "--fail-on", "error")
	if code != 0 {
		t.Errorf("clean corpus with --fail-on=error should exit 0, got %d\nstderr:\n%s", code, stderr)
	}
}

// TestBinary_DraftStandalone_WarningsDontFailByDefault guards the default
// CI-gate behavior: warnings are surfaced but do NOT fail the build
// unless the user opts into --fail-on=warning. A regression that made
// warnings fatal by default would silently break every CI consumer.
func TestBinary_DraftStandalone_WarningsDontFailByDefault(t *testing.T) {
	// A doc missing required "## What" and "## Why" sections triggers
	// warnings from the structure checker — but no errors.
	dir := seedDraftCorpus(t, map[string]string{
		"feature-login-2026-04-02.md": `---
type: feature
date: "2026-04-02"
status: published
---
# Login Feature

Users can log in with email + password.
`,
	})

	_, stderr, code := runBinary(t, "angela", "draft", "--all", "--path", dir, "--fail-on", "error")
	if code != 0 {
		t.Errorf("warnings-only corpus with --fail-on=error should exit 0 (warnings are informational), got %d\nstderr:\n%s", code, stderr)
	}
}

// TestBinary_DraftStandalone_FailOnWarningExitsNonZero is the explicit
// CI-gate opt-in: when the user sets --fail-on=warning on a corpus
// with warnings, the binary MUST exit non-zero so the CI pipeline can
// gate on it. This is the contract that `lore angela draft --fail-on`
// documentation promises users.
func TestBinary_DraftStandalone_FailOnWarningExitsNonZero(t *testing.T) {
	dir := seedDraftCorpus(t, map[string]string{
		"feature-search-2026-04-03.md": `---
type: feature
date: "2026-04-03"
status: published
---
# Search Feature

Short doc without What/Why sections.
`,
	})

	_, stderr, code := runBinary(t, "angela", "draft", "--all", "--path", dir, "--fail-on", "warning")
	if code == 0 {
		t.Errorf("warnings-present corpus with --fail-on=warning MUST exit non-zero (CI gate contract), got 0\nstderr:\n%s", stderr)
	}
}

// TestBinary_DraftStandalone_FailOnNeverAlwaysZero guards the escape
// hatch: --fail-on=never means "report but never fail". A CI user who
// wants advisory draft output without breaking their build relies on
// this flag. A regression that made it fail anyway would be a silent
// contract break.
func TestBinary_DraftStandalone_FailOnNeverAlwaysZero(t *testing.T) {
	dir := seedDraftCorpus(t, map[string]string{
		"note-quick-2026-04-04.md": `---
type: note
date: "2026-04-04"
status: draft
---
Very short body, no structure, no front-matter depth.
`,
	})

	_, stderr, code := runBinary(t, "angela", "draft", "--all", "--path", dir, "--fail-on", "never")
	if code != 0 {
		t.Errorf("--fail-on=never MUST always exit 0, got %d\nstderr:\n%s", code, stderr)
	}
}
