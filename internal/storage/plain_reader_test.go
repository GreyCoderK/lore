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

func TestInferTagsFromFilename_DateFiltered(t *testing.T) {
	tags := inferTagsFromFilename("auth-2026-03-15.md")
	tagSet := map[string]bool{}
	for _, tag := range tags {
		tagSet[tag] = true
	}
	if !tagSet["auth"] {
		t.Errorf("expected 'auth' tag, got %v", tags)
	}
	// Date segments should be filtered: "2026" is 4-digit starting with digit
	if tagSet["2026"] {
		t.Errorf("date segment '2026' should be filtered out, got %v", tags)
	}
	// Short segments "03" and "15" (len < 3) should be filtered
	if tagSet["03"] || tagSet["15"] {
		t.Errorf("short date segments should be filtered out, got %v", tags)
	}
}

func TestInferTagsFromFilename_ShortSegments(t *testing.T) {
	tags := inferTagsFromFilename("a-bb-ccc-dddd.md")
	tagSet := map[string]bool{}
	for _, tag := range tags {
		tagSet[tag] = true
	}
	if tagSet["a"] || tagSet["bb"] {
		t.Errorf("segments shorter than 3 chars should be filtered: got %v", tags)
	}
	if !tagSet["ccc"] || !tagSet["dddd"] {
		t.Errorf("expected 'ccc' and 'dddd', got %v", tags)
	}
}

func TestInferTagsFromFilename_Lowercase(t *testing.T) {
	tags := inferTagsFromFilename("API-Design-Guide.md")
	for _, tag := range tags {
		if tag != strings.ToLower(tag) {
			t.Errorf("tag %q should be lowercase", tag)
		}
	}
}

func TestPlainCorpusStore_ReadDoc_Subdirectory(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "commands")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "test.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	store := &PlainCorpusStore{Dir: dir}
	content, err := store.ReadDoc("commands/test")
	if err != nil {
		t.Fatalf("ReadDoc subdirectory: %v", err)
	}
	if !strings.Contains(content, "# Test") {
		t.Errorf("expected content with '# Test', got %q", content)
	}
}

func TestPlainCorpusStore_ReadDoc_AddsExtension(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte("# Notes\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	store := &PlainCorpusStore{Dir: dir}
	content, err := store.ReadDoc("notes")
	if err != nil {
		t.Fatalf("ReadDoc without extension: %v", err)
	}
	if !strings.Contains(content, "# Notes") {
		t.Errorf("expected content with '# Notes', got %q", content)
	}
}

func TestPlainCorpusStore_ListDocs_NestedSubdirs(t *testing.T) {
	dir := t.TempDir()

	// Create nested structure
	for _, sub := range []string{
		filepath.Join(dir, "commands"),
		filepath.Join(dir, "guides", "advanced"),
	} {
		if err := os.MkdirAll(sub, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", sub, err)
		}
	}

	// Write files at various levels
	files := map[string]string{
		filepath.Join(dir, "root-doc.md"):                   "# Root\n",
		filepath.Join(dir, "commands", "angela.md"):         "# Angela\n",
		filepath.Join(dir, "guides", "advanced", "tips.md"): "# Tips\n",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	store := &PlainCorpusStore{Dir: dir}
	docs, err := store.ListDocs(domain.DocFilter{})
	if err != nil {
		t.Fatalf("ListDocs: %v", err)
	}
	if len(docs) != 3 {
		t.Errorf("expected 3 docs from nested dirs, got %d", len(docs))
		for _, d := range docs {
			t.Logf("  found: %s", d.Filename)
		}
	}

	// Verify relative paths are used as filenames
	filenames := map[string]bool{}
	for _, d := range docs {
		filenames[d.Filename] = true
	}
	for _, want := range []string{
		"root-doc.md",
		filepath.Join("commands", "angela.md"),
		filepath.Join("guides", "advanced", "tips.md"),
	} {
		if !filenames[want] {
			t.Errorf("expected filename %q in results, got %v", want, filenames)
		}
	}
}

func TestPlainCorpusStore_ListDocs_PlainMarkdownGetsInferredTags(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "api-design-guide.md"), []byte("# API Design\n"), 0644)

	store := &PlainCorpusStore{Dir: dir}
	docs, err := store.ListDocs(domain.DocFilter{})
	if err != nil {
		t.Fatalf("ListDocs: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	if len(docs[0].Tags) == 0 {
		t.Error("expected inferred tags from filename for plain markdown")
	}
}

func TestBuildPlainMeta_TagCount(t *testing.T) {
	meta := BuildPlainMeta("api-authentication-guide.md")
	if len(meta.Tags) < 3 {
		t.Errorf("expected >= 3 tags for 'api-authentication-guide.md', got %v", meta.Tags)
	}
}
