// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

func TestPlainCorpusStore_ListDocs_PlainMarkdown(t *testing.T) {
	dir := t.TempDir()

	// Write a plain markdown file (no front matter)
	content := "# API Guide\n\nThis is a guide about the API.\n"
	if err := os.WriteFile(filepath.Join(dir, "api-guide.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	store := &PlainCorpusStore{Dir: dir}
	docs, err := store.ListDocs(domain.DocFilter{})
	if err != nil {
		t.Fatalf("ListDocs: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}

	doc := docs[0]
	if doc.Filename != "api-guide.md" {
		t.Errorf("Filename = %q, want %q", doc.Filename, "api-guide.md")
	}
	if doc.Type != "note" {
		t.Errorf("Type = %q, want %q (synthetic fallback)", doc.Type, "note")
	}
	if doc.Status != "published" {
		t.Errorf("Status = %q, want %q", doc.Status, "published")
	}
}

func TestPlainCorpusStore_ListDocs_WithFrontMatter(t *testing.T) {
	dir := t.TempDir()

	// Write a lore-style file with front matter
	content := "---\ntype: decision\nstatus: published\ndate: \"2026-03-19\"\ntags: [api]\n---\n## What\nA decision.\n"
	if err := os.WriteFile(filepath.Join(dir, "decision-api-2026-03-19.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	store := &PlainCorpusStore{Dir: dir}
	docs, err := store.ListDocs(domain.DocFilter{})
	if err != nil {
		t.Fatalf("ListDocs: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}

	doc := docs[0]
	if doc.Type != "decision" {
		t.Errorf("Type = %q, want %q (parsed from front matter)", doc.Type, "decision")
	}
	if len(doc.Tags) == 0 || doc.Tags[0] != "api" {
		t.Errorf("Tags = %v, want [api]", doc.Tags)
	}
}

func TestPlainCorpusStore_ListDocs_MixedFiles(t *testing.T) {
	dir := t.TempDir()

	// One with front matter, one without
	withFM := "---\ntype: feature\nstatus: published\ndate: \"2026-04-01\"\n---\nFeature doc.\n"
	withoutFM := "# Architecture Overview\n\nPlain markdown.\n"

	_ = os.WriteFile(filepath.Join(dir, "feature-auth.md"), []byte(withFM), 0644)
	_ = os.WriteFile(filepath.Join(dir, "architecture.md"), []byte(withoutFM), 0644)
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Index\n"), 0644) // should be skipped

	store := &PlainCorpusStore{Dir: dir}
	docs, err := store.ListDocs(domain.DocFilter{})
	if err != nil {
		t.Fatalf("ListDocs: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 docs (README skipped), got %d", len(docs))
	}
}

func TestPlainCorpusStore_ReadDoc(t *testing.T) {
	dir := t.TempDir()
	content := "# Test\n\nSome content.\n"
	_ = os.WriteFile(filepath.Join(dir, "test.md"), []byte(content), 0644)

	store := &PlainCorpusStore{Dir: dir}
	result, err := store.ReadDoc("test.md")
	if err != nil {
		t.Fatalf("ReadDoc: %v", err)
	}
	if result != content {
		t.Errorf("ReadDoc = %q, want %q", result, content)
	}
}

func TestPlainCorpusStore_ReadDoc_NotFound(t *testing.T) {
	dir := t.TempDir()
	store := &PlainCorpusStore{Dir: dir}
	_, err := store.ReadDoc("nonexistent.md")
	if err == nil {
		t.Fatal("expected error for nonexistent doc")
	}
}

func TestPlainCorpusStore_ListDocs_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	store := &PlainCorpusStore{Dir: dir}
	docs, err := store.ListDocs(domain.DocFilter{})
	if err != nil {
		t.Fatalf("ListDocs: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("expected 0 docs, got %d", len(docs))
	}
}

func TestPlainCorpusStore_ListDocs_NonexistentDir(t *testing.T) {
	store := &PlainCorpusStore{Dir: "/tmp/nonexistent-dir-lore-test"}
	docs, err := store.ListDocs(domain.DocFilter{})
	if err != nil {
		t.Fatalf("expected nil error for nonexistent dir, got: %v", err)
	}
	if docs != nil {
		t.Errorf("expected nil docs, got %v", docs)
	}
}

func TestBuildPlainMeta(t *testing.T) {
	meta := BuildPlainMeta("api-authentication-guide.md")
	if meta.Type != "note" {
		t.Errorf("Type = %q, want %q", meta.Type, "note")
	}
	if meta.Status != "published" {
		t.Errorf("Status = %q, want %q", meta.Status, "published")
	}
	if meta.Date == "" {
		t.Error("Date should not be empty")
	}
	if len(meta.Tags) == 0 {
		t.Error("Tags should be inferred from filename")
	}
}

func TestPlainCorpusStore_ReadDoc_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	store := &PlainCorpusStore{Dir: dir}
	_, err := store.ReadDoc("../../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
}

func TestPlainCorpusStore_ReadDoc_AbsolutePath(t *testing.T) {
	dir := t.TempDir()
	store := &PlainCorpusStore{Dir: dir}
	_, err := store.ReadDoc("/tmp/test.md")
	if err == nil {
		t.Fatal("expected error for absolute path")
	}
}

func TestPlainCorpusStore_ReadDoc_ReservedFilename(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Index"), 0644)
	store := &PlainCorpusStore{Dir: dir}
	_, err := store.ReadDoc("README.md")
	if err == nil {
		t.Fatal("expected error for reserved filename")
	}
}

func TestInferTagsFromFilename(t *testing.T) {
	tests := []struct {
		name string
		want int // minimum expected tags
	}{
		{"api-authentication-guide.md", 3},
		{"readme.md", 1},
		{"a-b.md", 0}, // too short segments
	}
	for _, tt := range tests {
		tags := inferTagsFromFilename(tt.name)
		if len(tags) < tt.want {
			t.Errorf("inferTagsFromFilename(%q) = %v, want at least %d tags", tt.name, tags, tt.want)
		}
	}
}
