package storage

import (
	"testing"
	"time"

	"github.com/museigen/lore/internal/domain"
)

func TestCorpusStore_ReadDoc(t *testing.T) {
	dir := t.TempDir()
	meta := domain.DocMeta{
		Type:   "decision",
		Date:   domain.NewDateString(time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC)),
		Status: "demo",
	}
	body := "# Test Doc\n"

	filename, err := WriteDoc(dir, meta, "test doc", body)
	if err != nil {
		t.Fatalf("storage: write: %v", err)
	}

	store := &CorpusStore{Dir: dir}

	content, err := store.ReadDoc(filename)
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
		Date:   domain.NewDateString(time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC)),
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
