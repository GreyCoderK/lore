// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

func TestCorpusStore_ReadDoc(t *testing.T) {
	dir := t.TempDir()
	meta := domain.DocMeta{
		Type:   "decision",
		Date:   "2026-03-07",
		Status: "demo",
	}
	body := "# Test Doc\n"

	result, err := WriteDoc(dir, meta, "test doc", body)
	if err != nil {
		t.Fatalf("storage: write: %v", err)
	}

	store := &CorpusStore{Dir: dir}

	content, err := store.ReadDoc(result.Filename)
	if err != nil {
		t.Fatalf("storage: read: %v", err)
	}

	if content == "" {
		t.Error("expected non-empty content")
	}
}

func TestCorpusStore_ReadDoc_NotFound(t *testing.T) {
	dir := t.TempDir()
	store := &CorpusStore{Dir: dir}

	_, err := store.ReadDoc("nonexistent")
	if err == nil {
		t.Error("expected error for missing doc")
	}
}

func TestCorpusStore_ListDocs(t *testing.T) {
	dir := t.TempDir()
	meta := domain.DocMeta{
		Type:   "decision",
		Date:   "2026-03-07",
		Status: "demo",
	}

	_, err := WriteDoc(dir, meta, "doc one", "# One\n")
	if err != nil {
		t.Fatalf("storage: write: %v", err)
	}

	store := &CorpusStore{Dir: dir}
	docs, err := store.ListDocs(domain.DocFilter{})
	if err != nil {
		t.Fatalf("storage: list: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 doc, got %d", len(docs))
	}
}

func TestCorpusStore_ListDocs_EmptyDir(t *testing.T) {
	store := &CorpusStore{Dir: t.TempDir()}
	docs, err := store.ListDocs(domain.DocFilter{})
	if err != nil {
		t.Fatalf("storage: list empty: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("expected 0 docs, got %d", len(docs))
	}
}

func TestCorpusStore_ListDocs_FilterByType(t *testing.T) {
	dir := t.TempDir()

	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-07", Status: "published"}, "auth", "# Auth\n")
	WriteDoc(dir, domain.DocMeta{Type: "feature", Date: "2026-03-08", Status: "published"}, "api", "# API\n")

	store := &CorpusStore{Dir: dir}
	docs, err := store.ListDocs(domain.DocFilter{Type: "decision"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 decision, got %d", len(docs))
	}
	if docs[0].Type != "decision" {
		t.Errorf("expected type 'decision', got %q", docs[0].Type)
	}
}

func TestCorpusStore_ListDocs_FilterByStatus(t *testing.T) {
	dir := t.TempDir()

	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-07", Status: "published"}, "one", "body\n")
	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-08", Status: "draft"}, "two", "body\n")

	store := &CorpusStore{Dir: dir}
	docs, err := store.ListDocs(domain.DocFilter{Status: "draft"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 draft, got %d", len(docs))
	}
}

func TestCorpusStore_ListDocs_FilterByDateRange(t *testing.T) {
	dir := t.TempDir()

	WriteDoc(dir, domain.DocMeta{Type: "note", Date: "2026-03-05", Status: "published"}, "old", "body\n")
	WriteDoc(dir, domain.DocMeta{Type: "note", Date: "2026-03-07", Status: "published"}, "mid", "body\n")
	WriteDoc(dir, domain.DocMeta{Type: "note", Date: "2026-03-10", Status: "published"}, "new", "body\n")

	store := &CorpusStore{Dir: dir}
	docs, err := store.ListDocs(domain.DocFilter{After: "2026-03-06", Before: "2026-03-08"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 doc in range, got %d", len(docs))
	}
}

func TestCorpusStore_ListDocs_FilterByTags(t *testing.T) {
	dir := t.TempDir()

	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-07", Status: "published", Tags: []string{"auth", "api"}}, "tagged", "body\n")
	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-08", Status: "published", Tags: []string{"perf"}}, "other", "body\n")

	store := &CorpusStore{Dir: dir}
	docs, err := store.ListDocs(domain.DocFilter{Tags: []string{"auth"}})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 tagged doc, got %d", len(docs))
	}
}

func TestCorpusStore_ListDocs_FilterByText(t *testing.T) {
	dir := t.TempDir()

	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-07", Status: "published"}, "jwt", "# JWT auth middleware\n")
	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-08", Status: "published"}, "db", "# Database choice\n")

	store := &CorpusStore{Dir: dir}
	docs, err := store.ListDocs(domain.DocFilter{Text: "JWT"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 doc matching text, got %d", len(docs))
	}
}

func TestCorpusStore_ListDocs_CombinedFilters(t *testing.T) {
	dir := t.TempDir()

	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-01", Status: "published", Tags: []string{"auth", "security"}}, "jwt-auth", "# JWT Auth\nJWT implementation\n")
	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-05", Status: "draft", Tags: []string{"auth"}}, "oauth-flow", "# OAuth\nOAuth flow\n")
	WriteDoc(dir, domain.DocMeta{Type: "feature", Date: "2026-03-03", Status: "published", Tags: []string{"api"}}, "new-api", "# API\nNew API\n")

	store := &CorpusStore{Dir: dir}

	// Type + Status filter
	docs, err := store.ListDocs(domain.DocFilter{Type: "decision", Status: "published"})
	if err != nil {
		t.Fatalf("Type+Status filter: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("Type+Status filter: got %d, want 1", len(docs))
	}

	// Type + Tags filter
	docs, err = store.ListDocs(domain.DocFilter{Type: "decision", Tags: []string{"auth"}})
	if err != nil {
		t.Fatalf("Type+Tags filter: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("Type+Tags filter: got %d, want 2", len(docs))
	}

	// Type + Status + Tags filter (all three)
	docs, err = store.ListDocs(domain.DocFilter{Type: "decision", Status: "published", Tags: []string{"auth"}})
	if err != nil {
		t.Fatalf("Type+Status+Tags filter: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("Type+Status+Tags filter: got %d, want 1", len(docs))
	}

	// Type + Date range filter
	docs, err = store.ListDocs(domain.DocFilter{Type: "decision", After: "2026-03-04"})
	if err != nil {
		t.Fatalf("Type+Date filter: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("Type+Date filter: got %d, want 1", len(docs))
	}

	// Tags + Status filter (no Type)
	docs, err = store.ListDocs(domain.DocFilter{Status: "published", Tags: []string{"api"}})
	if err != nil {
		t.Fatalf("Tags+Status filter: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("Tags+Status filter: got %d, want 1", len(docs))
	}
}

func TestCorpusStore_ListDocs_NoResults(t *testing.T) {
	dir := t.TempDir()
	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-07", Status: "published"}, "doc", "body\n")

	store := &CorpusStore{Dir: dir}
	docs, err := store.ListDocs(domain.DocFilter{Type: "nonexistent"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("expected 0 docs, got %d", len(docs))
	}
}

func TestCorpusStore_ListDocs_SkipsREADME(t *testing.T) {
	dir := t.TempDir()
	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-07", Status: "published"}, "doc", "body\n")

	// README.md should already exist from WriteDoc's RegenerateIndex call
	readmePath := filepath.Join(dir, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		// Write a README manually if it wasn't created
		os.WriteFile(readmePath, []byte("# Index\n"), 0644)
	}

	store := &CorpusStore{Dir: dir}
	docs, err := store.ListDocs(domain.DocFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	// Should only get 1 doc, not README
	if len(docs) != 1 {
		t.Errorf("expected 1 doc (README excluded), got %d", len(docs))
	}
}

func TestCorpusStore_ReadDoc_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	store := &CorpusStore{Dir: dir}

	tests := []string{
		"../../../etc/passwd",
		"/etc/passwd",
		"subdir/file",
		"foo\\bar",
		"..",
	}
	for _, id := range tests {
		_, err := store.ReadDoc(id)
		if err == nil {
			t.Errorf("expected error for path traversal id %q", id)
		}
	}
}

func TestCorpusStore_ListDocs_PopulatesFilename(t *testing.T) {
	dir := t.TempDir()
	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-07", Status: "published"}, "auth", "body\n")

	store := &CorpusStore{Dir: dir}
	docs, err := store.ListDocs(domain.DocFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	if docs[0].Filename == "" {
		t.Error("expected Filename to be populated")
	}
	if docs[0].Filename != "decision-auth-2026-03-07.md" {
		t.Errorf("Filename: got %q", docs[0].Filename)
	}
}

func TestCorpusStore_ListDocs_ParseErrors(t *testing.T) {
	dir := t.TempDir()

	// Write a valid doc
	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-07", Status: "published"}, "valid", "body\n")

	// Write a broken .md file manually (no front matter)
	os.WriteFile(filepath.Join(dir, "broken.md"), []byte("no front matter here"), 0644)

	store := &CorpusStore{Dir: dir}
	docs, err := store.ListDocs(domain.DocFilter{})

	// Should return partial results AND an error
	if len(docs) != 1 {
		t.Errorf("expected 1 valid doc, got %d", len(docs))
	}
	if err == nil {
		t.Error("expected parse error for broken.md")
	}
}

func TestCorpusStore_ListDocs_FilterByTextInFilename(t *testing.T) {
	dir := t.TempDir()
	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-07", Status: "published"}, "jwt auth", "some body\n")
	WriteDoc(dir, domain.DocMeta{Type: "feature", Date: "2026-03-08", Status: "published"}, "db choice", "other body\n")

	store := &CorpusStore{Dir: dir}
	docs, err := store.ListDocs(domain.DocFilter{Text: "jwt"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 doc matching filename text, got %d", len(docs))
	}
}

// --- SearchDocs tests (Story 3.1) ---

func TestSearchDocs_Keyword(t *testing.T) {
	dir := t.TempDir()
	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-07", Status: "published"}, "auth strategy", "# Auth Strategy\n\nWe chose JWT.\n")
	WriteDoc(dir, domain.DocMeta{Type: "feature", Date: "2026-03-08", Status: "published"}, "db choice", "# Database Choice\n\nPostgres.\n")

	results, err := SearchDocs(dir, "auth", domain.DocFilter{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Meta.Type != "decision" {
		t.Errorf("expected type 'decision', got %q", results[0].Meta.Type)
	}
	if results[0].Title != "Auth Strategy" {
		t.Errorf("expected title 'Auth Strategy', got %q", results[0].Title)
	}
}

func TestSearchDocs_FilterType(t *testing.T) {
	dir := t.TempDir()
	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-07", Status: "published"}, "auth", "# Auth\n")
	WriteDoc(dir, domain.DocMeta{Type: "feature", Date: "2026-03-08", Status: "published"}, "auth api", "# Auth API\n")

	results, err := SearchDocs(dir, "auth", domain.DocFilter{Type: "feature"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Meta.Type != "feature" {
		t.Errorf("expected type 'feature', got %q", results[0].Meta.Type)
	}
}

func TestSearchDocs_FilterDate(t *testing.T) {
	dir := t.TempDir()
	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-02-15", Status: "published"}, "old auth", "# Old\n")
	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-10", Status: "published"}, "new auth", "# New\n")

	results, err := SearchDocs(dir, "auth", domain.DocFilter{After: "2026-03-01"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Meta.Date != "2026-03-10" {
		t.Errorf("expected date '2026-03-10', got %q", results[0].Meta.Date)
	}
}

func TestSearchDocs_FilterDateYYYYMM(t *testing.T) {
	dir := t.TempDir()
	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-02-15", Status: "published"}, "feb auth", "# Feb\n")
	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-10", Status: "published"}, "mar auth", "# Mar\n")

	// YYYY-MM format should be normalized to YYYY-MM-01
	results, err := SearchDocs(dir, "auth", domain.DocFilter{After: "2026-03"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Meta.Date != "2026-03-10" {
		t.Errorf("expected date '2026-03-10', got %q", results[0].Meta.Date)
	}
}

func TestSearchDocs_CombinedKeywordTypeDate(t *testing.T) {
	dir := t.TempDir()
	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-07", Status: "published"}, "auth jwt", "# JWT Auth\n")
	WriteDoc(dir, domain.DocMeta{Type: "feature", Date: "2026-03-08", Status: "published"}, "auth api", "# Auth API\n")
	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-02-01", Status: "published"}, "auth old", "# Old Auth\n")

	results, err := SearchDocs(dir, "auth", domain.DocFilter{Type: "decision", After: "2026-03-01"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Title != "JWT Auth" {
		t.Errorf("expected 'JWT Auth', got %q", results[0].Title)
	}
}

func TestSearchDocs_NoMatch(t *testing.T) {
	dir := t.TempDir()
	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-07", Status: "published"}, "auth", "# Auth\n")

	results, err := SearchDocs(dir, "nonexistent", domain.DocFilter{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchDocs_CaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-07", Status: "published"}, "jwt auth", "# JWT Auth Strategy\n")

	results, err := SearchDocs(dir, "JWT", domain.DocFilter{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for case-insensitive search, got %d", len(results))
	}
}

func TestSearchDocs_MatchesTags(t *testing.T) {
	dir := t.TempDir()
	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-07", Status: "published", Tags: []string{"authentication", "security"}}, "login flow", "# Login Flow\n\nBasic login.\n")

	results, err := SearchDocs(dir, "security", domain.DocFilter{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result matching tag, got %d", len(results))
	}
}

func TestSearchDocs_SortedByDateDesc(t *testing.T) {
	dir := t.TempDir()
	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-01", Status: "published"}, "auth one", "# Auth One\n")
	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-10", Status: "published"}, "auth two", "# Auth Two\n")
	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-05", Status: "published"}, "auth three", "# Auth Three\n")

	results, err := SearchDocs(dir, "auth", domain.DocFilter{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Meta.Date != "2026-03-10" {
		t.Errorf("first result should be newest, got %q", results[0].Meta.Date)
	}
	if results[1].Meta.Date != "2026-03-05" {
		t.Errorf("second result date: got %q", results[1].Meta.Date)
	}
	if results[2].Meta.Date != "2026-03-01" {
		t.Errorf("third result should be oldest, got %q", results[2].Meta.Date)
	}
}

func TestSearchDocs_AllDocs(t *testing.T) {
	dir := t.TempDir()
	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-07", Status: "published"}, "auth", "# Auth\n")
	WriteDoc(dir, domain.DocMeta{Type: "feature", Date: "2026-03-08", Status: "published"}, "api", "# API\n")

	// Empty keyword → all docs
	results, err := SearchDocs(dir, "", domain.DocFilter{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results for empty keyword, got %d", len(results))
	}
}

func TestReadDocContent(t *testing.T) {
	dir := t.TempDir()
	res, err := WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-07", Status: "published"}, "test", "# Test\n\nContent.\n")
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	content, err := ReadDocContent(res.Path)
	if err != nil {
		t.Fatalf("read content: %v", err)
	}
	if content == "" {
		t.Error("expected non-empty content")
	}
	if !strings.Contains(content, "# Test") {
		t.Error("expected body in content")
	}
	if !strings.Contains(content, "type: decision") {
		t.Error("expected front matter in content")
	}
}

func TestCorpusStore_ReadDoc_SymlinkTraversal(t *testing.T) {
	docsDir := t.TempDir()
	store := &CorpusStore{Dir: docsDir}

	// Create a file outside the docs directory.
	outsideDir := t.TempDir()
	secretPath := filepath.Join(outsideDir, "secret.md")
	os.WriteFile(secretPath, []byte("---\ntype: decision\ndate: 2026-03-07\nstatus: published\n---\n# Secret\n"), 0644)

	// Create a symlink inside docsDir that points to the outside file.
	symlinkPath := filepath.Join(docsDir, "evil.md")
	if err := os.Symlink(secretPath, symlinkPath); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	_, err := store.ReadDoc("evil")
	if err == nil {
		t.Fatal("expected error for symlink-based path traversal, got nil")
	}
	if !strings.Contains(err.Error(), "escapes") {
		t.Errorf("expected 'escapes' in error, got: %v", err)
	}
}

func TestDeleteDoc_SymlinkTraversal(t *testing.T) {
	docsDir := t.TempDir()

	// Create a file outside the docs directory.
	outsideDir := t.TempDir()
	secretPath := filepath.Join(outsideDir, "victim.md")
	os.WriteFile(secretPath, []byte("important data"), 0644)

	// Create a symlink inside docsDir pointing outside.
	symlinkPath := filepath.Join(docsDir, "evil.md")
	if err := os.Symlink(secretPath, symlinkPath); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	err := DeleteDoc(docsDir, "evil.md")
	if err == nil {
		t.Fatal("expected error for symlink-based path traversal, got nil")
	}
	if !strings.Contains(err.Error(), "escapes") {
		t.Errorf("expected 'escapes' in error, got: %v", err)
	}

	// Verify the outside file was NOT deleted.
	if _, statErr := os.Stat(secretPath); os.IsNotExist(statErr) {
		t.Error("outside file was deleted despite symlink check")
	}
}

func TestExtractSlug(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"decision-auth-strategy-2026-03-07.md", "auth-strategy"},
		{"feature-add-jwt-middleware-2026-03-05.md", "add-jwt-middleware"},
		{"bugfix-token-expiry-fix-2026-03-01.md", "token-expiry-fix"},
		{"note-simple-2026-01-01.md", "simple"},
	}
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := ExtractSlug(tt.filename)
			if got != tt.want {
				t.Errorf("ExtractSlug(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

func TestExtractSlug_NoSlug(t *testing.T) {
	got := ExtractSlug("feature-2026-03-07.md")
	if got != "untitled" {
		t.Errorf("ExtractSlug(%q) = %q, want %q", "feature-2026-03-07.md", got, "untitled")
	}
}

func TestNormalizeAfter(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"2026-03", "2026-03-01"},
		{"2026-03-15", "2026-03-15"},
		{"", ""},
	}
	for _, tt := range tests {
		got := NormalizeAfter(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeAfter(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- FindDocByCommit tests (Story 4.1) ---

func TestFindDocByCommit_Found(t *testing.T) {
	dir := t.TempDir()
	commitHash := "abc1234567890abcdef1234567890abcdef123456"
	WriteDoc(dir, domain.DocMeta{
		Type:        "decision",
		Date:        "2026-03-16",
		Status:      "published",
		Commit:      commitHash,
		GeneratedBy: "retroactive",
	}, "auth strategy", "# Auth Strategy\n")

	result, err := FindDocByCommit(dir, commitHash)
	if err != nil {
		t.Fatalf("FindDocByCommit: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.Meta.Commit != commitHash {
		t.Errorf("commit = %q, want %q", result.Meta.Commit, commitHash)
	}
	if result.Title != "Auth Strategy" {
		t.Errorf("title = %q, want %q", result.Title, "Auth Strategy")
	}
}

func TestFindDocByCommit_NotFound(t *testing.T) {
	dir := t.TempDir()
	WriteDoc(dir, domain.DocMeta{
		Type:   "decision",
		Date:   "2026-03-16",
		Status: "published",
		Commit: "aaa1111111111111111111111111111111111111111",
	}, "other", "# Other\n")

	result, err := FindDocByCommit(dir, "bbb2222222222222222222222222222222222222222")
	if err != nil {
		t.Fatalf("FindDocByCommit: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for non-matching hash, got %+v", result)
	}
}

func TestFindDocByCommit_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	result, err := FindDocByCommit(dir, "abc123")
	if err != nil {
		t.Fatalf("FindDocByCommit empty: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for empty dir, got %+v", result)
	}
}

func TestValidateFilename(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty", "", true},
		{"valid", "decision-auth-2026-03-07.md", false},
		{"path_traversal", "../evil.md", true},
		{"absolute", "/etc/passwd", true},
		{"reserved_readme", "README.md", true},
		{"reserved_index", "index.md", true},
		{"reserved_lock", ".index.lock", true},
		{"with_separator", "sub/file.md", true},
		{"dot_dot", "..", true},
		{"valid_simple", "notes.md", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilename(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFilename(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestExtractTitle_HeadingFallback(t *testing.T) {
	// With heading
	title := ExtractTitle("# My Title\n\nSome content.\n", "decision-auth-2026-03-07.md")
	if title != "My Title" {
		t.Errorf("expected 'My Title', got %q", title)
	}

	// Without heading — fallback to slug
	title = ExtractTitle("No heading here.\n", "decision-auth-strategy-2026-03-07.md")
	if title != "auth-strategy" {
		t.Errorf("expected 'auth-strategy', got %q", title)
	}
}
