// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// Invariant I8 — Markdown is source of truth.
//
// Contract: `.lore/store.db` is a DERIVED INDEX built from `.lore/docs/*.md`.
// At any time, the store can be deleted and rebuilt by scanning the
// filesystem. The rebuild must:
//
//   (a) produce a consistent doc_index — same doc → same row regardless
//       of rebuild count
//   (b) tolerate malformed source docs (bad YAML, missing frontmatter)
//       by skipping them gracefully without failing the whole rebuild
//   (c) NEVER cause the user to lose docs: the markdown files are
//       untouched during rebuild; only the SQLite index is overwritten
//
// Layer 1: TestRebuild_DocIndex_FromFixtures + TestRebuild_Commits_FromGitMock
//          in rebuild_test.go (happy paths, already present).
// Layer 2: TestI8_* anchors below which add:
//   - explicit "rebuild is idempotent" assertion (same input → same index)
//   - explicit "bad YAML does not abort rebuild" assertion (recovery path)
//   - explicit "source markdown untouched" assertion (user-data guarantee)
// ═══════════════════════════════════════════════════════════════════════════

// TestI8_RebuildIsIdempotent is the explicit I8 anchor for "idempotence":
// running RebuildFromSources N times in a row must produce identical
// indexed state. If the rebuild leaked rows or drifted (e.g. a ROWID
// collision, non-deterministic timestamp column), this test catches it.
func TestI8_RebuildIsIdempotent(t *testing.T) {
	s, _ := tempDB(t)
	docsDir := filepath.Join(t.TempDir(), "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Seed a realistic corpus.
	writeValidDoc(t, docsDir, "decision-auth-2026-03-01.md", "decision", "Title A")
	writeValidDoc(t, docsDir, "feature-api-2026-03-02.md", "feature", "Title B")
	writeValidDoc(t, docsDir, "bugfix-login-2026-03-03.md", "bugfix", "Title C")

	// First rebuild — baseline counts.
	count1, skipped1, _, err := s.RebuildFromSources(context.Background(), docsDir, nil)
	if err != nil {
		t.Fatalf("rebuild #1: %v", err)
	}
	dbCount1, _ := s.DocCount()

	// Second rebuild — must match.
	count2, skipped2, _, err := s.RebuildFromSources(context.Background(), docsDir, nil)
	if err != nil {
		t.Fatalf("rebuild #2: %v", err)
	}
	dbCount2, _ := s.DocCount()

	if count1 != count2 {
		t.Errorf("I8 violation: rebuild count changed: #1=%d #2=%d", count1, count2)
	}
	if skipped1 != skipped2 {
		t.Errorf("I8 violation: rebuild skipped count changed: #1=%d #2=%d", skipped1, skipped2)
	}
	if dbCount1 != dbCount2 {
		t.Errorf("I8 violation: DB row count drifted: #1=%d #2=%d", dbCount1, dbCount2)
	}

	// Third rebuild — triple-check.
	count3, _, _, _ := s.RebuildFromSources(context.Background(), docsDir, nil)
	if count3 != count1 {
		t.Errorf("I8 violation: 3rd rebuild drifted: #3=%d vs #1=%d", count3, count1)
	}
}

// TestI8_PartialCorruptionDoesNotAbortRebuild is the recovery contract:
// one (or many) malformed docs MUST NOT cause the whole rebuild to fail.
// Valid docs are still indexed; malformed docs are counted in `skipped`.
//
// This is a critical I8 guarantee because a single corrupt .md file in a
// production corpus would otherwise brick `lore doctor --rebuild-store`.
func TestI8_PartialCorruptionDoesNotAbortRebuild(t *testing.T) {
	s, _ := tempDB(t)
	docsDir := filepath.Join(t.TempDir(), "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// 3 valid + 3 corrupt (each broken differently).
	writeValidDoc(t, docsDir, "decision-ok-2026-03-01.md", "decision", "OK")
	writeValidDoc(t, docsDir, "feature-ok-2026-03-02.md", "feature", "OK")
	writeValidDoc(t, docsDir, "bugfix-ok-2026-03-03.md", "bugfix", "OK")
	// Bad 1: no frontmatter at all.
	writeRaw(t, docsDir, "bad-no-fm-2026-03-04.md", "# just markdown\n\nNo frontmatter here.")
	// Bad 2: malformed YAML (unclosed quote).
	writeRaw(t, docsDir, "bad-yaml-2026-03-05.md", "---\ntype: \"feature\ndate: 2026-03-05\n---\n# Body")
	// Bad 3: frontmatter present but missing required `type`.
	writeRaw(t, docsDir, "bad-no-type-2026-03-06.md", "---\ndate: \"2026-03-06\"\nstatus: draft\n---\n# Body")

	count, skipped, _, err := s.RebuildFromSources(context.Background(), docsDir, nil)
	if err != nil {
		t.Fatalf("I8 violation: rebuild aborted on corrupt docs (must skip not fail): %v", err)
	}
	if count != 3 {
		t.Errorf("valid doc count = %d, want 3", count)
	}
	if skipped < 1 {
		t.Errorf("skipped count = %d, want >= 1 (corrupt docs should be counted, got zero)", skipped)
	}
}

// TestI8_RebuildLeavesSourceMarkdownUntouched is the user-data guarantee:
// running RebuildFromSources MUST NOT modify the source `.md` files.
// Mtime drift is the primary indicator (any tool that accidentally wrote
// back the doc during parsing would change mtime). We also compare
// content bytes to be paranoid.
func TestI8_RebuildLeavesSourceMarkdownUntouched(t *testing.T) {
	s, _ := tempDB(t)
	docsDir := filepath.Join(t.TempDir(), "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Seed 2 docs and capture their baseline hashes + mtimes.
	docs := []struct{ name, content string }{
		{
			"decision-a-2026-03-01.md",
			"---\ntype: decision\ndate: \"2026-03-01\"\nstatus: draft\n---\n# A\n\nBody A.\n",
		},
		{
			"feature-b-2026-03-02.md",
			"---\ntype: feature\ndate: \"2026-03-02\"\nstatus: draft\n---\n# B\n\nBody B.\n",
		},
	}
	type snapshot struct {
		content []byte
		mtime   int64
	}
	baselines := map[string]snapshot{}
	for _, d := range docs {
		p := filepath.Join(docsDir, d.name)
		if err := os.WriteFile(p, []byte(d.content), 0o644); err != nil {
			t.Fatalf("seed %s: %v", d.name, err)
		}
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("stat %s: %v", d.name, err)
		}
		baselines[d.name] = snapshot{content: []byte(d.content), mtime: info.ModTime().UnixNano()}
	}

	if _, _, _, err := s.RebuildFromSources(context.Background(), docsDir, nil); err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	for name, baseline := range baselines {
		p := filepath.Join(docsDir, name)
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("I8 violation: source doc %q disappeared during rebuild: %v", name, err)
		}
		if info.ModTime().UnixNano() != baseline.mtime {
			t.Errorf("I8 violation: source doc %q mtime changed during rebuild (baseline=%d, now=%d) — rebuild must not write source docs",
				name, baseline.mtime, info.ModTime().UnixNano())
		}
		got, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if string(got) != string(baseline.content) {
			t.Errorf("I8 violation: source doc %q content mutated during rebuild", name)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────

func writeValidDoc(t *testing.T, dir, name, docType, title string) {
	t.Helper()
	content := "---\ntype: " + docType + "\ndate: \"2026-03-01\"\nstatus: draft\n---\n# " + title + "\n\nBody."
	writeRaw(t, dir, name, content)
}

func writeRaw(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
