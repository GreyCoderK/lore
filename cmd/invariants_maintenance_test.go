// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/git"
	"github.com/greycoderk/lore/internal/storage"
)

// ═══════════════════════════════════════════════════════════════════════════
// Invariants for maintenance commands (init, doctor, release).
//
// These commands are used rarely per user but are DESTRUCTIVE by design:
// they touch `.lore/`, git hooks, CHANGELOG.md. A regression here can
// silently corrupt a team's documentation state.
//
//   I19 — Doctor repair idempotent: `lore doctor --fix` run N times on
//         the same corpus must converge to a stable state. A bug that
//         re-fixes a phantom issue each run (e.g., AtomicWrite creates a
//         .tmp that Diagnose then flags + Fix removes, looping forever)
//         is the core failure mode this invariant guards against.
//
//   I20 — Release output stable: the release notes + CHANGELOG section
//         headers use a documented, stable format. Keep-a-Changelog
//         categories ("Added"/"Fixed"/"Changed") are deliberately EN-only
//         per KaC convention — this is pinned so a future localization
//         doesn't accidentally break downstream parsers.
//
//   I21 — Init non-destructive: `lore init` on an already-initialized
//         repo MUST NOT overwrite `.lore/`, `.lorerc`, `.lorerc.local`,
//         or any doc in `.lore/docs/`. Second invocation = silent no-op
//         warning, byte-for-byte identical file state.
// ═══════════════════════════════════════════════════════════════════════════

// ─────────────────────────────────────────────────────────────────────────
// I19 — Doctor --fix idempotent
// ─────────────────────────────────────────────────────────────────────────

// TestI19_DoctorFixIsIdempotent is the explicit named anchor. Seeds a
// corpus with a known-fixable issue (orphan .tmp file), runs Fix once,
// then runs Diagnose + Fix again, and asserts the second run reports
// zero issues fixed (converged).
func TestI19_DoctorFixIsIdempotent(t *testing.T) {
	docsDir := t.TempDir()
	// Seed: one valid doc so the corpus isn't empty.
	_ = os.WriteFile(filepath.Join(docsDir, "feature-ok-2026-04-17.md"),
		[]byte("---\ntype: feature\ndate: \"2026-04-17\"\nstatus: draft\n---\n# OK\n\nBody.\n"), 0o644)
	// Seed the README index so "stale index" isn't flagged as an issue.
	if err := storage.RegenerateIndex(docsDir); err != nil {
		t.Fatalf("seed RegenerateIndex: %v", err)
	}
	// Seed: orphan .tmp (Diagnose flags as auto-fixable).
	tmpPath := filepath.Join(docsDir, "stale-2026-04-17.md.tmp")
	_ = os.WriteFile(tmpPath, []byte("x"), 0o644)
	// fixOrphanTmp skips files younger than 5s (avoids racing with
	// concurrent writes); backdate so Fix actually removes it.
	past := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(tmpPath, past, past); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	// First run: Diagnose + Fix — should fix the orphan.
	r1, err := storage.Diagnose(docsDir)
	if err != nil {
		t.Fatalf("Diagnose #1: %v", err)
	}
	f1, err := storage.Fix(docsDir, r1)
	if err != nil {
		t.Fatalf("Fix #1: %v", err)
	}
	if f1.Fixed == 0 {
		t.Fatal("expected Fix #1 to clean the orphan .tmp, got 0 fixed")
	}

	// Second run: Diagnose + Fix — must converge (0 new fixes).
	r2, err := storage.Diagnose(docsDir)
	if err != nil {
		t.Fatalf("Diagnose #2: %v", err)
	}
	f2, err := storage.Fix(docsDir, r2)
	if err != nil {
		t.Fatalf("Fix #2: %v", err)
	}
	if f2.Fixed != 0 {
		t.Errorf("I19 violation: Fix #2 reported %d additional fixes (expected 0 — doctor must be idempotent)",
			f2.Fixed)
	}

	// Third run to triple-check convergence.
	r3, _ := storage.Diagnose(docsDir)
	f3, _ := storage.Fix(docsDir, r3)
	if f3.Fixed != 0 {
		t.Errorf("I19 violation: Fix #3 drifted: %d fixes on already-stable corpus", f3.Fixed)
	}
}

// TestI19_DoctorFixConvergesOnCleanCorpus — a cleaner baseline: running
// Fix on an already-clean corpus must be a no-op (0 fixed, 0 errors).
// Protects against a regression where Fix creates spurious activity on
// an empty issue list (e.g., rewriting README.md every time).
func TestI19_DoctorFixConvergesOnCleanCorpus(t *testing.T) {
	docsDir := t.TempDir()
	// Seed clean valid doc + up-to-date index so Diagnose has nothing to flag.
	_ = os.WriteFile(filepath.Join(docsDir, "feature-clean-2026-04-17.md"),
		[]byte("---\ntype: feature\ndate: \"2026-04-17\"\nstatus: draft\n---\n# Clean\n\nBody.\n"), 0o644)
	if err := storage.RegenerateIndex(docsDir); err != nil {
		t.Fatalf("seed RegenerateIndex: %v", err)
	}

	report, err := storage.Diagnose(docsDir)
	if err != nil {
		t.Fatalf("Diagnose: %v", err)
	}
	fix, err := storage.Fix(docsDir, report)
	if err != nil {
		t.Fatalf("Fix: %v", err)
	}
	if fix.Fixed > 0 {
		t.Errorf("I19 violation: Fix on clean corpus reported %d fixes (expected 0)", fix.Fixed)
	}
}

// ─────────────────────────────────────────────────────────────────────────
// I20 — Release output stable (pinning current EN-only behavior)
// ─────────────────────────────────────────────────────────────────────────

// TestI20_ReleaseNotesSectionHeadersAreStable pins the current EN-only
// release-notes section headers. The Keep-a-Changelog convention uses
// English verbs ("Added", "Fixed", "Changed") as a machine-parseable
// standard — bilingual CHANGELOG parsing tools (release-please, changie,
// etc.) depend on this wording.
//
// This test PINS the current behavior so a well-intentioned future
// localization doesn't accidentally break downstream parsers. If the
// team decides to localize release notes section titles (not the
// CHANGELOG categories), this test will need a conscious update.
//
// Known gap documented in phase-11b-maintenance.md: release notes
// "Features"/"Bug Fixes"/"Refactors" headers are user-facing and could
// be localized as a future story. For MVP v1, they're EN-only.
func TestI20_ReleaseNotesSectionHeadersAreStable(t *testing.T) {
	docsDir := t.TempDir()
	docs := []storage.ReleaseDoc{
		{DocMeta: domain.DocMeta{Type: "feature", Filename: "feature-auth-2026-03-01.md"}, Title: "JWT auth"},
		{DocMeta: domain.DocMeta{Type: "bugfix", Filename: "bugfix-login-2026-03-02.md"}, Title: "Login hang"},
		{DocMeta: domain.DocMeta{Type: "refactor", Filename: "refactor-db-2026-03-03.md"}, Title: "DB pool"},
		{DocMeta: domain.DocMeta{Type: "decision", Filename: "decision-db-2026-03-04.md"}, Title: "PostgreSQL"},
	}

	path, err := storage.GenerateReleaseNotes("v1.0.0", "2026-03-10", docs, docsDir)
	if err != nil {
		t.Fatalf("GenerateReleaseNotes: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(docsDir, path))
	if err != nil {
		t.Fatalf("read release notes: %v", err)
	}
	content := string(data)

	// Pin: each type MUST produce a specific English section header.
	// A regression (typo, localization without test update) fails here.
	wantHeaders := []string{
		"## Features",
		"## Bug Fixes",
		"## Refactors",
		"## Decisions",
	}
	for _, want := range wantHeaders {
		if !strings.Contains(content, want) {
			t.Errorf("I20 violation: release notes missing header %q\n\n%s", want, content)
		}
	}
}

// TestI20_ChangelogCategoriesFollowKeepAChangelog pins the CHANGELOG
// categories to the Keep-a-Changelog standard (Added/Changed/Fixed).
// These MUST stay English — they are a machine-readable convention.
func TestI20_ChangelogCategoriesFollowKeepAChangelog(t *testing.T) {
	projectDir := t.TempDir()
	docs := []storage.ReleaseDoc{
		{DocMeta: domain.DocMeta{Type: "feature", Filename: "feature-a-2026-03-01.md"}, Title: "A"},
		{DocMeta: domain.DocMeta{Type: "bugfix", Filename: "bugfix-b-2026-03-02.md"}, Title: "B"},
		{DocMeta: domain.DocMeta{Type: "refactor", Filename: "refactor-c-2026-03-03.md"}, Title: "C"},
	}
	if _, err := storage.UpdateChangelog(projectDir, "1.0.0", "2026-03-10", docs); err != nil {
		t.Fatalf("UpdateChangelog: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(projectDir, "CHANGELOG.md"))
	if err != nil {
		t.Fatalf("read CHANGELOG: %v", err)
	}
	content := string(data)

	for _, want := range []string{"### Added", "### Fixed", "### Changed"} {
		if !strings.Contains(content, want) {
			t.Errorf("I20 violation: CHANGELOG missing KaC category %q\n\n%s", want, content)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────
// I21 — Init non-destructive
// ─────────────────────────────────────────────────────────────────────────

// TestI21_InitSecondRunIsNoOp is the explicit I21 anchor: a second
// `lore init` on an already-initialized repo must not touch any file
// created by the first init. SHA256 compare on `.lorerc`, `.lorerc.local`,
// and existing docs.
func TestI21_InitSecondRunIsNoOp(t *testing.T) {
	dir := initRealGitRepo(t)
	streams, _, _ := testStreams()
	deps := initDeps{
		git:     git.NewAdapter(dir),
		workDir: dir,
	}

	// First init: establishes baseline.
	if err := runInit(context.Background(), &config.Config{}, deps, streams, true); err != nil {
		t.Fatalf("runInit #1: %v", err)
	}

	// Seed a user doc between the two inits — a second init must NOT
	// delete user content.
	userDocPath := filepath.Join(dir, ".lore", "docs", "feature-my-work-2026-04-17.md")
	userDoc := "---\ntype: feature\ndate: \"2026-04-17\"\nstatus: draft\n---\n# My work\n\nImportant.\n"
	if err := os.WriteFile(userDocPath, []byte(userDoc), 0o644); err != nil {
		t.Fatalf("seed user doc: %v", err)
	}

	// Snapshot state.
	files := []string{
		filepath.Join(dir, ".lorerc"),
		filepath.Join(dir, ".lorerc.local"),
		userDocPath,
	}
	before := snapshotHashes(t, files)

	// Second init: must be no-op (early-return on existing `.lore/`).
	if err := runInit(context.Background(), &config.Config{}, deps, streams, true); err != nil {
		t.Fatalf("runInit #2: %v", err)
	}

	after := snapshotHashes(t, files)
	for _, f := range files {
		if before[f] != after[f] {
			t.Errorf("I21 violation: file changed across init runs\n  file:    %s\n  before:  %s\n  after:   %s",
				f, before[f], after[f])
		}
	}

	// Sanity: user doc still readable.
	got, err := os.ReadFile(userDocPath)
	if err != nil {
		t.Fatalf("I21 violation: user doc disappeared after second init: %v", err)
	}
	if string(got) != userDoc {
		t.Errorf("I21 violation: user doc content mutated across init runs")
	}
}

// TestI21_InitEmitsWarningOnSecondRun — the second init is not silent:
// the user sees a message explaining that `.lore/` is already set up.
// A silent no-op would confuse users who genuinely wanted to reset.
func TestI21_InitEmitsWarningOnSecondRun(t *testing.T) {
	dir := initRealGitRepo(t)
	streams, _, errBuf := testStreams()
	deps := initDeps{
		git:     git.NewAdapter(dir),
		workDir: dir,
	}

	// First init — populate baseline + reset stderr.
	_ = runInit(context.Background(), &config.Config{}, deps, streams, true)
	errBuf.Reset()

	// Second init — must warn.
	if err := runInit(context.Background(), &config.Config{}, deps, streams, true); err != nil {
		t.Fatalf("runInit #2: %v", err)
	}

	got := errBuf.String()
	if got == "" {
		t.Error("I21 violation: second init must write a message to stderr explaining the no-op (user confusion prevention)")
	}
	// The catalog key is `InitAlreadyInitialized`. We don't hard-code the
	// wording but check the message contains "init" or mentions Lore is
	// already set up.
	lower := strings.ToLower(got)
	if !strings.Contains(lower, "lore") && !strings.Contains(lower, "init") && !strings.Contains(lower, "already") {
		t.Errorf("I21: second-init message should mention Lore/init/already, got: %q", got)
	}
}

// ─────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────

// snapshotHashes returns a map path → SHA256 hex. Missing files get "".
func snapshotHashes(t *testing.T, files []string) map[string]string {
	t.Helper()
	out := make(map[string]string, len(files))
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			out[f] = ""
			continue
		}
		sum := sha256.Sum256(data)
		out[f] = string(sum[:])
	}
	return out
}
