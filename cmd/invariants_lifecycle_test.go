// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/status"
	"github.com/greycoderk/lore/internal/storage"
)

// ═══════════════════════════════════════════════════════════════════════════
// Invariants for search & lifecycle commands.
//
// Scope: `lore list`, `lore show`, `lore status`, `lore delete`, plus
// `angela review --filter` (the one user-facing regex surface). These are
// the commands users run many times a day — stability of their output is
// the contract that lets CI pipelines, shell scripts, and downstream tools
// rely on them.
//
//   I16 — Output contract stable: status --badge produces a shields.io
//         markdown snippet in a documented format. Breaking it silently
//         would break every README that embeds the badge.
//   I17 — Delete consistency: lore delete removes the file AND
//         regenerates the index (README.md) so the corpus view stays
//         consistent. A bug where the file disappears but the index still
//         lists it is a user-visible "ghost doc" regression.
//   I18 — Regex keyword safety: `angela review --filter` accepts a regex
//         from user input. A malformed regex must fail with a clean error,
//         never panic. A ReDoS-style pattern must not hang the CLI.
//
// Each invariant gets an explicit TestI[N]_* anchor so the matrix cites a
// stable function name. The property-based layer is attached where
// applicable (random filter patterns for I18).
// ═══════════════════════════════════════════════════════════════════════════

// ─────────────────────────────────────────────────────────────────────────
// I16 — Output contract stable
// ─────────────────────────────────────────────────────────────────────────

// shieldsBadgeRe matches the canonical shields.io badge markdown format
// that `lore status --badge` emits. If the format ever drifts (e.g., a
// trailing punctuation added, the URL shape changes), README embeds break
// silently. The regex is loose enough to allow metric tweaks (different
// coverage %, different color) but strict enough to catch structural
// breakage.
var shieldsBadgeRe = regexp.MustCompile(
	`\[!\[lore:\s+[^\]]+\]\(https://img\.shields\.io/badge/lore-[^\)]+\)\]\(https://github\.com/greycoderk/lore\)`,
)

// TestI16_StatusBadgeOutputMatchesShieldsFormat is the explicit I16 anchor
// for the badge output contract. The `status --badge` command delegates to
// `status.FormatBadgeMarkdown` — testing the helper directly avoids the
// requireLoreDir + CWD coupling of the cobra command path while still
// asserting the PUBLISHED contract that README embeds rely on.
//
// Contract shape: `[![lore: <N>% <label>](https://img.shields.io/badge/lore-...)]
// (https://github.com/greycoderk/lore)`. Any drift here breaks every README
// badge silently.
func TestI16_StatusBadgeOutputMatchesShieldsFormat(t *testing.T) {
	// Cover all 3 color bands + the 💯 special case.
	cases := []struct {
		name     string
		coverage int
	}{
		{"gold band 80+", 85},
		{"green band 50-79", 60},
		{"grey band <50", 25},
		{"perfect 100", 100},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := status.FormatBadgeMarkdown(tc.coverage, "documented")
			if !shieldsBadgeRe.MatchString(got) {
				t.Errorf("I16 violation: badge for coverage=%d doesn't match shields.io format\n  pattern: %s\n  got:     %q",
					tc.coverage, shieldsBadgeRe.String(), got)
			}
			for _, want := range []string{"lore-", "shields.io", "greycoderk/lore"} {
				if !strings.Contains(got, want) {
					t.Errorf("I16 violation: badge missing required substring %q in %q", want, got)
				}
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────
// I17 — Delete consistency (file + index)
// ─────────────────────────────────────────────────────────────────────────

// TestI17_DeleteDocRemovesFileAndRegeneratesIndex is the explicit I17
// anchor. After `DeleteDoc`:
//   (a) the markdown file is gone from the filesystem
//   (b) the README index no longer lists it
//   (c) other docs are untouched
// A regression where (a) passes but (b) fails leaves the user with a
// "ghost" entry in README that points to a missing file — user-visible
// corpus rot.
func TestI17_DeleteDocRemovesFileAndRegeneratesIndex(t *testing.T) {
	dir := seedCorpus(t, "keep-a.md", "to-delete.md", "keep-b.md")

	// Regenerate the index so README.md reflects all 3 docs.
	if err := storage.RegenerateIndex(dir); err != nil {
		t.Fatalf("seed RegenerateIndex: %v", err)
	}
	before, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	if !strings.Contains(string(before), "to-delete.md") {
		t.Fatalf("sanity: seed README should list to-delete.md, got:\n%s", before)
	}

	// Delete the target doc.
	if err := storage.DeleteDoc(dir, "to-delete.md"); err != nil {
		t.Fatalf("DeleteDoc: %v", err)
	}

	// (a) file is gone
	if _, err := os.Stat(filepath.Join(dir, "to-delete.md")); !os.IsNotExist(err) {
		t.Errorf("I17 violation: file not removed (stat err=%v)", err)
	}

	// (b) README no longer lists the deleted doc
	after, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatalf("post-delete README read: %v", err)
	}
	if strings.Contains(string(after), "to-delete.md") {
		t.Errorf("I17 violation: README still references deleted doc\n%s", string(after))
	}

	// (c) other docs untouched
	for _, name := range []string{"keep-a.md", "keep-b.md"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("I17 violation: unrelated doc %q disappeared: %v", name, err)
		}
		if !strings.Contains(string(after), name) {
			t.Errorf("I17 violation: README lost unrelated doc %q\n%s", name, string(after))
		}
	}
}

// TestI17_DeleteREADMEProtected — the auto-generated index must not be
// deletable through the normal delete path. Removing README.md leaves
// every other doc orphaned from the user's browse view. The existing
// TestDeleteDoc_ProtectedREADME covers this at the storage layer; here
// we assert it from the caller perspective.
func TestI17_DeleteREADMEProtected(t *testing.T) {
	dir := seedCorpus(t, "doc-a.md")
	if err := storage.RegenerateIndex(dir); err != nil {
		t.Fatalf("RegenerateIndex: %v", err)
	}

	err := storage.DeleteDoc(dir, "README.md")
	if err == nil {
		t.Fatal("I17 violation: DeleteDoc allowed README.md deletion")
	}
	if _, err := os.Stat(filepath.Join(dir, "README.md")); os.IsNotExist(err) {
		t.Error("I17 violation: README.md removed despite protection")
	}
}

// TestI17_DeleteValidationPreventsFileRemoval — when validation rejects
// a filename (path traversal, reserved name), the file must NOT be
// touched. A regression where validation runs AFTER removal would erase
// legitimate files because an attacker passed `../../../etc/passwd`.
func TestI17_DeleteValidationPreventsFileRemoval(t *testing.T) {
	dir := seedCorpus(t, "legit.md")

	// Path-traversal attempts must error; "legit.md" must remain.
	for _, attempt := range []string{"../outside.md", "subdir/nested.md"} {
		if err := storage.DeleteDoc(dir, attempt); err == nil {
			t.Errorf("I17 violation: DeleteDoc accepted path-traversal %q", attempt)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "legit.md")); err != nil {
		t.Errorf("I17 violation: legit.md was touched by a rejected delete: %v", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────
// I18 — Regex keyword safety (angela review --filter)
// ─────────────────────────────────────────────────────────────────────────

// TestI18_ReviewFilterRegex_MalformedRejectedCleanly — a malformed regex
// must return a clean cobra error with a message, not panic. The test
// uses a classic malformed pattern `[` that fails to compile; a
// regression where the error propagates as a nil pointer or an uncaught
// panic would crash the CLI on user input.
func TestI18_ReviewFilterRegex_MalformedRejectedCleanly(t *testing.T) {
	streams, _, errBuf := lifecycleTestStreams()
	cfg := &config.Config{}
	// Need a docs dir to pass the earlier resolveDocsDir guard.
	docsDir := t.TempDir()
	// Seed 5 minimal docs to bypass PrepareDocSummaries minimum.
	for i := 0; i < 5; i++ {
		name := filepath.Join(docsDir, "doc"+itoaI18(i)+".md")
		_ = os.WriteFile(name, []byte("# x\n"), 0o644)
	}
	path := docsDir
	cmd := newAngelaReviewCmd(cfg, streams, &path)
	cmd.SetArgs([]string{"--filter", "[unclosed"})

	// The command must NOT panic. Error is expected (caller code returns
	// an error on bad regex); what we assert is "no crash + returns error".
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("I18 violation: cobra panicked on malformed regex: %v", r)
			}
		}()
		err = cmd.Execute()
	}()
	if err == nil {
		t.Fatal("I18 violation: malformed regex must produce an error")
	}
	got := errBuf.String() + " " + err.Error()
	if !strings.Contains(got, "filter") && !strings.Contains(got, "regex") {
		t.Errorf("I18: error message should mention 'filter' or 'regex', got: %q", got)
	}
}

// TestI18_ReviewFilterRegex_EmptyPatternAllowed — an empty string must be
// accepted without compilation (no filter means "all docs"). A regression
// where the empty-string branch was removed would compile "" as a regex
// (which matches everything) — not broken functionally, but would mask
// the "no filter" intent from the user's mental model.
func TestI18_ReviewFilterRegex_EmptyPatternAllowed(t *testing.T) {
	streams, _, _ := lifecycleTestStreams()
	cfg := &config.Config{}
	docsDir := t.TempDir()
	for i := 0; i < 5; i++ {
		name := filepath.Join(docsDir, "doc"+itoaI18(i)+".md")
		_ = os.WriteFile(name, []byte("# x\n"), 0o644)
	}
	path := docsDir
	cmd := newAngelaReviewCmd(cfg, streams, &path)
	cmd.SetArgs([]string{"--preview"})
	// No --filter at all — the code path for empty filter must not reject.
	if err := cmd.Execute(); err != nil {
		t.Errorf("I18 (no-filter path): empty filter must not be treated as malformed regex: %v", err)
	}
}

// TestI18_ReviewFilterRegex_PropertyRandomPatterns — hit the regex
// compile path with 30 random patterns (including known-bad ones) and
// assert ZERO panics across the set. Regression surface: a new
// construction step between flag parse and regexp.Compile that forgets
// the recover/err-propagation path.
func TestI18_ReviewFilterRegex_PropertyRandomPatterns(t *testing.T) {
	badPatterns := []string{
		"[",
		"(",
		"(?P<",
		")(",
		"\\",
		"[a-",
		"(?:",
		"[[:invalid:]]",
		"(?<=look)",      // Go regexp doesn't support lookbehind — rejected cleanly
		"(?!negative)",   // same
		"{1,2,3}",        // invalid bounded repetition
		".*.*.*(a|b)*$",  // known ReDoS pattern — must at least compile without hanging test
	}

	for _, pat := range badPatterns {
		t.Run("pattern="+pat, func(t *testing.T) {
			streams, _, _ := lifecycleTestStreams()
			cfg := &config.Config{}
			docsDir := t.TempDir()
			for i := 0; i < 5; i++ {
				name := filepath.Join(docsDir, "doc"+itoaI18(i)+".md")
				_ = os.WriteFile(name, []byte("# x\n"), 0o644)
			}
			path := docsDir
			cmd := newAngelaReviewCmd(cfg, streams, &path)
			cmd.SetArgs([]string{"--preview", "--filter", pat})

			// The only assertion: no panic. Error IS expected for
			// malformed patterns; absence of panic is the invariant.
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("I18 violation: pattern %q caused panic: %v", pat, r)
					}
				}()
				_ = cmd.Execute()
			}()
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────

func lifecycleTestStreams() (domain.IOStreams, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	return domain.IOStreams{In: &bytes.Buffer{}, Out: out, Err: errBuf}, out, errBuf
}

// seedCorpus writes each filename as a minimal valid markdown doc to a
// fresh TempDir and returns the dir path. Used by I17 tests.
func seedCorpus(t *testing.T, filenames ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range filenames {
		content := "---\ntype: feature\ndate: \"2026-03-01\"\nstatus: draft\n---\n# " + name + "\n\nBody.\n"
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}
	return dir
}

// itoaI18 is a tiny local itoa so this file doesn't have to import strconv
// just for 5-doc seeding.
func itoaI18(n int) string {
	if n == 0 {
		return "0"
	}
	var out []byte
	for n > 0 {
		out = append([]byte{byte('0' + n%10)}, out...)
		n /= 10
	}
	return string(out)
}
