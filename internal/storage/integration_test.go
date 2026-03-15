package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/museigen/lore/internal/domain"
)

func TestIntegration_FullLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := t.TempDir()

	// --- Write 3 documents ---
	r1, err := WriteDoc(dir, domain.DocMeta{
		Type:   "decision",
		Date:   "2026-03-05",
		Status: "published",
		Tags:   []string{"auth", "api"},
	}, "auth strategy", "# Authentication Strategy\n\nWe chose JWT for auth.\n")
	if err != nil {
		t.Fatalf("write doc 1: %v", err)
	}

	r2, err := WriteDoc(dir, domain.DocMeta{
		Type:   "feature",
		Date:   "2026-03-07",
		Status: "published",
		Tags:   []string{"api"},
	}, "api versioning", "# API Versioning\n\nVersion via URL prefix.\n")
	if err != nil {
		t.Fatalf("write doc 2: %v", err)
	}

	r3, err := WriteDoc(dir, domain.DocMeta{
		Type:   "note",
		Date:   "2026-03-10",
		Status: "draft",
	}, "meeting notes", "# Meeting Notes\n\nDiscussed roadmap.\n")
	if err != nil {
		t.Fatalf("write doc 3: %v", err)
	}

	// --- Verify files exist ---
	for _, r := range []WriteResult{r1, r2, r3} {
		if _, err := os.Stat(r.Path); err != nil {
			t.Errorf("file should exist: %s", r.Path)
		}
	}

	// --- Verify README index ---
	readme, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	readmeContent := string(readme)
	if !strings.Contains(readmeContent, "3 documents total") {
		t.Errorf("README should show 3 docs, got:\n%s", readmeContent)
	}

	// --- Read a document ---
	store := &CorpusStore{Dir: dir}
	content, err := store.ReadDoc(r1.Filename)
	if err != nil {
		t.Fatalf("read doc: %v", err)
	}
	if !strings.Contains(content, "JWT") {
		t.Error("read doc should contain body content")
	}

	// --- List all documents ---
	docs, err := store.ListDocs(domain.DocFilter{})
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(docs) != 3 {
		t.Errorf("expected 3 docs, got %d", len(docs))
	}

	// --- Filter by type ---
	docs, err = store.ListDocs(domain.DocFilter{Type: "decision"})
	if err != nil {
		t.Fatalf("filter type: %v", err)
	}
	if len(docs) != 1 || docs[0].Type != "decision" {
		t.Errorf("expected 1 decision, got %d", len(docs))
	}

	// --- Filter by status ---
	docs, err = store.ListDocs(domain.DocFilter{Status: "draft"})
	if err != nil {
		t.Fatalf("filter status: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 draft, got %d", len(docs))
	}

	// --- Filter by tags ---
	docs, err = store.ListDocs(domain.DocFilter{Tags: []string{"api"}})
	if err != nil {
		t.Fatalf("filter tags: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("expected 2 docs with 'api' tag, got %d", len(docs))
	}

	// --- Filter by date range ---
	docs, err = store.ListDocs(domain.DocFilter{After: "2026-03-06", Before: "2026-03-08"})
	if err != nil {
		t.Fatalf("filter date: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 doc in date range, got %d", len(docs))
	}

	// --- Filter by text ---
	docs, err = store.ListDocs(domain.DocFilter{Text: "JWT"})
	if err != nil {
		t.Fatalf("filter text: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 doc matching JWT, got %d", len(docs))
	}

	// --- Combined filters ---
	docs, err = store.ListDocs(domain.DocFilter{Type: "decision", Tags: []string{"auth"}})
	if err != nil {
		t.Fatalf("combined filter: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 doc matching combined, got %d", len(docs))
	}

	// --- Delete a document ---
	if err := DeleteDoc(dir, r2.Filename); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Verify file gone
	if _, err := os.Stat(r2.Path); !os.IsNotExist(err) {
		t.Error("deleted file should not exist")
	}

	// Verify index updated
	readme, _ = os.ReadFile(filepath.Join(dir, "README.md"))
	readmeContent = string(readme)
	if !strings.Contains(readmeContent, "2 documents total") {
		t.Errorf("README should show 2 docs after delete, got:\n%s", readmeContent)
	}
	if strings.Contains(readmeContent, r2.Filename) {
		t.Error("deleted doc should not appear in index")
	}

	// Verify list updated
	docs, err = store.ListDocs(domain.DocFilter{})
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("expected 2 docs after delete, got %d", len(docs))
	}
}

func TestIntegration_AtomicWriteCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := t.TempDir()

	// Write a document successfully
	result, err := WriteDoc(dir, domain.DocMeta{
		Type:   "decision",
		Date:   "2026-03-07",
		Status: "published",
	}, "test", "body\n")
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	// No .tmp files should remain
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("leftover tmp file: %s", e.Name())
		}
	}

	// File should have correct content
	data, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.HasPrefix(string(data), "---\n") {
		t.Error("file should start with front matter delimiter")
	}
}

func TestIntegration_WriteAndIndex_MultiFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := t.TempDir()

	// Each WriteDoc should atomically write the doc AND regenerate the index
	WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-05", Status: "published"}, "one", "body\n")
	WriteDoc(dir, domain.DocMeta{Type: "feature", Date: "2026-03-07", Status: "published"}, "two", "body\n")
	WriteDoc(dir, domain.DocMeta{Type: "note", Date: "2026-03-10", Status: "draft"}, "three", "body\n")

	// Verify all files exist (3 docs + README)
	entries, _ := os.ReadDir(dir)
	mdCount := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			mdCount++
		}
	}
	if mdCount != 4 { // 3 docs + README.md
		t.Errorf("expected 4 .md files (3 docs + README), got %d", mdCount)
	}

	// Index should reflect all 3 docs
	readme, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	if !strings.Contains(string(readme), "3 documents total") {
		t.Errorf("README should show 3 docs, got:\n%s", string(readme))
	}
}
