// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

// writeDocWithCommit creates a doc file with front matter including a commit field.
func writeDocWithCommit(t *testing.T, docsDir string, meta domain.DocMeta, slug string) {
	t.Helper()
	body := "# " + slug + "\n\nTest document.\n"
	data, err := Marshal(meta, body)
	if err != nil {
		t.Fatalf("marshal doc: %v", err)
	}
	filename := meta.Type + "-" + slug + "-" + meta.Date + ".md"
	meta.Filename = filename
	if err := os.WriteFile(filepath.Join(docsDir, filename), data, 0o644); err != nil {
		t.Fatalf("write doc: %v", err)
	}
}

func setupDocsDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	docsDir := filepath.Join(dir, ".lore", "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	return dir
}

func TestCollectReleaseDocuments_MatchingCommits(t *testing.T) {
	dir := setupDocsDir(t)
	docsDir := filepath.Join(dir, ".lore", "docs")

	writeDocWithCommit(t, docsDir, domain.DocMeta{
		Type: "feature", Date: "2026-03-05", Status: "draft",
		Commit: "aaaa1111222233334444555566667777aaaabbbb",
	}, "jwt-middleware")

	writeDocWithCommit(t, docsDir, domain.DocMeta{
		Type: "bugfix", Date: "2026-03-06", Status: "draft",
		Commit: "bbbb1111222233334444555566667777aaaabbbb",
	}, "token-expiry")

	writeDocWithCommit(t, docsDir, domain.DocMeta{
		Type: "decision", Date: "2026-03-04", Status: "draft",
		Commit: "cccc1111222233334444555566667777aaaabbbb",
	}, "auth-strategy")

	commits := []string{
		"aaaa1111222233334444555566667777aaaabbbb",
		"bbbb1111222233334444555566667777aaaabbbb",
	}

	docs, _, err := CollectReleaseDocuments(docsDir, commits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 docs, got %d", len(docs))
	}
	// Should NOT include the decision doc (commit not in range)
	for _, d := range docs {
		if d.Type == "decision" {
			t.Errorf("should not include decision doc (commit not in range)")
		}
	}
}

func TestCollectReleaseDocuments_NoMatch(t *testing.T) {
	dir := setupDocsDir(t)
	docsDir := filepath.Join(dir, ".lore", "docs")

	writeDocWithCommit(t, docsDir, domain.DocMeta{
		Type: "feature", Date: "2026-03-05", Status: "draft",
		Commit: "aaaa1111222233334444555566667777aaaabbbb",
	}, "jwt-middleware")

	docs, _, err := CollectReleaseDocuments(docsDir, []string{"dddd0000000000000000000000000000dddddddd"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 0 {
		t.Fatalf("expected 0 docs, got %d", len(docs))
	}
}

func TestCollectReleaseDocuments_EmptyCommits(t *testing.T) {
	docs, _, err := CollectReleaseDocuments("/nonexistent", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if docs != nil {
		t.Fatalf("expected nil, got %v", docs)
	}
}

func TestGenerateReleaseNotes(t *testing.T) {
	dir := setupDocsDir(t)
	docsDir := filepath.Join(dir, ".lore", "docs")

	docs := []ReleaseDoc{
		{DocMeta: domain.DocMeta{Type: "feature", Date: "2026-03-05", Status: "draft", Filename: "feature-jwt-middleware-2026-03-05.md"}, Title: "Add JWT middleware"},
		{DocMeta: domain.DocMeta{Type: "bugfix", Date: "2026-03-06", Status: "draft", Filename: "bugfix-token-expiry-2026-03-06.md"}, Title: "Fix token expiry"},
	}

	filename, err := GenerateReleaseNotes("v1.2.0", "2026-03-08", docs, docsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if filename != "release-v1.2.0-2026-03-08.md" {
		t.Errorf("expected release-v1.2.0-2026-03-08.md, got %s", filename)
	}

	data, err := os.ReadFile(filepath.Join(docsDir, filename))
	if err != nil {
		t.Fatalf("read release file: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "# Release v1.2.0") {
		t.Error("missing release title")
	}
	if !strings.Contains(content, "## Features") {
		t.Error("missing Features section")
	}
	if !strings.Contains(content, "## Bug Fixes") {
		t.Error("missing Bug Fixes section")
	}
	if !strings.Contains(content, "type: release") {
		t.Error("missing front matter type: release")
	}
	// Verify entry format: - **slug** — title (not filename)
	if !strings.Contains(content, "- **jwt-middleware** — Add JWT middleware") {
		t.Errorf("expected entry with title, got:\n%s", content)
	}
	if !strings.Contains(content, "- **token-expiry** — Fix token expiry") {
		t.Errorf("expected bugfix entry with title, got:\n%s", content)
	}
}

func TestUpdateReleasesJSON_NewFile(t *testing.T) {
	dir := t.TempDir()
	loreDir := filepath.Join(dir, ".lore")
	if err := os.MkdirAll(loreDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	err := UpdateReleasesJSON(loreDir, "v1.0.0", "2026-03-08", []string{"feature-jwt-2026-03-05.md"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(loreDir, "releases.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var entries []ReleaseEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Version != "v1.0.0" {
		t.Errorf("expected version v1.0.0, got %s", entries[0].Version)
	}
}

func TestUpdateReleasesJSON_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	loreDir := filepath.Join(dir, ".lore")
	if err := os.MkdirAll(loreDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Write initial entry
	initial := []ReleaseEntry{{Version: "v1.0.0", Date: "2026-03-01", Documents: []string{"doc1.md"}}}
	data, _ := json.MarshalIndent(initial, "", "  ")
	if err := os.WriteFile(filepath.Join(loreDir, "releases.json"), data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	err := UpdateReleasesJSON(loreDir, "v1.1.0", "2026-03-08", []string{"doc2.md"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, _ := os.ReadFile(filepath.Join(loreDir, "releases.json"))
	var entries []ReleaseEntry
	if err := json.Unmarshal(result, &entries); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestUpdateReleasesJSON_DuplicateVersion(t *testing.T) {
	dir := t.TempDir()
	loreDir := filepath.Join(dir, ".lore")
	if err := os.MkdirAll(loreDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	initial := []ReleaseEntry{{Version: "v1.0.0", Date: "2026-03-01", Documents: []string{"doc1.md"}}}
	data, _ := json.MarshalIndent(initial, "", "  ")
	if err := os.WriteFile(filepath.Join(loreDir, "releases.json"), data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	err := UpdateReleasesJSON(loreDir, "v1.0.0", "2026-03-08", []string{"doc2.md"})
	if err == nil {
		t.Fatal("expected error for duplicate version")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' in error, got: %v", err)
	}
}

func TestUpdateChangelog_NewFile(t *testing.T) {
	dir := t.TempDir()
	docs := []ReleaseDoc{
		{DocMeta: domain.DocMeta{Type: "feature", Date: "2026-03-05", Status: "draft", Filename: "feature-jwt-middleware-2026-03-05.md"}},
		{DocMeta: domain.DocMeta{Type: "bugfix", Date: "2026-03-06", Status: "draft", Filename: "bugfix-token-expiry-2026-03-06.md"}},
	}

	headerMissing, err := UpdateChangelog(dir, "v1.2.0", "2026-03-08", docs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if headerMissing {
		t.Error("headerMissing should be false for new file")
	}

	data, err := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "# Changelog") {
		t.Error("missing Changelog header")
	}
	if !strings.Contains(content, "## [v1.2.0] - 2026-03-08") {
		t.Error("missing release section")
	}
	if !strings.Contains(content, "### Added") {
		t.Error("missing Added section")
	}
	if !strings.Contains(content, "### Fixed") {
		t.Error("missing Fixed section")
	}
}

func TestUpdateChangelog_ExistingFile(t *testing.T) {
	dir := t.TempDir()

	existing := "# Changelog\n\nAll notable changes.\n\n## [v1.0.0] - 2026-03-01\n\n### Added\n- Initial release\n"
	if err := os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte(existing), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	docs := []ReleaseDoc{
		{DocMeta: domain.DocMeta{Type: "feature", Date: "2026-03-05", Status: "draft", Filename: "feature-jwt-middleware-2026-03-05.md"}},
	}

	headerMissing, err := UpdateChangelog(dir, "v1.1.0", "2026-03-08", docs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if headerMissing {
		t.Error("headerMissing should be false for file with header")
	}

	data, _ := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	content := string(data)

	// New release should appear before old
	v110Idx := strings.Index(content, "## [v1.1.0]")
	v100Idx := strings.Index(content, "## [v1.0.0]")
	if v110Idx < 0 || v100Idx < 0 {
		t.Fatalf("missing release sections in:\n%s", content)
	}
	if v110Idx >= v100Idx {
		t.Error("new release should appear before old release")
	}
}

func TestUpdateChangelog_MissingHeader(t *testing.T) {
	dir := t.TempDir()

	// Write a CHANGELOG without the standard "# Changelog" header
	existing := "Some random content at the top.\n\nMore content here.\n"
	if err := os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte(existing), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	docs := []ReleaseDoc{
		{DocMeta: domain.DocMeta{Type: "feature", Date: "2026-03-05", Status: "draft", Filename: "feature-jwt-middleware-2026-03-05.md"}},
	}

	headerMissing, err := UpdateChangelog(dir, "v1.0.0", "2026-03-08", docs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !headerMissing {
		t.Error("expected headerMissing=true when header is absent")
	}

	data, _ := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	content := string(data)

	// Should insert at top
	if !strings.HasPrefix(content, "\n## [v1.0.0]") {
		t.Errorf("expected release section at top, got:\n%s", content[:min(len(content), 100)])
	}
	// Original content should still be present
	if !strings.Contains(content, "Some random content") {
		t.Error("original content should be preserved")
	}
}

func TestSanitizeVersion_Safe(t *testing.T) {
	cases := []struct{ in, want string }{
		{"v1.2.0", "v1.2.0"},
		{"v1.0.0-rc.1", "v1.0.0-rc.1"},
		{"v2.0.0+build.123", "v2.0.0+build.123"},
	}
	for _, c := range cases {
		got := sanitizeVersion(c.in)
		if got != c.want {
			t.Errorf("sanitizeVersion(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSanitizeVersion_PathTraversal(t *testing.T) {
	cases := []struct{ in, want string }{
		{"../../../etc/evil", "......etcevil"},  // dots kept, slashes stripped
		{"v1/../../bad", "v1....bad"},            // slashes stripped
		{"", "unversioned"},
	}
	for _, c := range cases {
		got := sanitizeVersion(c.in)
		if got != c.want {
			t.Errorf("sanitizeVersion(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	// Verify no path separators survive
	for _, evil := range []string{"../../../etc/evil", "a/b\\c"} {
		got := sanitizeVersion(evil)
		if strings.ContainsAny(got, `/\`) {
			t.Errorf("sanitizeVersion(%q) should strip path separators, got %q", evil, got)
		}
	}
}

func TestGenerateReleaseNotes_PathTraversalVersion(t *testing.T) {
	dir := setupDocsDir(t)
	docsDir := filepath.Join(dir, ".lore", "docs")

	docs := []ReleaseDoc{
		{DocMeta: domain.DocMeta{Type: "feature", Date: "2026-03-05", Status: "draft", Filename: "feature-test-2026-03-05.md"}},
	}

	// sanitizeVersion strips slashes but keeps dots → validateFilename catches ".."
	_, err := GenerateReleaseNotes("../../../etc/evil", "2026-03-08", docs, docsDir)
	if err == nil {
		t.Fatal("expected error for path traversal version")
	}
	if !strings.Contains(err.Error(), "..") {
		t.Errorf("expected '..' rejection in error, got: %v", err)
	}
}
