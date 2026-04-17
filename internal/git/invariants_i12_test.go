// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package git

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// Invariant I12 — Hook install idempotent + non-destructive.
//
// Contract: `lore init` (which calls InstallHook) must be safe to run any
// number of times. The resulting `.git/hooks/post-commit` MUST be
// byte-identical to the single-run version, regardless of how many times
// install runs. Any drift (appended duplicate block, modified content,
// changed permissions) is an I12 violation that could:
//   - corrupt hooks that existed before Lore (user's pre-existing content)
//   - create duplicated dispatch (post-commit fires twice)
//   - desync `.git/hooks/post-commit` content with the scripted expectation
//
// Layer 1: existing TestInstallHook_Idempotent (counts markers — shallow).
// Layer 2: TestI12_InstallHookIdempotentByteIdentical (SHA256 compare).
// ═══════════════════════════════════════════════════════════════════════════

// TestI12_InstallHookIdempotentByteIdentical runs InstallHook three times on
// a fresh git repo and asserts the hook file is byte-identical after each
// run. SHA256 comparison catches whitespace drift, re-appended blocks,
// re-emitted shebangs — anything the existing "1 LORE-START marker" check
// would miss.
func TestI12_InstallHookIdempotentByteIdentical(t *testing.T) {
	dir := initGitRepo(t)
	a := NewAdapter(dir)
	hookPath := filepath.Join(dir, ".git", "hooks", "post-commit")

	// First install — establish the baseline.
	if _, err := a.InstallHook("post-commit"); err != nil {
		t.Fatalf("install #1: %v", err)
	}
	baseline, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read hook after install #1: %v", err)
	}
	baselineHash := sha256.Sum256(baseline)

	// Second install — should converge to the same content.
	if _, err := a.InstallHook("post-commit"); err != nil {
		t.Fatalf("install #2: %v", err)
	}
	run2, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read hook after install #2: %v", err)
	}
	run2Hash := sha256.Sum256(run2)

	if baselineHash != run2Hash {
		t.Errorf("I12 violation: install #2 produced different bytes\n  #1 SHA: %s\n  #2 SHA: %s\n  diff:\n%s",
			hex.EncodeToString(baselineHash[:]),
			hex.EncodeToString(run2Hash[:]),
			describeDiff(string(baseline), string(run2)))
	}

	// Third install — triple-check convergence.
	if _, err := a.InstallHook("post-commit"); err != nil {
		t.Fatalf("install #3: %v", err)
	}
	run3, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read hook after install #3: %v", err)
	}
	run3Hash := sha256.Sum256(run3)

	if baselineHash != run3Hash {
		t.Errorf("I12 violation: install #3 drifted vs baseline\n  #1 SHA: %s\n  #3 SHA: %s",
			hex.EncodeToString(baselineHash[:]),
			hex.EncodeToString(run3Hash[:]))
	}

	// Sanity: exactly one LORE block (not 2 or 3). If the byte-identical
	// check passes but somehow this fails, the idempotence is broken at a
	// deeper level (e.g. the first run itself produced duplicate markers).
	if got := strings.Count(string(run3), loreStartMarker); got != 1 {
		t.Errorf("expected exactly 1 LORE-START marker after 3 installs, got %d", got)
	}
}

// TestI12_InstallHook_PreservesExistingContent is the companion "non-
// destructive" guarantee: if a user had a pre-existing post-commit hook
// (common on teams with custom tooling), installing Lore MUST preserve
// that content verbatim above and below the LORE block. Then uninstalling
// Lore MUST restore the file to exactly the pre-Lore state.
func TestI12_InstallHook_PreservesExistingContent(t *testing.T) {
	dir := initGitRepo(t)
	hookPath := filepath.Join(dir, ".git", "hooks", "post-commit")

	// Simulate a pre-existing hook with custom content.
	existing := "#!/bin/sh\n# team-custom\necho 'team hook ran'\n"
	if err := os.WriteFile(hookPath, []byte(existing), 0o755); err != nil {
		t.Fatalf("seed existing hook: %v", err)
	}

	a := NewAdapter(dir)
	if _, err := a.InstallHook("post-commit"); err != nil {
		t.Fatalf("install: %v", err)
	}

	afterInstall, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read after install: %v", err)
	}
	contents := string(afterInstall)
	// Pre-existing custom content must survive verbatim.
	if !strings.Contains(contents, "echo 'team hook ran'") {
		t.Errorf("I12 violation: install destroyed pre-existing content\n%s", contents)
	}
	// And the Lore block must have been added.
	if !strings.Contains(contents, loreStartMarker) || !strings.Contains(contents, loreEndMarker) {
		t.Errorf("install did not add LORE block to existing hook\n%s", contents)
	}

	// Uninstall must restore the pre-Lore content.
	if err := a.UninstallHook("post-commit"); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	afterUninstall, err := os.ReadFile(hookPath)
	if err != nil {
		// If uninstall removed the file entirely, check that too is reasonable
		// for a hook that only contained Lore content. But with pre-existing
		// content, it must still exist.
		t.Fatalf("hook missing after uninstall (must preserve pre-existing content): %v", err)
	}
	if !strings.Contains(string(afterUninstall), "echo 'team hook ran'") {
		t.Errorf("I12 violation: uninstall destroyed pre-existing content\n%s", string(afterUninstall))
	}
	if strings.Contains(string(afterUninstall), loreStartMarker) {
		t.Errorf("uninstall did not remove LORE block\n%s", string(afterUninstall))
	}
}

// describeDiff is a tiny diff helper for the byte-identical failure mode.
// It reports the first line where the two contents diverge plus a short
// excerpt so the test output is actionable without a full-file dump.
func describeDiff(a, b string) string {
	aLines := strings.Split(a, "\n")
	bLines := strings.Split(b, "\n")
	n := len(aLines)
	if len(bLines) < n {
		n = len(bLines)
	}
	for i := 0; i < n; i++ {
		if aLines[i] != bLines[i] {
			return "    line " + itoa(i+1) + ":\n      #1: " + aLines[i] + "\n      #2: " + bLines[i]
		}
	}
	if len(aLines) != len(bLines) {
		return "    line counts differ: #1=" + itoa(len(aLines)) + ", #2=" + itoa(len(bLines))
	}
	return "    (no line-level diff found but SHA differs — trailing whitespace?)"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
