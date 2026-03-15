package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/museigen/lore/internal/domain"
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

	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-07", Status: "published", Tags: []string{"auth"}}, "one", "body\n")
	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-07", Status: "draft", Tags: []string{"auth"}}, "two", "body\n")
	WriteDoc(dir, domain.DocMeta{Type: "feature", Date: "2026-03-07", Status: "published", Tags: []string{"auth"}}, "three", "body\n")

	store := &CorpusStore{Dir: dir}
	docs, err := store.ListDocs(domain.DocFilter{Type: "decision", Status: "published", Tags: []string{"auth"}})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 doc matching all filters, got %d", len(docs))
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
