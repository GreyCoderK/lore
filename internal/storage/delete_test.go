package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/museigen/lore/internal/domain"
)

func TestDeleteDoc_Success(t *testing.T) {
	dir := t.TempDir()

	result, err := WriteDoc(dir, domain.DocMeta{
		Type:   "decision",
		Date:   "2026-03-07",
		Status: "published",
	}, "auth strategy", "# Auth\n")
	if err != nil {
		t.Fatalf("setup write: %v", err)
	}

	// Verify file exists before delete
	if _, err := os.Stat(result.Path); err != nil {
		t.Fatalf("file should exist before delete: %v", err)
	}

	if err := DeleteDoc(dir, result.Filename); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// File should be gone
	if _, err := os.Stat(result.Path); !os.IsNotExist(err) {
		t.Error("file should not exist after delete")
	}
}

func TestDeleteDoc_NotFound(t *testing.T) {
	dir := t.TempDir()

	err := DeleteDoc(dir, "nonexistent.md")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "storage: delete") {
		t.Errorf("error should follow storage pattern, got: %v", err)
	}
}

func TestDeleteDoc_PathTraversal(t *testing.T) {
	dir := t.TempDir()

	tests := []string{
		"../../../etc/passwd",
		"/etc/passwd",
		"subdir/file.md",
		"..",
	}
	for _, filename := range tests {
		err := DeleteDoc(dir, filename)
		if err == nil {
			t.Errorf("expected error for path traversal %q", filename)
		}
	}
}

func TestDeleteDoc_RegeneratesIndex(t *testing.T) {
	dir := t.TempDir()

	// Write two docs
	r1, _ := WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-07", Status: "published"}, "one", "body\n")
	WriteDoc(dir, domain.DocMeta{Type: "feature", Date: "2026-03-08", Status: "published"}, "two", "body\n")

	// README should show 2 docs
	data, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	if !strings.Contains(string(data), "2 documents total") {
		t.Fatalf("expected 2 documents before delete, got:\n%s", string(data))
	}

	// Delete first doc
	if err := DeleteDoc(dir, r1.Filename); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// README should now show 1 doc
	data, _ = os.ReadFile(filepath.Join(dir, "README.md"))
	content := string(data)
	if !strings.Contains(content, "1 documents total") {
		t.Errorf("expected 1 document after delete, got:\n%s", content)
	}
	if strings.Contains(content, r1.Filename) {
		t.Error("deleted doc should not appear in index")
	}
}
